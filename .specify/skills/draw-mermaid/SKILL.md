---
name: draw-mermaid
description: |
  Draw system architecture diagrams with Mermaid, render to SVG/PNG via the mermaid.ink render server or local mermaid-cli, and output as HTML with rendered images.
  Use standard UML semantics (Sequence, Class, State, ER, Flowchart-based Component/Deployment/Package) to describe system architecture — a Mermaid counterpart for every PlantUML diagram type that has a native match.
  Also supports native specialty diagrams: Gantt (甘特图), MindMap (思维导图, also covers PlantUML WBS 工作分解结构), ER 实体关系图 (erDiagram, crow's foot), C4 架构图 (C4Context/C4Container/C4Component/C4Dynamic/C4Deployment), plus flowchart-based approximations for Use Case / JSON / YAML / Salt 线框图.
  Use when the user mentions "mermaid", "Mermaid图", "mermaid diagram", "架构图", "architecture diagram", "UML图", "系统架构图", "画架构", "设计图", "组件图", "部署图", "时序图", "类图", "包图", "系统设计",
  "流程图", "状态图", "活动图", "用例图", "状态机图", "模块图", "交互图",
  "sequence diagram", "class diagram", "component diagram", "deployment diagram",
  "activity diagram", "state diagram", "use case diagram", "package diagram",
  "工作分解结构", "WBS", "甘特图", "gantt", "项目计划图", "进度图", "思维导图", "mindmap", "脑图",
  "ER图", "实体关系图", "数据库设计", "数据建模", "表结构", "ERD", "entity relationship", "crow's foot",
  "C4", "C4图", "C4模型", "context diagram", "container diagram",
  "UI原型", "线框图", "wireframe", "界面原型", "界面草图",
  "复刻图", "图片重绘", "图片转图", "replicate diagram", "redraw", "image to diagram"
skill_id: "<SKILL:.specify/skills/draw-mermaid/SKILL.md>"
---

# 架构图绘制技能（Mermaid）

使用 Mermaid 语法和标准 UML 语义绘制系统架构图，通过 mermaid.ink 渲染服务器或本地 mermaid-cli 渲染为 SVG/PNG，并输出为包含渲染图表和说明文字的完整 HTML 文档。

## 核心原则

- **UML 语义，而非随意方框**：UML 类图表必须遵循标准 UML 图表类型，使用正确的 UML 元素和关系（Mermaid 的 classDiagram / sequenceDiagram / stateDiagram-v2 / erDiagram / flowchart 各司其职）
- **架构优先的叙事**：图和文字互补——文字解释*为什么*，图展示*什么*
- **统一样式**：用 `%%{init: {themeVariables}}%%` / `classDef` 保持统一样式，UML 图每张核心元素 ≤7 个（硬上限 ≤15）
- **专项图表遵循其原生语义**：Gantt / MindMap（含 WBS）/ ER / C4 使用各自原生语法与原生配色，不套用 flowchart 的通用样式规则；Use Case / JSON / YAML / Salt 在 Mermaid 无原生图表类型，用 flowchart 语义化近似并明示近似关系

### 方法论总纲（贯穿全流程，先「对」与「达意」再「好看」）

下述四支柱是本技能所有优化手段的固化总纲，**单一事实来源为 [guide/diagram-principles.md](references/guide/diagram-principles.md)**（图表类型无关，适用于任意图；大图专项另见 [guide/large-diagram-playbook.md](references/guide/large-diagram-playbook.md)）。工作流各步都服从它：

1. **上下文驱动**：UML 脱离程序上下文无意义——先吃透文档/代码/描述、产出带出处的上下文摘要，保证程序整体正确、不臆造（principles §4.1）。
2. **减法与拆分**：信息量大时优先整洁美观而非面面俱到，每图突出**一个核心点**；单图表达不下则按架构接缝**拆为图集**（概览图 + 下钻子图，图间层次与交叉引用，每图自足，图集共享稳定词汇）（principles §4.2/§4.3）。
3. **UML 语义 + 视觉语义**：先选对图类型/元素种类/关系/构造型/接口（§1）；再按人类视角规划视觉语义——角色即位置、一对多用「单代表+多重性」、关联即同色、分组即框选（§2）。
4. **文字修饰 + 收尾美化**：元素上只留简洁标题、详细说明外置到布局安全的注释（flowchart 的 `:::注释节点`/link 注释、sequence 的 note、class 的 note）、字号层级跨图统一（§3）；最后做对齐/着色/线条与大图专项美化（playbook）。

## PlantUML ↔ Mermaid 图表类型对照

本技能以**复刻 PlantUML 图表能力**为目标：PlantUML 有的图表类型，Mermaid 有的类型直接对应（✓），语义可映射的用 Mermaid 最接近的原生类型承接（≈），Mermaid 无任何匹配类型的才标记「不匹配」（✗）并在正文说明替代建议。

| PlantUML 类型 | Mermaid 对应 | 匹配度 | 说明 |
|---------------|-------------|--------|------|
| Class 类图（`@startuml`） | `classDiagram` | ✓ | 类、属性、方法、6 种关系（`<|--` `*--` `o--` `-->` `..>` `..|>`）、多重性 |
| Sequence 时序图 | `sequenceDiagram` | ✓ | participant/actor、消息箭头、activate、note、alt/loop/par/opt 片段、autonumber |
| Activity 活动图 | `flowchart` | ✓ | 活动节点 + 菱形决策 + fork/join（subgraph 泳道）；状态驱动流程用 `stateDiagram-v2` 亦可 |
| State 状态机图 | `stateDiagram-v2` | ✓ | `[*]` 初终态、转换/守卫/动作、复合状态、fork/join、并发 `--` |
| Gantt 甘特图（`@startgantt`） | `gantt` | ✓ | section、任务/里程碑、依赖、完成度、todayMarker、axisFormat |
| MindMap 思维导图（`@startmindmap`） | `mindmap` | ✓ | 根/分支/节点形状（square/cloud/bang）、无框节点 |
| WBS 工作分解结构（`@startwbs`） | `mindmap`（树形） | ≈ | WBS 即树 → mindmap 承接；需要左右展开+算术记法的，用 `flowchart LR` 近似 |
| ER 实体关系图（entity） | `erDiagram` | ✓ | 实体、属性、PK/FK、乌鸦脚基数（`||--o{` 等） |
| C4（`C4_Context` 等宏） | `C4Context` / `C4Container` / `C4Component` / `C4Dynamic` / `C4Deployment` | ✓ | Mermaid v10.9+ 原生 C4 系列 |
| Use Case 用例图 | `flowchart`（语义近似） | ≈ | Mermaid 无原生用例图；用「角色节点 + 用例椭圆 + 系统边界 subgraph」近似 |
| Component 组件图 | `flowchart`（subgraph 边界）或 `C4Component` | ≈ | 无原生组件图；分层组件用 flowchart subgraph，组件级 C4 用 C4Component |
| Deployment 部署图 | `flowchart`（subgraph 节点）或 `C4Deployment` | ≈ | 无原生部署图；节点/容器用 subgraph 分层近似 |
| Package 包图 | `flowchart`（subgraph）或 `classDiagram`（namespace） | ≈ | 包间依赖用 flowchart；类在包内的组织用 classDiagram namespace |
| Object 对象图 | `classDiagram`（实例标注） | ≈ | 无原生对象图；类图实例 + 关系近似 |
| JSON 数据可视化（`@startjson`） | `flowchart`（结构树近似） | ≈ | Mermaid 无原生 JSON 图；对象树用 flowchart 层级 + 类型标注近似 |
| YAML 显示效果图（`@startyaml`） | `flowchart`（结构树近似） | ≈ | 同上 |
| Salt UI 线框图（`@startsalt`） | `flowchart`（界面结构近似） | ≈ | 无原生线框图；控件/区块布局用 flowchart 近似 |
| Network 网络图（nwdiag） | `flowchart`（子网 subgraph 近似） | ≈ | 无原生网络图 |
| Timing 时序波形图（`@startuml timing`） | `xychart-beta`（时间线近似）或 `timeline`（仅历史时间线语义） | ✗ | Mermaid 无等价时序波形类型；`timeline` 是时间线叙事图，语义不同 |
| Composite Structure / Profile / Archimate / Ditaa / EBNF / Regex | — | ✗ | Mermaid 无匹配类型；如需请说明用 flowchart 手动建模或改用其他工具 |

> **Mermaid 独有而 PlantUML 无的类型**（按需选用）：`pie`、`quadrantChart`、`requirementDiagram`、`gitGraph`、`journey`、`timeline`、`sankey-beta`、`xychart-beta`、`packet-beta`、`kanban`。详见 [howto/01-choose-diagram-type.md](references/howto/01-choose-diagram-type.md)。

## 工作流

按以下 8 个步骤顺序执行；每步都服从上面的「方法论总纲」四支柱。每步核心说明如下，详细操作阅读对应参考文档。

### Step 1: 语义解析 + 吃透上下文（上下文驱动）

分析用户输入以理解绘制意图；通过补充推断或交互式提问（`AskUserQuestion`，最多一轮 ≤4 个问题）确认意图。**面对文档/代码等丰富上下文时，先产出一份带出处的上下文摘要**（组件、关系、核心流程、关键决策），后续绘图与自检都对着它，保证程序整体正确、不臆造。

→ [00-semantic-analysis.md](references/howto/00-semantic-analysis.md)；上下文驱动见 [diagram-principles.md §4.1](references/guide/diagram-principles.md)

### Step 2: 选图类型 + 定「单图 or 图集」（减法与拆分）

按上方 **PlantUML ↔ Mermaid 类型对照表** 从 Mermaid 支持的类型中选最合适的一或多种，每图聚焦**单一视角/一个核心点**。**信息量大或多面时做减法与拆分**：优先整洁美观而非面面俱到；单图表达不下则按架构接缝（分层/控制面数据面/静态行为/请求制品流/系统节点边界）**拆为图集**——一张概览/索引图在顶 + 下钻子图，图间体现层次与交叉引用（`▶ 见 图N`），每图自足，图集共享稳定词汇（编号/颜色/构造型跨图同义）。

→ [01-choose-diagram-type.md](references/howto/01-choose-diagram-type.md)；减法与拆分见 [diagram-principles.md §4.2/§4.3](references/guide/diagram-principles.md)

### Step 3: 选元素 + 关系（UML 语义正确）

选正确的 Mermaid 图类型与元素（classDiagram 的类/关系、sequenceDiagram 的参与者/消息、stateDiagram-v2 的状态/转换、erDiagram 的实体/基数、flowchart 的节点形状/边型），为对外契约补接口与端口语义。**元素种类本身即语义，勿一律用 flowchart 矩形。**

→ [references/howto/](references/howto/)（02–09）；UML 语义先行见 [diagram-principles.md §1](references/guide/diagram-principles.md)

### Step 4: 规划布局 + 视觉语义（人类视角）

编码前先规划空间语义：
- **视觉语义**：角色即位置（枢纽居中偏上、节点沿边/底，Hub/Edge/Entry/Sink）；一对多用**单代表元素 + 多重性标注**（`collections`/`«×N»`），不画 N 份兄弟盒；关联即同色（同子系统同色相族，`classDef`/`style`）；分组即框选（`subgraph` 具名边界、同色系分组）。
- **方向/宽高比决策**：数「最宽层宽 B」与「主流深 D」选方向（`TD` 宽浅、`LR` 深窄长链）；`C≈round(sqrt(N×1.3))` 估列数摆近正方形网格（嵌套 subgraph 内同理）；单层兄弟 ≤6，超出下沉/拆 subgraph。

→ [10-layout-planning.md](references/howto/10-layout-planning.md)、[layout.md §一/§2.1/§2.5](references/guide/layout.md)；视觉语义见 [diagram-principles.md §2](references/guide/diagram-principles.md)

### Step 5: 生成 Mermaid 代码

按所选图类型操作指南与语法编写代码：首行写图类型声明（`classDiagram` / `sequenceDiagram` / `stateDiagram-v2` / `erDiagram` / `gantt` / `mindmap` / `flowchart TD` 等），样式指令 `%%{init: {...}}%%` 置于首行之前，先声明元素再声明关系，用方向关键字与 `subgraph` 控制布局。源文件保存为 `.mmd`。

→ [11-code-generation.md](references/howto/11-code-generation.md)、[syntax-reference.md](references/guide/syntax-reference.md)

### Step 6: 文字修饰（独立一步）

单独治理图元文字：**元素上只留很简洁的标题**（先去重——已被类型/嵌套表达的删掉）；**详细清晰的说明外置到布局安全的注释**（flowchart 用 `A -->|说明| B` 边标签或 `:::note` 节点、sequence 用 `note right/left of`、class 用 `note for`）；**字号层级用 `%%{init: {themeVariables: {fontSize, ...}}}%%` 统一设定**（标题>容器>组件>note>legend>箭头>stereotype），图内与跨图集一致，**禁用零散内联字号与加粗**（字号/粗细不一的头号成因）。

→ [diagram-principles.md §3](references/guide/diagram-principles.md)、[content.md](references/guide/content.md)

### Step 7: 应用样式 + 大图专项（对齐·着色·线条）

用 `%%{init: {themeVariables}}%%` / `classDef` / `style` 应用统一主题与色彩模式，确保视觉一致。**大图（节点多/尺寸大）套用大图技术栈**：×N 语义折叠、弱化管线突出语义色、subgraph 分层控宽高比消交叉、连线治理、legend 作单一细节仓；只用 SVG 交付大图。

→ [style.md](references/guide/style.md)、[large-diagram-playbook.md](references/guide/large-diagram-playbook.md)

### Step 8: 渲染、匹配与微调

用渲染脚本渲染 SVG/PNG；读取生成图片与用户要求比对，发现差异微调代码重渲；图集则逐图检查自足性、交叉引用与跨图一致（配色/字号/编号/页脚）；最终组装为 HTML 文档输出。

→ [12-rendering-and-output.md](references/howto/12-rendering-and-output.md)

## 专项图表（非 UML）

除 8 类 UML 图（含近似）外，本技能还支持 6 种专项图表。Gantt/MindMap/ER/C4 有原生语法与原生配色；Use Case/JSON/YAML/Salt 为 flowchart 语义近似（在下方表格中标注「近似」）。当用户意图属于以下场景时，在 Step 2 直接选用对应专项图表，并阅读其操作指南：

| 专项图表 | 适用场景 | Mermaid 类型 | 操作指南 |
|---------|---------|-------------|---------|
| **甘特图 Gantt** | 项目进度、任务依赖、里程碑 | `gantt` | [14-gantt-diagram.md](references/howto/14-gantt-diagram.md) |
| **思维导图 MindMap** | 知识梳理、发散规划 | `mindmap` | [15-mindmap-diagram.md](references/howto/15-mindmap-diagram.md) |
| **WBS 工作分解结构** | 项目/交付物层级分解 | `mindmap`（树形，≈） | [13-wbs-diagram.md](references/howto/13-wbs-diagram.md) |
| **ER 实体关系图** | 数据库表结构、数据建模、表间基数 | `erDiagram` | [18-er-diagram.md](references/howto/18-er-diagram.md) |
| **C4 架构图** | 上下文/容器/组件/动态/部署视图 | `C4Context` 等五类 | [20-c4-diagram.md](references/howto/20-c4-diagram.md) |
| **Use Case / JSON / YAML / Salt** | 用例视图、数据结构展示、UI 线框 | `flowchart`（近似） | [07-usecase-diagram.md](references/howto/07-usecase-diagram.md) / [16-json-diagram.md](references/howto/16-json-diagram.md) / [17-yaml-diagram.md](references/howto/17-yaml-diagram.md) / [19-salt-diagram.md](references/howto/19-salt-diagram.md) |

> 专项图表的渲染同样走 Step 8 的渲染脚本（`render-mermaid.sh`），服务器后端（mermaid.ink）与本地后端（mermaid-cli）对全部类型一视同仁，无需 Graphviz。
>
> **WBS / 甘特图交付前必做量测自检**：这两类图的"清晰度、版面、日期定位"都不能靠肉眼判断，用 [measure-svg-layout.py](scripts/measure-svg-layout.py) 量三条判据——**正文有效字号 ≥12px**（`= font-size × 显示宽度 ÷ viewBox 宽`；放大 zoom 无效）、**长宽比 1.2~1.8:1**、**标签不越过时间轴右边界**；判断某个写法（依赖箭头、资源分配、标题字号）有没有改写排期，用 `--compare` 做 A/B 并看 `scheduleChanged`。详见 [13-wbs-diagram.md](references/howto/13-wbs-diagram.md)、[14-gantt-diagram.md](references/howto/14-gantt-diagram.md)。

## 输出要求

- 输出为单个 HTML 文档，包含渲染的图表（不嵌入原始 Mermaid 文本）
- 图表通过 [render-mermaid.sh](scripts/render-mermaid.sh) 渲染，同时产出 PNG 与 SVG（后端自动选择：mermaid.ink 服务器 > 本地 mermaid-cli）
- **默认优先选用 PNG 格式**引用/嵌入图片（最美观，且在 Preview / Markdown 预览中可直接查看）；仅当图表过宽/过大或需任意无损缩放时改用 SVG
- **嵌入 Markdown 文档时（最佳实践）**：默认看 PNG、细节不够可开 SVG 无损放大，**SVG/PNG 必须同时产出**。⚠️ Markdown 图片 `![]()` 与内联 HTML `<a>` 走**不同的路径解析管线**（有的渲染器会代理/改写 Markdown 图片 URL 却透传 HTML `href`），混用会导致两条路径不一致——故 **PNG 与 SVG 引用须用同一机制**：首选**全内联 HTML**（`<a href=x.svg target=_blank rel=noopener><img src=x.png></a>`，点图即开 SVG 新标签），渲染器会剥 HTML 时回退**全纯 Markdown**（同标签打开）。→ 见 [12-rendering-and-output.md §4.3](references/howto/12-rendering-and-output.md)
- PNG/SVG 与 HTML 保存在同一目录，HTML 通过相对路径引用图片
- Mermaid 源文件（`.mmd`）保存以供未来编辑
- 每张图至少包含标题、渲染图片和简要说明

## 参考文档

所有参考文档（操作指南、最佳实践、官方文档）的完整索引和说明，参见 [references/index.md](references/index.md)。

## Feedback

**Runtime-mode gate.** If `${SKILL_WORKDIR}/.specify/` does not exist, this skill is
running in standalone mode (a non–Spec Kit deployment, e.g. a global agent skills
directory) — skip this entire Feedback step: no engine call, no feedback entry.

At wrap-up (the same lifecycle point where this unit would prompt for a Git commit),
run this self-reflection step. It is agent self-reflection — **never** solicit feedback
content from the user.

1. **Gate on qualification & completion.** Only proceed if this run reached wrap-up and
   did substantial work. Skip entirely for trivial/no-op runs. If the run was aborted or
   failed before wrap-up, follow the *Abort / partial-run rule* below.
2. **Reflect (no user input).** Review the just-completed run against this unit's declared
   purpose/description. Produce a short prose review plus **≥1 concrete, unit-specific
   optimization point**. If the run was clean, record exactly one line:
   `No significant optimization points identified this run.`
   **Token 效率自评**(纪律定义见 `.specify/shared/guidelines/token-efficiency.md`)——同步自查三问:本次运行是否发生 (1) **原文转储**(机器管理数据文件整体注入上下文)、(2) LLM **代做确定性工作**(固定规则判断未交程序)、(3) **重复读取**同一内容?有发现 → 对应优化点条目行 MUST 内嵌字面量 `token-efficiency`(稳定标记,供 `--action list --contains token-efficiency` 检索聚合);干净运行 MUST NOT 追加空洞的 Token 观察条目。量化口径:定性描述或行/字节代理指标,精确 Token 计数不可得时 MUST **不编造**具体数值。
3. **Scope guard.** Keep strictly to *this* unit's operation. Do NOT produce a
   global/whole-project assessment — that is `/speckit.review`'s job. Every entry is
   `scope: local`.
4. **Dedup guard.** Choose a stable `run_id` for this run (e.g. the feature key + a run
   timestamp). If a parent flow already recorded feedback for this same `(unit_id, run_id)`,
   the engine will no-op — do not force a duplicate.
5. **Persist** via the engine:
   ```bash
   python3 "${SKILL_WORKDIR:-.}/.specify/scripts/python/feedback-utils.py" --action record \
     --unit-id "skill:draw-mermaid" --unit-type skill \
     --run-id "<stable-run-id>" --feature "<feature-key-if-any>" \
     --review "<review prose>" --points-file "<points file>"
   ```
6. **Consolidated submission prompt.** Read `should_prompt` from the `record` output
   (or run `--action status`). When it is `true`, surface a **single** consolidated
   notification inviting the user to submit collected feedback to the Spec Kit developers;
   on user confirmation run `--action mark-submitted`. Below threshold, do NOT prompt.
   The detailed prompt semantics (package → manual send → mark-submitted, plus the
   skip / silence options) live in the canonical protocol:
   `.specify/shared/workflow/feedback-step.md` § *Threshold prompt protocol*.

**Abort / partial-run rule.** If the run failed or was interrupted before wrap-up, either
skip recording OR record with `--partial` and a `## Review` that begins with
`**Partial run** — `. Never present a partial run as a complete review.

**Nesting rule.** When a command invokes a skill (or a skill invokes a skill), each
qualifying unit records feedback for **its own** scope only, keyed by its own
`(unit_id, run_id)`. The same unit+run MUST NOT be recorded twice.
