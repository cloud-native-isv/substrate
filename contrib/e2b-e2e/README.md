# contrib/e2b-e2e — E2B protocol acceptance assets (migrated)

Migrated verbatim from the wasm-sandbox repository (`cloud-native-webassembly/sandbox`,
`scripts/`) as part of its substrate migration (see that repo's
`docs/tasks/substrate-migration.md`, milestone M3).

These scripts currently target the **legacy Rust gateway API** (axum, deleted from the
source repo; recoverable from its git history). They are the acceptance baseline for
`cmd/e2bgw` (Go rewrite of the E2B gateway, M3): once e2bgw serves the same E2B REST
surface, repoint these at it and they must pass unchanged.

| Asset | Purpose |
|---|---|
| `test_sdk_nondebug_e2e.py` | Unmodified official E2B SDK end-to-end (create/exec/kill) |
| `test_files_api_e2e.py` | E2B files API end-to-end |
| `run_smolagent.py` / `run_smolagent_interactive.py` | Real agent workload (smolagents) via E2B |
| `e2b_mcp_server.py` | MCP server exposing the sandbox API as tools |
| `e2b_tls_relay.py` | Local TLS :443 → gateway TCP relay for SDK default endpoints |
| `sign_jwt.py` | Per-tenant HS256 JWT minting |
| `sandbox_tools.py` | Shared client helpers |
| `test_epoch.py`, `test_sandbox_identity.py` | Behavioural regression checks |
| `wildcard_dns_stub.py`, `gen_sslip_cert.sh` | Wildcard-domain plumbing for data-plane routing |
| `demo_agent_cli.py`, `demo_forms.py` | Interactive demos |
