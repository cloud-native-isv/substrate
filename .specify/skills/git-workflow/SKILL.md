---
name: git-workflow
description: |
  Three-tier Git workflow management skill that dynamically discovers or defines branch names (trunk/pre-release/dev) and records them into the project's instructions file (`.specify/instructions.md` → `## Git Workflow` managed block) as the single source of truth — no separate workflow document is generated. Runs as a single **reconcile engine** (see `.specify/shared/patterns/reconcile-pattern.md`): the desired state is the declared three-tier branch structure + sync relations + per-branch `.gitexcludes` rules; every invocation observes the current repo state, diffs, and converges — bootstrap (interactive branch naming + creation + .gitexcludes init when the block is absent or unfilled), health-check (structure/sync/.gitexcludes observation + residual report), and directed convergence (rebase sync, merge, and safe push with automatic .gitexcludes enforcement) are the same engine under different inputs. Covers pre-checks, rebase synchronization, conflict resolution, force-with-lease push strategies, and per-branch file exclusion via `.gitexcludes`. Use this when the user mentions ["git workflow", "branch sync", "rebase sync", "分支同步", "git rebase", "force-with-lease", "发布流程", "分支策略", "主干分支", "预发分支", "开发分支", "three-tier git", "git workflow setup", "创建git工作流", "工作流维护", "workflow health check", "工作流检查", "selective merge", "选择性合并", "排除文件", "忽略配置文件", "分支排除", "branch-exclusive", ".gitexcludes", "开发专属文件", "merge filter", "调谐工作流", "git 调谐"]
skill_id: "<SKILL:.specify/skills/git-workflow/SKILL.md>"
---

# git-workflow

## Overview

三层 Git 开发工作流管理技能。**只有一个运行模式——调谐（Reconcile）**，遵循 [`.specify/shared/patterns/reconcile-pattern.md`](../../shared/patterns/reconcile-pattern.md)：

- **期望态（spec）** = 归口 instructions 文件的 `## Git Workflow` 块（`<!-- GIT_WORKFLOW_START -->` … `<!-- GIT_WORKFLOW_END -->`）声明的三层分支结构（MAIN/PRE/DEV 存在且 tracking 正确）+ 固定 rebase 同步关系（无积压分叉）+ 各分支 `.gitexcludes` 规则就位且被排除路径未被跟踪 + 本次用户操作指令。
- **当前态（status）** = 仓库实际分支结构、同步状态、`.gitexcludes` 分布。
- **调谐** = 观测 → diff（过容忍带）→ 收敛（分级确认）→ 校验。

同一调谐引擎在不同输入下表现为三种作用域：

| 输入 | 作用域 | 行为 |
|------|--------|------|
| instructions 的 `## Git Workflow` 块缺失，或仍是 `None yet.` 占位行 | **Bootstrap** | 交互确认分支名、创建分支、初始化 `.gitexcludes`、写入 Git Workflow 块 |
| 块已填写 且 无操作参数 | **全维度体检** | R1 观测 + R3 diff + 残差汇报，不自动收敛 |
| 块已填写 且 有操作参数 | **定向收敛** | 用户指令并入期望态，执行具体 git 操作（自动尊重 `.gitexcludes`） |

**容忍带**：同步状态 `0 0`、`.gitexcludes` 内容语义等价、Git Workflow 块与实际分支名一致——均标记「已一致（容忍）」不触发收敛；仅 ahead/behind 分叉、分支缺失、排除路径仍被跟踪等实质偏离进入收敛建议。

**分级确认门禁**（与模式文档对齐）：

| 动作类 | 门禁 |
|--------|------|
| 只读观测、体检报告、本地分支创建 | 自动执行 |
| 写远端（push / 建远端分支）、直接 merge 入 PRE/MAIN | **停下确认**（外部权威源） |
| 共享分支 force-with-lease 强推 | **停下确认** + 团队同步窗口 + 回滚预案 |

**强制产物**：体检的「工作流维护报告」同时充当观测快照与残差汇报；定向收敛的审计由 git 提交历史天然承担。

### 分支角色

| 角色 | 含义 | 说明 |
|------|------|------|
| **`MAIN`** | 主干分支 | 上游主干，只接收已通过版本验证的代码 |
| **`PRE`** | 预发分支 | 预发发布分支，用于版本集成与环境验证 |
| **`DEV`** | 开发分支 | 本地开发分支，所有新改动先在此开发与自测 |

> **重要**：分支名称因项目而异（如 `master` / `xuanji/prepub`，或 `main` / `staging` / `dev`）。本技能在执行时**动态确认**实际分支名，将其记录到归口 instructions 文件的 `## Git Workflow` 块（优先 `${SKILL_WORKDIR}/.specify/instructions.md`），后续操作以该块为准。

核心链路（角色代号）：

```
代码同步：MAIN -> PRE -> DEV
代码合入：MAIN <- PRE <- DEV
```

固定 rebase 关系：`PRE` 基于 `MAIN` rebase；`DEV` 基于 `PRE` rebase。

---

## Workflow

### Phase 0: 调谐作用域判定

1. 若存在任一遗留配置文档（旧位置 `docs/git-workflow.md`，或前一代位置 `.specify/memory/git-workflow.md`），先把其 frontmatter 的分支映射提取到 Git Workflow 块，并把该遗留文件报告为「已冗余，待用户确认处理」——**不自动删除**（命令见 [bootstrap-commands.md](./references/bootstrap-commands.md#遗留配置迁移)）。
2. 检查归口 instructions 文件的 `## Git Workflow` 块是否已填写（存在 MAIN/PRE/DEV 行，且不是 `None yet.` 占位行）。
3. 检查用户是否传入了操作参数（具体的 git 操作指令）。

| 块已填写 | 有操作参数 | 进入作用域 |
|----------|------------|------------|
| 否 | — | Bootstrap 调谐（Setup） |
| 是 | 否 | 全维度体检（Maintain） |
| 是 | 是 | 定向收敛（Execute） |

---

### Bootstrap 调谐（Setup）— 建立工作流

当归口 instructions 文件的 `## Git Workflow` 块缺失或仍是占位行时进入：期望态 = 完整三层工作流骨架，通过最少必要问题（逐一确认，不臆造分支名）+ 自动探测补齐信息。命令序列见 [bootstrap-commands.md](./references/bootstrap-commands.md)。

#### 1.1 检测现有分支

命令见 [bootstrap-commands.md#分支检测](./references/bootstrap-commands.md#分支检测)。

#### 1.2 交互式确认分支名

逐一向用户确认（每次只问一个问题），不臆造分支名：

1. **主干分支 `MAIN`**：推荐远端常见候选（`main`/`master`），询问选择或自定义。
2. **预发分支 `PRE`**：询问是否存在；不存在则建议命名（`staging`/`release`/`prepub`）。
3. **开发分支 `DEV`**：同上逻辑，推荐命名（`dev`/`develop`）。

#### 1.3 创建缺失分支

命令见 [bootstrap-commands.md#创建缺失分支](./references/bootstrap-commands.md#创建缺失分支)。

#### 1.4 写入 Git Workflow 块

读取模板 `${SKILL_HOME}/assets/git-workflow-block.md`，替换 `<MAIN>` / `<PRE>` / `<DEV>`（分支名）、`<MAIN_TRACKING>` / `<PRE_TRACKING>` / `<DEV_TRACKING>`（各分支 upstream，无则 `-`）、`<DATE>`（运行日期），写入归口 instructions 文件的 `## Git Workflow` 块。

- 目标文件查找优先级与写入规则见 [instructions-lookup.md](./references/instructions-lookup.md)。
- 已存在 `<!-- GIT_WORKFLOW_START -->` … `<!-- GIT_WORKFLOW_END -->` 标记：**只替换标记之间的内容**，标记与块外内容字节保持。
- 标记不存在：在 `## Resource Registry` 之前插入完整 `## Git Workflow` 章节（含标记）。
- 该块由本技能独占维护，`/speckit.instructions` 只观测不收敛（managed range）——不要在块外另存分支信息，避免出现第二个数据源。

#### 1.5 初始化 `.gitexcludes`（分支排除规则）

`.gitexcludes` 定义本分支专属文件（语法同 `.gitignore`），目标分支的规则在同步时保护其专属文件不被覆盖。机制详见 [gitexcludes-subroutine.md](./references/gitexcludes-subroutine.md) 的设计原则，初始化命令见 [bootstrap-commands.md#初始化-gitexcludes](./references/bootstrap-commands.md#初始化-gitexcludes)。

---

### 全维度体检（Maintain）— 观测 + diff + 残差汇报

当归口 instructions 文件的 `## Git Workflow` 块已填写且用户未传入操作参数时进入。只观测与 diff，不自动收敛；收敛建议列入报告尾部待用户确认。命令序列见 [health-check-commands.md](./references/health-check-commands.md)。

#### 2.1 加载配置

从归口 instructions 文件的 `## Git Workflow` 块读取分支映射：表格中 `MAIN` / `PRE` / `DEV` 行的 `Branch` 列即三层分支名（反引号内），`Tracking` 列为各自 upstream。

#### 2.2 分支结构检查

命令见 [health-check-commands.md#分支结构检查](./references/health-check-commands.md#分支结构检查)。检查项：MAIN/PRE/DEV 分支是否存在于本地和远端、tracking 关系是否正确。

#### 2.3 同步状态检查

命令见 [health-check-commands.md#同步状态检查](./references/health-check-commands.md#同步状态检查)。

#### 2.4 `.gitexcludes` 一致性检查

命令见 [health-check-commands.md#gitexcludes-一致性检查](./references/health-check-commands.md#gitexcludes-一致性检查)。检查项：各分支是否都有 `.gitexcludes`、排除路径是否仍被跟踪、规则是否符合预期。

#### 2.5 Git Workflow 块一致性检查

- 块中记录的分支名与实际分支是否一致（不一致属实质偏离，进入收敛建议）
- 块结构完整（MAIN/PRE/DEV 三行齐备、`Tracking` 与实际 upstream 相符、`Last updated` 存在）
- 标记 `<!-- GIT_WORKFLOW_START -->` / `<!-- GIT_WORKFLOW_END -->` 成对存在（缺失会让 `/speckit.instructions` 无法保护该块）
- 不存在第二个数据源：遗留 `docs/git-workflow.md` 或 `.specify/memory/git-workflow.md` 若仍在，报告为冗余待处理

#### 2.6 输出维护报告

容忍带内的项标记「✅ 已一致」；仅实质偏离进入「建议操作」。报告模板见 [health-check-commands.md#维护报告模板](./references/health-check-commands.md#维护报告模板)。

---

### 定向收敛（Execute）— 执行工作流

当归口 instructions 文件的 `## Git Workflow` 块已填写且用户传入了具体操作参数时进入：用户指令并入期望态，按下列预定义操作收敛，写远端/直接合入类动作遵循 Overview 的分级确认门禁。命令序列见 [execute-commands.md](./references/execute-commands.md)。

#### 3.1 加载配置

同全维度体检 Step 2.1，从 `## Git Workflow` 块的表格读取 `MAIN` / `PRE` / `DEV` 分支名。

#### 3.2 前置校验

命令见 [execute-commands.md#前置校验](./references/execute-commands.md#前置校验)。

> **Gate**：`git status --short` 必须为空，才能继续执行。

#### 3.3 通用排除处理子程序（.gitexcludes Subroutine）

所有同步/合并操作均自动调用此子程序。详细实现见 [gitexcludes-subroutine.md](./references/gitexcludes-subroutine.md)。

**核心逻辑**：前置（保存状态 + 打印保护清单）→ 执行 rebase/merge → 后置（恢复排除文件 + 移除新引入文件 + 打印结果 + 提交 + 清理标签）。

> **设计原则**：谁接收代码（目标分支），谁的 `.gitexcludes` 说了算（方向无关）；`.gitexcludes` 本身是固定排除项，各分支内容可不同，永不被其他分支覆盖。

#### 3.4 解析并执行操作

根据用户指令匹配预定义操作：

##### 操作 A: 代码同步（MAIN → PRE → DEV）

触发词：同步、sync、拉取上游更新。命令见 [execute-commands.md#操作-a-代码同步](./references/execute-commands.md#操作-a-代码同步)。
推送策略：仅 ahead → 正常推送；ahead + behind → 确认团队同步窗口后 `--force-with-lease` 推送（Security 规则 3-4）。

##### 操作 B: 提交到预发（DEV → PRE）

触发词：提交到预发、合入预发、merge to pre、提测。建议通过 PR 流程；或直接合入（需用户确认）。命令见 [execute-commands.md#操作-b-提交到预发](./references/execute-commands.md#操作-b-提交到预发)。

##### 操作 C: 提交到主干（PRE → MAIN）

触发词：提交到主干、合入主干、merge to main、发布。建议通过 PR 流程；或直接合入（需用户确认）。命令见 [execute-commands.md#操作-c-提交到主干](./references/execute-commands.md#操作-c-提交到主干)。

> **安全检查**：禁止跳过 `<PRE>` 直接把 `<DEV>` 合入 `<MAIN>`。

##### 操作 D: 基于指定分支 rebase

触发词：rebase、变基。命令见 [execute-commands.md#操作-d-rebase](./references/execute-commands.md#操作-d-rebase)。

##### 操作 E: 自定义操作

对于无法匹配预定义操作的用户指令，根据本技能的 Security 底线与 `## Git Workflow` 块声明的分支关系理解用户意图，拆解为安全的 git 操作序列。

##### 操作 F: 管理 `.gitexcludes`

触发词：排除规则、gitexcludes、配置排除、分支专属文件、添加排除、移除排除。命令见 [execute-commands.md#操作-f-管理-gitexcludes](./references/execute-commands.md#操作-f-管理-gitexcludes)。

##### 操作 G: 验证排除效果

触发词：验证排除、检查排除状态、确认排除生效。命令见 [execute-commands.md#操作-g-验证排除效果](./references/execute-commands.md#操作-g-验证排除效果)。

---

## Security / 安全底线

1. `<MAIN>` 禁止直接 push 未审查代码。
2. 禁止跳过 `<PRE>` 直接把 `<DEV>` 合入 `<MAIN>`。
3. 禁止 `git push -f`，仅允许 `git push --force-with-lease`。
4. 对共享分支执行强推前，必须完成"通知 + 同步窗口 + 回滚预案"。
5. 同步/合并后必须验证 `.gitexcludes` 匹配的路径未被意外修改。
6. `.gitexcludes` 文件本身是**固定排除项**，各分支独立维护，永不被其他分支版本覆盖。
7. 排除子程序的前置和后置必须打印明确的文件清单，供用户确认哪些文件被保护/移除。

## Known Issues & Mitigations

常见异常现象与应对策略见 [troubleshooting.md](./references/troubleshooting.md)。

## Git Workflow 块维护

- **创建**：Bootstrap 调谐（Setup）Step 1.4 用 `${SKILL_HOME}/assets/git-workflow-block.md` 写入归口 instructions 文件。
- **更新**：分支改名或 tracking 变化时改对应表格单元并刷新 `Last updated`；新增异常经验追加到 [troubleshooting.md](./references/troubleshooting.md)。
- **数据源**：`## Git Workflow` 块是后续所有操作的**唯一**分支名数据源；各分支的 `.gitexcludes` 是排除规则的数据源。本技能**不生成独立工作流文档**——操作规程留在本技能与其 references 中，避免第二个数据源。
- **边界**：块由本技能独占写入，`/speckit.instructions` 视其为 managed range（只观测不收敛）；块外的 instructions 内容不属本技能职责。

## Resource ID

- Canonical ID: `<SKILL:.specify/skills/git-workflow/SKILL.md>`
- Canonical Path: `.specify/skills/git-workflow/SKILL.md`

## Path Conventions

- `${SKILL_HOME}/<relative-path>` — Skill-owned resources (scripts, references, assets).
- `${SKILL_WORKDIR}/<relative-path>` — runtime/user-facing paths.

## Resources

### References (`${SKILL_HOME}/references/`)
- `instructions-lookup.md` — 归口 instructions 文件查找优先级与 Git Workflow 块写入规则。
- `gitexcludes-subroutine.md` — `.gitexcludes` 通用排除子程序详细实现。
- `bootstrap-commands.md` — Bootstrap 调谐（Phase 0-1）命令序列：迁移、分支检测/创建、`.gitexcludes` 初始化。
- `health-check-commands.md` — 全维度体检（Phase 2）命令序列：结构/同步/排除检查、维护报告模板。
- `execute-commands.md` — 定向收敛（Phase 3）命令序列：前置校验、操作 A-G 命令。
- `troubleshooting.md` — 异常现象与应对策略（Known Issues）。

### Assets (`${SKILL_HOME}/assets/`)
- `git-workflow-block.md` — `## Git Workflow` 块生成模板，含 `<MAIN>` / `<PRE>` / `<DEV>` / `<*_TRACKING>` / `<DATE>` 占位符与 START/END 标记。

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
     --unit-id "skill:git-workflow" --unit-type skill \
     --run-id "<stable-run-id>" --feature "<feature-key-if-any>" \
     --review "<review prose>" --points-file "<points file>"
   ```
6. **Consolidated submission prompt.** If the returned `should_prompt` is `true`, surface a single consolidated prompt inviting the user to submit collected feedback to the Spec Kit developers; on confirmation run `--action mark-submitted`. Below threshold, do not prompt.

**Abort / partial-run rule.** If the run failed before wrap-up, either skip recording or record with `--partial` and a `## Review` beginning `**Partial run** — `.
