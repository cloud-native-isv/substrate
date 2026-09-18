# 0006. 安全光谱模型：三维滑标（pod runtime × 执行体形态 × 工具/host function 面）

- Status: Proposed
- Date: 2026-09-18
- 关联: 统一 [ADR 0002](0002-two-tier-supervisor-worker-sandbox.md) D1（runtime 旋钮）、[ADR 0003](0003-wasm-host-function-security.md) D1（host function 最小面）、[ADR 0004](0004-runc-worker-pod-hardening.md) D6（runc 准入）、[ADR 0005](0005-rust-agent-execution-body.md)（执行体形态）——本 ADR 是四者的**总框架**，各维细节仍归所属 ADR
- Supersedes: -
- Superseded by: -

## Context

ADR 0002–0005 各自定义了与安全相关的旋钮：runtime `runc`/`rund`（0002 D1）、host function 最小面（0003 D1）、runc 外层硬化与准入（0004）、执行体 `rust-wasm` vs `cpython-wasm`（0005）。但这些旋钮此前被**隐含地当作「二选一/非黑即白」**（runc OR rund、rust OR cpython、最小面 OR 不限），且彼此关系未显式化。

**用户决策（2026-09-18）**：整体设计上，**安全应是光谱逻辑而非非黑即白**——存在**三个滑标/维度**：① pod runtime class（runc ↔ rund）；② agent 程序主体（cpython-wasm 解释执行 ↔ rust-wasm 编译执行）；③ 暴露的工具/host function 的多寡。**滑标挪到不同位置即呈现不同安全等级**。

本 ADR 把分散的旋钮统一为**三维安全光谱模型**，作为每租户池安全策略的总框架：定义三维、各维梯度、为何是光谱（中间位置 + 复合等级）、跨维补偿不变式、与既有判据（0002 D1 / 0004 D6 / 0005）的统一、以及策略档位与度量。

## Decision

### D1. 三维滑标（每维两端 = 安全梯度方向）

| 维度 | 滑标两端 | 安全梯度 | 成本（向安全端的代价） | 所属 ADR |
|---|---|---|---|---|
| **① pod runtime class** | `runc` ↔ `rund` | **rund 更安全**（独立 guest kernel、硬件虚拟化边界、逃逸难度数量级高）；runc 共享宿主内核 | rund 付 VM 开销（密度/成本/启动）；runc 拿密度红利 | ADR 0002 D1 + ADR 0004 |
| **② agent 程序主体** | `cpython-wasm`（解释执行）↔ `rust-wasm`（编译执行） | **rust-wasm 更安全**（小 TCB、safe Rust 内存安全、静态可审计、无 ambient 动态代码加载、导入面=标准 WASI）；cpython-wasm 兼容任意 Python 负载 | rust-wasm 付兼容损失（任意 Python 不能作主体）+ 迭代成本；cpython-wasm 付大 TCB/非内存安全/动态面 | ADR 0005 |
| **③ 工具/host function 面** | 少（最小面）↔ 多（丰富工具） | **少更安全**（攻击面小、capability 收敛、Ring 0/1 审计完备性论证最干净）；多 = agent 能力更强 | 最小面付 agent 能力受限；丰富面付每工具须过 D1 登记 + D8 门禁的扩张成本 | ADR 0003 D1 |

### D2. 为什么是光谱不是非黑即白（中间位置 + 复合等级）

**每维都有中间位置，不止两端**：

- **① runtime**：`runc` / `runsc`(gVisor，强于 runc、轻于 rund 的中间档，ADR 0002 D1 已提) / `rund`。
- **② 执行体**：
  - 纯 `cpython-wasm` 主体（兼容端，Python agent 即主体）；
  - **`rust agent + python_exec 工具`（混合中间位，ADR 0005 设计）**——主体是可信 safe Rust，动态 Python 执行被降为受 capability+审计+子沙箱约束的**工具**；这是推荐的平衡点；
  - 纯 `rust-wasm` 主体（安全端，无 Python）。
- **③ 工具面**：从「仅 `egress_request` + `fs`」（最小）到「丰富工具集」，每新增工具过 ADR 0003 D1 登记 + D8 门禁，面**受控扩张**（非任意膨胀）。

**复合安全等级 = 三维位置的组合**，不由任一维单独决定：

- 安全等级随三维向各自「安全端」移动而**单调升高**，但每维的**边际收益与成本不同**（rund 的内核隔离收益 vs VM 开销；rust-wasm 的 TCB 收益 vs 兼容损失；最小面的攻击面收益 vs 能力损失）。
- 光谱模型让平台**按租户威胁模型在三维上分别取舍**，而非全局二选一——这正是 ADR 0002 D1「runtime = 每租户池策略旋钮」的推广：**安全策略 = 每租户池的三维滑标位置**。

### D3. 与既有判据的统一（本 ADR 是总框架）

- **ADR 0002 D1 runtime 判据 → 泛化为三维策略**：原判据「wasm 隔离可信度 × 节点租户 × 敏感度 × 密度/成本」中，runtime（①）只是一维；而「**wasm 隔离可信度**」本身由 **②（执行体形态）+ ③（工具面）+ ADR 0003 D8 门禁**共同决定。即 ① 的选型依赖 ②③ 的位置。
- **ADR 0004 D6 runc 准入 → 是光谱的一个切片**：runc 准入 = 「**① 滑到 runc 端时，对 ②③ + L2/L3/L4 的约束**」。runc（① 弱隔离）要求 ② 偏 rust-wasm（小 TCB）+ ③ 偏最小面 + D8 门禁全绿 + L2/L3/L4 硬化全开，以**补偿** runc 较弱的内核隔离。即「**① 越靠 runc，②③ 须越靠安全端**」。
- **ADR 0005「转向」→ 重述为 ② 的默认位置**：rust-wasm 端是 ② 的**安全优先默认**；但 cpython-wasm 主体仍是 ② 的**合法兼容端位置**（较低安全等级，须由 ① 靠 rund 或 ③ 靠最小面补偿，且限定低敏感/受控租户）；ADR 0005 的 rust agent + python_exec 工具是 ② 的**混合中间位**（推荐平衡点）。ADR 0005 不是「禁止 cpython 主体」，而是「把默认与推荐位定在 ② 的安全端/混合位」。

### D4. 策略档位（每租户池的三维配置，便于表达与核查）

> 一个租户池的安全策略 = 三维滑标的一组位置 + ADR 0004 L2/L3/L4 硬化状态。下列**命名档位**是常用组合的便捷表达（非强制枚举，平台可按租户威胁模型自由设定三维位置）：

| 档位 | ① runtime | ② 执行体 | ③ 工具面 | 适用场景 |
|---|---|---|---|---|
| **hardened（高安全）** | `rund` | rust-wasm 或 rust+python_exec 混合 | 最小面 | 高敏感租户、跨租户共享节点、未 vetted 负载 |
| **balanced（平衡·推荐默认）** | `runc`(+L2/L3/L4 全开) 或 `runsc` | **rust agent + python_exec 工具**（混合） | 受控扩张（每工具过门禁） | 中敏感、专属节点、需动态 Python 的 agent 负载 |
| **density（密度优先）** | `runc`(+全硬化+D8 全绿) | rust-wasm | 最小面 | 低敏感、可信 vetted 负载、高密度/低成本诉求 |
| **compat（兼容优先）** | `rund`（补偿 ② 弱） | cpython-wasm 主体 | 较宽（须 rund 内核兜底） | 需任意 Python 负载、兼容优先、低密度诉求 |

**跨维补偿不变式（硬约束）**：

1. **① 越靠 runc（弱隔离），②③ 须越靠安全端 + L2/L3/L4 须全开**（ADR 0004 D6）；
2. **② 越靠 cpython（大 TCB），① 须越靠 rund（内核兜底）或 ③ 须越窄**；
3. **三维不可同时全在「弱」端**（runc + cpython 主体 + 宽工具面 = 无补偿，不可接受，准入须拒绝）；
4. 任一维向「弱」端移动，须有另一维向「安全端」移动补偿，或经显式风险接受（记录于租户策略）。

### D5. 度量、声明与可观测

- **可声明**：每维位置应能在 WorkerPool/ActorTemplate 策略字段表达（① = `runtimeClassName`，ADR 0002 F2；② = 执行体形态/资产选择；③ = 授予的 capability/工具集，ADR 0002 D3 capability 表）；落地字段形态见开放问题。
- **可核查**：准入 webhook / 巡检核查三维位置 + L2/L3/L4 硬化状态 + 跨维补偿不变式（D4），拒绝非法组合（如不变式 3）。这是 ADR 0004 D6 核查的**推广**（从「runc 准入四条件」到「三维位置 + 补偿」）。
- **持续状态**：安全等级是持续的（D8 门禁、Ring 2 部署、硬化核查随时间漂移），非一次性认证（同 ADR 0004 D6）。
- **审计完备性与 ③ 正交相关**：③ 工具面越宽，Ring 0/1 须覆盖的 host function 越多；**最小面使 ADR 0003 D7「Ring 0+1 = 100% 外部可见行为」的完备性论证最干净**——③ 靠安全端不仅缩小攻击面，也强化审计可论证性。

## Consequences

### 正向

- **统一框架**：把 ADR 0002–0005 的分散旋钮显式化为三维光谱，关系清晰（① 依赖 ②③；runc 准入是切片；ADR 0005 是 ② 的默认位）；
- **按租户精细取舍**：安全策略 = 每租户池三维位置，平台按威胁模型（敏感度 × 节点租户 × 兼容需求 × 密度/成本）逐池设定，避免全局二选一；
- **安全等级可声明、可核查、可持续度量**：档位 + 准入 webhook + 持续核查；
- **跨维补偿显式化**：不变式（D4）把「runc 须配 rust-wasm + 最小面」「cpython 主体须配 rund」等隐含规则写明，可机检；
- **兼容与安全不再对立**：② 的混合中间位（rust agent + python_exec 工具）让安全端主体与兼容端工具共存，是光谱模型的直接产物。

### 代价与风险

- **策略空间复杂**：三维 × 各中间位 → 组合多；档位（D4）是简化，但自由组合的核查规则须严谨；
- **跨维补偿规则须明确且可机检**：不变式（D4）的边界（「越靠」的量化）需细化，否则准入判定含糊；
- **核查自动化成本**：从 ADR 0004 D6 的四条件核查扩展到三维位置 + 补偿，准入 webhook 复杂度上升；
- **②③ 的「位置」度量不易**：② 的混合态边界（python_exec 的 capability 子集）、③ 的「多寡」（计数 vs 风险加权）需定义（开放问题）。

### 与 ADR 0002–0005 的关系

- 本 ADR 是**总框架/统一模型**，不取代各维 ADR：① 细节归 ADR 0002 D1 + ADR 0004；② 细节归 ADR 0005；③ 细节归 ADR 0003 D1；runc 准入（ADR 0004 D6）是本模型的一个切片；
- ADR 0002 D1 的 runtime 判据、ADR 0004 D6 的 runc 准入、ADR 0005 的执行体转向，均应**回指本 ADR** 作为统一光谱中的对应维度。

### 后续行动

1. **回写交叉引用**：ADR 0002 D1（runtime = 维度 ①）、ADR 0003 D1（工具面 = 维度 ③）、ADR 0004 D6（runc 准入 = 光谱切片）、ADR 0005（执行体 = 维度 ②，转向 = 默认位）各加指向本 ADR 的引用；
2. **三维位置的 API 表达**：WorkerPool/ActorTemplate 如何声明 ①②③（① 已有 `runtimeClassName`；②③ 需设计字段，与 ADR 0002 D3 capability 表 / F2-F4 fork delta 联动）；
3. **准入 webhook**：核查三维位置 + L2/L3/L4 硬化 + 跨维补偿不变式（D4），拒绝非法组合（推广 ADR 0004 D6 后续行动 5）；
4. **档位落地**：hardened/balanced/density/compat 是否固化为 API 预设或仅文档约定；
5. **sandbox 层设计回写**：S2（③ 工具面）/S4（② 执行体）/S8（① 的 L4 检测）标注对应维度，执行顺序按档位编排。

## 开放问题

- **三维的量化安全等级评分**：是否给每维打分（如 ① runc=0/runsc=1/rund=2；② cpython=0/混合=1/rust=2；③ 工具数或风险加权），复合公式（加权和 vs 最短板 vs 补偿规则）如何定；
- **档位 vs 自由组合**：hardened/balanced/density/compat 固化为 API 枚举，还是仅作便捷预设、底层允许任意三维组合；
- **② 混合态的精确边界**：`python_exec` 工具的 capability 子集（能否再调 egress/fs，还是只纯计算 + 受限 fs）、子沙箱隔离强度（ADR 0005 开放问题）；
- **③「多寡」的度量**：工具计数 vs 风险加权（一个 `egress_request` 与一个 `tool_calc` 风险不等价）；
- **跨维补偿不变的强制形态**：准入 webhook 硬拒绝非法组合 vs 告警 + 显式风险接受记录；
- **与 ADR 0002 D1 判据字段的统一**：三维位置如何与既有 WorkerSelector/runtimeClassName/capability 表协调，避免策略字段碎片化。
