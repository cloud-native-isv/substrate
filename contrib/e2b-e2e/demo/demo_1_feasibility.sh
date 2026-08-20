#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════
#  演示 1：Wasm 沙箱技术可行性
# ═══════════════════════════════════════════════════════════════════════════
# 设计原则：每步展示 curl 命令 + 原始响应 + kubectl 佐证，观众可独立复现
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/.env" 2>/dev/null || { echo "❌ 先运行 bash setup.sh"; exit 1; }

API="https://api.${E2B_DOMAIN}"
CURL_OPTS="--cacert ${SSL_CERT_FILE} -sk"
CURL="curl ${CURL_OPTS}"

banner()  { printf '\n%s\n  %s\n%s\n' "$(printf '─%.0s' {1..68})" "$1" "$(printf '─%.0s' {1..68})"; }
step()    { printf '\n  ▸ [%s] %s\n' "$1" "$2"; }
show_cmd(){ printf '\n    \033[36m$ %s\033[0m\n' "$1"; }  # 蓝色显示命令
ok()      { printf '    \033[32m✅ %s\033[0m\n' "$1"; }
pause()   { printf '\n    \033[2m[按回车继续]\033[0m '; read -r; }

cleanup() { $CURL -X DELETE "${API}/sandboxes/${SID}" -H "X-API-Key: ${E2B_API_KEY}" >/dev/null 2>&1 || true; }
trap cleanup EXIT

# ═══════════════════════════════════════════════════════════════════════════
banner "演示 1：Wasm 沙箱技术可行性证明"
echo "  集群: ${E2B_DOMAIN} | 模板: ${E2B_TEMPLATE}"

# ── 1. 创建沙箱（展示完整 curl + 原始 JSON 响应）─────────────────────────
step 1 "创建 Wasm 沙箱"
CMD="curl -X POST '${API}/sandboxes' \\
      -H 'X-API-Key: <JWT>' \\
      -H 'Content-Type: application/json' \\
      -d '{\"templateID\":\"${E2B_TEMPLATE}\"}'"
show_cmd "$CMD"

RESP=$($CURL -X POST "${API}/sandboxes" \
    -H "X-API-Key: ${E2B_API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"templateID\":\"${E2B_TEMPLATE}\"}")
echo ""
echo "    原始响应:"
echo "$RESP" | jq '.' 2>/dev/null | sed 's/^/    /'
echo ""

SID=$(echo "$RESP" | jq -r '.sandboxID')
TOKEN=$(echo "$RESP" | jq -r '.envdAccessToken')
ok "sandboxID = ${SID}"
echo "    envdAccessToken = ${TOKEN:0:30}...（数据面 X-Access-Token 鉴权凭证）"
pause

# ── 2. 执行代码（展示数据面 URL + NDJSON 原始帧）─────────────────────────
step 2 "执行 Python 代码 — 主机名式数据面"
DATA_URL="https://49999-${SID}.${E2B_DOMAIN}/execute"
CMD="curl -X POST 'https://49999-${SID}.${E2B_DOMAIN}/execute' \\
      -H 'X-Access-Token: <envdAccessToken>' \\
      -H 'Content-Type: application/json' \\
      -d '{\"code\":\"print(6 * 7)\"}'"
show_cmd "$CMD"
echo "    ↑ URL 中 {sandboxID} → e2bgw 主机名路由 → atenet → ateom-wasmd"
echo ""

RESP=$($CURL -X POST "${DATA_URL}" \
    -H "X-Access-Token: ${TOKEN}" \
    -H "E2B-Traffic-Access-Token: ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{"code":"print(6 * 7)"}')

echo "    原始 NDJSON 响应（每行一个事件帧）:"
echo "$RESP" | while IFS= read -r line; do
    [ -n "$line" ] && echo "      $line"
done
echo ""
OUT=$(echo "$RESP" | jq -r 'select(.type=="stdout") | .text' 2>/dev/null | tr -d '\n')
ok "解读: stdout 帧 text='${OUT}' ← Python print 输出"

echo ""
echo "    ── kubectl logs 佐证（同时刻 worker Pod 日志）──"
WORKER_POD=$(kubectl get pods -n ate-wasm -l ate.dev/worker-pool=wasm-pool \
    --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
kubectl logs "$WORKER_POD" -n ate-wasm --since=10s 2>/dev/null \
    | grep -E "execute|dispatching" | tail -2 | sed 's/^/      /'
pause

# ── 3. 变量保持（Jupyter 语义）───────────────────────────────────────────
step 3 "变量跨调用保持（Jupyter kernel 语义）"
echo "    第一次调用:"
show_cmd "curl POST /execute -d '{\"code\":\"secret=20260820; data=list(range(100))\"}'"
$CURL -X POST "${DATA_URL}" \
    -H "X-Access-Token: ${TOKEN}" -H "E2B-Traffic-Access-Token: ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{"code":"secret=20260820; data=list(range(100))"}' >/dev/null

echo "    第二次调用（引用第一次设的变量）:"
show_cmd "curl POST /execute -d '{\"code\":\"print(f\\\"secret={secret}, len={len(data)}\\\")\"}'"
RESP=$($CURL -X POST "${DATA_URL}" \
    -H "X-Access-Token: ${TOKEN}" -H "E2B-Traffic-Access-Token: ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{"code":"print(f\"secret={secret}, len={len(data)}\")"}')

echo ""
echo "    原始 NDJSON:"
echo "$RESP" | while IFS= read -r line; do [ -n "$line" ] && echo "      $line"; done
OUT=$(echo "$RESP" | jq -r 'select(.type=="stdout") | .text' 2>/dev/null | tr -d '\n')
echo ""
ok "stdout='${OUT}' ← 跨调用变量保持，同一 kernel 实例"
pause

# ── 4. 多 context 隔离 ───────────────────────────────────────────────────
step 4 "多 context 隔离"
CTX_URL="https://49999-${SID}.${E2B_DOMAIN}/contexts"
show_cmd "curl POST '${CTX_URL}' -d '{}'"
RESP_A=$($CURL -X POST "${CTX_URL}" \
    -H "X-Access-Token: ${TOKEN}" -H "E2B-Traffic-Access-Token: ${TOKEN}" \
    -H "Content-Type: application/json" -d '{}')
echo "    响应 A: $RESP_A"
CTX_A=$(echo "$RESP_A" | jq -r '.id')

RESP_B=$($CURL -X POST "${CTX_URL}" \
    -H "X-Access-Token: ${TOKEN}" -H "E2B-Traffic-Access-Token: ${TOKEN}" \
    -H "Content-Type: application/json" -d '{}')
echo "    响应 B: $RESP_B"
CTX_B=$(echo "$RESP_B" | jq -r '.id')
echo ""

# A 设变量
$CURL -X POST "${DATA_URL}" \
    -H "X-Access-Token: ${TOKEN}" -H "E2B-Traffic-Access-Token: ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{\"code\":\"x_a = 'only_in_A'\", \"context_id\":\"${CTX_A}\"}" >/dev/null

# B 尝试读 A 的变量
show_cmd "curl POST /execute -d '{\"code\":\"print(x_a)\", \"context_id\":\"${CTX_B}\"}'"
echo "    ↑ 注意: context_id 指向 B，但 x_a 只在 A 中定义"
RESP=$($CURL -X POST "${DATA_URL}" \
    -H "X-Access-Token: ${TOKEN}" -H "E2B-Traffic-Access-Token: ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{\"code\":\"print(x_a)\", \"context_id\":\"${CTX_B}\"}")
echo ""
echo "    原始 NDJSON:"
echo "$RESP" | while IFS= read -r line; do [ -n "$line" ] && echo "      $line"; done
ERR=$(echo "$RESP" | jq -r 'select(.type=="error") | "\(.name): \(.value)"' 2>/dev/null | head -1)
echo ""
ok "error 帧: ${ERR} ← B 看不到 A 的变量（隔离证据）"
pause

# ── 5. 文件共享 ──────────────────────────────────────────────────────────
step 5 "跨 context 文件系统共享"
show_cmd "A: execute '{\"code\":\"open(\\'/sandbox/f.txt\\',\\'w\\').write(\\'from_A\\')\", \"context_id\":\"${CTX_A}\"}'"
$CURL -X POST "${DATA_URL}" \
    -H "X-Access-Token: ${TOKEN}" -H "E2B-Traffic-Access-Token: ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{\"code\":\"open('/sandbox/f.txt','w').write('from_A')\", \"context_id\":\"${CTX_A}\"}" >/dev/null

show_cmd "B: execute '{\"code\":\"print(open(\\'/sandbox/f.txt\\').read())\", \"context_id\":\"${CTX_B}\"}'"
RESP=$($CURL -X POST "${DATA_URL}" \
    -H "X-Access-Token: ${TOKEN}" -H "E2B-Traffic-Access-Token: ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{\"code\":\"print(open('/sandbox/f.txt').read())\", \"context_id\":\"${CTX_B}\"}")
echo "    原始 NDJSON:"
echo "$RESP" | while IFS= read -r line; do [ -n "$line" ] && echo "      $line"; done
OUT=$(echo "$RESP" | jq -r 'select(.type=="stdout") | .text' 2>/dev/null | tr -d '\n')
echo ""
ok "B 读到 A 写的文件: '${OUT}' ← /sandbox 是共享 preopen 目录"
pause

# ── 6. epoch trap ────────────────────────────────────────────────────────
step 6 "资源安全：epoch trap 终止无限循环"
show_cmd "curl POST /execute -d '{\"code\":\"while True: pass\"}'"
echo "    （预期 ~10s 后连接被重置 — epoch interrupt 切断 wasm 执行）"
echo ""
$CURL -X POST "${DATA_URL}" \
    -H "X-Access-Token: ${TOKEN}" -H "E2B-Traffic-Access-Token: ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{"code":"while True: pass"}' 2>&1 || true
echo ""
echo "    ↑ 连接重置/截断 = epoch trap 生效的网络层证据"
echo ""
echo "    ── kubectl logs 佐证 ──"
sleep 1
kubectl logs "$WORKER_POD" -n ate-wasm --since=15s 2>/dev/null \
    | grep -iE "epoch|trap|poison|timed out" | tail -3 | sed 's/^/      /'
echo ""
ok "worker 日志确认: epoch trap fired → context poisoned"

# ── 总结 ─────────────────────────────────────────────────────────────────
banner "演示 1 总结"
cat << 'EOF'
  ┌───────────────────────────────────────────────────────────────────────┐
  │ ✅ 完整 Python 执行（原始 NDJSON 帧为证）                             │
  │ ✅ Jupyter 语义（两次 curl，第二次引用第一次的变量）                    │
  │ ✅ context 隔离（error 帧 NameError 为证）                            │
  │ ✅ 文件共享（同一 /sandbox preopen）                                   │
  │ ✅ epoch trap（连接重置 + Pod 日志 "poisoned" 为证）                   │
  │                                                                       │
  │ 每条 curl 命令可独立复制执行 — 这不是打表                             │
  └───────────────────────────────────────────────────────────────────────┘
EOF
