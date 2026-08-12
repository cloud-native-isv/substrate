# 归口 instructions 文件查找与 Git Workflow 块写入

本技能把三层分支信息写入**归口 instructions 文件**的 `## Git Workflow` 块，不生成独立的工作流文档。

## 查找优先级

1. `${SKILL_WORKDIR}/.specify/instructions.md`（优先，Spec Kit 项目的规范位置）
2. 当前 AI 工具对应的 instructions 文件（找到第一个即停止）：

| 工具 | 兼容性 instructions 文件 |
|------|--------------------------|
| GitHub Copilot | `${SKILL_WORKDIR}/.github/copilot-instructions.md` |
| Claude Code | `${SKILL_WORKDIR}/CLAUDE.md` |
| Qoder CLI (`qodercli`) | `${SKILL_WORKDIR}/AGENTS.md`（CLI 只读 AGENTS.md，不读 project_rules.md） |
| Qoder IDE | `${SKILL_WORKDIR}/QODER.md` 或 `${SKILL_WORKDIR}/.qoder/project_rules.md`（IDE 旧格式） |
| opencode | `${SKILL_WORKDIR}/AGENTS.md` |

3. 若以上文件均不存在，创建 `${SKILL_WORKDIR}/.specify/instructions.md` 并写入 `## Git Workflow` 章节。

> **符号链接注意**：在 Spec Kit 项目中，`CLAUDE.md` / `QODER.md` / `AGENTS.md` / `.github/copilot-instructions.md` 通常是指向 `.specify/instructions.md` 的**符号链接**。写入任一别名都会写穿到同一个规范文件——因此只写一次，**不要**对多个别名重复写入，也不要删除重建这些链接。

## 写入规则

```bash
# 判断块是否已存在
grep -q '<!-- GIT_WORKFLOW_START -->' "<instructions-file>" && echo "block present" || echo "block missing"
```

- **块已存在**：只替换 `<!-- GIT_WORKFLOW_START -->` 与 `<!-- GIT_WORKFLOW_END -->` **之间**的内容；标记本身与块外内容保持字节不变。
- **块不存在**：在 `## Resource Registry` 章节之前插入完整章节（标题 + 说明句 + 标记 + 表格）。文件没有 `## Resource Registry` 时追加到文件末尾。
- **占位状态**：未建立工作流时块内保留 `| None yet. | - | - | - |` 行——这一行即 Bootstrap 作用域的判定依据。
- **单一数据源**：分支信息只写入该块。不要同时在 Documentation Map 里加引用行（无独立文档可引用），也不要另建 `docs/git-workflow.md` 或 `.specify/memory/git-workflow.md`。

## 块内容

模板见 `${SKILL_HOME}/assets/git-workflow-block.md`。填充后的形态：

```markdown
<!-- GIT_WORKFLOW_START -->
<!-- Record one row per branch role (MAIN / PRE / DEV). While no workflow is established, keep the `None yet.` row. -->
| Role | Branch | Tracking | Purpose |
|------|--------|----------|---------|
| MAIN | `master` | `origin/master` | 上游主干，只接收已通过版本验证的代码 |
| PRE | `xuanji/prepub` | `origin/xuanji/prepub` | 预发发布分支，用于版本集成与环境验证 |
| DEV | `xuanji/hanzhi` | `origin/xuanji/hanzhi` | 本地开发分支，所有新改动先在此开发与自测 |

- **Sync chain (rebase)**: `master -> xuanji/prepub -> xuanji/hanzhi`
- **Merge chain (PR)**: `master <- xuanji/prepub <- xuanji/hanzhi`
- **Last updated**: 2026-08-10
<!-- GIT_WORKFLOW_END -->
```

## 读取（体检与定向收敛的数据源）

```bash
# 提取三层分支名（Branch 列反引号内的值）
awk '/<!-- GIT_WORKFLOW_START -->/,/<!-- GIT_WORKFLOW_END -->/' "<instructions-file>" \
  | awk -F'|' '$2 ~ /MAIN|PRE|DEV/ {gsub(/[ `]/, "", $2); gsub(/[ `]/, "", $3); print $2"="$3}'
```

输出形如 `MAIN=master` / `PRE=xuanji/prepub` / `DEV=xuanji/hanzhi`，供后续 git 操作使用。若某行的 Branch 为空或仍是 `None yet.`，视为块未填写 → 进入 Bootstrap。
