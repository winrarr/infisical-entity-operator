# Resource index

All operator CRDs are namespaced and use API group `infisical.infisical-operator.io/v1alpha1`.

| Resource | Depends on | Manages |
| --- | --- | --- |
| [InfisicalConnection](../crds/infisical-connection.md) | Same-namespace bearer-token or Universal Auth Secret | Infisical API endpoint and authentication health |
| [InfisicalOrganization](../crds/infisical-organization.md) | InfisicalConnection | Infisical organization and tenant boundary |
| [InfisicalProject](../crds/infisical-project.md) | InfisicalConnection, optional InfisicalOrganization | Infisical project |
| [InfisicalEnvironment](../crds/infisical-environment.md) | InfisicalConnection, InfisicalProject | Project environment |
| [InfisicalIdentity](../crds/infisical-identity.md) | InfisicalConnection, InfisicalProject, or InfisicalOrganization | Project- or organization-scoped machine identity and optional permanent membership in built-in roles |
| [InfisicalKubernetesAuth](../crds/infisical-kubernetes-auth.md) | InfisicalConnection, InfisicalIdentity, Kubernetes API credentials as configured | Kubernetes Auth attached to an identity |
| [InfisicalUniversalAuth](../crds/infisical-universal-auth.md) | InfisicalConnection, InfisicalIdentity | Universal Auth configuration and one-time client Secret publication |

References contain names only. Cross-namespace references are not supported. The [generated API reference](api.md) contains the complete OpenAPI-derived field and validation details.
