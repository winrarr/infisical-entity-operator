# Infisical Entity Operator

Kubernetes-native lifecycle management for Infisical projects, environments, project roles, machine identities, and Kubernetes Auth.

The operator gives platform teams a declarative boundary around the Infisical control plane:

```text
InfisicalConnection → InfisicalProject → InfisicalEnvironment
                                      ├→ InfisicalProjectRole
                                      └→ InfisicalIdentity → InfisicalKubernetesAuth
```

It handles create-or-adopt workflows, drift correction, dependency-aware status conditions, finalizers, and explicit deletion policy. Secret synchronization remains outside this project; use Infisical’s official Kubernetes operator for `InfisicalSecret`-style workloads.

## Quick start

Create a Secret containing an Infisical bearer token and apply a connection, project, and identity in the same namespace:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: infisical-token
type: Opaque
stringData:
  token: ${INFISICAL_TOKEN}
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalConnection
metadata:
  name: infisical
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
spec:
  connectionRef:
    name: infisical
  projectName: payments
  slug: payments-platform
  deletionPolicy: Orphan
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalIdentity
metadata:
  name: payments-workload
spec:
  connectionRef:
    name: infisical
  projectRef:
    name: payments
  identityName: payments-workload
```

An environment, project role, or Kubernetes Auth resource can reference the project or identity in the same namespace and will wait for that dependency to become ready.

Set `spec.roleSlugs` on an identity to manage its permanent Infisical project roles. Omit the field to leave an existing membership unmanaged; use the built-in `no-access` role explicitly when an identity should have no project permissions.

Install the chart and CRDs from a repository checkout:

```sh
helm upgrade --install infisical-entity-operator \
  ./charts/infisical-entity-operator \
  --namespace infisical-entity-operator-system --create-namespace
```

For a checkout of this repository, use `make deploy IMG=ghcr.io/winrarr/infisical-entity-operator:<tag>` or build a bundle with `make build-installer`.

## Local development

The complete local path is intentionally reproducible:

```sh
make kind-e2e
```

This creates an isolated Kind cluster with Cilium, installs the official Infisical standalone Helm chart, bootstraps a short-lived local instance-admin token, deploys the operator, applies a Cilium-aware egress policy, and verifies connection, project, identity, and environment reconciliation. It also exercises project-role and Kubernetes Auth reconciliation; the local Infisical chart may report those features as unavailable when its plan rejects custom roles or cluster-local review URLs. See [local Kind operations](docs/operations/local-kind.md) for cleanup and troubleshooting.

## API and safety notes

- `creationPolicy` defaults to `Create`; use `Adopt` or `CreateOrAdopt` to manage existing entities.
- `deletionPolicy` defaults to `Orphan`. `Delete` is opt-in and invokes irreversible Infisical deletion.
- Connection and ownership references are immutable after creation so an external object cannot silently move between endpoints or projects.
- The operator stores external identifiers and observed state in Kubernetes status, never external bearer tokens.

See the [documentation map](docs/index.md), [product scope](docs/product.md), [architecture](docs/architecture.md), [verification guide](docs/verification.md), [backlog](docs/backlog.md), [tech-debt register](docs/tech-debt.md), [research](docs/research/2026-09-09-infisical-api-and-versions.md), and [decisions](docs/decisions/index.md).

The [published documentation site](https://winrarr.github.io/infisical-entity-operator/) provides the navigable reference and quickstart.

## Contributing

Run `make check` before submitting a change. API marker changes require `make manifests generate`; generated output is checked into the repository so installation and review are self-contained.

## License

Apache License 2.0. See [LICENSE](LICENSE).
