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

	"github.com/agent-substrate/substrate/pkg/api/v1alpha1"
)

// PoolConfigFromSpec projects a declared McpPool spec (the CRD, F10 deployment
// layer) into the router's immutable PoolConfig. It is a pure 1:1 mapping of the
// allowlist and the execution units — the CRD is the declarative source of truth,
// PoolConfig is the value handed to NewRouter. Slices are cloned so the resulting
// config never aliases the CRD object's backing arrays (a caller mutating the
// spec, or a lister cache reusing it, cannot corrupt a live router's table; this
// mirrors the copy-on-return discipline of Router.resolveUnit).
func PoolConfigFromSpec(spec *v1alpha1.McpPoolSpec) PoolConfig {
	if spec == nil {
		return PoolConfig{}
	}
	cfg := PoolConfig{AllowedTenants: slices.Clone(spec.AllowedTenants)}
	if len(spec.Units) > 0 {
		cfg.Units = make([]ExecutionUnit, 0, len(spec.Units))
		for _, u := range spec.Units {
			cfg.Units = append(cfg.Units, ExecutionUnit{
				Tenant:   u.Tenant,
				Name:     u.Name,
				Endpoint: u.Endpoint,
				Tools:    slices.Clone(u.Tools),
			})
		}
	}
	return cfg
}

// RouterFromMcpPool builds a Router enforcing the pool declared by an McpPool CRD.
// audit may be nil for decision-only use (tests); production MUST pass a real
// emitter so every routing decision joins the F12 cross-domain audit plane (C2).
func RouterFromMcpPool(pool *v1alpha1.McpPool, audit AuditEmitter) *Router {
	if pool == nil {
		return NewRouter(PoolConfig{}, audit)
	}
	return NewRouter(PoolConfigFromSpec(&pool.Spec), audit)
}
