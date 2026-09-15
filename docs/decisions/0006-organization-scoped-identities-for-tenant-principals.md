# 0006: Use organization-scoped identities for tenant principals

Status: superseded by [0007: Make Infisical organizations explicit tenant boundaries](0007-explicit-organization-tenant-boundaries.md)

Date: 2026-09-14

## Decision

Extend `InfisicalIdentity` with an immutable `spec.scope` that defaults to `Project` and may be set to `Organization`.

For organization scope:

- `spec.organizationRef` references a ready `InfisicalProject` and supplies the observed Infisical organization ID.
- `spec.organizationRole` is optional and defaults in the reconciler to Infisical’s `no-access` role.
- `spec.projectRoleBindings` declares permanent roles for selected projects in that organization.
- The controller verifies every bound project reports the same organization as the anchor before calling Infisical.

The resource owns the identity and the project memberships listed in its bindings. It leaves memberships in unlisted projects unmanaged. Project scope retains the existing `spec.projectRef` and `spec.roleSlugs` contract.

## Rationale

Infisical documents project identities as limited to one project and organization identities as assignable to one or more projects. An organization-scoped identity with an organization-level `no-access` role and explicit project memberships maps cleanly to a platform-created tenant principal without inventing a provider-independent tenant object or adding vCluster-specific branches to the core controller.

This anchor model was intentionally replaced because it made the Infisical organization boundary implicit and did not support a tenant creating projects.

The operator does not try to be a universal Kubernetes tenancy policy engine. A platform can install one operator instance per tenant cluster and issue that instance a machine-identity credential whose Infisical project memberships are defined by the trusted platform layer. Kubernetes RBAC and the deployment boundary remain responsible for limiting who can create or read CRs.

## Constraints

- Keep all references same-namespace and immutable where they identify an external ownership boundary.
- Do not claim that project bindings revoke memberships created by another system; omitted projects are intentionally unmanaged.
- Reject project bindings whose observed organization differs from the organization anchor.
- Keep organization identity API calls in the narrow typed client; do not add an SDK dependency for this surface.
- Continue to treat a cluster-wide manager and its Secret cache as trusted infrastructure.

## Consequences

The platform team can create one Infisical identity per tenant and grant it roles across multiple tenant projects. A tenant-specific operator can then use that identity without requiring a separate operator fleet inside every other tenant cluster.

The resource does not create machine-identity authentication credentials, manage Infisical organizations, or enforce Kubernetes admission. Those remain separate concerns and can be composed with the project’s existing Kubernetes Auth support, native Kubernetes RBAC, or an external policy engine.

## External evidence

- [Infisical machine identities](https://infisical.com/docs/documentation/platform/identities/machine-identities) distinguishes project- and organization-level identities.
- [Create machine identity](https://infisical.com/docs/api-reference/endpoints/identities/create) defines the organization identity endpoint and its `no-access` role.
- [Project identity membership](https://infisical.com/docs/api-reference/endpoints/project-identities-membership/update-identity-membership) defines project role assignments for an identity.
