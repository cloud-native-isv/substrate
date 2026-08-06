<!--
SYNC IMPACT REPORT
==================
Version change: 0.0.0.0 (placeholder) → 1.0.0.1 (MAJOR — initial ratification)

Modified principles:
- "Library-First Design" → REMOVED (not applicable: multi-binary Kubernetes system,
  not a reusable-library product)
- "CLI & Text I/O Interface" → REMOVED (not applicable: control-plane gRPC/ttrpc
  services, not a CLI-text-I/O product)
- "Documentation-First" → kept (was III, now III)
- "Test-First Development" → kept, adapted to Go/Makefile workflow (was IV, now VI)
- "Integration & Contract Testing" → kept, adapted to `make e2e` (was V, now VII)
- "Observability, Versioning & Simplicity" → kept, adapted (was VI, now VIII)
- "Continuous Integration & Quality Gates" → kept, adapted to `make verify` (was VII, now IX)
- "Feature-Centric Development" → kept (was VIII, now II)
- "Better-Harness Orientation" → kept (was IX, now X)
- "Non-Destructive Remediation" → kept (was X, now XI)
- "Complete & Correct Options, or None" → kept (was XI, now XII)

Added principles:
- I. Upstream-First, Dual-Branch Discipline (fork workflow from user input)
- III. Documentation-First (mandated)
- IV. Documentation Naming & Location Conventions (mandated)
- V. Code as the Single Source of Truth (mandated)

Added sections:
- "Technology & Platform Constraints" (replaces [SECTION_2_NAME])
- "Development Workflow" (replaces [SECTION_3_NAME])

Removed sections: none

Templates requiring updates:
- ✅ .specify/templates/plan-template.md — Constitution Check is rendered dynamically
  from this file; no edit needed
- ✅ .specify/templates/requirements-template.md — Related Feature binding references
  features.md, no hard-coded principle number; no edit needed
- ✅ .specify/templates/tasks-template.md — Tests Mode derived dynamically; no edit needed
- ✅ README.md — no constitution/principle references found; no edit needed
- ✅ docs/quickstart.md — file does not exist; nothing to update

Follow-up TODOs: none
-->

# Xuanji Substrate (Agent Substrate Fork) Constitution

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
updates fast indefinitely.

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
capabilities.

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
hierarchy.

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
descriptions.

### VI. Test-First Development
Implementation MUST follow a Test-Driven Development style for core logic:
- Write or update tests BEFORE implementing new behavior.
- Ensure tests FAIL first (Red), then implement to make them PASS (Green).
- Refactor only with all tests passing (Refactor).

At minimum:
- Pure functions/utilities MUST have unit tests (`go test`, run via
  `make test`, which enables the race detector).
- Critical flows MUST have automated regression coverage.
- Code that lacks tests MUST NOT be merged.

Rationale: reduces regressions and clarifies intent; upstream CI enforces this,
so custom code must meet the same bar to keep rebases green.

### VII. Integration & Contract Testing
Integration/contract tests SHOULD cover:
- Cross-component communication (control-plane gRPC APIs in `pkg/proto/`,
  internal ttrpc interfaces, Envoy xDS configuration).
- Shared schemas or data contracts (actor snapshots, E2B protocol payloads).
- Critical end-to-end journeys via `make e2e` on a provisioned cluster.

When real dependencies are hard to run locally (GCP cluster, gVisor runtime),
abstract them behind interfaces and document follow-up contract tests in the
plan/tasks.

Rationale: validates real-world behavior beyond unit tests in a system whose
core value is low-latency lifecycle orchestration.

### VIII. Observability, Versioning & Simplicity
All components MUST be observable and versioned:
- Use structured logs and OpenTelemetry/Prometheus instrumentation for
  important events and errors, consistent with existing upstream patterns.
- Document any breaking changes to the control-plane API and migration notes.
- Keep designs as simple as possible; avoid speculative features (YAGNI).

Rationale: makes systems debuggable, upgradable, and maintainable — especially
when diagnosing behavior across rebase boundaries.

### IX. Continuous Integration & Quality Gates
Changes MUST be safe to merge:
- `make verify` (go vet, gofmt, boilerplate/license headers, go modules) MUST
  pass before review.
- A minimal smoke test or example run SHOULD be provided for new features.
- New behavior MUST be reflected in specs/plan/tasks/docs where applicable.
- All new files MUST carry the required copyright/license headers
  (`hack/boilerplate/`).

Rationale: ensures consistent quality and keeps the fork's CI story compatible
with upstream expectations.

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
irreversible actions under human control.

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
- **Security**: gVisor sandboxing is the isolation boundary; changes MUST NOT
  weaken sandbox isolation or bypass security checks. The security story is
  early — new code MUST respect security best practices and keep the security
  section of `agents.md` current.

## Development Workflow

1. **Upstream sync**: fetch upstream and fast-forward `main`. `main` never
   diverges from upstream.
2. **Custom development**: all feature work happens on `xuanji` (or short-lived
   branches cut from `xuanji`, merged/rebased back into `xuanji`).
3. **Rebase cadence**: after each upstream sync that matters to us, rebase
   `xuanji` onto `main`. Resolve conflicts in favor of preserving upstream
   intent for upstream-owned files, while re-applying minimal custom deltas.
4. **Pre-review gates**: `make verify` and `make test` MUST pass locally
   before requesting review; E2E (`make e2e`) for changes touching lifecycle,
   scheduling, or routing paths.
5. **Spec-driven custom features**: custom features follow the Spec Kit flow
   (`/speckit.feature` → `/speckit.requirements` → `/speckit.plan` →
   `/speckit.tasks` → implement) so that feature registry, requirements, and
   plans stay traceable per Principle II.

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
- **Compliance review**: every `/speckit.plan` MUST run the Constitution Check
  against the current principle list; unjustified violations block the plan.
  Templates referencing principles MUST be kept in sync via the Sync Impact
  Report checklist.

**Version**: 1.0.0.1 | **Ratified**: 2026-08-06 | **Last Amended**: 2026-08-06
