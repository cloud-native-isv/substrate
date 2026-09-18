#!/usr/bin/env python3
"""POC driver: validate two-tier design in simplest form (runc + cpython-wasm + simple python agent).

Path-based E2B data plane via e2bgw (port-forwarded to 127.0.0.1:8080).
Self-contained: mints the HS256 JWT from the in-cluster e2bgw-jwt-secret (read via
kubectl, never printed), creates a sandbox on the runc wasm pool, then runs:
  step 1 smoke (print)         -> data plane alive
  step 2 host_proxy egress     -> wasm host-mediated capability path
  step 3 simple python agent   -> ReAct loop + tool dispatch + real egress tool
"""
import base64, hashlib, hmac, json, os, subprocess, sys, time, urllib.request, urllib.error

GW = os.environ.get("E2BGW_URL", "http://127.0.0.1:8080")
KUBECONFIG = os.environ.get("KUBECONFIG", "/Users/liuqiming.lqm/project/profiles/config/ack/cluster/cluster-msaFE8/kube/config.yaml")
TEMPLATE = os.environ.get("POC_TEMPLATE", "python-interpreter")
ATESPACES = [a for a in os.environ.get("POC_ATESPACES", "tenant-a,team-a,user-a,ate-wasm").split(",") if a]

def log(*a):
    print(*a, flush=True)

def kubectl(*args):
    env = dict(os.environ, KUBECONFIG=KUBECONFIG)
    return subprocess.run(["kubectl", *args], capture_output=True, text=True, env=env)

def mint_jwt(atespace, secret_b):
    def b64(x): return base64.urlsafe_b64encode(x).rstrip(b"=")
    h = b64(json.dumps({"alg": "HS256", "typ": "JWT"}, separators=(",", ":")).encode())
    p = b64(json.dumps({"atespace": atespace, "exp": int(time.time()) + 3600}, separators=(",", ":")).encode())
    sig = b64(hmac.new(secret_b, h + b"." + p, hashlib.sha256).digest())
    return (h + b"." + p + b"." + sig).decode()

def get_secret():
    r = kubectl("get", "secret", "e2bgw-jwt-secret", "-n", "ate-system", "-o", "jsonpath={.data.secret}")
    if r.returncode != 0 or not r.stdout.strip():
        log("[FATAL] cannot read e2bgw-jwt-secret:", r.stderr.strip()[:200]); sys.exit(2)
    return base64.b64decode(r.stdout.strip())

def http(method, path, body=None, token=None, raw=False, timeout=180):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(GW + path, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    if token: req.add_header("X-API-Key", token)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status, resp.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", "replace")
    except Exception as e:
        return -1, f"{type(e).__name__}: {e}"

def create_sandbox(token):
    for body in ({"templateID": TEMPLATE, "timeout": 300}, {"template": TEMPLATE, "timeout": 300}):
        st, txt = http("POST", "/sandboxes", body, token)
        if st in (200, 201):
            try:
                d = json.loads(txt)
                sid = d.get("sandboxID") or d.get("sandboxId") or d.get("id") or (d.get("sandbox") or {}).get("id")
                if sid: return sid, d
            except Exception: pass
        log(f"  create attempt {list(body)[0]} -> HTTP {st}: {txt[:200]}")
    return None, None

def execute(token, sid, code, label):
    st, txt = http("POST", f"/sandboxes/{sid}/execute", {"code": code}, token)
    log(f"\n===== {label} (HTTP {st}) =====")
    frames = []
    for line in txt.splitlines():
        line = line.strip()
        if not line: continue
        try: frames.append(json.loads(line))
        except Exception: frames.append({"_raw": line})
    for f in frames:
        log("  frame:", json.dumps(f, ensure_ascii=False)[:4000])
    return st, frames

def main():
    secret_b = get_secret()
    log(f"[ok] read e2bgw-jwt-secret ({len(secret_b)} bytes, value withheld)")
    sid = token = None
    for at in ATESPACES:
        token = mint_jwt(at, secret_b)
        log(f"\n[try] atespace={at}")
        sid, d = create_sandbox(token)
        if sid:
            log(f"[ok] sandbox created: id={sid} atespace={at}")
            log("     create resp:", json.dumps(d, ensure_ascii=False)[:300])
            break
    if not sid:
        log("[FATAL] could not create sandbox on any atespace"); sys.exit(3)

    execute(token, sid, "print('SMOKE_OK', 6*7)", "STEP 1 smoke (data plane alive)")

    probe = (
        "import json\n"
        "try:\n"
        "    import host_proxy\n"
        "    r = host_proxy.http_request('GET','https://www.aliyun.com',None,None)\n"
        "    print('HOSTPROXY_OK', str(r)[:200])\n"
        "except Exception as e:\n"
        "    print('HOSTPROXY_ERR', type(e).__name__, str(e)[:200])\n"
    )
    execute(token, sid, probe, "STEP 2 host_proxy egress (wasm host-mediated capability)")

    agent = open(os.path.join(os.path.dirname(__file__), "poc_two_tier_agent_code.py")).read()
    execute(token, sid, agent, "STEP 3 simple python agent (ReAct loop + tools + real egress)")

    log("\n===== cleanup: kill sandbox =====")
    st, txt = http("DELETE", f"/sandboxes/{sid}", None, token)
    log(f"  DELETE -> HTTP {st}: {txt[:120]}")

if __name__ == "__main__":
    main()
