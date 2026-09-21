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
	"slices"
	"testing"
)

func createNames(p ProvisionPlan) []string {
	names := make([]string, 0, len(p.Create))
	for _, u := range p.Create {
		names = append(names, u.Name)
	}
	return names
}

// TestPlanProvisioningCreatesAdmittedDeclaredUnits: with nothing live, every
// declared unit whose tenant is admitted is provisioned (C1 per-tenant units).
func TestPlanProvisioningCreatesAdmittedDeclaredUnits(t *testing.T) {
	cfg := PoolConfig{
		AllowedTenants: []string{"team-a", "team-b"},
		Units: []ExecutionUnit{
			{Tenant: "team-a", Name: "unit-a", Endpoint: "a:8443", Tools: []string{"host.ident"}},
			{Tenant: "team-b", Name: "unit-b", Endpoint: "b:8443", Tools: []string{"host.ident"}},
		},
	}
	got := cfg.PlanProvisioning(nil)
	if !slices.Equal(createNames(got), []string{"unit-a", "unit-b"}) {
		t.Errorf("Create = %v, want [unit-a unit-b]", createNames(got))
	}
	if len(got.Destroy) != 0 || len(got.Keep) != 0 {
		t.Errorf("Destroy = %v, Keep = %v, want both empty", got.Destroy, got.Keep)
	}
}

// TestPlanProvisioningKeepsLiveUnits: an admitted declared unit already live is
// kept, not re-created; the not-yet-live sibling is still created.
func TestPlanProvisioningKeepsLiveUnits(t *testing.T) {
	cfg := PoolConfig{
		AllowedTenants: []string{"team-a", "team-b"},
		Units: []ExecutionUnit{
			{Tenant: "team-a", Name: "unit-a", Endpoint: "a:8443"},
			{Tenant: "team-b", Name: "unit-b", Endpoint: "b:8443"},
		},
	}
	got := cfg.PlanProvisioning([]string{"unit-a"})
	if !slices.Equal(got.Keep, []string{"unit-a"}) {
		t.Errorf("Keep = %v, want [unit-a]", got.Keep)
	}
	if !slices.Equal(createNames(got), []string{"unit-b"}) {
		t.Errorf("Create = %v, want [unit-b]", createNames(got))
	}
	if len(got.Destroy) != 0 {
		t.Errorf("Destroy = %v, want empty", got.Destroy)
	}
}

// TestPlanProvisioningDestroysDeregisteredUnits: a live unit no longer declared
// is torn down (disposable, C1 用完即毁).
func TestPlanProvisioningDestroysDeregisteredUnits(t *testing.T) {
	cfg := PoolConfig{
		AllowedTenants: []string{"team-a"},
		Units:          []ExecutionUnit{{Tenant: "team-a", Name: "unit-a", Endpoint: "a:8443"}},
	}
	got := cfg.PlanProvisioning([]string{"unit-a", "unit-old"})
	if !slices.Equal(got.Keep, []string{"unit-a"}) {
		t.Errorf("Keep = %v, want [unit-a]", got.Keep)
	}
	if !slices.Equal(got.Destroy, []string{"unit-old"}) {
		t.Errorf("Destroy = %v, want [unit-old]", got.Destroy)
	}
	if len(got.Create) != 0 {
		t.Errorf("Create = %v, want empty", createNames(got))
	}
}

// TestPlanProvisioningDestroysDeAdmittedTenantUnit (R3): a unit whose tenant has
// been removed from the allowlist is torn down and never re-created — a
// de-admitted tenant keeps no execution unit.
func TestPlanProvisioningDestroysDeAdmittedTenantUnit(t *testing.T) {
	cfg := PoolConfig{
		AllowedTenants: []string{"team-a"}, // team-b de-admitted
		Units: []ExecutionUnit{
			{Tenant: "team-a", Name: "unit-a", Endpoint: "a:8443"},
			{Tenant: "team-b", Name: "unit-b", Endpoint: "b:8443"},
		},
	}
	got := cfg.PlanProvisioning([]string{"unit-a", "unit-b"})
	if !slices.Equal(got.Keep, []string{"unit-a"}) {
		t.Errorf("Keep = %v, want [unit-a]", got.Keep)
	}
	if !slices.Equal(got.Destroy, []string{"unit-b"}) {
		t.Errorf("Destroy = %v, want [unit-b] (de-admitted tenant)", got.Destroy)
	}
	if len(got.Create) != 0 {
		t.Errorf("Create = %v, want empty (never provision for de-admitted tenant)", createNames(got))
	}
}

// TestPlanProvisioningNeverCreatesUnadmittedUnit: an unadmitted tenant's declared
// unit that is not live is simply skipped — neither created nor destroyed.
func TestPlanProvisioningNeverCreatesUnadmittedUnit(t *testing.T) {
	cfg := PoolConfig{
		AllowedTenants: []string{"team-a"},
		Units: []ExecutionUnit{
			{Tenant: "team-a", Name: "unit-a", Endpoint: "a:8443"},
			{Tenant: "team-b", Name: "unit-b", Endpoint: "b:8443"},
		},
	}
	got := cfg.PlanProvisioning(nil)
	if !slices.Equal(createNames(got), []string{"unit-a"}) {
		t.Errorf("Create = %v, want [unit-a] only", createNames(got))
	}
	if len(got.Destroy) != 0 || len(got.Keep) != 0 {
		t.Errorf("Destroy = %v, Keep = %v, want both empty", got.Destroy, got.Keep)
	}
}

// TestPlanProvisioningEmptyAllowlistFailClosed (ADR 0003 D9): an empty allowlist
// admits no one, so nothing is created and every live unit is destroyed.
func TestPlanProvisioningEmptyAllowlistFailClosed(t *testing.T) {
	cfg := PoolConfig{
		AllowedTenants: nil,
		Units:          []ExecutionUnit{{Tenant: "team-a", Name: "unit-a", Endpoint: "a:8443"}},
	}
	got := cfg.PlanProvisioning([]string{"unit-a"})
	if len(got.Create) != 0 {
		t.Errorf("Create = %v, want empty (fail-closed)", createNames(got))
	}
	if !slices.Equal(got.Destroy, []string{"unit-a"}) {
		t.Errorf("Destroy = %v, want [unit-a] (fail-closed tears down live units)", got.Destroy)
	}
}

// TestPlanProvisioningCreateIsCopy: Create units are copies, so a caller mutating
// the plan cannot alias-corrupt the pool config (Router copy discipline).
func TestPlanProvisioningCreateIsCopy(t *testing.T) {
	cfg := PoolConfig{
		AllowedTenants: []string{"team-a"},
		Units:          []ExecutionUnit{{Tenant: "team-a", Name: "unit-a", Endpoint: "a:8443", Tools: []string{"t1"}}},
	}
	plan := cfg.PlanProvisioning(nil)
	if len(plan.Create) != 1 {
		t.Fatalf("Create = %v, want one unit", createNames(plan))
	}
	plan.Create[0].Tools[0] = "MUTATED"
	plan.Create[0].Endpoint = "MUTATED"
	if cfg.Units[0].Tools[0] != "t1" || cfg.Units[0].Endpoint != "a:8443" {
		t.Errorf("PlanProvisioning aliased cfg.Units: tools=%v endpoint=%q", cfg.Units[0].Tools, cfg.Units[0].Endpoint)
	}
}

// TestPlanProvisioningDestroyDeterministic: Destroy is sorted even though the
// deregistered-unit pass iterates a map (non-deterministic order).
func TestPlanProvisioningDestroyDeterministic(t *testing.T) {
	cfg := PoolConfig{AllowedTenants: []string{"team-a"}} // no declared units
	got := cfg.PlanProvisioning([]string{"unit-z", "unit-a", "unit-m"})
	if !slices.Equal(got.Destroy, []string{"unit-a", "unit-m", "unit-z"}) {
		t.Errorf("Destroy = %v, want sorted [unit-a unit-m unit-z]", got.Destroy)
	}
}
