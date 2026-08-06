#!/usr/bin/env python3
"""§L3 tool 层：agent 显式选择沙箱形态的工具集。

架构分工（与 gateway/envd 的分层决策体系对应）：
  - run_python(code)            → Path B。不声明 engine，一次性/长驻由 gateway AST
                                  推断 + envd 粘性守卫决定（agent 无需关心 B 内形态）。
  - run_python(code, stateful=…)→ 可选显式声明：True→kernel / False→oneshot（优先级 0）。
  - request_full_os(code, reason)→ Path C。必须给理由（落 gateway 审计日志）。

工具描述即策略：run_python 是首选（快、便宜）；request_full_os 只在需要
装包/shell/子进程/原生 socket 时使用，有 reason 摩擦防滥用。

环境变量：
  GATEWAY_URL     数据面入口（默认 http://127.0.0.1:49999，对应 kubectl port-forward）
  E2B_SANDBOX_ID  目标沙箱 id（不设则走 gateway debug 单例，自动惰性建沙箱）

用法：
  1) 命令行冒烟：python3 scripts/sandbox_tools.py
  2) smolagents：from sandbox_tools import SANDBOX_TOOLS; CodeAgent(tools=SANDBOX_TOOLS, ...)
"""

import json
import os
import urllib.request
from typing import Optional

GATEWAY_URL = os.environ.get("GATEWAY_URL", "http://127.0.0.1:49999").rstrip("/")
SANDBOX_ID = os.environ.get("E2B_SANDBOX_ID", "")


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
    headers = {"Content-Type": "application/json"}
    if SANDBOX_ID:
        headers["E2b-Sandbox-Id"] = SANDBOX_ID
    req = urllib.request.Request(
        GATEWAY_URL + "/execute",
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
    不支持：pip 装包、shell 命令、子进程、原生 socket（这些请用 request_full_os）。
    变量默认跨调用保持（平台自动推断）；stateful 可显式声明：
      True  → 强制长驻 kernel（后续调用要复用本次变量时）
      False → 强制一次性执行（确定无需保留任何状态时，最省资源）
    """
    payload = {"code": code}
    if stateful is True:
        payload["engine"] = "kernel"
    elif stateful is False:
        payload["engine"] = "wasm"
    return _execute(payload)


def request_full_os(code: str, reason: str) -> SandboxResult:
    """在完整 OS 容器中执行 Python 代码（贵一个数量级，仅在 wasm 干不了时使用）。

    适用：pip/apt 装包、subprocess/shell、需要真实文件系统或系统权限的任务。
    reason 必填：说明为什么 wasm 沙箱无法完成（将进入平台审计日志）。
    """
    if not reason or len(reason.strip()) < 5:
        raise ValueError("request_full_os 必须提供具体理由（≥5 字符，审计要求）")
    return _execute({"code": code, "engine": "full-os", "reason": reason.strip()})


# --- smolagents 集成（可选，未安装时静默跳过） ---
SANDBOX_TOOLS = []
try:
    from smolagents import tool  # type: ignore

    @tool
    def sandbox_run_python(code: str) -> str:
        """Execute Python code in a lightweight wasm sandbox (preferred: fast & cheap).
        No pip/shell/subprocess/socket support - use sandbox_request_full_os for those.
        Variables persist across calls automatically.

        Args:
            code: The Python code to execute.
        """
        r = run_python(code)
        if r.error:
            return f"ERROR {r.error.get('name')}: {r.error.get('value')}\n{r.error.get('traceback', '')}"
        return r.stdout or (r.results[-1] if r.results else "(no output)")

    @tool
    def sandbox_request_full_os(code: str, reason: str) -> str:
        """Execute Python code in a full-OS container (10x more expensive; use ONLY
        when wasm cannot do it: pip install, shell commands, subprocess, raw sockets).

        Args:
            code: The Python code to execute.
            reason: Why the wasm sandbox is insufficient (required, audited).
        """
        r = request_full_os(code, reason)
        if r.error:
            return f"ERROR {r.error.get('name')}: {r.error.get('value')}\n{r.error.get('traceback', '')}"
        return r.stdout or (r.results[-1] if r.results else "(no output)")

    SANDBOX_TOOLS = [sandbox_run_python, sandbox_request_full_os]
except ImportError:
    pass


if __name__ == "__main__":
    print("=== L3 tool 层冒烟测试（需 gateway 可达，默认走 debug 单例沙箱） ===")

    print("\n[1] run_python 无状态 print（期望：gateway 推断 → pathB-oneshot）")
    print("   ", run_python('print("hello from wasm")'))

    print("\n[2] run_python 赋值（期望：gateway 推断 → pathB-kernel）")
    print("   ", run_python("magic = 20260724"))

    print("\n[3] run_python 读回变量（期望：envd 粘性守卫保证 kernel 连续性）")
    print("   ", run_python("print(magic)"))

    print("\n[4] run_python 显式一次性（期望：L3 声明 oneshot，envd 守卫仍粘回 kernel）")
    print("   ", run_python("print(magic)", stateful=False))

    print("\n[5] request_full_os（期望：L3 声明 → pathC；Path C 未启用则报错可见）")
    try:
        print("   ", request_full_os("import platform\nprint(platform.uname())", reason="需要真实 OS 信息验证 Path C 链路"))
    except Exception as e:  # noqa: BLE001
        print("    Path C 调用失败（本地无 containerd 属预期）:", e)

    print("\n[6] reason 缺失防御（期望：本地抛 ValueError）")
    try:
        request_full_os("print(1)", reason="")
        print("    未拦截——BUG")
    except ValueError as e:
        print("    正确拦截:", e)
