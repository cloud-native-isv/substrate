# 两层沙箱模型：Supervisor Sandbox + Worker Sandbox（概念文档）

> 配套决策：[ADR 0002](../decisions/0002-two-tier-supervisor-worker-sandbox.md)（Proposed）。
> 本文定义实体、生命周期、capability 模型与 API 映射的目标语义；实现 schema 以后续 feature 为准。
> 启示来源：AgentForce 数字员工沙箱运维会话档案（2026-09-16，kata container 主体 + 请求级进程隔离）。

## 0. 两层总纲

- **层面一（Tier-1）**：**Kubernetes + Substrate + Containerd + runc/rund** 构建基于 Kubernetes 生态的 Agent Sandbox，作为 **supervisor sandbox**。特性要求：**复用 Kubernetes 生态**、**低频生命周期 + 高稳定性**、提供**休眠唤醒**等 agent sandbox 基础特性。
- **层面二（Tier-2）**：**e2b + 自研管控 + runc/rund container + WebAssembly 沙箱**（跑在 supervisor sandbox 内部），以 WebAssembly 沙箱提供**高频创建能力、隔离机制、审计机制以及 Capability-based security 的安全模型**。

## 1. 实体与职责

| 实体 | 层级 | 形态 | 生命周期 | 职责 |
|------|------|------|----------|------|
| **Supervisor Sandbox** | Tier-1 | Kubernetes Pod：Containerd + rund（kata）RuntimeClass，由 Substrate WorkerPool 池化 | 会话级**低频**：resume 绑定 / suspend 休眠 / resume 唤醒 / 会话结束销毁 | 复用 K8s 生态（CNI/CSI/RuntimeClass/镜像）；会话配置存储、会话工作区挂载、网络出口预配置；休眠唤醒；为 Tier-2 提供 runc/rund container 宿主 |
| **e2b 协议面 + 自研管控** | Tier-2（请求面控制） | e2bgw（E2B-compatible REST）+ 构建于 substrate 之上的请求面管控 | 常驻 | 请求入口与路由、capability 表下发、审计归集；不重造会话生命周期 |
| **wasm host 运行时** | Tier-2（supervisor 内） | ateom-supervisor 守护进程（wasmtime host），跑在 runc/rund container 内 | 与 supervisor 同生命周期 | WebAssembly 沙箱四项机制的承载体：**高频创建**（亚毫秒 spawn worker）、**隔离**（线性内存边界 + 用过即销毁）、**审计**（全事件审计流）、**Capability-based security**（host function 唯一执行点，含出口代理与 capability 校验） |
| **Worker Sandbox** | Tier-2 | wasm 实例（python-wasm 或用户模块） | 请求级**高频**：spawn → execute → **destroy**（无快照） | 执行单次请求的代码；仅持授予的 capability 子集 |
| Supervisor Pool | 池化 | `WorkerPool(sandboxClass: supervisor)` 温容器 | 常驻 | 无会话绑定的温 supervisor，resume 时绑定会话 |
| 会话配置（SessionConfig） | 控制面态 | 平台侧权威声明 | 会话级，请求级可刷新 | 类比 DE 的 awareness/harness 声明：resume 注入，worker 只读 |

隔离叠加：kata VM 边界（supervisor ↔ 宿主/邻会话）＋ wasm 线性内存边界（worker ↔ supervisor 内其他 worker）。
密度换隔离：单层 wasm 池的"单机数千实例"让位于"每会话一个 kata container"，以温池 + 空闲 suspend 缓解。

## 2. 生命周期序列

```
Tier-1 控制面(ateapi/atecontroller)   Tier-2 请求面(e2bgw+自研管控)   Supervisor(rund container)      Worker(wasm)
        |                                        |                          |                          |
 create actor(session)                           |                          |                          |
        |--- resume(sessionConfig, caps) ---------------------------------->|                          |
        |                                        |        绑定温池 supervisor|                          |
        |                                        |        注入 SessionConfig(ro)                       |
        |                                        |        挂载 workspace(subdir rw)                    |
        |                                        |        wasm host 运行时就绪(出口代理+审计)           |
        |<------------ READY(session) --------------------------------------|                          |
 e2b request                                     |                          |                          |
        |                                        |-- RunWorkload(caps, req)->|                          |
        |                                        |                          |-- spawn(caps 子集) ----->|
        |                                        |                          |                          | execute
        |                                        |                          |  egress via host 出口代理(token) --> 外部(allowlist 内)
        |                                        |                          |  审计流记录全事件          |
        |                                        |                          |<-- result / 写回 workspace|
        |                                        |                          |-- destroy worker --------| (用过即销毁)
        |                                        |<----- response + 审计归集-|                          |
 idle -> suspend(休眠)                           |                          |                          |
        |--- suspend ------------------------------------------------------>| VM 快照 + 存储 flush       |
        |--- resume(唤醒) -------------------------------------------------->| 恢复会话边界               |
 session end -> delete                           |                          | 销毁 supervisor + 回收存储 |
```

要点：会话生命周期（低频）走 Tier-1 substrate 控制面；请求执行（高频）走 Tier-2 e2b+自研管控请求面。worker **永不 suspend**；跨请求状态只能显式写回 supervisor 会话存储；请求间隔离 = 新 worker 实例 + 零共享内存。

## 3. Capability 模型（Capability-based security）

会话 capability 表由 Tier-2 自研管控在 resume 时下发、supervisor 内的 **wasm host 运行时**持有；worker 实例化时仅获授予子集。**未授予即不可见**（默认零权限，与 WASI 同构但来源为会话级显式授予）。**全部执行点收敛在 wasm host function**——这是 WebAssembly 沙箱承担的安全模型，而非 supervisor 层独立组件。

| Capability | 句柄语义 | 执行点 | 类比 DE 机制 |
|------------|----------|--------|--------------|
| `cap:fs:session-config:ro` | 会话配置文件只读 preopen | worker WASI preopen | 请求级 PATCH /role 下行配置 |
| `cap:fs:workspace:rw:<subdir>` | 会话工作区子目录读写 preopen | worker WASI preopen | per-session mount namespace 遮蔽 |
| `cap:net:egress:<token>` | 出口令牌；wasm 无直接 socket，出网经 host function → wasm host 运行时出口代理校验 allowlist | wasm host 出口代理 | 出口 MITM（升级为令牌粒度） |
| `cap:secret:<name>` | 命名凭据句柄，wasm host 运行时代取 | wasm host broker | 凭据不落 NAS/仓库 |

审计机制：wasm host 侧对 spawn/destroy、host function 调用、出口请求、capability 使用产生**全事件审计流**，由 Tier-2 自研管控归集（类比 DE 的 agentshell 全事件审计）。

吊销与刷新：出口代理 allowlist 即时生效（网络）；fs 句柄在 worker 生命周期内冻结、新 worker 取新表（开放问题见 ADR）。

## 4. substrate API 映射（目标语义）

- **平面分工**：substrate 控制面（ateapi/atecontroller/Ateom）= Tier-1 会话面；e2bgw + 自研管控 = Tier-2 请求面（构建于 substrate 之上，不重造会话生命周期）；
- **Actor = 会话**；resume/suspend（休眠唤醒）作用于 supervisor；
- **SandboxClass**：新增顶层 `supervisor`（可池化，runtime rund/kata）；`wasm` 降级为 worker 运行时标识（不再独立池化）；gvisor/microvm 不变；
- **WorkerPool**：`sandboxClass: supervisor` + `ateomImage: <ateom-supervisor>`；池化温 supervisor；
- **ActorTemplate**（sketch 字段）：`supervisorClass`、`workerRuntime: wasm`、`capabilities: [...]`、`egressPolicy: {defaultDeny: true, allow: [...]}`、`sessionConfigRef`；
- **Ateom 协议**：`RunWorkload` = 一次请求执行（携 capability manifest，supervisor 内 spawn/destroy worker）；`Checkpoint/RestoreWorkload` 作用域 = supervisor；
- **SandboxConfig**：supervisor class 的资产 = python-wasm（worker 用）+ supervisor 运行时配置；wasm-default 过渡期保留为 legacy。

示例清单：[manifests/xuanji/two-tier-example.yaml](../../manifests/xuanji/two-tier-example.yaml)。

## 5. 与单层模型（legacy）的共存

- 现有 `wasm-pool`/`wasm-pool-xl` 标记 legacy，继续服务旧 ActorTemplate；
- 新会话默认两层模型；迁移完成条件 = 所有 ActorTemplate 切到 `supervisorClass` 且 legacy 池缩容至 0；
- 对照证据：DE 底座为 kata+OSS/NAS（非 wasm），本模型在其上把"请求级进程"替换为"请求级 wasm worker"，保留其会话边界与出口强管语义。

## 6. 图示（PlantUML 源，渲染另立）

```plantuml
@startuml two-tier-sandbox
skinparam componentStyle rectangle
package "Tier-1 会话面：Kubernetes + Substrate（会话生命周期 / 休眠唤醒 / 池化）" {
  [ateapi] ; [atecontroller] ; [SessionConfig Store]
}
package "Tier-2 请求面：e2b + 自研管控（请求路由 / caps 下发 / 审计归集）" {
  [e2bgw (E2B REST)] ; [自研管控]
}
package "Node (ACK：Containerd + RuntimeClass rund)" {
  rectangle "Supervisor Sandbox (runc/rund kata container) — session-scoped, 低频+高稳定" as sup {
    rectangle "WebAssembly 沙箱层（高频创建 / 隔离 / 审计 / Capability-based security）" as wasmlayer {
      [wasm host 运行时 (ateom-supervisor)]
      [出口代理 (per-session allowlist)]
      [capability 校验 + 审计流]
      rectangle "Worker Sandbox (wasm, request-scoped, destroy-after-use)" as w1
      rectangle "Worker Sandbox (wasm)" as w2
    }
  }
  database "session workspace volume" as vol
}
[ateapi] --> sup : resume(sessionConfig, caps) / suspend（休眠唤醒）
[atecontroller] --> sup : pool binding
[e2bgw (E2B REST)] --> [自研管控] : request
[自研管控] --> [wasm host 运行时 (ateom-supervisor)] : RunWorkload(caps, request)
[wasm host 运行时 (ateom-supervisor)] --> w1 : spawn(caps subset) / destroy
[wasm host 运行时 (ateom-supervisor)] --> w2 : spawn / destroy
w1 --> [出口代理 (per-session allowlist)] : cap:net:egress token
w1 --> vol : cap:fs:workspace rw subdir
[SessionConfig Store] --> sup : inject at resume
[自研管控] ..> [capability 校验 + 审计流] : 审计归集
@enduml
```
