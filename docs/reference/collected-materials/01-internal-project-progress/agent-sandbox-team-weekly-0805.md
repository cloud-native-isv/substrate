# Agent Sandbox 8.05周会

- **来源**: 钉钉文档（云盘搜索命中，全文读取）
- **原文链接**: https://alidocs.dingtalk.com/i/nodes/KGZLxjv9VGkoG9YwH6aBOn9BV6EDybno?utm_scene=team_space
- **采集时间**: 2026-08-07 22:00

### **【规模】全面落地 runD MicroSandbox 架构，支撑三大核心场景并完成客户切换，全年达到**<span style="color: rgb(193, 0, 2);">**500w核(待对齐)**</span> @杨嵘(欧一)@段云鹏(天平)

以 FC Sandbox 的 RL/Agent Harness/Tool Use 场景为核心，推动全量新客户接入，并完成存量用户迁移, 新增客户中优先完成通义评测/RL场景接入FC Sandbox

<span style="background-color: #F9DDB2;">KR1 ：</span>支持FC Sandbox新增客户(优先通义评测/RL场景 支持osworld/androidworld)全量接入 MicroSandbox，新增客户上线率 100%，在财年结束前，实现 100% 存量 FC Sandbox 用户切换至 runD MicroSandbox 架构 @杨嵘(欧一)@段云鹏(天平)

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：
- 830 动态挂载 microsandbox技术方案已确认，使用fc替用户在guest内直接挂载的方案，下一步需要确认存储客户端如何内置，fc侧正在整理完整方案，时间无法确定，815存在风险，延期至月底

![image.png](https://alidocs2.oss-cn-zhangjiakou.aliyuncs.com/res/8K4nyeLKEZ1R5nLb/img/989f5b6c-6bcb-494f-8475-8607c0087d8a.png?Expires=1786118390&OSSAccessKeyId=LTAI5tKTjg4Kq1HCdBJ8qpSp&Signature=nennSQ6KYcZB3QoohkoWTOT4PgQ%3D "")
- 810 深休眠/snapshot 对接，microsandbox checkpoint开发已完成，fc对接中
- 网络 出口流量白/黑名单控制，预计发布延期
- 网络 L4 打通  8.3 成都发布已延期，暂未发布

![image.png](https://alidocs2.oss-cn-zhangjiakou.aliyuncs.com/res/2M9qP5kJPJGVDO01/img/74b0f69a-9f25-4d7f-a5b8-de594e3bb07f.png?Expires=1786118390&OSSAccessKeyId=LTAI5tKTjg4Kq1HCdBJ8qpSp&Signature=HC7rpFD5tbkrGaVhSmSqCao9r5c%3D "")
- 网络 证书替换 
    - 新增一个自定义字段的需求，已开发完成，待联调
- fc计费方案讨论：明确了当前实际的存储内容，待fc pd计费模式讨论确认后，再确定研发需求。可能存在两个分支：
    - 如果需要guest实际用量，当前采用envd metrics的方式存在被用户hack的风险，若后续依然需要guest指标，需要考虑下沉方案
    - 如果需要实际存储用量，多次深休眠后实际存储可能过大，后续需要实现部分镜像层合并的能力
- 与fc产品 研发对齐e2b架构中envd的角色地位，结论：
    - 由于替换用户自定义envd影响过大，暂<span style="color: rgb(20, 20, 20);">不开放产品化</span>
    - 一定要替换的场景，用户自行承担不兼容后果
- fc microsandbox 透传用户标签已沟通完需求，待fc提供account id标签
- agenticfs on rund 完成适配，考虑 agenticfs 后续可能频繁更新，rund rootfs 不做修改，由 FC 侧追加到 rootfs 中。
- amd perf 能力适配完成，已合入 rund/microsandbox 分支，目前 FC AMD 裸金属适配回归遇到的问题都已解决，还差 CPU template 新需求未完成。
- sysak host 指标采集组件部署验证中，预计今日可合入







<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>
- <span style="color: #f33e3e;">~~悟空~~</span><span style="color: #f33e3e;">千问办公e2b microsandbox 7.20已上海上线，最近几天每天会跑2w\+ sandbox</span>
- <span style="color: #f33e3e;">qoder还在讨论兼容fc microsandbox的统一镜像模式，qoder希望一个镜像在acs sandbox/agentRun/FC runD/FC microsandbox 下都可用</span>
- <span style="color: #f33e3e;">蔚来强要求替换自定义envd，pd建议先验证是否可以由用户自定义替换，否则建议走rund sandbox</span>





<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
- 动态挂载 FC存储客户端内置方案 未出，microsandbox侧方案未定，810 开发完成存在风险
- <span style="color: ;">~~L7 rules~~</span> <span style="color: #252527; background-color: #FFFFFF;">~~ansm 开发完成，在跟fc同学rund端到端联调, microsandbox已经开始~~</span>
- <span style="color: #E03E3E; background-color: #FFFFFF;">dadi镜像转换工具已延期两周，预计下周提供（resize工具本周提供了测试版），本周同时讨论了其他镜像转换方案，由fc控制面通过现有fc转换模式完成加速镜像转换，绕过节点侧转换，待进一步明确方案</span> 



<span style="background-color: #F9DDB2;">KR2:</span> 同舟平台完成 Agent Sandbox 架构升级，实现整机快照恢复能力，新增用户 100% 基于新架构上线，预期全年达到100w\+核（明确数量或比例） @郭豪(昊石)

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：
1. pod vm 复用二期（带热插拔）：预计830上线，上线后会开始承接agent流量
2. 深休眠/浅休眠：管控阻塞 --\> delay podvm复用之后上线。
3. 规模：
    1. vcpu峰值数量变化 17w（6.25日） --\> 22w(6.30日)，灰度仍处于暂停状态，目前峰值没有变化。

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：



<span style="background-color: #F9DDB2;">KR3:</span> ack rund 比优于开源kata\+firecracker和gvisor要好，交付3\+个标杆客户 @郭蔚(萧封)

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：

 与ACK联合进行falco的测试：   

\- ack同学负责搭环境和确认分析E2E结果.  
        - rund负责加上falcon测试跑起来 （得遇在帮忙先探索）
- 目前还在确定集群配置和规模。

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
- gvisor在启动和密度的竞对  

\[ACK\] 支持cri链路多容器pod的整机休眠唤醒save/restore 的方案



<span style="background-color: #F9DDB2;">KR4:</span> 集团支持aone agent沙箱弹性扩缩容、电商ODPS混池能力突破，业务规模增长xx% @文悦力(悦霄)

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：
- aone 规模 9w core、220w 核时
- Agent platform 在 shanghai\_core 集群继续做验收，项目 DDL 提前至 810 交付
- 研发事项
    - 支持 data-root 方案的 dind 能力上线支持 bench
    - 大规模压测中发现并发拉 pod 时 host 上 sys 冲高，挂掉了一些机器，正在继续定位中，目前看可能和频繁创建 pod 时操作 cgroup 产生 uevent，唤醒 udev 处理 event 时间扫描 sysfs 的时候产生了锁竞争
    - 优化了一个 mcs 中的多次 cp 以及 双版本部署中 odps 相关逻辑
    - 上线支持 blk 设备 discard 能力，优化创建 pod 时的写 io 和磁盘占用
    - aone 发现多例pod 卡 terminating，逐项排查中

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：



<span style="background-color: #F9DDB2;">KR5:</span> 支持 EAS 支持通义等内外部客户使用 microsandbox，业务规模增长xx% @卢永强(云住)

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：
- 乌兰察布公有云客户集群开服：先行发布 rund 3.1.10.6，等待机器资源导入；
- 在上海&香港生产集群发布 microsandbox 正式版本 1.0.32-20260723170911.6941fd7，以解决 mount options char limit 缺陷，并携带磁盘 GC 特性；
- sandbox 支持多 network service rule，以访问 eas 提供的网络加速等公共服务：开发中，计划八月中联调，随后业务发起安全评审；@侯开宇(宇常)@冯世舫(旭彤)
- ANSM 提供 Sandbox 访问集群本地宿主机网络方案：联调中，ANSM 支持自定义带宽以及针对单个 sandbox 的限流；@侯志远(至原)@侯开宇(宇常)

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：



### **【性能】显著提升沙箱性能，打造行业领先(优于友商)的冷启动与并发能力以及低资源开销能** @文悦力(悦霄)@王小哲(笑哲)@徐誉畅(星檐)

围绕启动速度、唤醒延迟和大规模并发交付，建立性能标杆

<span style="background-color: #BDD8EE;">KR1</span>：MicroSandbox 单实例平均启动时间从 500ms 降低至 \<50ms（P50），P99 ≤ 100ms 单机QPS：200 sandbox/s ；挑战单集群并发QPS: 10000 sandbox/s（5000 sandbox/s），QPM：\>=60w sandbox/min；整体保障弹性和运行性能优于友商、资源开销低于友商@王小哲(笑哲)@文悦力(悦霄)

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：
- 友商对比：

本周开始做agentEnv的对比测试，目前同机型上，agentEnv官方提供的resume测试，1并发和50并发的性能结果都比microsandbox要好得多，目前发现一个疑点是agentEnv使用的vm规格是1c 128M，比我我们默认测试的2C2G要小得多，待对齐标准后再做验证分析。
- microsandbox优化
    - 存储优化：mem-file pagecache的瓶颈问题，目前erofs和复用mem-file都可以使用pagecache。另外对比同样适用ublk的agentEnv，也需要dadi尝试针对池化方案做优化。
    - dragonball & 网络：除mem-file外，在多并发场景存在dragonball:resume和UpdateNetwork的瓶颈，准备和天千、宇常针对这一块继续深入分析。
- 集群E2E：

本周fc 50机器的集群测试已经跑起来了，目前500镜像的并发，E2E耗时整体在200-300ms，个别节点存在dadi io慢导致的长尾。



<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：

<span style="background-color: #BDD8EE;">KR2：</span>实现浅休眠实例 P99 唤醒延迟 ≤1ms，深休眠 P99 唤醒 \<1s，沙箱克隆 P99 \<1s，全部纳入自动化压测基线 @徐誉畅(星檐)

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：
- 深休眠：
    - 深休眠能力交付适配关键节点：
        - ✅FC 完成 AsyncArchive 接口对接：7/31
        - msbx 支持增量快照和状态查询等接口的版本：
            - ✅8/4 提供测试版本，FC 已验证通过
            - 【进行中】8/5 提供正式版本
            - 8/6发布到 mulzones 集群
        - 可以在mulzones 集群进行 happypath 的深休眠端到端测试：8/10
        - 可以在 mulzones 集群进行端到端集群压测：8/20
    - msbx 深休眠功能开发@卢永强(云住)@傅城瑜(浮鱼)
        -  microsandbox 侧：
            - ✅memfile 采用 in-place 模式，首次 checkpoint attach 新设备，后续 checkpoint in-place 复用
            - ✅hypervisor 能力接口，业务联调中
            - ✅AsyncAchieve 接口支持并发上传 rootfs 和 snapfile，实测并行化之后对 2C4G10G 的 sandbox进行深休眠，产物 大小rootfs （390KB）snapfile(172MB) ，并行上传可以节省 3.8s 左右，提升 24%
        - dragonball 侧：
            - ✅增量快照代码已合入
            - ✅hypervisor 能力代码已合入
            - ✅rund 3.1.10.7 出包中
        - 脏页追踪性能@王小哲(笑哲)
            - pvm/嵌套虚拟化/裸金属环境，运行常规性能测试套：unixbench，sysbench，mlc，mem-lat，stream等，均未发现性能出现明显下降的现象。
            - 构建的特殊用例：在pvm反复写4G内存，其中第二次写入性能下降70%，而首次和第三次往后性能持平。该现象在嵌套虚拟化和裸金属上均未发现。
    - 单机深休眠功能性能测试@卢永强(云住)@刘关骁雄(福克斯)
        - 本周主要聚焦在深休眠增量功能测试用例接入
    - FC 环境基于 AsyncArchive 深休眠性能测试，三方面性能问题：
        - Archive 上传慢 11s，其中 snapfile 上传耗时 9.79s
            - 目前看到 streamvollume push ACR 是单流模式，而 ACR 后端默认 4 并发上传，联系 ACR 同学将fc-dadi-acr-ee-ci这个 ACR 实例后端调成 16 并发（期望并发 1 测试深休眠的上传带宽线性扩展）。
            - 优化后的效果：AsyncArchive msbx 这边记录耗时 7s，其中上传 snapfile 耗时 4.69s，从数据来看，ACR 服务端并发扩展4 倍后上传带宽扩展 2 倍，单机并发度为 1 时深休眠上传速度的瓶颈主要是在于 ACR 这里。不过 ACR 实例单流并发调高之后整体的多层并发就越少，而且目前 ACR 没有开放配置（当前是找 ACR 同学后台修改的）
            - ✅已优化：AsyncArchive 接口rootfs 和 snapfile 并行上传，两个数据流并发
        - delete 10s 走到强杀
            - 优雅关机确保 io 清理干净，避免 overlaybd 设备 有 inflight-IO 问题 -- 待优化卷宝实现
        - 休眠唤醒 resume 3s
            - 普通 resume 可以通过 trace 优化，但是目前深休眠场景还没有使能 trace -- 待确认方案



<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：

FC 双日会黄刚提出 830 前需要提供深休眠通过云盘或 CPFS 等方案实现常态高性能能力，需要进一步明确产品能力边界以及白寂反馈分工需要再确认。



<span style="background-color: #BDD8EE;">KR3：</span>在模拟swe RL场景，完成 5000 个异构镜像（平均 3GB）并发冷启动测试，端到端 P99 ≤1s，链路各环节瓶颈定位闭环，完成通义等RL场景上线@徐誉畅(星檐)@张义飞(天千)

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：
- FC mulzones 集群 1s 集中下发 500 并发压测到 50 节点，sandbox 规格为 2C2G10T，在没有 CPFS 预热情况、有 P2P root 缓存情况下，排除持续异常节点后，Resume P50/P99/Max 为`205.28/1234.34/1806.06ms`
    - 待优化：
        - 目前500镜像没有开启trace录制，会导致wait for envd性能非最佳 -- FC 正在重新构建打开 trace 镜像中
        - overlyabd dev\_open 有一个dev\_mtx全局锁，会导致同一节点上设备创建串行排队；以及每个 sandbox 都有两个设备，但是节点并发排队时两个设备在队列中的位置随机，性能会取最慢的设备。-- dadi 同学正在准备并发优化版本
        - 节点上即使从 p2p root 拉数据延迟也比较高，原因是目前 p2png 会从 root 拉取数据后同步写磁盘，节点PL1 盘会拖累 p2p 性能。可以打开 p2p 分离线程开关测试效果。-- dadi 同学确认配置中
    - 调查问题点：
        - 长尾性能异常节点：测试 18 轮中，有 4 组 max 去到 2s\+，慢在 wait for envd 阶段的长尾集中在同一个节点，@杜佳薇(沅初)@王小哲(笑哲) 目前调查发现异常节点 p2p root 拉取数据延迟比正常节点高 3 个数量级，同时数据盘监控看到写延迟近 40ms。

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：

<span style="background-color: #BDD8EE;">KR4:</span> 基于 fork 与 Memory Trace 技术实现 MicroSandbox 的极致高密与快速弹性、端到端弹性时间下降 50% @曾嘉豪@李柯樾(麟止)

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：
- <span style="color: ;">Memory trace: 完成poc，和dragonball方面对齐uffd方案，开发中</span>
- <span style="color: ;">ublk共享page cache：</span>
    - <span style="color: ;">改造ublk用 digest索引，而不是 refname</span>
    - <span style="color: ;">在 TTL 上设计一个约束，最多缓存N个</span>
- <span style="color: ;">Fork: 由于gshmem方案存在内存难以回收和父文件不可写的限制，探索使用xfs\+dax方案替代gshmem，代码开发中</span>



<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：



<span style="background-color: #BDD8EE;">KR5:</span> 探索FC Sandbox 场景支持整机6倍超卖，并完成全链路的上线；支撑集团aone sandbox cpu超卖20倍、内存超卖8倍上线 @杨嵘(欧一)@邹旭(世尊)@文悦力(悦霄)

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：
- 上周五和业务研发侧、PD 讨论 rund/microsandbox 内存超卖方案
    - rund：
        - 静态配置方案：沿用`securecontainer.alibabacloud.com/runtime.default_memory_reclaim_level`，值范围 0～3，表示内存回收激进程度
        - 动态回收：通过 containerd-shim-rund-v2-cli 接口，需要优化老接口，预期对外接口形式为`containerd-shim-rund-v2-cli reclaim-memory --sandbox xxx --pressure-level 4 --size xxx --timeout 30`
        - 内存可回收量估算算法：通过 rund cli stats 接口获取 metric 进行计算，[RunD 内存超卖可回收量预估](https://alidocs.dingtalk.com/i/nodes/Y1OQX0akWmzdBowLFbebDeklVGlDd3mE?utm_scene=person_space)

    - microsandbox：
        - 业务倾向希望将内存超卖能力下沉到 guest 内核态实现，避免管控面逻辑被用户破坏。[microsandbox 内存超卖接口讨论](https://alidocs.dingtalk.com/i/nodes/amweZ92PV6DbOdgzUMokKlG38xEKBD6p?utm_scene=person_space)


<span style="color: rgb(20, 20, 20);">**业务重要信息：**</span>
- 神龙裸金属超卖率：![image.png](https://alidocs2.oss-cn-zhangjiakou.aliyuncs.com/res/8K4nyeLKEZ1R5nLb/img/dc300208-e9e7-4800-a55d-b148516fed34.png?Expires=1786118390&OSSAccessKeyId=LTAI5tKTjg4Kq1HCdBJ8qpSp&Signature=rrBbDo3J4kpf9B9ZWdSgyN%2FhoiU%3D "")![image.png](https://alidocs2.oss-cn-zhangjiakou.aliyuncs.com/res/8K4nyeLKEZ1R5nLb/img/ac181cc7-df57-4077-8e25-5b9d460b29bf.png?Expires=1786118390&OSSAccessKeyId=LTAI5tKTjg4Kq1HCdBJ8qpSp&Signature=6FjLGJ6qu%2FVBxV92kq3jkpRAnm8%3D "")

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
- microsandbox 本身的安全模型边界问题需要先解决，不仅涉及到内存超卖，包括guest 指标采集等在 rund 上通过 agent 在 guest 用户态实现的方案都需要重新设计。



<span style="background-color: #BDD8EE;">KR6:</span> 【挑战型】实现IP不变和迁移的能力，支持沙箱克隆功能，保障沙箱有状态任务无感，downtime时间平均\<=200ms @张义飞(天千)

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：
1. 无进展，等@侯志远(至原)这边投入人力评估网络迁移方案的可行性，给结论。预计本周三给出

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：

<span style="background-color: #BDD8EE;">KR7:</span> 与云网络协作AgentSandbox网络优化，基于eni-director和VPC NAT，下沉ANSM部分数据平面，降低资源开销，提升转发性能 @侯志远(至原)

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：

详细方案设计中

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：



<span style="background-color: #BDD8EE;">KR8:</span> <span style="color: rgb(13, 18, 57); background-color: rgb(246, 247, 248);">联合虚拟化团队完成 microsandbox 在嵌套虚拟化场景的兼容性攻关与性能调优，确保功能稳定运行并实现性能提升 5%。</span>@郭蔚(萧封)@吴丹(栀予)

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：
- FC同学总结了嵌套虚拟化性能问题，https://yuque.alibaba-inc.com/nrz83s/uu0rac/cfsx7aqe2my1apky， 给出了目标：与六代裸金属对齐。 
    - 我们联合虚拟化同学一起整理的micro 路径开销分析
https://yuque.alibaba-inc.com/xiaofeng.gw/ulf54s/ekwy2s0efsekfkh8?singleDoc# 《嵌套虚拟化L2\_VMEXIT分析》
总启动时间差值为 563ms. EPT violation（\+310.4 ms，占 56.4%） 和 IO instruction（\+222.2 ms，占 40.3%） 是两个主要的问题。
    - 计划今天整合方案，争取明天评审 
        - L0优化(如mmu\_lock, 4k 页） ： @张必宽(张必宽)
        - L1 内存优化： @萧封
        - L1 console IO优化： @王小哲(笑哲)
- 嵌套虚拟化机型适配 @吴丹(栀予)
    - 主要是测试开发，计划是放到812的迭代。
    - 测试
        - 功能测试没有柱塞的问题，有一些aone.
        - 性能测试



<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：



### **【稳定性&质量&安全】构建高可用、可观测的沙箱运行体系，保障系统稳定与质量水位** @赵虹钧(宫贰)@刘起明(寒枝)

强化监控、告警、trace 和测试覆盖，形成闭环质量保障机制

<span style="background-color: #C5DFB4;">KR1:</span> 完成 MicroSandbox 全链路监控采集（host/microsandbox/sandbox 层），100% 接入璇玑平台，关键指标覆盖率 ≥95% 核心指标采集的频率 1秒(现状30秒) @刘起明(寒枝)

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：



<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：

<span style="background-color: #C5DFB4;">KR2:</span> 建立核心场景的 trace 链路追踪能力，关键路径（如创建、唤醒、销毁）trace 覆盖率达 100%，异常归因效率提升 50%@刘起明(寒枝)

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：



<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：

<span style="background-color: #C5DFB4;">KR3:</span> 完成运行时与弹性性能基线建设，完成microsandbox CI/nightly 自动化测试/异常场景测试（当前值需补全）@刘关骁雄(福克斯)@朱若彬(佑杉)

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：
- <span style="color: rgb(20, 20, 20);">**版本和项目交付**</span>
    - <span style="color: rgb(20, 20, 20);">cpu template 嵌套虚拟化/裸金属之间跨机恢复测试</span>
        - <span style="color: rgb(20, 20, 20);">测试发现裸金属build -\> 嵌套虚拟化resume失败</span>[（aone）](https://project.aone.alibaba-inc.com/#viewIdentifier=2f9ff75d4e920343aa97c064&openWorkitemIdentifier=84880158)<span style="color: rgb(20, 20, 20);">需要基于嵌套虚拟化机型重新制作CPU template模版文件</span>
- <span style="color: rgb(20, 20, 20);">**microsandbox CI/nightly 自动化测试体系构建**</span>
    - <span style="color: rgb(20, 20, 20);">新增nightly测试用例覆盖：</span>
        - <span style="color: ;">dadi 短路加载panic问题回归测试</span> [(aone)](https://project.aone.alibaba-inc.com/#viewIdentifier=1c46ee8637e0c978f115b6f7&openWorkitemIdentifier=84752225)
    - <span style="color: ;">构建dragonball支持windows的nightly测试</span>
        - <span style="color: ;">测试计划（含结果）：</span>[<span style="color: ;">构建dragonball支持windows的nightly测试计划</span>](https://alidocs.dingtalk.com/i/nodes/o14dA3GK8gQlkoYwcnnkxDAXV9ekBD76)
        - <span style="color: ;">测试覆盖：基础测试：build template -\> resume sandbox(ping \+ curl) -\> archive -\> resume new sandbox(ping \+ curl)</span>；<span style="color: ;">多线程同步</span><span style="color: ;">并发；稀疏内存文件。</span>
    - <span style="color: ;">验证 memfile 稀疏文件有效性</span>
        - <span style="color: ;">在 windows 环境中加入相关代码，测试进行中</span>
    - <span style="color: ;">支持 dragonball 增量快照能力功能测试</span>
        - <span style="color: ;">测试计划（含结果）：</span>[支持 dragonball 增量快照能力功能测试](https://alidocs.dingtalk.com/i/nodes/vy20BglGWOxjGpq0CgpjKRGoVA7depqY)
        - 测试覆盖：fio，典型的net打流，stress-ng 内存读写，时间连贯性
- **性能基线构建**[（grafana）](https://os-grafana.alibaba-inc.com/d/PbA-4_hDz/microsandbox-performance-test?spm=7d0b313d.2ef5001f.0.0.496376717GUxeG&orgId=1)
    - <span style="color: ;">新增fc环境性能基线</span>
    - <span style="color: ;">更新min/max/p50/p90数据指标</span>
    - <span style="color: ;">支持通过sandbox规格和并发量筛选趋势图</span>
    - <span style="color: ;">构建深休眠单机性能基线</span>
        - <span style="color: ;">单机 60 并发、100 并发 - 全量 checkpoint，同 ACR</span>
        - <span style="color: ;">双机 60 并发、100 并发 - 全量 checkpoint，同 ACR</span>
        - <span style="color: ;">双机 60 并发、100 并发 - 全量 checkpoint，双 ACR</span>
- **新增问题**
    - 本周新增问题18个（[aone](https://project.aone.alibaba-inc.com/v2/project/2155383/bug#viewIdentifier=5dfb195e2e2b84f6b2f24718&initFilterParams=%5B%7B%22fieldIdentifier%22:%22creator%22,%22value%22:%5B%22526582%22,%22445571%22%5D,%22toValue%22:null,%22operator%22:%22CONTAINS%22%7D,%7B%22fieldIdentifier%22:%22gmtCreate%22,%22value%22:%5B%222026-07-29%2000:00:00%22%5D,%22toValue%22:null,%22operator%22:%22MORE_THAN_AND_EQUAL%22%7D%5D)）
        - 已解决并复验6个



<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：

<span style="background-color: #C5DFB4;">KR4:</span> 构建Agent的细粒度的网络行为审计和防护能力，覆盖公网流量、vpc流量，支持L4/L7多种主流协议 @侯志远(至原)

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：
- IP/域名访问控制完成开发、联调，线上灰度发布中；
- Http/Https的domain rule支持完成开发，本周开发发布；
- http/https rule新增支持qoder需求（支持value占位符的匹配），完成开发，测试中；
- 沙箱实例的L4访问，线上灰度发布了3个集群，后续发布进行中；

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：





<span style="background-color: #C5DFB4;">KR5:</span> 【挑战型】联合云安全和产品团队，探索构建Agent整体安全防护能力，落地成为Rund Agent Sandbox标准化安全解决方案 @冯世舫(旭彤)

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：

本周无新增进展。

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：



### **【预研】探索下一代轻量化 语言运行时级Agent Sandbox 技术路径， 完成可行性技术验证** @刘起明(寒枝)

面向未来轻量化、安全隔离需求，开展关键技术预研

<span style="background-color: #DFEBF6;">KR1:</span> 定义轻量 Agent Sandbox 的核心评估指标（启动、内存、隔离性、兼容性），建立原型选型框架，并输出性能对比报告

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：



<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：



<span style="background-color: #DFEBF6;">KR2:</span> 输出 1 份《轻量化 Agent Sandbox 技术演进路线图》，包含至少 2 种备选方案（如 WASM、unified runtime）及适用场景建议

<span style="color: rgb(20, 20, 20);">**本周进展**</span>：



<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：


---
> 采集方式：`dws drive search` + `dws doc read`（只读）。
