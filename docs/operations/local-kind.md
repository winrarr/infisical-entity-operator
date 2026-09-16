# Local Kind operations

The local environment is disposable and isolated. The fast `kind-e2e` path uses Kind's default CNI, the official Infisical standalone chart with in-cluster PostgreSQL and Redis, and the operator chart built from the checkout. Cilium remains available by overriding the setup with `KIND_CNI=cilium`.

Kind Make targets verify that the current kubectl context is `kind-infisical-entity-operator` (or the value of `KIND_CLUSTER`) before installing anything. Switch contexts explicitly when another local cluster is active.

## Start and test

```sh
make kind-e2e
```

The command:

- creates the `infisical-entity-operator` Kind cluster;
- keeps Kind's default CNI enabled;
- installs Infisical `v0.165.8` from standalone chart `1.10.0`;
- generates a random bootstrap password for this cluster invocation;
- waits for the chart bootstrap job and reads its generated token only inside the cluster;
- builds and loads the operator image while the independent cluster services start;
- installs the CRDs and operator chart;
- applies the standard `config/network-policy/allow-infisical-egress-network-policy.yaml`;
- creates a connection, project, project-scoped identity, and environment and waits for them to become `Ready=True`;
- leaves Kubernetes Auth out of the default smoke test because the standalone chart may reject the cluster-local review URL;
- creates and verifies Kubernetes Auth only when `KUBERNETES_AUTH_REVIEW_URL` is supplied by the opt-in target.

The default smoke test omits Kubernetes Auth so it is independent of the standalone chart's cluster-local URL limitation. To run the complete Kubernetes Auth acceptance, expose the disposable Kind API server through a public HTTPS URL that Infisical can reach, then use the opt-in target:

```sh
api_server_url="$(kubectl config view --raw --minify -o jsonpath='{.clusters[0].cluster.server}')"
cloudflared tunnel --url "${api_server_url}" --no-tls-verify --no-autoupdate

KUBERNETES_AUTH_REVIEW_URL=https://<generated>.trycloudflare.com \\
KUBERNETES_AUTH_VERIFY_TLS=false \\
make kind-kubernetes-auth-e2e
```

`cloudflared` is only needed for this local acceptance path and is not a project dependency. The Quick Tunnel terminates TLS before forwarding to the disposable Kind API server, so the example disables certificate verification for that tunnel only. A production endpoint should use a stable HTTPS name and `KUBERNETES_AUTH_VERIFY_TLS=true` with the Kubernetes CA configured as usual.

The opt-in target fails on any Kubernetes Auth reconciliation error; it does not accept the standalone chart's local-IP validation message. It then logs in with the allowed service account, verifies that the disallowed service account is rejected, and removes the test namespace. The reviewer JWT, CA data, and all credentials remain in cluster Secrets or process memory and must never be copied into the checkout.

The default Kind CNI makes this a fast smoke test. It does not provide evidence that NetworkPolicy rules are enforced; that depends on the installed CNI. Use `make kind-up KIND_CNI=cilium` when a local scenario needs Cilium. The default and Cilium modes use the same named cluster, so run `make kind-down` before switching between them.

The E2E deployment consumes committed CRD and chart artifacts, so it does not run documentation generation. Its independent image, cluster, and operator preparation tasks run in parallel; adjust the worker count with `KIND_PARALLEL_JOBS` when needed.

The generated instance-admin token is a cluster Secret named `infisical-bootstrap-token` in namespace `infisical`. It is intentionally not written to the checkout or printed by the test.

The multi-tenancy targets use the bootstrap user credentials stored in `infisical-bootstrap-credentials` to create temporary top-level organizations. The vCluster target gives its in-cluster operator an organization-scoped machine identity with `organizationRole: admin`, so the tenant creates a project in that organization. The Capsule target creates one such identity per Capsule tenant and uses Kyverno to require each tenant’s `InfisicalProject.spec.organizationRef` to point at its own adopted organization.

## Inspect a failed run

```sh
kubectl get pods -A
kubectl get networkpolicy -n infisical-entity-operator-system
kubectl describe infisicalproject e2e-project -n infisical-entity-operator-e2e
kubectl describe infisicalkubernetesauth e2e-kubernetes-auth -n infisical-entity-operator-e2e
kubectl logs deployment/infisical-entity-operator-infisical-entity-operator -n infisical-entity-operator-system
kubectl logs job/infisical-bootstrap-1 -n infisical
```

The standard policy describes egress to the Kubernetes API Service CIDR, DNS, and the labeled Infisical service. Its enforcement depends on the installed CNI. If a Cilium-backed scenario fails after policy changes, inspect the Cilium policy and agent status before loosening the rule.

## Cleanup

Delete only the isolated cluster when finished:

```sh
make kind-down
```

This removes the local Infisical data, generated bootstrap Secret, and operator deployment together. It does not touch any other Kubernetes context or cluster.
