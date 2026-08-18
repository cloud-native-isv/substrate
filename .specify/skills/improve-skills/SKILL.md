---
name: improve-skills
description: This skill continuously improves one local Skill from a user-provided Skill description, execution history, user feedback, failure cases, and observed inefficiencies. Use this when the user mentions ["improve skills after use", "skill execution feedback", "refine SKILL.md", "skill retrospective", "skill iteration", "技能执行反馈", "基于执行问题优化skill", "持续改进Skill"]
skill_id: "<SKILL:.specify/skills/improve-skills/SKILL.md>"
---

# improve-skills

## Goal

Continuously improve one existing local SpecKit Skill from a user-provided Skill description and evidence from real executions. The expected result is a focused Skill update that fixes observed problems, captures reusable lessons, and makes the next execution more reliable.

Goal anchor (Constitution Principle XIII): this skill is a Better-Harness instrument — improving a Skill strengthens the **Controlled Execution** dimension (the supported, repeatable path the Skill provides) and closes the **Learning Capture** loop; goal model in `.specify/shared/guidelines/better-harness.md`.

## Input Contract

The input is a description of the Skill to improve. It must be interpreted as follows:

- **Target identifier**: identify exactly one local Skill by `skill_id`, frontmatter `name`, canonical path, or Skill directory name. If multiple Skills match or no local Skill can be found, ask one targeted clarification before editing.
- **Optimization direction**: extract the requested direction when present (execution failures, efficiency, input/output clarity, tool usage, validation strength). If absent, infer it only from concrete execution history and user feedback.
- **User emphasis**: treat details in the user's description as high-priority evidence. Analyze them explicitly even when broader execution history suggests additional improvements.
- **Batch mode** (user names a *set* of Skills or a whole directory): run **one** batch pass — collect evidence once, freeze the per-skill candidate list, prefer mechanical sanctioned conformance edits over speculative rewrites, verify mirrors once at the end. Do not loop the full workflow N times blindly.
- **Batch triage ownership**: a Skill absent from *this* repo is not automatically out of scope — resolve where it actually lives and process it there when the user owns that location.
- **Intent routing guard**: if the intent is to *monitor/score an ongoing execution* rather than modify a Skill, this is the wrong entry point (editing conflicts with monitoring's zero-write red line) — route to a `continuous` team and say why.

Batch procedure, ownership resolution, and routing rationale: [`./references/loop-playbook.md`](./references/loop-playbook.md) `## Input Contract`.

## Workflow

1. **Identify the target Skill, optimization direction, and execution window**
   - Resolve the description to exactly one local Skill (`skill_id` / frontmatter `name` / canonical path / directory name). For “this Skill”, infer from the active file or recent conversation, then verify it resolves uniquely.
   - Treat `.specify/skills/<name>/SKILL.md` as the canonical source of truth; `.github/skills/<name>` is only a compatibility entrypoint. **Re-read the canonical file before editing** — a user edit, formatter, or refresh script may have changed it.
   - Carry the optimization direction through evidence, analysis, edits, validation, and reporting.
   - Define the execution window: current conversation, last run, failed command output, user correction, test failure, or recent edits.
   - When improving `improve-skills` itself, use the most recent improvement loop as the window and do not reapply a lesson already applied unless new evidence shows the previous fix was insufficient.

2. **Measure execution effectiveness from evidence (evidence-step A/B)**
   - Execute Step A/B per `.specify/shared/workflow/evidence-step.md` (single source of truth; do not restate its rules here): reuse fresh findings via `evidence-utils.py --action latest --target skill:<name>`, or collect via `--action collect --target skill:<name> --lanes all` (session + feedback lanes carry the strongest signals for skill improvement: episode-level rework/failure evidence and recurring optimization themes with `recurrence` signals).
   - Triage findings evidence by `evidenceState` per the evidence-step table, then **freeze the candidate list** — later steps must not add or drop candidates. Red lines: `Unobserved` items are recorded only and MUST NOT be treated as defects to fix; counting signals MUST NOT directly generate optimization points.
   - Gather supplementary evidence from the execution window and measure the Skill against the optimization goal; separate facts from interpretation, and never optimize from generic best practice with no supporting evidence. What to collect and what to measure: [`./references/loop-playbook.md`](./references/loop-playbook.md) `## Step 2`.
   - **Runtime failures are first-class evidence and outrank any prior "it works" claim** — including *silent under-extraction* (a helper that "ran fine" but returned empty/thin results is a defect, not proof the source is empty). Both cases, with what to capture: same reference, `## Step 2`.
   - **Degenerate evidence** (byte-identical digests across targets = no per-skill signal): do NOT fabricate per-skill defects; fall back to the sanctioned checks plus filesystem ground truth, and state the limitation in the report. Detail: same reference.
   - If evidence is insufficient, ask one targeted question about what failed, what was inefficient, or what should happen differently next time.

3. **Analyze user-provided emphasis and organize improvement items**
   - Give the user's stated optimization direction a dedicated analysis pass: confirm which parts are already satisfied, which parts are missing, and which edits will directly address the request.
   - Group observations by failure mode: trigger/discovery, scope inference, missing context, wrong tool choice, unsafe step, unclear output, validation gap, resource/reference issue, **constraint non-compliance** (workflow followed but a hard rule violated — diagnose as a constraint-*placement* problem per [`./references/constraint-placement.md`](./references/constraint-placement.md) before rewording the rule), or **cross-skill ownership boundary** (the undocumented boundary is the root cause, not any single step).
   - For each item, record: observed symptom, likely cause in the current Skill instructions, desired next behavior, and the file section to change.
   - **Run the matching fact-check gate before writing any capability or data claim** (delegation capability / data tables / tier-coverage tables) — verify the real surface, never write values or capabilities from memory. Gates: [`./references/loop-playbook.md`](./references/loop-playbook.md) `## Step 3`.
   - When an item must be **deferred for lack of evidence**, record which concrete evidence would unlock it. Discard one-off environment noise unless the Skill should handle it next run. Detail: same reference.
   - **Legacy path idioms** (bare relative paths, `${SKILL_ROOT}/X`, agent-specific install paths in prose) → rewrite as `${SKILL_HOME}/...`; mapping table in the same reference.
   - **Feedback-section conformance**: verify the Skill's final workflow section is `## Feedback` opening with the runtime-mode gate; repair if missing or malformed, and apply to both mirrors. Malformed-criteria list, canonical-block source, and the standalone-mode exception: same reference, `### Feedback-section conformance`.

4. **Correct the root causes with minimal changes**
   - Fix the instruction that caused the observed failure first (wrong arguments, nonexistent paths, invalid expected formats, incompatible metadata, missing prerequisite checks). For inefficiency, replace the inefficient step with a more direct method, deterministic script, or clearer decision branch.
   - Prefer changing the step that caused the problem over adding broad new rules. Convert repeated user corrections into explicit decision branches, and repeated manual checks into checklist items or scripts.
   - **Harden reference helpers that read a live third-party/framework DOM.** Two causes recur: *interaction-gated content* (options/tabs/popovers that mount only on click) and *version-drifting selectors* (renamed classes/`data-testid`s silently matching 0 nodes). Prefer targeted robustness edits over widening the extractor. Symptoms, before/after code, and the reusable checklist: [`./references/hardening-examples.md`](./references/hardening-examples.md).
   - **Prefer ground-truth signals over convenient-but-misleading ones, and scope extraction to the intended node.** A count/label the target system prints is often stale or collapsed-state — derive the value from the underlying elements, and when they disagree the elements win. A container selector silently absorbs sibling text: re-scope to the leaf and verify each captured field is the datum the doc claims. Details and worked cases: same reference, `## The reusable checklist`.
   - **For constraint non-compliance, fix placement before wording** — apply the evidence-based fix ladder in [`./references/constraint-placement.md`](./references/constraint-placement.md): consolidate hard constraints into one compact block positioned late in `SKILL.md`, restate the concrete rule text inline at the violating decision-point step, de-duplicate copies, pair prohibitions with the required alternative, and make completion conditions objective. Never accept a generic "strictly follow the constraints" reminder as the fix (measured no-op), and do not duplicate the block into multiple sections (measured to reduce compliance).
   - Move detailed lessons to `./references/` only when they are useful but not needed every run.
   - **Keep `SKILL.md` a contract — for content you REMOVE and for content you ADD.** Run `python3 ${SKILL_HOME}/scripts/skill-shape.py <SKILL.md>` and treat its verdict as the objective condition. Two directions, both mandatory:
     - *Authoring direction (new content)*: detail you write in this loop defaults to `./references/` (explanations, worked examples, checklists) or `./scripts/` (deterministic logic). `SKILL.md` may gain only contract lines — a step heading with its one-sentence goal, a hard rule, a decision branch, a resource-index row, a pointer with an anchor. A full command sequence, a worked example, or a multi-line snippet belongs in L2/L3 even when it is brand-new and correct.
     - *Cleanup direction (existing content)*: move already-present manual content out per [`./references/skill-slimming-principles.md`](./references/skill-slimming-principles.md), always **delete-and-absorb** (copy into the target reference in the same edit), never delete-and-drop.
     - **Never slim a section that a feature spec or contract test mandates inline.** Before moving/removing any named heading, grep `.specify/specs/**` and `tests/contract/**` for it; if a contract asserts its inline presence (e.g. `## Agent-Specific Configuration` with `### Step 1/2/3` per `021-agent-specific-config` C-002), keep it inline and slim elsewhere. Heading-visible structural moves are the highest-risk slimming edits.
     - Defer environment-level recovery (auto-install, shell switching, OS branching) to the user — surface the error and the fix command, then stop.
   - **Codify deterministic logic; reserve natural language for judgment.** Scan the Skill for deterministic logic still expressed as prose (path derivation, state detection, validation, decision trees, transforms, topological ordering) and extract it into a self-describing script under `${SKILL_HOME}/scripts/` with structured input/output; replace the prose with a script-invocation instruction. Judgment logic (trade-offs, intent, ambiguity) stays prose. Extract only when it recurs and meets a complexity signal — line count alone is not one. Identification criteria, the preset-catalog pattern, and script requirements: [`./references/loop-playbook.md`](./references/loop-playbook.md) `## Step 4`.

5. **Update the Skill for the next execution**
   - Edit `SKILL.md` to make the improved behavior executable and checkable.
   - Update frontmatter `description` only when execution feedback shows trigger/discovery mismatch.
   - Update `./references/`, `./scripts/`, or `./assets/` only when the evidence shows they will reduce future mistakes.
   - When Step 4 extracted deterministic logic into a new `${SKILL_HOME}/scripts/` script, list that script in the Resources table so the executable resource stays discoverable.
   - Avoid adding process logs, changelogs, or full retrospectives to the Skill; distill only reusable lessons.
   - **Never narrate the document's own history**: corrections state the current fact only — never `旧文档断言…已证伪` / `重要勘误` / `本条不再是限制` frames (exceptions: confidence markers, misuse guards). Rewrite examples: [`./references/skill-slimming-principles.md`](./references/skill-slimming-principles.md) `## No History Narration`.
   - **Rename/removal downstream-wiring checklist**: a rename/consolidation/removal is not done until every downstream pointer moves with it (obsolete-skills list, contract tests, skills-count list, feature history, dogfooded artifacts), and both mirrors are synced and verified byte-equal. Five-step checklist and the move-then-edit ordering rule: [`./references/loop-playbook.md`](./references/loop-playbook.md) `## Step 5`.

6. **Validate the improvement loop**
   - Re-read the changed Skill and verify that each edit maps to an observed execution issue.
   - **Run the shape gate on every `SKILL.md` you touched**: `python3 ${SKILL_HOME}/scripts/skill-shape.py <SKILL.md>`. Exit `0` = contract-shaped; exit `10` = blocking findings. A run that ends with the target still at exit `10` MUST either slim until it passes or state the explicit reason it cannot (a contract-mandated inline section, or a deliberate exception recorded in the report). Never finish an improvement loop having grown a body past the gate without saying so — that is the defect this gate exists to catch.
   - **When an edit touches reference code or example snippets, validate that the code actually RUNS — not merely that the file exists or parses.** Execute it (or the smallest reproducing harness) against a real target, or trace it line-by-line against the documented API to confirm the control flow does what the prose claims. Files-exist / links-resolve checks do not catch a snippet that throws, loops incorrectly, or no-ops; those surface only at runtime. If you cannot run it this loop, say so and mark it as needing runtime validation in the next real execution rather than reporting it as verified.
   - Check frontmatter, resource paths, line count, compatibility entry, and discoverability when metadata changed. If `skill_id` is added or corrected, ensure the directory name and frontmatter `name` agree and no other skill directory carries the same `name` (no registration table — see `.specify/skills.md`).
   - Accept a directory-level `.github/skills -> ../.specify/skills` symlink as a valid compatibility entrypoint; do not require a separate per-Skill symlink when the directory symlink already exposes the Skill.
   - **After any structural edit to a SKILL.md or its references (moving/renaming/removing a section or file), run the affected contract tests** (`tests/contract/` for the feature that governs the skill) — not just grep for headings. A slimming move can silently break a heading-presence contract test; only executing the tests proves it still passes.
   - **When the suite has pre-existing failures, prove zero regression with a clean-baseline failure-set diff** — capture the sorted `FAILED|ERROR` set, rerun the same suite on a clean `HEAD` baseline (prefer `git worktree`, not `git stash`), and diff the two sets. Counts eyeballed from a red suite are not proof. Method: [`./references/loop-playbook.md`](./references/loop-playbook.md) `## Step 6`.
   - **Behavior-changing edits get a pressure re-test (RED-GREEN)**: when the edit changes what the Skill *permits or forbids* (new MUST/MUST NOT, gate, or workflow constraint), re-run the pressure scenario that motivated it — a fresh subagent WITHOUT the edit reproduces the failure, WITH the edit complies. Method: `create-skills/references/pressure-testing.md`. Wording-only or resource-path edits are exempt; state which case applied.
   - Do not document `.specify/scripts/` as a Skill-owned resource directory; Skill-owned executable resources belong in `./scripts/`.
   - **Intervention ledger (evidence-step Step E)**: when the run consumed findings evidence, write `intervention.json` into the baseline evidence-run directory (targetFinding / change / baselineRunId / expectedSignal). The next same-target run's `--action compare` decides `Outcome-supported` vs `Unobserved`; never claim "fixed" without that before/after comparison.

7. **Report the feedback-driven changes**
   - Summarize the driving evidence, list changed files and the behavior expected to improve, and note unresolved feedback that needs another real execution to validate.

## Quality Checklist

Use [the Skill quality checklist](./references/skill-quality-checklist.md) to structure execution feedback, root-cause analysis, and validation when the improvement involves more than one observed issue.

## Resource ID

- Canonical ID: `<SKILL:.specify/skills/improve-skills/SKILL.md>`
- Canonical Path: `.specify/skills/improve-skills/SKILL.md`

## Agent-Specific Configuration

### Step 1: Identify Executing Agent

Before executing this skill's workflow, identify which AI agent you are:

| Agent | Detection Signals |
|-------|-------------------|
| **Claude Code** | System prompt contains "Claude Code"; tools include `Agent`, `Edit`, `Bash`, `Read`; `.claude/` directory exists |
| **GitHub Copilot** | Running in VS Code Copilot Chat context; `.github/copilot-instructions.md` loaded; tools include `workspace edit`, `@terminal` |
| **Qoder CLI** | `.qoder/` directory exists; `AGENTS.md` instructions loaded |
| **opencode** | `.opencode/` directory exists |
| **Codex CLI** | `.codex/` directory exists |
| **Hermes Agent** | `.hermes/` directory exists |

If you cannot identify your agent, skip Step 2 and proceed with the standard workflow.

### Step 2: Load Agent-Specific Guidance

If you identified your agent in Step 1, check if a guide exists at:

```
${SKILL_HOME}/references/<agent-slug>-guide.md
```

Where `<agent-slug>` is: `claude-code`, `copilot`, `qoder`, `opencode`, `codex`, `hermes`.

If the guide exists, read it and apply the agent-specific tool mappings, best practices, and pitfall avoidances during execution. If no guide exists for your agent, proceed with the standard workflow.

### Step 3: Capture Execution Feedback

If you encounter an agent-specific obstacle during execution (e.g., a tool call is unavailable, output format doesn't match expectations, a workaround was needed), generate a feedback document at:

```
.specify/memory/feedback/improve-skills-<agent-slug>-<YYYY-MM-DDTHH-MM-SS>.md
```

The feedback document MUST contain:

```markdown
# Agent Execution Feedback

**Source**: improve-skills
**Agent**: <agent-slug>
**Timestamp**: <ISO-8601>
**Outcome**: <success-with-workaround | partial-failure | full-failure>

## Obstacle
[Description of the agent-specific issue encountered]

## Workaround Applied
[What was done to work around the issue, if anything]

## Suggested Improvement
[Specific change to the skill or reference document that would prevent this issue]
```

Only generate feedback when a genuine agent-specific obstacle was encountered.

## Resources

| Directory | Contents |
|-----------|----------|
| `${SKILL_HOME}/references/` | `skill-slimming-principles.md`, `loop-playbook.md`, `skill-quality-checklist.md`, `hardening-examples.md`, `constraint-placement.md`, `claude-code-guide.md`, `copilot-guide.md` |
| `${SKILL_HOME}/scripts/` | `skill-shape.py` — deterministic L1 shape gate for a `SKILL.md` (token budget, fence ratio, long fences, example sections); exit `0` pass / `10` slim-recommended; `--help` for thresholds and calibration |

## Hard Constraints

Objective conditions for finishing a loop. Each is checkable, not a matter of judgement.

1. **Contract shape is gated, not estimated.** Every `SKILL.md` touched this loop must end at `skill-shape.py` exit `0`, or the report must name the specific reason it cannot (contract-mandated inline section / recorded exception). Do not substitute an impression that the file "looks fine".
2. **New detail lands in L2/L3 by default.** Do not add worked examples, full command sequences, or multi-line snippets to `SKILL.md` — write them into `./references/` or `./scripts/` and leave a one-line pointer with an anchor in the body.
3. **Evidence before defect.** Do not label anything a defect without an observed symptom (error text, wrong output, user correction, artifact). `Unobserved` findings are recorded only; counting signals alone never becomes an optimization point.
4. **Capability verified before it is documented.** Before writing a delegation path, data table, or coverage claim, read the delegate's real surface (`--help`, contract, cached facts) and encode honest limitation branches. Do not write values or capabilities from memory.
5. **Reference code is executed, not eyeballed.** Any snippet or script added/changed must be run (or traced line-by-line against the documented API). "File exists" and "links resolve" are not validation — state explicitly when runtime validation is deferred. For detection scripts also run a **reverse case on a domain-legitimate sample** (must NOT flag required syntax) — catching bad samples alone can induce deleting required syntax.
6. **Removal preserves content.** Every slimming move is delete-and-absorb in the same edit; never delete a section and defer relocating its substance.
7. **No claim of "fixed" without before/after.** Improvement outcomes are decided by the intervention ledger's next-run comparison, not by asserting the edit works.
8. **Wrap-up commits verify the staging area.** Before any loop-end commit, `git status --short` and confirm only this loop's files are staged; unstage unrelated pre-staged entries or commit by explicit pathspec — never `git add -A`.

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
     --unit-id "skill:improve-skills" --unit-type skill \
     --run-id "<stable-run-id>" --feature "<feature-key-if-any>" \
     --review "<review prose>" --points-file "<points file>"
   ```
   Probe attribution: the engine resolves the unit to its probe object automatically — the entry inherits kind/slice from the probe registry. External custom units record via `--unit-id custom:<owner>/<name> --unit-type custom-unit`; their entries stay host-project-local and never enter upstream packages.
6. **Consolidated submission prompt.** If the returned `should_prompt` is `true`, surface a single consolidated prompt inviting the user to submit collected feedback to the Spec Kit developers; on confirmation run `--action mark-submitted`. Below threshold, do not prompt.

**Abort / partial-run rule.** If the run failed before wrap-up, either skip recording or record with `--partial` and a `## Review` beginning `**Partial run** — `.
