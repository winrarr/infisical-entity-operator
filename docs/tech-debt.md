# Tech-debt register

This register contains material shortcomings that are known in the current implementation and intentionally remain unresolved. It is not a list of style preferences or hypothetical enhancements. Planned product outcomes belong in the [backlog](backlog.md); entries here should have concrete evidence and exit criteria.

## TD-001: Complete live acceptance for project roles and Kubernetes Auth

Status: accepted limitation

### Evidence

The default Kind environment uses the standalone Infisical chart. Its local plan rejects custom project roles, and its API rejects the cluster-local Kubernetes token-review URL. The e2e script verifies those bounded error paths and the HTTP contract tests cover successful controller/client behavior, but the default local run cannot prove successful live role creation or Kubernetes Auth login.

### Impact

The most important end-to-end acceptance paths for two CRDs depend on an Infisical deployment with a suitable plan and reachable token-review endpoint. Regressions that occur only in the live role or authentication integration could therefore escape the default local e2e run.

### Constraints

- Keep the default test isolated, disposable, and free of credentials in the checkout.
- Preserve the current bounded error assertions for the local standalone chart.
- Do not weaken URL or network-policy validation merely to make the local test pass.

### Exit criteria

Add a separately provisioned live test target or CI environment that can create a project role and complete allowed/disallowed Kubernetes Auth login checks, then document its credentials and network boundary without storing secrets in the repository. Related planned work: [BL-003](backlog.md#bl-003-add-infisicalkubernetesauth) and [BL-004](backlog.md#bl-004-add-infisicalprojectrole).

## TD-002: Establish an Infisical API compatibility and drift policy

Status: open

### Evidence

The typed client is based on endpoint documentation and a checked-in research note. The available OpenAPI fragment did not describe the public project and identity endpoints used here, and there is no automated check against a versioned server schema or an Infisical release matrix.

### Impact

Infisical API response envelopes, permission shapes, or endpoint behavior can change without a deterministic signal that the client and controllers need review.

### Constraints

- Keep the supported client surface narrow and typed.
- Treat official endpoint documentation and tested server behavior as separate evidence sources.
- Do not broaden the client or pin an Infisical compatibility promise without an explicit decision.

### Exit criteria

Define the supported Infisical server/API versions, record the authoritative schema source for each supported endpoint, and add contract or compatibility checks that fail or clearly report incompatible changes. Revisit [0003: Use a narrow typed Infisical client](decisions/0003-small-typed-client.md) if a mature client becomes a better fit.

## TD-003: Tighten project-role permission validation

Status: open

### Evidence

`InfisicalProjectRole` has a structural typed permission model and minimum cardinality checks, but subjects, actions, condition operators, and condition combinations remain mostly free-form because the supported Infisical permission surface can evolve. Client decoding intentionally accepts both string and array action responses.

### Impact

Malformed or server-version-specific rules may pass Kubernetes admission and fail only during reconciliation. Users receive the error late, and a future API change could require a compatibility adjustment across the CRD, client, and status model.

### Constraints

- Preserve typed fields; do not replace the CRD with arbitrary JSON.
- Do not hard-code an incomplete enum set without evidence from the supported API versions.
- Keep the string/array response compatibility behavior until the support policy is settled.

### Exit criteria

Use the compatibility policy from TD-002 to define supported subjects, actions, operators, and valid combinations; add CEL validation and contract tests that reject malformed rules before external reconciliation.

## TD-004: Make status persistence conflict-safe without silent loss

Status: open

### Evidence

`persistStatus` currently ignores Kubernetes resource-version conflicts so a dependency update cannot fail an otherwise successful reconcile. The next reconcile is expected to observe the latest object, but the status written by the losing attempt is discarded.

### Impact

A transient conflict can leave status stale for longer than the normal reconciliation interval and provides no direct signal that the observation was not persisted.

### Constraints

- Preserve requeue behavior for dependency and external failures.
- Avoid broad retries that can duplicate external create or update operations.
- Keep status writes free of credentials and bounded in size.

### Exit criteria

Use a conflict-safe status patch or targeted retry against the latest resource version, add a focused test for a concurrent status/spec update, and show that external side effects remain idempotent.

## TD-005: Evaluate multi-tenant isolation before making a support claim

Status: deferred evaluation

### Evidence

Resources, ownership references, and credential references are namespaced and same-namespace by design, but the default manager watches cluster-wide and its ClusterRole can read Secrets across namespaces. Infisical organization/project boundaries and Cilium policy boundaries are not yet evaluated as a combined tenant model.

### Impact

The project can provide a useful single-tenant or trusted-cluster installation, but it must not claim tenant isolation. A deployment that shares one manager across mutually untrusted tenants could expose a wider blast radius than the resource API suggests.

### Constraints

- Do not add cross-namespace references or shared credentials as a shortcut.
- Evaluate Kubernetes RBAC, caches, watches, status, logs, events, metrics, network policy, and Infisical authorization together.
- Preserve the current single-tenant installation path while the evaluation is pending.

### Exit criteria

Complete [BL-006](backlog.md#bl-006-evaluate-and-harden-multi-tenancy) with a documented support model, threat assumptions, comparison of cluster-wide and namespace-scoped deployments, isolation tests, and either required implementation changes or an explicit deferral decision.
