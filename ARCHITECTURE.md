# Architecture Overview

> 薄根入口：一页架构总览；细节在 `docs/`，见文末链接。

Xuanji Substrate 是 Agent Substrate 的定制 fork（`main` 镜像 upstream，`xuanji` 承载定制开发）：
基于 Kubernetes 的高密度 agent 工作负载运行时，把大量 "actor"（agent 应用）映射到较小的
就绪 "worker"（Pod）池上，利用 agent 大部分时间空闲的特性实现重度复用，并以全状态快照
达成亚秒级 suspend/resume，沙箱隔离采用 gVisor/microVM。

## 系统组成

- **控制面**（ate-api-server、atecontroller）：把 Kubernetes 移出关键路径；声明式配置走 CRD，实例动态状态走数据库（Redis/Valkey）
- **节点监督**（atelet + ateom）：每节点 actor 生命周期；沙箱类 gvisor / microvm / wasm，各有独立 ateom 实现
- **网络**（atenet DNS + atunnel）：agent 感知路由、请求 parking
- **辅助组件**：podcertcontroller（证书）、kubectl-ate（CLI）

## Xuanji 定制扩展（upstream 无）

- **Wasm 沙箱类**：外部运行时 ateom-wasmd（sandbox 仓维护）；改动登记见 [xuanji.md](xuanji.md)
- **E2B 协议面**：`cmd/e2bgw` E2B REST → `ateapipb.Control` 翻译网关；验收资产 `contrib/e2b-e2e/`

## 详细文档

| 文档 | 内容 |
|---|---|
| [docs/overview.md](docs/overview.md) | **upstream 架构深度分析**（组件架构图/部署图/设计权衡/已知缺口） |
| [docs/concepts/core-concepts.md](docs/concepts/core-concepts.md) | 核心概念详解（Actor/快照体系/生命周期/停车） |
| [docs/reference/actor-lifecycle-flows.md](docs/reference/actor-lifecycle-flows.md) | 关键流程精确参考（时序图 + 契约速查） |
| [docs/architecture.md](docs/architecture.md) | 完整系统架构（upstream 原始文档） |
| [docs/api-guide.md](docs/api-guide.md) | API 配置参考 |
| [docs/api-style-guide.md](docs/api-style-guide.md) / [docs/code-style-guide.md](docs/code-style-guide.md) | API / 代码风格指南 |
| [docs/observability.md](docs/observability.md) | 日志 / 指标 / 追踪 |
| [docs/threat-model.md](docs/threat-model.md) | 威胁模型 |
| [docs/roadmap.md](docs/roadmap.md) | 路线图 |
| [docs/glossary.md](docs/glossary.md) | 术语表 |
| [docs/decisions/](docs/decisions/index.md) | 架构决策记录（ADR） |
