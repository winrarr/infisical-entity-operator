#!/usr/bin/env bash
set -euo pipefail

KUBECTL=${KUBECTL:-kubectl}
OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE:-infisical-entity-operator-system}
INFISICAL_NAMESPACE=${INFISICAL_NAMESPACE:-infisical}
TEST_NAMESPACE=${TEST_NAMESPACE:-infisical-entity-operator-e2e}
E2E_NETWORKING=${E2E_NETWORKING:-default-cni}
NETWORK_POLICY_FILE=${NETWORK_POLICY_FILE:-config/network-policy/allow-infisical-egress-network-policy.yaml}
REVIEWER_SERVICE_ACCOUNT=${REVIEWER_SERVICE_ACCOUNT:-infisical-auth-reviewer}
REVIEWER_BINDING=${REVIEWER_BINDING:-infisical-auth-reviewer}
KUBERNETES_AUTH_REVIEW_URL=${KUBERNETES_AUTH_REVIEW_URL:-https://kubernetes.default.svc}
KUBERNETES_AUTH_VERIFY_TLS=${KUBERNETES_AUTH_VERIFY_TLS:-true}
KUBERNETES_AUTH_EXPECTED_FAILURE=${KUBERNETES_AUTH_EXPECTED_FAILURE-"Local IPs not allowed as URL"}
port_forward_pid=""

case "${KUBERNETES_AUTH_VERIFY_TLS}" in
  true|false) ;;
  *)
    echo "KUBERNETES_AUTH_VERIFY_TLS must be true or false" >&2
    exit 1
    ;;
esac

kubernetes_ca_cert_ref=""
if [[ "${KUBERNETES_AUTH_VERIFY_TLS}" == true ]]; then
  kubernetes_ca_cert_ref=$'  caCertSecretRef:\n    name: e2e-kubernetes-ca\n    key: ca.crt'
fi

if [[ ! -f "${NETWORK_POLICY_FILE}" ]]; then
  echo "Network policy manifest does not exist: ${NETWORK_POLICY_FILE}" >&2
  exit 1
fi

cleanup() {
  if [[ -n "${port_forward_pid}" ]]; then
    kill "${port_forward_pid}" >/dev/null 2>&1 || true
    wait "${port_forward_pid}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${token:-}" ]]; then
    "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalkubernetesauth/e2e-kubernetes-auth --ignore-not-found --wait=true >/dev/null 2>&1 || true
    "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalenvironment/e2e-environment --ignore-not-found --wait=true >/dev/null 2>&1 || true
    "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalorganization/e2e-organization --ignore-not-found --wait=true >/dev/null 2>&1 || true
    "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalidentity/e2e-tenant-identity --ignore-not-found --wait=true >/dev/null 2>&1 || true
    "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalidentity/e2e-identity --ignore-not-found --wait=true >/dev/null 2>&1 || true
    "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalproject/e2e-tenant-secondary --ignore-not-found --wait=true >/dev/null 2>&1 || true
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
    if [[ -n "${known_message}" && "${message}" == *"${known_message}"* ]]; then
      echo "${label} is unavailable in this local Infisical configuration: ${message}"
      return 2
    fi
    sleep 2
  done
  echo "${label} did not become Ready" >&2
  "${KUBECTL}" -n "${TEST_NAMESPACE}" get "${resource}" -o yaml >&2 || true
  return 1
}

assert_api_rejects() {
  local label="$1"
  local expected="$2"
  local output
  if output="$(${KUBECTL} apply --dry-run=server -f - 2>&1)"; then
    echo "${label} was unexpectedly accepted by the Kubernetes API server" >&2
    return 1
  fi
  if [[ "${output}" != *"${expected}"* ]]; then
    echo "${label} was rejected with an unexpected error: ${output}" >&2
    return 1
  fi
  echo "${label} was rejected by the Kubernetes API server"
}

"${KUBECTL}" create namespace "${TEST_NAMESPACE}" --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null

assert_api_rejects "organizationID with creationPolicy Create" "organizationID cannot be set with creationPolicy Create" <<EOF
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalOrganization
metadata:
  name: invalid-organization
  namespace: ${TEST_NAMESPACE}
spec:
  connectionRef:
    name: infisical
  organizationID: org-1
  creationPolicy: Create
EOF

assert_api_rejects "Kubernetes Auth TLS verification without a CA" "caCertSecretRef is required when verifyTLSCertificate is true" <<EOF
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalKubernetesAuth
metadata:
  name: invalid-kubernetes-auth
  namespace: ${TEST_NAMESPACE}
spec:
  connectionRef:
    name: infisical
  identityRef:
    name: e2e-identity
  allowedNamespaces:
    - ${TEST_NAMESPACE}
  allowedNames:
    - e2e-authenticated
  verifyTLSCertificate: true
EOF

assert_api_rejects "Kubernetes Auth CA with disabled TLS verification" "caCertSecretRef cannot be set when verifyTLSCertificate is false" <<EOF
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalKubernetesAuth
metadata:
  name: invalid-kubernetes-auth
  namespace: ${TEST_NAMESPACE}
spec:
  connectionRef:
    name: infisical
  identityRef:
    name: e2e-identity
  allowedNamespaces:
    - ${TEST_NAMESPACE}
  allowedNames:
    - e2e-authenticated
  verifyTLSCertificate: false
  caCertSecretRef:
    name: e2e-kubernetes-ca
    key: ca.crt
EOF

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

"${KUBECTL}" apply -f "${NETWORK_POLICY_FILE}" >/dev/null
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
kind: InfisicalIdentity
metadata:
  name: e2e-identity
spec:
  connectionRef:
    name: infisical
  projectRef:
    name: e2e-project
  identityName: e2e-identity
  roleSlugs:
    - no-access
  metadata:
    - key: test
      value: ${E2E_NETWORKING}
  creationPolicy: Create
  deletionPolicy: Delete
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalProject
metadata:
  name: e2e-tenant-secondary
spec:
  connectionRef:
    name: infisical
  projectName: e2e-tenant-secondary
  slug: e2e-tenant-secondary
  description: Second project for organization identity membership testing
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
cat <<EOF | "${KUBECTL}" -n "${TEST_NAMESPACE}" apply -f -
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
EOF
"${KUBECTL}" -n "${TEST_NAMESPACE}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalproject/e2e-project --timeout=5m
"${KUBECTL}" -n "${TEST_NAMESPACE}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalproject/e2e-tenant-secondary --timeout=5m
"${KUBECTL}" -n "${TEST_NAMESPACE}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalidentity/e2e-identity --timeout=5m
"${KUBECTL}" -n "${TEST_NAMESPACE}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalenvironment/e2e-environment --timeout=5m

organization_id="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalproject/e2e-project -o jsonpath='{.status.organizationID}')"
if [[ -z "${organization_id}" ]]; then
  echo "the project did not expose an Infisical organization ID" >&2
  exit 1
fi
cat <<EOF | "${KUBECTL}" -n "${TEST_NAMESPACE}" apply -f -
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalOrganization
metadata:
  name: e2e-organization
spec:
  connectionRef:
    name: infisical
  organizationID: ${organization_id}
  creationPolicy: Adopt
  deletionPolicy: Orphan
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalIdentity
metadata:
  name: e2e-tenant-identity
spec:
  connectionRef:
    name: infisical
  scope: Organization
  organizationRef:
    name: e2e-organization
  projectRoleBindings:
    - projectRef:
        name: e2e-project
      roleSlugs:
        - no-access
    - projectRef:
        name: e2e-tenant-secondary
      roleSlugs:
        - member
  metadata:
    - key: test
      value: kind-organization-identity
  creationPolicy: Create
  deletionPolicy: Delete
EOF
"${KUBECTL}" -n "${TEST_NAMESPACE}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalorganization/e2e-organization --timeout=5m
"${KUBECTL}" -n "${TEST_NAMESPACE}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalidentity/e2e-tenant-identity --timeout=5m

cat <<EOF | "${KUBECTL}" -n "${TEST_NAMESPACE}" apply -f -
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalKubernetesAuth
metadata:
  name: e2e-kubernetes-auth
spec:
  connectionRef:
    name: infisical
  identityRef:
    name: e2e-identity
  kubernetesHost: ${KUBERNETES_AUTH_REVIEW_URL}
  allowedNamespaces:
    - ${TEST_NAMESPACE}
  allowedNames:
    - e2e-authenticated
${kubernetes_ca_cert_ref}
  verifyTLSCertificate: ${KUBERNETES_AUTH_VERIFY_TLS}
  tokenReviewerJWTSecretRef:
    name: e2e-kubernetes-token-reviewer
    key: token
  tokenReviewMode: api
  creationPolicy: Create
  deletionPolicy: Delete
EOF

kubernetes_auth_available=true
if wait_for_ready_or_known_block infisicalkubernetesauth/e2e-kubernetes-auth "${KUBERNETES_AUTH_EXPECTED_FAILURE}" "InfisicalKubernetesAuth"; then
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
membership_id="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalidentity e2e-identity -o jsonpath='{.status.membershipID}')"
identity_role_slug="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalidentity e2e-identity -o jsonpath='{.status.roles[0].slug}')"
tenant_identity_id="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalidentity e2e-tenant-identity -o jsonpath='{.status.identityID}')"
tenant_organization_id="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalidentity e2e-tenant-identity -o jsonpath='{.status.organizationID}')"
tenant_organization_role="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalidentity e2e-tenant-identity -o jsonpath='{.status.organizationRole}')"
tenant_membership_one="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalidentity e2e-tenant-identity -o jsonpath='{.status.projectMemberships[0].membershipID}')"
tenant_membership_two="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalidentity e2e-tenant-identity -o jsonpath='{.status.projectMemberships[1].membershipID}')"
tenant_role_one="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalidentity e2e-tenant-identity -o jsonpath='{.status.projectMemberships[0].roles[0].slug}')"
tenant_role_two="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalidentity e2e-tenant-identity -o jsonpath='{.status.projectMemberships[1].roles[0].slug}')"
environment_id="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalenvironment e2e-environment -o jsonpath='{.status.environmentID}')"
auth_id="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalkubernetesauth e2e-kubernetes-auth -o jsonpath='{.status.authID}')"
if [[ -z "${project_id}" || -z "${identity_id}" || -z "${membership_id}" || "${identity_role_slug}" != "no-access" || -z "${tenant_identity_id}" || -z "${tenant_organization_id}" || "${tenant_organization_role}" != "no-access" || -z "${tenant_membership_one}" || -z "${tenant_membership_two}" || "${tenant_role_one}" != "no-access" || "${tenant_role_two}" != "member" || -z "${environment_id}" ]]; then
  echo "unexpected identity membership status: projectID=${project_id} identityID=${identity_id} membershipID=${membership_id} roleSlug=${identity_role_slug} environmentID=${environment_id}" >&2
  echo "tenant identity status: identityID=${tenant_identity_id} organizationID=${tenant_organization_id} organizationRole=${tenant_organization_role} membershipOne=${tenant_membership_one} membershipTwo=${tenant_membership_two} roleOne=${tenant_role_one} roleTwo=${tenant_role_two}" >&2
  "${KUBECTL}" -n "${TEST_NAMESPACE}" get infisicalidentity/e2e-identity infisicalidentity/e2e-tenant-identity -o yaml >&2 || true
  exit 1
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
  echo "Kubernetes Auth allowed service account login: passed"
  echo "Kubernetes Auth disallowed service account login: rejected as expected"
fi

echo "Live reconciliation succeeded with ${E2E_NETWORKING} and ${NETWORK_POLICY_FILE}"
echo "Project status ID: ${project_id}"
echo "Identity status ID: ${identity_id}"
echo "Environment status ID: ${environment_id}"
if [[ "${kubernetes_auth_available}" == true ]]; then
  echo "Kubernetes Auth status ID: ${auth_id}"
else
  echo "Kubernetes Auth live check skipped because the local Infisical API rejects the cluster-local review URL"
fi
