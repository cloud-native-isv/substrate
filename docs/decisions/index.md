# Architecture Decision Records（ADR 索引）

本目录 append-only：ADR 只追加、只以状态流转标注（Proposed → Accepted → Deprecated / Superseded by），
绝不重写历史。命名 `NNNN-slug.md`，编号连续无空洞；新 ADR 复制 [template.md](template.md) 并在下表登记。

| 编号 | 标题 | 状态 | 日期 |
|------|------|------|------|
| [0002](0002-two-tier-supervisor-worker-sandbox.md) | 两层沙箱模型：Supervisor Sandbox（Kubernetes+Substrate+Containerd+runc/rund，低频生命周期+高稳定性+休眠唤醒）+ Worker 面（e2b+自研管控+WebAssembly，高频创建/隔离/审计/Capability-based security） | Proposed | 2026-09-16 |
