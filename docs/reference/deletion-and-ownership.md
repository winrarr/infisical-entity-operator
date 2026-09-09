# Lifecycle and ownership

The operator uses stable Kubernetes object names and immutable identity fields to make remote ownership explicit.

## Acquisition

Resources support three `creationPolicy` values:

- `Create` creates the remote entity and records its identifier.
- `Adopt` looks up an existing matching entity and fails if it cannot find one.
- `CreateOrAdopt` first attempts adoption, then creates if no matching entity exists.

Connection, project, identity, role, and environment identity references are immutable after creation. Kubernetes rejects a change that could silently move a resource to another endpoint or project; delete and recreate the custom resource when that move is intended.

## Deletion

`deletionPolicy: Orphan` is the default. The Kubernetes object disappears without deleting the Infisical entity. `deletionPolicy: Delete` adds a finalizer and asks Infisical to delete the remote entity before allowing Kubernetes deletion to finish.

Deletion is dependency-aware. Keep parent resources until child resources have been deleted or orphaned according to the intended ownership policy. If a remote entity is already absent, the operator treats it as converged and removes its finalizer.

Delete protection is an Infisical-side setting for projects and identities. It can intentionally prevent remote deletion; inspect the `Ready` condition and controller events when a `Delete` policy cannot complete.

See [safe external deletion](../decisions/0002-safe-external-deletion.md) for the design rationale.
