# InfisicalProjectTemplate

`InfisicalProjectTemplate` manages a reusable Infisical project blueprint. It can define the project type, environments, custom roles, and optional users, groups, organization identities, and project-owned identities to add when a project is created from it.

Templates are created or adopted in the organization selected by the connection. Set `spec.organizationRef` when the template is part of an explicit tenant boundary. Set `spec.templateRef` on an `InfisicalProject` to use the template; the project waits until the template is ready.

```yaml
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalProjectTemplate
metadata:
  name: platform-defaults
spec:
  connectionRef:
    name: infisical
  organizationRef:
    name: platform-organization
  templateName: platform-defaults
  type: secret-manager
  environments:
    - name: Production
      slug: prod
      position: 1
    - name: Development
      slug: dev
      position: 2
  creationPolicy: CreateOrAdopt
  deletionPolicy: Orphan
```

`type` is immutable because Infisical does not expose it as a template update field. Template contents are reconciled after creation. Deleting the Kubernetes resource leaves the remote template by default; use `deletionPolicy: Delete` only when that ownership is intended. See the [generated schema](../reference/api.md#infisicalprojecttemplate).
