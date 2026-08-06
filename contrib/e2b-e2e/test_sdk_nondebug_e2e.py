#!/usr/bin/env python3
"""未改动 E2B SDK 的「非 debug」端到端验证。

证明目标：官方 e2b-code-interpreter SDK 在**非 debug（生产形态）**下，经由
  控制面  https://api.<domain>              （X-API-Key = 租户 JWT）
  数据面  https://<port>-<id>.<domain>      （X-Access-Token = create 返回的 envdAccessToken）
全程 HTTPS + 生产式主机名，驱动真实 wasm kernel 执行代码。

本脚本不改 SDK 一行，全部通过环境变量注入：
  E2B_DOMAIN            例：127.0.0.1.sslip.io（sslip.io 把 *.127.0.0.1.sslip.io 解析到本机）
  E2B_API_KEY           租户 JWT（scripts/sign_jwt.py 用 AUTH_SECRET 签发）
  E2B_VALIDATE_API_KEY  必须 false（JWT 不符合 e2b_ 前缀，需关闭 SDK 客户端格式校验）
  SSL_CERT_FILE         自签通配符证书 PEM（httpx verify=True/trust_env=True 会读它作 CA）
  E2B_DEBUG             必须 false / 不设

前置：TLS 转发监听 :443 → gateway；gateway 已由 kubectl port-forward 暴露到 127.0.0.1:8080。
"""
import os
import sys

from e2b_code_interpreter import Sandbox

JUPYTER_PORT = 49999


def _require_env() -> str:
    dom = os.environ.get("E2B_DOMAIN")
    if not dom:
        sys.exit("[e2e] 缺少 E2B_DOMAIN")
    if os.environ.get("E2B_DEBUG", "false").lower() == "true":
        sys.exit("[e2e] E2B_DEBUG 必须为 false（本测试验证非 debug 生产形态）")
    if not os.environ.get("E2B_API_KEY"):
        sys.exit("[e2e] 缺少 E2B_API_KEY（租户 JWT）")
    if not os.environ.get("SSL_CERT_FILE"):
        sys.exit("[e2e] 缺少 SSL_CERT_FILE（自签 CA），否则 httpx 会拒绝自签证书")
    return dom


def main() -> None:
    dom = _require_env()
    print(f"[e2e] E2B_DOMAIN      = {dom}")
    print(f"[e2e] SSL_CERT_FILE   = {os.environ.get('SSL_CERT_FILE')}")
    print(f"[e2e] validate_key    = {os.environ.get('E2B_VALIDATE_API_KEY')}")

    print("[e2e] 控制面 create（https://api.%s） ..." % dom)
    sbx = Sandbox.create(timeout=120)
    try:
        sid = sbx.sandbox_id
        host = sbx.get_host(JUPYTER_PORT)
        url = sbx._jupyter_url
        print(f"[e2e] sandbox_id     = {sid}")
        print(f"[e2e] kernel host    = {host}")
        print(f"[e2e] kernel url     = {url}")

        assert url.startswith("https://"), "非 debug 下必须 https"
        assert "localhost" not in host, "host 不应是 debug 的 localhost"
        assert dom in host and sid in host, "host 必须是生产式 {port}-{id}.{domain}"

        # 1) 基本执行：证明数据面 https run_code 真的打到 wasm kernel
        ex = sbx.run_code("print(6 * 7)")
        if ex.error:
            sys.exit(f"[e2e] run_code 报错：{ex.error}")
        out = "".join(ex.logs.stdout).strip()
        print(f"[e2e] run_code print(6*7)      -> stdout={out!r}")
        assert out == "42", f"期望 42，实际 {out!r}"

        # 2) 同 kernel 状态保持：证明同一 sandbox 多次 execute 共享 Python 全局
        sbx.run_code("_secret = 20260716")
        ex2 = sbx.run_code("print(_secret)")
        out2 = "".join(ex2.logs.stdout).strip()
        print(f"[e2e] same-kernel print(_secret) -> stdout={out2!r}")
        assert out2 == "20260716", f"kernel 未保持状态：{out2!r}"

        print(
            "\n[e2e] PASS ✅ 未改动 SDK 非 debug 端到端跑通："
            "控制面 https create + 数据面 https run_code + 生产式 host + X-Access-Token 鉴权"
        )
    finally:
        sbx.kill()
        print("[e2e] sandbox killed")


if __name__ == "__main__":
    main()
