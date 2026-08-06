#!/usr/bin/env python3
"""TLS→TCP 字节转发（本地 PoC 端到端用）。

终结 :443 的 TLS，把解密后的明文字节原样双向转给 gateway（默认 127.0.0.1:8080，
即 `kubectl port-forward svc/gateway 8080:8080` 暴露的端口）。

为什么用字节转发而不是 HTTP 反向代理：
  - 不解析 HTTP，Host 头、chunked/NDJSON 流式响应都天然透传，零改写；
  - 不设置 ALPN，TLS 协商结果为 None，httpx 自动回落到 HTTP/1.1，
    与 gateway(axum, HTTP/1.1) 对齐，避免 h2c 不匹配。

E2B SDK 非 debug 下：
  控制面 https://api.<domain>          → :443 → 本转发 → gateway
  数据面 https://<port>-<id>.<domain>  → :443 → 本转发 → gateway（gateway 按 Host + X-Access-Token 定位 sandbox）

环境变量（均可选，有默认值）：
  RELAY_LISTEN         监听地址，默认 0.0.0.0
  RELAY_PORT           监听端口，默认 443（<1024 需 root/sudo）
  RELAY_UPSTREAM_HOST  上游地址，默认 127.0.0.1
  RELAY_UPSTREAM_PORT  上游端口，默认 8080
  RELAY_CERT           证书 PEM，默认 .secrets/sslip-cert.pem
  RELAY_KEY            私钥 PEM，默认 .secrets/sslip-key.pem
"""
import os
import socket
import ssl
import threading

LISTEN_HOST = os.environ.get("RELAY_LISTEN", "0.0.0.0")
LISTEN_PORT = int(os.environ.get("RELAY_PORT", "443"))
UPSTREAM_HOST = os.environ.get("RELAY_UPSTREAM_HOST", "127.0.0.1")
UPSTREAM_PORT = int(os.environ.get("RELAY_UPSTREAM_PORT", "8080"))
CERT = os.environ.get("RELAY_CERT", ".secrets/sslip-cert.pem")
KEY = os.environ.get("RELAY_KEY", ".secrets/sslip-key.pem")


def _pipe(src: socket.socket, dst: socket.socket) -> None:
    try:
        while True:
            data = src.recv(65536)
            if not data:
                break
            dst.sendall(data)
    except OSError:
        pass
    finally:
        try:
            dst.shutdown(socket.SHUT_WR)
        except OSError:
            pass


def _handle(tls_conn: socket.socket) -> None:
    try:
        up = socket.create_connection((UPSTREAM_HOST, UPSTREAM_PORT))
    except OSError as e:
        print(f"[relay] upstream connect failed: {e}", flush=True)
        tls_conn.close()
        return
    t1 = threading.Thread(target=_pipe, args=(tls_conn, up), daemon=True)
    t2 = threading.Thread(target=_pipe, args=(up, tls_conn), daemon=True)
    t1.start()
    t2.start()
    t1.join()
    t2.join()
    for s in (tls_conn, up):
        try:
            s.close()
        except OSError:
            pass


def main() -> None:
    ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    ctx.load_cert_chain(certfile=CERT, keyfile=KEY)
    # 故意不调用 set_alpn_protocols：协商结果 None → httpx 回落 HTTP/1.1

    srv = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    srv.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    srv.bind((LISTEN_HOST, LISTEN_PORT))
    srv.listen(128)
    print(
        f"[relay] TLS :{LISTEN_PORT} → {UPSTREAM_HOST}:{UPSTREAM_PORT}  "
        f"cert={CERT}",
        flush=True,
    )
    try:
        while True:
            raw, _addr = srv.accept()
            try:
                tls = ctx.wrap_socket(raw, server_side=True)
            except ssl.SSLError as e:
                print(f"[relay] TLS handshake failed: {e}", flush=True)
                raw.close()
                continue
            threading.Thread(target=_handle, args=(tls,), daemon=True).start()
    except KeyboardInterrupt:
        print("\n[relay] shutting down", flush=True)
    finally:
        srv.close()


if __name__ == "__main__":
    main()
