# Team Definition & Persistence (create mode)

Detailed guide for `/speckit.team create`. The `SKILL.md` contract keeps the skeleton; this reference holds the full procedure, schema, and persistence rules.

---

## Goal-first procedure

Produce a team from a user **goal** and (unless one-shot) persist it as `.specify/teams/<slug>/team.md`. The goal is the team's north star — **establish it first, then derive both structures from it** (goal concept: `references/goal.md`).

1. **Establish the goal (first)** — extract the goal from `$ARGUMENTS`/conversation/repo context, ask if missing, and confirm it with the user; write it in a **verifiable** form (success criteria / threshold). When the goal is missing or too vague to derive a roster and pattern from, elicit it via the interview pattern (`.specify/shared/patterns/interview-pattern.md`) instead of one flat question — ask the settled-prerequisite decisions in a round as **open** questions carrying their context (no option menus, no recommended answers: the goal is the user's to state), write each answer into the goal statement before the next round, and stop at the user's confirmation that the goal is stable. If no live user is available, say the goal is unestablished and stop; never invent one. Goal semantics — the project-level Goal concept, one-goal-per-team singularity, criteria authority, and the `goal_slug` binding — are defined once in `.specify/shared/definitions/goal-definitions.md`. If the goal's theme is **optimization**, classify it (one-time vs continuous) and pick a strategy (elimination vs progressive) per `references/optimization-goals.md`.
2. **Match against team presets (before deriving anything)** — run `${SKILL_HOME}/scripts/match-team-preset.py --goal "<goal text>"` and act on the returned `confidence`: `high` → present the top preset (goal skeleton + roster + pattern) and **recommend reuse**; `medium` → present the top 2 candidates alongside the from-scratch option; `low`/`none` → say no preset matched and continue to step 3. Presets are known-good shapes distilled from teams that actually ran — reusing one is what keeps a vague goal from producing an arbitrary team. Never instantiate a preset silently, and never let it override an explicit user instruction. See `references/team-presets.md`.
3. **Select the pattern** via the Pattern Selection decision tree (`references/patterns.md`) — independent → parallel; sequenced → serial; iterative-quality → iteration; long-lived operation → continuous — **derived from the goal**. On preset reuse, the preset supplies the pattern; still confirm it fits.
4. **Build the roster (static structure)** — a Role × Stage × Type matrix. If the user did not supply members, **propose** them from the goal: prefer existing agents under `.specify/agents/{templates,instances}/`, otherwise temporary stage/worker templates from `templates/`. An **iteration or continuous team MUST include exactly one Team Supervisor** (Meta role). **Judge each member's `type` explicitly by its operating object** — operates on business artifacts/information → `Worker`; operates on other agents/skills/agent-defining configuration → `Meta`. Do **not** derive Type from Stage: an evaluator scoring a business artifact (repo state, rendered output, a document) is a `Worker`, not Meta. The implication runs **one way only**: a member that must **write** team config / agent definitions / skill definitions MUST be `Meta` (necessary); but holding an evaluator / optimizer / "continuous improvement" role does **not** by itself make it Meta (not sufficient) — an agent that iteratively improves a *business artifact* is still a `Worker`. Decide each member's Type from **what it writes to**, never from its role name (see `references/conceptual-model.md` → Type criterion + "Meta and write authority"). Roster rows carry **responsibility** (stage, territory, `blockedBy`, reporting duty); the referenced agent carries **capacity** — never fork a capacity artifact to express a new seat (`references/capacity-vs-responsibility.md`).
5. **Build the pattern config (dynamic structure)** — parallelism + territories (parallel), DAG `blockedBy` edges + per-handoff verification + file-path-only handoff (serial), quality dimensions + threshold + max_iterations + regression_limit (iteration), or the operating config — maturity + cadence + budget + constraints + independent verifier + state spine (continuous; see `references/operating-loops.md`).
6. **Confirm** the proposed **goal** + roster + pattern with the user, then persist the `Team` to `.specify/teams/<slug>/team.md` using the schema below (skip persistence only for an explicit one-shot run). When a preset was reused, record `preset: <preset_id>` in the frontmatter and apply the preset's `## Instantiation` steps (including any `constraints.md` / `STATE.md` bootstrap).

---

## Persisted `team.md` schema

Each persistent team owns a **directory** `.specify/teams/<slug>/` (no per-tool symlink — framework-internal). The definition is stored at `.specify/teams/<slug>/team.md`; per-run reports accumulate under `.specify/teams/<slug>/runs/`:

```markdown
---
name: <display name>
slug: <kebab-slug>
description: <one-line label>
goal: <overall final objective + success criteria / threshold>
goal_slug: <kebab-slug>        # optional — the GOAL's identity; distinct from `slug` (the team's identity). See references/summary-mapping.md
territory:                     # optional — TEAM-level coverage; all four patterns. Lifts member Territory Division one level up
  write:                       #   paths this team may create/modify (glob / brace / relative all normalised before compare)
    - <path-or-glob>
  read:                        #   paths this team may read (read overlap between teams is allowed)
    - <path-or-glob>
  forbidden:                   #   shared paths this team MUST NOT modify
    - <path-or-glob>
  non_path:                    #   coverage that is not file-shaped (framework itself, runtime, …); listed for arbitration, never intersected
    - { type: <dimension>, target: <free text> }
pattern: parallel | serial | iteration | continuous
preset: <preset_id>            # optional — set when instantiated from a team preset
created: YYYY-MM-DD
updated: YYYY-MM-DD
members:
  - agent: <slug-or-template-id>
    role: <role>
    lifecycle: persistent | temporary
    # territory: [...]        # parallel
    # blockedBy: [...]        # serial
config:
  # pattern-specific block (parallelism / DAG / loop settings)
  # optimization goals: optimization_target (single target path), or co_targets
  # (coordinated multi-target optimization — list of target paths + the layering
  # principle stating which kind of content belongs to which target; see
  # references/optimization-goals.md)
  summary:                     # optional — omit the block entirely and summary is still ENABLED (opt-out)
    enabled: true
    every: 5                   # refresh once per N phase boundaries; continuous default 5, bounded patterns 1
    delivery_dir: .specify/goal/<goal-slug>/summary/   # derived from goal_slug, never from the team slug
    interactive: false         # team-triggered refresh is automated → invoke the skill non-interactively
---

## Goal
<the team's overall final objective + success criteria; authored FIRST — the static and dynamic sections are organized to serve it. See references/goal.md>

## Static Structure
<Role × Stage × Type matrix table for this team's roster>

## Dynamic Structure
<pattern description, parallelism/DAG/loop settings, and the execution flow diagram.
 For iteration/continuous patterns this section MUST end with an explicit Loop Card
 (WHEN→SEE→DO→CHECK→STOP→LEAVE table): WHEN = trigger/cadence, SEE = state read first
 (STATE.md/constraints), DO = the bounded unit per cycle, CHECK = concrete verification
 (evaluator command/criterion), STOP = termination condition + hard caps
 (threshold/max_iterations/budget/kill-switch), LEAVE = what is persisted/handed off
 (runs report, STATE update, escalations). A loop whose CHECK or STOP cannot be
 stated concretely is not ready to persist.>
```

### Schema notes

- `slug` MUST be unique within `.specify/teams/`; it also names the team directory `.specify/teams/<slug>/`.
- `goal_slug` identifies the **goal**, not the team — it is deliberately a different axis from `slug` (the team slug). Two teams that declare the same `goal_slug` are pursuing one goal and share one summary; a team that omits it falls back to its own `slug` as an *inferred* goal identity. It MUST NOT be derived from the goal prose, so rewriting the goal text never relocates the summary. See `references/summary-mapping.md`.
- `territory` is the **team-level** coverage declaration and applies to all four patterns (the member-level Territory Division lifted one level up). An absent key means **undeclared** — never conflated with an empty scope. Path entries are normalised (brace-expanded, canonicalised) before comparison. Among teams sharing one `goal_slug`, no two `write` scopes may intersect (read overlap is allowed); a write intersection is a **contested area** that must be resolved to a single owner or a forbidden-write entry. The refresh only **detects and proposes** a re-division — a human ratifies it and the agreed scopes are written back into each `team.md`, never into the goal directory.
- `config.summary` tunes the periodic summary refresh (enable/disable, cadence, delivery directory, interactivity). Omitting the block leaves the summary **enabled** with the pattern's default cadence — `continuous` defaults to every 5th cycle, never every cycle.
- `members` MUST resolve to `.specify/agents/{templates,instances}/<slug>.agent.md` (instance wins on filename collision) or a temporary stage/worker template; unresolved members are surfaced as broken references.
- `config` MUST match `pattern`.
