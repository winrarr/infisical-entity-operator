# Backlog

Items are ordered by current priority and should be refined against the current Infisical API before implementation. Only incomplete work belongs here: remove an item once its acceptance criteria are met. Preserve lasting rationale in an ADR or the relevant design/operations document, and record unresolved current shortcomings in the [tech-debt register](tech-debt.md).

## BL-007: Add declarative identity templates and authentication methods

### Goal

Manage the Infisical identity-template lifecycle and selected identity authentication methods as Kubernetes resources, starting with the Kubernetes, OIDC, and Universal Auth paths needed by the existing identity and Kubernetes Auth resources.

### Rationale

The current OpenAPI surface includes `/api/v1/identity-templates` and Universal Auth client-secret endpoints. `InfisicalKubernetesAuth.spec.templateID` can reference an Infisical Kubernetes Auth template, but the operator does not currently manage that template. `InfisicalConnection` consumes a bearer token and does not perform Universal Auth login or credential rotation.

### Constraints

- Keep authentication credentials out of status, logs, and generated manifests.
- Separate reusable identity-template configuration from per-identity auth-method state.
- Define ownership, rotation, and Secret output semantics before implementing client-secret automation.
- Preserve the current bearer-token `InfisicalConnection` contract until an auth-method design is accepted.

### Acceptance criteria

- Add a declarative identity-template resource with the live API’s supported template variants and lifecycle policies.
- Allow `InfisicalKubernetesAuth` to reference a managed Kubernetes identity template without requiring a manually copied remote UUID.
- Define and implement, or explicitly defer, Universal Auth configuration, client-secret rotation, token exchange, and Secret publication with contract tests.
- Document plan-gated features and verify the supported paths against the live OpenAPI contract and a compatible Infisical deployment.

The accepted tenancy design is recorded in [0007: Make Infisical organizations explicit tenant boundaries](decisions/0007-explicit-organization-tenant-boundaries.md); unresolved limitations remain in the [tech-debt register](tech-debt.md).
