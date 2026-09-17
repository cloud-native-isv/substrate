# 0002. 两层沙箱模型：Supervisor Sandbox（Kubernetes+Substrate+Containerd+runc/rund）+ Worker 面（e2b+自研管控+WebAssembly）

- Status: Proposed
- Date: 2026-09-16
- Supersedes: -
- Superseded by: -

## Context

### 现状（xuanji 分支单层 wasm 模型）

当前 wasm SandboxClass 是**单层、集群级池化**模型：

- `WorkerPool(sandboxClass: wasm)` 池化一组 ateom-wasmd Pod（wasmtime 内嵌 + 持久 python kernel 池）；
- Actor（≈会话）多路复用到池内 wasm kernel 上，suspend/resume 以 wasm kernel 快照承载；
- 隔离边界 = wasm 线性内存 + WASI 能力模型；但**网络出口是 Pod 级ambient 权限**（NetworkPolicy 粒度到池，不到会话），**配置是集群级 SandboxConfig assets**，**存储无会话级边界**；
- 会话之间共享同一个池 Pod 的进程环境、出口网络与节点视图。

该模型密度高、冷启动快，但缺少一个**会话级的配置/网络/存储承载体**：会话身份、会话配置、会话工作区、出口策略都没有自然的归属点，只能下沉到集群级或上浮到控制面内存态。

### 启示（AgentForce 数字员工沙箱，2026-09-16 会话档案）

数字员工（DE）沙箱底座为 kata container（virtiofs）+ OSS/NAS 持久面，其运行模型经实测验证：

1. **一个 container 作为主体**保存全部配置与基础信息（agent HOME、awareness、harness 声明）；
2. **每个用户请求启动一个对应进程**（Claude Code）处理该请求；
3. **请求间完全隔离、用过即销毁**；配套 per-session mount namespace 遮蔽、技能按会话 force re-link、出口 MITM 强管、agentshell 全事件审计；
4. **配置权威在平台侧**，经请求级下行通道注入，沙箱内不自持配置真源。

### 问题陈述

WebAssembly 不应再作为**独立自足**的沙箱技术存在：它擅长请求级的高密度、亚毫秒执行隔离，但不擅长承载会话级的网络身份、存储挂载与配置生命周期。需要在上层补一个 **supervisor sandbox** 承担 DE 主容器角色，wasm 退居 **worker sandbox** 语义（请求级、用过即销毁），并以 **Capability-based security** 把会话级配置与网络访问能力显式授予 worker。

## Decision

整个 sandbox 体系分成**两个层面**，wasm 从"顶层 SandboxClass"降级为"worker 运行时语义"：

> **层面一（Tier-1）**：Kubernetes + Substrate + Containerd + runc/rund 构建**基于 Kubernetes 生态的 Agent Sandbox**，作为 **supervisor sandbox**。特性要求：**复用 Kubernetes 生态**、**低频生命周期 + 高稳定性**、提供**休眠唤醒**等 agent sandbox 基础特性。
>
> **层面二（Tier-2）**：**e2b + 自研管控 + runc/rund container + WebAssembly 沙箱**（跑在 supervisor sandbox 内部），以 WebAssembly 沙箱提供**高频创建能力、隔离机制、审计机制以及 Capability-based security 的安全模型**。

### D1. Tier-1 Supervisor Sandbox（会话级，Kubernetes 生态面）

- 技术栈：**Kubernetes（编排/调度/网络/存储/RuntimeClass）+ Substrate（Actor/WorkerPool/Ateom/suspend-resume）+ Containerd + runc/rund**；容器形态为 **rund（kata）container**（ACK RuntimeClass `rund`；无 rund 的环境回退 runc/microvm class 的 kata 形态），每**会话**一个实例；
- 特性契约：
  - **复用 Kubernetes 生态**——不另建调度/网络/存储/镜像体系，supervisor 就是一个 K8s Pod（kata 隔离），CNI/CSI/RuntimeClass/镜像仓库全部复用；
  - **低频生命周期 + 高稳定性**——supervisor 的创建/绑定/休眠/唤醒是会话级低频事件，追求长驻稳定而非高频周转；
  - **休眠唤醒等 agent sandbox 基础特性**——复用 substrate 现有 suspend/resume 机制于 supervisor 层：空闲 suspend = VM 快照 + 会话存储 flush；resume = 绑定会话并注入会话配置；
- 承载：会话配置存储（平台侧权威、resume 时注入）、会话工作区存储（挂载卷）、网络出口预配置（NetworkPolicy/路由到 supervisor），并为 Tier-2 提供宿主环境（runc/rund container 本体）；
- 池化：`WorkerPool` 池化**温 supervisor 容器**（无会话绑定），resume 时绑定会话并注入会话配置。

### D2. Tier-2 Worker 面（请求级，e2b + 自研管控 + wasm）

- 组成：**e2b 协议面**（e2bgw，E2B-compatible REST 入口）+ **自研管控**（请求面控制逻辑：请求路由、capability 表下发、审计归集；演进自 substrate 控制面之上的 xuanji 自研部分）+ **runc/rund container**（即 D1 的 supervisor 本体，作为 wasm 沙箱的宿主）+ **WebAssembly 沙箱**（跑在 supervisor sandbox 内部）；
- **WebAssembly 沙箱承担四项核心机制**（安全与执行语义都收敛到 wasm 层，而非 supervisor 层独立组件）：
  1. **高频创建能力**——supervisor 内按**请求**spawn wasm 实例（wasmtime，python-wasm 或用户模块），亚毫秒级实例化，支撑请求级高频周转（与 Tier-1 低频形成分工）；
  2. **隔离机制**——请求间完全隔离、用过即销毁：wasm 线性内存边界隔离 worker ↔ supervisor 内其他 worker，叠加 kata VM 边界隔离 supervisor ↔ 宿主/邻会话，两道防线职责分明；
  3. **审计机制**——wasm host 侧对每次 worker 执行产生全事件审计流（类比 DE 的 agentshell 全事件审计）：spawn/destroy、host function 调用、出口请求、capability 使用均可审计；
  4. **Capability-based security 安全模型**——worker 的全部权限来自实例化时授予的 capability 子集（见 D3），host function 是唯一执行点；
- 生命周期：worker **请求结束即销毁**，不做快照/恢复；跨请求状态只允许显式写回 supervisor 的会话存储，worker 自身无持久态。

### D3. Capability-based security 授权模型

- 会话 capability 表由控制面（Tier-2 自研管控）在 resume 时下发、supervisor 内的 **wasm host 运行时**持有；capability 为**不可伪造句柄**，worker 实例化时仅获得授予子集。**执行点收敛在 wasm host function**——出口代理与 capability 校验都是 wasm host 运行时的一部分（Tier-2 机制），不是 supervisor 层的独立组件：
  - `cap:fs:session-config:ro` —— 会话配置文件只读句柄（WASI preopen）；
  - `cap:fs:workspace:rw:<subdir>` —— 会话工作区子目录读写句柄（per-request 遮蔽，类比 DE 的 mount namespace masking）；
  - `cap:net:egress:<token>` —— 出口能力令牌：wasm 实例**无直接 socket 权限**，一切出网经 host function 到 wasm host 运行时的出口代理，代理校验令牌内 allowlist（类比 DE 出口 MITM，但粒度为 capability 令牌而非 ambient MITM）；
  - `cap:secret:<name>` —— 命名凭据句柄（可选，经 wasm host 运行时代取，不下明文进 worker 内存之外的通道）。
- 未授予即不可见：worker 默认零权限，与 WASI 能力模型同构但把"能力来源"从镜像/池配置提升到**会话级显式授予**。

### D4. API 语义映射（两层平面分工）

- **平面分工**：substrate 控制面（ateapi/atecontroller/Ateom）服务 **Tier-1 会话面**（会话生命周期：resume/suspend/池绑定）；**e2b 协议面（e2bgw）+ 自研管控**服务 **Tier-2 请求面**（E2B-compatible 请求入口、请求路由、capability 表下发、审计归集）。自研管控构建在 substrate 之上，不重造会话生命周期；
- **Actor = 会话**：resume = 绑定/拉起 supervisor + 注入会话配置与 capability 表；suspend = supervisor 级 checkpoint（VM 快照 + 存储 flush）；worker 永不 suspend；
- **SandboxClass 语义重定义**：`wasm` 不再作为可独立池化的顶层 class，仅保留为 **worker 运行时标识**；新增顶层 class `supervisor`（runtime: rund/kata）作为可池化实体；枚举变更按 xuanji 约定登记 XUANJI.md；
- **ActorTemplate** 增补目标 schema（ sketch，实现另立 feature）：`supervisorClass: supervisor`、`workerRuntime: wasm`、`capabilities: [...]`、`egressPolicy: {...}`；
- **Ateom 协议演进**：`RunWorkload` 语义改为"一次请求执行"——携带 capability manifest 调用 supervisor，supervisor 内部 spawn worker、执行、销毁、返回；`CheckpointWorkload/RestoreWorkload` 作用域收敛到 supervisor（会话）；
- **配置权威在控制面**：会话配置/awareness 类声明不落 worker 镜像、不落 supervisor 镜像，resume 注入 + 请求级可刷新（类比 DE 的请求级 PATCH /role）。

### D5. 与既有单层模型的关系

- 过渡期保留现有 wasm-pool 部署作为 legacy 路径（`wasm-legacy` 标签），新会话默认走两层模型；
- 密度换隔离：单层模型单机数千 wasm 实例的密度让位于"每会话一个 kata container"的重量级边界；以温池 + 空闲 suspend 缓解成本；
- gvisor/microvm class 不受本决策影响，仍为独立顶层 class。

## Consequences

### 正向

- 会话级配置/网络/存储边界落地，与 DE 实测验证的运维模型同构（请求隔离、用过即销毁、per-session 遮蔽、出口强管）；
- worker 爆炸半径收敛到单请求；会话间不再共享进程环境与出口身份；
- capability 模型把"谁能看什么配置、能出什么网"变成可审计的显式授予（对齐 agentshell 全事件审计诉求）；
- wasm 的价值回归其最强项：请求级高密度执行隔离，而非会话承载。

### 代价与风险

- 每活跃会话一个 kata container：内存/启动成本高于池化 wasm kernel；依赖温池命中率与 suspend 策略；
- kata VM 快照体积 > wasm-only 快照：suspend/resume 延迟与存储成本上升；
- rund RuntimeClass 在目标集群的可用性需逐集群验证（ACK 已具备；自建集群需 kata/rund 部署）；
- Ateom 协议与 CRD enum 变更涉及上游文件改动（XUANJI.md 登记）与 ateom-wasmd → ateom-supervisor 的运行时重构（sandbox 仓）；
- capability 令牌的中途吊销、跨请求状态写回语义需在下个 feature 明确。

### 后续行动

1. 概念文档 [two-tier-sandbox-model.md](../concepts/two-tier-sandbox-model.md) 与示例清单 `manifests/xuanji/two-tier-example.yaml`（本 ADR 配套）；
2. sandbox 仓：ateom-wasmd 演进为 supervisor 内的 **wasm host 运行时**守护进程（ateom-supervisor：per-request worker spawn/destroy + 出口代理 + capability 校验 + 全事件审计流）；e2bgw 侧补齐请求面自研管控（capability 表下发、审计归集）；
3. substrate 仓：SandboxClass enum 增 `supervisor`、`wasm` 语义降级；ActorTemplate/WorkerPool schema 增补 capability/egress 字段；Ateom proto 演进；
4. 集群验证：cluster-msaFE8 上以 rund RuntimeClass 拉起 supervisor 池，跑会话级 e2e（resume 注入 → 请求执行 → 出口 allowlist 生效 → suspend/restore）。

## 开放问题

- supervisor 空闲 suspend 的粒度与唤醒延迟预算（会话体验 vs 成本）；
- capability 中途吊销的传播机制（proxy 侧即时生效 vs worker 生命周期内冻结）；
- 会话存储的配额与回收（DE 用 OSS/NAS 持久面；本模型默认 rustfs/OSS，配额策略待定）；
- 单 supervisor 内并发 worker 上限与燃料/内存预算的会话级配额模型。
