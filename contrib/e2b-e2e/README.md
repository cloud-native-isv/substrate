# contrib/e2b-e2e — E2B protocol acceptance assets

Migrated verbatim from the wasm-sandbox repository (`cloud-native-webassembly/sandbox`,
`scripts/`) as part of its substrate migration (see that repo's
`docs/tasks/substrate-migration.md`, milestones M3/M4), then repointed at
`cmd/e2bgw` (M3 step 3.4).

## Target

- **Control plane**: e2bgw REST (`POST/GET/DELETE /sandboxes`, `pause`/`resume`/
  `timeout`) — served by `cmd/e2bgw`, port-forward with
  `kubectl -n ate-system port-forward svc/e2bgw 8080:80`.
- **Auth**: HS256 JWT in `X-API-Key`, `atespace` claim (mint with `sign_jwt.py`;
  the secret must equal the gateway's `--jwt-secret-file`, in-cluster Secret
  `e2bgw-jwt-secret`).
- **Data plane (path-based)**: `POST /sandboxes/{id}/execute`, `/files`,
  `/filesystem.Filesystem/*` are reverse-proxied by e2bgw to the actor's atenet
  domain (M4 4.2).
- **Data plane (host-based)**: unmodified E2B SDKs address the data plane as
  `https://{port}-{sandboxID}.{domain}`. e2bgw intercepts that host shape
  before its REST mux and proxies to the actor authority; the caller is
  authenticated by the `envdAccessToken` returned from create/get/resume
  (SDKs echo it as `X-Access-Token`), whose claims pin the atespace and the
  sandbox ID. The port label is advisory (one actor HTTP surface behind
  atunnel :443).

## Asset status

| Asset | Purpose | e2bgw-ready? |
|---|---|---|
| `sign_jwt.py` | Per-atespace HS256 API-key minting | ✅ repointed (`atespace` claim) |
| `test_sdk_nondebug_e2e.py` | Unmodified official E2B SDK end-to-end | ✅ full run (host-based data plane wired); `E2E_LIFECYCLE_ONLY=1` kept as a debugging aid |
| `sandbox_tools.py` | Shared client helpers (smolagents tools) | ✅ repointed (path-based `/sandboxes/{id}/execute` + `X-API-Key`) |
| `e2b_tls_relay.py` | Local TLS :443 → gateway TCP relay | ✅ protocol-agnostic (point upstream at e2bgw :8080) |
| `wildcard_dns_stub.py`, `gen_sslip_cert.sh` | Wildcard-domain plumbing | ✅ protocol-agnostic |
| `test_files_api_e2e.py` | E2B files API end-to-end | ✅ host-based data plane wired — run as-is on cluster (M4 4.4) |
| `migrate_sandbox.py` | Cross-tier migration: fresh sandbox on target template + full files relocation (verified) + retire old | ✅ public E2B surface only (create/files/kill); tiers in `manifests/xuanji/wasm-tiers-example.yaml` |
| `run_smolagent.py` / `run_smolagent_interactive.py` | Real agent workload via E2B | ✅ via `sandbox_tools.py` (needs a pre-created sandbox) |
| `e2b_mcp_server.py` | MCP server exposing the sandbox API | ⏳ M4 4.4 |
| `demo_agent_cli.py`, `demo_forms.py` | Interactive demos | ⏳ M4 4.4 |
| `test_epoch.py`, `test_sandbox_identity.py` | Behavioural regression checks | ❌ legacy `E2B_DEBUG=true` mode; e2bgw has no debug singleton — rework or retire at M4 4.4 |

## Quick start (cluster e2e)

```sh
# 1. Mint an API key (secret = the e2bgw-jwt-secret value)
export E2B_API_KEY=$(python3 sign_jwt.py --atespace team-a --secret <hex>)

# 2. Expose e2bgw and terminate TLS for the SDK's default https endpoints
kubectl -n ate-system port-forward svc/e2bgw 8080:80 &
sudo RELAY_UPSTREAM_PORT=8080 python3 e2b_tls_relay.py &

# 3. Run the SDK e2e (full: control plane + host-based data plane)
export E2B_DOMAIN=127.0.0.1.sslip.io E2B_VALIDATE_API_KEY=false \
       SSL_CERT_FILE=.secrets/sslip-cert.pem \
       E2B_TEMPLATE=python-interpreter
python3 test_sdk_nondebug_e2e.py
python3 test_files_api_e2e.py
```

`E2B_TEMPLATE` must name an ActorTemplate in e2bgw's `--template-namespace`
(default `ate-wasm`; `manifests/xuanji/wasm-example.yaml` provides
`python-interpreter`; `manifests/xuanji/wasm-tiers-example.yaml` adds the
XL capacity tier `python-interpreter-xl` — move an existing sandbox across
tiers with `migrate_sandbox.py <sandbox_id> --template python-interpreter-xl`).
