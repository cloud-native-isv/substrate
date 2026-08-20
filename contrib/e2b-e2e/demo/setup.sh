#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════
#  setup.sh — 演示前置环境准备（一次性运行）
# ═══════════════════════════════════════════════════════════════════════════
#  做四件事：
#    1. 检查 kubectl 连通性
#    2. 启动 port-forward（e2b-tls-relay:8443 + e2bgw:8080）
#    3. 签发 JWT API key
#    4. 导出环境变量到 .env 文件供演示脚本 source
#
#  用法：
#    cd contrib/e2b-e2e/demo
#    bash setup.sh
#    # 成功后会生成 .env 文件，后续演示脚本自动 source 它
# ═══════════════════════════════════════════════════════════════════════════
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
E2E_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ENV_FILE="$SCRIPT_DIR/.env"

echo "═══════════════════════════════════════════════════════════════"
echo "  Wasm 沙箱演示 — 环境准备"
echo "═══════════════════════════════════════════════════════════════"

# ── 0. 检查依赖工具 ─────────────────────────────────────────────────────
echo ""
echo "▸ [0/4] 检查工具依赖..."
for cmd in jq bc perl curl kubectl; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "  ❌ 缺少命令: $cmd（请先安装）"
        exit 1
    fi
done
echo "  ✅ jq, bc, perl, curl, kubectl 就绪"

# ── 1. 检查 kubectl ─────────────────────────────────────────────────────
echo ""
echo "▸ [1/4] 检查集群连通性..."
if ! kubectl cluster-info >/dev/null 2>&1; then
    echo "  ❌ kubectl 无法连接集群。请检查 kubeconfig。"
    exit 1
fi
CONTEXT=$(kubectl config current-context 2>/dev/null)
echo "  ✅ 集群连通 (context: ${CONTEXT:0:40})"

# 确认 worker Pod 在跑
WORKER_COUNT=$(kubectl get pods -n ate-wasm -l ate.dev/worker-pool=wasm-pool \
  --field-selector=status.phase=Running -o name 2>/dev/null | wc -l | tr -d ' ')
if [ "$WORKER_COUNT" -eq 0 ]; then
    echo "  ❌ ate-wasm 命名空间无 Running 的 worker Pod。请确认镜像已部署。"
    exit 1
fi
echo "  ✅ wasm-pool worker: $WORKER_COUNT 个 Running"

# ── 2. 启动 port-forward ────────────────────────────────────────────────
echo ""
echo "▸ [2/4] 确保 port-forward 运行中..."

# TLS relay (8443)
if lsof -i :8443 -sTCP:LISTEN >/dev/null 2>&1; then
    echo "  ✅ e2b-tls-relay :8443 已在监听"
else
    echo "  → 启动 kubectl port-forward svc/e2b-tls-relay 8443:443 ..."
    kubectl -n ate-system port-forward svc/e2b-tls-relay 8443:443 >/dev/null 2>&1 &
    sleep 2
    if lsof -i :8443 -sTCP:LISTEN >/dev/null 2>&1; then
        echo "  ✅ e2b-tls-relay :8443 启动成功"
    else
        echo "  ❌ 无法启动 TLS relay port-forward"
        exit 1
    fi
fi

# e2bgw HTTP (8080)
if lsof -i :8080 -sTCP:LISTEN >/dev/null 2>&1; then
    echo "  ✅ e2bgw :8080 已在监听"
else
    echo "  → 启动 kubectl port-forward svc/e2bgw 8080:80 ..."
    kubectl -n ate-system port-forward svc/e2bgw 8080:80 >/dev/null 2>&1 &
    sleep 2
    if lsof -i :8080 -sTCP:LISTEN >/dev/null 2>&1; then
        echo "  ✅ e2bgw :8080 启动成功"
    else
        echo "  ❌ 无法启动 e2bgw port-forward"
        exit 1
    fi
fi

# ── 3. 签发 JWT ─────────────────────────────────────────────────────────
echo ""
echo "▸ [3/4] 签发 E2B API key (JWT)..."

JWT_SECRET=$(kubectl get secret e2bgw-jwt-secret -n ate-system \
    -o jsonpath='{.data.secret}' 2>/dev/null | base64 -d)
if [ -z "$JWT_SECRET" ]; then
    echo "  ❌ 无法获取 e2bgw-jwt-secret。请确认 ate-system 命名空间。"
    exit 1
fi

API_KEY=$(python3 "$E2E_DIR/sign_jwt.py" --atespace team-a --secret "$JWT_SECRET" 2>/dev/null)
if [ -z "$API_KEY" ]; then
    echo "  ❌ JWT 签发失败。请确认 PyJWT 已安装: pip3 install pyjwt"
    exit 1
fi
echo "  ✅ API key 已签发 (${API_KEY:0:25}...)"

# ── 4. 检查 TLS 证书 + 生成 .env ────────────────────────────────────────
echo ""
echo "▸ [4/4] 检查 TLS 证书并生成 .env ..."

CERT_FILE="$E2E_DIR/.secrets/sslip-cert.pem"
if [ ! -f "$CERT_FILE" ]; then
    echo "  → 从集群导出 TLS 证书..."
    mkdir -p "$E2E_DIR/.secrets"
    kubectl get secret e2b-tls-cert -n ate-system \
        -o jsonpath='{.data.tls\.crt}' | base64 -d > "$CERT_FILE"
fi
echo "  ✅ 证书: $CERT_FILE"

# 连通性验证
HTTP_CODE=$(curl -sk "https://127.0.0.1:8443/" --cacert "$CERT_FILE" \
    -o /dev/null -w "%{http_code}" 2>/dev/null || echo "000")
if [ "$HTTP_CODE" = "000" ]; then
    echo "  ❌ TLS 连通性测试失败"
    exit 1
fi
echo "  ✅ TLS 连通验证通过 (HTTP $HTTP_CODE)"

# 写 .env
cat > "$ENV_FILE" << EOF
# 自动生成 — $(date '+%Y-%m-%d %H:%M:%S')
export E2B_DOMAIN="127.0.0.1.sslip.io:8443"
export E2B_API_KEY="$API_KEY"
export E2B_VALIDATE_API_KEY="false"
export SSL_CERT_FILE="$CERT_FILE"
export E2B_TEMPLATE="python-interpreter"
EOF

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "  ✅ 环境准备完成！"
echo ""
echo "  生成的配置: $ENV_FILE"
echo ""
echo "  接下来按顺序运行演示："
echo "    cd $(basename "$SCRIPT_DIR")"
echo "    bash demo_1_feasibility.sh     # 可行性"
echo "    bash demo_2_performance.sh     # 性能"
echo "    bash demo_3_audit_trail.sh     # 审计"
echo "═══════════════════════════════════════════════════════════════"
