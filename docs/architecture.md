# Architecture

## Components

```text
Kubernetes API
    │ watches and status updates
    ▼
Controller manager ── bearer-token HTTP ──▶ Infisical API
    │
    ├── InfisicalConnection: reachability and credential dependency
    ├── InfisicalProject: external project lifecycle and observed environments
    └── InfisicalIdentity: project-scoped machine identity lifecycle
```

The manager uses controller-runtime for caching, reconciliation, status subresources, and finalizer updates. The external integration is a narrow typed HTTP client rather than a broad abstraction: its public surface mirrors only the project and project-identity endpoints used by the controllers.

## Ownership and identity

The Kubernetes object is the desired-state owner. External IDs are persisted in status after create or adopt and are then used for all subsequent reads, updates, and deletes. When an external object disappears, the controller clears the ID and follows the resource’s creation policy on the next reconciliation.

Names are used only for the initial adopt lookup. A project is matched by requested slug first and then name; an identity is matched by name inside the observed project. References are same-namespace and immutable where changing them would otherwise move an existing external object.

## Reconciliation flow

1. Fetch the CR and exit if it was deleted from the API.
2. Add the finalizer only when `deletionPolicy: Delete` is selected. On deletion, orphan immediately or delete the recorded external ID and then remove the finalizer.
3. Resolve the connection Secret and construct a client with the configured timeout.
4. Adopt if allowed and no external ID is recorded; otherwise create if allowed.
5. Read by external ID, patch mutable fields when they drift, write status, and schedule a periodic drift check.
6. Set a dependency or external failure condition and requeue with a shorter dependency delay or longer external delay.

## Failure and security model

Bearer tokens are read only from Secret data and sent as Authorization headers. Error conditions include API status and a bounded response body, but neither logs nor status contain the token. Same-namespace Secret references and explicit API URL validation limit accidental credential and endpoint confusion. Local Cilium policy tests allow only Kubernetes API, DNS, and the Infisical service egress from the manager.
