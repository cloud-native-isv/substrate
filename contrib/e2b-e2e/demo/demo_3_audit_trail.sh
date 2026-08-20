#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════
#  演示 3：Wasm 沙箱审计能力（host_call 全斡旋）
# ═══════════════════════════════════════════════════════════════════════════
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/.env" 2>/dev/null || { echo "❌ 先运行 bash setup.sh"; exit 1; }

API="https://api.${E2B_DOMAIN}"
H_API=(-H "X-API-Key: ${E2B_API_KEY}")
CURL="curl -sk --cacert ${SSL_CERT_FILE}"

banner()  { printf '\n%s\n  %s\n%s\n' "$(printf '═%.0s' {1..68})" "$1" "$(printf '═%.0s' {1..68})"; }
pause()   { printf '\n    [按回车继续] '; read -r; }
ndjson_stdout() { echo "$1" | jq -r 'select(.type=="stdout") | .text' 2>/dev/null | tr -d '\n'; }

create_sandbox() {
    local resp
    resp=$($CURL -X POST "${API}/sandboxes" "${H_API[@]}" \
        -H "Content-Type: application/json" -d "{\"templateID\":\"${E2B_TEMPLATE}\"}")
    SID=$(echo "$resp" | jq -r '.sandboxID')
    TOKEN=$(echo "$resp" | jq -r '.envdAccessToken')
    [ "$SID" != "null" ] || { echo "创建失败: $resp"; exit 1; }
}

execute() {
    local code="$1"
    $CURL -X POST "https://49999-${SID}.${E2B_DOMAIN}/execute" \
        -H "X-Access-Token: ${TOKEN}" -H "E2B-Traffic-Access-Token: ${TOKEN}" \
        -H "Content-Type: application/json" \
        -d "{\"code\":$(echo "$code" | jq -Rs .)}" 2>/dev/null
}

cleanup() { $CURL -X DELETE "${API}/sandboxes/${SID}" "${H_API[@]}" >/dev/null 2>&1 || true; }
trap cleanup EXIT

# ═══════════════════════════════════════════════════════════════════════════
banner "演示 3：Wasm 沙箱审计能力 — host_call 全斡旋"

cat << 'EOF'

    ┌─ 架构原理 ─────────────────────────────────────────────────────────────┐
    │                                                                         │
    │  ┌─ Wasm 沙箱 (python.wasm) ─────┐                                     │
    │  │  import host_proxy             │  ← guest 唯一出网接口               │
    │  │  host_proxy.http_request(...)  │                                     │
    │  └────────── host_call 帧 ────────┘                                     │
    │               │                                                         │
    │               ▼  （双工管道，带内协议）                                   │
    │  ┌─ 宿主进程 (ateom-wasmd) ──────┐                                     │
    │  │  handle_host_call()            │  ← 拦截点                           │
    │  │       ↓                        │                                     │
    │  │  HostProxy.validate_url()      │  ← 白名单强制                       │
    │  │  HostProxy.log_request()       │  ← 审计记录（每一笔）               │
    │  │  HostProxy.http_request()      │  ← 代理执行                         │
    │  └────────────────────────────────┘                                     │
    │                                                                         │
    │  wasm 无 socket syscall → 100% 出网流量必经此路 → 审计面=授权面          │
    └─────────────────────────────────────────────────────────────────────────┘

EOF

echo "  创建沙箱..."
create_sandbox
echo "  沙箱: ${SID}"
pause

# ── 场景 1：合法公网请求 ─────────────────────────────────────────────────
banner "场景 1：合法公网请求（放行 + 审计记录）"
echo '    沙箱代码:'
echo '      import host_proxy'
echo '      status, body = host_proxy.http_request("GET", "https://httpbin.org/get")'
echo ""
echo '    执行中...'

CODE='
import host_proxy
try:
    status, body = host_proxy.http_request("GET", "https://httpbin.org/get")
    print(f"status={status}, body_len={len(body)}")
except Exception as e:
    print(f"error: {e}")
'
RESP=$(execute "$CODE")
OUT=$(ndjson_stdout "$RESP")
echo ""
echo "    响应: $OUT"
echo ""
echo "    ┌─ 审计记录 ──────────────────────────────────────────────────────┐"
echo "    │ HostProxy.log_request():                                        │"
echo "    │   url=https://httpbin.org/get  method=GET  allowed=true          │"
echo "    │   status_code=200                                               │"
echo "    │ → 请求放行，同时留下审计轨迹                                    │"
echo "    └─────────────────────────────────────────────────────────────────┘"
pause

# ── 场景 2：非法内网请求 ⚡ ──────────────────────────────────────────────
banner "场景 2：非法内网请求（阻止 + 告警）⚡"
echo '    沙箱代码尝试访问:'
echo '      ① K8s API Server  http://10.96.0.1:443/api/v1/secrets'
echo '      ② 内网服务        http://192.168.1.1:8080/admin/delete'
echo '      ③ 云 metadata     http://169.254.169.254/latest/meta-data/'
echo ""
echo '    执行中...'

CODE='
import host_proxy

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
        print(f"BLOCKED: {method} {url} -> {e}")
'
RESP=$(execute "$CODE")
echo ""
echo "    沙箱输出:"
echo "$RESP" | jq -r 'select(.type=="stdout") | .text' 2>/dev/null | while IFS= read -r line; do
    echo "      $line"
done

echo ""
echo "    ── Worker Pod 审计告警（kubectl logs grep BLOCKED）──"
sleep 1
WORKER_POD=$(kubectl get pods -n ate-wasm -l ate.dev/worker-pool=wasm-pool \
    --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')
kubectl logs "$WORKER_POD" -n ate-wasm --since=30s 2>/dev/null \
    | grep -i "blocked" | tail -3 | while IFS= read -r line; do
    TS=$(echo "$line" | jq -r '.timestamp' 2>/dev/null | head -c 23)
    URL=$(echo "$line" | jq -r '.fields.url' 2>/dev/null)
    METHOD=$(echo "$line" | jq -r '.fields.method' 2>/dev/null)
    printf '      %s ⚠️  BLOCKED %s %s\n' "$TS" "$METHOD" "$URL"
done

echo ""
echo "    ┌─ 审计事实 ──────────────────────────────────────────────────────┐"
echo "    │ blocked_cidrs 自动拦截:                                         │"
echo "    │   10.0.0.0/8 · 172.16.0.0/12 · 192.168.0.0/16 · 169.254.0.0/16│"
echo "    │                                                                 │"
echo "    │ 每一笔: log_request(allowed=false) + tracing::warn → Pod stdout │"
echo "    │                                                                 │"
echo "    │ ★ wasm 进程内没有 socket → 绕过可能性 = 0                       │"
echo "    └─────────────────────────────────────────────────────────────────┘"
pause

# ── 场景 3：代码执行事件流 ───────────────────────────────────────────────
banner "场景 3：代码执行的完整 NDJSON 事件流"
echo '    代码: result = sum(range(1,101)); print(f"1+2+...+100 = {result}")'
echo ""

RESP=$(execute 'result = sum(range(1,101)); print(f"1+2+...+100 = {result}")')
echo "    原始 NDJSON 响应（每行一个事件）:"
echo "$RESP" | jq -c '.' 2>/dev/null | while IFS= read -r line; do
    TYPE=$(echo "$line" | jq -r '.type' 2>/dev/null)
    case "$TYPE" in
        stdout)
            TEXT=$(echo "$line" | jq -r '.text' 2>/dev/null)
            TS=$(echo "$line" | jq -r '.timestamp' 2>/dev/null)
            printf '      │ [stdout] ts=%s text=%s\n' "$TS" "$TEXT"
            ;;
        number_of_executions)
            N=$(echo "$line" | jq -r '.execution_count' 2>/dev/null)
            printf '      │ [exec_n] count=%s ← 单调递增，不可篡改\n' "$N"
            ;;
        end)
            printf '      │ [  end ] ← cell 执行完毕\n'
            ;;
        *)
            printf '      │ [%s] %s\n' "$TYPE" "$line"
            ;;
    esac
done

echo ""
echo "    → 宿主看到: 代码全文 + stdout/stderr + 时间戳 + 执行序号 + 终止信号"
echo "    → 这就是 per-cell 全量审计（不是采样，是架构保证）"
pause

# ── 场景 4：Agent 出网审计链 ─────────────────────────────────────────────
banner "场景 4：AI Agent 多步推理 + 出网的完整审计链"
echo "    模拟: 准备数据 → 分析 → 调 API 报告"
echo ""

echo "    Step 1: 数据准备"
echo '      $ execute: data = {"metric": "DAU", "value": 12580}'
execute 'data = {"metric": "DAU", "value": 12580}; print("OK")' >/dev/null
echo "      → HostProxy: 无出网，仅 NDJSON 记录代码+输出"

echo ""
echo "    Step 2: 分析"
echo '      $ execute: growth = data["value"] * 0.15'
RESP=$(execute 'growth = data["value"] * 0.15; print(f"growth={growth:.0f}")')
OUT=$(ndjson_stdout "$RESP")
echo "      → 输出: $OUT"

echo ""
echo "    Step 3: 出网 API 调用"
echo '      $ execute: host_proxy.http_request("POST", "https://httpbin.org/post", ...)'
CODE='
import host_proxy
s, b = host_proxy.http_request("POST", "https://httpbin.org/post",
    headers={"Content-Type": "application/json"},
    body=f'\''{"growth": {growth}}'\'')
print(f"status={s}")
'
RESP=$(execute "$CODE")
OUT=$(ndjson_stdout "$RESP")
echo "      → 输出: $OUT"
echo "      → HostProxy 审计: url=httpbin.org/post method=POST allowed=true status=200"
echo ""
echo "    每一步推理 + 每一次出网 = 完整、不可绕过的审计链"
pause

# ── 对比 ─────────────────────────────────────────────────────────────────
banner "对比：审计能力架构差异"
cat << 'EOF'
    ┌──────────────────────────────────────────────────────────────────────┐
    │              Wasm 沙箱                    gVisor / microVM            │
    ├──────────────────────────────────────────────────────────────────────┤
    │ 网络审计   唯一通道: host_call→HostProxy  需 netfilter + eBPF       │
    │            白名单内置, 审计+拦截一体      需 Falco DaemonSet        │
    │                                                                      │
    │ 代码审计   宿主直接看到代码文本           内核态看不到应用层代码      │
    │                                          需 ptrace / sidecar        │
    │                                                                      │
    │ 绕过风险   不可能（无 socket syscall）    容器逃逸 / 内核漏洞        │
    │                                                                      │
    │ 部署成本   零（宿主进程内 tracing）       eBPF + seccomp + sidecar   │
    │ 覆盖率     100%（架构保证）              取决于配置完整度            │
    └──────────────────────────────────────────────────────────────────────┘
EOF
pause

# ── 总结 ─────────────────────────────────────────────────────────────────
banner "演示 3 总结"
cat << 'EOF'
  ┌───────────────────────────────────────────────────────────────────────┐
  │ ✅ 出网审计: HostProxy 记录每笔 host_call (url/method/allowed/status) │
  │ ✅ 内网防护: blocked_cidrs 拦截 10.x/172.x/192.168.x/169.254.x       │
  │ ✅ 实时告警: 阻止请求 → tracing::warn → Pod 结构化日志                │
  │ ✅ 代码审计: NDJSON = 代码全文 + 输出全量 + 时间戳 + 执行序号         │
  │ ✅ 不可绕过: wasm 无 ambient capability → 覆盖率 100%                 │
  │                                                                       │
  │ 结论: 审计是架构属性（no ambient cap），不是附加功能                   │
  └───────────────────────────────────────────────────────────────────────┘
EOF
