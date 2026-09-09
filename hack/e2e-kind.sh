#!/usr/bin/env bash
set -euo pipefail

KUBECTL=${KUBECTL:-kubectl}
OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE:-infisical-entity-operator-system}
INFISICAL_NAMESPACE=${INFISICAL_NAMESPACE:-infisical}
TEST_NAMESPACE=${TEST_NAMESPACE:-infisical-entity-operator-e2e}
REVIEWER_SERVICE_ACCOUNT=${REVIEWER_SERVICE_ACCOUNT:-infisical-auth-reviewer}
REVIEWER_BINDING=${REVIEWER_BINDING:-infisical-auth-reviewer}
port_forward_pid=""

cleanup() {
  if [[ -n "${port_forward_pid}" ]]; then
    kill "${port_forward_pid}" >/dev/null 2>&1 || true
    wait "${port_forward_pid}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${token:-}" ]]; then
    "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalkubernetesauth/e2e-kubernetes-auth --ignore-not-found --wait=true >/dev/null 2>&1 || true
    "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalprojectrole/e2e-project-role --ignore-not-found --wait=true >/dev/null 2>&1 || true
    "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalenvironment/e2e-environment --ignore-not-found --wait=true >/dev/null 2>&1 || true
    "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalidentity/e2e-identity --ignore-not-found --wait=true >/dev/null 2>&1 || true
    "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalproject/e2e-project --ignore-not-found --wait=true >/dev/null 2>&1 || true
    "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalconnection/infisical --ignore-not-found --wait=true >/dev/null 2>&1 || true
  fi
  "${KUBECTL}" delete clusterrolebinding "${REVIEWER_BINDING}" --ignore-not-found >/dev/null 2>&1 || true
  "${KUBECTL}" delete namespace "${TEST_NAMESPACE}" --ignore-not-found --wait=false >/dev/null 2>&1 || true
}
trap cleanup EXIT

wait_for_ready_or_known_block() {
  local resource="$1"
  local known_message="$2"
  local label="$3"
  local ready message
  for _ in {1..150}; do
    ready="$(${KUBECTL} -n "${TEST_NAMESPACE}" get "${resource}" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || true)"
    if [[ "${ready}" == "True" ]]; then
      return 0
    fi
    message="$(${KUBECTL} -n "${TEST_NAMESPACE}" get "${resource}" -o jsonpath='{.status.conditions[?(@.type=="Ready")].message}' 2>/dev/null || true)"
    if [[ "${message}" == *"${known_message}"* ]]; then
      echo "${label} is unavailable in this local Infisical configuration: ${message}"
      return 2
    fi
    sleep 2
  done
  echo "${label} did not become Ready" >&2
  "${KUBECTL}" -n "${TEST_NAMESPACE}" get "${resource}" -o yaml >&2 || true
  return 1
}

"${KUBECTL}" create namespace "${TEST_NAMESPACE}" --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null

token="$(${KUBECTL} -n "${INFISICAL_NAMESPACE}" get secret infisical-bootstrap-token -o jsonpath='{.data.token}' | base64 --decode)"
if [[ -z "${token}" ]]; then
  echo "Infisical bootstrap secret contained an empty token" >&2
  exit 1
fi

"${KUBECTL}" -n "${TEST_NAMESPACE}" create secret generic infisical-token \
  --from-literal=token="${token}" --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null

ca_data="$(${KUBECTL} config view --raw --minify -o jsonpath='{.clusters[0].cluster.certificate-authority-data}')"
if [[ -z "${ca_data}" ]]; then
  echo "Kind kubeconfig did not contain certificate-authority-data" >&2
  exit 1
fi
"${KUBECTL}" -n "${TEST_NAMESPACE}" create secret generic e2e-kubernetes-ca \
  --from-file=ca.crt=<(printf '%s' "${ca_data}" | base64 --decode) \
  --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null

"${KUBECTL}" -n "${TEST_NAMESPACE}" create serviceaccount "${REVIEWER_SERVICE_ACCOUNT}" \
  --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null
"${KUBECTL}" create clusterrolebinding "${REVIEWER_BINDING}" \
  --clusterrole=system:auth-delegator \
  --serviceaccount="${TEST_NAMESPACE}:${REVIEWER_SERVICE_ACCOUNT}" \
  --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null
reviewer_token="$(${KUBECTL} -n "${TEST_NAMESPACE}" create token "${REVIEWER_SERVICE_ACCOUNT}" --duration=24h)"
if [[ -z "${reviewer_token}" ]]; then
  echo "Kubernetes token reviewer service account produced an empty token" >&2
  exit 1
fi
"${KUBECTL}" -n "${TEST_NAMESPACE}" create secret generic e2e-kubernetes-token-reviewer \
  --from-literal=token="${reviewer_token}" --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null

"${KUBECTL}" -n "${TEST_NAMESPACE}" create serviceaccount e2e-authenticated \
  --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null
"${KUBECTL}" -n "${TEST_NAMESPACE}" create serviceaccount e2e-disallowed \
  --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null

"${KUBECTL}" apply -f config/network-policy/allow-infisical-egress.yaml >/dev/null
"${KUBECTL}" -n "${OPERATOR_NAMESPACE}" wait --for=condition=available \
  deployment/infisical-entity-operator-infisical-entity-operator --timeout=5m >/dev/null

cat <<EOF | "${KUBECTL}" -n "${TEST_NAMESPACE}" apply -f -
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalConnection
metadata:
  name: infisical
spec:
  hostAPI: http://infisical-infisical-standalone-infisical.${INFISICAL_NAMESPACE}.svc.cluster.local:8080/api
  authSecretRef:
    name: infisical-token
    key: token
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalProject
metadata:
  name: e2e-project
spec:
  connectionRef:
    name: infisical
  projectName: e2e-project
  slug: e2e-project
  description: Reconciled by the live Kind test
  creationPolicy: Create
  deletionPolicy: Delete
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalIdentity
metadata:
  name: e2e-identity
spec:
  connectionRef:
    name: infisical
  projectRef:
    name: e2e-project
  identityName: e2e-identity
  metadata:
    - key: test
      value: kind-cilium
  creationPolicy: Create
  deletionPolicy: Delete
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalEnvironment
metadata:
  name: e2e-environment
spec:
  connectionRef:
    name: infisical
  projectRef:
    name: e2e-project
  environmentName: E2E
  slug: e2e
  position: 4
  creationPolicy: Create
  deletionPolicy: Delete
---
EOF

"${KUBECTL}" -n "${TEST_NAMESPACE}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalconnection/infisical --timeout=5m
"${KUBECTL}" -n "${TEST_NAMESPACE}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalproject/e2e-project --timeout=5m
"${KUBECTL}" -n "${TEST_NAMESPACE}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalidentity/e2e-identity --timeout=5m
"${KUBECTL}" -n "${TEST_NAMESPACE}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalenvironment/e2e-environment --timeout=5m

cat <<EOF | "${KUBECTL}" -n "${TEST_NAMESPACE}" apply -f -
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalProjectRole
metadata:
  name: e2e-project-role
spec:
  connectionRef:
    name: infisical
  projectRef:
    name: e2e-project
  roleName: E2E Secret Reader
  slug: e2e-secret-reader
  permissions:
    - subject: secrets
      action:
        - readValue
      conditions:
        environment:
          \$eq: e2e
  creationPolicy: Create
  deletionPolicy: Delete
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalKubernetesAuth
metadata:
  name: e2e-kubernetes-auth
spec:
  connectionRef:
    name: infisical
  identityRef:
    name: e2e-identity
  kubernetesHost: https://kubernetes.default.svc
  allowedNamespaces:
    - ${TEST_NAMESPACE}
  allowedNames:
    - e2e-authenticated
  caCertSecretRef:
    name: e2e-kubernetes-ca
    key: ca.crt
  verifyTLSCertificate: true
  tokenReviewerJWTSecretRef:
    name: e2e-kubernetes-token-reviewer
    key: token
  tokenReviewMode: api
  creationPolicy: Create
  deletionPolicy: Delete
EOF

project_role_available=true
if wait_for_ready_or_known_block infisicalprojectrole/e2e-project-role "plan RBAC restriction" "InfisicalProjectRole"; then
  :
else
  wait_result=$?
  if [[ "${wait_result}" != 2 ]]; then
    exit "${wait_result}"
  fi
  project_role_available=false
fi
kubernetes_auth_available=true
if wait_for_ready_or_known_block infisicalkubernetesauth/e2e-kubernetes-auth "Local IPs not allowed as URL" "InfisicalKubernetesAuth"; then
  :
else
  wait_result=$?
  if [[ "${wait_result}" != 2 ]]; then
    exit "${wait_result}"
  fi
  kubernetes_auth_available=false
fi

project_id="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalproject e2e-project -o jsonpath='{.status.projectID}')"
identity_id="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalidentity e2e-identity -o jsonpath='{.status.identityID}')"
environment_id="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalenvironment e2e-environment -o jsonpath='{.status.environmentID}')"
role_id="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalprojectrole e2e-project-role -o jsonpath='{.status.roleID}')"
auth_id="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalkubernetesauth e2e-kubernetes-auth -o jsonpath='{.status.authID}')"
[[ -n "${project_id}" && -n "${identity_id}" && -n "${environment_id}" ]]
if [[ "${project_role_available}" == true ]]; then
  [[ -n "${role_id}" ]]
fi
if [[ "${kubernetes_auth_available}" == true ]]; then
  [[ -n "${auth_id}" ]]
fi

if [[ "${kubernetes_auth_available}" == true ]]; then
  "${KUBECTL}" -n "${INFISICAL_NAMESPACE}" port-forward service/infisical-infisical-standalone-infisical 18080:8080 >/dev/null 2>&1 &
  port_forward_pid=$!
  port_forward_ready=false
  for _ in {1..30}; do
    if curl --silent --show-error --max-time 2 -o /dev/null http://127.0.0.1:18080/api/v1; then
      port_forward_ready=true
      break
    fi
    if ! kill -0 "${port_forward_pid}" >/dev/null 2>&1; then
      echo "Infisical port-forward exited before becoming ready" >&2
      exit 1
    fi
    sleep 1
  done
  if [[ "${port_forward_ready}" != true ]]; then
    echo "Infisical port-forward did not become ready" >&2
    exit 1
  fi

  authenticated_token="$(${KUBECTL} -n "${TEST_NAMESPACE}" create token e2e-authenticated --duration=10m)"
  authenticated_status="$(curl --silent --show-error --max-time 10 -o /dev/null -w '%{http_code}' \
    -H 'Content-Type: application/json' -X POST http://127.0.0.1:18080/api/v1/auth/kubernetes-auth/login \
    --data "{\"identityId\":\"${identity_id}\",\"jwt\":\"${authenticated_token}\"}")"
  if [[ "${authenticated_status}" != 2* ]]; then
    echo "allowed service account failed Kubernetes Auth login with HTTP ${authenticated_status}" >&2
    exit 1
  fi

  disallowed_token="$(${KUBECTL} -n "${TEST_NAMESPACE}" create token e2e-disallowed --duration=10m)"
  disallowed_status="$(curl --silent --show-error --max-time 10 -o /dev/null -w '%{http_code}' \
    -H 'Content-Type: application/json' -X POST http://127.0.0.1:18080/api/v1/auth/kubernetes-auth/login \
    --data "{\"identityId\":\"${identity_id}\",\"jwt\":\"${disallowed_token}\"}")"
  if [[ "${disallowed_status}" == 2* ]]; then
    echo "disallowed service account unexpectedly succeeded with Kubernetes Auth" >&2
    exit 1
  fi
fi

echo "Live reconciliation succeeded through the Cilium egress policy"
echo "Project status ID: ${project_id}"
echo "Identity status ID: ${identity_id}"
echo "Environment status ID: ${environment_id}"
if [[ "${project_role_available}" == true ]]; then
  echo "Project role status ID: ${role_id}"
else
  echo "Project role live check skipped because the local Infisical plan disallows custom roles"
fi
if [[ "${kubernetes_auth_available}" == true ]]; then
  echo "Kubernetes Auth status ID: ${auth_id}"
else
  echo "Kubernetes Auth live check skipped because the local Infisical API rejects the cluster-local review URL"
fi
