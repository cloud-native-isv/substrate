<!--
SYNC IMPACT REPORT
==================
Version change: 1.0.0.1 → 1.1.0.1 (MINOR — principle additions + deliberate scope
contraction of the testing/quality principles)

Modified principles:
- "VI. Test-First Development" → "VI. Pragmatic Testing (Exploration-Grade)": TDD demoted
  from MUST to SHOULD; the "code lacking tests MUST NOT be merged" bar is REMOVED per the
  project's functional-first exploration mandate. Declares Tests Mode default = OFF.
- "VII. Integration & Contract Testing" → scope contracted: integration/E2E coverage is
  opportunistic during exploration; `make e2e` no longer implied as a routine gate.
- "IX. Continuous Integration & Quality Gates" → "IX. Mechanical Quality Gates": retains
  only the low-cost machine-checkable gates (build, `make verify`, license headers) that
  protect Principle I rebase health; human-judgment quality bars demoted to SHOULD.

Added principles:
- XIII. Staged Exploration Discipline (three-stage model; concept instability tolerance)
- XIV. Hybrid Stack Fusion (cloud-native K8s/E2B + AI-agent stack integration rules)
- XV. Design Documents as Durable Reference Material (prior design/thinking docs are
  maintained reference assets, not disposable artifacts)

Added sections:
- "Project Nature & Three-Stage Roadmap" (states the exploration mandate and current stage)

Removed sections: none

Numbering note: new principles were APPENDED (XIII–XV) rather than inserted, deliberately
preserving the existing I–XII numbers so that live references in
`.specify/instructions.md` (Principle I) and `.specify/memory/features/*.md`
(Principles I, VIII, XI) remain valid.

Templates requiring updates:
- ✅ .specify/templates/plan-template.md — Constitution Check renders principles
  dynamically from this file; no edit needed
- ✅ .specify/templates/requirements-template.md — Feature binding references features.md
  with no hard-coded principle number; no edit needed
- ✅ .specify/templates/tasks-template.md — Tests Mode derived dynamically; Principle VI
  now states the default explicitly (OFF)
- ✅ README.md — no constitution/principle references; no edit needed
- ✅ docs/quickstart.md — does not exist; nothing to update
- ✅ .specify/shared/workflow/feedback-step.md + .specify/skills/improve-{docs,team,agent,
  skills,tools}/SKILL.md — PRE-EXISTING drift fixed: cited "Principle XIII" for
  Better-Harness Orientation, which is Principle X here (and XIII is now a different
  principle, so the stale refs would have become actively wrong)

Follow-up TODOs:
- TODO(TOOL_REUSE_PRINCIPLE_REF): `.specify/instructions.md`,
  `.specify/templates/instructions-template.md`, and the generated
  `.github/prompts/speckit.tools.prompt.md` / `.opencode/command/speckit.tools.md` cite
  "Constitution Principle XII" for the tool-reuse gate, but Principle XII here is
  "Complete & Correct Options, or None". This constitution declares no tool-reuse
  principle; either add one or drop the citation. Left unresolved — inherited from the
  upstream Spec Kit framework, not introduced by this amendment.
- TODO(GENERATED_PROMPT_COPIES): `.github/prompts/speckit.tasks.prompt.md` and
  `.opencode/command/speckit.tasks.md` cite "Principle IV Test-First Development"
  (Test-First is VI here, and is no longer NON-NEGOTIABLE). These are generated copies;
  refresh via `/speckit.instructions` rather than hand-editing.
-->

# Xuanji Substrate (Agent Substrate Fork) Constitution

## Project Nature & Three-Stage Roadmap

This project is a **technical exploration**, not a production product. It starts from the
open-source Agent Substrate project and pursues its goal through three stages:

1. **Stage 1 — Absorb**: digest and internalize the upstream open-source project's own
   definitions, concepts, and mechanisms.
2. **Stage 2 — Customize**: make substantial custom modifications on top of the upstream
   base.
3. **Stage 3 — Redefine**: consolidate the result into a new, coherent project identity of
   its own.

**Current stage: Stage 1 (absorb-dominant).** Existing custom surfaces (`cmd/e2bgw`, the
`wasm` sandbox class) are treated as exploratory pilots that probe Stage 2, not as evidence
that Stage 1 is complete.

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

### I. Upstream-First, Dual-Branch Discipline
This project is a custom fork of the open-source Agent Substrate project, adding
custom features ("E2B 协议面" / E2B protocol plane, "Wasm Sandbox") on top of
upstream. The dual-branch workflow is NON-NEGOTIABLE:
- `main` MUST continuously track the upstream open-source repository. Upstream
  changes land on `main` only; no custom development is committed to `main`.
- `xuanji` is the sole branch for custom development. `xuanji` MUST be kept
  current by rebasing onto `main` (never by merging `main` into `xuanji`), so
  upstream updates integrate cleanly and history stays linear.
- Custom changes MUST be structured to survive rebases: prefer new files,
  packages, and extension points over edits to upstream-owned files; where an
  upstream file must change, keep the diff minimal and clearly attributable to
  a custom feature.
- Every commit on `xuanji` MUST be classifiable as either "upstream cherry-pick"
  or "custom feature work"; mixed commits are forbidden.

Rationale: rebase-based upstream tracking is only sustainable when custom code
is separable from upstream code; this keeps merge friction low and upstream
updates fast indefinitely. This principle stays NON-NEGOTIABLE even under the
functional-first mandate, because a broken rebase story would end Stage 2 before
it starts.

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
hierarchy. In an exploration project this ranking is sharper still: understanding
upstream (Stage 1) is a documentation problem, not a test-coverage problem.

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
descriptions. This matters doubly for the reference material covered by
Principle XV, which records past thinking that may never have been built.

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
  never deleted to silence a failure (see Principle XI).
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
  (Stage 1), customizing on top of it (Stage 2), or consolidating a new identity
  (Stage 3). Stage 2/3 work during Stage 1 is permitted as an exploratory pilot,
  but MUST be labeled as such rather than presented as settled direction.
- **Absorb before overriding**: during Stage 1, an upstream mechanism MUST be
  understood and documented before it is replaced. "Understood" means grounded
  in code per Principle V, not inferred from documentation alone.
- **Provisional concepts**: terminology and abstractions MUST be treated as
  provisional. Renaming churn, deprecation shims, and backward-compatibility
  layers for internal concepts MUST NOT be introduced merely to preserve a name
  that is expected to change in a later stage.
- **Stage transitions** are recorded by amending the roadmap section of this
  constitution, so "which stage are we in" always has one answer.

Rationale: an exploration project fails by drifting — either by rewriting
upstream before understanding it, or by prematurely hardening vocabulary that
the next stage will discard. Naming the stage makes both failures visible.

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
localize the translation cost instead of spreading it through the codebase.

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
or repeating past conclusions.

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
- **Custom feature surface**: the "E2B 协议面" (E2B protocol plane) and "Wasm
  Sandbox" features are xuanji-branch-only capabilities. Their code MUST live
  in clearly identifiable locations so upstream tracking (Principle I) is not
  impeded; they MUST NOT be pushed to or proposed against upstream `main`.
- **Documentation space is published**: `docs/` doubles as a Hugo site root and
  everything under it is publishable to public remotes. Sensitive or
  internal-only material MUST NOT be placed there (Principle XV).
- **Security**: gVisor sandboxing is the isolation boundary; changes MUST NOT
  weaken sandbox isolation or bypass security checks. The relaxed quality bar
  does NOT extend to isolation boundaries. The security story is early — new
  code MUST respect security best practices and keep the security section of
  `agents.md` current.

## Development Workflow

1. **Upstream sync**: fetch upstream and fast-forward `main`. `main` never
   diverges from upstream.
2. **Custom development**: all feature work happens on `xuanji` (or short-lived
   branches cut from `xuanji`, merged/rebased back into `xuanji`).
3. **Rebase cadence**: after each upstream sync that matters to us, rebase
   `xuanji` onto `main`. Resolve conflicts in favor of preserving upstream
   intent for upstream-owned files, while re-applying minimal custom deltas.
4. **Pre-review gates**: the build and `make verify` MUST pass locally before
   review (Principle IX). `make test` is RECOMMENDED; `make e2e` only when the
   change touches lifecycle/scheduling/routing and a cluster is available.
5. **Spec-driven custom features**: custom features follow the Spec Kit flow
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
  still holds.
- **Principle numbering stability**: principle numbers are cited from other
  artifacts. New principles SHOULD be appended rather than inserted; when a
  renumber is unavoidable, every citing artifact MUST be updated in the same
  amendment and listed in the Sync Impact Report.
- **Compliance review**: every `/speckit.plan` MUST run the Constitution Check
  against the current principle list; unjustified violations block the plan.
  Templates referencing principles MUST be kept in sync via the Sync Impact
  Report checklist.

**Version**: 1.1.0.1 | **Ratified**: 2026-08-06 | **Last Amended**: 2026-08-12
