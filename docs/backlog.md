# Backlog

Items are ordered by current priority. Each item should be refined against the current Infisical API before implementation.

## BL-001: Add `InfisicalEnvironment`

Status: implemented.

### Goal

Provide a namespaced Kubernetes resource for declaring the lifecycle of an Infisical environment owned by an `InfisicalProject`.

### Rationale

Projects currently expose environments only as observed status. A first-class resource is needed for custom environments and for reliable adoption, drift correction, and dependency-aware workload configuration.

### Constraints

- Keep project and connection references same-namespace.
- Treat the environment slug as the stable external lookup key and make it immutable after creation.
- Preserve Infisical soft-delete and restore semantics instead of treating every missing environment as an ordinary create failure.
- Default to safe orphaning; external deletion must be explicit.
- Do not add a second environment-desired-state field to `InfisicalProject`.

### Acceptance criteria

- The resource can create, adopt, observe, update, and optionally delete an environment.
- Status records the external ID, project ID, name, slug, position, and useful conditions without exposing credentials.
- Missing projects and connections produce dependency conditions and trigger reconciliation when dependencies change.
- Remote drift in mutable fields is corrected and remote deletion is handled deterministically.
- CRDs, RBAC, Helm manifests, samples, unit tests, and live Kind coverage are generated or updated as appropriate.

## BL-002: Make project roles declarative for `InfisicalIdentity`

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

## BL-003: Add `InfisicalKubernetesAuth`

Status: implemented. The local e2e test exercises the controller’s external error handling when the self-hosted chart rejects cluster-local review URLs; the successful attach and login path is covered by HTTP contract tests and runs when a suitable Infisical endpoint is available.

### Goal

Configure and reconcile Infisical Kubernetes Auth for a managed machine identity so Kubernetes workloads can authenticate without a long-lived client secret.

### Rationale

Kubernetes Auth is the most Kubernetes-native authentication method in Infisical and completes the path from an identity resource to workload authentication.

### Constraints

- Reference an identity in the same namespace and keep the identity dependency explicit.
- Require explicit namespace and service-account allowlists; do not inherit Infisical’s broad wildcard defaults silently.
- Accept token-review credentials only through a Kubernetes Secret reference and never copy them into status or logs.
- Keep authentication configuration separate from secret synchronization, which remains outside this operator.
- Make deletion of the remote auth method opt-in and safe by default.

### Acceptance criteria

- The resource can attach, adopt, observe, update, and optionally remove Kubernetes Auth configuration.
- Status reports safe configuration observations and conditions without token or certificate disclosure.
- A Kind-based test proves an allowed service account can authenticate and a disallowed one cannot.
- Tests cover API-server certificate handling, token-review modes, dependency ordering, drift, and deletion policy.
- Network-policy coverage verifies only the required Kubernetes and Infisical paths are available.

## BL-004: Add `InfisicalProjectRole`

Status: implemented. The local standalone chart currently rejects custom roles on its plan, so the live test records that expected limitation while the controller path is covered by HTTP contract tests.

### Goal

Manage custom Infisical project roles and their permission rules declaratively.

### Rationale

Identity role assignments are only reproducible when the referenced custom roles can also be managed as code.

### Constraints

- Model the permission subject/action/condition combinations as a deliberately typed API rather than accepting unvalidated arbitrary JSON.
- Make role slugs immutable after creation because identities reference them.
- Detect and report roles that cannot be deleted because they are still in use.
- Do not expand into organization-level users, groups, or identity-provider administration.

### Acceptance criteria

- The resource can create, adopt, observe, update, and optionally delete a project role.
- Permission changes converge deterministically and are visible in status conditions.
- Identity role references can wait for a role resource and recover from role drift or deletion.
- Validation and tests cover supported permission subjects, actions, conditions, and malformed rules.
- The generated CRD and Helm chart remain usable without requiring secret-management resources from this operator.

## BL-005: Publish GitHub Pages documentation

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
