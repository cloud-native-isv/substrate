#!/usr/bin/env python3
"""签发 wasm-e2b-k8s Gateway JWT。

用法：
  sign_jwt.py <tenant_id>                        # 兼容旧用法：密钥读 .secrets/AUTH_SECRET
  sign_jwt.py --tenant user-a --secret <hex>     # 多租户：指定租户独立密钥
"""
import argparse, jwt, time, pathlib

p = argparse.ArgumentParser(description="签发 wasm-e2b-k8s Gateway JWT")
p.add_argument("tenant_pos", nargs="?", default=None,
               help="tenant_id（位置参数，兼容旧用法）")
p.add_argument("--tenant", default=None,
               help="tenant_id（优先于位置参数）")
p.add_argument("--secret", default=None,
               help="JWT HMAC 密钥（默认读 .secrets/AUTH_SECRET，单控制面时代的共享密钥）")
a = p.parse_args()

secret = a.secret
if not secret:
    secret = (pathlib.Path(__file__).parent.parent / ".secrets" / "AUTH_SECRET").read_text().strip()
tenant = a.tenant or a.tenant_pos or "user-A"
token = jwt.encode(
    {"tenant_id": tenant, "exp": int(time.time()) + 3600 * 24 * 7},
    secret,
    algorithm="HS256",
)
print(token)
