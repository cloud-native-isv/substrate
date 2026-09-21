// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/agent-substrate/substrate/internal/mcppool"
	atev1alpha1 "github.com/agent-substrate/substrate/pkg/api/v1alpha1"
)

const workerPoolFieldOwner = "workerpool-controller"

type WorkerPoolReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	OTelEndpoint string
	// OTelMetricExportInterval is the OTEL_METRIC_EXPORT_INTERVAL propagated to
	// ateom pods. Empty keeps the SDK's default.
	OTelMetricExportInterval string
	// OTelMetricExportTimeout is the OTEL_METRIC_EXPORT_TIMEOUT propagated to
	// ateom pods. Empty keeps the SDK's default.
	OTelMetricExportTimeout string
	// OTelTracesSampler is the OTEL_TRACES_SAMPLER propagated to ateom pods.
	// Empty keeps the ateom binary's default.
	OTelTracesSampler string
	// OTelTracesSamplerArg is the OTEL_TRACES_SAMPLER_ARG propagated to ateom
	// pods. Ignored unless OTelTracesSampler is set.
	OTelTracesSamplerArg string
	// WorkerCertSource selects the volume sources for worker pod TLS material.
	// Empty defaults to WorkerCertSourcePodCertificate.
	WorkerCertSource WorkerCertSource
}

//+kubebuilder:rbac:groups=ate.dev,resources=workerpools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ate.dev,resources=workerpools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ate.dev,resources=workerpools/finalizers,verbs=update
//+kubebuilder:rbac:groups=ate.dev,resources=mcppools,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *WorkerPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch worker pool
	wp := &atev1alpha1.WorkerPool{}
	if err := r.Get(ctx, req.NamespacedName, wp); err != nil {
		if k8errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get worker pool %q: %w", req.NamespacedName, err)
	}

	// Handle deletion
	if !wp.GetDeletionTimestamp().IsZero() {
		log.Info("WorkerPool is being deleted")
		return ctrl.Result{}, nil
	}

	if err := r.reconcileWorkerPool(ctx, wp); err != nil {
		log.Error(err, "Failed to reconcile worker pool")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *WorkerPoolReconciler) reconcileWorkerPool(ctx context.Context, wp *atev1alpha1.WorkerPool) error {
	log := log.FromContext(ctx)
	log.Info("Reconciling worker pool")

	if err := r.applyDeployment(ctx, wp); err != nil {
		return err
	}

	dep := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: wp.Name, Namespace: wp.Namespace}, dep); err != nil {
		if k8errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	return r.syncStatus(ctx, wp, dep)
}

func (r *WorkerPoolReconciler) applyDeployment(ctx context.Context, wp *atev1alpha1.WorkerPool) error {
	certSource := r.WorkerCertSource
	if certSource == "" {
		certSource = WorkerCertSourcePodCertificate
	}
	mcpBackend := r.resolveMcpBackend(ctx, wp)
	depAC := buildDeploymentApplyConfig(wp, ateomOTelSettings{
		Endpoint:             r.OTelEndpoint,
		MetricExportInterval: r.OTelMetricExportInterval,
		MetricExportTimeout:  r.OTelMetricExportTimeout,
		TracesSampler:        r.OTelTracesSampler,
		TracesSamplerArg:     r.OTelTracesSamplerArg,
	}, certSource, mcpBackend)
	if err := r.Apply(ctx, depAC, client.FieldOwner(workerPoolFieldOwner), client.ForceOwnership); err != nil {
		return fmt.Errorf("failed to apply Deployment: %w", err)
	}
	return nil
}

// resolveMcpBackend resolves the cross-S shared MCP execution-unit endpoint for
// this worker pool's tenant from the referenced McpPool (F10 Stage B P-side
// delivery). Returns "" — projecting no MCP backend, fail-closed — when no
// McpPoolRef is set, the McpPool is missing, or the tenant has no admitted unit
// (R3 allowlist / no unit). A worker pod is single-tenant (one S domain = one
// digital employee = one atespace, trust-domain-panorama §2), so every actor it
// hosts shares this one endpoint; per-tool enforcement stays at the S-layer
// mediator (contract + R3 scope) and the unit's Tools allowlist.
func (r *WorkerPoolReconciler) resolveMcpBackend(ctx context.Context, wp *atev1alpha1.WorkerPool) string {
	if wp.Spec.McpPoolRef == "" {
		return ""
	}
	log := log.FromContext(ctx)
	pool := &atev1alpha1.McpPool{}
	// McpPool is cluster-scoped, so only Name is set.
	if err := r.Get(ctx, types.NamespacedName{Name: wp.Spec.McpPoolRef}, pool); err != nil {
		log.Error(err, "cannot get McpPool for worker MCP backend; projecting none (fail-closed)",
			"mcpPool", wp.Spec.McpPoolRef)
		return ""
	}
	cfg := mcppool.PoolConfigFromSpec(&pool.Spec)
	unit := cfg.ResolveUnitForTenant(wp.Spec.McpAtespace)
	if unit == nil {
		log.Info("no admitted MCP execution unit for tenant; projecting no MCP backend (fail-closed)",
			"atespace", wp.Spec.McpAtespace, "mcpPool", wp.Spec.McpPoolRef)
		return ""
	}
	return mcpBackendURL(unit.Endpoint)
}

// mcpBackendURL renders an execution-unit endpoint as the WASM_MCP_BACKEND value
// consumed by ateom-wasmd (sandbox S10). cross-S pool units are reached over mTLS
// (I-3 mandatory), so a bare host:port becomes tls://host:port; an endpoint that
// already carries a scheme is passed through unchanged.
func mcpBackendURL(endpoint string) string {
	if strings.Contains(endpoint, "://") {
		return endpoint
	}
	return "tls://" + endpoint
}

func (r *WorkerPoolReconciler) syncStatus(ctx context.Context, wp *atev1alpha1.WorkerPool, dep *appsv1.Deployment) error {
	selector, err := metav1.LabelSelectorAsSelector(dep.Spec.Selector)
	if err != nil {
		return fmt.Errorf("failed to convert Deployment selector: %w", err)
	}

	want := atev1alpha1.WorkerPoolStatus{
		Replicas: dep.Status.Replicas,
		Selector: selector.String(),
	}
	if equality.Semantic.DeepEqual(wp.Status, want) {
		return nil
	}

	wp.Status = want
	if err := r.Status().Update(ctx, wp); err != nil {
		return fmt.Errorf("failed to update WorkerPool status: %w", err)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkerPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&atev1alpha1.WorkerPool{}).
		Owns(&appsv1.Deployment{}).
		// A WorkerPool that borrows from a cross-S MCP pool (spec.mcpPoolRef)
		// projects the resolved per-tenant endpoint as WASM_MCP_BACKEND. Watch
		// the referenced McpPool so an operator change to a unit endpoint
		// re-reconciles every referencing worker; without this the pods keep a
		// stale backend until the WorkerPool itself changes.
		Watches(&atev1alpha1.McpPool{}, handler.EnqueueRequestsFromMapFunc(r.mcpPoolToWorkerPools)).
		Complete(r)
}

// mcpPoolToWorkerPools maps a changed McpPool to reconcile requests for every
// WorkerPool that references it by name (spec.mcpPoolRef), across all
// namespaces. McpPool is cluster-scoped, so referencing workers may live in any
// namespace. A List error yields no requests (the periodic resync still
// re-reconciles eventually); this only affects how promptly the change lands.
func (r *WorkerPoolReconciler) mcpPoolToWorkerPools(ctx context.Context, obj client.Object) []reconcile.Request {
	var pools atev1alpha1.WorkerPoolList
	if err := r.List(ctx, &pools); err != nil {
		log.FromContext(ctx).Error(err, "cannot list WorkerPools for McpPool watch; skipping re-reconcile",
			"mcpPool", obj.GetName())
		return nil
	}
	var reqs []reconcile.Request
	for i := range pools.Items {
		wp := &pools.Items[i]
		if wp.Spec.McpPoolRef == obj.GetName() {
			reqs = append(reqs, reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      wp.Name,
				Namespace: wp.Namespace,
			}})
		}
	}
	return reqs
}
