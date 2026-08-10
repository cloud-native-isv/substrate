# Agent Sandbox 7月总结

- **来源**: 钉钉文档（云盘搜索命中，全文读取）
- **原文链接**: https://alidocs.dingtalk.com/i/nodes/a9E05BDRVQRkezKGCDP561b4J63zgkYA?utm_scene=team_space
- **采集时间**: 2026-08-07 22:00

### **【规模】全面落地 runD MicroSandbox 架构，支撑三大核心场景并完成客户切换，全年达到**<span style="color: rgb(193, 0, 2);">**500w核(待对齐)**</span> @杨嵘(欧一)@段云鹏(天平)

以 FC Sandbox 的 RL/Agent Harness/Tool Use 场景为核心，推动全量新客户接入，并完成存量用户迁移, 新增客户中优先完成通义评测/RL场景接入FC Sandbox

KR1 ：支持FC Sandbox新增客户(优先通义评测/RL场景 支持osworld/androidworld)全量接入 MicroSandbox，新增客户上线率 100%，在财年结束前，实现 100% 存量 FC Sandbox 用户切换至 runD MicroSandbox 架构 @杨嵘(欧一)@段云鹏(天平)

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>
- 730 FC 深休眠/snapshot 能力全链路就绪（延期至810）。
- 730 出口流量白/黑名单控制

![image.png](https://alidocs2.oss-cn-zhangjiakou.aliyuncs.com/res/2M9qP5kJPJGVDO01/img/74b0f69a-9f25-4d7f-a5b8-de594e3bb07f.png?Expires=1786118349&OSSAccessKeyId=LTAI5tKTjg4Kq1HCdBJ8qpSp&Signature=10Lfd0lF5n%2B%2BGJQD0hP%2Bz2TItro%3D "")
- 730 动态挂载 microsandbox技术方案已沟通，e2b\+microsandbox链路 fc延期至815（小米可能来不及）
- 730 安全漏洞封堵与 CVE 治理闭环：cgroup/vsock/seccomp紧急修复→KVM逃逸CVE-2026-53359内核签名→AMD裸金属安全策略升级。FC 6代机 rund2.8 提权漏洞待排期。

<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>
- <span style="color: #f33e3e;">悟空e2b microsandbox 7.20已上海上线，但没有大规模宣传，7.29开始放量（1000/h左右），目前符合预期</span>
- <span style="color: #f33e3e;">小米：</span>
- Qoder北京、新加坡流量 100%（RunD 容器），准备切 microsandbox





<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>

| <span style="color: rgb(20, 20, 20);">规模档</span> | <span style="color: rgb(20, 20, 20);">CPU超卖率</span> | <span style="color: rgb(20, 20, 20);">CPU利用率</span> | <span style="color: rgb(20, 20, 20);">内存利用率</span> | <span style="color: rgb(20, 20, 20);">交付核数/min</span> | <span style="color: rgb(20, 20, 20);">**交付核数/日 (×1440)**</span> | <span style="color: rgb(20, 20, 20);">交付容器/min</span> | <span style="color: rgb(20, 20, 20);">调度P99(ms)</span> |
|------------------------------------------------------|---------------------------------------------------------|---------------------------------------------------------|------------------------------------------------------------|-------------------------------------------------------------|--------------------------------------------------------------------------|-------------------------------------------------------------|----------------------------------------------------------|
| <span style="color: rgb(20, 20, 20);">1万核以上</span> | <span style="color: rgb(20, 20, 20);">65.25% (均104%)</span> | <span style="color: rgb(20, 20, 20);">9.69% (均21.6%)</span> | <span style="color: rgb(20, 20, 20);">17.85% (均38.1%)</span> | <span style="color: rgb(20, 20, 20);">764 (均538)</span> | <span style="color: rgb(20, 20, 20);">**1,100,160 (均774,720)**</span> | <span style="color: rgb(20, 20, 20);">920 (均595)</span> | <span style="color: rgb(20, 20, 20);">52 (均769)</span> |
| <span style="color: rgb(20, 20, 20);">5千-1万核</span> | <span style="color: rgb(20, 20, 20);">**155.47%**</span> <span style="color: rgb(20, 20, 20);">(均165%)</span> | <span style="color: rgb(20, 20, 20);">13.09% (均19.3%)</span> | <span style="color: rgb(20, 20, 20);">25.63% (均42%)</span> | <span style="color: rgb(20, 20, 20);">637 (均440)</span> | <span style="color: rgb(20, 20, 20);">**917,280 (均633,600)**</span> | <span style="color: rgb(20, 20, 20);">606 (均451)</span> | <span style="color: rgb(20, 20, 20);">26 (均263)</span> |
| <span style="color: rgb(20, 20, 20);">1千-5千核</span> | <span style="color: rgb(20, 20, 20);">121.74% (均132%)</span> | <span style="color: rgb(20, 20, 20);">**25.33%**</span> <span style="color: rgb(20, 20, 20);">(均23%)</span> | <span style="color: rgb(20, 20, 20);">32.2% (均33.5%)</span> | <span style="color: rgb(20, 20, 20);">394 (均239)</span> | <span style="color: rgb(20, 20, 20);">**567,360 (均344,160)**</span> | <span style="color: rgb(20, 20, 20);">614 (均386)</span> | <span style="color: rgb(20, 20, 20);">82 (均560)</span> |
| <span style="color: rgb(20, 20, 20);">\<1千核</span> | <span style="color: rgb(20, 20, 20);">128.82% (均111%)</span> | <span style="color: rgb(20, 20, 20);">7.95% (均6.5%)</span> | <span style="color: rgb(20, 20, 20);">21.28% (均17.7%)</span> | <span style="color: rgb(20, 20, 20);">12 (均9)</span> | <span style="color: rgb(20, 20, 20);">**17,280 (均12,960)**</span> | <span style="color: rgb(20, 20, 20);">25 (均19)</span> | <span style="color: rgb(20, 20, 20);">8 (均6.7)</span> |

![image.png](https://alidocs2.oss-cn-zhangjiakou.aliyuncs.com/res/2M9qP5kJPJGVDO01/img/7a70cf6d-e540-4689-a910-19695d7a4da1.png?Expires=1786118349&OSSAccessKeyId=LTAI5tKTjg4Kq1HCdBJ8qpSp&Signature=t6CeVXZIrwj7maw7yIk2mbrH%2FSI%3D "")

![image.png](https://alidocs2.oss-cn-zhangjiakou.aliyuncs.com/res/2M9qP5kJPJGVDO01/img/ef256d9f-addc-43f9-9c38-904c9596584f.png?Expires=1786118349&OSSAccessKeyId=LTAI5tKTjg4Kq1HCdBJ8qpSp&Signature=ynHWcwLdtcEE3Z0Y7YY0A6Wq32w%3D "")





<span style="color: rgb(20, 20, 20);">**关键进展**</span>：
- 网络 出口流量白/黑名单控制 已开发联调完成，预计7.30上线成都 
- 深休眠/snapshot 与FC接口设计已对齐 @徐誉畅(星檐)@卢永强(云住)
- Qoder 深休眠压测稳定性问题已修复
- 可观测 microsandbox大盘已上线，全链路指标数据大量完善，环境拆分初步完成  @杜佳薇(沅初)@史文杰(鸿砚)
- 模版预取已开发完成，测试数据wait for envd冷启动加速5倍以上（2s -\> 200ms），测试环境灰度中 @郭豪(昊石)
- 动态挂载/Volume 技术方案已与FC 沟通完成
- android/windows OSWorld 联调演示完成  @高瀚翔(斯哒)@郭豪(昊石)

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
- <span style="color: ;">RunD agenticfs 支持：happy case POC 跑通，alinas 本身超时处理行为仍有问题在对齐，以及后续 alinas on RunD rootfs 职责边界待讨论</span>
- <span style="color: ;">AMD 裸金属/嵌套虚拟化 CPU template：仍在开发中，预计 8 月底支持</span>
- <span style="color: ;">microsandbox 新增需要对齐 RunD 功能点：</span>
    - <span style="color: ;">仅保存/恢复文件系统：业务已上线仅恢复文件系统功能，正在适配仅保存文件系统功能</span>
    - <span style="color: ;">instance Metric：业务对客展示实例 Metric 信息，microsandbox 目前欠缺相关 Metric 采集</span>
- <span style="color: #E03E3E; background-color: #FFFFFF;">FC没定volume在microsandbox的全链路详细设计，目前还在支持runD场景，上线时间延期为815</span>
- <span style="color: ;">L4网络打通特性</span>  <span style="color: #E03E3E; background-color: #FFFFFF;">ansm 开发完成，已经在灰度</span><span style="color: ;">；microsandbox 上待跟FC同学联调；依赖网络组件节点和管控两部分升级</span>
- <span style="color: ;">L7 rules</span> <span style="color: #E03E3E; background-color: #FFFFFF;">ansm 开发完成，在跟fc同学rund端到端联调, microsandbox待开始</span>
- <span style="color: ;">存储dadi 存在稳定性问题，0.6.21上线后出现core，目前已回滚 0.6.15，出现其他稳定性问题。0.6.21-fix版本预计本周五提供</span>
- <span style="color: #E03E3E; background-color: #FFFFFF;">dadi镜像转换和resize工具已延期一周</span>



KR2: 同舟平台完成 Agent Sandbox 架构升级，实现整机快照恢复能力，新增用户 100% 基于新架构上线，存量用户完成 xx 核迁移计划，预期全年达到100w\+核（明确数量或比例） @郭豪(昊石)

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>
- 浅休眠节点联调完成
- 深休眠rund相关问题都已修复

<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>
1. 浅休眠，原计划730上线浅休眠，目前由于团队调整 delay, 新的时间还没有决定
2. 计划通过pod vm复用二期功能来增大同舟规模。

<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>
- 总量峰值22w 核 （6.30日），当前16.6核

<span style="color: rgb(20, 20, 20);">**关键进展**</span>：

待填写

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
- 浅休眠/深休眠，原计划730上线，目前由于团队调整 delay, 新的时间还没有决定



KR3: ack rund 比优于开源kata\+firecracker和gvisor要好，交付3\+个标杆客户 @郭蔚(萧封)

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>
- 

<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>
- 

<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>

<span style="color: rgb(20, 20, 20);">待填写</span>

<span style="color: rgb(20, 20, 20);">**关键进展**</span>：

\-  PDSA反馈， gvisor启动也是分钟级： 客户在ACK上，使用gvisor的效果：79个pod时候，大概5分钟左右全部拉起。单个pod 80s。 需要继续跟进 客户真正关注的指标是什么； 另外需要加上falco 的负载继续测试。

 - 其他需求inprogress：

  a)  [https://project.aone.alibaba-inc.com/v2/project/815835/req/84076527#](https://project.aone.alibaba-inc.com/v2/project/815835/req/84076527#) 《【Minimax】嵌套虚拟化需求：嵌套虚拟化支持Alinux4》    

   目标(八月底）

  b)  [https://project.aone.alibaba-inc.com/v2/project/815835/req/84402845#](https://project.aone.alibaba-inc.com/v2/project/815835/req/84402845#) 《ACK RunD 场景安全能力接入（Service API & VEPF 支持）》  

   同舟正在联调的版本ready后就可以使用。

  c) [https://project.aone.alibaba-inc.com/v2/project/815835/req/80577441#](https://project.aone.alibaba-inc.com/v2/project/815835/req/80577441#) 《\[ACK\] 支持cri链路多容器pod的整机休眠唤醒》

出方案中

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
- <span style="color: #f33e3e;">gvisor在启动和密度的竞对</span> 

<span style="color: #f33e3e;">\[ACK\] 支持cri链路多容器pod的整机休眠唤醒save/restore 的方案</span>

KR4: 集团支持aone agent沙箱弹性扩缩容、电商ODPS混池能力突破，业务规模增长xx% @文悦力(悦霄)

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>
- <span style="color: rgb(20, 20, 20);">日 vcpu：9wcore，日核时 220w</span>

<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>
- <span style="color: rgb(20, 20, 20);">AP 业务于 730 完成资源交付链路打通，目前已经在做大规模压测（1000、5000、10w并发）反馈出来的问题包括功能和启动性能，业务对启动性能要求比较高（2min 内所有作业就绪，目前5000并发的P90在五分钟左右），目前仍未达标，主要瓶颈在于管控调度链路。</span>
- <span style="color: rgb(20, 20, 20);">AP 业务将于 815 交付 bench 使用，在此之前保障联调和功能使用</span>
- <span style="color: rgb(20, 20, 20);">Aone 业务为了扩大资源用量，正在实现在大资源池上做联邦调度，发布支持中</span>
- <span style="color: rgb(20, 20, 20);">Aone 业务线 7 月 29 日发生一起</span><span style="color: #C10002;">**线上故障**</span><span style="color: rgb(20, 20, 20);">：业务方资源紧缺，调大超卖比后由 200\+ 台神龙机器挂掉。初步排查是超卖场景下内存被打爆并且没有得到有效处理，千问 C 端、devix 等多用户反馈服务不可用或启动超时。</span>

<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>

<span style="color: rgb(20, 20, 20);">日 vcpu：9wcore，日核时 220w</span>

<span style="color: rgb(20, 20, 20);">**关键进展**</span>：
- 支持 Agent platform 业务使用集团 rund 资源
    - 交付 guest-volume 功能
    - 交付 vsock 实现 host guest 通信功能
    - 筹备 3.1.14.1 修复 kvm 逃逸 CVE
    - 在大资源池实现 odps 和 sandbox 混部
- 支持 aone 业务使用集团 rund 资源
    - 在大资源池实现 odps 和 sandbox 混部

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：

集团 agent 场景用量逐渐增加，建议袋鼠内部和业务侧联动搭建日常测试环境和版本出口覆盖

### **【性能】显著提升沙箱性能，打造行业领先(优于友商)的冷启动与并发能力以及低资源开销能** @文悦力(悦霄)@王小哲(笑哲)@徐誉畅(星檐)

围绕启动速度、唤醒延迟和大规模并发交付，建立性能标杆

KR1：MicroSandbox 单实例平均启动时间从 500ms 降低至 \<50ms（P50），P99 ≤ 100ms 单机QPS：200 sandbox/s ；挑战单集群并发QPS: 10000 sandbox/s（5000 sandbox/s），QPM：\>=60w sandbox/min；整体保障弹性和运行性能优于友商、资源开销低于友商 @文悦力(悦霄)@王小哲(笑哲)

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>
- 

<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>
- <span style="color: #f33e3e;">\*请填写7月业务重要信息</span>

<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>

<span style="color: rgb(20, 20, 20);">待填写</span>

<span style="color: rgb(20, 20, 20);">**关键进展**</span>：

同机型条件下，与cubesandbox的单机模版恢复性能对比，从目前测试数据来看，microsandbox在对齐存储方案(同一模版mem-file复用pagecache)后，性能略好于cubesandbox。

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
- <span style="color: #f33e3e;">\*待填写</span>



KR2：实现浅休眠实例 P99 唤醒延迟 ≤1ms，深休眠 P99 唤醒 \<1s，沙箱克隆 P99 \<1s，全部纳入自动化压测基线 @徐誉畅(星檐)

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>

<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>
- <span style="color: #f33e3e;">计划 830 对客交付 msbx深休眠能力</span>

<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>

createSnapshot<span style="color: rgb(20, 20, 20);">：</span>期望 1.5w(待确认) 个实例 40s   单机并发 10  集群并发 XX

基于快照1:N fork / 1:1 resume： 期望1.5w 3s 内  单机并发X  集群并发 XX

<span style="color: rgb(20, 20, 20);">**关键进展**</span>[：]()
- 深休眠：
    - 和 FC 对齐 msbx 提供的接口，并明确了交付适配时间节点，关键节点如下：
        - FC 完成 AsyncArchive 接口对接：7/31
        - msbx 支持增量快照和状态查询等接口的版本：8/4 提供测试版本，8/5 提供正式版本，8/6发布到 mulzones 集群
        - 可以在mulzones 集群进行 happypath 的深休眠端到端测试：8/10
        - 可以在 mulzones 集群进行端到端集群压测：8/20
    - msbx 深休眠功能开发@卢永强(云住)@傅城瑜(浮鱼)
        -  microsandbox 侧：
            - memfile in-place 模式联调完成，review 中，attach 新设备方案评估中
            - hypervisor 能力接口，联调中
        - dragonball 侧：
            - 增量快照代码已合入
            - hypervisor 能力接口，联调中
        - 脏页追踪性能@王小哲(笑哲)
            - pvm/嵌套虚拟化/裸金属环境，运行常规性能测试套：unixbench，sysbench，mlc，mem-lat，stream等，均未发现性能出现明显下降的现象。
            - 构建的特殊用例：在pvm反复写4G内存，其中第二次写入性能下降70%，而首次和第三次往后性能持平。该现象在嵌套虚拟化和裸金属上均未发现。
    - 单机深休眠性能测试@卢永强(云住)@刘关骁雄(福克斯)
        - 用例支持了覆盖AsyncArchive、全量/增量 checkpoint方式、单机 1/10/20/40/100并发的性能测试和两机各 100 并发测试，发现两机并发测试上传带宽之和与单机上限近似，目前分别 push 两个账号 ACR可以达到 600MB/s 带宽，但是仍没有达到 VPC 或者 ACR 带宽上限，继续分析中。

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
- <span style="color: #f33e3e;">\*待填写</span>



KR3：在模拟swe RL场景，完成 5000 个异构镜像（平均 3GB）并发冷启动测试，端到端 P99 ≤1s，链路各环节瓶颈定位闭环，完成通义等RL场景上线@徐誉畅(星檐)@张义飞(天千)

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>
- 待补充

<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>
- <span style="color: #f33e3e;">FC 目前购买 30T CPFS \+ CL2 缓存（6GB/s 读写，30GB/s 读）经业务评估由于成本和性能考虑，预热不会作为通用能力常态使能，而是提供 openapi 用户显式使能</span>

<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>

<span style="color: rgb(20, 20, 20);">待填写</span>

<span style="color: rgb(20, 20, 20);">**关键进展**</span>：
1. [冷启动 500 并发相同/不同镜像测试](https://alidocs.dingtalk.com/i/nodes/EpGBa2Lm8aZxe5myCwEX4KaPWgN7R35y)（ 5 并发），第一轮测试 500 相同镜像冷启动耗时 max 3.4s，发现&解决如下问题后最新数据待测试：
    1. dadi queue wait 耗时大问题（首个 queue wait 耗时 357ms） -- dadi 提供了新的测试包，目前观察到没有复现 queue wait 问题，修复计划@杜佳薇(沅初)跟进中
    2. wait for envd 长尾问题（max 1.9s） -- 在打开 trace 之后耗时 max 200ms 以内@郭豪(昊石)
    3. FC 默认使用 oci 镜像 Build导致层无法复用，空间占用大且影响性能 -- 已建议 FC 管控使用 dadi 镜像构建

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
- <span style="color: #f33e3e;">\*待填写</span>

KR4: 基于 fork 与 Memory Trace 技术实现 MicroSandbox 的极致高密与快速弹性、端到端弹性时间下降 50% @曾嘉豪@李柯樾(麟止)

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>
- <span style="color: rgb(20, 20, 20);">Memory Trace: 实现冷启动时间降低50%及以上</span>
- <span style="color: rgb(20, 20, 20);">fork，降低fork wait for envd ready时间，并提高fork时的内存超卖率</span>

<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>
- <span style="color: #f33e3e;">\*请填写7月业务重要信息</span>

<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>
- <span style="color: rgb(20, 20, 20);">Memory trace:</span>
    - <span style="color: rgb(20, 20, 20);">冷启动端对端启动时间优化前为360ms，在使用memory trace测试后，其冷启动时间可以下降到200ms</span>
- <span style="color: rgb(20, 20, 20);">Fork</span>
    - <span style="color: rgb(20, 20, 20);">fork使用gshmem后端，fork到uffd启动，总时间100ms，wait for envd阶段耗时20ms。</span>

<span style="color: rgb(20, 20, 20);">**关键进展**</span>：
- <span style="color: ;">Memory trace: 完成poc，和dragonball方面对齐uffd方案，开发中</span>
- <span style="color: ;">gshmem：完成poc方案设计</span>[<span style="color: ;">https://aliyuque.antfin.com/mgzksh/canqwd/mpcpfw6kwwbm7xla?singleDoc#</span>](https://aliyuque.antfin.com/mgzksh/canqwd/mpcpfw6kwwbm7xla?singleDoc#) <span style="color: ;">《基于gshmem的fork方案设计》，可以将fork的端到端时间控制在100ms以内，wait for envd阶段20ms，但同时引入父文件不可写，内存不可回收等限制，需要和microsandbox共同设计fork方案</span>

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
- <span style="color: #f33e3e;">\*待填写</span>



KR5: 探索FC Sandbox 场景支持整机6倍超卖，并完成全链路的上线；支撑集团aone sandbox cpu超卖20倍、内存超卖8倍上线 @杨嵘(欧一)@邹旭(世尊)@文悦力(悦霄)

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>
- 待补充

<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>
- <span style="color: #f33e3e;">FC 大盘：CPU/内存利用率整体低于 50%，有较大超卖空间</span>![image.png](https://alidocs2.oss-cn-zhangjiakou.aliyuncs.com/res/2M9qP5kJPJGVDO01/img/5bda9365-4cf8-415c-a5fb-0231774276ac.png?Expires=1786118349&OSSAccessKeyId=LTAI5tKTjg4Kq1HCdBJ8qpSp&Signature=LnhB89IcjFrZEO0MdXO93MA5CAs%3D "")![image.png](https://alidocs2.oss-cn-zhangjiakou.aliyuncs.com/res/2M9qP5kJPJGVDO01/img/c83a1e60-e175-438b-83d0-f9cbf1b09159.png?Expires=1786118349&OSSAccessKeyId=LTAI5tKTjg4Kq1HCdBJ8qpSp&Signature=40Ez9xthULv1335MWzW4CWsO2pM%3D "")

<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>

<span style="color: rgb(20, 20, 20);">待填写</span>

<span style="color: rgb(20, 20, 20);">**关键进展**</span>：
- 超卖分析依赖大盘，FC 大盘细节较少，袋鼠大盘建设进展：

| 场景 | host 观测大盘 | guest 观测大盘 |
|------|-----------------|------------------|
| RunD 容器场景 | sysak 组件适配中 | 已采集，需要迁移至璇玑 |
| RunD microsandbox 场景 | sysak 组件适配中 | 暂无guest 指标采集方案 |

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
- <span style="color: ;">单机 200 实例内存占用测试，microsandbox 和 rund 使用相同镜像、cpu/内存规格，</span><span style="color: ;">Microsandbox 约 116 MB/sandbox，Rund 约 87 MB/sandbox；Microsandbox 比 Rund 高 33%。相较 rund2.8 平均 20MB 内存占用都有内存膨胀情况。</span>[FC 模版启动内存占用分析](https://alidocs.dingtalk.com/i/nodes/YndMj49yWjlAYoxjtRRogmZwJ3pmz5aA?utm_scene=person_space)

- <span style="color: ;">RunD 内存超卖方案（L0~L5内存回收能力）不能直接照搬上 microsandbox，需要先和 FC 产品对齐内存回收能力边界</span>@杨嵘(欧一)

KR6: 【挑战型】实现IP不变和迁移的能力，支持沙箱克隆功能，保障沙箱有状态任务无感，downtime时间平均\<=200ms @张义飞(天千)

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>

8.30 完成整个热迁移方案的POC



<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>

<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>

<span style="color: rgb(20, 20, 20);">待填写</span>

<span style="color: rgb(20, 20, 20);">**关键进展**</span>：
- @侯志远(至原) <span style="color: ;">给出网络无感迁移的方案分析，7.27日给一个初步的结论</span>
- DADI这边已经给到一份存储热迁移的方案， 内部review完后有一些时序问题，待拉会讨论 [microsandbox热迁移存储方案设计（精简版）](https://alidocs.dingtalk.com/i/nodes/93NwLYZXWyxXroNzCNXoMbQK8kyEqBQm?cid=5040165%3A98439829&utm_source=im&utm_scene=team_space&iframeQuery=utm_medium%3Dim_card%26utm_source%3Dim&utm_medium=im_card&corpId=dingd8e1123006514592)

- <span style="color: ;">热迁移方案完成整体流程的初步设计，和fc研发整体过了一轮，API接口完成了第一版</span>
- <span style="color: ;">dragonball 侧已经给出poc的版本和sdk给到micro sandbox侧进行集成</span>

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
- <span style="color: #f33e3e;">网络热迁移方案还没有给出，是否可行也没有结论</span>

KR7: 与云网络协作AgentSandbox网络优化，基于eni-director和VPC NAT，下沉ANSM部分数据平面，降低资源开销，提升转发性能 @侯志远(至原)

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>
- 完成项目KO，计划下周开始投入

<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>
- <span style="color: #f33e3e;">\*请填写7月业务重要信息</span>

<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>

<span style="color: rgb(20, 20, 20);">待填写</span>

<span style="color: rgb(20, 20, 20);">**关键进展**</span>：

待填写

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
- <span style="color: #f33e3e;">\*待填写</span>



KR8: <span style="color: rgb(13, 18, 57); background-color: rgb(246, 247, 248);">联合虚拟化团队完成 microsandbox 在嵌套虚拟化场景的兼容性攻关与性能调优，确保功能稳定运行并实现性能提升 5%。</span>@郭蔚(萧封)

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>
- 

<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>
- 

<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>

<span style="color: rgb(20, 20, 20);">待填写</span>

<span style="color: rgb(20, 20, 20);">**关键进展**</span>：
- 性能：主要在统一看嵌套虚拟化优化的方案：从两个角度 a) 用大页提升启动时间。 b) 保持4k页，用ept 的预填充和模版加快启动速度，同时兼顾内存开销（保超卖）
 这些方案都被证明有效，还在整理 各个方案的测试结果（启动时间 和 内存开销）对比，以统一整个嵌套虚拟化的内存方案。计划周四架构评审。
- \- 嵌套虚拟化机型适配 

[https://aliyuque.antfin.com/zhiyu.wd/aoaatz/hm4gwcym99k1prav?singleDoc#](https://aliyuque.antfin.com/zhiyu.wd/aoaatz/hm4gwcym99k1prav?singleDoc#) 《嵌套虚拟化机型适配》

目前仅适配9i，前期信息采集已完成，本次机型适配需要覆盖rund和micorsandbox两个场景，测试评估需要6天，开发预留3天。

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：

<span style="color: ;">无</span>

### **【稳定性&质量&安全】构建高可用、可观测的沙箱运行体系，保障系统稳定与质量水位** @赵虹钧(宫贰)@刘起明(寒枝)

【项目跟踪 · 寒枝】KR1 全链路监控采集、KR2 trace 链路追踪的项目跟踪文档已建立（财年口径，含里程碑/WBS/甘特图与责任分工）。截至 2026/07/29 相关里程碑达成 3/11（全量日志采集、全量指标采集、trace 落盘\+TraceStore 已完成）。详见：https://alidocs.dingtalk.com/i/nodes/YndMj49yWjlAYoxjtRgo4rrvJ3pmz5aA

强化监控、告警、trace 和测试覆盖，形成闭环质量保障机制

KR1: 完成 MicroSandbox 全链路监控采集（host/microsandbox/sandbox 层），100% 接入璇玑平台，关键指标覆盖率 ≥95% 核心指标采集的频率 5秒(现状30秒) @刘起明(寒枝)

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>
- 待补充
- ![image.png](https://alidocs2.oss-cn-zhangjiakou.aliyuncs.com/res/2M9qP5kJPJGVDO01/img/f9ea3170-48ac-48c7-87b1-f5504fbc39a5.png?Expires=1786118349&OSSAccessKeyId=LTAI5tKTjg4Kq1HCdBJ8qpSp&Signature=zh2TQRdWjZaFSV8Ywue2bDfqpRQ%3D "")



<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>
- <span style="color: #f33e3e;">807号完成microsandbox璇玑接入</span>@刘起明(寒枝)
    - <span style="color: #f33e3e;">前端休假；delay 到807号</span>
- <span style="color: #f33e3e;">高频采集\+覆盖率 \>95%</span> @卓腾龙(海岳)
    - <span style="color: #f33e3e;">当前5秒的时间已经满足不在进一步看1秒；host 指标采集支持：已有的指标sysak已经提供了，目前在深休眠的测试环境，预计一周后部署到线上，新增的指标已经提给sysak 提需求，预8月15的版本支持</span>

<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>

<span style="color: rgb(20, 20, 20);">待填写</span>

<span style="color: rgb(20, 20, 20);">**关键进展**</span>：

待填写

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
- <span style="color: #f33e3e;">\*待填写</span>

KR2: 建立核心场景的 trace 链路追踪能力，关键路径（如创建、唤醒、销毁）trace 覆盖率达 100%，异常归因效率提升 50%@刘起明(寒枝)

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>
- ![image.png](https://alidocs2.oss-cn-zhangjiakou.aliyuncs.com/res/2M9qP5kJPJGVDO01/img/7fca2abd-3b54-4c3f-bfe3-18f661952bf2.png?Expires=1786118349&OSSAccessKeyId=LTAI5tKTjg4Kq1HCdBJ8qpSp&Signature=YJjBoqAjVGGZeYEH9MVcdhvMhf8%3D "")

<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>
- 

<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>

<span style="color: rgb(20, 20, 20);">待填写</span>

<span style="color: rgb(20, 20, 20);">**关键进展**</span>：

待填写

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
- <span style="color: #f33e3e;">\*待填写</span>



KR3: 完成运行时与弹性性能基线建设，完成microsandbox CI/nightly 自动化测试/异常场景测试（当前值需补全）@刘关骁雄(福克斯)@朱若彬(佑杉)

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>
- <span style="color: rgb(20, 20, 20);">**版本和项目交付**</span>
    - <span style="color: rgb(20, 20, 20);">**版本发布前回归测试**</span>
- <span style="color: rgb(20, 20, 20);">**microsandbox CI/nightly 自动化测试体系构建**</span>
    - <span style="color: #aeada9;">**asi场景 2026.4**</span>
    - <span style="color: #aeada9;">**FC场景 2026.6**</span>
    - <span style="color: rgb(20, 20, 20);">**eas场景 2026.7**</span>
    - <span style="color: rgb(20, 20, 20);">**其他**</span>
- **性能基线构建**
    - <span style="color: #aeada9;">**单镜像resume 2026.4**</span>
    - <span style="color: #aeada9;">**并发resume 2026.4**</span>
    - **多镜像并发resume 2026.7**
    - **深休眠唤醒 2026.7**
- **稳定性测试构建**
    - **业务场景稳定性**
    - **异常场景构建**
- **AI native on microsandbox**
    - **AI构建异常场景**

<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>

<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>

<span style="color: rgb(20, 20, 20);">**关键进展**</span>：
- <span style="color: rgb(20, 20, 20);">**版本和项目交付**</span>
    - <span style="color: rgb(20, 20, 20);">1.0.31功能回归测试</span>
    - <span style="color: rgb(20, 20, 20);">cpu template 嵌套虚拟化/裸金属之间跨机恢复测试</span>
        - <span style="color: rgb(20, 20, 20);">测试发现裸金属build -\> 嵌套虚拟化resume失败</span>[（aone）](https://project.aone.alibaba-inc.com/#viewIdentifier=2f9ff75d4e920343aa97c064&openWorkitemIdentifier=84794048)
- <span style="color: rgb(20, 20, 20);">**microsandbox CI/nightly 自动化测试体系构建**</span>
    - <span style="color: rgb(20, 20, 20);">已完成eas场景 nightly自动打包、部署、测试流程</span>
    - <span style="color: rgb(20, 20, 20);">已完成嵌套虚拟化机型回归测试</span>
    - <span style="color: rgb(20, 20, 20);">新增nightly测试用例覆盖：</span>
        - <span style="color: ;">时钟同步检查</span>
        - <span style="color: ;">ublk device/overlaybd process leak</span><span style="color: rgb(20, 20, 20);">检查</span>
        - <span style="color: rgb(20, 20, 20);">trace template测试</span>
        - <span style="color: rgb(20, 20, 20);">冷启动高并发resume测试</span>
        - <span style="color: ;">dragonball支持windows - 验证中</span>
    - <span style="color: ;">k8s 场景基础功能测试已完成</span>
- **性能基线构建**
    - <span style="color: ;">启动性能数据可视化</span>[（grafana）](https://os-grafana.alibaba-inc.com/d/PbA-4_hDz/microsandbox-performance-test?spm=7d0b313d.2ef5001f.0.0.496376717GUxeG&orgId=1)
    - 基于不同镜像冷启动性能测试 - 测试开发已完成，数据可视化开发中
        - 100/50/20/10并发及单点resume
        - oci/dadi镜像对比
        - trace/untrace template对比
    - microsandbox 内存占用分析（[Microsandbox 内存占用分析](https://alidocs.dingtalk.com/i/nodes/dpYLaezmVNRMGX56CPk30YN1VrMqPxX6?utm_scene=person_space)
）
        - 与Rund模版启动对比（[FC 模版启动内存占用分析](https://alidocs.dingtalk.com/i/nodes/YndMj49yWjlAYoxjtRRogmZwJ3pmz5aA?utm_scene=person_space)
）
    - SWE单节点50镜像并发resume性能（[SWE-Bench 单节点 50 镜像压测测试结果](https://alidocs.dingtalk.com/i/nodes/dQPGYqjpJYZnRbNYCByjDjY08akx1Z5N)）\- 测试开发已完成，数据可视化开发中
    - 深休眠性能基线 - 进行中
        - 嵌套虚拟化机型 - 已完成
            - 单机 1/10/20/40/60/100并发，袋鼠账号/FC 账号，asyncArchive/全量Checkpoint/增量Checkpoint，同ACR（[FC 账号 × ACR — 单机深休眠并发梯度性能测试报告](https://alidocs.dingtalk.com/i/nodes/20eMKjyp810mMdK4HeKD4oOrJxAZB1Gv)）
            - 双机 1/10/20/40/60/100并发，袋鼠账号/FC 账号，asyncArchive/全量Checkpoint/增量Checkpoint，同ACR（[FC 账号 × ACR — 双机 60\+60 并发压测报告](https://alidocs.dingtalk.com/i/nodes/vy20BglGWOxjGpq0CgG0PYXEVA7depqY)）
            - 双机 60/100并发，袋鼠账号/FC 账号，asyncArchive/全量Checkpoint/增量Checkpoint，双ACR（[双机 60\+60 并发：单 ACR vs 双 ACR 对比分析报告](https://alidocs.dingtalk.com/i/nodes/2Amq4vjg89jyZdNnCmxKk2MmW3kdP0wQ)）
        - 非嵌套虚拟化机型 - 进行中
            - 单机 60 并发、100 并发 - 全量 checkpoint，同 ACR
            - 双机 60 并发、100 并发 - 全量 checkpoint，同 ACR
            - 双机 60 并发、100 并发 - 全量 checkpoint，双 ACR
    - 深休眠唤醒性能基线 - 开发中
- **新增问题**
    - 本月新增问题25个（[aone](https://project.aone.alibaba-inc.com/v2/project/2155383/bug#viewIdentifier=5dfb195e2e2b84f6b2f24718&initFilterParams=%5B%7B%22fieldIdentifier%22:%22creator%22,%22value%22:%5B%22445571%22,%22526582%22%5D,%22toValue%22:null,%22operator%22:%22CONTAINS%22%7D,%7B%22fieldIdentifier%22:%22gmtCreate%22,%22value%22:%5B%222026-07-01%2023:59:59%22%5D,%22toValue%22:null,%22operator%22:%22MORE_THAN%22%7D%5D)）
        - 高优先级19个，低优先级6个

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：

[https://project.aone.alibaba-inc.com/v2/project/2155383/bug/84794048#](https://project.aone.alibaba-inc.com/v2/project/2155383/bug/84794048#) 《\[microsandbox\] cpu template ali.nest.ebm-ecs.g8i.toml  在ebm build的template ，在嵌套虚拟化上resume 报错Err(LoadSnapshot(RestoreVmState(VcpuState(Vcpu(VcpuFd(Error(22)))))))》阻塞730发布



KR4: 构建Agent的细粒度的网络行为审计和防护能力，覆盖公网流量、vpc流量，支持L4/L7多种主流协议 @侯志远(至原)

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>
- IP/域名访问控制完成开发、联调，线上灰度发布中；
- Http/Https的domain rule支持（访问凭证）完成开发； 联调中；
- 沙箱实例的L4访问，完成联调，线上灰度发布中；

<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>
- <span style="color: #f33e3e;">\*请填写7月业务重要信息</span>

<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>

<span style="color: rgb(20, 20, 20);">待填写</span>

<span style="color: rgb(20, 20, 20);">**关键进展**</span>：

待填写

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
- <span style="color: #f33e3e;">\*待填写</span>



KR5: 【挑战型】联合云安全和产品团队，探索构建Agent整体安全防护能力，落地成为Rund Agent Sandbox标准化安全解决方案 @冯世舫(旭彤)@常冰(凝江)

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>
    - <span style="color: rgb(20, 20, 20);">（630）完成Agent Sandbox场景的安全攻击面分析，调研业内的相关安全防御手段（完成）</span>
    - <span style="color: rgb(20, 20, 20);">（730）梳理RunD Agent Sandbox的技术架构，明确安全模型和安全边界， 完成安全评审（资料整理80%， 待评审）</span>
    - <span style="color: rgb(20, 20, 20);">（815）联合云安全和产品图队， 确定RunD Agent Sandbox需要支持的安全能力 完成需求拆解（TODO）</span>

<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>

<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>

<span style="color: rgb(20, 20, 20);">**无**</span>

<span style="color: rgb(20, 20, 20);">**关键进展**</span>：
- <span style="color: rgb(20, 20, 20);">**支持安骑士网络防御功能， 满足Agent Sandbox场景（同舟、ACK）的使用需求**</span>
    - <span style="color: rgb(20, 20, 20);">**全部功能开发完成， 同舟环境联调中。 目前遗留一个vebpf功能问题待查**</span>
- <span style="color: rgb(20, 20, 20);">**CVE 处理**</span>
    - 和安全同学确认了RunD CVE 处理流程[RunD CVE 处理流程](https://alidocs.dingtalk.com/i/nodes/MNDoBb60VLYDGNPytmd0djX9JlemrZQ3?utm_scene=team_space)

    - 7月份共计完成 9个CVE 漏洞治理， 目前遗留2个漏洞未处理完成
        - device cgroup ： 智算和cgroup V2场景还没有完成修复
        - vsock 提权： 权限kubeagent 等场景还没有修复完成（依赖ASI侧修复）
- AI Agent场景 安全模型分析
    - [Agent Sandbox安全相关梳理](https://alidocs.dingtalk.com/i/nodes/gvNG4YZ7Jnxop15OCNvR22bwW2LD0oRE)
（包含rund 落地路线roadmap）
    - [RunD MicroSanbox 架构设计和安全模型分析](https://alidocs.dingtalk.com/i/nodes/7NkDwLng8Za7QYkeHNgk0zpPJKMEvZBY?utm_scene=team_space)
（整理中， 预计本周完成并发起安全评审）
- 

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
    - 无

### **【预研】探索下一代轻量化 语言运行时级Agent Sandbox 技术路径， 完成可行性技术验证** @刘起明(寒枝)

【项目跟踪 · 寒枝】轻量化 Agent Sandbox 预研（评估框架\+原型选型、性能对比报告、技术演进路线图）已纳入 AI Agent可观测性项目跟踪文档（财年口径，FY27S2 为主）。详见：https://alidocs.dingtalk.com/i/nodes/YndMj49yWjlAYoxjtRgo4rrvJ3pmz5aA

面向未来轻量化、安全隔离需求，开展关键技术预研

KR1: 定义轻量 Agent Sandbox 的核心评估指标（启动、内存、隔离性、兼容性），建立原型选型框架，并输出性能对比报告

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>
- 待补充

<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>
- <span style="color: #f33e3e;">\*请填写7月业务重要信息</span>

<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>

<span style="color: rgb(20, 20, 20);">待填写</span>

<span style="color: rgb(20, 20, 20);">**关键进展**</span>：

待填写

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
- <span style="color: #f33e3e;">\*待填写</span>



KR2: 输出 1 份《轻量化 Agent Sandbox 技术演进路线图》，包含至少 2 种备选方案（如 WASM、unified runtime）及适用场景建议

<span style="color: rgb(20, 20, 20);">**当前里程碑：**</span>
- ![image.png](https://alidocs2.oss-cn-zhangjiakou.aliyuncs.com/res/2M9qP5kJPJGVDO01/img/1a318e39-b927-40f1-95e0-3274f0da0b6d.png?Expires=1786118349&OSSAccessKeyId=LTAI5tKTjg4Kq1HCdBJ8qpSp&Signature=JduLpbR3ZiYxRh5bYcvvaxP9Bto%3D "")



<span style="color: rgb(20, 20, 20);">**业务重要信息同步：**</span>
- <span style="color: #f33e3e;">\*请填写7月业务重要信息</span>

<span style="color: rgb(20, 20, 20);">**数据与指标：**</span>

<span style="color: rgb(20, 20, 20);">待填写</span>

<span style="color: rgb(20, 20, 20);">**关键进展**</span>：

待填写

<span style="color: rgb(20, 20, 20);">**风险与阻塞**</span>：
- <span style="color: #f33e3e;">\*待填写</span>


---
> 采集方式：`dws drive search` + `dws doc read`（只读）。
