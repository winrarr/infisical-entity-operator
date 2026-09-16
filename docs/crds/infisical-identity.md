# InfisicalIdentity

`InfisicalIdentity` manages a project- or organization-scoped Infisical machine identity. Project scope is the default and keeps the original one-project workflow. Organization scope is intended for a platform-created tenant principal that can create or access projects within one `InfisicalOrganization`.

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
    - viewer
  metadata:
    - key: owner
      value: platform
  creationPolicy: CreateOrAdopt
  deletionPolicy: Orphan
```

If `roleSlugs` is omitted, existing membership is not managed. Set it explicitly to a desired set of built-in roles: `admin`, `member`, `viewer`, or `no-access`. Role reconciliation is part of the identity lifecycle and is reflected in `status.roles`.

The connection and project references are immutable. `hasDeleteProtection` controls Infisical-side protection. See the [generated schema](../reference/api.md#infisicalidentity).

## Organization scope

An organization-scoped identity references an `InfisicalOrganization`. The operator waits for that resource’s observed organization ID, and every bound project must report the same organization. Set `organizationRole: admin` when the tenant must create its own projects; leave it empty for Infisical’s `no-access` role and grant access only through explicit `projectRoleBindings`.

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
    name: tenant-boundary
  organizationRole: admin
  projectRoleBindings:
    - projectRef:
        name: tenant-project
      roleSlugs:
        - viewer
    - projectRef:
        name: tenant-observability
      roleSlugs:
        - member
  creationPolicy: Create
  deletionPolicy: Orphan
```

`organizationRole` may be set to one of Infisical’s built-in organization roles: `admin`, `member`, or `no-access`. Each listed project membership owns its complete permanent role list. Projects not listed are left unmanaged, so the operator does not accidentally revoke access that another controller or platform workflow owns. The organization identity itself is deleted through Infisical’s organization-identity endpoint when `deletionPolicy: Delete` is selected.

This is an Infisical authorization model, not a Kubernetes isolation boundary. For a vCluster or other tenant cluster, install an operator instance in that cluster and provide it with a machine-identity credential whose Infisical organization role is limited to that tenant organization. Keep the cluster-wide deployment trusted unless Kubernetes RBAC and Secret access are independently isolated.
