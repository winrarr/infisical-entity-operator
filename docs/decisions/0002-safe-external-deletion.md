# 0002: Safe external deletion defaults

Status: accepted

## Decision

Default every managed entity to `deletionPolicy: Orphan`; require `Delete` to be set explicitly.

## Rationale

Deleting a Kubernetes object is common during refactoring and can be accidental. Infisical project deletion is irreversible and can affect secrets and identities beyond the object being removed. Orphaning preserves the external resource for recovery or later adoption. A finalizer still supports deliberate cleanup when a user chooses `Delete`.
