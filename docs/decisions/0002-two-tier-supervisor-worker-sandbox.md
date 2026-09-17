# 0002. 两层沙箱模型：Worker Pod 面（Kubernetes+Substrate+Containerd+runc/rund）+ Actor 工作负载面（e2b+自研管控+WebAssembly）

- Status: Proposed
- Date: 2026-09-16
- 术语归一: 2026-09-17（概念全面对齐上游 substrate 词汇，见下节）
- Supersedes: -
- Superseded by: -

## 术语归一（2026-09-17，对齐上游 substrate 定义）

本决策早期版本使用自造实体名（supervisor sandbox / worker sandbox / session）。经与上游
`docs/architecture.md`（@85ce8ed5）的定义核对，全部概念归一到 substrate 词汇；**两层
（Tier-1/Tier-2）只保留为平面分工的描述，不再作为实体名**：

| 旧术语（ADR 早期版） | 归一后术语 | 上游锚点 |
|---|---|---|
| Supervisor sandbox（会话级 rund/kata 容器） | **Worker Pod**（sandbox class `supervisor`） | Worker/WorkerPod：温沙箱 Pod，host 一个 RUNNING actor |
| Supervisor pool | **WorkerPool**（`sandboxClass: supervisor`） | WorkerPool CRD（名称本就一致） |
| Session（绑定到 supervisor 的会话） | **Actor**（= 会话实例；E2B sandbox id = actor name） | Actor：有状态实例，拥有快照，suspend/resume 的对象 |
| Worker sandbox（wasm 实例） | 拆为两层：**actor 的执行体 = wasm sandbox**（worker pod 内嵌套隔离层）；**请求级 workload 执行（context）**（用过即销毁） | sandbox = worker pod 的隔离环境；`RunWorkload` = 一次工作负载执行 |
| ateom-supervisor 守护进程 | **ateom**（supervisor class 的 herder，实现仍名 ateom-supervisor） | ateom-\<class\>：worker pod 内的沙箱 herder |
| SessionConfig（会话配置） | **ActorTemplate**（不可变版本定义）+ actor 级配置注入（xuanji 扩展字段 `sessionConfigRef`） | ActorTemplate；可变 per-actor 配置为 xuanji 扩展 |
| capability 表 / egressPolicy | ActorTemplate 的 xuanji 扩展字段（`capabilities` / `egressPolicy`），ateom 在 workload 执行 spawn 时强制执行 | 上游无对应物（本仓安全模型扩展） |
| suspend = supervisor 的 VM 快照 | **SuspendActor = actor 级 checkpoint**（wasm 状态 + workspace tar，经 `CheckpointWorkload`），**worker pod 归还池** | 上游快照模型：快照属于 actor，worker 是无状态温资源 |
| 休眠/唤醒 | **SuspendActor / ResumeActor** | 上游生命周期动词 |

归一带来的一个实质性修正：早期版把 suspend 定义为「supervisor 容器 VM 快照」，偏离上游
（上游从不快照 worker pod，只快照 actor 的沙箱状态并归还 worker）。归一后 suspend =
actor 级 checkpoint，worker pod 回池复用——池效率与上游语义同时改善；会话级服务（出口代理、
capability broker）是 ateom 的组成，resume 时由控制面状态（capability 表、sessionConfigRef）
重建，不进快照。

## Context

### 现状（xuanji 分支单层 wasm 模型）

当前 wasm SandboxClass 是**单层、集群级池化**模型：

- `WorkerPool(sandboxClass: wasm)` 池化一组 worker pod（ateom-wasmd：wasmtime 内嵌 + 持久 python kernel 池）；
- Actor（≈会话）多路复用到池内 wasm kernel 上，suspend/resume 以 wasm kernel 快照承载；
- 隔离边界 = wasm 线性内存 + WASI 能力模型；但**网络出口是 pod 级 ambient 权限**（NetworkPolicy 粒度到池，不到 actor），**配置是集群级 SandboxConfig assets**，**存储无 actor 级边界**；
- 多个 actor 共享同一个 worker pod 的进程环境、出口网络与节点视图。

该模型密度高、冷启动快，但缺少一个 **actor 级的配置/网络/存储承载体**：actor 身份、actor 配置、工作区、出口策略都没有自然的归属点，只能下沉到集群级或上浮到控制面内存态。

### 启示（AgentForce 数字员工沙箱，2026-09-16 会话档案）

数字员工（DE）沙箱底座为 kata container（virtiofs）+ OSS/NAS 持久面，其运行模型经实测验证：

1. **一个 container 作为主体**保存全部配置与基础信息（agent HOME、awareness、harness 声明）；
2. **每个用户请求启动一个对应进程**（Claude Code）处理该请求；
3. **请求间完全隔离、用过即销毁**；配套 per-session mount namespace 遮蔽、技能按会话 force re-link、出口 MITM 强管、agentshell 全事件审计；
4. **配置权威在平台侧**，经请求级下行通道注入，沙箱内不自持配置真源。

### 问题陈述

WebAssembly 不应再作为**独立自足**的沙箱 class 存在：它擅长请求级的高密度、亚毫秒执行隔离，但不擅长承载 actor 级的网络身份、存储挂载与配置生命周期。需要一种**重量级 sandbox class 的 worker pod** 承担 DE 主容器角色（rund/kata），actor 的执行体退居 worker pod 内的 **wasm sandbox 嵌套隔离层**（请求级 workload 执行用过即销毁），并以 **Capability-based security** 把 actor 级配置与网络访问能力显式授予每次 workload 执行。

## Decision

整个 sandbox 体系分成**两个平面**，wasm 从"顶层可池化 SandboxClass"降级为"worker pod 内的嵌套隔离层 + workload 运行时语义"：

> **平面一（Tier-1，worker pod 面）**：Kubernetes + Substrate + Containerd + runc/rund 构建**基于 Kubernetes 生态的 agent sandbox worker pod**（sandbox class `supervisor`）。特性要求：**复用 Kubernetes 生态**、**低频生命周期 + 高稳定性**、提供**休眠唤醒（Suspend/ResumeActor）**等 agent sandbox 基础特性。
>
> **平面二（Tier-2，actor 工作负载面）**：**e2b + 自研管控 + wasm workload**（跑在 worker pod 内部），以 WebAssembly 沙箱提供**高频创建能力、隔离机制、审计机制以及 Capability-based security 的安全模型**。

### D1. Tier-1 Worker Pod（sandbox class `supervisor`）

- 技术栈：**Kubernetes（编排/调度/网络/存储/RuntimeClass）+ Substrate（Actor/WorkerPool/Ateom/suspend-resume）+ Containerd + runc/rund**；worker pod 形态为 **rund（kata）container**（ACK RuntimeClass `rund`；无 rund 的环境回退 runc/microvm class 的 kata 形态）；
- **一个 worker pod 同一时刻 host 一个 RUNNING actor**（上游 Worker 语义：IDLE/BUSY）；actor suspend 后 worker pod 擦除归还池；
- 特性契约：
  - **复用 Kubernetes 生态**——不另建调度/网络/存储/镜像体系，worker pod 就是一个 K8s Pod（kata 隔离），CNI/CSI/RuntimeClass/镜像仓库全部复用；
  - **低频生命周期 + 高稳定性**——worker pod 的 provisioning 与 actor 的 resume/suspend 编排是低频事件，追求长驻稳定而非高频周转；
  - **休眠唤醒等 agent sandbox 基础特性**——复用 substrate 现有 Suspend/ResumeActor 机制：suspend = actor 级 checkpoint（wasm 状态 + workspace flush）+ worker pod 归还池；resume = 从池取温 worker pod + `RestoreWorkload` + 注入 actor 配置与 capability 表；
- 承载：actor 工作区存储（挂载卷）、网络出口预配置（NetworkPolicy/路由到 worker pod）；pod 内的 ateom（supervisor class herder）同时承载 Tier-2 的 wasm host 运行时、出口代理与 capability broker；
- 池化：`WorkerPool(sandboxClass: supervisor)` 池化**温 worker pod**（无 actor 绑定）。

### D2. Tier-2 Actor 工作负载面（e2b + 自研管控 + wasm workload）

- 组成：**e2b 协议面**（e2bgw，E2B-compatible REST 入口，E2B sandbox ↔ actor）+ **自研管控**（请求面控制逻辑：请求路由、capability 表下发、审计归集；构建在 substrate 控制面之上）+ **wasm workload**（actor 的执行体，跑在 worker pod 内）；
- **actor 的执行体 = wasm sandbox**：ateom（supervisor class）在 worker pod 内以 wasmtime host 承载 actor 的 wasm 沙箱（python-wasm 或用户模块的 kernel 池）；actor RUNNING 期间执行体常驻，suspend 时随 actor 一起 checkpoint；
- **请求级 workload 执行（context）**：每个请求在 actor 的 wasm 沙箱内 spawn 一个**用过即销毁**的执行单元（DE 每请求进程的对应物），capability 子集在 spawn 时授予；
- **WebAssembly 沙箱承担四项核心机制**（安全与执行语义收敛在 wasm 层，即 ateom 内，而非 worker pod 层独立组件）：
  1. **高频创建能力**——请求级 context 亚毫秒 spawn/destroy，支撑请求级高频周转（与 Tier-1 低频分工）；
  2. **隔离机制**——请求间完全隔离：wasm 线性内存边界隔离 context ↔ 同 actor 其他 context，叠加 kata VM 边界隔离 worker pod ↔ 宿主/邻 actor，两道防线职责分明；
  3. **审计机制**——ateom 对每次 workload 执行产生全事件审计流（类比 DE 的 agentshell）：spawn/destroy、host function 调用、出口请求、capability 使用均可审计；
  4. **Capability-based security 安全模型**——workload 执行的全部权限来自 spawn 时授予的 capability 子集（见 D3），host function 是唯一执行点；
- 生命周期：context **请求结束即销毁**，不做快照/恢复；跨请求状态只允许显式写回 actor 工作区（workspace 卷），context 自身无持久态。

### D3. Capability-based security 授权模型

- actor capability 表由控制面（Tier-2 自研管控）在 ResumeActor 时下发、worker pod 内的 **ateom** 持有；capability 为**不可伪造句柄**，workload 执行 spawn 时仅获得授予子集。**执行点收敛在 wasm host function**——出口代理与 capability 校验都是 ateom（wasm host 运行时）的一部分，不是 worker pod 层独立组件：
  - `cap:fs:actor-config:ro` —— actor 配置文件只读句柄（WASI preopen；原 cap:fs:session-config）；
  - `cap:fs:workspace:rw:<subdir>` —— actor 工作区子目录读写句柄（per-request 遮蔽，类比 DE 的 mount namespace masking）；
  - `cap:net:egress:<token>` —— 出口能力令牌：wasm 执行**无直接 socket 权限**，一切出网经 host function 到 ateom 的出口代理，代理校验令牌内 allowlist（类比 DE 出口 MITM，但粒度为 capability 令牌而非 ambient MITM）；
  - `cap:secret:<name>` —— 命名凭据句柄（可选，经 ateom 代取，不下明文进 wasm 内存之外的通道）。
- 未授予即不可见：workload 执行默认零权限，与 WASI 能力模型同构但把"能力来源"从镜像/池配置提升到 **actor 级显式授予**。

### D4. API 语义映射（两层平面分工）

- **平面分工**：substrate 控制面（ateapi/atecontroller/Ateom）服务 **Tier-1**（actor/worker 生命周期：Create/Resume/Suspend/Delete、池绑定）；**e2b 协议面（e2bgw）+ 自研管控**服务 **Tier-2 请求面**（E2B-compatible 请求入口、请求路由、capability 表下发、审计归集）。自研管控构建在 substrate 之上，不重造 actor 生命周期；
- **Actor = 会话**（E2B sandbox id = actor name）：ResumeActor = 取温 worker pod + `RestoreWorkload` + 注入 actor 配置与 capability 表；SuspendActor = actor 级 `CheckpointWorkload`（wasm 状态 + workspace flush）+ worker pod 归还池；context 永不 suspend；
- **SandboxClass 语义重定义**：`wasm` 不再作为可独立池化的顶层 class，仅保留为 **worker pod 内的嵌套隔离层 / workload 运行时标识**；新增顶层 class `supervisor`（runtime: rund/kata）作为可池化 worker pod 形态；枚举变更按 xuanji 约定登记 xuanji.md；
- **WorkerPool / ActorTemplate 用上游既有字段表达**：`WorkerPool.spec.sandboxClass: supervisor`、`ActorTemplate.spec.sandboxClass: supervisor`（上游字段，新枚举值）；xuanji 扩展字段仅四个：`workloadRuntime: wasm`、`capabilities: [...]`、`egressPolicy: {...}`、`sessionConfigRef`（actor 级配置注入，平台侧权威、请求级可刷新，类比 DE 的请求级 PATCH /role）；
- **Ateom 协议演进**：supervisor class 的 `RunWorkload` 语义 = 一次请求执行——携带 capability manifest 调用 ateom，ateom 在 actor 的 wasm 沙箱内 spawn context、执行、销毁、返回；`CheckpointWorkload/RestoreWorkload` 作用域 = actor（wasm 状态 + workspace）。

### D5. 与既有单层模型的关系

- 过渡期保留现有 wasm-pool 部署作为 legacy 路径（`wasm-legacy` 标签），新 actor 默认走两层模型；
- 密度换隔离：单层模型单机数千 wasm 实例的密度让位于"每 RUNNING actor 一个 kata worker pod"的重量级边界；以温池 + actor suspend 归还 worker 缓解成本；
- gvisor/microvm class 不受本决策影响，仍为独立顶层 class。

## Consequences

### 正向

- actor 级配置/网络/存储边界落地，与 DE 实测验证的运维模型同构（请求隔离、用过即销毁、per-session 遮蔽、出口强管）；
- workload 执行爆炸半径收敛到单请求；actor 间不再共享进程环境与出口身份；
- capability 模型把"谁能看什么配置、能出什么网"变成可审计的显式授予（对齐 agentshell 全事件审计诉求）；
- suspend 归一为 actor 级 checkpoint 后，worker pod 成为真正无状态的温资源，池利用率对齐上游模型；
- 术语与上游一致后，回归开源（见 substrate-upstream-regression 评估）时的概念翻译成本归零。

### 代价与风险

- 每 RUNNING actor 一个 kata worker pod：内存/启动成本高于池化 wasm kernel；依赖温池命中率与 suspend 策略；
- kata 沙箱内 actor checkpoint（wasm 状态 + workspace）体积 > 纯 wasm kernel 快照：suspend/resume 延迟与存储成本上升；
- rund RuntimeClass 在目标集群的可用性需逐集群验证（ACK 已具备；自建集群需 kata/rund 部署）；
- Ateom 协议与 CRD enum 变更涉及上游文件改动（xuanji.md 登记）与 ateom-wasmd → ateom-supervisor 的运行时重构（sandbox 仓）；
- capability 令牌的中途吊销、跨请求状态写回语义需在下个 feature 明确。

### 后续行动

1. 概念文档 [two-tier-sandbox-model.md](../concepts/two-tier-sandbox-model.md) 与示例清单 `manifests/xuanji/two-tier-example.yaml`（本 ADR 配套，已同步归一术语）；
2. sandbox 仓：ateom-wasmd 演进为 supervisor class 的 **ateom herder**（ateom-supervisor：wasm host 运行时 + per-request context spawn/destroy + 出口代理 + capability 校验 + 全事件审计流）；e2bgw 侧补齐请求面自研管控（capability 表下发、审计归集）；
3. substrate 仓：SandboxClass enum 增 `supervisor`、`wasm` 语义降级；ActorTemplate/WorkerPool schema 增补四个扩展字段；Ateom proto 演进；
4. 集群验证：cluster-msaFE8 上以 rund RuntimeClass 拉起 supervisor class worker pool，跑 actor 级 e2e（ResumeActor 注入 → 请求执行 → 出口 allowlist 生效 → Suspend/Resume）。

## 开放问题

- worker pod 空闲时 actor suspend 的粒度与唤醒延迟预算（actor 体验 vs 成本）；
- capability 中途吊销的传播机制（proxy 侧即时生效 vs context 生命周期内冻结）；
- actor 工作区的配额与回收（DE 用 OSS/NAS 持久面；本模型默认 rustfs/OSS，配额策略待定）；
- 单 worker pod 内并发 context 上限与燃料/内存预算的 actor 级配额模型。
