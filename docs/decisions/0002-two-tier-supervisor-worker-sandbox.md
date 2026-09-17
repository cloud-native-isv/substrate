# 0002. 两层沙箱模型：Worker Pod 面（Kubernetes+Substrate+Containerd+runc/rund，wasm class；runtime runc/rund 正交可选）+ Actor 工作负载面（e2b+自研管控+WebAssembly）

- Status: Proposed
- Date: 2026-09-16
- 术语归一: 2026-09-17（概念全面对齐上游 substrate 词汇，见下节）
- 去 supervisor 语义: 2026-09-17（不新增 enum；两层形态由 wasm class + `WorkerPool.runtimeClassName` 承载；herder 即 ateom-wasmd 演进版）
- DE 场景对标修正: 2026-09-17（worker pod 业务语义泛化为**租户**→多租户能力；固定**双容器**切控制面/业务面，actor 不以容器隔离；wasm agent 用**定制工具**、不支持原生工具）
- runtime 解耦: 2026-09-17（worker pod runtime 同时支持 **runc/rund**，与两层结构**正交**；二者**各具独立价值**——runc 资源利用率高/场景广、rund 安全性高，是**每租户池的策略旋钮**，混合机群为预期形态；runc 充分性以「wasm 隔离可信」为前提，判据 = wasm 可信度 × 节点租户 × 敏感度 × 密度/成本）
- Supersedes: -
- Superseded by: -

## 术语归一（2026-09-17，对齐上游 substrate 定义）

本决策早期版本使用自造实体名（supervisor sandbox / worker sandbox / session）。经与上游
`docs/architecture.md`（@85ce8ed5）的定义核对，全部概念归一到 substrate 词汇；**两层
（Tier-1/Tier-2）只保留为平面分工的描述，不再作为实体名**：

| 旧术语（ADR 早期版） | 归一后术语 | 上游锚点 |
|---|---|---|
| Supervisor sandbox（会话级 rund/kata 容器） | **Worker Pod**（sandbox class `wasm`；runtime runc/rund 正交可选） | Worker/WorkerPod：温沙箱 Pod，host RUNNING actor |
| Supervisor pool | **WorkerPool**（`sandboxClass: wasm` + `runtimeClassName: runc|rund`） | WorkerPool CRD |
| Session（绑定到 supervisor 的会话） | **Actor**（= 会话实例；E2B sandbox id = actor name） | Actor：有状态实例，拥有快照，suspend/resume 的对象 |
| Worker sandbox（wasm 实例） | 拆为两层：**actor 的执行体 = wasm sandbox**（worker pod 内嵌套隔离层）；**请求级 workload 执行（context）**（用过即销毁） | sandbox = worker pod 的隔离环境；`RunWorkload` = 一次工作负载执行 |
| ateom-supervisor（新 herder 名，早期版提议） | **ateom（wasm class herder）= ateom-wasmd 演进版**，不新增二进制名 | ateom-\<class\>：worker pod 内的沙箱 herder |
| supervisor（新 SandboxClass enum 值，早期版提议） | **废除：不新增 enum**——两层结构 = 既有 `wasm` class（runtime runc/rund 由 `WorkerPool.runtimeClassName` 正交选定，两层性不依赖 kata） | SandboxClass enum 保持 gvisor/microvm/wasm |
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

整个 sandbox 体系分成**两个平面**，wasm class 从"单层 runc pod + 共享 kernel 多路复用"演进为"worker pod（runtime runc/rund 可选）+ per-actor wasm sandbox 嵌套隔离 + 请求级 workload 执行"（**不新增 enum**；**两层结构与 pod runtime 正交**——两层性来自 wasm 执行体 + 双容器 + 每租户池，不来自 kata）：

> **平面一（Tier-1，worker pod 面）**：Kubernetes + Substrate + Containerd + runc/rund 构建**基于 Kubernetes 生态的 agent sandbox worker pod**（sandbox class `wasm`；runtime runc/rund 正交可选）。特性要求：**复用 Kubernetes 生态**、**低频生命周期 + 高稳定性**、提供**休眠唤醒（Suspend/ResumeActor）**等 agent sandbox 基础特性。
>
> **平面二（Tier-2，actor 工作负载面）**：**e2b + 自研管控 + wasm workload**（跑在 worker pod 内部），以 WebAssembly 沙箱提供**高频创建能力、隔离机制、审计机制以及 Capability-based security 的安全模型**。

### D1. Tier-1 Worker Pod（sandbox class `wasm`；runtime runc/rund 可选）

- 技术栈：**Kubernetes（编排/调度/网络/存储/RuntimeClass）+ Substrate（Actor/WorkerPool/Ateom/suspend-resume）+ Containerd + runc/rund**；worker pod 的 **runtime 由 `WorkerPool.runtimeClassName` 选定，同时支持 `runc` 与 `rund`（kata）**——二者**各具独立价值、非「默认 vs 兜底」的不对称关系**：`runc` 资源利用率高、使用场景更广（无 VM 开销、密度高、启动快、内存/CPU 更省、任何 OCI 集群可用、无 kata RuntimeClass 部署依赖）；`rund` 安全性更高（独立 guest kernel、硬件虚拟化边界、逃逸难度数量级更高）。runtime 与两层结构**正交**：runc worker pod 同样是两层（两层性来自 wasm 执行体 + 双容器 + 每租户池，不来自 kata）；
- **runtime = 每租户池的策略旋钮；混合机群（runc 池 + rund 池并存）是预期形态**（2026-09-17 补充）：与「每租户一池」天然契合——平台按各租户威胁模型逐池选 runc/rund，在不需要 VM 隔离处不付 kata 开销、在需要处拿到强隔离，整体价值最大化；
- **选型判据 = wasm 隔离可信度 × 节点租户模型 × 租户敏感度 × 密度/成本诉求**——
  - **倾向 runc**：wasm 隔离可信 + 低/中敏感租户 + 专属节点 + 高密度/低成本诉求（拿 runc 的密度与兼容红利）；
  - **倾向 rund**：wasm 隔离未经充分验证（host function 未 fuzzing/审计、租户模块未 vetted）或 高敏感租户 或 跨租户共享节点——kata 独立 guest kernel 提供**不依赖 wasm 正确性**的兜底强隔离；
  - **runc 的充分性前提**（界定「倾向 runc」的边界，非否定 runc 价值）：runc 安全性条件依赖「wasm 隔离可信」；失效级联（缺陷 host function / wasmtime 漏洞 → 租户代码逃逸进控制面容器原生进程 → runc 共享宿主内核不足兜底 → 危及宿主与同节点邻 pod）见 D2/D3；故 wasm 未验证前，高敏感/共享节点场景**必须** rund；
  - **成熟度趋势**：早期 host function 未硬化时 rund 覆盖面更大；随 wasm 隔离验证成熟，runc 适用面扩大（更多租户/场景下放 runc 拿密度红利）。中间档可选 runsc/gVisor（强于 runc、轻于 rund，需验证集群 RuntimeClass 支持）；
- **业务语义 = 租户级业务体**（2026-09-17 DE 对标修正）：worker pod 不再是纯控制面温资源，而承载**租户**（DE「员工」的泛化，工牌身份是租户身份的特例）的身份/配置/持久卷。与 substrate 原生语义的差别即在此——worker pod 含业务语义。为不丢池化优势，租户身份**权威仍在控制面**，worker pod 仅在**绑定期间缓存**、suspend 时擦除归还（每租户）池；
- **固定双容器布局**（2026-09-17 DE 对标修正）：容器**固定**、不随 actor 变动；**actor 不以容器隔离**（actor 隔离用 wasm，见 D2），容器只用来切分 worker pod 内的**控制面/业务面**逻辑——
  - **控制面容器**：保留 substrate 原生 worker 组件与逻辑（ateom herder / wasm host / 出口代理 / capability broker），**upstream-faithful**，利于回归开源（substrate 容器不改）；
  - **业务面容器**：引入租户/员工逻辑（xuanji 扩展，sidecar 式：租户身份代理、配置投射、持久卷管理、计费钩子等，职责边界见开放问题）；
  - **信任域前提（与 runtime 选型解耦）**：两容器**都必须是平台运营代码**——业务面是平台的租户管理逻辑，**不是租户自带的任意代码**；租户代码只跑在控制面容器内的 wasm 沙箱里。故两容器同属**平台信任域**，co-location 成立；pod 级强隔离（rund）**不用于分隔它们**（rund 只隔离 pod↔宿主/邻 pod，不隔离同 pod 内容器）——runtime 选 runc 还是 rund 由上述判据（wasm 可信度 × 节点租户 × 敏感度 × 密度/成本）定，与双容器 co-location 无关。**若业务面要跑租户不可信代码，co-location 即不合理**——那时应拆 pod 或在容器间另设隔离，而非依赖 rund；
- **共居度 = 容量申报模型**（上游语义，非 1:1 硬不变式）：worker 的容量（含最大 actor 数）由其 ateom 经 `SetWorkerCapacity` 自报（`cmd/ateapi/internal/controlapi/worker.go:149`、`cmd/ateapi/internal/scheduling/scheduling.go:155`）；worker pod 默认申报 `actors=1`（每 actor 强隔离），申报 N 即密度优先——旋钮在 herder，不在控制面；actor suspend 后 worker pod 擦除归还池；
- 特性契约：
  - **复用 Kubernetes 生态**——不另建调度/网络/存储/镜像体系，worker pod 就是一个 K8s Pod（runc/rund 隔离，按 runtime 旋钮），CNI/CSI/RuntimeClass/镜像仓库全部复用；
  - **低频生命周期 + 高稳定性**——worker pod 的 provisioning 与 actor 的 resume/suspend 编排是低频事件，追求长驻稳定而非高频周转；
  - **休眠唤醒等 agent sandbox 基础特性**——复用 substrate 现有 Suspend/ResumeActor 机制：suspend = actor 级 checkpoint（wasm 状态 + workspace flush）+ worker pod 归还池；resume = 从池取温 worker pod + `RestoreWorkload` + 注入 actor 配置与 capability 表；
- 承载：actor 工作区存储（挂载卷）、网络出口预配置（NetworkPolicy/路由到 worker pod）；pod 内控制面容器的 ateom（wasm class herder，即 ateom-wasmd 演进版）同时承载 Tier-2 的 wasm host 运行时、出口代理与 capability broker；
- 池化：`WorkerPool(sandboxClass: wasm, runtimeClassName: runc|rund)` 池化**温 worker pod**（无 actor 绑定）；**每租户（或每租户模板）一池**——池化从集群级退化为租户级（可接受，换来多租户隔离边界），温池命中率与成本按租户规模权衡。

### D2. Tier-2 Actor 工作负载面（e2b + 自研管控 + wasm workload）

- 组成：**e2b 协议面**（e2bgw，E2B-compatible REST 入口，E2B sandbox ↔ actor）+ **自研管控**（请求面控制逻辑：请求路由、capability 表下发、审计归集；构建在 substrate 控制面之上）+ **wasm workload**（actor 的执行体，跑在 worker pod 内）；
- **actor 的执行体 = wasm runtime + wasm agent program**（2026-09-17 DE 对标精确化）：与原生 agent CLI（如 DE 的 claude-code）**同级替换**——不是把原生 CLI 跑进 wasm 沙箱，而是用 wasm 程序本身充当 agent（agent 循环：LLM 调用、工具分发；先例：sandbox 仓 legacy `agent-in-wasm` 的 componentize-py CPython→wasm LLM agent / wasm-rust-agent）。ateom（wasm class herder）在 worker pod 控制面容器内以 wasmtime host 承载 actor 的 wasm 沙箱（python-wasm 或用户 agent 模块的 kernel 池）；actor RUNNING 期间执行体常驻，suspend 时随 actor 一起 checkpoint；
- **wasm 定制工具**（2026-09-17 DE 对标修正）：wasm agent 的工具一律为 **wasm 定制**（WASI host function / wasm 模块），**不支持原生工具**（git/chromium/dws/a1 等），无 native-tools sidecar；每个工具即一个 capability 授予点（host function 唯一执行点），比原生 CLI 的 ambient 工具攻击面更收敛；
- **请求级 workload 执行（context）**：每个请求在 actor 的 wasm 沙箱内 spawn 一个**用过即销毁**的执行单元（DE 每请求进程的对应物），capability 子集在 spawn 时授予；
- **WebAssembly 沙箱承担四项核心机制**（安全与执行语义收敛在 wasm 层，即 ateom 内，而非 worker pod 层独立组件）：
  1. **高频创建能力**——请求级 context 亚毫秒 spawn/destroy，支撑请求级高频周转（与 Tier-1 低频分工）；
  2. **隔离机制**——**主要边界 = wasm**：wasm 线性内存边界隔离 context ↔ 同 actor 其他 context、并隔离租户代码（真正承载租户隔离的层）；**次要边界 = pod runtime**（按旋钮）：runc 容器级隔离 worker pod ↔ 宿主/邻 pod，rund 时再叠加 kata VM 内核隔离（用于跨租户共享节点）；**actor 间隔离在控制面容器内以 wasm 承载，不以容器隔离 actor**（容器固定，见 D1 双容器布局）；**两层是纵深防御栈、非冗余替代**（2026-09-17 补充前提）——外层（pod runtime）必须强到能兜住内层（wasm）失效：**失效级联** = 缺陷 host function / wasmtime 漏洞 → 租户代码逃逸出 wasm 落入控制面容器原生进程 → 若 runc（共享宿主内核）一次容器/内核逃逸即危及宿主与邻 pod（跨租户），若 rund（kata 独立 guest kernel）逃逸被关在 guest VM 内；**故 runc 的充分性条件依赖「wasm 隔离可信」**（见 D1 双因子判据与 D3 host function 面）；
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
- **host function = wasm 边界最可能失效之处**（2026-09-17 补充前提）：D3 把出口/fs/secret/capability 校验**全收敛到 host function**，这片 host function 即安全关键的原生（Rust）代码、也是租户代码逃逸 wasm 的主要途径（内存安全/逻辑/能力校验绕过缺陷）。**其正确性是「wasm 隔离可信」亦即 runc 基线成立的前提**；须最小化 host function 面 + fuzzing + 审计，并跟踪 wasmtime 自身 CVE。host function 未经验证前，按 D1 双因子判据默认 rund 兜底。

### D4. API 语义映射（两层平面分工）

- **平面分工**：substrate 控制面（ateapi/atecontroller/Ateom）服务 **Tier-1**（actor/worker 生命周期：Create/Resume/Suspend/Delete、池绑定）；**e2b 协议面（e2bgw）+ 自研管控**服务 **Tier-2 请求面**（E2B-compatible 请求入口、请求路由、capability 表下发、审计归集）。自研管控构建在 substrate 之上，不重造 actor 生命周期；
- **DE → 设计实体映射**（2026-09-17 对标修正）：DE 员工（AI工号 + 其 kata sandbox，长持久业务体）= **worker pod**（租户级业务体，持租户身份/配置/持久卷）；DE agent CLI（claude-code，每会话）= **actor 执行体**（wasm runtime + wasm agent program）；DE 每请求进程 = **请求级 context**；DE 原生工具链 = **wasm 定制工具**（非原生）。worker pod 内**固定双容器**：控制面容器（substrate 原生 worker 逻辑）+ 业务面容器（租户逻辑）；
- **Actor = 会话**（E2B sandbox id = actor name）：ResumeActor = 取温 worker pod + `RestoreWorkload` + 注入 actor 配置与 capability 表；SuspendActor = actor 级 `CheckpointWorkload`（wasm 状态 + workspace flush）+ worker pod 归还池；context 永不 suspend；
- **SandboxClass 不新增 enum**：两层结构由既有 `wasm` class 承载；**pod runtime 是正交旋钮**——`WorkerPool.runtimeClassName`（xuanji 扩展字段）选 `runc`（基线默认）或 `rund`（kata，可选强隔离），**两层结构不依赖 kata**（runc worker pod 同样两层）；legacy 单层 wasm-pool 是迁移前的旧结构（见 D5），与 runtime 旋钮无关；wasm 仍是可池化顶层 class，其隔离语义 = worker pod 内嵌套 wasm sandbox；扩展字段变更按 xuanji 约定登记 xuanji.md；
- **WorkerPool / ActorTemplate 用上游既有字段表达**：`WorkerPool.spec.sandboxClass: wasm` + `WorkerPool.spec.runtimeClassName: runc|rund`（扩展字段，runtime 旋钮）、`ActorTemplate.spec.sandboxClass: wasm`（上游字段）；xuanji 扩展字段仅三个：`capabilities: [...]`、`egressPolicy: {...}`、`sessionConfigRef`（actor 级配置注入，平台侧权威、请求级可刷新，类比 DE 的请求级 PATCH /role）；
- **Ateom 协议演进**：wasm class 的 `RunWorkload` 语义 = 一次请求执行——携带 capability manifest 调用 ateom，ateom 在 actor 的 wasm 沙箱内 spawn context、执行、销毁、返回；`CheckpointWorkload/RestoreWorkload` 作用域 = actor（wasm 状态 + workspace）。

### D5. 与既有单层模型的关系

- 过渡期保留现有 wasm-pool 部署作为 legacy 路径（`wasm-legacy` 标签），新 actor 默认走两层模型；
- 密度换隔离：单层模型单机数千 wasm 实例的密度让位于"每 RUNNING actor 一个 worker pod"的重量级边界；以温池 + actor suspend 归还 worker 缓解成本；
- gvisor/microvm class 不受本决策影响，仍为独立顶层 class。

## Consequences

### 正向

- actor 级配置/网络/存储边界落地，与 DE 实测验证的运维模型同构（请求隔离、用过即销毁、per-session 遮蔽、出口强管）；
- workload 执行爆炸半径收敛到单请求；actor 间不再共享进程环境与出口身份；
- capability 模型把"谁能看什么配置、能出什么网"变成可审计的显式授予（对齐 agentshell 全事件审计诉求）；
- **多租户能力**（2026-09-17 对标修正）：worker pod 业务语义泛化为租户后，两层模型自然长出多租户边界（每租户一池、租户级身份/配置/持久卷）；
- **runc/rund 双 runtime + 每租户池策略**（2026-09-17 补充）：混合机群按需权衡密度/成本（runc：资源利用率高、场景广）与强隔离（rund：安全性高），不为不需要的 VM 隔离付 kata 开销、也不在高敏感/共享节点处省掉强隔离；
- **控制面/业务面双容器切分**：控制面容器保持 substrate 原生 worker 逻辑（upstream-faithful，substrate 容器不改），业务面容器为 xuanji sidecar 扩展——回归开源时控制面容器 delta 归零；
- **wasm 定制工具**把 agent 工具面收敛为 capability 授予点（host function 唯一执行点），攻击面小于原生 ambient 工具；
- suspend 归一为 actor 级 checkpoint 后，worker pod 解绑即恢复为无状态温资源（绑定期缓存租户身份、suspend 擦除），池利用率对齐上游模型；
- 术语与上游一致后，回归开源（见 substrate-upstream-regression 评估）时的概念翻译成本归零。

### 代价与风险

- 每 RUNNING actor 一个 worker pod：内存/启动成本高于池化 wasm kernel；依赖温池命中率与 suspend 策略；选 rund 时再叠加 kata VM 的内存/启动开销（runc 基线无此开销）；
- **每租户一池**（2026-09-17 对标修正）：池化从集群级退化为租户级，小租户池利用率低，温池命中率与成本按租户规模权衡；
- **wasm 定制工具生态需自建**（无 git/chromium/dws/a1 等原生工具复用），agent 能力面受 wasm 工具成熟度约束；
- **worker pod 含业务语义后非纯无状态**：绑定期缓存租户身份，须 suspend 擦除保证归还池后无残留（擦除完整性是安全边界）；
- worker pod 内 actor checkpoint（wasm 状态 + workspace）体积 > 纯 wasm kernel 快照：suspend/resume 延迟与存储成本上升；
- **runc 基线的充分性条件依赖「wasm 隔离可信」**（2026-09-17 补充前提）：host function / wasmtime 缺陷会击穿 wasm 边界，此时 runc 共享内核不足兜底、须 rund；故 host function 最小化 + 硬化 + fuzzing/审计是 runc 默认成立的前置工程，成熟度早期默认 rund（kata 开销更高）；
- rund RuntimeClass 在目标集群的可用性需逐集群验证（ACK 已具备；自建集群需 kata/rund 部署）——**仅当选择 rund 时**；runc 基线无此依赖（在「wasm 隔离可信」前提下 runc 设为默认的原因之一）；
- Ateom 协议演进与 WorkerPool `runtimeClassName` 扩展字段涉及上游文件改动（xuanji.md 登记）与 ateom-wasmd 的语义演进（per-actor wasm sandbox + capability 强制执行 + 容量申报，sandbox 仓）；
- capability 令牌的中途吊销、跨请求状态写回语义需在下个 feature 明确。

### 后续行动

1. 概念文档 [two-tier-sandbox-model.md](../concepts/two-tier-sandbox-model.md) 与示例清单 `manifests/xuanji/two-tier-example.yaml`（本 ADR 配套，已同步归一术语 + DE 对标修正）；
2. sandbox 仓：ateom-wasmd 演进为 wasm class 的 **ateom herder**（wasm host 运行时 + per-request context spawn/destroy + 出口代理 + capability 校验 + 全事件审计流 + `SetWorkerCapacity` 容量申报）；e2bgw 侧补齐请求面自研管控（capability 表下发、审计归集、注入 `ate-target-actor` 头）；
3. substrate 仓：WorkerPool 增 `runtimeClassName` 扩展字段（**enum 不变**）；ActorTemplate schema 增补三个扩展字段；Ateom proto 演进；
4. **业务面容器**（2026-09-17 对标新增）：设计租户逻辑 sidecar（身份代理/配置投射/持久卷管理/计费钩子）及其与控制面容器的接口契约；worker pod 模板落固定双容器布局；
5. **wasm 定制工具生态**（2026-09-17 对标新增）：定义优先工具集与 host function ABI，绑定 capability 模型（不支持原生工具）；
6. 集群验证：cluster-msaFE8 上以 wasm class（`runtimeClassName: runc` 基线 + `rund` 对照）拉起 worker pool，跑 actor 级 e2e（ResumeActor 注入 → 请求执行 → 出口 allowlist 生效 → Suspend/Resume）。

## 开放问题

> 2026-09-17 DE 对标已决：张力 A（员工钉住 vs 池化）→ 混合解（控制面权威 + 绑定缓存 + suspend 擦除）+ 泛化租户、每租户一池；张力 B（多容器隔离）→ 固定双容器切控制面/业务面，actor 不以容器隔离；张力 C（工具生态）→ wasm 定制工具、不支持原生工具；张力 D（runtime 选型）→ runc/rund 与两层结构**正交**、**各具独立价值**（runc 密度/成本/兼容、rund 安全），是**每租户池策略旋钮**（混合机群为预期形态）；判据 = wasm 隔离可信度 × 节点租户模型 × 租户敏感度 × 密度/成本——runc 充分性**以「wasm 隔离可信」为前提**（host function/wasmtime 缺陷击穿 wasm 时 runc 不足、须 rund 兜底），早期倾向 rund 覆盖面更大、随 wasm 验证成熟 runc 适用面扩大。下列为衍生 open item。

- worker pod 空闲时 actor suspend 的粒度与唤醒延迟预算（actor 体验 vs 成本）；
- capability 中途吊销的传播机制（proxy 侧即时生效 vs context 生命周期内冻结）；
- actor 工作区的配额与回收（DE 用 OSS/NAS 持久面；本模型默认 rustfs/OSS，配额策略待定）；
- 单 worker pod 内并发 context 上限与燃料/内存预算的 actor 级配额模型；
- **多租户**：租户级配额/隔离边界、每租户一池的容量规划与租户间公平性、跨租户调度约束；
- **节点租户模型 → runtime 默认值**：节点按租户专属（runc 足够）vs 多租户共享（rund 提供跨租户内核隔离）；每租户池的节点亲和/污点策略；
- **wasm 隔离可信度的建立**（runc 基线的前置）：host function 面最小化 + fuzzing + 审计、wasmtime CVE 跟踪、能力模型形式化；runtime 默认值随成熟度翻转（早期 rund → host function 验证后 runc for 可信租户/专属节点）；runsc/gVisor 中间档的集群 RuntimeClass 支持验证；
- **业务面容器信任边界**：当前前提是业务面=平台运营代码（co-location 成立）；若未来业务面需跑租户不可信代码，co-location 不再成立——须拆 pod 或容器间另设隔离（与 runtime 旋钮正交）；
- **业务面容器职责边界**：租户逻辑具体承载什么（身份代理/配置投射/持久卷管理/计费钩子），与控制面容器的接口契约（共享卷 / localhost IPC / unix socket）；
- **wasm 定制工具生态**：优先工具集、host function ABI、与 capability 模型的绑定方式；
- **租户身份权威与注入**：控制面权威 + worker pod 绑定缓存 + suspend 擦除的具体机制；是否纳入 DE 式企业身份（工牌/BUC SSO/按工号服务授权）；
- **绑定擦除完整性**：worker pod 归还池前租户身份/配置/持久卷残留的擦除验证（安全边界）。
