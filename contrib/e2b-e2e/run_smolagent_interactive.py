#!/usr/bin/env python3
"""smolagents CodeAgent 交互式驱动 wasm-e2b-k8s（未改动 SDK，非 debug 走公网 CLB）。

与 run_smolagent.py 的区别：
  1) 非 debug 生产路径（经 E2B_DOMAIN 指向公网 gateway），而非 localhost debug；
  2) 循环读取用户手动输入的任务，而非写死单条 task。

前置环境变量（与 scripts/test_sdk_nondebug_e2e.py 一致）：
  E2B_DOMAIN            例：sbx.test（本机 DNS 桩解析到 CLB）
  E2B_API_KEY           租户 JWT（scripts/sign_jwt.py 签发）
  SSL_CERT_FILE         自签通配符证书 PEM
  DASHSCOPE_API_KEY     DashScope key（缺省则读 .secrets/DASHSCOPE_API_KEY）
  QWEN_MODEL_ID         默认 qwen-max

用法：
  export E2B_DOMAIN=sbx.test E2B_API_KEY=$(.venv-agent/bin/python scripts/sign_jwt.py user-A)
  export SSL_CERT_FILE=$PWD/.secrets/sbx-cert.pem
  .venv-agent/bin/python scripts/run_smolagent_interactive.py
  # 然后逐行输入任务，exit / quit / Ctrl-D 退出
"""
import os
import sys
from pathlib import Path

# 非 debug 生产形态：由 E2B_DOMAIN 指向公网 gateway
os.environ.setdefault("E2B_DEBUG", "false")
os.environ["E2B_VALIDATE_API_KEY"] = "false"  # 租户 JWT 非 e2b_ 格式，须关客户端校验

REPO = Path(__file__).resolve().parent.parent


def _require_env() -> None:
    missing = [k for k in ("E2B_DOMAIN", "E2B_API_KEY", "SSL_CERT_FILE") if not os.environ.get(k)]
    if missing:
        sys.exit(f"[agent] 缺少环境变量：{', '.join(missing)}（见脚本头部用法）")
    if not os.environ.get("DASHSCOPE_API_KEY"):
        f = REPO / ".secrets" / "DASHSCOPE_API_KEY"
        if f.exists():
            os.environ["DASHSCOPE_API_KEY"] = f.read_text().strip()
        else:
            sys.exit("[agent] 缺少 DASHSCOPE_API_KEY，且 .secrets/DASHSCOPE_API_KEY 不存在")


def main() -> None:
    _require_env()
    from smolagents import CodeAgent, OpenAIServerModel  # noqa: E402

    model = OpenAIServerModel(
        model_id=os.environ.get("QWEN_MODEL_ID", "qwen-max"),
        api_base="https://dashscope.aliyuncs.com/compatible-mode/v1",
        api_key=os.environ["DASHSCOPE_API_KEY"],
    )

    print(f"[agent] E2B_DOMAIN={os.environ['E2B_DOMAIN']}  (非 debug，走公网 CLB)")
    print("[agent] 初始化 CodeAgent + E2B 远程执行器（wasm 沙箱）...")
    agent = CodeAgent(tools=[], model=model, executor_type="e2b", max_steps=6)
    # final_answer 工具声明依赖 numpy/pillow（C 扩展，wasm 内装不了），标记为已装以跳过 !pip install
    agent.python_executor.installed_packages += ["numpy", "PIL"]
    print("[agent] 就绪。输入任务后回车（exit / quit / Ctrl-D 退出）。\n")

    while True:
        try:
            task = input("task> ").strip()
        except (EOFError, KeyboardInterrupt):
            print("\n[agent] 退出。")
            break
        if not task:
            continue
        if task.lower() in ("exit", "quit"):
            print("[agent] 退出。")
            break
        try:
            result = agent.run(task)
            print(f"\n>>> FINAL: {result}\n")
        except Exception as e:  # 单条任务失败不退出 REPL
            print(f"\n[agent] 该任务执行出错：{type(e).__name__}: {e}\n")


if __name__ == "__main__":
    main()
