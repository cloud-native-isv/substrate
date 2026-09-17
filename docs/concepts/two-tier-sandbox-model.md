# 两层沙箱模型：Worker Pod 面 + Actor 工作负载面（概念文档）

> 配套决策：[ADR 0002](../decisions/0002-two-tier-supervisor-worker-sandbox.md)（Proposed，2026-09-17 术语归一）。
> 本文定义实体、生命周期、capability 模型与 API 映射的目标语义；实现 schema 以后续 feature 为准。
> 术语一律采用上游 substrate 词汇（`docs/architecture.md` @85ce8ed5）；两层（Tier-1/Tier-2）仅作平面分工描述。
> 启示来源：AgentForce 数字员工沙箱运维会话档案（2026-09-16，kata container 主体 + 请求级进程隔离）。

## 0. 两层总纲与术语归一

- **平面一（Tier-1，worker pod 面）**：**Kubernetes + Substrate + Containerd + runc/rund** 构建基于 Kubernetes 生态的 agent sandbox **worker pod**（sandbox class `wasm`；runtime runc/rund 正交可选）。特性要求：**复用 Kubernetes 生态**、**低频生命周期 + 高稳定性**、提供**休眠唤醒（Suspend/ResumeActor）**等 agent sandbox 基础特性。
- **平面二（Tier-2，actor 工作负载面）**：**e2b + 自研管控 + wasm workload**（跑在 worker pod 内部），以 WebAssembly 沙箱提供**高频创建能力、隔离机制、审计机制以及 Capability-based security 的安全模型**。

术语归一表（旧 → 新 → 上游锚点）：

| 旧术语 | 归一后 | 上游锚点 |
|---|---|---|
| supervisor sandbox | **worker pod**（sandbox class `wasm`；runtime runc/rund 可选） | Worker/WorkerPod：温沙箱 pod，host RUNNING actor |
| supervisor pool | **WorkerPool**（`sandboxClass: wasm` + `runtimeClassName: runc|rund`） | WorkerPool CRD |
| session | **actor**（E2B sandbox id = actor name） | Actor：有状态实例，拥有快照 |
| worker sandbox（wasm 实例） | **actor 执行体 = wasm sandbox**（worker pod 内嵌套隔离层）；请求级单元 = **workload 执行（context）** | sandbox = worker pod 隔离环境；RunWorkload = 一次工作负载执行 |
| ateom-supervisor | **ateom**（wasm class herder = ateom-wasmd 演进版） | ateom-\<class\> |
| supervisor（新 enum 值，早期版提议） | **废除：不新增 enum**——两层结构 = `wasm` class（runtime runc/rund 由 `WorkerPool.runtimeClassName` 正交选定） | SandboxClass enum 保持 gvisor/microvm/wasm |
| session config | **ActorTemplate** + actor 级配置注入（扩展 `sessionConfigRef`） | ActorTemplate |
| suspend = VM 快照 | **SuspendActor = actor 级 checkpoint**，worker pod 归还池 | 上游快照模型 |

## 1. 实体与职责

| 实体 | 平面 | 形态 | 生命周期 | 职责 |
|------|------|------|----------|------|
| **Worker Pod**（sandbox class `wasm`；runtime runc/rund 可选） | Tier-1 | Kubernetes Pod：Containerd + RuntimeClass（`runc` 基线默认 / `rund` kata 可选强隔离，由 `WorkerPool.runtimeClassName` 选定），由 WorkerPool 池化；**固定双容器**（控制面 + 业务面，见下） | **低频**：provision → host actor → actor suspend 后擦除归还（每租户）池 | 复用 K8s 生态（CNI/CSI/RuntimeClass/镜像）；**租户级业务体**（持租户身份/配置/持久卷，DE「员工」泛化为「租户」）；actor 工作区卷、网络出口预配置；为 Tier-2 提供宿主 |
| ├ **控制面容器** | Tier-1/Tier-2 | 固定容器（不随 actor 变动） | 与 worker pod 同生命周期 | 保留 substrate 原生 worker 组件/逻辑（ateom herder / wasm host / 出口代理 / capability broker）；**upstream-faithful**，actor 隔离以 wasm 承载（不以容器隔离 actor） |
| └ **业务面容器** | Tier-1 | 固定容器（xuanji sidecar 扩展） | 与 worker pod 同生命周期 | 租户/员工逻辑（身份代理、配置投射、持久卷管理、计费钩子等；职责边界见 ADR 开放问题）；与控制面容器经共享卷/localhost IPC 协作 |
| **WorkerPool** | Tier-1 | `WorkerPool(sandboxClass: wasm, runtimeClassName: runc|rund)` CRD | 常驻 | 池化温 worker pod（无 actor 绑定）；**每租户（或每租户模板）一池** |
| **Actor** | Tier-2（状态单元） | 控制面记录 + worker pod 内的 wasm 沙箱执行体 | **会话级**：CreateActor(SUSPENDED) → ResumeActor(RUNNING) → SuspendActor(checkpoint+归还 worker) → Delete | 有状态会话单元；拥有快照（wasm 状态 + workspace）；E2B sandbox 的对应物 |
| **ateom**（wasm class herder = ateom-wasmd 演进版） | Tier-2（worker pod 内） | worker pod 内守护进程（wasmtime host） | 与 worker pod 同生命周期 | 四项机制承载体：**高频创建**（context 亚毫秒 spawn）、**隔离**（wasm 线性内存 + 用过即销毁）、**审计**（全事件流）、**Capability-based security**（host function 唯一执行点，含出口代理与 capability 校验）；并实现 Run/Checkpoint/RestoreWorkload 与 `SetWorkerCapacity` 容量申报 |
| **Workload 执行（context）** | Tier-2 | actor wasm 沙箱内的请求级执行单元 | **请求级高频**：spawn → execute → **destroy**（无快照） | 执行单次请求；仅持授予的 capability 子集（DE 每请求进程的对应物） |
| **e2b 协议面 + 自研管控** | Tier-2（请求面控制） | e2bgw（E2B-compatible REST）+ 构建于 substrate 之上的请求面管控 | 常驻 | 请求入口与路由（E2B sandbox ↔ actor）、capability 表下发、审计归集 |
| **ActorTemplate**（+ 扩展字段） | 控制面态 | 不可变版本定义 + `sessionConfigRef`/`capabilities`/`egressPolicy` | 版本级 | actor 配置权威（平台侧），resume 注入、请求级可刷新 |

隔离分层：**主要边界 = wasm 线性内存**（context ↔ 同 actor 其他 context，并隔离租户代码——真正承载租户隔离的层）；**次要边界 = pod runtime**（按旋钮：runc 容器级 / rund kata VM 内核级，隔离 worker pod ↔ 宿主/邻 pod，rund 用于跨租户共享节点）。**actor 不以容器隔离**——容器固定（控制面/业务面双容器，同属平台信任域），actor 间隔离在控制面容器内以 wasm 承载。
密度换隔离：单层 wasm 池的"单机数千实例"让位于"每 RUNNING actor 一个 worker pod"，以温池 + actor suspend 归还 worker 缓解；worker pod 含租户业务语义后，池化为**每租户一池**（集群级→租户级）。

## 2. 生命周期序列

```
Tier-1 控制面(ateapi/atecontroller)   Tier-2 请求面(e2bgw+自研管控)   Worker Pod(wasm class, runc/rund)   Context(wasm 执行)
        |                                        |                          |                          |
 CreateActor(session)                            |                          |                          |
        |--- ResumeActor(actorConfig, caps) ------------------------------>|                          |
        |                                        |        取温 worker pod    |                          |
        |                                        |        RestoreWorkload   |                          |
        |                                        |        注入 actor 配置(ro)+ workspace(rw subdir)    |
        |                                        |        ateom 就绪(出口代理+capability broker+审计)   |
        |<------------ READY(actor, workerIP) ----------------------------|                          |
 e2b request                                     |                          |                          |
        |                                        |-- RunWorkload(caps, req)->|                          |
        |                                        |                          |-- spawn(caps 子集) ----->|
        |                                        |                          |                          | execute
        |                                        |                          |  egress via ateom 出口代理(token) --> 外部(allowlist 内)
        |                                        |                          |  审计流记录全事件          |
        |                                        |                          |<-- result / 写回 workspace|
        |                                        |                          |-- destroy context ------| (用过即销毁)
        |                                        |<----- response + 审计归集-|                          |
 idle -> SuspendActor(休眠)                      |                          |                          |
        |--- SuspendActor ------------------------------------------------->| CheckpointWorkload:      |
        |                                        |                          |  wasm 状态 + workspace flush
        |                                        |                          | worker pod 擦除归还池      |
        |--- ResumeActor(唤醒) -------------------------------------------->| 取温 pod + RestoreWorkload |
 session end -> DeleteActor                      |                          | 回收 actor 存储            |
```

要点：actor 生命周期（低频）走 substrate 控制面；请求执行（高频）走 e2b+自研管控请求面。context **永不 suspend**；跨请求状态只能显式写回 actor workspace；请求间隔离 = 新 context + 零共享内存；worker pod 是无状态温资源（suspend 即归还）。

## 3. Capability 模型（Capability-based security）

actor capability 表由 Tier-2 自研管控在 ResumeActor 时下发、worker pod 内 **ateom** 持有；context spawn 时仅获授予子集。**未授予即不可见**（默认零权限，与 WASI 同构但来源为 actor 级显式授予）。**全部执行点收敛在 wasm host function**。

| Capability | 句柄语义 | 执行点 | 类比 DE 机制 |
|------------|----------|--------|--------------|
| `cap:fs:actor-config:ro` | actor 配置文件只读 preopen | context WASI preopen | 请求级 PATCH /role 下行配置 |
| `cap:fs:workspace:rw:<subdir>` | actor 工作区子目录读写 preopen | context WASI preopen | per-session mount namespace 遮蔽 |
| `cap:net:egress:<token>` | 出口令牌；wasm 执行无直接 socket，出网经 host function → ateom 出口代理校验 allowlist | ateom 出口代理 | 出口 MITM（升级为令牌粒度） |
| `cap:secret:<name>` | 命名凭据句柄，ateom 代取 | ateom broker | 凭据不落 NAS/仓库 |

审计机制：ateom 对 context spawn/destroy、host function 调用、出口请求、capability 使用产生**全事件审计流**，由 Tier-2 自研管控归集（类比 DE 的 agentshell 全事件审计）。

吊销与刷新：出口代理 allowlist 即时生效（网络）；fs 句柄在 context 生命周期内冻结、新 context 取新表（开放问题见 ADR）。

## 4. substrate API 映射（目标语义）

- **平面分工**：substrate 控制面（ateapi/atecontroller/Ateom）= Tier-1（actor/worker 生命周期）；e2bgw + 自研管控 = Tier-2 请求面（构建于 substrate 之上，不重造 actor 生命周期）；
- **DE → 设计实体映射**（2026-09-17 对标）：DE 员工（AI工号 + kata sandbox）= **worker pod**（租户级业务体）；DE agent CLI（claude-code）= **actor 执行体**（wasm runtime + wasm agent program，同级替换非嵌套）；DE 每请求进程 = **请求级 context**；DE 原生工具链 = **wasm 定制工具**（不支持原生工具）。worker pod 内**固定双容器**：控制面容器（substrate 原生 worker 逻辑，upstream-faithful）+ 业务面容器（租户逻辑 sidecar）；
- **Actor = 会话**（E2B sandbox id = actor name）；Resume/Suspend 作用于 actor，worker pod 随取随还（每租户池）；
- **SandboxClass**：**不新增 enum**——两层结构 = 既有 `wasm` class；**pod runtime 正交旋钮** `WorkerPool.runtimeClassName`（xuanji 扩展字段）选 `runc`（基线默认）或 `rund`（kata 可选强隔离），两层不依赖 kata；legacy 单层 wasm-pool 是旧结构（§5），与 runtime 无关；gvisor/microvm 不变；
- **WorkerPool**：`sandboxClass: wasm` + `runtimeClassName: runc|rund` + `ateomImage: <ateom-wasmd>`；池化温 worker pod（每租户一池）；
- **ActorTemplate**：上游字段 `sandboxClass: wasm`；xuanji 扩展字段 `capabilities: [...]`、`egressPolicy: {defaultDeny: true, allow: [...]}`、`sessionConfigRef`；
- **Ateom 协议**：wasm class 的 `RunWorkload` = 一次请求执行（携 capability manifest，ateom 内 spawn/destroy context）；`Checkpoint/RestoreWorkload` 作用域 = actor（wasm 状态 + workspace）；
- **SandboxConfig**：wasm class 的资产 = python-wasm（workload 用）+ ateom 运行时配置；wasm-default 过渡期保留为 legacy。

示例清单：[manifests/xuanji/two-tier-example.yaml](../../manifests/xuanji/two-tier-example.yaml)。

## 5. 与单层模型（legacy）的共存

- 现有 `wasm-pool`/`wasm-pool-xl` 标记 legacy，继续服务旧 ActorTemplate；
- 新 actor 默认两层模型；迁移完成条件 = 所有 ActorTemplate 切到 `sandboxClass: wasm` 两层结构（runtime runc/rund 按节点租户模型选）且 legacy 池缩容至 0；
- 对照证据：DE 底座为 kata+OSS/NAS（非 wasm），本模型在其上把"请求级进程"替换为"请求级 wasm context"，保留其会话边界与出口强管语义。

## 6. 图示（PlantUML 源，渲染另立）

```plantuml
@startuml two-tier-sandbox
skinparam componentStyle rectangle
package "Tier-1 控制面：Kubernetes + Substrate（actor/worker 生命周期 / 休眠唤醒 / 池化）" {
  [ateapi] ; [atecontroller] ; [ActorTemplate + SessionConfig Store]
}
package "Tier-2 请求面：e2b + 自研管控（请求路由 / caps 下发 / 审计归集）" {
  [e2bgw (E2B REST)] ; [自研管控]
}
package "Node (ACK：Containerd + RuntimeClass runc|rund)" {
  rectangle "Worker Pod (sandbox class wasm, runtime runc/rund；租户级业务体) — hosts RUNNING actor(s)" as wp {
    rectangle "控制面容器 (substrate 原生 worker 逻辑, upstream-faithful)" as cpcontainer {
      rectangle "Actor 执行体：WebAssembly 沙箱（高频创建 / 隔离 / 审计 / Capability-based security）" as wasmlayer {
        [ateom (wasm class herder / wasm host)]
        [出口代理 (per-actor allowlist)]
        [capability 校验 + 审计流]
        rectangle "Workload 执行 context (request-scoped, destroy-after-use)" as c1
        rectangle "Workload 执行 context" as c2
      }
    }
    rectangle "业务面容器 (租户逻辑 sidecar：身份代理/配置投射/持久卷/计费钩子)" as bizcontainer
  }
  database "actor workspace volume" as vol
}
[ateapi] --> wp : ResumeActor(actorConfig, caps) / SuspendActor
[atecontroller] --> wp : WorkerPool 池绑定 (每租户一池)
[e2bgw (E2B REST)] --> [自研管控] : request
[自研管控] --> [ateom (wasm class herder / wasm host)] : RunWorkload(caps, request)
[ateom (wasm class herder / wasm host)] --> c1 : spawn(caps subset) / destroy
[ateom (wasm class herder / wasm host)] --> c2 : spawn / destroy
c1 --> [出口代理 (per-actor allowlist)] : cap:net:egress token
c1 --> vol : cap:fs:workspace rw subdir
[ActorTemplate + SessionConfig Store] --> wp : inject at ResumeActor
bizcontainer ..> cpcontainer : 共享卷 / localhost IPC
[自研管控] ..> [capability 校验 + 审计流] : 审计归集
@enduml
```
