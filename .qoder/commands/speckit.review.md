---
description: 批判性评审当前特性的 SDD 过程，出具改进导向报告
---
<!-- AUTO-GENERATED from templates/commands/review.md — do not edit; edit the source template, then run scripts/python/regen-command-copies.py -->
## User Input

```text
$ARGUMENTS
```

Process `$ARGUMENTS` per the [User Input Protocol](.specify/shared/workflow/user-input-protocol.md). Treat as command parameters guiding review focus.

## Goal

Produce a **self-contained, improvement-focused** review report for spec-kit framework maintainers. The report MUST:
1. Be readable standalone — embedded evidence, absolute paths or `{REPO_URL}@{COMMIT_SHA}` references
2. Prioritize problems, friction, ambiguities in the SDD process
3. Focus on process/tooling quality, not business merits

## Operating Constraints

- **Problem-first**: Do not narrate artifact contents. Identify process gaps.
- **Self-contained**: Evidence quoted inline. No references to local-only resources.
- **Evidence-backed**: Every finding cites a specific quoted excerpt. Drop unsupported findings.
- **Process scope only**: Don't judge feature business content.

## Outline

### 1. Capture portable project context

Run `.specify/scripts/bash/check-prerequisites.sh --json --require-spec --include-spec --include-plan --include-tasks`; parse REQUIREMENTS_DIR, FEATURE_ID, FEATURE_NAME, AVAILABLE_DOCS. Then capture via git/shell: REPO_NAME, REPO_URL, BRANCH, COMMIT_SHA, REPO_ROOT_ABS, REVIEW_DATE, REVIEWER, ENVIRONMENT, SPECKIT version, ARTIFACT_INVENTORY (basename, path, line count, one-line summary per artifact).

### 2. Reconstruct process execution history

From `git log` scoped to REQUIREMENTS_DIR: commit ordering, command traces (distinctive artifacts), deviations from prescribed workflow, friction moments (dirty tree, version skew, manual rewrites, repeated template fixes).

**Fallback when `git log` for REQUIREMENTS_DIR is empty** (the feature was implemented but never committed): reconstruct from working-tree state instead — `git status --short` for the spec dir, staged/untracked artifact set, and `verification.md` self-reports — and record the missing-commit condition itself as a process-history finding.

**Degradation when commits exist but are NOT story/task-grouped** (e.g. one bulk "implement everything" commit, or commits that mix several phases): git history cannot attribute artifacts to individual tasks. Degrade to working-tree + artifact reconstruction (task `[X]` markers, file mtimes, `verification.md` self-reports) for the per-task timeline, use the commits only for coarse ordering, and annotate every timeline claim drawn from this path with its evidence strength (e.g. "inferred from artifact state, not commit trace"). Record the missing story-grouped commit discipline itself as a Workflow finding citing the commit gate in the `/speckit.implement` command.

### 3. Load core SDD artifacts

From REQUIREMENTS_DIR: requirements.md, plan.md, tasks.md (REQUIRED). Plus data-model.md, contracts/, research.md, checklists/, feature detail (IF EXISTS). Also load constitution, templates, scripts, command files as reference targets for recommendations.

### 4. Diagnostic review — problem-first

For each artifact and workflow as a whole, find issues:
- **Friction**: Extra work forced by template/prompt/script gaps
- **Ambiguity/contradiction**: Conflicting instructions
- **Cargo-cult/boilerplate**: Irrelevant template content
- **Missing structure**: Ad-hoc prose where template fields should exist
- **Drift risk**: Same facts in multiple places without single source of truth
- **Process gaps**: Lifecycle steps lacking automation

Per finding: **ID** (F1, F2...), **Severity** (P0/P1/P2), **Category** (Template|Command Prompt|Automation|Workflow|Documentation), **Location** (absolute path or URL), **Evidence** (quoted excerpt), **Why**, **Proposed fix**.

### 4.5 Finding validation (independent subagent pass)

Every **P0** finding MUST be confirmed by an independent read-only validation subagent before it enters the report:

- The validator receives ONLY the finding (id, claim, severity, location, quoted evidence) — never the diagnostic reasoning or sibling findings — and returns `confirm` / `reject` (with why) / `downgrade` (with proposed severity).
- Only `confirm`ed findings keep P0; `downgrade`d rows take the proposed severity with a `(validated: downgraded)` note; `reject`ed rows go to an **Unvalidated Findings** appendix in the report — never silently dropped. P1/P2 skip validation.
- Do not flag: deliberate `[~]` deferrals with recorded reasons, mirror-by-design duplication under `.specify/`, or pre-existing baseline failures already recorded for the feature.

### 5. Generate report

Use `.specify/templates/review-template.md` structure. Fill all sections: Context → Timeline → Findings Summary → Findings → What Worked (brief bullets only) → Recommendations (cite target files) → Priority Roadmap. Self-containment check before writing. Write to `REQUIREMENTS_DIR/review.md`.

### 6. Report summary

REQUIREMENTS_KEY, feature name, commit SHA, report path, finding counts (P0/P1/P2 by category), top 3 recommendations.

## Position in Workflow

Use after `/speckit.implement` completes. Typical flow: feature → requirements → plan → tasks → implement → **review**.

## Feedback

At wrap-up (the same lifecycle point where this command prompts for a Git commit), perform an agent self-reflection step (never solicit feedback content from the user), following the canonical convention in `.specify/shared/workflow/feedback-step.md`:

> Note: this is a *local* self-review of the `/speckit.review` run itself, kept strictly distinct from the global project report `/speckit.review` produces.

1. **Gate on qualification & completion.** Only proceed if this command reached its wrap-up stage. Skip trivial/no-op runs; for an aborted run use the abort/partial rule below.
2. **Reflect (no user input).** Review this run against `/speckit.review`'s declared purpose and produce a short review plus ≥1 concrete, command-specific optimization point. If the run was clean, use exactly: `No significant optimization points identified this run.`
3. **Scope guard.** Keep strictly to this command's operation; do NOT produce a global/whole-project assessment (that is `/speckit.review`'s job). Entries are `scope: local`.
4. **Dedup guard.** Use a stable `run_id` (e.g. the feature key + a run timestamp); if a nested skill/command already recorded feedback for this same `(unit_id, run_id)`, the engine no-ops.
5. **Persist** via the engine:
   ```bash
   python3 "${SKILL_WORKDIR:-.}/.specify/scripts/python/feedback-utils.py" --action record \
     --unit-id "/speckit.review" --unit-type command \
     --run-id "<stable-run-id>" --feature "<feature-key-if-any>" \
     --review "<review prose>" --points-file "<points file>"
   ```
   Probe attribution: the engine resolves the unit to its probe object automatically — the entry inherits kind/slice from the probe registry. External custom units record via `--unit-id custom:<owner>/<name> --unit-type custom-unit`; their entries stay host-project-local and never enter upstream packages.
6. **Consolidated submission prompt(非阻塞).** If the returned `should_prompt` is `true`, append ONE non-blocking line to the wrap-up report inviting submission (point the user to the `/speckit.feedback package` command — the user-facing path; never paste the raw `feedback-utils.py` engine call into the user-facing line); it MUST NOT block the wrap-up flow and MUST NOT trigger any 自动传输 (manual delivery only; `--action mark-submitted` runs only if the user initiates submission). Below threshold, do not prompt.

**Abort / partial-run rule.** If the run failed before wrap-up, either skip recording or record with `--partial` and a `## Review` beginning `**Partial run** — `.

## Documentation

At the same wrap-up point as the Feedback step, apply the docs-sync evaluation per the canonical convention in `.specify/shared/workflow/docs-step.md`: assess whether information produced by this run (new capabilities, key decisions, structural changes) needs to be recorded into the project documentation space, and conclude with exactly one of `需记录（目标文档 + 要点）` or `无需记录`. Never block wrap-up; incremental judgment only (no full reconcile sweep); when a move/archive-level change is needed, recommend running `/speckit.docs` instead of executing it here.

## Handoffs

**Before**: Run after `/speckit.implement` for complete artifact chain.

**After**: Apply improvements to spec-kit .specify/templates/commands/scripts. Optionally iterate requirements/plan.