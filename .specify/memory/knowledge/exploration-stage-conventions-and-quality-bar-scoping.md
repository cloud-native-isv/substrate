---
id: "exploration-stage-conventions-and-quality-bar-scoping"
scope: "knowledge"
source: "/speckit.instructions"
tags: ["convention", "decision", "preference", "constitution", "quality-bar"]
title: "Exploration-stage conventions and quality-bar scoping"
created: "2026-08-12T03:21:13Z"
summary: "Durable conventions confirmed by the user (2026-08-12) and now binding on future refreshes:"
---

Durable conventions confirmed by the user (2026-08-12) and now binding on future refreshes:

1. Project nature: technical exploration in a three-stage roadmap (absorb upstream -> customize -> redefine). User declared CURRENT STAGE = Stage 1 (absorb-dominant), even though cmd/e2bgw and the wasm sandbox class already exist (they are Stage-2 pilots).
2. Quality bar: "functionality first, no high quality bar" was scoped precisely by user choice — tests demoted to SHOULD (code without tests MAY merge; Tests Mode default OFF), but MECHANICAL gates are KEPT mandatory (build, make verify, license headers, go.mod tidy) because they protect Principle I rebase health. Do not relax mechanical gates; do not re-impose TDD.
3. Docs publish scope: docs/ doubles as a Hugo site root with public remotes, so internal/confidential source material MUST NOT be placed under docs/. Internal collected materials were distilled to public-source-only docs and the originals removed.
4. Principle numbering: new constitution principles are APPENDED (not inserted) to keep existing citations (I, VIII, XI) valid in .specify/instructions.md and .specify/memory/features/*.md.
5. Known framework-inherited defect: several shipped Spec Kit files cite "Constitution Principle XII" for the tool-reuse gate; this project's XII is "Complete & Correct Options, or None" and it declares NO tool-reuse principle. Cite .specify/shared/workflow/tool-reuse-gate.md instead of a principle number.
