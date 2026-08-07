# Notes（临时区）

notes 是 docs 空间唯一的临时区：每篇 note 必须声明生命周期，成熟后合入正式区
（concepts/ tutorials/ tasks/ reference/ decisions/ contribute/），超期后按下述状态机处理。

状态机：draft →（合入 target）→ archived；draft →（超期）→ expired；
expired →（续期）→ draft；expired →（人工确认）→ deleted（notes 区是唯一允许确认后真删除的区域）。

自动化命令（确定性引擎，可脱离会话重复执行）：

```bash
python3 .specify/scripts/python/docs-utils.py --action scan --root .            # 分组报告 + invalid 修复建议
python3 .specify/scripts/python/docs-utils.py --action expire --root .          # 超期 draft → expired（绝不删除）
python3 .specify/scripts/python/docs-utils.py --action clean --root .           # 删除候选 dry-run
python3 .specify/scripts/python/docs-utils.py --action clean --yes --root .     # 人工确认后的真删除（仅 notes 区）
python3 .specify/scripts/python/docs-utils.py --action archive-check --root .   # 归档完整性（target 必须存在）
```

每篇 note 必须携带如下 frontmatter：

```yaml
---
title: "<one-line title>"
created: YYYY-MM-DD
expires: YYYY-MM-DD    # required; default = created + 60 days
status: draft          # draft | expired | archived
target: ""             # intended formal destination, required when archived
tags: []
---
```

暂无 note。
