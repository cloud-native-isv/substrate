# Execution Guide

Detailed reference for `/speckit.team run` runtime discipline: workspace layout, report contract details, shared protocols, and sub-agent conventions.

---

## Run Workspace, Reports & Output Discipline

Every team run produces files in **four distinct classes**. Keep them strictly separated — this is what makes runs reproducible, the team directory clean, and later skill/command optimization possible.

| Class | Location | Git | What goes here |
|-------|----------|-----|----------------|
| **Team definition** | `.specify/teams/<slug>/team.md` | tracked | The persisted team (frontmatter + Goal/Static/Dynamic sections). |
| **Run reports** | `.specify/teams/<slug>/runs/<UTC-timestamp>-report.md` | tracked, **accumulate** | One report per execution; filename carries the date. |
| **Deliverables (standard output)** | the **declared target path** (a real project path the user/goal specifies) | tracked | The team's actual product — final artifacts only. **Never** the team directory. |
| **Run intermediates** | `.specify/teams/.work/<slug>/` | **git-ignored** | Everything else, any filename: progress files, parallel status manifests, per-iteration candidate renders, evaluator score dumps, team working memory, executor/optimizer scratch, intermediate serial-stage handoff files. |

Rules:

- The team directory `.specify/teams/<slug>/` holds the team's own tracked run information — `team.md`, `runs/`, the append-only item ledger `items.jsonl`, and (continuous only) `constraints.md` / `STATE.md` / `run-log.jsonl`. No intermediate files, no deliverables, and **no summary artifacts**: the summary is a derived product that belongs to the *goal* index at `.specify/goal/<goal-slug>/summary/` (see `references/summary-mapping.md`).
- **Only final deliverables** count as standard output and escape to real target paths. Intermediate handoff files between serial stages are run intermediates → `.specify/teams/.work/<slug>/` (file-path-only handoff still works: downstream stages read from the workspace).
- The run workspace is created on demand by the orchestrator at run time; it is transient and safe to delete. Do not rely on it across runs — durable knowledge belongs in the tracked report.
- Token efficiency (see `.specify/shared/guidelines/token-efficiency.md`): agent prompts and stage handoffs carry digests/paths, never whole machine-managed data files; deterministic checks (counting, diff, pattern match) run as program steps, not LLM judgments — validate this when persisting a team.

---

## Shared Protocols

### File Handshake Protocol

All patterns use **file-path-only** communication:
- Agents write deliverables to designated paths
- Downstream agents receive ONLY file paths (not content)
- Never paste file content between agents — saves 50%+ tokens

### Progress Tracking

- Parallel: manifest files at `.specify/teams/.work/<slug>/parallel-result-<agent-id>.md`
- Serial: progress file at `.specify/teams/.work/<slug>/progress.md`
- Iteration: iteration history in the run workspace; final summary in the tracked run report
- Continuous: cross-run `STATE.md` + append-only `run-log.jsonl` in the team directory; per-cycle report under `runs/` (see `references/operating-loops.md`)

### Structured Returns from Loop Sub-Agents

In iteration/continuous loops, every dispatched sub-agent (optimizer, executor/renderer, scorer/evaluator) MUST finish by writing a small **structured result manifest** into the run workspace (e.g. `.specify/teams/.work/<slug>/gen-<N>/<role>-result.md` or `.json`), containing: `status` (done/failed), `output_paths` (file-path-only, no content), per-dimension scores (evaluators), and the single biggest improvement point observed.

- **Sub-agents never write tracked team files.** `runs/<ts>-report.md`, `team.md`, `STATE.md`, and `run-log.jsonl` are written **only by the Team Supervisor (orchestrator)**, which aggregates the manifests. Sub-agents run in isolated contexts — concurrent writes to tracked files race, a sub-agent cannot see sibling variants to aggregate them, and a partial write corrupts the durable record.
- **Interruption recovery.** If a sub-agent dispatch is cut off mid-flight (connection dropped, tool failure), first check whether its result manifest exists. **A missing manifest means the work is incomplete**, regardless of how many files the agent already wrote. Recover by re-dispatching with a resume prompt that (a) lists the changes already found on disk and (b) instructs the agent to verify those landed changes first, then complete only the remainder — this keeps recovery idempotent instead of redoing or double-applying work.
- The supervisor **validates each manifest before DECIDE**; a missing or empty manifest counts that variant/cycle as failed (it does not silently score zero-quality work as zero points).
- The scored deliverable path in the manifest is what the supervisor passes to the evaluator — evaluators read the artifact from that path, never from pasted content.

### Model Selection Guidance

| Sub-task Type | Examples | Recommended Tier |
|---------------|----------|-----------------|
| **Deterministic** | Template filling, format conversion | Light (fast, cheap) |
| **Judgment** | Code review, scoring, standard implementation | Standard |
| **Deep Synthesis** | Architecture design, novel algorithms | Heavy (high capability) |
