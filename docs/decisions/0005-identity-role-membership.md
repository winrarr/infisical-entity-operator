# 0005: Manage identity roles through optional permanent role slugs

Status: accepted

Date: 2026-09-09

## Decision

Extend `InfisicalIdentity` with an optional `spec.roleSlugs` list. When the field is set, the operator owns the identity’s complete permanent project-role set through Infisical’s project identity-membership API. When the field is omitted, the operator leaves the existing project membership unmanaged. Users can explicitly set the built-in `no-access` role when they want to remove project permissions.

Use built-in role slugs rather than role IDs in desired state. The supported project role slugs are `admin`, `member`, `viewer`, and `no-access`; always send `isTemporary: false`.

## Rationale

Infisical’s current membership API assigns roles by project ID and identity ID, and supports a role list on create and update. Built-in role slugs keep manifests portable across environments. Optional management preserves existing identities whose manifests do not yet declare role membership, while `no-access` provides an explicit least-privilege state.

Project-scoped identities remain a one-project resource. Organization-scoped identities and their multiple project memberships are handled by the scope and binding fields described in [0006: Use organization-scoped identities for tenant principals](0006-organization-scoped-identities-for-tenant-principals.md), rather than by a separate membership CRD.

## Constraints

- Manage permanent assignments only; time-bound roles remain outside the current API contract.
- Treat the declared list as a complete set and correct manual additions, removals, and temporary assignments.
- Keep role slugs in desired state and safe role observations in status; never store bearer tokens or other authentication material.
- Reject custom role slugs before making an external API call; custom roles are outside the supported free-tier surface.

## Consequences

Existing identity resources remain compatible when `roleSlugs` is omitted. A user who wants the operator to remove all project access must declare `no-access`; omission intentionally means “do not manage membership,” not “remove all roles.”
