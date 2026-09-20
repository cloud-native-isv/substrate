# Feature Registry

**Project**: Xuanji Substrate (Agent Substrate fork — custom "E2B 协议面" and "Wasm Sandbox" features on top of upstream)
**Project Type**: Distributed Kubernetes-based runtime platform (Go, control plane + node agents + data plane)
**Delivery Model**: runtime code
**Feature Count**: 20
**Last Reconciled**: 2026-09-20 (panorama-derived additions: 019 MCP Native Capability Backend, 020 Multi-Actor Worker Pod; 014/015 converged to authored design)

Feature classification: **functional** = user-facing capability; **non-functional** = quality attribute (DFX). Status semantics and the canonical status state machine live in each detail file (`.specify/memory/features/<ID>.md`, "Status Tracking" section).

## Functional Features

| ID  | Name                        | Description                                                                                                     | Status      | Feature Details                        | Last Updated |
|-----|-----------------------------|-----------------------------------------------------------------------------------------------------------------|-------------|----------------------------------------|--------------|
| 001 | Actor Lifecycle Management  | Create, delete, suspend, pause, and resume actors with sub-second activation via the control-plane API.          | Implemented | .specify/memory/features/001.md        | 2026-08-06   |
| 002 | Actor Scheduling & Assignment | Real-time assignment of actors to ready workers with caching, taking Kubernetes out of the critical path.      | Implemented | .specify/memory/features/002.md        | 2026-08-06   |
| 003 | Actor Snapshot Persistence  | Full-state snapshots (RAM + filesystem) streamed to GCS/S3 object storage, with tag-based snapshot management.  | Implemented | .specify/memory/features/003.md        | 2026-08-06   |
| 004 | Request Routing & Parking   | Envoy-based traffic routing to actors with xDS control, actor DNS, and request parking during pool saturation.  | Implemented | .specify/memory/features/004.md        | 2026-08-06   |
| 005 | Worker Pool & CRD Controller | Kubernetes controller reconciling ActorTemplate, WorkerPool, and SandboxConfig CRDs, with /scale autoscaling.  | Implemented | .specify/memory/features/005.md        | 2026-08-06   |
| 006 | Multi-Sandbox Runtimes      | gVisor (runsc) and microVM (Kata/Cloud Hypervisor) sandbox herders with checkpoint/restore for isolation.       | Implemented | .specify/memory/features/006.md        | 2026-08-06   |
| 007 | Workload Identity & Authentication | SPIFFE-based workload identity with JWT/cert minting, pod certificate signers, and mTLS data-path verification. | Implemented | .specify/memory/features/007.md | 2026-08-06   |
| 008 | kubectl-ate CLI             | kubectl plugin for actor/atespace/worker management: CRUD, lifecycle, logs, top, snapshots, admin tasks.        | Implemented | .specify/memory/features/008.md        | 2026-08-06   |
| 009 | GCP Provisioning & Install  | Idempotent GCP bootstrap (GKE, GCS, IAM, Monitoring) via tools/setup-gcp plus install scripts and kind dev flow. | Implemented | .specify/memory/features/009.md       | 2026-08-06   |
| 010 | Control Plane State Store   | Redis/Valkey-backed control-plane state store for actor and scheduling metadata.                                | Implemented | .specify/memory/features/010.md        | 2026-08-06   |
| 013 | E2B Protocol Plane          | E2B-compatible protocol surface on Substrate for sandbox lifecycle and interaction APIs. (xuanji custom)         | Draft       | .specify/memory/features/013.md        | 2026-08-06   |
| 014 | Wasm Sandbox Runtime        | WebAssembly sandbox backend plugged into the atelet/ateom lifecycle contract; = the Agent-trust-domain execution body (ADR 0002/0005, Rust agent + WASI). (xuanji custom) | Draft       | .specify/memory/features/014.md        | 2026-09-20   |
| 019 | MCP Native Capability Backend | S-layer native-capability backend (compiler/git/browser) exposed as defined MCP tools, mediated by trusted ateom; cross-S shared pool with per-tenant microvm execution units. (xuanji custom) | Draft       | .specify/memory/features/019.md        | 2026-09-20   |
| 020 | Multi-Actor Worker Pod      | Break "1 worker pod = 1 active actor": one worker concurrently hosts N agent sandboxes of a digital employee (spatial multiplexing). (xuanji custom) | Draft       | .specify/memory/features/020.md        | 2026-09-20   |

## Non-Functional Features

| ID  | Name                        | Description                                                                                                     | Status      | Feature Details                        | Last Updated |
|-----|-----------------------------|-----------------------------------------------------------------------------------------------------------------|-------------|----------------------------------------|--------------|
| 011 | OpenTelemetry Observability | OTel metrics/traces and Prometheus across all components, with shared bootstrap and GCP dashboards. (DFO)       | Implemented | .specify/memory/features/011.md        | 2026-08-06   |
| 012 | E2E & Load Test Suites      | In-cluster e2e suites plus a Locust-based scale benchmarking harness with glutton workloads. (DFT/DFP)          | Implemented | .specify/memory/features/012.md        | 2026-08-06   |
| 015 | Workload Security Hardening | Authorization policies, dependency scanning/SBOM, threat-model hardening; umbrella for the four-trust-domain security ADRs (0003/0004/0006/0007) + cross-domain audit/allowlist (F12). (DFSec) | Draft       | .specify/memory/features/015.md        | 2026-09-20   |
| 016 | Upstream Sync Automation    | Scripted upstream tracking: rebase xuanji onto main, digest upstream changes, surface conflicts. (DFM)          | Draft       | .specify/memory/features/016.md        | 2026-08-06   |
| 017 | Snapshot Storage Extensibility | Pluggable snapshot storage backend abstraction beyond GCS/S3 for self-hosted object stores.                  | Draft       | .specify/memory/features/017.md        | 2026-08-06   |
| 018 | API Compatibility Policy    | Stability/versioning policy for v1alpha1 CRDs and control-plane gRPC API with fork-compat guardrails. (DFC)     | Draft       | .specify/memory/features/018.md        | 2026-08-06   |

## Registry Notes

- IDs 001–012 are upstream-derived capabilities present in the codebase as of
  the 2026-08-06 bootstrap; they stay `Implemented` unless a spec iteration
  reopens them.
- IDs 013–018 are future/gap features (status `Draft`). 013 and 014 come from
  the xuanji roadmap (user-stated); 015–018 come from DFX gap analysis
  (`.specify/shared/constants/dfx-catalog.md`).
- IDs 019–020 (added 2026-09-20) come from the **four-trust-domain panorama**
  (`docs/concepts/trust-domain-panorama.md`) + ADR 0007 + the fork/sandbox split §8
  (F9–F12): 019 = MCP Native Capability Backend (F10/F11/F12 fork side), 020 =
  Multi-Actor Worker Pod (F9). Both `Draft`, both xuanji-custom, sandbox-side
  counterparts are S10–S13. 014/015 converged on 2026-09-20 to reference their
  now-authored design (ADR 0002/0005 for 014; ADR 0003/0004/0006/0007 for 015);
  status unchanged (Draft).
- Per the reconcile model, rows are archived, never deleted; status never
  regresses without explicit user confirmation.
