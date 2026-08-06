#!/usr/bin/env python3
"""验证 epoch 兜底：死循环 cell 被强制 trap，且 sandbox 之后能恢复。

前置：
  kubectl port-forward svc/gateway 3000:8080 &
  kubectl port-forward svc/gateway 49999:8080 &
"""
import os
import time

os.environ["E2B_DEBUG"] = "true"
os.environ["E2B_VALIDATE_API_KEY"] = "false"
os.environ.setdefault("E2B_API_KEY", "gateway-debug")

from e2b_code_interpreter import Sandbox  # noqa: E402


def dump(tag, ex):
    print(f"--- {tag} ---")
    print("  stdout:", ex.logs.stdout)
    print("  results:", [r.text for r in ex.results])
    print("  error:", ex.error)


sbx = Sandbox.create()
print(">>> sandbox created:", sbx.sandbox_id)

# 1) globals 持久化 sanity
dump("set x=123", sbx.run_code("x = 123"))
dump("read x", sbx.run_code("x"))

# 2) 死循环 cell —— epoch 兜底应在默认 60s 超时后强制 trap
print(">>> submitting `while True: pass` (expect epoch trap ~60s)...")
t0 = time.time()
ex = sbx.run_code("while True:\n    pass")
dt = time.time() - t0
dump(f"infinite loop (elapsed {dt:.1f}s)", ex)

# 3) 恢复验证 —— 上一 cell trap 后 context poisoned，本次应自动重建并正常执行
print(">>> post-trap recovery cell `1+1`...")
ex2 = sbx.run_code("1 + 1")
dump("recovery 1+1", ex2)

# 4) 新 context 应已重置 globals（x 不再存在）
dump("read x after recovery (expect NameError)", sbx.run_code("x"))

sbx.kill()
print(">>> killed. DONE")
