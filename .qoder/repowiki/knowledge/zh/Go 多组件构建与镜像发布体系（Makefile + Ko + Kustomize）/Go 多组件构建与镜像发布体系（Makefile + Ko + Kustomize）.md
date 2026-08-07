---
kind: build_system
name: Go 多组件构建与镜像发布体系（Makefile + Ko + Kustomize）
slug: build_system
category: build_system
scope:
    - '**'
---

## 构建系统与工具链

本项目采用 **单一 go.mod 聚合** 的 Go 工程，通过 Makefile 统一编排编译、测试、验证与镜像构建流程，核心依赖如下：
- **Ko**：基于 `go build` 直接构建 OCI 镜像，无需 Dockerfile；默认基础镜像为 `gcr.io/distroless/static-debian13`，支持 linux/amd64 与 linux/arm64 双架构。
- **Kustomize**：用于组织 Kubernetes 部署清单（`manifests/ate-install/`），提供 base、kind 开发环境、token-client 覆盖等可组合变体。
- **golangci-lint**：代码质量检查，通过 `hack/run-tool.sh` 在 `GOOS=linux` 环境下运行以屏蔽平台相关包差异。
- **controller-gen / client-gen / informer-gen**：从 CRD 类型定义生成 deepcopy、clientset、listers、informers 及 gRPC 代码。
- **protoc-gen-go / protoc-gen-go-grpc**：从 `.proto` 文件生成 Go 客户端与服务端代码。

## 版本注入与二进制标识

所有二进制通过 `-ldflags -X github.com/agent-substrate/substrate/internal/version.Version=<VERSION>` 注入版本号。`internal/version` 包在运行时回退到 `runtime/debug.ReadBuildInfo()` 获取 commit、build date 与 dirty 标记，保证 `go build` / `go install` 也能输出有意义的版本信息。

## 构建目标与产物

| 目标 | 说明 |
|------|------|
| `make build` | 调用 `ko build` 构建 ateapi、atelet、podcertcontroller、atenet 四个控制面/节点侧镜像，并编译 `kubectl-ate` CLI |
| `make build-images` | 仅构建容器镜像（KO_DOCKER_REPO 指向 `gcr.io/<PROJECT_ID>/ate-images`） |
| `make build-atectl` | 单独编译 kubectl-ate 插件二进制 |
| `make test` | 执行 `go test ./...` |
| `make e2e` | 先构建镜像与 demo，再运行 `hack/run-e2e.sh` |
| `make verify` | 运行 `hack/verify-all.sh`（遍历 `hack/verify/*.sh`） |
| `make fmt` / `make lint` | 格式化与 golangci-lint 检查 |

## CI/CD 流水线（GitHub Actions）

`.github/workflows/pr-workflow.yaml` 定义了 PR 与 main 分支触发的工作流：
- **run-tests**：在 ubuntu-latest 上安装 Go（版本取自 go.mod）、运行单元测试与 root-gated 测试（`hack/run-root-tests.sh`），最后执行 `hack/verify-all.sh`。
- **e2e-test-matrix**：矩阵覆盖 `cert` 与 `token` 两种 ATE API 客户端认证方式；启用 KVM、缓存 micro-vm 资产、创建 Kind 集群、安装 Agent Substrate、部署 gVisor 与 micro-vm 两种 demo，分别运行 E2E 套件。
- 每周定时任务刷新 micro-vm 资产缓存，避免 GitHub Actions 7 天空闲清理导致重复下载。

## 工具管理策略

所有外部工具（ko、golangci-lint、controller-gen、protoc-gen-*、kind、setup-envtest 等）通过 `hack/tools/<tool>/go.mod` 独立声明版本，由 `hack/run-tool.sh` 统一解析并执行，确保开发者本地与 CI 环境一致。

## 模块与依赖更新

`hack/update/go-modules.sh` 执行 `go mod vendor && go mod tidy` 同步依赖；`hack/update-all.sh` 顺序执行 `hack/update/*.sh` 完成全量更新。

## 约束与约定

- 所有 Go 二进制必须通过 `internal/version` 暴露版本字符串，且构建时必须传入 `-ldflags`。
- 镜像构建统一使用 Ko，禁止自行编写 Dockerfile（demo/sandbox 与 ateom-microvm 除外，已在 `.ko.yaml` 中通过 `baseImageOverrides` 指定 alpine 与 debian:stable-slim）。
- 跨平台交叉编译通过 Ko 的 `defaultPlatforms` 配置自动处理，无需手动设置 GOOS/GOARCH。
- 验证脚本必须在 `LC_ALL=C` 下运行以保证排序一致性。
- golangci-lint 必须以 `GOOS=linux` 运行，避免 macOS 上 netlink 等平台相关包类型检查失败。
