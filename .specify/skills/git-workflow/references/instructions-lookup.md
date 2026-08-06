# Instructions 文档查找优先级

在 Setup 模式完成后需要更新 instructions 文档的 Documentation Map。按以下优先级查找目标文档：

1. `${SKILL_WORKDIR}/.specify/instructions.md`（优先）
2. 当前 AI 工具对应的 instructions 文件（找到第一个即停止）：

| 工具 | 兼容性 instructions 文件 |
|------|--------------------------|
| GitHub Copilot | `${SKILL_WORKDIR}/.github/copilot-instructions.md` |
| Claude Code | `${SKILL_WORKDIR}/CLAUDE.md` |
| Qwen Code | `${SKILL_WORKDIR}/QWEN.md` |
| Qoder CLI (`qodercli`) | `${SKILL_WORKDIR}/AGENTS.md`（CLI 只读 AGENTS.md，不读 project_rules.md） |
| Qoder IDE | `${SKILL_WORKDIR}/QODER.md` 或 `${SKILL_WORKDIR}/.qoder/project_rules.md`（IDE 旧格式） |
| opencode | `${SKILL_WORKDIR}/AGENTS.md` |

3. 若以上文件均不存在，创建 `${SKILL_WORKDIR}/.specify/instructions.md` 并写入引用行。

## 写入内容

在目标文档的 Documentation Map 表格中添加一行：

```markdown
| **Git Workflow** | `.specify/memory/git-workflow.md` | 分支同步机制与操作文件 | 三层分支模型、rebase 同步流程、推送策略、安全底线 |
```
