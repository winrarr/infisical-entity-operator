# Resource index

All operator CRDs are namespaced and use API group `infisical.infisical-operator.io/v1alpha1`.

| Resource | Depends on | Manages |
| --- | --- | --- |
| [InfisicalConnection](../crds/infisical-connection.md) | Same-namespace bearer-token or Universal Auth Secret | Infisical API endpoint and authentication health |
| [InfisicalOrganization](../crds/infisical-organization.md) | InfisicalConnection | Infisical organization and tenant boundary |
| [InfisicalProjectTemplate](../crds/infisical-project-template.md) | InfisicalConnection, optional InfisicalOrganization | Reusable Infisical project blueprint |
| [InfisicalProject](../crds/infisical-project.md) | InfisicalConnection, optional InfisicalOrganization and InfisicalProjectTemplate | Infisical project |
| [InfisicalEnvironment](../crds/infisical-environment.md) | InfisicalConnection, InfisicalProject | Project environment |
| [InfisicalProjectRole](../crds/infisical-project-role.md) | InfisicalConnection, InfisicalProject | Project permission role |
| [InfisicalIdentity](../crds/infisical-identity.md) | InfisicalConnection, InfisicalProject, or InfisicalOrganization | Project- or organization-scoped machine identity and optional permanent project-role membership |
| [InfisicalIdentityTemplate](../crds/infisical-identity-template.md) | InfisicalConnection, InfisicalOrganization, and method-specific Secrets | Organization-owned identity authentication template |
| [InfisicalKubernetesAuth](../crds/infisical-kubernetes-auth.md) | InfisicalConnection, InfisicalIdentity, Kubernetes API credentials as configured | Kubernetes Auth attached to an identity |
| [InfisicalUniversalAuth](../crds/infisical-universal-auth.md) | InfisicalConnection, InfisicalIdentity | Universal Auth configuration and one-time client Secret publication |

References contain names only. Cross-namespace references are not supported. The [generated API reference](api.md) contains the complete OpenAPI-derived field and validation details.
