# Reference — 精确规范

参考文档：API、字段、命令、契约的精确描述，追求准确而非叙述性。
新文档用小写 `kebab-case.md` 命名并登记于本索引。

| 文档 | 内容 |
|---|---|
| [actor-lifecycle-flows.md](actor-lifecycle-flows.md) | Actor 生命周期关键流程精确参考：Resume/Suspend/Pause/parking/golden 逐步序列 + 端口/Header/键契约 |
| [e2b-api-surface.md](e2b-api-surface.md) | E2B REST 接口面 × `cmd/e2bgw` 覆盖矩阵（代码核实）：端点→Control RPC 映射、状态/错误码映射、运行参数 |
| [upstream-evolution-research.md](upstream-evolution-research.md) | upstream 演进研究（xuanji 演进支持）：漂移主题聚类、新子系统代码级解读、in-flight 分支、rebase 冲突风险与吸收清单 |
| [xuanji-evolution-survey.md](xuanji-evolution-survey.md) | xuanji 分支演进调研：三阶段框架、试点解剖（e2bgw/wasm）、preflight M4 管线、阶段转换决策点 |

upstream 参考性文档位于 `docs/` 根部（api-guide.md、api-style-guide.md、code-style-guide.md、glossary.md），索引见根 `README.md`。
