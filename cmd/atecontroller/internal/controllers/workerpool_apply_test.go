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
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1ac "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	metav1ac "k8s.io/client-go/applyconfigurations/meta/v1"

	"github.com/agent-substrate/substrate/internal/ateompath"
	"github.com/agent-substrate/substrate/internal/resources"
	atev1alpha1 "github.com/agent-substrate/substrate/pkg/api/v1alpha1"
)

func TestBuildDeploymentApplyConfig(t *testing.T) {
	requiredNodeAffinity := &corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{{
				MatchExpressions: []corev1.NodeSelectorRequirement{{
					Key:      "workload",
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{"substrate"},
				}},
			}},
		},
	}
	preferredNodeAffinity := &corev1.NodeAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{{
			Weight: 50,
			Preference: corev1.NodeSelectorTerm{
				MatchExpressions: []corev1.NodeSelectorRequirement{{
					Key:      "disk",
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{"ssd"},
				}},
			},
		}},
	}
	tolerationSeconds := int64(300)
	toleration := corev1.Toleration{
		Key:               "dedicated",
		Operator:          corev1.TolerationOpEqual,
		Value:             "workerpool",
		Effect:            corev1.TaintEffectNoSchedule,
		TolerationSeconds: &tolerationSeconds,
	}

	tests := []struct {
		name string
		wp   *atev1alpha1.WorkerPool
		want *appsv1ac.DeploymentApplyConfiguration
	}{
		{
			name: "default workerpool",
			wp:   testWorkerPoolApplyConfig(nil),
			want: expectedDeploymentApplyConfig(nil),
		},
		{
			name: "with node selector",
			wp: testWorkerPoolApplyConfig(&atev1alpha1.WorkerPoolPodTemplate{
				NodeSelector: map[string]string{
					"accelerator": "gpu",
					"topology":    "high-mem",
				},
			}),
			want: expectedDeploymentApplyConfig(func(podSpecAC *corev1ac.PodSpecApplyConfiguration) {
				podSpecAC.WithNodeSelector(map[string]string{
					"accelerator": "gpu",
					"topology":    "high-mem",
				})
			}),
		},
		{
			name: "with tolerations",
			wp: testWorkerPoolApplyConfig(&atev1alpha1.WorkerPoolPodTemplate{
				Tolerations: []corev1.Toleration{toleration},
			}),
			want: expectedDeploymentApplyConfig(func(podSpecAC *corev1ac.PodSpecApplyConfiguration) {
				podSpecAC.Tolerations = []corev1ac.TolerationApplyConfiguration{
					*corev1ac.Toleration().
						WithKey("dedicated").
						WithOperator(corev1.TolerationOpEqual).
						WithValue("workerpool").
						WithEffect(corev1.TaintEffectNoSchedule).
						WithTolerationSeconds(300),
				}
			}),
		},
		{
			name: "with node affinity",
			wp: testWorkerPoolApplyConfig(&atev1alpha1.WorkerPoolPodTemplate{
				NodeAffinity: requiredNodeAffinity,
			}),
			want: expectedDeploymentApplyConfig(func(podSpecAC *corev1ac.PodSpecApplyConfiguration) {
				podSpecAC.WithAffinity(corev1ac.Affinity().WithNodeAffinity(
					corev1ac.NodeAffinity().WithRequiredDuringSchedulingIgnoredDuringExecution(
						corev1ac.NodeSelector().WithNodeSelectorTerms(
							corev1ac.NodeSelectorTerm().WithMatchExpressions(
								corev1ac.NodeSelectorRequirement().
									WithKey("workload").
									WithOperator(corev1.NodeSelectorOpIn).
									WithValues("substrate"),
							),
						),
					),
				))
			}),
		},
		{
			name: "with priority class name",
			wp: testWorkerPoolApplyConfig(&atev1alpha1.WorkerPoolPodTemplate{
				PriorityClassName: "interactive-workerpool",
			}),
			want: expectedDeploymentApplyConfig(func(podSpecAC *corev1ac.PodSpecApplyConfiguration) {
				podSpecAC.WithPriorityClassName("interactive-workerpool")
			}),
		},
		{
			name: "with resources",
			wp: testWorkerPoolApplyConfig(&atev1alpha1.WorkerPoolPodTemplate{
				Resources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				},
			}),
			want: expectedDeploymentApplyConfig(func(podSpecAC *corev1ac.PodSpecApplyConfiguration) {
				podSpecAC.Containers[0].WithResources(corev1ac.ResourceRequirements().
					WithRequests(corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					}).
					WithLimits(corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					}))
			}),
		},
		{
			name: "with combined scheduling fields",
			wp: testWorkerPoolApplyConfig(&atev1alpha1.WorkerPoolPodTemplate{
				NodeSelector: map[string]string{
					"accelerator": "gpu",
					"topology":    "high-mem",
				},
				Tolerations:       []corev1.Toleration{toleration},
				PriorityClassName: "interactive-workerpool",
				NodeAffinity:      preferredNodeAffinity,
			}),
			want: expectedDeploymentApplyConfig(func(podSpecAC *corev1ac.PodSpecApplyConfiguration) {
				podSpecAC.WithNodeSelector(map[string]string{
					"accelerator": "gpu",
					"topology":    "high-mem",
				})
				podSpecAC.Tolerations = []corev1ac.TolerationApplyConfiguration{
					*corev1ac.Toleration().
						WithKey("dedicated").
						WithOperator(corev1.TolerationOpEqual).
						WithValue("workerpool").
						WithEffect(corev1.TaintEffectNoSchedule).
						WithTolerationSeconds(300),
				}
				podSpecAC.WithPriorityClassName("interactive-workerpool")
				podSpecAC.WithAffinity(corev1ac.Affinity().WithNodeAffinity(
					corev1ac.NodeAffinity().WithPreferredDuringSchedulingIgnoredDuringExecution(
						corev1ac.PreferredSchedulingTerm().
							WithWeight(50).
							WithPreference(corev1ac.NodeSelectorTerm().WithMatchExpressions(
								corev1ac.NodeSelectorRequirement().
									WithKey("disk").
									WithOperator(corev1.NodeSelectorOpIn).
									WithValues("ssd"),
							)),
					),
				))
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildDeploymentApplyConfig(tt.wp, ateomOTelSettings{}, WorkerCertSourcePodCertificate)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("buildDeploymentApplyConfig() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestMicroVMPodShape asserts the micro-VM sandbox class adds the /dev/kvm
// device (volume + container mount) and node placement (nodeSelector +
// toleration on ate.dev/sandboxClass); other classes get none of it.
func TestMicroVMPodShape(t *testing.T) {
	tests := []struct {
		name        string
		class       atev1alpha1.SandboxClass
		wantMicroVM bool
	}{
		{"gvisor default", "", false},
		{"gvisor explicit", atev1alpha1.SandboxClassGvisor, false},
		{"microvm", atev1alpha1.SandboxClassMicroVM, true},
		{"wasm", atev1alpha1.SandboxClassWasm, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := testWorkerPoolApplyConfig(nil)
			wp.Spec.SandboxClass = tt.class
			ps := buildDeploymentApplyConfig(wp, ateomOTelSettings{}, WorkerCertSourcePodCertificate).Spec.Template.Spec

			hasVol := false
			for _, v := range ps.Volumes {
				if v.Name != nil && *v.Name == "dev-kvm" {
					hasVol = true
					if v.HostPath == nil || v.HostPath.Path == nil || *v.HostPath.Path != "/dev/kvm" ||
						v.HostPath.Type == nil || *v.HostPath.Type != corev1.HostPathCharDev {
						t.Errorf("dev-kvm volume = %+v, want /dev/kvm CharDevice", v.HostPath)
					}
				}
			}
			hasMount := false
			for _, c := range ps.Containers {
				for _, m := range c.VolumeMounts {
					if m.MountPath != nil && *m.MountPath == "/dev/kvm" {
						hasMount = true
					}
				}
			}
			_, hasSelector := ps.NodeSelector["ate.dev/sandboxClass"]
			hasTol := false
			for _, tol := range ps.Tolerations {
				if tol.Key != nil && *tol.Key == "ate.dev/sandboxClass" {
					hasTol = true
				}
			}
			if hasVol != tt.wantMicroVM || hasMount != tt.wantMicroVM || hasSelector != tt.wantMicroVM || hasTol != tt.wantMicroVM {
				t.Errorf("microvm shape: vol=%v mount=%v selector=%v toleration=%v, want all %v",
					hasVol, hasMount, hasSelector, hasTol, tt.wantMicroVM)
			}
		})
	}
}

// TestAteomSecurityContextByClass asserts the gVisor worker runs unprivileged
// with the explicit capability set, the micro-VM worker stays privileged, the
// wasm worker drops every capability without adding any, and that an empty
// class defaults to gVisor.
func TestAteomSecurityContextByClass(t *testing.T) {
	tests := []struct {
		name           string
		class          atev1alpha1.SandboxClass
		wantPrivileged bool
		wantDropAll    bool
		wantGvisorCaps bool
		wantUnconfined bool
	}{
		{"gvisor default", "", false, true, true, true},
		{"gvisor explicit", atev1alpha1.SandboxClassGvisor, false, true, true, true},
		{"microvm", atev1alpha1.SandboxClassMicroVM, true, false, false, false},
		{"wasm", atev1alpha1.SandboxClassWasm, false, true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := ateomSecurityContext(tt.class)
			if sc.Privileged == nil || *sc.Privileged != tt.wantPrivileged {
				t.Errorf("Privileged = %v, want %v", sc.Privileged, tt.wantPrivileged)
			}
			if sc.RunAsUser == nil || *sc.RunAsUser != 0 || sc.RunAsGroup == nil || *sc.RunAsGroup != 0 {
				t.Errorf("RunAsUser/Group = %v/%v, want 0/0", sc.RunAsUser, sc.RunAsGroup)
			}

			gotDropAll := sc.Capabilities != nil &&
				len(sc.Capabilities.Drop) == 1 &&
				sc.Capabilities.Drop[0] == "ALL"
			if gotDropAll != tt.wantDropAll {
				drop := []corev1.Capability(nil)
				if sc.Capabilities != nil {
					drop = sc.Capabilities.Drop
				}
				t.Errorf("capabilities drop = %v, want drop-ALL = %v", drop, tt.wantDropAll)
			}

			var gotAdd []corev1.Capability
			if sc.Capabilities != nil {
				gotAdd = sc.Capabilities.Add
			}
			if tt.wantGvisorCaps {
				if diff := cmp.Diff(ateomGvisorCapabilities, gotAdd); diff != "" {
					t.Errorf("capabilities add mismatch (-want +got):\n%s", diff)
				}
			} else if len(gotAdd) != 0 {
				// The wasm worker must not inherit the gVisor set: wasmtime runs
				// in-process and needs none of it.
				t.Errorf("capabilities add = %v, want none", gotAdd)
			}

			// Only the gVisor worker runs AppArmor-unconfined (runsc + cgroup
			// remount need mount). The privileged micro-VM worker and the
			// mount-free wasm worker both leave it unset.
			hasAppArmor := sc.AppArmorProfile != nil &&
				sc.AppArmorProfile.Type != nil &&
				*sc.AppArmorProfile.Type == corev1.AppArmorProfileTypeUnconfined
			if hasAppArmor != tt.wantUnconfined {
				t.Errorf("AppArmor Unconfined = %v, want %v", hasAppArmor, tt.wantUnconfined)
			}
		})
	}
}

// TestTerminationGracePeriodSeconds asserts the pod's grace period is hardcoded to 3600s.
func TestTerminationGracePeriodSeconds(t *testing.T) {
	wp := testWorkerPoolApplyConfig(nil)
	ps := buildDeploymentApplyConfig(wp, ateomOTelSettings{}, WorkerCertSourcePodCertificate).Spec.Template.Spec
	if ps.TerminationGracePeriodSeconds == nil {
		t.Fatalf("TerminationGracePeriodSeconds not set")
	}
	if *ps.TerminationGracePeriodSeconds != 3600 {
		t.Errorf("TerminationGracePeriodSeconds = %d, want 3600", *ps.TerminationGracePeriodSeconds)
	}
}

// TestWorkerCertSourceVolumes asserts each cert source projects the expected
// volume sources: pod-certificate keeps the podCertificate/clusterTrustBundle
// projections, cert-manager swaps in the issued Secret plus the trust-manager
// ConfigMaps at the same mount paths.
func TestWorkerCertSourceVolumes(t *testing.T) {
	findVolume := func(t *testing.T, ps *corev1ac.PodSpecApplyConfiguration, name string) *corev1ac.VolumeApplyConfiguration {
		t.Helper()
		for i := range ps.Volumes {
			if ps.Volumes[i].Name != nil && *ps.Volumes[i].Name == name {
				return &ps.Volumes[i]
			}
		}
		t.Fatalf("volume %q not found", name)
		return nil
	}

	t.Run("pod-certificate", func(t *testing.T) {
		ps := buildDeploymentApplyConfig(testWorkerPoolApplyConfig(nil), ateomOTelSettings{}, WorkerCertSourcePodCertificate).Spec.Template.Spec

		identity := findVolume(t, ps, atunnelIdentityVolume)
		if len(identity.Projected.Sources) != 2 ||
			identity.Projected.Sources[0].PodCertificate == nil ||
			identity.Projected.Sources[1].ClusterTrustBundle == nil {
			t.Errorf("identity sources = %+v, want podCertificate + clusterTrustBundle", identity.Projected.Sources)
		}
		egress := findVolume(t, ps, atunnelEgressTrustVolume)
		if len(egress.Projected.Sources) != 1 || egress.Projected.Sources[0].ClusterTrustBundle == nil {
			t.Errorf("egress trust sources = %+v, want clusterTrustBundle", egress.Projected.Sources)
		}
	})

	t.Run("cert-manager", func(t *testing.T) {
		ps := buildDeploymentApplyConfig(testWorkerPoolApplyConfig(nil), ateomOTelSettings{}, WorkerCertSourceCertManager).Spec.Template.Spec

		identity := findVolume(t, ps, atunnelIdentityVolume)
		if len(identity.Projected.Sources) != 2 {
			t.Fatalf("identity sources = %+v, want secret + configMap", identity.Projected.Sources)
		}
		secret := identity.Projected.Sources[0].Secret
		if secret == nil || secret.Name == nil || *secret.Name != workerCredentialSecretName ||
			len(secret.Items) != 1 || *secret.Items[0].Key != combinedPEMKey || *secret.Items[0].Path != "credential-bundle.pem" {
			t.Errorf("identity secret source = %+v, want %s %s->credential-bundle.pem", secret, workerCredentialSecretName, combinedPEMKey)
		}
		cm := identity.Projected.Sources[1].ConfigMap
		if cm == nil || cm.Name == nil || *cm.Name != podIdentityCABundleConfigMap ||
			len(cm.Items) != 1 || *cm.Items[0].Key != trustBundleKey || *cm.Items[0].Path != "trust-bundle.pem" {
			t.Errorf("identity configMap source = %+v, want %s %s->trust-bundle.pem", cm, podIdentityCABundleConfigMap, trustBundleKey)
		}

		egress := findVolume(t, ps, atunnelEgressTrustVolume)
		if len(egress.Projected.Sources) != 1 {
			t.Fatalf("egress trust sources = %+v, want one configMap", egress.Projected.Sources)
		}
		egressCM := egress.Projected.Sources[0].ConfigMap
		if egressCM == nil || egressCM.Name == nil || *egressCM.Name != servicednsCABundleConfigMap ||
			len(egressCM.Items) != 1 || *egressCM.Items[0].Key != trustBundleKey || *egressCM.Items[0].Path != "trust-bundle.pem" {
			t.Errorf("egress configMap source = %+v, want %s %s->trust-bundle.pem", egressCM, servicednsCABundleConfigMap, trustBundleKey)
		}
	})
}

// TestBuildDeploymentApplyConfigOTelEndpoint asserts the OTLP endpoint and the
// pod-scoped resource identity are set on the ateom container only when an endpoint
// is configured, and that the $(POD_*) refs precede OTEL_RESOURCE_ATTRIBUTES.
func TestBuildDeploymentApplyConfigOTelEndpoint(t *testing.T) {
	const endpoint = "http://collector.otel-system.svc:4317"
	tests := []struct {
		name          string
		endpoint      string
		wantTelemetry bool
	}{
		{"endpoint empty", "", false},
		{"endpoint set", endpoint, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := buildDeploymentApplyConfig(testWorkerPoolApplyConfig(nil), ateomOTelSettings{Endpoint: tt.endpoint}, WorkerCertSourcePodCertificate).
				Spec.Template.Spec.Containers[0]
			env := envByName(c.Env)

			if _, ok := env["POD_UID"]; !ok {
				t.Error("POD_UID must always be set")
			}

			if !tt.wantTelemetry {
				for _, k := range []string{"OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_RESOURCE_ATTRIBUTES", "OTEL_METRIC_EXPORT_INTERVAL", "OTEL_METRIC_EXPORT_TIMEOUT", "POD_NAME", "POD_NAMESPACE"} {
					if _, ok := env[k]; ok {
						t.Errorf("%s must be absent without an OTLP endpoint", k)
					}
				}
				return
			}

			if got := env["OTEL_EXPORTER_OTLP_ENDPOINT"].value; got != endpoint {
				t.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT = %q, want %q", got, endpoint)
			}
			if got := env["OTEL_RESOURCE_ATTRIBUTES"].value; got != ateomOTelResourceAttributes {
				t.Errorf("OTEL_RESOURCE_ATTRIBUTES = %q, want %q", got, ateomOTelResourceAttributes)
			}
			raIdx := env["OTEL_RESOURCE_ATTRIBUTES"].index
			for _, ref := range []string{"POD_UID", "POD_NAME", "POD_NAMESPACE"} {
				if _, ok := env[ref]; !ok {
					t.Errorf("%s must be set for OTEL_RESOURCE_ATTRIBUTES substitution", ref)
					continue
				}
				if env[ref].index > raIdx {
					t.Errorf("%s (index %d) must precede OTEL_RESOURCE_ATTRIBUTES (index %d)", ref, env[ref].index, raIdx)
				}
			}
		})
	}
}

// TestActorCapacityProjection covers F9 Stage B: a wasm-class pool with
// actorCapacity > 1 projects the capacity onto worker pods as BOTH the
// ate.dev/actor-capacity annotation (read by the ateapi syncer → Worker
// .actor_capacity → scheduler) and the WASM_MAX_ACTORS env (read by ateom-wasmd,
// sandbox S12), so the scheduler and the in-pod runtime agree on the same N.
// Non-wasm classes and capacity <= 1 project nothing (upstream single-actor
// default; other classes have no in-pod multi-actor support).
func TestActorCapacityProjection(t *testing.T) {
	tests := []struct {
		name          string
		sandboxClass  atev1alpha1.SandboxClass
		capacity      int32
		wantProjected bool
		wantValue     string
	}{
		{"wasm capacity 4 projects", atev1alpha1.SandboxClassWasm, 4, true, "4"},
		{"wasm capacity 1 is default", atev1alpha1.SandboxClassWasm, 1, false, ""},
		{"wasm capacity 0 is default", atev1alpha1.SandboxClassWasm, 0, false, ""},
		{"gvisor capacity 4 ignored", atev1alpha1.SandboxClassGvisor, 4, false, ""},
		{"microvm capacity 4 ignored", atev1alpha1.SandboxClassMicroVM, 4, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := testWorkerPoolApplyConfig(nil)
			wp.Spec.SandboxClass = tt.sandboxClass
			wp.Spec.ActorCapacity = tt.capacity

			dep := buildDeploymentApplyConfig(wp, ateomOTelSettings{}, WorkerCertSourcePodCertificate)
			ann := dep.Spec.Template.Annotations
			env := envByName(dep.Spec.Template.Spec.Containers[0].Env)

			gotAnn, hasAnn := ann[resources.WorkerActorCapacityAnnotation]
			gotEnv, hasEnv := env["WASM_MAX_ACTORS"]

			if !tt.wantProjected {
				if hasAnn {
					t.Errorf("annotation %q must be absent, got %q", resources.WorkerActorCapacityAnnotation, gotAnn)
				}
				if hasEnv {
					t.Errorf("WASM_MAX_ACTORS must be absent, got %q", gotEnv.value)
				}
				return
			}
			if !hasAnn || gotAnn != tt.wantValue {
				t.Errorf("annotation %q = %q, want %q", resources.WorkerActorCapacityAnnotation, gotAnn, tt.wantValue)
			}
			if !hasEnv || gotEnv.value != tt.wantValue {
				t.Errorf("WASM_MAX_ACTORS = %q, want %q", gotEnv.value, tt.wantValue)
			}
		})
	}
}

// TestBuildDeploymentApplyConfigMetricExportTuning asserts the export interval and
// per-export timeout reach the ateom container only when each is set alongside an
// endpoint. ateom is invisible to the collector until its first successful export
// tick, so the kind stack shortens the SDK's 60s interval to keep that gap inside
// the e2e budget, and its 30s timeout so a failing tick cannot swallow three
// shortened intervals.
func TestBuildDeploymentApplyConfigMetricExportTuning(t *testing.T) {
	const endpoint = "http://collector.otel-system.svc:4317"
	tests := []struct {
		name string
		otel ateomOTelSettings
		want map[string]string // env name -> value; absent key means must not be set
	}{
		{
			name: "unset keeps SDK defaults",
			otel: ateomOTelSettings{Endpoint: endpoint},
			want: nil,
		},
		{
			name: "both set with endpoint",
			otel: ateomOTelSettings{Endpoint: endpoint, MetricExportInterval: "10000", MetricExportTimeout: "10000"},
			want: map[string]string{"OTEL_METRIC_EXPORT_INTERVAL": "10000", "OTEL_METRIC_EXPORT_TIMEOUT": "10000"},
		},
		{
			name: "interval alone",
			otel: ateomOTelSettings{Endpoint: endpoint, MetricExportInterval: "10000"},
			want: map[string]string{"OTEL_METRIC_EXPORT_INTERVAL": "10000"},
		},
		{
			name: "timeout alone",
			otel: ateomOTelSettings{Endpoint: endpoint, MetricExportTimeout: "10000"},
			want: map[string]string{"OTEL_METRIC_EXPORT_TIMEOUT": "10000"},
		},
		{
			name: "ignored without endpoint",
			otel: ateomOTelSettings{MetricExportInterval: "10000", MetricExportTimeout: "10000"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := buildDeploymentApplyConfig(testWorkerPoolApplyConfig(nil), tt.otel, WorkerCertSourcePodCertificate).
				Spec.Template.Spec.Containers[0]
			env := envByName(c.Env)
			for _, k := range []string{"OTEL_METRIC_EXPORT_INTERVAL", "OTEL_METRIC_EXPORT_TIMEOUT"} {
				got, ok := env[k]
				want, wantSet := tt.want[k]
				if ok != wantSet {
					t.Errorf("%s present = %v, want %v", k, ok, wantSet)
					continue
				}
				if ok && got.value != want {
					t.Errorf("%s = %q, want %q", k, got.value, want)
				}
			}
		})
	}
}

// TestBuildDeploymentApplyConfigTracesSamplerPropagation asserts the sampler
// env pair reaches the ateom container only alongside an endpoint, and the arg
// only alongside a sampler: an arg without a sampler name is dead config the
// SDK ignores.
func TestBuildDeploymentApplyConfigTracesSamplerPropagation(t *testing.T) {
	const endpoint = "http://collector.otel-system.svc:4317"
	tests := []struct {
		name string
		otel ateomOTelSettings
		want map[string]string // value by env name; absent key means must not be set
	}{
		{
			name: "unset keeps binary default",
			otel: ateomOTelSettings{Endpoint: endpoint},
			want: nil,
		},
		{
			name: "sampler and arg with endpoint",
			otel: ateomOTelSettings{Endpoint: endpoint, TracesSampler: "parentbased_traceidratio", TracesSamplerArg: "0.25"},
			want: map[string]string{"OTEL_TRACES_SAMPLER": "parentbased_traceidratio", "OTEL_TRACES_SAMPLER_ARG": "0.25"},
		},
		{
			name: "sampler alone",
			otel: ateomOTelSettings{Endpoint: endpoint, TracesSampler: "parentbased_always_on"},
			want: map[string]string{"OTEL_TRACES_SAMPLER": "parentbased_always_on"},
		},
		{
			name: "arg alone stays unset",
			otel: ateomOTelSettings{Endpoint: endpoint, TracesSamplerArg: "0.25"},
			want: nil,
		},
		{
			name: "ignored without endpoint",
			otel: ateomOTelSettings{TracesSampler: "parentbased_traceidratio", TracesSamplerArg: "0.25"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := buildDeploymentApplyConfig(testWorkerPoolApplyConfig(nil), tt.otel, WorkerCertSourcePodCertificate).
				Spec.Template.Spec.Containers[0]
			env := envByName(c.Env)
			for _, k := range []string{"OTEL_TRACES_SAMPLER", "OTEL_TRACES_SAMPLER_ARG"} {
				got, ok := env[k]
				want, wantSet := tt.want[k]
				if ok != wantSet {
					t.Errorf("%s present = %v, want %v", k, ok, wantSet)
					continue
				}
				if ok && got.value != want {
					t.Errorf("%s = %q, want %q", k, got.value, want)
				}
			}
		})
	}
}

type envInfo struct {
	index int
	value string
}

func envByName(env []corev1ac.EnvVarApplyConfiguration) map[string]envInfo {
	m := make(map[string]envInfo, len(env))
	for i, e := range env {
		if e.Name == nil {
			continue
		}
		info := envInfo{index: i}
		if e.Value != nil {
			info.value = *e.Value
		}
		m[*e.Name] = info
	}
	return m
}

func testWorkerPoolApplyConfig(tmpl *atev1alpha1.WorkerPoolPodTemplate) *atev1alpha1.WorkerPool {
	return &atev1alpha1.WorkerPool{
		ObjectMeta: metav1.ObjectMeta{Name: "pool", Namespace: "default", UID: "uid"},
		Spec: atev1alpha1.WorkerPoolSpec{
			Replicas:   2,
			AteomImage: "ateom:v1",
			Template:   tmpl,
		},
	}
}

func expectedDeploymentApplyConfig(mutatePodSpec func(*corev1ac.PodSpecApplyConfiguration)) *appsv1ac.DeploymentApplyConfiguration {
	wp := testWorkerPoolApplyConfig(nil)

	podSpecAC := corev1ac.PodSpec().
		WithSecurityContext(corev1ac.PodSecurityContext().
			WithRunAsUser(0).
			WithRunAsGroup(0)).
		WithVolumes(
			corev1ac.Volume().
				WithName("run-ateom").
				WithHostPath(corev1ac.HostPathVolumeSource().
					WithPath(ateompath.BasePath).
					WithType(corev1.HostPathDirectoryOrCreate)),
			corev1ac.Volume().
				WithName(atunnelIdentityVolume).
				WithProjected(corev1ac.ProjectedVolumeSource().
					WithSources(
						corev1ac.VolumeProjection().
							WithPodCertificate(corev1ac.PodCertificateProjection().
								WithSignerName("podidentity.podcert.ate.dev/identity").
								WithKeyType("ECDSAP256").
								WithCredentialBundlePath("credential-bundle.pem")),
						corev1ac.VolumeProjection().
							WithClusterTrustBundle(corev1ac.ClusterTrustBundleProjection().
								WithSignerName("podidentity.podcert.ate.dev/identity").
								WithLabelSelector(metav1ac.LabelSelector().
									WithMatchLabels(map[string]string{"podcert.ate.dev/canarying": "live"})).
								WithPath("trust-bundle.pem")),
					),
				),
			corev1ac.Volume().
				WithName(atunnelEgressTrustVolume).
				WithProjected(corev1ac.ProjectedVolumeSource().
					WithSources(
						corev1ac.VolumeProjection().
							WithClusterTrustBundle(corev1ac.ClusterTrustBundleProjection().
								WithSignerName("servicedns.podcert.ate.dev/identity").
								WithLabelSelector(metav1ac.LabelSelector().
									WithMatchLabels(map[string]string{"podcert.ate.dev/canarying": "live"})).
								WithPath("trust-bundle.pem")),
					),
				),
		).
		WithContainers(corev1ac.Container().
			WithName("ateom").
			WithImage(wp.Spec.AteomImage).
			WithArgs(
				"--pod-uid=$(POD_UID)",
				"--atunnel-listen-address=0.0.0.0:443",
				"--atunnel-credential-bundle="+atunnelIdentityMountPath+"/credential-bundle.pem",
				"--atunnel-trust-bundle="+atunnelIdentityMountPath+"/trust-bundle.pem",
				"--atunnel-egress-listen-address=0.0.0.0:15001",
				"--atunnel-egress-trust-bundle="+atunnelEgressTrustMountPath+"/trust-bundle.pem",
			).
			WithPorts(corev1ac.ContainerPort().
				WithName("https").
				WithContainerPort(443).
				WithProtocol(corev1.ProtocolTCP)).
			WithSecurityContext(corev1ac.SecurityContext().
				WithRunAsUser(0).
				WithRunAsGroup(0).
				WithPrivileged(false).
				WithCapabilities(corev1ac.Capabilities().
					WithDrop("ALL").
					WithAdd(ateomGvisorCapabilities...)).
				WithAppArmorProfile(corev1ac.AppArmorProfile().
					WithType(corev1.AppArmorProfileTypeUnconfined))).
			WithEnv(corev1ac.EnvVar().
				WithName("POD_UID").
				WithValueFrom(corev1ac.EnvVarSource().
					WithFieldRef(corev1ac.ObjectFieldSelector().
						WithFieldPath("metadata.uid")))).
			WithVolumeMounts(
				corev1ac.VolumeMount().
					WithName("run-ateom").
					WithMountPath(ateompath.BasePath).
					WithMountPropagation(corev1.MountPropagationHostToContainer),
				corev1ac.VolumeMount().
					WithName(atunnelIdentityVolume).
					WithMountPath(atunnelIdentityMountPath).
					WithReadOnly(true),
				corev1ac.VolumeMount().
					WithName(atunnelEgressTrustVolume).
					WithMountPath(atunnelEgressTrustMountPath).
					WithReadOnly(true),
			).
			WithResources(corev1ac.ResourceRequirements()))

	podSpecAC.NodeSelector = map[string]string{}
	podSpecAC.Tolerations = []corev1ac.TolerationApplyConfiguration{}
	podSpecAC.WithPriorityClassName("")
	podSpecAC.WithAffinity(corev1ac.Affinity())
	podSpecAC.WithTerminationGracePeriodSeconds(workerTerminationGracePeriodSeconds)
	if mutatePodSpec != nil {
		mutatePodSpec(podSpecAC)
	}

	return appsv1ac.Deployment(wp.Name, wp.Namespace).
		WithOwnerReferences(metav1ac.OwnerReference().
			WithAPIVersion(atev1alpha1.GroupVersion.String()).
			WithKind("WorkerPool").
			WithName(wp.Name).
			WithUID(wp.UID).
			WithController(true).
			WithBlockOwnerDeletion(true)).
		WithSpec(appsv1ac.DeploymentSpec().
			WithReplicas(wp.Spec.Replicas).
			WithSelector(metav1ac.LabelSelector().
				WithMatchLabels(map[string]string{"ate.dev/worker-pool": wp.Name})).
			WithTemplate(corev1ac.PodTemplateSpec().
				WithLabels(map[string]string{"ate.dev/worker-pool": wp.Name}).
				WithSpec(podSpecAC)))
}
