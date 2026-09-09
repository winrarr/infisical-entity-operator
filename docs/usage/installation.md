# Installation

The Helm chart is the primary installation path. It installs the operator Deployment, RBAC, ServiceAccount, and all six CRDs.

## Install from a checkout

```sh
helm upgrade --install infisical-entity-operator \
  ./charts/infisical-entity-operator \
  --namespace infisical-entity-operator-system \
  --create-namespace \
  --wait
```

Build and publish an image when running a local checkout:

```sh
make docker-build IMG=ghcr.io/winrarr/infisical-entity-operator:dev
make deploy IMG=ghcr.io/winrarr/infisical-entity-operator:dev
```

The chart defaults to a non-root, read-only operator container with dropped Linux capabilities. Metrics are disabled by default. Set chart values deliberately for a production deployment; inspect the available values with:

```sh
helm show values ./charts/infisical-entity-operator
```

## Upgrade and uninstall

Upgrade with the same release name and namespace:

```sh
helm upgrade infisical-entity-operator \
  ./charts/infisical-entity-operator \
  --namespace infisical-entity-operator-system \
  --wait
```

Uninstalling the chart removes the operator and leaves custom resources and CRDs for an explicit cleanup decision. Resources default to `deletionPolicy: Orphan`; use `Delete` only when remote deletion is intentional. See [lifecycle and ownership](reference/deletion-and-ownership.md).

## Kustomize bundle

The repository also supports a generated standalone bundle:

```sh
make build-installer
kubectl apply -f dist/install.yaml
```

The Helm chart remains the tested deployment path, while Kustomize is useful for environments that require a single rendered manifest.
