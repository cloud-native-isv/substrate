---
kind: external_dependency
name: Google Cloud Storage（Actor 快照持久化）
slug: google-cloud-storage-actor-快照持久化
category: external_dependency
scope:
    - '**'
---

Agent Substrate 使用 Google Cloud Storage 作为 Actor 快照的主要持久化后端。快照包含进程内存和文件系统状态，用于实现 sub-second 的 suspend/resume 操作。系统支持 GCS 和 S3 两种对象存储后端，通过 ActorTemplate 的 snapshotsConfig 指定存储位置。atelet 组件负责在 Worker Pod 和对象存储之间流式传输快照文件。GCS 集成通过 google-cloud-go/storage SDK 实现，支持 IAM 认证和加密。