---
id: "20260807T065046Z-skill-study-project"
unit_id: "skill:study-project"
unit_type: "skill"
run_id: "study-upstream-20260807T135500"
scope: "local"
partial: false
created: "2026-08-07T06:50:46Z"
summary: "对 upstream Agent Substrate（约 59k LOC）执行 Standard 模式深度研究：业务语境→6 个并行子代理分域深挖（控制面/节点监督/网络/资源模型/身份CLI观测/部署形态）→三项关键结论代码交叉验证→8 张 PlantUML 图（远端渲染）→三份文档落入项目六型 taxonomy 并更新索引与 xuanji 登记。子代理证据密度高，交叉验证全部通过；本地 jar"
---

## Review
对 upstream Agent Substrate（约 59k LOC）执行 Standard 模式深度研究：业务语境→6 个并行子代理分域深挖（控制面/节点监督/网络/资源模型/身份CLI观测/部署形态）→三项关键结论代码交叉验证→8 张 PlantUML 图（远端渲染）→三份文档落入项目六型 taxonomy 并更新索引与 xuanji 登记。子代理证据密度高，交叉验证全部通过；本地 jar 渲染因容器缺字体失败后切换远端成功。

## Optimization Points
- 交付物位置与项目 docs taxonomy 冲突：skill 硬性要求 $WORK_DIR/docs/overview.md 单文件交付，但本项目刚经 /speckit.docs 建立六型 taxonomy（concepts/reference/...），且宪法原则 III 要求复杂文档拆分。实际落地为 overview.md（主报告）+ concepts/core-concepts.md + reference/actor-lifecycle-flows.md 的文档集。建议 Execution Contract 增加一条：当项目存在既定 docs 规范/分类法时，交付物遵循该分类法（overview 可保留为主报告入口），本地约定优先于固定路径。
- 图表渲染环境预检缺失：Phase 8 委托 draw-plantuml 渲染，在无 fontconfig/字体的容器里本地 jar 路径必然失败，浪费一轮排障。建议在 Phase 5 规划图表时做渲染后端预检（远端 server 可达性优先），或在 skill 文档中明示"容器环境优先远端渲染"。
