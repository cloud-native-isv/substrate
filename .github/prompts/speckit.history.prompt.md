<!-- AUTO-GENERATED from templates/commands/history.md — do not edit; edit the source template, then run scripts/python/regen-command-copies.py -->
## User Input

```text
$ARGUMENTS
```

Process `$ARGUMENTS` per the [User Input Protocol](.specify/shared/workflow/user-input-protocol.md). Treat as command parameters, not standalone instructions. Recognized optional parameters:

- `--full` — reprocess **all** sessions and regenerate the knowledge base (ignore the incremental manifest).
- A theme focus, date range, or topic hint — narrow which sessions to emphasize.

## Outline

`/speckit.history` distills the **current AI tool's** past conversations **for the current project** into a durable, theme-aggregated knowledge base at `.specify/history/`.

**Scope & unit of work** — what makes this command distinct from `/speckit.session`:

- **Scope = ALL history**: it reads the tool's *entire* past conversation record for this project (every session in the store), not one session. Runs are incremental only to avoid re-reading; the analytical horizon is always the full corpus.
- **Unit of work = concepts/themes**: the output is mined *information* — decisions, lessons, TODOs, flows, conflicts — grouped into concepts or themes (one or many), each aggregating evidence across sessions. A concept may draw on 20 sessions; a session may feed 5 concepts.
- **Counterpart**: operating on a *single* session as a data object (export, naming, archiving) is `/speckit.session`'s job — that command manages session **data** and does not parse content into concepts.

This command does **not** store transcripts verbatim — it extracts long-term value: decisions, reusable lessons, open TODOs, key interaction flows, and points of user↔model disagreement.

It is **incremental**: a manifest tracks which sessions were already distilled, so re-runs only process new ones and merge into the existing documents.

### Step 1: Identify Tool & Project

1. **Identify the executing agent** following `.specify/shared/workflow/agent-configuration.md` → "Step 1: Identify Executing Agent" (Claude Code / Copilot / Qoder / opencode / Codex / Hermes / …).
2. Run `.specify/scripts/bash/collect-history.sh --json` from repo root. It auto-detects the tool and prints JSON:
   ```json
   {
     "action": "locate", "tool": "<key>", "supported": <bool>,
     "project_path": "<abs path>", "project_name": "<name>",
     "session_store": "<path or null>", "session_count": <int>,
     "output_dir": "<...>/.specify/history",
     "manifest_path": "<...>/.specify/history/.manifest.json",
     "supported_tools": ["claude", "qoder"], "note": "<explanation>"
   }
   ```
   - If the detected `tool` is wrong, re-run with `--tool <key>` (append to the script call).
3. **Gate on support**:
   - If `supported` is `false`: report the detected tool, the `note`/`hint` (where its history would live), and that extraction for this tool is **not yet implemented** (only `supported_tools` are). Explain that support is pluggable via `STORE_RESOLVERS` in `.specify/scripts/python/history-utils.py`, then **stop**.
   - If `session_count` is `0`: report "No history found for this tool/project." and stop.

### Step 2: Extract Clean Sessions

1. Run `.specify/scripts/bash/collect-history.sh --action extract --tool <tool> --json` (add `--full` if the user requested it). This writes a de-noised transcript per session to `.specify/history/.work/<sid>.txt` and returns an inventory:
   ```json
   {
     "total_sessions": <int>, "pending_sessions": <int>, "total_chars": <int>,
     "sessions": [{"sid": "...", "short": "...", "title": "...",
       "first_user": "...", "chars": <int>, "n_user": <int>, "n_asst": <int>,
       "mtime": "...", "processed": <bool>, "work_file": "<path or null>"}]
   }
   ```
2. **Select sessions to distill**: those with `work_file != null` (i.e. `processed=false`, unless `--full`). **Skip trivial/meta sessions**: very small `chars` (heuristic < ~1500) or clearly meta ("how do I get session history", empty-content). Note skipped ones in the final report — never silently drop.
3. If nothing to process (all already in the manifest and no `--full`): report "History is already up to date (N sessions)." and stop.

### Step 3: Distill Each Session (delegate in parallel)

For each selected session, read its `work_file` and extract **only long-term value** along these **five dimensions** (no verbatim retelling):

1. **关键决策与理由 (Decisions & rationale)** — what was decided and why; **explicitly include abandoned alternatives** and why they were dropped.
2. **可复用经验 / 踩坑 (Reusable lessons / pitfalls)** — problems, fixes, general techniques, anti-patterns worth avoiding next time.
3. **未完成 / 待办 (Unfinished / TODO)** — deferred ideas, open questions, follow-ups.
4. **关键交互流程 (Key interaction flow)** — notable multi-step workflows or conventions (else "无").
5. **用户 ↔ 模型的冲突/分歧点 (User↔model conflicts)** — where the user overrode/corrected the model. Format: `用户主张 X;模型原本 Y;最终 Z`.

**Delegation**: for large sessions, spawn parallel subagents (one per session) that each read one `work_file` and return the five-dimension extraction as compact Markdown. Batch to respect the concurrency cap. Small sessions may be read directly. Keep identifiers (file paths, command names, commit hashes) accurate.

### Step 4: Theme Aggregation → `.specify/history/`

Group the distilled sessions into **themes** (derive them from the actual content; do not force a fixed taxonomy). Generate or **update** these files (create `.specify/history/` if missing; if a root-owned dir blocks writes, recreate it as the current user):

- `README.md` — index: theme table (theme · covered sessions · date span), reading order, and a few cross-session "meta-conclusions".
- `00-cross-cutting-lessons.md` — lessons/pitfalls that recur across many sessions.
- `NN-<theme-slug>.md` — one file per theme, organized by the five dimensions above; cross-link related docs with `[[name]]`.

**Merge strategy (incremental)**: if a target file exists, **integrate** new findings into the right sections without overwriting valid prior content or duplicating entries. Add new theme files as needed. Only regenerate wholesale under `--full`.

### Step 5: Update Manifest

Record the sessions distilled this run so future runs skip them:
```bash
.specify/scripts/bash/collect-history.sh --action manifest-update --tool <tool> \
  --sids "<space/comma-separated full sids>" --json
```
(Use the full `sid` values from the inventory, not the `short` form.)

### Step 6: Report

Summarize: tool & project detected, sessions distilled this run vs skipped (with reasons), themes created/updated, and the output location. Remind the user that `.specify/history/.work/` is disposable scratch (git-ignored) and the `.md` docs + `.manifest.json` are the durable output.

## Safety Rules

1. **Read-only on history**: never modify or delete files under the tool's session store; only read them.
2. **No verbatim dumps**: `.specify/history/` holds distilled value, not raw transcripts.
3. **No silent truncation**: if you cap coverage (e.g. skip trivial sessions or batch-limit), say so in the report.
4. **Unsupported tools stop cleanly**: never guess an unadapted tool's storage format and parse it blindly.

## Feedback

At wrap-up (the same lifecycle point where this command prompts for a Git commit), perform an agent self-reflection step (never solicit feedback content from the user), following the canonical convention in `.specify/shared/workflow/feedback-step.md`:

1. **Gate on qualification & completion.** Only proceed if this command reached its wrap-up stage. Skip trivial/no-op runs; for an aborted run use the abort/partial rule below.
2. **Reflect (no user input).** Review this run against `/speckit.history`'s declared purpose and produce a short review plus ≥1 concrete, command-specific optimization point. If the run was clean, use exactly: `No significant optimization points identified this run.`
3. **Scope guard.** Keep strictly to this command's operation; do NOT produce a global/whole-project assessment (that is `/speckit.review`'s job). Entries are `scope: local`.
4. **Dedup guard.** Use a stable `run_id` (e.g. the feature key + a run timestamp); if a nested skill/command already recorded feedback for this same `(unit_id, run_id)`, the engine no-ops.
5. **Persist** via the engine:
   ```bash
   python3 "${SKILL_WORKDIR:-.}/.specify/scripts/python/feedback-utils.py" --action record \
     --unit-id "/speckit.history" --unit-type command \
     --run-id "<stable-run-id>" --feature "<feature-key-if-any>" \
     --review "<review prose>" --points-file "<points file>"
   ```
   Probe attribution: the engine resolves the unit to its probe object automatically — the entry inherits kind/slice from the probe registry. External custom units record via `--unit-id custom:<owner>/<name> --unit-type custom-unit`; their entries stay host-project-local and never enter upstream packages.
6. **Consolidated submission prompt(非阻塞).** If the returned `should_prompt` is `true`, append ONE non-blocking line to the wrap-up report inviting submission (point the user to the `/speckit.feedback package` command — the user-facing path; never paste the raw `feedback-utils.py` engine call into the user-facing line); it MUST NOT block the wrap-up flow and MUST NOT trigger any 自动传输 (manual delivery only; `--action mark-submitted` runs only if the user initiates submission). Below threshold, do not prompt.

**Abort / partial-run rule.** If the run failed before wrap-up, either skip recording or record with `--partial` and a `## Review` beginning `**Partial run** — `.

## Handoffs

**Before running this command**:
- Have some conversation history with the current tool for this project (this command distills the past, it does not create it).

**After running this command**:
- Run `/speckit.constitution` or `/speckit.instructions` if the distilled history surfaces conventions worth codifying.
- Invoke `memory-record` to persist key distilled knowledge into long-term project memory.