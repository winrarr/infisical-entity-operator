#!/usr/bin/env bash
set -euo pipefail

KUBECTL=${KUBECTL:-kubectl}
HELM=${HELM:-helm}
KIND_CLUSTER=${KIND_CLUSTER:-infisical-entity-operator}
OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE:-infisical-entity-operator-system}
INFISICAL_NAMESPACE=${INFISICAL_NAMESPACE:-infisical}
INFISICAL_SERVICE=${INFISICAL_SERVICE:-infisical-infisical-standalone-infisical}
IMG=${IMG:-ghcr.io/winrarr/infisical-entity-operator:dev}
CAPSULE_VERSION=${CAPSULE_VERSION:-0.14.5}
KYVERNO_VERSION=${KYVERNO_VERSION:-3.9.1}
CERT_MANAGER_VERSION=${CERT_MANAGER_VERSION:-1.21.1}
TEST_NAMESPACE=${TEST_NAMESPACE:-infisical-entity-operator-capsule-e2e}
TENANT_A=${TENANT_A:-e2e-capsule-a}
TENANT_B=${TENANT_B:-e2e-capsule-b}
NAMESPACE_A=${NAMESPACE_A:-e2e-capsule-a}
NAMESPACE_B=${NAMESPACE_B:-e2e-capsule-b}
OWNER_A=${OWNER_A:-gitops}
OWNER_B=${OWNER_B:-gitops}
OWNER_ROLE=${OWNER_ROLE:-e2e-capsule-infisical-owner}
CAPSULE_ADMIN_USER=${CAPSULE_ADMIN_USER:-kubernetes-admin}
RUN_ID=${RUN_ID:-$(date +%s)}
TENANT_A_ORGANIZATION_NAME=${TENANT_A_ORGANIZATION_NAME:-e2e-capsule-a-org-${RUN_ID}}
TENANT_B_ORGANIZATION_NAME=${TENANT_B_ORGANIZATION_NAME:-e2e-capsule-b-org-${RUN_ID}}
TENANT_A_PROJECT_NAME=${TENANT_A_PROJECT_NAME:-e2e-capsule-a-project-${RUN_ID}}
TENANT_B_PROJECT_NAME=${TENANT_B_PROJECT_NAME:-e2e-capsule-b-project-${RUN_ID}}
port_forward_pid=""

host_kubectl() {
  "${KUBECTL}" --context="kind-${KIND_CLUSTER}" "$@"
}

host_helm() {
  "${HELM}" --kube-context="kind-${KIND_CLUSTER}" "$@"
}

as_tenant() {
  local namespace="$1"
  local owner="$2"
  shift 2
  host_kubectl --as="system:serviceaccount:${namespace}:${owner}" \
    --as-group=system:serviceaccounts --as-group="system:serviceaccounts:${namespace}" \
    --as-group=system:authenticated "$@"
}

cleanup() {
  host_kubectl -n "${NAMESPACE_A}" delete infisicalenvironment/tenant-environment infisicalproject/tenant-project infisicalorganization/tenant-boundary --ignore-not-found --wait=true --timeout=5m >/dev/null 2>&1 || true
  host_kubectl -n "${NAMESPACE_B}" delete infisicalproject/tenant-project infisicalorganization/tenant-boundary --ignore-not-found --wait=true --timeout=5m >/dev/null 2>&1 || true
  host_kubectl delete clusterpolicy e2e-capsule-project-a e2e-capsule-project-b \
    --ignore-not-found >/dev/null 2>&1 || true
  host_kubectl delete tenant "${TENANT_A}" "${TENANT_B}" --ignore-not-found --wait=true --timeout=5m >/dev/null 2>&1 || true
  host_kubectl delete clusterrole "${OWNER_ROLE}" --ignore-not-found >/dev/null 2>&1 || true
  host_kubectl delete namespace "${NAMESPACE_A}" "${NAMESPACE_B}" "${TEST_NAMESPACE}" \
    --ignore-not-found --wait=true --timeout=5m >/dev/null 2>&1 || true
  if [[ -n "${tenant_a_anchor_project_id:-}" && -n "${tenant_a_admin_token:-}" ]]; then
    curl --silent --show-error --max-time 10 -H "Authorization: Bearer ${tenant_a_admin_token}" \
      -X DELETE "http://127.0.0.1:18082/api/v1/projects/${tenant_a_anchor_project_id}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${tenant_b_anchor_project_id:-}" && -n "${tenant_b_admin_token:-}" ]]; then
    curl --silent --show-error --max-time 10 -H "Authorization: Bearer ${tenant_b_admin_token}" \
      -X DELETE "http://127.0.0.1:18082/api/v1/projects/${tenant_b_anchor_project_id}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${tenant_a_organization_id:-}" && -n "${tenant_a_admin_token:-}" ]]; then
    curl --silent --show-error --max-time 10 -H "Authorization: Bearer ${tenant_a_admin_token}" \
      -X DELETE "http://127.0.0.1:18082/api/v2/organizations/${tenant_a_organization_id}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${tenant_b_organization_id:-}" && -n "${tenant_b_admin_token:-}" ]]; then
    curl --silent --show-error --max-time 10 -H "Authorization: Bearer ${tenant_b_admin_token}" \
      -X DELETE "http://127.0.0.1:18082/api/v2/organizations/${tenant_b_organization_id}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${port_forward_pid}" ]]; then
    kill "${port_forward_pid}" >/dev/null 2>&1 || true
    wait "${port_forward_pid}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

if [[ "${IMG}" != *:* ]]; then
  echo "IMG must include a tag for the mutable Kind image test: ${IMG}" >&2
  exit 1
fi

host_helm upgrade --install kyverno oci://ghcr.io/kyverno/charts/kyverno \
  --version "${KYVERNO_VERSION}" --namespace kyverno --create-namespace \
  --set admissionController.replicas=1 --set backgroundController.replicas=1 \
  --set cleanupController.replicas=1 --set reportsController.replicas=1 \
  --wait --timeout=10m >/dev/null
host_helm upgrade --install cert-manager oci://quay.io/jetstack/charts/cert-manager \
  --version "${CERT_MANAGER_VERSION}" --namespace cert-manager --create-namespace \
  --set crds.enabled=true --wait --timeout=10m >/dev/null
host_helm upgrade --install capsule oci://ghcr.io/projectcapsule/charts/capsule \
  --version "${CAPSULE_VERSION}" --namespace capsule-system --create-namespace \
  --set proxy.enabled=false \
  --set "manager.options.administrators[0].kind=User" \
  --set "manager.options.administrators[0].name=${CAPSULE_ADMIN_USER}" \
  --wait --timeout=10m >/dev/null

host_kubectl -n kyverno wait --for=condition=available deployment/kyverno-admission-controller --timeout=5m >/dev/null
host_kubectl -n capsule-system wait --for=condition=available deployment/capsule-controller-manager --timeout=5m >/dev/null
host_kubectl -n "${OPERATOR_NAMESPACE}" wait --for=condition=available \
  deployment/infisical-entity-operator-infisical-entity-operator --timeout=5m >/dev/null

host_kubectl delete namespace "${NAMESPACE_A}" "${NAMESPACE_B}" "${TEST_NAMESPACE}" \
  --ignore-not-found --wait=true --timeout=5m >/dev/null
host_kubectl create namespace "${TEST_NAMESPACE}" >/dev/null

cat <<EOF | host_kubectl apply -f - >/dev/null
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ${OWNER_ROLE}
rules:
  - apiGroups:
      - infisical.infisical-operator.io
    resources:
      - infisicalconnections
      - infisicalenvironments
      - infisicalidentities
      - infisicalkubernetesauths
      - infisicalorganizations
      - infisicalprojectroles
      - infisicalprojects
    verbs: [create, delete, get, list, patch, update, watch]
---
apiVersion: capsule.clastix.io/v1beta2
kind: Tenant
metadata:
  name: ${TENANT_A}
spec:
  owners:
    - name: system:serviceaccount:${NAMESPACE_A}:${OWNER_A}
      kind: ServiceAccount
      clusterRoles:
        - ${OWNER_ROLE}
---
apiVersion: capsule.clastix.io/v1beta2
kind: Tenant
metadata:
  name: ${TENANT_B}
spec:
  owners:
    - name: system:serviceaccount:${NAMESPACE_B}:${OWNER_B}
      kind: ServiceAccount
      clusterRoles:
        - ${OWNER_ROLE}
EOF

host_kubectl wait --for=jsonpath='{.status.state}'=Active "tenant/${TENANT_A}" --timeout=5m >/dev/null
host_kubectl wait --for=jsonpath='{.status.state}'=Active "tenant/${TENANT_B}" --timeout=5m >/dev/null

tenant_a_uid="$(host_kubectl get tenant "${TENANT_A}" -o jsonpath='{.metadata.uid}')"
tenant_b_uid="$(host_kubectl get tenant "${TENANT_B}" -o jsonpath='{.metadata.uid}')"
# The platform creates the tenant namespace shells with their Capsule ownership
# references. Tenant identities and resources are submitted through the tenant
# service accounts below.
cat <<EOF | host_kubectl create -f - >/dev/null
apiVersion: v1
kind: Namespace
metadata:
  name: ${NAMESPACE_A}
  labels:
    capsule.clastix.io/tenant: ${TENANT_A}
  ownerReferences:
    - apiVersion: capsule.clastix.io/v1beta2
      kind: Tenant
      name: ${TENANT_A}
      uid: ${tenant_a_uid}
      controller: true
      blockOwnerDeletion: true
EOF
cat <<EOF | host_kubectl create -f - >/dev/null
apiVersion: v1
kind: Namespace
metadata:
  name: ${NAMESPACE_B}
  labels:
    capsule.clastix.io/tenant: ${TENANT_B}
  ownerReferences:
    - apiVersion: capsule.clastix.io/v1beta2
      kind: Tenant
      name: ${TENANT_B}
      uid: ${tenant_b_uid}
      controller: true
      blockOwnerDeletion: true
EOF
host_kubectl -n "${NAMESPACE_A}" create serviceaccount "${OWNER_A}" >/dev/null
host_kubectl -n "${NAMESPACE_B}" create serviceaccount "${OWNER_B}" >/dev/null

host_kubectl -n "${INFISICAL_NAMESPACE}" port-forward "service/${INFISICAL_SERVICE}" 18082:8080 >/dev/null 2>&1 &
port_forward_pid=$!
port_forward_ready=false
for _ in {1..60}; do
  if curl --silent --show-error --max-time 2 -o /dev/null http://127.0.0.1:18082/api/v1; then
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
  -X POST http://127.0.0.1:18082/api/v3/auth/login \
  --data "$(jq -cn --arg email "${admin_email}" --arg password "${admin_password}" '{email:$email,password:$password}')")"
if ! platform_token="$(jq -er '.accessToken' <<<"${login_response}")"; then
  echo "Infisical admin login failed: ${login_response}" >&2
  exit 1
fi

tenant_a_org_response="$(curl --silent --show-error --max-time 10 -H 'Content-Type: application/json' -H "Authorization: Bearer ${platform_token}" \
  -X POST http://127.0.0.1:18082/api/v2/organizations \
  --data "$(jq -cn --arg name "${TENANT_A_ORGANIZATION_NAME}" '{name:$name}')")"
tenant_a_organization_id="$(jq -er '.organization.id' <<<"${tenant_a_org_response}")"
tenant_b_org_response="$(curl --silent --show-error --max-time 10 -H 'Content-Type: application/json' -H "Authorization: Bearer ${platform_token}" \
  -X POST http://127.0.0.1:18082/api/v2/organizations \
  --data "$(jq -cn --arg name "${TENANT_B_ORGANIZATION_NAME}" '{name:$name}')")"
tenant_b_organization_id="$(jq -er '.organization.id' <<<"${tenant_b_org_response}")"

select_organization() {
  local organization_id="$1"
  curl --silent --show-error --max-time 10 -H 'Content-Type: application/json' -H "Authorization: Bearer ${platform_token}" \
    -X POST http://127.0.0.1:18082/api/v3/auth/select-organization \
    --data "$(jq -cn --arg id "${organization_id}" '{organizationId:$id}')" | jq -er '.token'
}

tenant_a_admin_token="$(select_organization "${tenant_a_organization_id}")"
tenant_b_admin_token="$(select_organization "${tenant_b_organization_id}")"

create_anchor_project() {
  local token="$1"
  local project_name="$2"
  local response
  response="$(curl --silent --show-error --max-time 10 -H 'Content-Type: application/json' -H "Authorization: Bearer ${token}" \
    -X POST http://127.0.0.1:18082/api/v1/projects \
    --data "$(jq -cn --arg name "${project_name}" '{projectName:$name,projectDescription:"Capsule tenant boundary anchor",slug:$name,template:"default",type:"secret-manager",shouldCreateDefaultEnvs:true,hasDeleteProtection:false}')")"
  jq -er '.project.id' <<<"${response}"
}

create_tenant_identity() {
  local token="$1"
  local organization_id="$2"
  local identity_name="$3"
  local response
  response="$(curl --silent --show-error --max-time 10 -H 'Content-Type: application/json' -H "Authorization: Bearer ${token}" \
    -X POST http://127.0.0.1:18082/api/v1/identities \
    --data "$(jq -cn --arg name "${identity_name}" --arg organizationID "${organization_id}" '{name:$name,organizationId:$organizationID,role:"admin",hasDeleteProtection:false}')")"
  jq -er '.identity.id' <<<"${response}"
}

configure_universal_auth() {
  local token="$1"
  local identity_id="$2"
  local description="$3"
  local universal_auth_response client_id client_secret_response client_secret token_response
  universal_auth_response="$(curl --silent --show-error --max-time 10 -H "Authorization: Bearer ${token}" -H 'Content-Type: application/json' \
    -X POST "http://127.0.0.1:18082/api/v1/auth/universal-auth/identities/${identity_id}" \
    --data '{"clientSecretTrustedIps":[{"ipAddress":"0.0.0.0/0"},{"ipAddress":"::/0"}],"accessTokenTrustedIps":[{"ipAddress":"0.0.0.0/0"},{"ipAddress":"::/0"}],"accessTokenTTL":7200,"accessTokenMaxTTL":7200,"accessTokenNumUsesLimit":0,"accessTokenPeriod":0,"lockoutEnabled":true,"lockoutThreshold":3,"lockoutDurationSeconds":300,"lockoutCounterResetSeconds":30}')"
  client_id="$(jq -er '.identityUniversalAuth.clientId' <<<"${universal_auth_response}")"
  client_secret_response="$(curl --silent --show-error --max-time 10 -H "Authorization: Bearer ${token}" -H 'Content-Type: application/json' \
    -X POST "http://127.0.0.1:18082/api/v1/auth/universal-auth/identities/${identity_id}/client-secrets" \
    --data "$(jq -cn --arg description "${description}" '{description:$description,numUsesLimit:1,ttl:3600}')")"
  client_secret="$(jq -er '.clientSecret' <<<"${client_secret_response}")"
  token_response="$(curl --silent --show-error --max-time 10 -H 'Content-Type: application/json' \
    -X POST http://127.0.0.1:18082/api/v1/auth/universal-auth/login \
    --data "$(jq -cn --arg clientId "${client_id}" --arg clientSecret "${client_secret}" '{clientId:$clientId,clientSecret:$clientSecret}')")"
  jq -er '.accessToken' <<<"${token_response}"
}

tenant_a_anchor_project_id="$(create_anchor_project "${tenant_a_admin_token}" "${TENANT_A_PROJECT_NAME}-anchor")"
tenant_b_anchor_project_id="$(create_anchor_project "${tenant_b_admin_token}" "${TENANT_B_PROJECT_NAME}-anchor")"
tenant_a_identity_id="$(create_tenant_identity "${tenant_a_admin_token}" "${tenant_a_organization_id}" "${TENANT_A}-operator")"
tenant_b_identity_id="$(create_tenant_identity "${tenant_b_admin_token}" "${tenant_b_organization_id}" "${TENANT_B}-operator")"
tenant_a_token="$(configure_universal_auth "${tenant_a_admin_token}" "${tenant_a_identity_id}" "Capsule tenant A integration test")"
tenant_b_token="$(configure_universal_auth "${tenant_b_admin_token}" "${tenant_b_identity_id}" "Capsule tenant B integration test")"

host_kubectl -n "${NAMESPACE_A}" create secret generic infisical-token \
  --from-literal=token="${tenant_a_token}" --dry-run=client -o yaml | host_kubectl apply -f - >/dev/null
host_kubectl -n "${NAMESPACE_B}" create secret generic infisical-token \
  --from-literal=token="${tenant_b_token}" --dry-run=client -o yaml | host_kubectl apply -f - >/dev/null

cat <<EOF | host_kubectl apply -f - >/dev/null
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: e2e-capsule-project-a
spec:
  admission: true
  background: false
  validationFailureAction: Enforce
  rules:
    - name: organization-must-belong-to-capsule-tenant-a
      match:
        any:
          - resources:
              kinds: [InfisicalOrganization]
              namespaceSelector:
                matchLabels:
                  capsule.clastix.io/tenant: ${TENANT_A}
      validate:
        message: InfisicalOrganization must use the organization assigned to this Capsule tenant
        pattern:
          spec:
            organizationID: ${tenant_a_organization_id}
    - name: projects-must-use-capsule-tenant-a-organization
      match:
        any:
          - resources:
              kinds: [InfisicalProject]
              namespaceSelector:
                matchLabels:
                  capsule.clastix.io/tenant: ${TENANT_A}
      validate:
        message: InfisicalProject must use the organization assigned to this Capsule tenant
        pattern:
          spec:
            organizationRef:
              name: tenant-boundary
    - name: references-must-stay-within-capsule-tenant-a
      match:
        any:
          - resources:
              kinds:
                - InfisicalEnvironment
                - InfisicalIdentity
                - InfisicalKubernetesAuth
              namespaceSelector:
                matchLabels:
                  capsule.clastix.io/tenant: ${TENANT_A}
      validate:
        message: Infisical references must point to the project assigned to this Capsule tenant
        pattern:
          spec:
            projectRef:
              name: e2e-capsule-project-a
    - name: connection-must-use-platform-secret
      match:
        any:
          - resources:
              kinds: [InfisicalConnection]
              namespaceSelector:
                matchLabels:
                  capsule.clastix.io/tenant: ${TENANT_A}
      validate:
        message: InfisicalConnection must use the namespace-local platform credential
        pattern:
          spec:
            authSecretRef:
              name: infisical-token
---
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: e2e-capsule-project-b
spec:
  admission: true
  background: false
  validationFailureAction: Enforce
  rules:
    - name: organization-must-belong-to-capsule-tenant-b
      match:
        any:
          - resources:
              kinds: [InfisicalOrganization]
              namespaceSelector:
                matchLabels:
                  capsule.clastix.io/tenant: ${TENANT_B}
      validate:
        message: InfisicalOrganization must use the organization assigned to this Capsule tenant
        pattern:
          spec:
            organizationID: ${tenant_b_organization_id}
    - name: projects-must-use-capsule-tenant-b-organization
      match:
        any:
          - resources:
              kinds: [InfisicalProject]
              namespaceSelector:
                matchLabels:
                  capsule.clastix.io/tenant: ${TENANT_B}
      validate:
        message: InfisicalProject must use the organization assigned to this Capsule tenant
        pattern:
          spec:
            organizationRef:
              name: tenant-boundary
    - name: references-must-stay-within-capsule-tenant-b
      match:
        any:
          - resources:
              kinds:
                - InfisicalEnvironment
                - InfisicalIdentity
                - InfisicalKubernetesAuth
              namespaceSelector:
                matchLabels:
                  capsule.clastix.io/tenant: ${TENANT_B}
      validate:
        message: Infisical references must point to the project assigned to this Capsule tenant
        pattern:
          spec:
            projectRef:
              name: e2e-capsule-project-b
    - name: connection-must-use-platform-secret
      match:
        any:
          - resources:
              kinds: [InfisicalConnection]
              namespaceSelector:
                matchLabels:
                  capsule.clastix.io/tenant: ${TENANT_B}
      validate:
        message: InfisicalConnection must use the namespace-local platform credential
        pattern:
          spec:
            authSecretRef:
              name: infisical-token
EOF

policy_a=""
policy_b=""
for _ in {1..60}; do
	policy_a="$(host_kubectl get clusterpolicy/e2e-capsule-project-a -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || true)"
	policy_b="$(host_kubectl get clusterpolicy/e2e-capsule-project-b -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || true)"
	if [[ "${policy_a}" == "True" && "${policy_b}" == "True" ]]; then
    break
  fi
  sleep 1
done
if [[ "${policy_a}" != "True" || "${policy_b}" != "True" ]]; then
  echo "Kyverno policies did not become ready" >&2
  host_kubectl get clusterpolicy/e2e-capsule-project-a clusterpolicy/e2e-capsule-project-b -o yaml >&2 || true
  exit 1
fi

if [[ "$(as_tenant "${NAMESPACE_A}" "${OWNER_A}" auth can-i create infisicalprojects -n "${NAMESPACE_A}")" != "yes" ]]; then
  echo "Capsule did not grant tenant A its declared Infisical CR permissions" >&2
  exit 1
fi
if [[ "$(as_tenant "${NAMESPACE_A}" "${OWNER_A}" auth can-i create infisicalprojects -n "${NAMESPACE_B}")" == "yes" ]]; then
  echo "Capsule unexpectedly granted tenant A access to tenant B" >&2
  exit 1
fi

cat <<EOF | as_tenant "${NAMESPACE_A}" "${OWNER_A}" -n "${NAMESPACE_A}" apply -f - >/dev/null
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalConnection
metadata:
  name: infisical
spec:
  hostAPI: http://${INFISICAL_SERVICE}.${INFISICAL_NAMESPACE}.svc.cluster.local:8080/api
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
  organizationID: ${tenant_a_organization_id}
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
  projectName: ${TENANT_A_PROJECT_NAME}
  slug: ${TENANT_A_PROJECT_NAME}
  organizationRef:
    name: tenant-boundary
  creationPolicy: Create
  deletionPolicy: Delete
EOF

host_kubectl -n "${NAMESPACE_A}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalconnection/infisical --timeout=5m >/dev/null
cat <<EOF | as_tenant "${NAMESPACE_B}" "${OWNER_B}" -n "${NAMESPACE_B}" apply -f - >/dev/null
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalConnection
metadata:
  name: infisical
spec:
  hostAPI: http://${INFISICAL_SERVICE}.${INFISICAL_NAMESPACE}.svc.cluster.local:8080/api
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
  organizationID: ${tenant_b_organization_id}
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
  projectName: ${TENANT_B_PROJECT_NAME}
  slug: ${TENANT_B_PROJECT_NAME}
  creationPolicy: Create
  deletionPolicy: Delete
EOF
host_kubectl -n "${NAMESPACE_A}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalorganization/tenant-boundary --timeout=5m >/dev/null
host_kubectl -n "${NAMESPACE_A}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalproject/tenant-project --timeout=5m >/dev/null
host_kubectl -n "${NAMESPACE_B}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalconnection/infisical --timeout=5m >/dev/null
host_kubectl -n "${NAMESPACE_B}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalorganization/tenant-boundary --timeout=5m >/dev/null
host_kubectl -n "${NAMESPACE_B}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalproject/tenant-project --timeout=5m >/dev/null

disallowed_manifest="$(cat <<EOF
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalProject
metadata:
  name: forbidden-project
spec:
  connectionRef:
    name: infisical
  organizationRef:
    name: other-boundary
  projectName: forbidden-cross-tenant-project
  slug: forbidden-cross-tenant-project
  creationPolicy: Adopt
  deletionPolicy: Orphan
EOF
)"
if disallowed_output="$(as_tenant "${NAMESPACE_A}" "${OWNER_A}" -n "${NAMESPACE_A}" apply -f - <<<"${disallowed_manifest}" 2>&1)"; then
  echo "Kyverno unexpectedly accepted tenant A's reference to tenant B" >&2
  exit 1
fi
if ! grep -Eiq 'tenant|organization|denied|forbidden' <<<"${disallowed_output}"; then
  echo "Kyverno rejected the cross-tenant project without a useful admission message:" >&2
  echo "${disallowed_output}" >&2
  exit 1
fi
if host_kubectl -n "${NAMESPACE_A}" get infisicalproject/forbidden-project >/dev/null 2>&1; then
  echo "Kyverno denied the request but a forbidden InfisicalProject was persisted" >&2
  exit 1
fi

echo "Capsule scenario passed: tenant ownership, namespace-scoped RBAC, single-operator reconciliation, and Kyverno project-boundary admission"
echo "Tenant A project status ID: $(host_kubectl -n "${NAMESPACE_A}" get infisicalproject/tenant-project -o jsonpath='{.status.projectID}')"
echo "Tenant B project status ID: $(host_kubectl -n "${NAMESPACE_B}" get infisicalproject/tenant-project -o jsonpath='{.status.projectID}')"
echo "Kyverno admission rejection: ${disallowed_output}"
