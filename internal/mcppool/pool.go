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

// Package mcppool implements the P-layer (platform) routing core of the cross-S
// MCP shared capability pool — the "公用 MCP 能力后端" orchestration point of the
// four-trust-domain panorama (docs/concepts/trust-domain-panorama.md §3.4/§3.5,
// ADR 0007 D4, self-consistency condition C2).
//
// Trust model. An agent (Agent domain, least trusted, no secrets, no raw socket)
// invokes a native capability through the mcp_call host function; the call is
// mediated by ateom inside the worker pod (S domain, the trusted enforcement
// point). When the capability is served by a cross-S shared pool rather than a
// within-S local unit, the pool's routing layer is the P-trusted control point:
// hardened, deny-by-default, and itself audited. This package is that decision
// core — it answers "may this tenant's actor route this MCP tool to a shared
// execution unit, and which one?" and emits one F12 cross-domain audit event per
// decision so the pool's own routing joins the unified cross-domain audit plane.
//
// Scope. It is deliberately a pure, dependency-light library. Provisioning the
// execution units themselves (per-tenant microvm/gvisor, C1) is F11; the mTLS
// transport from ateom into the pool is the S10 seam (sandbox repo). Isolating
// the trust decision here — unit-testable without a cluster — is what makes the
// P layer "可信、硬化、自身被审计" (C2) verifiable independent of deployment
// wiring, and it bounds the cross-tenant blast radius (R2) to one audited gate.
package mcppool

import (
	"slices"

	"github.com/agent-substrate/substrate/internal/actorlog"
	"github.com/agent-substrate/substrate/internal/resources"
)

// ExecutionUnit is a registered MCP execution unit: the disposable, untrusted
// MCP server instance (F11) that actually runs a native capability. Units in the
// cross-S shared pool are per-tenant — a unit serves exactly one tenant's actors
// — so cross-tenant isolation never rests on the unit's own OS accounting (C1:
// OS multi-user is NOT a cross-tenant boundary; a per-tenant microvm/gvisor unit
// is). The unit is untrusted (信任序 ateom > 执行单元 > agent); the router treats
// it as a routing target only and never executes capability code itself.
type ExecutionUnit struct {
	// Tenant is the atespace this unit is provisioned for. A unit never serves a
	// tenant other than its own (per-tenant isolation, C1 / R3).
	Tenant string
	// Name identifies the unit within the pool (audit + routing diagnostics).
	Name string
	// Endpoint is where the unit's MCP server listens, reached over mTLS by the
	// S-layer ateom mediator. The P router returns it but does not dial it.
	Endpoint string
	// Tools is the allowlist of MCP tool names this unit serves. A tool not
	// listed here is not routable to this unit (deny-by-default, ADR 0007 D7:
	// narrow, parameterized tools; capability scoping per tenant).
	Tools []string
}

// serves reports whether the unit may run tool on behalf of tenant.
func (u ExecutionUnit) serves(tenant, tool string) bool {
	return u.Tenant == tenant && slices.Contains(u.Tools, tool)
}

// PoolConfig is the P-layer shared-pool policy. It is immutable once handed to
// NewRouter; mutating it afterward has no effect on routing decisions.
type PoolConfig struct {
	// AllowedTenants is the cross-S pool allowlist: which atespaces may use the
	// shared pool at all. Empty denies everyone (fail-closed, ADR 0003 D9). This
	// is the R3 defense — a tenant's actor is never routed unless its tenant is
	// explicitly admitted, which is what prevents a cross-tenant confused-deputy
	// (an actor tricking the pool into acting for a tenant it does not belong
	// to). It is the pool-side counterpart to the S-layer per-tenant capability
	// scoping in S10.
	AllowedTenants []string
	// Units are the registered execution units (F11 provisions and deregisters
	// them; a one-shot unit is removed after use).
	Units []ExecutionUnit
}

// Decision is the routing verdict for one MCP call.
type Decision struct {
	// Allowed is true iff the call may proceed to Unit.
	Allowed bool
	// Unit is the execution unit to route to; non-nil iff Allowed.
	Unit *ExecutionUnit
	// Reason explains a denial; non-empty iff !Allowed.
	Reason string
}

// AuditEmitter is the cross-domain audit sink (F12). *actorlog.ActorLogger
// satisfies it; tests inject a recording fake. Routing through this interface
// keeps the trust decision testable without a logger backend, while production
// wiring passes the real logger so the P layer is "自身被审计" (C2).
type AuditEmitter interface {
	EmitCrossDomainAudit(a actorlog.CrossDomainAudit, actorRef resources.ActorRef, actorUID, actorTemplateNamespace, actorTemplateName string)
}

// Router is the P-trusted MCP shared-pool routing layer (F10, C2). Its config is
// immutable after construction, so a Router is safe for concurrent use by the
// many S-layer mediators that call into one shared pool.
type Router struct {
	cfg   PoolConfig
	audit AuditEmitter
}

// NewRouter returns a Router enforcing cfg. audit may be nil, in which case no
// audit events are emitted (useful for decision-only unit tests); in production
// it MUST be non-nil so every routing decision is audited (C2).
func NewRouter(cfg PoolConfig, audit AuditEmitter) *Router {
	return &Router{cfg: cfg, audit: audit}
}

// Route decides whether ref's actor may call tool through the shared pool, and to
// which execution unit. It is deny-by-default at two gates:
//
//  1. tenant allowlist — ref.Atespace must be an admitted pool tenant (R3);
//  2. unit resolution — a unit owned by ref.Atespace must serve tool (C1/D7).
//
// Every decision, allow or deny, emits exactly one F12 cross-domain audit event
// with src_domain=S (the calling worker pod's ateom mediator) and dst_domain=MCP
// (the shared-pool execution unit). actorUID / actorTemplate* are carried only
// as audit identity labels; the router does not otherwise interpret them.
func (r *Router) Route(ref resources.ActorRef, actorUID, actorTemplateNamespace, actorTemplateName, tool string) Decision {
	// Gate 1: tenant allowlist (deny-by-default; R3 confused-deputy defense).
	if !slices.Contains(r.cfg.AllowedTenants, ref.Atespace) {
		return r.deny(ref, actorUID, actorTemplateNamespace, actorTemplateName, tool,
			"tenant not admitted to shared MCP pool")
	}
	// Gate 2: resolve a per-tenant unit serving the tool. resolveUnit only ever
	// matches units whose Tenant == ref.Atespace, so one tenant can never be
	// routed onto another tenant's execution unit (C1).
	unit := r.resolveUnit(ref.Atespace, tool)
	if unit == nil {
		return r.deny(ref, actorUID, actorTemplateNamespace, actorTemplateName, tool,
			"no execution unit serves tool for tenant")
	}
	r.emit(ref, actorUID, actorTemplateNamespace, actorTemplateName, tool, true, "", unit)
	return Decision{Allowed: true, Unit: unit}
}

// resolveUnit returns a copy of the first unit owned by tenant that serves tool,
// or nil. Returning a copy (not a pointer into cfg.Units) keeps callers from
// mutating the pool's routing table through the Decision.
func (r *Router) resolveUnit(tenant, tool string) *ExecutionUnit {
	for _, u := range r.cfg.Units {
		if u.serves(tenant, tool) {
			unit := u
			return &unit
		}
	}
	return nil
}

func (r *Router) deny(ref resources.ActorRef, actorUID, actorTemplateNamespace, actorTemplateName, tool, reason string) Decision {
	r.emit(ref, actorUID, actorTemplateNamespace, actorTemplateName, tool, false, reason, nil)
	return Decision{Allowed: false, Reason: reason}
}

// emit records the F12 cross-domain audit event for one routing decision. The
// field names live in the shared CrossDomainAudit schema (event/src_domain/
// dst_domain/allowed/reason), so a pool routing decision is queryable alongside
// the sandbox-side S13 mcp.call events as one audit plane.
func (r *Router) emit(ref resources.ActorRef, actorUID, actorTemplateNamespace, actorTemplateName, tool string, allowed bool, reason string, unit *ExecutionUnit) {
	if r.audit == nil {
		return
	}
	fields := map[string]any{
		"tool":   tool,
		"tenant": ref.Atespace,
		"pool":   "shared-mcp",
	}
	if unit != nil {
		fields["unit"] = unit.Name
		fields["endpoint"] = unit.Endpoint
	}
	r.audit.EmitCrossDomainAudit(actorlog.CrossDomainAudit{
		Event:     "mcp.pool.route",
		SrcDomain: actorlog.DomainService, // S: the calling worker pod's ateom mediator
		DstDomain: actorlog.DomainMCP,     // MCP: the shared-pool execution unit
		Operation: "route cross-S MCP call to shared-pool execution unit (P-trusted router)",
		Allowed:   allowed,
		Reason:    reason,
		Fields:    fields,
	}, ref, actorUID, actorTemplateNamespace, actorTemplateName)
}
