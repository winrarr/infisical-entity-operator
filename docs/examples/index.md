# Examples

Examples are intentionally small and omit credentials. Apply them only after creating the required same-namespace Secrets.

- [Project and identity](project-and-identity.md) creates a project, environment, role, and machine identity.
- The complete checked-in manifests are in [`config/samples/`](https://github.com/winrarr/infisical-entity-operator/tree/main/config/samples).
- The [quickstart](../quickstart.md) covers the minimum connection and project path.

For Kubernetes Auth, provide a CA and reviewer JWT Secret only when the selected Infisical authentication mode requires them. Do not commit those values.
