# Concepts

The operator turns a small Infisical control-plane graph into Kubernetes resources. References are local and same-namespace, which makes the dependency boundary visible in each manifest.

```text
Connection
    ↓
Project ──→ Environment
    ├────→ ProjectRole
    └────→ Identity ──→ KubernetesAuth
```

Each reconciler follows the same broad lifecycle:

1. Validate local references and required Secrets.
2. Wait for dependent custom resources to become ready.
3. Create, adopt, or observe the corresponding Infisical entity.
4. Correct supported drift and record stable remote identifiers in status.
5. Optionally delete the remote entity through an explicit finalizer policy.

`creationPolicy` controls how an entity is acquired. `Create` creates a new entity, `Adopt` requires an existing matching entity, and `CreateOrAdopt` tries to use an existing entity before creating one.

`deletionPolicy` controls remote cleanup. `Orphan` is the default and removes only the Kubernetes object. `Delete` requests remote deletion when the resource is deleted; it is intentionally opt-in because the operation may be irreversible.

Project-scoped identities manage project-role membership only when `spec.roleSlugs` is present. Omitting the field leaves an existing membership unmanaged. Set `roleSlugs: [no-access]` when a project identity should explicitly have no project permissions. Organization-scoped identities use `spec.organizationRef` as an organization anchor and manage the listed project memberships through `spec.projectRoleBindings`; the organization role defaults to Infisical’s `no-access` role.

The [resource reference](../reference/resources.md), [status conventions](../reference/status-and-conditions.md), and [generated API reference](../reference/api.md) describe the exact contract.
