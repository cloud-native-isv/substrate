# XUANJI 分支改动登记

> 约定（宪法 v1.2.0.1 原则 I，按阶段生效）：**阶段 1** 对 upstream 既有文件*倾向*最小改动、
> 新能力优先以新增文件/目录进入 —— 这是保护尚在形成的理解，不是硬性约束。**阶段 2** 明确允许
> 重写 upstream 实现、改动其原本架构。
>
> 作为交换，**登记义务是强制的且不分阶段**：每一处对 upstream 既有文件的改动都必须在此登记
> （改了什么、为什么）。放弃 diff 最小化之后，这份登记表——而非小 diff——才是 rebase 冲突核对
> 与将来分叉决策的唯一依据。rebase 时，本表列为「有意重写」的文件以我方版本为准并更新登记。

基线：upstream `main` @ `aa9b7b82`。

## Remote 与分支定义（治理基线，2026-09-17 确立）

### Remote 定义（四个，各司一职）

| Remote | URL | 角色 | 读写约定 |
|---|---|---|---|
| `origin` | `https://github.com/agent-substrate/substrate` | **原库（upstream）** | 只 fetch，永不 push |
| `github` | `git@github.com:cloud-native-isv/substrate.git` | 针对原库的 **fork**（对外公开面 / upstream PR 出口） | push 仅限 PR 分支与 main 同步 |
| `gitlab` | `git@gitlab.alibaba-inc.com:cloud-native-isv/substrate.git`（code.alibaba-inc.com 镜像） | **日常开发主力代码库** | xuanji 分支的权威（authoritative）远端 |
| `gitee` | `git@gitee.com:cloud-native-isv/substrate.git` | **代码备份** | 定期从 gitlab 推送备份，不作开发 |

同步流向：`origin/main` →（fetch+ff push）→ `gitlab/main` 镜像、`github/main`；`gitlab/xuanji` →（备份 push）→ `github/xuanji`、`gitee/xuanji`。一切写远端操作须经确认（GATED）。

### 分支定义（长期分支只有两条）

| 分支 | 定义 | 跟踪 |
|---|---|---|
| `main` | **跟踪 upstream 的代码修改**：只镜像 `origin/main`，不承载任何定制提交（Constitution Principle I） | 本地 `main` → `origin/main` |
| `xuanji` | **本地自定义的修改**：全部 xuanji 定制的唯一长期分支，rebase 到 main 保持最新 | 本地 `xuanji` → `gitlab/xuanji`（主力权威） |

### 残留分支（逐步归一清理，不再新增）

除上述两条外的一切分支均为开发过程残留，归一方向 = 内容并入 `xuanji`（或确认被覆盖）后删除：

| 残留分支 | 位置 | 现状（2026-09-17 实测） | 清理路径 |
|---|---|---|---|
| `xuanji-wasm-preflight` @9f375565 | gitlab | 持有全部 wasm/e2bgw/cert-manager 特性提交（13 个），xuanji 未含 | **优先归一**：特性并入 xuanji（rebase/cherry-pick，冲突核对 xuanji.md 台账）后删除 |
| `xuanji-wasm` @1e338827 | gitlab | preflight 的平行旧线（同主题不同 sha，含 cherry-pick 冲突修复痕迹），15 个提交不被任何分支包含 | 先做**内容级**比对（tree/patch-id，勿只看祖先关系）确认已被 preflight 覆盖或拣回差异，再删除 |
| `security-md-fixup`、`feat/long-running-actor-support`、`feature/nanoclaw-multiplex-demo`、`feature/openclaw-integration` | github fork | 各有独有提交（1/1/31/3），非本项目会话产物，疑为 fork 上其他实验 | **不盲删**：逐个确认归属与价值后裁决（保留/并回/删除） |

配套事实（2026-09-17 实测）：`gitlab/main` @aa9b7b82 落后 `origin/main` @85ce8ed5 **469 提交**（纯滞后、零领先，可安全 ff）；`github/xuanji` 与 `gitee/xuanji` 均 @d067d4d0，落后 `gitlab/xuanji` @2771ba7c 1 提交（备份滞后）。`origin` 另有 `release-0.1` 分支（上游发布线，只读关注）。


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
| `xuanji.md` | 本登记文件（原名 `XUANJI.md`，2026-08-07 按命名规范改小写） |
| `ARCHITECTURE.md`、`CHANGELOG.md`、`docs/{decisions,notes,concepts,tutorials,tasks,reference,contribute}/` | xuanji 文档空间骨架（/speckit.docs reconcile 建立；薄根入口 + ADR/notes 生命周期 + 六型索引） |
| `docs/overview.md`、`docs/concepts/core-concepts.md`、`docs/reference/actor-lifecycle-flows.md`、`docs/figures/`（8 图 ×puml/png/svg） | upstream 架构深度研究（study-project）：组件架构图/部署图/Resume/Suspend/停车时序图/状态机/资源模型/golden 流程 + 概念与流程文档；`ARCHITECTURE.md` 详细文档表同步更新 |
| `manifests/xuanji/wasm-example.yaml` | wasm 类示例：SandboxConfig(wasm-default, python-wasm 资产) + WorkerPool(sandboxClass=wasm, 外部 ateom-wasmd 镜像) + ActorTemplate(python-interpreter) |
| `contrib/e2b-e2e/` | E2B 协议验收资产（自 sandbox 仓 `scripts/` 原样迁入）：官方 SDK e2e、files e2e、smolagents、MCP server、TLS relay 等；作为 `cmd/e2bgw` 的验收基线，M3 改指新网关 |
| `cmd/e2bgw/` | E2B REST → `ateapipb.Control` 翻译网关（Go）：HS256 API-Key 鉴权（atespace/tenant claim）、POST /sandboxes→CreateActor+ResumeActor(boot)（失败回滚 DeleteActor）、pause/resume→Suspend/Resume、gRPC↔HTTP 错误映射、数据面 /execute /files 307 跳转 atenet 域名（M4 接通）；handler 单测全绿 |
| `docs/reference/e2b-api-surface.md`、`docs/concepts/sandbox-landscape.md` | /speckit.docs 扇出蒸馏产出：E2B 接口面×e2bgw 覆盖矩阵（代码核实）+ Agent 沙箱竞品格局（仅公开来源）。原 `docs/reference/collected-materials/`（含内网原文）已移出 tracked docs 树（本仓有公开 remote，内网原文不入库） |
| `docs/reference/upstream-evolution-research.md`、`docs/reference/xuanji-evolution-survey.md`、`docs/figures/{xuanji-evolution-stages,upstream-new-since-baseline,xuanji-extension-plane}.{puml,png,svg}` | 演进研究（study-project，2026-08-19）：upstream 演进方向×xuanji 演进支持（38 提交漂移主题/新子系统/in-flight 分支/rebase 冲突风险/吸收清单）+ xuanji 分支演进调研（试点解剖、preflight M4 管线、阶段转换决策点），配套三图 |
| `docs/hugo.toml`、`docs/layouts/`、`docs/static/css/site.css`、`docs/.gitignore`、`docs/contribute/docs-site-build.md` | Hugo 呈现层（create-docs 技能 scaffold + 人工修正）：`docs/` 兼作 Hugo 项目根，内容挂载不复制、Markdown 保持无 frontmatter。人工补 4 条挂载（根级 `*.md` 入 content；`figures` 镜像到 concepts/reference/overview 三处 static）与 `layouts/partials/page-title.html`（H1 兜底标题），修复根级文档缺页/死链、导航标题空白、raw-HTML 图片 404。构建 29 页零 warning；产物 `docs/public/` 永不提交 |

## 设计约定

- **ateom-wasmd 不在本仓**：wasm 运行时（实现 `internal/proto/ateompb` 的 `Ateom` 服务）由
  sandbox 仓（cloud-native-webassembly/sandbox）以 Rust 维护并发布镜像，WorkerPool 按
  `ateomImage` 引用 —— 与 ateom-gvisor/ateom-microvm 的 proto 缝完全一致。
- **wasm 类 v1 能力面**：`onResume` 仅 ColdBoot（Golden 为 microvm 专属）；durableDir ≤1
  （同 gvisor 限制）；worker 非特权（复用 gvisor 分支的 security context，能力集后续可再收窄）。
- **资产约定**：wasm 类 SandboxConfig 必须提供 `python-wasm` 资产（CPython 解释器 wasm 模块，
  arch 无关但按 arch 键重复登记）；atelet 原样拉取并交给 ateom-wasmd。
