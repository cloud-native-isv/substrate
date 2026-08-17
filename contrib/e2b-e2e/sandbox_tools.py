#!/usr/bin/env python3
"""§L3 tool 层：agent 显式选择沙箱形态的工具集（已改指 e2bgw，M3 3.4）。

数据面走 e2bgw 路径式代理：POST {E2BGW}/sandboxes/{id}/execute → atenet →
ateom-wasmd 进程内 actor HTTP 面（NDJSON 流）。旧 Rust gateway 的 debug 单例、
engine 推断（oneshot/kernel）与 Path C（full-OS 升级）已随迁移退役：
  - wasm actor 的 kernel 常驻，变量跨调用保持（Jupyter 语义）；
  - full-OS 形态在 substrate 中是另建 gvisor/microvm 模板的 sandbox，
    不再是同一沙箱内的升级路径，request_full_os 直接报错说明。

环境变量：
  GATEWAY_URL     e2bgw 入口（默认 http://127.0.0.1:8080，对应
                  kubectl -n ate-system port-forward svc/e2bgw 8080:80）
  E2B_SANDBOX_ID  目标沙箱 id（必需：先用 SDK/curl POST /sandboxes 建好）
  E2B_API_KEY     e2bgw API key（sign_jwt.py 签发，atespace claim）

用法：
  1) 命令行冒烟：python3 sandbox_tools.py
  2) smolagents：from sandbox_tools import SANDBOX_TOOLS; CodeAgent(tools=SANDBOX_TOOLS, ...)
"""

import json
import os
import urllib.request
from typing import Optional

GATEWAY_URL = os.environ.get("GATEWAY_URL", "http://127.0.0.1:8080").rstrip("/")
SANDBOX_ID = os.environ.get("E2B_SANDBOX_ID", "")
API_KEY = os.environ.get("E2B_API_KEY", "")


class SandboxResult:
    """聚合 NDJSON 帧为易用结果对象"""

    def __init__(self, frames):
        self.frames = frames
        self.stdout = "".join(f.get("text", "") for f in frames if f.get("type") == "stdout")
        self.stderr = "".join(f.get("text", "") for f in frames if f.get("type") == "stderr")
        self.results = [f.get("text", "") for f in frames if f.get("type") == "result"]
        errs = [f for f in frames if f.get("type") == "error"]
        self.error = errs[0] if errs else None

    def __repr__(self):
        if self.error:
            return f"<SandboxResult ERROR {self.error.get('name')}: {self.error.get('value')}>"
        out = self.stdout.strip() or (self.results[-1] if self.results else "")
        return f"<SandboxResult ok: {out!r}>"


def _execute(payload: dict) -> SandboxResult:
    if not SANDBOX_ID:
        raise RuntimeError(
            "E2B_SANDBOX_ID 未设置：e2bgw 无 debug 单例，需先建沙箱"
            "（POST /sandboxes，见 contrib/e2b-e2e/README.md）"
        )
    headers = {"Content-Type": "application/json"}
    if API_KEY:
        headers["X-API-Key"] = API_KEY
    req = urllib.request.Request(
        f"{GATEWAY_URL}/sandboxes/{SANDBOX_ID}/execute",
        data=json.dumps(payload).encode(),
        headers=headers,
        method="POST",
    )
    frames = []
    with urllib.request.urlopen(req, timeout=180) as resp:
        for line in resp:
            line = line.strip()
            if line:
                frames.append(json.loads(line))
    return SandboxResult(frames)


def run_python(code: str, stateful: Optional[bool] = None) -> SandboxResult:
    """在轻量 wasm 沙箱中执行 Python 代码（首选工具：毫秒级启动、成本最低）。

    适用：计算、数据处理、字符串/JSON 操作等纯 Python 任务。
    不支持：pip 装包、shell 命令、子进程、原生 socket。
    变量跨调用保持（wasm actor kernel 常驻，Jupyter 语义）；stateful 参数
    保留仅为工具签名兼容，对 e2bgw 链路无效果（无 oneshot 形态）。
    """
    return _execute({"code": code})


def request_full_os(code: str, reason: str) -> SandboxResult:
    """（已随 substrate 迁移退役）full-OS 不再是同一沙箱内的升级路径。

    substrate 下 full-OS 形态 = 用 gvisor/microvm SandboxClass 的 ActorTemplate
    另建一个 sandbox（POST /sandboxes，templateID 指向对应模板）。
    """
    raise RuntimeError(
        "full-OS 升级路径已退役：请用 gvisor/microvm 模板另建 sandbox"
        f"（reason={reason!r}）"
    )


# --- smolagents 集成（可选，未安装时静默跳过） ---
SANDBOX_TOOLS = []
try:
    from smolagents import tool  # type: ignore

    @tool
    def sandbox_run_python(code: str) -> str:
        """Execute Python code in a lightweight wasm sandbox (preferred: fast & cheap).
        No pip/shell/subprocess/socket support. Variables persist across calls
        automatically (resident Jupyter-style kernel).

        Args:
            code: The Python code to execute.
        """
        r = run_python(code)
        if r.error:
            return f"ERROR {r.error.get('name')}: {r.error.get('value')}\n{r.error.get('traceback', '')}"
        return r.stdout or (r.results[-1] if r.results else "(no output)")

    @tool
    def sandbox_request_full_os(code: str, reason: str) -> str:
        """Request a full-OS execution environment. NOTE: retired on the substrate
        stack - full-OS now means creating a separate sandbox from a gvisor or
        microvm template; this tool only explains that to the agent.

        Args:
            code: The Python code to execute.
            reason: Why the wasm sandbox is insufficient.
        """
        try:
            request_full_os(code, reason)
        except RuntimeError as e:
            return f"UNSUPPORTED: {e}"
        return "(unreachable)"

    SANDBOX_TOOLS = [sandbox_run_python, sandbox_request_full_os]
except ImportError:
    pass


if __name__ == "__main__":
    print("=== L3 tool 层冒烟测试（需 e2bgw 可达且 E2B_SANDBOX_ID/E2B_API_KEY 已设） ===")

    print("\n[1] run_python print（期望：e2bgw 路径式代理 → wasm kernel stdout）")
    print("   ", run_python('print("hello from wasm")'))

    print("\n[2] run_python 赋值（期望：常驻 kernel 接受赋值）")
    print("   ", run_python("magic = 20260724"))

    print("\n[3] run_python 读回变量（期望：同 kernel 变量保持，Jupyter 语义）")
    print("   ", run_python("print(magic)"))

    print("\n[4] request_full_os（期望：报错说明已退役，指引另建 gvisor/microvm 沙箱）")
    try:
        request_full_os("import platform\nprint(platform.uname())", reason="验证退役路径")
        print("    未拦截——BUG")
    except RuntimeError as e:
        print("    正确拦截:", e)
