#!/usr/bin/env python3
"""跨档搬家：把一个 wasm sandbox 的文件全量搬到另一个容量档位的新 sandbox。

「档位」= WorkerPool 资源档（见 manifests/xuanji/wasm-tiers-example.yaml）：
E2B template 名选档（python-interpreter → 2Gi/4 命名 context，
python-interpreter-xl → 8Gi/24 命名 context）。存量 sandbox 不能原地变配
（与官方 E2B 一致，template 创建时钉死），跨档 = 换一个新 sandbox。

wasm 类的 ColdBoot 语义使这一操作**相对类语义无损**：解释器变量本就不跨
pause/resume 存活，搬家丢失的只是它本来就会丢的东西；文件全量保留——这是
wasm 相对 gvisor/microvm（快照含 guest 内存，跨模板迁移必然语义降级）的
结构性差异。

实现只用公开 E2B 面（create / files.list / files.read / files.write /
files.make_dir / kill），零协议偏离；SDK 用户可直接 import migrate() 抄用。

环境变量前置（与 test_files_api_e2e.py 相同）：
  E2B_DOMAIN / E2B_API_KEY / E2B_VALIDATE_API_KEY=false / SSL_CERT_FILE

用法：
  python3 migrate_sandbox.py <sandbox_id> --template python-interpreter-xl
  可选：--timeout 300（新 sandbox TTL）、--keep-src（保留旧 sandbox 不 kill，
  闲置后由 e2bgw TTL 自动挂起）

限制（如实声明）：
  - 仅文件迁移；运行中的解释器状态不迁移（ColdBoot 语义下无此期望）；
  - 源与目标都应是 wasm 类模板（files 路径按 /sandbox 视图原样搬运）；
  - 复制期间源 sandbox 不应有并发写入（无一致性快照，逐文件读写）。
"""
import argparse
import os
import sys
import time

from e2b_code_interpreter import Sandbox

ROOT = "/sandbox"


def _require_env() -> None:
    if not os.environ.get("E2B_DOMAIN"):
        sys.exit("[migrate] 缺少 E2B_DOMAIN")
    if not os.environ.get("E2B_API_KEY"):
        sys.exit("[migrate] 缺少 E2B_API_KEY（租户 JWT）")
    if not os.environ.get("SSL_CERT_FILE"):
        print("[migrate] 警告：SSL_CERT_FILE 未设（自签 CA 场景会 TLS 失败）")


def _walk(fs, root: str = ROOT):
    """BFS 遍历，产出 (path, is_dir)。目录先于其内容产出（父先于子），
    用 depth=1 逐层列举而非一次大 depth——不猜服务端递归上限。"""
    queue = [root]
    while queue:
        cur = queue.pop(0)
        for entry in fs.list(cur, depth=1):
            kind = entry.type.value if getattr(entry, "type", None) else None
            if kind == "dir":
                yield entry.path, True
                queue.append(entry.path)
            elif kind == "file":
                yield entry.path, False


def migrate(
    src: Sandbox,
    template: str,
    *,
    timeout: int = 300,
    keep_src: bool = False,
) -> Sandbox:
    """把 src 的 /sandbox 文件树全量复制到 template 档位的新 sandbox。

    成功返回新 Sandbox（复制已校验：目标文件集合 = 源文件集合）；复制或校验
    失败时 kill 新 sandbox 并抛异常，源 sandbox 原样保留（先立后破）。
    默认迁移后 kill 源；keep_src=True 时保留（闲置后 TTL 自动挂起）。
    """
    t0 = time.monotonic()
    dst = Sandbox.create(template=template, timeout=timeout)
    print(f"[migrate] dst sandbox = {dst.sandbox_id} (template={template})")
    try:
        n_dirs = n_files = n_bytes = 0
        src_files = set()
        for path, is_dir in _walk(src.files):
            if is_dir:
                dst.files.make_dir(path)
                n_dirs += 1
            else:
                data = bytes(src.files.read(path, format="bytes"))
                dst.files.write(path, data)
                src_files.add(path)
                n_files += 1
                n_bytes += len(data)

        # 校验：目标文件集合与源一致（make_dir/write 自身失败会先抛异常，
        # 这里兜的是路径视图不一致之类的静默偏差）
        dst_files = {p for p, d in _walk(dst.files) if not d}
        if dst_files != src_files:
            raise RuntimeError(
                f"迁移校验失败：目标 {len(dst_files)} 个文件 != 源 {len(src_files)} 个；"
                f"缺失 {sorted(src_files - dst_files)[:5]}"
            )
    except BaseException:
        dst.kill()  # 先立后破：目标未就绪就回滚，源不动
        raise

    took = time.monotonic() - t0
    print(
        f"[migrate] copied {n_files} files / {n_dirs} dirs / {n_bytes} bytes "
        f"in {took:.2f}s"
    )
    if not keep_src:
        src.kill()
        print(f"[migrate] src sandbox {src.sandbox_id} killed")
    else:
        print(f"[migrate] src sandbox {src.sandbox_id} kept（闲置后 TTL 自动挂起）")
    return dst


def main() -> None:
    ap = argparse.ArgumentParser(description="跨档搬家：文件全量迁移到新档位 sandbox")
    ap.add_argument("sandbox_id", help="源 sandbox ID（sbx-…）")
    ap.add_argument("--template", required=True, help="目标档位的 ActorTemplate 名")
    ap.add_argument("--timeout", type=int, default=300, help="新 sandbox TTL 秒数")
    ap.add_argument("--keep-src", action="store_true", help="保留源 sandbox（默认迁移后 kill）")
    args = ap.parse_args()

    _require_env()
    src = Sandbox.connect(args.sandbox_id)
    print(f"[migrate] src sandbox = {args.sandbox_id}")
    dst = migrate(src, args.template, timeout=args.timeout, keep_src=args.keep_src)
    print(f"[migrate] PASS ✅ 新 sandbox：{dst.sandbox_id}")


if __name__ == "__main__":
    main()
