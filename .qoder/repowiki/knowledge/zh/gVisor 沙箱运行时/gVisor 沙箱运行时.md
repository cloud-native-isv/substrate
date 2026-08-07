---
kind: external_dependency
name: gVisor 沙箱运行时
slug: gvisor-沙箱运行时
category: external_dependency
scope:
    - '**'
---

Agent Substrate 默认使用 gVisor 作为沙箱运行时，通过 runsc 二进制提供内核级隔离。每个 Worker Pod 内运行 ateom-gvisor 进程，调用 runsc 执行 checkpoint/restore 操作。gVisor 后端要求特定的 runsc 版本（需要 --allow-connected-on-save 标志）以支持网络恢复。gVisor 模板限制为单个 DurableDir 卷，因为 gVisor 当前只接受单个可挂载卷。Checkpoint 机制捕获进程树状态，支持跨 Worker 迁移。