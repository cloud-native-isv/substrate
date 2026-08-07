---
kind: external_dependency
name: Kata Containers + Cloud Hypervisor（microvm 沙箱）
slug: kata-containers-cloud-hypervisor-microvm-沙箱
category: external_dependency
scope:
    - '**'
---

Agent Substrate 支持通过 Kata Containers 和 Cloud Hypervisor 实现 micro-VM 沙箱。每个 Worker Pod 运行 ateom-microvm 进程，管理基于 KVM 的轻量级虚拟机。checkpoint/restore 通过捕获内存快照和使用 userfaultfd 内存需求分页实现。容器 rootfs 写入通过 tmpfs overlay 捕获，DurableDir 卷通过 virtio-fs 共享。microvm 模板支持多个 DurableDir 卷（无额外设备成本），这是与 gVisor 的关键区别。快照包含 VM 内存状态，恢复时通过内存映射快速启动。