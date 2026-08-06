#!/usr/bin/env python3
"""B11 文件 API：未改动 E2B SDK 的 Filesystem 层非 debug 端到端验证。

验证双协议契约（REST /files + connect-RPC /filesystem.Filesystem/*）与
「文件 API ⇄ kernel 同一文件视图」的双向一致性——这是 Wasm 沙箱 preopen
挂载语义的正确性证明：SDK 写的文件 kernel 立即可读，kernel 写的文件
SDK 立即可读，两者只是同一宿主机 sandbox_dir 的两个视图。

环境变量前置（与 test_sdk_nondebug_e2e.py 相同）：
  E2B_DOMAIN / E2B_API_KEY / E2B_VALIDATE_API_KEY=false / SSL_CERT_FILE / E2B_DEBUG=false

覆盖：
  1) write/read（str 与 bytes）、read 不存在 → FileNotFoundException
  2) exists/get_info/list（含 depth 递归）/make_dir 幂等语义/rename/remove
  3) 嵌套路径自动补目录
  4) 路径别名：/home/user ≡ /sandbox ≡ 相对路径（同一文件视图）
  5) 双向一致性：files.write→kernel open；kernel open('w')→files.read
  6) 逃逸防护：.. 越界被拒绝；watch_dir → connect unimplemented（POC 边界）
"""
import os
import sys

from e2b_code_interpreter import Sandbox
from e2b.exceptions import FileNotFoundException


def _require_env() -> str:
    dom = os.environ.get("E2B_DOMAIN")
    if not dom:
        sys.exit("[fs-e2e] 缺少 E2B_DOMAIN")
    if os.environ.get("E2B_DEBUG", "false").lower() == "true":
        sys.exit("[fs-e2e] E2B_DEBUG 必须为 false（验证非 debug 生产形态）")
    if not os.environ.get("E2B_API_KEY"):
        sys.exit("[fs-e2e] 缺少 E2B_API_KEY（租户 JWT）")
    if not os.environ.get("SSL_CERT_FILE"):
        sys.exit("[fs-e2e] 缺少 SSL_CERT_FILE（自签 CA）")
    return dom


def main() -> None:
    dom = _require_env()
    print(f"[fs-e2e] E2B_DOMAIN = {dom}")

    sbx = Sandbox.create(timeout=300)
    try:
        sid = sbx.sandbox_id
        print(f"[fs-e2e] sandbox_id = {sid}")
        fs = sbx.files

        # ---- 1) 写/读：str（multipart 路径）----
        info = fs.write("/home/user/hello.txt", "hello fs api")
        print(f"[fs-e2e] write str   -> path={info.path} type={info.type}")
        assert info.path == "/sandbox/hello.txt", f"entry.path 应输出 wasm 视图：{info.path}"
        assert fs.read("/home/user/hello.txt") == "hello fs api"

        # bytes 写入 + bytes 读回
        fs.write("/home/user/blob.bin", b"\x00\x01\x02binary")
        data = fs.read("/home/user/blob.bin", format="bytes")
        assert bytes(data) == b"\x00\x01\x02binary", f"bytes 往返失败：{bytes(data)!r}"
        print("[fs-e2e] write/read str+bytes 往返一致")

        # read 不存在 → FileNotFoundException（REST 404 契约）
        try:
            fs.read("/home/user/nope.txt")
            sys.exit("[fs-e2e] 读不存在文件未报错——BUG")
        except FileNotFoundException:
            print("[fs-e2e] read 不存在文件 -> FileNotFoundException ✓")

        # ---- 2) exists / get_info / list ----
        assert fs.exists("/home/user/hello.txt") is True
        assert fs.exists("/home/user/nope.txt") is False
        gi = fs.get_info("/home/user/hello.txt")
        print(f"[fs-e2e] get_info    -> name={gi.name} size={gi.size} perm={gi.permissions}")
        assert gi.name == "hello.txt" and gi.size == len("hello fs api")

        # make_dir：首次 True，重复 False（already_exists 契约）
        assert fs.make_dir("/home/user/sub") is True
        assert fs.make_dir("/home/user/sub") is False
        print("[fs-e2e] make_dir True/False 幂等语义 ✓")

        # 嵌套写入自动补目录（E2B 语义）
        fs.write("/home/user/sub/nested/deep.txt", "deep")
        entries = fs.list("/home/user", depth=3)
        names = {(e.path, e.type.value if e.type else None) for e in entries}
        print(f"[fs-e2e] list depth=3 -> {sorted(n for n, _ in names)}")
        assert ("/sandbox/sub", "dir") in names
        assert ("/sandbox/sub/nested", "dir") in names
        assert ("/sandbox/sub/nested/deep.txt", "file") in names
        depth1 = fs.list("/home/user", depth=1)
        assert all("/" not in e.path[len("/sandbox/"):] for e in depth1), "depth=1 不应含孙级"

        # ---- 3) rename / remove ----
        moved = fs.rename("/home/user/hello.txt", "/home/user/sub/moved.txt")
        assert moved.path == "/sandbox/sub/moved.txt", f"rename 返回路径：{moved.path}"
        assert fs.exists("/home/user/hello.txt") is False
        assert fs.read("/home/user/sub/moved.txt") == "hello fs api"
        fs.remove("/home/user/sub")
        assert fs.exists("/home/user/sub") is False
        print("[fs-e2e] rename/remove ✓")

        # ---- 4) 路径别名：/home/user ≡ /sandbox ≡ 相对路径 ----
        fs.write("/home/user/alias.txt", "alias-check")
        assert fs.read("/sandbox/alias.txt") == "alias-check"
        assert fs.read("alias.txt") == "alias-check"
        try:
            fs.read("/")  # 目录不可 read（正确行为：抛 InvalidArgument）
            assert False, "read 目录应抛异常"
        except Exception:
            pass
        print("[fs-e2e] 路径别名 /home/user ≡ /sandbox ≡ 相对路径 ✓")

        # ---- 5) 双向一致性：文件 API ⇄ kernel 同一视图 ----
        fs.write("/home/user/from_api.py", "VALUE = 20260728\n")
        ex = sbx.run_code("print(open('/sandbox/from_api.py').read().strip())")
        out = "".join(ex.logs.stdout).strip()
        assert out == "VALUE = 20260728", f"kernel 读文件 API 写入的文件失败：{out!r} (err={ex.error})"
        print("[fs-e2e] files.write → kernel open() 一致 ✓")

        sbx.run_code("open('/sandbox/from_kernel.txt', 'w').write('kernel-was-here')")
        assert fs.read("/home/user/from_kernel.txt") == "kernel-was-here"
        print("[fs-e2e] kernel open('w') → files.read 一致 ✓")

        # ---- 6) 逃逸防护与 POC 边界 ----
        for bad in ["/../etc/passwd", "/sandbox/../../escape.txt"]:
            try:
                fs.read(bad)
                sys.exit(f"[fs-e2e] 逃逸路径 {bad} 未被拒绝——BUG")
            except Exception as e:
                print(f"[fs-e2e] 逃逸 {bad} -> {type(e).__name__} ✓")

        try:
            fs.watch_dir("/home/user")
            sys.exit("[fs-e2e] watch_dir 未报 unimplemented——BUG")
        except Exception as e:
            msg = str(e)
            assert "unimplemented" in msg or "not supported" in msg or "not implemented" in msg, \
                f"watch_dir 错误不符合预期：{msg}"
            print(f"[fs-e2e] watch_dir -> unimplemented（POC 边界）✓")

        print(
            "\n[fs-e2e] PASS ✅ B11 文件 API 全链路跑通："
            "REST+connect 双协议、kernel 双向一致、别名映射、逃逸防护"
        )
    finally:
        sbx.kill()
        print("[fs-e2e] sandbox killed")


if __name__ == "__main__":
    main()
