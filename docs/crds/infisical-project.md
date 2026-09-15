# InfisicalProject

`InfisicalProject` manages an Infisical project and is the parent for environments, roles, and identities.

When a project belongs to a tenant boundary, set `spec.organizationRef` to a ready `InfisicalOrganization`. This is optional for existing single-organization installations and immutable when set.

```yaml
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalProject
metadata:
  name: payments
spec:
  connectionRef:
    name: infisical
  organizationRef:
    name: payments-boundary
  projectName: payments
  slug: payments-platform
  shouldCreateDefaultEnvs: true
  creationPolicy: CreateOrAdopt
  deletionPolicy: Orphan
```

The connection reference, project type, and default-environment behavior are immutable. The project status records the remote project and organization identifiers plus observed environments, but no secret values.

Set `hasDeleteProtection: true` when the remote project should be protected by Infisical. See [lifecycle and ownership](../reference/deletion-and-ownership.md) and the [generated schema](../reference/api.md#infisicalproject).
