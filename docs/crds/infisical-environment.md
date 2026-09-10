# InfisicalEnvironment

`InfisicalEnvironment` manages one environment inside an `InfisicalProject`.

```yaml
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalEnvironment
metadata:
  name: payments-qa
spec:
  connectionRef:
    name: infisical
  projectRef:
    name: payments
  environmentName: QA
  slug: qa
  position: 4
  deletionPolicy: Orphan
```

The connection, project, and slug are immutable. The reconciler waits for both referenced resources and records the environment ID, project ID, name, slug, position, and readiness in status.

See the [generated schema](../reference/api.md#infisicalenvironment) for all validation rules.
