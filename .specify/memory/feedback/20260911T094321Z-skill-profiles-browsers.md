---
id: "20260911T094321Z-skill-profiles-browsers"
unit_id: "skill:profiles-browsers"
unit_type: "skill"
run_id: "ack-msafe8-takeover-20260911"
scope: "local"
partial: false
created: "2026-09-11T09:43:21Z"
summary: "cs.console.aliyun.com 新建 capability ack-copy-kubeconfig（absent→exploration）；实测三条自动化路径均不可行（kangaroo/test 被占无 CDP、MCP 浏览器无登录态、本机无 Playwright），改零泄露手工复制兜底并沉淀 dom 记录；capability 保留 exploration 并记录 blocker。"
---

## Review
cs.console.aliyun.com 新建 capability ack-copy-kubeconfig（absent→exploration）；实测三条自动化路径均不可行（kangaroo/test 被占无 CDP、MCP 浏览器无登录态、本机无 Playwright），改零泄露手工复制兜底并沉淀 dom 记录；capability 保留 exploration 并记录 blocker。

## Optimization Points
- 沉淀「手工复制+剪贴板交接」兜底模式：凭证类页面在无可附加登录态实例且无 Playwright 时，由用户点击复制、agent 经 pbpaste 落盘，内容零进入上下文
- 探索前应做主机能力预检（Playwright/CDP 可附加性），避免承诺 Tier-3 自动化后中途降级
