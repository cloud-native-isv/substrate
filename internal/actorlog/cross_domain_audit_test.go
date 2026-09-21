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

package actorlog

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/agent-substrate/substrate/internal/resources"
)

// F12（全景图 §5「每次跨域操作皆白名单+审计」的 P 侧落地）：跨域审计事件须用与
// sandbox 仓 S13（src/runtime/src/audit.rs 的 mcp.call）对齐的统一 schema
// （event/src_domain/dst_domain/allowed），构成跨仓统一审计平面；保留键不得被
// Fields 覆盖（防污染 schema）。
func TestEmitCrossDomainAuditUnifiedSchema(t *testing.T) {
	var buf bytes.Buffer
	al := NewActorLogger(&buf, false)
	al.EmitCrossDomainAudit(CrossDomainAudit{
		Event:     "actor.assign",
		SrcDomain: DomainPlatform,
		DstDomain: DomainService,
		Operation: "assign actor to worker pod (P->S)",
		Allowed:   true,
		Fields: map[string]any{
			"worker_pod": "wasm-pool-abc",
			"event":      "HACK-attempt", // 保留键，必须被忽略
			"src_domain": "Agent",        // 保留键，必须被忽略
		},
	}, resources.ActorRef{Atespace: "team-a", Name: "act-1"}, "uid-1", "tmpl-ns", "tmpl-1")

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if m["event"] != "actor.assign" {
		t.Errorf("event = %v, want 'actor.assign' (reserved key must not be overridden by Fields)", m["event"])
	}
	if m["src_domain"] != "P" {
		t.Errorf("src_domain = %v, want 'P' (reserved key must not be overridden)", m["src_domain"])
	}
	if m["dst_domain"] != "S" {
		t.Errorf("dst_domain = %v, want 'S'", m["dst_domain"])
	}
	if m["allowed"] != true {
		t.Errorf("allowed = %v, want true", m["allowed"])
	}
	if m["worker_pod"] != "wasm-pool-abc" {
		t.Errorf("custom field worker_pod = %v, want 'wasm-pool-abc'", m["worker_pod"])
	}
	if _, ok := m["reason"]; ok {
		t.Errorf("reason must be absent when allowed=true")
	}
	labels, ok := m[al.labelsKey].(map[string]any)
	if !ok {
		t.Fatal("missing/invalid labels group")
	}
	if labels["ate.dev/actor_name"] != "act-1" {
		t.Errorf("actor_name label = %v, want 'act-1'", labels["ate.dev/actor_name"])
	}
}

// 拒绝路径：allowed=false 须带 reason（deny-by-default 白名单裁决审计，ADR 0003 D1），
// 且 S→MCP 跨越标注 dst_domain=MCP（与 sandbox S13 同模型）。
func TestEmitCrossDomainAuditDenyCarriesReason(t *testing.T) {
	var buf bytes.Buffer
	al := NewActorLogger(&buf, false)
	al.EmitCrossDomainAudit(CrossDomainAudit{
		Event:     "mcp.pool.route",
		SrcDomain: DomainService,
		DstDomain: DomainMCP,
		Operation: "route MCP call to shared-pool execution unit",
		Allowed:   false,
		Reason:    "tenant not in pool allowlist",
	}, resources.ActorRef{Atespace: "team-a", Name: "act-1"}, "uid-1", "ns", "t")

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if m["allowed"] != false {
		t.Errorf("allowed = %v, want false", m["allowed"])
	}
	if m["reason"] != "tenant not in pool allowlist" {
		t.Errorf("reason = %v, want 'tenant not in pool allowlist'", m["reason"])
	}
	if m["src_domain"] != "S" || m["dst_domain"] != "MCP" {
		t.Errorf("domains = %v->%v, want S->MCP", m["src_domain"], m["dst_domain"])
	}
}
