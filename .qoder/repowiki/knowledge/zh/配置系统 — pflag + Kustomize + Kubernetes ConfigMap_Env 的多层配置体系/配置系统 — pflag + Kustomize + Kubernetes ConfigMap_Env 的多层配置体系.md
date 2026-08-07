---
kind: configuration_system
name: 配置系统 — pflag + Kustomize + Kubernetes ConfigMap/Env 的多层配置体系
slug: configuration_system
category: configuration_system
scope:
    - '**'
---

## 1. 使用的系统与框架
- **命令行参数**: 所有 Go 二进制均通过 `github.com/spf13/pflag` 声明 CLI 标志（`--grpc-listen-addr`、`--log-level`、`--otel-*` 等），在 `main()` 中调用 `pflag.Parse()` 解析。
- **环境变量覆盖**: 部分关键标志支持以 `@env` 哨兵值从环境变量注入，由 `loadFlagsFromEnv()` 统一替换（如 `ATE_API_REDIS_ADDRESS`、`ATE_API_K8SJWT_ISSUER`）；另有大量 OTel 相关标志直接以 `os.Getenv("OTEL_*")` 作为默认值。
- **Kubernetes 配置载体**: 运行时配置通过 Pod 的 `env`、`volumeMount`（挂载 PEM/证书文件路径）、以及 `ConfigMap`（如 `atenet-router-agentgateway-config` 中的 `config.yaml`）提供。
- **部署与变体管理**: 使用 **Kustomize** 组织 `manifests/ate-install/` 下的 base/kind/components 等多层清单，通过 `kustomization.yaml` 组合不同组件与补丁。
- **构建期版本注入**: `Makefile` 通过 Ko 将 `internal/version.Version` 以 `-ldflags` 注入镜像，`--version` 输出该版本。

## 2. 核心文件与位置
- 各进程入口与 pflag 定义:
  - `cmd/ateapi/main.go` — gRPC 控制面服务，最多标志位，含 Redis/JWT/TLS/审计等配置
  - `cmd/atecontroller/main.go` — controller-runtime 控制器，连接 ateapi 及 OTel 导出配置
  - `cmd/atelet/main.go` — 节点侧代理，镜像缓存/GCS/S3/OCI bundle 等配置
  - `cmd/podcertcontroller/main.go` — 证书签发控制器，kubeconfig/in-cluster/CA pool 文件路径
  - `cmd/atenet/internal/root.go` — Cobra 根命令，子命令 router/dns 各自带 pflag
- 环境变量覆盖逻辑: `cmd/ateapi/main.go` 的 `loadFlagsFromEnv()`
- Kustomize 清单:
  - `manifests/ate-install/base/kustomization.yaml` — 基础资源集合
  - `manifests/ate-install/components/agentgateway/configmap.yaml` — AgentGateway 路由/网关/ExtProc 配置
  - `manifests/ate-install/kind/` — Kind 开发环境覆盖
- 构建脚本: `Makefile`（Ko 构建、ldflags 版本注入）

## 3. 架构与约定
- **单一入口 + pflag**: 每个二进制独立解析自身标志，无集中式 config 包；标志名即文档，`--log-level`、`--version` 为通用约定。
- **分层加载顺序**: ① pflag 默认值 → ② 命令行覆盖 → ③ `@env` 哨兵从环境变量替换 → ④ 启动时读取文件路径（PEM、CA pool、credential-bundle）→ ⑤ 运行时依赖（Redis、GCS/S3、Kubernetes API）按配置建立连接。
- **安全敏感项走文件**: TLS 证书、CA 池、JWT pool、ServiceAccount token 均以文件路径形式传入，由 `credbundle`、`localca` 等内部包在运行时加载，避免硬编码或明文传参。
- **可观测性配置标准化**: OTel 相关标志统一遵循 `OTEL_EXPORTER_OTLP_ENDPOINT`、`OTEL_METRIC_EXPORT_INTERVAL`、`OTEL_TRACES_SAMPLER` 等标准环境变量命名，便于跨组件复用。
- **Kustomize 多环境编排**: base 定义核心组件，kind 覆盖开发环境（Prometheus/Otel Collector/RustFS），components 提供可选功能（agentgateway 等），通过 `kustomization.yaml` 组合。
- **ConfigMap 驱动外部化**: atenet 的 AgentGateway 配置以 `config.yaml` 放入 ConfigMap，通过环境变量 `$AGENTGATEWAY_OTLP_ADDRESS` 动态注入，实现同一份清单适配不同 OTEL 后端。

## 4. 约定与约束
- **标志命名**: 全部使用 kebab-case（如 `grpc-listen-addr`、`redis-cluster-address`），与 Kubernetes 风格一致。
- **日志级别**: 统一通过 `--log-level` 控制，取值 debug/info/warn/error，由 `serverboot.SetLogLevel` 验证。
- **版本输出**: 所有二进制支持 `--version`，打印由 ldflags 注入的 `internal/version.Version`。
- **环境变量覆盖范围**: 仅 `loadFlagsFromEnv` 显式列出的标志支持 `@env` 哨兵，未列入者不会自动从 env 注入，需显式设置默认值或通过命令行覆盖。
- **TLS/认证文件路径**: 所有证书、CA、token 文件路径必须指向 Pod 卷挂载的真实路径（如 `/run/podidentity.podcert.ate.dev/...`），空值表示禁用对应能力（如 `podIdentityCACerts` 为空则客户端证书校验关闭）。
- **存储后端选择**: `ATE_STORAGE_BACKEND=s3` 切换 S3，否则默认 GCS；AWS 端点通过标准 `AWS_*` 环境变量配置。
- **Kustomize 不可变性**: base 清单只增不改，变体通过 patches 和 overlays 叠加，禁止直接修改 base 文件。
- **构建产物**: 所有镜像通过 Ko 构建，`KO_DOCKER_REPO` 由 Makefile 导出，版本通过 `VERSION_PKG=github.com/agent-substrate/substrate/internal/version` 注入。