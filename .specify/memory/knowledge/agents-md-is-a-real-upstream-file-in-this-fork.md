---
id: "agents-md-is-a-real-upstream-file-in-this-fork"
scope: "knowledge"
source: "/speckit.instructions"
tags: ["convention", "symlink", "agents-md", "fork"]
title: "AGENTS.md is a real upstream file in this fork"
created: "2026-08-06T06:42:09Z"
summary: "In this repo the root AGENTS.md is the upstream Agent Substrate project guide — git-tracked real file, NOT a symlink to .specify/instructions.md. generate-instructions.sh will try to convert it (ln -s"
---

In this repo the root AGENTS.md is the upstream Agent Substrate project guide — git-tracked real file, NOT a symlink to .specify/instructions.md. generate-instructions.sh will try to convert it (ln -sf at line ~152); if it happens, restore with 'git checkout HEAD -- AGENTS.md' (on macOS case-insensitive FS remove the lowercase agents.md symlink first). Only CLAUDE.md/QODER.md/QWEN.md/.github/copilot-instructions.md are compatibility symlinks.
