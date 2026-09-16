# Local Kind operations

The local environment is disposable and isolated. The fast `kind-e2e` path uses Kind's default CNI, the official Infisical standalone chart with in-cluster PostgreSQL and Redis, and the operator chart built from the checkout. Cilium remains available by overriding the setup with `KIND_CNI=cilium`.

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
- creates connection, two projects, an explicit organization adoption resource, a project-scoped identity, an organization-scoped tenant identity with memberships in both projects, and environment resources and waits for them to become `Ready=True`;
- creates a Kubernetes Auth resource and verifies it when the local Infisical API accepts the configured review endpoint; the standalone chart may reject the cluster-local URL;
- runs the Kubernetes Auth allowed/disallowed service-account login checks when the local Infisical API accepts the configured review endpoint.

The default Kind CNI makes this a fast reconciliation test. It does not provide evidence that NetworkPolicy rules are enforced; that depends on the installed CNI. Use `make kind-up KIND_CNI=cilium` when a local scenario needs Cilium. The default and Cilium modes use the same named cluster, so run `make kind-down` before switching between them.

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
