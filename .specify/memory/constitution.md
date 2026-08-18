<!--
SYNC IMPACT REPORT
==================
Version change: 1.1.0.1 → 1.2.0.1 (MINOR — roadmap section materially expanded; Principle I
scope contracted from unconditional to stage-scoped; Principle XIII realigned)

Bump reasoning: this is not a MAJOR rewrite (the principle set and governance structure are
intact, numbering unchanged), but it is more than a clarification: Principle I's
"NON-NEGOTIABLE ... indefinitely" upstream-tracking mandate is deliberately contracted to
Stages 1–2 and terminated at a declared Stage 3 fork point, and its rebase-survival rule
(prefer new files / minimal upstream diff) is dropped for Stage 2. Contracting a principle's
scope is a MINOR bump per the versioning policy.

Modified principles:
- "I. Upstream-First, Dual-Branch Discipline" → "I. Dual-Branch Discipline (Stage-Scoped)":
  the dual-branch mechanic stays NON-NEGOTIABLE while upstream tracking is active, but
  (a) Stage 2 now PERMITS rewriting upstream implementations and modifying upstream
  architecture — the "prefer new files, keep upstream diff minimal" rule is RETIRED as a
  constraint and demoted to a Stage-1 preference, and (b) tracking ENDS at the Stage 3 fork
  point, after which `main` is frozen as historical reference.
- "XIII. Staged Exploration Discipline" → realigned to the expanded roadmap: per-stage
  primary work, proposal-friendliness in Stage 1, and the fork-point declaration procedure.

Added principles: none (I–XV numbering preserved).

Added sections: none. "Project Nature & Three-Stage Roadmap" is substantially expanded
(per-stage focus / primary work / permitted-and-forbidden / exit criteria table).

Removed sections: none.

Templates requiring updates:
- ✅ .specify/templates/plan-template.md — Constitution Check renders principles dynamically
  from this file; no edit needed
- ✅ .specify/templates/requirements-template.md — Feature binding references features.md
  with no hard-coded principle number; no edit needed
- ✅ .specify/templates/tasks-template.md — Tests Mode derived dynamically; no edit needed
- ✅ README.md — no constitution/principle references; no edit needed
- ✅ docs/quickstart.md — does not exist; nothing to update

Propagated in this amendment (consequences of relaxing Principle I):
- ✅ `xuanji.md` — convention paragraph rewritten: the "minimal upstream diff" rule is now
  stated as a Stage-1 preference, and the registration duty is stated as mandatory and
  stage-independent (it is what replaces diff minimalism).
- ✅ `.specify/memory/features/001.md`, `.specify/memory/features/014.md` — "must stay
  separable from upstream code" notes rewritten to the stage-scoped rule + registration duty.
- ✅ `.specify/instructions.md` — dual-branch paragraph no longer claims unconditional
  rebase-survival; three-stage summary now names each stage's primary work and the Stage 3
  fork point.

Follow-up TODOs:
- TODO(FEATURE_018_COMPAT_POLICY): `.specify/memory/features/018.md` assumes indefinite
  upstream rebasing when defining API compatibility policy; it should account for the Stage 3
  fork point. Refresh via `/speckit.feature`.
- TODO(FEATURE_016_TRACKING_AUTOMATION): `.specify/memory/features/016.md` (upstream tracking
  automation) is scoped to Stages 1–2 by this amendment and becomes obsolete at the fork
  point; record that end-of-life condition.
- TODO(TOOL_REUSE_PRINCIPLE_REF): inherited from upstream Spec Kit — generated command copies
  cite "Constitution Principle XII" for the tool-reuse gate, but Principle XII here is
  "Complete & Correct Options, or None"; this constitution declares no tool-reuse principle.
  Cite `.specify/shared/workflow/tool-reuse-gate.md` instead.
- TODO(GENERATED_PROMPT_COPIES): `.github/prompts/speckit.tasks.prompt.md` and
  `.opencode/command/speckit.tasks.md` cite "Principle IV Test-First Development" (Test-First
  is VI here and no longer NON-NEGOTIABLE). Generated copies — refresh via
  `/speckit.instructions`.
-->

# Xuanji Substrate (Agent Substrate Fork) Constitution

## Project Nature & Three-Stage Roadmap

This project is a **technical exploration**, not a production product. It starts from the
open-source Agent Substrate project and pursues its goal through three stages. All custom
work happens on the `xuanji` branch of this fork; **contributing back to upstream is
explicitly out of scope** for the whole roadmap.

### Stage 1 — Absorb (current)

**Goal**: internalize upstream's concepts and core structure well enough to modify them
deliberately.

- **Primary work**: producing documentation and curating external reference material —
  concept explanations, architecture analysis, flow references, competitor/ecosystem
  landscape. This is the stage's *deliverable*, not a side activity.
- **Also permitted**: proposals for new solutions and requirements (design proposals, ADRs
  in `Proposed` status, feature registry entries in `Draft`). A proposal is a legitimate
  Stage 1 output even when nothing is implemented for it.
- **Code work**: exploratory pilots are permitted (e.g. `cmd/e2bgw`, the `wasm` sandbox
  class) but they probe Stage 2 and MUST NOT be read as evidence that Stage 1 is complete.
  During this stage, keeping upstream files at minimal diff remains the PREFERRED style,
  because understanding is still forming.
- **Exit criteria**: upstream's core concepts and structure are documented accurately enough
  (grounded in code per Principle V) that a modification can state what it changes and why.

### Stage 2 — Fuse (customize upstream's architecture)

**Goal**: add E2B support and the Wasm Sandbox to the substrate codebase and progressively
fuse them with it, modifying upstream's original architecture where the fusion requires it.

- **Primary work**: implementation. E2B protocol support and the Wasm sandbox runtime move
  from pilot to integrated capability.
- **Explicitly permitted** (this is the deliberate relaxation): rewriting upstream
  implementations, restructuring upstream packages, and changing upstream architecture when
  the fusion requires it. The Stage-1 preference for "new files only / minimal upstream
  diff" is NOT a constraint here.
- **Mandatory in exchange**: every modification to an upstream-owned file MUST be registered
  in `xuanji.md` with what changed and why. As rewrites grow, that registry — not diff
  minimalism — is what keeps rebases and the eventual fork decision tractable.
- **Still forbidden**: pushing custom features to upstream `main` or proposing them upstream;
  committing custom work to the local `main` branch.
- **Exit criteria**: the custom capabilities are integrated rather than bolted on, and the
  cost of continued upstream rebasing visibly exceeds its benefit.

### Stage 3 — Independent Delivery (shed the substrate dependency)

**Goal**: deliver the customized project on its own terms, progressively ending its
dependence on upstream substrate.

- **Primary work**: consolidating a coherent project identity — naming, boundaries, and
  delivery artifacts that stand without reference to upstream.
- **The fork point**: Stage 3 begins with an explicitly declared **fork point** — a recorded
  upstream commit at which rebase tracking STOPS. After it, `main` is frozen as historical
  reference, `xuanji` (or its successor trunk) becomes the sole line of development, and
  upstream changes are no longer integrated as a matter of course.
- **Declaring the fork point** is a constitutional amendment (see Governance): it records the
  upstream commit, the date, and the rationale. Until that amendment lands, upstream tracking
  per Principle I remains in force.
- **Exit criteria**: the project is delivered and maintained independently, with no standing
  obligation to upstream.

**Current stage: Stage 1 (absorb-dominant).**

Two consequences bind the whole constitution:

- **Concepts are provisional.** Terminology, boundaries, and abstractions are expected to
  change substantially across the three stages. Effort MUST NOT be spent freezing
  vocabulary or building migration/compatibility machinery for concepts that are still
  moving.
- **Functionality outranks polish.** Working capability is the first priority; this project
  deliberately does NOT hold itself to production quality bars. Where a rule below would
  trade delivered functionality for polish, the functional-first mandate wins — except for
  the mechanical gates in Principle IX, which are cheap and protect upstream tracking.

## Core Principles

### I. Dual-Branch Discipline (Stage-Scoped)
This project is a custom fork of the open-source Agent Substrate project, adding custom
features (E2B protocol plane, Wasm Sandbox) on top of upstream, on the `xuanji` branch only.

**While upstream tracking is active (Stages 1–2), the dual-branch mechanic is
NON-NEGOTIABLE:**
- `main` MUST continuously track the upstream open-source repository. Upstream changes land
  on `main` only; no custom development is committed to `main`.
- `xuanji` is the sole branch for custom development. `xuanji` MUST be kept current by
  rebasing onto `main` (never by merging `main` into `xuanji`), so upstream updates integrate
  cleanly and history stays linear.
- Every commit on `xuanji` MUST be classifiable as either "upstream cherry-pick" or "custom
  feature work"; mixed commits are forbidden.
- Custom features MUST NOT be pushed to, or proposed against, upstream `main`. Contributing
  upstream is out of scope for this project.

**How invasive custom changes may be is stage-dependent:**
- **Stage 1**: prefer new files, packages, and extension points; keep any upstream-file diff
  minimal. This is a PREFERENCE that protects a still-forming understanding, not a hard rule.
- **Stage 2**: rewriting upstream implementations and modifying upstream architecture is
  PERMITTED (see roadmap). In exchange, every modified upstream file MUST be registered in
  `xuanji.md` with what changed and why.
- **Stage 3**: upstream tracking ENDS at the declared fork point. From then on `main` is
  frozen as historical reference and this principle's tracking obligations no longer apply;
  the registry in `xuanji.md` becomes the record of divergence.

Rationale: rebase-based tracking is a means, not an end — it exists to keep absorbing
upstream cheap while we still benefit from it. Forbidding architectural change forever would
make Stage 2 impossible, and tracking a project we intend to leave would be waste; so the
obligation is scoped to the stages where it pays, and the registration duty grows as diff
minimalism is given up.

### II. Feature-Centric Development
Feature is the long-term core framework of the project:
- The Feature list (`.specify/memory/features.md`) MUST remain the "single
  source of truth" for the project's capability scope.
- Every phase of spec → plan → tasks → implement MUST review Feature
  additions/merges/splits/deletions.
- Feature changes MUST be traceable to corresponding spec/plan evidence and
  recorded in the Feature detail.

Rationale: keep project evolution Feature-centric to ensure long-term
consistency and maintainability across both upstream-derived and custom
capabilities. Features are also the stable spine while concepts churn across
stages (see Principle XIII).

### III. Documentation-First
Documentation is a first-class deliverable and MUST take priority over test
coverage:
- The project MUST maintain sufficiently detailed and accurate documentation so
  it can serve as reliable context for AI agents / large language models,
  supplying project knowledge and background information.
- Documentation MUST NOT record implementation details; implementation details
  belong in the code itself.
- Documents MUST stay focused and reasonably sized; when a document becomes too
  complex it MUST be split into smaller, cohesive documents.
- Documents MUST cross-reference one another so that basic navigation can be
  accomplished purely through internal links.
- Markdown documents MUST maintain basic metadata (e.g. title, purpose/summary,
  status, last-updated date, and related links).

Rationale: as agent capabilities grow, high-quality documentation becomes the
primary knowledge context that lets LLMs understand a project accurately and
reason effectively; it therefore ranks above testing in the principle
hierarchy. In Stage 1 this ranking is not merely a priority but the stage's
definition of work: absorbing upstream IS a documentation deliverable.

### IV. Documentation Naming & Location Conventions
- ALL-CAPS Markdown filenames are RESERVED for conventional,
  ecosystem-recognized root-level artifacts (e.g. `README.md`, `LICENSE`,
  `CHANGELOG.md`, `CONTRIBUTING.md`); ordinary content documents MUST use
  lowercase `kebab-case.md` and MUST NOT squat reserved names.
- A document's meaning derives from its FULL PATH, not just its filename: place
  docs so that `<area>/<topic>.md` reads as "the <topic> of <area>" (e.g.
  `docs/e2b/design.md`, `docs/wasm-sandbox/overview.md`), reusing generic
  filenames scoped by directory rather than inventing globally-unique names.
- Tool/framework-mandated filenames are NON-NEGOTIABLE and MUST match the exact
  required pattern and location (e.g. Spec Kit memory lives in
  `.specify/memory/`, agent definitions in `.specify/agents/`); such names MUST
  NOT be renamed to fit project conventions.

Rationale: predictable naming and path semantics let both humans and agents
locate and reason about documents without external indexes.

### V. Code as the Single Source of Truth
- Source code is the authoritative source of truth for the project's actual
  state; documentation describes intended/target behavior that may not yet be
  realized.
- When establishing or citing facts about how the system currently behaves,
  code MUST take precedence over documentation, unless a document is explicitly
  designated as authoritative for that fact.
- When code and documentation disagree, treat the divergence as a signal to
  update the documentation (or flag the code as not-yet-implementing the
  intended goal), not to trust the document as current reality.

Rationale: in a fast-moving fork that rebases upstream frequently, docs drift is
inevitable; anchoring factual claims in code prevents decisions based on stale
descriptions. This is also what makes Stage 1's documentation trustworthy: an
absorbed concept counts as understood only when it was read out of the code, and
Stage 1 proposals must not be mistaken for realized behavior.

### VI. Pragmatic Testing (Exploration-Grade)
Testing is valued but is explicitly NOT a merge gate during technical
exploration:
- Tests SHOULD be written for logic that is costly to re-verify by hand: pure
  functions, protocol/parameter mapping, state machines, and anything already
  covered by upstream tests.
- TDD (Red → Green → Refactor) is a RECOMMENDED style for such logic, not a
  mandate.
- Code lacking tests MAY be merged. Absence of tests MUST NOT be used to block
  functional progress.
- Changes MUST NOT knowingly break existing tests. When a change invalidates an
  upstream test, the test MUST be updated deliberately (with the reason stated),
  never deleted to silence a failure (see Principle XI). In Stage 2, rewriting an
  upstream implementation legitimately invalidates its tests — rewrite them with
  the change, and say so in the `xuanji.md` entry.
- **Tests Mode default: OFF.** `/speckit.tasks` MUST default Tests Mode to OFF and
  cite this principle; a feature MAY opt in to ON when its own risk justifies it.

Rationale: the functional-first mandate means exploration throughput beats
coverage. Regression protection still matters where breakage is silent and
expensive, so the recommendation is targeted rather than universal.

### VII. Opportunistic Integration Validation
Integration/contract validation is opportunistic during exploration:
- Cross-component seams (control-plane gRPC APIs in `pkg/proto/`, internal ttrpc
  interfaces, Envoy xDS configuration, E2B protocol payloads) SHOULD be
  validated when they are the subject of the change.
- A manual smoke run or demo path is an ACCEPTABLE substitute for automated
  integration coverage; the substitution SHOULD be stated in the plan or PR.
- `make e2e` requires a provisioned cluster and is therefore NOT a routine gate;
  run it when changing lifecycle, scheduling, or routing behavior and a cluster
  is available.

Rationale: real-dependency environments (GCP cluster, gVisor runtime) are
expensive to stand up; requiring them per change would stall exploration while
adding little signal at this stage.

### VIII. Observability, Versioning & Simplicity
All components MUST be observable and versioned:
- Use structured logs and OpenTelemetry/Prometheus instrumentation for
  important events and errors, consistent with existing upstream patterns.
- Document any breaking changes to the control-plane API and migration notes.
- Keep designs as simple as possible; avoid speculative features (YAGNI).

Rationale: makes systems debuggable, upgradable, and maintainable — especially
when diagnosing behavior across rebase boundaries. YAGNI is load-bearing here:
with concepts still provisional (Principle XIII), speculative generality is
likely to be discarded.

### IX. Mechanical Quality Gates
Only machine-checkable, low-cost gates are mandatory; they exist to protect
upstream tracking rather than to enforce polish:
- The code MUST build, and `make verify` (go vet, gofmt, boilerplate/license
  headers, go modules) MUST pass before review.
- All new files MUST carry the required copyright/license headers
  (`hack/boilerplate/`).
- `go.mod` MUST stay tidy when dependencies change.
- A smoke test or example run, and reflecting new behavior in
  specs/plan/tasks/docs, are RECOMMENDED rather than required.

Rationale: formatting, vet, and header gates cost seconds and directly determine
whether a rebase onto upstream stays clean (Principle I); judgment-heavy quality
bars are what the functional-first mandate relaxes.

### X. Better-Harness Orientation
The project is a *harness* for AI agent work — an environment in which an agent
can understand the task, execute on supported and repeatable paths, validate its
changes, deliver safely, and carry lessons forward. Improvement work MUST be
oriented toward making that harness better:
- Locate and motivate improvements against the five Agent Work Loop dimensions
  (Task Understanding, Controlled Execution, Change Validation, Reliable
  Delivery, Learning Capture); the canonical goal model is
  `.specify/shared/guidelines/better-harness.md` — reference it, do not restate
  it.
- Evidence discipline governs improvement claims: a configured asset proves at
  most that a mechanism exists (configured ≠ used); unobserved evidence MUST
  NOT be treated as a defect or a conclusion; "improved" MUST only be claimed
  from comparable before/after evidence.
- This principle adds orientation, not machinery: it MUST NOT justify new
  scoring systems, maturity reports, or tracking/recording engines.

Rationale: agents change code fast, but the workflow around them is usually the
weak point; naming the goal lets every improvement answer "which part of the
harness does this strengthen?".

### XI. Non-Destructive Remediation
Problems MUST be fixed at their root cause, never by destroying the artifact
that exhibits them:
- Deleting files, code, or data MUST NOT be used as a shortcut to make an
  immediate problem disappear.
- When an obstacle is encountered, the underlying cause MUST be identified and
  repaired; bypassing safety checks (e.g. `git push --force` on shared
  branches, `--no-verify`) is forbidden.
- When a destructive operation is genuinely required (deletion, force-overwrite,
  `git rebase` on published history), it MUST be explicitly confirmed with the
  user before execution.
- Stage 2 rewrites are NOT an exception: replacing an upstream implementation is
  a deliberate, registered change (Principle I), not a licence to delete code
  whose purpose is not yet understood.

Rationale: destructive shortcuts destroy information and trust, and in a
rebase-based fork can silently erase custom work; mandatory confirmation keeps
irreversible actions under human control. The relaxed quality bar does NOT
relax this principle — exploration produces irreplaceable context.

### XII. Complete & Correct Options, or None
Presenting options that are incomplete or incorrect is WORSE than presenting
no options:
- Any option set offered to the user (choices, plans, recommendations) MUST be
  verified correct and MUST cover the realistic decision space.
- When correctness or completeness cannot be ensured, the open question MUST be
  presented directly instead of a partial or speculative option list.
- A known-imperfect option MUST be labeled with its gap rather than presented
  as solid.

Rationale: wrong or partial options mislead decisions more than no guidance at
all.

### XIII. Staged Exploration Discipline
Work MUST be located within the three-stage roadmap declared above, and MUST
respect the rules of the current stage:
- **Stage awareness**: a change SHOULD be classifiable as absorbing upstream
  (Stage 1), fusing custom capability into it (Stage 2), or consolidating an
  independent identity (Stage 3). Work that belongs to a later stage is
  permitted as an exploratory pilot, but MUST be labeled as such rather than
  presented as settled direction.
- **Absorb before overriding**: an upstream mechanism MUST be understood and
  documented before it is rewritten. "Understood" means grounded in code per
  Principle V, not inferred from documentation alone. This is what makes the
  Stage 2 licence to rewrite safe rather than reckless.
- **Proposals are first-class in Stage 1**: new solution and requirement
  proposals are expected output, recorded as ADRs (`Proposed`) or Feature
  entries (`Draft`). A proposal MUST NOT be presented as implemented behavior
  (Principle V), and MUST NOT be silently dropped (Principle XV).
- **Provisional concepts**: terminology and abstractions MUST be treated as
  provisional. Renaming churn, deprecation shims, and backward-compatibility
  layers for internal concepts MUST NOT be introduced merely to preserve a name
  that is expected to change in a later stage.
- **Stage transitions** are recorded by amending the roadmap section of this
  constitution, so "which stage are we in" always has one answer. The Stage 3
  transition additionally records the fork point (upstream commit + date +
  rationale).

Rationale: an exploration project fails by drifting — either by rewriting
upstream before understanding it, or by prematurely hardening vocabulary that
the next stage will discard. Naming the stage makes both failures visible, and
makes the Stage 2 relaxation conditional on Stage 1 actually having been done.

### XIV. Hybrid Stack Fusion
This project deliberately fuses an established cloud-native stack (Kubernetes,
CRDs/controllers, Envoy, gVisor/microVM isolation) with the emerging AI-agent
stack (E2B-compatible sandbox protocol, agent lifecycle and tooling surfaces):
- Concepts from the two stacks MUST be mapped explicitly rather than conflated.
  Where an agent-stack concept is realized by a cloud-native primitive, the
  mapping MUST be written down (e.g. E2B *Sandbox* ↔ Substrate *Actor*).
- Adopting an external protocol (E2B and successors) MUST NOT require forking
  its client ecosystem: compatibility surfaces belong in dedicated translation
  components, keeping the external contract at the edge.
- Neither stack's vocabulary is authoritative over the other. Terms MUST be
  qualified when ambiguous across stacks (e.g. "worker" in Kubernetes vs agent
  senses).

Rationale: most defects at this boundary are conceptual, not mechanical — two
mature vocabularies describe overlapping things differently. Explicit mappings
localize the translation cost instead of spreading it through the codebase, and
they are the artifact Stage 2's fusion work is built on.

### XV. Design Documents as Durable Reference Material
The repository carries a substantial body of prior design and exploratory
thinking. These documents are maintained reference assets:
- Prior design/analysis documents MUST be kept as living reference material for
  later development, and MUST be maintained (corrected, split, re-linked) rather
  than left to rot.
- Superseded material MUST NOT be silently deleted; it is annotated with its
  status or archived so the reasoning trail survives (consistent with Principle
  XI and the docs archive-not-delete rule).
- Because such documents record intent that may never have been implemented,
  each SHOULD carry enough status metadata (per Principle III) for a reader to
  tell aspiration from realized behavior; when in doubt, Principle V applies and
  code wins.
- Reference material that originates outside this project MUST be usable
  publicly: this repository has public remotes, so confidential or
  internal-only source material MUST NOT be committed into the documentation
  space.

Rationale: in an exploration project the reasoning behind rejected and pending
designs is a primary asset — it is what prevents the next stage from re-deriving
or repeating past conclusions. Stage 1 deliberately produces documents and
proposals faster than implementation, so their upkeep is the stage's main
maintenance burden.

## Technology & Platform Constraints

- **Language**: Go (toolchain pinned by `go.mod`; currently 1.26.x), formatted
  with `gofmt`.
- **Platform**: Kubernetes-based control plane; worker Pods, gVisor (`runsc`)
  sandboxing for workload isolation, Envoy for traffic routing, OCI container
  images built with `ko`.
- **Repository layout** is normative (see `agents.md` and
  `docs/dev/code-layout.md`): binaries in `cmd/<binary>/`, shared-internal
  packages in `internal/`, public packages in `pkg/`, public control-plane
  protos in `pkg/proto/`, internal protos in `internal/proto/`, dev scripts in
  `hack/`, standalone tools in `tools/<name>` (own `go.mod`).
- **Custom feature surface**: the E2B protocol plane and Wasm Sandbox are
  xuanji-branch-only capabilities and MUST NOT be pushed to or proposed against
  upstream `main`. Through Stage 1 they stay in clearly identifiable locations;
  from Stage 2 they may reach into upstream-owned code, with every such file
  registered in `xuanji.md` (Principle I).
- **Documentation space is published**: `docs/` doubles as a Hugo site root and
  everything under it is publishable to public remotes. Sensitive or
  internal-only material MUST NOT be placed there (Principle XV).
- **Security**: gVisor sandboxing is the isolation boundary; changes MUST NOT
  weaken sandbox isolation or bypass security checks. The relaxed quality bar
  does NOT extend to isolation boundaries. The security story is early — new
  code MUST respect security best practices and keep the security section of
  `agents.md` current.

## Development Workflow

1. **Upstream sync** (Stages 1–2 only): fetch upstream and fast-forward `main`.
   `main` never diverges from upstream. After the Stage 3 fork point this step
   ceases and `main` is frozen.
2. **Custom development**: all feature work happens on `xuanji` (or short-lived
   branches cut from `xuanji`, merged/rebased back into `xuanji`).
3. **Rebase cadence** (Stages 1–2): after each upstream sync that matters to us,
   rebase `xuanji` onto `main`. Resolve conflicts in favor of preserving upstream
   intent for upstream-owned files that we have not deliberately rewritten; for
   files listed in `xuanji.md` as intentionally rewritten, our version wins and
   the registry entry is updated.
4. **Registering upstream edits**: any change to an upstream-owned file is
   recorded in `xuanji.md` (what changed, why). This is the conflict checklist
   for rebases and the divergence record for the eventual fork decision.
5. **Pre-review gates**: the build and `make verify` MUST pass locally before
   review (Principle IX). `make test` is RECOMMENDED; `make e2e` only when the
   change touches lifecycle/scheduling/routing and a cluster is available.
6. **Spec-driven custom features**: custom features follow the Spec Kit flow
   (`/speckit.feature` → `/speckit.requirements` → `/speckit.plan` →
   `/speckit.tasks` → implement) so that feature registry, requirements, and
   plans stay traceable per Principle II. Exploratory spikes MAY skip the flow,
   but a spike that becomes a kept capability MUST be registered as a Feature
   before further work.

## Governance

- This constitution supersedes other ad-hoc guidelines. Where project docs
  conflict with this constitution, the constitution wins until amended.
- **Amendment procedure**: amendments are proposed via
  `/speckit.constitution`, reviewed together with their Sync Impact Report,
  and take effect with a version bump. Principle additions/removals/renames
  are MINOR; rewrites of governance structure are MAJOR; clarifications are
  PATCH.
- **Versioning policy**: version follows `x.y.z.ddd` (MAJOR.MINOR.PATCH.DAILY);
  the daily counter increments on every update and resets on any other bump.
- **Stage transitions**: advancing between the three roadmap stages is an
  amendment to the "Project Nature & Three-Stage Roadmap" section and MUST
  restate which rules newly apply — notably whether the relaxed quality bar
  still holds and how invasive custom changes may be (Principle I).
- **Fork-point declaration**: entering Stage 3 requires an amendment recording
  the upstream commit at which tracking stops, the date, and the rationale.
  Until it lands, Principle I's tracking obligations remain in force; no
  informal "we stopped rebasing" state is recognized.
- **Principle numbering stability**: principle numbers are cited from other
  artifacts. New principles SHOULD be appended rather than inserted; when a
  renumber is unavoidable, every citing artifact MUST be updated in the same
  amendment and listed in the Sync Impact Report.
- **Compliance review**: every `/speckit.plan` MUST run the Constitution Check
  against the current principle list; unjustified violations block the plan.
  Templates referencing principles MUST be kept in sync via the Sync Impact
  Report checklist.

**Version**: 1.2.0.1 | **Ratified**: 2026-08-06 | **Last Amended**: 2026-08-12
