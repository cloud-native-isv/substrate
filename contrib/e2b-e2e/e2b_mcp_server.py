#!/usr/bin/env python3
"""§5.1 MCP Server：把 wasm-e2b-k8s 沙箱能力以 MCP tools 暴露给 Agent 宿主（B12）。

传输形态（MCP_TRANSPORT 切换，默认 stdio）：
  - stdio（MCP 标准）：Claude Code / Cursor 等本地宿主以子进程方式拉起本脚本，
    经 JSON-RPC 发现工具并调用。
  - streamable-http（对齐 design §5.1「API Server 内置 MCP Server（SSE / Streamable-HTTP）」）：
    以 HTTP 服务监听 MCP_HOST:MCP_PORT（默认 127.0.0.1:8081），远程 Agent 经网络直连，
    可独立部署为长期在线的 MCP 端点。
工具集对齐 design.md §5.1 的沙箱能力面：
  - 生命周期：sandbox_create / sandbox_kill / sandbox_set_timeout
  - 代码执行：sandbox_run_code（kernel NDJSON 聚合）
  - 文件系统：file_read / file_write / file_list / file_stat / file_make_dir
              / file_move / file_remove（B11 落地的主沙箱文件 API）

环境变量：
  E2B_GATEWAY_URL   Gateway 入口（默认 http://127.0.0.1:8080；
                    集群演示用 kubectl port-forward 映射，见 deploy/redeploy.sh）
  E2B_API_KEY       租户 JWT（scripts/sign_jwt.py 用 AUTH_SECRET 签发）。
                    数据面访问令牌由 sandbox_create 响应的 envdAccessToken 获得，
                    本进程内存按 sandbox_id 缓存（POC：进程重启后需重新 create）。
  MCP_TRANSPORT     stdio（默认）| streamable-http
  MCP_HOST          streamable-http 监听地址（默认 127.0.0.1；容器内暴露用 0.0.0.0）
  MCP_PORT          streamable-http 监听端口（默认 8081）

宿主配置示例（Claude Code .mcp.json）：
  {"mcpServers": {"e2b-sandbox": {
      "command": "/path/to/.venv-agent/bin/python",
      "args": ["/path/to/scripts/e2b_mcp_server.py"],
      "env": {"E2B_GATEWAY_URL": "http://127.0.0.1:8080", "E2B_API_KEY": "<jwt>"}}}}

远程配置示例（MCP_TRANSPORT=streamable-http 起服后，宿主直连 URL）：
  {"mcpServers": {"e2b-sandbox": {"type": "http", "url": "http://<host>:8081/mcp"}}}
"""

import json
import os
from typing import Any, Dict, Optional

import requests
from mcp.server.fastmcp import FastMCP

GATEWAY_URL = os.environ.get("E2B_GATEWAY_URL", "http://127.0.0.1:8080").rstrip("/")
API_KEY = os.environ.get("E2B_API_KEY", "")

# sandbox_id → envdAccessToken（数据面能力令牌，create 时签发）
_TOKENS: Dict[str, str] = {}

mcp = FastMCP(
    "e2b-sandbox",
    # streamable-http 监听地址（stdio 模式下不生效，构造时传入无害）
    host=os.environ.get("MCP_HOST", "127.0.0.1"),
    port=int(os.environ.get("MCP_PORT", "8081")),
)


# ---------------------------------------------------------------
# 内部 HTTP 辅助
# ---------------------------------------------------------------

def _control_headers() -> Dict[str, str]:
    h = {"Content-Type": "application/json"}
    if API_KEY:
        h["X-API-Key"] = API_KEY
    return h


def _data_headers(sandbox_id: str) -> Dict[str, str]:
    """数据面鉴权：X-Access-Token（能力令牌）+ E2b-Sandbox-Id 头。"""
    h = {"Content-Type": "application/json", "E2b-Sandbox-Id": sandbox_id}
    token = _TOKENS.get(sandbox_id)
    if token:
        h["X-Access-Token"] = token
    if API_KEY:
        # JWT Bearer 兜底（gateway 数据面支持 JWT + E2b-Sandbox-Id 路径）
        h["Authorization"] = f"Bearer {API_KEY}"
    return h


def _check(resp: requests.Response, what: str) -> requests.Response:
    if resp.status_code >= 400:
        raise RuntimeError(f"{what} failed: HTTP {resp.status_code}: {resp.text[:500]}")
    return resp


def _aggregate_ndjson(resp: requests.Response) -> Dict[str, Any]:
    """聚合 kernel /execute 的 NDJSON 帧流为结构化结果。

    gateway 帧契约（access/gateway/src/main.rs）：
      stdout/stderr       {"type": "stdout", "text": ...}
      result              {"type": "result", "text": <repr>, "is_main_result": true}
      error               {"type": "error", "name", "value", "traceback"}
      number_of_executions {"type": "number_of_executions", "execution_count": N}
    注意无统一 "data" 字段——曾误用 frame.get("data") 导致输出聚合成字面 "null"。
    """
    stdout_parts, stderr_parts, results, error = [], [], [], None
    executions = None
    for line in resp.iter_lines():
        if not line:
            continue
        frame = json.loads(line)
        ftype = frame.get("type")
        if ftype == "stdout":
            stdout_parts.append(frame.get("text") or "")
        elif ftype == "stderr":
            stderr_parts.append(frame.get("text") or "")
        elif ftype == "result":
            results.append(frame.get("text") or "")
        elif ftype == "error":
            error = {
                "name": frame.get("name"),
                "value": frame.get("value"),
                "traceback": frame.get("traceback"),
            }
        elif ftype == "number_of_executions":
            executions = frame.get("execution_count")
    return {
        "stdout": "".join(stdout_parts),
        "stderr": "".join(stderr_parts),
        "results": results,
        "error": error,
        "execution_count": executions,
    }


# ---------------------------------------------------------------
# 生命周期工具
# ---------------------------------------------------------------

@mcp.tool()
def sandbox_create(template: str = "python", timeout: int = 300) -> str:
    """Create a new sandbox (Wasm by default). Returns sandbox_id and metadata.

    The sandbox is a lightweight Wasm-based Python environment (millisecond
    startup). Use the returned sandbox_id for all subsequent tool calls.

    Args:
        template: Sandbox template/engine ("python" for Wasm; "full-os"/"docker"
                  for a full-OS container when Wasm capabilities are insufficient).
        timeout: Idle timeout in seconds before the sandbox auto-pauses.
    """
    resp = _check(
        requests.post(
            f"{GATEWAY_URL}/sandboxes",
            headers=_control_headers(),
            json={"templateID": template, "timeout": timeout},
            timeout=60,
        ),
        "sandbox_create",
    )
    data = resp.json()
    sid = data.get("sandboxID", "")
    token = data.get("envdAccessToken")
    if sid and token:
        _TOKENS[sid] = token
    return json.dumps(data, ensure_ascii=False)


@mcp.tool()
def sandbox_kill(sandbox_id: str) -> str:
    """Kill a sandbox and release its resources (kernel memory, tmpfs, route).

    Args:
        sandbox_id: The sandbox to kill (from sandbox_create).
    """
    resp = _check(
        requests.delete(
            f"{GATEWAY_URL}/sandboxes/{sandbox_id}",
            headers=_control_headers(),
            timeout=30,
        ),
        "sandbox_kill",
    )
    _TOKENS.pop(sandbox_id, None)
    return resp.text or "killed"


@mcp.tool()
def sandbox_set_timeout(sandbox_id: str, timeout: int) -> str:
    """Reset the sandbox idle countdown (E2B set_timeout semantics).

    Args:
        sandbox_id: Target sandbox.
        timeout: New idle timeout in seconds.
    """
    resp = _check(
        requests.post(
            f"{GATEWAY_URL}/sandboxes/{sandbox_id}/timeout",
            headers=_control_headers(),
            json={"timeout": timeout},
            timeout=30,
        ),
        "sandbox_set_timeout",
    )
    return f"ok (HTTP {resp.status_code})"


# ---------------------------------------------------------------
# 代码执行工具
# ---------------------------------------------------------------

@mcp.tool()
def sandbox_run_code(sandbox_id: str, code: str, engine: Optional[str] = None) -> str:
    """Execute Python code inside the sandbox's persistent Jupyter-like kernel.

    Variables/imports persist across calls on the same sandbox. Returns JSON with
    stdout, stderr, results (repr of last expression), error and execution_count.

    Args:
        sandbox_id: Target sandbox (from sandbox_create).
        code: Python source code to execute.
        engine: Optional explicit engine hint: "kernel" (persistent Wasm kernel),
                "wasm" (one-shot), "full-os" (full-OS container for pip/shell/
                subprocess/raw sockets). Omit to let the platform infer.
    """
    payload: Dict[str, Any] = {"code": code}
    if engine:
        payload["engine"] = engine
    resp = _check(
        requests.post(
            f"{GATEWAY_URL}/execute",
            headers=_data_headers(sandbox_id),
            json=payload,
            timeout=180,
            stream=True,
        ),
        "sandbox_run_code",
    )
    return json.dumps(_aggregate_ndjson(resp), ensure_ascii=False)


# ---------------------------------------------------------------
# 文件系统工具（B11：E2B Filesystem 兼容层）
# ---------------------------------------------------------------

@mcp.tool()
def file_read(sandbox_id: str, path: str) -> str:
    """Read a text file from the sandbox filesystem.

    Paths resolve against the sandbox work root: "/sandbox/..." (Wasm view),
    "/home/user/..." (E2B HOME alias), absolute or relative paths all map to
    the same root. The kernel sees exactly these files under /sandbox.

    Args:
        sandbox_id: Target sandbox.
        path: File path (see resolution rules above).
    """
    resp = _check(
        requests.get(
            f"{GATEWAY_URL}/files",
            headers=_data_headers(sandbox_id),
            params={"path": path},
            timeout=60,
        ),
        "file_read",
    )
    return resp.text


@mcp.tool()
def file_write(sandbox_id: str, path: str, content: str) -> str:
    """Write a text file into the sandbox filesystem (creates parent dirs).

    Args:
        sandbox_id: Target sandbox.
        path: Destination path (same resolution rules as file_read).
        content: Text content to write.
    """
    resp = _check(
        requests.post(
            f"{GATEWAY_URL}/files",
            headers={**_data_headers(sandbox_id), "Content-Type": "application/octet-stream"},
            params={"path": path},
            data=content.encode("utf-8"),
            timeout=60,
        ),
        "file_write",
    )
    return resp.text


def _fs_rpc(sandbox_id: str, method: str, payload: Dict[str, Any]) -> Dict[str, Any]:
    resp = _check(
        requests.post(
            f"{GATEWAY_URL}/filesystem.Filesystem/{method}",
            headers=_data_headers(sandbox_id),
            json=payload,
            timeout=60,
        ),
        f"filesystem.{method}",
    )
    return resp.json()


@mcp.tool()
def file_list(sandbox_id: str, path: str = "/sandbox", depth: int = 1) -> str:
    """List directory entries (name, type, path, size, mode, owner, modified time).

    Args:
        sandbox_id: Target sandbox.
        path: Directory path (default sandbox root).
        depth: Recursion depth (1 = direct children; clamped server-side to 8).
    """
    return json.dumps(_fs_rpc(sandbox_id, "ListDir", {"path": path, "depth": depth}),
                      ensure_ascii=False)


@mcp.tool()
def file_stat(sandbox_id: str, path: str) -> str:
    """Stat a file or directory (existence + metadata: size/mode/owner/mtime).

    Args:
        sandbox_id: Target sandbox.
        path: Path to stat.
    """
    return json.dumps(_fs_rpc(sandbox_id, "Stat", {"path": path}), ensure_ascii=False)


@mcp.tool()
def file_make_dir(sandbox_id: str, path: str) -> str:
    """Create a directory (and all missing parents). Fails if it already exists.

    Args:
        sandbox_id: Target sandbox.
        path: Directory path to create.
    """
    return json.dumps(_fs_rpc(sandbox_id, "MakeDir", {"path": path}), ensure_ascii=False)


@mcp.tool()
def file_move(sandbox_id: str, source: str, destination: str) -> str:
    """Rename/move a file or directory (parent dirs of destination are created).

    Args:
        sandbox_id: Target sandbox.
        source: Current path.
        destination: New path.
    """
    return json.dumps(
        _fs_rpc(sandbox_id, "Move", {"source": source, "destination": destination}),
        ensure_ascii=False,
    )


@mcp.tool()
def file_remove(sandbox_id: str, path: str) -> str:
    """Remove a file or directory (directories are removed recursively).

    Args:
        sandbox_id: Target sandbox.
        path: Path to remove.
    """
    _fs_rpc(sandbox_id, "Remove", {"path": path})
    return "removed"


if __name__ == "__main__":
    transport = os.environ.get("MCP_TRANSPORT", "stdio")
    if transport == "streamable-http":
        # 网络形态（design §5.1）：远程 Agent 经 HTTP 直连 /mcp 端点
        mcp.run(transport="streamable-http")
    else:
        # stdio 传输（MCP 标准）：宿主进程经 stdin/stdout 与本进程交换 JSON-RPC
        mcp.run()
