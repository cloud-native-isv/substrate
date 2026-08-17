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
package main

import (
	"os"

	"github.com/agent-substrate/substrate/cmd/atecontroller/internal/controllers"
	"github.com/agent-substrate/substrate/internal/ateapiauth"
	clientv1alpha1 "github.com/agent-substrate/substrate/pkg/api/v1alpha1"
	"github.com/agent-substrate/substrate/pkg/proto/ateapipb"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")

	ateAPIConnSpec = pflag.String("ateapi-conn-spec", "dns:///api.ate-system.svc:443", "")

	otelEndpoint = pflag.String("otel-exporter-otlp-endpoint", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		"OTLP endpoint set on ateom worker pods so they push telemetry. Defaults to the controller's own OTEL_EXPORTER_OTLP_ENDPOINT.")

	otelMetricExportInterval = pflag.String("otel-metric-export-interval", os.Getenv("OTEL_METRIC_EXPORT_INTERVAL"),
		"Metric export interval in milliseconds set on ateom worker pods. Empty keeps the OTel SDK's 60s default. Defaults to the controller's own OTEL_METRIC_EXPORT_INTERVAL.")

	otelMetricExportTimeout = pflag.String("otel-metric-export-timeout", os.Getenv("OTEL_METRIC_EXPORT_TIMEOUT"),
		"Per-export timeout in milliseconds set on ateom worker pods. Empty keeps the OTel SDK's 30s default. Defaults to the controller's own OTEL_METRIC_EXPORT_TIMEOUT.")

	otelTracesSampler = pflag.String("otel-traces-sampler", os.Getenv("OTEL_TRACES_SAMPLER"),
		"Trace sampler set on ateom worker pods. Empty keeps the ateom binary's default. Defaults to the controller's own OTEL_TRACES_SAMPLER.")

	otelTracesSamplerArg = pflag.String("otel-traces-sampler-arg", os.Getenv("OTEL_TRACES_SAMPLER_ARG"),
		"Trace sampler argument set on ateom worker pods, ignored unless --otel-traces-sampler is set. Defaults to the controller's own OTEL_TRACES_SAMPLER_ARG.")

	ateapiCAFile     = pflag.String("ateapi-ca-file", ateapiauth.DefaultServiceAccountCAFile, "PEM file with CAs trusted to verify the ateapi server cert.")
	ateapiServerName = pflag.String("ateapi-server-name", "", "SNI / hostname expected on the ateapi server cert. Optional.")
	ateapiTokenAuth  = pflag.Bool("ateapi-use-token-auth", false, "Authenticate to ateapi with the Bearer token from --ateapi-token-file instead of the client certificate from --ateapi-client-cert.")
	ateapiTokenFile  = pflag.String("ateapi-token-file", "", "Projected SA token file used as Bearer credential. Required with --ateapi-use-token-auth, ignored otherwise.")
	ateapiClientCert = pflag.String("ateapi-client-cert", "", "Credential bundle presented as the client certificate when dialing ateapi. Required unless --ateapi-use-token-auth is set, ignored otherwise.")

	workerCertSource = pflag.String("worker-cert-source", string(controllers.WorkerCertSourcePodCertificate),
		"Where worker pod TLS material comes from: 'pod-certificate' projects podCertificate/clusterTrustBundle volume sources (requires the PodCertificateRequest feature gate), 'cert-manager' mounts the cert-manager issued Secret and trust-manager CA ConfigMaps expected by manifests/ate-install/cert-manager-pki.")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(clientv1alpha1.AddToScheme(scheme)) // Register our CRD
}

func main() {
	pflag.Parse()
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	certSource := controllers.WorkerCertSource(*workerCertSource)
	if certSource != controllers.WorkerCertSourcePodCertificate && certSource != controllers.WorkerCertSourceCertManager {
		setupLog.Error(nil, "invalid --worker-cert-source", "value", *workerCertSource)
		os.Exit(1)
	}
	if certSource == controllers.WorkerCertSourceCertManager {
		setupLog.Info("WARNING: worker TLS material comes from cert-manager " +
			"(--worker-cert-source=cert-manager): worker credentials degrade from one " +
			"podCertificate projection per pod to a single Secret shared by every worker pod in the " +
			"namespace, so a leaked worker credential is no longer attributable to one pod. Actor " +
			"routing is unaffected (atunnel authorizes on the activated actor, not the certificate). " +
			"Every WorkerPool namespace needs its own Certificate — see " +
			"hack/install-ate-certmanager.sh --create-worker-cert. Intended only for clusters " +
			"without PodCertificateRequest.")
	}

	dialOpts, err := ateapiauth.DialOptions(ateapiauth.ClientConfig{
		UseTokenAuth:     *ateapiTokenAuth,
		CAFile:           *ateapiCAFile,
		ServerName:       *ateapiServerName,
		TokenFile:        *ateapiTokenFile,
		ClientCredBundle: *ateapiClientCert,
	})
	if err != nil {
		setupLog.Error(err, "building ateapi dial options")
		os.Exit(1)
	}

	ateapiConn, err := grpc.NewClient(*ateAPIConnSpec, dialOpts...)
	if err != nil {
		setupLog.Error(err, "Error creating grpc connection to ate api")
		os.Exit(1)
	}

	ateapiClient := ateapipb.NewControlClient(ateapiConn)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.WorkerPoolReconciler{
		Client:                   mgr.GetClient(),
		Scheme:                   mgr.GetScheme(),
		OTelEndpoint:             *otelEndpoint,
		OTelMetricExportInterval: *otelMetricExportInterval,
		OTelMetricExportTimeout:  *otelMetricExportTimeout,
		OTelTracesSampler:        *otelTracesSampler,
		OTelTracesSamplerArg:     *otelTracesSamplerArg,
		WorkerCertSource:         certSource,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "WorkerPool")
		os.Exit(1)
	}

	if err = (&controllers.NetworkPolicyReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NetPolicy")
		os.Exit(1)
	}

	if err = (&controllers.ActorTemplateReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		AteClient: ateapiClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ActorTemplate")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
