---
id: "20260811T182837Z-skill-create-docs"
unit_id: "skill:create-docs"
unit_type: "skill"
run_id: "presentation-20260811T181311Z"
scope: "local"
partial: false
created: "2026-08-11T18:28:37Z"
summary: "呈现层收敛：scaffold 9 个文件后 hugo 0.141.0 建站成功（29 页 / 74 静态文件 / rc=0 / 零 warning），mount-not-copy、index.md 原名、零churn、kept 用户改动等不变量全部保持。但首次构建产物存在 3 类真实发布缺陷，均在构建成功后靠人工全站爬链才发现：根级 10 篇 upstream md 未挂载（含死链）、无 fron"
---

## Review
呈现层收敛：scaffold 9 个文件后 hugo 0.141.0 建站成功（29 页 / 74 静态文件 / rc=0 / 零 warning），mount-not-copy、index.md 原名、零churn、kept 用户改动等不变量全部保持。但首次构建产物存在 3 类真实发布缺陷，均在构建成功后靠人工全站爬链才发现：根级 10 篇 upstream md 未挂载（含死链）、无 frontmatter 导致导航/列表标题空白、raw-HTML 图片 16 处 404。修复均落在用户可拥有区（hugo.toml 托管块之外 + layouts），未改动任何 Markdown。

## Optimization Points
- scaffold-hugo.py 的 discover() 只遍历 docs/ 子目录（第60-61行 `if not child.is_dir(): continue`），根级 .md 从不挂载。本项目 docs/ 根有 10 篇 upstream 文档，首建后全部缺页且指向它们的相对链接全成死链。建议 discover() 增加根级 `includeFiles=["/*.md"]` 挂载，或在 check 输出中显式告警"根级 md 未纳入发布范围"。
- 期望态要求"Markdown 保持纯净（无 frontmatter）"，但 scaffold 的 layouts 用 .Title/.LinkTitle 取标题 —— 二者直接冲突：导航、section cards、page list 的链接文字全为空白。建议 assets/hugo/layouts 自带一个 H1 兜底 partial（本次已在项目侧补 partials/page-title.html），使"无 frontmatter"与站点可读性不矛盾；注意 .Fragments 的 heading 已 HTML 转义，需 htmlUnescape 否则出现 &amp;amp;。
- render hooks 只覆盖 Markdown 语法的链接/图片；文档为实现"点图放大"普遍使用 raw HTML <a><img>，这类相对路径不被 hook 改写，发布后 404（本次 16 处）。建议 check 动作扫描 md 中的 raw-HTML 相对 src/href 并报告需要的 static 镜像挂载，而非让用户在构建后自行发现。
- check/build 未做产物自检：本次全部缺陷都是构建成功（rc=0、无 warning）之后靠人工爬链发现的。建议 build 后可选做一次内部链接/资产存在性爬取并计入 clean 判定。
