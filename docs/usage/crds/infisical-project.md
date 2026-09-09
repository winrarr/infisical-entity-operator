# InfisicalProject

`InfisicalProject` manages an Infisical project and is the parent for environments, roles, and identities.

```yaml
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalProject
metadata:
  name: payments
spec:
  connectionRef:
    name: infisical
  projectName: payments
  slug: payments-platform
  shouldCreateDefaultEnvs: true
  creationPolicy: CreateOrAdopt
  deletionPolicy: Orphan
```

The connection reference, project type, and default-environment behavior are immutable. The project status records the remote project and organization identifiers plus observed environments, but no secret values.

Set `hasDeleteProtection: true` when the remote project should be protected by Infisical. See [lifecycle and ownership](../reference/deletion-and-ownership.md) and the [generated schema](../reference/api.md#infisicalproject).
