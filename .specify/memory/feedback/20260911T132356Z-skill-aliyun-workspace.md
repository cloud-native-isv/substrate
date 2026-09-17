---
id: "20260911T132356Z-skill-aliyun-workspace"
unit_id: "skill:aliyun-workspace"
unit_type: "skill"
run_id: "ack-msafe8-images-20260911"
scope: "local"
partial: false
created: "2026-09-11T13:23:56Z"
summary: "为集群换镜像需推 ACR EE：cws 的 acr list-instances 在错误路径抛 KeyError 掩盖真实报错；无 GetAuthorizationToken 命令，用 cws venv SDK 脚本取临时凭据完成 docker login 与推送；内部 registry 推送可达但集群不可拉（实测 ImagePullBackOff），ACR EE vpc 域才是集群可用路径。"
---

## Review
为集群换镜像需推 ACR EE：cws 的 acr list-instances 在错误路径抛 KeyError 掩盖真实报错；无 GetAuthorizationToken 命令，用 cws venv SDK 脚本取临时凭据完成 docker login 与推送；内部 registry 推送可达但集群不可拉（实测 ImagePullBackOff），ACR EE vpc 域才是集群可用路径。

## Optimization Points
- aliyun-acr-list-instances 异常路径崩溃（KeyError 'Message'，错误处理假设响应必含 Message），且命令面缺 ACR EE GetAuthorizationToken 取临时推送凭据的能力；本次靠 cws venv 内 alibabacloud_cr20181201 SDK 手写脚本绕过，建议补 aliyun-acr-ee-token 命令并修错误路径
