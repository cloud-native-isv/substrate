# [PROJECT_NAME] Constitution
<!--
This is a PRESET template containing a set of generic project principles
and workflow guidelines. It is intended as a starting point only.

When adopting this constitution for a specific project, you MUST review,
add, remove, or modify principles and sections to match the actual
context, risk profile, and organizational policies of your project.
Do not assume this template is sufficient without customization.
-->

## Core Principles

### I. Library-First Design
Every significant feature MUST begin as a cohesive, reusable library (module/package).
Libraries MUST:
- Be self-contained and independently testable.
- Have a single, clearly documented responsibility.
- Avoid being mere organizational/wrapper shells without real behavior.

Rationale: encourages reuse, clear boundaries, and easier testing.

### II. CLI & Text I/O Interface
Each library SHOULD expose a command-line interface (CLI) for core operations.
CLIs MUST:
- Accept input via stdin/arguments/files using plain text or JSON.
- Write normal results to stdout and errors to stderr.
- Prefer JSON for machine consumption and human-readable text for operators.

Rationale: standardizes integration, observability, and automation.

### III. Documentation-First
Documentation is a first-class deliverable and MUST take priority over test coverage:
- The project MUST maintain sufficiently detailed and accurate documentation so it can
  serve as reliable context for AI agents / large language models, supplying project
  knowledge and background information.
- Documentation MUST NOT record implementation details; implementation details belong in
  the code itself.
- Documents MUST stay focused and reasonably sized; when a document becomes too complex it
  MUST be split into smaller, cohesive documents.
- Documents MUST cross-reference one another so that basic navigation can be accomplished
  purely through internal links.
- Markdown documents MUST maintain basic metadata (e.g. title, purpose/summary, status,
  last-updated date, and related links).

Rationale: As agent capabilities grow, high-quality documentation becomes the primary
knowledge context that lets LLMs understand a project accurately and reason effectively;
it therefore ranks above testing in the principle hierarchy.

### IV. Test-First Development
Implementation MUST follow a Test-Driven Development style for core logic:
- Write or update tests BEFORE implementing new behavior.
- Ensure tests FAIL first (Red), then implement to make them PASS (Green).
- Refactor only with all tests passing (Refactor).

At minimum:
- Pure functions/utilities MUST have unit tests.
- Critical flows MUST have automated regression coverage.

Rationale: reduces regressions and clarifies intent.

### V. Integration & Contract Testing
Integration/contract tests SHOULD cover:
- Cross-service communication and external APIs.
- Shared schemas or data contracts.
- Critical end-to-end user journeys.

When real dependencies are hard to run locally, abstract them behind interfaces
and document follow-up contract tests in the plan/tasks.

Rationale: validates real-world behavior beyond unit tests.

### VI. Observability, Versioning & Simplicity
All components MUST be observable and versioned:
- Use structured logs for important events and errors.
- Prefer semantic versioning (MAJOR.MINOR.PATCH).
- Document any breaking changes and migration notes.
- Keep designs as simple as possible; avoid speculative features (YAGNI).

Rationale: makes systems debuggable, upgradable, and maintainable.

### VII. Continuous Integration & Quality Gates
Changes MUST be safe to merge:
- Linting, formatting, and basic tests MUST pass in CI.
- A minimal smoke test or example run SHOULD be provided for new features.
- New behavior MUST be reflected in specs/plan/tasks/docs where applicable.

Rationale: ensures consistent quality and predictable releases.

### VIII. Feature-Centric Development
Feature is the long-term core framework of the project:
- The Feature list MUST remain the "single source of truth" for the project.
- Every phase of spec → plan → tasks → implement MUST review Feature additions/merges/splits/deletions.
- Feature changes MUST be traceable to corresponding spec/plan evidence and recorded in the Feature detail.

Rationale: Keep project evolution Feature-centric to ensure long-term consistency and maintainability.

### IX. Better-Harness Orientation
The project is a *harness* for AI agent work — an environment in which an agent can
understand the task, execute on supported and repeatable paths, validate its changes,
deliver safely, and carry lessons forward. Improvement work MUST be oriented toward
making that harness better:
- Locate and motivate improvements against the five Agent Work Loop dimensions
  (Task Understanding, Controlled Execution, Change Validation, Reliable Delivery,
  Learning Capture); the canonical goal model is `.specify/shared/guidelines/better-harness.md`
  — reference it, do not restate it.
- Evidence discipline governs improvement claims: a configured asset proves at most that a
  mechanism exists (configured ≠ used); unobserved evidence MUST NOT be treated as a defect
  or a conclusion; "improved" MUST only be claimed from comparable before/after evidence.
- This principle adds orientation, not machinery: it MUST NOT justify new scoring systems,
  maturity reports, or tracking/recording engines.

Rationale: Like Dogfooding, Better Harness is a core mindset of the agent era — agents
change code fast, but the workflow around them (fuzzy goals, improvised steps, unproven
"it works", bypassed safeguards, lost lessons) is usually the weak point. Naming the goal
lets every improvement answer "which part of the harness does this strengthen?".

### X. Non-Destructive Remediation
Problems MUST be fixed at their root cause, never by destroying the artifact that
exhibits them:
- Deleting files, code, or data MUST NOT be used as a shortcut to make an immediate
  problem disappear; a "fix" that works by destruction proves the approach is wrong,
  not that the problem is solved.
- When an obstacle is encountered, the underlying cause MUST be identified and repaired;
  bypassing safety checks or removing the surface that exposes the problem is forbidden.
- When a destructive operation is genuinely required (deletion, force-overwrite, reset),
  it MUST be explicitly confirmed with the user before execution.

Rationale: destructive shortcuts destroy information and trust, and usually recreate the
same problem elsewhere; mandatory confirmation keeps irreversible actions under human control.

### XI. Complete & Correct Options, or None
Presenting options that are incomplete or incorrect is WORSE than presenting no options:
- Any option set offered to the user (choices, plans, recommendations) MUST be verified
  correct and MUST cover the realistic decision space.
- When correctness or completeness cannot be ensured, the open question MUST be presented
  directly instead of a partial or speculative option list.
- A known-imperfect option MUST be labeled with its gap rather than presented as solid.

Rationale: wrong or partial options mislead decisions more than no guidance at all — they
manufacture false confidence and foreclose alternatives the user never saw.

## [SECTION_2_NAME]
<!-- Example: Additional Constraints, Security Requirements, Performance Standards, etc. -->

[SECTION_2_CONTENT]
<!-- Example: Technology stack requirements, compliance standards, deployment policies, etc. -->

## [SECTION_3_NAME]
<!-- Example: Development Workflow, Review Process, Quality Gates, etc. -->

[SECTION_3_CONTENT]
<!-- Example: Code review requirements, testing gates, deployment approval process, etc. -->

## Governance
<!-- Projects SHOULD refine this to match their org/governance needs. -->

[GOVERNANCE_RULES]
<!-- Example: Constitution supersedes other guidelines; Amendments require proposal,
review, and version bump; All PRs MUST check compliance with core principles. -->

**Version**: [CONSTITUTION_VERSION] | **Ratified**: [RATIFICATION_DATE] | **Last Amended**: [LAST_AMENDED_DATE]
<!-- Example: Version: 1.0.0 | Ratified: 2025-01-01 | Last Amended: 2025-01-01 -->
