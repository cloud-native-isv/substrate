---
kind: build_system
name: 基于 Ko + Kustomize 的 Go 多组件构建与 CI/CD 流水线
category: build_system
scope:
    - '**'
source_files:
    - Makefile
    - .ko.yaml
    - internal/version/version.go
    - hack/run-tool.sh
    - hack/verify-all.sh
    - hack/update-all.sh
    - hack/install-ate-kind.sh
    - hack/microvm-assets/assemble.sh
    - .github/workflows/pr-workflow.yaml
    - hack/verify/golangci-lint.sh
    - hack/update/go-generate.sh
    - hack/tools/ko/go.mod
    - hack/tools/golangci-lint/go.mod
---

## 1. 使用的系统与工具

- **语言与包管理**: Go (go.mod/go.sum)，使用 `go test -race` 运行单元测试。
- **容器镜像构建**: [ko](https://github.com/ko-build/ko)（通过 `hack/run-tool.sh ko` 调用），将 Go 模块直接编译为 OCI 镜像，无需 Dockerfile；默认基础镜像由 `.ko.yaml` 指定为 `gcr.io/distroless/static-debian13`，并针对特定模块覆盖为 `alpine` 或 `debian:stable-slim`（如 `cmd/ateom-microvm` 需要 glibc）。
- **Kubernetes 清单**: Kustomize（`manifests/` 下按组件拆分 base/component/kind overlay），通过 `kubectl apply -k manifests/ate-install/kind` 安装。
- **代码生成**: `go generate ./...` 驱动 protobuf、CRD deepcopy、clientset/informer/lister 等生成；第三方工具通过 `hack/tools/<tool>/go.mod` 版本锁定，统一经 `hack/run-tool.sh <name>` 解析路径后执行。
- **CI**: GitHub Actions (`.github/workflows/pr-workflow.yaml`)，在 PR 和 main push 上触发，包含单元测试、root-gated 测试、verify 全量检查以及 Kind 集群上的 e2e 矩阵（按 `ateapi-client-auth: [cert, token]` 双认证模式并行）。

## 2. 关键文件

| 文件 | 作用 |
|---|---|
| `Makefile` | 顶层入口：`build`、`test`、`lint`、`verify`、`e2e`、`build-images`、`build-atectl` 等目标 |
| `.ko.yaml` | ko 全局配置：默认基础镜像、`linux/amd64`+`linux/arm64` 多架构平台、按模块的基础镜像覆盖 |
| `internal/version/version.go` | 注入 `Version/Commit/BuildDate` 并通过 `runtime/debug.ReadBuildInfo()` 回退 |
| `hack/run-tool.sh` | 统一解析 `hack/tools/*` 下的工具二进制路径，避免开发者环境差异 |
| `hack/verify-all.sh` / `hack/update-all.sh` | 遍历 `hack/verify/*.sh` 与 `hack/update/*.sh` 顺序执行 |
| `hack/install-ate-kind.sh` | Kind 本地部署脚本：设置 `KO_DOCKER_REPO=localhost:5001`、单架构构建、启用 kind overlay |
| `hack/microvm-assets/assemble.sh` | 下载并缓存 kata-static、cloud-hypervisor、virtiofsd 等微虚拟机资产 |
| `.github/workflows/pr-workflow.yaml` | PR/主分支 CI：单元测试、root-gated 测试、verify、Kind e2e（含 micro-VM 与 gVisor） |
| `manifests/ate-install/` | Kustomize 清单：base + component + kind overlay，按组件拆分为 ateapi/atelet/atenet/podcertcontroller 等 |
| `go.mod` | 单一 Go module，所有 cmd/* 共享同一依赖图 |

## 3. 架构与约定

### 构建产物
- **容器镜像**：`make build-images` 调用 `ko build ./cmd/ateapi ./cmd/atelet ./cmd/podcertcontroller ./cmd/atenet`，输出到 `KO_DOCKER_REPO`（默认 `gcr.io/${PROJECT_ID}/ate-images`）。每个 `cmd/*` 对应一个镜像，镜像标签来自 git tag/commit。
- **本地二进制**：`make build-atectl` 产出 `bin/kubectl-ate`；`make build-atenet` 产出 `bin/atenet`。这些是纯 Go 静态链接产物，用于 CLI 或 sidecar 场景。
- **演示镜像**：`make build-demos` 构建 `./demos/counter` 等示例镜像。

### 版本注入
`VERSION` 默认取自 `git describe --tags --always --dirty`，通过 `-X=github.com/agent-substrate/substrate/internal/version.Version` 注入；`Commit` 与 `BuildDate` 若未显式注入则从 `debug.ReadBuildInfo()` 的 `vcs.revision`/`vcs.time`/`vcs.modified` 回退，保证 `go build` 也能输出有意义的版本串。

### 工具链版本锁定
所有第三方工具（`golangci-lint`、`controller-gen`、`protoc-gen-go`、`kind`、`setup-envtest`、`ko`、`go-licenses`、`code-generator` 等）各自拥有独立的 `hack/tools/<tool>/go.mod`，通过 `go tool -n <tool>` 解析出已安装的二进制路径，再由 `hack/run-tool.sh` 包装执行。这避免了全局 `go install` 的版本漂移。

### 验证与格式化
- `make fmt` → `hack/update/gofmt.sh`
- `make verify-fmt` → `hack/verify/gofmt.sh`
- `make lint` → `hack/verify/golangci-lint.sh`（强制 `GOOS=linux` 以正确类型检查 netlink 等平台相关包）
- `make verify` → `hack/verify-all.sh`，依次执行 `boilerplate`、`go-generate`、`go-modules`、`gofmt`、`golangci-lint`、`licenses`、`proto-fmt`、`python-licenses`、`shellcheck`。

### 测试
- 单元测试：`make test` = `go test -race ./...`。
- root-gated 测试：CI 中单独通过 `hack/run-root-tests.sh` 以 sudo 运行，因为涉及 overlay mount、mknod、trusted.* xattr 等需 root 权限的操作。
- E2E：`make e2e` 先构建镜像与 demo，再调用 `hack/run-e2e.sh`；CI 中使用 Kind 集群，分别以 gVisor 与 micro-VM 两种运行时执行 demo/e2e 套件。

### 部署
- 本地开发：`hack/install-ate-kind.sh` 将镜像推送到 `localhost:5001`，用 Kustomize kind overlay 安装到 Kind。
- 远程 GKE：`hack/install-ate.sh`（被 kind 脚本复用）配合 `manifests/ate-install/base` 与 `components/*` 进行部署。
- 微虚拟机 Demo：`hack/run-microvm-demo-kind.sh` 利用 `hack/microvm-assets/assemble.sh` 预下载的资产，上传至集群内 rustfs 并应用 demo YAML。

## 4. 约定与约束

- **所有可执行入口位于 `cmd/*`**，每个目录是一个独立 Go 模块入口，由 ko 直接构建镜像；CLI 工具（如 `kubectl-ate`）也可单独 `go build` 产出二进制。
- **镜像基础镜像策略**：默认 distroless static，仅对确实需要 glibc/系统调用的模块（`cmd/ateom-microvm`）覆盖为 `debian:stable-slim`，其他 demo 可用 `alpine`。
- **多架构支持**：`.ko.yaml` 声明 `linux/amd64` 与 `linux/arm64`，CI 在 ubuntu-latest（amd64）runner 上构建，同时缓存 micro-VM 资产以加速重复构建。
- **版本必须可追溯**：`VERSION` 应来自 git tag；CI 中通过 `git describe` 获取，本地开发时 fallback 为 `dev`。
- **工具链不可随意升级**：所有第三方工具必须通过 `hack/tools/<tool>/go.mod` 锁定版本，新增工具需遵循相同模式，并由 `hack/update-all.sh` 统一管理。
- **CI 不写仓库**：PR workflow 明确注释“Nothing here writes to the repository”，所有 job 仅 checkout、构建、在临时 Kind 集群上运行测试。
- **根权限隔离**：普通 runner 跑 `go test -race ./...`，root-gated 测试通过 `hack/run-root-tests.sh` 单独以 sudo 执行，避免污染常规测试环境。
- **Kustomize 分层**：`manifests/ate-install/base` 定义通用资源，`components/*` 按功能开关，`kind/*` overlay 注入本地化配置（如 OTEL/Prometheus/rustfs），便于不同环境复用。
- **Protobuf/CRD 生成纳入 verify**：`hack/verify/proto-fmt.sh` 与 `hack/verify/go-generate.sh` 确保提交前生成的代码与源码同步，否则 CI 失败。
