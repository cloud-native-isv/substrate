---
id: "20260806T064222Z-speckit-instructions"
unit_id: "/speckit.instructions"
unit_type: "command"
run_id: "speckit-instructions-20260806T1437"
scope: "local"
partial: false
created: "2026-08-06T06:42:22Z"
summary: "Bootstrap run: setup script succeeded (success-with-warnings) — it converted the tracked upstream AGENTS.md into a symlink, which I restored from git HEAD and documented as an exception in the symlink"
---

## Review
Bootstrap run: setup script succeeded (success-with-warnings) — it converted the tracked upstream AGENTS.md into a symlink, which I restored from git HEAD and documented as an exception in the symlink-model section. Section refresh, glossary seeding (14 terms), memory records all completed cleanly.

## Optimization Points
- generate-instructions.sh unconditionally symlinks AGENTS.md -> .specify/instructions.md (ln -sf, ~line 152) with no backup. In fork-style repos where AGENTS.md is a git-tracked, upstream-owned real file, this silently typechanges it (git shows "T AGENTS.md") and, on macOS case-insensitive filesystems, also produces a colliding lowercase agents.md symlink. Suggest: skip symlink creation when AGENTS.md is a tracked regular file (or write a timestamped backup first), and honor an opt-out (e.g. env var or a marker in instructions.md) so fork repos can keep AGENTS.md authoritative.
