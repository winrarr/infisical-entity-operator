# Quickstart

This path assumes a Kubernetes cluster, Helm, and an Infisical bearer token with permission to manage the selected organization and project resources.

Create a namespace and keep the token in a Kubernetes Secret. The Secret and every referenced custom resource must be in the same namespace.

```sh
kubectl create namespace infisical-demo
kubectl -n infisical-demo create secret generic infisical-token \
  --from-literal=token="$INFISICAL_TOKEN"
```

Install the operator from a repository checkout:

```sh
helm upgrade --install infisical-entity-operator \
  ./charts/infisical-entity-operator \
  --namespace infisical-entity-operator-system \
  --create-namespace \
  --wait
```

Apply a connection and a project. The project is created only after the connection can authenticate to Infisical.

```yaml
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalConnection
metadata:
  name: infisical
  namespace: infisical-demo
spec:
  hostAPI: https://app.infisical.com/api
  authSecretRef:
    name: infisical-token
    key: token
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalProject
metadata:
  name: payments
  namespace: infisical-demo
spec:
  connectionRef:
    name: infisical
  projectName: payments
  slug: payments-platform
  deletionPolicy: Orphan
```

```sh
kubectl apply -f connection-and-project.yaml
kubectl -n infisical-demo get infisicalconnection,infisicalproject
kubectl -n infisical-demo describe infisicalproject payments
```

Then add environments, roles, identities, and Kubernetes Auth from the [examples](examples/index.md) or the [CRD guides](crds/index.md). Read [installation](introduction/installation.md) for release and upgrade details and [troubleshooting](reference/troubleshooting.md) if a resource is not ready.
