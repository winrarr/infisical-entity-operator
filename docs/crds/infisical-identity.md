# InfisicalIdentity

`InfisicalIdentity` manages a project- or organization-scoped Infisical machine identity. Project scope is the default and keeps the original one-project workflow. Organization scope is intended for a platform-created tenant principal that can receive roles in one or more projects.

```yaml
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalIdentity
metadata:
  name: payments-workload
spec:
  connectionRef:
    name: infisical
  scope: Project
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

## Organization scope

An organization-scoped identity uses an `InfisicalProject` as an organization anchor. The operator reads that project’s observed `organizationID`; it does not create or manage an Infisical organization object. Project access is granted through explicit `projectRoleBindings`, and every bound project must belong to the anchor organization.

```yaml
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalIdentity
metadata:
  name: tenant-platform
spec:
  connectionRef:
    name: infisical
  scope: Organization
  organizationRef:
    name: tenant-project
  # Infisical's least-privilege organization role is used when omitted.
  projectRoleBindings:
    - projectRef:
        name: tenant-project
      roleSlugs:
        - tenant-secret-reader
    - projectRef:
        name: tenant-observability
      roleSlugs:
        - member
  creationPolicy: Create
  deletionPolicy: Orphan
```

`organizationRole` may be set when the identity needs an Infisical organization role; leave it empty for the least-privilege `no-access` role. Each listed project membership owns its complete permanent role list. Projects not listed are left unmanaged, so the operator does not accidentally revoke access that another controller or platform workflow owns. The organization identity itself is deleted through Infisical’s organization-identity endpoint when `deletionPolicy: Delete` is selected.

This is an Infisical authorization model, not a Kubernetes isolation boundary. For a vCluster or other tenant cluster, install an operator instance in that cluster and provide it with a machine-identity credential whose Infisical roles are limited by this resource. Keep the cluster-wide deployment trusted unless Kubernetes RBAC and Secret access are independently isolated.
