#!/usr/bin/env python3
"""一锤定音：证明 run_code 确实在 wasm 沙箱内执行，而非本机/模型硬背。

对比三路证据：
  1) 本机 Mac Python 的运行时指纹（对照组）
  2) 沙箱内 run_code 打印的运行时指纹（应为 vmware-labs wasm CPython：
     stdlib 路径在 /wlr/... 、machine=wasm32 之类，与 Mac 截然不同）
  3) 在沙箱内写一个带随机 nonce 的标记文件再读回（证明是沙箱自己的文件系统，
     且该 nonce 只可能来自沙箱内执行）

前置：kubectl port-forward svc/gateway 3000:8080 & 49999:8080（同一 gateway pod）
"""
import os, sys, platform, uuid

os.environ["E2B_DEBUG"] = "true"
os.environ["E2B_VALIDATE_API_KEY"] = "false"
os.environ.setdefault("E2B_API_KEY", "gateway-debug")

from e2b_code_interpreter import Sandbox  # noqa: E402

print("=== [对照组] 本机 Mac Python 指纹 ===")
print("  sys.version   :", sys.version.replace("\n", " "))
print("  sys.platform  :", sys.platform)
print("  machine       :", platform.machine())
print("  os.__file__   :", os.__file__)

nonce = uuid.uuid4().hex
probe = f'''
import sys, platform, os
print("SANDBOX sys.version   :", sys.version.replace(chr(10)," "))
print("SANDBOX sys.platform  :", sys.platform)
print("SANDBOX machine       :", platform.machine())
print("SANDBOX os.__file__   :", os.__file__)
print("SANDBOX sys.prefix    :", sys.prefix)
# 写标记文件到沙箱 CWD 再读回，nonce 只可能来自沙箱内执行
open("/sandbox/_proof.txt","w").write("{nonce}")
print("SANDBOX marker_readback:", open("/sandbox/_proof.txt").read())
'''

sbx = Sandbox.create()
print("\n>>> sandbox created:", sbx.sandbox_id)
ex = sbx.run_code(probe)
print("\n=== [沙箱内] run_code stdout ===")
print(ex.logs.stdout)
if ex.error:
    print("  error:", ex.error)
print("=== 校验 ===")
out = "".join(ex.logs.stdout) if isinstance(ex.logs.stdout, list) else str(ex.logs.stdout)
print("  nonce 注入前值 :", nonce)
print("  沙箱回读命中   :", nonce in out)
print("  运行时非本机   :", ("wasm" in out.lower()) or ("/wlr/" in out) or (sys.platform not in out))
sbx.kill()
print(">>> killed. DONE")
