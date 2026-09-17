# Architecture Decision Records（ADR 索引）

本目录 append-only：ADR 只追加、只以状态流转标注（Proposed → Accepted → Deprecated / Superseded by），
绝不重写历史。命名 `NNNN-slug.md`，编号连续无空洞；新 ADR 复制 [template.md](template.md) 并在下表登记。

| 编号 | 标题 | 状态 | 日期 |
|------|------|------|------|
| [0002](0002-two-tier-supervisor-worker-sandbox.md) | 两层沙箱模型（substrate 词汇归一，去 supervisor 语义）：Tier-1 Worker Pod 面（Kubernetes+Substrate+Containerd+runc/rund，sandbox class wasm 的 kata 形态，低频+高稳定+休眠唤醒）+ Tier-2 Actor 工作负载面（e2b+自研管控+wasm workload：高频创建/隔离/审计/Capability-based security）；不新增 SandboxClass enum | Proposed | 2026-09-16 |
| [0003](0003-wasm-host-function-security.md) | Wasm Host Function 安全模型（runc 基线的前置硬化）：最小 host function 面（allowlist 默认拒绝）+ ABI 边界校验契约 + capability 唯一执行点 + 内存安全纪律 + 出口代理隔离 + 凭据不落 wasm + 全事件审计 + **硬化/验证门禁（runc 准入）** + fail-closed 降级；落实 ADR 0002「wasm 隔离可信度」开放问题 | Proposed | 2026-09-18 |
