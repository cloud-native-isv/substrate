#!/usr/bin/env python3
"""smolagents CodeAgent 驱动 wasm-e2b-k8s sandbox（通过 gateway 的 E2B 兼容层）。

前置：
  kubectl port-forward svc/gateway 3000:8080 &
  kubectl port-forward svc/gateway 49999:8080 &

环境变量：
  DASHSCOPE_API_KEY  必填，DashScope OpenAI 兼容端点的 key
  QWEN_MODEL_ID      默认 qwen-max
"""
import os

# E2B SDK 指向 gateway：debug 模式 → 控制面 localhost:3000、数据面 localhost:49999
os.environ["E2B_DEBUG"] = "true"
os.environ["E2B_VALIDATE_API_KEY"] = "false"
os.environ.setdefault("E2B_API_KEY", "gateway-debug")  # debug 模式控制面不校验

from smolagents import CodeAgent, OpenAIServerModel, E2BExecutor  # noqa: E402

DASHSCOPE_BASE = "https://dashscope.aliyuncs.com/compatible-mode/v1"
model_id = os.environ.get("QWEN_MODEL_ID", "qwen-max")
api_key = os.environ["DASHSCOPE_API_KEY"]

model = OpenAIServerModel(
    model_id=model_id,
    api_base=DASHSCOPE_BASE,
    api_key=api_key,
)

# CodeAgent + E2B 远程执行器（run_code 打到 gateway 的 /execute）
# 注意：不要传 additional_authorized_imports——smolagents 的 E2B 执行器会对其中每一项
# 执行 `!pip install`，而我们的 python.wasm 内核没有 shell/pip；math、statistics 本就是
# 标准库，wasm 内直接可 import，无需声明。
agent = CodeAgent(
    tools=[],
    model=model,
    executor_type="e2b",
    max_steps=4,
)

# smolagents 默认的 final_answer 工具声明依赖 numpy/pillow（仅用于图像型 final answer）。
# python.wasm 无 pip，且这些是 C 扩展包，wasm 内本就装不了、跑不了；本任务只返回文本结论，
# 不需要它们。把它们标记为"已安装"以跳过 wasm 内不可用的 `!pip install`
# （等价于真实 E2B 用预装包的 template）。
agent.python_executor.installed_packages += ["numpy", "PIL"]

task = (
    "计算 1 到 100 所有素数的和，并单独给出其中最大的素数。"
    "用 Python 在沙箱里算，最后用 final_answer 返回一句话结论。"
)
print(">>> task:", task)
result = agent.run(task)
print(">>> FINAL:", result)
