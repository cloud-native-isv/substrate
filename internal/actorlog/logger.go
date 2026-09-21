// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package actorlog provides structured JSON logging for actor sandboxes shared
// by the gVisor and micro-VM ateom runtimes. It forwards an actor container's
// stdout/stderr to the worker pod's stdout, annotated with ate.dev/* labels, and
// emits synthetic actor lifecycle events.
package actorlog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"
	"time"

	"github.com/agent-substrate/substrate/internal/resources"
)

// SyncedWriter wraps an io.Writer and synchronizes writes across goroutines.
type SyncedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

// Write writes the byte slice to the underlying writer, synchronized by a mutex.
func (sw *SyncedWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.w.Write(p)
}

// NewSyncedWriter returns a new SyncedWriter wrapping the given io.Writer.
func NewSyncedWriter(w io.Writer) *SyncedWriter {
	return &SyncedWriter{w: w}
}

// ActorLogger handles structured logging for actor sandboxes and lifecycle events.
type ActorLogger struct {
	writer    io.Writer
	labelsKey string
}

// NewActorLogger creates a new ActorLogger wrapping the provided destination writer.
func NewActorLogger(w io.Writer, isOnGCE bool) *ActorLogger {
	labelsKey := "labels"
	if isOnGCE {
		labelsKey = "logging.googleapis.com/labels"
	}
	return &ActorLogger{
		writer:    w,
		labelsKey: labelsKey,
	}
}

// EmitLifecycleLog logs a synthetic actor lifecycle event.
func (al *ActorLogger) EmitLifecycleLog(msg string, actorRef resources.ActorRef, actorUID, actorTemplateNamespace, actorTemplateName string) {
	envelope := map[string]any{
		"time":    time.Now().Format(time.RFC3339Nano),
		"message": msg,
		al.labelsKey: map[string]string{
			"ate.dev/actor_atespace":           actorRef.Atespace,
			"ate.dev/actor_name":               actorRef.Name,
			"ate.dev/actor_uid":                actorUID,
			"ate.dev/actor_template_namespace": actorTemplateNamespace,
			"ate.dev/actor_template_name":      actorTemplateName,
		},
	}
	if envBytes, err := json.Marshal(envelope); err == nil {
		envBytes = append(envBytes, '\n')
		_, _ = al.writer.Write(envBytes)
	}
}

// TrustDomain 标识四层信任域全景图（docs/concepts/trust-domain-panorama.md §0）中的
// 一个信任域，外加 MCP 能力后端（S 层对外的原生能力面，ADR 0007）。信任级自外向内
// 递减 I > P > S > Agent；MCP 执行单元是不可信、可抛弃的（一次性）。
type TrustDomain string

const (
	// DomainInfrastructure = I 层：ECS / ACK / k8s / node / 云账号。
	DomainInfrastructure TrustDomain = "I"
	// DomainPlatform = P 层：substrate 控制面（ateapi/atelet/atecontroller）+ MCP 池编排。
	DomainPlatform TrustDomain = "P"
	// DomainService = S 层：worker pod（一个数字员工的身份/配置/能力表 + ateom 中介）。
	DomainService TrustDomain = "S"
	// DomainAgent = Agent 层：wasm 沙箱内由 LLM 驱动的 agent（最低信任）。
	DomainAgent TrustDomain = "Agent"
	// DomainMCP = MCP 能力后端（执行单元）：S 层经中介触达的原生能力面。
	DomainMCP TrustDomain = "MCP"
)

// CrossDomainAudit 是一次**跨信任域操作**的统一审计事件（全景图 §5「每次跨域操作皆
// 白名单 + 审计」的 P 侧落地，F12）。字段名刻意与 sandbox 仓 `src/runtime/src/audit.rs`
// 的 `mcp.call` 事件对齐（`event` / `src_domain` / `dst_domain` / `allowed` / `reason`），
// 使 substrate 侧的 P↔S / S↔S / I↔P 跨越与 sandbox 侧的 S↔Agent / S↔MCP 跨越归集到
// **同一审计平面**（跨仓统一审计模型，split doc §8.3 F12）。
type CrossDomainAudit struct {
	// Event 是稳定事件名（如 "actor.assign" / "worker.drain" / "rbac.check" /
	// "mcp.pool.route"），供日志归集与告警按名筛选。
	Event string
	// SrcDomain / DstDomain 标注本次跨越的方向（信任域边界 = 审计点，全景图 §5）。
	SrcDomain TrustDomain
	DstDomain TrustDomain
	// Operation 是人类可读的具体操作描述。
	Operation string
	// Allowed 是白名单裁决结果；false 表示被拒（deny-by-default，ADR 0003 D1）。
	Allowed bool
	// Reason 仅在 Allowed=false 时填充拒绝原因。
	Reason string
	// Fields 是事件特定的结构化字段（如 worker_pod / tool / target）。调用方负责
	// 截断敏感/超长值——审计绝不记录秘密原值（与 sandbox token 只记指纹一致，I-1）。
	// 保留键（time/event/src_domain/dst_domain/operation/allowed/reason/labels）不会被覆盖。
	Fields map[string]any
}

// reservedAuditKeys 是 envelope 的固定键，Fields 不得覆盖（防止污染统一 schema）。
var reservedAuditKeys = map[string]struct{}{
	"time": {}, "event": {}, "src_domain": {}, "dst_domain": {},
	"operation": {}, "allowed": {}, "reason": {},
}

// EmitCrossDomainAudit 发出一条跨信任域审计事件（统一 schema），带 actor 身份标签。
// 这是 F12 的发射点：control-plane（P 层）在每次跨域操作（如把 actor 派到 worker =
// P→S、MCP 池路由 = S→S、RBAC/NetworkPolicy 校验 = I→P）调用它，与 sandbox 侧 S13 的
// S↔Agent / S↔MCP 审计共用字段名，构成全景图 §5 的统一跨域审计平面。
func (al *ActorLogger) EmitCrossDomainAudit(a CrossDomainAudit, actorRef resources.ActorRef, actorUID, actorTemplateNamespace, actorTemplateName string) {
	envelope := map[string]any{
		"time":       time.Now().Format(time.RFC3339Nano),
		"event":      a.Event,
		"src_domain": string(a.SrcDomain),
		"dst_domain": string(a.DstDomain),
		"operation":  a.Operation,
		"allowed":    a.Allowed,
		al.labelsKey: map[string]string{
			"ate.dev/actor_atespace":           actorRef.Atespace,
			"ate.dev/actor_name":               actorRef.Name,
			"ate.dev/actor_uid":                actorUID,
			"ate.dev/actor_template_namespace": actorTemplateNamespace,
			"ate.dev/actor_template_name":      actorTemplateName,
		},
	}
	if a.Reason != "" {
		envelope["reason"] = a.Reason
	}
	for k, v := range a.Fields {
		if _, reserved := reservedAuditKeys[k]; reserved {
			continue
		}
		if k == al.labelsKey {
			continue
		}
		envelope[k] = v
	}
	if envBytes, err := json.Marshal(envelope); err == nil {
		envBytes = append(envBytes, '\n')
		_, _ = al.writer.Write(envBytes)
	}
}

// StartJSONLogPipe intercepts container raw stdout/stderr streams and pipes them
// through the logger. containerName tags every line with the originating container;
// callers that multiplex multiple containers should give each its own pipe so the
// tag is meaningful.
func (al *ActorLogger) StartJSONLogPipe(actorRef resources.ActorRef, actorUID, actorTemplateNamespace, actorTemplateName, containerName string) (io.WriteCloser, error) {
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	go func() {
		al.WrapContainerLogs(pr, actorRef, actorUID, actorTemplateNamespace, actorTemplateName, containerName)
		pr.Close()
	}()
	return pw, nil
}

// WrapContainerLogs reads log lines from r, parses them, and logs them in a unified
// structured format. containerName is added as the ate.dev/container_name label so
// multi-container actors can be demultiplexed.
func (al *ActorLogger) WrapContainerLogs(r io.Reader, actorRef resources.ActorRef, actorUID, actorTemplateNamespace, actorTemplateName, containerName string) {
	rdr := bufio.NewReader(r)
	for {
		lineBytes, err := rdr.ReadBytes('\n')

		// Strip trailing newline from ReadBytes if present
		if len(lineBytes) > 0 && lineBytes[len(lineBytes)-1] == '\n' {
			lineBytes = lineBytes[:len(lineBytes)-1]
		}

		if len(lineBytes) > 0 {
			var m map[string]any
			var envelope map[string]any

			dec := json.NewDecoder(bytes.NewReader(lineBytes))
			dec.UseNumber()

			unmarshalErr := dec.Decode(&m)
			if unmarshalErr == nil {
				var trailing any
				if err := dec.Decode(&trailing); err != io.EOF {
					unmarshalErr = errors.New("trailing garbage detected after JSON object")
				}
			}

			if unmarshalErr != nil {
				labels := map[string]string{
					"ate.dev/actor_atespace":           actorRef.Atespace,
					"ate.dev/actor_name":               actorRef.Name,
					"ate.dev/actor_uid":                actorUID,
					"ate.dev/actor_template_namespace": actorTemplateNamespace,
					"ate.dev/actor_template_name":      actorTemplateName,
					"ate.dev/container_name":           containerName,
				}
				envelope = map[string]any{
					"time":       time.Now().Format(time.RFC3339Nano),
					"message":    string(lineBytes),
					al.labelsKey: labels,
				}
			} else {
				if _, ok := m["time"]; !ok {
					m["time"] = time.Now().Format(time.RFC3339Nano)
				}
				labels, ok := m[al.labelsKey].(map[string]any)
				if !ok {
					labels = make(map[string]any)
					m[al.labelsKey] = labels
				}
				labels["ate.dev/actor_atespace"] = actorRef.Atespace
				labels["ate.dev/actor_name"] = actorRef.Name
				labels["ate.dev/actor_template_namespace"] = actorTemplateNamespace
				labels["ate.dev/actor_template_name"] = actorTemplateName
				labels["ate.dev/container_name"] = containerName
				envelope = m
			}

			if envBytes, err := json.Marshal(envelope); err == nil {
				envBytes = append(envBytes, '\n')
				_, _ = al.writer.Write(envBytes)
			}
		}

		if err != nil {
			break
		}
	}
}
