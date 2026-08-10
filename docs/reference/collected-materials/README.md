# substrate 资料库（final 交付版）

> 项目 scope：Google Agent Substrate 开源项目主题（K8s agent 沙箱/运行时底座、gVisor/microVM、挂起恢复、高密度多路复用、E2B 兼容）+ 竞对开源项目（Kimi AgentENV / E2B / Daytona / OpenSandbox 等）。
> 来源：经 material-auditor 审核放行的 24 篇资料（internal 18：ATA 14 + 语雀 2 + 钉钉文档 2；public 6），采集日期 2026-08-07。
> 每篇文档开头保留原始来源元信息块，正文未做改写。

## 来源域标注（internal vs public）

- **internal（内网）**：`01-internal-project-progress/`、`02-internal-platform-practice/` — 来源域为 ATA（ata.atatech.org）、语雀/KM（aliyuque.antfin.com、km.aone.alibaba-inc.com）、钉钉文档（alidocs.dingtalk.com），含团队自有进展纪要，注意内网保密要求。
- **public（公开竞对）**：`03-public-competitor-landscape/` — 来源域为 GitHub README 与公开技术媒体，可自由对外引用。

## 分类结构

| 目录 | 主题 | 来源域 | 篇数 | 说明 |
|------|------|--------|------|------|
| `01-internal-project-progress/` | 项目直接相关（团队进展） | internal | 4 | MicroSandbox/FC 团队 7 月总结、8.05 周会、E2B 兼容对齐会纪要、eleme xboot 沙箱 SDK |
| `02-internal-platform-practice/` | 内部平台实践与技术调研 | internal | 14 | 集团内 Agent 沙箱平台落地（ACS/AgentRun/FC/OpenKruise/OpenSandbox/OpenShell）+ 隔离技术调研 + E2B 用法 + 竞对分析文（内部视角） |
| `03-public-competitor-landscape/` | 竞对分析（公开资料） | public | 6 | AgentENV(Kimi)/E2B/Daytona/OpenSandbox 官方 README + 发布报道 + 五平台横评 |

## 全部文档清单（24 篇）

### 01-internal-project-progress/ — 项目直接相关（internal）

| 文件名 | 标题 | 一句话摘要 | 来源 |
|--------|------|-----------|------|
| agent-sandbox-team-july-summary.md | Agent Sandbox 7月总结 | runD MicroSandbox 全面落地、深休眠/snapshot、E2B 兼容、CVE 治理与规模指标 | [钉钉文档](https://alidocs.dingtalk.com/i/nodes/a9E05BDRVQRkezKGCDP561b4J63zgkYA) |
| agent-sandbox-team-weekly-0805.md | Agent Sandbox 8.05 周会纪要 | 动态挂载方案、深休眠对接、envd 角色结论、计费与网络进展 | [钉钉文档](https://alidocs.dingtalk.com/i/nodes/KGZLxjv9VGkoG9YwH6aBOn9BV6EDybno) |
| fc-sandbox-e2b-compat-alignment-20260529.md | FC sandbox 功能对齐梳理 20260529 | MicroSandbox×FC 团队 E2B 兼容对齐会纪要：SDK/CLI 功能逐项对齐表、参数映射与一期交付范围 | [语雀](https://aliyuque.antfin.com/kangaroo-container/di61r8/og6kobitydlapy8o) |
| code-sandbox-eleme-xboot-ai.md | 代码沙箱 Code Sandbox (eleme xboot) | eleme xboot-ai-sandbox 模块架构：SandboxToolProvider 双策略 + Aone Sandbox SDK 集成（块拼接摘要） | [KM](https://km.aone.alibaba-inc.com/#/repo/71835/doc/46aa4a99-eac2-4e9e-b353-fb6358f7a15c) |

### 02-internal-platform-practice/ — 内部平台实践与技术调研（internal）

| 文件名 | 标题 | 一句话摘要 | 来源 |
|--------|------|-----------|------|
| acs-container-agent-sandbox-infrastructure.md | 基于容器构建的 AI Agent Sandbox 基础设施落地实践 | 安全隔离/极致弹性/状态保持恢复，OpenKruise Agents 开源标准化 | [ATA](https://ata.atatech.org/articles/11020533607) |
| agentrun-serverless-agent-sandbox.md | 探秘 AgentRun：Serverless AI Agent 沙箱工程化之路 | 函数计算 AgentRun：会话亲和/安全隔离/状态保持/按需计费 | [ATA](https://ata.atatech.org/articles/11020572422) |
| acs-agent-sandbox-openclaw-deploy.md | 一键部署 OpenClaw：ACS Agent Sandbox 企业级应用 | 按需休眠/秒级唤醒、K8s 集成与钉钉应用对接 | [ATA](https://ata.atatech.org/articles/11020589221) |
| fc-sandbox-glm5-launch-support.md | FC Sandbox 助力智谱 GLM-5 宣发 | 高并发实例管理、镜像缓存、上下文存储、快照恢复关键技术 | [ATA](https://ata.atatech.org/articles/11020590023) |
| openkruise-agents-v030-release.md | OpenKruise Agents v0.3.0 发布 | 批量升级、原地 CPU 变配、多租户隔离，K8s Agent 沙箱可运维 | [ATA](https://ata.atatech.org/articles/11020656441) |
| k8s-agent-sandbox-opensandbox-plan.md | 智能体沙箱环境方案 | 基于 OpenSandbox 的 K8s 方案：资源池化、只读挂载、安全隔离与高效并发 | [ATA](https://ata.atatech.org/articles/11020645223) |
| opensandbox-security-open-practice.md | OpenSandbox：AI 场景下的安全基石与开放实践 | 安全容器隔离/网络管控/资源限制，Protocol First 与 OSEP 提案机制 | [ATA](https://ata.atatech.org/articles/11020589292) |
| agent-sandbox-isolation-tech-research.md | Agent Sandbox 调研 | Firecracker/gVisor/KataContainers 隔离技术对比与投入方向 | [ATA](https://ata.atatech.org/articles/11020628519) |
| agent-sandbox-tech-product-overview.md | AI Agent 沙箱技术与产品 | 五大隔离层级、安全风险、产品生态（E2B/OpenSandbox 等）与选型策略 | [ATA](https://ata.atatech.org/articles/11020604092) |
| mainstream-sandbox-services-hands-on.md | Agent 的执行引擎：主流 Sandbox 服务上手初探 | 腾讯云/火山引擎/阿里云 + E2B/Daytona/OpenSandbox 上手横评 | [ATA](https://ata.atatech.org/articles/11020602075) |
| e2b-core-capabilities-guide.md | E2B 使用篇（上）：核心能力 | 命令执行/文件管理/生命周期/CodeInterpreter 等 SDK 用法 | [ATA](https://ata.atatech.org/articles/11020667629) |
| nvidia-openshell-architecture-analysis.md | NVIDIA OpenShell 深度架构分析 | Linux 内核四层沙箱（netns+Landlock+seccomp+权限降级）+ 声明式 YAML 策略 | [ATA](https://ata.atatech.org/articles/11020603781) |
| claude-managed-agents-analysis.md | Claude Managed Agents 全面解析 | Session/Harness/Sandbox 三层解耦架构的托管式 Agent 平台 | [ATA](https://ata.atatech.org/articles/11020606138) |
| agent-env-snapshot-rollback.md | Agent 时代，最先崩的不是模型，是环境 | 环境破坏与状态回滚问题：环境快照/恢复到之前状态继续工作的能力需求 | [ATA](https://ata.atatech.org/articles/11020674849) |

### 03-public-competitor-landscape/ — 竞对分析（public）

| 文件名 | 标题 | 一句话摘要 | 来源 |
|--------|------|-----------|------|
| agentenv-kimi-readme.md | AgentENV (kvcache-ai / Kimi) README | Firecracker microVM、Kimi K3 agentic RL 底座、E2B 兼容 API、<50ms 恢复 | [GitHub](https://github.com/kvcache-ai/AgentENV) |
| agentenv-release-announcement.md | AgentENV 开源发布报道 | marktechpost 报道（正文截断已标注）：时间线与官方链接 | [marktechpost](https://www.marktechpost.com/2026-07-27/kimi-ai-and-kvcache-ai-open-sources-agentenv/) |
| e2b-readme.md | E2B README | 开源 agent 沙箱云，Python/JS SDK、Code Interpreter 与安全沙箱能力 | [GitHub](https://github.com/e2b-dev/E2B) |
| daytona-readme.md | Daytona README | <90ms 沙箱；含 2026-06 起核心开发转入私有仓库的停维公告 | [GitHub](https://github.com/daytonaio/daytona) |
| opensandbox-alibaba-readme.md | OpenSandbox（阿里巴巴开源）README | AI 沙箱平台架构、安全容器隔离与生态 | [GitHub](https://github.com/opensandbox-group/OpenSandbox) |
| sandbox-platforms-comparison.md | 沙箱平台五家横评 | E2B/Daytona/Modal/Cloudflare/Vercel：隔离技术、启动/恢复性能、计费与选型建议 | [developersdigest](https://www.developersdigest.tech/blog/ai-agent-code-sandbox-comparison-2026) |

## 使用建议

- 了解项目现状：`01-internal-project-progress/` 四篇即团队一手进展（7 月总结 + 8.05 周会 + E2B 兼容对齐表），是理解 MicroSandbox/FC 现状的最短路径。
- 对标 Agent Substrate 设计（K8s 底座/挂起恢复/高密度复用/E2B 兼容）：内部看 `02-internal-platform-practice/` 的 OpenKruise Agents、ACS/AgentRun 落地文；外部看 `03-public-competitor-landscape/` 的 AgentENV（Firecracker + <50ms 恢复）与横评文。
- 注意 `02-internal-platform-practice/` 中的竞对分析文（NVIDIA OpenShell、Claude Managed Agents、主流服务横评）为**内部作者视角**的二手分析，与 `03-public-competitor-landscape/` 的一手公开 README 互为补充，引用结论时注意区分。

> 互参提示：sandbox 项目（`../sandbox/final/`）收录了单机方向的 Agent 沙箱方案与 Firecracker/gVisor 隔离技术深挖，与本项目 `02-internal-platform-practice/` 的隔离技术调研主题互补。
