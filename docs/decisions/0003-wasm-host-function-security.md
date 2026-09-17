# 0003. Wasm Host Function 安全模型（runc 基线的前置硬化）

- Status: Proposed
- Date: 2026-09-18
- 关联: [ADR 0002](0002-two-tier-supervisor-worker-sandbox.md)（落实其开放问题「wasm 隔离可信度的建立」与后续行动「wasm 定制工具生态」的安全面）
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

### D7. 审计全事件、不可绕过

- 每次 host function 调用产生审计事件：context id、capability、函数名、参数摘要（脱敏，不含明文凭据/全量 body）、结果、时延。
- 审计流 **append-only**、不可被 wasm 关闭或篡改；归集到 Tier-2 自研管控（ADR 0002 D2 审计机制，类比 DE agentshell）。

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

### D9. 失败模式与降级（fail-closed）

- host function panic / 异常 → **终止该 context**，不放行、不静默吞错、不降级到「无校验」路径。
- 资源限额触发 → context 终止 + 审计。
- 检测到逃逸 / 越权迹象 → 隔离 context + 告警；**runc 池**额外触发节点级响应（驱逐 / 取证 / 复核同节点邻租户）。

## Consequences

### 正向

- host function 面收敛 + 硬化 + 可验证 → **「wasm 隔离可信」从主观假设变为可量化的门禁状态**，runc 适用面可随门禁成熟逐步放宽（ADR 0002 张力 D 的落地路径）；
- capability 强制点、审计点、出口管控收敛在 host function 一处，安全语义集中、可审计、可测试；
- 攻击面显著小于原生 agent CLI 的 ambient 工具（bash/git/任意 syscall），与 ADR 0002「wasm 定制工具」一致；
- 默认拒绝 + 最小面 + 登记制 → 面的扩张是受控的、可追溯的。

### 代价与风险

- 最小面 + 验证门禁是**持续工程投入**（fuzzing 基建、红队、审计、CVE 响应），非一次性；
- host function 面受限 → wasm agent 能力面受约束（与 ADR 0002 张力 C 同源）；新工具须先过 D1 登记 + D8 门禁，交付节奏受安全门约束；
- wasmtime 升级须重跑全门禁 → 运行时版本演进有验证成本；
- 门禁未全绿期间，runc 适用面窄、多数租户池须 rund（kata 开销）——这是「安全优先于密度」的阶段性代价。

### 与 ADR 0002 的关系

- 本 ADR **落实** ADR 0002 开放问题「wasm 隔离可信度的建立」与后续行动 5「wasm 定制工具生态」的安全面；
- **D8 门禁全绿是 ADR 0002 中 runc 基线成立的前置条件**；门禁状态即「wasm 隔离可信度」的度量，直接喂给 ADR 0002 D1 的 runtime 选型判据（每租户池策略）。

### 后续行动

1. sandbox 仓 ateom-wasmd：按 D1 最小面收敛现有 host function，补 D2 边界校验、D3 capability 校验、D9 fail-closed；
2. 建 fuzzing 基建（每个 host function 一个 fuzz target）+ property-based 测试 + 红队回归套件，纳入 CI（D8）；
3. wasmtime 版本钉扎 + CVE 跟踪流程；`cargo-audit`/`cargo-deny` 入 CI；
4. 出口代理（D5）与凭据代取（D6）实现 + 双校验；
5. 门禁状态 → ADR 0002 runtime 判据的对接（门禁全绿的租户池方可选 runc）。

## 开放问题

- host function 最小集的**具体清单与 ABI 版本管理**（wit 组件模型 vs 经典 WASI 扩展）；
- fuzzing 覆盖率门槛与 CI 时长预算；结构感知 fuzzing 的语料维护；
- wasmtime CVE 响应 SLA 与升级窗口；
- 红队 / 逃逸测试的范围、频率与外部审计介入；
- 多语言 wasm agent（python-wasm vs rust wasm 模块）对 host function 面的差异（解释器自身是否引入额外面）；
- 控制面 ↔ 业务面容器的接口契约（ADR 0002 开放问题）是否引入额外 host function 或 IPC 面，及其安全语义；
- capability 令牌的具体密码学方案（MAC vs 不透明索引 + 服务端表）与吊销传播（ADR 0002 开放问题「capability 中途吊销」）。
