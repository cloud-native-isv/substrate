<!-- AUTO-GENERATED from templates/commands/interview.md — do not edit; edit the source template, then run scripts/python/regen-command-copies.py -->
## User Input

```text
$ARGUMENTS
```

Process `$ARGUMENTS` per the [User Input Protocol](.specify/shared/workflow/user-input-protocol.md). Treat as command parameters, not standalone instructions.

## Positioning

`/speckit.interview` is the standalone, general-purpose entry for **interview mode**. The mode itself — design tree, frontier, rounds, question format, fact-vs-decision split, write-through, ledger, exit gate, anti-patterns — is defined once in `.specify/shared/patterns/interview-pattern.md`. **Read that document and follow it; never restate or re-derive its rules here.** This command owns only the *invocation*: resolving the target artifact, resolving the ledger, and running the loop to the exit gate.

Use it when the information needed lives in the user's head and the decisions branch. It is **not** a clarification pass with a question cap (`/speckit.clarify`), not a design generator (`/speckit.plan`), and not an artifact-space converger (`/speckit.docs`, `/speckit.feature`).

## Glossary

Consult `.specify/memory/glossary.md` and apply `.specify/shared/workflow/glossary.md`: map recorded homophone/confusable variants to canonical terms before acting, surfacing each correction. Vocabulary discipline matters more here than in most commands — an interview records the user's own words into a durable artifact, so a misheard proper noun propagates. At wrap-up propose new project-specific terms (`origin=auto`, `status=proposed`) with user confirmation.

## Interview Mode

This command runs the pattern unmodified. Its four required declarations:

| Declaration | This command's value |
|-------------|----------------------|
| **Target artifact** | Resolved in step 2 below — an existing artifact named by the user, or a new interview record |
| **Ledger** | In a feature context `<REQUIREMENTS_DIR>/interview-log.md`; otherwise `.specify/memory/session/interview-<topic-slug>.md` |
| **Exit gate** | Frontier empty → present the consolidated understanding → **explicit user confirmation**; never act on the design before it |
| **Round budget** | **Unbounded.** This is the escalation target for capped flows; it stops when the frontier empties or the user stops it |

## Outline

1. **Resolve context**: run `.specify/scripts/bash/check-prerequisites.sh --json --paths-only` from repo root once; parse JSON for `REPO_ROOT`, `BRANCH`, `REQUIREMENT_ID`, `REQUIREMENTS_DIR`, `FEATURE_SPEC`, `IMPL_PLAN`, `TASKS`. A non-feature branch is not an error — it selects the non-feature ledger location.

2. **Resolve the target artifact** (what converges), in this precedence:
   - a path or artifact named in `$ARGUMENTS` → that file;
   - otherwise, in a feature context, the current phase's primary artifact (`tasks.md` → `plan.md` → `requirements.md`, first that exists);
   - otherwise a new interview record beside the ledger.

   State the resolved target back to the user before asking anything. If the user meant a different one, they will say so now rather than after ten rounds.

3. **Writability probe (fail fast)**: verify the target artifact and its directory are writable before generating any questions. If unwritable, STOP with the owning path and a fix command (e.g. `sudo chown -R $USER <dir>`) — never run the loop first and lose the answers at the final write.

4. **Resolve the ledger and resume**: per the table above. If a ledger exists, read its header conventions — **never renegotiate them** — and resume from the first unsettled branch. Report to the user what is already settled, so they are not re-asked.

5. **Load context summary-first** (see `.specify/shared/guidelines/token-efficiency.md`): read the target artifact (it is the edit target, so read it fully); pull `.specify/memory/features.md` as index rows only; take `.specify/memory/constitution.md`, `research.md`, and `docs/` as **targeted excerpts only when a candidate question actually needs them**. Never bulk-load context "to be safe" — an interview's context grows every round.

6. **Run the loop I0–I5** per `.specify/shared/patterns/interview-pattern.md`. Two obligations that are easy to skip and expensive to miss:
   - **I3** — dispatch a subagent for any environment fact a question needs; do not block the round on it, and do not ask the user for it.
   - **I5** — write through to the target artifact **before** asking the next round, overwrite-style (latest round wins), and update the ledger row.

7. **Exit gate**: when the frontier is empty, present the consolidated understanding (decisions, what each changed, what was deferred) and ask for confirmation. On confirmation, hand off to the next command per the handoff guidance at the end of this document. If the user reopens a branch, it re-enters the tree and the loop continues — an interview is done when the *user* says the result is stable, not when the agent runs out of questions.

## Report

- Target artifact path and ledger path
- Rounds run, questions asked / answered / deferred
- Branches settled vs. deferred (⏭ with reasons)
- Sections of the target artifact touched
- Facts resolved by probe rather than by asking (evidence that the fact/decision split held)
- Suggested next command

## Behavior Rules

- **Never fabricate an answer.** An unasked branch stays open in the ledger. This is the one unrecoverable failure of the mode.
- **Never ask for a discoverable fact.** Look it up or probe for it.
- **Never proceed on silence** and never self-certify completion — the exit gate is the user's confirmation.
- **Stop only at round boundaries.** On interruption or a turn/time limit, finish the current round's write-through, leave the ledger complete, and report where to resume.
- **No live user** (fire-and-forget subagent invocation): process only the batch already answered in the ledger and return. Never advance the frontier without answers.
- **Respect termination signals** ("stop", "done", "enough"): close the current round, write through, record the remaining frontier as deferred, and report — a stopped interview must still leave a usable artifact.
- **Stay on the resolved target.** If answers imply changes to a different artifact, record the implication and surface it at the exit gate; do not silently edit artifacts outside the resolved target.

## Feedback

At wrap-up (the same lifecycle point where this command prompts for a Git commit), perform an agent self-reflection step (never solicit feedback content from the user), following the canonical convention in `.specify/shared/workflow/feedback-step.md`:

1. **Gate on qualification & completion.** Only proceed if this command reached its wrap-up stage. Skip trivial/no-op runs; for an aborted run use the abort/partial rule below.
2. **Reflect (no user input).** Review this run against `/speckit.interview`'s declared purpose and produce a short review plus ≥1 concrete, command-specific optimization point. Interview-specific signals worth reflecting on: questions the user answered with "you could have looked that up" (fact/decision split leaked), rounds that asked a blocked question, and branches that had to be re-asked because a write-through was skipped. If the run was clean, use exactly: `No significant optimization points identified this run.`
3. **Scope guard.** Keep strictly to this command's operation; do NOT produce a global/whole-project assessment (that is `/speckit.review`'s job). Entries are `scope: local`.
4. **Dedup guard.** Use a stable `run_id` (e.g. the target-artifact key + a run timestamp); if a nested skill/command already recorded feedback for this same `(unit_id, run_id)`, the engine no-ops.
5. **Persist** via the engine:
   ```bash
   python3 "${SKILL_WORKDIR:-.}/.specify/scripts/python/feedback-utils.py" --action record \
     --unit-id "/speckit.interview" --unit-type command \
     --run-id "<stable-run-id>" --feature "<feature-key-if-any>" \
     --review "<review prose>" --points-file "<points file>"
   ```
6. **Consolidated submission prompt.** If the returned `should_prompt` is `true`, surface a single consolidated prompt inviting the user to submit collected feedback to the Spec Kit developers; on confirmation run `--action mark-submitted`. Below threshold, do not prompt.

**Abort / partial-run rule.** If the run failed before wrap-up, either skip recording or record with `--partial` and a `## Review` beginning `**Partial run** — `.

## Documentation

At the same wrap-up point as the Feedback step, apply the docs-sync evaluation per the canonical convention in `.specify/shared/workflow/docs-step.md`: assess whether information produced by this run (new capabilities, key decisions, structural changes) needs to be recorded into the project documentation space, and conclude with exactly one of `需记录（目标文档 + 要点）` or `无需记录`. Never block wrap-up; incremental judgment only (no full reconcile sweep); when a move/archive-level change is needed, recommend running `/speckit.docs` instead of executing it here.

## Handoffs

**Before**: none — an interview may run at any time, on any artifact. In a feature context it is most useful when `/speckit.clarify` hits its question cap with the decision space still branching.

**After**: determined by what converged — a converged `requirements.md` → `/speckit.plan`; a converged `plan.md` → `/speckit.tasks`; a converged `tasks.md` → `/speckit.implement`; a converged goal → `/speckit.goal` or `/speckit.team`; anything else → the command that owns the target artifact.