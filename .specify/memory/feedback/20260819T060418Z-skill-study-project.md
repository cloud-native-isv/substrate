---
id: "20260819T060418Z-skill-study-project"
unit_id: "skill:study-project"
unit_type: "skill"
run_id: "20260819-evolution-study"
scope: "local"
probe: "skill-study-project-wrapup"
kind: "internal"
slice: "skills"
partial: false
created: "2026-08-19T06:04:18Z"
summary: "演进导向双文档研究（upstream 演进研究 + xuanji 演进调研）：3 并行子代理（e2bgw/wasm+preflight/upstream drift）+ 38 提交漂移考古 + 3 张 PlantUML 图 + 双文档登记（reference index/ARCHITECTURE/xuanji.md/CHANGELOG）+ Hugo 构建验证。交叉验证通过（durableDir C"
---

## Review
演进导向双文档研究（upstream 演进研究 + xuanji 演进调研）：3 并行子代理（e2bgw/wasm+preflight/upstream drift）+ 38 提交漂移考古 + 3 张 PlantUML 图 + 双文档登记（reference index/ARCHITECTURE/xuanji.md/CHANGELOG）+ Hugo 构建验证。交叉验证通过（durableDir CEL 删除、tenant_id claim 不一致、ensure-steps 重构均经代码核实）。

## Optimization Points
- study-project 默认产物路径 docs/overview.md 与「仓库已存在上一轮 overview.md」冲突：本次需额外向用户确认覆盖/新建。建议技能在 Phase 1 增加既有产物探测（docs/overview.md 是否已存在且被引用），并在冲突时默认提供「新建姊妹文档 + 登记」选项，减少一轮问答。
- 对 fork 型仓库（存在 upstream remote + 定制分支），标准模块深剖模板不覆盖「upstream 漂移考古」（baseline..origin/main 主题聚类、in-flight 分支、rebase 冲突面）。建议 Phase 3/6 增加 fork-drift 分析 lane：以 xuanji.md 类登记表 × upstream 触碰提交做冲突矩阵，本次该 lane 产出了报告最高价值章节。
