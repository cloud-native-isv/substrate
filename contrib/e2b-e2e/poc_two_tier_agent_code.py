import json

# --- capability probe: host_proxy is the bootstrap shim -> env_proxy host function
#     -> HostProxy egress allowlist (the design's host-mediated capability path).
try:
    import host_proxy
    HAVE_HP = True
    HP_ERR = ""
except Exception as e:
    HAVE_HP = False
    HP_ERR = "%s: %s" % (type(e).__name__, e)

# --- tools (in the real design each is a capability-bound host function) ---
def tool_calc(expr):
    return {"tool": "calc", "expr": expr, "result": eval(expr, {"__builtins__": {}}, {})}

def tool_web_get(url):
    if not HAVE_HP:
        return {"tool": "web_get", "ok": False, "error": "no host_proxy", "detail": HP_ERR}
    try:
        r = host_proxy.http_request("GET", url, None, None)
        return {"tool": "web_get", "ok": True, "url": url, "resp": str(r)[:160]}
    except Exception as e:
        return {"tool": "web_get", "ok": False, "url": url, "error": "%s: %s" % (type(e).__name__, e)}

TOOLS = {"calc": tool_calc, "web_get": tool_web_get}

# --- reasoning step: deterministic ReAct planner standing in for the LLM.
#     (No LLM API key in this POC env; the loop/tool-dispatch/host-mediated
#      egress are exercised for real. Swap this fn for a qwen call via
#      host_proxy.http_request when a key is available.)
def plan(task, done):
    acts = [d["action"] for d in done]
    if "web_get" not in acts:
        return {"thought": "prove host-mediated egress from inside wasm",
                "action": "web_get", "args": {"url": "https://www.aliyun.com"}}
    if "calc" not in acts:
        return {"thought": "do an in-sandbox computation tool call",
                "action": "calc", "args": {"expr": "40+2"}}
    return {"thought": "both tools exercised; finish",
            "action": "finish", "args": {"answer": "agent loop completed: web_get(egress)+calc tools ok"}}

def run_agent(task, max_steps=6):
    trace, done = [], []
    for i in range(max_steps):
        d = plan(task, done)
        if d["action"] == "finish":
            trace.append({"step": i, "final": d["args"]["answer"]})
            break
        obs = TOOLS[d["action"]](**d["args"])
        done.append({"action": d["action"]})
        trace.append({"step": i, "thought": d["thought"], "action": d["action"], "observation": obs})
    return {"task": task, "have_host_proxy": HAVE_HP, "steps": len(done), "trace": trace}

print("AGENT_RESULT " + json.dumps(run_agent("demo: egress + calc"), ensure_ascii=False))
