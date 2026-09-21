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

package mcppool

import (
	"testing"

	"github.com/agent-substrate/substrate/internal/actorlog"
	"github.com/agent-substrate/substrate/internal/resources"
)

// recordedAudit captures one F12 cross-domain audit emission so tests can assert
// the P router is "自身被审计" (C2) without a real logger backend.
type recordedAudit struct {
	a   actorlog.CrossDomainAudit
	ref resources.ActorRef
}

// fakeEmitter is a recording AuditEmitter.
type fakeEmitter struct {
	events []recordedAudit
}

func (f *fakeEmitter) EmitCrossDomainAudit(a actorlog.CrossDomainAudit, ref resources.ActorRef, _, _, _ string) {
	f.events = append(f.events, recordedAudit{a: a, ref: ref})
}

func (f *fakeEmitter) last(t *testing.T) recordedAudit {
	t.Helper()
	if len(f.events) == 0 {
		t.Fatal("expected an audit event, got none")
	}
	return f.events[len(f.events)-1]
}

// twoTenantPool is the shared fixture: tenants team-a and team-b both admitted,
// each with its own per-tenant execution unit serving git.clone (team-a's unit
// also serves compile.file). A tool not listed is never routable.
func twoTenantPool(emitter AuditEmitter) *Router {
	return NewRouter(PoolConfig{
		AllowedTenants: []string{"team-a", "team-b"},
		Units: []ExecutionUnit{
			{Tenant: "team-a", Name: "unit-a", Endpoint: "mcp://unit-a", Tools: []string{"git.clone", "compile.file"}},
			{Tenant: "team-b", Name: "unit-b", Endpoint: "mcp://unit-b", Tools: []string{"git.clone"}},
		},
	}, emitter)
}

// Gate 1: a tenant not on the allowlist is denied even though a unit exists for
// another tenant — and the denial is audited (allowed=false + reason). This is
// the R3 cross-tenant confused-deputy defense.
func TestRouteDeniesTenantNotInAllowlist(t *testing.T) {
	em := &fakeEmitter{}
	r := twoTenantPool(em)

	d := r.Route(resources.ActorRef{Atespace: "team-c", Name: "act-1"}, "uid-1", "ns", "tmpl", "git.clone")
	if d.Allowed {
		t.Fatalf("team-c is not admitted; got Allowed=true (unit=%v)", d.Unit)
	}
	if d.Unit != nil {
		t.Errorf("denied decision must carry no unit, got %v", d.Unit)
	}
	if d.Reason == "" {
		t.Error("denied decision must carry a reason")
	}

	ev := em.last(t).a
	if ev.Allowed {
		t.Error("audit allowed = true, want false")
	}
	if ev.SrcDomain != actorlog.DomainService || ev.DstDomain != actorlog.DomainMCP {
		t.Errorf("audit domains = %v->%v, want S->MCP", ev.SrcDomain, ev.DstDomain)
	}
	if ev.Event != "mcp.pool.route" {
		t.Errorf("audit event = %q, want 'mcp.pool.route'", ev.Event)
	}
	if ev.Reason == "" {
		t.Error("denied audit must carry a reason")
	}
}

// Gate 2 happy path: an admitted tenant calling a tool its own unit serves is
// allowed, routed to the right unit, and audited (allowed=true, no reason).
func TestRouteAllowsAdmittedTenantToOwnUnit(t *testing.T) {
	em := &fakeEmitter{}
	r := twoTenantPool(em)

	d := r.Route(resources.ActorRef{Atespace: "team-a", Name: "act-1"}, "uid-1", "ns", "tmpl", "compile.file")
	if !d.Allowed {
		t.Fatalf("team-a compile.file should be allowed, denied: %s", d.Reason)
	}
	if d.Unit == nil || d.Unit.Name != "unit-a" {
		t.Fatalf("routed to unit %v, want unit-a", d.Unit)
	}
	if d.Unit.Endpoint != "mcp://unit-a" {
		t.Errorf("endpoint = %q, want mcp://unit-a", d.Unit.Endpoint)
	}

	ev := em.last(t)
	if !ev.a.Allowed {
		t.Error("audit allowed = false, want true")
	}
	if ev.a.Reason != "" {
		t.Errorf("allowed audit must carry no reason, got %q", ev.a.Reason)
	}
	if ev.a.Fields["tool"] != "compile.file" || ev.a.Fields["unit"] != "unit-a" || ev.a.Fields["tenant"] != "team-a" {
		t.Errorf("audit fields = %v, want tool/unit/tenant populated", ev.a.Fields)
	}
	if ev.ref.Atespace != "team-a" || ev.ref.Name != "act-1" {
		t.Errorf("audit actor ref = %v, want team-a/act-1", ev.ref)
	}
}

// Gate 2 denial: an admitted tenant calling a tool no unit of its own serves is
// denied (deny-by-default on the tool allowlist, ADR 0007 D7).
func TestRouteDeniesToolNotServedByTenantUnit(t *testing.T) {
	em := &fakeEmitter{}
	r := twoTenantPool(em)

	// team-b's unit serves only git.clone, not compile.file.
	d := r.Route(resources.ActorRef{Atespace: "team-b", Name: "act-2"}, "uid-2", "ns", "tmpl", "compile.file")
	if d.Allowed {
		t.Fatalf("team-b unit does not serve compile.file; got Allowed=true")
	}
	if d.Reason == "" {
		t.Error("denied decision must carry a reason")
	}
	if ev := em.last(t).a; ev.Allowed {
		t.Error("denial must be audited with allowed=false")
	}
}

// C1/R3 isolation: each tenant resolves only to its OWN unit, never the other
// tenant's, even though both serve the same tool name. This is the core
// cross-tenant boundary of the shared pool.
func TestRouteNeverCrossesTenantUnitBoundary(t *testing.T) {
	em := &fakeEmitter{}
	r := twoTenantPool(em)

	a := r.Route(resources.ActorRef{Atespace: "team-a", Name: "act-a"}, "uid-a", "ns", "tmpl", "git.clone")
	if !a.Allowed || a.Unit.Name != "unit-a" {
		t.Fatalf("team-a git.clone -> %v (allowed=%v), want unit-a", a.Unit, a.Allowed)
	}
	b := r.Route(resources.ActorRef{Atespace: "team-b", Name: "act-b"}, "uid-b", "ns", "tmpl", "git.clone")
	if !b.Allowed || b.Unit.Name != "unit-b" {
		t.Fatalf("team-b git.clone -> %v (allowed=%v), want unit-b", b.Unit, b.Allowed)
	}
	if a.Unit.Tenant != "team-a" || b.Unit.Tenant != "team-b" {
		t.Errorf("unit tenant leakage: a.Tenant=%q b.Tenant=%q", a.Unit.Tenant, b.Unit.Tenant)
	}
	if len(em.events) != 2 {
		t.Errorf("expected 2 audit events (one per decision), got %d", len(em.events))
	}
}

// Fail-closed (ADR 0003 D9): an empty allowlist denies everyone, even a tenant
// that has a registered unit. A misconfigured/empty pool routes nothing.
func TestRouteFailClosedOnEmptyAllowlist(t *testing.T) {
	em := &fakeEmitter{}
	r := NewRouter(PoolConfig{
		AllowedTenants: nil, // empty = deny all
		Units: []ExecutionUnit{
			{Tenant: "team-a", Name: "unit-a", Endpoint: "mcp://unit-a", Tools: []string{"git.clone"}},
		},
	}, em)

	d := r.Route(resources.ActorRef{Atespace: "team-a", Name: "act-1"}, "uid-1", "ns", "tmpl", "git.clone")
	if d.Allowed {
		t.Fatal("empty allowlist must deny even a tenant that has a unit (fail-closed)")
	}
	if ev := em.last(t).a; ev.Allowed {
		t.Error("fail-closed denial must be audited with allowed=false")
	}
}

// A nil emitter must not panic; decisions remain correct. (Production passes a
// real logger; this guards the decision path from being coupled to auditing.)
func TestRouteWithNilEmitterStillDecides(t *testing.T) {
	r := twoTenantPool(nil)

	if d := r.Route(resources.ActorRef{Atespace: "team-a", Name: "act-1"}, "uid", "ns", "tmpl", "git.clone"); !d.Allowed || d.Unit.Name != "unit-a" {
		t.Fatalf("nil emitter broke the allow path: %+v", d)
	}
	if d := r.Route(resources.ActorRef{Atespace: "team-z", Name: "act-1"}, "uid", "ns", "tmpl", "git.clone"); d.Allowed {
		t.Fatal("nil emitter broke the deny path")
	}
}

// The Decision.Unit must be a copy, so a caller mutating it cannot corrupt the
// pool's routing table for subsequent calls.
func TestRouteReturnsUnitCopyNotConfigAlias(t *testing.T) {
	em := &fakeEmitter{}
	r := twoTenantPool(em)

	d := r.Route(resources.ActorRef{Atespace: "team-a", Name: "act-1"}, "uid", "ns", "tmpl", "git.clone")
	if !d.Allowed {
		t.Fatalf("setup: expected allow, got %s", d.Reason)
	}
	d.Unit.Tenant = "evil"
	d.Unit.Tools = append(d.Unit.Tools, "exfil")

	again := r.Route(resources.ActorRef{Atespace: "team-a", Name: "act-2"}, "uid", "ns", "tmpl", "git.clone")
	if !again.Allowed || again.Unit.Tenant != "team-a" {
		t.Fatalf("config aliasing: second route saw mutated unit %+v", again.Unit)
	}
}
