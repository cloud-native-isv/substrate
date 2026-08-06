---
name: summarize-project
description: |
  项目总结呈现技能（由 manage-project 重构而来）。定位是项目的**呈现/输出工具**，不是管理/输入工具：只读项目现有事实源，产出一份派生的项目总结报告，不修改项目的任何管理工件。报告**文本综述与可视化图表并重**——文字总结覆盖项目概览、需求与特性叙述，图形呈现覆盖功能分解（WBS 工作分解图）、里程碑视图（按独立成图规则条件出图，否则合并进甘特图）、任务进展甘特图；四张图统一编码状态（**工作项四态色板**：已完成绿 / 进行中蓝 / **延期红** / 未开始灰，每态配冗余符号 `✓ ● ⚠ ○`）、完成度（甘特 `is N% complete` 条内填充 + WBS 节点 `NN%` 第三行）、日期（yyyy-mm-dd；里程碑菱形自带日期与已达成/待达成/逾期符号 `✓ ◇ ⚠`）与**负责人**——人员维度在 WBS 节点、甘特条形、里程碑标签及报告 `### 人员与分工` 小节均有强制承载位，缺失一律显式声明、绝不静默丢弃。报告按五个呈现层面分解，每个层面回答外部读者的一个问题（项目目标是什么、要交付哪些能力、包含哪些任务、里程碑是什么/完成了哪些、每个任务什么状态与整体进度安排），并对应 references/ 下一份层参考文档。**进度是贯穿五个章节的一等呈现维度**，不只出现在《任务进展》：`## 项目概览` 末尾固定 `### 整体进度摘要`、`## 需求与特性` 的特性表带「进度%」列与开篇量化句、`## 功能分解` 固定 `### 工作项进度`、`## 项目里程碑` 出里程碑进度汇总表（达成率 / 已达成 / 逾期 / 待达成 / 最大逾期天数）与跟踪表「达成进度」「延期天数」列、`## 任务进展` 固定 `### 进度叙述`；同一实体的进度跨章节同值同源（呈现位总表、呈现模板、叙述规范、跨章节一致门禁与空值降级见 references/progress-presentation.md）。**项目关键信息用 SQLite 关系建模，校验由数据库强约束承担**：七个核心实体各一张表（`project` / `people` / `phases` / `work_items` / `milestones` / `features` / `sources`，另有依赖联结表 `work_item_deps` 等），**字段定义与约束的唯一权威是 `schema/project.sql`（DDL）**——`NOT NULL`（R 档必填）、`PRIMARY KEY` + 触发器（`*_id` 跨实体全局唯一）、`FOREIGN KEY`（`phase_id`/`owner_id`/`anchor_item_id`/依赖关系可解析）、`CHECK`（日期是零填充 `yyyy-mm-dd` 且真实日历日、状态取值域、`progress_pct` 0–100、无出处的百分比/权重一律拒收），**具备强约束的数据不用 Markdown 的模糊描述来记录**。项目管理信息通常**不在代码仓库里**（一个项目常横跨多个 repo，真正的排期/负责人/里程碑活在项目管理程序里），因此本技能**默认不扫 repo**——三段式输入：① 从当前上下文与用户提供的外部材料（管理系统导出、需求/任务/进度文档、Excel/Word/PDF、看板导出）按**规范字段名**（snake_case，字段名即列名、也是实体间关联键：`project_name`/`baseline_date`/`phase_id`/`item_id`/`owner_id`/`milestone_id`/`anchor_item_id`/`depends_on`…）摄取成表单（人工可写的输入面，落位交付目录 `data/project-input.yaml`，空白模板 `templates/project-input.template.yaml`，技能**只读不改**）；② **装载进 SQLite（`scripts/project-db.py --load`，装载即校验**，约束违规即报错并给出可读原因，默认每次运行重建数据库；用户希望基于已有历史库按最新信息更新时用 `--update`，UPSERT + 变更摘要）；③ 齐备则**查询数据库**驱动后续章节与图表，R 档缺失（`project_name`、`baseline_date`，且 `work_items` 与 `milestones` 至少一组非空）或数据库拒绝任何一行则**阻断**并呈现只含真正缺失项的待填表单。**装载后数据库是查询与校验的唯一事实源**；I 档（阶段划分、依赖、角色、状态归一化、ID 生成等）由模型/装载器**推断并在 `inferred_fields` 留痕（`inferred_from` 依据非空）**、汇总进元信息推断字段清单，O 档缺失显式降级（`NULL` 语义明确：`progress_pct IS NULL` = 无可计数依据、不是 0）。字段名在 DDL、必要信息表（references/required-info.md 讲业务含义与三档必填性）、引擎输出与各文档引用**四处一致**。**git repo 取材完全 opt-in**：仅当表单 `project.repos[]` 声明了仓库、且某字段被标为「从 repo 推导」（`derive_fields`）时，才对该字段做**定向小范围查询**，默认一律不查、禁止全仓扫描；可选补充源阶梯与按语言的取材点（如 `.specify/specs/*/requirements.md`、`tasks.md`、`memory/features.md`、语言清单、git 标签）见 references/source-tiers.md，全程只读、可溯源、输入不限于代码。材料稀疏或缺失时**诚实降级**：缺什么就显式声明什么（章节声明句 / 图内 caption / 元信息「材料缺口」清单三处承载），进度百分比、里程碑日期、工期、负责人、任务状态五类信息无依据时一律走合法终态而**绝不臆造**；无里程碑材料时该章节仅保留声明、无排期材料时甘特不出图，报告随材料量伸缩、宁短勿虚。所有图表的 PlantUML 源码**不进报告正文**，只作为交付目录 `assets/<图名>.puml` 文件交付——`.puml` 文件是可编辑、可 diff、可版本管理的权威形态，`summary.md` 正文每张图只放渲染图片的相对路径引用 + 图说；渲染校验一律委托 draw-plantuml 技能完成。四项核心内容（项目概览、项目里程碑、功能分解 WBS、任务进展甘特图）默认逐层交互式确认后落盘——WBS 与甘特图须先渲染出图再确认；非交互模式（用户显式声明跳过）自动确认并在元信息标注。报告是派生产物，支持重复运行刷新。**日期与进度的计算全部下沉到脚本、且用 SQL 完成**：`scripts/progress-engine.py` 以交付目录 `data/project.db`（派生产物，默认每次重建、绝不写入目标项目管理工件）为输入，从库里取数并用 SQL 完成查询与聚合（天数差 `julianday`、阶段与项目级 `GROUP BY`、时间轴 `ORDER BY`），一次算出状态、进度百分比与算式、延期天数、阶段/项目聚合、里程碑达成与逾期、甘特 today 偏移与出图判据、覆盖闭合等式；Markdown 文档与报告**只调用脚本并引用其输出字段**，不比较日期先后、不算天数差、不重述任何算法；无计划完成日的条目引擎给 `unknown-schedule`，报告如实声明「无计划日期，无法判定延期」而**不判逾期、不上红色**。**交付物是一个自包含的交付目录**（SpecKit 项目默认 `.specify/project/summary/`，非 SpecKit 项目默认 `docs/project-summary/`）：`summary.md` 主报告 + `assets/`（每张图的 `.puml` 源与渲染出的 `.svg`/`.png`，同名配对）+ `data/`（**项目输入表单 `project-input.yaml`** + **项目数据库 `project.db`**、引擎输出、脚本校验结果等）；报告正文每张图只以**相对路径**引用渲染图片（如 `![功能分解 WBS](assets/wbs.svg)`）+ 图说，**正文不出现 PlantUML 源码块**（源码在 `assets/` 内的同名 `.puml` 文件里）；目录内相对路径引用齐全、禁止引用目录外文件与外链 URL——整个目录可整体移动、打包、对外分发；重复运行刷新整个目录。
  Use when the user mentions "项目总结", "总结项目", "项目现状", "项目报告", "项目汇报", "项目进展", "项目概览", "项目可视化", "需求特性", "功能分解", "里程碑", "进度追踪", "项目进度", "summarize project", "project summary", "project report", "project overview", "project status", "project dashboard", "project visualization", "milestone", "progress tracking", "WBS", "工作分解", "甘特图".
skill_id: "<SKILL:.specify/skills/summarize-project/SKILL.md>"
---

# 项目总结呈现技能

以**一个项目总结交付目录**（SpecKit 项目默认 `.specify/project/summary/`，非 SpecKit 项目默认 `docs/project-summary/`，用户可指定其他位置）呈现项目当前现状。目录结构固定：

```
<交付目录>/
├── summary.md   # 最终报告（主文档，正文只以相对路径引用 assets/ 内渲染图片 + 图说，不含 PlantUML 源码块）
├── assets/      # 每张图的 .puml 源码文件 + 渲染出的 .svg 与 .png（同名配对，如 wbs.puml / wbs.svg / wbs.png）
└── data/        # 项目输入表单 project-input.yaml（人工可写的输入面，技能只读）+ 派生产物：项目数据库 project.db（关系模型，查询与校验的唯一事实源）、引擎输出 engine-out.json、脚本校验结果
```

**整个目录是交付物**：目录内相对路径引用齐全、可整体移动/打包/分发。**文本综述与可视化图表并重**：文字总结回答"项目是什么、要交付什么、进展如何"，图表让外部读者一眼看清结构与进度。报告是**派生产物**：事实源永远在项目自身材料中，本技能只读取、总结、可视化，不代替项目管理工具维护任何事实。

报告按**五个呈现层面**分解，每个层面回答外部读者的一个问题，并对应 `references/` 目录下**一份层参考文档**（一层一文档；生成该章节前先读对应层文档）：

| 呈现层面 | 回答外部读者的问题 | 报告章节 | 形态 | **进度呈现位（贯穿五章节）** | 层参考文档 |
|----------|--------------------|----------|------|------------------------------|------------|
| 1. 项目概览 | 项目的目标是什么？ | `## 项目概览` 节 | Markdown 文本（背景、目标、范围——提炼自事实源，注明出处） | `### 整体进度摘要`：整体完成度句 + 一句进度判断 + 进度概览表（4 行）+ 进度条（引擎给 `progress_bar` 则必出、未给则不出） | [references/project-overview.md](references/project-overview.md) |
| 2. 需求与特性 | 项目要交付哪些能力？ | `## 需求与特性` 节 | 特性清单表格（名称、来源、状态、进度%）+ **条件出图**的 `@startmindmap` 概览图（判据见该层文档「概览图出图判据」：行数 ≥8 且材料给了分组依据；四态着色 + 状态符号 + 中性分组色 + 可选人员后缀） | 开篇量化句（特性维度完成度）+ 特性表「进度%」列 | [references/requirements-features.md](references/requirements-features.md) |
| 3. 功能分解 | 项目包含哪些任务？谁负责？ | `## 功能分解` 节（含 `### 工作项进度`、`### 人员与分工` 小节） | WBS 工作分解图：`@startwbs`（四态色 + 状态符号 + `【负责人】` 第二行 + `NN%` 第三行 + ◆Mn 锚点）+ 人员分工表与覆盖率声明 | `### 工作项进度`：阶段级聚合表 + 工作项级表（含进度% 与逾期天数）；WBS 图说结论句带完成度 | [references/work-breakdown.md](references/work-breakdown.md)、[references/people-encoding.md](references/people-encoding.md) |
| 4. 项目里程碑 | 里程碑是什么？完成了哪些里程碑？有无逾期？ | `## 项目里程碑` 节 | 跟踪表（含达成进度、延期天数、负责人列）+ 达成率声明与达成率表 + 条件出图的里程碑视图或达成进度图：仅含 `happens` 条目的 `@startgantt`，每个菱形带 `yyyy-mm-dd` + `✓/◇/⚠` + ` ▪负责人` | 里程碑进度汇总表（达成率 / 已达成 / 逾期 / 待达成 / 最大逾期天数）+ 跟踪表「达成进度」「延期天数」列 + 达成叙述带百分比 | [references/milestones.md](references/milestones.md) |
| 5. 任务进展 | 每个任务什么状态？整体进度安排？ | `## 任务进展` 节 | 甘特图：`@startgantt`，四态着色 + `is N% complete` 完成度双通道（延期条红 + `⚠`）+ 条形 ` ▪ 姓名` 负责人后缀 + 带日期的里程碑菱形 + 当前日期参照线 | `### 进度叙述`：整体完成度句 + 分阶段进度表 + 完成度分布 + 逾期条目数 | [references/task-progress.md](references/task-progress.md) |

进度列的呈现模板、字段引用契约、叙述规范、跨章节口径一致门禁与空值降级，全部以 [references/progress-presentation.md](references/progress-presentation.md) 为唯一权威出处；**其中一切数字都"取引擎输出字段"，本技能的任何 Markdown 文档与生成的报告正文都不含百分比算式、日期先后判定或天数差计算**。

**图表即文本，目录自包含**：每张图的 PlantUML 源码是**可编辑、可 diff、可随 git 演进的权威形态**，但它的载体是**文件而非报告正文**——源码只落 `assets/<图名>.puml`（随交付目录一起交付、一起版本管理）；`summary.md` 正文**不出现 PlantUML 源码块**，每张图只写渲染图片的**相对路径引用** + 图说（**固定引用 `.svg`**，如 `![功能分解 WBS](assets/wbs.svg)`；同名 `.png` 一并交付但正文不引用、不写「png 备选」字样）。每张图**必须渲染**为 `assets/` 下的 `.svg` 与 `.png`，与同名 `.puml` 三件套配对，使交付目录成为**目录自包含**交付物：读者无需 PlantUML、draw-plantuml 或本项目任何文件即可完整阅读，需要改图的人直接编辑 `assets/*.puml` 重渲。报告内的图片引用必须指向交付目录内的相对路径；**禁止引用交付目录外的文件，禁止外链图片 URL**。

图表语法、渲染与产物约定全部委托 **draw-plantuml** 技能；本技能只做信息读取、组织与可视化呈现，不含渲染脚本。

## 核心原则

- **呈现而非管理**：本技能是项目的最终呈现工具——只读事实源、产出派生报告，**绝不修改** `.specify/`、需求文档、任务清单等任何源工件。项目事实的录入与维护由项目管理框架（如 SpecKit 的 `/speckit.*` 流程）或用户自行完成。
- **关系建模 + 数据库强约束校验，不靠挖仓库、也不靠文档提醒**：项目管理信息不在代码仓库里（项目常横跨多个 repo，排期/负责人/里程碑活在项目管理程序里），因此输入是三段式——**上下文摄取**（对话内已有信息 + 用户提供的管理系统导出/文档，归集成表单）→ **装载进 SQLite**（`project-db.py --load`，**约束即校验**：非空/唯一/外键/日期/枚举/条件必填组合全部由数据库判定，违规即报错并给出可读原因）→ **阻断出表单**（R 档缺失或数据库拒绝行时，只把真正需要人补的内容做成表单交给用户）。**核心原则：具备强约束的数据不用 Markdown 的模糊描述来记录**——字段定义与约束的权威是 [`schema/project.sql`](schema/project.sql)，业务含义与三档必填性见 [references/required-info.md](references/required-info.md)。必填项刻意精简（`project_name`、`baseline_date`、`work_items`/`milestones` 至少一组非空），能由上下文推断的一律推断并在 `inferred_fields` 留痕，可选缺口显式降级。**表单是用户的，技能只读不改；数据库是派生物**（默认每次运行重建，用户要求基于历史库演进时用 `--update`）；**git repo 取材完全 opt-in**（未声明 `project.repos[]` 就不查任何仓库）。
- **报告可再生**：重复运行本技能时**刷新**报告（重新读取事实源、重生成图表与表格）。报告正文不承载人工维护的事实；用户若在报告内手工补充了 `## 附注` 节，刷新时原样保留该节，其余章节视为可再生的派生内容。
- **单一呈现口径**：先产出一棵**功能分解树**（阶段 → 任务 →（有子任务材料时的）子任务），WBS 图、甘特图、里程碑视图都由它派生——三处工作项/里程碑命名逐字一致，无孤儿条目。
- **进度贯穿五章节，计算下沉引擎**：进度是外部读者最先要看的信息，因此**五个章节各有固定的进度呈现位**（概览 `### 整体进度摘要` / 特性表「进度%」列 + 量化句 / 功能分解 `### 工作项进度` / 里程碑进度汇总表 + 逾期天数列 / 任务进展 `### 进度叙述`），任一缺失即骨架不完整。**一切进度与日期的计算在引擎、不在文档**：百分比、达成率、天数差、逾期判定、阶段聚合由 `${SKILL_HOME}/scripts/progress-engine.py` 产出结构化进度数据，报告与本技能文档只"取引擎输出的 `<字段>`"——**不写除法算式、不判日期先后、不算天数差**；同一实体的进度跨章节**同值同源同精度**，不一致时回引擎重取而非人工调平；字段为空值时按诚实降级写 `-（无可计数依据）`，**绝不写 `0%`**。唯一权威出处见 [references/progress-presentation.md](references/progress-presentation.md)。
- **可溯源，不臆造**：每个呈现条目（特性、工作项、里程碑）必须能溯源到**表单里该条目的 `source` 字段**（管理系统导出 / 用户填写 / 上下文 / 仅在 opt-in 时的 repo 定向推导，来源性质在 `sources[]` 声明）。表单里没有的内容，宁可阻断出表单提问也绝不编造；推断性内容显式标注（`inferred` + `inferred_from`）。**字段缺失时的诚实降级**（R/I/O 三档与声明位、五类禁造物的合法终态、无里程碑/无排期字段时章节与图如何退化、最小可交付报告形态）以 [references/degradation.md](references/degradation.md) 为唯一权威出处——缺失必须成为可见信息（三个声明位之一），绝不静默省略、绝不用占位数字/日期/人名填充。
- **计算下沉，文档只呈现**：**一切日期与进度的计算都在脚本里，不在 Markdown 里**——状态判定、延期判定、天数差、进度百分比与算式、阶段/项目聚合、里程碑逾期天数、甘特 `today` 偏移、覆盖闭合等式，全部由 `scripts/progress-engine.py` 一次算出（基准日经 `--baseline` 显式传入，引擎不读系统时钟）；报告与各层参考文档**只引用引擎输出字段**，不重述算法、不比较日期、不做除法。引擎的输入就是**关系模型 `data/project.db`**（列名 = DDL 列名，见 [`schema/project.sql`](schema/project.sql)），取数与聚合都用 SQL；它是**派生产物**：落在交付目录 `data/` 子目录、默认每次运行由 `project-db.py --load` 从表单重建并随交付目录整体刷新、**绝不写入目标项目管理工件**。无计划完成日时引擎给 `unknown-schedule`，报告如实声明「无计划日期，无法判定延期」——**不判逾期、不上红色**。细则见 [references/consistency-rules.md](references/consistency-rules.md) §0.1。
- **图源成文件，渲染委托**：PlantUML 源码以 `assets/<图名>.puml` 文件交付（不进报告正文），正文只引用渲染图片；WBS 走 draw-plantuml 的 `@startwbs` 能力，里程碑视图与甘特图走 `@startgantt` 能力。**通用绘图规则以 draw-plantuml 为唯一权威出处**——语法、字形与冗余编码、图例契约、版面/刻度/zoom 量测、渲染踩坑见其 `references/howto/13-wbs-diagram.md`、`references/howto/14-gantt-diagram.md`、`references/howto/15-mindmap-diagram.md`、`references/guide/style.md`、`references/howto/12-rendering-and-output.md`；**本报告的交付契约**（渲染成功标准、失败修正轮次、"失败必须可见"的告示规则）见 [references/reporting-playbook.md](references/reporting-playbook.md)「跨层图表呈现公约 · 渲染契约」。
- **跨图统一编码**：所有图表遵守 playbook「跨层图表呈现公约」——统一**四态色板**（completed / in-progress / delayed / not-started，同一状态跨图同色、每态配冗余符号 `✓ ● ⚠ ○`、图例与图内声明同字面量；工作项用浅色、里程碑菱形用饱和色）、全报告日期一律 **yyyy-mm-dd**、里程碑统一 Mn 短编号且每个菱形标签自带 `yyyy-mm-dd` + 三态符号（`✓` 已达成 / `◇` 待达成 / `⚠` 逾期或风险）、负责人标注跨图逐字一致（WBS `【…】` = 甘特条形后缀 ` ▪ 姓名` = 里程碑 ` ▪姓名`）、图例只列图内实际出现的编码（零实例不列）。本报告采用的字面量与成品示例见 [references/people-encoding.md](references/people-encoding.md)；字形选择、图例契约等通用规则见 draw-plantuml `references/guide/style.md`。**状态/完成度/延期天数一律取进度引擎输出字段**（`status` / `progress_pct` / `delay_days`，字段清单见 people-encoding 第 1.0 节）——呈现层与本技能文档都**不得**出现日期先后比较、天数差或百分比算式。
- **人员是一等呈现维度，缺失也要说话**：采集到的负责人必须在图与正文中有落点——WBS 节点第二行、甘特条形后缀、里程碑标签后缀，以及 `## 功能分解` 内固定的 `### 人员与分工` 小节（首句为人员维度覆盖率声明）。个别条目缺失写 `未记录`，全项目无人员数据则显式声明"材料未记录任何负责人信息"并列出已检索的来源；**任何情况下都不允许把人员数据静默丢弃**。甘特不使用 `on {资源}`（资源语法不会把人名画在条形上、底部资源盒又吃版面；实测见 draw-plantuml `references/howto/14-gantt-diagram.md`，本报告的处置见 people-encoding.md 第 3 节）。
- **目录自包含交付**：交付物面向外部读者分发，是**一个自包含的交付目录**（`summary.md` + `assets/` + `data/`），不依赖目录之外的任何依赖——每张图的渲染产物（`.svg`/`.png`）与 `.puml` 源文件一并落在 `assets/`（同名配对），报告正文以交付目录内的**相对路径**引用图片、不内嵌源码；**禁止引用交付目录外的文件**（含目标项目内其他路径与绝对路径），**禁止外链图片 URL**，不要求读者端安装任何渲染工具；目录整体可移动、打包、分发而不失效。报告内注明的事实源路径仅作出处标注，不构成阅读依赖。重复运行时**刷新整个目录**（`summary.md`、`assets/`、`data/` 全量重建；`## 附注` 节仍保留）。
- **确认后落盘**：四项项目管理核心内容——项目概览（背景介绍）、项目里程碑、功能分解（WBS）、任务进展（甘特图）——默认逐层**交互式确认**，用户确认通过的内容才允许写入报告；WBS 与甘特图必须先经 draw-plantuml **渲染出图**、连图带源码一并呈现确认，不得拿未渲染源码要求确认。非交互运行（用户显式声明跳过确认）自动通过全部门禁，并在 `## 元信息` 标注「未经交互确认」。
- **优化技能能力，而非某个项目上的表现（可移植性）**：本技能的规则必须**项目无关**——任何具体数字、路径、文件名、语言特性、模块名、特性 ID、人名都是**示例**，随项目而变、须在目标项目上实测；阈值只存在于脚本常量里，文档只引用引擎字段。任何判定都要有**确定性依据**（DB 约束 / SQL 查询 / 脚本输出 / 材料明写），**不得**依赖「在某个项目上观察到的经验值」。在新项目上发现规则与该项目材料形态不匹配时，**如实记入 `## 元信息` 的材料缺口**，而不是硬套规则。唯一权威出处见 [references/portability.md](references/portability.md)。
- **标题简洁、图说详实**：图题与图内节点/条目标题从简（上限见 playbook 公约），业务语言、无内部黑话；每张图后必配"图说"三要素——是什么、怎么读（颜色/符号编码）、一句结论——外部读者不读代码也能看懂。

## 项目信息输入模型（上下文摄取 → 装载进 SQLite（约束即校验）→ 查询驱动 / 缺则阻断出表单补填）

项目管理信息**不在代码仓库里**：一个项目常横跨多个 repo，而排期、负责人、里程碑、验收节点通常活在项目管理程序（PMO 系统、看板、周报、路线图）里。因此本技能的输入模型是**上下文摄取 + 关系建模 + 数据库强约束校验**，**默认不扫任何 repo**（从 repo 挖信息既低效又挖不到项目管理事实）。

**项目关键信息用 SQLite 关系建模**：七个核心实体各一张表——`project`（单行）、`people`（人员）、`phases`（阶段）、`work_items`（任务）、`milestones`（里程碑）、`features`（特性）、`sources`（来源声明），另有 `work_item_deps`（依赖，M:N 联结表）、`source_covers`、`inferred_fields`（推断留痕）、`repos` / `repo_derive_fields`、`coverage`、`git_window`、`status_map`、`entity_ids`（全局 ID 命名空间）等辅助表。**字段定义与约束的唯一权威是 [`schema/project.sql`](schema/project.sql)（DDL）**；每个字段的**业务含义、三档必填性 R/I/O、呈现用途与缺失后果**见 [references/required-info.md](references/required-info.md)。

**核心原则：具备强约束的数据不用 Markdown 的模糊描述来记录。** 因此下列校验**全部由数据库承担**（装载即校验，违规即报错并给出可读原因），Markdown 不再重述：

| 业务规则 | 承担它的数据库约束 |
|----------|--------------------|
| R 档必填（项目名、基准日、各实体名称） | `NOT NULL` |
| `*_id` 跨实体全局唯一 + ID 字面量合法 | `entity_ids` 主键 + 各表 `AFTER INSERT` 触发器 + `CHECK` |
| 外键可解析（`phase_id` / `owner_id` / `anchor_item_id` / 依赖） | `FOREIGN KEY`（**必须 `PRAGMA foreign_keys = ON`**——SQLite 默认关闭，关闭时外键只是注释，这是最常见的坑；DDL 与每个脚本连接都显式开启，`--check` 会回读该 PRAGMA） |
| 日期是零填充 `yyyy-mm-dd` 且真实日历日 | 日期列 `CHECK`（`GLOB` + `date(julianday(x)) = x`） |
| 状态与来源性质的取值域 | `status_norm` / `source_kind` 等列的 `CHECK ... IN (...)` |
| `progress_pct` 落在 0–100；无出处的百分比/权重拒收；全 0 的 `checks` 拒收 | 列级与表级 `CHECK`（组合条件） |
| 阶段次序不并列、`project` 只有一行、依赖不得自指 | `UNIQUE` / `CHECK (id = 1)` / `CHECK (item_id <> depends_on_item_id)` |
| 推断值必须有依据 | `inferred_fields.inferred_from NOT NULL`（无依据的填补即臆造，库不收） |

**跨表的业务必填组合**（`work_items` 与 `milestones` 至少一组非空）表级 `CHECK` 表达不了，由 `validate-project-input.py` 的齐备性检查 + `project-db.py --check` 的 SQL 断言承担。**O 档允许 `NULL`，且 `NULL` 语义明确**：`progress_pct IS NULL` = 「无可计数依据」（**不是 0**）、`planned_end IS NULL` = 「无计划日期，无法判定延期」（**不是按期**）。

三段式概览：

| 段 | 做什么 | 产物 |
|----|--------|------|
| **A 上下文摄取** | 从当前对话上下文中已有的项目信息 + 用户提供的外部材料（管理系统导出、需求/任务/进度文档、Excel/Word/PDF、看板导出）按**规范字段名**归集成表单（人工可写的输入面；也接受同结构 JSON 与按实体分文件的 CSV） | `<交付目录>/data/project-input.yaml` |
| **B 装载进 SQLite（约束即校验）** | `project-db.py --load` 建库建表并装载；**数据库拒绝任何违反约束的行**并给出「哪一行 / 哪个字段 / 违了哪条规则 / 怎么改」的可读原因；随后 `--check` 做完整性体检（FK / 枚举 / 日期 / 孤儿 / 唯一性 / 组级必填，全部 SQL 断言） | `<交付目录>/data/project.db` + 装载与体检结果 |
| **C 齐备则查询驱动 / 缺则阻断出表单补填** | 齐备（退出码 0）→ **查询数据库**驱动后续章节与图表；R 档缺失或数据库拒绝了行（退出码 3）→ **阻断**，向用户呈现**只含真正缺失必填项与被拒行**的待填表单 + 模板路径 + 填写说明；补填后回到 B | 用户补填后的表单 |

**规范字段命名**：snake_case ASCII 标识符，字段名即数据库列名、也是实体间关联的键——`project`(`project_name` `project_desc` `baseline_date` `project_start` `repos[]`)、`phases`(`phase_id` `phase_name` `phase_order`)、`work_items`(`item_id` `item_name` `phase_id` `owner_id` `planned_start` `planned_end` `actual_start` `actual_end` `status` `progress_pct` `depends_on` `source`)、`milestones`(`milestone_id` `milestone_name` `planned_date` `actual_date` `status` `anchor_item_id` `owner_id` `source`)、`people`(`owner_id` `owner_name` `owner_role`)、`features`(`feature_id` `feature_name` `status` `source`)、`sources`（每组信息的来源声明）。

**三档必填性（尽量推断、最小阻断）**：

- **R 必填-阻断**（只三条）：`project.project_name`、`project.baseline_date`，且 `work_items` 与 `milestones` **至少一组非空**（组内每行名称字段必填）。缺这些就阻断出表单。
- **I 可推断**：`phase_id`/`phase_name`/`phase_order`（可由工作项分组推断）、`depends_on`、`owner_role`、`status` 归一化、`item_id`/`milestone_id`/`feature_id` 生成、条目 `source`（由 `sources` 声明推断）等——**推断值必须在 `inferred_fields` 留痕（`field` + `inferred_from` 依据，依据非空由数据库保证）**，并在报告 `## 元信息` 汇总**推断字段清单**。
- **O 可选**：`owner_id`/`owner_name`、计划/实际日期（缺则引擎给 `unknown-schedule`、无法判延期）、`progress_pct`（缺则 `NULL`，不臆造）。

**数据库生命周期（两种模式）**：默认 `--load` **每次运行重建**（数据库是派生物，与"只读事实源、不管理数据"的定位一致）；**用户希望基于已有数据库按最新信息更新**时用 `--update`——读取历史库并 UPSERT（已存在的主键行按新值更新、新行插入、本次未提及的历史行原样保留），输出**变更摘要**，并在 `## 元信息` 注明本次为更新模式。

**装载、校验、查询命令**（确定性逻辑，勿凭猜测手工判断）：

```bash
python3 ${SKILL_HOME}/scripts/validate-project-input.py --blank-form > <交付目录>/data/project-input.yaml
python3 ${SKILL_HOME}/scripts/validate-project-input.py --input <交付目录>/data/project-input.yaml --json
python3 ${SKILL_HOME}/scripts/project-db.py --db <交付目录>/data/project.db \
  --load <交付目录>/data/project-input.yaml            # 装载即校验（默认重建）
python3 ${SKILL_HOME}/scripts/project-db.py --db <交付目录>/data/project.db --check
python3 ${SKILL_HOME}/scripts/validate-project-input.py --input <交付目录>/data/project-input.yaml \
  --db <交付目录>/data/project.db --form-skeleton      # 阻断时呈现待填表单
python3 ${SKILL_HOME}/scripts/project-db.py --db <交付目录>/data/project.db --update <新表单>
```

退出码语义：`0` 齐备 → 直接进入技能自身流程；`3` **阻断**（R 档缺失 / 数据库约束违规 / 体检不通过）→ 呈现待填表单、等补填后重新装载；`2` 输入错误（表单不可读 / 非法 YAML-JSON-CSV / 缺 DDL / 缺库）。

- **表单是用户的，技能只读**：技能读表单、装载表单、据数据库出报告，**不改写**用户填写的任何字段值（只读呈现定位）。空白模板在 `${SKILL_HOME}/templates/project-input.template.yaml`（每字段带档位注释），已填样例在 `${SKILL_HOME}/templates/project-input.example.yaml`。
- **装载后数据库是查询与校验的唯一事实源**：后续一切读取走 SQL（`project-db.py --query <预置查询名>` / `--sql "<只读 SQL>"` / `--export-json`），不回读表单、不在 Markdown 里重算。

### git repo 取材：完全 opt-in 的可选补充源

- **默认不查**：未在表单 `project.repos[]` 声明仓库时，**全程不读取、不扫描任何 git 仓库**。
- **声明才查、按字段授权**：声明了 `repos[]`（`repo_id` / `repo_path` / `repo_role` / `derive_fields[]`）后，**只对 `derive_fields` 列出的字段**做**定向小范围查询**（单条只读命令、限定路径），**禁止全仓扫描、目录遍历、深挖工件树**。可选补充源阶梯（规格工件 / 记忆 / 代码与构建 / 版本历史 / 外部文档）、按语言的取材点与「提交 ≠ 里程碑」红线见 [references/source-tiers.md](references/source-tiers.md)。
- **多 repo**：`repos[]` 允许多个仓库；同一 `item_id` **不得在多个 repo 中重复定义**（工作项身份由表单唯一确立，repo 只补字段值）；多仓对同一字段冲突时不合并、把冲突写进 `## 元信息`。
- **repo 不能凑必填**：repo 只影响 I / O 档字段，R 档缺失一律阻断出表单。
- **可选的仓库探测**（仅在 opt-in 时调用，用于确认声明的仓库里有哪些补充源可用）：

  ```bash
  python3 ${SKILL_HOME}/scripts/detect-project-sources.py --target <repo_path>
  ```

- **推断与降级必留痕**：本次报告哪些字段来自表单、哪些是推断（含 repo 定向推导）、哪些因缺失而降级，逐条记入 `## 元信息`（推断字段清单 + 材料缺口）。
- **只读**：对表单、外部材料与（opt-in 的）仓库一律只读取、不写入、不"整理修复"。

## 工作流

按以下 7 个步骤顺序执行。

### Step 1: 输入摄取与装载校验（A 上下文摄取 → B 装载进 SQLite（约束即校验）→ C 阻断出表单补填）

**Step A — 上下文摄取（不默认扫 repo）**：先把**当前上下文/对话里已经给出的项目信息**与**用户提供的外部材料**（项目管理系统导出、需求/任务/进度文档、Excel/Word/PDF、看板导出、会议纪要、路线图；office 文档可借助 document-utils 技能读取）按 [references/required-info.md](references/required-info.md) 的**规范字段名**归集成**项目输入表单** `<交付目录>/data/project-input.yaml`（空白模板：`python3 ${SKILL_HOME}/scripts/validate-project-input.py --blank-form`；已填样例见 `${SKILL_HOME}/templates/project-input.example.yaml`）。每组信息在 `sources` 里声明来源（管理系统导出 / 用户填写 / 上下文 / repo）。**此步不读取任何 git 仓库**——除非用户在表单 `project.repos[]` 声明了仓库并用 `derive_fields` 指定「从 repo 推导」的字段，此时才对**那些字段**做定向小范围查询（规程见「git repo 取材：完全 opt-in 的可选补充源」与 [references/source-tiers.md](references/source-tiers.md)）。

**Step B — 装载进 SQLite：约束即校验（齐备就直接往下走）**：

```bash
python3 ${SKILL_HOME}/scripts/validate-project-input.py \
  --input <交付目录>/data/project-input.yaml --json > <交付目录>/data/input-validation.json
python3 ${SKILL_HOME}/scripts/project-db.py --db <交付目录>/data/project.db \
  --load <交付目录>/data/project-input.yaml            # 默认重建；违规行即报错并给出可读原因
python3 ${SKILL_HOME}/scripts/project-db.py --db <交付目录>/data/project.db --check
```

（用户希望**基于已有历史数据库按最新信息更新**时，把 `--load` 换成 `--update`：读取历史库并 UPSERT，输出变更摘要，Step 7 在 `## 元信息` 注明本次为更新模式。）

据其输出向用户简述三件事：① 哪些字段**已获取**、来自哪里（`entity_counts` + `sources`）；② 哪些是**可推断**项及推断依据（`inferable[]`，入库后在 `inferred_fields` 表）；③ 哪些**必填缺失**（`missing_required[]`）或**被数据库拒绝**（`db_constraint_errors[]`，按性质归类为外键 / 标识 / 日期 / 结构问题——**判定来自数据库约束，不是文档提醒**）。**两个脚本退出码都为 0（`status=ready` 且体检 `ok`）→ 不打扰用户，直接进入 Step 2**。

**Step C — 阻断出表单、等用户表单补填（仅当 R 档缺失或数据库拒绝了行，退出码 3）**：**阻断**，不生成任何报告内容。向用户呈现：

```bash
python3 ${SKILL_HOME}/scripts/validate-project-input.py \
  --input <交付目录>/data/project-input.yaml --db <交付目录>/data/project.db --form-skeleton
```

即**只列真正缺失的必填字段与被数据库拒绝的行**（+ 建议补充的可选字段）的待填表单骨架，并同时给出**表单路径**（`<交付目录>/data/project-input.yaml`）、**空白模板路径**（`${SKILL_HOME}/templates/project-input.template.yaml`）与填写说明（能推断的字段留空即可，**不要**把可推断项塞给用户填；字段的取值规则不用背——违规会被数据库拦下并说明原因）。用户填好后**重新走 Step B**，直到齐备。

最后检查报告目标位置是否已有既有报告：存在 → 读取其中的 `## 附注` 节以备保留；不存在 → 直接生成。

### Step 2: 归集记录层并运行进度引擎（数据库即引擎输入）

项目信息已经在 Step 1 装载进数据库，**本步不再"挖材料"**，而是把库中数据按记录层四结构组织（结构 A 项目目标块 / B 任务记录表 / C 里程碑记录表 / D 人员映射表，见 [references/data-model.md](references/data-model.md)——四结构与 `project` / `work_items` / `milestones` / `people` 四张表一一对应），并完成两件事：① **I 档推断已在装载时留痕**——阶段划分、`depends_on`、`owner_role`、缺失 ID、条目 `source` 等由模型/装载器推断，逐条记入 `inferred_fields`（`field` + `inferred_from`，这些将汇总进 `## 元信息` 的推断字段清单）；② **基准日 D0 确认**——D0 取库中 `project.baseline_date`；用户指定"截至某日"时以其为准并在元信息注明「基准日由用户指定」。一切与 D0 的比较（里程碑逾期、逾期天数、进行中条目边界、排期判断）**由引擎用 SQL 完成**；本技能文档与报告正文都不做日期先后比较、不算天数差。层参考文档的「取材优先级」在新输入模型下用于**决定同一字段有多个来源时以哪个为准**（表单/管理系统导出 > 上下文 > opt-in repo 定向推导）。

**Step 2 收尾（强制）：运行进度计算引擎（直接读数据库）。** 引擎的输入就是 Step 1 建好的关系模型——取数与聚合都走 SQL，中间不再换一套字段名：

```bash
python3 ${SKILL_HOME}/scripts/progress-engine.py --print-schema          # 首次使用：查看字段契约（权威 = schema/project.sql）
python3 ${SKILL_HOME}/scripts/progress-engine.py \
  --db <交付目录>/data/project.db --baseline <D0> \
  --out <交付目录>/data/engine-out.json --summary
```

（`--baseline` 可省略——省略时引擎取库中的 `project.baseline_date`，输出 `baseline_source` 注明来源；引擎**永不读系统时钟**。需要一份 JSON 中间产物做比对或归档时用 `project-db.py --export-json`，字段名不变。）

- **一次运行、全报告共用**：引擎输出**条目级**（状态、进度百分比与算式、计划/实际时间、延期天数、工期、依据）、**分组聚合**（阶段/主题级进度）与**项目级汇总**（整体完成度及其分子分母、完成度分布计数、排期判断、逾期条目数与最大逾期天数、里程碑达成与逾期、today 偏移、甘特是否出图、覆盖闭合等式）。后续 Step 3~7 的每一处进度数字**只引用这一次运行的输出字段**，不再另行计算；**报告与参考文档里不做任何日期比较或算式**（见 [references/consistency-rules.md](references/consistency-rules.md) §0.1）；引擎调用命令与字段 → 呈现位对照在 Step 7 写入 `## 元信息 · 进度数据来源`。
- **派生产物纪律**：数据库（`project.db`）与引擎输出（`engine-out.json`）都是**派生产物**，默认每次运行从事实源**重建**并随交付目录整体刷新（`--update` 模式下基于历史库更新并在元信息注明），**绝不写入被总结的目标项目的管理工件**（只读红线的写入白名单 W-3 仅涵盖交付目录）。
- **诚实性**：引擎给 `progress_pct` 空值 → 进度写 `-（无可计数依据）`；给 `schedule_status/status = unknown-schedule`（**无计划日期**）→ 报告显式声明「无计划日期，无法判定延期」，**不判逾期、不上红色**，也不得用 git 日期等推断基线补一个计划日。引擎的 `diagnostics.declarations` 提供可直接引用的声明句。
- **报错即停**：这类问题多数在装载时就被**数据库约束**拦下（退出码 3）；漏到引擎的（如未映射状态字面量配 `--strict`）引擎一律报错退出（码 2）——修正表单、重新装载后重跑，**不要绕过引擎手工填数**。

**人员维度由 `people` 表 + `owner_id` 外键承载，是必做项、不是可选项**（规则见 [references/data-model.md](references/data-model.md) 结构 D 与 [references/people-encoding.md](references/people-encoding.md)）。取值优先级：① 表单 / 管理系统导出里的负责人字段（`people` 表 + 条目 `owner_id`）；② 上下文与用户提供的外部文档（会议纪要、看板导出、路线图的指派人）——归集进表单同样的字段；③ **仅当 repo opt-in 且该仓 `derive_fields` 声明了 `people.owner_name`** 时，才用 git 作者做定向推断（`git shortlog -sne --no-merges` 或 `git log --format='%an <%ae>' --no-merges -- <路径>`，值后加 `（推断）` 并登记 `inferred_from: repo:<repo_id> …`）。三者皆无 → 该条目负责人记 `未记录`（合法终态）。**人员维度覆盖率取引擎输出 `people.coverage_formula`**（分子分母由引擎给出，报告不自行做除法）；名册取 `people.roster`——两者是后续 `### 人员与分工` 小节的唯一来源。

字段缺失（O 档，库中为 `NULL`）时按「信息不足与澄清」节处理，不臆造：缺失字段一律取 [references/degradation.md](references/degradation.md) 第 3 节规定的**合法终态**（进度% `-（无可计数依据）`、里程碑日期/工期 `未排期`、负责人 `未记录`、任务状态 `未知`），绝不凭空补值；`未知` 与 `not-started` 不得混用。

**判定语义一律以 [references/consistency-rules.md](references/consistency-rules.md) 为准、判定执行一律由引擎完成**：源状态字面量 → 三态的映射语义见 §1（`Planned`→not-started、`Implemented`/`Ready for Review`→in-progress、`[X]`/`[x]` 双写、`[~]` deferred、`[P]` 非状态标记、未映射 fail-loud）——库列 `status` 存**原文字面量**（不预先翻译；归一化态另存受枚举 `CHECK` 约束的 `status_norm`）、把 `[P]` 之类非状态标记计入 `checks_excluded`，映射由引擎执行（项目自有口径用表单 `status_map` 键覆盖，入库为 `status_map` 表）；聚合与计数口径见 §4；「已交付但记录未闭合」禁止臆造百分比见 §5（引擎给空值，不给猜测值）。**未提供 `checks_*` / `progress_pct` 等可计数依据时**，进度按 RC-3 走诚实空值、只报状态计数（§4、§5.1、§7.2），**不得**用状态反推百分比。

### Step 3: 组织呈现模型

把工作项自顶向下组织为**阶段 → 任务 →（有子任务材料时的）子任务**的单一功能分解树，并确定特性清单与里程碑清单（关键评审、发布、验收节点）。这棵树与这两份清单是后续所有图表的唯一数据源：先在上下文中定稿（命名、层级、归属、锚定），再进入确认门禁。分解深度与命名规范见 [references/work-breakdown.md](references/work-breakdown.md)；里程碑识别与锚定规则见 [references/milestones.md](references/milestones.md)。

**建树时的判定规则**（见 [references/consistency-rules.md](references/consistency-rules.md)）：阶段（非叶子）状态与进度**由引擎聚合**（§2 语义："部分完成 + 部分未开始 ⇒ in-progress"，不是 todo）——树的父子关系通过 `work_items.phase_id` 外键表达，报告直接取父项的 `status` / `progress_pct` / `progress_formula`；单图 WBS 节点数口径与 ≤15 上限见 §3，**禁止靠合并真实工作项压缩节点数**（超限走图集拆分）。

**分解树覆盖完整性门禁（§11，本步必做，不留到落盘才补）**：先**穷尽候选工作来源**成候选全集 `C`——`C` = `work_items` 表 ∪ `features` 表 ∪（仅 opt-in 时）repo 定向补充出的条目，再把 `C` 的每一项**逐一归属**到树上的某个工作项或落入**残差清单 `R`**（R-未排期 / R-未归属 / R-已剔除 / R-粒度截断，有名有数）。**进度百分比的分母集合必须逐字等于分解树覆盖的那个 `C`（减去已剔除）**——不得"图按一套口径画、完成度按另一套算"。候选清点结果（`candidate_total` / `excluded` / `granularity_truncated` / `unattributed` / `source_label`）写进**表单的 `coverage` 块**（入库为 `coverage` 表），由引擎算出闭合等式与覆盖率（`coverage.closure_equation` / `closure_ok` / `coverage_formula` / `progress_denominator_set`）；Step 7 把这些字符串**照抄**进 `## 元信息 · 分解树覆盖`，并由 `check-coherence.py` 的 CG-COVERAGE 机械兜底（含 WBS 却未声明覆盖 = FAIL、等式算错 = FAIL）。

### Step 4: 逐层交互式确认（四道门禁）

四项核心内容各设一道确认门禁，按以下顺序逐层进行。每道门禁流程固定：**起草内容 → 向用户呈现 → 等待确认**——借助当前工具的交互提问能力（如 AskUserQuestion）给出「确认通过 / 提出修改 / 跳过本门禁（记为未确认）」三个选项；确认通过才进入下一道门禁，提出修改则原地修订后**再次呈现确认**（同一门禁内迭代直至通过）。**未经确认的内容不得写入最终报告。**

1. **门禁 1 — 项目概览（背景介绍）**：按 [references/project-overview.md](references/project-overview.md) 起草 `## 项目概览` 文本（背景、目标、范围，逐条注明出处），并**同时起草章节末的 `### 整体进度摘要`**（整体完成度句 + 一句进度判断 + 进度概览表 + 进度条（引擎给 `progress_bar` 则必出、未给则不出），数值逐个取引擎输出字段，模板见 [references/progress-presentation.md](references/progress-presentation.md) 3.1）→ 呈现全文（含进度摘要）→ 确认。
2. **门禁 2 — 项目里程碑**：按 [references/milestones.md](references/milestones.md) 起草里程碑跟踪材料与（条件出图的）里程碑视图 → 呈现 → 确认。**无任何里程碑材料时**（0 标签、无发布/验收/路线图、features 无带日期的发布评审事件）按 [references/degradation.md](references/degradation.md) 第 4 节走完全缺失形态：保留章节、只呈现「材料声明句 + 已检索来源」、不出跟踪表与视图（**也不出里程碑进度汇总表**——不写"达成率 0%"），且**绝不把普通提交/版本号/`Implemented` 特性升格为里程碑**；此时门禁确认的是"确实无里程碑材料"。有材料时要点：① 先按该文档「独立成图判定」决定里程碑视图是否独立出图（日期侧条件取引擎 `milestones.view.standalone_condition_a`、条目规模取 `gantt.split_recommended`；不出图则在本节写一行「合并进任务进展甘特图」指引）；② 每个里程碑带 Mn 短编号，**图元标签自带状态符号 + `yyyy-mm-dd` + ` ▪负责人`**（`[M1 需求冻结 ✓2026-03-13 ▪张三]`），锚定工作项结束点的在本视图写引擎输出的等价绝对日期（换算由引擎完成）；③ **状态、锚定日与逾期天数一律取引擎输出**（`items[].status` / `anchor_date` / `delay_days` / `evidence`）——`at-risk` 红 `⚠` 并把 `delay_days` 写进依据列（`逾期 N 天`）、`achieved` 绿 `✓`、`pending` 琥珀 `◇`、**`unknown-schedule`（无计划日期）琥珀 `◇` + 标签写 `未排期` + 依据列写「无计划日期，无法判定延期」且绝不上红**；配图例与 today 参照线（`today is <gantt.today_offset_days> days after start`）；**门禁内不做日期比较、不算天数差**（判定语义见 [references/consistency-rules.md](references/consistency-rules.md) §6，呈现映射见 [references/people-encoding.md](references/people-encoding.md) 4.2）；④ 配套**里程碑进度汇总表**（达成率、已达成 / 逾期 / 待达成、最大逾期天数，全部取引擎里程碑汇总字段，模板见 [references/progress-presentation.md](references/progress-presentation.md) 3.4）+「编号 | 里程碑 | 锚定 | 状态 | **达成进度** | **延期天数** | 负责人 | 依据」跟踪表 + 达成率声明与达成率表（形态 A，进度条格串取引擎 `progress_bar`）+ 达成叙述（总数、已达成数与达成率百分比、逾期数与最长逾期天数、待达成数、推进到几成）；里程碑视图与达成进度图（形态 B）按 [references/people-encoding.md](references/people-encoding.md) 4.4 的**单一决策阶梯**判定（独立视图 / 形态 B / 不出图三者互斥，**至多出一张**；4.3 的两级判据 + 4.4 的三步阶梯全部确定性，命中的步骤号记入 `## 元信息`）。
3. **门禁 3 — 功能分解（WBS，须渲染确认）**：按 [references/work-breakdown.md](references/work-breakdown.md) 生成 `@startwbs` 源码——按其「图形编码规范」把**状态（统一四态色板样式类 `.done/.doing/.late/.todo` + 标题行状态符号 `✓ ● ⚠ ○`）、负责人（`\n【…】` 第二行，缺失写 `【未记录】`，全项目无人员则整图不标并加 `caption` 声明）、完成度（`\nNN%` 第三行，延期条目附 `（延期 N 天）`；引擎给 `null` 则整行不写）、里程碑锚点（◆Mn 标记）**编码进节点，节点标题简洁、顶层阶段从左到右按时间顺序排列，配 `legend bottom` 图例（只列图内实际出现的编码，零实例不列）；源码写入 `assets/wbs.puml` → **委托 draw-plantuml 渲染出图** → 将渲染图片（路径）与 `.puml` 源码**一并呈现** → 确认。本门禁同时起草并确认 `### 工作项进度` 小节（阶段级聚合表 + 工作项级表，进度% 与逾期天数逐格取引擎分组/条目字段；模板见 [references/progress-presentation.md](references/progress-presentation.md) 3.3）与其后的 `### 人员与分工` 小节（覆盖率声明 + 人员表），并确认 WBS 图说结论句已带完成度与逾期条目数。渲染失败按 playbook「渲染契约」修正重试；不得拿未渲染的源码要求确认。
4. **门禁 4 — 任务进展（甘特图，须渲染确认）**：**先判甘特是否出图**——取引擎 `gantt.schedule_material.gantt_recommended`（`false` = 无计划日期，或仅有 git 提交且材料过弱；提交数与跨度天数的判定在引擎内，声明句可直接引其 `reason`）时，按 [references/degradation.md](references/degradation.md) 第 5 节**不出甘特**，`## 任务进展` 退化为「材料声明句 + 复用 WBS + 特性/任务状态表 + 整体状态叙述」，git 提交日期**不得**当作任务 `starts/ends`（仅作项目整体活跃区间推断且标 `（推断）`）；此时门禁确认的是该降级形态。有排期材料时按 [references/task-progress.md](references/task-progress.md) 生成 `@startgantt` 源码——按其「图形编码规范」以阶段分隔带分组、统一四态色板着色（延期条 `#EF9A9A/#C62828` + 显示名 `⚠`）、**负责人写入条形显示名后缀 `[任务名 ▪ 姓名] as [别名]`（不使用 `on {资源}`）**、全部日期 yyyy-mm-dd（基础语法遵循 `references/howto/14-gantt-diagram.md`），源码写入 `assets/gantt.puml` → **渲染出图** → 图与 `.puml` 源码一并呈现 → 确认。源码必须包含：
   - **进度状态语义**：completed（`is 100% complete`）/ in-progress（`is N% complete`）/ delayed（红 + `⚠` + `is N% complete`）/ not-started（不写 complete 子句）四态取引擎 `items[].status` 着色，**颜色与条内填充比例双通道**——条内百分比取 `progress_pct`（`null` 则不写 `is N% complete`）、算式串照抄 `progress_formula`；配图例（每行写清填充语义、含人员标注说明行、遵守零实例退化）；项目进行期须标出当前日期参照线，`today is <gantt.today_offset_days> days after start and is colored in #1565C0`，`title` 用引擎 `gantt.title_baseline` 串——**N 取引擎字段，不在报告里做减法、不依赖渲染环境时钟**；
   - **里程碑**：**逐条复制**门禁 2 已确认的全部里程碑（同编号、同名、同锚定、同状态色与符号，**标签自带 `yyyy-mm-dd`**）——一致性在生成时保证，而非留待落盘前自检兜底；
   - **依赖关系**：工作项间先后依赖按材料以 `-[dotted]->` 低调虚线呈现，无依据时不虚构依赖。

   本门禁同时起草并确认甘特图后的 `### 进度叙述` 小节（整体完成度句 + 分阶段进度表 + 完成度分布计数 + 逾期条目数与最长逾期天数，全部取引擎字段；模板见 [references/progress-presentation.md](references/progress-presentation.md) 3.5），并核对甘特图说结论句与该小节**同字段同值**。甘特不出图的降级形态下，`### 进度叙述` 仍保留，只报状态计数与材料声明句。

- **确认即冻结**：门禁通过后对应章节内容即冻结；后续步骤发现一致性问题时，凡改动冻结内容须回到对应门禁**重新确认**。
- **刷新运行的批量确认**：重复运行刷新报告时，与既有报告对应章节逐字一致的门禁内容可合并为一次「全部沿用」确认；有变化的章节仍逐层确认。
- **非交互模式**：用户显式声明跳过确认（或调用方以非交互方式运行）时，四道门禁自动通过，并在 `## 元信息` 标注「未经交互确认」。

### Step 5: 组装报告与剩余内容

1. **需求与特性**（无门禁层）：按 [references/requirements-features.md](references/requirements-features.md) 起草**开篇量化句**（特性总数 + 三态计数 + 特性维度完成度，取引擎字段）与特性清单表格（含「进度%」列，取引擎条目字段，无依据写 `-（无可计数依据）`）；特性数量适合图示时以 `@startmindmap` 附特性分组概览图——按该层文档「图形编码规范」用统一四态色板着色 + 叶子节点状态符号（分组节点用中性 `.group` 色）、有材料时带 ` ▪姓名` 负责人后缀、配 `legend bottom`；特性以表格为主、图为辅。
2. 按报告骨架（见 [references/reporting-playbook.md](references/reporting-playbook.md)）组装五个章节：四道门禁冻结的内容（含 `## 项目概览` 的 `### 整体进度摘要`、`## 功能分解` 内的 `### 工作项进度` 与 `### 人员与分工`、`## 项目里程碑` 的进度汇总表、`## 任务进展` 的 `### 进度叙述`）+ 需求与特性 + 保留的既有 `## 附注` 节。**五个章节的进度呈现位逐一在位**（缺一即骨架不完整，见 [references/progress-presentation.md](references/progress-presentation.md) §2）。

### Step 6: 渲染与图片落位（委托 draw-plantuml）

`assets/` 下的每个 `.puml` 源文件**必须**逐个经 draw-plantuml 渲染：WBS 与甘特图在门禁 3/4 已渲染确认、确认后源码未再改动的，直接沿用其渲染结果；其余图源（里程碑视图、特性概览图，以及门禁确认后被修订过的 WBS/甘特）重新渲染。渲染的成功标准、失败修正（≤2 轮）与渲染后量测自检，全部按 [references/reporting-playbook.md](references/reporting-playbook.md)「跨层图表呈现公约 · 渲染契约」执行（常见语法诱因、版面量测判据与布局/输出约定见 draw-plantuml 的 `references/howto/14-gantt-diagram.md`、`references/guide/style.md` 与 `references/howto/12-rendering-and-output.md`）。校验失败则修正 `.puml` 源文件后重试——若修正涉及门禁冻结内容，须回到对应门禁重新确认；2 轮修正仍失败的图**不得静默缺图**，按渲染契约在图位插入 `> ⚠ 本图渲染失败：<原因>` 告示（`.puml` 源文件仍留在 `assets/` 内）并记入 `## 元信息` 渲染状态清单。

渲染产物按「目录自包含」原则落位交付目录：

1. 每张图的 **`.puml` 源文件与渲染出的 `.svg`、`.png` 一并落在交付目录 `assets/` 子目录**（同名配对，如 `wbs.puml` / `wbs.svg` / `wbs.png`）；报告正文在该图位置只写**相对路径**图片引用（**固定** `![图题](assets/<名称>.svg)`）+ 图说，**不写 PlantUML 源码块**；`.puml` 源文件永不删除，始终是权威形态（可编辑、可 diff、随目录交付）；
2. **报告内的一切图片引用必须是交付目录内的相对路径**；**禁止**引用交付目录外的文件（含目标项目内其他路径与绝对路径），**禁止**外链图片 URL——整个交付目录移动、打包、发送给外部读者均不失效；
3. 单图过大或条目过多时按「大项目与图集拆分」节拆分为多张图（各自一份 `.puml` + 渲染产物入 `assets/`），而非退化为目录外文件引用；
4. 面向只收单文件的分发场景时，可**附加**产出一份自包含 HTML（图片 base64 内嵌，组装约定见 draw-plantuml `references/howto/12-rendering-and-output.md`）——HTML 是附加分发物，**交付目录仍是权威交付物**。

### Step 7: 一致性自检与报告落盘

执行**三图一致性自检**：WBS 叶子工作项（带时间信息的）与甘特条目一一对应、命名一致（比对标题行，负责人/◆Mn 附加标注除外）；里程碑视图（若出图）、跟踪表、甘特图、WBS ◆Mn 标记四处里程碑同编号同名同锚定同状态；四态口径与统一色板在图与叙述间一致（延期条目在 WBS、甘特、记录表、叙述四处均已标注；完成度与延期天数均为引擎字段的转录）；全报告日期均为 yyyy-mm-dd；特性清单表格与概览叙述一致。清单见 [references/reporting-playbook.md](references/reporting-playbook.md)。

同时执行**进度呈现自检**（细则见 [references/progress-presentation.md](references/progress-presentation.md) 第 8 节）：

- **五章节进度呈现位齐全**：概览 `### 整体进度摘要`、需求与特性开篇量化句 + 表「进度%」列、功能分解 `### 工作项进度`（阶段聚合段 + 工作项段）+ WBS 图说结论句带进度、里程碑进度汇总表 + 跟踪表「达成进度」「延期天数」列 + 达成叙述带百分比、任务进展 `### 进度叙述`——任一缺失即骨架不完整；
- **每个进度数字可指名到引擎字段**（`## 元信息 · 进度数据来源` 有字段 → 呈现位对照）；报告正文无百分比算式、无日期先后比较、无自算天数；
- **跨章节口径一致**：同一指标（整体完成度、完成度分布、阶段进度、里程碑达成率、逾期条目数/天数）在各章节同值、同源、同精度（图说结论句也算一处）；不一致**回引擎重取并整体替换**，禁止人工调平其中一处；
- **空值诚实降级**：引擎给空值的进度位写 `-（无可计数依据）`（不写 `0%`），且已落到三个声明位之一、`### 材料缺口` 含"进度百分比"一行；禁用写法（"约/大概/过半/九成"、裸百分比、字段名渗入正文）零出现。

同时执行**人员与图元维度自检**（细则见 [references/people-encoding.md](references/people-encoding.md) 第 8 节）：

- 负责人四处逐字一致：结构 D 规范名 = WBS `【…】` = 甘特 ` ▪ 姓名` = 里程碑 ` ▪姓名`；`### 人员与分工` 小节存在且首句为覆盖率声明；缺失均已显式声明（`未记录` / `caption`），无静默省略；甘特未使用 `on {}`；
- 每个里程碑菱形标签自带 `yyyy-mm-dd`（照抄引擎 `anchor_date`）+ `✓/◇/⚠`（字形合规按 draw-plantuml `references/guide/style.md`「状态编码：颜色 + 符号冗余」检查）；状态与逾期天数逐条等于引擎 `status`/`delay_days`，`unknown-schedule` 条目已声明「无计划日期，无法判定延期」且未上红；today 线左侧无 `◇`、右侧无 `✓`；
- WBS / 特性脑图 / 里程碑视图 / 进展甘特四图均已按统一色板着色（无灰白单色图）；
- 逐图跑过 draw-plantuml `references/guide/style.md`「图例契约」的自检脚本，无零实例行、无缺失行；人员图例行仅在图内确有人员标注时出现。

再执行**目录自包含检查**：报告内每个相对路径引用的文件都存在于交付目录内（`assets/` 图片、`data/` 数据文件），无交付目录外引用、无绝对路径图片引用、无外链图片 URL；**每张被引用的图在 `assets/` 内有同名 `.puml` 图源与渲染产物配对**（`<名称>.puml`/`.svg`/`.png` 齐全），反向也成立——`assets/` 内每个 `.puml` 都被正文引用（或带渲染契约允许的 `> ⚠` 失败告示）；正文**无残留 PlantUML 源码块**；每张图配有图说三要素。

再执行**可移植性自检**（细则见 [references/portability.md](references/portability.md) 第 5 节）：本次报告的每个数字/日期/百分比都来自引擎输出或材料明写、**未**沿用技能文档里的示例值；取材点按材料**类别判据**定位（示例路径不存在时已按类别另找或记入 `### 材料缺口`）；规则与本项目材料形态不匹配之处已逐条记入 `### 材料缺口`；全部**呈现决策**（出图与否 / 人员是否标注 / 图集是否拆分 / 里程碑图选型 / 进度条出不出）均有确定性依据并已记入 `## 元信息`，**无临场取舍**——同一输入 + 同一基准日两次运行必得同一呈现结果。

**机械门禁（落盘前必跑，人工自检不替代脚本）**：

0. **数据库已装载体检、引擎已跑且被引用**（CG-11）：`python3 ${SKILL_HOME}/scripts/project-db.py --db <交付目录>/data/project.db --check` 与 `python3 ${SKILL_HOME}/scripts/progress-engine.py --db <交付目录>/data/project.db --baseline <D0>` 退出码均为 0；报告中的每个百分比、天数与延期结论都能在引擎输出里逐字找到，**报告内零手工日期计算**；引擎运行记录已写入 `## 元信息`。
1. **报告文本门禁** `python3 ${SKILL_HOME}/scripts/check-coherence.py <报告路径>`——机械覆盖 CG-2/3/4/6/8、日期公约、目录自包含（相对路径引用的文件须存在于交付目录内；目录外引用/外链 FAIL）、裸编号，以及 **CG-COVERAGE**（含 `@startwbs` 却未声明分解树覆盖 = FAIL；覆盖闭合等式算错 = FAIL）。**FAIL=0 才允许落盘**；WARN 人工判读后把结论写入元信息（判读纪律见 [references/consistency-rules.md](references/consistency-rules.md) §8.1）。
2. **数据侧图元校验** `python3 ${SKILL_HOME}/scripts/verify-chart-data.py --svg <甘特.svg> --kind gantt --expect <expect.json>`——`expect.json` 由**引擎输出转出**（`project_start` = `gantt.project_start`、`today` = `baseline`、`bars` 取各条目 `planned_start`/`planned_end`/`progress_pct`、`milestones` 取 `anchor_date`），逐条核对甘特条形起止 ⇄ 引擎日期、里程碑菱形 ⇄ 锚定日、进行中条结束边界 ⇄ 基准日。（通用版面判据——长宽比/有效字号/标签溢出——已在 Step 6 由 draw-plantuml `measure-svg-layout.py` 过检，此处不重复。）

全部通过后写入报告（保留既有 `## 附注` 节），并刷新元信息：生成日期与基准日（yyyy-mm-dd）、**项目数据库记录**（数据库路径 `data/project.db` + schema 版本 + 装载模式（`--load` 重建 / `--update` 基于历史库更新）+ 装载与完整性体检结果，并声明它是交付目录 `data/` 内的派生产物、未写入目标项目管理工件）、**进度引擎运行记录**（命令 + `--db` 路径 + `--baseline`/`baseline_source`）、**输入信息源清单**（`sources` 声明：管理系统导出 / 用户填写 / 上下文 / opt-in repo，含各 repo 的 `repo_id` 与被授权的 `derive_fields`）、**齐备性校验记录**（`validate-project-input.py` 命令 + `status=ready`）、**推断字段清单**（照抄引擎 `inferred_fields`，逐条 `<字段> ← <inferred_from 依据>`）、估计假设逐条显式标注、**人员维度覆盖率与人员数据缺口**（取引擎 `people.coverage_formula`）、**分解树覆盖（候选全集 C、残差清单 R、覆盖闭合等式，见 [references/consistency-rules.md](references/consistency-rules.md) §11）**、**里程碑视图独立成图判定结论**、渲染状态清单（逐图 ok/failed）、图例比对结果、**四道门禁确认状态**（全部确认 / 部分跳过 / 未经交互确认）、固定的 **`### 进度数据来源` 小节**（基准日、引擎调用命令、分母口径、未量化条目数、字段 → 呈现位对照，结构见 [references/progress-presentation.md](references/progress-presentation.md) 第 7 节）、以及固定的 **`### 材料缺口` 小节**（缺什么 / 已检索来源 / 报告影响，结构见 [references/degradation.md](references/degradation.md) 第 6 节；字段齐备时写单行 `无`）。

## 信息不足与澄清

缺失分两类，处置不同：

1. **R 档（必填）缺失 → 阻断，不降级**：按 Step 1 的 Step C 呈现待填表单，等用户补填后重新装载（`project_name` / `baseline_date` / `work_items`+`milestones` 皆空属齐备性阻断；关联断裂、ID 重号、日期非法等由数据库拒绝，同级阻断）。**不得**用降级或猜测把必填项糊过去。
2. **I / O 档缺失 → 诚实降级**（不是默认提问，也不是默认填默认值）：按 [references/degradation.md](references/degradation.md) 对每个呈现层面判定"完全缺失 / 部分缺失 / 可推断"三档——可推断的推断并标 `inferred` + 依据，其余取合法终态（`未记录`/`未排期`/`未知`/`-`）并在三个声明位（章节声明句 / 图内 caption / 元信息「材料缺口」）显式呈现——**缺失即可见信息，绝不静默省略、绝不臆造**。无里程碑字段时该章节仅保留声明、无排期字段时甘特不出图、报告随信息量伸缩（见 degradation.md 第 4、5、7 节）。

在此基础上，个别字段仍需落值时**按固定顺序判定，不是任选**：**先判 (b) 的条件**——仅凭合法终态/估计会**实质性误导**范围/时间/状态时，走 (b)：最多发起**一轮**澄清提问（不超过 4 个问题），随后继续执行；**(b) 条件不成立时一律走 (a)**：给出合理默认并在报告中**显式标注为估计假设**（`（推断）` + `## 元信息` 假设区登记）。猜测不得静默扭曲工作范围、日期或状态。

## 大项目与图集拆分

单图放不下时做图集拆分：一张概览图（只到阶段层）+ 每个阶段一张下钻子图，每张子图各自一份 `assets/<名称>.puml` 源文件与渲染产物、在正文各占一个图位（图片引用 + 图说），图间命名/配色/编号一致并互相交叉引用。拆分规则与阈值见 [references/reporting-playbook.md](references/reporting-playbook.md)。

## 呈现范围与受众粒度

用户可限定**呈现周期**（如本迭代/本季度）或**受众粒度**（如高管层只看阶段级）：周期受限时甘特时间轴与叙述只覆盖该范围，范围外工作省略或明显弱化；粒度受限时分解到指定深度即止，保持阶段级结构完整。详见 [references/reporting-playbook.md](references/reporting-playbook.md)。

## 参考文档

**层参考文档**（一层一文档，各含：呈现要素、取材优先级、组织/推断规则、落笔检查）：

- [references/project-overview.md](references/project-overview.md) — 项目概览层：项目的目标是什么（背景、目标、范围提炼）
- [references/requirements-features.md](references/requirements-features.md) — 需求与特性层：特性清单、状态映射、**条件出图**的概览图（两条判据）
- [references/work-breakdown.md](references/work-breakdown.md) — 功能分解层：项目包含哪些任务（分解深度、命名规范、单一数据源约定）
- [references/milestones.md](references/milestones.md) — 里程碑层：里程碑是什么、完成了哪些（识别、锚定、achieved/pending/at-risk 跟踪）
- [references/task-progress.md](references/task-progress.md) — 任务进展层：每个任务的状态与整体进度安排（状态推断、退化情形、today 锚定、估计假设、依赖）

**跨层公共约定与工具**：

- [references/required-info.md](references/required-info.md) — **业务语义层唯一权威（必要信息表）**：七个实体与全部规范字段名（snake_case，字段名即数据库列名也即关联键）、**三档必填性 R/I/O**（R 只三条：`project_name`、`baseline_date`、`work_items`/`milestones` 至少一组非空）、逐字段的业务含义/呈现用途/缺失后果、**哪条业务规则由哪种数据库约束承担**（外键可解析、`*_id` 全局唯一等一律指向 DDL，不在文档重述）、表单落位与只读纪律、数据库生命周期（默认重建 / `--update` 基于历史库更新）、opt-in repo 声明与**多 repo 归集规则**、装载/校验/查询/导出命令、字段名四处一致自证
- [references/consistency-rules.md](references/consistency-rules.md) — **判定规则单一权威 + 落盘前自洽性门禁**：状态映射全集、阶段聚合真值表（四分支）、节点计数口径、聚合与计数口径（含稀疏输入分支）、「已交付但记录未闭合」诚实处理、逾期/at-risk 判定、统计口径契约、三个 ID 命名空间、**§11 分解树覆盖完整性门禁**、Feedback 与只读红线裁决、§8 落盘门禁清单 CG-1~CG-10+CG-COVERAGE。层参考文档只讲"呈现什么"，一切"如何判定/如何自洽"以本文档为准
- [references/reporting-playbook.md](references/reporting-playbook.md) — 层参考文档索引、**跨层图表呈现公约**（统一四态色板与冗余符号、yyyy-mm-dd 日期公约、图题图说规范、渲染契约与失败可见性、人员承载公约、图例契约、读者用语纪律）、**进度呈现公约（§1.8）**、报告结构、刷新规则、图集拆分、三图一致性清单、范围/粒度控制、落盘检查单、外部文档取材
- [references/progress-presentation.md](references/progress-presentation.md) — **进度贯穿五章节的呈现骨架**：五章节进度呈现位/形态/引用字段总表、逐章节呈现模板（整体进度摘要 / 特性进度列与量化句 / 工作项与阶段聚合进度 / 里程碑进度汇总表 / 进度叙述）、进度叙述规范（百分比写法、枚举→读者用语固定映射、禁用写法清单）、跨章节进度口径一致门禁、进度数据空值时的逐章节降级（与 degradation 一致）、`## 元信息 · 进度数据来源`。**边界：一切进度与日期计算在引擎 `scripts/progress-engine.py`，本文档与报告正文只"取引擎输出字段"，零算式、零日期比较**
- [references/data-model.md](references/data-model.md) — 记录层四结构（项目目标块 / 任务记录表 / 里程碑记录表 / 人员映射表）与关系模型四张表的一一对应、报告表格取值、日期呈现与推断标注、派生映射、记录层完整性检查（**字段定义引用 DDL，本文档不另写一份 schema**）
- [references/people-encoding.md](references/people-encoding.md) — **人员承载与图元编码**：三层人员承载位与缺失显式声明、统一图元字面量、甘特人员编码（负责人进条形显示名，不用 `on {}`）、里程碑日期/三态符号、**引擎字段消费契约与四态色板映射**、里程碑视图独立成图判定、里程碑/目标进度看板（达成率表 + 达成进度图）、七个实测渲染通过的 PlantUML 示例
- [references/degradation.md](references/degradation.md) — **字段缺失时的诚实降级**：按必要信息表 **R/I/O 三档**的缺失分级与三个声明位（R 缺失→阻断出表单、I 缺失→推断并标注、O 缺失→显式声明降级）、五类禁造物（进度%/里程碑日期/工期/负责人/任务状态）的合法终态与三处一致呈现、无里程碑字段时 `## 项目里程碑` 的合法形态（不升格普通提交/版本号）、无排期字段时 `## 任务进展` 甘特出图判据、`## 元信息` 的 `### 材料缺口` 清单结构、稀疏输入的最小可交付报告形态、声明句模板库
- [`schema/project.sql`](schema/project.sql) — **字段定义与约束的唯一权威（DDL）**：七个核心实体各一张表 + `work_item_deps` / `source_covers` / `inferred_fields` / `repos` / `coverage` / `git_window` / `status_map` / `entity_ids` 等辅助表；`NOT NULL`（R 档必填）、`PRIMARY KEY` + 触发器（`*_id` 跨实体全局唯一）、`FOREIGN KEY`（关联可解析，**须 `PRAGMA foreign_keys = ON`**）、`CHECK`（日期字面量与日历日、状态取值域、0–100 百分比、无出处数字拒收、禁自依赖、单例 project）、必要索引与预置视图（`v_work_items` / `v_milestones` / `v_phase_rollup` / `v_check_sums` / `v_people_coverage` / `v_timeline` / `v_entity_counts` / `v_orphans` / `v_unknown_schedule`）
- [references/portability.md](references/portability.md) — **可移植性纪律（技能作者 + 执行器双向约束）**：规则必须项目无关、具体数字/路径/语言细节一律标注为示例；任何判定必须有确定性依据（DB 约束 / SQL 查询 / 脚本输出字段 / 材料明写），不得依赖「在某个项目上观察到的经验值」；规则与目标项目材料形态不匹配时如实记入 `### 材料缺口` 而非硬套规则；落盘设「可移植性自检」门禁
- [references/source-tiers.md](references/source-tiers.md) — **opt-in 可选补充源的定向取材规程（不是主路径）**：仅当表单 `project.repos[]` 已声明、且字段被 `derive_fields` 标注为「从 repo 推导」时才启用；给出允许的**定向查询**形态（单条只读命令、限定路径，禁止全仓扫描）、可选补充源阶梯（规格工件 / 记忆 / 代码与构建 / 版本历史 / 外部文档）与其只作**佐证**的地位、按语言分类的取材点、CLI 子命令→特性佐证、"提交≠里程碑"红线、构建/CI 证据强度分级、多 repo 归集与冲突处置
- `scripts/project-db.py`（**关系模型的读写、校验与查询落点**）：`--init` 建库建表、`--load` 装载表单（**约束即校验**，默认重建）、`--update` 基于历史库 UPSERT 更新并输出变更摘要、`--check` 完整性体检、`--query <预置查询名>` / `--sql`（只读查询）、`--export-json` 导出引擎输入、`--print-ddl` 打印字段定义权威
- `scripts/progress-engine.py`（**日期与进度计算的唯一权威出处**）：`--db` 读关系模型并**用 SQL 完成查询与聚合**（`--print-schema` 给出字段契约，权威 = `schema/project.sql`；`--baseline` 可省略、缺省取库中 `project.baseline_date`）；输出字段 `items[].status` / `progress_pct` / `progress_formula` / `schedule_status` / `delay_days` / `duration_days` / `evidence` / `owner_name` / `depends_on`、`groups[]`、`project.progress`（含分桶与备选口径）、`milestones`（计数、逾期明细、视图判定）、`gantt`（起点、`today_offset_days`、出图判据）、`timeline`、`coverage`、**`people`（名册 + 覆盖率算式）**、**`inferred_fields`（推断字段清单）**、`diagnostics.declarations`
- draw-plantuml 技能（**通用绘图规则的唯一权威出处**）：`references/howto/13-wbs-diagram.md`（WBS，含父节点四分支聚合规则）、`references/howto/14-gantt-diagram.md`（甘特图与里程碑、刻度/zoom 量测矩阵、量测自检三条几何判据、`on {}` 实测结论）、`references/howto/15-mindmap-diagram.md`（脑图）、`references/guide/style.md`（字形与冗余编码、图例契约与自检脚本）、`references/guide/large-diagram-playbook.md`（大图与图集拆分技术）、`references/howto/12-rendering-and-output.md`（渲染与输出约定）；其 `scripts/measure-svg-layout.py` 是**通用版面量测的权威引擎**（长宽比 / 有效字号 / 标签溢出 / A-B 排期比对）

### 本技能脚本清单（六支，职责互斥，不与 draw-plantuml 重复）

| 脚本 | 职责 | 何时运行 | 与 draw-plantuml 的边界 |
|------|------|----------|--------------------------|
| `scripts/project-db.py` | **关系模型的唯一读写落点（校验由数据库约束承担）**：`--init` 按 DDL 建库建表；`--load` 装载表单/JSON/CSV（**装载即校验**——非空、`*_id` 全局唯一、外键、日期、枚举、条件必填组合全由约束判定，违规即报错并给出「哪一行 / 哪个字段 / 违了哪条规则 / 怎么改」）；`--update` **基于已有历史库按最新信息更新**（UPSERT + 变更摘要）；`--check` 完整性体检（FK / 枚举 / 日期 / 孤儿 / 唯一性 / 组级必填，全 SQL 断言）；`--query` / `--sql` 只读查询；`--export-json` 导出引擎输入；`--print-ddl` 打印字段定义权威。默认库位 `data/project.db`。退出码 0 成功 / 3 约束违规 / 2 输入错误 | Step 1 的 Step B（装载 + 体检）；表单修正后重新装载；Step 7 门禁 0 复核 | 无关（不涉及绘图） |
| `scripts/validate-project-input.py` | **R 档齐备性检查 + 待填表单生成（数据库约束表达不了的那部分）**：检查跨表的业务必填组合（`project_name`、`baseline_date`、`work_items`/`milestones` 至少一组非空）与行内名称，登记 I 档可推断项与 O 档缺口，输出结构化校验报告（`status` / `missing_required[]` / `inferable[]` / `optional_gaps[]` / `db_constraint_errors[]` / `constraint_owner` / `repo_optin`）与**待填表单骨架**（只列真正缺失的必填项与被数据库拒绝的行）；`--blank-form` 给空白模板、`--print-required` 打印必要信息表（含**校验归属** db/script）、`--db` 顺带装载让约束校验、`--emit-json` 经数据库导出引擎输入。**机械约束已下沉到 DDL，本脚本不再手写**。退出码 0 齐备 / 3 阻断 / 2 输入错误 | Step 1 的 Step B、Step C | 无关（不涉及绘图） |
| `scripts/detect-project-sources.py` | **可选**的 repo 补充源探测（**仅在 opt-in 时调用**）：对表单 `project.repos[]` 声明的某个仓库，探测其中有哪些可用的补充源（`source_tiers`/`primary_tier`/`build`/`git`）并给默认交付路径（JSON 输出，确定性）。**不是主路径**——未声明 `repos[]` 时不运行 | 仅当 repo opt-in 时，在 Step 1 的 Step A 内 | 无关（不涉及绘图） |
| `scripts/progress-engine.py` | **日期与进度的唯一计算者（本技能唯一计算落点）**：日期归一化与校验（非法/歧义**报错**而非静默）、状态映射、**延期判定（含"无计划日期 → `unknown-schedule`，不判延期、不上红"）**、天数差与工期、进度% 及自带分子分母的算式串、父项/阶段与项目级聚合、里程碑达成与逾期天数、时间轴排序聚合、甘特 `today` 偏移与出图判据、覆盖闭合等式、人员覆盖率、推断字段汇总。**输入** = 交付目录 `data/` 下的**关系模型 `project.db`**（`--db`；列名 = DDL 列名，派生产物，默认每次重建，**不写入目标项目管理工件**）；**取数与聚合全部用 SQL**（天数差 `julianday`、阶段与项目级 `GROUP BY`、时间轴 `ORDER BY`；只读取数，除 `temp.` 临时表外不改任何持久化表）；**输出** = 结构化 JSON，报告只引用字段。`--baseline` 可省略（缺省取库中 `project.baseline_date`），**不读系统时钟**，同库同基准日必得同输出（同一份数据经 `--db` 与经兼容的 `--input` JSON 亦得同一份输出） | Step 2 收尾（**一次运行、全报告共用**）；数据修正后重跑；Step 7 门禁 0 复核 | 完全不涉及绘图——引擎产出"画什么数"，draw-plantuml 负责"怎么画" |
| `scripts/verify-chart-data.py` | **数据侧**图元校验：甘特条形起止 ⇄ 引擎日期、里程碑菱形 ⇄ `anchor_date`、进行中条结束边界 ⇄ 基准日（`--expect` 由**引擎输出转出**做真值，落 `data/` 子目录） | Step 6 渲染出图后、Step 7 落盘前 | **只做需项目日期做真值的检查**；长宽比/有效字号/标签溢出等通用几何**委托** draw-plantuml `measure-svg-layout.py`，本脚本不再实现 |
| `scripts/check-coherence.py` | **落盘前自洽性门禁**：CG-2/3/4/6/8、CG-COVERAGE（分解树覆盖声明 + 闭合等式）、日期公约、目录自包含（报告内相对路径引用的文件须存在于交付目录内；**每张引用图在 `assets/` 内有同名 `.puml` 与渲染产物配对**；正文残留源码块 = FAIL）、裸编号；图元/图例判据读 `assets/*.puml` 图源，数字类检查在正文（去代码块+SVG）上做，WBS 规则只作用于 `@startwbs` | Step 7 落盘前（FAIL=0 才允许写） | 只查"报告文本自洽"，不量版面几何 |

- **计算只在引擎里**：进度百分比、达成率、天数差、逾期判定、阶段聚合等一切算术与日期比较**只出现在 `progress-engine.py`**；其余脚本只做校验，本技能的**所有 Markdown 文档与生成的报告正文都不含算式与日期比较**——需要数字就引用引擎输出字段（见 [references/progress-presentation.md](references/progress-presentation.md)）。
- **六支顺序固定、判据不重叠**：`validate-project-input.py`（Step 1，检 R 档齐备性、缺则出表单）→ `project-db.py --load/--check`（Step 1，**装载即校验**，机械约束由数据库判定、不合格数据进不了模型）→ `progress-engine.py --db`（Step 2，从库里用 SQL 把全部数字算出来）→ 出图 → draw-plantuml `measure-svg-layout.py` 过**通用版面判据**（三条几何判据）→ `verify-chart-data.py` 过**数据侧判据**（图元 ⇄ 引擎日期）→ `check-coherence.py` 过**报告文本门禁**（`detect-project-sources.py` 只在 repo opt-in 时插在最前，作可选探测）。一句话分工：**齐备性检查器守必填组合、数据库守约束、引擎算数、绘图技能量版面、数据校验对日期、门禁查文本自洽**。

## Feedback

**Runtime-mode gate.** If `${SKILL_WORKDIR}/.specify/` does not exist, this skill is
running in standalone mode (a non–Spec Kit deployment, e.g. a global agent skills
directory) — skip this entire Feedback step: no engine call, no feedback entry.

At the end of a substantial run of this skill, perform an agent self-reflection step (never solicit feedback content from the user), following the canonical convention in `.specify/shared/workflow/feedback-step.md`:

1. **Gate on qualification & completion.** Only proceed if this run reached a meaningful wrap-up. Skip trivial/no-op runs; for an aborted run use the abort/partial rule below.
2. **Reflect (no user input).** Review this run against this skill's declared purpose and produce a short review plus ≥1 concrete, skill-specific optimization point. If the run was clean, use exactly: `No significant optimization points identified this run.`
3. **Scope guard.** Keep strictly to this skill's operation; do NOT produce a global/whole-project assessment (that is `/speckit.review`'s job). Entries are `scope: local`.
4. **Dedup guard.** Use a stable `run_id`; if a parent flow already recorded feedback for this same `(unit_id, run_id)`, the engine no-ops.
5. **Persist** via the engine:
   ```bash
   python3 "${SKILL_WORKDIR:-.}/.specify/scripts/python/feedback-utils.py" --action record \
     --unit-id "skill:summarize-project" --unit-type skill \
     --run-id "<stable-run-id>" --feature "<feature-key-if-any>" \
     --review "<review prose>" --points-file "<points file>"
   ```
6. **Consolidated submission prompt.** If the returned `should_prompt` is `true`, surface a single consolidated prompt inviting the user to submit collected feedback to the Spec Kit developers; on confirmation run `--action mark-submitted`. Below threshold, do not prompt.

**Abort / partial-run rule.** If the run failed before wrap-up, either skip recording or record with `--partial` and a `## Review` beginning `**Partial run** — `.
