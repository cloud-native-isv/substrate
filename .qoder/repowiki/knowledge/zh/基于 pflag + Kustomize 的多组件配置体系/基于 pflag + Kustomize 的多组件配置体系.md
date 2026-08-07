---
kind: configuration_system
name: 基于 pflag + Kustomize 的多组件配置体系
category: configuration_system
scope:
    - '**'
source_files:
    - cmd/ateapi/main.go
    - cmd/atecontroller/main.go
    - cmd/atelet/main.go
    - cmd/atenet/internal/root.go
    - internal/serverboot/serverboot.go
    - internal/ateapiauth/client.go
    - internal/ateapiauth/server.go
    - internal/credbundle/credbundle.go
    - manifests/ate-install/kind/kustomization.yaml
    - manifests/ate-install/components/agentgateway/configmap.yaml
---

## 1. 使用的系统/方法

仓库采用 **Go 原生命令行标志（`spf13/pflag`）** 作为所有长驻进程（`ateapi`、`atelet`、`atecontroller`、`atenet`、`ateom-*`）的统一配置入口，配合 **Kubernetes Kustomize** 清单在部署层注入参数。没有引入 `viper`、`cobra`（仅 `atenet` 用 cobra 组织子命令）、YAML/TOML 等外部配置文件格式；运行时配置通过三种来源组合：
- 二进制启动参数（`--grpc-listen-addr`、`--redis-cluster-address`、`--log-level` 等）
- 环境变量（`ATE_API_REDIS_ADDRESS`、`OTEL_*`、`ATE_STORAGE_BACKEND`、`AWS_S3_USE_PATH_STYLE` 等）
- Kubernetes 集群内资源（`rest.InClusterConfig()` 获取 kubeconfig；ServiceAccount Token 文件用于 OIDC/JWKS 访问）

可观测性（日志、追踪、指标）统一由共享包 `internal/serverboot` 初始化，遵循 OpenTelemetry SDK 约定并通过 `OTEL_*` 环境变量覆盖。

## 2. 关键文件与包

| 路径 | 职责 |
|---|---|
| `cmd/ateapi/main.go` | 控制面主入口；声明全部 `pflag` 标志、`loadFlagsFromEnv` 哨兵替换、Redis/TLS/JWT 配置组装、gRPC 服务器启动 |
| `cmd/atecontroller/main.go` | Controller 入口；通过 `pflag` 暴露 ateapi 连接、OTel 导出相关参数，调用 `ctrl.GetConfigOrDie()` 取 in-cluster config |
| `cmd/atelet/main.go` | 节点侧代理；标志包括 gRPC 端口、证书 bundle、镜像缓存目录、存储后端选择（GCS/S3 via `ATE_STORAGE_BACKEND`） |
| `cmd/atenet/internal/root.go` | 网络组件入口，使用 cobra 组织 `router` / `dns` 子命令 |
| `internal/serverboot/serverboot.go` | 共享启动引导：`InitLogger`、`SetLogLevel`、`InitTracing`、`InitMetrics`、`StartMetricsServer`（`/metrics`、`/readyz`、`/healthz`） |
| `internal/ateapiauth/*.go` | 客户端/服务端认证配置（默认 SA CA 文件路径、Token 文件路径常量） |
| `manifests/ate-install/` | Kustomize 基座与多环境 overlay（`kind/`），以 ConfigMap/Patch 形式向 Pod 注入 flag 与环境变量 |

## 3. 架构与设计约定

### 3.1 标志 → 环境变量哨兵覆盖
`ateapi` 实现了一个轻量级哨兵机制：若某个 `pflag.String` 的值为字面量 `@env`，则在 `pflag.Parse()` 之后由 `loadFlagsFromEnv()` 将其替换为对应 `os.Getenv(...)` 的值（如 `ATE_API_REDIS_ADDRESS`）。这允许同一份 Kustomize 清单在不同分支/环境中复用，只需在 ConfigMap 中提供不同值。

### 3.2 环境变量直读模式
其他组件直接通过 `os.Getenv` 读取环境变量作为配置源，典型模式是 `pflag.String("otel-exporter-otlp-endpoint", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"), ...)` —— 将环境变量设为 flag 的默认值，从而支持 CLI 覆盖。

### 3.3 运行时动态配置
- **日志级别**：`serverboot.SetLogLevel` 通过 `slog.LevelVar` 实现运行期可调，无需重启。
- **就绪探针**：`serverboot.Readiness` 在 SIGTERM 后标记 not-ready，使 `/readyz` 返回 503，而 `/healthz` 保持 200，配合优雅停机（`drain-delay` + `drain-timeout`）。
- **存储后端切换**：`atelet` 根据 `ATE_STORAGE_BACKEND` 在 GCS（默认）与 S3 之间切换，S3 走 AWS SDK 标准环境变量（`AWS_S3_USE_PATH_STYLE`）。

### 3.4 集群内凭据
所有组件通过 `k8s.io/client-go/rest.InClusterConfig()` 自动发现 kubeconfig，并挂载 ServiceAccount Token 文件（`DefaultServiceAccountTokenFile`）用于 OIDC discovery 与 JWKS 拉取；TLS 证书通过 PEM bundle 文件（`grpc-server-cred-bundle`、`pod-identity-ca-certs`、`client-jwt-ca-cert` 等）加载，由 `internal/credbundle` 解析。

### 3.5 可观测性统一初始化
`internal/serverboot` 为所有长驻进程提供一致的 OTel 初始化：TracerProvider + MeterProvider 通过 OTLP gRPC 推送，Prometheus reader 暴露 `/metrics`，资源属性包含 `service.name`、`service.instance.id` 以及 `resource.WithFromEnv()` 支持的 `OTEL_*` 覆盖。

## 4. 约定与约束

- **所有服务必须通过 `pflag` 暴露配置项**，禁止硬编码字符串常量作为地址/端口/路径（除少数内部常量如 `ateompath.ImageCacheDir`）。
- **敏感信息（证书、密钥、CA bundle）一律通过文件路径传入**，而非明文 flag 或环境变量；由 `credbundle` 或 `x509.CertPool.AppendCertsFromPEM` 在启动时一次性读取。
- **日志级别统一通过 `--log-level` 设置**，取值限定为 `debug|info|warn|error`，由 `serverboot.SetLogLevel` 校验。
- **健康检查端点固定为 `/metrics`、`/readyz`、`/healthz`**，由 `serverboot.StartMetricsServer` 统一注册；`/readyz` 在优雅停机期间返回 503。
- **OpenTelemetry 配置完全遵循 OTEL_* 环境变量约定**（`OTEL_EXPORTER_OTLP_ENDPOINT`、`OTEL_METRIC_EXPORT_INTERVAL`、`OTEL_TRACES_SAMPLER`、`OTEL_TRACES_SAMPLER_ARG` 等），组件不定义自有前缀。
- **Kubernetes 部署通过 Kustomize 管理**：`manifests/ate-install/kind/` 等 overlay 通过 `patches.yaml`、`configmap.yaml` 注入 flag 与环境变量，避免修改基础模板。
- **`atenet` 是唯一使用 cobra 的组件**，用于组织 `router` 和 `dns` 两个子命令；其余组件均为单入口 main。
- **不支持热重载配置**：代码中存在 TODO 注释指出 CA 池应周期性重载（见 `buildServerCreds` 中的注释），但当前未实现；证书变更需滚动重启 Pod。
- **`ateapi` 的 `@env` 哨兵仅覆盖显式列出的 flag**，新增 flag 必须手动加入 `loadFlagsFromEnv` 的映射表，否则不会从环境变量生效。