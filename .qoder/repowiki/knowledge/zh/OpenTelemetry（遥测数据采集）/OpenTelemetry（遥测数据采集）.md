---
kind: external_dependency
name: OpenTelemetry（遥测数据采集）
slug: opentelemetry-遥测数据采集
category: external_dependency
scope:
    - '**'
---

Agent Substrate 全面集成 OpenTelemetry 进行分布式追踪和指标收集。系统使用 otelgrpc 和 otelhttp 中间件自动采集 gRPC 和 HTTP 请求的追踪信息。指标通过 prometheus exporter 暴露，日志通过 OTLP 协议发送到 OpenTelemetry Collector。资源属性包含 k8s.namespace.name、k8s.pod.name、k8s.pod.uid 等元数据。atenet-router 还提供专门的 parking 相关指标，如 atenet.router.parking.active 和 atenet.router.parking.wait.duration。