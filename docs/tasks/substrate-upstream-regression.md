# substrate 回归开源评估：fork 改动盘点与向 sandbox 仓迁移方案

> 状态：评估报告（待用户裁决迁移顺序）
> 更新（2026-09-17）：§4 前置 1「分支收敛」已完成——xuanji-wasm-preflight 的 13 个特性提交已 cherry-pick 归一进 xuanji（代码树与 preflight 逐字节一致），设计产物随 xuanji 提交；残留分支删除与 §5 其余步骤待执行。
> 日期：2026-09-16
> 基线：upstream `main` @ aa9b7b82 · fork 超集分支 `xuanji-wasm-preflight` @ 9f375565（`xuanji` @ 2771ba7c 仅多 3 个文档提交、少全部特性提交）
> 目标（用户指令）：substrate 尽可能回归开源，fork 上的修改尽可能移入 sandbox 仓（cloud-native-webassembly/sandbox）。
> 配套：[ADR 0002 两层沙箱模型](../decisions/0002-two-tier-supervisor-worker-sandbox.md)、sandbox 仓 `docs/tasks/substrate-migration.md`（正向迁移）——本文是其**反向收口**：把落在 fork 里的定制搬出去。

## 1. 改动全量盘点（preflight vs upstream main）

代码面 49 文件（+5198/−82）；文档/.specify 面 ~1080 文件（+237K 行，主体是 collected-materials 参考资料）。
其中**触碰 upstream 既有文件的仅 14 个**（`xuanji.md` 台账全部登记在案），其余均为纯新增目录/文件。

## 2. 分类与处置

### A 类 — 可直接搬去 sandbox（纯新增，零 upstream 冲突面）

| # | 改动 | 体量 | 去向 | 备注 |
|---|---|---|---|---|
| A1 | `cmd/e2bgw/`（E2B REST 网关，Go） | ~1.5K 行 + ~680 行测试 | sandbox 仓 `cmd/e2bgw/`（独立 go.mod） | Tier-2「e2b+自研管控」的本体，本就该在 sandbox（ADR 0001 核心价值①）。**唯一迁移阻碍**：import 了 `internal/ateapiauth`（internal 包，跨 module 不可引用）——4 文件、非测试 ~320 行；两个解法：(a) 随迁复制（保留 Apache-2.0 头与出处注释），(b) 推动 upstream 提升为 `pkg/ateapiauth`（更干净，可提 PR）。其余依赖已验证全部来自 upstream 公开包 `pkg/proto/ateapipb`（含 `ListActorSnapshots*`，main @ aa9b7b82 实测存在）→ **迁移后 sandbox 经公开 Go module 依赖 upstream，彻底脱离 fork** |
| A2 | `contrib/e2b-e2e/`（E2B 验收资产） | ~2.4K 行 py/sh | sandbox 仓（原属 sandbox `scripts/`，2026-08 迁入 fork，现物归原主） | 与 e2bgw 同仓后验收脚本直接指向本仓网关 |
| A3 | `manifests/xuanji/**`（wasm-example、wasm-tiers-example、e2bgw.yaml、e2bgw-certmanager/、ack/、ack-m4/）+ 未提交的 two-tier-example.yaml | ~700 行 yaml | sandbox 仓 `deploy/kustomize/` | kustomize base 改指 upstream `manifests/ate-install`（git resource 或版本化 vendored 副本，需验证渲染路径）；ACK overlay 属部署环境事实，与 fork 代码无关 |
| A4 | `manifests/ate-install/cert-manager-pki*/` + `hack/install-ate-certmanager.sh` | ~1.4K 行 | sandbox 仓 `deploy/`（或独立 upstream PR） | **非 wasm 专属**——是「无 PodCertificateRequest gate 集群」的通用部署适配；长期更优解是提 upstream（对 ACK/EKS 用户普遍有价值） |
| A5 | 文档空间：`docs/{decisions,concepts,tasks,reference,figures,...}`、`ARCHITECTURE.md`、`CHANGELOG.md`、collected-materials（~149K 行）、`xuanji.md` 台账 | +237K 行 | 设计文档/ADR → sandbox `docs/`（其 ADR 0001 已在彼处，本文与 ADR 0002 随迁）；collected-materials → sandbox `docs/reference/` 或 profiles 04_知识管理 | fork 回归开源后不应携带私有研究资料；xuanji.md 台账随最后一批 fork delta 消亡前保留于 sandbox 作为迁移记录 |
| A6 | `.specify/` 框架、`.claudeignore`、`.gitignore` 追加行 | 大 | 不迁移，随 fork 收敛直接删除 | sandbox 仓已有自己的 .specify；fork 侧治理文档无保留价值 |

### B 类 — 搬不走，只能「最小 fork delta / upstream 化 / 条件消失」

触碰 upstream 既有代码的 14 个文件，全部属于以下两组：

| # | 改动 | 体量 | 为什么搬不走 | 处置 |
|---|---|---|---|---|
| B1 | **wasm SandboxClass enum 缝**：`pkg/api/v1alpha1/{sandboxconfig,actortemplate,workerpool}_types.go`（Enum + 常量 + CEL 文案）+ 2 个 validation test + 3 个 generated CRD + `sandboxconfig-validation.yaml` VAP 规则 + `workerpool_apply.go` 的 wasm securityContext 分支（drop ALL）+ 对应测试 | ~40 行实质改动 + 生成物 | CRD/enum 是 substrate API 本体；securityContext 分支在 atecontroller 内 | 三选一：(a) **upstream PR**（新增 runtime class + wasm 零特权 securityContext 属安全加固，可拆 2 个 PR）；(b) 保留为**唯一 fork delta**（面小、rebase 稳定，xuanji.md 已登记）；(c) ADR 0002 去 supervisor 语义后**不新增任何 enum 值**（两层形态 = 既有 wasm class + `WorkerPool.runtimeClassName: rund` 扩展字段），enum 缝的 upstream 诉求只剩 wasm 一项——推荐 (a) 单独推进，被拒则退 (b) |
| B2 | **cert-manager PKI 三开关**：`cmd/ateapi/main.go` + `controlapi/dialer.go`（`--atelet-require-pod-identity-ext`）、`cmd/atecontroller/main.go` + `workerpool_controller.go` + `workerpool_apply.go`（`--worker-cert-source`）、`internal/ateclient/builder.go`（CA ConfigMap 回退）+ 测试 | ~130 行 | 改的是 substrate 二进制自身行为 | 全部**默认 OFF、保持 upstream 行为**，仅 overlay 翻转 → (a) upstream PR（外部签发器支持，降级面已有 WARN 显式化）；(b) 被拒则留 fork delta；(c) **ACK 支持 PCR gate 后整组删除**（连同 A4 overlay）——这是有自然消失路径的 delta |

### C 类 — 已在 sandbox，无需动作

ateom-wasmd（Rust 运行时）、Ateom proto vendoring（`cmd/ateom-wasmd/proto/ateom.proto`，fork rebase 时需重拷——该约定随 fork 收敛消失，改为依赖 upstream 发布版 proto）。

## 3. 目标终态

```
substrate（fork）  delta → 0（B1/B2 全部 upstream 化）
                   最坏  → 2 块 rebase 稳定的小 delta（B1 enum 缝 ~40 行 + B2 PKI 开关 ~130 行，均默认无行为变化）
sandbox（定制唯一承载者）
  ├── cmd/ateom-wasmd/     wasm host 运行时（Rust，已在）
  ├── cmd/e2bgw/           E2B 网关 + 自研管控（Go，A1 迁入；go.mod require github.com/agent-substrate/substrate）
  ├── contrib|e2e/         E2B 验收资产（A2）
  ├── deploy/              xuanji/ACK overlay + cert-manager PKI + 安装脚本（A3/A4）
  └── docs/                ADR/设计/台账/参考资料（A5）
依赖方向：sandbox → upstream substrate（公开 module + proto），fork 不再是任何构建的必经依赖
```

这与 ADR 0002 两层框架对齐：Tier-1 尽量用开源 substrate 原样（+最小 enum 缝），Tier-2 的 e2b/自研管控/wasm 全部在 sandbox 仓。

## 4. 迁移前置与风险（按序）

1. **[GATED] 分支收敛先行**：`xuanji-wasm-preflight`（全部特性）与 `xuanji`（3 个文档提交）分叉，迁移前须定唯一基线（建议 preflight rebase/merge 收口，需用户裁决，勿自动 merge）。
2. **未提交设计产物**：substrate-preflight worktree 处 detached HEAD，ADR 0002 + 概念文档 + two-tier-example.yaml + 本评估未提交——随 A5 直接落 sandbox（避免在 fork 里再产生新提交），或先提交到分支再迁（用户裁决）。
3. **internal/ateapiauth**：复制随迁需保留版权头与出处；推动 upstream 提升 pkg/ 是更干净路径，但周期不可控——建议先复制随迁、PR 并行。
4. **sandbox 仓新增 Go 工具链**：Rust+Go 双栈，Makefile/CI/镜像管线（`make image-e2bgw`）需在 sandbox 仓补齐；构建容器从 substrate_alios_8 切到 sandbox_alios_8（或新容器）。
5. **kustomize base 跨仓引用**：A3/A4 overlay 目前 `resources: ../../ate-install`；迁到 sandbox 后改 git resource（`github.com/agent-substrate/substrate/manifests/ate-install?ref=<tag>`）或 vendored 副本，两种方式都需渲染验证（`--load-restrictor LoadRestrictionsNone` 路径约束会变）。
6. **upstream 接受度未知**：substrate 处于 pre-alpha；B1/B2 PR 被拒即回退「最小 fork delta」路径，不阻塞迁移主体（A 类）。
7. **集群侧回归**：迁移后镜像构建源变化（e2bgw 从 sandbox 仓构建），cluster-msaFE8 需一轮 e2e 复验（wasm Actor 全链路 + e2bgw E2B 面）。

## 5. 建议执行顺序

1. 分支收敛 + 设计产物落位（GATED，用户裁决）
2. A1 e2bgw + ateapiauth 随迁 sandbox，建 go.mod，本地 `go build && go test` 绿
3. A2 e2e 资产回迁；A3/A4 manifests+脚本迁 deploy/ 并验证 kustomize 渲染
4. A5 文档迁移（ADR 编号衔接 sandbox 侧：0002 起）；fork 侧删除已迁目录
5. B1/B2 拆 PR 提 upstream（enum+securityContext 一个、PKI 开关一个、可选 ateapiauth 提升一个）
6. fork delta 收敛验证：`git diff main..xuanji` 只剩 B 类（或为零）；xuanji.md 台账收口归档
7. 集群 e2e 回归（新构建源）
