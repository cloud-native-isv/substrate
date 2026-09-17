---
id: "20260911T094321Z-skill-aliyun-workspace"
unit_id: "skill:aliyun-workspace"
unit_type: "skill"
run_id: "ack-msafe8-takeover-20260911"
scope: "local"
partial: false
created: "2026-09-11T09:43:21Z"
summary: "接管 ACK cluster-msaFE8：preflight 通过；13 个配置账号逐一探测 404/403 后借本地云资源清单定位归属账号 1094633731167611 与资源组 rg-acfmwbuu2cojk6y；该账号唯一凭证被资源组级 ImplicitDeny 拦截（直连与 AssumeRole 均 403），按兜底条款转控制台剪贴板路径完成 kubeconfig 落盘与 kube"
---

## Review
接管 ACK cluster-msaFE8：preflight 通过；13 个配置账号逐一探测 404/403 后借本地云资源清单定位归属账号 1094633731167611 与资源组 rg-acfmwbuu2cojk6y；该账号唯一凭证被资源组级 ImplicitDeny 拦截（直连与 AssumeRole 均 403），按兜底条款转控制台剪贴板路径完成 kubeconfig 落盘与 kubectl 接管；巡检经 kubectl 完成，云侧项如实标注受限。

## Optimization Points
- aliyun-account-table --plain 会把 AK/SK 明文注入 agent 上下文；账号发现阶段应只用脱敏投影（UID+名称），完整凭证仅在脚本内经 aliyun_account data 取用
- practices/ack.md 增补 pitfall：404 ErrorClusterNotFound=账号不对；403 ResourceGroupLevelIdentityBasedPolicy ImplicitDeny 且 list-cluster 为空=账号对但凭证被资源组授权隔离；并提示核对账号目录下兄弟 secret 目录（本次发现 liuqiming.lqm 目录误存其他子账号 AK）
