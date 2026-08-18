# Improvement-Loop Playbook

Method detail for the `improve-skills` workflow. `SKILL.md` carries the contract — step
skeleton, hard rules, decision branches; this file carries the *how*, loaded on demand.

Anchors are referenced from `SKILL.md` by step number.

---

## Step 2 — Evidence collection detail

### Degenerate-evidence fallback

When collected findings are effectively identical project-level output for every target
(byte-identical digests across skills — no per-skill signal), do **not** fabricate per-skill
defects from them. Restrict the candidate list to:

- the standing sanctioned checks (Feedback-section conformance, legacy path idioms), plus
- **filesystem ground truth** — dead links, missing referenced scripts/files.

State the evidence limitation explicitly in the final report.

### Supplementary evidence from the execution window

Gather: user feedback, steps that were confusing, tool failures, wrong assumptions, repeated
manual fixes, validation gaps, and files changed during the execution. Include terminal/test
output and error messages when they explain what went wrong. Review changed files as
evidence, but classify generated validation artifacts (e.g. `tools/*.json`) separately from
hand-edited Skill instructions.

### What to measure against the optimization goal

Whether the Skill could be invoked; whether its expected input format was accepted; whether
the workflow produced the expected output; how many avoidable manual/tool steps occurred;
whether validation caught the issue. Then name the execution-flow steps that missed:
broken command-line parameters, mismatched formats, missing prerequisites, ambiguous target
resolution, inefficient tool choices, repeated searches, unnecessary user handoffs.

Separate facts from interpretation. Do not optimize from generic best-practice principles
when no execution evidence supports the change.

### Runtime failures outrank any prior "it works" claim

A reference/example snippet that throws or silently no-ops at runtime (missing module, a loop
that toggles its own state, an extractor that skips iframes) is a **confirmed defect** even if
the file exists and a previous run asserted the Skill was fine. Anchor the fix to the observed
stack trace / wrong output, not to the earlier assertion; a stale "already works" is itself a
finding to correct.

### Silent under-extraction is a defect, not just a throw

When an executor reports a helper "ran fine" but returned *empty or thin* results (a select
with no options, a dashboard with "0 panels", `variables: []` on a page that visibly has
them), that is evidence the helper read the wrong or too-early DOM — not evidence the page is
empty. Capture the concrete observation (which selector matched 0 nodes, which content was
missing) as the reusable fact; it is often more actionable than an error message precisely
because nothing crashed. Record it even when headline metrics (coverage, screenshots) look
green.

---

## Step 3 — Analysis detail

### Fact-check gates before writing capability/data claims

Recurring defects came from authoring claims without ground truth. Run the matching gate:

- **Delegation capability** — when adding a capability to a Skill that delegates to another
  skill/tool, verify the delegate's real capability surface first (its docs / `--help` /
  contract), and encode honest limitation branches where the capability is absent. Never
  design a delegation path that must fail.
- **Data tables** — when adding or maintaining a data table (endpoints, service info,
  inventories), extract values from an existing cached fact source (meta caches, registries,
  `--help` output). Never write values from memory.
- **Tier/coverage tables** — when grading or tabulating capability coverage, reconcile the
  requested list against the tool's actual coverage (run real `--help` / command probes) and
  explicitly mark entries that are graded but not implemented. A coverage table that
  overstates capability is worse than an incomplete one.

### Deferred items must name their unlocking evidence

When an item must be deferred for lack of execution evidence (the no-fabrication discipline
blocks writing it this loop), record **which concrete evidence would unlock it** — which
scenario run, command output, or artifact would make it writable — so a later run can collect
it deliberately instead of leaving a generic "to be filled" backlog entry.

Discard one-off environment noise unless the Skill should explicitly handle it next run. A
refresh command that exits successfully with a fallback after an optional-source warning is a
validation note, not a root cause.

### Cross-skill ownership boundary

When two sibling skills own overlapping artifact types with no documented boundary (e.g.
capacity templates vs responsibility templates), the root cause is the **undocumented
boundary itself**, not any single failing step. The fix usually includes writing the boundary
down in both skills.

### Legacy path idioms

Flag these as migration candidates and apply the Migration Mapping table from
`templates/commands/skills.md` (`## Migration Mapping`):

| Legacy idiom | Rewrite as |
|--------------|-----------|
| Bare relative paths (`./scripts/init.sh`, `./references/checklist.md`) | `${SKILL_HOME}/...` |
| `${SKILL_ROOT}/X` | `${SKILL_HOME}/X` |
| Agent-specific install paths in prose (`${HOME}/.copilot/skills/<name>/...`, hard-coded `.specify/skills/<name>/...`) | `${SKILL_HOME}/...` |

### Feedback-section conformance

Verify the Skill carries a `## Feedback` section as its final workflow section, beginning with
the **runtime-mode gate** (`.specify/shared/workflow/runtime-mode.md`).

- **Missing** → append the canonical block from `.specify/shared/workflow/feedback-step.md`,
  substituting `skill:<name>` / `--unit-type skill`.
- **Malformed** → realign to the canonical block. Malformed means any of: missing runtime-mode
  gate, missing qualification/completion gate, missing no-user-input reflection rule, missing
  scope guard vs `/speckit.review`, missing stable-`run_id` dedup guard, missing
  `feedback-utils.py --action record` invocation, or missing consolidated threshold-prompt
  behavior.
- Apply the fix to **both** `skills/<name>/SKILL.md` and `.specify/skills/<name>/SKILL.md`.
- **Standalone-mode exception** — for a Skill in a standalone (non–Spec Kit) skills directory
  (no `.specify/` at the working-directory root) the engine-backed block is NOT required: a
  self-contained gated reflection section is conformant, the dual-copy rule does not apply,
  and no registry/agent propagation repair should be attempted.

---

## Step 4 — Codify deterministic logic; reserve natural language for judgment

Governing pattern: *deterministic logic → code, judgment logic → LLM.*

**Identify deterministic fragments.** Path derivation, sequence/number incrementing, state
detection, format/input validation, condition-branch decision trees, input/output transforms,
topological ordering, framework/version detection, prerequisite checks, structured parsing —
all have one correct result for a given input.

**Preset catalog + deterministic matcher.** When a skill repeatedly re-derives a whole
artifact shape (team roster, config skeleton, document layout) from vague free-form input, the
fix is a catalog of vetted presets plus a deterministic matcher mapping input signals to a
preset — not a longer prose decision tree. Presets must be distilled from real, evidenced
instances, and the matcher must be executed against sample inputs before wiring.

**Extract into a self-describing script.** Move the fragment into a shell or python script
under `${SKILL_HOME}/scripts/`. It must accept structured input (CLI arguments or a stdin JSON
payload), return structured output (JSON on stdout or an explicit exit code), and be
self-documenting (a `--help` flag, or a comment header stating purpose, inputs, outputs).

**Reference the script from `SKILL.md`.** Replace the prose describing the logic with a
script-invocation instruction — "run `${SKILL_HOME}/scripts/detect-framework.sh` and branch on
its JSON output" — instead of restating the steps in words.

**Keep judgment logic in natural language.** Option trade-offs, quality review, intent
understanding, and ambiguity resolution stay as LLM-directed prose.

**Apply only when it pays off.** Extract when the logic meets *any* complexity signal:
conditional branching, multi-step sequential operations with intermediate state, parsing or
transformation of structured data, or error-prone when restated in natural language (regex,
path arithmetic, version comparisons). Line count alone does not indicate complexity — a
one-line regex validation may warrant extraction while ten lines of straightforward
enumeration may not. The logic should also recur across executions or across Skills; truly
one-off trivial checks can stay inline.

---

## Step 5 — Rename/removal downstream-wiring checklist

When an improvement renames, consolidates, or removes a skill, the edit is not done until
every downstream pointer moves with it. In this repo:

1. Add the old name to `_OBSOLETE_SKILLS` in `src/specify_cli/__init__.py`, extending the
   rename-chain comment.
2. Rename/realign the skill's contract-test file and its assertions, including guards that the
   old name is gone (directory absent, obsolete-manifest entry, no second directory carrying
   the same frontmatter `name`).
3. Update the skills-count list in the instructions file's Key Directories section (no
   registry row exists — discovery is by directory).
4. Add a feature-history entry recording the rename and rationale.
5. Fix stale pointers in artifacts the old skill dogfooded (e.g. report headers naming the
   predecessor skill).

Sync the `skills/<name>/` ↔ `.specify/skills/<name>/` mirror with `\cp -rf` (plain `cp` may be
aliased to `cp -i` and silently skip overwrites) and verify byte-equivalence with `diff -rq`.

**Move-then-edit order**: when a rename uses `git mv`, re-read the file at its NEW path before
editing — file-editing tools reject writes to paths not yet read in-session, so edits aimed at
the old path land nowhere.

---

## Step 6 — Zero-regression proof on a red suite

When the test suite has pre-existing failures, "the same tests still fail" eyeballed from
counts is not sufficient. Prove zero regression with a **failure-set diff**:

1. Capture the sorted `FAILED|ERROR` lines of the full run.
2. Produce a clean baseline of `HEAD` and rerun the same suite there.
3. Diff the two failure sets — an identical set means your change introduced no regression.

Prefer `git worktree add <tmp> HEAD` for the clean baseline: it is read-only with respect to
the working tree and immune to concurrent writers (a running continuous team writing into
`.specify/teams/` mid-run can block `git stash pop` and strand a stash entry). Use
`git stash -u` + rerun + `git stash pop` only when worktrees are unavailable.

If a combined validation command returns only partial output or omits later checks, rerun the
missing checks individually before concluding validation passed.

---

## Input Contract — batch mode detail

### Running a batch (explicit multi-skill request)

When the user explicitly names a *set* of Skills (or a whole skills directory), do not force
single-target resolution and do not loop the full workflow N times blindly. Run one batch pass:

1. Collect evidence **once** for the batch, then derive per-skill candidates.
2. **Freeze the per-skill candidate list** before editing — no mid-batch additions.
3. Prefer **mechanical, sanctioned conformance edits applied directly** (Feedback-section
   conformance, legacy path idioms, dead links, missing script refs) over speculative per-skill
   rewrites.
4. Verify mirror/write-through **once at the end** for the whole batch (e.g. `diff -rq`).

A batch that repeatedly fails on speculative rewrites should be restarted in this minimal-edit
shape rather than retried harder.

### Batch triage ownership

A unit naming a Skill absent from *this* repo is not automatically out of scope. First resolve
where the Skill actually lives — host skills dirs (`~/.qoder/skills/`), sibling project roots'
`.specify/skills/`, hardlink/write-through copies — and process it *there* when the user owns
that location (e.g. the framework developer handling cross-project feedback). Only mark a unit
out of scope after its install location is unknown or owned elsewhere.

### Intent routing guard

When the intent is to **monitor or evaluate an ongoing execution process** ("continuously
watch/score another agent's work") rather than to modify a specific Skill, this skill is the
wrong entry point: editing a SKILL.md conflicts with monitoring red lines (zero writes to the
monitored target). Route to a `continuous` monitoring team (`create-team` / `improve-team`) and
tell the user why, instead of forcing target-Skill resolution.
