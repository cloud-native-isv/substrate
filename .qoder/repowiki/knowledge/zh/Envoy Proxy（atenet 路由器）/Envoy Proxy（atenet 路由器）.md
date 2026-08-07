---
kind: external_dependency
name: Envoy Proxy（atenet 路由器）
slug: envoy-proxy-atenet-路由器
category: external_dependency
scope:
    - '**'
---

Agent Substrate 使用 Envoy Proxy 作为 atenet 路由器，提供 DNS 解析、HTTP 路由和请求泊车功能。Envoy 通过 ext_proc 外部处理器拦截 HTTP 请求，从 Host 头提取 Actor 名称和 Atespace，然后调用 Control Plane 进行 ResumeActor 操作。当 WorkerPool 饱和时，路由器会"泊车"请求而不是立即返回 503，使用指数退避重试直到 Actor 可用。Envoy 还建立 mTLS 隧道到 Worker 的 atunnel 监听器，转发请求到实际的 Actor 进程。