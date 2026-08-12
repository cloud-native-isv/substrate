# Agent 沙箱竞品格局与 Substrate 定位（概念）

> 面向对标分析的概念文档（What & Why）：梳理开源/商业 Agent 代码沙箱的隔离技术、启动恢复性能与
> E2B 兼容策略，据此定位 Agent Substrate。**本文仅采信公开来源**（各项目 GitHub README 与公开技术媒体），
> 性能数字均为厂商公开披露值，随版本变化，引用前请回查原始来源。
>
> 相关：Substrate 自身架构见 [../../ARCHITECTURE.md](../../ARCHITECTURE.md) 与 [../overview.md](../overview.md)；
> 核心概念见 [core-concepts.md](core-concepts.md)；E2B 兼容网关见 [../reference/e2b-api-surface.md](../reference/e2b-api-surface.md)。

---

## 1. 为什么关注这个赛道

Agent 类工作负载需要一个**安全隔离、可秒级挂起/恢复、能高密度复用**的代码执行底座。这一层已形成一批
开源与商业方案，彼此在三点上竞争：隔离强度（安全边界）、启动/恢复延迟（成本与体验）、生态兼容（能否直接
被现有 SDK 驱动）。理解它们的取舍，有助于校准 Substrate 的设计选择。

## 2. 隔离技术谱系

各方案的隔离手段可归为几类，安全强度与开销递增：

- **用户态内核**（gVisor `runsc`）：拦截 syscall，无需硬件虚拟化，启动快、开销低。
- **microVM**（Firecracker / Cloud Hypervisor + Kata）：每沙箱独立内核，硬件级隔离，需 `/dev/kvm`。
- **内核原语组合**（netns + Landlock + seccomp + 权限降级）：OS 级强制控制，轻量但依赖内核特性。

Substrate 自身支持 gVisor 与 microVM 两条沙箱类，xuanji 分支另加 **wasm** 类（外部 ateom-wasmd 运行时）。

## 3. 主要开源方案对比

| 项目 | 隔离技术 | 启动/恢复（公开披露） | E2B 兼容 | 形态 | 状态 |
|---|---|---|---|---|---|
| **E2B** | microVM（技术未详细披露） | 暂停环境恢复约 1s | 原生（自身即 E2B） | 开源基础设施 + 云服务；Python/JS SDK + Code Interpreter | 活跃 |
| **AgentENV**（kvcache-ai / Kimi） | Firecracker microVM + overlaybd + ublk | resume/boot <50ms，pause <100ms，快照 <100ms | E2B 兼容 HTTP API（`E2B_API_URL` 直接切换，SDK 零改动） | 开源；为 Kimi K3 agentic RL 训练供能；`aenv` CLI | 活跃（2026-07 开源）；当前无鉴权，仅限可信网络 |
| **Daytona** | 每环境独立内核/文件系统/网络栈 | 沙箱创建 <90ms | 自有 SDK/API/CLI | 开源平台；OCI/Docker 兼容 + 状态快照 | **2026-06 起核心开发转私有仓，公开仓停维** |
| **OpenSandbox**（阿里巴巴开源） | 可选 gVisor / Kata / Firecracker | 未统一披露 | 自有 Sandbox Protocol（OSEP 提案机制） | 开源通用沙箱平台；多语言 SDK + osb CLI + MCP；Docker/K8s 运行时 | 活跃；CNCF Landscape |

商业/闭源横评补充（公开对比文，2026）：Modal 用 gVisor、亚秒级调度；Cloudflare 单 VM 内 Ubuntu 容器
+ Workers/Durable Objects/Containers 三层；Vercel 用 Firecracker microVM、毫秒级启动。计费普遍为按秒/按活跃
用量。

## 4. 两种生态策略：E2B 兼容 vs 自有协议

- **E2B 兼容**（AgentENV、以及 Substrate 的 `cmd/e2bgw`）：暴露 E2B REST 接口，让现有 E2B SDK 零改动接入，
  快速复用成熟客户端生态。代价是受 E2B 接口面约束。
- **自有协议**（OpenSandbox 的 Sandbox Protocol/OSEP、Daytona 的 SDK）：以 Protocol First 定义生命周期与执行
  API，可扩展自定义运行时，但需要自建/推广 SDK 生态。

## 5. Agent 沙箱的差异化诉求

对比传统工作负载，Agent 沙箱有一条独特诉求：**不仅是与其他负载互不干扰，更要能在"自己搞坏环境"之后
回到之前某个状态继续工作**——即环境快照/回滚成为一等能力，而非附属特性。这解释了为何本赛道普遍把
snapshot/fork 与秒级 resume 作为核心指标，也印证了 Substrate 以全状态快照实现亚秒级 suspend/resume 的方向。

## 6. Substrate 的定位

在上述格局中，Agent Substrate 的差异点：

- **Kubernetes 原生、控制面移出关键路径**：把大量 actor 映射到较小 worker 池，声明式配置走 CRD、
  高频实例状态走数据库（Valkey），针对"单集群十亿级 agent、高频唤醒"的规模目标设计（见 core-concepts §1）。
- **高密度多路复用**：利用 agent 大部分时间空闲的特性重度复用 worker，而非一沙箱一常驻实例。
- **多沙箱类可插拔**：gVisor / microVM / wasm 共用同一 ateom proto 缝，隔离强度可按场景选择。
- **E2B 兼容作为接入面**：`cmd/e2bgw` 提供 E2B REST 兼容层（覆盖矩阵见 e2b-api-surface.md），
  在不绑定 E2B 实现的前提下复用其 SDK 生态。

> 取舍小结：AgentENV 证明了 Firecracker + 增量快照可把 resume 压到 <50ms 的单机密度路线；Substrate 走的是
> K8s 集群编排 + 控制面旁路的规模路线，两者在"秒级恢复 + 高密度"目标上一致，实现层次不同。
