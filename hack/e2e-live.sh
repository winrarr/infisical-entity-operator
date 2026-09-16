#!/usr/bin/env bash
set -euo pipefail

KUBECTL=${KUBECTL:-kubectl}
TEST_NAMESPACE=${TEST_NAMESPACE:-infisical-entity-operator-live-e2e}
RUN_ID=${RUN_ID:-$(date +%s)}
INFISICAL_HOST_API=${INFISICAL_E2E_HOST_API:-}
INFISICAL_TOKEN=${INFISICAL_E2E_TOKEN:-}
KUBERNETES_HOST=${INFISICAL_E2E_KUBERNETES_HOST:-}
KUBERNETES_CA_FILE=${INFISICAL_E2E_KUBERNETES_CA_FILE:-}
CONNECTION_SECRET=infisical-live-e2e-token
REVIEWER_SERVICE_ACCOUNT=infisical-live-e2e-reviewer
REVIEWER_BINDING=infisical-live-e2e-reviewer
REVIEWER_SECRET=infisical-live-e2e-reviewer-token
ALLOWED_SERVICE_ACCOUNT=infisical-live-e2e-allowed
DISALLOWED_SERVICE_ACCOUNT=infisical-live-e2e-disallowed
PROJECT_RESOURCE=live-project
IDENTITY_RESOURCE=live-identity
ROLE_RESOURCE=live-project-role
AUTH_RESOURCE=live-kubernetes-auth
PROJECT_NAME="infisical-entity-operator-live-${RUN_ID}"
PROJECT_SLUG="infisical-entity-operator-live-${RUN_ID}"
IDENTITY_NAME="infisical-entity-operator-live-${RUN_ID}"

required_variables=(
  INFISICAL_E2E_HOST_API
  INFISICAL_E2E_TOKEN
  INFISICAL_E2E_KUBERNETES_HOST
)
for variable in "${required_variables[@]}"; do
  if [[ -z "${!variable:-}" ]]; then
    echo "${variable} must be set for live acceptance" >&2
    exit 1
  fi
done

if [[ ! "${INFISICAL_HOST_API}" =~ /api/?$ ]]; then
  echo "INFISICAL_E2E_HOST_API must end in /api" >&2
  exit 1
fi

cleanup() {
  "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalkubernetesauth/"${AUTH_RESOURCE}" --ignore-not-found --wait=true --timeout=5m >/dev/null 2>&1 || true
  "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalprojectrole/"${ROLE_RESOURCE}" --ignore-not-found --wait=true --timeout=5m >/dev/null 2>&1 || true
  "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalidentity/"${IDENTITY_RESOURCE}" --ignore-not-found --wait=true --timeout=5m >/dev/null 2>&1 || true
  "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalproject/"${PROJECT_RESOURCE}" --ignore-not-found --wait=true --timeout=5m >/dev/null 2>&1 || true
  "${KUBECTL}" delete clusterrolebinding "${REVIEWER_BINDING}" --ignore-not-found >/dev/null 2>&1 || true
  "${KUBECTL}" delete namespace "${TEST_NAMESPACE}" --ignore-not-found --wait=false >/dev/null 2>&1 || true
}
trap cleanup EXIT

wait_for_ready() {
  local resource="$1"
  local condition
  for _ in {1..150}; do
    condition="$(${KUBECTL} -n "${TEST_NAMESPACE}" get "${resource}" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || true)"
    if [[ "${condition}" == "True" ]]; then
      return 0
    fi
    sleep 2
  done
  echo "${resource} did not become Ready" >&2
  "${KUBECTL}" -n "${TEST_NAMESPACE}" get "${resource}" -o yaml >&2 || true
  return 1
}

create_ca_secret() {
  if [[ -n "${KUBERNETES_CA_FILE}" ]]; then
    if [[ ! -f "${KUBERNETES_CA_FILE}" ]]; then
      echo "INFISICAL_E2E_KUBERNETES_CA_FILE does not exist: ${KUBERNETES_CA_FILE}" >&2
      exit 1
    fi
    "${KUBECTL}" -n "${TEST_NAMESPACE}" create secret generic e2e-kubernetes-ca \
      --from-file=ca.crt="${KUBERNETES_CA_FILE}" --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null
    return
  fi

  local ca_data ca_file
  ca_data="$(${KUBECTL} config view --raw --minify -o jsonpath='{.clusters[0].cluster.certificate-authority-data}')"
  if [[ -n "${ca_data}" ]]; then
    "${KUBECTL}" -n "${TEST_NAMESPACE}" create secret generic e2e-kubernetes-ca \
      --from-file=ca.crt=<(printf '%s' "${ca_data}" | base64 --decode) \
      --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null
    return
  fi

  ca_file="$(${KUBECTL} config view --raw --minify -o jsonpath='{.clusters[0].cluster.certificate-authority}')"
  if [[ -z "${ca_file}" || ! -f "${ca_file}" ]]; then
    echo "the current kubeconfig does not contain certificate-authority-data or a readable certificate-authority file" >&2
    exit 1
  fi
  "${KUBECTL}" -n "${TEST_NAMESPACE}" create secret generic e2e-kubernetes-ca \
    --from-file=ca.crt="${ca_file}" --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null
}

create_service_account_credentials() {
  "${KUBECTL}" -n "${TEST_NAMESPACE}" create serviceaccount "${REVIEWER_SERVICE_ACCOUNT}" \
    --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null
  "${KUBECTL}" create clusterrolebinding "${REVIEWER_BINDING}" \
    --clusterrole=system:auth-delegator \
    --serviceaccount="${TEST_NAMESPACE}:${REVIEWER_SERVICE_ACCOUNT}" \
    --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null
  local reviewer_token
  reviewer_token="$(${KUBECTL} -n "${TEST_NAMESPACE}" create token "${REVIEWER_SERVICE_ACCOUNT}" --duration=24h)"
  if [[ -z "${reviewer_token}" ]]; then
    echo "Kubernetes token reviewer service account produced an empty token" >&2
    exit 1
  fi
  "${KUBECTL}" -n "${TEST_NAMESPACE}" create secret generic "${REVIEWER_SECRET}" \
    --from-literal=token="${reviewer_token}" --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null
}

login_status() {
  local token="$1"
  local response_file status
  response_file="$(mktemp)"
  status="$(curl --silent --show-error --max-time 20 -o "${response_file}" -w '%{http_code}' \
    -H 'Content-Type: application/json' -X POST "${INFISICAL_HOST_API%/}/v1/auth/kubernetes-auth/login" \
    --data "$(jq -cn --arg identity_id "${identity_id}" --arg jwt "${token}" '{identityId:$identity_id,jwt:$jwt}')")"
  rm -f "${response_file}"
  printf '%s' "${status}"
}

"${KUBECTL}" delete namespace "${TEST_NAMESPACE}" --ignore-not-found --wait=true --timeout=5m >/dev/null
"${KUBECTL}" create namespace "${TEST_NAMESPACE}" >/dev/null

"${KUBECTL}" -n "${TEST_NAMESPACE}" create secret generic "${CONNECTION_SECRET}" \
  --from-literal=token="${INFISICAL_TOKEN}" --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null
create_ca_secret
create_service_account_credentials

"${KUBECTL}" -n "${TEST_NAMESPACE}" create serviceaccount "${ALLOWED_SERVICE_ACCOUNT}" \
  --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null
"${KUBECTL}" -n "${TEST_NAMESPACE}" create serviceaccount "${DISALLOWED_SERVICE_ACCOUNT}" \
  --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null

cat <<EOF | "${KUBECTL}" -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalConnection
metadata:
  name: infisical
spec:
  hostAPI: ${INFISICAL_HOST_API}
  authSecretRef:
    name: ${CONNECTION_SECRET}
    key: token
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalProject
metadata:
  name: ${PROJECT_RESOURCE}
spec:
  connectionRef:
    name: infisical
  projectName: ${PROJECT_NAME}
  slug: ${PROJECT_SLUG}
  creationPolicy: Create
  deletionPolicy: Delete
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalIdentity
metadata:
  name: ${IDENTITY_RESOURCE}
spec:
  connectionRef:
    name: infisical
  projectRef:
    name: ${PROJECT_RESOURCE}
  identityName: ${IDENTITY_NAME}
  roleSlugs:
    - no-access
  creationPolicy: Create
  deletionPolicy: Delete
EOF

"${KUBECTL}" -n "${TEST_NAMESPACE}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalconnection/infisical --timeout=5m
wait_for_ready infisicalproject/"${PROJECT_RESOURCE}"
wait_for_ready infisicalidentity/"${IDENTITY_RESOURCE}"

project_id="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalproject/"${PROJECT_RESOURCE}" -o jsonpath='{.status.projectID}')"
identity_id="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalidentity/"${IDENTITY_RESOURCE}" -o jsonpath='{.status.identityID}')"
if [[ -z "${project_id}" || -z "${identity_id}" ]]; then
  echo "live resources did not publish their Infisical identifiers" >&2
  exit 1
fi

cat <<EOF | "${KUBECTL}" -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalProjectRole
metadata:
  name: ${ROLE_RESOURCE}
spec:
  connectionRef:
    name: infisical
  projectRef:
    name: ${PROJECT_RESOURCE}
  roleName: Live E2E Secret Reader
  slug: live-e2e-secret-reader-${RUN_ID}
  permissions:
    - subject: secrets
      action:
        - describeSecret
        - readValue
      conditions:
        environment:
          \$eq: E2E
  creationPolicy: Create
  deletionPolicy: Delete
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalKubernetesAuth
metadata:
  name: ${AUTH_RESOURCE}
spec:
  connectionRef:
    name: infisical
  identityRef:
    name: ${IDENTITY_RESOURCE}
  kubernetesHost: ${KUBERNETES_HOST}
  allowedNamespaces:
    - ${TEST_NAMESPACE}
  allowedNames:
    - ${ALLOWED_SERVICE_ACCOUNT}
  caCertSecretRef:
    name: e2e-kubernetes-ca
    key: ca.crt
  verifyTLSCertificate: true
  tokenReviewerJWTSecretRef:
    name: ${REVIEWER_SECRET}
    key: token
  tokenReviewMode: api
  creationPolicy: Create
  deletionPolicy: Delete
EOF

wait_for_ready infisicalprojectrole/"${ROLE_RESOURCE}"
wait_for_ready infisicalkubernetesauth/"${AUTH_RESOURCE}"
role_id="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalprojectrole/"${ROLE_RESOURCE}" -o jsonpath='{.status.roleID}')"
auth_id="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalkubernetesauth/"${AUTH_RESOURCE}" -o jsonpath='{.status.authID}')"
if [[ -z "${role_id}" || -z "${auth_id}" ]]; then
  echo "live ProjectRole or Kubernetes Auth did not publish its Infisical identifier" >&2
  exit 1
fi

allowed_token="$(${KUBECTL} -n "${TEST_NAMESPACE}" create token "${ALLOWED_SERVICE_ACCOUNT}" --duration=10m)"
allowed_status="$(login_status "${allowed_token}")"
if [[ "${allowed_status}" != 2* ]]; then
  echo "allowed service account failed Kubernetes Auth login with HTTP ${allowed_status}" >&2
  exit 1
fi

disallowed_token="$(${KUBECTL} -n "${TEST_NAMESPACE}" create token "${DISALLOWED_SERVICE_ACCOUNT}" --duration=10m)"
disallowed_status="$(login_status "${disallowed_token}")"
if [[ "${disallowed_status}" == 2* ]]; then
  echo "disallowed service account unexpectedly succeeded with Kubernetes Auth" >&2
  exit 1
fi

echo "Live acceptance succeeded against ${INFISICAL_HOST_API}"
echo "Project status ID: ${project_id}"
echo "Project role status ID: ${role_id}"
echo "Identity status ID: ${identity_id}"
echo "Kubernetes Auth status ID: ${auth_id}"
echo "Allowed login HTTP status: ${allowed_status}"
echo "Disallowed login HTTP status: ${disallowed_status}"
