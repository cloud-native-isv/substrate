# Changelog

自包含时间线：仅记录 xuanji 分支的定制变化；upstream 演进通过 rebase 跟踪，不在此记录。
条目新者在前，按特性归属。

## Unreleased

### Added

- **E2B 协议面**：`cmd/e2bgw` E2B REST → `ateapipb.Control` 翻译网关（HS256 API-Key 鉴权、
  sandboxes 生命周期（创建失败回滚）、pause/resume 映射、gRPC↔HTTP 错误映射、数据面
  307 跳转）；验收资产 `contrib/e2b-e2e/`（官方 SDK e2e、files e2e、smolagents、MCP server、TLS relay）。
- **Wasm 沙箱类**：`SandboxClass` 枚举新增 `wasm` 及 VAP 校验（每 arch 必须提供 `python-wasm`
  资产）；外部运行时 `ateom-wasmd`；示例 `manifests/xuanji/wasm-example.yaml`。
- **演进研究**：`docs/reference/upstream-evolution-research.md`（upstream 演进研究，xuanji 演进支持）+
  `docs/reference/xuanji-evolution-survey.md`（xuanji 分支演进调研）+ 配套三图
  （演进路线/上游新能力落位/定制扩展面，`docs/figures/`）。

## 2026-08-05

### Baseline

- Fork 自 upstream Agent Substrate `main` @ `aa9b7b82`；xuanji 分支全部改动登记于 [xuanji.md](xuanji.md)。
