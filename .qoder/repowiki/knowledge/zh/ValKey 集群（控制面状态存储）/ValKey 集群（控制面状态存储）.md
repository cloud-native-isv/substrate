---
kind: external_dependency
name: ValKey 集群（控制面状态存储）
slug: valkey-集群-控制面状态存储
category: external_dependency
scope:
    - '**'
---

Agent Substrate 使用 ValKey 集群作为控制面的高性能状态存储，替代 etcd 来管理 Actor/Worker 的高频状态。系统通过 go-redis v9 客户端连接，启用 TLS 和集群模式，6 个节点组成集群，端口 6379（数据）和 16379（总线）。ate-api-server 通过环境变量注入集群地址、CA 证书和 IAM 认证配置。ValKey 集群由 StatefulSet 部署，包含初始化 Job 自动创建集群拓扑。所有内部通信使用 mTLS 认证，包括 ate-api-server 到 ValKey 的连接。