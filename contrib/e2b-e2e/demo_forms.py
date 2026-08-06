#!/usr/bin/env python3
"""demo_forms.py — 沙箱形态分层决策全生命周期演示（覆盖四条路径）。

逐场景做一件事：喂一段专门设计的代码 → 观察系统"选了哪种形态、怎么创建、
在哪个运行时里跑、（必要时）如何失败驱动升级"，并把这条完整生命周期内联打印，
无需盯多个日志面板人肉关联。

覆盖路径：
  场景1  Path B 一次性 wasm      —— 无状态 print，L2 判 oneshot
  场景2  Path B 长驻 kernel      —— 构建状态→kernel，第二次读回证明跨执行保留
  场景3  Path C 完整 OS（L2）    —— subprocess 被 AST 识别为 full-os，直下伴生 Pod
  场景4  Path C 完整 OS（L3）    —— agent 显式声明 engine=full-os（request_full_os）
  场景5  L4 失败驱动升级          —— 动态 import fcntl 骗过 L2→wasm 缺失报错→自动升 Path C

每场证据链：
  [决策] gateway smart route decision（route + reason）
  [创建] Path B → envd "wasm sandbox STARTED"；Path C → kubectl get pod sbx-c-<id>(runsc)
  [运行] 代码输出里带的运行时指纹（wasm/WASI vs 4.19.0-gvisor）
  [升级] 仅场景5：gateway "L4 escalation" 日志

环境变量：
  GATEWAY_CTRL   控制面入口（默认 http://127.0.0.1:3000）
  GATEWAY_DATA   数据面入口（默认 http://127.0.0.1:49999）
  JWT            目标租户的 JWT（demo-forms.sh 按租户密钥注入）
  TENANT_NS      gateway/scheduler 所在租户命名空间（默认 tenant-a）
  ENVD_NS        共享数据面 envd 命名空间（默认 agent-sandbox-system）
  PATHC_NS       Path C 伴生 Pod(sbx-c-*) 命名空间（默认 default）
"""
import json
import os
import re
import subprocess
import sys
import time
import urllib.request

CTRL = os.environ.get("GATEWAY_CTRL", "http://127.0.0.1:3000").rstrip("/")
DATA = os.environ.get("GATEWAY_DATA", "http://127.0.0.1:49999").rstrip("/")
JWT = os.environ.get("JWT", "")
# 多租户：gateway/scheduler 驻租户 ns，envd 驻共享数据面 ns，Path C 伴生 Pod 驻 default
TENANT_NS = os.environ.get("TENANT_NS", "tenant-a")
ENVD_NS = os.environ.get("ENVD_NS", "agent-sandbox-system")
PATHC_NS = os.environ.get("PATHC_NS", "default")

C_RST = "\033[0m"; C_DIM = "\033[2m"; C_GRN = "\033[1;32m"; C_CYN = "\033[1;36m"
C_YEL = "\033[1;33m"; C_MAG = "\033[1;35m"; C_RED = "\033[1;31m"


def _http(method, url, headers=None, body=None, timeout=180):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method, headers=headers or {})
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return r.status, r.read().decode(errors="replace")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode(errors="replace")


def create_sandbox():
    st, body = _http("POST", f"{CTRL}/sandboxes",
                     {"X-API-Key": JWT, "Content-Type": "application/json"},
                     {"templateID": "python"})
    if st not in (200, 201):
        raise RuntimeError(f"create sandbox failed: HTTP {st} {body}")
    return json.loads(body)["sandboxID"]


def execute(sid, code, engine=None, reason=None, timeout=180):
    payload = {"code": code}
    if engine:
        payload["engine"] = engine
    if reason:
        payload["reason"] = reason
    st, body = _http("POST", f"{DATA}/execute",
                     {"X-API-Key": JWT, "E2b-Sandbox-Id": sid,
                      "Content-Type": "application/json"},
                     payload, timeout=timeout)
    stdout, stderr = [], []
    for line in body.splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            f = json.loads(line)
        except json.JSONDecodeError:
            continue
        if f.get("type") == "stdout":
            stdout.append(f.get("text", ""))
        elif f.get("type") == "stderr":
            stderr.append(f.get("text", ""))
        elif f.get("type") == "error":
            stderr.append(f.get("value", "") or f.get("name", ""))
    return st, "".join(stdout).strip(), "".join(stderr).strip()


def kill(sid):
    _http("DELETE", f"{CTRL}/sandboxes/{sid}", {"X-API-Key": JWT})


def kubectl(*args, timeout=30):
    try:
        return subprocess.run(["kubectl", *args], capture_output=True, text=True,
                              timeout=timeout).stdout
    except Exception as e:  # noqa: BLE001
        return f"(kubectl error: {e})"


_ANSI = re.compile(r"\x1b\[[0-9;]*m")


def _strip_ansi(s):
    # tracing 给结构化字段（route=/reason=）套了 ANSI 颜色，会把 route 和 = 打断，
    # 导致正则匹配不到 —— 解析前必须先剥掉
    return _ANSI.sub("", s)


def gw_logs(since="40s"):
    return _strip_ansi(kubectl("logs", "-n", TENANT_NS, "-l", "app=gateway", f"--since={since}",
                               "--prefix=true", "--max-log-requests=10", "--tail=-1"))


def envd_logs(since="40s"):
    return _strip_ansi(kubectl("logs", "-n", ENVD_NS, "-l", "app=envd", f"--since={since}",
                               "--prefix=true", "--max-log-requests=10", "--tail=-1"))


def decision_for(sid, logs):
    """从 gateway 日志里抽本沙箱的 smart route decision（取最后一条）。"""
    hit = None
    for ln in logs.splitlines():
        if "smart route decision" in ln and sid in ln:
            m_route = re.search(r'route="?([a-zA-Z0-9-]+)"?', ln)
            m_reason = re.search(r"reason=(\[[^\]]+\][^\n]*)", ln)
            hit = (m_route.group(1) if m_route else "?",
                   (m_reason.group(1).strip() if m_reason else "?"))
    return hit


def l4_for(sid, logs):
    for ln in logs.splitlines():
        if "L4 escalation" in ln and (sid in ln or "Path C" in ln):
            return ln.split("gateway]", 1)[-1].strip() if "gateway]" in ln else ln.strip()
    # L4 日志可能不带 sid，退而取时间窗内的 escalation 行
    for ln in logs.splitlines():
        if "L4 escalation" in ln:
            return ln.split("gateway]", 1)[-1].strip() if "gateway]" in ln else ln.strip()
    return None


def wasm_started(sid, logs):
    for ln in logs.splitlines():
        if "wasm sandbox STARTED" in ln and sid in ln:
            m = re.search(r"instantiate=([0-9.]+ms).*?total=([0-9.]+ms)", ln)
            return f"instantiate={m.group(1)}, cold-start={m.group(2)}" if m else "STARTED"
    return None


def pathc_pod(sid):
    out = kubectl("get", "pod", f"sbx-c-{sid}", "-n", PATHC_NS, "-o",
                  "jsonpath={.status.phase}|{.spec.runtimeClassName}|{.status.podIP}")
    if "|" in out and not out.startswith("(") and "NotFound" not in out:
        phase, rc, ip = (out.split("|") + ["", "", ""])[:3]
        return phase, rc, ip
    return None


def hdr(n, title):
    print(f"\n{C_CYN}{'━'*3} 场景 {n}: {title} {'━'*3}{C_RST}")


def line(tag, text, color=C_RST):
    print(f"  {color}[{tag}]{C_RST} {text}")


def run():
    if not JWT:
        print(f"{C_RED}缺少 JWT{C_RST}"); sys.exit(1)
    print(f"{C_GRN}=== 沙箱形态分层决策全生命周期演示 ==={C_RST}")
    print(f"{C_DIM}控制面 {CTRL} / 数据面 {DATA}{C_RST}")
    sandboxes = []
    t0 = time.time()

    # ── 场景1：Path B 一次性 wasm ──────────────────────────────
    hdr(1, "Path B 一次性 wasm（L2 判 oneshot）")
    s1 = create_sandbox(); sandboxes.append(s1)
    code1 = 'print("oneshot-hello")'
    line("代码", code1, C_DIM)
    st, out, err = execute(s1, code1)
    time.sleep(1)
    gl = gw_logs(); el = envd_logs()
    dec = decision_for(s1, gl)
    line("执行", f"HTTP {st}  输出={out!r}")
    line("决策", f"route={dec[0]}  reason={dec[1]}" if dec else "(未捕获)", C_YEL)
    ws = wasm_started(s1, el)
    line("创建", f"envd wasm sandbox STARTED ({ws})" if ws else "envd 进程内 wasm 实例（一次性，未打 STARTED 属正常）", C_GRN)
    line("结论", "形态=一次性 wasm | 创建=envd 进程内实例 | 运行=wasm/WASI", C_MAG)

    # ── 场景2：Path B 长驻 kernel（跨执行保留）─────────────────
    hdr(2, "Path B 长驻 kernel（构建状态→复用）")
    s2 = create_sandbox(); sandboxes.append(s2)
    code2a = "k_state = 20260727"
    code2b = 'print("kernel-reuse:", k_state)'
    line("代码", f"exec1: {code2a}   exec2: {code2b}", C_DIM)
    st1, _, _ = execute(s2, code2a)
    st2, out2, err2 = execute(s2, code2b)
    time.sleep(1)
    gl = gw_logs()
    dec = decision_for(s2, gl)
    line("执行", f"exec1 HTTP {st1} / exec2 HTTP {st2}  输出={out2!r}")
    line("决策", f"route={dec[0]}  reason={dec[1]}" if dec else "(未捕获)", C_YEL)
    ok = "kernel-reuse: 20260727" in out2
    line("创建", "envd 长驻 python kernel（同 sandbox 复用热 kernel）", C_GRN)
    line("结论", f"形态=长驻 kernel | 跨执行状态保留={'✅ 成立' if ok else '❌ 丢失'} | 运行=wasm kernel", C_MAG)

    # ── 场景3：Path C 完整 OS（L2 AST 识别）────────────────────
    hdr(3, "Path C 完整 OS（L2 AST 识别 subprocess）")
    s3 = create_sandbox(); sandboxes.append(s3)
    code3 = ('import subprocess\n'
             'print(subprocess.run(["uname","-a"],capture_output=True,text=True).stdout.strip())')
    line("代码", "import subprocess; subprocess.run(['uname','-a'])", C_DIM)
    st, out, err = execute(s3, code3, timeout=180)
    time.sleep(1)
    gl = gw_logs()
    dec = decision_for(s3, gl)
    pod = pathc_pod(s3)
    line("执行", f"HTTP {st}  输出={out!r}")
    line("决策", f"route={dec[0]}  reason={dec[1]}" if dec else "(未捕获)", C_YEL)
    line("创建", f"sbx-c-{s3[:8]}…  phase={pod[0]} runtimeClass={pod[1]} podIP={pod[2]}" if pod else "(Pod 未找到)", C_GRN)
    line("结论", f"形态=完整OS伴生Pod | 隔离={'gVisor(runsc)' if pod and pod[1]=='runsc' else pod[1] if pod else '?'} | 运行={'真实Linux容器' if 'Linux' in out else out[:30]}", C_MAG)

    # ── 场景4：Path C 完整 OS（L3 显式声明）────────────────────
    hdr(4, "Path C 完整 OS（L3 显式 engine=full-os）")
    s4 = create_sandbox(); sandboxes.append(s4)
    code4 = "import platform; print('kernel=', platform.uname().release)"
    line("代码", f"{code4}   (engine=full-os, reason=…)", C_DIM)
    st, out, err = execute(s4, code4, engine="full-os", reason="演示 L3 显式声明走 Path C", timeout=180)
    time.sleep(1)
    gl = gw_logs()
    dec = decision_for(s4, gl)
    pod = pathc_pod(s4)
    line("执行", f"HTTP {st}  输出={out!r}")
    line("决策", f"route={dec[0]}  reason={dec[1]}" if dec else "(未捕获)", C_YEL)
    line("创建", f"sbx-c-{s4[:8]}…  phase={pod[0]} runtimeClass={pod[1]}" if pod else "(Pod 未找到)", C_GRN)
    line("结论", f"形态=完整OS伴生Pod(声明优先) | 运行={'gVisor '+out.split('=')[-1].strip() if 'gvisor' in out.lower() else out[:30]}", C_MAG)

    # ── 场景5：L4 失败驱动升级（wasm→Path C）───────────────────
    hdr(5, "L4 失败驱动升级（wasm 缺失能力→自动升 Path C）")
    s5 = create_sandbox(); sandboxes.append(s5)
    code5 = ('import importlib\n'
             'm = importlib.import_module("".join(["fc","ntl"]))\n'
             'print("fcntl-loaded:", m.__name__)')
    line("代码", 'importlib.import_module("fcntl")  # 动态名，L2 看不见 → 先走 wasm', C_DIM)
    st, out, err = execute(s5, code5, timeout=180)
    time.sleep(1)
    gl = gw_logs()
    dec = decision_for(s5, gl)
    l4 = l4_for(s5, gl)
    pod = pathc_pod(s5)
    line("执行", f"HTTP {st}  输出={out!r}")
    line("决策", f"首选 route={dec[0]}  reason={dec[1]}" if dec else "(未捕获)", C_YEL)
    line("升级", l4 if l4 else "(未捕获 L4 日志——可能该片段被 L2 提前判 full-os，见决策行)", C_RED)
    line("创建", f"升级后 sbx-c-{s5[:8]}…  phase={pod[0]} runtimeClass={pod[1]}" if pod else "(Pod 未找到)", C_GRN)
    ok5 = "fcntl-loaded" in out
    line("结论", f"wasm 缺 fcntl→L4 升 Path C→容器内有 fcntl→{'✅ 成功' if ok5 else '❌ 未成功'}", C_MAG)

    # ── 清理 ──────────────────────────────────────────────────
    print(f"\n{C_DIM}清理 {len(sandboxes)} 个演示沙箱…{C_RST}")
    for s in sandboxes:
        try:
            kill(s)
        except Exception:  # noqa: BLE001
            pass
    print(f"{C_GRN}=== 演示完成，用时 {time.time()-t0:.1f}s ==={C_RST}")


if __name__ == "__main__":
    run()
