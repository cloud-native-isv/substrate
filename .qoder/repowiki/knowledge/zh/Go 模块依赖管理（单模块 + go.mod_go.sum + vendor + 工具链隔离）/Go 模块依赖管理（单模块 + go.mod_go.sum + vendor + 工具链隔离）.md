---
kind: dependency_management
name: Go 模块依赖管理（单模块 + go.mod/go.sum + vendor + 工具链隔离）
slug: dependency_management
category: dependency_management
scope:
    - '**'
---

## 1. 使用的系统与方案
- 单一 Go 模块：根目录 `go.mod` 定义 `module github.com/agent-substrate/substrate`，所有子组件（cmd/*、internal/*、pkg/*、demos/*）共享同一个模块路径与依赖声明。
- 版本锁定：通过 `go.sum` 记录所有直接/间接依赖的精确校验和；构建产物使用 Ko 将 `go.mod`/`go.sum` 打入镜像，保证可重现。
- Vendor 策略：`hack/update/go-modules.sh` 会执行 `go mod vendor` 生成 `vendor/` 目录，`hack/verify/go-modules.sh` 通过 Kubernetes 上游的 `verify-generated.sh` 检查工作区干净且生成的 vendor 与代码一致。
- 工具链隔离：`hack/tools/<tool>/go.mod` + `go.sum` 为每个开发工具（controller-gen、protoc-gen-go、ko、golangci-lint、go-licenses、kind、setup-envtest 等）单独声明依赖，避免污染主模块依赖。
- 许可证收集：`LICENSES/` 目录由 `hack/update/licenses.sh` 使用 `go-licenses` 按多目标平台扫描并合并第三方许可证，同时复制 `third_party/` 下 fork 源码的 LICENSE/NOTICE/COPYING 到 `LICENSES/third_party/`。

## 2. 关键文件与包
- 模块清单与锁文件：`go.mod`、`go.sum`
- 更新/验证脚本：`hack/update/go-modules.sh`、`hack/verify/go-modules.sh`、`hack/third_party/kubernetes/verify-generated.sh`
- 许可证管理：`hack/update/licenses.sh`、`hack/verify/licenses.sh`、`LICENSES/`（含大量第三方 LICENSE）
- 工具链独立模块：`hack/tools/*/go.mod`（如 ko、controller-gen、protoc-gen-go、golangci-lint、go-licenses、kind、setup-envtest 等）
- 构建入口：`Makefile`（Ko 构建镜像、Go build 生成二进制）、`.ko.yaml`（Ko 配置）

## 3. 架构与约定
- 单模块聚合：所有 Go 代码位于同一模块下，通过 `internal/` 共享内部包，`pkg/` 暴露对外 API（CRD types、clientset/informers/listers），`cmd/` 下各子命令作为独立可执行程序。
- 依赖来源：主要依赖来自 k8s.io、google.golang.org/grpc、cloud.google.com/go、aws-sdk-go-v2、envoyproxy、prometheus、opentelemetry 等生态，均通过 `require` 显式声明在根 `go.mod`。
- 构建与镜像：`Makefile` 中 `KO_DOCKER_REPO` 指向 GCR，`build-images` 目标用 Ko 一次性构建 ateapi、atelet、podcertcontroller、atenet 四个镜像，确保依赖与源码一起固化。
- 生成代码约束：`hack/third_party/kubernetes/verify-generated.sh` 以 git worktree 方式运行 update 脚本后 diff 检查，任何未提交的生成差异都会导致 verify 失败。

## 4. 约定与约束
- 依赖更新流程：统一通过 `./hack/update/go-modules.sh` 调用 `go mod vendor` 和 `go mod tidy`，禁止手动编辑 `vendor/` 或 `go.sum`。
- 许可证合规：`./hack/verify/licenses.sh` 强制要求 `LICENSES/` 目录与当前依赖树一致，新增依赖必须能通过 `go-licenses save` 导出。
- 工具依赖隔离：`hack/tools/` 下每个工具拥有独立 `go.mod`/`go.sum`，通过 `hack/run-tool.sh` 调用，避免与业务模块依赖冲突。
- 版本固定：`go.mod` 中所有依赖均为具体版本号（含 `-0.2026...` 形式的 commit-hash 版本），不依赖 `latest` 或模糊版本。
- CI 校验：`hack/verify-all.sh` 串联 go-modules、licenses、gofmt、golangci-lint 等验证，PR 必须通过全部检查。
- 私有仓库/GOPRIVATE：当前仓库未发现 `.gitconfig`、`GOPRIVATE` 或私有代理配置，依赖均从公共 Go 模块代理获取。