# Architecture Decision Records（ADR 索引）

本目录 append-only：ADR 只追加、只以状态流转标注（Proposed → Accepted → Deprecated / Superseded by），
绝不重写历史。命名 `NNNN-slug.md`，编号连续无空洞；新 ADR 复制 [template.md](template.md) 并在下表登记。

| 编号 | 标题 | 状态 | 日期 |
|------|------|------|------|
| [0002](0002-two-tier-supervisor-worker-sandbox.md) | 两层沙箱模型（substrate 词汇归一，去 supervisor 语义）：Tier-1 Worker Pod 面（Kubernetes+Substrate+Containerd+runc/rund，sandbox class wasm 的 kata 形态，低频+高稳定+休眠唤醒）+ Tier-2 Actor 工作负载面（e2b+自研管控+wasm workload：高频创建/隔离/审计/Capability-based security）；不新增 SandboxClass enum | Proposed | 2026-09-16 |
| [0003](0003-wasm-host-function-security.md) | Wasm Host Function 安全模型（runc 基线的前置硬化）：最小 host function 面（allowlist 默认拒绝）+ ABI 边界校验契约 + capability 唯一执行点 + 内存安全纪律 + 出口代理隔离 + 凭据不落 wasm + **三环外部可信审计**（采纳既有方案 v2.1：Ring 0 host function 边界 / Ring 1 应用层 / Ring 2 内核 eBPF，含 Ring 2 × runc/rund 交互）+ **硬化/验证门禁（runc 准入）** + fail-closed 降级；落实 ADR 0002「wasm 隔离可信度」开放问题 | Proposed | 2026-09-18 |
| [0004](0004-runc-worker-pod-hardening.md) | runc Worker Pod 安全硬化（纵深防御外层 + runc 准入判据）：补全 ADR 0003 内层（wasm 预防）之外的 **substrate 外层遏制**——四层纵深栈（L1 预防/L2 容器·pod 遏制/L3 节点·集群爆炸半径/L4 检测，分属两仓）+ L2 硬化（drop ALL·allowPrivilegeEscalation:false·钉紧 seccomp·只读 rootfs·**userns**·hostPath 收窄）+ L3（专属节点·NetworkPolicy 东西向默认拒绝·出网仅代理）+ L4 Ring 2 强制 + 逃逸节点级响应 + **runc 准入判据（D8 门禁 ∧ L2 ∧ L3 ∧ Ring 2，操作化 ADR 0002 D1 判据）**；兑现 runc 密度/成本/兼容价值 | Proposed | 2026-09-18 |
| [0005](0005-rust-agent-execution-body.md) | 执行体形态转向：**Rust agent + 标准 WASI SDK + 定制 host function（安全优先于兼容）**——sandbox 执行体不再以 CPython-wasm 作主体（兼容优先），改为 safe Rust agent 编译为 wasm 作主体、标准 WASI 定义最小导入面、定制 host function 作为 agent 工具（capability 唯一执行点）；**CPython 降级为 `python_exec` 工具**（保留、受 capability+审计+子沙箱约束，非主体）。收益：guest TCB 最小化 + 内存安全贯穿 guest + 工具 host 强制 + 出口统一代理 → 兑现 ADR 0003 D1/D3/D4/D5、强化 ADR 0004 L1（runc 准入更易满足）；具体化 ADR 0002 D2「wasm agent program」 | Proposed | 2026-09-18 |
| [0006](0006-security-spectrum-model.md) | 安全光谱模型：三维滑标（**安全是光谱非非黑即白**）——① pod runtime class（runc↔rund）× ② agent 程序主体（cpython-wasm 解释执行↔rust-wasm 编译执行）× ③ 暴露的工具/host function 多寡；三维位置组合 = 复合安全等级，每维有中间位（runsc / rust+python_exec 混合 / 受控扩张工具面），按租户威胁模型逐池取舍。统一 ADR 0002 D1（①）/0003 D1（③）/0004 D6（runc 准入=光谱切片）/0005（②，转向=默认位）；含跨维补偿不变式（① 越 runc 则 ②③ 须越安全端）+ 策略档位（hardened/balanced/density/compat）+ 可声明可核查 | Proposed | 2026-09-18 |
