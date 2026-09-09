# Backlog

Items are ordered by current priority and should be refined against the current Infisical API before implementation. Only incomplete work belongs here: remove an item once its acceptance criteria are met. Preserve lasting rationale in an ADR or the relevant design/operations document, and record unresolved current shortcomings in the [tech-debt register](tech-debt.md).

## BL-006: Evaluate and harden multi-tenancy

Status: evaluation pending.

### Goal

Define the supported tenancy model and determine whether the operator needs additional isolation controls before claiming multi-tenant support.

### Rationale

The current resources are namespaced, references and bearer-token Secrets are same-namespace, and cross-namespace sharing is intentionally unsupported. However, the default manager deployment uses cluster-wide watches and a ClusterRole that can read Secrets across namespaces, so this is a useful foundation—not a complete multi-tenancy design.

### Constraints

- Treat Kubernetes namespace isolation, Infisical organization/project boundaries, and operator deployment boundaries as separate concerns.
- Do not claim tenant isolation until RBAC, caches, watches, credentials, logs, events, status, and network policies have been evaluated together.
- Preserve a clear single-tenant installation path while considering namespace-scoped installations and a cluster-wide controller.
- Do not add cross-namespace references or shared credentials as a shortcut.

### Acceptance criteria

- The supported tenancy model and threat assumptions are documented.
- The design compares a cluster-wide controller with independently namespace-scoped controller deployments.
- RBAC is audited for CRDs, Secrets, status, finalizers, leader election, and metrics access.
- Tests demonstrate that a tenant cannot cause reconciliation with another tenant’s connection, observe its credentials, or mutate its resources through this operator.
- Infisical organization/project boundaries, API credentials, rate limits, Cilium policies, logs, events, and status data are included in the evaluation.
- The result records required implementation changes or an explicit decision to defer multi-tenant support.
