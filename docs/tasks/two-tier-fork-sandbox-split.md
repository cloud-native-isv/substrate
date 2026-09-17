# 两层架构下的 substrate fork delta 与 sandbox 仓分解

> 状态：规划报告（待用户裁决 upstream 化范围与执行顺序）
> 日期：2026-09-18
> 依据：[ADR 0002 两层沙箱模型](../decisions/0002-two-tier-supervisor-worker-sandbox.md) + [ADR 0003 Wasm Host Function 安全模型](../decisions/0003-wasm-host-function-security.md)
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
| **F8** | worker pod 构建：注入 runtimeClassName（F2）、业务面容器（F3）、rund 的 kata 设备挂载/形态（SandboxClass 已「drives worker pod shape」） | `cmd/atecontroller/.../workerpool_apply.go`（+ test） | 中 | 随 F2/F3 |

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
