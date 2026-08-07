---
kind: logging_system
name: 结构化日志系统（slog + OpenTelemetry）
slug: logging_system
category: logging_system
scope:
    - '**'
---

Agent Substrate 采用 Go 标准库 `log/slog` 作为统一的日志框架，结合 OpenTelemetry 实现结构化、可追踪的日志输出。整个系统的日志体系围绕以下核心组件构建：

**1. 全局日志初始化与配置**
- `internal/serverboot/serverboot.go` 提供 `InitLogger()` 和 `InitLoggerWithWriter()` 函数，所有长生命周期进程（ateapi、atelet、ateom-gvisor、ateom-microvm）在启动时调用这些函数设置全局 slog 默认记录器
- 日志级别通过 `--log-level` 命令行参数控制，支持 debug/info/warn/error 四个级别，使用 `slog.LevelVar` 实现动态调整
- 所有日志输出为 JSON 格式，写入 stdout，便于 Kubernetes 日志收集

**2. 上下文感知日志增强**
- `internal/contextlogging/contextlogging.go` 实现了 `slog.Handler` 包装器，自动从 gRPC/HTTP 请求上下文中提取 OpenTelemetry TraceID，并添加为 `ate.dev/trace-id` 字段
- 这使得每条日志都能关联到对应的分布式追踪链路，便于问题排查

**3. Actor 沙箱日志转发**
- `internal/actorlog/logger.go` 提供 `ActorLogger` 专门处理容器化工作负载的 stdout/stderr 日志
- 自动解析 JSON 格式的容器日志，补充 ate.dev/* 标签（actor_atespace、actor_name、actor_uid、actor_template_namespace、actor_template_name、container_name）
- 对非 JSON 日志行进行包装，统一输出格式
- 支持 GCE 环境下的 `logging.googleapis.com/labels` 字段兼容

**4. 日志级别管理策略**
- 通过 `serverboot.SetLogLevel()` 集中管理日志级别，避免各组件自行维护级别配置
- 支持运行时动态调整，无需重启服务
- 错误处理使用 `serverboot.Fatal()` 统一记录并退出进程

**5. 与 OpenTelemetry 集成**
- 日志系统与 tracing、metrics 共享相同的资源标识（ServiceName、ServiceInstanceID）
- OTel SDK 的错误也通过 slog 输出，确保调试信息的一致性
- 采样策略通过环境变量 `OTEL_TRACES_SAMPLER` 和 `OTEL_TRACES_SAMPLER_ARG` 控制

**6. 多组件一致性**
- 所有二进制文件遵循相同的日志初始化模式：`serverboot.InitLogger()` → `serverboot.SetLogLevel()` → 业务逻辑
- 测试代码中直接设置 `slog.SetDefault()` 覆盖全局记录器，便于断言日志输出
- 演示程序和工具程序也遵循同样的 slog 使用模式