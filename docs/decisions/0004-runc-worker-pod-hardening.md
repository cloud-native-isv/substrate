# 0004. runc Worker Pod 安全硬化（纵深防御外层 + runc 准入判据）

- Status: Proposed
- Date: 2026-09-18
- 关联: [ADR 0002](0002-two-tier-supervisor-worker-sandbox.md) D1（runtime 选型判据 / runc 充分性前提）、D2（隔离机制 = 纵深防御栈、失效级联）；[ADR 0003](0003-wasm-host-function-security.md)（内层 = wasm host function 预防 + 三环检测，本 ADR 是其**外层补全**）
- Supersedes: -
- Superseded by: -

## Context

ADR 0002 把 worker pod runtime 定为**每租户池的正交旋钮**，并明确 `runc` 与 `rund` **各具独立价值**：runc 资源利用率高、使用场景广（无 VM 开销、密度高、启动快、任何 OCI 集群可用、无 kata RuntimeClass 部署依赖），rund 安全性高（独立 guest kernel、硬件虚拟化边界）。**runc 的实用价值很大**——它是拿密度/成本/兼容红利的形态，因此平台有强动机把尽可能多的租户池下放 runc。

但 runc worker pod **共享宿主内核**：ADR 0002 D2 的失效级联指出，缺陷 host function / wasmtime 0day 会让租户代码逃逸出 wasm、落入**控制面容器原生进程**；此时 runc 不提供 guest kernel 兜底，一次容器/内核逃逸即危及宿主与同节点邻 pod（跨租户）。故 ADR 0002 D1 钉死：**runc 充分性以「wasm 隔离可信」为前提**。

ADR 0003 已硬化**内层**（wasm host function 面：最小面 + ABI 校验 + capability 唯一执行点 + 内存安全 + 出口代理隔离 + 凭据不落 wasm + 三环审计 + D8 准入门禁 + fail-closed），把「wasm 隔离可信」从主观假设变为可量化的门禁状态。**但外层（runc 容器/节点）的遏制设计尚缺**：ADR 0002 D2 只声明「外层须强到能兜住内层失效」这一原则，未设计 runc worker pod 本身如何硬化、爆炸半径如何限制、逃逸如何响应。

**本 ADR 补全外层**，并把内外层统一为**可核查的 runc 准入判据**。核心命题：runc 下控制面容器（ateom 进程，uid 0）就是 wasm 逃逸的落点；外层硬化的目标是——**即使 wasm 被击穿，逃逸者也落在一个权能最小、宿主面最窄、邻租户不可达、且被内核 eBPF 全程监视的容器里**，把「runc 共享内核」的风险压到可接受，从而安全地兑现 runc 的实用价值。

> 实现载体分两仓：**L1 预防 / L4 检测**在 sandbox 仓（ateom-wasmd，ADR 0003）；**L2 容器·pod 遏制 / L3 节点·集群爆炸半径**在 substrate fork（`cmd/atecontroller/.../workerpool_apply.go` + CRD/字段 + 部署清单）。本 ADR 是跨两仓的 runc 安全设计权威。

### 威胁模型（runc 特定）

- **攻击者**：租户 wasm 代码，已经由缺陷 host function / wasmtime 0day **逃逸出 wasm 线性内存**，落在控制面容器（ateom-wasmd 进程，当前 **uid 0**）的原生执行上下文。这是本 ADR 的起点——L1（ADR 0003）已假设失效。
- **攻击目标与外层须遏制的逐项**：
  1. **容器内提权 / 持久化**：篡改容器 rootfs、ateom 二进制、植入持久化；
  2. **经宿主面触达宿主 fs**：当前 worker pod 挂 `run-ateom` **hostPath**（`ateompath.BasePath`，DirectoryOrCreate，MountPropagation HostToContainer）——逃逸者若得容器 root + 该挂载，可写宿主该路径；
  3. **经容器 root + 共享内核做内核提权 → 逃逸到宿主**：runc 无 guest kernel，容器 root（无 userns 时）= 宿主 root，内核漏洞即宿主沦陷；
  4. **东西向触达邻 pod / 集群服务 / 云元数据**：跨租户数据面、169.254.169.254 元数据、kubelet/API；
  5. **抹除审计**：杀掉 ateom 的审计发射、篡改 stdout 流。
- **信任边界（runc 池）**：wasm 线性内存（L1 已破）→ 控制面容器原生进程（**本 ADR 的遏制对象**）→ pod/容器边界（runc，L2）→ 节点/集群（L3）→ 宿主内核（runc 下与容器共享，**最终兜底是 L4 检测 + L3 爆炸半径限制**，而非内核隔离——那是 rund 的角色）。
- **不在本模型内**：业务面容器（平台运营代码，ADR 0002 D1 信任域前提；与控制面容器同属平台信任域，co-location 不依赖 runtime 隔离）；rund 池的 kata VM 边界（ADR 0002 D2 / ADR 0003 D7.1，rund 下 L2/L3 部分由 VM 边界承担）。

## Decision

### D1. 纵深防御栈（runc 池）：四层、分属两仓、逐层兜底

| 层 | 角色 | 内容 | 归属 | 权威 |
|---|---|---|---|---|
| **L1 预防** | 不让逃逸发生 | wasm host function 最小面 + ABI 校验 + capability 唯一执行点 + 内存安全 + 出口代理隔离 + 凭据不落 wasm + fail-closed | sandbox 仓（ateom-wasmd） | ADR 0003 D1–D6,D9 |
| **L2 遏制** | 逃逸后限权 | runc 容器/pod 硬化：非特权、drop ALL、`allowPrivilegeEscalation:false`、钉紧 seccomp、只读 rootfs、**userns**、hostPath 收窄 | substrate fork（workerpool_apply.go） | **本 ADR D2** |
| **L3 爆炸半径** | 限触达面 | 节点专属（不与不可信/跨租户 co-schedule）+ NetworkPolicy 东西向默认拒绝 + 出网仅经代理 + 节点内核/LSM 基线 | substrate fork（调度 + 部署清单） | **本 ADR D3** |
| **L4 检测** | 逃逸可见 + 响应 | Ring 2 内核 eBPF（Tetragon，宿主侧直采 ateom syscall）+ 节点级逃逸响应 | 两仓（部署 + 响应编排） | ADR 0003 D7.1 / **本 ADR D4,D5** |

**逐层兜底不变式**：L1 失效（wasm 被击穿）→ L2 把逃逸者关进最小权能容器 → L2 被绕（容器/内核提权）→ L3 限其触达面（无邻租户、无元数据、出网仅代理）→ L3 被绕 → L4 内核侧不可抹除地**检测**并触发节点级响应。**runc 下 L1–L3 是预防/遏制、L4 是检测；runc 不提供 rund 那样的内核级预防兜底**——这正是 D6 准入判据要求四层全开的原因。

### D2. L2 — runc 容器/pod 遏制硬化（substrate `workerpool_apply.go`）

> 现状经 `cmd/atecontroller/internal/controllers/workerpool_apply.go:306-339`（`ateomSecurityContext`，wasm 分支）核实。wasm 类**已** drop ALL capabilities、privileged=false、无 /dev 挂载（不像 microvm 挂 /dev/kvm）——这是好的起点。下表是**在此基础上须补的硬化**：

| 项 | 现状（wasm 类） | 目标 | 理由（runc 共享内核） |
|---|---|---|---|
| capabilities | **drop ALL** ✅ | 保持 | wasm worker 不 exec runsc、不 pivot root、不 trace、不 program veth/nftables、不 unpack OCI rootfs，无需任何 capability（注释已论证） |
| privileged | **false** ✅ | 保持 | — |
| `allowPrivilegeEscalation` | **未设**（默认 true） | **false** | 阻断 setuid 二进制 / 子进程提权路径；runc 下尤其关键 |
| `seccompProfile` | **隐式默认**（RuntimeDefault） | **显式钉扎 RuntimeDefault** + 评估**自定义收紧 profile** | ateom-wasmd 的 syscall 面极小（wasmtime 运行时 + unix socket + 文件 I/O），可测量后大幅收紧白名单，缩小内核攻击面（L1 逃逸者能发起的 syscall 越少，内核提权越难） |
| **userns（`hostUsers: false`）** | **未启用**（容器 uid 0 = 宿主 uid 0） | **启用 user namespace**，容器 uid 0 映射到宿主高位非特权 uid | **runc 下最关键单点硬化**：消除「容器 root = 宿主 root」。即使逃逸到容器 root，在宿主也是非特权 uid，内核提权与宿主 fs 触达难度数量级上升。**前置依赖**：ateom 当前以 uid 0 在 root-owned hostPath 下建目录（注释 316-320），userns 后该写权限失效 → 须配合 hostPath 属主调整 / `fsGroup` / 改非 hostPath 卷（见开放问题） |
| `readOnlyRootFilesystem` | **未设**（可写） | **true** + 必要处挂 emptyDir（/tmp、ateom 工作目录） | 遏制容器内持久化 / 篡改 ateom 二进制 / 植入工具 |
| **hostPath 爆炸半径** | `run-ateom` hostPath（`ateompath.BasePath`，DirectoryOrCreate，**MountPropagation HostToContainer**） | **收窄**：仅挂 `ateompath.BasePath` 单一子树（已是）、降 MountPropagation、评估部分改 emptyDir/per-pod 卷 | hostPath 是 worker pod 最大的宿主 fs 窗口。它**载荷承重**（atelet 在宿主、ateom 在 pod，经此共享目录上的 unix socket 通信，substrate 节点模型基础，不可移除），但须确保挂载范围最小、属主受控（与 userns 联动） |
| `maskedPaths`/`readOnlyPaths`（/proc 敏感面） | K8s/runc 默认 mask 部分 | 确认 runc 下生效 + 按需加严 | 屏蔽 /proc 内核可写面，减少容器 root 的内核信息/操控面 |
| 资源 limits（CPU/memory） | 经 `WorkerPool.Template.Resources` 注入 | **强制**（CPU≥2、memory limit） | 既是 final-design 池容量派生依据（cgroup 自读），也是 DoS/资源耗尽遏制（ADR 0003 D2 资源限额的 pod 层对应） |

**净效应**：L2 把控制面容器从「uid 0 + 可写 rootfs + 宽 hostPath + 默认提权」收敛为「userns 非特权 + 只读 rootfs + 最小 hostPath + 无提权 + 紧 seccomp + drop ALL」——逃逸者落点从一个接近宿主 root 的环境，变为一个权能最小、宿主面最窄的容器。

### D3. L3 — 节点/集群爆炸半径限制（substrate 调度 + 部署清单）

| 项 | 设计 | 理由 |
|---|---|---|
| **节点租户模型：runc 池专属节点** | runc wasm worker 池经 taint/toleration + nodeSelector 调度到**专属节点池**，**不与不可信工作负载、其他租户的 worker pod co-schedule** | ADR 0002 D1 判据「跨租户共享节点 → 倾向 rund」的 substrate 落地：runc 池 = 专属/同租户节点。runc 共享内核 → 同节点邻 pod 在逃逸爆炸半径内；专属节点把「邻租户」从爆炸半径中移除（与 ADR 0002 F7 每租户池 + 模板 NodeSelector/Affinity 一致，零 CRD delta） |
| **NetworkPolicy 东西向默认拒绝** | runc 池强制 NetworkPolicy：入向仅放行 atenet router（数据面 :443）；出向仅放行 atelet（控制面，注：unix socket 不经网络）、egress 代理、必要 DNS；**显式阻断**云元数据（169.254.169.254）、kubelet/API、邻 pod | ADR 0002 line 45 已指出「出口是 pod 级 ambient 权限、NetworkPolicy 粒度到池」——本 ADR 把它列为 **runc 池强制**。逃逸者即使拿到容器网络栈，东西向也触达不到邻租户/元数据/集群控制面 |
| **出网仅经独立代理** | egress 走 ADR 0003 D5 的独立出口代理（allowlist + 二次校验），叠加节点级出网限制 | 双重限制：L1 代理 allowlist（host function 侧）+ L3 NetworkPolicy（pod 侧）；逃逸者绕过 host function 直接发包仍被 NetworkPolicy 拦 |
| **节点内核/LSM 基线** | runc 池节点统一内核参数 + LSM（AppArmor/SELinux）enforcing 基线；**userns / eBPF 引入前验证 AliSecGuard 兼容** | runc 下宿主内核是最终攻击面；LSM 提供 syscall 之外的强制访问控制。ADR 0003 代价节已记 AliSecGuard × gVisor `max_user_namespaces` 冲突前科——userns（D2）与 Ring 2 eBPF（D4）都须单节点先行验证兼容性 |

### D4. L4 — Ring 2 检测在 runc 池强制（引用 ADR 0003 D7.1）

- **runc 池必须部署 Ring 2（Tetragon）**：runc 下 ateom 直接跑在宿主内核，宿主侧 Tetragon DaemonSet **直接看到 ateom 真实 syscall**（`openat`/`unlinkat`/`renameat`/`connect`/`execve`），是 wasmtime 逃逸（0day）的**唯一进程外、不可抹除检测兜底**（ADR 0003 D7.1）。
- uprobe 扩展挂 wasmtime trap/实例化符号点 = Ring 0 与 Ring 2 间的第三独立信道（subscriber 被攻破/配错时仍由内核采集）。
- **检测 ≠ 预防**：Ring 2 让逃逸**可见**，不阻止；故 D6 准入要求 L1（预防）+ L2/L3（遏制）同时全开，Ring 2 是补偿不是替代。rund 池下 Ring 2 转为 kata VM 边界检测（预防已由 VM 承担，ADR 0003 D7.1）。

### D5. 逃逸响应（runc 池节点级）

检测到逃逸/越权（Ring 2 内核事件，或 L1 host function fail-closed 告警，ADR 0003 D9）→ 编排节点级响应：

1. **隔离 context**（L1，ADR 0003 D9：终止该 context，不放行）；
2. **驱逐 + 冻结节点**：驱逐该 worker pod，cordon 节点（停止新调度）；
3. **取证快照**：保留容器/进程状态、Ring 2 事件链、hostPath 内容（在 L3 专属节点上，取证不波及他租户）；
4. **复核同节点邻租户**：runc 共享内核 → 邻 pod 在爆炸半径内（即便 D3 专属节点已移除「他租户」，仍须复核同池其他 worker）；
5. **触发该租户池 runtime 复核**：逃逸事件 = 该池「wasm 隔离可信」证据被打破 → 按 ADR 0002 D1 判据**降级回 rund**，直至 L1 门禁修复并重验。

### D6. runc 准入判据（统一内外层，操作化 ADR 0002 D1）

> 一个租户池可跑 **runc** 当且仅当下列**全部**成立；任一不成立 → 该池**须 rund**。准入是**持续状态**（CI + 定期复核 + 事件触发重验），非一次性认证。这把 ADR 0002 D1「倾向 runc」的定性判据落成**可核查清单**。
>
> **本准入是 [ADR 0006](0006-security-spectrum-model.md) 三维安全光谱的一个切片**：runc 准入 = 「维度 ① 滑到 runc 端时，对维度 ②（执行体形态）③（工具面）+ L2/L3/L4 的约束」——① 越靠 runc（弱内核隔离），②③ 须越靠安全端（rust-wasm 主体 + 最小工具面）以补偿（ADR 0006 D4 跨维补偿不变式）。

| # | 准入条件 | 层 | 核查方式 | 权威 |
|---|---|---|---|---|
| 1 | ADR 0003 **D8 门禁全绿**（fuzzing 零崩溃 + property 不变式 + 红队无绕过 + wasmtime 无未处置高危 CVE + unsafe 审计 + 供应链干净）= 「wasm 隔离可信」 | L1 | CI 持续 + 定期长时跑 | ADR 0003 D8 |
| 2 | **D2 容器/pod 硬化全开**（drop ALL + allowPrivilegeEscalation:false + seccomp 钉扎 + **userns** + readOnlyRootFilesystem + hostPath 收窄） | L2 | worker pod spec 核查（可脚本化为准入 webhook / 巡检） | 本 ADR D2 |
| 3 | **D3 节点专属 + NetworkPolicy 东西向默认拒绝 + 出网仅代理** | L3 | 调度约束 + NetworkPolicy 存在性核查 | 本 ADR D3 |
| 4 | **D4 Ring 2（Tetragon）已部署且采集正常** | L4 | DaemonSet 就绪 + TracingPolicy 生效 + 事件可达 | ADR 0003 D7.1 / 本 ADR D4 |

**与 ADR 0002 D1 判据的对接**：ADR 0002 D1 的「倾向 runc = wasm 隔离可信 + 低/中敏感租户 + 专属节点 + 高密度诉求」中，「wasm 隔离可信」= 条件 1（ADR 0003 D8），「专属节点」= 条件 3（D3）；本 ADR 补上条件 2（L2 硬化）与条件 4（L4 检测），四者合取即 runc 准入。**高敏感租户 / 跨租户共享节点 → 直接 rund**（不进 runc 准入评估）。

### D7. 与 rund 的关系（纵深不冗余）

- rund 池下，L2/L3 的部分遏制由 **kata VM 硬件虚拟化边界**承担（逃逸被关在 guest VM 内 = 内核级预防，ADR 0002 D2）；但 **L2 容器硬化仍应保留**（纵深防御不冗余：VM 边界本身也可能有漏洞，且容器硬化成本低）。
- rund 池的 Ring 2 转为 **VM 边界检测**（宿主侧看 kata VM 进程对宿主的 syscall + virtio 通道；in-guest eBPF 在 VM 信任域内，guest-kernel 逃逸时可能一并失守，ADR 0003 D7.1 开放问题）。
- **runc 是「拿密度红利但须把 L1–L4 全部做硬」的形态**；rund 是「付 VM 开销换不依赖 wasm 正确性的内核级兜底」的形态。runtime 旋钮（ADR 0002 D1）的每个 runc 池都须过 D6 准入；门禁未全绿期间，多数池落 rund（ADR 0003 代价节「安全优先于密度」的阶段性代价）。

## Consequences

### 正向

- **runc 的实用价值可被安全兑现**：四层纵深（L1 预防 + L2 遏制 + L3 爆炸半径 + L4 检测）把「runc 共享宿主内核」的风险压到可核查、可接受，使密度/成本/兼容红利不再被「runc 不安全」一票否决；
- **userns 消除 runc 最大单点风险**：「容器 root = 宿主 root」是 runc 逃逸致命的根因，userns（`hostUsers:false`）把它转为「容器 root = 宿主非特权 uid」，是性价比最高的外层硬化；
- **准入判据操作化**：ADR 0002 D1 的定性「倾向 runc」落成 D6 四条可核查清单（CI + 巡检 + 事件触发重验），runtime 选型从主观判断变为门禁状态；
- **爆炸半径显式限制**：专属节点（移除邻租户）+ NetworkPolicy（移除东西向/元数据）+ 出网仅代理，使「逃逸后能触达什么」从默认开放变为默认拒绝；
- **检测兜底不可抹除**：Ring 2 内核 eBPF 在 L1–L3 全失效时仍记录逃逸（进程外、guest 与 ateom 均不可绕过），并触发节点级响应 + runtime 降级复核。

### 代价与风险

- **userns / seccomp 收紧可能碰 ateom 既有依赖**：ateom 当前以 uid 0 在 root-owned hostPath 建目录（workerpool_apply.go:316-320），userns 后须解属主/卷形态（开放问题）；自定义 seccomp 须先测量 wasmd 真实 syscall 面，过紧会误杀；
- **AliSecGuard × userns/eBPF 兼容性**：ADR 0003 已记 gVisor `max_user_namespaces` 冲突前科；userns（D2）与 Ring 2 eBPF（D4）引入前**必须单节点验证**集群加固/内核参数不冲突；
- **专属节点池降低调度弹性、抬升成本**：runc 池节点专属 = 不能与其他负载混部，密度红利部分被节点利用率折损抵消（须与「拿 runc 密度」的初衷权衡）；
- **NetworkPolicy / 节点基线运维成本**：东西向默认拒绝须精确放行控制面/数据面/代理链路，错配即断链；
- **L1 门禁是持续投入**（ADR 0003 代价节）：fuzzing 基建、红队、CVE 响应非一次性；门禁任一红 → 对应池退回 rund；
- **hostPath 不可移除**：atelet↔ateom 经共享 hostPath 上的 unix socket 通信（substrate 节点模型基础），L2 只能收窄不能消除该宿主面。

### 与 ADR 0002 / 0003 的关系

- **落实** ADR 0002 D2「外层（pod runtime）必须强到能兜住内层（wasm）失效」这一原则的**外层设计**（ADR 0002 只声明原则，本 ADR 设计 runc 容器/节点硬化）；
- **操作化** ADR 0002 D1 的 runc 充分性前提与 runtime 选型判据（D6 准入清单）；
- **补全** ADR 0003：0003 是内层（L1 预防 + L4 检测的审计模型），本 ADR 是外层（L2 遏制 + L3 爆炸半径）+ 逃逸响应（D5，0003 D9 只提「runc 池额外触发节点级响应」未设计）+ 统一准入（D6，把 0003 D8 门禁纳入四条件之一）；
- 三 ADR 合力：0002（架构 + runtime 旋钮）→ 0003（内层 wasm 安全）→ 0004（外层 runc 遏制 + 准入），构成 runc 安全增强的完整设计链。

### 后续行动

1. **substrate `workerpool_apply.go`（wasm 分支）L2 硬化**：补 `allowPrivilegeEscalation:false`、显式 `seccompProfile: RuntimeDefault`、`readOnlyRootFilesystem` + emptyDir、评估 `hostUsers:false`（userns）+ hostPath 属主/卷形态联动；更新 `ateomSecurityContext` 测试（`workerpool_apply_test.go`）；按 xuanji 约定登记 `xuanji.md`（触碰 upstream 文件）；
2. **测量 ateom-wasmd 真实 syscall 面** → 评估自定义 seccomp profile 白名单（sandbox 仓 + substrate 部署清单）；
3. **L3 部署清单**：runc 池专属节点（taint/toleration/nodeSelector）+ NetworkPolicy（东西向默认拒绝、阻断元数据/邻 pod、放行控制面/数据面/代理）落 `manifests/xuanji/`；
4. **L4 Ring 2 部署**（ADR 0003 后续行动 6 二期）：Tetragon TracingPolicy（syscall + uprobe）按 runc/rund 分置，单节点验证 AliSecGuard 兼容 → 灰度 → DaemonSet；
5. **D6 准入核查自动化**：worker pod spec 硬化核查（准入 webhook / 巡检脚本）+ Ring 2 就绪核查 + ADR 0003 D8 门禁状态对接 → 输出「该租户池是否可 runc」的持续判定；
6. **D5 逃逸响应编排**：Ring 2 事件 / host function fail-closed 告警 → 驱逐 + cordon + 取证 + 邻租户复核 + runtime 降级复核的自动化 runbook；
7. **userns × hostPath × AliSecGuard 单节点验证**（前置阻塞项，须先于 D2 userns 上线）。

## 开放问题

- **userns 与 ateom hostPath uid 0 写依赖的解法**：`hostUsers:false` 后容器 uid 0 映射宿主高位 uid，root-owned hostPath（`ateompath.BasePath`）写权限失效——解法候选：① kubelet/初始化把该子树 chown 到映射范围；② `fsGroup` + 卷属主协调；③ atelet↔ateom 共享面改用非 hostPath 卷形态（但 substrate 节点模型依赖宿主共享目录，改动面大）。须实测；
- **自定义 seccomp profile 的 syscall 白名单**：ateom-wasmd（wasmtime + unix socket + 文件 I/O + atunnel TLS）的真实 syscall 面测量与最小化；过紧误杀风险与回退策略；
- **AliSecGuard × userns / eBPF 兼容性**：本集群加固方案是否允许 unprivileged userns、是否拦截 Tetragon eBPF 加载（ADR 0003 已有 gVisor `max_user_namespaces` 冲突前科）；
- **专属节点池的成本/弹性权衡**：runc 池节点专属与「拿 runc 密度红利」初衷的张力——是否允许同租户多池共享专属节点、跨租户绝对禁止的边界；
- **hostPath 收窄的极限**：atelet↔ateom unix socket 共享目录能否进一步缩小挂载范围 / 降 MountPropagation 而不破坏节点模型；
- **准入核查的强制形态**：D6 是建议性巡检还是强制准入 webhook（拒绝不满足 L2/L3/L4 的 runc WorkerPool 创建）；
- **L2 硬化对 gvisor/microvm 类的外溢**：本 ADR 聚焦 wasm 类 runc 池；`allowPrivilegeEscalation:false` / seccomp 钉扎等是否也适用于 gvisor 类（其需 ateomGvisorCapabilities，约束不同）。
