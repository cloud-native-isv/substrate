#!/usr/bin/env bash
# 生成本地 PoC 用的自签证书（两份）：
# 1) sslip 路径（裸机/本机全本地 demo）：覆盖 {api,<port>-<id>}.<IP>.sslip.io 全部子域。
# sslip.io 特性：任意前缀.<IP>.sslip.io 都解析到 <IP>（无需自建 DNS）。
# 2) sbx.test 路径（K8s + CLB 演示形态，多租户）：多 SAN 通配符证书，覆盖
# sbx.test / *.sbx.test（迁移窗口旧路径）+ 每租户 tenant-<t>.sbx.test /
# *.tenant-<t>.sbx.test。控制面 api.tenant-<t>.sbx.test 与数据面
# {port}-{id}.tenant-<t>.sbx.test 均为单级子域，由 *.tenant-<t>.sbx.test 覆盖。
# 同时把 certifi 公共 CA 与 sbx 证书合并为 .secrets/sbx-ca-bundle.pem
# （SDK 所在进程的 SSL_CERT_FILE 指向它：自签 gateway 与公网 LLM API 双向可信）。
#
# 用法：
# scripts/gen_sslip_cert.sh # 默认 IP=127.0.0.1，租户短名 a b
# scripts/gen_sslip_cert.sh 10.0.0.5 # 指定其他 IP（如需从别的机器访问）
# TENANTS="a b c" scripts/gen_sslip_cert.sh # 自定义租户短名列表（tenant-<t>）
#
# 产物：
# .secrets/sslip-cert.pem / sslip-key.pem sslip 路径证书（自签 CA，可直接作 CA 锚）
# .secrets/sbx-cert.pem / sbx-key.pem sbx.test 多 SAN 证书（→ K8s Secret sandbox-wildcard-tls）
# .secrets/sbx-ca-bundle.pem certifi + sbx-cert 合并 CA 包（SSL_CERT_FILE）
#
# 证书轮换后需刷新集群 Secret 并同步到各租户 ns（见脚本末尾输出）。
set -euo pipefail

IP="${1:-127.0.0.1}"
TENANTS="${TENANTS:-a b}"
OUT_DIR="$(cd "$(dirname "$0")/.." && pwd)/.secrets"
mkdir -p "$OUT_DIR"

# ── 1) sslip 路径（保持原行为）──
SSLIP_CERT="$OUT_DIR/sslip-cert.pem"
SSLIP_KEY="$OUT_DIR/sslip-key.pem"
SSLIP_CN="*.${IP}.sslip.io"
openssl req -x509 -newkey rsa:2048 -nodes -days 3650 \
 -keyout "$SSLIP_KEY" -out "$SSLIP_CERT" \
 -subj "/CN=${SSLIP_CN}" \
 -addext "subjectAltName=DNS:*.${IP}.sslip.io,DNS:${IP}.sslip.io" \
 -addext "basicConstraints=critical,CA:TRUE" \
 -addext "keyUsage=critical,digitalSignature,keyCertSign" 2>/dev/null

# ── 2) sbx.test 多 SAN 证书（每租户独立接入面）──
SBX_CERT="$OUT_DIR/sbx-cert.pem"
SBX_KEY="$OUT_DIR/sbx-key.pem"
SAN="DNS:sbx.test,DNS:*.sbx.test"
for t in $TENANTS; do
 SAN="${SAN},DNS:tenant-${t}.sbx.test,DNS:*.tenant-${t}.sbx.test"
done
openssl req -x509 -newkey rsa:2048 -nodes -days 3650 \
 -keyout "$SBX_KEY" -out "$SBX_CERT" \
 -subj "/CN=*.sbx.test" \
 -addext "subjectAltName=${SAN}" \
 -addext "basicConstraints=critical,CA:TRUE" \
 -addext "keyUsage=critical,digitalSignature,keyCertSign" 2>/dev/null

# ── 3) 合并 CA 包（certifi 公共 CA + sbx 自签证书）──
if CERTIFI=$(python3 -c 'import certifi; print(certifi.where())' 2>/dev/null); then
 cat "$CERTIFI" "$SBX_CERT" > "$OUT_DIR/sbx-ca-bundle.pem"
 echo "ca bundle : $OUT_DIR/sbx-ca-bundle.pem (certifi + sbx-cert)"
else
 echo "WARN: 未找到 certifi，sbx-ca-bundle.pem 未更新（可手动 cat <certifi> sbx-cert.pem 合成）" >&2
fi

echo "sslip cert: $SSLIP_CERT (CN $SSLIP_CN)"
echo "sbx cert: $SBX_CERT"
echo "sbx SAN : ${SAN}"
echo ""
echo "集群 Secret 刷新（轮换后执行；各租户 ns 由 scripts/provision_tenant.sh 复制）："
echo " kubectl -n default create secret tls sandbox-wildcard-tls \\"
echo " --cert=$SBX_CERT --key=$SBX_KEY --dry-run=client -o yaml | kubectl apply -f -"
