---
kind: error_handling
name: gRPC AIP-193 错误模型与拦截器统一错误处理
category: error_handling
scope:
    - '**'
source_files:
    - internal/ateerrors/ateerrors.go
    - internal/ateinterceptors/ateinterceptors.go
    - cmd/ateapi/internal/actoridentity/actoridentity.go
    - internal/resources/validate.go
    - cmd/ateapi/internal/controlapi/crash.go
---

## 1. 采用的错误系统

仓库采用 **gRPC + AIP-193 ErrorInfo** 作为跨进程、跨组件的统一错误模型，核心在 `internal/ateerrors` 包。

- 每个业务失败通过 `ateerrors.NewGRPCError(ctx, code, reason, metadata, err)` 构造，将 `google.rpc.ErrorInfo`（`domain=substrate.dev`、`reason` 枚举、结构化 `metadata`）附加到 gRPC status 上，使调用方能按 AIP-193 规范解析原因。
- `Reason` 是受控的 UPPER_SNAKE_CASE 枚举（如 `INVALID_SANDBOX_ASSET`、`WORKER_POD_GONE`、`CORRUPTED_ASSIGNMENT`），通过 `AllReasons` 列表校验；未分类的错误以 `UNSET` 标记。
- 控制面通过 `CrashIfReason` 在 RPC 边界“认领”特定 Reason，将其升级为带 `actorCrashed=true` 元数据的 `DataLoss` 错误，从而指示控制面必须终止 Actor；`ActorCrashRequested` 用于下游读取该指令。
- 所有 gRPC 服务端使用 `ateinterceptors.ServerUnaryInterceptor` / `InternalServerUnaryInterceptor`：记录请求/响应/耗时/主体信息，并将非 status 错误兜底为 `codes.Internal`，保证对外只暴露 gRPC status。

## 2. 关键文件与位置

| 文件 | 职责 |
|---|---|
| `internal/ateerrors/ateerrors.go` | AIP-193 ErrorInfo 构建、Reason 枚举、Actor 崩溃指令 |
| `internal/ateinterceptors/ateinterceptors.go` | gRPC 服务器端拦截器（日志、超时、status 规范化） |
| `cmd/ateapi/internal/controlapi/*.go` | 各 RPC handler 返回 `fmt.Errorf` 或 `status.Errorf`，由拦截器统一处理 |
| `cmd/ateapi/internal/actoridentity/*.go` | 认证/鉴权分支直接返回 `Unauthenticated` / `PermissionDenied` |
| `internal/resources/validate.go` | CRD/请求字段验证，返回 `field.ErrorList`（Kubernetes 风格结构化字段错误） |
| `pkg/api/v1alpha1/*_validation_test.go` | 针对 CRD 类型的验证测试 |

## 3. 架构与约定

### 3.1 错误分类与传播
- **业务层**：handler 内部用 `fmt.Errorf("while ...: %w", err)` 包装底层错误，保留链式上下文；对需要控制面特殊处理的失败，使用 `ateerrors.NewGRPCError` 并携带 Reason。
- **RPC 边界**：`ateinterceptors` 拦截器统一把任何非 `*status.Status` 的错误转换为 `codes.Internal`，避免泄露内部细节；同时注入 `x-server-elapsed-us` trailer 供客户端度量。
- **控制面决策**：上层通过 `ateerrors.ActorCrashRequested(err)` 检查是否要求 crash Actor，实现“错误即指令”的语义。

### 3.2 输入验证
- 所有外部可配置的字符串（资源名、UID、容器名、URI 前缀等）在 `internal/resources` 中集中验证，返回 Kubernetes `field.ErrorList`，由控制器/CRD webhook 转为 API 错误。
- 安全相关验证（如 `ValidateAteomUID`、`ValidateRunscHash`、`ValidateSnapshotURIPrefix`）在 RPC 入口尽早失败，防止路径穿越或缓存污染。

### 3.3 认证与鉴权错误
- `actoridentity` 模块对缺失授权头、证书校验失败、权限不足分别返回 `Unauthenticated` / `InvalidArgument` / `PermissionDenied`，并在测试中显式断言期望的 gRPC code。

## 4. 约定与约束

| 约定 | 说明 | 依据 |
|---|---|---|
| 禁止对 `NewGRPCError` 传入 `nil` 错误或 `codes.OK` | 函数会主动 `fmt.Errorf` 报错，强制调用方区分成功与失败 | `ateerrors.go` L98-101 |
| 所有 Reason 必须加入 `AllReasons` | 新增 Reason 时注释明确要求同步更新列表 | `ateerrors.go` L45 |
| 外部错误不得直接透传 | 拦截器会将非 status 错误包装为 `codes.Internal` | `ateinterceptors.go` L61-72, L101-112 |
| 认证失败统一走 gRPC code | 不使用自定义错误类型，直接 `status.Errorf(codes.Unauthenticated|PermissionDenied, ...)` | `actoridentity.go` 多处 |
| 资源名遵循 DNS-1123 label 子集 | 通过 `ValidateResourceName` / `IsValidResourceName` 统一校验 | `resources/validate.go` L29-48 |
| 常量级网络配置解析失败使用 panic | 因为值来自源码常量，不应发生运行时错误 | `ateomnet/net.go` 多处 `panic(fmt.Sprintf(...))` |
| 环境变量从日志中脱敏 | 拦截器在记录前递归清除 proto 中的 `env` 字段 | `ateinterceptors.go` L117-151 |

## 5. 总结

该项目没有使用 Go 原生的 sentinel error 模式作为对外契约，而是以 **gRPC status + AIP-193 ErrorInfo** 为核心，配合 `ateerrors.Reason` 枚举和 `CrashIfReason` 机制，实现了“错误即控制指令”的跨进程协议。所有服务端的日志、超时、status 规范化集中在 `ateinterceptors`，业务层只需关注语义化的 Reason 与结构化 metadata，控制面据此决定重试、降级或终止 Actor。CRD/请求字段的输入验证则独立于 gRPC 层，使用 Kubernetes `field.ErrorList` 提供细粒度的字段级错误定位。