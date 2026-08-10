# FC sandbox功能对齐梳理 20260529

- **来源**: 语雀知识库（kangaroo-container / MicroSandbox），经 aone-km 全文获取
- **原文链接**: https://aliyuque.antfin.com/kangaroo-container/di61r8/og6kobitydlapy8o
- **KM 镜像**: https://km.aone.alibaba-inc.com/#/repo/130429/doc/og6kobitydlapy8o
- **采集时间**: 2026-08-07 21:55

**会议日期：** 2026年5月29日  
**参与团队：** MicroSandbox团队、FC团队  
**会议目标：** 对齐MicroSandbox场景下E2B兼容情况、确认一期Feature交付范围、明确正式上线环境工作项

---

## 议题一：MicroSandbox 场景下 E2B 功能兼容逐项对齐
FC此前已在Rund模式下兼容了E2B接口，当前需确认切换到MicroSandbox后，各功能的支持情况。以下按原始对比表逐条列出，会上逐一check并填写 Microsandbox 状态列和 FC控制面(MicroSandbox) 列。

> **原始对比表：** [FC / E2B 功能兼容情况对比表（钉钉文档）](https://alidocs.dingtalk.com/uni-preview?spaceId=2125320914&extension=md&bizType=text&fileSize=14261&dentryUuid=kDnRL6jAJMLgNkw7tX9Y00DqVyMoPYe1&previewAtta=7&version=1&fileId=223015372171&scene=universalSpace)
>

---

### 一、SDK 兼容情况
#### 1. Sandbox 管理
FC Infra TODO：

+ metrics
    - 节点侧(EE) [@快帆](undefined/yangmanyou.ymy)
    - 系统管控组件 [@泽尘](undefined/zechen.hg)
+ failover（报错是否可重试）
    - 节点侧  [@快帆](undefined/yangmanyou.ymy)   06月05日
    - 系统侧  [@泽尘](undefined/zechen.hg)
+ E2E 测试集
    - infra Funtion   [@泽尘](undefined/zechen.hg)
    - SDK + E2B控制面 + 数据链路 [@西流](undefined/ls147258)
    - 节点侧集成测试 [@快帆](undefined/yangmanyou.ymy)

E2B 全链路接口参数对应能力映射整理： 06月01日

+ microsandbox [@天平](undefined/duanyunpeng.dyp)
+ SDK <-> FC控制面(E2B) [@西流](undefined/ls147258)
+ FC控制面(E2B) <-> FC infra [@西流](undefined/ls147258)
+ FC infra <-> microsandbox [@泽尘](undefined/zechen.hg)

发布 

+ 确认发布地域、节点资源情况  
+ FC infra系统 [@泽尘](undefined/zechen.hg) [@快帆](undefined/yangmanyou.ymy)
+ E2B 网关 [@西流](undefined/ls147258)



一期：6.5日之前

| 功能 | E2B 原生 SDK 方法 | FC&RunD 适配状态 | Microsandbox 状态 [@天平](undefined/duanyunpeng.dyp) | FC控制面(E2B) [@西流](undefined/ls147258) | FC Infra [@泽尘](undefined/zechen.hg) | 说明 |
| --- | --- | :---: | --- | --- | --- | --- |
| 创建沙箱 | `Sandbox.create()`<br/><br/>para:<br/>+ env<br/>+ timeout | ✅ 支持 | ✅ 支持<br/><br/>E2B → microsandbox 参数映射：   • templateID → 控制面查模板 → 组装 ResumeSandboxRequest.template（TemplateConfig: repo, template_id, auth, repo_network_config）<br/>• envVars → ResumeSandboxRequest.envs（map[string,string]，透传给 envd /init）<br/>• secure → ResumeSandboxRequest.access_token（控制面生成 token 传入，envd 校验）• network / allow_internet_access → ResumeSandboxRequest.net（NetworkConfig: net_type[INTERNET/NONE], network_service_id）<br/>• timeout / autoPause → 节点不感知，控制面管理 TTL（<font style="color:#DF2A3F;">目前FC不支持timeout刷新</font>）<br/>控制面需要做： 生成 sandbox_id → 查询模板组装 TemplateConfig → 分配 NetworkConfig → 生成 access_token → 调 Resume → 管理 TTL（<font style="color:#DF2A3F;">FC网关做了，不需要e2b accesstoken</font>)<br/><br/><br/><font style="color:#DF2A3F;">Metadata：FC&runD 支持了E2B不支持的oss/nas用法（一期不支持）</font> | 控制链路完成<br/>问题：数据链路还在调试中<br/><br/>Action：明确e2b功能字段 映射到 FC控制面，以及承接FC infra的能力<br/>```json "fc:ListFunctions" "fc:DeleteFunction" "fc:UpdateFunction" "fc:CreateFunction" "fc:GetFunction" "fc:CreateSession" "fc:GetSession" "fc:UpdateSession" "fc:PauseSession" "fc:ResumeSession" "fc:ListSessions" "fc:DeleteSession" "fc:InvokeFunction" ```  涉及到的函数属性：<br/>```json region: cn-shanghai-cloudspe handler: index.handler diskSize: 10240 runtime: custom-container cpu: 2 instanceConcurrency: 200 vpcConfig:   securityGroupId: sg-hn392rwo83k8k7j1voy1   vpcId: vpc-hn3ik1waxs4ffepuwd80o   vSwitchIds:     - vsw-hn34ndz1yrdyn3vj9w6i8 role: '' description: 'E2B on FC Function: 7ed4br2tnpm6yysba9cy|default' timeout: 3600 sessionAffinityConfig:   affinityHeaderFieldName: fc-affinity-session-id   disableSessionIdReuse: false   enableAutoPause: false   sessionConcurrencyPerInstance: 1   sessionIdleTimeoutInSeconds: 1800   sessionTTLInSeconds: 21600 internetAccess: true instanceIsolationMode: SESSION_EXCLUSIVE functionName: e2b-sandbox-template-64dfc84c9d7c0498 sessionAffinity: HEADER_FIELD customContainerConfig:   image: >-     fc-test-registry-vpc.cn-shanghai-cloudspe.cr.aliyuncs.com/e2b-dev/code-interpreter-v1:v0.0.16   port: 5000 memorySize: 2048 environmentVariables:   PERSISTENT_VAR: persisted_value ```  | CreateFunction/UpdateFunction<br/>+ 核心<br/>    - Runtime: micro-sandbox<br/>    - 触发 Template 模版构建<br/>+ 依赖<br/>    - EEAgent BuildTemplate GRPC<br/>    - EEAgent GetBuildStatus GRPC<br/>+ 限制<br/>    - 必须使用同地域 ACR/ACREE<br/>    - 非 ACR/ACREE 的自定义镜像暂不支持（DockerHub 之类的）<br/><br/>GetFunction<br/>+ Function.Status 依赖 Template 构建状态(参考了加速镜像)<br/>+ Function.StatusReason：待确认是否使用了，现状还没有透出 MicroSandbox 日志，希望将来 E2B 能用上<br/>    - <font style="color:#DF2A3F;">未来希望能对齐 E2B BuildTemplate 拿到实时构建日志（一期不强依赖）</font><br/><br/>CreateSession<br/>+ 已支持<br/><br/>InvokeFunction<br/>+ 已支持<br/><br/>ScalingConfig<br/>+ <font style="color:#DF2A3F;">预留链路不支持（一期不强依赖）</font><br/>+ 当前 E2B 没有提供预留链路<br/><br/><font style="color:#DF2A3F;">待确认：其他 FunctionMeta 字段是否使用到？</font><br/>+ **<font style="background-color:#FBDE28;">LogConfig：不支持，待和 PD 确认产品行为（原生 E2B 支持采集的是 envd 以及通过 run_command 启动的任务）</font>**<br/>    - 现状：<br/>        * Rund 场景 FC 采集的是容器标准输出<br/>        * <font style="color:#DF2A3F;">MicroSandbox 场景不确定采集谁的标准输出</font><br/>    - MicroSandbox 支持现状：<br/>        * 通过 envd 启动的能采集到标准输出<br/>        * 用户镜像的 startCommand 以及通过 MicroSandbox run_command 就是通过 envd 采集的<br/>    - FC E2B 现状：<br/>        * FC run_command 走的是端口转发，直接请求到 MicroSandbox envd<br/>+ **<font style="background-color:#FBDE28;">TracingConfig： 不支持，待和 PD 确认产品行为（原生 E2B 不支持）</font>**<br/>    - 要确认采集哪些进程的 Trace，走的采集方案是什么<br/>+ <font style="color:#DF2A3F;">NASConfig：一期不支持，让用户自己挂载 </font>**<font style="background-color:#FBDE28;">待和 PD 确认产品行为（原生 E2B 不支持）</font>**<br/>+ <font style="color:#DF2A3F;">OSSMountConfig：一期不支持，让用户自 己挂载 </font>**<font style="background-color:#FBDE28;">待和 PD 确认产品行为（原生 E2B 不支持）</font>**<br/>+ <font style="color:#DF2A3F;">PolarFsConfig：一期不支持，让用户自己挂载 </font>**<font style="background-color:#FBDE28;">待和 PD 确认产品行为（原生 E2B 不支持）</font>**<br/>+ <font style="color:#DF2A3F;">CustomDNS：一期不依赖</font><br/>+ <font style="color:#DF2A3F;">InstanceLifecycleConfig：一期不依赖</font><br/>+ Layers：不支持，并且不依赖<br/>+ GPUConfig：不支持，并且不依赖<br/>+ 其他：<br/>    - Code：不支持，并且不依赖<br/>    - CustomRuntimeConfig：不支持，并且不依赖<br/>    - EnableLongLiving：不依赖<br/>    - DisableInjectCredentials：不依赖<br/>    - IdleTimeOutInSeconds：不依赖<br/>    - Tags：不依赖，现在 E2B 走的是 InnerTag<br/>    - ResourceGroupID：不依赖<br/>    - WriteProtection：不依赖<br/>    - DeleteProtection：不依赖<br/><br/>TODO：补充 元数据/GRPC 架构图 | 支持 template / timeout / envVars / metadata。**FC现状：** ttl语义和E2B不一致（FC用session实现，6月中旬对齐）；AccessToken实现有差异，目前没有调到envd；需要动态env、动态NAS挂载（E2B没有） |
| 连接已有沙箱 | `Sandbox.connect(sandboxId)` | ✅ 支持* | ✅ 支持。控制面通过 `Get()` 查状态：running 直接返回 `NetworkProxy` 连接信息；paused 则调 `Resume()` 恢复后返回 |  |  | 已暂停则自动恢复；恢复功能需香港/新加坡地域 |
| 列表沙箱 | `Sandbox.list()` | ✅ 支持 | ✅ 支持。`SandboxService.List()`，支持按 Status 枚举过滤（RUNNING/PAUSED/STOPPED 等），也可以用来做服务发现（e2b是轮训节点实现sandbox信息收集，可通过该接口实现类似效果） |  |  | v1 / v2 |
| 获取沙箱详情 | `Sandbox.getInfo(sandboxId)` | ✅ 支持 | ✅ 支持。`SandboxService.Get()` 返回 `Sandbox` 消息（含 status / template / metadata / envs / net_info），控制面记录的话可以直接返回 |  |  | 静态方法 |
| 终止沙箱 | `Sandbox.kill(sandboxId)` | ✅ 支持 | ✅ 支持。`SandboxService.Delete(sandbox_id)`，同步接口，清理 VM/网络/存储资源 |  |  | 静态方法 |
| 设置超时 | `Sandbox.setTimeout(sandboxId, ms)` | ✅ 支持 | 节点无需支持。timeout / TTL 由控制面管理 |  |  | 静态方法 |
| 实例终止 | `sandbox.kill()` | ✅ 支持 | ✅ 支持。同 `SandboxService.Delete()` |  |  | 实例方法 |
| 实例详情 | `sandbox.getInfo()` | ✅ 支持 | ✅ 支持。同 `SandboxService.Get()` |  |  | 实例方法 |
| 实例超时 | `sandbox.setTimeout(ms)` | ✅ 支持 | 节点无需支持。控制面管理 TTL |  |  | 实例方法 |
| 运行状态 | `sandbox.isRunning()` | ✅ 支持 | ✅ 支持。`Get()` 返回 `Sandbox.status == STATUS_RUNNING` |  |  | 通过 getInfo 判断状态 |
| 暂停沙箱 | `sandbox.pause()` | ✅ 支持* | ✅ 支持。控制面需联合调用 2 个 microsandbox API：   ① SandboxService.Pause(sandbox_id) — 同步接口，暂停 VM（CPU 停止 + 内存快照到本地），状态变为 PAUSED   ② SandboxService.ArchiveAsync(sandbox_id, template_config) — 异步接口，将暂停状态（rootfs + snapfile）上传到 ACR 仓库，立即返回 template_id   ③ 控制面轮询 GetArchiveTask(template_id) 等待 Archive 完成（COMPLETED / FAILED）   ④ Archive 完成后调 SandboxService.Delete(sandbox_id) 清理本地资源<br/><br/>**注意事项：**也可以调用同步SandboxService.Archive，但时间可能较长（几十秒），会长时间阻塞 gRPC，同步接口有 `timeout_seconds` 参数（默认 10 分钟），超时会失败   • 如果 Archive 失败，sandbox 仍处于 PAUSED 状态，可重试 Archive 或 Resume 恢复<br/><br/><br/><font style="color:#DF2A3F;">确认通义是否需要这个功能：不需要</font> | TODO: <br/><font style="color:#DF2A3F;">Pause/Resume如何使用FC得API以及FC的快照管理，是否要提供新的API</font><br/>1:N，当前调用的是什么FC infra接口<br/><br/> | TODO：FC支持快照管理设计，兼容1:N | 暂停功能需香港/新加坡地域。**FC现状：** FC是深休眠方案 |
| 下载 URL | `sandbox.downloadUrl(path)` | ✅ 支持 | ✅ envd 支持<br/>文件下载通过 envd REST `GET /files`，URL 由控制面/客户端结合sandbox路由生成 |  |  | 客户端生成 URL |
| 上传 URL | `sandbox.uploadUrl(path)` | ✅ 支持 | ✅ envd 支持。文件上传通过 envd REST `POST /files`（multipart / raw），URL 同上 |  |  | 客户端生成 URL |
| 获取 Host | `sandbox.getHost(port)` | ✅ 支持 | 节点无需支持。控制面/客户端基于 Resume 返回的Network相关信息生成 |  |  | 客户端生成 |


#### 2. Process / Commands
| 功能 | E2B 原生 SDK 方法 | FC&RunD 适配状态 | Microsandbox 状态 | FC控制面(MicroSandbox) | 说明 |
| --- | --- | :---: | :---: | :---: | --- |
| 执行命令 | `sandbox.commands.run()` | ✅ 支持 | ✅ envd 支持<br/>`Process.Start` RPC（server-streaming），支持 PTY 模式、envs 注入、cwd、tag |  | 含 PTY 模式。**FC现状：** 由于启动速度原因，单独起一个python进程而非jupyter，功能存在缺失 |
| 列表进程 | `sandbox.commands.list()` | ✅ 支持 | ✅ envd 支持。`Process.List` RPC |  | 列出运行中进程 |
| 连接进程 | `sandbox.commands.connect()` | ✅ 支持 | ✅ envd 支持。`Process.Connect` RPC（server-streaming） |  | 连接到已有进程 |
| 发送输入 | `sandbox.commands.sendStdin()` | ✅ 支持 | ✅ envd 支持。`Process.SendInput` |  | 发送标准输入 |
| 终止进程 | `sandbox.commands.kill()` | ✅ 支持 | ✅ envd 支持。`Process.SendSignal` RPC（SIGTERM/SIGKILL） |  | 终止进程 |


6.1日envd链路已经跑通

<font style="color:#DF2A3F;">TODO： 当前envd为0.4.4，需要支持到0.5.2+，一期支持</font>

#### 3. Filesystem
| 功能 | E2B 原生 SDK 方法 | FC&RunD 适配状态 | Microsandbox 状态 | FC控制面(MicroSandbox) | 说明 |
| --- | --- | :---: | :---: | :---: | --- |
| 列出目录 | `sandbox.files.list()` | ✅ 支持 | ✅ envd 支持。`Filesystem.ListDir` RPC |  | 列出目录内容 |
| 文件存在性 | `sandbox.files.exists()` | ✅ 支持 | ✅ envd 支持。`Filesystem.Stat` RPC，NotFound 即不存在 |  | 404 表示不存在 |
| 读取文件 | `sandbox.files.read()` | ✅ 支持 | ✅ envd 支持。REST `GET /files` |  | REST 下载 |
| 写入文件 | `sandbox.files.write()` | ✅ 支持 | ✅ envd 支持。REST `POST /files`（multipart / raw octet-stream） |  | REST 上传 |
| 创建目录 | `sandbox.files.makeDir()` | ✅ 支持 | ✅ envd 支持。`Filesystem.MakeDir` RPC |  | 创建目录 |
| 删除文件/目录 | `sandbox.files.remove()` | ✅ 支持 | ✅ envd 支持。`Filesystem.Remove` RPC |  | 删除文件/目录 |
| 移动/重命名 | `sandbox.files.rename()` | ✅ 支持 | ✅ envd 支持。`Filesystem.Move` RPC |  | 移动/重命名 |
| 文件监听 | `sandbox.files.watch()` | ✅ 支持 | ✅ envd 支持。`Filesystem.WatchDir`（server-streaming）/ `CreateWatcher` + `GetWatcherEvents`（轮询） |  | 文件/目录变更监听 |


#### 4. Code Interpreter
| 功能 | E2B 原生 SDK 方法 | FC&RunD 适配状态 | Microsandbox 状态 | FC控制面(MicroSandbox) | 说明 |
| --- | --- | :---: | :---: | :---: | --- |
| 创建解释器 | `CodeInterpreter.create()` | ✅ 支持 | ✅ 支持。通过 `Resume()` 启动 code-interpreter 模板，Code Interpreter 能力由**模板内预装组件**（Jupyter kernel）提供 | <font style="color:#DF2A3F;">TODO：6.1 code interpreter的镜像正在build中，6.1同步结论</font> | 使用 code-interpreter 模板 |
| 执行代码 | `sandbox.runCode()` | ✅ 支持 | ✅ envd 底层支持 |  | Python + JavaScript |
| 创建上下文 | `sandbox.createContext()` | ✅ 支持 | ✅ 模板内 Code Interpreter 组件实现，非 microsandbox 节点原生 API |  | 创建执行上下文 |
| 上下文生命周期 | Context 创建 / 查询 / 删除 | ✅ 支持 | ✅ 同上，模板内组件实现 |  | /contexts 全生命周期 |
| ~~context_id 兼容~~ | `~~context_id~~`~~ 字段~~ | ~~✅~~~~ 支持~~ | ~~✅~~~~ 同上，模板内组件实现~~ | ~~~~ | ~~snake_case + camelCase 双字段兼容~~ |
| 时间戳 | stdout / stderr timestamp | ✅ 支持 | ✅ envd 支持。`LogService.StreamLogs` 提供纳秒时间戳 |  | NDJSON 纳秒时间戳 |
| 执行计数 | execution count | ✅ 支持 | ✅ 模板内 Code Interpreter 组件实现 |  | number_of_executions.execution_count |
| 文本输出 | `execution.text` | ✅ 支持 | ✅ 模板内 Code Interpreter 组件实现 |  | 裸表达式进入 text，print 进入 stdout |
| 状态持久化 | — | ✅ 支持 | ✅ 支持 |  | 多次 run_code 之间变量保持 |


6.1 结论：

+ OSWorld镜像（ubuntu、Windows、android），在build流程e2e跑通后，后续由FC官方提供对应模版

#### 5. Template
| 功能 | E2B 原生 SDK 方法 | FC&RunD 适配状态 | Microsandbox 状态 | FC控制面(MicroSandbox) | 说明 |
| --- | --- | :---: | :---: | :---: | --- |
| 模板 CRUD | Template CRUD | ✅ 支持 | ✅ 支持。`TemplateService.Build()` / `Delete()` / `Get()` |  | v1 / v2 / v3 全版本 |
| 模板构建 | Template Build | ⚠️ 部分支持 | ✅ 支持。`TemplateService.Build(TemplateConfig)`，支持 TemplateStep（USER/WORKDIR/ENV/RUN/COPY），`GetStatus()` 查进度与日志。构建进度上报增强 🔶 实现未合入 | 6.1结论：<font style="color:#DF2A3F;">一期只支持 FromImage build，不使用step自定义命令</font> | 触发构建、获取状态与日志；目前支持 template.FromImage（ACREE 镜像）。**FC现状：** 目前还不支持通过E2B SDK做template build，提供了单独工具，六月初支持 |
| 模板标签 | Template Tags | ✅ 支持 | 节点无需支持，控制面管理 |  | 标签分配与删除 |
| 模板别名 | Template Alias | ✅ 支持 | 节点无需支持，控制面管理 |  | 通过别名获取模板 |


#### 6. CLI
| 功能 | E2B 原生能力 | FC&RunD 适配状态 | Microsandbox 状态 | FC控制面(MicroSandbox) | 说明 |
| --- | --- | :---: | :---: | :---: | --- |
| Sandbox CLI | sandbox create / list / kill / logs / metrics | ✅ 支持 | 节点无需支持，控制面管理 |  | E2B CLI 核心命令。**FC现状：** E2B鉴权在FC控制面已实现 |
| Template CLI | template list | ✅ 支持 | 节点无需支持，控制面管理 |  | 模板列表查询 |


#### 7. Git
| 功能 | E2B 原生 SDK 方法 | FC&RunD 适配状态 | Microsandbox 状态 | FC控制面(MicroSandbox) | 说明 |
| --- | --- | :---: | :---: | :---: | --- |
| Git 操作 | `git.*` | ✅ 支持 | ✅ envd 支持。通过 `Process.Start` 执行 git 命令，无独立 Git API |  | SDK 封装为 commands 执行 git 命令 |


#### 8. SDK 受限兼容项
| 功能 | E2B 原生 SDK 方法 | FC&RunD 适配状态 | Microsandbox 状态 | FC控制面(MicroSandbox) | 说明 |
| --- | --- | :---: | :---: | :---: | --- |
| 沙箱指标 | Sandbox Metrics（批量/单沙箱） | ⏳ Stub 实现 | ✅ 支持。`StatService.GetSandboxStat()`（CPU/内存请求配置）+ envd REST `GET /metrics`（CPU/内存/磁盘实时用量） |  | 返回模拟 CPU / Mem / Disk 数据或空数组。**FC现状：** 目前还没实现 |
| 沙箱日志 | Sandbox Logs（v1 / v2） | ⏳ Stub 实现 | ✅ 支持。envd `LogService.StreamLogs`（server-streaming），支持 envd进程 stdout/stderr/lifecycle |  | 返回空数组；后续支持 |
| 快照 | Snapshots（创建/列表） | ❌ 后续支持 | ✅ 支持。通过 `Pause` + `Archive`/`ArchiveAsync` 实现（等价于 E2B snapshot），创建后可通过 `Resume` 恢复 |  | 暂不支持 |
| 网络配置 | Network Config Update | ⚠️ Meta 方式 | ❌ 后续支持 | 6.1结论：Allow/Deny不支持，目前支持VPC扩展，allowinternet不不支持<br/>一期不支持以上功能 | 返回 204，无实际变更 |
| Volume 管理 | `Volume.create / list / delete` | ❌ 后续支持 | ❌ 暂不支持 |  | E2B Volume 暂不支持；meta 方式提供挂载 |
| Volume 文件操作 | `volume.readFile / writeFile / listDir / stat` | ❌ 后续支持 | ❌ 暂不支持 |  | Volume 独立文件操作不支持 |
| Volume 挂载 | `Sandbox.create({ volumeMounts })` | ❌ 后续支持 | ❌ 暂不支持 |  | volumeMounts 字段被忽略；meta 方式挂载 |
| API Key 管理 | API Key CRUD | ✅ 支持 | 节点无需支持，控制面管理 |  | FC已实现 |
| Access Token 管理 | Access Token CRUD | ⚠️ RAM 支持 | ✅ 支持透传。`ResumeSandboxRequest.access_token` 传递给 envd，envd 校验 `X-Access-Token` header |  | 使用静态配置；需与 RAM 打通 |


---

### 二、API 兼容情况
#### 1. 控制面 API — Sandbox 管理
| 功能 | E2B 原生 API | FC&RunD 适配状态 | Microsandbox 状态 | FC控制面(MicroSandbox) | 说明 |
| --- | --- | :---: | :---: | :---: | --- |
| 创建沙箱 | `POST /sandboxes` | ✅ 支持 | ✅ 支持。`SandboxService.Resume()` |  | 完整实现 |
| 列表运行中沙箱 | `GET /sandboxes` | ✅ 支持 | ✅ 支持。`SandboxService.List(status=RUNNING)` |  | 完整实现 |
| 列表所有沙箱（含暂停） | `GET /v2/sandboxes` | ✅ 支持 | ✅ 支持。`SandboxService.List()`，汇总和分页由控制面处理 |  | 支持分页 |
| 获取沙箱详情 | `GET /sandboxes/{sandboxID}` | ✅ 支持 | ✅ 支持。`SandboxService.Get()` |  | 完整实现 |
| 删除/终止沙箱 | `DELETE /sandboxes/{sandboxID}` | ✅ 支持 | ✅ 支持。`SandboxService.Delete()` |  | 完整实现 |
| 暂停沙箱 | `POST /sandboxes/{sandboxID}/pause` | ✅ 支持 | ✅ 支持。控制面调 `Pause()`（同步）→ `ArchiveAsync()`（异步）→ 轮询 `GetArchiveTask()` → `Delete()` |  | 香港/新加坡 |
| 恢复沙箱 | `POST /sandboxes/{sandboxID}/resume` | ✅ 支持 | ✅ 支持。`SandboxService.Resume()`（从 archived 模板恢复） |  | 香港/新加坡 |
| 连接沙箱 | `POST /sandboxes/{sandboxID}/connect` | ✅ 支持 | ✅ 支持。控制面 `Get()` 查状态 + 条件性 `Resume()` |  | 完整实现 |
| 设置超时 | `POST /sandboxes/{sandboxID}/timeout` | ✅ 支持 | 节点无需支持，控制面管理 TTL |  | 完整实现 |
| 刷新沙箱 | `POST /sandboxes/{sandboxID}/refreshes` | ✅ 支持 | 节点无需支持，控制面管理 |  | 完整实现 |
| 批量指标 | `GET /sandboxes/metrics` | ⏳ Stub | ✅ 支持。控制面聚合各节点 `StatService.GetSandboxStat()` |  | 返回空数组 |
| 单沙箱指标 | `GET /sandboxes/{sandboxID}/metrics` | ⏳ Stub | ✅ 支持。`StatService.GetSandboxStat()` + envd `/metrics` |  | 返回模拟数据 |
| v1 日志 | `GET /sandboxes/{sandboxID}/logs` | ⏳ Stub | ⚠️ 节点侧采集已支持（envd `StreamLogs` → daemon 本地文件）。对齐 E2B REST 查询 API 需控制面补齐日志持久化 + 查询层（E2B 链路：envd → Hyperloop → Vector → Loki → API 查 Loki） |  | 返回空数组 |
| v2 日志 | `GET /v2/sandboxes/{sandboxID}/logs` | ⏳ Stub | ⚠️ 同上 |  | 返回空数组 |
| 网络配置更新 | `PUT /sandboxes/{sandboxID}/network` | ⚠️ Meta 方式 | ⚠️ 支持QoS，进出口黑白名单暂不支持 |  | 返回 204，无实际变更 |
| 快照 | `POST /sandboxes/{sandboxID}/snapshots` / `GET /snapshots` | ❌ 暂不支持 | ✅ 支持。等价于 `Pause` + `Archive`（深休眠快照） |  | — |


#### 2. 控制面 API — Template 管理
| 功能 | E2B 原生 API | FC&RunD 适配状态 | Microsandbox 状态 | FC控制面(MicroSandbox) | 说明 |
| --- | --- | :---: | :---: | :---: | --- |
| 创建模板 | `POST /templates` / `/v2/templates` / `/v3/templates` | ✅ 支持 | ✅ 支持。`TemplateService.Build(TemplateConfig)` |  | 完整实现 |
| 列表模板 | `GET /templates` | ✅ 支持 | 节点无需支持，控制面管理模板元数据 |  | 完整实现 |
| 获取模板详情 | `GET /templates/{templateID}` | ✅ 支持 | ✅ 支持。`TemplateService.Get(template_id)` 返回 `TemplateInfo` |  | 完整实现 |
| 更新模板 | `PATCH /templates/{templateID}` / `/v2/...` | ✅ 支持 | 节点无需支持，控制面管理 |  | 完整实现 |
| 更新模板（POST 兼容） | `POST /templates/{templateID}` | ✅ 支持 | ✅ 支持。`TemplateService.Delete(template_id)` （只是删除node数据，acr存储数据没有删除） |  | 完整实现 |
| 删除模板 | `DELETE /templates/{templateID}` | ✅ 支持 | ✅ 支持。`TemplateService.Delete(template_id)` |  | 完整实现 |
| 别名获取模板 | `GET /templates/aliases/{alias}` | ✅ 支持 | 节点无需支持，控制面管理别名映射 |  | 完整实现 |
| 列表标签 | `GET /templates/{templateID}/tags` | ✅ 支持 | 节点无需支持，控制面管理 |  | 完整实现 |
| 分配标签 | `POST /templates/tags` | ✅ 支持 | 节点无需支持，控制面管理 |  | 完整实现 |
| 删除标签 | `DELETE /templates/tags` | ✅ 支持 | 节点无需支持，控制面管理 |  | 完整实现 |
| 触发构建 | `POST /templates/{templateID}/builds/{buildID}` / `/v2/...` | ⚠️ 部分支持 | ✅ 支持。`TemplateService.Build()` |  | 语义有差异 |
| 构建状态 | `GET /templates/{templateID}/builds/{buildID}/status` | ✅ 支持 | ✅ 支持。`TemplateService.GetStatus(template_id, build_id)`，含 metadata/log_entries/reason |  | 完整实现 |
| 构建日志 | `GET /templates/{templateID}/builds/{buildID}/logs` | ✅ 支持 | ✅ 支持。`GetStatus()` 返回 `log_entries`（TemplateBuildLogEntry），支持分页/级别过滤/方向 |  | 完整实现 |


#### 3. 控制面 API — 不支持的 API
| 功能 | E2B 原生 API | FC&RunD 适配状态 | Microsandbox 状态 | FC控制面(MicroSandbox) | 说明 / 替代方案 |
| --- | --- | :---: | :---: | :---: | --- |
| Volume 管理 | `GET/POST /volumes`、`GET/DELETE /volumes/{volumeID}` | ❌ 不支持 | ❌ 暂不支持。 |  | 可通过 meta 方式使用 OSS / JuiceFS |
| API Key 管理 | `GET/POST /api-keys`、`PATCH/DELETE /api-keys/{apiKeyID}` | ❌ 不支持 | 节点无需支持，控制面管理 |  | 当前通过静态 API Key + RAM 方式支持 |
| Access Token 管理 | `GET/POST /access-tokens`、`PATCH/DELETE /access-tokens/{accessTokenID}` | ❌ 不支持 | ✅ 支持透传。Resume 时通过 `access_token` 传递给 envd |  | 目前通过另外方式支持 |
| Team 管理 | `GET /teams`、`GET /teams/{teamID}/metrics` | ❌ 不支持 | 节点无需支持，控制面管理 |  | — |


#### 4. 数据面 API — 进程管理（Connect-RPC）
| 功能 | E2B 原生 API | FC&RunD 适配状态 | Microsandbox 状态 | FC控制面(MicroSandbox) | 说明 |
| --- | --- | :---: | :---: | :---: | --- |
| 启动进程 | `POST /process.Process/Start` | ✅ 支持 | ✅ envd 支持 |  | 支持 PTY + 环境变量注入 |
| 列出进程 | `POST /process.Process/List` | ✅ 支持 | ✅ envd 支持 |  | 列出运行中进程 |
| 连接进程 | `POST /process.Process/Connect` | ✅ 支持 | ✅ envd 支持 |  | 连接到已有进程 |
| 发送输入 | `POST /process.Process/SendInput` | ✅ 支持 | ✅ envd 支持 |  | 发送标准输入 |
| 发送信号 | `POST /process.Process/SendSignal` | ✅ 支持 | ✅ envd 支持 |  | SIGTERM / SIGKILL 等 |
| 更新进程配置 | `POST /process.Process/Update` | ✅ 支持 | ✅ envd 支持 |  | PTY 窗口大小调整 |
| 终止进程 | `POST /process.Process/Kill` | ✅ 支持 | ✅ envd 支持 |  | 终止进程 |


#### 5. 数据面 API — 文件系统（Connect-RPC + REST）
| 功能 | E2B 原生 API | FC&RunD 适配状态 | Microsandbox 状态 | FC控制面(MicroSandbox) | 说明 |
| --- | --- | :---: | :---: | :---: | --- |
| 列出目录 | `POST /filesystem.Filesystem/ListDir` | ✅ 支持 | ✅ envd 支持 |  | — |
| 获取文件信息 | `POST /filesystem.Filesystem/Stat` | ✅ 支持 | ✅ envd 支持 |  | — |
| 创建目录 | `POST /filesystem.Filesystem/MakeDir` | ✅ 支持 | ✅ envd 支持 |  | — |
| 删除文件/目录 | `POST /filesystem.Filesystem/Remove` | ✅ 支持 | ✅ envd 支持 |  | — |
| 移动/重命名 | `POST /filesystem.Filesystem/Move` | ✅ 支持 | ✅ envd 支持 |  | — |
| 创建监听器 | `POST /filesystem.Filesystem/CreateWatcher` | ✅ 支持 | ✅ envd 支持 |  | 文件/目录监听 |
| 文件下载 | `GET /files` | ✅ 支持 | ✅ envd 支持 |  | REST 下载 |
| 文件上传 | `POST /files` | ✅ 支持 | ✅ envd 支持 |  | REST 上传 |


#### 6. 数据面 API — Code Interpreter（REST）
| 功能 | E2B 原生 API | FC&RunD 适配状态 | Microsandbox 状态 | FC控制面(MicroSandbox) | 说明 |
| --- | --- | :---: | :---: | :---: | --- |
| 创建上下文 | `POST /sandboxes/{sandboxId}/contexts` | ✅ 支持 | ✅ 模板内 Code Interpreter 组件实现，envd 提供底层 Process/Filesystem 支持 |  | — |
| 列出上下文 | `GET /sandboxes/{sandboxId}/contexts` | ✅ 支持 | ✅ 同上 |  | — |
| 重启上下文 | `POST /sandboxes/{sandboxId}/contexts/{contextId}/restart` | ✅ 支持 | ✅ 同上 |  | — |
| 删除上下文 | `DELETE /sandboxes/{sandboxId}/contexts/{contextId}` | ✅ 支持 | ✅ 同上 |  | — |
| 同步执行代码 | `POST /sandboxes/{sandboxId}/execute` | ✅ 支持 | ✅ 同上。另有 `SandboxService.Exec()` RPC 可直接在 VM 内执行命令（非 Code Interpreter） |  | — |


#### 7. 数据面 API — 健康检查（REST）
| 功能 | E2B 原生 API | FC&RunD 适配状态 | Microsandbox 状态 | FC控制面(MicroSandbox) | 说明 |
| --- | --- | :---: | :---: | :---: | --- |
| 健康检查 | `GET /health` | ✅ 支持 | ✅ envd 支持。`GET /health` 返回 204 No Content |  | envd 健康检查 |
| 服务就绪检查 | `GET /sandboxes/{sandboxId}/health/envd` | ✅ 支持 | ✅ envd 支持。同上，控制面访问 envd `/health` |  | envd 服务就绪检查 |


---

### 三、认证方式
| 认证方式 | E2B 原生 Header | FC&RunD 适配状态 | Microsandbox 状态 | FC控制面(MicroSandbox) |
| --- | --- | :---: | :---: | :---: |
| API Key | `X-API-Key` | ✅ 支持 | 节点无需支持 |  |


---

### 四、汇总
| 维度 | E2B 原生能力 | FC&RunD 适配状态 | Microsandbox 状态 | FC控制面(MicroSandbox) |
| --- | --- | --- | --- | :---: |
| Python SDK 核心功能 | Sandbox、Files、Commands、PTY、Code Interpreter、Template、CLI | ✅ 完全兼容 | ✅ 节点侧 + envd 数据面完整支持 |  |
| Node.js SDK 核心功能 | 同上 | ✅ 完全兼容 | ✅ 同上 |  |
| 控制面 API | 核心功能 + 可观测/团队/API Key 管理等 | 核心完整；可观测/团队/APIKey暂不支持（预计5.30） | 核心 RPC 完整（Resume/Delete/Pause/Archive/Get/List/Fork/Exec）；timeout/refresh/tags/alias 由控制面管理 |  |
| 数据面 API | 进程 / 文件系统 / Code Interpreter / 健康检查（共 21 个端点） | ✅ 21/21 全部完整实现 | ✅ envd 完整实现 |  |
| Volume API | Volume 独立管理、挂载、文件操作 | ❌ 暂不支持；metadata扩展替代 | ❌ 暂不支持 |  |
| Snapshot API | 快照创建 / 列表 | ❌ 暂不支持 | ✅ 通过 Pause + Archive 实现深休眠快照 |  |
| API Key / Access Token | 独立端点 CRUD | ❌ 暂不支持；静态APIKey+RAM（预计5.30 RAM SSC） | API Key 节点无需支持；Access Token 透传给 envd |  |
| Team 管理 | Team 信息 / 指标 | ❌ 暂不支持（预计5.30） | 节点无需支持 |  |


---

### 需重点讨论的差异点
+ Template Build：RunD模式支持FromImage(ACR镜像)，不支持Docker build（E2B原生支持），MicroSandbox需对齐
+ Volume：当前通过metadata扩展方式（NAS/OSS/JuiceFS）支持，后续需原生支持E2B Volume接口
+ 可观测（Metrics/Logs）：RunD模式为Stub实现，530需实际对接云监控
+ pause/resume/connect(自动恢复)：RunD限香港/新加坡地域，MicroSandbox是否有地域限制
+ 需要确认envd的透出版本（当前microsandbox支持的是4.4 envd)

**本议题结论：**（会议中填写）

---

## 议题二：FC on MicroSandbox 正式版 Feature 需求与排期确认
基于客户需求和E2B兼容现状，梳理MicroSandbox层面的完整需求列表，确认各项必须在哪个时间点交付。

### MicroSandbox 近期已支持的特性（5-14 ~ 5-29）
以下为 MicroSandbox 在 2026-05-14 至 2026-05-29 期间已交付或待合入的主要功能特性，供会上对照一期需求确认覆盖情况。

#### 已合入主线（✅）
| 特性 | 说明 | 关联一期需求 |
| --- | --- | --- |
| **模板构建步骤 (Template Build Steps)** | 支持 USER/WORKDIR/ENV/RUN/COPY 声明式构建操作，类似 Dockerfile | E2B build能力对齐 |
| **CPU 模板文件 (cpu_template_file)** | 支持跨机型/代际宿主机 build/resume，指定 cpu_template_file 解决兼容性 | 深休眠跨机恢复 |
| **Free Page Reporting (FPR)** | 基于 virtio-balloon 自动释放空闲内存页，实现内存超卖 | 对齐RunD：内存超卖 |
| **AMX 指令集支持** | 模板构建时开启 AMX，为 AI/ML 负载提供硬件加速 | 对齐RunD：智谱RL场景 |
| **Sandbox Metadata** | Sandbox 支持用户自定义元数据，创建/恢复时传入，GetSandbox 回显，Fork 深拷贝 |  |
| **Sandbox Envs 端到端传递** | 环境变量在 Create/Resume/Fork 流程中持久化，GetSandbox 响应新增 envs 字段 | 环境变量/动态注入 |
| **repo_network_config** | Template metadata 新增 VPC 网络通道配置字段 | 沙箱网络访问 |
| **Sandbox 网络信息暴露** | GetSandbox 新增 network_service_id、net_type，CLI 支持 -o text 输出 | 四层通信 |
| **VMM 异常退出自动停止** | VMM 进程异常退出时 sandbox 自动转 Stopped 并清理资源 | 实例生命周期 |
| **TemplateBuild 错误码** | 新增结构化错误码 3200-3260，精确定位构建失败阶段 | Template Build |
| **模板缓存服务 (TemplateCacheService)** | 独立 gRPC 服务，支持 Prepare/ListCached/Evict 模板设备缓存 | 冷启动性能 |
| **镜像摘要校验 (Image Digest)** | 构建时校验镜像 digest，确保完整性 | Template Build 安全 |
| **Dragonball 优雅关闭** | VMM 支持优雅 shutdown，等待 guest 清理后再退出 | 实例生命周期/PreStop |
| **构建资源清理与恢复** | 构建任务删除/daemon异常重启后自动清理残留资源（VMM进程、网络、DADI volumes） | 稳定性 |
| **Hypervisor 版本校验** | 构建时校验 hypervisor_version，无效版本提前报错 | Template Build |
| **Dragonball 日志采集与指标上报** | VMM 运行时日志持久化 + Daemon TPS/延迟/在途请求 metrics | 日志监控/实例指标 |
| **API 模块独立化** | Proto 定义独立为 Go module + Client SDK，方便上游集成 | FC控制面对接 |
| **全组件版本管理** | CLI/Daemon/envd/template-build 均支持 --version，gRPC GetVersion API | 运维 |
| **Rootfs 挂载优化** | 启用 virtio-blk discard，提升存储回收效率 | 性能 |
| **QoS 网络能力 (Local后端)** | microsandbox 侧已实现 Network.Qos 接口，Local 后端可用 | 网络QoS |


#### API 已定义，实现待合入（🔶）
| 特性 | 说明 | 关联一期需求 |
| --- | --- | --- |
| **构建进度上报 (Build Progress Reporting)** | GetTemplateStatus 增强：支持构建日志/元数据/失败原因查询，结构化 log_entries | Template Build / 日志监控 |
| **CPU & Memory 弹性伸缩 (Resize)** | Build 时声明 max_cpu/max_mem_mb，Resume 时动态调整 vcpu_count/memory_mb | cpu/memory变配 |
| **QoS ANSM 后端对接** | Network.Qos 在 ANSM 网络模式下尚未对接传递 | 网络QoS |


#### 待合入分支（Feature Branch）
| 特性 | 分支 | 说明 | 关联一期需求 |
| --- | --- | --- | --- |
| **SandboxEventService（生命周期事件透传）** | `feature/event-pass-through-phase0` | server-streaming RPC SubscribeSandboxEvents，支持三类事件：STATE_CHANGED / VMM_PROCESS_EXITED / SANDBOX_DELETED | 实例生命周期状态实时更新 |


#### 开发中（🚧）
| 特性 | 说明 | 关联一期需求 |
| --- | --- | --- |
| **Android OSWorld Sandbox** | Android 沙箱环境，支持 GPU 渲染加速 | osworld / 蔚来、通义 |
| **Windows OSWorld Sandbox** | Windows 沙箱环境，GUI 能力支持 | osworld / 智谱RL、通义 |


---

### 2.1 P0 需求 — 排期确认
| 需求大类 | 需求描述 | 需求来源 | MicroSandbox进展 | 目标时间点 | 备注 |
| --- | --- | --- | --- | :---: | --- |
| 实例生命周期 | 沙箱实例状态实时更新（容器退出） | 兼容存量 |  |  |  |
| 实例轮转 | OSS凭证轮转（SLS本地更新+OSS注入容器） | 兼容存量 |  |  |  |
| 日志监控 | 客户业务侧日志采集 | 兼容存量 |  |  |  |
| 日志监控 | 实例指标采集 | 兼容存量 |  |  |  |
| 动态挂载 | JuiceFS/NAS/OSS/PolarLakebase（Pod挂载） | 智谱RL |  |  |  |
| 动态挂载 | JuiceFS config泄漏防护 | 智谱RL |  |  |  |
| 动态挂载 | 动态更新挂载点（mount/unmount） | 兼容+智谱RL |  |  |  |
| 深休眠/Fork | 休眠唤醒后NAS/OSS超时重连，与Rund性能一致 | 兼容存量 |  |  |  |
| 深休眠/Fork | 深休眠后跨机、跨版本恢复 | 兼容存量 |  |  |  |
| 深休眠/Fork | 1:N Fork（文件系统+内存） | 智谱RL |  |  |  |
| 沙箱访问 | 四层通信（内网） | 智谱RL |  |  | VNC/数据库等非HTTP协议 |
| 沙箱访问 | exec执行代码 | 兼容存量 |  |  | E2B已支持 |
| osworld | Android（GPU渲染加速） | 蔚来 |  |  |  |
| osworld | ubuntu(主)、windows、Android + IP端口访问 | 通义 |  |  |  |
| 热迁移 | 类比ECS热迁移，支持FC longrunning形态 | 新需求 |  |  |  |
| Template Build | Docker build对齐E2B原生行为 | 兼容存量 |  |  |  |
| 冷启动性能 | 单实例冷启动百毫秒，与Rund一致 | 兼容性能 |  |  |  |
| 冷启动性能 | 5000并发3G异构镜像+JuiceFS冷启动与Rund一致 | 智谱RL |  |  |  |


### 2.2 P1 需求 — 排期确认
| 需求大类 | 需求描述 | 需求来源 | MicroSandbox进展 | 目标时间点 | 备注 |
| --- | --- | --- | --- | :---: | --- |
| 实例生命周期 | PreStop钩子 | 兼容存量 |  |  |  |
| 健康检查 | liveness/readiness | 兼容存量 |  |  |  |
| 环境变量 | 热实例动态注入环境变量 | 新需求 |  |  | 单模板多实例+WS地址注入 |
| 日志监控 | 浅休眠idle CPU阈值判断 | 兼容存量 |  |  |  |
| 动态挂载 | JuiceFS节点侧挂载 | 智谱RL |  |  | 商业版客户端10万限制 |
| DinD | 安全受限DinD执行环境 | 智谱RL |  |  | 最小化权限+强隔离 |
| osworld | windows/ubuntu/macos | 智谱RL |  |  | GUI能力训练 |
| 安全防护 | 出口流量白/黑名单控制 | 兼容+新需求 |  |  |  |


### 2.3 产品化上线策略确认
| 阶段 | 策略 | 目标 |
| --- | --- | --- |
| 阶段1：试运行 | 加白提供给客户使用（任意场景） | 验证功能齐全度+稳定性 |
| 阶段2：并行服务 | RL场景默认推荐 + Agent/Toolcall加白 | MicroSandbox+Rund并行 |
| 阶段3：全量替换 | 所有场景全量使用MicroSandbox | 功能对齐后全量切换 |


**本议题结论：**（会议中填写）

---

## 会议 Action Items 汇总




---
> 采集方式：`aone-km::fetchExternalContentByUrl` 全文读取。
