---
kind: frontend_style
name: 前端样式系统：无独立前端，仅演示用内联 HTML/CSS
slug: frontend_style
category: frontend_style
scope:
    - '**'
---

该仓库是一个基于 Kubernetes CRD 的 Agent 运行时平台（Go 后端），整体不包含任何独立的前端工程、CSS 框架或设计令牌系统。唯一涉及 UI 的代码位于 `demos/claude-code-multiplex/ui/` 目录下的一个极简演示页面：

- `index.html`：单文件 HTML，内嵌 `<style>` 块使用 CSS 变量定义深色主题（`--bg`、`--panel`、`--accent`、`--green`、`--yellow`、`--red` 等），通过 `:root` 声明全局色板，配合 CSS Grid/Flexbox 布局与 `@media` 响应式断点（720px）。
- `server.go`：纯 Go stdlib HTTP 服务器，静态服务 `index.html`，并通过 `/api/pods`、`/api/actors`、`/api/logs/*`、`/api/task-status`、`/api/give-task` 等 JSON 接口提供数据，前端通过 `fetch` 轮询刷新。

该 UI 是演示用途的单页应用，没有构建步骤、没有组件库、没有样式预处理、没有响应式框架（如 Tailwind），也没有跨组件共享的主题配置。整个项目的其余部分均为 Go 后端代码（ateapi、atecontroller、atelet、atenet、ateom-* 等二进制），不涉及任何前端样式体系。