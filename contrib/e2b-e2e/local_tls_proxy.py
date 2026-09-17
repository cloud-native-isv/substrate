#!/usr/bin/env python3
"""Local TLS-terminating TCP proxy for E2B SDK e2e (zero cluster changes).

Listens :8443 with a locally-issued 397-day cert (macOS-acceptable), terminates
TLS, and forwards raw bytes to 127.0.0.1:8080 (the existing e2bgw port-forward,
HTTP). Pure L4 splice — Host header / NDJSON streaming pass through untouched.
Does NOT touch the cluster; e2b-tls-relay stays running.
"""
import asyncio, ssl, sys, os

CERT = os.path.join(os.path.dirname(os.path.abspath(__file__)), ".secrets", "local-relay-cert.pem")
KEY  = os.path.join(os.path.dirname(os.path.abspath(__file__)), ".secrets", "local-relay-key.pem")
LISTEN_HOST, LISTEN_PORT = "127.0.0.1", 8443
UPSTREAM_HOST, UPSTREAM_PORT = "127.0.0.1", 8080

async def pipe(r, w):
    try:
        while True:
            data = await r.read(65536)
            if not data:
                break
            w.write(data)
            await w.drain()
    except Exception:
        pass
    finally:
        try: w.close()
        except Exception: pass

async def handle(client_r, client_w):
    try:
        up_r, up_w = await asyncio.open_connection(UPSTREAM_HOST, UPSTREAM_PORT)
    except Exception as e:
        client_w.close(); return
    await asyncio.gather(pipe(client_r, up_w), pipe(up_r, client_w))

async def main():
    ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    ctx.load_cert_chain(CERT, KEY)
    server = await asyncio.start_server(handle, LISTEN_HOST, LISTEN_PORT, ssl=ctx)
    print(f"[tls-proxy] listening https://{LISTEN_HOST}:{LISTEN_PORT} -> {UPSTREAM_HOST}:{UPSTREAM_PORT}", flush=True)
    async with server:
        await server.serve_forever()

if __name__ == "__main__":
    asyncio.run(main())
