# Team Goal — 团队目标（操作规范）

**Owner**: the team domain (`create-team` / `/speckit.team`). This file is the single source of truth for the **team-side goal operations** — how a team establishes, carries, and deliberately revises the goal it serves. The **Goal concept itself** (project-level first-class entity at `.specify/goal/<goal-slug>/`, Goal vs Requirement boundary, criteria authority, singularity, and the `goal_slug` Goal–Team binding) is defined once in `.specify/shared/definitions/goal-definitions.md` — this file defers to it and MUST NOT re-define the concept. `improve-team` and other team skills link here for operations, there for the concept. It is the goal-side companion to [`conceptual-model.md`](conceptual-model.md).

## 定义

一个 **team** 由三部分组成：**goal**（总体最终目标）+ **static structure**（Role × Stage × Type 花名册）+ **dynamic structure**（协作模式）。**goal 是北极星，静态与动态结构仅为达成它而存在**——无论它们是什么，都必须围绕 goal 组织与运行。

## 四条性质

1. **北极星，不是任务清单**：goal 描述期望的**最终结果**，不是步骤。角色/阶段与协作模式都**由 goal 推导且服从于 goal**。
2. **具体且可验证**：写到「进展可被判断」——尽量带显式成功标准/质量维度与可度量目标（分数阈值、测试通过、覆盖率）。**evaluator 阶段**与 **iteration/continuous 的 `threshold`/质量维度**正是对着 goal 度量进展；**无法评估的 goal 无法驱动循环**。
3. **区别于 `description`**：`description` 是一句话标签；**goal 是整个团队围绕组织的操作性目标**。一个 team **只有一个** goal。
4. **可刻意修改，但不会漂移**：goal 在一次运行中固定，不因重构结构而作为副作用改动；但可经 `modify` 刻意重定义（见下）。

## create 模式：goal 优先

定义团队时**必须先确立 goal**，再据此推导结构：

1. **Establish the goal（第一步）**——从 `$ARGUMENTS`/对话/仓库上下文提取 goal；缺失则询问；与用户确认。写成可验证形式（含成功标准/阈值）。
2. **Derive structure from the goal**——由 goal 决定需要哪些角色/阶段（静态）与哪种协作模式（动态）。
3. 若 **goal 主题为「优化」**，进一步按 [`optimization-goals.md`](optimization-goals.md) 区分**一次性 / 持续**，并为持续优化选择**淘汰 / 渐进**策略。

## 持久化:引用优先,内联可选

一个具体 goal 的**内容**(目标叙述 + 成功判据)的单一事实源是它在 `.specify/goal/<goal-slug>/goal.md` 的**定义**,由 `/speckit.goal` 撰写。团队侧只声明它服务于哪个 goal,不复制其内容:

- frontmatter 以 **`goal_slug`** 声明所引用的 goal(**引用,不是副本**);同一 `goal_slug` 的多个团队解析到同一份定义。
- 内联的 **`goal`** 字段与 **`## Goal`** 小节仍为合法,用于可读性与存量兼容(字段顺序:`slug, name, description, goal, goal_slug, territory, pattern, members, config, created, updated`;先写 `## Goal`,静态与动态小节围绕它组织)。**当定义存在时以定义为权威**;内联与定义不一致会被显式报出供人裁决,不静默降级。
- 未声明 `goal_slug` 的存量团队仍以内联 `goal` 正常工作,MUST NOT 要求先迁移。

## modify 模式:改内容去定义,改结构在团队

- 修改某个 goal 的**内容**(重写目标或判据)是对**定义**的编辑,经 `/speckit.goal`(`modify` / `criteria` / `status`),不在 `team.md` 里就地改。
- 因 goal 变更而**重对齐花名册与协作模式**仍是团队侧的 `modify`(`improve-team`)。详见 [`../../improve-team/references/goal-editing.md`](../../improve-team/references/goal-editing.md)。
