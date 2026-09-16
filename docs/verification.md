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
| `make generate-api-reference` | The generated CRD field reference reflects the Go API definitions. |
| `make docs-build` | The generated API reference and Zensical site build succeed with strict link validation. |
| `make check` | The complete local static suite: generation, formatting, vet, tests, lint, Helm lint, and documentation build. |
| `make validate-release RELEASE_TAG=v0.1.0` | The release tag is a supported semantic version and matches the chart and image metadata. |
| `make build` | The controller binary can be built from the current source and generated artifacts. |
| `make kind-e2e` | The disposable Kind cluster can run Kind's default CNI, Infisical, the chart, the operator, the standard egress policy manifest, and one representative live reconciliation path. The setup overlaps independent image, cluster, and operator preparation and uses committed artifacts without documentation generation. |
| `make kind-kubernetes-auth-e2e KUBERNETES_AUTH_REVIEW_URL=...` | Runs the full Kubernetes Auth acceptance with a reachable review endpoint, including an allowed and a rejected service-account login. This is an opt-in local tunnel path because the standalone Infisical chart may reject the cluster-local URL. |
| `make kind-vcluster-e2e` | A core operator creates an Infisical organization and machine identity, then an operator inside a vCluster adopts that organization and creates a project with the tenant credential. |
| `make kind-capsule-e2e` | One cluster-wide operator reconciles two Capsule tenants with separate organization credentials while Kyverno rejects a cross-tenant organization reference. |
| `make kind-multitenancy-e2e` | Runs both tenant-boundary scenarios in sequence. |
| `make kind-down` | Only the named disposable Kind cluster is removed. |

CI uses the same repository commands: the test workflow checks generated output and runs `make test`, the lint workflow runs `make lint-config lint helm-lint`, the docs workflow runs `make docs-build` and deploys the result from `main`, the e2e workflow runs `make kind-e2e` followed by cleanup, and the release workflow validates release metadata and generated artifacts before publishing.

## Test boundaries

The Go tests are the primary coverage for controller lifecycle behavior, including adoption, drift correction, deletion, credential rotation, dependency handling, and recovery. The default Kind workflow is intentionally a smoke test for the packaged deployment and one representative live reconciliation. Kubernetes Auth, vCluster, Capsule, Kyverno, and Cilium scenarios remain opt-in because they validate external integration boundaries rather than every controller branch.

## kstatus compatibility

All seven CRDs expose Kubernetes `Ready`, `Reconciling`, and `Stalled` conditions with `status.observedGeneration`. Successful reconciles report `Ready=True`; dependency and retryable external failures report `Reconciling=True`; invalid configuration, ownership mismatches, and forbidden creates report `Stalled=True`. This lets generic kstatus consumers classify resources without controller-specific condition conventions.

See the [kstatus condition conventions](https://github.com/kubernetes-sigs/cli-utils/blob/master/pkg/kstatus/README.md) for the external interpretation.

## Evidence boundaries

Unit and HTTP contract tests use fake Kubernetes clients and `httptest` servers. They prove request construction, response decoding, dependency handling, status transitions, drift correction, finalizers, and error classification without requiring a live Infisical account.

The default-CNI Kind workflow adds deployment evidence: CRDs and chart installation, manager-to-API connectivity, application of the standard egress policy manifest, and live free-tier project, environment, and project-scoped identity reconciliation. It intentionally remains a smoke test and does not prove NetworkPolicy enforcement because that depends on the CNI. The opt-in Kubernetes Auth workflow additionally verifies that a reachable review endpoint accepts the allowed service account and rejects the disallowed one. The vCluster workflow additionally verifies organization creation, Universal Auth issuance, tenant-side organization adoption, and tenant-created projects. The Capsule workflow verifies separate tenant credentials, shared-manager reconciliation, and Kyverno admission of the explicit organization reference.

The default local workflow does not configure Kubernetes Auth; the opt-in workflow fails on an unavailable review endpoint instead. The token-review JWT and CA data used by the opt-in test are generated or stored inside the cluster and must never be copied into the checkout, logs, status, or documentation.

## Generated output and repository hygiene

API types and controller markers are source files. `api/infisical/v1alpha1/zz_generated.deepcopy.go`, `config/crd/bases/`, `config/rbac/role.yaml`, `charts/infisical-entity-operator/crds/`, the chart ClusterRole copy, and `docs/reference/api.md` are derived. Run `make manifests generate` after marker or API changes, then use `make verify-generated` to detect drift.

Do not commit `bin/`, `dist/`, `tmp/`, coverage profiles, kubeconfig files, local cluster state, bootstrap credentials, or any Infisical token. A test that passes after leaking one of these is not acceptable evidence.
