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

import "slices"

// This file is the F11 provisioning DECISION core: the pure, cluster-free logic
// that converges the live set of per-tenant MCP execution units toward the
// declared PoolConfig. It is the F11 counterpart to Router (the F10 routing
// decision core) — same discipline: dependency-light, unit-testable without a
// cluster, fail-closed, copy-safe.
//
// The trust-domain panorama (docs/concepts/trust-domain-panorama.md §3.4, C1)
// fixes the model this implements: a cross-S shared pool runs ONE isolated
// execution unit PER TENANT ("共享池、隔离执行单元"), each a microvm/gvisor
// instance (OS multi-user is NOT a cross-tenant boundary), DISPOSABLE — "用完即毁、
// 不复用" — and the P orchestrator "按租户拉起/销毁隔离执行单元". PlanProvisioning
// is that 拉起/销毁 decision; the F11 controller (cluster-gated: it creates the
// microvm actors via the substrate control plane and tears them down) executes the
// plan. Keeping the decision here, separate from the gated reconcile wiring, is
// what makes C1/R3 provisioning verifiable independent of a cluster.

// ProvisionPlan is the F11 provisioning decision: which per-tenant execution
// units to create, destroy, or keep so the live pool converges on the declared
// config under C1 (per-tenant isolated, disposable units) and R3 (a tenant that
// is not — or is no longer — admitted keeps no execution unit).
type ProvisionPlan struct {
	// Create lists declared, admitted units that are not yet live, to provision as
	// fresh per-tenant microvm execution units. Each is a copy (no alias into
	// PoolConfig.Units), mirroring the Router's copy discipline.
	Create []ExecutionUnit
	// Destroy lists live unit names to tear down: deregistered (no longer declared)
	// or whose tenant is no longer admitted (R3). Disposable units are destroyed and
	// re-created rather than updated in place (C1 用完即毁). Sorted for determinism.
	Destroy []string
	// Keep lists live unit names that already match a declared, admitted unit.
	Keep []string
}

// PlanProvisioning computes the convergence plan for the declared pool config
// against observed, the names of execution units currently provisioned. It is a
// pure decision: no cluster side effects and no audit emission (the audited
// cross-domain events are the per-call Route decisions on the P side and the
// mediator's mcp.call/mcp.blocked on the S side; provisioning is lifecycle, not a
// capability invocation).
//
// Deny-by-default discipline, mirroring Route / ResolveUnitForTenant:
//
//   - a declared unit whose tenant is NOT admitted is never created, and if it is
//     somehow live it is destroyed — a de-admitted tenant keeps no execution unit
//     (R3 confused-deputy defense at the provisioning layer);
//   - a live unit no longer declared is destroyed (deregistered / disposable, C1);
//   - an empty AllowedTenants admits no one, so nothing is created and every live
//     unit is destroyed (fail-closed, ADR 0003 D9).
//
// Endpoint/ready drift of a live unit (same name, changed endpoint) is NOT a
// create/destroy concern here — the controller re-derives live state each resync
// and a disposable unit is replaced by deregister-then-declare, not patched.
func (c PoolConfig) PlanProvisioning(observed []string) ProvisionPlan {
	live := make(map[string]struct{}, len(observed))
	for _, name := range observed {
		live[name] = struct{}{}
	}

	var plan ProvisionPlan
	declared := make(map[string]struct{}, len(c.Units))
	for _, u := range c.Units {
		declared[u.Name] = struct{}{}
		_, isLive := live[u.Name]

		// Gate: tenant allowlist (R3). An unadmitted tenant gets no unit — never
		// provisioned, and torn down if one is somehow live.
		if !slices.Contains(c.AllowedTenants, u.Tenant) {
			if isLive {
				plan.Destroy = append(plan.Destroy, u.Name)
			}
			continue
		}
		if isLive {
			plan.Keep = append(plan.Keep, u.Name)
			continue
		}
		unit := u
		unit.Tools = slices.Clone(u.Tools)
		plan.Create = append(plan.Create, unit)
	}

	// Live units no longer declared → destroy (deregistered / disposable, C1).
	for name := range live {
		if _, ok := declared[name]; !ok {
			plan.Destroy = append(plan.Destroy, name)
		}
	}
	slices.Sort(plan.Destroy)
	return plan
}
