# 0007. MCP 接口中介的原生能力后端（S 层，VM+多用户为特例）

- Status: Proposed
- Date: 2026-09-20
- 关联: 落实 [四层信任域全景图](../concepts/trust-domain-panorama.md) §3（S 层 MCP 能力后端）——本文是其**决策版**；承接 [ADR 0002](0002-two-tier-supervisor-worker-sandbox.md)（S/Agent 两层）、[ADR 0003](0003-wasm-host-function-security.md)（D1 host function 面 / D3 capability 唯一执行点 / D5 出口代理 / D6 凭据不落 wasm / D7 三环审计）、[ADR 0004](0004-runc-worker-pod-hardening.md)（纵深栈 / blast radius）、[ADR 0005](0005-rust-agent-execution-body.md)（工具 = host function）、[ADR 0006](0006-security-spectrum-model.md)（维度③工具面 / 维度①runtime）
- 采纳来源: 用户裁定（2026-09-20）——① 打破 1 worker pod = 1 active actor、② cross-S 共享池受益最大化（须建健壮多用户 MCP 系统）、③ 零信任不留秘密、④ 输出敌意不变、⑤ 所有跨域操作有审计+白名单
- Supersedes: -
- Superseded by: -

## Context

wasm 沙箱给 agent 的是**最小受控能力面**（[ADR 0003](0003-wasm-host-function-security.md) D1，deny-by-default），但数字员工常需要 wasm 内无法/不宜提供的**原生能力**：编译器、git、原生二进制、浏览器、重型科学计算、任意 CLI 工具。如何在不破坏 wasm 最小面与 capability 模型的前提下扩展 agent 的原生能力，是本 ADR 要决策的问题。

**被否的朴素方案**：给 agent 一条「登录到独立 VM 自由操作」的通道。这把原生能力做成了一个**不可信 shell**——边界仅靠「VM 是一次性的」这一容器化论据支撑，agent 在 VM 内可执行任意操作，无显式能力边界、无强制点、与 ADR 0003 D1/D3 的最小面+capability 模型相悖。

**本 ADR 的抽象**：把原生能力后端收敛为「**以一组定义好的 MCP 接口（工具）暴露、由可信侧 ateom 中介、后端可信度可滑**」的能力后端。agent **只能调这些 MCP 工具，绝不自由登录**。「VM + 多用户登录」只是该抽象的一个**特例**（后端 = 共享 VM、工具 = OS 账号下的参数化操作）；后端实现可插拔（共享一次性 VM / 容器池 / 远端 API / per-tenant gvisor·microvm actor），从而把既有的 `python-exec-gvisor`/`python-exec-microvm` 逃生舱也统一进同一模型。

**信任域定位**（全景图）：MCP 后端在 **S 层**，但是 S 内的「**一次性 / 不可信执行机**」子层；中介（ateom）在**可信侧**。信任序：`ateom（S 可信中介）> MCP 执行单元（S 内一次性后端）> agent（wasm，Agent 层）`。

### 威胁模型

- **攻击者**：agent（Agent 层，最低信任，可被检索/网页/工具返回等内容注入操纵），目标是经 MCP 工具越权、触达其他租户、泄密、绕过出口管控或操纵 LLM。
- **攻击面**：① MCP 工具入参（跨 wasm→native 边界的 tool/args/路径/命令/URL/标识符）；② 后端输出（敌意输入，喂给 LLM → confused-deputy / 提示注入）；③ 后端执行单元逃逸；④ **cross-S 共享池的跨租户越权**（账号/资源命名空间串味）；⑤ 后端自身网络出口（绕过 wasm 侧白名单）；⑥ 共享池编排层（跨租户 blast radius 集中点）。
- **信任边界**：wasm 内 = 不可信；host function（`mcp_call`）入口 = Agent→S 信任切换点（全量校验）；ateom = 可信中介（capability 唯一执行点）；MCP 执行单元 = 不可信后端（即使被攻破也无驻留秘密）；cross-S 执行单元隔离 = 租户边界。

## Decision

### D1. 抽象：MCP 接口中介的原生能力后端（契约与实现解耦）

原生能力后端 = **以一组定义好的 MCP 工具暴露**的能力面；agent 只能经这些工具调用，**无自由登录/自由 shell**。「MCP 接口」是**契约**，后端实现是**可插拔**的：共享一次性 VM、容器池、远端 SaaS API、per-tenant gvisor/microvm actor 皆是其实现。「VM + 多用户」是契约的一个特例。工具面因此分两层（D10）。

### D2. 强制点在可信侧（ateom = MCP 传输客户端 + 中介）

- **agent = MCP 语义客户端**：只表达工具调用意图；wasm 无裸 socket，**不能直连后端**；经 host function `mcp_call(tool, args)` 穿越 Agent→S 边界。
- **ateom = MCP 传输客户端 + 可信中介**：真正说 MCP 协议、持账号/凭据映射、做 ADR 0003 D1 白名单 / D2 入参全量校验 / D3 capability 强制（fail-closed）。**强制点必须在 ateom（可信侧），绝不在后端**（否则边界由可被攻破的代码强制，违反 D3）。
- **MCP server cluster = MCP 服务端**：哑的不可信后端，在该租户隔离单元内执行原生操作。
- 本链是既有 `host_proxy.http_request → env_proxy → HostProxy`（HTTP 出口可信中介）模式的**同构推广**（原生能力可信中介），复用同一 host 斡旋不变式（无 ambient capability、一切外部行为必经 host 授权点）。

### D3. 两种共享范围 + 隔离单元强度滑标

| 共享范围 | 部署形态 | 隔离单元强度（bar） | 适用 |
|---|---|---|---|
| **within-S**（同员工多 agent 公用） | worker pod 自管本地 VM / sidecar | **会话/任务级**：OS 多用户账号 / 容器即可（同租户，无跨租户风险） | 单数字员工内部能力复用 |
| **cross-S**（多员工共享池，受益最大化） | **P 层托管共享池**：统一编排/路由 + 热容量 + 公共工具，按需为每租户拉起隔离执行单元 | **租户级（强）**：每租户执行单元**必须 microvm / gvisor**——**OS 多用户账号不足以做跨租户边界**（共享内核 = 侧信道 + 提权面） | 跨数字员工最大化基础设施利用率 |

**自洽条件 C1（cross-S 的关键约束）**：跨 S 共享的是「**基础设施 + 编排 + 热容量 + 公共工具**」，**不是执行内核**；每租户原生执行跑在**强隔离的 per-tenant 执行单元**里——「**共享池、隔离执行单元**」。复用 substrate 既有 **microvm sandbox class（kata + cloud-hypervisor）** 作为执行单元。

### D4. 编排层 P 可信、执行单元一次性不可信（自洽条件 C2）

共享池的**编排/路由层 = P 层可信**（平台运营、强硬化、自身被审计）——它是所有租户调用的必经之路，**跨租户 blast radius 集中点**，一旦被攻破 = 全租户失守；**只有一次性执行单元是不可信的**（无状态、用完即毁、被攻破即弃重建无损）。即「**可信编排 + 不可信执行**」分离。

### D5. 零信任、不留秘密（I-1）

- **工作负载身份**：复用 substrate **SPIFFE / podcert**（podcertcontroller），每 actor / 执行单元有密码学身份；ateom↔池走 **mTLS**，按身份认证授权（不靠网络位置信任）。
- **工具按秘密需求标注 (a)/(b)**：**(a) 无秘密操作**（编译/计算/公开数据抓取）直接跑——**应当最大化的类别**；**(b) 带秘密操作**由 ateom / 身份服务 **per-call mint 短时效 scoped token**，注入该次调用、用后即弃、**绝不落后端存储**（ADR 0003 D6 代取不返值的 MCP 版），或路由到可信 per-tenant 后端。
- cross-S 共享只有在「后端无驻留秘密」时成立——即使隔离有缝也无密可窃。

### D6. 输出即敌意输入（I-2）

后端返回的内容在回 agent 前一律经 ateom **净化 / 标注为不可信**，防 confused-deputy 与提示注入（agent 把后端输出喂给 LLM）。MCP 化**不消除**此风险，须独立处置。

### D7. 工具设计即安全 + 入参校验 + capability scoping

- **窄参数化工具优先**：`compile_file(lang, src)` 而非 `run_command(cmd)`——一个 `run_command` 工具与裸 shell 同样危险，工具设计决定面是否安全。
- **入参全量校验**：每个 MCP 工具的 args 跨 wasm→native 边界按 ADR 0003 D2 校验（路径规范化、checked 算术、拒逃逸/注入）。
- **per-tenant capability scoping**：每 actor 账号只授**工具子集**（D3 spawn 授子集），且只能触达自己 S 域的资源/命名空间——防 cross-S 共享池内的跨租户 confused-deputy。

### D8. 后端出口受控（I-4）

MCP 后端的网络出口须**另设白名单**，或后端**无直接出口**、其网络操作回 ateom 斡旋；**不得绕过** wasm 侧出口白名单（ADR 0003 D5）。

### D9. 跨域审计 + 白名单（裁定 ⑤）

每次 S↔MCP 调用产生一条审计事件（tool / args 截断 / target 执行单元 / allowed / status / 字节 / 耗时 + `src_domain=S`/`dst_domain=MCP`），归 ADR 0003 D7 统一审计模型（`target="audit"`）；MCP 工具白名单 per capability。policy.blocked（白名单拦截）提级 WARN。

### D10. 工具面分层（ADR 0006 维度③）

- **ateom 内纯 host function 工具**：calc / time / fs / egress——无需原生后端，直接在 ateom 内实现（ADR 0005 D2 映射表）。
- **MCP 后端原生工具**：编译 / git / browser / 任意原生——经 `mcp_call` → ateom → MCP 池实现。

维度③滑标 = 「开多少纯 host function 工具 + 开多少 MCP 后端工具」；维度①滑标 = 「MCP 执行单元隔离强度」（一次性共享 runc VM ↔ per-tenant microvm）。

## Consequences

### 正向

- 原生能力扩展**不破坏** wasm 最小面 / capability 模型——后端是工具面的「原生能力层」，经同一 host function 强制点（D2），与 ADR 0003 D1/D3 同构；
- **契约/实现解耦**：后端可滑（一次性 VM ↔ per-tenant microvm），把既有 `python-exec-gvisor`/`microvm` 逃生舱统一进同一抽象；
- **cross-S 共享池**最大化基础设施利用率/热待机/公共工具，同时以强隔离执行单元（C1）+ 编排层可信（C2）+ 零信任不留秘密（D5）+ 跨域审计（D9）保租户边界；
- 与 ADR 0003/0004/0005/0006 全自洽（见下「关系」）。

### 代价与风险

- **cross-S 共享把租户边界移进共享池**，抬高 MCP 系统健壮性要求（C1/C2）；编排层成为跨租户 blast radius 集中点，须 P 可信 + 硬化 + 审计；
- 一次性执行单元 + per-tenant microvm 有**冷启动/调度成本**（温池缓解）；
- **MCP 工具面是新攻击面**（入参校验 + 工具设计纪律）；窄参数化工具约束 agent 能力（与 ADR 0005 张力 C 同源）；
- **输出敌意 / confused-deputy 不被 MCP 化消除**（D6），须持续净化；
- 后端出口若管控不严会**绕过 wasm 白名单**（D8）。

### 与既有 ADR 的关系

- **ADR 0003**：`mcp_call` = D1 host function 面的一类；ateom 中介 = D3 唯一执行点；token 注入 = D6；MCP 审计 = D7；后端出口 = D5。
- **ADR 0004**：编排层 = L3 blast radius 集中点；执行单元隔离 = L2 遏制。
- **ADR 0005**：MCP 后端原生工具 = Rust agent 工具面的「原生能力层」。
- **ADR 0006**：维度③（工具面两层）+ 维度①（执行单元隔离强度）。
- **分解**（两仓）：substrate **F10/F11/F12**（池编排 / microvm 执行单元 / 跨域审计白名单）+ **F9**（多 actor 调度）；sandbox **S10/S11/S13**（ateom MCP 中介 / `mcp_call` + 输出净化 / MCP 审计 + 工具契约）+ **S12**（多 active actor 并发）。详见 `two-tier-fork-sandbox-split.md` §8 与 sandbox 仓 `two-tier-sandbox-layer-design.md` §6。

### 后续行动

1. **定 MCP 工具契约/schema**（单一事实源，跨仓缝）+ (a)/(b) 标注规范 + 最小工具集；
2. **多 active actor**（F9/S12）先行——是 MCP「公用」语义的前提；
3. **sandbox**：ateom MCP 客户端中介（S10）+ host function `mcp_call`（S11）+ 输出净化（D6）+ MCP 审计（S13）；
4. **substrate**：MCP 共享池编排/路由（F10，P 可信硬化）+ per-tenant microvm 执行单元（F11）+ 跨域审计白名单（F12）+ 零信任身份 wiring（D5）；
5. **后端出口白名单**（D8/I-4）+ 执行单元一次性重建流程；
6. **cross-S 共享池** per-tenant capability scoping（D7）+ 编排层 blast-radius 硬化（D4）。

## 开放问题

- **MCP 工具最小集与 ABI/版本管理**（MCP schema vs wit 组件模型；与 ADR 0003 开放问题「host function 最小集 ABI 版本」合流）；
- **后端实现形态选型**：共享 VM 多用户 vs per-tenant microvm 池 vs 容器池——成本/隔离/冷启动权衡；within-S 与 cross-S 是否用不同形态；
- **编排层进程边界**：复用 substrate WorkerPool/actor 机制（一个 microvm-class WorkerPool，actor = MCP server 实例）vs 新控制器/服务；
- **(b) 带秘密操作的短时效 token 方案**：OAuth token exchange / SPIFFE 下游身份 / 撤销传播（与 ADR 0002 开放问题「capability 中途吊销」合流）；
- **输出净化（D6/I-2）的具体形态**：标注 / 沙箱化 / 与 LLM 推理隔离；
- **cross-S 共享池的 per-tenant 资源配额与 DoS 防护**（裁定 ② 的并发争用，R4）；
- **MCP server 二进制的仓归属**（substrate / sandbox / 新仓）；
- **后端出口与 wasm 出口白名单的统一/分立**（D8）；
- **与既有 `python-exec-gvisor`/`microvm` 逃生舱的统一**：是否都收敛为 MCP 后端特例（D1）。
