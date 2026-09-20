# Concepts — What & Why

概念文档：解释系统的概念是什么、为何这样设计。
新文档用小写 `kebab-case.md` 命名并登记于本索引。

| 文档 | 内容 |
|---|---|
| [core-concepts.md](core-concepts.md) | Agent Substrate 核心概念详解：Actor/Atespace/Worker/ActorTemplate/快照体系/生命周期动词/DurableDir/停车/身份（upstream 基线） |
| [sandbox-landscape.md](sandbox-landscape.md) | Agent 沙箱竞品格局与 Substrate 定位（公开来源）：隔离技术谱系、启动/恢复性能、E2B 兼容 vs 自有协议 |
| [two-tier-sandbox-model.md](two-tier-sandbox-model.md) | 两层沙箱模型（substrate 词汇归一，去 supervisor 语义）：Tier-1 worker pod 面（K8s+Substrate+Containerd+runc/rund，sandbox class wasm 的 kata 形态，低频+休眠唤醒）+ Tier-2 actor 工作负载面（actor=会话、执行体=wasm sandbox、请求级 context 用过即销毁；e2b+自研管控、capability 授权与审计）；不新增 SandboxClass enum（ADR 0002 配套） |
| [trust-domain-panorama.md](trust-domain-panorama.md) | 数字员工 / Cloud Agent 四层信任域全景（ADR 0002–0006 的共同上框）：I 基础设施 / P 平台 / S 服务·租户 / Agent 四层（信任级自外向内递减），Agent 层=传统云没有的「agent 比 owner 更不可信」隔离层；S 域=一个数字员工（**打破 1 worker pod=1 active actor**，多并发 agent 沙箱）；**S 层 MCP 能力后端**（within-S/cross-S 共享池、host function 为调用链一环、ateom 作 MCP 可信中介、零信任不留秘密）；统一原则=每次跨域皆白名单+审计；自洽条件 C1–C3 + 残留张力 R1–R5 |

upstream 概念性文档位于 `docs/` 根部（architecture.md、request-parking.md 等），索引见根 `README.md`。
