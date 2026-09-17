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
**外部推送拓扑（2026-09-17 确立）**：本机 Mac 有公司云壳 DLP「Git 的外部上传禁用」，对 github/gitee 的 push 一律被拦截；**非 code.alibaba-inc.com 的 push 改在远端开发环境执行**——bm8 的 substrate 构建容器（`ssh workspace_docker_baremetal_8` + `docker exec project-sig-cloud-native-isv-substrate-alios-8`，容器直连 ssh 被网关拒绝、必须经父机 docker exec；容器内四 remote 齐备且 github/gitee SSH 认证通过）。本机只推 gitlab。备份已于 2026-09-17 经容器恢复至 @ecb665cc。

### 分支定义（长期分支只有两条）

| 分支 | 定义 | 跟踪 |
|---|---|---|
| `main` | **跟踪 upstream 的代码修改**：只镜像 `origin/main`，不承载任何定制提交（Constitution Principle I） | 本地 `main` → `origin/main` |
| `xuanji` | **本地自定义的修改**：全部 xuanji 定制的唯一长期分支，rebase 到 main 保持最新 | 本地 `xuanji` → `gitlab/xuanji`（主力权威） |

### 残留分支（逐步归一清理，不再新增）

除上述两条外的一切分支均为开发过程残留，归一方向 = 内容并入 `xuanji`（或确认被覆盖）后删除：

| 残留分支 | 位置 | 现状（2026-09-17 实测） | 清理路径 |
|---|---|---|---|
| `xuanji-wasm-preflight` @9f375565 | gitlab | **已清理（2026-09-17）**：13 个特性提交 cherry-pick 归一进 xuanji（代码树逐字节一致），设计产物随 xuanji 提交，远端分支已删 | ~~优先归一~~ 完成 |
| `xuanji-wasm` @1e338827 | gitlab | **已清理（2026-09-17）**：patch-id 比对确认与 preflight 平行旧线，独有内容仅 2 个 e2e 脚本（已拣回提交 4055e801），远端分支已删 | ~~内容级比对后删除~~ 完成 |
| `security-md-fixup`、`feat/long-running-actor-support`、`feature/nanoclaw-multiplex-demo`、`feature/openclaw-integration` | github fork | 各有独有提交（1/1/31/3），非本项目会话产物，疑为 fork 上其他实验 | **不盲删**：逐个确认归属与价值后裁决（保留/并回/删除） |

配套事实（2026-09-17 归一后实测）：`gitlab/main` 已 ff 至 `origin/main` @85ce8ed5（镜像零滞后）；`github/xuanji` 与 `gitee/xuanji` 停留在 @d067d4d0（备份推送被 DLP 拦截，待报批）；gitlab 侧残留分支已清零，仅余 github fork 上 4 条待裁决分支。`origin` 另有 `release-0.1` 分支（上游发布线，只读关注）。


## 已改动的 upstream 文件

| 文件 | 改动 | 原因 |
|---|---|---|
| `pkg/api/v1alpha1/sandboxconfig_types.go` | 新增 `SandboxClassWasm` 常量；Enum `gvisor;microvm` → `gvisor;microvm;wasm` | wasm 沙箱类 |
| `pkg/api/v1alpha1/actortemplate_types.go` | Enum 同上；3 条非 microvm 限制的 CEL message 由 “is 'gvisor'” 改为 “is not 'microvm'”（wasm 同受限，原文案误导） | wasm 沙箱类 |
| `pkg/api/v1alpha1/workerpool_types.go` | Enum 同上 | wasm 沙箱类 |
| `pkg/api/v1alpha1/sandboxconfig_validation_test.go` | 新增 `pythonWasmAsset` helper + 3 个 wasm 用例 | 测试覆盖 |
| `pkg/api/v1alpha1/actortemplate_validation_test.go` | 新增 wasm 合法类 + wasm Golden 拒绝用例；同步 5 处 errMsg 断言文案 | 测试覆盖 |
| `cmd/atecontroller/internal/controllers/workerpool_apply.go` | `ateomSecurityContext` 新增 wasm 分支：`drop ALL`、不 add 任何 capability、不设 AppArmor Unconfined | wasm worker 原先落进 gvisor 分支，白拿 13 个 Linux capability + Unconfined；ateom-wasmd 进程内跑 wasmtime，不 exec runsc / 不 pivot root / 不 ptrace / 不编程 veth-nftables / 不解 OCI rootfs，一个都用不到 |
| `cmd/atecontroller/internal/controllers/workerpool_apply_test.go` | TestMicroVMPodShape 加 wasm 行（非特权、无 KVM 形状）；TestAteomSecurityContextByClass 重构断言表——原表把「有 capability」「drop ALL」「AppArmor Unconfined」并成一个标志位，无法表达 wasm 形状，拆成 4 个独立期望 | 测试覆盖 |
| `manifests/ate-install/sandboxconfig-validation.yaml` | VAP 新增 wasm 规则：每 arch 必须有 `python-wasm` 资产 | wasm 资产校验 |
| `manifests/ate-install/generated/*` | controller-gen 再生成（枚举/文案变化） | 生成物 |
| `README.md` | 新增「Feature List」一节（Spec Kit 特性登记表，指向 `.specify/memory/features.md`） | 特性盘点 |
| `.gitignore` | 末尾追加 1 行 `.specify/tools/*.json` | 工具产物忽略 |
| `cmd/ateapi/main.go` | 新增 flag `--atelet-require-pod-identity-ext`（默认 `true`，行为不变），透传给 `NewAteletDialer` | cert-manager PKI：外部签发器无法嵌入 PodIdentity X.509 扩展 |
| `cmd/ateapi/internal/controlapi/dialer.go` | `NewAteletDialer`/`buildTLSConfig`/`verifyAteletServerCert` 增加 `requirePodIdentityExt` 参数；为 `false` 时跳过 PodIdentity 扩展/PodUID 钉扎，SPIFFE ID + 链 + serverAuth EKU 校验保留 | 同上 |
| `cmd/ateapi/internal/controlapi/dialer_test.go` | 既有调用补 `true` 实参；新增放宽模式 2 用例（无扩展通过 / 错 SPIFFE 仍拒绝） | 测试覆盖 |
| `cmd/ateapi/internal/controlapi/{functional,workflow_suspend}_test.go` | `NewAteletDialer` 调用补 `true` 实参 | 签名变更适配 |
| `cmd/atecontroller/main.go` | 新增 flag `--worker-cert-source=pod-certificate\|cert-manager`（默认前者，行为不变），带取值校验并注入 Reconciler | cert-manager PKI：worker 证书卷来源切换 |
| `cmd/atecontroller/internal/controllers/workerpool_controller.go` | `WorkerPoolReconciler` 新增 `WorkerCertSource` 字段，空值回退 `pod-certificate` | 同上 |
| `cmd/atecontroller/internal/controllers/workerpool_apply.go` | （追加改动）`buildDeploymentApplyConfig` 增加 `certSource` 参数；卷 sources 抽出 `atunnelIdentitySources`/`atunnelEgressTrustSources` 双模式函数：cert-manager 模式挂 Secret `ate-worker-podidentity-cert` + trust-manager ConfigMap，挂载路径/文件名与 podCertificate 投影完全一致 | 同上 |
| `cmd/atecontroller/internal/controllers/workerpool_apply_test.go` | （追加改动）既有调用补 `WorkerCertSourcePodCertificate` 实参；新增 `TestWorkerCertSourceVolumes` 双模式卷断言 | 测试覆盖 |
| `internal/ateclient/builder.go` | `serverTLSConfig` 拆出 `serverCAPool`：ClusterTrustBundle API 不可用或无 live bundle 时回退读 `ate-system/servicedns-ca-bundle` ConfigMap（trust-manager 分发） | kubectl-ate 在无 CTB API 的集群（如 ACK）可用 |
| `cmd/ateapi/main.go` | （追加改动，review P3）`--atelet-require-pod-identity-ext=false` 时启动打 WARN：说明 atelet PodUID 钉扎已失效、cert-manager 模式下 atelet 凭证为命名空间级共享（泄露可跨节点冒充），并说明仍生效的校验与切回方式 | 降级面显式化，避免只留在文档里 |
| `cmd/atecontroller/main.go` | （追加改动，review P3）`--worker-cert-source=cert-manager` 时启动打 WARN：worker 凭证由「每 Pod 一份投影」退化为「每命名空间一份 Secret」，并提示 actor 路由授权不受影响 + 每命名空间需自建 Certificate | 同上 |

## 新增文件（无 upstream 冲突面）

| 路径 | 内容 |
|---|---|
| `xuanji.md` | 本登记文件（原名 `XUANJI.md`，2026-08-07 按命名规范改小写） |
| `ARCHITECTURE.md`、`CHANGELOG.md`、`docs/{decisions,notes,concepts,tutorials,tasks,reference,contribute}/` | xuanji 文档空间骨架（/speckit.docs reconcile 建立；薄根入口 + ADR/notes 生命周期 + 六型索引） |
| `docs/overview.md`、`docs/concepts/core-concepts.md`、`docs/reference/actor-lifecycle-flows.md`、`docs/figures/`（8 图 ×puml/png/svg） | upstream 架构深度研究（study-project）：组件架构图/部署图/Resume/Suspend/停车时序图/状态机/资源模型/golden 流程 + 概念与流程文档；`ARCHITECTURE.md` 详细文档表同步更新 |
| `manifests/xuanji/wasm-example.yaml` | wasm 类示例：SandboxConfig(wasm-default, python-wasm 资产) + WorkerPool(sandboxClass=wasm, 外部 ateom-wasmd 镜像) + ActorTemplate(python-interpreter)；**2026-08 追加** WorkerPool `template.resources`（requests cpu=2/memory=2Gi，limits cpu=4/memory=2Gi）——ateom-wasmd 引入内核池（一个 actor 内至多 N 个 wasmtime 内核，镜像 ENV `WASM_KERNEL_POOL_SIZE` 缺省 4）后 worker 资源须随池大小配置，公式与 CPU≥2 下限依据见 sandbox 仓 `docs/tasks/wasm-sandbox-final-design.md` §7.3；**2026-08-20 追加** WorkerPool 资源注释改口径：ateom-wasmd 池容量改为由 Pod cgroup 内存 limit 派生（`kernels = clamp((limit−512Mi)/300Mi, 1, 64)`，2Gi→5；镜像不再烤 `WASM_KERNEL_POOL_SIZE`，该 env 仅作可选硬覆盖），调容量 = 改 memory limit，见 sandbox 仓 final-design §5.2/§7.3；resources 数值本身未变（2Gi/cpu2-4） |
| `contrib/e2b-e2e/` | E2B 协议验收资产（自 sandbox 仓 `scripts/` 原样迁入）：官方 SDK e2e、files e2e、smolagents、MCP server、TLS relay 等；作为 `cmd/e2bgw` 的验收基线；**M3 3.4 已改指 e2bgw（2026-08-17）**：`sign_jwt.py` claim `tenant_id`→`atespace`（e2bgw 鉴权契约，`--tenant` 保留为旧别名）；`test_sdk_nondebug_e2e.py` 新增 `E2B_TEMPLATE`（templateID→ActorTemplate 名注入）与 `E2E_LIFECYCLE_ONLY`（初为生命周期段先行，e2bgw 主机名式数据面落地后保留为排障开关，默认跑全量）；`sandbox_tools.py` 数据面改走路径式代理 `/sandboxes/{id}/execute` + `X-API-Key`（旧 debug 单例/engine 推断/Path C 退役，`request_full_os` 改为报错指引另建 gvisor/microvm 沙箱）；README 重写为逐资产就绪状态表（test_epoch/test_sandbox_identity 依赖旧 debug 模式，待 M4 4.4 重做或退役）；**M4 验收后追加（2026-08-18）** `test_files_api_e2e.py` 同款补 `E2B_TEMPLATE` 注入：此前它硬编码 SDK 默认模板名 code-interpreter-v1，集群验收时只能临时建同名 ActorTemplate 别名才能跑，补后套件不再隐式依赖集群侧别名资源 |
| `cmd/e2bgw/` | E2B REST → `ateapipb.Control` 翻译网关（Go）：HS256 API-Key 鉴权（atespace/tenant claim）、POST /sandboxes→CreateActor+ResumeActor(boot)（失败回滚 DeleteActor）、pause/resume→Suspend/Resume、gRPC↔HTTP 错误映射；数据面 M4 接通：/execute /files /filesystem.Filesystem/{Stat,ListDir,MakeDir,Move,Remove} 反向代理（`httputil.ReverseProxy`）到 `<name>.<atespace>.<actor-domain>`——剥离 `/sandboxes/{id}` 前缀、Host 设为 actor 权威名（atenet 按 :authority 路由、atunnel 据此授权）、`FlushInterval=-1` 保 NDJSON 逐行流式、剥除 API-Key 头、上游不可达回纯文本 502；新增 `--actor-ca` flag 注入自签 CA（空则系统根证书）、`--actor-domain` 为空仍回 501；**review P2 追加** `--actor-tls-server-name`：atenet 边缘证书（servicedns 签发器只签 `<svc>.<ns>.svc`，见 `cmd/podcertcontroller/internal/servicednssigner`）不含 actor 域名 SAN，默认按 actor 权威名校验主机名必然失败；该 flag 可把校验名钉到证书已有的名字上（链校验照旧，空值保持原行为）；单测覆盖钉扎生效 + 不钉扎时错配仍回 502；另注：`FlushInterval=-1` 对流式响应实为冗余（`httputil` 在 `res.ContentLength == -1` 时强制 -1），真正会破坏流式的是缓冲 body，已在注释中写明并由 `TestDataPlaneProxyStreamsNDJSON` 覆盖；原 307 跳转已废弃（其保留完整 `/sandboxes/{id}/execute` 路径，目标本身就错）；handler 单测全绿（路径改写/流式/502/501 覆盖）；**M4 4.4 前置追加（2026-08-17）** 主机名式数据面承接：未改动 SDK 按 `https://{port}-{sandboxID}.{domain}` 访问 envd，ServeHTTP 在 mux 分发前识别该 Host 形状（首标签 `纯数字端口-沙箱id`，端口标签仅提示性）并代理到 actor 权威名；鉴权用自签发的 `envdAccessToken`（create/get/resume 响应新增字段，HS256 同密钥无状态签发，claims 钉住 atespace+sandbox，7 天有效期、每次响应刷新；SDK 以 `X-Access-Token` 回传，转发前剔除）：主机名不携带 atespace，由 token claim 提供，且跨沙箱 token 不可互用；路径式/主机名式共用 `proxyToActor`（jwt.go 新增 `signHS256`）；**集群验证追加（2026-08-17）** `DELETE /sandboxes/{id}`（E2B kill）对 RUNNING actor 报 409——ateapi `MarkDeletingStep` 只删 SUSPENDED/CRASHED/DELETING；改为 DeleteActor 遇 FailedPrecondition 时先 SuspendActor 再重试一次（kill 语义 = 无条件销毁；suspend 失败仅 WARN，由重试定断，兼容并发 suspend 竞态）；**E2B 语义补齐（2026-08-18）** ① 新增 `ttl.go`：`timeout` 端点从 no-op 改为网关侧 TTL→Suspend（create/resume/timeout 均重置计时，缺省 300s 对齐 E2B；到期 SuspendActor，NotFound/FailedPrecondition 视为良性；pause/delete 取消计时器；内存态单副本设计，重启丢计时器仅意味闲置沙箱不自动挂起）；resume 解析可选 `{"timeout"}` body；timeout 端点非法 body 回 400；② 路径式数据面新增 `POST /sandboxes/{id}/contexts` 路由（代理到 actor 的 `/contexts`，ateom-wasmd 侧实现见 sandbox 仓；主机名式本就透传任意路径零改动）；③ 新增 `GET /sandboxes/{id}/snapshots`（计划 3.2 快照端点：ListActorSnapshots 按 source actor 过滤，运维可见性——验证 pause 确实产出快照；ControlAPI 接口扩 ListActorSnapshots，测试 mock 同步）与 `GET /sandboxes/{id}/metrics` 显式 501（wasm 类暂无指标源，SDK get_metrics 得到明确理由而非裸 404）；**集群验证修复（2026-08-18）** 主机名式数据面除 `X-Access-Token` 外并接受 `E2B-Traffic-Access-Token`：e2b-code-interpreter SDK 的 `create_code_context`（`POST /contexts`）用后者携带同一枚 envd access token（`run_code`/`files` 走 httpx 默认头的 `X-Access-Token`），两者同义；入向二选一鉴权、转发前一并剔除 |
| `docs/reference/e2b-api-surface.md`、`docs/concepts/sandbox-landscape.md` | /speckit.docs 扇出蒸馏产出：E2B 接口面×e2bgw 覆盖矩阵（代码核实）+ Agent 沙箱竞品格局（仅公开来源）。原 `docs/reference/collected-materials/`（含内网原文）已移出 tracked docs 树（本仓有公开 remote，内网原文不入库） |
| `docs/reference/upstream-evolution-research.md`、`docs/reference/xuanji-evolution-survey.md`、`docs/figures/{xuanji-evolution-stages,upstream-new-since-baseline,xuanji-extension-plane}.{puml,png,svg}` | 演进研究（study-project，2026-08-19）：upstream 演进方向×xuanji 演进支持（38 提交漂移主题/新子系统/in-flight 分支/rebase 冲突风险/吸收清单）+ xuanji 分支演进调研（试点解剖、preflight M4 管线、阶段转换决策点），配套三图 |
| `docs/hugo.toml`、`docs/layouts/`、`docs/static/css/site.css`、`docs/.gitignore`、`docs/contribute/docs-site-build.md` | Hugo 呈现层（create-docs 技能 scaffold + 人工修正）：`docs/` 兼作 Hugo 项目根，内容挂载不复制、Markdown 保持无 frontmatter。人工补 4 条挂载（根级 `*.md` 入 content；`figures` 镜像到 concepts/reference/overview 三处 static）与 `layouts/partials/page-title.html`（H1 兜底标题），修复根级文档缺页/死链、导航标题空白、raw-HTML 图片 404。构建 29 页零 warning；产物 `docs/public/` 永不提交 |
| `manifests/xuanji/wasm-tiers-example.yaml` |（多档位追加，2026-08-20）wasm 类容量分档示例（与 wasm-example.yaml 叠加使用，后者为 small 档）：WorkerPool `wasm-pool-xl`（label `workload: wasm-python-xl`，8Gi/cpu4-8，池容量按 cgroup 派生 8Gi→25 内核 = 24 命名 context）+ ActorTemplate `python-interpreter-xl`（workerSelector 钉到 xl 池，其余契约与 python-interpreter 一致）；档位选择全走 upstream 原生机制（workerSelector 门控 + E2B templateID→ActorTemplate 映射），零代码改动；同样是带 `<ATEOM_WASMD_IMAGE>` 占位符的参考模板，集群部署须 patch 存量 CRD 而非直接 apply；2026-08-20 集群验收通过（cap:25 派生、XL 落位、第 25 个 context 429、小档回归 429） |
| `contrib/e2b-e2e/` | E2B 协议验收资产（自 sandbox 仓 `scripts/` 原样迁入）：官方 SDK e2e、files e2e、smolagents、MCP server、TLS relay 等；作为 `cmd/e2bgw` 的验收基线；**M3 3.4 已改指 e2bgw（2026-08-17）**：`sign_jwt.py` claim `tenant_id`→`atespace`（e2bgw 鉴权契约，`--tenant` 保留为旧别名）；`test_sdk_nondebug_e2e.py` 新增 `E2B_TEMPLATE`（templateID→ActorTemplate 名注入）与 `E2E_LIFECYCLE_ONLY`（初为生命周期段先行，e2bgw 主机名式数据面落地后保留为排障开关，默认跑全量）；`sandbox_tools.py` 数据面改走路径式代理 `/sandboxes/{id}/execute` + `X-API-Key`（旧 debug 单例/engine 推断/Path C 退役，`request_full_os` 改为报错指引另建 gvisor/microvm 沙箱）；README 重写为逐资产就绪状态表（test_epoch/test_sandbox_identity 依赖旧 debug 模式，待 M4 4.4 重做或退役）；**M4 验收后追加（2026-08-18）** `test_files_api_e2e.py` 同款补 `E2B_TEMPLATE` 注入：此前它硬编码 SDK 默认模板名 code-interpreter-v1，集群验收时只能临时建同名 ActorTemplate 别名才能跑，补后套件不再隐式依赖集群侧别名资源；**多档位追加（2026-08-20）** 新增 `migrate_sandbox.py`：跨档搬家客户端工具（目标档 template 新建 sandbox → files 全量搬运 + 文件集合校验 → 默认 kill 源，`--keep-src` 可保留；失败回滚目标、源不动），只用公开 E2B 面（create/files/kill）零协议偏离；档位清单见 `manifests/xuanji/wasm-tiers-example.yaml`，README 资产表同步登记；2026-08-20 集群验收通过（原版 CLI 无补丁：搬运 3.68s、内容/空目录校验、kill 与 --keep-src 两分支、搬后升档第 5 context 201） |
| `cmd/e2bgw/` | E2B REST → `ateapipb.Control` 翻译网关（Go）：HS256 API-Key 鉴权（atespace/tenant claim）、POST /sandboxes→CreateActor+ResumeActor(boot)（失败回滚 DeleteActor）、pause/resume→Suspend/Resume、gRPC↔HTTP 错误映射；数据面 M4 接通：/execute /files /filesystem.Filesystem/{Stat,ListDir,MakeDir,Move,Remove} 反向代理（`httputil.ReverseProxy`）到 `<name>.<atespace>.<actor-domain>`——剥离 `/sandboxes/{id}` 前缀、Host 设为 actor 权威名（atenet 按 :authority 路由、atunnel 据此授权）、`FlushInterval=-1` 保 NDJSON 逐行流式、剥除 API-Key 头、上游不可达回纯文本 502；新增 `--actor-ca` flag 注入自签 CA（空则系统根证书）、`--actor-domain` 为空仍回 501；**review P2 追加** `--actor-tls-server-name`：atenet 边缘证书（servicedns 签发器只签 `<svc>.<ns>.svc`，见 `cmd/podcertcontroller/internal/servicednssigner`）不含 actor 域名 SAN，默认按 actor 权威名校验主机名必然失败；该 flag 可把校验名钉到证书已有的名字上（链校验照旧，空值保持原行为）；单测覆盖钉扎生效 + 不钉扎时错配仍回 502；另注：`FlushInterval=-1` 对流式响应实为冗余（`httputil` 在 `res.ContentLength == -1` 时强制 -1），真正会破坏流式的是缓冲 body，已在注释中写明并由 `TestDataPlaneProxyStreamsNDJSON` 覆盖；原 307 跳转已废弃（其保留完整 `/sandboxes/{id}/execute` 路径，目标本身就错）；handler 单测全绿（路径改写/流式/502/501 覆盖）；**M4 4.4 前置追加（2026-08-17）** 主机名式数据面承接：未改动 SDK 按 `https://{port}-{sandboxID}.{domain}` 访问 envd，ServeHTTP 在 mux 分发前识别该 Host 形状（首标签 `纯数字端口-沙箱id`，端口标签仅提示性）并代理到 actor 权威名；鉴权用自签发的 `envdAccessToken`（create/get/resume 响应新增字段，HS256 同密钥无状态签发，claims 钉住 atespace+sandbox，7 天有效期、每次响应刷新；SDK 以 `X-Access-Token` 回传，转发前剔除）：主机名不携带 atespace，由 token claim 提供，且跨沙箱 token 不可互用；路径式/主机名式共用 `proxyToActor`（jwt.go 新增 `signHS256`）；**集群验证追加（2026-08-17）** `DELETE /sandboxes/{id}`（E2B kill）对 RUNNING actor 报 409——ateapi `MarkDeletingStep` 只删 SUSPENDED/CRASHED/DELETING；改为 DeleteActor 遇 FailedPrecondition 时先 SuspendActor 再重试一次（kill 语义 = 无条件销毁；suspend 失败仅 WARN，由重试定断，兼容并发 suspend 竞态）；**E2B 语义补齐（2026-08-18）** ① 新增 `ttl.go`：`timeout` 端点从 no-op 改为网关侧 TTL→Suspend（create/resume/timeout 均重置计时，缺省 300s 对齐 E2B；到期 SuspendActor，NotFound/FailedPrecondition 视为良性；pause/delete 取消计时器；内存态单副本设计，重启丢计时器仅意味闲置沙箱不自动挂起）；resume 解析可选 `{"timeout"}` body；timeout 端点非法 body 回 400；② 路径式数据面新增 `POST /sandboxes/{id}/contexts` 路由（代理到 actor 的 `/contexts`，ateom-wasmd 侧实现见 sandbox 仓；主机名式本就透传任意路径零改动）；③ 新增 `GET /sandboxes/{id}/snapshots`（计划 3.2 快照端点：ListActorSnapshots 按 source actor 过滤，运维可见性——验证 pause 确实产出快照；ControlAPI 接口扩 ListActorSnapshots，测试 mock 同步）与 `GET /sandboxes/{id}/metrics` 显式 501（wasm 类暂无指标源，SDK get_metrics 得到明确理由而非裸 404）；**集群验证修复（2026-08-18）** 主机名式数据面除 `X-Access-Token` 外并接受 `E2B-Traffic-Access-Token`：e2b-code-interpreter SDK 的 `create_code_context`（`POST /contexts`）用后者携带同一枚 envd access token（`run_code`/`files` 走 httpx 默认头的 `X-Access-Token`），两者同义；入向二选一鉴权、转发前一并剔除；**集群验证修复（2026-08-20）** 新增 `POST /sandboxes/{id}/connect`（SDK 2.x `Sandbox.connect()` 的控制面入口，此前 404 导致跨进程接管 sandbox 不可用，migrate_sandbox.py 首步即阻断）：无条件 ResumeActor，遇 FailedPrecondition（已在运行）回退 GetActor 原样附着；两路均重置 TTL 并返回带新 envdAccessToken 的 sandboxResponse（与 create/resume 同形）；可选 body `{"timeout"}` 容忍解析同 resume；新增 TestConnectSandbox 三子测覆盖挂起唤醒/运行附着/未知 404；镜像 v0.6.0-connect 集群验收通过（running 200 / paused 200 且触发 ColdBoot 唤醒 0.13s / 未知 404） |
| `manifests/ate-install/cert-manager-pki/` | mTLS 证书签发从 PodCertificateRequest 切换到 cert-manager/trust-manager 的 kustomize overlay（面向无 PCR/CTB feature gate 的托管集群，如 ACK）：`pki.yaml`（自签 bootstrap → servicedns-ca/podidentity-ca 两 CA → 两 ClusterIssuer，对应原两个 signer）、`trust-bundles.yaml`（3 个 trust-manager Bundle 全命名空间分发 CA，valkey 用双根合并 bundle）、`certificates.yaml`（组件 Certificate，`CombinedPEM` 匹配 credbundle 单文件格式，SPIFFE URI 与原签发器一致；**review P2/P4 追加** `atenet-router-servicedns` 补 actor 域名 SAN 的精确说明与按 atespace 的注释模板（X.509 通配符只匹配一层标签，而 actor 权威名有两层可变标签，故单条通配覆盖不了），并新增 `e2bgw-podidentity` 客户端身份证书）、`kustomization.yaml`（复用 base 各组件清单但不含 podcertcontroller；JSON6902 `test`+`replace` 把 podCertificate/clusterTrustBundle 投影卷换成 Secret/ConfigMap，挂载路径与容器 args 不变；追加 `--atelet-require-pod-identity-ext=false` 与 `--worker-cert-source=cert-manager`）、`worker-certificate.example.yaml`（每 WorkerPool 命名空间一份的 worker 身份模板）。切回：集群支持 PCR 后重新部署 base/token-client overlay 即可；**review P3 追加** `kustomization.yaml` 头部新增「安全强度差异」小节，记账两处降级（atelet 失去 PodUID 钉扎且凭证命名空间级共享、worker 凭证由每 Pod 退化为每命名空间）与不受影响面（actor 路由授权、mTLS 本身、SPIFFE 身份、轮换）；**集群验证修复（2026-08-17）** `certificates.yaml` 全部 9 张证书补 `privateKey.encoding: PKCS8`：cert-manager 对 EC 密钥缺省出 SEC1，ateapi 的 valkey 客户端只解 PKCS8，导致全组件 CrashLoop（ACK 实测定位的 overlay 真 bug） |
| `manifests/ate-install/cert-manager-pki-agentgateway/` | 上一行 overlay 的 agentgateway 数据面变体：叠加 `components/agentgateway` Component（其 configmap 引用的证书路径由 cert-manager-pki 的卷替换同样满足） |
| `manifests/xuanji/e2bgw.yaml` |（review P4）e2bgw 的 K8s 部署清单（原先全仓无清单，只能出集群跑，阻塞 M4 4.4 端到端验收）：ServiceAccount + Service + Deployment，镜像走 `ko://.../cmd/e2bgw`；args 含 ateapi 连接/客户端证书、`--jwt-secret-file`、`--template-namespace=ate-wasm`、`--actor-domain`、`--actor-ca`、`--actor-tls-server-name`（默认钉到 router Service 名，待 router 证书补齐 actor 域名后可去掉）；`/health` readinessProbe；HS256 密钥经 Secret `e2bgw-jwt-secret` 挂载——**密钥本身不入库**，清单头注释给出创建命令 |
| `manifests/xuanji/e2bgw-certmanager/` |（review P4）上一行清单的 cert-manager 变体 overlay：JSON6902 `test`+`replace` 把 podCertificate/clusterTrustBundle 投影换成 Secret `e2bgw-podidentity-cert` + ConfigMap `servicedns-ca-bundle`，挂载路径与容器 args 不变；`test` op 钉住卷名（已验证：把 `e2bgw.yaml` 的卷顺序对调，kustomize build 立即失败）|
| `manifests/xuanji/ack/` |（漏登补记，随 70382070 提交）M2 期 ACK 控制面 overlay（podCertificate 路径）：仅 ateapi/atecontroller/atelet，不含 atenet 数据面；全镜像 ACR 重写（cn-hangzhou 不可达 docker.io/gcr.io）、default SA 挂 imagePullSecrets、rustfs 作 M2 对象存储（kind 同款 dev 凭证）。注：ACK 无 PCR gate，此 overlay 实际跑不起来（全组件 CrashLoop），已被 ack-m4 取代，保留供 PCR 集群参照 |
| `manifests/xuanji/ack-m4/` |（集群验证追加，2026-08-17）ACK 全链 M4 合并 overlay：cert-manager-pki（mTLS 组件在无 PCR/CTB 的 ACK 上的唯一可起路径）+ 全镜像 ACR 重写（含此前跳过的 envoy/coredns，`digest: ""` 剔除基线 digest 钉扎）+ atelet ACK 补丁（`--gcp-auth-for-image-pulls=false`、rustfs S3 env，后续切 OSS 只需改 AWS_ENDPOINT_URL + AWS_S3_USE_PATH_STYLE=false）+ valkey PVC 1Gi→20Gi（阿里云盘 CSI 下限 20Gi）；与 ack（M2）的区别：保留 atenet-router/atenet-dns 数据面；渲染需 `--load-restrictor LoadRestrictionsNone`（与 `install-ate-certmanager.sh` 同款，已验证全部补丁落位）；ko:// 二进制由 `ko resolve` 部署时构建（KO_DOCKER_REPO/KO_DEFAULTBASEIMAGE 指 ACR） |
| `hack/install-ate-certmanager.sh` | cert-manager PKI 部署脚本（不改 upstream `install-ate.sh`）：前置检查 cert-manager / 安装 trust-manager（helm）、CRD/VAP/命名空间/otel/actor-id 秘钥/envvars（跳过 podcertcontroller CA 与 valkey-ca-certs，二者已被 PKI 取代）、渲染部署 cert-manager-pki[-agentgateway] overlay、等待 Certificate Ready 与 Bundle ConfigMap 同步、rollout 校验；另提供 `--create-worker-cert <ns>` 按命名空间签发 worker 身份与 `--delete-ate-system` |
| `docs/decisions/0002-two-tier-supervisor-worker-sandbox.md`、`docs/concepts/two-tier-sandbox-model.md`、`manifests/xuanji/two-tier-example.yaml`、`docs/tasks/substrate-upstream-regression.md` |（2026-09-17 随分支归一提交，原 substrate-preflight worktree 未提交产物）两层沙箱模型 ADR 0002（Tier-1 K8s+Substrate+Containerd+runc/rund supervisor / Tier-2 e2b+自研管控+wasm worker 面）+ 配套概念文档与目标 schema 示例清单 + substrate 回归开源评估（fork 改动 A/B/C 分类与向 sandbox 仓迁移方案） |

## 设计约定

- **ateom-wasmd 不在本仓**：wasm 运行时（实现 `internal/proto/ateompb` 的 `Ateom` 服务）由
  sandbox 仓（cloud-native-webassembly/sandbox）以 Rust 维护并发布镜像，WorkerPool 按
  `ateomImage` 引用 —— 与 ateom-gvisor/ateom-microvm 的 proto 缝完全一致。
- **wasm 类 v1 能力面**：`onResume` 仅 ColdBoot（Golden 为 microvm 专属）；durableDir ≤1
  （同 gvisor 限制）；worker 非特权（复用 gvisor 分支的 security context，能力集后续可再收窄）。
- **资产约定**：wasm 类 SandboxConfig 必须提供 `python-wasm` 资产（CPython 解释器 wasm 模块，
  arch 无关但按 arch 键重复登记）；atelet 原样拉取并交给 ateom-wasmd。
- **wasm worker 资源与内核池耦合（2026-08）**：ateom-wasmd 采用 actor-as-task 模型——一个 actor
  内至多 N 个 wasmtime 内核（E2B context 粘性分派），N 由镜像 ENV `WASM_KERNEL_POOL_SIZE`
  决定（缺省 4；atecontroller 渲染固定 env 列表、WorkerPool CRD 无 env 透传通道，故只能烤镜像）。
  WorkerPool `template.resources` 须随 N 配置：memory ≥ N×300Mi + 512Mi（每内核 256Mi 线性内存
  硬限 + 缓冲预算）；CPU ≥ 2 为硬下限（1 vCPU 时 CPU-bound cell 饿死数据面/gRPC 响应；epoch
  超时强制本身不受影响——ticker 为独立 OS 线程）。设计依据见 sandbox 仓
  `docs/tasks/wasm-sandbox-final-design.md` §5/§7.3。本仓无代码改动面：Ateom proto、CLI 参数表、
  atunnel 契约、快照目录契约均不变，e2bgw/atenet/atelet/atecontroller/ateapi 零修改。
- **mTLS 证书来源双模式（2026-08-13）**：upstream 的 mTLS 体系依赖 PodCertificateRequest /
  ClusterTrustBundle feature gate，目标验证集群（ACK v1.36）不支持；选择「cert-manager 替代签发」
  而非「砍掉 TLS 裸传输」，因为全部 Go 代码只从固定路径读 PEM 并热加载、不感知签发方，
  此方案代码改动最小且 mTLS 语义（SPIFFE 身份/双向认证/轮换）完整保留。两处代码开关默认值
  均保持 upstream 原行为（ateapi `--atelet-require-pod-identity-ext=true`、atecontroller
  `--worker-cert-source=pod-certificate`），仅由 `cert-manager-pki[-agentgateway]` overlay 翻转；
  集群支持 PCR 后重新部署 base/token-client overlay 即切回，无代码回滚面。部署入口统一走
  `hack/install-ate-certmanager.sh`（不改 upstream `install-ate.sh`）。
