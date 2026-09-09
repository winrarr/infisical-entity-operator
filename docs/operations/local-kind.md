# Local Kind operations

The local environment is disposable and isolated. It uses Kind with the default CNI disabled, Cilium for networking, the official Infisical standalone chart with in-cluster PostgreSQL and Redis, and the operator chart built from the checkout.

## Start and test

```sh
make kind-e2e
```

The command:

- creates the `infisical-entity-operator` Kind cluster;
- installs Cilium `1.20.1` and enables Hubble relay;
- installs Infisical `v0.165.8` from standalone chart `1.10.0`;
- generates a random bootstrap password for this cluster invocation;
- waits for the chart bootstrap job and reads its generated token only inside the cluster;
- builds and loads the operator image;
- installs the CRDs and operator chart;
- applies `config/network-policy/allow-infisical-egress.yaml`;
- creates connection, project, and identity resources and waits for all three `Ready=True`.

The generated instance-admin token is a cluster Secret named `infisical-bootstrap-token` in namespace `infisical`. It is intentionally not written to the checkout or printed by the test.

## Inspect a failed run

```sh
kubectl get pods -A
kubectl get ciliumnetworkpolicy -n infisical-entity-operator-system
kubectl describe infisicalproject e2e-project -n infisical-entity-operator-e2e
kubectl logs deployment/infisical-entity-operator-infisical-entity-operator -n infisical-entity-operator-system
kubectl logs job/infisical-bootstrap-1 -n infisical
```

The policy intentionally restricts the selected manager pods to Kubernetes API, DNS, and the labeled Infisical service. If reconciliation fails after the policy is applied, inspect the policy endpoint labels and the Cilium agent status before loosening the rule.

## Cleanup

Delete only the isolated cluster when finished:

```sh
make kind-down
```

This removes the local Infisical data, generated bootstrap Secret, and operator deployment together. It does not touch any other Kubernetes context or cluster.
