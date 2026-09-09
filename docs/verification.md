# Verification and evidence

This project has both local controller tests and a disposable live environment. A passing check proves only the scope described below; it does not turn a local limitation into a supported Infisical capability.

## Canonical commands

| Command | What it proves |
| --- | --- |
| `make format-check` | Go sources are formatted. |
| `make verify-generated` | Regenerated deepcopy code, CRDs, RBAC, and Helm CRD copies match the committed derived output. Run it from a clean checkout. |
| `make test` | Generation is current enough to compile, static vetting passes, and unit/HTTP contract tests pass. |
| `make lint-config lint` | The linter configuration and Go implementation pass golangci-lint. |
| `make helm-lint` | The operator Helm chart is structurally valid. |
| `make check` | The complete local static suite: generation, formatting, vet, tests, lint, and Helm lint. |
| `make build` | The controller binary can be built from the current source and generated artifacts. |
| `make kind-e2e` | The disposable Kind cluster can run Cilium, Infisical, the chart, the operator, the egress policy, and the live reconciliation path. |
| `make kind-down` | Only the named disposable Kind cluster is removed. |

CI uses the same repository commands: the test workflow checks generated output and runs `make test`, the lint workflow runs `make lint-config lint helm-lint`, and the e2e workflow runs `make kind-e2e` followed by cleanup.

## Evidence boundaries

Unit and HTTP contract tests use fake Kubernetes clients and `httptest` servers. They prove request construction, response decoding, dependency handling, status transitions, drift correction, finalizers, and error classification without requiring a live Infisical account.

The Kind workflow adds deployment evidence: CRDs and chart installation, manager-to-API connectivity, the Cilium egress policy, and live project, environment, and identity reconciliation. The default standalone Infisical chart can reject custom project roles because of its plan and can reject the cluster-local Kubernetes review URL because of its URL policy. The script records those exact known conditions and continues; it does not claim successful live role creation or Kubernetes Auth login in that configuration. Successful role and authentication flows require an Infisical endpoint whose plan and network policy permit them.

Kubernetes Auth allow/deny login checks run only when the live API accepts the configured review endpoint. The token-review JWT and CA data used by the test are generated or stored inside the cluster and must never be copied into the checkout, logs, status, or documentation.

## Generated output and repository hygiene

API types and controller markers are source files. `api/infisical/v1alpha1/zz_generated.deepcopy.go`, `config/crd/bases/`, `config/rbac/role.yaml`, and `charts/infisical-entity-operator/crds/` plus the chart ClusterRole copy are derived. Run `make manifests generate` after marker or API changes, then use `make verify-generated` to detect drift.

Do not commit `bin/`, `dist/`, `tmp/`, coverage profiles, kubeconfig files, local cluster state, bootstrap credentials, or any Infisical token. A test that passes after leaking one of these is not acceptable evidence.
