# 异常现象与应对策略

Known Issues & Mitigations — 常见 git 工作流异常的根因分析与应对。

| 异常现象 | 根因 | 应对策略 |
|----------|------|----------|
| `git checkout` 报错，本地改动会被覆盖 | 工作区不干净 | 先完成前置校验再切换分支 |
| rebase 后变为 `M N`（双向分叉） | 共享分支 rebase 重写历史 | `--force-with-lease` 受控推送，走团队同步窗口 |
| `skipped previously applied commit` | 分支存在重复补丁或历史漂移 | 记录 commit id，继续 rebase，`git log --left-right --cherry-pick` 差异核对 |
| 选择性合并后排除路径仍被修改 | `.gitexcludes` 未存在或子程序未调用 | 确认目标分支存在 `.gitexcludes`，重新执行同步并确保子程序正常运行 |
| `git rm --cached` 后本地文件消失 | 某些 Git 版本行为差异 | 从工作区恢复（`git checkout HEAD -- <path>`）或从 stash 恢复 |
| `.gitexcludes` 文件被同步覆盖 | 子程序未将 `.gitexcludes` 加入隐含保护 | 检查子程序后置是否包含 `git checkout _gitexcludes_pre_sync -- .gitexcludes` |
