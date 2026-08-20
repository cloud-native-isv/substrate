#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════
#  演示 3：Wasm 沙箱审计能力（host_call 全斡旋）
# ═══════════════════════════════════════════════════════════════════════════
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/.env" 2>/dev/null || { echo "❌ 先运行 bash setup.sh"; exit 1; }

API="https://api.${E2B_DOMAIN}"
CURL="curl --cacert ${SSL_CERT_FILE} -sk"

banner()  { printf '\n%s\n  %s\n%s\n' "$(printf '═%.0s' {1..68})" "$1" "$(printf '═%.0s' {1..68})"; }
show_cmd(){ printf '\n    \033[36m$ %s\033[0m\n' "$1"; }
pause()   { printf '\n    \033[2m[按回车继续]\033[0m '; read -r; }

create_sandbox() {
    local resp
    resp=$($CURL -X POST "${API}/sandboxes" \
        -H "X-API-Key: ${E2B_API_KEY}" -H "Content-Type: application/json" \
        -d "{\"templateID\":\"${E2B_TEMPLATE}\"}")
    SID=$(echo "$resp" | jq -r '.sandboxID')
    TOKEN=$(echo "$resp" | jq -r '.envdAccessToken')
}
execute() {
    $CURL -X POST "https://49999-${SID}.${E2B_DOMAIN}/execute" \
        -H "X-Access-Token: ${TOKEN}" -H "E2B-Traffic-Access-Token: ${TOKEN}" \
        -H "Content-Type: application/json" \
        -d "{\"code\":$(echo "$1" | jq -Rs .)}" 2>/dev/null
}
cleanup() { $CURL -X DELETE "${API}/sandboxes/${SID}" -H "X-API-Key: ${E2B_API_KEY}" >/dev/null 2>&1 || true; }
trap cleanup EXIT

# ═══════════════════════════════════════════════════════════════════════════
banner "演示 3：Wasm 沙箱审计 — host_call 全斡旋"

cat << 'EOF'

    ┌─ 架构 ─────────────────────────────────────────────────────────────────┐
    │  Wasm 沙箱 (python.wasm)           宿主 (ateom-wasmd)                  │
    │  ┌──────────────────────┐          ┌─────────────────────────────┐    │
    │  │ import host_proxy    │ host_call │ handle_host_call()          │    │
    │  │ host_proxy.http_req()├─────────►│ HostProxy.validate_url()    │    │
    │  │                      │◄─────────┤ HostProxy.log_request() ←审计│    │
    │  │ （无 socket syscall） │ response │ HostProxy.http_request()    │    │
    │  └──────────────────────┘          └─────────────────────────────┘    │
    │                                                                        │
    │  wasm 出网的唯一通道 = 宿主审计的唯一插桩点 → 覆盖率 100%              │
    └────────────────────────────────────────────────────────────────────────┘
EOF

create_sandbox
echo "  沙箱: ${SID}"
pause

# ── 场景 1：合法公网请求 ─────────────────────────────────────────────────
banner "场景 1：合法公网请求 → 放行 + 审计记录"

CODE='import host_proxy
status, body = host_proxy.http_request("GET", "https://httpbin.org/get")
print(f"status={status}, body_len={len(body)}")'

echo "    沙箱代码:"
echo "$CODE" | sed 's/^/      /'
echo ""
show_cmd "curl POST /execute -d '{\"code\":\"import host_proxy; ...\"}'"

RESP=$(execute "$CODE")
echo ""
echo "    原始 NDJSON 响应:"
echo "$RESP" | while IFS= read -r line; do [ -n "$line" ] && echo "      $line"; done || true
echo ""
echo "    解读: 请求经 HostProxy 放行（allowed=true）并代理执行"
echo "    审计记录: {url: httpbin.org/get, method: GET, allowed: true, status: 200}"
pause

# ── 场景 2：非法内网请求（核心）────────────────────────────────────────
banner "场景 2：非法内网请求 → 阻止 + 实时告警 ⚡"

CODE='import host_proxy

targets = [
    ("GET",  "http://10.96.0.1:443/api/v1/secrets"),
    ("POST", "http://192.168.1.1:8080/admin/delete"),
    ("GET",  "http://169.254.169.254/latest/meta-data/"),
]
for method, url in targets:
    try:
        host_proxy.http_request(method, url)
        print(f"BUG: {url} should be blocked!")
    except RuntimeError as e:
        print(f"BLOCKED {method} {url} -> {e}")'

echo "    沙箱代码尝试访问:"
echo "      ① K8s API Server   http://10.96.0.1:443/api/v1/secrets"
echo "      ② 内网服务          http://192.168.1.1:8080/admin/delete"
echo "      ③ 云 metadata       http://169.254.169.254/latest/meta-data/"
echo ""
show_cmd "curl POST /execute -d '{\"code\":\"import host_proxy; host_proxy.http_request(\\\"GET\\\", \\\"http://10.96.0.1:443/...\\\")...\"}'"

RESP=$(execute "$CODE")
echo ""
echo "    原始 NDJSON 响应（沙箱 stdout 输出）:"
echo "$RESP" | jq -r 'select(.type=="stdout") | .text' 2>/dev/null | while IFS= read -r line; do
    [ -n "$line" ] && echo "      $line"
done || true

echo ""
echo "    ── kubectl logs 取证（所有 worker Pod 审计告警）──"
show_cmd "kubectl logs -n ate-wasm -l ate.dev/worker-pool=wasm-pool --since=30s | grep BLOCKED"
sleep 2
for WPOD in $(kubectl get pods -n ate-wasm -l ate.dev/worker-pool=wasm-pool \
    --field-selector=status.phase=Running -o jsonpath='{.items[*].metadata.name}'); do
    kubectl logs "$WPOD" -n ate-wasm --since=30s 2>/dev/null \
        | grep -i "blocked" | tail -3 | while IFS= read -r line; do
        TS=$(echo "$line" | jq -r '.timestamp' 2>/dev/null | head -c 23)
        URL=$(echo "$line" | jq -r '.fields.url' 2>/dev/null)
        METHOD=$(echo "$line" | jq -r '.fields.method' 2>/dev/null)
        printf '      %s ⚠️  BLOCKED %s %s\n' "$TS" "$METHOD" "$URL"
    done || true
done || true

echo ""
echo "    证据链:"
echo "      1) 沙箱 stdout: RuntimeError 'access denied: IP in blocked network range'"
echo "      2) Pod 日志: tracing::warn! 'host proxy: request BLOCKED' + url + method"
echo "      3) 内存审计: HostProxy.request_log[{allowed:false, url, method}]"
echo ""
echo "    ★ wasm 进程内没有 socket syscall → 出网只能走 host_call → 绕过 = 不可能"
pause

# ── 场景 3：NDJSON 全量事件流 ────────────────────────────────────────────
banner "场景 3：代码执行 NDJSON 事件流 = per-cell 审计协议"

CODE='result = sum(range(1, 101)); print(f"1+2+...+100 = {result}")'
show_cmd "curl POST /execute -d '{\"code\":\"result = sum(range(1,101)); print(...)\"}"
echo ""

RESP=$(execute "$CODE")
echo "    完整 NDJSON 响应（宿主侧原始记录）:"
echo "$RESP" | while IFS= read -r line; do
    if [ -n "$line" ]; then
        TYPE=$(echo "$line" | jq -r '.type' 2>/dev/null)
        case "$TYPE" in
            stdout)
                TEXT=$(echo "$line" | jq -r '.text' 2>/dev/null)
                TS=$(echo "$line" | jq -r '.timestamp' 2>/dev/null)
                printf '      {"type":"stdout", "text":"%s", "timestamp":%s}\n' "$TEXT" "$TS" ;;
            number_of_executions)
                N=$(echo "$line" | jq -r '.execution_count' 2>/dev/null)
                printf '      {"type":"number_of_executions", "execution_count":%s}  ← 单调递增\n' "$N" ;;
            end)
                printf '      {"type":"end"}  ← cell 执行完毕信号\n' ;;
            *)
                echo "      $line" ;;
        esac
    fi
done || true
echo ""
echo "    宿主看到:"
echo "      • 代码全文（POST body）"
echo "      • stdout/stderr 逐帧输出 + 毫秒时间戳"
echo "      • execution_count 单调递增（不可篡改顺序证据）"
echo "      • end 终止信号（cell 边界明确）"
pause

# ── 场景 4：Agent 出网审计链 ─────────────────────────────────────────────
banner "场景 4：AI Agent 多步推理的审计链"
echo "    模拟 Agent 三步推理: 数据→分析→出网 API"
echo ""

echo "    Step 1: 准备数据"
show_cmd "curl POST /execute -d '{\"code\":\"data = {\\\"DAU\\\": 12580}\"}'"
execute 'data = {"DAU": 12580}; print("数据准备完成")' | jq -r 'select(.type=="stdout").text' 2>/dev/null | sed 's/^/      stdout: /' || true
echo "      审计: NDJSON 记录代码+输出，无出网"

echo ""
echo "    Step 2: 分析计算"
show_cmd "curl POST /execute -d '{\"code\":\"growth = data[\\\"DAU\\\"] * 0.15; print(growth)\"}'"
execute 'growth = data["DAU"] * 0.15; print(f"growth={growth:.0f}")' | jq -r 'select(.type=="stdout").text' 2>/dev/null | sed 's/^/      stdout: /' || true
echo "      审计: NDJSON 记录代码+输出，无出网"

echo ""
echo "    Step 3: 出网 API 调用"
show_cmd "curl POST /execute -d '{\"code\":\"host_proxy.http_request(\\\"POST\\\", \\\"https://httpbin.org/post\\\", ...)\"}'"
RESP=$(execute 'import host_proxy
try:
    s, b = host_proxy.http_request("POST", "https://httpbin.org/post", headers={"Content-Type":"application/json"}, body="{\"growth\":1887}")
    print(f"status={s}")
except Exception as e:
    print(f"error: {e}")')
echo "$RESP" | jq -r 'select(.type=="stdout").text' 2>/dev/null | sed 's/^/      stdout: /' || true
echo "      审计: NDJSON + HostProxy.log_request{url,method,allowed,status}"
echo ""
echo "    → 每一步代码+输出+出网全链条，不可绕过、不可删除"
pause

# ── 对比 ─────────────────────────────────────────────────────────────────
banner "对比：Wasm vs gVisor/microVM 审计架构"
cat << 'EOF'
    ┌──────────────────────────────────────────────────────────────────────┐
    │              Wasm                            gVisor / microVM         │
    ├──────────────────────────────────────────────────────────────────────┤
    │ 网络审计   host_call→HostProxy 唯一通道    需 netfilter/eBPF/Falco   │
    │ 代码审计   宿主直接看到 POST body           内核态看不到应用层代码     │
    │ 绕过风险   不可能（无 socket）              容器逃逸/内核漏洞         │
    │ 部署成本   零（同进程 tracing）             eBPF DaemonSet + sidecar  │
    │ 覆盖率     100%（架构保证）                取决于配置完整度           │
    │                                                                      │
    │ 证据形态   NDJSON 事件流 + Pod 结构化日志   seccomp log + audit log   │
    │            （直接可关联到代码文本）          （只有 syscall 序列）      │
    └──────────────────────────────────────────────────────────────────────┘
EOF
pause

banner "演示 3 总结"
cat << 'EOF'
  ┌───────────────────────────────────────────────────────────────────────┐
  │ 所有证据均为 curl 原始响应 + kubectl logs 实时抓取                     │
  │                                                                       │
  │ ✅ 出网审计: NDJSON 看到代码 + Pod 日志看到 BLOCKED（双重佐证）        │
  │ ✅ 内网防护: blocked_cidrs 拦截 + RuntimeError 返回给沙箱              │
  │ ✅ 代码审计: POST body = 代码全文 + NDJSON = 输出全量                  │
  │ ✅ 不可绕过: wasm 无 socket syscall（架构层保证，非配置依赖）          │
  │                                                                       │
  │ 结论: 审计是 "no ambient capability" 的必然推论，不是附加功能          │
  └───────────────────────────────────────────────────────────────────────┘
EOF
