#!/usr/bin/env bash

# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# install-ate-certmanager.sh deploys the ate system with TLS material issued
# by cert-manager/trust-manager (manifests/ate-install/cert-manager-pki)
# instead of the PodCertificateRequest machinery, for clusters without the
# PodCertificateRequest / ClusterTrustBundle feature gates (e.g. managed
# clusters such as ACK).
#
# It intentionally does not touch hack/install-ate.sh: that script deploys the
# podcertificate-controller and waits on ClusterTrustBundles, both of which
# do not exist on such clusters. Compared to it, this script:
#   - checks cert-manager is installed and installs trust-manager if missing;
#   - skips the podcertificate-controller CAs and the valkey-ca-certs secret
#     (both replaced by the cert-manager PKI and trust-manager Bundles);
#   - deploys the cert-manager-pki overlay (or its agentgateway variant) and
#     waits on Certificates/Bundles instead of ClusterTrustBundles.
#
# Switching back once the cluster supports PodCertificateRequest: deploy with
# hack/install-ate.sh again; the two flags the overlay sets
# (--atelet-require-pod-identity-ext, --worker-cert-source) default to the
# original behavior.

set -o errexit -o nounset -o pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "${ROOT}"

# Source the environment variables if configured (same convention as
# hack/install-ate.sh).
if [[ -f .ate-dev-env.sh ]] && [[ -z "${NO_DEV_ENV:-}" ]]; then
  source .ate-dev-env.sh
fi

COLOR_CYAN='\033[1;36m'
COLOR_RESET='\033[0m'

log_step() {
  echo -e "${COLOR_CYAN}[step]: $1${COLOR_RESET}"
}

usage() {
  echo "Usage: $0 [options]"
  echo ""
  echo "  --deploy-ate-system                  Deploy the ate system with cert-manager issued TLS"
  echo "  --delete-ate-system                  Delete the ate system (keeps cert-manager/trust-manager)"
  echo "  --atenet-router=envoy|agentgateway   Select the atenet router dataplane (default: envoy)"
  echo "  --create-worker-cert <namespace>     Issue the worker identity Certificate in a WorkerPool namespace"
  echo ""
  echo "Environment: KUBECTL_CONTEXT selects the kubeconfig context (optional)."
}

run_kubectl() {
  kubectl \
    ${KUBECTL_CONTEXT:+--context=${KUBECTL_CONTEXT}} \
    "$@"
}

run_kubectl_ate() {
  go run ./cmd/kubectl-ate \
    ${KUBECTL_CONTEXT:+--context=${KUBECTL_CONTEXT}} \
    "$@"
}

run_ko() {
  local ldflags=()
  while IFS= read -r line || [[ -n "${line}" ]]; do
    [[ -n "${line}" ]] && ldflags+=("--ldflags=${line}")
  done < <(make ldflags)

  case "${1:-}" in
    apply|create|delete|run)
      ./hack/run-tool.sh ko "$@" \
          "${ldflags[@]}" \
          ${KUBECTL_CONTEXT:+-- --context="${KUBECTL_CONTEXT}"}
      ;;
    *)
      ./hack/run-tool.sh ko "$@" \
          "${ldflags[@]}"
      ;;
  esac
}

atenet_router() {
  case "${ATE_ATENET_ROUTER:-envoy}" in
    envoy|agentgateway)
      echo "${ATE_ATENET_ROUTER:-envoy}"
      ;;
    *)
      echo "Error: --atenet-router must be envoy or agentgateway, got '${ATE_ATENET_ROUTER}'" >&2
      exit 1
      ;;
  esac
}

overlay_dir() {
  if [[ "$(atenet_router)" == "agentgateway" ]]; then
    echo "manifests/ate-install/cert-manager-pki-agentgateway"
  else
    echo "manifests/ate-install/cert-manager-pki"
  fi
}

# cert-manager must already be running; trust-manager is installed on demand
# into the cert-manager namespace (its default trust namespace, where the CA
# secrets from pki.yaml live).
ensure_cert_manager_stack() {
  log_step "ensure_cert_manager_stack"
  if ! run_kubectl get crd certificates.cert-manager.io clusterissuers.cert-manager.io >/dev/null 2>&1; then
    echo "error: cert-manager CRDs not found. Install cert-manager first, e.g.:" >&2
    echo "  helm upgrade -i cert-manager oci://quay.io/jetstack/charts/cert-manager -n cert-manager --create-namespace --set crds.enabled=true --wait" >&2
    exit 1
  fi

  if run_kubectl get crd bundles.trust.cert-manager.io >/dev/null 2>&1; then
    return
  fi
  if ! command -v helm &>/dev/null; then
    echo "error: trust-manager is not installed and helm is unavailable to install it. Install it manually:" >&2
    echo "  helm upgrade -i trust-manager oci://quay.io/jetstack/charts/trust-manager -n cert-manager --wait" >&2
    exit 1
  fi
  log_step "installing trust-manager"
  helm ${KUBECTL_CONTEXT:+--kube-context=${KUBECTL_CONTEXT}} \
    upgrade -i trust-manager oci://quay.io/jetstack/charts/trust-manager \
    -n cert-manager --wait
}

create_jwt_authority_pool_secret() {
  log_step "create_jwt_authority_pool_secret"
  run_kubectl_ate admin make-jwt-pool \
    --key-id="1" \
    --name="actor-id-jwt-pool" \
    --secret-namespace=ate-system
}

create_actor_id_ca_pool_secret() {
  log_step "create_actor_id_ca_pool_secret"
  run_kubectl_ate admin make-ca-pool \
    --ca-id="1" \
    --name="actor-id-ca-pool" \
    --secret-namespace=ate-system
}

# Same settings as hack/install-ate.sh: the flag values resolved from these
# env vars are path- and DNS-compatible with the cert-manager overlay (the
# valkey cert carries valkey-cluster.ate-system.svc, and the podidentity
# client-cert path is unchanged).
create_api_server_env_vars() {
  log_step "create_api_server_env_vars"
  local jwt_issuer=""
  jwt_issuer=$(run_kubectl get --raw /.well-known/openid-configuration 2>/dev/null | grep -o '"issuer":"[^"]*' | sed 's/"issuer":"//' || true)
  if [[ -z "${jwt_issuer}" ]]; then
    jwt_issuer="https://kubernetes.default.svc"
  fi

  run_kubectl create configmap -n ate-system ate-api-server-envvars \
    --from-literal=ATE_API_REDIS_ADDRESS="valkey-cluster.ate-system.svc:6379" \
    --from-literal=ATE_API_REDIS_USE_IAM_AUTH="false" \
    --from-literal=ATE_API_REDIS_TLS_SERVER_NAME="valkey-cluster.ate-system.svc" \
    --from-literal=ATE_API_REDIS_CLIENT_CERT="/run/podidentity.podcert.ate.dev/credential-bundle.pem" \
    --from-literal=ATE_API_K8SJWT_ISSUER="${jwt_issuer}" \
    --dry-run=client -o yaml \
    | run_kubectl apply -f -
}

ensure_apiserver_prerequisites() {
  log_step "ensure_apiserver_prerequisites"
  run_kubectl get secret -n ate-system actor-id-jwt-pool >/dev/null 2>&1 \
    || create_jwt_authority_pool_secret
  run_kubectl get secret -n ate-system actor-id-ca-pool >/dev/null 2>&1 \
    || create_actor_id_ca_pool_secret
  run_kubectl get configmap -n ate-system ate-api-server-envvars >/dev/null 2>&1 \
    || create_api_server_env_vars
}

wait_for_pki() {
  log_step "Waiting for CA and component Certificates to be Ready..."
  run_kubectl wait --for=condition=Ready certificate/servicedns-ca certificate/podidentity-ca \
    -n cert-manager --timeout=120s
  run_kubectl wait --for=condition=Ready certificate --all \
    -n ate-system --timeout=120s

  log_step "Waiting for trust-manager Bundles to sync into ate-system..."
  local cm=""
  for cm in servicedns-ca-bundle podidentity-ca-bundle valkey-ca-bundle; do
    until run_kubectl get configmap -n ate-system "${cm}" >/dev/null 2>&1; do
      sleep 1
    done
  done
}

deploy_ate_system() {
  log_step "deploy_ate_system (cert-manager PKI, router=$(atenet_router))"
  ensure_cert_manager_stack

  run_ko apply -f manifests/ate-install/generated
  run_kubectl apply -f manifests/ate-install/sandboxconfig-validation.yaml
  run_kubectl apply -f manifests/ate-install/sandboxconfig-gvisor.yaml

  run_kubectl apply -f manifests/ate-install/ate-system-namespace.yaml \
    && run_kubectl wait --for=jsonpath='{.status.phase}'=Active namespace/ate-system --timeout=60s

  # Applied ahead of the bundle for the same reason as in install-ate.sh:
  # every workload pulls this ConfigMap in via envFrom and will not start
  # without it.
  run_kubectl apply -f manifests/ate-install/ate-otel-config.yaml

  ensure_apiserver_prerequisites

  local manifests=""
  manifests="$(kubectl kustomize "$(overlay_dir)" --load-restrictor LoadRestrictionsNone | run_ko resolve -f -)"
  echo "${manifests}" | run_kubectl apply -f -

  wait_for_pki

  log_step "Waiting for ATE system components to be ready..."
  run_kubectl rollout status deployment/ate-api-server -n ate-system --timeout=180s
  run_kubectl rollout status deployment/ate-controller -n ate-system --timeout=180s
  run_kubectl rollout status deployment/atenet-router -n ate-system --timeout=180s
  run_kubectl rollout status statefulset/valkey-cluster -n ate-system --timeout=180s
  run_kubectl rollout status daemonset/atelet -n ate-system --timeout=180s

  echo ""
  echo "Done. For every namespace that hosts WorkerPools, issue the worker identity:"
  echo "  $0 --create-worker-cert <namespace>"
}

# Issues the worker identity Certificate a --worker-cert-source=cert-manager
# controller expects in a WorkerPool namespace (Secret
# ate-worker-podidentity-cert; see workerpool_apply.go).
create_worker_cert() {
  local ns="$1"
  log_step "create_worker_cert (${ns})"
  run_kubectl get namespace "${ns}" >/dev/null 2>&1 || {
    echo "error: namespace ${ns} not found" >&2
    exit 1
  }
  sed "s/WORKER_NAMESPACE/${ns}/g" \
    manifests/ate-install/cert-manager-pki/worker-certificate.example.yaml \
    | run_kubectl apply -f -
  run_kubectl wait --for=condition=Ready certificate/ate-worker-podidentity \
    -n "${ns}" --timeout=120s
}

delete_ate_system() {
  log_step "delete_ate_system"
  kubectl kustomize "$(overlay_dir)" --load-restrictor LoadRestrictionsNone \
    | run_kubectl delete --ignore-not-found -f -
  run_kubectl delete --ignore-not-found -f manifests/ate-install/generated
}

if [ "$#" -eq 0 ]; then
  usage
  exit 1
fi

for arg in "$@"; do
  case "$arg" in
    -h|--help)
      usage
      exit 0
      ;;
  esac
done

# Pre-scan --atenet-router so it can appear before or after the action flag.
prescan_args=("$@")
for ((i = 0; i < ${#prescan_args[@]}; i++)); do
  case "${prescan_args[i]}" in
    --atenet-router=*) ATE_ATENET_ROUTER="${prescan_args[i]#*=}" ;;
    --atenet-router)
      if (( i + 1 >= ${#prescan_args[@]} )); then
        echo "Error: --atenet-router requires envoy or agentgateway" >&2
        exit 1
      fi
      ATE_ATENET_ROUTER="${prescan_args[$((i + 1))]}"
      ;;
  esac
done
atenet_router >/dev/null

while [[ "$#" -gt 0 ]]; do
  case $1 in
    --atenet-router=*) ;;
    --atenet-router) shift ;;

    --deploy-ate-system) deploy_ate_system ;;
    --delete-ate-system) delete_ate_system ;;

    --create-worker-cert)
      shift
      if [[ "$#" -eq 0 ]]; then
        echo "Error: --create-worker-cert requires a namespace" >&2
        exit 1
      fi
      create_worker_cert "$1"
      ;;
    --create-worker-cert=*) create_worker_cert "${1#*=}" ;;

    *)
      echo "Error: unknown option: $1" >&2
      echo ""
      usage
      exit 1
      ;;
  esac
  shift
done
