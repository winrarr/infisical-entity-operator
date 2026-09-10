# Custom resources

The six namespaced CRDs form one small dependency graph. Begin with `InfisicalConnection`, then create a project and its project-scoped resources.

Each guide contains the resource’s purpose, dependency rules, lifecycle behavior, and a minimal manifest. For all fields, defaults, validation, and status properties, use the [generated API reference](../reference/api.md).

| Guide | Scope |
| --- | --- |
| [InfisicalConnection](infisical-connection.md) | API endpoint and bearer-token Secret |
| [InfisicalProject](infisical-project.md) | Infisical project |
| [InfisicalEnvironment](infisical-environment.md) | Project environment |
| [InfisicalProjectRole](infisical-project-role.md) | Project permission role |
| [InfisicalIdentity](infisical-identity.md) | Machine identity and optional role membership |
| [InfisicalKubernetesAuth](infisical-kubernetes-auth.md) | Kubernetes Auth attached to an identity |
