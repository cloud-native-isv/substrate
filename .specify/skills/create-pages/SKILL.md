---
name: create-pages
description: |
  Publish a project's documentation as a static site — the optional presentation
  layer on top of a documentation space, delivered as a three-stage pipeline:
  local doc library (the docs directory) → Hugo-rendered static site
  (docs/public) → pages service (local preview / aoneci / github). Content is
  mounted, never copied, and everything the skill writes stays inside the docs
  directory except the one CI file a hosting platform requires at the repository
  root. Use when the user mentions ["setup pages", "hugo build", "hugo serve",
  "deploy docs", "create-pages", "documentation site", "Hugo", "hugo site",
  "static site", "hugo.toml", "site build", "publish docs", "gh-pages",
  "aoneci", "文档构建", "页面部署", "CI配置", "静态网站", "文档站点", "文档网站",
  "对外呈现", "发布文档", "本地预览"]
skill_id: "<SKILL:.specify/skills/create-pages/SKILL.md>"
---

# Create Pages — doc library → Hugo site → pages service

## Overview

The **optional presentation layer** on top of a documentation space. Optional
means a space is complete and valid without it: the library's structure belongs
to `create-docs`, the content of a document belongs to `improve-docs`, and this
skill only adds the machinery that renders and serves what they produce — it
never rewrites documentation.

One pipeline, three stages, run in order. Each stage has exactly one owner and
one output, so a failure is always attributable to a stage:

| Stage | Question it answers | Output | Tooling |
|-------|--------------------|--------|---------|
| **1 — 本地文档库** | What content is published? | The docs directory (default `docs/`), pure Markdown | none of this skill's — read-only here |
| **2 — Hugo 渲染** | How does Markdown become HTML? | The docs directory *is* a Hugo project; site builds to `<docs>/public` | `${SKILL_HOME}/scripts/scaffold-hugo.py` |
| **3 — Pages 服务** | Who serves the HTML? | `local` preview, or a hosting platform's CI pipeline | `hugo serve` / `${SKILL_HOME}/scripts/scaffold-ci.sh` |

Stage 3 targets — pick one, ask when the input does not say:

| Target | What it is | Status |
|--------|-----------|--------|
| `local` | A web server on this machine (`hugo serve`), for preview | works wherever `hugo` is installed |
| `aoneci` | The Alibaba-internal, GitLab-like hosting platform's pages service | implemented (`--platform aoneci`) |
| `github` | GitHub Actions + GitHub Pages (`gh-pages`) | **not implemented** — writes nothing and warns; contract in `scripts/ci-templates/github/README.md` |

`gitlab` means the open-source GitLab project and its hosting service. It is
**not** a target here — do not treat `aoneci` as "gitlab", and do not accept
`--platform gitlab` (the registry rejects it).

Do NOT use this skill when the project already has a different front-end build
pipeline, or when there is no documentation directory to serve.

## Workflow

Run the stages in order. A later stage never repairs an earlier one — it reports
back. Verification for every stage: [`./references/verification.md`](./references/verification.md).

### Stage 1 — 本地文档库 (read-only)

Establish what gets published, and nothing more:

- Resolve the docs directory (default `docs/`, else `--docs-dir`). It must exist
  and contain `.md` content; if it does not, stop and ask — do not scaffold a
  site over an empty library.
- The library is the **only** content source: every published page comes from a
  file inside it. Content outside it (root `README.md`, a sibling directory) is
  **excluded by default** and may be included only on an explicit user request,
  recorded in the report along with the mechanism chosen.
- Do not restructure, rename, or rewrite anything in the library. Placement and
  taxonomy → `create-docs`; document content → `improve-docs`.

### Stage 2 — Hugo 渲染

Make the docs directory the **Hugo project root** and render it. Deterministic — the
scaffold is a script, never hand-written HTML. Check for drift first, then
scaffold; full command catalogue, ownership map, mount rationale and
troubleshooting: [`./references/hugo-site.md`](./references/hugo-site.md).

```bash
python3 "${SKILL_HOME}/scripts/scaffold-hugo.py" --action check --root .   # drift only, no writes
```

Guarantees this stage provides and that you MUST NOT break:

- **Mount, never copy** — no `.md` is duplicated, moved, or rewritten; no staged
  copy of the tree, and `content/` never materializes on disk.
- **`index.md` stays `index.md`** on disk; it is mounted as `_index.md` (branch
  bundle) so sibling pages stay pages.
- **Repo-native links keep working** — relative `.md` links and relative image
  paths resolve through render hooks at build time; never rewrite links.
- **Only the mount block is machine-owned** — everything else in `hugo.toml`,
  and any layout or stylesheet the user edited, is reported `kept` (`--force`
  overrides). A repeat run on an unchanged tree writes nothing.
- **Build output is `<docs>/public`**, inside the library's directory — never a
  repository-root `dist/`.
- **No network, no theme dependency** — an absent `hugo` binary skips only the
  build; the scaffold is still complete and that is not a failure.

There is exactly **one** renderer. Do not add a second config, a staging copy,
or a parallel build script for a particular hosting target.

### Stage 3 — Pages 服务

Serve the rendered site. For `local`, nothing is scaffolded — run Hugo's own
server from the docs directory (`hugo serve`, live reload) and report the URL.

For a hosting platform, render its pipeline (one file, no Hugo artifacts):

```bash
bash "${SKILL_HOME}/scripts/scaffold-ci.sh" --site-name <name> [--platform aoneci|github] [--branch ...] [--image ...] [--force]
```

- **Preflight**: an existing CI file is skipped, not overwritten — confirm
  overwrite intent with the user before passing `--force`. Review the JSON
  summary (`created` / `skipped` / `warnings`); a `skipped` entry without a
  prior confirmation is a stop condition.
- **The rendered CI file is the only artifact outside the docs directory**,
  because platforms discover pipelines at a fixed repository-root path. That is
  the single sanctioned exception to stage 1's containment rule — never widen it.
- **`--image` is environment-specific.** The default is pullable only in the
  Alibaba-internal environment; outside it, ask for the user's Hugo image (or a
  local-target plan) before scaffolding.
- Platform without a template (`github` today) → nothing is written and a
  warning names the manual-authoring contract. Report that honestly; do not
  claim the target is wired.

### Wrap-up

Report per stage: the library that was published, the scaffolded/kept Hugo
files, the stage-3 target with its verification outcome, and any content
deliberately excluded. Suggest committing. Then run the Feedback step below.

## Resources

| Path | Contents |
|------|----------|
| `${SKILL_HOME}/scripts/scaffold-hugo.py` | **Stage 2**: deterministic mount-mode scaffolder (`scaffold` / `check` / `mounts` / `build`) — computes the module-mount block from the live docs tree, places layouts and stylesheet, never clobbers user-edited files. Stdlib-only, one JSON object per invocation |
| `${SKILL_HOME}/assets/hugo/` | **Stage 2**: `hugo.toml.tmpl` (`{{SITE_TITLE}}` / `{{SITE_DESCRIPTION}}` / `{{MOUNTS}}`), `layouts/` (`_default/baseof,list,single`, `index.html`, `_markup/render-link,render-image`), `static/css/site.css`, `dotgitignore` → `<docs>/.gitignore` |
| `${SKILL_HOME}/references/hugo-site.md` | **Stage 2**: ownership map, mount rationale (`index.md` → `_index.md`), link/image render hooks, publish scope, commands, local serve, CI guidance, troubleshooting |
| `${SKILL_HOME}/scripts/scaffold-ci.sh` | **Stage 3**: renders one hosting platform's CI pipeline and nothing else (run with `--help`) |
| `${SKILL_HOME}/scripts/ci-templates/` | **Stage 3**: per-platform pipeline templates + registry/extension contract (`README.md`); `aoneci/` implemented, `github/` structural stub |
| `${SKILL_HOME}/references/design-rationale.md` | Why behind each stage boundary and guarantee (observed failures) |
| `${SKILL_HOME}/references/verification.md` | Per-stage verification: render check, output checks, local serve, no-docs guard |

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
   non-blocking notification inviting submission (point the user to the `/speckit.feedback package` command — the user-facing path; never paste the raw `feedback-utils.py` engine call into the user-facing line); the wrap-up MUST NOT pause for the choice and MUST NOT trigger any automated transmission (silence = skip). Below threshold, do NOT prompt.
   The detailed prompt semantics (package → manual send → mark-submitted, plus the
   skip / silence options) live in the canonical protocol:
   `.specify/shared/workflow/feedback-step.md` § *Threshold prompt protocol*.

**Abort / partial-run rule.** If the run failed or was interrupted before wrap-up, either
skip recording OR record with `--partial` and a `## Review` that begins with
`**Partial run** — `. Never present a partial run as a complete review.

**Nesting rule.** When a command invokes a skill (or a skill invokes a skill), each
qualifying unit records feedback for **its own** scope only, keyed by its own
`(unit_id, run_id)`.
