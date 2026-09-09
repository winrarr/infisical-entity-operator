# Backlog

Items are ordered by current priority and should be refined against the current Infisical API before implementation. Only incomplete work belongs here: remove an item once its acceptance criteria are met. Preserve lasting rationale in an ADR or the relevant design/operations document, and record unresolved current shortcomings in the [tech-debt register](tech-debt.md).

## BL-002: Make project roles declarative for `InfisicalIdentity`

Status: planned.

### Goal

Allow a project-scoped machine identity to declare the permanent Infisical project roles it should have.

### Rationale

Infisical accepts roles during identity creation, but role membership has its own read, update, and delete API. This is part of making an identity actually usable by workloads and is currently an explicit product gap.

### Constraints

- Extend `InfisicalIdentity` first; do not introduce a separate membership CRD until identities shared across multiple projects are supported.
- Support permanent role assignments initially. Time-bound roles require a separate lifecycle design because relative expirations do not map cleanly to stable Kubernetes desired state.
- Reconcile role slugs, not opaque role IDs, while recording observed role details safely in status.
- Preserve safe orphaning by default when the Kubernetes resource is deleted.

### Acceptance criteria

- Identity creation, adoption, and reconciliation converge on the declared permanent role set.
- Manual role changes are detected and corrected according to the operator’s ownership policy.
- Role assignment and removal errors are visible through conditions and do not cause credential leakage in logs or status.
- The controller handles an identity membership that was removed or deleted remotely without wedging the identity resource.
- Tests cover creation, adoption, drift, dependency ordering, deletion policy, and role API error cases.

## BL-005: Publish GitHub Pages documentation

Status: planned.

### Goal

Publish the project documentation on GitHub Pages as one documentation deliverable, with the repository Markdown remaining the source of truth.

### Rationale

The project now has product, architecture, operations, research, and decision documentation that should be discoverable without browsing the repository directly.

### Constraints

- Keep documentation and the publishing workflow in this repository.
- Do not place credentials, generated Infisical tokens, kubeconfigs, or private environment data in the published site.
- Preserve direct links to the README, API examples, Helm installation, local Kind workflow, network-policy behavior, and security boundaries.
- Choose a maintained static documentation tool only if it materially improves navigation and generated API reference; avoid adding unnecessary runtime infrastructure.

### Acceptance criteria

- A GitHub Actions workflow builds and deploys the documentation to the repository’s GitHub Pages site.
- The published site has navigable sections for getting started, installation, CRDs, architecture, operations, security, development, and troubleshooting.
- The README links to the site, and the site links back to the repository and relevant source documents.
- Documentation publication is tested on documentation changes and fails clearly when links or the site build are invalid.
- The site contains no credentials or environment-specific local-cluster state.

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
