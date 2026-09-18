# 0005. 执行体形态转向：Rust agent + 标准 WASI SDK + 定制 host function（安全优先于兼容）

- Status: Proposed
- Date: 2026-09-18
- 关联: [ADR 0002](0002-two-tier-supervisor-worker-sandbox.md) D2（actor 执行体 = wasm runtime + wasm agent program，本 ADR 把「wasm agent program」具体化）；[ADR 0003](0003-wasm-host-function-security.md)（host function 安全模型——本 ADR 改变其威胁模型与最小面构成）；[ADR 0004](0004-runc-worker-pod-hardening.md) L1（本 ADR 强化 runc 准入的预防层）；**[ADR 0006](0006-security-spectrum-model.md)（本 ADR = 三维安全光谱的维度 ②「执行体形态」；下文的「转向」是 ② 的*默认/推荐位*（rust-wasm 安全端 + rust+python_exec 混合位），非禁止 cpython-wasm 主体——后者是 ② 的合法兼容端位置，须由 ① 靠 rund 或 ③ 靠最小面补偿，见 ADR 0006 D3/D4）**
- Supersedes: -
- Superseded by: -

## Context

sandbox 仓当前的执行体是**兼容优先**形态：**CPython-wasm（`python-3.12.0.wasm`）作 wasm 程序主体** + Python 程序为负载——既有 `src/program/kernel/`（持久 REPL，code-interpreter，已建集群验收的内核池 S1），也有 `src/program/agent/`（Python agent：HTTP server + LLM 循环 + Python 工具注册表 files/http/oss/calc/time，LLM 出口经 guest 内 `http.client`/raw socket）。优点是兼容任意 Python 负载；**缺点是与 ADR 0003/0004 的安全目标结构性冲突**：

| # | 兼容优先形态的安全缺陷 | 冲突的安全设计 |
|---|---|---|
| 1 | CPython-wasm 是数百万行 C 编译成的 wasm，**guest 内 TCB 巨大且非内存安全**（C 解释器自身） | ADR 0003 D4 内存安全纪律**无法覆盖 guest**（只约束 host function） |
| 2 | 解释器需要**大导入面**（componentize-py / WASI + 解释器自定义 import） | ADR 0003 D1 **最小 host function 面**（allowlist 默认拒绝） |
| 3 | Python 工具是 **guest 内动态分发**（`registry.dispatch`），不经宿主斡旋 | ADR 0003 D3 capability **唯一执行点**无从收敛；Ring 0/1 审计**覆盖不到工具语义** |
| 4 | 动态代码加载/执行是 Python 主体的本质 | ADR 0003 D1 默认拒绝「**动态代码加载**」 |
| 5 | LLM 出口经 guest 内 **raw socket**（`llm.py`） | ADR 0003 D5 **出口代理隔离**被绕过 |

**转向（2026-09-18 用户决策）**：sandbox 执行体**不再追求通用化，而以安全性为第一目标进行定制化**——形态改为 **wasm runtime（wasmtime）+ 标准 WASI SDK + Rust agent（编译为 wasm 作主体）+ 定制 host function（作为 tools 供 agent 调用）**。当前 CPython **保留为「一个 Python 动态解释器」（`python_exec` 工具）**，只是**不再作为 wasm 程序的主体**。

> 实现载体：sandbox 仓（`src/agent/` 新 Rust agent crate + `src/runtime/` host function 扩展 + 复用 `python_kernel.rs` 内核池作 python_exec 后端）；本 ADR 是其执行体形态的设计权威。

### 威胁模型的改变

ADR 0003 的攻击者是「租户提供的 wasm agent 模块（不可信代码）」。本转向后，**主体（Rust agent）是平台编写的、静态编译的、可审计的 safe Rust**——它本身**不是**不可信负载，而是平台可信执行体；不可信的是**经工具流入的数据**（LLM 输出、用户任务文本、`python_exec` 提交的 Python 代码、外部 API 响应）。这不降低 host function 面的安全要求（D1–D9 全部仍适用，且因主体可信而**更易论证**），但把「不可信代码」从「主体本身」收窄到「主体的输入与 python_exec 子沙箱」——TCB 与攻击面同步缩小。

## Decision

### D1. 执行体主体 = Rust agent → wasm（标准 WASI SDK）

- agent 循环（任务接收 → LLM 推理 → 工具分发 → 观察 → 再推理 → 记忆）以 **safe Rust** 编写，经**标准 WASI SDK**（`wasm32-wasip1`/`wasip2` + `cargo component`/componentize-rs）编译为 wasm 模块，由 ateom（wasmtime host）承载。
- 替换 `src/program/agent/`（Python agent）。主体是**定制的、静态编译的、可静态审计的小二进制**，非通用解释器。
- 与 ADR 0002 D2「执行体 = wasm runtime + wasm agent program，同级替换原生 agent CLI」一致——本 ADR 把「wasm agent program」具体化为 **Rust agent**（而非 CPython 上的 Python agent）。

### D2. 导入面 = 标准 WASI + 定制 host function（tools）

主体的全部外部能力来自两类导入，**deny-by-default**（ADR 0003 D1）：

| 类别 | 内容 | 安全属性 |
|---|---|---|
| **标准 WASI** | clocks / random / fs preopen（workspace）/ ... | wasmtime-wasi 提供；**Ring 0 档位 A（`#[instrument]` span）天然全覆盖**（ADR 0003 D7）；小、稳定、标准 |
| **定制 host function（tools）** | agent 的工具，每个 = **capability 唯一执行点**（ADR 0003 D3）+ 单独登记 + 过 D8 门禁 | 宿主斡旋（total mediation）→ Ring 0/1 覆盖工具语义；spawn 时仅授予子集 |

当前 Python 工具（`src/program/agent/tools/`）映射为 host function：

| 现 Python 工具 | → host function（tool） | 备注 |
|---|---|---|
| `http_tool` / `llm.py`（LLM + web 出口） | `egress_request` | 经 D5 出口代理 allowlist；**消除 guest raw socket**；LLM API 是 allowlist 内的特殊出口 |
| `files_tool` | `tool_fs_*`（read/write/list/stat，限定 workspace preopen 子树） | D2 ABI 路径规范化校验 |
| `oss_tool` | `tool_oss_*`（经 egress 代理或专用 host function） | 凭据经 D6 `secret_get` 代取，不落 wasm |
| `calc_tool` / `time_tool` | `tool_calc` / `tool_time`（或纯 WASI clock） | 轻量，可内联 |
| （新）Python 执行 | `python_exec` | 见 D3 |

### D3. CPython 降级为 `python_exec` 工具（保留，非主体）

- **保留**现有 `python_kernel.rs` + 内核池（S1，已建集群验收）+ `src/program/kernel/`（bootstrap.py + shims），作为 **`python_exec` host function 的后端**。
- agent 需要动态 Python 执行（数据分析、绘图、临时计算）时，**调用 `python_exec` 工具**——这是一个被 **capability 授予、被 Ring 1 审计（执行前代码存档 + 静态审计）、被沙箱化**的工具，而非主体。
- `python_exec` 提交的 Python 代码跑在**内嵌 CPython-wasm 子沙箱**（独立 wasmtime Store/instance、WASI preopen 限定、**无原生工具、无 raw socket**——其出口同样须经 host function 代理）；与 agent 主体的 capability 隔离（python_exec 子沙箱只拿到授予它的子集）。
- 即：**「Python 解释器是一个工具」，不是「Python 程序是主体」**。兼容红利（任意 Python 负载）以「受约束的工具」形态保留，而非以「不可信主体」形态存在。

### D4. 安全收益（对照 ADR 0003 / 0004）

| 收益 | 机制 | 对应安全设计 |
|---|---|---|
| **guest TCB 最小化 + 内存安全贯穿 guest** | 主体是 safe Rust 定制二进制，非数百万行 C 解释器 | ADR 0003 D4 内存安全纪律**扩展到 guest 主体**（不再只约束 host function） |
| **导入面小且标准** | 标准 WASI（Ring 0 全覆盖）+ 显式登记的 tool host function | ADR 0003 D1 最小面；无 componentize-py 大解释器导入面 |
| **工具 = host 强制点** | 工具调用经宿主斡旋，非 guest 内 Python 动态分发 | ADR 0003 D3 capability 唯一执行点 + Ring 0/1 覆盖工具语义 |
| **无 ambient 动态代码加载** | 主体静态编译；Python 执行是显式授予、沙箱化、审计的工具 | ADR 0003 D1 默认拒绝「动态代码加载」的兑现 |
| **出口统一经代理** | LLM API + web 出口走 `egress_request` → D5 代理 allowlist | ADR 0003 D5；消除 guest raw socket 绕过 |
| **强化 runc 准入（L1）** | 更小 TCB + 内存安全 guest + 工具 host 强制 → 「wasm 隔离可信」更易达成、更易论证 | ADR 0004 D6 准入条件 1（ADR 0003 D8 门禁）更易全绿 |
| **静态可审计** | Ring 1 执行前存档作用于**固定可审的主体二进制** + python_exec 提交的代码 | ADR 0003 D7 Ring 1 |

### D5. 与既有模型 / E2B 兼容的衔接

- **actor = 任务、context 粘性分派**（sandbox `wasm-sandbox-final-design`）保持；context 内的主体从「CPython kernel」变为「**Rust agent 实例**」，`python_exec` 工具复用内核池。
- **E2B code-interpreter 面**（`/execute` run_code）：保留为 **`python_exec` 工具路径**（内嵌 CPython），兼容既有 e2e（`contrib/e2b-e2e/`）；**agent 循环是 DE 式任务入口**（新通道，承载「同级替换原生 agent CLI」）。两入口共存（统一还是双轨见开放问题）。
- **外部同构不变式保持**：数据面线协议、Ateom proto、CLI、快照格式**零变化**（final-design §8.4 不变式）；本转向是**主体内部形态**的改变，对 SDK/e2bgw/atenet/atelet 透明。

### D6. 迁移路径（增量演进，不推倒重建）

| 处置 | 组件 | 说明 |
|---|---|---|
| **保留·复用** | `python_kernel.rs` + 内核池（S1）+ `src/program/kernel/` | → `python_exec` 工具后端（D3） |
| **保留·扩展** | `src/runtime/src/host_functions.rs`（`env_proxy` → `egress_request`）+ `audit.rs`（Ring 1）+ atunnel/http 数据面 | 工具 host function 注册表 + capability（ADR 0003 D3）在此扩展 |
| **新建** | `src/agent/`（Rust agent crate，标准 WASI SDK 编译）+ tool host functions + `python_exec` 桥接 + `src/runtime/src/capability.rs`（句柄模型，与 ADR 0003 D3 / sandbox 层设计 S2 共建） | 主体 + 工具面 |
| **退役** | `src/program/agent/`（Python agent）→ Rust 重写；`llm.py` guest raw socket 出口 → `egress_request` host function | 兼容优先形态的主体 |

## Consequences

### 正向

- **安全第一**：guest TCB 从「通用 CPython 解释器」收窄为「定制 safe Rust agent」，内存安全贯穿 guest，工具经 host 强制，出口统一代理——直接兑现 ADR 0003 D1/D3/D4/D5 与 ADR 0004 L1，使「wasm 隔离可信」更易达成、runc 准入更易满足；
- **定制而非通用**：与 ADR 0002 D2「执行体同级替换原生 agent CLI」精确对齐——主体就是 agent 本身（Rust），不是「跑 agent 的通用解释器」；
- **审计/能力收敛**：工具 = host function = capability 执行点 = 审计点，三位一体；Ring 0（标准 WASI span）+ Ring 1（工具/python_exec 代码存档）覆盖完整；
- **兼容红利以受约束形态保留**：`python_exec` 工具让 agent 仍能执行动态 Python（数据分析等），但在 capability + 审计 + 子沙箱约束下，而非作为不可信主体。

### 代价与风险

- **失去通用 Python 主体的兼容红利**：任意 pip 包 / 任意 Python 程序**不能作为主体**，只能经 `python_exec` 工具且受子沙箱限制（无原生工具、无 raw socket、preopen 限定）——这是「安全优先于兼容」的**自觉取舍**；
- **Rust agent 迭代慢于 Python**：agent 逻辑用 Rust 编写、编译为 wasm，迭代成本高于一改即跑的 Python；缓解：动态逻辑下放 `python_exec` 工具，LLM 驱动的工具循环主体保持稳定；
- **迁移成本**：`src/program/agent/`（Python）→ Rust 重写；Python 工具 → host function（Rust）；需建 Rust→wasm（WASI SDK）工具链与 CI；
- **E2B code-interpreter 语义须以 `python_exec` 工具维系**：既有 e2e（run_code/contexts/files）走 python_exec 路径，须保证行为同构（变量跨调用保持等 final-design 语义）；
- **`python_exec` 子沙箱仍是 CPython-wasm**：其 TCB 仍大（解释器自身），但被降为工具、受 capability + 审计 + 子沙箱隔离约束——风险被限界而非消除（开放问题：是否进一步收窄其导入面）。

### 与 ADR 0002 / 0003 / 0004 的关系

- **具体化** ADR 0002 D2「wasm agent program」= **Rust agent + 标准 WASI SDK + 定制 host function（tools）**；
- **改变** ADR 0003 的威胁模型构成（主体从「不可信租户模块」变为「平台可信 Rust agent + 不可信输入/python_exec 子沙箱」）与最小面构成（标准 WASI + tool host functions）；D1–D9 全部仍适用且**更易论证**；
- **强化** ADR 0004 L1（预防层）：更小 TCB + 内存安全 guest → D8 门禁更易全绿 → runc 准入（D6 条件 1）更易满足。

### 后续行动

1. **sandbox 仓**：建 `src/agent/` Rust agent crate（标准 WASI SDK 工具链 + CI）；定义 agent 循环（LLM 推理 / 工具分发 / 记忆）；
2. **sandbox 仓**：tool host function 注册表（`host_functions.rs` 扩展）+ `capability.rs`（句柄模型，ADR 0003 D3）+ 每工具过 D2 ABI 校验 / D8 门禁；
3. **sandbox 仓**：`python_exec` host function 桥接内核池（D3），子沙箱 capability 隔离 + Ring 1 代码存档；
4. **sandbox 仓**：`egress_request` 统一 LLM/web 出口（D5 代理），退役 `llm.py` guest raw socket；
5. **退役** `src/program/agent/`（Python agent）；保留 `src/program/kernel/` 作 python_exec 后端；
6. **E2B 兼容**：`/execute` run_code → python_exec 路径，跑既有 e2e 验证行为同构；
7. **回写** sandbox 层设计 S1/S4/S5（主体形态 + 工具面 + 内核池重定位）与 substrate ADR 0002 D2 交叉引用。

## 开放问题

- **Rust agent 的 WASI 版本与工具契约**：`wasm32-wasip1`（经典 WASI，wasmtime-wasi 成熟、Ring 0 档位 A span 现成）vs `wasip2`/Component Model（wit 契约更结构化、对齐 ADR 0003 D7 Ring 0 档位 B，但 adapter/工具链漂移为已知债务）；
- **`python_exec` 子沙箱的隔离强度**：与 agent 主体共享 wasmtime engine 还是独立 instance；capability 子集如何界定（python_exec 内能否再调 egress / fs，还是只算纯计算 + 受限 fs）；
- **E2B code-interpreter 入口与 agent 任务入口的统一/双轨**：`/execute` 同时服务「run Python（python_exec）」与「给 agent 下任务」两种语义，还是分端点；
- **agent 记忆/状态跨 context 的承载**：多轮记忆（现 `_session`）随 context（内核）还是随 actor；suspend/resume（ColdBoot）下 agent 状态的取舍（与 final-design ColdBoot 语义对齐）；
- **多语言主体**：是否只允许 Rust agent，或开放其他→wasm 语言（凡静态编译、safe、经 WASI SDK 者可准入；动态解释器一律走 `python_exec` 式工具）；
- **`python_exec` 内 CPython 导入面是否进一步收窄**：它仍是 CPython-wasm（TCB 大），作为工具被限界后，是否再裁剪其 WASI/import 面以缩小残余攻击面。
