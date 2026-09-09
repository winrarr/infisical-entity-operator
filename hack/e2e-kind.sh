#!/usr/bin/env bash
set -euo pipefail

KUBECTL=${KUBECTL:-kubectl}
OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE:-infisical-entity-operator-system}
INFISICAL_NAMESPACE=${INFISICAL_NAMESPACE:-infisical}
TEST_NAMESPACE=${TEST_NAMESPACE:-infisical-entity-operator-e2e}

cleanup() {
	if [[ -n "${token:-}" ]]; then
	  "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalidentity/e2e-identity --ignore-not-found --wait=true >/dev/null 2>&1 || true
	  "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalproject/e2e-project --ignore-not-found --wait=true >/dev/null 2>&1 || true
	  "${KUBECTL}" -n "${TEST_NAMESPACE}" delete infisicalconnection/infisical --ignore-not-found --wait=true >/dev/null 2>&1 || true
fi
  "${KUBECTL}" delete namespace "${TEST_NAMESPACE}" --ignore-not-found --wait=false >/dev/null 2>&1 || true
}
trap cleanup EXIT

"${KUBECTL}" create namespace "${TEST_NAMESPACE}" --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null

token="$(${KUBECTL} -n "${INFISICAL_NAMESPACE}" get secret infisical-bootstrap-token -o jsonpath='{.data.token}' | base64 --decode)"
if [[ -z "${token}" ]]; then
  echo "Infisical bootstrap secret contained an empty token" >&2
  exit 1
fi

"${KUBECTL}" -n "${TEST_NAMESPACE}" create secret generic infisical-token \
  --from-literal=token="${token}" --dry-run=client -o yaml | "${KUBECTL}" apply -f - >/dev/null

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
EOF

"${KUBECTL}" -n "${TEST_NAMESPACE}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalconnection/infisical --timeout=5m
"${KUBECTL}" -n "${TEST_NAMESPACE}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalproject/e2e-project --timeout=5m
"${KUBECTL}" -n "${TEST_NAMESPACE}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  infisicalidentity/e2e-identity --timeout=5m

project_id="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalproject e2e-project -o jsonpath='{.status.projectID}')"
identity_id="$(${KUBECTL} -n "${TEST_NAMESPACE}" get infisicalidentity e2e-identity -o jsonpath='{.status.identityID}')"
[[ -n "${project_id}" && -n "${identity_id}" ]]

echo "Live reconciliation succeeded through the Cilium egress policy"
echo "Project status ID: ${project_id}"
echo "Identity status ID: ${identity_id}"
