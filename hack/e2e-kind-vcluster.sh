#!/usr/bin/env bash
set -euo pipefail

KUBECTL=${KUBECTL:-kubectl}
HELM=${HELM:-helm}
VCLUSTER=${VCLUSTER:-vcluster}
KIND_CLUSTER=${KIND_CLUSTER:-infisical-entity-operator}
OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE:-infisical-entity-operator-system}
INFISICAL_NAMESPACE=${INFISICAL_NAMESPACE:-infisical}
INFISICAL_SERVICE=${INFISICAL_SERVICE:-infisical-infisical-standalone-infisical}
IMG=${IMG:-ghcr.io/winrarr/infisical-entity-operator:dev}
VCLUSTER_VERSION=${VCLUSTER_VERSION:-0.37.1}
TEST_NAMESPACE=${TEST_NAMESPACE:-infisical-entity-operator-vcluster-e2e}
VCLUSTER_NAMESPACE=${VCLUSTER_NAMESPACE:-infisical-entity-operator-vcluster}
VCLUSTER_NAME=${VCLUSTER_NAME:-tenant-a}
TENANT_OPERATOR_NAMESPACE=${TENANT_OPERATOR_NAMESPACE:-infisical-entity-operator-system}
RUN_ID=${RUN_ID:-$(date +%s)}
TENANT_ORGANIZATION_NAME=${TENANT_ORGANIZATION_NAME:-e2e-vcluster-org-${RUN_ID}}
TENANT_PROJECT_NAME=${TENANT_PROJECT_NAME:-e2e-vcluster-project-${RUN_ID}}
TENANT_IDENTITY_NAME=${TENANT_IDENTITY_NAME:-e2e-vcluster-tenant-${RUN_ID}}
port_forward_pid=""

host_kubectl() {
  "${KUBECTL}" --context="kind-${KIND_CLUSTER}" "$@"
}

host_helm() {
  "${HELM}" --kube-context="kind-${KIND_CLUSTER}" "$@"
}

delete_remote_organization() {
  local organization_id="$1"
  local token="$2"
  [[ -z "${organization_id}" ]] && return 0
  curl --silent --show-error --max-time 10 \
    -H "Authorization: Bearer ${token}" \
    -X DELETE "http://127.0.0.1:18081/api/v2/organizations/${organization_id}" >/dev/null 2>&1 || true
}

tenant_kubectl() {
  "${VCLUSTER}" connect "${VCLUSTER_NAME}" --namespace "${VCLUSTER_NAMESPACE}" \
    --context "kind-${KIND_CLUSTER}" --background-proxy=false --silent -- kubectl "$@"
}

tenant_helm() {
  "${VCLUSTER}" connect "${VCLUSTER_NAME}" --namespace "${VCLUSTER_NAMESPACE}" \
    --context "kind-${KIND_CLUSTER}" --background-proxy=false --silent -- helm "$@"
}

cleanup() {
  tenant_kubectl -n "${TEST_NAMESPACE}" delete infisicalenvironment/tenant-environment --ignore-not-found --wait=true --timeout=5m >/dev/null 2>&1 || true
  tenant_kubectl -n "${TEST_NAMESPACE}" delete infisicalproject/tenant-project --ignore-not-found --wait=true --timeout=5m >/dev/null 2>&1 || true
  tenant_kubectl -n "${TEST_NAMESPACE}" delete infisicalorganization/other-boundary --ignore-not-found --wait=false >/dev/null 2>&1 || true
  tenant_kubectl -n "${TEST_NAMESPACE}" delete infisicalorganization/tenant-boundary --ignore-not-found --wait=false >/dev/null 2>&1 || true
  tenant_kubectl delete namespace "${TENANT_OPERATOR_NAMESPACE}" --ignore-not-found --wait=false >/dev/null 2>&1 || true
  host_kubectl -n "${TEST_NAMESPACE}" delete infisicalidentity/vcluster-tenant-identity --ignore-not-found --wait=true --timeout=5m >/dev/null 2>&1 || true
  host_kubectl -n "${TEST_NAMESPACE}" delete infisicalproject/vcluster-anchor --ignore-not-found --wait=true --timeout=5m >/dev/null 2>&1 || true
  host_kubectl -n "${TEST_NAMESPACE}" delete infisicalorganization/vcluster-organization --ignore-not-found --wait=true --timeout=5m >/dev/null 2>&1 || true
  host_kubectl -n "${TEST_NAMESPACE}" delete infisicalconnection/tenant-admin infisicalconnection/platform --ignore-not-found --wait=true --timeout=5m >/dev/null 2>&1 || true
  host_kubectl -n "${TEST_NAMESPACE}" delete secret/infisical-tenant-admin-token secret/infisical-platform-token --ignore-not-found >/dev/null 2>&1 || true
  host_kubectl delete namespace "${VCLUSTER_NAMESPACE}" --ignore-not-found --wait=true --timeout=5m >/dev/null 2>&1 || true
  host_kubectl delete namespace "${TEST_NAMESPACE}" --ignore-not-found --wait=true --timeout=5m >/dev/null 2>&1 || true
  delete_remote_organization "${tenant_organization_id:-}" "${tenant_admin_token:-}"
  delete_remote_organization "${other_organization_id:-}" "${other_admin_token:-}"
  if [[ -n "${port_forward_pid}" ]]; then
    kill "${port_forward_pid}" >/dev/null 2>&1 || true
    wait "${port_forward_pid}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

wait_host_ready() {
  local resource="$1"
  host_kubectl -n "${TEST_NAMESPACE}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
    "${resource}" --timeout=5m >/dev/null
}

wait_tenant_ready() {
  local resource="$1"
  local ready
  for _ in {1..150}; do
    ready="$(tenant_kubectl -n "${TEST_NAMESPACE}" get "${resource}" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || true)"
    if [[ "${ready}" == "True" ]]; then
      return 0
    fi
    sleep 2
  done
  echo "${resource} in vCluster did not become Ready" >&2
  tenant_kubectl -n "${TEST_NAMESPACE}" get "${resource}" -o yaml >&2 || true
  return 1
}

wait_tenant_condition() {
  local resource="$1"
  local expected_reason="$2"
  local reason
  for _ in {1..90}; do
    reason="$(tenant_kubectl -n "${TEST_NAMESPACE}" get "${resource}" -o jsonpath='{.status.conditions[?(@.type=="Ready")].reason}' 2>/dev/null || true)"
    if [[ "${reason}" == "${expected_reason}" ]]; then
      return 0
    fi
    sleep 2
  done
  echo "${resource} did not reach condition reason ${expected_reason}" >&2
  tenant_kubectl -n "${TEST_NAMESPACE}" get "${resource}" -o yaml >&2 || true
  return 1
}

if [[ "${IMG}" != *:* ]]; then
  echo "IMG must include a tag for the mutable Kind image test: ${IMG}" >&2
  exit 1
fi
image_repository=${IMG%:*}
image_tag=${IMG##*:}

host_kubectl delete namespace "${VCLUSTER_NAMESPACE}" "${TEST_NAMESPACE}" --ignore-not-found --wait=true --timeout=5m >/dev/null
host_kubectl create namespace "${TEST_NAMESPACE}" >/dev/null

host_kubectl -n "${INFISICAL_NAMESPACE}" port-forward "service/${INFISICAL_SERVICE}" 18081:8080 >/dev/null 2>&1 &
port_forward_pid=$!
port_forward_ready=false
for _ in {1..60}; do
  if curl --silent --show-error --max-time 2 -o /dev/null "http://127.0.0.1:18081/api/v1"; then
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

admin_email="$(host_kubectl -n "${INFISICAL_NAMESPACE}" get secret infisical-bootstrap-credentials -o jsonpath='{.data.INFISICAL_ADMIN_EMAIL}' | base64 --decode)"
admin_password="$(host_kubectl -n "${INFISICAL_NAMESPACE}" get secret infisical-bootstrap-credentials -o jsonpath='{.data.INFISICAL_ADMIN_PASSWORD}' | base64 --decode)"
login_response="$(curl --silent --show-error --max-time 10 -H 'Content-Type: application/json' -H 'User-Agent: infisical-entity-operator-e2e' \
  -X POST http://127.0.0.1:18081/api/v3/auth/login \
  --data "$(jq -cn --arg email "${admin_email}" --arg password "${admin_password}" '{email:$email,password:$password}')")"
if ! platform_token="$(jq -er '.accessToken' <<<"${login_response}")"; then
  echo "Infisical admin login failed: ${login_response}" >&2
  exit 1
fi

host_kubectl -n "${TEST_NAMESPACE}" create secret generic infisical-platform-token \
  --from-literal=token="${platform_token}" --dry-run=client -o yaml | host_kubectl apply -f - >/dev/null
host_kubectl apply -f config/network-policy/allow-infisical-egress-network-policy.yaml >/dev/null
host_kubectl -n "${OPERATOR_NAMESPACE}" wait --for=condition=available \
  deployment/infisical-entity-operator-infisical-entity-operator --timeout=5m >/dev/null

cat <<EOF | host_kubectl -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalConnection
metadata:
  name: platform
spec:
  hostAPI: http://${INFISICAL_SERVICE}.${INFISICAL_NAMESPACE}.svc.cluster.local:8080/api
  authSecretRef:
    name: infisical-platform-token
    key: token
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalOrganization
metadata:
  name: vcluster-organization
spec:
  connectionRef:
    name: platform
  organizationName: ${TENANT_ORGANIZATION_NAME}
  creationPolicy: Create
  deletionPolicy: Orphan
EOF
wait_host_ready infisicalorganization/vcluster-organization

tenant_organization_id="$(host_kubectl -n "${TEST_NAMESPACE}" get infisicalorganization/vcluster-organization -o jsonpath='{.status.organizationID}')"
if [[ -z "${tenant_organization_id}" ]]; then
  echo "platform organization did not publish an Infisical organization ID" >&2
  exit 1
fi
tenant_admin_response="$(curl --silent --show-error --max-time 10 -H 'Content-Type: application/json' -H "Authorization: Bearer ${platform_token}" \
  -X POST http://127.0.0.1:18081/api/v3/auth/select-organization \
  --data "$(jq -cn --arg id "${tenant_organization_id}" '{organizationId:$id}')")"
if ! tenant_admin_token="$(jq -er '.token' <<<"${tenant_admin_response}")"; then
  echo "Infisical organization selection failed: ${tenant_admin_response}" >&2
  exit 1
fi
host_kubectl -n "${TEST_NAMESPACE}" create secret generic infisical-tenant-admin-token \
  --from-literal=token="${tenant_admin_token}" --dry-run=client -o yaml | host_kubectl apply -f - >/dev/null

cat <<EOF | host_kubectl -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalConnection
metadata:
  name: tenant-admin
spec:
  hostAPI: http://${INFISICAL_SERVICE}.${INFISICAL_NAMESPACE}.svc.cluster.local:8080/api
  authSecretRef:
    name: infisical-tenant-admin-token
    key: token
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalProject
metadata:
  name: vcluster-anchor
spec:
  connectionRef:
    name: tenant-admin
  organizationRef:
    name: vcluster-organization
  projectName: ${TENANT_ORGANIZATION_NAME}-anchor
  slug: ${TENANT_ORGANIZATION_NAME}-anchor
  creationPolicy: Create
  deletionPolicy: Delete
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalIdentity
metadata:
  name: vcluster-tenant-identity
spec:
  connectionRef:
    name: tenant-admin
  scope: Organization
  organizationRef:
    name: vcluster-organization
  organizationRole: admin
  identityName: ${TENANT_IDENTITY_NAME}
  metadata:
    - key: test
      value: vcluster-tenant-operator
  creationPolicy: Create
  deletionPolicy: Delete
EOF
wait_host_ready infisicalproject/vcluster-anchor
wait_host_ready infisicalidentity/vcluster-tenant-identity

tenant_identity_id="$(host_kubectl -n "${TEST_NAMESPACE}" get infisicalidentity/vcluster-tenant-identity -o jsonpath='{.status.identityID}')"
tenant_organization_role="$(host_kubectl -n "${TEST_NAMESPACE}" get infisicalidentity/vcluster-tenant-identity -o jsonpath='{.status.organizationRole}')"
if [[ -z "${tenant_identity_id}" || "${tenant_organization_role}" != "admin" ]]; then
  echo "vCluster tenant identity did not become an organization admin" >&2
  host_kubectl -n "${TEST_NAMESPACE}" get infisicalidentity/vcluster-tenant-identity -o yaml >&2
  exit 1
fi

universal_auth_response="$(curl --silent --show-error --max-time 10 \
  -H "Authorization: Bearer ${tenant_admin_token}" -H 'Content-Type: application/json' \
  -X POST "http://127.0.0.1:18081/api/v1/auth/universal-auth/identities/${tenant_identity_id}" \
  --data '{"clientSecretTrustedIps":[{"ipAddress":"0.0.0.0/0"},{"ipAddress":"::/0"}],"accessTokenTrustedIps":[{"ipAddress":"0.0.0.0/0"},{"ipAddress":"::/0"}],"accessTokenTTL":7200,"accessTokenMaxTTL":7200,"accessTokenNumUsesLimit":0,"accessTokenPeriod":0,"lockoutEnabled":true,"lockoutThreshold":3,"lockoutDurationSeconds":300,"lockoutCounterResetSeconds":30}')"
if ! jq -e '.identityUniversalAuth' >/dev/null <<<"${universal_auth_response}"; then
  echo "Infisical rejected the Universal Auth configuration request: ${universal_auth_response}" >&2
  exit 1
fi
tenant_client_id="$(jq -er '.identityUniversalAuth.clientId' <<<"${universal_auth_response}")"
client_secret_response="$(curl --silent --show-error --max-time 10 \
  -H "Authorization: Bearer ${tenant_admin_token}" -H 'Content-Type: application/json' \
  -X POST "http://127.0.0.1:18081/api/v1/auth/universal-auth/identities/${tenant_identity_id}/client-secrets" \
  --data '{"description":"vcluster integration test","numUsesLimit":1,"ttl":3600}')"
if ! client_secret="$(jq -er '.clientSecret' <<<"${client_secret_response}")"; then
  echo "Infisical rejected the Universal Auth client-secret request: ${client_secret_response}" >&2
  exit 1
fi
tenant_token_response="$(curl --silent --show-error --max-time 10 -H 'Content-Type: application/json' \
  -X POST http://127.0.0.1:18081/api/v1/auth/universal-auth/login \
  --data "$(jq -cn --arg clientId "${tenant_client_id}" --arg clientSecret "${client_secret}" '{clientId:$clientId,clientSecret:$clientSecret}')")"
if ! tenant_token="$(jq -er '.accessToken' <<<"${tenant_token_response}")"; then
  echo "Infisical Universal Auth login failed: ${tenant_token_response}" >&2
  exit 1
fi

second_org_response="$(curl --silent --show-error --max-time 10 -H 'Content-Type: application/json' -H "Authorization: Bearer ${platform_token}" \
  -X POST http://127.0.0.1:18081/api/v2/organizations \
  --data "$(jq -cn --arg name "e2e-vcluster-other-${RUN_ID}" '{name:$name}')")"
other_organization_id="$(jq -er '.organization.id' <<<"${second_org_response}")"
other_admin_response="$(curl --silent --show-error --max-time 10 -H 'Content-Type: application/json' -H "Authorization: Bearer ${platform_token}" \
  -X POST http://127.0.0.1:18081/api/v3/auth/select-organization \
  --data "$(jq -cn --arg id "${other_organization_id}" '{organizationId:$id}')")"
other_admin_token="$(jq -er '.token' <<<"${other_admin_response}")"

"${VCLUSTER}" create "${VCLUSTER_NAME}" --namespace "${VCLUSTER_NAMESPACE}" \
  --context "kind-${KIND_CLUSTER}" --chart-version "${VCLUSTER_VERSION}" \
  --connect=false --background-proxy=false
tenant_kubectl create namespace "${TENANT_OPERATOR_NAMESPACE}" --dry-run=client -o yaml | tenant_kubectl apply -f - >/dev/null
tenant_kubectl -n "${TENANT_OPERATOR_NAMESPACE}" create secret generic infisical-token \
  --from-literal=token="${tenant_token}" --dry-run=client -o yaml | tenant_kubectl apply -f - >/dev/null
tenant_helm upgrade --install tenant-operator charts/infisical-entity-operator \
  --namespace "${TENANT_OPERATOR_NAMESPACE}" --create-namespace \
  --set image.repository="${image_repository}" --set image.tag="${image_tag}" \
  --set leaderElection=false --wait --timeout=5m >/dev/null
tenant_kubectl -n "${TENANT_OPERATOR_NAMESPACE}" wait --for=condition=available \
  deployment/tenant-operator-infisical-entity-operator --timeout=5m >/dev/null

host_api="http://$(host_kubectl -n "${INFISICAL_NAMESPACE}" get service "${INFISICAL_SERVICE}" -o jsonpath='{.spec.clusterIP}'):8080/api"
tenant_kubectl create namespace "${TEST_NAMESPACE}" --dry-run=client -o yaml | tenant_kubectl apply -f - >/dev/null
cat <<EOF | tenant_kubectl -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: v1
kind: Secret
metadata:
  name: infisical-token
type: Opaque
stringData:
  token: ${tenant_token}
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalConnection
metadata:
  name: infisical
spec:
  hostAPI: ${host_api}
  authSecretRef:
    name: infisical-token
    key: token
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalOrganization
metadata:
  name: tenant-boundary
spec:
  connectionRef:
    name: infisical
  organizationID: ${tenant_organization_id}
  creationPolicy: Adopt
  deletionPolicy: Orphan
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalProject
metadata:
  name: tenant-project
spec:
  connectionRef:
    name: infisical
  organizationRef:
    name: tenant-boundary
  projectName: ${TENANT_PROJECT_NAME}
  slug: ${TENANT_PROJECT_NAME}
  creationPolicy: Create
  deletionPolicy: Delete
EOF
wait_tenant_ready infisicalconnection/infisical
wait_tenant_ready infisicalorganization/tenant-boundary
wait_tenant_ready infisicalproject/tenant-project

cat <<EOF | tenant_kubectl -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalEnvironment
metadata:
  name: tenant-environment
spec:
  connectionRef:
    name: infisical
  projectRef:
    name: tenant-project
  environmentName: Tenant
  slug: tenant
  creationPolicy: Create
  deletionPolicy: Orphan
EOF
wait_tenant_ready infisicalenvironment/tenant-environment

tenant_project_id="$(tenant_kubectl -n "${TEST_NAMESPACE}" get infisicalproject/tenant-project -o jsonpath='{.status.projectID}')"
tenant_environment_id="$(tenant_kubectl -n "${TEST_NAMESPACE}" get infisicalenvironment/tenant-environment -o jsonpath='{.status.environmentID}')"
if [[ -z "${tenant_project_id}" || -z "${tenant_environment_id}" ]]; then
  echo "vCluster operator did not reconcile the tenant-created project and environment" >&2
  exit 1
fi

cat <<EOF | tenant_kubectl -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalOrganization
metadata:
  name: other-boundary
spec:
  connectionRef:
    name: infisical
  organizationID: ${other_organization_id}
  creationPolicy: Adopt
  deletionPolicy: Orphan
EOF
wait_tenant_condition infisicalorganization/other-boundary RemoteOrganizationMissing
if [[ "$(tenant_kubectl -n "${TEST_NAMESPACE}" get infisicalorganization/other-boundary -o jsonpath='{.status.organizationID}')" != "" ]]; then
  echo "tenant credential unexpectedly adopted another organization" >&2
  exit 1
fi

echo "vCluster scenario passed: platform organization creation, organization-admin Universal Auth, tenant-side adoption, tenant-created project, and cross-organization rejection"
echo "Tenant organization ID: ${tenant_organization_id}"
echo "Tenant project status ID: ${tenant_project_id}"
echo "Tenant environment status ID: ${tenant_environment_id}"
echo "Cross-organization adoption rejection: $(tenant_kubectl -n "${TEST_NAMESPACE}" get infisicalorganization/other-boundary -o jsonpath='{.status.conditions[?(@.type=="Ready")].message}')"
