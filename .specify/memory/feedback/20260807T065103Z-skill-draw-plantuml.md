---
id: "20260807T065103Z-skill-draw-plantuml"
unit_id: "skill:draw-plantuml"
unit_type: "skill"
run_id: "draw-plantuml-20260807T135500"
scope: "local"
partial: false
created: "2026-08-07T06:51:03Z"
summary: "为 study-project 交付渲染 8 张 UML 图（组件/部署/4 时序/状态机/类图），PNG+SVG+puml 齐全。本地 jar 因容器缺 fontconfig/字体失败，切换技能内置远端服务器（xuanji-plantuml）全部成功；deployment.puml 一次语法错误经修复后通过。Catalyst：应把远端渲染作为容器环境首选路径。"
---

## Review
为 study-project 交付渲染 8 张 UML 图（组件/部署/4 时序/状态机/类图），PNG+SVG+puml 齐全。本地 jar 因容器缺 fontconfig/字体失败，切换技能内置远端服务器（xuanji-plantuml）全部成功；deployment.puml 一次语法错误经修复后通过。Catalyst：应把远端渲染作为容器环境首选路径。

## Optimization Points
- 渲染失败时缺少服务端诊断信息：deployment.puml 因 namespace+artifact 混用被服务器拒绝，脚本只报 "SVG rendering failed"，不得不手工重新 deflate+base64 编码后请求 /txt 端点才拿到真实错误（"Use 'allowmixing'..."）。建议 render-plantuml.sh 在 server 后端渲染失败时自动请求 ${PLANTUML_SERVER}/txt/{enc} 并把错误文本前几行打到 stderr，可大幅缩短排障时间。
