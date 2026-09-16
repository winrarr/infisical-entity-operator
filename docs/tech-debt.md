# Tech-debt register

This register contains material shortcomings that are known in the current implementation and intentionally remain unresolved. It is not a list of style preferences or hypothetical enhancements. Planned product outcomes belong in the [backlog](backlog.md); entries here should have concrete evidence and exit criteria.

## TD-001: Complete live acceptance for project roles and Kubernetes Auth

Status: acceptance path added; provisioned run pending

### Evidence

The default Kind environment uses the standalone Infisical chart. Its local plan rejects custom project roles, and its API rejects the cluster-local Kubernetes token-review URL. The default e2e script verifies those bounded error paths and successfully exercises built-in `no-access` identity membership, while the HTTP contract tests cover successful custom-role-controller/client behavior. The repository now provides `make live-e2e` and a manually triggered Live Acceptance workflow for a separately provisioned Infisical environment with custom project-role capability and a Kubernetes API endpoint reachable from Infisical. That path has not yet been run because no provisioned acceptance credentials or GitHub Actions secrets exist.

### Impact

The most important end-to-end acceptance paths for two CRDs depend on an Infisical deployment with a suitable plan and reachable token-review endpoint. Regressions that occur only in the live role or authentication integration could therefore escape the default local e2e run.

### Constraints

- Keep the default test isolated, disposable, and free of credentials in the checkout.
- Preserve the current bounded error assertions for the local standalone chart.
- Do not weaken URL or network-policy validation merely to make the local test pass.

### Exit criteria

Run `make live-e2e` in the documented provisioned environment, or configure and dispatch the Live Acceptance workflow, and record a successful allowed/disallowed login result. Keep credentials, kubeconfig, CA data, and reviewer tokens outside the repository and document the environment’s network boundary.

## TD-005: Cluster-wide manager remains a trusted deployment

Status: accepted limitation

### Evidence

Resources, ownership references, and credential references are namespaced and same-namespace by design. The accepted tenant-boundary model uses one Infisical organization per tenant and organization-scoped machine identities with project memberships, but the default manager still watches cluster-wide and its ClusterRole can read Secrets across namespaces. That deployment is therefore suitable for a trusted platform team, not mutually untrusted tenant workloads.

### Impact

The project can provide a useful single-tenant or trusted-cluster installation, but it must not claim tenant isolation for the shared manager. A deployment that shares one manager across mutually untrusted tenants could expose a wider blast radius than the resource API suggests.

### Constraints

- Do not add cross-namespace references or shared credentials as a shortcut.
- Keep Kubernetes RBAC, caches, watches, status, logs, events, metrics, network policy, and Infisical authorization as separate controls.
- Preserve the current trusted-cluster installation path and document the per-tenant deployment boundary.

### Exit criteria

Add and verify a deployment mode with an independently isolated manager cache and Secret permission set, or retain the trusted-cluster limitation with an explicit product decision not to claim shared-cluster tenant isolation.
