---
kind: logging_system
name: 基于 slog + OpenTelemetry 的结构化日志系统
category: logging_system
scope:
    - '**'
source_files:
    - internal/serverboot/serverboot.go
    - internal/contextlogging/contextlogging.go
    - internal/actorlog/logger.go
    - cmd/ateom-gvisor/main.go
    - cmd/ateom-microvm/main.go
---

## 1. 使用的框架与工具

- **标准库 `log/slog`**：仓库统一使用 Go 1.21+ 的 `slog` 作为结构化日志框架，所有服务通过 JSON Handler 输出 JSON 格式日志。
- **OpenTelemetry (OTel)**：追踪（TracerProvider）和指标（MeterProvider）通过 OTLP gRPC 导出；日志与追踪通过 `trace.SpanContextFromContext` 关联。
- **Prometheus**：指标通过 Prometheus reader 暴露 `/metrics` HTTP 端点。
- **自定义 Handler**：`internal/contextlogging` 提供 `ContextHandler`，在每条日志记录上自动注入 `ate.dev/trace-id` 字段。
- **GCE/GKE 适配**：`internal/actorlog` 根据是否运行在 GCE 上切换标签键名（`labels` vs `logging.googleapis.com/labels`），使容器 stdout/stderr 可直接被 Google Cloud Logging 消费。

## 2. 关键文件与包

| 路径 | 作用 |
|---|---|
| `internal/serverboot/serverboot.go` | 所有长生命周期二进制（ateapi、atelet、ateom-gvisor、ateom-microvm）共享的启动引导：初始化 slog、OTel Tracing/Metrics、/metrics 与 /readyz |
| `internal/contextlogging/contextlogging.go` | `ContextHandler`：从上下文提取 trace ID 并注入到每条日志的 `ate.dev/trace-id` 字段 |
| `internal/actorlog/logger.go` | `ActorLogger`：拦截沙箱容器的 stdout/stderr，将其包装为带 `ate.dev/*` 标签的统一 JSON 结构，同时生成合成生命周期事件 |
| `cmd/ateom-gvisor/main.go`、`cmd/ateom-microvm/main.go` | 运行时进程中将 actorlog 的输出写入与主日志相同的同步 writer，避免行交错 |

## 3. 架构与设计决策

### 3.1 全局日志器初始化
`serverboot.InitLogger()` 调用一次设置全局 `slog` 默认 logger：
```
slog.SetDefault(slog.New(contextlogging.NewHandler(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: &logLevel}))))
```
- 输出目标默认是 `os.Stdout`，也可通过 `InitLoggerWithWriter(w)` 指定（用于 ateom 将主日志与 actor 日志合并到同一同步 writer）。
- 日志级别由 `serverboot.logLevel`（`slog.LevelVar`）动态控制，通过 `SetLogLevel(level string)` 从命令行 flag 或配置设置，支持 `debug`/`info`/`warn`/`error`。

### 3.2 追踪与日志关联
`contextlogging.ContextHandler` 实现 `slog.Handler` 接口，在 `Handle` 中读取 `trace.SpanContextFromContext(ctx)`，若存在 TraceID 则追加属性 `ate.dev/trace-id`。这使得任何通过该 handler 输出的日志都能被外部链路追踪系统按 trace 聚合。

### 3.3 沙箱 Actor 日志管道
`actorlog.ActorLogger` 解决“容器内应用直接写 stdout/stderr”的日志采集问题：
- `StartJSONLogPipe` 创建 pipe，后台 goroutine 逐行读取容器输出。
- 每行尝试解析为 JSON；解析成功则复用原 JSON 对象，补入缺失的 `time` 字段，并在 `labels`（或 GCE 的 `logging.googleapis.com/labels`）下注入 `ate.dev/actor_atespace`、`ate.dev/actor_name`、`ate.dev/actor_uid`、`ate.dev/actor_template_namespace`、`ate.dev/actor_template_name`、`ate.dev/container_name`。
- 解析失败则将原始行作为 `message` 放入统一信封，同样附加上述标签。
- `EmitLifecycleLog` 主动产出合成事件（如容器启动/停止），供平台侧审计。

### 3.4 可观测性集成
`serverboot.InitTracing` 注册全局 TracerProvider，使用 OTLP gRPC exporter 推送到 collector；`InitMetrics` 注册 MeterProvider，同时包含 Prometheus reader（`/metrics`）和 OTLP periodic reader。资源属性统一携带 `service.name`、`service.instance.id`（进程级 UUID）以及 OTEL_* 环境变量覆盖值。

## 4. 约定与约束

- **统一入口**：长生命周期服务必须通过 `serverboot.InitLogger()` 或 `InitLoggerWithWriter` 初始化日志，禁止各自新建独立的全局 logger。
- **日志级别**：仅允许 `debug`、`info`、`warn`、`error` 四个级别，通过 `serverboot.SetLogLevel` 设置；空字符串表示不修改当前级别。
- **结构化字段命名**：所有跨组件共享的标识符使用 `ate.dev/*` 前缀（如 `ate.dev/trace-id`、`ate.dev/actor_name`、`ate.dev/container_name`），便于日志聚合与查询。
- **Trace 关联强制**：通过 `ContextHandler` 自动注入 trace ID，业务代码无需手动添加，但需确保日志调用传入正确的 `context.Context`（使用 `slog.InfoContext`/`ErrorContext` 等带 ctx 的 API）。
- **GCE 兼容**：`actorlog` 根据 `metadata.OnGCE()` 切换标签键名，保证在 GKE 上能被原生 Cloud Logging 识别。
- **并发安全**：`actorlog.SyncedWriter` 用互斥锁保护底层 writer，防止多 goroutine 写入时行交错；ateom 主进程与 actor 日志管道共享同一个 writer 实例。
- **错误处理**：OTel SDK 的错误通过 `otel.SetErrorHandler` 重定向到 `slog.Warn`，避免绕过 JSON 日志；`serverboot.Fatal` 在启动阶段失败时记录错误并 `os.Exit(1)`。
- **测试隔离**：测试中通过 `slog.SetDefault` 替换默认 logger 到内存 buffer，并在 cleanup 中恢复，避免污染全局状态。