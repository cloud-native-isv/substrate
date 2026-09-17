# 0003. Wasm Host Function 安全模型（runc 基线的前置硬化）

- Status: Proposed
- Date: 2026-09-18
- 关联: [ADR 0002](0002-two-tier-supervisor-worker-sandbox.md)（落实其开放问题「wasm 隔离可信度的建立」与后续行动「wasm 定制工具生态」的安全面）
- 采纳来源: 《WASM 沙箱外部可信审计方案（技术调研 + 三环模型）》设计定稿 v2.1（钉钉知识库「基于轻量级沙箱的下一代agent sandbox技术 / wasm-E2B-K8s 实现」，[节点](https://alidocs.dingtalk.com/i/nodes/NZQYprEoWoxKPoqwCBwrorbeV1waOeDk)）——本 ADR D7 采纳其三环外部可信审计模型，并把术语从 sandbox 仓 envd/wasm-runner 映射到本设计的 ateom herder
- Supersedes: -
- Superseded by: -

## Context

ADR 0002 把 actor 执行体定为 **wasm runtime + wasm agent program**（跑在 worker pod 控制面容器内，由 ateom 以 wasmtime host 承载），并在 D3 把**出口 / fs / secret / capability 校验全部收敛到 host function**——host function 是 wasm 沙箱的唯一执行点（capability-based security 的强制点）。

ADR 0002 的 runtime 解耦进一步确立：**runc 基线的充分性条件依赖「wasm 隔离可信」**，隔离是纵深防御栈、外层（pod runtime）须兜住内层（wasm）失效。而 **host function 正是 wasm 边界最可能失效之处**：它是租户不可信代码离开 wasm 线性内存、触达控制面容器原生进程的**唯一通道**，本身是安全关键的原生（Rust）代码。一旦 host function 存在内存安全 / 逻辑 / 能力校验绕过缺陷，租户代码即可逃逸出 wasm；此时若 worker pod 是 runc（共享宿主内核），一次容器/内核逃逸即危及宿主与同节点邻 pod（跨租户）。

**因此 host function 面的安全质量直接决定 runc 适用面能放多宽**：host function 越收敛、越硬化、越经验证，「wasm 隔离可信」越成立，越多租户/场景可下放 runc 拿密度红利；反之则必须 rund 兜底。本 ADR 定义 host function 的安全模型、最小面、ABI 边界契约与**runc 准入的硬化/验证门禁**。

实现载体：sandbox 仓的 **ateom-wasmd 演进版 herder**（ADR 0002 后续行动 2）；本 ADR 是其安全设计权威。

### 威胁模型

- **攻击者**：租户提供的 wasm agent 模块（不可信代码），目标是逃逸出 wasm 沙箱、越过 capability 限制、触达控制面容器原生进程乃至宿主/邻租户。
- **攻击面**：① host function 入参（wasm→native 边界的 ptr/len/offset/字符串/路径/令牌/结构体）；② capability 校验逻辑（绕过/重放/伪造句柄）；③ 出口代理（SSRF、allowlist 绕过、TLS 处理）；④ 凭据通道（明文泄漏进 wasm 内存）；⑤ wasmtime 运行时自身漏洞；⑥ 资源耗尽（燃料/内存/句柄/出口频率）。
- **信任边界**：wasm 线性内存内 = 不可信；host function 入口 = 信任切换点（必须全量校验）；控制面容器原生进程 = 平台可信（但被 host function 缺陷击穿后即失守）；pod 边界（runc/rund）= 纵深外层。
- **审计信任原则**（采纳既有方案 v2.1）：审计数据的**产生与存储必须全部位于信任边界外侧**——guest 侧审计（CPython `sys.addaudithook`、wasi-observe/wasi-otel）与被审计代码同权级、位于边界内侧：hook 门控状态可被内省手段（`sys.modules`/`gc`/frame 遍历）篡改、数据上报前可被污染、一次性执行路径零覆盖，故一律定级 **untrusted 遥测、永不参与安全结论**（仅作明确标注的可选辅助信号）。可信审计 = 外部三环（D7）。
- **不在本模型内**：业务面容器（平台运营代码，与 host function 面正交，见 ADR 0002 D1 信任域前提）；控制面/业务面接口契约（另见开放问题）。

## Decision

### D1. 最小 host function 面（allowlist，默认拒绝）

host function 集合是**显式 allowlist**，未列出的一律不可见（与 WASI 能力模型同构、与 ADR 0002 D3「未授予即不可见」一致）。新增任何 host function 须走 review + 在本表登记 + 补 fuzzing/测试，否则不得进入运行时。

| 能力域 | host function（最小集） | 授予的 capability | 备注 |
|---|---|---|---|
| 文件 · actor-config（只读） | `fs_open_ro` / `fs_read` / `fs_close` | `cap:fs:actor-config:ro` | 限定 preopen 子树；只读 |
| 文件 · workspace（读写子目录） | `fs_open_rw` / `fs_read` / `fs_write` / `fs_close` / `fs_stat` | `cap:fs:workspace:rw:<subdir>` | per-context 遮蔽到授予子目录 |
| 出口 | `egress_request` | `cap:net:egress:<token>` | **不直接开 socket**，转发独立出口代理（D5） |
| 凭据 | `secret_get` | `cap:secret:<name>` | ateom 代取，**明文不入 wasm 线性内存**（D6） |
| 时钟 / 随机 | WASI `clock_time_get` / `random_get` | （WASI baseline） | 标准 wasmtime-wasi，安全随机源 |
| 燃料 / 让出 | wasmtime epoch interruption / fuel（隐式） | （runtime 内建） | 限额触发 context 终止 |
| 审计 | 隐式（每次调用记录）+ 可选 `audit_emit` | （runtime 内建） | append-only、不可被 wasm 关闭（D7） |
| **默认拒绝** | 原始 socket / process spawn / thread spawn / ambient fs（preopen 外）/ env 读取（除显式注入）/ 任意 FFI / 动态代码加载 | — | 未授予即不可见 |

> 工具生态（ADR 0002 张力 C「wasm 定制工具」）一律实现为**上表能力的组合**或**经审计的新增 host function**，不引入原生工具（git/chromium/dws/a1）通道。

### D2. ABI 与边界校验契约（信任切换点的强制规则）

每个 host function 在 wasm→native 边界处**默认不信任 wasm 侧任何输入**，进入原生逻辑前完成全量校验：

- **内存边界**：所有 `(ptr, len, offset)` 解引用前经 wasmtime 内存边界检查（落在该 context 线性内存内），拒绝越界；不缓存跨调用的裸指针。
- **整数安全**：所有 size/offset/count 用 checked 算术，拒绝溢出；强制上限（单次 body 大小、路径长度、句柄数）。
- **字符串 / 路径**：强制 UTF-8；路径**规范化后**校验仍在授予子树内（解析 `..`、符号链接后重验，拒绝逃逸 preopen）；拒绝 NUL 注入与控制字符。
- **结构体**：经组件模型（wit）或显式反序列化，**不**把 wasm 内存直接 `transmute` 为 native 结构。
- **令牌**：capability 令牌为不可伪造句柄（D3），格式/签名/MAC 在入口校验，拒绝重放（绑定 context + 单调计数/时效）。
- **资源限额**：每 context 燃料上限、内存上限、句柄数上限、出口 body 大小与频率限额；限额即终止（D9）。

### D3. capability 校验 = 唯一执行点 + 不可伪造句柄

- 每次 host function 调用**先校验 capability**：句柄不可伪造（密码学 MAC / 不透明索引 + 服务端表）、spawn 时仅授予子集、用尽/过期即失效。
- 校验失败 **fail-closed**（终止调用 + 审计），无 ambient 权限、无「默认放行」分支。
- capability 表权威在控制面（ResumeActor 下发，ADR 0002 D3）；host function 侧只读校验，不可被 wasm 改写。

### D4. 内存安全与实现纪律

- host function 实现 = **safe Rust**；`unsafe` 块最小化、集中、单独审计并注释安全不变式。
- 不在 host function 内做裸指针运算 / C FFI（除非经审计的最小核，如 crypto 原语），其余一律 safe 抽象。
- 每 context 独立 wasmtime `Store` / 实例，**无跨 context 共享可变状态**（杜绝 context 间侧信道与状态污染）。
- 依赖供应链：crates 钉版本 + `cargo-audit`/`cargo-deny` 纳入 CI；wasmtime 版本钉扎 + CVE 跟踪（D8）。

### D5. 出口代理隔离

- `egress_request` host function **本身不持网络权限**；转发到**独立出口代理**（最小权限进程/受限线程），代理持 allowlist 与 TLS 终止/发起。
- 令牌与 allowlist 在**代理侧二次校验**（host function 侧 + 代理侧双校验）；防 SSRF（拒绝内网/元数据地址、限定 allowlist host:port）。
- 代理与控制面容器其余部分故障隔离（代理崩溃不波及 ateom 主体）。

### D6. 凭据不落 wasm 内存

- `cap:secret` 的值由 ateom 代取并**用于代理侧操作**（如签名出口请求、注入凭据头），明文**不写入 wasm 线性内存**。
- 若语义上必须向 wasm 暴露值，用后 `zeroize`，且经审计标记为高敏（默认走「代取不返值」）。

### D7. 三环外部可信审计（采纳既有方案 v2.1）

审计采用**三环纵深、全部位于信任边界外侧**的模型（采纳来源见 header）。核心：guest 侧审计不可信（见威胁模型「审计信任原则」），可信审计点全部在 host/进程外。术语映射：原方案的 envd/wasm-runner → 本设计的 **ateom herder（控制面容器内）**；sandbox_id → **actor / context id**；HostProxy → **D5 出口代理**；gateway/scheduler → **e2bgw + Tier-2 自研管控**。

| 环 | 位置 | 防绕过能力 | 内容 | 默认状态 |
|---|---|---|---|---|
| **Ring 0**（主审计面） | ateom 进程内、WASI/host function 边界 host 侧 | guest 无法绕过 | **档位 A**：消费 wasmtime-wasi upstream `#[instrument]` span（~45 个 WASI p1 syscall 入口，host 侧已插好观测点；INFO 级、仅入参、文本日志非稳定契约）；**档位 B（演进）**：WASI preview2 / Component Model + p2 公开 Host trait 结构化拦截——每个 host function 先产出结构化事件（**含返回值**）再委托内置实现，可升级为**可阻断的策略执行点**（= D3 capability 强制点的审计强化形态） | 档位 A 按需开（取证）；档位 B 常开（演进） |
| **Ring 1** | ateom 进程内（沙箱外部） | guest 无法绕过 | ① **执行前代码存档 + 静态审计**（ateom 在 guest 看到代码前先拿到，不可绕过：sha256 + 长度 + 截断原文 + 高危模式标记；定位为**信号**非防线，可被混淆绕过，运行中行为由 Ring 0 兜底）；② **出口代理审计**（= D5，每请求一条，含 allowed/blocked）；③ **资源计量**（fuel 差值 `store.get_fuel()`、epoch/OOM trap）；④ **I/O 存档**（可选开关，默认关） | 常开 |
| **Ring 2** | ateom 进程**外**（内核 eBPF） | guest 与 ateom **均**无法绕过 | Tetragon（Cilium）DaemonSet + TracingPolicy CRD：跟踪 ateom 真实 syscall（`openat`/`unlinkat`/`renameat`/`connect`/`execve`），**即使 ateom 被攻破（wasmtime 逃逸 0day）内核侧记录不可抹除**；uprobe 扩展挂 wasmtime trap/实例化稳定符号点 = Ring 0 与 Ring 2 间的**第三独立信道**（tracing subscriber 被攻破/配错时仍由内核采集） | 第二期部署 |

- **完备性论证（支撑 D8「wasm 隔离可信」）**：WASM 安全模型 = **deny-by-default 导入表**——guest 影响外部世界的唯一途径是调用 host 显式导入函数（即 D1 的 host function 面；帧协议本身也是 `fd_read`/`fd_write`）。纯线性内存内计算外部不可见、但也无法影响外部，无需观测。故 **完整覆盖 host function 边界（Ring 0）+ 应用层 I/O（Ring 1）+ 资源计量 = 100% 覆盖沙箱所有外部可见行为**——这是相对容器审计（数百 syscall、`/proc`、共享内核攻击面）的**结构性优势**，也是「wasm 隔离可信」可论证的根据。
- **语义上限（诚实声明）**：外部审计看到的是**行为**（开了什么文件、发了什么请求、耗了多少 fuel），不是「Python/agent 函数名」；函数级语义从沙箱外不可得（信息论限制）。可信的「审计 agent 干了什么」= 执行前静态审计（信号）+ 执行中边界行为审计（防线）组合近似。
- **信任链闭合**：Ring 0/1 的可信前提 = 「ateom 进程未被攻破」；绕过它们的唯一路径 = wasmtime 逃逸（0day）——**Ring 2 内核层恰好覆盖该场景**。三环分属三个故障域（wasmtime 边界 / ateom 进程 / 内核），无单点信任；uprobe 再加第三条独立信道。
- **统一事件模型**：`target="audit"` 结构化日志，必带 `actor/context id + ring ∈ {0,1,2} + event`，明细按环各异（ring0: syscall/入参；ring1: code_sha256/url/method/allowed/fuel_used；ring2: syscall/path/daddr/uprobe 符号）→ stdout → K8s 日志采集 → 归集到 Tier-2 自研管控（ADR 0002 D2 审计机制，类比 DE agentshell 直投安全 SLS）。**审计流不写 Etcd**（遵守状态层写约束）。
- **分期**：一期 = Ring 0 档位 A + Ring 1（约 80 行、零新增依赖、零协议改动、e2bgw/控制面零改动、E2B SDK 无感知）；二期 = Ring 2 Tetragon 单节点试点→灰度→DaemonSet；演进项（独立立项）= Ring 0 档位 B（adapter PoC 前置）。一期为 **audit-only**，阻断能力依赖档位 B。

#### D7.1 Ring 2 与 pod runtime（runc/rund）的交互（本设计新增的关键考量）

原审计方案针对单层 sandbox（runc/gVisor 路径）撰写；落到 ADR 0002 的 runc/rund 双 runtime 后，**Ring 2 的可观测性随 runtime 不同而不同**，必须显式处理：

- **runc worker pod**：ateom 直接跑在宿主内核上 → 宿主侧 Tetragon（DaemonSet）**直接看到 ateom 的真实 syscall**，Ring 2 按原方案成立（进程外、不可抹除）。
- **rund/kata worker pod**：ateom 跑在 **guest VM 的独立 guest kernel** 内 → 宿主侧 Tetragon 只看到 kata VM 进程对宿主的 syscall（virtio-fs/virtio-blk/vhost-net 等），**看不到 guest 内 ateom 的 syscall**。要审计 guest 内 syscall，Ring 2 须以 **in-guest eBPF（guest 内 Tetragon agent）** 部署——但 guest kernel 在 kata VM 信任域内，若逃逸正是 guest-kernel 漏洞，in-guest eBPF 可能被一并攻破。
- **结论**：rund 下「Ring 2 兜底 wasmtime 逃逸」的角色**部分由 kata VM 硬件虚拟化边界本身承担**（逃逸被关在 guest VM 内 = 预防），宿主侧 Tetragon 转为审计 **VM 对宿主的边界行为**（virtio 通道、VM 进程 syscall = 检测）。即 **rund 把 Ring 2 的「检测」语义升级为「预防 + 边界检测」**；runc 下 Ring 2 仍是纯检测、预防依赖 wasm + 容器边界。
- **对 runtime 判据的反哺**：三环审计（尤其 Ring 2）提供**检测纵深**，部分补偿 runc 较弱的预防——runc + 三环审计下，wasm 逃逸→容器逃逸会被内核 eBPF **检测到**（虽非阻止）。但**检测 ≠ 预防**，故 ADR 0002 D1「runc 充分性以 wasm 隔离可信为前提」不变；高敏感/共享节点仍须 rund（预防）。Ring 2 在 runc 池是「逃逸可见性」的关键补偿，应优先在 runc 池部署。

### D8. 硬化与验证门禁（runc 准入）

> **本门禁全绿 = 「wasm 隔离可信」达到 runc 准入**；任一红 = 该 host function 面下 runc 不成立、对应租户池须 rund（ADR 0002 D1 判据）。门禁是**持续状态**（CI + 定期），不是一次性认证。

| 门禁 | 内容 | 通过判据 |
|---|---|---|
| **Fuzzing** | `cargo-fuzz`/libFuzzer 覆盖**每个** host function 的边界输入（结构感知优先）；CI 持续 + 定期长时跑 | 崩溃零容忍；新增 host function 必带 fuzz target |
| **Property-based 测试** | capability 校验、路径规范化、限额逻辑、令牌防重放的不变式 | 不变式无违反 |
| **红队 / 逃逸回归** | 已知 wasm 逃逸模式回归；capability 绕过尝试（伪造句柄、越权路径、令牌重放、TOCTOU） | 全部 fail-closed，无绕过 |
| **wasmtime 钉扎 + CVE** | 版本钉扎、CVE 跟踪、升级后**重跑全门禁** | 无未处置高危 CVE |
| **代码审计** | `unsafe` 集中审计；最小面 review（D1 新增登记） | unsafe 均有安全不变式注释 + 审计记录 |
| **供应链** | `cargo-audit`/`cargo-deny` 入 CI | 无未处置高危告警 |

> **审计覆盖作为准入证据**：D7 完备性论证（Ring 0 + Ring 1 = 100% 覆盖沙箱外部可见行为）是 runc 准入的**可观测性基础**——「wasm 隔离可信」不仅靠 host function 无缺陷（上表门禁），还靠**逃逸可被外部观测**。**runc 池必须部署 Ring 2**（D7.1：runc 下 Ring 2 是 wasmtime 逃逸的唯一进程外检测兜底）；rund 池的 Ring 2 转为 VM 边界检测（预防已由 kata VM 承担）。

### D9. 失败模式与降级（fail-closed）

- host function panic / 异常 → **终止该 context**，不放行、不静默吞错、不降级到「无校验」路径。
- 资源限额触发 → context 终止 + 审计。
- 检测到逃逸 / 越权迹象 → 隔离 context + 告警；**runc 池**额外触发节点级响应（驱逐 / 取证 / 复核同节点邻租户）。

## Consequences

### 正向

- host function 面收敛 + 硬化 + 可验证 → **「wasm 隔离可信」从主观假设变为可量化的门禁状态**，runc 适用面可随门禁成熟逐步放宽（ADR 0002 张力 D 的落地路径）；
- capability 强制点、审计点、出口管控收敛在 host function 一处，安全语义集中、可审计、可测试；
- 攻击面显著小于原生 agent CLI 的 ambient 工具（bash/git/任意 syscall），与 ADR 0002「wasm 定制工具」一致；
- 默认拒绝 + 最小面 + 登记制 → 面的扩张是受控的、可追溯的；
- **三环外部可信审计**（采纳既有方案 v2.1）：审计点全在信任边界外侧、guest 不可绕过；完备性论证（deny-by-default 导入表 → Ring 0+1 覆盖 100% 外部可见行为）把「wasm 隔离可信」从工程经验上升为**可论证结论**；三环分属三故障域、无单点信任，Ring 2 内核 eBPF 兜底 wasmtime 逃逸（0day）。

### 代价与风险

- 最小面 + 验证门禁是**持续工程投入**（fuzzing 基建、红队、审计、CVE 响应），非一次性；
- host function 面受限 → wasm agent 能力面受约束（与 ADR 0002 张力 C 同源）；新工具须先过 D1 登记 + D8 门禁，交付节奏受安全门约束；
- wasmtime 升级须重跑全门禁 → 运行时版本演进有验证成本；
- 门禁未全绿期间，runc 适用面窄、多数租户池须 rund（kata 开销）——这是「安全优先于密度」的阶段性代价；
- **Ring 2（Tetragon）部署成本与集群兼容性**：eBPF 组件与本集群 AliSecGuard 加固/内核参数有冲突前科（gVisor `user.max_user_namespaces` 事件），须单节点先行验证；rund 池下 Ring 2 语义变化（D7.1）需 in-guest eBPF 或转 VM 边界检测，运维更复杂；
- **审计 POC 边界**（采纳方案 §11）：一期 **audit-only 无阻断**（阻断依赖档位 B 演进）、仅 stdout 无持久化/检索/告警链路、Ring 2 一期不部署（wasmtime 逃逸兜底审计在二期前**空窗**）、档位 B 未经 PoC（官方 python.wasm 的 component adapter 可行性 + WASI 0.3/p3 接口漂移为已知债务）。

### 与 ADR 0002 的关系

- 本 ADR **落实** ADR 0002 开放问题「wasm 隔离可信度的建立」与后续行动 5「wasm 定制工具生态」的安全面；
- **D8 门禁全绿是 ADR 0002 中 runc 基线成立的前置条件**；门禁状态即「wasm 隔离可信度」的度量，直接喂给 ADR 0002 D1 的 runtime 选型判据（每租户池策略）。

### 后续行动

1. sandbox 仓 ateom-wasmd：按 D1 最小面收敛现有 host function，补 D2 边界校验、D3 capability 校验、D9 fail-closed；
2. 建 fuzzing 基建（每个 host function 一个 fuzz target）+ property-based 测试 + 红队回归套件，纳入 CI（D8）；
3. wasmtime 版本钉扎 + CVE 跟踪流程；`cargo-audit`/`cargo-deny` 入 CI；
4. 出口代理（D5）与凭据代取（D6）实现 + 双校验；
5. 门禁状态 → ADR 0002 runtime 判据的对接（门禁全绿的租户池方可选 runc）；
6. **三环审计落地**（采纳方案，sandbox 仓 ateom-wasmd）：一期 Ring 0 档位 A（subscriber 改造 + actor/context id span 归属）+ Ring 1（执行前代码存档/静态审计 + 出口代理审计整合 + fuel 计量）+ 统一 `target="audit"` 出口；二期 Ring 2 Tetragon（syscall + uprobe 两套 TracingPolicy，单节点→灰度→DaemonSet，**按 runc/rund 分置**——runc 池宿主侧直采、rund 池 VM 边界/in-guest，见 D7.1）；演进 Ring 0 档位 B（adapter PoC → p2 Host trait 结构化拦截 → 可阻断策略执行点）。

## 开放问题

- host function 最小集的**具体清单与 ABI 版本管理**（wit 组件模型 vs 经典 WASI 扩展）；
- fuzzing 覆盖率门槛与 CI 时长预算；结构感知 fuzzing 的语料维护；
- wasmtime CVE 响应 SLA 与升级窗口；
- 红队 / 逃逸测试的范围、频率与外部审计介入；
- 多语言 wasm agent（python-wasm vs rust wasm 模块）对 host function 面的差异（解释器自身是否引入额外面）；
- 控制面 ↔ 业务面容器的接口契约（ADR 0002 开放问题）是否引入额外 host function 或 IPC 面，及其安全语义；
- capability 令牌的具体密码学方案（MAC vs 不透明索引 + 服务端表）与吊销传播（ADR 0002 开放问题「capability 中途吊销」）；
- **Ring 2 在 rund/kata 下的部署形态**（D7.1）：in-guest eBPF（guest 内 Tetragon，但 guest kernel 在 VM 信任域内、guest-kernel 逃逸时可能一并失守）vs 宿主侧 VM 边界审计（virtio 通道 + VM 进程 syscall）——两者覆盖与信任属性不同，需定方案；
- **审计持久化 / 检索 / 告警链路**（采纳方案 POC 边界外）：stdout → K8s 日志采集之后的落库（类比 DE agentshell 直投安全 SLS）、检索 API、告警与阻断联动；
- **Ring 0 档位 B 的 WASI 0.3（p3）接口漂移**：锁 wasmtime 大版本、p3 迁移列为已知债务；
- **统一审计事件 schema 版本管理与跨环关联**：actor/context id 在 Ring 0/1/2 的一致性（Ring 2 按路径前缀/目的地址关联 sandbox 的可靠性）。
