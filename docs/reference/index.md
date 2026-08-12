# Reference — 精确规范

参考文档：API、字段、命令、契约的精确描述，追求准确而非叙述性。
新文档用小写 `kebab-case.md` 命名并登记于本索引。

| 文档 | 内容 |
|---|---|
| [actor-lifecycle-flows.md](actor-lifecycle-flows.md) | Actor 生命周期关键流程精确参考：Resume/Suspend/Pause/parking/golden 逐步序列 + 端口/Header/键契约 |
| [e2b-api-surface.md](e2b-api-surface.md) | E2B REST 接口面 × `cmd/e2bgw` 覆盖矩阵（代码核实）：端点→Control RPC 映射、状态/错误码映射、运行参数 |

upstream 参考性文档位于 `docs/` 根部（api-guide.md、api-style-guide.md、code-style-guide.md、glossary.md），索引见根 `README.md`。
