# Concepts — What & Why

概念文档：解释系统的概念是什么、为何这样设计。
新文档用小写 `kebab-case.md` 命名并登记于本索引。

| 文档 | 内容 |
|---|---|
| [core-concepts.md](core-concepts.md) | Agent Substrate 核心概念详解：Actor/Atespace/Worker/ActorTemplate/快照体系/生命周期动词/DurableDir/停车/身份（upstream 基线） |
| [sandbox-landscape.md](sandbox-landscape.md) | Agent 沙箱竞品格局与 Substrate 定位（公开来源）：隔离技术谱系、启动/恢复性能、E2B 兼容 vs 自有协议 |
| [two-tier-sandbox-model.md](two-tier-sandbox-model.md) | 两层沙箱模型（substrate 词汇归一）：Tier-1 worker pod 面（K8s+Substrate+Containerd+runc/rund，sandbox class supervisor，低频+休眠唤醒）+ Tier-2 actor 工作负载面（actor=会话、执行体=wasm sandbox、请求级 context 用过即销毁；e2b+自研管控、capability 授权与审计）（ADR 0002 配套） |

upstream 概念性文档位于 `docs/` 根部（architecture.md、request-parking.md 等），索引见根 `README.md`。
