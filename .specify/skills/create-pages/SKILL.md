---
name: create-pages
description: |
  Set up Hugo-based documentation pages deployment: a code-hosting-platform CI
  pipeline (parameterized per platform, default aoneci), Hugo site config,
  layouts, and build script — all confined to a docs/ directory with zero
  root-level contamination; build image and hosting platform are parameters.
  Use when the user mentions ["setup pages", "hugo build", "deploy docs",
  "create-pages", "documentation site", "文档构建", "页面部署", "CI配置"]
skill_id: "<SKILL:.specify/skills/create-pages/SKILL.md>"
---

# Create Pages — Hugo docs deployment setup

## Overview

One deterministic scaffold creates the complete Hugo deployment infrastructure
for a project's documentation: CI pipeline, site config, layouts, and build
script. All artifacts live inside the docs directory; the project root stays
clean, and deleting the docs directory never affects core logic or build flow.

Do NOT use this skill when the project already has a different front-end
build pipeline, or when there is no documentation directory to serve.

## Core Principles (non-negotiable)

1. **Isolation** — every Hugo artifact (config, layouts, build script) MUST
   live inside the docs directory. Zero root-level contamination.
2. **CI guard** — on every platform, the pipeline build step MUST be guarded
   by a docs-directory existence check so docs removal is always safe.
3. **Bundle transform** — the build MUST rename `index.md` → `_index.md` in a
   staging copy before invoking Hugo; never rename inside the docs directory.
4. **Raw HTML** — Hugo config MUST enable the unsafe goldmark renderer
   (collected materials contain inline HTML).
5. **Deterministic scaffolding** — file creation MUST go through
   `${SKILL_HOME}/scripts/scaffold.sh`; never retype templates manually.

Rationale for each principle: [`./references/design-rationale.md`](./references/design-rationale.md).

## Workflow

### Step 1 — Preflight

Verify preconditions before scaffolding:

- A docs directory with `.md` content exists (the six-type taxonomy from
  `/speckit.docs` is supported but not required). If absent, stop and ask the
  user whether to proceed anyway.
- Check for pre-existing targets — the chosen platform's CI file (registry:
  `${SKILL_HOME}/scripts/ci-templates/README.md`) and `<docs>/hugo.yaml`: if any
  exist, confirm overwrite intent with the user before passing `--force`.
- Local verification needs `docker` with the chosen Hugo image; if
  unavailable, surface this and defer verification to CI.

### Step 2 — Collect parameters (judgment)

Derive or ask for the scaffold parameters; do not guess silently:

| Parameter | Flag | Default |
|-----------|------|---------|
| Deploy site name | `--site-name` | (required — ask or derive from repo name) |
| Code-hosting platform | `--platform` | `aoneci` (registry: `${SKILL_HOME}/scripts/ci-templates/README.md`) |
| Site title | `--title` | `<site-name> 文档` |
| Production branch | `--branch` | `main` |
| Hugo docker image | `--image` | `reg.docker.alibaba-inc.com/xuanji-images/hugo:latest` |
| Docs directory | `--docs-dir` | `docs` |

Parameter notes:

- **`--image` is environment-specific.** The default image is only pullable
  in the aoneci/internal environment. Outside it, ask the user for their
  Hugo image (or a local `hugo` binary plan) before scaffolding.
- **`--platform` drives only the CI file.** Hugo config, layouts, and build
  script are platform-independent. Platforms without a template yet (e.g.
  `github`) still scaffold everything else and report a warning naming the
  manual-authoring contract (`scripts/ci-templates/<platform>/README.md`).

### Step 3 — Scaffold (deterministic)

Run the scaffold script from the project root with the collected parameters:

```bash
bash "${SKILL_HOME}/scripts/scaffold.sh" --site-name <name> \
  [--platform <platform>] [--image <image>] [--title ...] [--branch ...] [--force]
```

Review the JSON summary it prints (`created` / `skipped` / `warnings`). Any
`skipped` entry without a prior overwrite confirmation is a stop condition —
report it to the user instead of forcing.

### Step 4 — Verify

Follow [`./references/verification.md`](./references/verification.md): docker
build test, output checks (page count, title extraction, no config leaked into
`dist/`, no empty taxonomies), and the no-docs guard simulation. Report the
results; do not claim success without a clean build.

### Step 5 — Wrap-up

Report the created files and verification outcome; suggest committing them.
Then run the Feedback step below.

## Resources

| Path | Contents |
|------|----------|
| `${SKILL_HOME}/scripts/scaffold.sh` | Deterministic scaffolding of all deployment files (run with `--help`) |
| `${SKILL_HOME}/scripts/ci-templates/` | Per-platform CI templates + registry/extension contract (`README.md`); `aoneci/` implemented, `github/` structural stub |
| `${SKILL_HOME}/references/design-rationale.md` | Why behind each core principle (observed failures) |
| `${SKILL_HOME}/references/verification.md` | Docker build verification and no-docs guard test |

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
     --unit-id "skill:create-pages" --unit-type skill \
     --run-id "<stable-run-id>" --feature "<feature-key-if-any>" \
     --review "<review prose>" --points-file "<points file>"
   ```
   Probe attribution: the engine resolves the unit to its probe object automatically — the entry inherits kind/slice from the probe registry. External custom units record via `--unit-id custom:<owner>/<name> --unit-type custom-unit`; their entries stay host-project-local and never enter upstream packages.
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
