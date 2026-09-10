# InfisicalIdentity

`InfisicalIdentity` manages a project-scoped machine identity. It can also manage the identity’s permanent project-role membership.

```yaml
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalIdentity
metadata:
  name: payments-workload
spec:
  connectionRef:
    name: infisical
  projectRef:
    name: payments
  identityName: payments-workload
  roleSlugs:
    - payments-secret-reader
  metadata:
    - key: owner
      value: platform
  creationPolicy: CreateOrAdopt
  deletionPolicy: Orphan
```

If `roleSlugs` is omitted, existing membership is not managed. Set it explicitly to a desired set; use the built-in `no-access` slug when the identity should have no project permissions. Role reconciliation is part of the identity lifecycle and is reflected in `status.roles`.

The connection and project references are immutable. `hasDeleteProtection` controls Infisical-side protection. See the [generated schema](../reference/api.md#infisicalidentity).
