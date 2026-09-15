# InfisicalOrganization

`InfisicalOrganization` manages or adopts a top-level Infisical organization. It is the explicit ownership boundary for tenant-created projects and for organization-scoped machine identities.

Creating an Infisical organization requires a user JWT or API key because Infisical does not permit machine identities to create organizations. A tenant operator can adopt an existing organization by ID with its organization-scoped machine identity; the operator verifies the boundary through Infisical’s organization workspace access endpoint, with a project-visibility fallback for older Infisical versions.

## Platform-owned organization

Use a user/API-key connection when the platform should create the organization:

```yaml
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalOrganization
metadata:
  name: payments
spec:
  connectionRef:
    name: platform-infisical
  organizationName: Payments
  creationPolicy: Create
  deletionPolicy: Delete
```

After the organization is ready, select that organization in the platform user session and use the resulting connection to create an anchor project and an organization-scoped identity with `organizationRole: admin`. The identity’s Universal Auth credentials can then be supplied to a tenant operator.

## Tenant-side adoption

The tenant operator should receive only the organization machine identity credential:

```yaml
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalOrganization
metadata:
  name: tenant-boundary
spec:
  connectionRef:
    name: tenant-infisical
  organizationID: 00000000-0000-0000-0000-000000000000
  creationPolicy: Adopt
  deletionPolicy: Orphan
```

Projects created by that tenant reference `tenant-boundary` through `spec.organizationRef`. The reference is immutable and the operator rejects a project whose observed Infisical organization does not match.

Infisical does not automatically create a project membership for an organization-admin machine identity that creates a project. For an organization-scoped connection, the operator grants the creating machine identity permanent `admin` membership on the new project, allowing it to reconcile environments and other project children without an additional user-supplied identifier.

`organizationID` is intentionally visible in the tenant-side desired state. It makes the boundary auditable and allows adoption with a machine token even though Infisical’s organization lookup endpoint is user-only.
