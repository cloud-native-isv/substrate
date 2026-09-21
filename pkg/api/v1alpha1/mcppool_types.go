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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// McpExecutionUnit declares one registered MCP execution unit in the cross-S
// shared capability pool: the disposable, untrusted MCP server instance (F11)
// that actually runs a native capability on behalf of exactly one tenant. It is
// the declarative (CRD) counterpart of mcppool.ExecutionUnit; the projection in
// internal/mcppool maps it 1:1 into the router's routing table.
type McpExecutionUnit struct {
	// Tenant is the atespace this unit is provisioned for. A unit never serves a
	// tenant other than its own (per-tenant isolation, C1 / R3): OS multi-user is
	// NOT a cross-tenant boundary, a per-tenant microvm/gvisor unit is.
	//
	// +required
	// +kubebuilder:validation:MinLength=1
	Tenant string `json:"tenant"`

	// Name identifies the unit within the pool (audit + routing diagnostics).
	//
	// +required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Endpoint is where the unit's MCP server listens, reached over mTLS by the
	// S-layer ateom mediator (I-3 boundary). The P router returns it but never
	// dials it itself.
	//
	// +required
	// +kubebuilder:validation:MinLength=1
	Endpoint string `json:"endpoint"`

	// Tools is the allowlist of MCP tool names this unit serves. A tool not
	// listed here is not routable to this unit (deny-by-default, ADR 0007 D7:
	// narrow, parameterized tools; capability scoping per tenant).
	//
	// +optional
	Tools []string `json:"tools,omitempty"`
}

// McpPoolSpec is the desired state of an McpPool: the P-layer shared-pool policy
// (cross-S "公用 MCP 能力后端" orchestration, C2). It is the declarative source
// the router's immutable PoolConfig is projected from.
type McpPoolSpec struct {
	// AllowedTenants is the cross-S pool allowlist: which atespaces may use the
	// shared pool at all. Empty denies everyone (fail-closed, ADR 0003 D9). This
	// is the R3 defense — a tenant's actor is never routed unless its tenant is
	// explicitly admitted, which prevents a cross-tenant confused-deputy. It is
	// the pool-side counterpart to the S-layer per-tenant capability scoping (S10).
	//
	// +optional
	AllowedTenants []string `json:"allowedTenants,omitempty"`

	// Units are the registered execution units (F11 provisions and deregisters
	// them; a one-shot unit is removed after use).
	//
	// +optional
	Units []McpExecutionUnit `json:"units,omitempty"`
}

// McpPool is cluster-scoped configuration declaring the cross-S shared MCP
// capability pool — the P-trusted control point of the four-trust-domain design
// (docs/concepts/trust-domain-panorama.md §3.4/§3.5, ADR 0007 D4, condition C2).
// An agent (Agent domain) invokes a native capability through the mcp_call host
// function, mediated by ateom in the worker pod (S domain, the enforcement
// point); when the capability is served by a cross-S shared pool rather than a
// within-S local unit, this declaration feeds the deny-by-default, audited
// routing decision in internal/mcppool. Cluster-scoped and name-referenced like
// SandboxConfig (a pool spans tenants, so it is not namespaced to one).
//
// +genclient
// +genclient:nonNamespaced
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=mcppool
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type McpPool struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of McpPool
	// +required
	Spec McpPoolSpec `json:"spec"`
}

// McpPoolList contains a list of McpPools.
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
type McpPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []McpPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&McpPool{}, &McpPoolList{})
}
