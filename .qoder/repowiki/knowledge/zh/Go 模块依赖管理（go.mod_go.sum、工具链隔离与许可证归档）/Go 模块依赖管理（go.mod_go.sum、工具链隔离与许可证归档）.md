---
kind: dependency_management
name: Go 模块依赖管理（go.mod/go.sum、工具链隔离与许可证归档）
category: dependency_management
scope:
    - '**'
source_files:
    - go.mod
    - go.sum
    - Makefile
    - hack/update/go-modules.sh
    - hack/verify/go-modules.sh
    - hack/third_party/kubernetes/verify-generated.sh
    - hack/update/licenses.sh
    - hack/verify/licenses.sh
    - .ko.yaml
    - hack/tools/controller-gen/go.mod
    - hack/tools/golangci-lint/go.mod
    - hack/tools/kind/go.mod
    - hack/tools/ko/go.mod
    - hack/tools/protoc-gen-go/go.mod
    - hack/tools/setup-envtest/go.mod
    - hack/tools/code-generator/go.mod
---

## 1. 使用的系统与工具

- **包管理器**：标准 Go Modules（`module github.com/agent-substrate/substrate`，Go 版本 `1.26.3`），所有第三方依赖通过 `go.mod` / `go.sum` 声明。
- **构建镜像**：使用 `ko`（配置在 `.ko.yaml`，`KO_DOCKER_REPO := gcr.io/$(PROJECT_ID)/ate-images`）直接基于源码构建容器镜像，不产生传统二进制产物。
- **工具链隔离**：开发/生成/验证工具（controller-gen、protoc-gen-go、golangci-lint、kind、setup-envtest、go-licenses、code-generator 等）各自拥有独立的 `hack/tools/<tool>/go.mod` + `go.sum`，与主模块解耦，避免污染主依赖图。
- **许可证合规**：根目录 `LICENSES/` 按模块路径归档了每个第三方包的 LICENSE 文件，由 `hack/update/licenses.sh` 与 `hack/verify/licenses.sh` 驱动更新与校验。

## 2. 关键文件

| 文件 | 作用 |
|---|---|
| `go.mod` | 主模块依赖声明（direct + indirect），锁定全部第三方库版本 |
| `go.sum` | 对应依赖的哈希校验 |
| `Makefile` | 顶层入口：`build`、`build-images`、`test`、`lint`、`verify` 等 |
| `hack/update/go-modules.sh` | 更新流程：`go mod vendor` → `go mod tidy` |
| `hack/verify/go-modules.sh` | 验证流程：调用 `hack/third_party/kubernetes/verify-generated.sh go-modules` |
| `hack/third_party/kubernetes/verify-generated.sh` | 通用验证框架：在临时 worktree 中执行 update 脚本并 diff 检测 |
| `hack/update/licenses.sh` / `hack/verify/licenses.sh` | 许可证清单生成与校验 |
| `hack/tools/*/go.{mod,sum}` | 各工具的独立依赖快照 |
| `.ko.yaml` | ko 构建配置（镜像仓库、标签等） |
| `LICENSES/**` | 第三方许可证归档 |

## 3. 架构与约定

### 3.1 单一主模块 + 工具子模块
- 整个仓库只有一个业务模块 `github.com/agent-substrate/substrate`，所有 `cmd/*` 与 `internal/*` 共享同一份 `go.mod`。
- 仅用于 CI/本地开发的 CLI 工具被拆到 `hack/tools/<tool>`，每个工具是独立 Go module，通过 `hack/run-tool.sh` 调用，确保 `go install` 或 `go run` 时不会把 linter、codegen 等工具引入生产依赖。

### 3.2 依赖更新工作流
- 开发者运行 `./hack/update/go-modules.sh` 会先执行 `go mod vendor` 再 `go mod tidy`。该脚本同时触发 vendor 目录同步（尽管当前 `vendor/` 为空，但流程保留）。
- 验证侧 `./hack/verify/go-modules.sh` 借助 `verify-generated.sh` 在干净 worktree 中重新运行 update，并通过 `git status --porcelain` 比较差异；若存在未提交的变更则失败，强制将 `go.mod`/`go.sum` 变更纳入版本控制。

### 3.3 许可证治理
- `LICENSES/` 下按 Go module 路径组织第三方许可证副本，覆盖 GCP SDK、AWS SDK v2、Envoy、Prometheus、k8s、OTel、spiffe、grpc 等大量依赖。
- 通过 `hack/update/licenses.sh` 自动抓取并归档，`hack/verify/licenses.sh` 在 CI 中检查是否完整，形成“代码提交前必须同步许可证”的约束。

### 3.4 构建与发布
- 使用 `ko build` 将多个 `cmd/*` 直接打成容器镜像，版本号通过 `-ldflags` 注入 `internal/version.Version`。
- 镜像仓库默认推送到 `gcr.io/<PROJECT_ID>/ate-images`，可通过 `PROJECT_ID` 环境变量覆盖。

## 4. 约定与约束

- **依赖版本锁定**：所有 direct 与 indirect 依赖均显式写在 `go.mod` 中并由 `go.sum` 校验，禁止使用 `replace` 指向本地路径以外的私有替换（未发现 `replace` 指令）。
- **工具依赖隔离**：任何仅用于开发/CI 的工具不得添加到主模块 `require` 块中，应放入 `hack/tools/<tool>` 独立 module。
- **变更可重现**：`hack/verify/go-modules.sh` 要求工作区干净且 `go.mod`/`go.sum` 与源码一致，否则 CI 失败——这是强制约束。
- **许可证归档不可缺失**：新增依赖后必须运行 `./hack/update/licenses.sh`，否则 `./hack/verify/licenses.sh` 会在 CI 中标记缺失。
- **无私有代理/GOPRIVATE 配置**：仓库内未发现 `.envrc`、`.golangci.yml` 中的 `GOPROXY`/`GOPRIVATE` 设置，也未见 `go.work` 多模块编排，依赖解析依赖全局 Go 环境配置。
- **vendor 目录不参与版本控制**：虽然 `hack/update/go-modules.sh` 会执行 `go mod vendor`，但根级 `vendor/` 目录为空，实际构建走 `ko` 拉取远程模块缓存而非 vendor。

## 5. 主要依赖类别概览（来自 go.mod require）

- Kubernetes 生态：`k8s.io/api`、`client-go`、`controller-runtime`、`apiextensions-apiserver`、`metrics`、`kube-openapi`
- Google Cloud：`cloud.google.com/go/storage`、`container`、`iam`、`monitoring`、`resourcemanager`、`serviceusage`、`google.golang.org/api`、`grpc`
- AWS SDK v2：`aws-sdk-go-v2`、`config`、`service/s3`
- 可观测性：`go.opentelemetry.io/otel*`、`otelhttp`、`otelgrpc`、`prometheus/client_golang`、`exporters/otlp*`
- 网络/沙箱：`envoyproxy/go-control-plane`、`vishvananda/netlink`、`opencontainers/runtime-spec`、`containerd/ttrpc`
- 运行时通信：`redis/go-redis/v9`、`spf13/cobra`、`grpc`、`protobuf`、`spiffe/go-spiffe/v2`
- 测试/基准：`myzhan/boomer`、`alicebob/miniredis/v2`、`google/go-cmp`、`gotest.tools/v3`