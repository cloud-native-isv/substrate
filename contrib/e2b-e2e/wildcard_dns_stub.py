#!/usr/bin/env python3
"""极简通配符 DNS 桩（本地演示用，免疫公网 DNS 污染）。

只为一个用途：把交给它的所有 A 查询都回同一个 IPv4（默认 CLB 公网 IP），
让 *.sbx.test 这类演示域名在本机可靠解析到公网入口，查询根本不出网。

天然多级通配：本桩不解析域名结构，对任意深度子域一律回 STUB_IP——
多租户接入面 api.tenant-a.sbx.test（控制面）与
{port}-{id}.tenant-a.sbx.test（数据面，SDK 生成 {port}-{id}.{E2B_DOMAIN} 单级子域）
均无需任何配置即可解析。

通过 macOS 的 /etc/resolver/<suffix> 把某个后缀的解析定向到本桩：
  echo -e 'nameserver 127.0.0.1\nport 5354' | sudo tee /etc/resolver/sbx.test
单条 resolver 覆盖整个 sbx.test 子树（含全部现有/未来租户域）。

环境变量：
  STUB_IP    返回的 IPv4，默认 121.40.155.16
  STUB_HOST  监听地址，默认 127.0.0.1
  STUB_PORT  监听端口，默认 5354（非特权，免 sudo）
"""
import os
import socket
import struct

STUB_IP = os.environ.get("STUB_IP", "121.40.155.16")
STUB_HOST = os.environ.get("STUB_HOST", "127.0.0.1")
STUB_PORT = int(os.environ.get("STUB_PORT", "5354"))

TYPE_A = 1


def _parse_question(data: bytes):
    # header 12 字节；question 从 12 开始：QNAME(以 0 结尾的标签序列) + QTYPE(2) + QCLASS(2)
    idx = 12
    while idx < len(data) and data[idx] != 0:
        idx += data[idx] + 1
    idx += 1  # 跳过结尾 0
    qtype = struct.unpack("!H", data[idx : idx + 2])[0]
    q_end = idx + 4  # QTYPE + QCLASS
    return qtype, q_end


def _build_response(query: bytes, ip: str) -> bytes:
    tid = query[:2]
    qtype, q_end = _parse_question(query)
    question = query[12:q_end]
    # flags: QR=1, Opcode=0, AA=1, RD 复制请求位, RA=0；RCODE=0
    rd = query[2] & 0x01
    flags = 0x8400 | (rd << 8)
    if qtype == TYPE_A:
        ancount = 1
        answer = (
            b"\xc0\x0c"                      # 指向 offset 12 的名字
            + struct.pack("!HHI", TYPE_A, 1, 30)  # TYPE=A CLASS=IN TTL=30
            + struct.pack("!H", 4)          # RDLENGTH
            + socket.inet_aton(ip)          # RDATA
        )
    else:
        ancount = 0
        answer = b""
    header = tid + struct.pack("!HHHHH", flags, 1, ancount, 0, 0)
    return header + question + answer


def main() -> None:
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    sock.bind((STUB_HOST, STUB_PORT))
    print(f"[dns-stub] UDP {STUB_HOST}:{STUB_PORT}  A * -> {STUB_IP}", flush=True)
    try:
        while True:
            data, addr = sock.recvfrom(1024)
            if len(data) < 12:
                continue
            try:
                resp = _build_response(data, STUB_IP)
                sock.sendto(resp, addr)
            except Exception as e:  # 桩不因单条畸形查询退出
                print(f"[dns-stub] bad query from {addr}: {e}", flush=True)
    except KeyboardInterrupt:
        print("\n[dns-stub] shutting down", flush=True)
    finally:
        sock.close()


if __name__ == "__main__":
    main()
