# Project guide

## Orientation

This is a Go 1.27 Kubernetes operator. The API definitions in `api/infisical/v1alpha1` are the source of truth for the three namespaced CRDs:

- `InfisicalConnection` validates an Infisical API endpoint and reads a same-namespace bearer-token Secret.
- `InfisicalProject` creates, adopts, updates, observes, and optionally deletes an Infisical project.
- `InfisicalIdentity` creates, adopts, updates, observes, and optionally deletes a project-scoped machine identity.

Reconciliation lives in `internal/controller/infisical`; the intentionally small HTTP client lives in `internal/infisicalclient`. The Helm chart under `charts/infisical-entity-operator` is the primary installation path. Kustomize manifests under `config/` remain useful for CRD installation and bundle generation.

## Boundaries

- Edit API and controller source, samples, chart templates, and documentation directly.
- Do not edit `api/**/zz_generated.deepcopy.go`, `config/crd/bases/`, `config/rbac/role.yaml`, or the chart copies under `charts/infisical-entity-operator/crds/`; regenerate them with `make manifests generate`.
- `PROJECT` records Kubebuilder metadata and should only change when the project layout or API inventory changes.
- Never commit tokens, local bootstrap credentials, kubeconfig files, `bin/`, `dist/`, `tmp/`, or coverage output.

## Commands

```sh
make check             # format, generate, manifests, tests, vet, lint, Helm lint
make build             # build bin/manager
make manifests generate
make build-installer   # write dist/install.yaml
make kind-e2e          # Kind + Cilium + Infisical + live reconciliation/policy test
make kind-down         # delete only the named local Kind cluster
```

The default local environment is the isolated cluster named `infisical-entity-operator`. Override `KIND_CLUSTER`, `IMG`, `INFISICAL_IMAGE_TAG`, or other Make variables when needed. `make kind-up` creates ephemeral local Infisical bootstrap credentials and stores the resulting instance-admin token only in the cluster.

## Implementation expectations

Keep reconciliation idempotent and status-driven. Use finalizers only for resources with `deletionPolicy: Delete`; the default is safe orphaning. Treat connection, project, and Secret references as same-namespace dependencies. Preserve external API error details in conditions without logging bearer tokens. Add focused unit tests for client contracts and reconciliation transitions, then run `make check`.

Read `docs/architecture.md`, `docs/operations/local-kind.md`, and the relevant decision records before changing external API behavior or local cluster networking.
