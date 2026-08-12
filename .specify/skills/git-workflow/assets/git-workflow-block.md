<!-- Managed block written by the git-workflow skill into the project's canonical
     instructions file (see references/instructions-lookup.md for the lookup order).
     Replace <MAIN> / <PRE> / <DEV> with the confirmed branch names, <MAIN_TRACKING> /
     <PRE_TRACKING> / <DEV_TRACKING> with each branch's upstream (or `-` when none), and
     <DATE> with the run date. Keep the marker comments byte-exact: /speckit.instructions
     preserves this range only by its markers. -->
<!-- GIT_WORKFLOW_START -->
<!-- Record one row per branch role (MAIN / PRE / DEV). While no workflow is established, keep the `None yet.` row. -->
| Role | Branch | Tracking | Purpose |
|------|--------|----------|---------|
| MAIN | `<MAIN>` | `<MAIN_TRACKING>` | 上游主干，只接收已通过版本验证的代码 |
| PRE | `<PRE>` | `<PRE_TRACKING>` | 预发发布分支，用于版本集成与环境验证 |
| DEV | `<DEV>` | `<DEV_TRACKING>` | 本地开发分支，所有新改动先在此开发与自测 |

- **Sync chain (rebase)**: `<MAIN> -> <PRE> -> <DEV>`
- **Merge chain (PR)**: `<MAIN> <- <PRE> <- <DEV>`
- **Last updated**: <DATE>
<!-- GIT_WORKFLOW_END -->
