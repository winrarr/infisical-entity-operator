# Custom resources

The seven namespaced CRDs form one small dependency graph. Begin with `InfisicalConnection`, then create or adopt an organization when a tenant boundary is needed, and create projects and their project-scoped resources. `InfisicalIdentity` also supports an organization-scoped machine identity with a built-in organization role and optional project-role bindings. `InfisicalUniversalAuth` can publish credentials for a tenant-specific connection.

Each guide contains the resource’s purpose, dependency rules, lifecycle behavior, and a minimal manifest. For all fields, defaults, validation, and status properties, use the [generated API reference](../reference/api.md).

| Guide | Scope |
| --- | --- |
| [InfisicalConnection](infisical-connection.md) | API endpoint and bearer-token or Universal Auth Secret |
| [InfisicalOrganization](infisical-organization.md) | Infisical organization and tenant boundary |
| [InfisicalProject](infisical-project.md) | Infisical project |
| [InfisicalEnvironment](infisical-environment.md) | Project environment |
| [InfisicalIdentity](infisical-identity.md) | Project- or organization-scoped machine identity and role membership |
| [InfisicalKubernetesAuth](infisical-kubernetes-auth.md) | Kubernetes Auth attached to an identity |
| [InfisicalUniversalAuth](infisical-universal-auth.md) | Universal Auth configuration and client Secret |
