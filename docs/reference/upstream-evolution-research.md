# Upstream 演进研究（xuanji 演进支持）

> **定位**：本文以「为 xuanji 分支后续演进提供决策输入」为目的，研究 upstream Agent Substrate
> 的演进方向与节奏。基线：upstream `main` @ `aa9b7b82`（2026-08-05，xuanji fork 点）；数据源：
> `origin/main` @ `24cc538d`（2026-08-09，领先基线 38 提交，373 文件 +38495/−16884）、in-flight
> 分支、`docs/roadmap.md`。所有结论附 commit hash 或 `origin/main` 文件路径；upstream 架构本身见
> [../overview.md](../overview.md)，本文不复述。状态：研究笔记（2026-08-19）。
> **相关**：xuanji 分支演进调研 → [xuanji-evolution-survey.md](xuanji-evolution-survey.md)；
> upstream 原始架构 → [../architecture.md](../architecture.md)；竞品格局 → [../concepts/sandbox-landscape.md](../concepts/sandbox-landscape.md)。

---

## 1. 方法与数据口径

- **漂移窗口**：`aa9b7b82`（08-05）→ `24cc538d`（08-09）。38 个提交集中在约 4 天；upstream 自
  2026-05-13 开源以来共 490 提交，合 ~5–6 提交/天——这是一个**高速演进期**的项目。
- **口径声明**：`origin/main` 是本地最后一次 fetch 的状态（08-09），upstream 此后大概率仍在前进；
  本文结论以该快照为准，引用时注意时效。
- **方法**：commit 考古（主题聚类）→ 新子系统/重构代码级核实（`git show origin/main:<path>`）→
  in-flight 分支前瞻 → 与 `docs/roadmap.md` 交叉验证。
- **与 xuanji 的交面**：xuanji.md 登记的 8 个 upstream 改动文件中，4 个在漂移窗口内被 upstream 触碰
  （§7.1）——这是 rebase 的确定性冲突面。

## 2. 演进全景：六个主题

| 主题 | 代表提交 | 解读 |
|---|---|---|
| **T1 数据面扩张** | egress 网关 `dd764c1e`/`2f05a92f`；GPU 直通 `934c7418`(#502)；CSI 真卷 `231a5d20`/`8152de17`/`8f4c0016` | 从「入向唤醒路由」扩展到「出向策略 + 异构算力 + 真存储」，roadmap 优先级 5（policy）与 6（runtime modularity）的落地 |
| **T2 生产化加固** | HA `510761bf`(#582)；graceful drain `5f64ba4c`(#774)；TOCTOU 修复 `9b246176`(#829)；离线 envtest `3d331532`(#792)；imagecache 驱逐 `596cfb8a`(#735，未启用) | 可靠性/可运维性投入占比高，项目正从「能力演示」走向「可运营」 |
| **T3 快照语义精化** | PAUSED 直 suspend `3a2d0c1d`(#816)；capture scope 记录 `c4c58398`(#810)/`ca81df94`(#812)；本地 pause 修剪 `a44545ee`(#705)；稀疏拷贝 `285232a9`(#769)；snapshot name/uri 重做 `ddeee6c9`(#784) | 暂停/挂起/快照三者的语义边界在被精确化——这是 suspend/resume 核心承诺（roadmap 优先级 2）的深化 |
| **T4 可观测性** | `GetWorkloadStats` `b296f96f`(#667)+cgroup `b746fb40`(#739)；冷启动指标 `0dbe1523`(#776)；scheduler 直方图 `058b104d`(#682)；atecontroller OTLP `c155efd1`(#754) | 全组件 OTLP 化收尾（roadmap「Observability」） |
| **T5 身份重构** | atunnel 证书经 atelet 代理 `c9777b49`(#708)；`MintCertRequest` 重设计为 worker 身份 attest | 「identity 驱动 policy」路线（roadmap 优先级 4）的第一块基石，且是 egress 网关的信任底座 |
| **T6 工程重构** | workflow typed ensure-steps `07da819d`(#797)；router extproc 三包化 `c9a27e52`；parking fake clock `6c7e28a0`(#781) | 核心控制面与网络栈的**结构性重排**——对 fork 而言是落点失效风险区（§7.1） |

**总体判断**：upstream 的投入分布与 `docs/roadmap.md` 的六条优先级高度一致（性能/控制面/身份/策略/
运行时模块化），且 T2/T4 占比显示项目处于「演示能力 → 生产化」的转折期。对 xuanji 而言，这意味着
upstream 仍在快速产出**值得吸收的通用能力**（§7.3），同时其重构节奏也持续制造 rebase 成本（§7.1）——
两条曲线的交叉点就是 fork point 决策点。

## 3. 新子系统（代码级）

### 3.1 Egress 网关（【新】出向策略）

形态：`manifests/ate-install/atenet-egress.yaml` 一个 Pod 两容器——Envoy dynamic-forward-proxy +
`atenet-router --mode=egress --standalone` 的 ext_proc sidecar（localhost 互联）。链路：actor 发纯 HTTP →
gVisor 内 nftables REDIRECT 到 atunnel → atunnel 用 **actor 自己的证书**（ateapi 从 actor-identity CA 逐
actor 签发、带 ActorIdentity 扩展、purpose=atunnel）做 mTLS + 裸 CONNECT → 网关验链后隧道到目标
IP:port。身份唯一来源是 Envoy `forward_client_cert_details: SANITIZE_SET` 填充的
`x-forwarded-client-cert: chain=`（CEL 表达不了自定义 X.509 扩展，这是唯一能传整张证书的通道）；handler
二次验链 → 要求恰好一个 ActorIdentity 扩展且 purpose=atunnel → `GetActor` 核对 UID 且 RUNNING，否则 403；
无 CA 配置 fail-closed 503（`origin/main:cmd/atenet/internal/router/egress/egress.go`）。router 新增
`--mode=ingress|egress|all`（`cmd.go:44`）；egress 模式不跑 xDS、**完全不需要 Kubernetes 访问**。

**对 xuanji**：E2B sandbox 的出站审计/租户级出口策略未来可直接挂在这条链路上，而非自建代理；注意其
信任模型与 ingress 相反（身份来自已验证证书，header 皆未认证输入）——这是两者分包的原因。

### 3.2 CSI 卷集成（mock → 真卷）

基线时外部卷是 `volume.NewMockVolumePlugin()`；三步落地：vendor CSI spec（`internal/volume/csi/`）→
主干接入（`8152de17`，43 文件 +1394）→ kind hostpath/NFS driver 与 e2e（`8f4c0016`）。新增 cluster-scoped
CRD `CSIDriverConfig`（`pkg/api/v1alpha1/csidriverconfig_types.go`：driverName → controllerEndpoint +
nodeSocketOverride）；控制面 `controlapi/volumes.go` 改 `VolumePluginRegistry.GetPlugin(driverName)`，卷类型
取 StorageClass 的 `Provisioner`；节点面 `cmd/atelet/volumes.go` 惰性构造 `csi.NewCSIPlugin`，
NodeStageVolume → NodePublishVolume，`VolumeContext` 经 proto（`ExternalVolume.volume_context`）下发。
`internal/volume/mock.go` 保留供测试。

**对 xuanji**：E2B 的持久化 workspace / files 语义未来可走真卷链路；wasm 类若需持久目录同样受益。

### 3.3 GPU 直通（CDI，纯 ateom 侧）

`934c7418`(#502)：worker pod 经 device plugin 分到 GPU 后，ateom-gvisor 用宿主机 toolkit
（只读挂载 `/opt/nvidia-toolkit`）`nvidia-ctk cdi generate` 生成**按本 pod GPU 范围收敛**的 CDI spec，解析
JSON 后把 device nodes、driver 库 bind mount、env 合并进 actor OCI bundle——**不执行 CDI hooks**，SONAME
符号链直接 stage 进 rootfs，以此保住 worker 非特权姿态（`origin/main:cmd/ateom-gvisor/gpu.go`）。
`workerpool_types.go` 无新字段，只加两条 CEL：`nvidia.com/gpu` 仅 sandboxClass=gvisor；requests 有 gpu 必须
有 limits。

**对 xuanji**：GPU agent 是市场刚需且与 wasm 无耦合（CEL 明确锁 gvisor），rebase 后可直接继承；注意该
提交是 xuanji `workerpool_types.go`（wasm Enum）的确定性冲突源之一（§7.1）。

### 3.4 credentialbroker（身份重构的节点侧）

`c9777b49`(#708)：atelet 内 unix socket gRPC 服务（chmod 0600，`cmd/atelet/main.go:280-292`）。同节点
worker 里的 atunnel 经它申请短命 actor 证书：从 mTLS peer 证书提取 worker 身份（拒绝非本 node incarnation），
把 worker ns/pod/uid + `ExpectedActorUid` + CSR + purpose=ATUNNEL 转发给 ateapi 新 `MintCert`；私钥留在
atunnel，到期前续期（`origin/main:cmd/atelet/credentialbroker.go`、`internal/atunnel/credential.go`）。

**对 xuanji**：per-actor 证书体系是 upstream 的既定身份路线；E2B 方向若涉及 actor 出站/互访身份，应跟进
而非自建（与 preflight 的 e2bgw 自签 envdAccessToken 形成对照——后者是探索期权宜，见
[xuanji-evolution-survey.md §5](xuanji-evolution-survey.md)）。

### 3.5 imagecache 驱逐引擎（未启用）

`596cfb8a`(#735)：`internal/imagecache/gc.go`。层池两级 DAG（image record 按 diffID 引用 layer）；root set
从 actors 目录 bundle overlay spec 重算（「精确层集合签名」保护多架构孪生记录）；`EvictUnused` LRU 驱逐 +
三道否决（root set → min-age → 行动前磁盘复检）；两阶段删除（record Remove + 层目录 rename `.rm-*`，慢速
RemoveAll 后置）；启动 `RecoverOrphans` 清扫崩溃残留。**当前无调用方**——watermark 循环是下一个 PR。

**对 xuanji**：高密度多路复用的磁盘治理收益直接；跟随后与上游同步启用即可，无接口冲突面。

## 4. 重构与语义变化

### 4.1 workflow → typed ensure-steps（#797，−1652/+1162）

旧泛型引擎（`WorkflowStep` 接口 + `RunWorkflow` 循环 + 共享大状态）整体删除；每个 workflow 变为纯方法串
typed 幂等步骤：`ResumeActor` = `loadActorForResume` → `ensureVolumesCreated` → `ensureWorkerAssigned` →
`ensureVolumesAttached` → `ensureAteletRestored` → `finalizeRunning`（`origin/main:cmd/ateapi/internal/controlapi/workflow_resume.go`）。
每个 ensureX 从持久化状态推导进度（early-return 即幂等），乐观锁版本号逐步前穿，无共享状态。唯一行为变化：
重入 resume 的 worker 所有权校验提前到卷 attach 之前（失配 worker 不再白挂卷）。

**对 xuanji**：`controlapi/workflow*.go` 已成 upstream 高频改动区（38 提交里 #784/#813/#816 都动了它）——
xuanji 任何触碰这些文件的补丁都会付出高 rebase 代价，Stage 2 之前应尽量不碰。

### 4.2 router extproc 三包化

`c9a27e52` 把基线平铺的 router 包重排为 `extproc/`（只做多路复用：终结流、判方向、分发 Handler、记延迟；
拒绝实例未声明服务的方向）+ `ingress/`（原唤醒/停车逻辑迁入）+ `egress/`（§3.1）。分包动机是两种**相反
信任模型**。**对 xuanji**：任何基于基线 atenet router 结构的本地补丁落点全部失效。

### 4.3 multi durable-dirs（#787）——对 wasm 前提的冲击

`5429ee8d` **删除**了 `actortemplate_types.go` 里两条 durableDir≤1 的 XValidation（spec 级与每容器级），
atelet 的 gVisor annotation key 从固定 `durabledir.{type,share,source}` 改为按卷名参数化
`dev.gvisor.spec.mount.<volName>.{...}`，配套 gVisor release 20260727→20260803。已核实：origin/main 的
spec 级 XValidation 只剩 3 条（volumes-must-mount / microvm-no-ExternalVolumes / 非 microvm-no-Golden），
durableDir 数量限制完全消失；ateom.proto 侧 `repeated durable_dir_volume_mounts` + reserved 旧字段配套完成。

**对 xuanji 的直接后果**：wasm v1「durableDir≤1（同 gvisor 限制）」的前提在 upstream 层面已不存在——
xuanji 工作树里那两条规则只是旧基线残留。rebase 后若 ateom-wasmd 只实现单 durable-dir，必须重新引入
wasm 专属限制，否则 wasm 静默继承多 durable-dir 语义。详见
[xuanji-evolution-survey.md §4](xuanji-evolution-survey.md)。

### 4.4 snapshot name/uri 语义（#784）

`LocalSnapshotInfo.snapshot_prefix` → `snapshot_name`；`Actor.in_progress_snapshot` →
`in_progress_snapshot_name`，新增字段 12 `in_progress_local_snapshot_name`（durable 与 local 两条在途快照分开
跟踪）；`ActorSnapshot` 新增字段 9 `snapshot_uri`——「物理存储位置 private」的注释被删，URI 从暗变明。
**对 xuanji**：e2bgw 未用这些字段（§7.2），但 xuanji 未来自建快照生命周期工具时应沿 name/uri 新语义设计。

### 4.5 身份 proto 重设计

`MintCertRequest` 从「调用者自报 actor」改为「worker 身份 attest」：`worker_namespace/worker_pod/worker_pod_uid`
+ `expected_actor_uid`（陈旧激活防护）+ `ActorCertificatePurpose` 枚举；ateapi 从 worker assignment 推导权威
actor。这是 §3.4 的协议面，也是旧 overview 指出的「MintJWT 未交叉校验」缺口（`actoridentity.go` TODO）的
系统性修复方向。

## 5. In-flight 分支（前瞻）

| 分支 | 内容 | 信号 |
|---|---|---|
| `feat/long-running-actor-support`（5 提交） | ActorTemplate 容器级 readyz 新增 `TimeoutSeconds`（int32 非指针，CRD 默认 30，零值=未设置），经 ateletpb/ateompb 传到 `internal/readyz` 的 overall timeout | 重型运行时需要分钟级就绪等待——**upstream 正在为长运行 actor 松绑**，与原始「短脉冲多路复用」假设是方向性扩展；提交序列在精细打磨 API 形态，合入在即 |
| `feat/configurable-route-timeout`（2 提交） | Envoy workload route `Timeout` 从硬编码 10s 改可配（LLM 流式长 turn 不再被截成 504），并配等值 idle timeout | 与上一分支同主题的两半：探针等待 + 请求保持 |
| `fix/runsc-allow-connected-detection`（1 提交） | 按二进制路径探测 `runsc flags` 是否列有 `-allow-connected-on-save`（memoize；探测失败假定支持，Error 日志点名最低版本） | 对 xuanji 的临时 runsc patch 路线（README）是直接可吸收的模式 |
| `feature/nanoclaw-multiplex-demo` / `feature/openclaw-integration` | NanoClaw/OpenClaw 休眠 agent 多路复用 PoC（broker 维持外部触发连接 + 控制面 resume/suspend + 遥测 UI） | 市场向 demo，无核心代码改动；印证「平时睡在快照里、触发亚秒唤醒」是 upstream 对外叙事的核心 |

**方向性结论**：upstream 正同时向「长运行 actor」与「身份驱动的网络策略」（#708→egress）两个方向扩展其
短脉冲模型。xuanji 的 E2B 长会话场景与这两条线相交时，**应优先跟随而非自建**（如 TTL/超时语义、出站
策略），把定制预算花在 upstream 不会做的部分（E2B 协议面、wasm 沙箱类）。

## 6. Roadmap 交叉验证与缺口核验

- **roadmap 治理滞后**：38 提交中 `docs/roadmap.md` 仅被触碰一次（`24cc538d`，一处措辞修改）——egress/
  GPU/CSI/imagecache 等已落地能力**均未反映在 roadmap 里**。xuanji 引用 upstream roadmap 判断优先级时须打
  此折扣；真实优先级以提交分布为准。
- **快照 GC 仍未实现**：全量检索 origin/main，durable snapshot 无 GC/驱逐实现；proto 只有 tag 保留钉的
  「可 GC」钩子语义。易误判项：#705 是 atelet **本地** pause checkpoint 修剪，#735 是**镜像层**驱逐，都不是
  快照 GC。趋势：#784 公开 `snapshot_uri`、#810/#812 记录 capture scope，是在为未来保留策略控制器铺数据面。
- **North-star 指标**（p95 100ms 激活 / 10 亿 agent / 1000 唤醒每秒）未变；microVM benchmarking 启用
  （`4ec2bab2` #691）与 benchmarking 可插配置（`cda599fa` #780）是朝向「代表性负载」优先级的动作。

## 7. 对 xuanji 的影响评估

### 7.1 Rebase 冲突风险（与 xuanji.md 登记文件相交）

| xuanji 改动文件 | upstream 触碰提交 | 冲突性质 |
|---|---|---|
| `pkg/api/v1alpha1/actortemplate_types.go`（wasm CEL 文案 ×3） | `ddeee6c9`(#784)、`5429ee8d`(#787) | **确定性冲突**：xuanji 改 CEL message，upstream 删两条 CEL + 改文案区；in-flight readyz 还会再碰此文件 |
| `pkg/api/v1alpha1/workerpool_types.go`（wasm Enum） | `934c7418`(#502) | 相邻 hunk：Enum 行 vs GPU CEL 追加 |
| `pkg/api/v1alpha1/actortemplate_validation_test.go` | `3d331532`(#792)、`5429ee8d`(#787) | 双方都改测试（envtest 样板重构 + durable-dir 用例改回 gVisor） |
| `cmd/atecontroller/internal/controllers/workerpool_apply_test.go` | `934c7418`(#502) | GPU 测试行 vs wasm 测试行同区追加 |

结构性落点失效区（xuanji 当前无补丁，但未来 Stage 2 融合若触碰需整体重写）：`cmd/atenet/internal/router/`
三包化、`cmd/ateapi/internal/controlapi/workflow*.go` 引擎删除。**xuanji.md 登记表正是这两类风险的核对
清单**——rebase 时登记为「有意重写」的文件以我方版本为准，其余保 upstream 意图（宪法 Development
Workflow §3）。

### 7.2 proto 漂移对 e2bgw 的影响

e2bgw 只用 Control 16 RPC 中的 6 个（Create/Get/Delete/Suspend/Resume/ListActors），接触面窄且全是核心稳定
字段。漂移核对：§4.4/§4.5 的改名与重设计**均不在 e2bgw 使用面**；`SuspendActor` 的 PAUSED 语义扩展
（#816）与 e2bgw「E2B pause → SuspendActor」映射互补（pause 过的 actor 也能被网关 suspend）。最脆弱点有二：
`e2bState` 对枚举 `String()` 格式的隐式依赖（`server.go:283` 的 `TrimPrefix(..., "STATUS_")`，枚举重命名会
静默改变 E2B state 输出）；`ResumeActorRequest.boot` 语义。窄接口隔离 proto 漂移是 e2bgw 设计上最成功的
决策（详见 [xuanji-evolution-survey.md §3](xuanji-evolution-survey.md)）。

### 7.3 吸收候选清单（价值 × 成本）

| 能力 | 价值 | 吸收成本 | 建议 |
|---|---|---|---|
| GPU 直通 (#502) | 高（市场刚需） | 低（与 wasm 无耦合；仅 workerpool_types 相邻冲突） | rebase 即继承 |
| CSI 真卷 | 高（E2B workspace/files 的存储底座） | 中（新 CRD + 控制面/节点面，无 xuanji 交面） | rebase 即继承 |
| readyz/route-timeout（合入后） | 高（E2B 长会话） | 低 | 合入后随 rebase 吸收 |
| imagecache 驱逐 (#735) | 中（磁盘治理） | 低（未启用态，与上游同步启用） | 跟随 |
| runsc 探测模式 | 中（patch 路线韧性） | 低 | 吸收模式 |
| HA endpoint slice (#582) | 中（控制面可用性） | 低 | 跟随 |
| egress 网关 / actor 证书体系 | 高但**时机在 Stage 2** | 中-高（身份链整体） | 列入融合清单，不提前动 |

### 7.4 fork point 观察指标

宪法 Stage 2 退出准则「rebase 成本可见地超过收益」可用三个量化信号观察：① 确定性冲突文件数（当前 4，
趋势看 #787 类语义级冲突是否增多）；② 结构性重排频率（router/controlapi 两次大重排在 38 提交内发生）；
③ xuanji.md 登记增长速率（Stage 2 重写越多登记越快）。三者同时上升即 fork point 修正案窗口。

## 8. 演进策略建议

1. **rebase 节奏**：upstream ~6 提交/天，建议**每周一次** rebase xuanji→main；每次先按 §7.1 表处理四个
   冲突文件，其余保 upstream 意图。preflight 分支合入宜与一次 rebase 同窗口完成（其冲突面仅 xuanji.md 与
   .gitignore）。
2. **吸收顺序**：GPU/CSI/imagecache（无交面）→ readyz/route-timeout（合入后）→ egress/身份（Stage 2 融合
   清单）。
3. **不碰清单**：Stage 2 声明前不触碰 `controlapi/workflow*.go` 与 `router/` 三包——xuanji 当前在这两区零
   补丁，保持零接触是当下最便宜的 rebase 保险。
4. **语义对齐**：xuanji 未来自建快照/超时/出站工具时，沿 upstream 新语义设计（snapshot name/uri、
   TTL→Suspend 与 #816/#705 的语义对齐、egress 信任模型），避免二次返工。

## 外部参考

- [Bringing you Agent Sandbox on GKE and Agent Substrate（Google Cloud blog）](https://cloud.google.com/blog/products/containers-kubernetes/bringing-you-agent-sandbox-on-gke-and-agent-substrate)
- [How Google Agent Substrate Works: 250 Agents on 8 Pods（Solo.io）](https://www.solo.io/topics/ai-infrastructure/how-google-agent-substrate-works)
- [Google Built an Agent Runtime on Kubernetes（Pomerium）](https://www.pomerium.com/blog/google-built-an-agent-runtime-on-kubernetes-heres-how-to-build-a-cloud-agnostic-one-with-identity-included)
- [Agent Executor（google/ax）](https://github.com/google/ax)
- [OpenKruise KruiseAgents：Pod 级 pause/resume 的 K8s 原生相邻方案](https://openkruise.io/kruiseagents/user-manuals/pause-resume)
- [E2B Alternatives for AI Agent Sandboxes（2026）](https://opencomputer.dev/guides/e2b-alternatives/)
- [Daytona vs E2B in 2026（Northflank）](https://northflank.com/blog/daytona-vs-e2b-ai-code-execution-sandboxes)
