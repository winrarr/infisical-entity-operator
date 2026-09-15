# 0007: Make Infisical organizations explicit tenant boundaries

Status: accepted

Date: 2026-09-15

Supersedes: [0006: Use organization-scoped identities for tenant principals](0006-organization-scoped-identities-for-tenant-principals.md)

## Decision

Add a namespaced `InfisicalOrganization` resource and use it as the explicit Infisical tenant boundary.

- A platform connection using a user JWT or API key may create, adopt, and optionally delete a top-level Infisical organization.
- A tenant connection using an organization-scoped machine identity adopts the organization by explicit ID. The client verifies machine-token access through Infisical’s organization workspace endpoint, with a project-visibility fallback for older Infisical versions because the user-oriented organization lookup endpoints are not available to machine identities.
- An organization-scoped `InfisicalIdentity` references `InfisicalOrganization`, not a project. Its `organizationRole` defaults to `no-access`; the platform sets `admin` when the tenant must create its own projects.
- `InfisicalProject.spec.organizationRef` is optional for compatibility and, when set, requires the observed project to belong to the referenced organization.
- The operator does not implement Kubernetes tenant admission. Deployment boundaries, Kubernetes RBAC, Capsule, Kyverno, and the organization-scoped Infisical credential remain composable enforcement layers.

## Rationale

Infisical defines an organization as the outer boundary containing projects, and its organization `project.create` permission is the capability required for tenant-created projects. A dedicated top-level organization therefore maps cleanly to a tenant while allowing that tenant to create multiple projects without project-name conventions or operator-specific prefix logic.

The explicit resource keeps the core controller provider-aware but tenancy-model agnostic: vCluster can install the operator inside each virtual cluster, while a Capsule deployment can use one trusted cluster-wide operator and admission policies that constrain references. Neither path needs vCluster-specific branches in the reconciliation code.

## Constraints

- Keep organization and project references same-namespace and immutable.
- Never give a tenant connection a platform user credential; machine identity credentials are the tenant boundary.
- Do not claim that `InfisicalProject.spec.organizationRef` is an admission policy. It detects a mismatch during reconciliation; Kubernetes policy is needed to reject an invalid CR before persistence.
- Organization creation and deletion require a user JWT or API key in the currently supported Infisical API.

## Consequences

Tenants can create unlimited projects in their own organization when their organization machine identity has the Infisical `admin` organization role. The platform retains ownership of organization lifecycle and credential issuance. Tenant-side `InfisicalOrganization` resources are adoption-only and orphaned on deletion.

Infisical does not automatically add an organization-admin machine identity as a member of a project it creates. The project controller therefore grants the creating machine identity permanent project `admin` membership for organization-bound projects. This keeps project and child-resource reconciliation functional without adding an identity-ID field to the Kubernetes API.

The shared-operator Capsule pattern still requires trusted Secret access and a policy layer for Kubernetes-side boundaries. The vCluster pattern requires one operator instance per vCluster, which is an operational trade-off rather than a special branch in this operator.

## External evidence

- [Infisical organization permissions](https://infisical.com/docs/internals/permissions/organization-permissions) defines `project.create`.
- [Infisical role-based access control](https://infisical.com/docs/documentation/platform/access-controls/role-based-access-controls) describes organization Admin and Member capabilities.
- [Infisical organization structure](https://infisical.com/docs/documentation/guides/organization-structure) describes organizations as the outer project boundary.
