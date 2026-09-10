# InfisicalProjectRole

`InfisicalProjectRole` manages a custom permission role in an Infisical project. A role has one or more subject/action permission rules, optionally limited by resource conditions.

```yaml
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalProjectRole
metadata:
  name: payments-secret-reader
spec:
  connectionRef:
    name: infisical
  projectRef:
    name: payments
  roleName: Payments Secret Reader
  slug: payments-secret-reader
  permissions:
    - subject: secrets
      action:
        - readValue
      conditions:
        environment:
          $eq: production
  creationPolicy: CreateOrAdopt
  deletionPolicy: Orphan
```

The connection, project, and slug are immutable. Infisical plan capability still governs whether custom roles can be created or updated at a given endpoint. The [local verification guide](../verification.md) records the known standalone-chart limitation.

See the [generated schema](../reference/api.md#infisicalprojectrole) for condition operators and complete status fields.
