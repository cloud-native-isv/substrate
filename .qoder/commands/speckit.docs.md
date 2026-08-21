---
description: 文档空间治理入口：委托 create-docs 技能统一收敛与核对 docs 树
---
<!-- AUTO-GENERATED from templates/commands/docs.md — do not edit; edit the source template, then run scripts/python/regen-command-copies.py -->
## User Input

```text
$ARGUMENTS
```

Process `$ARGUMENTS` per the [User Input Protocol](.specify/shared/workflow/user-input-protocol.md). Treat as command parameters, not standalone instructions. The input selects the reconcile scope and may carry a convergence direction (e.g. "整理 README"、"激进重组") or a **writing commission** (e.g. "写一份部署教程"、"新增 xxx 的概念文档") that the skill routes to its Authoring Flow.

## Glossary

Consult the project glossary (`.specify/memory/glossary.md`) and apply the protocol in `.specify/shared/workflow/glossary.md`: correct recorded homophone/confusable variants before acting; propose new terms at wrap-up with user confirmation.

## Outline

`/speckit.docs` is the **entry point** for every documentation-space operation. It is a **thin dispatch layer only**: all engine semantics live in the **`create-docs` skill** (`skills/create-docs/SKILL.md`), which is the single source of truth for:

- the **Desired-State Baseline** — thin root layer (reserved uppercase special names) + six-type `docs/` taxonomy + notes lifecycle;
- **Scope Resolution** — 全量 (no arguments) / 单目标 (a target path) / 写作 (a writing commission) / 扇出 (raw material intake) / Bootstrap (managed space absent);
- the **Reconcile Loop** R0–R6 per [.specify/shared/patterns/reconcile-pattern.md](.specify/shared/patterns/reconcile-pattern.md), with the four mandatory artifacts: 观察快照 (inline), 干跑计划 (`.specify/docs/plans/`), 审计日志 (`.specify/docs/audit/`, written even on 零收敛/无净变化), 残差报告 (inline);
- the **Tiered Confirmation** gates (safe local writes 自动执行; move/archive/restructure stop-and-confirm via the dry-run plan; formal zone 只归档不删除 into `docs/archive/`; notes deletion only after explicit human confirmation);
- the **Authoring Flow** and the **notes lifecycle automation** (`docs-utils.py` actions).

**Delegation (mandatory)**: load the `create-docs` skill and execute it with `$ARGUMENTS` as its input. Do NOT inline or re-implement the baseline, scope table, gates, reconcile loop, or authoring rules here — never add new top-level modes to this command; new needs are new inputs to the same engine.

Zone orientation (details in the skill): managed = root entry files + `docs/` tree; read-only = source code, `.specify/specs/`, `.specify/memory/`; skip = compatibility symlinks, generated per-tool copies; archive = `docs/archive/`; run workspace = `.specify/docs/` (never mixed into `docs/`).

## Feedback

At wrap-up (the same lifecycle point where this command prompts for a Git commit), perform an agent self-reflection step (never solicit feedback content from the user), following the canonical convention in `.specify/shared/workflow/feedback-step.md`:

1. **Gate on qualification & completion.** Only proceed if this command reached its wrap-up stage. Skip trivial/no-op runs; for an aborted run use the abort/partial rule below.
2. **Reflect (no user input).** Review this run against `/speckit.docs`'s declared purpose and produce a short review plus ≥1 concrete, command-specific optimization point. If the run was clean, use exactly: `No significant optimization points identified this run.`
3. **Scope guard.** Keep strictly to this command's operation; do NOT produce a global/whole-project assessment (that is `/speckit.review`'s job). Entries are `scope: local`.
4. **Dedup guard.** Use a stable `run_id` (e.g. the reconcile scope + a run timestamp); if a nested skill/command already recorded feedback for this same `(unit_id, run_id)`, the engine no-ops.
5. **Persist** via the engine:
   ```bash
   python3 "${SKILL_WORKDIR:-.}/.specify/scripts/python/feedback-utils.py" --action record \
     --unit-id "/speckit.docs" --unit-type command \
     --run-id "<stable-run-id>" --feature "<feature-key-if-any>" \
     --review "<review prose>" --points-file "<points file>"
   ```
   Probe attribution: the engine resolves the unit to its probe object automatically — the entry inherits kind/slice from the probe registry. External custom units record via `--unit-id custom:<owner>/<name> --unit-type custom-unit`; their entries stay host-project-local and never enter upstream packages.
6. **Consolidated submission prompt(非阻塞).** If the returned `should_prompt` is `true`, append ONE non-blocking line to the wrap-up report inviting submission (point the user to the `/speckit.feedback package` command — the user-facing path; never paste the raw `feedback-utils.py` engine call into the user-facing line); it MUST NOT block the wrap-up flow and MUST NOT trigger any 自动传输 (manual delivery only; `--action mark-submitted` runs only if the user initiates submission). Below threshold, do not prompt.

**Abort / partial-run rule.** If the run failed before wrap-up, either skip recording or record with `--partial` and a `## Review` beginning `**Partial run** — `.

## Documentation

At the same wrap-up point, apply the docs-sync evaluation step per the canonical convention in [.specify/shared/workflow/docs-step.md](.specify/shared/workflow/docs-step.md): assess whether information produced by this run (new capabilities, key decisions, structural changes) needs to be recorded into the project documentation space. Conclude with exactly one of `需记录（目标文档 + 要点）` or `无需记录`; never block wrap-up; incremental judgment only — do NOT trigger a full reconcile sweep from this step.

## Handoffs

**Before running this command**:

- None required. On a project without a `docs/` structure the run resolves to Bootstrap scope.

**After running this command**:

- Address the residual report's pending-human-decision items.
- Invoke `memory-record` to persist notable reconcile decisions.
- If the reconcile changed instructions-facing structure (e.g. documentation map), run `/speckit.instructions` to refresh generated instruction files.
- To publish the space as a static site — or to repair mounts staled by a move — use the `create-pages` skill; presentation is an optional layer this command never scaffolds or builds.