#!/usr/bin/env python3
"""签发 e2bgw API key（HS256 JWT）。

e2bgw（fork `cmd/e2bgw`）从 X-API-Key / Authorization: Bearer 读取该 JWT，
取 `atespace` claim（兼容旧 `tenant`）作为调用者 Atespace。
密钥必须与网关 --jwt-secret-file 一致（集群内为 Secret `e2bgw-jwt-secret`，
见 manifests/xuanji/e2bgw.yaml 头注释）。

用法：
  sign_jwt.py <atespace>                          # 兼容旧用法：密钥读 .secrets/AUTH_SECRET
  sign_jwt.py --atespace team-a --secret <hex>    # 指定 atespace 与密钥
"""
import argparse, jwt, time, pathlib

p = argparse.ArgumentParser(description="签发 e2bgw API key（HS256 JWT，atespace claim）")
p.add_argument("atespace_pos", nargs="?", default=None,
               help="atespace（位置参数，兼容旧用法）")
p.add_argument("--atespace", "--tenant", dest="atespace", default=None,
               help="atespace（优先于位置参数；--tenant 为旧别名）")
p.add_argument("--secret", default=None,
               help="JWT HMAC 密钥（默认读 .secrets/AUTH_SECRET；须与 e2bgw --jwt-secret-file 一致）")
a = p.parse_args()

secret = a.secret
if not secret:
    secret = (pathlib.Path(__file__).parent.parent / ".secrets" / "AUTH_SECRET").read_text().strip()
atespace = a.atespace or a.atespace_pos or "user-a"
token = jwt.encode(
    {"atespace": atespace, "exp": int(time.time()) + 3600 * 24 * 7},
    secret,
    algorithm="HS256",
)
print(token)
