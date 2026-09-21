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

	"github.com/agent-substrate/substrate/internal/resources"
	"github.com/agent-substrate/substrate/pkg/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// declaredPool is the CRD-shaped fixture mirroring twoTenantPool: tenants team-a
// and team-b admitted, each with its own per-tenant unit (team-a's also serves
// compile.file). It is the declarative source PoolConfigFromSpec projects.
func declaredPool() *v1alpha1.McpPool {
	return &v1alpha1.McpPool{
		ObjectMeta: metav1.ObjectMeta{Name: "shared-mcp"},
		Spec: v1alpha1.McpPoolSpec{
			AllowedTenants: []string{"team-a", "team-b"},
			Units: []v1alpha1.McpExecutionUnit{
				{Tenant: "team-a", Name: "unit-a", Endpoint: "mcp://unit-a", Tools: []string{"git.clone", "compile.file"}},
				{Tenant: "team-b", Name: "unit-b", Endpoint: "mcp://unit-b", Tools: []string{"git.clone"}},
			},
		},
	}
}

// The projection is a faithful 1:1 mapping of the declaration into PoolConfig:
// allowlist and every unit field (tenant/name/endpoint/tools) carried over.
func TestPoolConfigFromSpecIsFaithful(t *testing.T) {
	cfg := PoolConfigFromSpec(&declaredPool().Spec)

	if got := cfg.AllowedTenants; len(got) != 2 || got[0] != "team-a" || got[1] != "team-b" {
		t.Fatalf("AllowedTenants = %v, want [team-a team-b]", got)
	}
	if len(cfg.Units) != 2 {
		t.Fatalf("len(Units) = %d, want 2", len(cfg.Units))
	}
	ua := cfg.Units[0]
	if ua.Tenant != "team-a" || ua.Name != "unit-a" || ua.Endpoint != "mcp://unit-a" {
		t.Errorf("unit[0] = %+v, want team-a/unit-a/mcp://unit-a", ua)
	}
	if len(ua.Tools) != 2 || ua.Tools[0] != "git.clone" || ua.Tools[1] != "compile.file" {
		t.Errorf("unit[0].Tools = %v, want [git.clone compile.file]", ua.Tools)
	}
	if ub := cfg.Units[1]; ub.Tenant != "team-b" || ub.Endpoint != "mcp://unit-b" || len(ub.Tools) != 1 {
		t.Errorf("unit[1] = %+v, want team-b/mcp://unit-b/[git.clone]", ub)
	}
}

// The projected config must not alias the CRD's slices: mutating the source spec
// (or a lister cache reusing it) after projection cannot corrupt the config. This
// is why PoolConfigFromSpec clones.
func TestPoolConfigFromSpecDoesNotAliasSource(t *testing.T) {
	spec := &declaredPool().Spec
	cfg := PoolConfigFromSpec(spec)

	// Mutate the source declaration in place.
	spec.AllowedTenants[0] = "evil"
	spec.Units[0].Tools[0] = "exfil"
	spec.Units[0].Endpoint = "mcp://attacker"

	if cfg.AllowedTenants[0] != "team-a" {
		t.Errorf("AllowedTenants aliased: got %q, want team-a", cfg.AllowedTenants[0])
	}
	if cfg.Units[0].Tools[0] != "git.clone" {
		t.Errorf("unit Tools aliased: got %q, want git.clone", cfg.Units[0].Tools[0])
	}
	if cfg.Units[0].Endpoint != "mcp://unit-a" {
		t.Errorf("unit Endpoint aliased: got %q, want mcp://unit-a", cfg.Units[0].Endpoint)
	}
}

// An empty/zero declaration projects to an empty PoolConfig, which fails closed:
// the router denies everyone (ADR 0003 D9). A nil spec is tolerated (zero config).
func TestPoolConfigFromSpecEmptyFailsClosed(t *testing.T) {
	if cfg := PoolConfigFromSpec(nil); len(cfg.AllowedTenants) != 0 || len(cfg.Units) != 0 {
		t.Fatalf("nil spec must project to zero PoolConfig, got %+v", cfg)
	}
	if cfg := PoolConfigFromSpec(&v1alpha1.McpPoolSpec{}); len(cfg.AllowedTenants) != 0 || len(cfg.Units) != 0 {
		t.Fatalf("empty spec must project to zero PoolConfig, got %+v", cfg)
	}

	// A router built from an empty declaration denies even a tenant that would
	// have a unit elsewhere (fail-closed on the empty allowlist).
	r := NewRouter(PoolConfigFromSpec(&v1alpha1.McpPoolSpec{}), nil)
	if d := r.Route(resources.ActorRef{Atespace: "team-a", Name: "act-1"}, "uid", "ns", "tmpl", "git.clone"); d.Allowed {
		t.Fatal("empty declaration must deny everyone (fail-closed)")
	}
}

// RouterFromMcpPool wires the declaration straight into a working router: the
// admitted tenant reaches its own unit, an unadmitted tenant is denied, and the
// per-tenant boundary holds — i.e. the CRD declaration alone is sufficient to
// drive the full F10 routing decision (no hand-built PoolConfig).
func TestRouterFromMcpPoolDrivesRouting(t *testing.T) {
	em := &fakeEmitter{}
	r := RouterFromMcpPool(declaredPool(), em)

	// Admitted tenant -> own unit, audited allowed.
	if d := r.Route(resources.ActorRef{Atespace: "team-a", Name: "act-a"}, "uid-a", "ns", "tmpl", "compile.file"); !d.Allowed || d.Unit == nil || d.Unit.Name != "unit-a" {
		t.Fatalf("team-a compile.file -> %+v, want allowed unit-a", d)
	}
	// Unadmitted tenant -> denied (R3), audited.
	if d := r.Route(resources.ActorRef{Atespace: "team-c", Name: "act-c"}, "uid-c", "ns", "tmpl", "git.clone"); d.Allowed {
		t.Fatal("team-c is not admitted; must be denied")
	}
	// Cross-tenant tool: team-b's unit does not serve compile.file -> denied.
	if d := r.Route(resources.ActorRef{Atespace: "team-b", Name: "act-b"}, "uid-b", "ns", "tmpl", "compile.file"); d.Allowed {
		t.Fatal("team-b unit does not serve compile.file; must be denied")
	}
	if len(em.events) != 3 {
		t.Errorf("expected 3 audit events (one per decision), got %d", len(em.events))
	}
}

// A nil McpPool yields a fail-closed router rather than panicking, so a missing
// declaration is safe (denies all) at the serving seam.
func TestRouterFromMcpPoolNilIsFailClosed(t *testing.T) {
	r := RouterFromMcpPool(nil, nil)
	if d := r.Route(resources.ActorRef{Atespace: "team-a", Name: "act-1"}, "uid", "ns", "tmpl", "git.clone"); d.Allowed {
		t.Fatal("nil pool must deny everyone (fail-closed)")
	}
}
