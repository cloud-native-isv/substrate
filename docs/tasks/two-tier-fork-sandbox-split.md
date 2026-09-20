# 两层架构下的 substrate fork delta 与 sandbox 仓分解

> 状态：规划报告（待用户裁决 upstream 化范围与执行顺序）
> 日期：2026-09-18
> 依据：[ADR 0002 两层沙箱模型](../decisions/0002-two-tier-supervisor-worker-sandbox.md) + [ADR 0003 Wasm Host Function 安全模型](../decisions/0003-wasm-host-function-security.md) + [ADR 0004 runc 硬化](../decisions/0004-runc-worker-pod-hardening.md)/[0005 Rust 执行体](../decisions/0005-rust-agent-execution-body.md)/[0006 安全光谱](../decisions/0006-security-spectrum-model.md) + [四层信任域全景图](../concepts/trust-domain-panorama.md)（2026-09-20，I/P/S/Agent + S 层 MCP 能力后端 → §8 增量分解）
> 配套：[substrate 回归开源评估](substrate-upstream-regression.md)（现状 A/B/C 分类，本文是其**前向延伸**：目标两层架构落地需要的 fork delta 与 sandbox 分解）
> 代码基线：xuanji @ 6d931cdc（schema 事实经 `pkg/api/v1alpha1/{workerpool,actortemplate}_types.go`、`internal/proto/ateompb/ateom.proto`、`cmd/atecontroller/.../workerpool_apply.go` 核实）

## 0. 结论摘要

**问题**：目标两层架构（ADR 0002 + 0003）相对开源 substrate，xuanji 分支要改什么、哪些归 sandbox 仓。

**一句话**：substrate fork 的 delta 从「当前仅 wasm enum 缝（~40 行）」**扩大**到「enum 缝 + WorkerPool 形态字段（runtimeClassName/业务面容器）+ Ateom proto 演进 + 控制面租户/capability 状态」；**绝大部分执行体逻辑（wasm host、host function 安全面、三环审计 Ring 0/1、wasm agent、定制工具、e2bgw、业务面容器镜像）归 sandbox 仓**。这与回归开源「delta→0」目标存在张力（§6）。

| 平面 | 归属 | 项 |
|---|---|---|
| Tier-1 CRD/控制面 | **substrate fork** | F1 wasm enum（已有）· F2 WorkerPool.runtimeClassName · F3 业务面容器字段 · F5 Ateom proto 演进 · F6 控制面租户/capability 状态 · F8 worker pod 构建逻辑 |
| Tier-1 调度绑定 | **upstream 既有，零 delta** | F7 每租户池绑定 = `ActorTemplate.WorkerSelector`（label）+ `WorkerPoolPodTemplate` 的 NodeSelector/Affinity/Tolerations（节点租户模型）——均上游已有 |
| Tier-2 执行体 + 安全 + 审计 | **sandbox 仓** | S1 ateom herder · S2 host function 安全面 · S3 三环审计 Ring 0/1 · S4 wasm agent program · S5 wasm 定制工具 · S6 e2bgw+自研管控 · S7 业务面容器镜像 · S8 Ring 2 Tetragon 部署 · S9 E2B e2e |
| 跨仓缝 | **fork 权威 + sandbox vendored** | Ateom proto（§4）· CRD 字段 · 镜像引用 |

**两处与代码事实冲突、需修正 ADR 0002 的设计点**（详见 §5）：
1. **ActorTemplate.Spec 不可变**（`+kubebuilder:validation:XValidation:rule="self == oldSelf"`）——ADR 0002 D4 把 `capabilities/egressPolicy/sessionConfigRef` 当作 ActorTemplate 扩展字段，但 sessionConfigRef 要求「请求级可刷新」，与不可变冲突。正解：三者是**控制面 actor 级状态 + 经 Ateom proto 在 resume/请求时注入**，ActorTemplate 只持不可变的版本级默认。
2. **`RunWorkload` 是 actor 激活级、非请求级**（proto 注释："begin running a new workload"，由 atelet 调用 boot/restore actor）——ADR 0002 D4「RunWorkload 语义 = 一次请求执行」与上游冲突。正解：substrate 的 Run/Checkpoint/RestoreWorkload 保持 **actor 级**（Tier-1，atelet 调用）；**请求级 context spawn/destroy 走独立的 Tier-2 RPC**（请求面 e2bgw/自研管控 → ateom），capability manifest 随该 RPC（及 actor 级 caps 随 RunWorkload）。

## 1. 核实的 schema 事实（决定 delta 边界）

| 事实 | 出处 | 对 delta 的影响 |
|---|---|---|
| `WorkerPoolPodTemplate` 只有 NodeSelector/Tolerations/PriorityClassName/NodeAffinity/Resources，**无 runtimeClassName、无 containers[]** | `workerpool_types.go:24-52` | kata runtime 与双容器**无法经既有 template 表达** → F2/F3 是真 fork delta |
| `WorkerPoolSpec` 有 `AteomImage`（单容器）、`SandboxClass`(enum gvisor;microvm;wasm)、`SandboxConfigName`；注释明示 SandboxClass「drives the worker pod shape (KVM/vhost device mounts and node placement)」 | `workerpool_types.go:54-89` | wasm enum 已在（F1）；rund 的 kata 设备挂载/形态走 apply 逻辑（F8） |
| `ActorTemplateSpec` **immutable**；已有 `Containers[]`(≤10)、`Volumes`(durableDir/externalVolumeTemplate)、`EnvVar.valueFrom.secretKeyRef`、`WorkerSelector`(LabelSelector) | `actortemplate_types.go:354-418,447` | 每租户池绑定（WorkerSelector）+ actor 工作区卷（Volumes）+ 凭据注入（secretKeyRef）**均上游已有，零 delta**；capabilities/egressPolicy/sessionConfigRef 因 immutable 不能放这（§5.1） |
| `RunWorkloadRequest` = actor 激活级：atespace/actor_*/runsc_path/spec(WorkloadSpec=containers)/runtime_asset_paths/**egress_gateway_address**(optional)；**无 capability manifest** | `ateom.proto:50-73` | capability 注入 + 请求级执行需 proto 演进（F5）；已有 egress_gateway_address 是 D5 出口代理的接入点 |
| Ateom proto 是 **internal** proto（`internal/proto/ateompb`） | 路径 | 演进 = fork delta；sandbox `cmd/ateom-wasmd/proto/ateom.proto` vendored 副本需同步（§4） |
| worker pod 由 atecontroller 构建 | `cmd/atecontroller/internal/controllers/workerpool_apply.go` | runtimeClassName/业务面容器/kata 设备挂载在此注入（F8） |

## 2. Part A — substrate fork（xuanji）delta

> 原则（ADR 0002 D1）：控制面容器**行为** upstream-faithful；下列 delta 尽量做成**加字段/加分支、默认不改变 upstream 行为**，以保 rebase 稳定与 upstream 化可能。每项标注 upstream 化倾向。

| # | 改动 | upstream 文件 | 体量 | upstream 化 |
|---|---|---|---|---|
| **F1** | wasm SandboxClass enum 缝（**当前已有**）：Enum+常量+CEL 文案、3 generated CRD、VAP 规则、`workerpool_apply.go` wasm securityContext(drop ALL) 分支 | `pkg/api/v1alpha1/{sandboxconfig,actortemplate,workerpool}_types.go` + CRDs + `sandboxconfig-validation.yaml` + `workerpool_apply.go` | ~40 行+生成物 | (a) 提 PR（新 runtime class + 零特权 securityContext = 安全加固）；被拒退 (b) 唯一 fork delta |
| **F2** | `WorkerPool` 增 **runtimeClassName**（runc/rund，正交旋钮）+ apply 注入 `pod.Spec.RuntimeClassName` | `workerpool_types.go`(WorkerPoolPodTemplate 或 Spec) + CRD + `workerpool_apply.go` | 小（~20 行+生成物） | **强 upstream 候选**：通用 K8s passthrough，对任何需 kata/gVisor RuntimeClass 的用户普遍有价值 |
| **F3** | `WorkerPool` 增 **业务面容器**（租户 sidecar 镜像/容器定义）+ apply 把它加进 worker pod（与 ateom 控制面容器并列，固定双容器） | `workerpool_types.go` + CRD + `workerpool_apply.go` | 中（容器定义结构 + apply 拼接 + 卷/IPC 约定） | 弱：tenant sidecar 是 xuanji 概念；可泛化为「worker pod 额外容器」提 PR，否则 fork delta |
| **F4** | `ActorTemplate` **immutable 版本级默认**（可选）：capability/egress 的版本级默认策略（非刷新部分） | `actortemplate_types.go` + CRD + CEL | 小 | 中：可作通用「actor 策略默认」提 PR；刷新态不放这（§5.1） |
| **F5** | **Ateom proto 演进**：① `RunWorkloadRequest` 加 actor 级 capability manifest；② 新增 **Tier-2 请求级执行 RPC**（如 `ExecuteWorkload`/`RunContext`，携 per-request capability 子集，spawn/destroy context）；③ actor 级 checkpoint 语义对齐（wasm 状态 + workspace） | `internal/proto/ateompb/ateom.proto`（+ 重新生成 .pb.go）+ atelet 调用侧 | 中 | 弱：capability 模型是 xuanji 安全语义，upstream 未必收；**大概率 fork delta + sandbox vendored 同步** |
| **F6** | **控制面租户/capability 状态**：ateapi 存租户身份 + actor 级 capabilities/egressPolicy/sessionConfig（Valkey），resume 注入、suspend 发擦除信号；atelet 把 capability manifest 经 proto 传 ateom | `cmd/ateapi/...`（store/controlapi）+ `cmd/atelet/...` + Valkey schema | 中大 | 弱：租户/capability 是 xuanji 业务语义；fork delta |
| **F7** | 每租户池绑定 + 节点租户模型 | **零 delta**：`ActorTemplate.WorkerSelector`(label) + `WorkerPoolPodTemplate.{NodeSelector,NodeAffinity,Tolerations}` 均上游已有 | 0 | — |
| **F8** | worker pod 构建：注入 runtimeClassName（F2）、业务面容器（F3）、rund 的 kata 设备挂载/形态（SandboxClass 已「drives worker pod shape」）；**runc 池 L2 容器/pod 安全硬化**（[ADR 0004](../decisions/0004-runc-worker-pod-hardening.md) D2：在 F1 wasm securityContext 分支上补 `allowPrivilegeEscalation:false`、显式 seccomp、`readOnlyRootFilesystem`、评估 `hostUsers:false`(userns)、hostPath 收窄）+ **L3 部署清单**（专属节点 taint/toleration、NetworkPolicy 东西向默认拒绝） | `cmd/atecontroller/.../workerpool_apply.go`（+ test）+ `manifests/xuanji/`（NetworkPolicy/节点约束） | 中（硬化本身面小，userns×hostPath 联动需验证） | 随 F2/F3；**L2 硬化（allowPrivilegeEscalation:false / seccomp / readOnlyRootFilesystem）是通用安全加固，强 upstream 候选**；userns/hostPath 收窄偏 xuanji 节点模型 |

**fork delta 净增量（相对当前）**：F2+F3+F5+F6+F8（F1 已有、F4 可选、F7 零）。其中 **F2 最该 upstream 化**（通用、面小、价值普适）；F5/F6 最可能长期留 fork（xuanji 安全/租户语义）。

## 3. Part B — sandbox 仓 delta

> 全部在 sandbox 仓（cloud-native-webassembly/sandbox）；多数是 ateom-wasmd 演进 + 新增组件。对应 ADR 0002 后续行动 2/4/5 与 ADR 0003 后续行动。

| # | 组件 | 内容 | 来源 ADR |
|---|---|---|---|
| **S1** | ateom herder（ateom-wasmd 演进） | wasmtime host 运行时 + per-actor wasm 沙箱（执行体常驻）+ per-request context spawn/destroy + `SetWorkerCapacity`(actors=1 默认) + Run/Checkpoint/RestoreWorkload 实现 | 0002 D1/D2 |
| **S2** | **host function 安全面** | 最小 host function 面（allowlist 默认拒绝）+ ABI 边界校验 + capability 唯一执行点（不可伪造句柄、fail-closed）+ 内存安全纪律（safe Rust）+ 出口代理隔离 + 凭据不落 wasm + fail-closed 降级 + 硬化门禁（fuzzing/property/红队/wasmtime CVE/供应链） | **0003 D1-D6,D8,D9** |
| **S3** | **三环审计 Ring 0/1**（ateom 内） | Ring 0 档位A（wasmtime-wasi `#[instrument]` span）+ 档位B（p2 Host trait 结构化拦截，演进）；Ring 1（执行前代码存档+静态审计、出口代理审计、fuel 计量、I/O 存档）；统一 `target="audit"` 事件（actor/context id+ring+event） | **0003 D7** |
| **S4** | wasm agent program | agent 循环（LLM 调用、工具分发）的 wasm 模块（python-wasm / rust wasm），**同级替换原生 agent CLI** | 0002 D2 |
| **S5** | wasm 定制工具 | 工具一律 WASI host function（无原生工具）；按 S2 登记制 + 过门禁 | 0002 D2 / 0003 D1 |
| **S6** | e2bgw + Tier-2 自研管控 | E2B REST 网关（回归评估 A1 迁入）+ 请求路由（注入 `ate-target-actor` 头）+ capability 表下发 + 审计归集 + **请求级执行通道**（→ ateom 的 F5 新 RPC） | 0002 D2/D4 |
| **S7** | 业务面容器镜像 | 租户逻辑 sidecar（身份代理/配置投射/持久卷管理/计费钩子），供 F3 引用；与控制面容器经共享卷/localhost IPC 协作 | 0002 D1 |
| **S8** | Ring 2 Tetragon 部署 | TracingPolicy YAML（syscall + uprobe 两套）+ 部署文档；**按 runc/rund 分置**（runc 池宿主侧直采、rund 池 VM 边界/in-guest，0003 D7.1）；单节点验证 AliSecGuard 兼容性 | 0003 D7/D7.1 |
| **S9** | E2B e2e 资产 | 回归评估 A2 回迁；指向本仓 e2bgw | 0002 |

## 4. 跨仓缝（契约）

| 缝 | fork 侧 | sandbox 侧 | 同步约定 |
|---|---|---|---|
| **Ateom proto** | `internal/proto/ateompb/ateom.proto`（权威） | `cmd/ateom-wasmd/proto/ateom.proto`（vendored） | F5 演进后**重拷** vendored 副本；长期改为依赖 upstream 发布版 proto（回归评估 C 类） |
| **CRD 字段** | F2 runtimeClassName / F3 业务面容器 / F4 默认策略（fork 定义 + 生成 CRD） | S7 镜像被 F3 引用；deploy 清单消费这些字段 | 字段稳定后 sandbox deploy/ 清单跟进 |
| **镜像** | WorkerPool.AteomImage（控制面）+ F3 业务面镜像字段 | S1 ateom 镜像 + S7 业务面镜像（sandbox 构建发布） | 镜像 pinned（CRD 强制 `@sha256`） |
| **请求级通道** | F5 新 RPC（proto）+ atelet/请求面调用 | S6 e2bgw 发起 + S1 ateom 实现 | RPC 契约随 proto 同步 |

## 5. 需修正 ADR 0002 的两处设计点（代码事实校正）

### 5.1 ActorTemplate 不可变 vs 可刷新配置

- **事实**：`ActorTemplateSpec` 标 `self == oldSelf`（immutable）。
- **冲突**：ADR 0002 D4/归一表把 `capabilities/egressPolicy/sessionConfigRef` 列为 ActorTemplate 扩展字段，但 sessionConfigRef「请求级可刷新」与不可变矛盾。
- **正解**：三者是**控制面 actor 级状态**（F6，存 Valkey），经 **Ateom proto 在 ResumeActor / 每请求注入**（F5）；ActorTemplate 至多持**不可变版本级默认**（F4，可选）。capability 中途吊销/刷新走控制面状态 + 新 context 取新表（ADR 0002 开放问题「capability 中途吊销」），不动 ActorTemplate。

### 5.2 RunWorkload 是 actor 级、非请求级

- **事实**：`RunWorkloadRequest`（proto:50）= 「begin running a new workload」，由 atelet 在 actor 激活时调用（boot/restore actor 的 containers）；`Checkpoint/RestoreWorkload` 同为 actor 级。
- **冲突**：ADR 0002 D4「wasm class 的 RunWorkload 语义 = 一次请求执行」。
- **正解**：保持 substrate Run/Checkpoint/RestoreWorkload = **actor 级**（Tier-1，atelet 调用，激活/挂起 actor 的 wasm 执行体）；**请求级 context spawn/destroy 走独立 Tier-2 RPC**（F5 新增，请求面 e2bgw/自研管控 → ateom），与 ADR 0002「Tier-2 请求面构建于 substrate 之上、不重造 actor 生命周期」一致。actor 级 capability 随 RunWorkload，请求级 capability 子集随 Tier-2 RPC。

> 建议：以本文 §5 为准修订 ADR 0002 D4 与归一表对应行（sessionConfigRef/capabilities/egressPolicy 的承载位置、RunWorkload 语义）。

## 6. 与回归开源目标的关系（张力 + 建议）

- **张力**：回归评估的目标是 fork「delta→0 或仅 2 块小 delta（B1 enum + B2 PKI）」；两层架构新增 F2/F3/F5/F6/F8，**扩大** fork delta。
- **建议分层处置**：
  - **优先 upstream 化**：F2（runtimeClassName，通用 K8s passthrough）+ F1（wasm enum + 零特权 securityContext，安全加固）——面小、价值普适、rebase 收益高；
  - **可泛化 upstream**：F3（泛化为「worker pod 额外容器」）、F8 的 kata 形态（SandboxClass 已 drives worker pod shape，rund 形态可作 microvm 类的自然延伸）；
  - **长期留 fork**：F5 capability manifest + F6 租户/capability 控制面状态（xuanji 安全/多租户语义，upstream 未必收）——做成默认 OFF 的加字段/加分支，保 upstream 行为不变；
  - **澄清「upstream-faithful」边界**：ADR 0002 D1 的「控制面容器 upstream-faithful」指 **ateom herder 运行时行为**不改 substrate 语义；但 F2/F5/F6 的 CRD/proto/控制面状态仍是 fork delta——两者不矛盾（容器行为忠实 ≠ 零 fork delta），文档需点明。
- **依赖方向不变**：sandbox → upstream substrate（公开 module + proto）；fork delta 收敛到「enum 缝 + 形态字段 + capability/租户控制面」，其余执行体全在 sandbox。

## 7. 建议执行顺序

1. **修订 ADR 0002 D4 + 归一表**（§5 两处校正）——先对齐设计，再动代码；
2. **F2 runtimeClassName**（最该 upstream、面小）：fork 加字段 + apply 注入 + CRD 生成；同步起 upstream PR；
3. **F5 Ateom proto 演进**（capability manifest + Tier-2 请求级 RPC）：fork 改 proto + 重生成；sandbox 重拷 vendored 副本；
4. **S1/S2/S3 sandbox 主体**：ateom herder + host function 安全面 + 三环审计 Ring 0/1（ADR 0003 D8 门禁同步建 CI）；
5. **F3 + S7 业务面容器**：fork 加字段 + apply 拼接；sandbox 出 sidecar 镜像；
6. **F6 控制面租户/capability 状态** + **S6 e2bgw 请求面**（含 ate-target-actor 注入 + 请求级通道）；
7. **F1/F4 enum 缝收口 + upstream PR**；**S8 Ring 2 Tetragon**（按 runc/rund 分置，单节点验证）；
8. **集群 e2e**（cluster-msaFE8）：runc 基线 + rund 对照，跑 actor 级全链路（resume 注入 → 请求执行 → 出口 allowlist → 三环审计事件 → suspend/resume）。

## 8. 全景图增量分解（四层信任域 / 多 active actor / MCP 能力后端）

> 依据：[四层信任域全景图](../concepts/trust-domain-panorama.md)（2026-09-20，I/P/S/Agent + S 层 MCP 能力后端）。本节把全景图相对 §2/§3 既有 **F1–F8 / S1–S9** 的**新增项**按分工分解；既有项不重复。
> 信任域定位：**F9–F12** 落 P 层（平台）+ S 层 worker pod 的**集群侧**；**S10–S13** 落 S 层 worker pod **内（ateom）+ S↔Agent / S↔MCP 边界**（详见 sandbox 仓 `two-tier-sandbox-layer-design.md` §6）。

### 8.1 新增 substrate fork delta（F9–F12）

| # | 改动 | 信任域 | upstream 文件 | upstream 化 |
|---|---|---|---|---|
| **F9** | **多 active actor / worker pod（空间复用调度）**：打破「1 worker pod = 1 active actor」，ateapi/scheduler 允许 N 个 actor 并发绑定同一 worker（actor→worker **N:1**），消费 ateom `SetWorkerCapacity(actors=N)`（S12 申报）；WorkerPool 容量模型 1→N + per-actor 资源核算。与既有 suspend/resume **时序复用并存**（N:1 = 空间复用） | S（worker pod）+ P（调度） | `cmd/ateapi/...`（scheduler/store 容量绑定）+ 可能 `workerpool_types.go` 容量字段 | 中：多路复用是 substrate「many actors→fewer workers」意图的延伸，但 1:1→N:1 容量语义偏 xuanji；可泛化提 PR |
| **F10** | **MCP 共享池编排/路由层（P 层新组件，自洽条件 C2）**：cross-S 共享 MCP 池的编排/路由 = P 层**可信、硬化、自身被审计**组件——调度 per-tenant 执行单元、把 ateom 的 MCP 调用路由到对应执行单元、持池侧 allowlist + 审计。形态可复用 WorkerPool/actor 机制（一个 microvm-class WorkerPool，其 actor = MCP server 实例）或新控制器/服务 | P | 新 `cmd/` 组件或 `cmd/atecontroller` 扩展 + CRD | 弱：MCP 能力后端是 xuanji 概念 |
| **F11** | **per-tenant MCP 执行单元（microvm）+ MCP server 运行时（自洽条件 C1）**：cross-S 执行单元 = 强隔离 **microvm**（复用既有 microvm sandbox class kata+cloud-hypervisor），内跑 MCP server（原生工具执行器：编译/git/browser/任意原生），一次性用完即毁；within-S 形态 = worker pod 本地 VM/sidecar（OS 账号/容器隔离即可）。**OS 多用户仅限 within-S；cross-S 必须 microvm/gvisor**（OS 账号隔离不足以做跨租户边界）。后端出口须白名单（I-4，R5） | S（执行单元） | 复用 microvm class（`workerpool_apply.go` microvm 形态已有）+ 新 MCP server 镜像/组件 | 弱-中：microvm 复用既有；MCP server 是 xuanji |
| **F12** | **跨域审计 + 白名单统一（P↔S、S↔S、I↔P）**：把 ADR 0003 D1 白名单 + D7 审计推广到全信任域跨越——P↔S（控制面 API + actorlog）、S↔S（**默认拒绝** + MCP 池侧 per-tenant allowlist + 审计）、I↔P（k8s RBAC/NetworkPolicy）；统一审计事件扩展 `src_domain`/`dst_domain`，归集 Tier-2 自研管控。**零信任身份**：复用 podcertcontroller(SPIFFE) 给 MCP 池执行单元 + ateom↔池 mTLS（多为既有基础设施 wiring，并入 F10/F11） | I/P/S | `internal/actorlog`（已有）+ 池侧审计 + `manifests/xuanji/` NetworkPolicy | 中：actorlog/NetworkPolicy 通用；跨域审计 schema 偏 xuanji |

### 8.2 新增 sandbox 仓项（S10–S13，归属 sandbox，详见其 §6）

| # | 组件 | 内容 | 信任域 | 来源 |
|---|---|---|---|---|
| **S10** | ateom MCP 客户端 / 可信中介 | ateom-wasmd 增 MCP 传输客户端：翻译 host function 调用为 MCP 协议、连后端（within-S 本地 / cross-S 池）、mTLS、按工具 (a)/(b) 标注路由、(b) 类 per-call 注入短时效 scoped token（D6 MCP 版，用后即弃不落后端）；**强制点在 ateom（D1 白名单/D2 入参校验/D3 capability，fail-closed）** | S（ateom） | 全景图 §3.4/§3.5 |
| **S11** | host function `mcp_call` + agent MCP 语义客户端 + 输出净化 | host_functions.rs 增 `mcp_call(tool,args)`（Agent→S 穿越点，过 D2/D3）；S4 Rust agent 作 MCP 语义客户端经此调工具（无裸 socket）；ateom 把后端输出当**敌意输入（I-2）**净化/标注后返回 | S↔Agent | 全景图 §3.4 |
| **S12** | 多 active actor 并发（ateom-wasmd） | 从 1 活跃 actor 改为承载 N 并发 actor 沙箱（每 actor 内核池 + capability 表 + MCP 账号映射）；`SetWorkerCapacity(actors=N)` 申报（F9 消费）；intra-pod 隔离 ride on wasm per-context fuel/mem 限额（D2） | S | 全景图 §2.1 |
| **S13** | MCP 后端审计（Ring 0/1 扩展）+ MCP 工具契约 | 每次 MCP 调用一条审计事件（tool/args 截断/target/allowed/status/字节/耗时 + src/dst domain）；policy.blocked WARN；**MCP 工具契约/schema**（agent 可见工具面 = S5 注册表扩展到 MCP-backed 工具，是 F11 MCP server 的实现契约） | S↔MCP | 全景图 §3.4/§5 |

### 8.3 新增跨仓缝（契约）

| 缝 | fork 侧 | sandbox 侧 | 同步约定 |
|---|---|---|---|
| **MCP 工具契约/schema** | F11 MCP server 实现工具 | S13/S5 定义 agent 可见工具面 | 工具名/参数 schema/capability 绑定/(a)/(b) 标注 = 单一事实源，类比 Ateom proto seam |
| **MCP 协议/传输 + 执行单元镜像** | F10 池路由 + F11 microvm 执行单元镜像 | S10 ateom MCP 客户端 | MCP 传输契约 + mTLS 身份（SPIFFE）两侧对齐 |
| **多 active actor 容量** | F9 scheduler 消费容量 | S12 `SetWorkerCapacity(actors=N)` 申报 | 容量申报/消费契约 |

### 8.4 自洽条件 C1–C3 与残留张力 R1–R5 的归属（全景图 §6）

| 条件/张力 | 内容 | 归属 |
|---|---|---|
| **C1** | cross-S 共享基础设施+编排，**执行单元强隔离 microvm/gvisor**；OS 多用户仅 within-S | **F11**（microvm class） |
| **C2** | 共享池编排/路由层 = P 可信 + 硬化 + 自身被审计 | **F10** |
| **C3** | 零信任不留秘密 + 跨域审计白名单 + 输出敌意 = cross-S 共享补偿控制 | F10/F12（身份+池审计）+ **S10**（token 注入）/**S11**（输出净化）/**S13**（审计） |
| R1 | OS 多用户非跨租户边界 | → C1 / **F11** |
| R2 | 编排层跨租户 blast radius | → C2 / **F10** |
| R3 | 跨租户 confused-deputy | **S10** per-tenant capability scoping + **F12** 池侧 allowlist |
| R4 | 多 actor 争用 pod CPU/mem | **S12** fuel/mem 限额 + **F9** pod sizing |
| R5 | 后端出口绕 wasm 白名单（I-4） | **F11** 后端出口白名单 |

> **执行顺序补充**（接 §7）：F9/S12（多 active actor）是 MCP「公用」语义的前提，宜先于 F10/F11/S10；F10/F11（共享池+执行单元）与 S10/S11/S13（ateom 中介链）成对推进；F12 跨域审计随各层落地增量补齐。MCP 工具契约（8.3）须先于 F11 server 与 S13 审计定稿。
