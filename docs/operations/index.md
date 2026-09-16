# Operations

The repository includes two operational paths:

- [Local Kind](local-kind.md) creates an isolated Kind cluster with Infisical, the operator, and fast default-CNI reconciliation checks; Cilium is available for local scenarios that require it.
- [Live Acceptance](live-acceptance.md) runs the focused ProjectRole and Kubernetes Auth acceptance checks against a separately provisioned Infisical and Kubernetes environment.

For production installation and upgrades, see [installation](../introduction/installation.md). For condition interpretation and evidence limits, see the [verification guide](../verification.md).
