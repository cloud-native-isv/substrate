# XUANJI 分支改动登记

> 约定（见 sandbox 仓 `docs/tasks/substrate-migration.md` §6）：本分支对 upstream 既有文件只做
> 「SandboxClass 枚举扩展」类最小改动；新能力一律以新增文件/目录进入。每次改动 upstream
> 既有文件必须在此登记，作为 rebase upstream 时的冲突核对清单。

基线：upstream `main` @ `aa9b7b82`。

## 已改动的 upstream 文件

| 文件 | 改动 | 原因 |
|---|---|---|
| `pkg/api/v1alpha1/sandboxconfig_types.go` | 新增 `SandboxClassWasm` 常量；Enum `gvisor;microvm` → `gvisor;microvm;wasm` | wasm 沙箱类 |
| `pkg/api/v1alpha1/actortemplate_types.go` | Enum 同上；3 条非 microvm 限制的 CEL message 由 “is 'gvisor'” 改为 “is not 'microvm'”（wasm 同受限，原文案误导） | wasm 沙箱类 |
| `pkg/api/v1alpha1/workerpool_types.go` | Enum 同上 | wasm 沙箱类 |
| `pkg/api/v1alpha1/sandboxconfig_validation_test.go` | 新增 `pythonWasmAsset` helper + 3 个 wasm 用例 | 测试覆盖 |
| `pkg/api/v1alpha1/actortemplate_validation_test.go` | 新增 wasm 合法类 + wasm Golden 拒绝用例；同步 5 处 errMsg 断言文案 | 测试覆盖 |
| `cmd/atecontroller/internal/controllers/workerpool_apply_test.go` | TestMicroVMPodShape / TestAteomSecurityContextByClass 各加 wasm 行（非特权、无 KVM 形状） | 测试覆盖 |
| `manifests/ate-install/sandboxconfig-validation.yaml` | VAP 新增 wasm 规则：每 arch 必须有 `python-wasm` 资产 | wasm 资产校验 |
| `manifests/ate-install/generated/*` | controller-gen 再生成（枚举/文案变化） | 生成物 |

## 新增文件（无 upstream 冲突面）

| 路径 | 内容 |
|---|---|
| `XUANJI.md` | 本登记文件 |
| `manifests/xuanji/wasm-example.yaml` | wasm 类示例：SandboxConfig(wasm-default, python-wasm 资产) + WorkerPool(sandboxClass=wasm, 外部 ateom-wasmd 镜像) + ActorTemplate(python-interpreter) |
| `contrib/e2b-e2e/` | E2B 协议验收资产（自 sandbox 仓 `scripts/` 原样迁入）：官方 SDK e2e、files e2e、smolagents、MCP server、TLS relay 等；作为 `cmd/e2bgw` 的验收基线，M3 改指新网关 |
| `cmd/e2bgw/`（规划，M3） | E2B REST/WS → ateapipb.Control 翻译网关（Go 重写） |

## 设计约定

- **ateom-wasmd 不在本仓**：wasm 运行时（实现 `internal/proto/ateompb` 的 `Ateom` 服务）由
  sandbox 仓（cloud-native-webassembly/sandbox）以 Rust 维护并发布镜像，WorkerPool 按
  `ateomImage` 引用 —— 与 ateom-gvisor/ateom-microvm 的 proto 缝完全一致。
- **wasm 类 v1 能力面**：`onResume` 仅 ColdBoot（Golden 为 microvm 专属）；durableDir ≤1
  （同 gvisor 限制）；worker 非特权（复用 gvisor 分支的 security context，能力集后续可再收窄）。
- **资产约定**：wasm 类 SandboxConfig 必须提供 `python-wasm` 资产（CPython 解释器 wasm 模块，
  arch 无关但按 arch 键重复登记）；atelet 原样拉取并交给 ateom-wasmd。
