# E2B 接口面 × e2bgw 覆盖矩阵（精确参考）

> `cmd/e2bgw` 是 xuanji 分支的 E2B 协议网关：把 E2B REST 接口翻译成 `ateapipb.Control` gRPC 调用，
> 让未改动的 E2B SDK 能直接驱动 substrate actor。本文以代码核实（`cmd/e2bgw/internal/server/server.go`）
> 逐端点标注覆盖状态，作为网关的验收基线。E2B 接口面语义来自 E2B 官方公开文档/SDK。
>
> 相关：网关设计登记见仓库根 `xuanji.md`；验收资产 `contrib/e2b-e2e/`；竞品对比见
> [../concepts/sandbox-landscape.md](../concepts/sandbox-landscape.md)。

---

## 1. 映射模型

E2B 的 **Sandbox** 概念映射到 substrate 的 **Actor**：

| E2B 概念 | substrate 概念 | 说明 |
|---|---|---|
| Sandbox | Actor | 一个可 suspend/resume 的实例；网关生成 `sbx-<hex>` 名字 |
| Template（templateID） | ActorTemplate | 网关按 `--template-namespace`（默认 `ate-wasm`）解析模板名 |
| API Key | HS256 JWT | `X-API-Key` 或 `Authorization: Bearer`；claim `atespace`（兼容旧 `tenant`）→ actor 租户边界 |
| 数据面（/execute /files） | atenet actor 域名 | 网关 307 跳转到 `<name>.<atespace>.<actor-domain>`（需 `--actor-domain`） |

鉴权：所有 `/sandboxes*` 端点经 `auth` 中间件校验 JWT；无 token → 401，无 `atespace`/`tenant` claim → 401
（`server.go:87-113`）。

## 2. 控制面端点覆盖

| E2B REST | e2bgw 状态 | 映射到 Control RPC | 备注 |
|---|:---:|---|---|
| `POST /sandboxes` | ✅ 已实现 | `CreateActor` → `ResumeActor{Boot:true}` | Resume 失败回滚 `DeleteActor`（避免泄漏 actor）；返回 201 |
| `GET /sandboxes` | ✅ 已实现 | `ListActors`（分页循环 `PageToken`） | 按 token 的 atespace 过滤 |
| `GET /sandboxes/{id}` | ✅ 已实现 | `GetActor` | — |
| `DELETE /sandboxes/{id}` | ✅ 已实现 | `DeleteActor` | 返回 204 |
| `POST /sandboxes/{id}/pause` | ✅ 已实现 | `SuspendActor` | 返回 204 |
| `POST /sandboxes/{id}/resume` | ✅ 已实现 | `ResumeActor` | — |
| `POST /sandboxes/{id}/timeout` | ⚠️ 占位 | —（返回 204） | 仅 ACK；TTL→Suspend 调度为后续项 |
| `GET /health` | ✅ 已实现 | —（返回 200） | 网关存活探针 |

## 3. 数据面端点

| E2B REST | e2bgw 状态 | 行为 |
|---|:---:|---|
| `/sandboxes/{id}/execute` | 🔶 跳转（M4 接通） | 配了 `--actor-domain` → 307 到 `https://<id>.<atespace>.<domain>/...`；否则 501 |
| `/sandboxes/{id}/files` | 🔶 跳转（M4 接通） | 同上 |

进程/文件系统/Code Interpreter 的实际数据面由 actor 内的 in-actor HTTP 面（ateom-wasmd）承接，
不经网关代理，仅由网关做重定向。

## 4. 状态映射

`e2bState`（`server.go:275-285`）把 `ateapipb.Actor_Status` 折叠为 E2B 的 `running`/`paused`：

| Actor_Status | E2B state |
|---|---|
| `RUNNING` / `RESUMING` | `running` |
| `SUSPENDED` / `SUSPENDING` / `PAUSED` / `PAUSING` | `paused` |
| 其他 | 小写去 `STATUS_` 前缀 |

## 5. gRPC → HTTP 错误映射

`writeGRPCErr`（`server.go:306-337`）：

| gRPC code | HTTP |
|---|---|
| `NotFound` | 404 |
| `AlreadyExists` / `FailedPrecondition` | 409 |
| `InvalidArgument` | 400 |
| `PermissionDenied` | 403 |
| `Unauthenticated` | 401 |
| `ResourceExhausted` | 429 |
| `Unavailable` | 503 |
| 其他 / 非 status err | 502 |

## 6. 尚未覆盖（E2B 有、网关未实现）

以下 E2B 接口面本网关**当前未提供**，如需对齐再评估：Metrics/Logs 查询、Snapshots 独立端点、
Volume 管理、Template CRUD/Build、API-Key/Access-Token/Team 管理。网关聚焦 Sandbox 生命周期核心链路。

## 7. 运行参数速查

`cmd/e2bgw/main.go`：`--listen-address`(0.0.0.0:8080) · `--ateapi-conn-spec` · `--jwt-secret-file`（必填，
HS256 密钥） · `--template-namespace`(ate-wasm) · `--actor-domain`（空则禁用数据面跳转） · ateapi TLS
相关（`--ateapi-ca-file`/`--ateapi-server-name`/`--ateapi-token-file`/`--ateapi-client-cred-bundle`）。
