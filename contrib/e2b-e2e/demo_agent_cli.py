#!/usr/bin/env python3
"""demo_agent_cli.py — wasm-e2b-k8s 演示用交互式 agent CLI

- 用未改动的 e2b SDK + smolagents CodeAgent（qwen-max via DashScope）
- SDK 走 debug 模式（E2B_DEBUG=true）→ 控制面 localhost:3000 / 数据面 localhost:49999
- 两条 port-forward 由 demo.sh 起，本脚本只关心 REPL 与时序输出

演示看点（在每步输出的时间戳里可读）：
  1) 初始化：模型 & E2B Sandbox（第一次会触发 wasm 沙箱创建 → 在 envd 日志看得到冷启）
  2) 每一次任务：LLM 生成代码 → wasm 内执行 → NDJSON 事件流回来 → final_answer
  3) 同一 sandbox 多任务共享 kernel globals（"你先定义 x=42，下一句让我 print(x)"）

用法：
    ./scripts/demo.sh           # 由 demo.sh 拉起（推荐）
    ./scripts/demo_agent_cli.py # 直接跑（需先手动起 port-forward + DASHSCOPE_API_KEY）
"""
from __future__ import annotations
import os
import sys
import time
import datetime

# —— SDK 走 debug 模式：控制面/数据面端口由 demo.sh 通过 E2B_API_URL / E2B_JUPYTER_PORT 注入 ——
# （E2B_API_URL 是 SDK 原生读的控制面地址；数据面 jupyter 端口 SDK 写死 49999，需下方覆盖）
os.environ["E2B_DEBUG"] = "true"
os.environ["E2B_VALIDATE_API_KEY"] = "false"
os.environ.setdefault("E2B_API_KEY", "gateway-debug")

# 每租户不同数据面端口：覆盖 e2b_code_interpreter 写死的 JUPYTER_PORT(49999)，
# 使 debug 模式下 _jupyter_url = http://localhost:<该端口>，支持多租户并排演示。
_jp = os.environ.get("E2B_JUPYTER_PORT")
if _jp:
    import importlib
    _port = int(_jp)
    try:
        import e2b_code_interpreter.constants as _c
        _c.JUPYTER_PORT = _port
        for _m in ("code_interpreter_sync", "code_interpreter_async"):
            try:
                importlib.import_module(f"e2b_code_interpreter.{_m}").JUPYTER_PORT = _port
            except Exception:  # noqa: BLE001
                pass
    except Exception:  # noqa: BLE001
        pass


C_GRN = "\033[1;32m"
C_CYN = "\033[1;36m"
C_YEL = "\033[1;33m"
C_MAG = "\033[1;35m"
C_RED = "\033[1;31m"
C_DIM = "\033[2m"
C_RST = "\033[0m"


def ts() -> str:
    """HH:MM:SS.mmm 时间戳。"""
    return datetime.datetime.now().strftime("%H:%M:%S.%f")[:-3]


def log(color: str, tag: str, msg: str) -> None:
    print(f"{C_DIM}[{ts()}]{C_RST} {color}{tag}{C_RST} {msg}", flush=True)


def die(msg: str) -> None:
    log(C_RED, "XX", msg)
    sys.exit(1)


def main() -> None:
    api_key = os.environ.get("DASHSCOPE_API_KEY")
    if not api_key:
        die("DASHSCOPE_API_KEY 未设置（demo.sh 会从 .secrets/DASHSCOPE_API_KEY 读入）")

    model_id = os.environ.get("QWEN_MODEL_ID", "qwen-max")
    dashscope_base = "https://dashscope.aliyuncs.com/compatible-mode/v1"

    log(C_CYN, "==", f"链路：smolagents → e2b SDK (debug) → localhost:3000/49999")
    log(C_CYN, "==", f"                       → kubectl port-forward → gateway → envd → wasm")
    log(C_CYN, "==", f"LLM：{model_id} via DashScope（{dashscope_base}）")
    print()

    log(C_YEL, "->", "导入 smolagents + 初始化 CodeAgent（会创建 wasm sandbox，首次触发冷启）")
    t_init = time.monotonic()
    from smolagents import CodeAgent, OpenAIServerModel  # noqa: E402

    model = OpenAIServerModel(model_id=model_id, api_base=dashscope_base, api_key=api_key)
    agent = CodeAgent(tools=[], model=model, executor_type="e2b", max_steps=6)
    # python.wasm 无 pip 也没 shell；跳过 numpy/PIL 的 `!pip install`（final_answer 图像分支用的）
    agent.python_executor.installed_packages += ["numpy", "PIL"]

    # 尽力从 executor 里拿 sandbox_id 打出来（不同版本的 e2b/smolagents 属性名略有差异）
    sbx = getattr(agent.python_executor, "sandbox", None)
    sbx_id = None
    for attr in ("sandbox_id", "_sandbox_id", "id"):
        if sbx is not None and hasattr(sbx, attr):
            sbx_id = getattr(sbx, attr)
            break
    init_elapsed = time.monotonic() - t_init
    log(C_GRN, "OK", f"agent 就绪，用时 {init_elapsed:.2f}s")
    if sbx_id:
        log(C_MAG, "->", f"sandbox_id = {sbx_id}")
        log(C_DIM, "  ", f"envd 侧同一 id 会打出 'wasm sandbox STARTED (instantiate=…ms, cold-start …ms)'")
    else:
        log(C_YEL, "!!", "未取到 sandbox_id（不影响执行，只是无法反查 envd 日志）")

    print()
    print(f"{C_DIM}在 task> 后输入任务（回车提交）；exit / Ctrl-D 退出。{C_RST}")
    print(f"{C_DIM}同一会话内多次 run 共享 kernel globals，可以先定义变量再让下一句 print。{C_RST}")
    print()

    n = 0
    try:
        while True:
            try:
                task = input(f"{C_DIM}[{ts()}]{C_RST} {C_GRN}task>{C_RST} ").strip()
            except EOFError:
                print()
                break
            if not task:
                continue
            if task.lower() in ("exit", "quit", ":q"):
                break

            n += 1
            print()
            log(C_CYN, "==", f"Task #{n} 已提交")
            t_task = time.monotonic()
            try:
                result = agent.run(task)
            except Exception as e:
                log(C_RED, "XX", f"Task #{n} 报错：{e}")
                print()
                continue
            elapsed = time.monotonic() - t_task
            print()
            log(C_GRN, "==", f"Task #{n} 完成，用时 {elapsed:.2f}s")
            log(C_MAG, "->", f"FINAL: {result}")
            print()
    finally:
        print()
        log(C_CYN, "==", f"退出，累计 {n} 个任务")
        try:
            if sbx is not None and hasattr(sbx, "kill"):
                sbx.kill()
                log(C_DIM, "->", "sandbox killed")
        except Exception:
            pass


if __name__ == "__main__":
    main()
