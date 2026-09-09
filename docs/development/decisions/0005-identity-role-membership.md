# 0005: Manage identity roles through optional permanent role slugs

Status: accepted

Date: 2026-09-09

## Decision

Extend `InfisicalIdentity` with an optional `spec.roleSlugs` list. When the field is set, the operator owns the identity’s complete permanent project-role set through Infisical’s project identity-membership API. When the field is omitted, the operator leaves the existing project membership unmanaged. Users can explicitly set the built-in `no-access` role when they want to remove project permissions.

Use role slugs rather than role IDs in desired state. Normalize Infisical’s built-in and custom-role response shapes to slugs for comparison, and always send `isTemporary: false`.

## Rationale

Infisical’s current membership API assigns roles by project ID and identity ID, and supports a role list on create and update. Role slugs keep manifests portable across environments and allow custom roles to be created independently. Optional management preserves existing identities whose manifests do not yet declare role membership, while `no-access` provides an explicit least-privilege state.

The operator does not introduce a separate membership CRD because the initial product scope is a project-scoped identity with one owning project. A separate resource can be considered if identities become shared across projects.

## Constraints

- Manage permanent assignments only; time-bound roles remain outside the current API contract.
- Treat the declared list as a complete set and correct manual additions, removals, and temporary assignments.
- Keep role slugs in desired state and safe role observations in status; never store bearer tokens or other authentication material.
- Reconcile matching `InfisicalProjectRole` changes so a role resource becoming available can unblock an identity.

## Consequences

Existing identity resources remain compatible when `roleSlugs` is omitted. A user who wants the operator to remove all project access must declare `no-access`; omission intentionally means “do not manage membership,” not “remove all roles.”
