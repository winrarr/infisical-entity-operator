# Resource index

All operator CRDs are namespaced and use API group `infisical.infisical-operator.io/v1alpha1`.

| Resource | Depends on | Manages |
| --- | --- | --- |
| [InfisicalConnection](../crds/infisical-connection.md) | Same-namespace bearer-token Secret | Infisical API endpoint and authentication health |
| [InfisicalProject](../crds/infisical-project.md) | InfisicalConnection | Infisical project |
| [InfisicalEnvironment](../crds/infisical-environment.md) | InfisicalConnection, InfisicalProject | Project environment |
| [InfisicalProjectRole](../crds/infisical-project-role.md) | InfisicalConnection, InfisicalProject | Project permission role |
| [InfisicalIdentity](../crds/infisical-identity.md) | InfisicalConnection, InfisicalProject | Machine identity and optional permanent project-role membership |
| [InfisicalKubernetesAuth](../crds/infisical-kubernetes-auth.md) | InfisicalConnection, InfisicalIdentity, Kubernetes API credentials as configured | Kubernetes Auth attached to an identity |

References contain names only. Cross-namespace references are not supported. The [generated API reference](api.md) contains the complete OpenAPI-derived field and validation details.
