---
kind: error_handling
name: 错误处理体系：AIP-193 gRPC 状态码与 Reason 分类
slug: error_handling
category: error_handling
scope:
    - '**'
---

## 系统概述

Agent Substrate 采用基于 Google AIP-193 的 gRPC 错误处理体系，通过 `internal/ateerrors` 包统一构建、传播和解析错误。核心设计围绕三个概念：**Reason（失败原因枚举）**、**ErrorInfo 元数据（actorCrashed 指令）**、**gRPC Status Code** 的组合，实现跨控制面与工作负载的错误分类与响应策略。

## 核心组件与文件

- **`internal/ateerrors/ateerrors.go`**：错误定义与构造的核心包，提供 `NewGRPCError`、`CrashIfReason`、`ActorCrashRequested` 等关键 API
- **`internal/ateinterceptors/ateinterceptors.go`**：gRPC 拦截器，统一处理错误日志、状态码转换和环境字段脱敏
- **`cmd/ateapi/internal/controlapi/crash.go`**：控制面错误处理逻辑，根据 `actorCrashed=true` 指令触发 Actor 崩溃
- **`cmd/atelet/main.go`**：节点侧工作负载编排，使用 `CrashIfReason` 标记需要崩溃的错误类型

## 架构设计与约定

### 错误分类体系

项目定义了有限的 Reason 枚举用于分类失败原因：
- `TERMINAL_FILE_SYSTEM_ERROR`：文件系统不可恢复错误
- `INVALID_SANDBOX_ASSET`：沙箱资源无效
- `INVALID_CHECKPOINT_RESULT`：检查点结果异常
- `FAILED_SAVE_SNAPSHOT`：快照保存失败
- `INVALID_OBJECT_URL`：对象 URL 无效
- `FAILED_GET_EXTERNAL_OBJECT`：外部对象获取失败
- `INVALID_CONTAINER_CONFIG`：容器配置无法运行

### 错误构造与传播

1. **结构化错误构造**：通过 `NewGRPCError(ctx, code, reason, metadata, err)` 创建包含 ErrorInfo 的详细错误
2. **条件崩溃标记**：`CrashIfReason(ctx, err, reasons...)` 将特定 Reason 包装为带 `actorCrashed=true` 元数据的 DataLoss 错误
3. **错误检测**：`ActorCrashRequested(err)` 检查错误链中是否包含崩溃指令

### gRPC 拦截器处理

`ServerUnaryInterceptor` 和 `InternalServerUnaryInterceptor` 提供统一的错误处理：
- 自动记录 RPC 请求/响应/错误信息
- 将非 status 错误转换为 `codes.Internal`
- 对敏感字段（如 env）进行脱敏处理
- 添加服务器耗时追踪 trailer

### 控制面崩溃决策

`maybeCrashActor` 函数在控制面接收 atelet 返回的错误后：
- 检查是否包含 `actorCrashed=true` 元数据
- 如果是，调用 `crashActor` 将 Actor 状态置为 CRASHED 并释放 Worker
- 返回 `codes.DataLoss` 给客户端

## 约束与规范

1. **禁止 OK 状态码**：`NewGRPCError` 会拒绝 `codes.OK` 或 nil error 参数
2. **Reason 必须显式声明**：未指定 Reason 时默认为 "UNSET"
3. **崩溃指令按调用点声明**：同一错误在不同 RPC 中可被不同地处理
4. **错误链必须可追溯**：使用 `%w` 包装底层错误以便 `errors.Is/As` 匹配
5. **panic 仅用于常量解析**：项目中 panic 主要用于启动时常量解析失败（如 CIDR、MAC 地址），运行时错误应返回而非 panic
6. **无 recover 机制**：代码库中未发现 `recover()` 使用，表明不依赖 panic/recover 进行错误恢复

## 与其他错误的混合使用

- **认证错误**：直接使用 `status.Errorf(codes.Unauthenticated, ...)` 
- **权限错误**：使用 `codes.PermissionDenied` 和 `codes.FailedPrecondition`
- **内部错误**：未包装的错误会被拦截器转换为 `codes.Internal`
- **Kubernetes 控制器**：使用标准 Go error 模式，通过 controller-runtime 的 reconcile 循环处理

该体系确保了跨进程边界的错误语义一致性，使控制面能够基于结构化错误信息做出精确的故障恢复决策。