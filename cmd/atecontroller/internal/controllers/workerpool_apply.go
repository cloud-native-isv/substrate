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
	"strconv"

	corev1 "k8s.io/api/core/v1"
	appsv1ac "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	metav1ac "k8s.io/client-go/applyconfigurations/meta/v1"

	"github.com/agent-substrate/substrate/internal/ateompath"
	"github.com/agent-substrate/substrate/internal/resources"
	atev1alpha1 "github.com/agent-substrate/substrate/pkg/api/v1alpha1"
)

// ateomOTelResourceAttributes mirrors atelet.yaml; service.instance.id is the pod
// uid so each worker pod is a distinct telemetry source.
const ateomOTelResourceAttributes = "k8s.namespace.name=$(POD_NAMESPACE),k8s.pod.name=$(POD_NAME),k8s.pod.uid=$(POD_UID),service.instance.id=$(POD_UID)"

// workerTerminationGracePeriodSeconds is the hardcoded pod termination grace
// period for worker pods (60 minutes).
const workerTerminationGracePeriodSeconds int64 = 3600

// ateomOTelSettings is the telemetry configuration propagated to ateom worker
// pods. A zero value leaves the pods without telemetry env.
type ateomOTelSettings struct {
	// Endpoint is the OTLP collector address. Empty disables ateom telemetry
	// entirely, so the other fields are ignored.
	Endpoint string
	// MetricExportInterval overrides the SDK's 60s PeriodicReader interval. It is
	// the raw OTEL_METRIC_EXPORT_INTERVAL value, i.e. whole milliseconds; the SDK
	// falls back to its default when it does not parse. Empty keeps the default.
	//
	// ateom registers no instruments of its own, so its only telemetry is
	// otelgrpc's rpc.server.* and it stays invisible to the collector until the
	// first export tick fires. Shortening the interval is what keeps that
	// startup gap inside an e2e budget; production leaves it unset.
	MetricExportInterval string
	// MetricExportTimeout overrides the SDK's  per-export timeout, in the same
	// whole-millisecond form as MetricExportInterval. Empty keeps the default.
	MetricExportTimeout string
	// TracesSampler and TracesSamplerArg are the raw OTEL_TRACES_SAMPLER and
	// OTEL_TRACES_SAMPLER_ARG values, passed through untouched: ateom's own
	// serverboot resolution validates them. Empty sampler keeps the worker's
	// default and drops the arg, which is dead config on its own.
	TracesSampler    string
	TracesSamplerArg string
}

const (
	atunnelIdentityVolume       = "atunnel-identity"
	atunnelIdentityMountPath    = "/run/podidentity.podcert.ate.dev"
	atunnelEgressTrustVolume    = "atunnel-egress-trust"
	atunnelEgressTrustMountPath = "/run/servicedns.podcert.ate.dev"
)

// WorkerCertSource selects where the TLS material mounted into worker pods
// comes from.
type WorkerCertSource string

const (
	// WorkerCertSourcePodCertificate projects podCertificate/clusterTrustBundle
	// volume sources, served by the in-cluster podcertcontroller signer.
	// Requires the PodCertificateRequest and ClusterTrustBundleProjection
	// feature gates.
	WorkerCertSourcePodCertificate WorkerCertSource = "pod-certificate"
	// WorkerCertSourceCertManager mounts a cert-manager issued credential
	// Secret and trust-manager distributed CA ConfigMaps instead, for clusters
	// without PodCertificateRequest support. The mount paths and file names
	// are identical, so the ateom container args do not change.
	WorkerCertSourceCertManager WorkerCertSource = "cert-manager"
)

// Object names the cert-manager worker cert source mounts from the
// WorkerPool's namespace. The Secret is produced by a cert-manager
// Certificate (see manifests/ate-install/cert-manager-pki), the ConfigMaps by
// trust-manager Bundles synced to every namespace.
const (
	workerCredentialSecretName   = "ate-worker-podidentity-cert"
	podIdentityCABundleConfigMap = "podidentity-ca-bundle"
	servicednsCABundleConfigMap  = "servicedns-ca-bundle"
	combinedPEMKey               = "tls-combined.pem"
	trustBundleKey               = "trust-bundle.pem"
)

// mcpProjection carries the cross-S shared MCP configuration the controller
// resolves from a WorkerPool's referenced McpPool and projects onto the worker
// pod's ateom container (F10 Stage B). The zero value projects nothing — the
// fail-closed outcome when the pool is absent, the tenant is not admitted (R3),
// or the tenant has no execution unit. A worker pod is single-tenant (one S
// domain = one digital employee = one atespace), so one projection covers every
// actor the worker hosts (F9/S12).
type mcpProjection struct {
	// Backend is the WASM_MCP_BACKEND value: the resolved per-tenant execution-unit
	// endpoint as `tls://<endpoint>` (cross-S mTLS, I-3). Empty projects no backend.
	Backend string
	// CapabilityScope is the WASM_MCP_CAPABILITY_SCOPE value: the unit's Tools
	// allowlist (D7) as `<atespace>=<tool,tool>`, feeding the S-layer mediator's R3
	// per-tenant CapabilityScope. Empty projects no scope (mediator unscoped, bounded
	// only by its tool contract D1).
	CapabilityScope string
}

// buildDeploymentApplyConfig constructs the SSA apply configuration for the
// Deployment managed by a WorkerPool. Only fields owned by this controller
// are declared here. otel, when it carries an endpoint, is propagated to the
// ateom container so it pushes telemetry to that collector. certSource
// selects the volume sources for the worker's TLS material. mcp carries the
// resolved cross-S MCP projection (backend endpoint + per-tenant capability
// scope) for this pool's tenant; its zero value projects nothing (F10 Stage B).
func buildDeploymentApplyConfig(wp *atev1alpha1.WorkerPool, otel ateomOTelSettings, certSource WorkerCertSource, mcp mcpProjection) *appsv1ac.DeploymentApplyConfiguration {
	containerAC := corev1ac.Container().
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
		WithSecurityContext(ateomSecurityContext(wp.Spec.SandboxClass)).
		WithEnv(ateomContainerEnv(otel, wp, mcp)...).
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
		)

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
					WithSources(atunnelIdentitySources(certSource)...),
				),
			corev1ac.Volume().
				WithName(atunnelEgressTrustVolume).
				WithProjected(corev1ac.ProjectedVolumeSource().
					WithSources(atunnelEgressTrustSources(certSource)...),
				),
		)

	applyWorkerPoolPodTemplate(podSpecAC, containerAC, wp.Spec.Template)
	maybeApplyMicroVMPodShape(podSpecAC, containerAC, wp.Spec.SandboxClass)
	podSpecAC.WithContainers(containerAC)
	podSpecAC.WithTerminationGracePeriodSeconds(workerTerminationGracePeriodSeconds)

	podTemplateAC := corev1ac.PodTemplateSpec().
		WithLabels(map[string]string{
			"ate.dev/worker-pool": wp.Name,
		}).
		WithSpec(podSpecAC)
	// F9 Stage B: project the actor capacity onto the pod as an annotation so the
	// ateapi syncer reads it into Worker.actor_capacity (the scheduler side of
	// N:1), mirroring the WASM_MAX_ACTORS env (the in-pod runtime side).
	if ann := actorCapacityAnnotations(wp); ann != nil {
		podTemplateAC.WithAnnotations(ann)
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
			WithTemplate(podTemplateAC))
}

// actorCapacityProjection returns the pool's actor capacity as a string when it
// should be projected onto worker pods — wasm class with capacity > 1 — or ""
// when the upstream single-actor default applies (no projection). Only the wasm
// class hosts multiple actors in-pod today (sandbox S12); a positive capacity on
// another class is ignored so the scheduler never places actors a worker cannot
// host. F9 Stage B.
func actorCapacityProjection(wp *atev1alpha1.WorkerPool) string {
	if wp.Spec.SandboxClass != atev1alpha1.SandboxClassWasm || wp.Spec.ActorCapacity <= 1 {
		return ""
	}
	return strconv.Itoa(int(wp.Spec.ActorCapacity))
}

// actorCapacityAnnotations returns the worker-pod annotation carrying the actor
// capacity to the ateapi syncer (→ Worker.actor_capacity → scheduler), or nil
// when not applicable. Shares the gate with the WASM_MAX_ACTORS env so both sides
// agree on N.
func actorCapacityAnnotations(wp *atev1alpha1.WorkerPool) map[string]string {
	if c := actorCapacityProjection(wp); c != "" {
		return map[string]string{resources.WorkerActorCapacityAnnotation: c}
	}
	return nil
}

// atunnelIdentitySources returns the projected volume sources for the
// worker's atunnel serving credential and the podidentity trust bundle,
// according to certSource.
func atunnelIdentitySources(certSource WorkerCertSource) []*corev1ac.VolumeProjectionApplyConfiguration {
	if certSource == WorkerCertSourceCertManager {
		return []*corev1ac.VolumeProjectionApplyConfiguration{
			corev1ac.VolumeProjection().
				WithSecret(corev1ac.SecretProjection().
					WithName(workerCredentialSecretName).
					WithItems(corev1ac.KeyToPath().
						WithKey(combinedPEMKey).
						WithPath("credential-bundle.pem"))),
			corev1ac.VolumeProjection().
				WithConfigMap(corev1ac.ConfigMapProjection().
					WithName(podIdentityCABundleConfigMap).
					WithItems(corev1ac.KeyToPath().
						WithKey(trustBundleKey).
						WithPath("trust-bundle.pem"))),
		}
	}
	return []*corev1ac.VolumeProjectionApplyConfiguration{
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
	}
}

// atunnelEgressTrustSources returns the projected volume sources for the
// servicedns trust bundle the worker's egress client validates the gateway
// against, according to certSource.
func atunnelEgressTrustSources(certSource WorkerCertSource) []*corev1ac.VolumeProjectionApplyConfiguration {
	if certSource == WorkerCertSourceCertManager {
		return []*corev1ac.VolumeProjectionApplyConfiguration{
			corev1ac.VolumeProjection().
				WithConfigMap(corev1ac.ConfigMapProjection().
					WithName(servicednsCABundleConfigMap).
					WithItems(corev1ac.KeyToPath().
						WithKey(trustBundleKey).
						WithPath("trust-bundle.pem"))),
		}
	}
	return []*corev1ac.VolumeProjectionApplyConfiguration{
		corev1ac.VolumeProjection().
			WithClusterTrustBundle(corev1ac.ClusterTrustBundleProjection().
				WithSignerName("servicedns.podcert.ate.dev/identity").
				WithLabelSelector(metav1ac.LabelSelector().
					WithMatchLabels(map[string]string{"podcert.ate.dev/canarying": "live"})).
				WithPath("trust-bundle.pem")),
	}
}

// ateomContainerEnv adds the OTLP endpoint and resource identity only when
// telemetry is configured. POD_* refs precede OTEL_RESOURCE_ATTRIBUTES so its
// $(POD_*) substitutions resolve. It also projects the pool's actor capacity as
// WASM_MAX_ACTORS for the wasm class (F9 Stage B) and, when mcp carries a
// resolved cross-S MCP projection, the execution-unit endpoint as
// WASM_MCP_BACKEND and the per-tenant tool allowlist as
// WASM_MCP_CAPABILITY_SCOPE (F10 Stage B).
func ateomContainerEnv(otel ateomOTelSettings, wp *atev1alpha1.WorkerPool, mcp mcpProjection) []*corev1ac.EnvVarApplyConfiguration {
	envs := []*corev1ac.EnvVarApplyConfiguration{
		fieldRefEnv("POD_UID", "metadata.uid"),
	}
	// F9 Stage B: tell the in-pod runtime (ateom-wasmd, sandbox S12) how many
	// actors it may host, so it agrees with the N the scheduler places via the
	// matching pod annotation. Only the wasm class supports multi-actor in-pod;
	// capacity <= 1 is the upstream default and needs no env.
	if multiActorCap := actorCapacityProjection(wp); multiActorCap != "" {
		envs = append(envs, corev1ac.EnvVar().WithName("WASM_MAX_ACTORS").WithValue(multiActorCap))
	}
	// F10 Stage B: the cross-S shared MCP capability backend endpoint, resolved by
	// the controller from the referenced McpPool for this pool's tenant (single
	// tenant per worker = one S domain = one digital employee). Consumed by
	// ateom-wasmd (sandbox S10) as the mediator's remote execution-unit backend
	// (`tls://<endpoint>` = cross-S mTLS, I-3). Empty = no controller-projected MCP
	// backend (workers keep any statically-set WASM_MCP_BACKEND, or none). The mTLS
	// client material (WASM_MCP_TLS_*_FILE) comes from the worker's SPIFFE/podcert
	// identity mount, not projected here.
	if mcp.Backend != "" {
		envs = append(envs, corev1ac.EnvVar().WithName("WASM_MCP_BACKEND").WithValue(mcp.Backend))
	}
	// F10 Stage B: the per-tenant MCP tool allowlist (the resolved McpPool unit's
	// Tools, D7) fed to the S-layer mediator's R3 CapabilityScope, so the P-declared
	// tools are the ones the agent may actually call (deny-by-default at S). Format
	// `<atespace>=<tool,tool>` matches ateom-wasmd's parse_capability_scope (sandbox
	// S10). Empty = project none: the mediator stays unscoped, bounded only by its
	// tool contract (D1) — used when the unit declares no tools.
	if mcp.CapabilityScope != "" {
		envs = append(envs, corev1ac.EnvVar().WithName("WASM_MCP_CAPABILITY_SCOPE").WithValue(mcp.CapabilityScope))
	}
	if otel.Endpoint == "" {
		return envs
	}
	envs = append(envs,
		fieldRefEnv("POD_NAME", "metadata.name"),
		fieldRefEnv("POD_NAMESPACE", "metadata.namespace"),
		corev1ac.EnvVar().WithName("OTEL_EXPORTER_OTLP_ENDPOINT").WithValue(otel.Endpoint),
		corev1ac.EnvVar().WithName("OTEL_RESOURCE_ATTRIBUTES").WithValue(ateomOTelResourceAttributes),
	)
	if otel.MetricExportInterval != "" {
		envs = append(envs, corev1ac.EnvVar().
			WithName("OTEL_METRIC_EXPORT_INTERVAL").
			WithValue(otel.MetricExportInterval))
	}
	if otel.MetricExportTimeout != "" {
		envs = append(envs, corev1ac.EnvVar().
			WithName("OTEL_METRIC_EXPORT_TIMEOUT").
			WithValue(otel.MetricExportTimeout))
	}
	if otel.TracesSampler != "" {
		envs = append(envs, corev1ac.EnvVar().
			WithName("OTEL_TRACES_SAMPLER").
			WithValue(otel.TracesSampler))
		if otel.TracesSamplerArg != "" {
			envs = append(envs, corev1ac.EnvVar().
				WithName("OTEL_TRACES_SAMPLER_ARG").
				WithValue(otel.TracesSamplerArg))
		}
	}
	return envs
}

func fieldRefEnv(name, fieldPath string) *corev1ac.EnvVarApplyConfiguration {
	return corev1ac.EnvVar().
		WithName(name).
		WithValueFrom(corev1ac.EnvVarSource().
			WithFieldRef(corev1ac.ObjectFieldSelector().
				WithFieldPath(fieldPath)))
}

// ateomGvisorCapabilities is the capability set an unprivileged gVisor worker
// needs. runsc's gofer maps a full-range identity in a user namespace
// (SETUID/SETGID/SETPCAP/SETFCAP), the sandbox pivots root and traces the
// application (SYS_ADMIN/SYS_CHROOT/SYS_PTRACE), actor networking programs the
// veth and nftables rules (NET_ADMIN/NET_RAW), and the OCI rootfs is unpacked
// and device nodes created as root over image-owned trees
// (DAC_OVERRIDE/FOWNER/CHOWN/MKNOD). This replaces the former privileged worker;
// the default seccomp and AppArmor profiles are sufficient (no Unconfined).
var ateomGvisorCapabilities = []corev1.Capability{
	"NET_ADMIN", "SYS_ADMIN", "SYS_CHROOT", "SYS_PTRACE",
	"SETUID", "SETGID", "SETPCAP", "DAC_OVERRIDE",
	"FOWNER", "CHOWN", "MKNOD", "NET_RAW", "SETFCAP",
}

// ateomSecurityContext returns the ateom container security context for a sandbox
// class. The gVisor worker runs unprivileged with an explicit capability set; the
// micro-VM worker stays privileged because kata + cloud-hypervisor needs broad
// host access (vhost devices, mounts); the wasm worker drops every capability.
// An empty class defaults to gVisor.
func ateomSecurityContext(class atev1alpha1.SandboxClass) *corev1ac.SecurityContextApplyConfiguration {
	sc := corev1ac.SecurityContext().
		WithRunAsUser(0).
		WithRunAsGroup(0)
	if class == atev1alpha1.SandboxClassMicroVM {
		return sc.WithPrivileged(true)
	}
	if class == atev1alpha1.SandboxClassWasm {
		// ateom-wasmd embeds a wasmtime interpreter in its own process: it never
		// execs runsc, pivots root, traces the workload, programs veth/nftables
		// or unpacks an OCI rootfs, so none of ateomGvisorCapabilities applies.
		// It only creates directories under the shared hostPath and binds a unix
		// socket, both as uid 0 over root-owned trees, so it needs no
		// DAC_OVERRIDE either. It performs no mounts, so the default AppArmor
		// profile is sufficient (no Unconfined).
		return sc.
			WithPrivileged(false).
			WithCapabilities(corev1ac.Capabilities().
				WithDrop("ALL"))
	}
	// runsc mounts and pivots root inside the sandbox, and the worker remounts
	// /sys/fs/cgroup read-write to nest per-actor cgroups; the default AppArmor
	// profile denies those mounts (enforced on GKE COS, a no-op on nodes that do
	// not load AppArmor). A privileged worker got this implicitly; unprivileged
	// must request it. seccomp stays at the runtime default. Confining runsc with
	// a tailored AppArmor profile instead of Unconfined is a follow-up.
	return sc.
		WithPrivileged(false).
		WithCapabilities(corev1ac.Capabilities().
			WithDrop("ALL").
			WithAdd(ateomGvisorCapabilities...)).
		WithAppArmorProfile(corev1ac.AppArmorProfile().
			WithType(corev1.AppArmorProfileTypeUnconfined))
}

// maybeApplyMicroVMPodShape adds the /dev/kvm device and node placement a
// micro-VM (kata + cloud-hypervisor) worker pool needs, on top of any
// pod-template settings. No-op unless sandboxClass is the micro-VM class.
//
// TODO: this hardcodes one sandbox class's pod requirements in the controller.
// Consider making it generic so a sandbox class can declare its own pod shape
// (e.g. required devices/mounts + node placement on the SandboxConfig spec)
// instead of branching on SandboxClass here, so new classes don't need a
// controller change.
func maybeApplyMicroVMPodShape(
	podSpecAC *corev1ac.PodSpecApplyConfiguration,
	containerAC *corev1ac.ContainerApplyConfiguration,
	sandboxClass atev1alpha1.SandboxClass,
) {
	if sandboxClass != atev1alpha1.SandboxClassMicroVM {
		return
	}

	// The micro-VM runtime needs /dev/kvm. The container is already privileged
	// (so it can also reach vhost devices), but we mount /dev/kvm explicitly.
	containerAC.WithVolumeMounts(corev1ac.VolumeMount().
		WithName("dev-kvm").
		WithMountPath("/dev/kvm"))
	podSpecAC.WithVolumes(corev1ac.Volume().
		WithName("dev-kvm").
		WithHostPath(corev1ac.HostPathVolumeSource().
			WithPath("/dev/kvm").
			WithType(corev1.HostPathCharDev)))

	// Pin placement to KVM-capable, nested-virt nodes via nodeSelector +
	// toleration on ate.dev/sandboxClass=microvm. This is our own convention
	// (GKE attaches no label/taint to nested-virt pools): applied to kind nodes
	// by hack/create-kind-cluster.sh and via --node-labels at GKE pool creation.
	// Additive on top of the WorkerPool's configurable scheduling fields
	// (spec.template nodeSelector/tolerations/affinity, added in #247) — merge,
	// don't overwrite.
	if podSpecAC.NodeSelector == nil {
		podSpecAC.NodeSelector = map[string]string{}
	}
	podSpecAC.NodeSelector["ate.dev/sandboxClass"] = string(atev1alpha1.SandboxClassMicroVM)
	podSpecAC.WithTolerations(corev1ac.Toleration().
		WithKey("ate.dev/sandboxClass").
		WithOperator(corev1.TolerationOpEqual).
		WithValue(string(atev1alpha1.SandboxClassMicroVM)).
		WithEffect(corev1.TaintEffectNoSchedule))
}

func applyWorkerPoolPodTemplate(
	podSpecAC *corev1ac.PodSpecApplyConfiguration,
	containerAC *corev1ac.ContainerApplyConfiguration,
	tmpl *atev1alpha1.WorkerPoolPodTemplate,
) {
	podSpecAC.NodeSelector = map[string]string{}
	podSpecAC.Tolerations = []corev1ac.TolerationApplyConfiguration{}
	podSpecAC.WithPriorityClassName("")
	podSpecAC.WithAffinity(corev1ac.Affinity())
	resourcesAC := corev1ac.ResourceRequirements()
	containerAC.WithResources(resourcesAC)

	if tmpl == nil {
		return
	}

	if tmpl.NodeSelector != nil {
		podSpecAC.WithNodeSelector(tmpl.NodeSelector)
	}
	podSpecAC.Tolerations = tolerationApplyValues(tolerationsToApply(tmpl.Tolerations))
	podSpecAC.WithPriorityClassName(tmpl.PriorityClassName)

	if tmpl.NodeAffinity != nil {
		podSpecAC.WithAffinity(corev1ac.Affinity().WithNodeAffinity(nodeAffinityToApply(tmpl.NodeAffinity)))
	}

	if tmpl.Resources != nil {
		if tmpl.Resources.Requests != nil {
			resourcesAC.WithRequests(tmpl.Resources.Requests)
		}
		if tmpl.Resources.Limits != nil {
			resourcesAC.WithLimits(tmpl.Resources.Limits)
		}
	}
}

func tolerationApplyValues(tolerations []*corev1ac.TolerationApplyConfiguration) []corev1ac.TolerationApplyConfiguration {
	out := make([]corev1ac.TolerationApplyConfiguration, 0, len(tolerations))
	for _, toleration := range tolerations {
		out = append(out, *toleration)
	}
	return out
}

func tolerationsToApply(tolerations []corev1.Toleration) []*corev1ac.TolerationApplyConfiguration {
	out := make([]*corev1ac.TolerationApplyConfiguration, 0, len(tolerations))
	for i := range tolerations {
		t := &tolerations[i]
		ac := corev1ac.Toleration()
		if t.Key != "" {
			ac.WithKey(t.Key)
		}
		if t.Operator != "" {
			ac.WithOperator(t.Operator)
		}
		if t.Value != "" {
			ac.WithValue(t.Value)
		}
		if t.Effect != "" {
			ac.WithEffect(t.Effect)
		}
		if t.TolerationSeconds != nil {
			ac.WithTolerationSeconds(*t.TolerationSeconds)
		}
		out = append(out, ac)
	}
	return out
}

func nodeAffinityToApply(na *corev1.NodeAffinity) *corev1ac.NodeAffinityApplyConfiguration {
	ac := corev1ac.NodeAffinity()
	if na.RequiredDuringSchedulingIgnoredDuringExecution != nil {
		ac.WithRequiredDuringSchedulingIgnoredDuringExecution(nodeSelectorToApply(na.RequiredDuringSchedulingIgnoredDuringExecution))
	}
	for i := range na.PreferredDuringSchedulingIgnoredDuringExecution {
		term := &na.PreferredDuringSchedulingIgnoredDuringExecution[i]
		ac.WithPreferredDuringSchedulingIgnoredDuringExecution(preferredSchedulingTermToApply(term))
	}
	return ac
}

func nodeSelectorToApply(ns *corev1.NodeSelector) *corev1ac.NodeSelectorApplyConfiguration {
	ac := corev1ac.NodeSelector()
	for i := range ns.NodeSelectorTerms {
		ac.WithNodeSelectorTerms(nodeSelectorTermToApply(&ns.NodeSelectorTerms[i]))
	}
	return ac
}

func preferredSchedulingTermToApply(term *corev1.PreferredSchedulingTerm) *corev1ac.PreferredSchedulingTermApplyConfiguration {
	return corev1ac.PreferredSchedulingTerm().
		WithWeight(term.Weight).
		WithPreference(nodeSelectorTermToApply(&term.Preference))
}

func nodeSelectorTermToApply(term *corev1.NodeSelectorTerm) *corev1ac.NodeSelectorTermApplyConfiguration {
	ac := corev1ac.NodeSelectorTerm()
	for i := range term.MatchExpressions {
		ac.WithMatchExpressions(nodeSelectorRequirementToApply(&term.MatchExpressions[i]))
	}
	for i := range term.MatchFields {
		ac.WithMatchFields(nodeSelectorRequirementToApply(&term.MatchFields[i]))
	}
	return ac
}

func nodeSelectorRequirementToApply(req *corev1.NodeSelectorRequirement) *corev1ac.NodeSelectorRequirementApplyConfiguration {
	ac := corev1ac.NodeSelectorRequirement().WithKey(req.Key).WithOperator(req.Operator)
	if len(req.Values) > 0 {
		ac.WithValues(req.Values...)
	}
	return ac
}
