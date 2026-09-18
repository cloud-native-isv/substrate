# 两层架构 POC 验证报告（最简形态：runc + cpython-wasm + simple python agent）

> 状态：**POC 通过**（设计可行性已验证）
> 日期：2026-09-18
> 测试环境：ACK 集群 **cluster-msaFE8**（`ce1fd092f0ce94243b17b9dab1349826b`，cn-hangzhou，K8s 1.36.1-aliyun.1，单 worker 节点，containerd 2.1.9）
> 目标：以最简化形式 **runc + cpython-wasm + simple python agent** 验证两层沙箱设计（[ADR 0002](../decisions/0002-two-tier-supervisor-worker-sandbox.md)–[0006](../decisions/0006-security-spectrum-model.md)）可行
> 复现资产：`contrib/e2b-e2e/poc_two_tier_agent.py`（驱动）+ `poc_two_tier_agent_code.py`（沙箱内 agent）

## 0. 结论

**设计方案在最简形态下可行（FEASIBLE）。** 在 cluster-msaFE8 上，一个 **runc** worker pod 承载 **cpython-wasm** 沙箱（ateom-wasmd），其内运行一个 **simple python agent**（ReAct 循环 + 工具分发），agent 的工具经**宿主斡旋的 capability 通道**（`host_proxy` → `env_proxy` host function → HostProxy 出口白名单）真实出网（HTTPS 200），全链路经 substrate 栈（e2bgw → atenet → atelet → ateom-wasmd → wasm kernel）跑通。

这对应 [ADR 0006](../decisions/0006-security-spectrum-model.md) 三维安全光谱的 **① runc 端 × ② cpython-wasm 端 × ③ 最小工具面** 组合（最简/最兼容档位），证明该档位的功能可行性。

## 1. 测试环境（核实的现状）

| 组件 | 值 |
|---|---|
| worker pool | `ate-wasm/wasm-pool`（3 副本 Running）+ `wasm-pool-xl`（2 副本） |
| worker pod runtime | **runc**（`runtimeClassName` 为空 = containerd 2.1.9 默认运行时；非 runsc/wasmtime RuntimeClass） |
| ateom 镜像 | `wasm-agent-registry-vpc.cn-hangzhou.cr.aliyuncs.com/wasm-poc/ateom-wasmd:dev-20260911` |
| ActorTemplate | `python-interpreter`（sandboxClass `wasm`，container command `python-kernel`，readyz :49983，onResume ColdBoot，workerSelector `workload: wasm-python`） |
| SandboxConfig | `wasm-default`（asset `python-wasm` = `s3://ate-assets/wasm/python-3.12.0.wasm`，即 CPython-wasm 解释器） |
| 网关 | `ate-system/e2bgw`（`--template-namespace=ate-wasm`，HS256 JWT，Secret `e2bgw-jwt-secret`） |
| 出口策略 | ateom-wasmd 用 `NetworkProxyConfig::default()` = 允许全部公网域名（`*`）、拦截内网 CIDR（10/8、172.16/12、192.168/16、100.64/10、169.254/16） |

## 2. POC 形态 → 三维光谱映射（ADR 0006）

| 维度 | POC 位置 | 证据 |
|---|---|---|
| **① pod runtime** | **runc**（最弱隔离端，拿密度/兼容红利） | worker pod `runtimeClassName` 空 = containerd 默认 runc |
| **② 执行体主体** | **cpython-wasm**（兼容端，解释执行；非 ADR 0005 的 rust-wasm） | `python-interpreter` 模板 + `python-wasm` 资产（python-3.12.0.wasm）+ bootstrap.py REPL kernel |
| **③ 工具/host function 面** | **最小面**（`host_proxy.http_request` 出口 + 沙箱内 calc） | bootstrap.py `host_proxy` shim → `env_proxy` host function（host_functions.rs） |

> 注：本 POC 是 ② 的 **cpython-wasm 端**（兼容优先），非 ADR 0005 推荐的 rust-wasm 安全端；① 在 runc 端但**未施加 ADR 0004 的 L2/L3/L4 硬化**（见 §6 局限）。即验证的是「最简功能形态可行」，非「runc 准入安全档位达标」。

## 3. 验收标准与结果

| # | 验收项 | 结果 | 关键证据 |
|---|---|---|---|
| 1 | runc worker pod 承载 cpython-wasm 沙箱 | ✅ | sandbox `sbx-c5ae1c5da27da06e` 创建于 `python-interpreter`（wasm class），state=running，落在 runc worker pod |
| 2 | substrate 数据面全链路执行代码 | ✅ | `POST /sandboxes/{id}/execute` → `SMOKE_OK 42`（e2bgw→atenet→atelet→ateom-wasmd→wasm kernel） |
| 3 | wasm 内宿主斡旋出网（capability 通道） | ✅ | `host_proxy.http_request('GET','https://www.aliyun.com')` → **HTTP 200 + 真实 HTML**（经 env_proxy host function + HostProxy 白名单） |
| 4 | simple python agent 自主循环 + 工具分发 | ✅ | ReAct 循环 3 步：web_get（真实出网 200）→ calc（42）→ finish |
| 5 | actor 生命周期（create→execute→delete） | ✅ | create 200 / execute 200 ×3 / DELETE **204** |

## 4. 证据（实测输出）

**STEP 1 smoke（数据面存活）**：
```
frame: {"text": "SMOKE_OK 42\n", "type": "stdout"}
frame: {"execution_count": 1, "type": "number_of_executions"}
frame: {"type": "end"}
```

**STEP 2 host_proxy 出网（宿主斡旋 capability）**：
```
HOSTPROXY_OK (200, b'<!DOCTYPE html>\n<html lang="zh-CN" traceid="...">\n  <head>...aliyun...')
```

**STEP 3 simple python agent（完整 ReAct trace）**：
```json
AGENT_RESULT {
  "task": "demo: egress + calc",
  "have_host_proxy": true,
  "steps": 2,
  "trace": [
    {"step": 0, "thought": "prove host-mediated egress from inside wasm",
     "action": "web_get",
     "observation": {"tool": "web_get", "ok": true, "url": "https://www.aliyun.com",
                      "resp": "(200, b'<!DOCTYPE html>...aliyun...')"}},
    {"step": 1, "thought": "do an in-sandbox computation tool call",
     "action": "calc", "observation": {"tool": "calc", "expr": "40+2", "result": 42}},
    {"step": 2, "final": "agent loop completed: web_get(egress)+calc tools ok"}
  ]
}
```

**create 响应**：`{"sandboxID":"sbx-c5ae1c5da27da06e","templateID":"python-interpreter","state":"running","envdVersion":"0.2.0",...}`（atespace=team-a；注：tenant-a 不是已注册 atespace，create 报 409 `Atespace tenant-a not found`，team-a 可用）

## 5. 这验证了设计的哪些核心命题

1. **两层数据面成立**：Tier-2 请求面（E2B REST → e2bgw → atenet → atunnel → 进程内 actor HTTP → wasm kernel）在 runc worker pod 上端到端跑通，与 gvisor/microvm 类同构（外部同构不变式）。
2. **wasm 执行体在 runc 上成立**：cpython-wasm 沙箱（ateom-wasmd 内嵌 wasmtime）在 runc worker pod（共享宿主内核）内正常运行——印证 ADR 0002 D1「runc worker pod 同样是两层」。
3. **agent-in-wasm 成立**：一个自主 agent 循环（任务→推理→工具→观察→完成）作为 wasm 内主体运行——印证 ADR 0002 D2 / ADR 0005「执行体 = wasm agent program」。
4. **宿主斡旋 capability 通道成立（关键）**：agent 的工具调用经 `host_proxy` shim → 帧协议 → `env_proxy` host function → HostProxy 出口白名单 → 真实公网，**guest 无 raw socket、出网唯一经宿主授权点**——印证 ADR 0003 D1（最小面/默认拒绝）/D3（capability 唯一执行点）/D5（出口代理隔离）的结构性优势「审计面=授权面」。
5. **ColdBoot 快照语义**：模板 `onResume: ColdBoot` 与既有 checkpoint/restore（fs-tar）一致（前序 M4/v0.4.0-aot 集群验收已覆盖，本 POC 未重测 suspend/resume）。

## 6. 局限与未尽（诚实声明）

| 项 | 说明 |
|---|---|
| **LLM 推理为确定性桩** | 测试环境无 LLM API key（DASHSCOPE/qwen 均不可得），agent 的「推理」步用确定性 ReAct 规划桩代替真实 LLM 调用。**agent 循环、工具分发、宿主斡旋出网均为真实**；仅 LLM 决策本身未接真实模型。接真实 LLM = 把 `plan()` 换成经 `host_proxy.http_request` 调 qwen（出口白名单已默认放行公网，机制同 web_get 工具，已验证可行） |
| **② 在 cpython 端，非 ADR 0005 rust-wasm** | 本 POC 是兼容端（cpython-wasm 主体 + Python agent）；ADR 0005 的 rust agent + 标准 WASI SDK 安全端**未建**（S4 缺口） |
| **① runc 未施加 ADR 0004 硬化** | worker pod 是现网 dev 镜像，**未做** ADR 0004 D2 的 L2 硬化（userns/seccomp 钉扎/只读 rootfs/allowPrivilegeEscalation:false）、D3 的 L3（专属节点/NetworkPolicy）、D4 的 L4（Ring 2）。即「runc 功能可行」≠「runc 准入安全档位达标」（ADR 0004 D6 四条件未核） |
| **审计 Ring 1 未采集** | 本 POC 未开 `WASM_AUDIT` 验证审计事件流（前序 audit demo 版已建 Ring 1，未在此 POC 断言） |
| **单节点集群** | cluster-msaFE8 单 worker 节点，未验证多节点调度/跨节点 resume；密度（多 actor/多池）未压测 |

**结论边界**：本 POC 确认**两层设计在最简形态（runc + cpython-wasm + simple python agent）下功能可行**——数据面、wasm 执行体、agent 循环、宿主斡旋 capability 通道全部跑通。**未**确认安全硬化档位（ADR 0004 L2/L3/L4 + ADR 0003 D8 门禁）与 ADR 0005 rust-wasm 安全端，这些是后续工作。

## 7. 复现步骤

```sh
export KUBECONFIG=/Users/liuqiming.lqm/project/profiles/config/ack/cluster/cluster-msaFE8/kube/config.yaml
# 1. 暴露 e2bgw（路径式数据面，无需 TLS/通配 DNS）
kubectl -n ate-system port-forward svc/e2bgw 8080:80 &
# 2. 跑 POC 驱动（自读 e2bgw-jwt-secret 签 JWT、建 sandbox、跑 smoke+host_proxy+agent、清理）
POC_ATESPACES=team-a python3 contrib/e2b-e2e/poc_two_tier_agent.py
# 驱动内部：POST /sandboxes {templateID:python-interpreter} → POST /sandboxes/{id}/execute {code}
#   step1 print 冒烟 / step2 host_proxy.http_request 出网 / step3 poc_two_tier_agent_code.py（ReAct agent）
#   → DELETE /sandboxes/{id}
```

`poc_two_tier_agent_code.py`（提交进沙箱执行的 simple python agent）：ReAct 循环 + `web_get`（经 host_proxy 真实出网）+ `calc`（沙箱内计算）两工具 + 确定性规划桩（接真实 LLM 时替换 `plan()`）。

## 8. 后续（按 ADR 推进）

1. **接真实 LLM**：`plan()` → 经 `host_proxy.http_request` 调 qwen/DashScope（需 API key；出口白名单默认放行公网，机制已由 web_get 工具验证）；
2. **ADR 0005 rust-wasm 安全端**：建 `src/agent/` Rust agent + 标准 WASI SDK，工具迁 host function（② 滑到安全端）；
3. **ADR 0004 runc 硬化 + 准入**：L2（userns/seccomp/只读 rootfs）+ L3（专属节点/NetworkPolicy）+ L4（Ring 2 Tetragon），核 D6 四条件 → runc 准入达标；
4. **审计闭环**：开 `WASM_AUDIT` 验 Ring 1 事件流（cell/net/policy.blocked）；
5. **suspend/resume + 多节点**：验 ColdBoot 快照轮回与跨节点 resume。
