---
id: "20260812T032136Z-speckit-instructions"
unit_id: "/speckit.instructions"
unit_type: "command"
run_id: "full-reconcile-20260812T032058Z"
scope: "local"
partial: false
created: "2026-08-12T03:21:36Z"
summary: "全量 reconcile：setup 脚本非破坏性确认（备份与 base 逐字节相同），7 处收敛全部是事实级修正（三阶段定位、+5 文档地图行、去掉外机绝对路径、补 cmd/e2bgw、Testing rules 对齐新原则 VI、去掉错误的 Principle XII 引用、Framework Map 三处路径漂移），补入模板新增的 ## Git Workflow 段；其余 12 个段落经核对"
---

## Review
全量 reconcile：setup 脚本非破坏性确认（备份与 base 逐字节相同），7 处收敛全部是事实级修正（三阶段定位、+5 文档地图行、去掉外机绝对路径、补 cmd/e2bgw、Testing rules 对齐新原则 VI、去掉错误的 Principle XII 引用、Framework Map 三处路径漂移），补入模板新增的 ## Git Workflow 段；其余 12 个段落经核对留在容忍带内未动。覆盖检查确认删除的 10 行全部是有意替换的过时事实，无用户手写内容丢失；术语表补 5 条（19 条）。

## Optimization Points
- 命令要求"scripted existence check"逐行校验 Documentation Map / Framework Map 路径，但没有提供引擎；本次靠手写 for 循环发现了 3 处路径漂移（.specify/teams/、.specify/specs/ 不存在，.specify/docs/ 未登记）。建议提供 instructions-utils.py --action validate 做路径存在性 + 数值事实（Feature Count vs features/*.md 计数）+ 保留标记完整性的确定性校验，避免每次靠即席脚本。
- Tech Stack 的 "Root Path" 字段天生会漂移：它记录的是首次生成时那台机器的绝对路径（本次为 /Users/liuqiming.lqm/...，与当前 /cws_work/substrate 无关），对任何其他 checkout 都是错的。建议模板取消该字段，或明确写成"仓库检出目录（仓库相对路径）"。
- 本次最有价值的收敛来自"宪法刚被修订"这一上游变化：Repository Guide 的 Testing rules 仍写着"无测试不得合入"，与新原则 VI 直接冲突。命令的 auto-update 列表（Documentation Map / Framework Map / Tech Stack / Key Directories / Build-Test）并未包含"与宪法冲突的表述"，容易被判为 hand-authored 而放进容忍带。建议把"与 constitution 的一致性"显式列为必查维度。
- 命令未提示"upstream 与本分支规则不同"的表述该如何处理：Testing rules 源自 upstream AGENTS.md，而本分支刻意放宽。本次做法是保留 upstream 事实并标注 xuanji 放宽点；建议模板给出这种 fork 场景的书写约定。
