# Goal Definitions Reference

Canonical definition of the **Goal** concept in Spec Kit, its boundary against **Requirement**, the criteria authority rule, the singularity rule, and the Goal–Team binding. This file is the single source of truth for the Goal concept; other documents (`/speckit.team`, `skills/create-team`, the glossary) link here rather than re-defining it. It sits alongside the other concept anchors: `tool-definitions.md`, `agent-definitions.md`, `subagent-definitions.md`.

## What a Goal Is

A **Goal** is a **project-level, first-class concept**: an **authored fact source** (never a derived artifact) that states a desired end state and how to tell it has been reached. Its **object is unrestricted**: a Goal may describe any desired outcome in any dimension — it is not limited to what the project's code implements (see Goal Dimensions below). It is persisted under `.specify/goal/<goal-slug>/` and is composed of exactly three parts:

1. **Goal narrative** — the desired end **outcome** (north star). Outcome, not steps: a Goal MUST NOT be written as a task list or an implementation plan.
2. **Verifiable success criteria** — thresholds / satisfaction conditions that an evaluator (a program or a scoring agent) can measure progress against.
3. **Lifecycle state** — `active` / `achieved` / `abandoned`. Terminal Goals are retained, never deleted (see Goal Archive).

Two operating properties follow:

- **Measured by degree**: a Goal is *pursued*; its verification is progress-shaped (percentage, threshold attainment, evaluator scores) — not a per-clause pass/fail.
- **Modified deliberately**: changing a Goal is a strategic act — the change is recorded and history is preserved; silent drift is prohibited.

### Goal Dimensions (illustrative, not a closed taxonomy)

Goals and Requirements live on **different planes** (decided 2026-08-04): a Requirement describes what the **current project's source code or configuration** must implement (a feature); a Goal may target **any dimension** of desired outcome, including ones no Requirement could ever express. Illustrative dimensions:

| Dimension | Example Goal | Why it is not a Requirement |
|-----------|--------------|-----------------------------|
| **Framework / harness itself** | "I want the spec-kit framework this project uses to stay continuously updated" | Its object is the toolchain, not this project's code — no FR on this repo can implement it |
| **Codebase-wide convention convergence** | "I want all project source code to be standard-idiomatic Golang" | A cross-cutting end state pursued by degree across the whole codebase, not a per-feature clause |
| **Delivered-capability outcome** | "I want feature X to run on platform Y" | An outcome about a capability in a target environment; Requirements specify the implementation, the Goal states the environmental end state being pursued |

Boundary note: when a dimension-2-style desire is frozen into a **binding rule** enforced by gates, it belongs to the **Constitution** (a governance principle); when it binds one deliverable as a testable clause, it is a **Requirement**; when it is a desired end state pursued and measured **by degree**, it is a **Goal**. The same sentence can move between the three homes only by deliberately changing its nature.

## Goal vs Requirement

**Goal** and **Requirement** are independent, parallel first-class concepts on **different planes** — **there is no necessary hierarchy between them** (decided 2026-08-04). A Requirement's object is fixed: this project's source code / configuration and the feature they implement. A Goal's object is free: any dimension of desired outcome.

| Dimension | Goal (目标) | Requirement (需求) |
|-----------|------------|--------------------|
| Object (作用对象) | Unrestricted — framework/harness, codebase conventions, delivered-capability outcomes, … (any dimension) | The project's source code / configuration — what feature they must implement |
| Essence | Desired end state + attainment criteria | Binding clauses on a deliverable |
| Question answered | *why / whither* — toward what, to what degree | *what / how-correct* — what MUST hold, what counts as done right |
| Home | `.specify/goal/<goal-slug>/` (authored fact source) | `.specify/specs/<ID>-<slug>/requirements.md` (bound to one Feature) |
| Shape | Singular narrative + criteria per Goal | Enumerated clauses (FR-xxx / SC-xxx) |
| Verification | By degree — progress, thresholds, evaluator scoring | Binary — per-clause pass/fail against tests and acceptance scenarios |
| Change discipline | Deliberate modification with recorded history (no silent drift) | Clarification/revision flow (Clarifications sessions, re-gated spec) |
| Failure semantics | Not yet attained → keep iterating / adjust strategy / deliberately re-scope | Not satisfied → implementation non-conforming, delivery blocked |
| Anti-pattern | Written as a task list / implementation steps | Written as an untestable aspiration |

**No structural link (decided)**: a requirements spec carries **no** Goal field — `requirements.md` does not reference a `goal-slug`, and a Goal never enumerates the FRs "under" it. When work on a Feature happens to advance a Goal, that connection surfaces **observationally** — in evaluation results, team summaries, and run reports — never as a mandatory field in either artifact. Neither concept derives from, contains, or validates the other.

**Litmus tests** (when writing a sentence, decide where it belongs):

1. **Deletion test** — deleting it loses *direction* → Goal; loses a *contract* → Requirement.
2. **Verification test** — verifying it needs scoring/progress → Goal criterion; needs a pass/fail check → Requirement clause.
3. **Change test** — changing it is a strategic decision with history → Goal; a clarification + spec revision → Requirement.
4. **Subject test** — the sentence's subject is an executing subject ("the team reaches …") → Goal; a system/deliverable ("the system MUST …") → Requirement.
5. **Object test** — it constrains what this project's source/config implements → Requirement; its object lies beyond the implementation surface (the framework itself, codebase-wide convergence, runtime/platform outcomes) → Goal.

## Criteria Authority Boundary

Success-shaped statements exist in both worlds; their authority is disjoint (decided 2026-08-04):

- **SC-xxx** (Success Criteria in a requirements spec) serve **only their own Feature** — they measure that Feature's delivery and nothing broader.
- **Goal success criteria** are **cross-feature** — they measure the end state regardless of which Features contributed.
- Criteria MUST NOT be copied between the two stores. Cross-feature aggregation happens at the **evaluation/summary layer** (evaluators, team summaries), with each side citing its own source — never by restating one store's criteria inside the other.

## Singularity Rule

One Goal = one objective (decided 2026-08-04):

- **Per Goal definition**: a Goal MUST NOT bundle several objectives into one composite definition — split them into separate `goal-slug`s, each with its own directory and lifecycle.
- **Per executing subject**: a team binds to exactly **one** Goal at a time (see Binding below). A team that "pursues two goals" is either two teams or a Goal that needs deliberate re-scoping.
- The **project** may hold multiple `active` Goals concurrently — each is its own `goal-slug`, each advanced by its own team(s).

## Storage & Goal Archive

```
.specify/goal/
└── <goal-slug>/     # one Goal definition — narrative + criteria + lifecycle state
```

- **Goal Archive** = the whole `.specify/goal/` tree: the materialized "current & historical goal list" of the project. Terminal (`achieved` / `abandoned`) Goals stay archived — never deleted.
- This document fixes only the **location and semantics**; the file layout inside `<goal-slug>/` is owned by the feature that implements Goal management.

## Goal–Team Binding

- A team references its Goal by **one-way identity**: the team declares a `goal_slug`; the binding is **N teams : 1 Goal**. The team side stores the identity only — never a copy of the Goal content.
- **Team Goal** therefore means *the reference* — which project-level Goal this team serves. Team evaluators measure progress against the referenced Goal's criteria.
- **Migration fallback**: teams created before Goal management exist may still carry an inline goal in `team.md`; wherever a Goal definition exists, the definition is authoritative and the inline copy is legacy.

## Terminology Boundaries

| Term | Meaning | Where defined |
|------|---------|---------------|
| **Goal** (this document) | Project-level authored end-state definition (narrative + criteria + lifecycle) | here; store `.specify/goal/<goal-slug>/` |
| **Goal Archive** | The `.specify/goal/` tree — current & historical Goals, terminal ones retained | here |
| **Goal–Team Binding** | One-way `goal_slug` reference, N teams : 1 Goal, identity-only | here |
| **Team Goal** | A team's reference to the project-level Goal it serves | here; declared in `.specify/teams/<slug>/team.md` |
| **Requirement** | Feature-bound spec of testable clauses (FR/SC) driving plan → tasks → implement → verification | `.specify/specs/<ID>-<slug>/requirements.md`; `/speckit.requirements` |
| **Feature** | Long-lived capability entry in the feature index | `.specify/memory/features.md` |
| **Success Criteria (SC-xxx)** | Per-feature measurable outcomes — authority limited to their Feature | requirements spec of that Feature |
| **Team** | Multi-agent structure organized around (exactly one) Goal | `/speckit.team`, `skills/create-team` |
