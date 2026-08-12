# Project Glossary (项目词汇表)

> **Note**: This file is initialized by `/speckit.instructions` and lives beside `constitution.md` / `features.md`. It is the project's single, project-wide vocabulary anchor: it corrects voice/dictated input (homophones, easily-confused words) and doubles as a lightweight domain-knowledge dictionary. It is loaded as ambient context by every `/speckit.*` command via the Documentation Map. See `.specify/shared/workflow/glossary.md` for the correction / enrichment / conflict protocol.

## Authoring Rules

- **Common words are NOT recorded** — only project-specific / domain terms that carry special meaning here.
- **User edits are authoritative (以用户输入为准)** — manual entries win over automatic proposals and are preserved across regenerations; automatic proposals MUST NOT silently overwrite a `user` entry.
- **Conflicts require confirmation** — a new term that collides with an existing entry (same term/different meaning, or a homophone/near-duplicate) is written only after the user confirms the resolution.

## Column Definitions

| Column | Meaning |
|--------|---------|
| Canonical | The agreed project term (unique, case-insensitive). |
| Variants | Comma-separated homophones / easily-confused / dictation-error forms that anchor back to Canonical; `-` when none. |
| Meaning | Brief one-line domain definition. |
| Origin | `auto` (framework-proposed) or `user` (manually authored/confirmed). |
| Status | `proposed` (awaiting confirmation) or `confirmed`. |

## Glossary

| Canonical | Variants | Meaning | Origin | Status |
|-----------|----------|---------|--------|--------|
| Actor | - | Application (e.g. an agent) managed by Substrate; multiplexed onto worker Pods. | auto | proposed |
| Atespace | - | Namespace/isolation resource that groups actors; must exist before creating actors. | auto | proposed |
| Worker | - | Ready Kubernetes Pod that hosts actors; assigned in real time by the scheduler. | auto | proposed |
| WorkerPool | - | CRD managing a pool of workers; exposes /scale for Kubernetes autoscaling. | auto | proposed |
| ActorTemplate | - | CRD defining an actor workload (image, resources, sandbox config). | auto | proposed |
| ateapi | ate-api, ate-api-server | Control-plane API server binary exposing Control/Debug/ActorIdentity gRPC services. | auto | proposed |
| atelet | - | Per-node DaemonSet supervisor managing worker pods and snapshot streaming. | auto | proposed |
| atenet | - | Networking daemon: actor DNS plus Envoy router control plane. | auto | proposed |
| ateom | ateom-gvisor, ateom-microvm | In-pod sandbox herder running workloads under runsc or Cloud Hypervisor. | auto | proposed |
| xuanji | 璇玑, Xuanji | Custom development branch/project name for this fork (main tracks upstream; xuanji rebases onto it). | auto | proposed |
| E2B Protocol Plane | E2B协议面, e2b | Planned xuanji feature 013: E2B-compatible protocol surface over Substrate sandboxes. | auto | proposed |
| Wasm Sandbox | WebAssembly sandbox | Planned xuanji feature 014: WebAssembly sandbox backend in the atelet/ateom lifecycle. | auto | proposed |
| request parking | - | Router behavior of holding inbound requests until a worker frees up instead of returning 503. | auto | proposed |
| snapshot | - | Full-state (RAM + filesystem) actor capture stored in GCS/S3; basis of suspend/resume. | auto | proposed |
| e2bgw | - | Xuanji-only gateway binary (cmd/e2bgw) translating the E2B REST surface into ateapi Control gRPC calls. | auto | proposed |
| golden snapshot | - | Shared pre-warmed snapshot of an ActorTemplate's first boot, used as the restore source for new actors. | auto | proposed |
| DurableDir | - | Actor-scoped persistent directory whose contents survive suspend/resume cycles. | auto | proposed |
| AgentENV | - | Kimi/kvcache-ai open-source Firecracker microVM sandbox platform; primary public comparison target with E2B-compatible API. | auto | proposed |
| three-stage roadmap | - | Constitution's exploration model: (1) absorb upstream, (2) customize, (3) redefine; currently Stage 1. | auto | proposed |
