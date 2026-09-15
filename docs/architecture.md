# Architecture

## Components

```text
Kubernetes API
    │ watches and status updates
    ▼
Controller manager ── bearer-token HTTP ──▶ Infisical API
    │
    ├── InfisicalConnection: reachability and credential dependency
    ├── InfisicalOrganization: top-level organization lifecycle and tenant boundary
    ├── InfisicalProjectTemplate: reusable project blueprint lifecycle
    ├── InfisicalProject: external project lifecycle and observed environments
    ├── InfisicalEnvironment: project environment lifecycle
    ├── InfisicalProjectRole: project permission role lifecycle
    ├── InfisicalIdentity: project- or organization-scoped machine identity and permanent role membership lifecycle
    └── InfisicalKubernetesAuth: Kubernetes service-account authentication configuration
```

The manager uses controller-runtime for caching, reconciliation, status subresources, and finalizer updates. The external integration is a narrow typed HTTP client rather than a broad abstraction: its public surface mirrors only the organization, project, project-template, environment, project-role, project-identity, organization-identity, project-membership, and Kubernetes Auth endpoints used by the controllers.

## Ownership and identity

The Kubernetes object is the desired-state owner. External IDs are persisted in status after create or adopt and are then used for all subsequent reads, updates, and deletes. When an external object disappears, the controller clears the ID and follows the resource’s creation policy on the next reconciliation.

Names are used only for the initial adopt lookup. An organization is matched by explicit ID or name; a project is matched by requested slug first and then name; a project template is matched by name; a project identity is matched by name inside the observed project; an organization identity is matched by name inside the observed organization; environments and roles are matched by stable slug. References are same-namespace and immutable where changing them would otherwise move an existing external object. Environment and role slugs are immutable after creation because downstream permissions and secret paths can refer to them.

## Reconciliation flow

1. Fetch the CR and exit if it was deleted from the API.
2. Add the finalizer only when `deletionPolicy: Delete` is selected. On deletion, orphan immediately or delete the recorded external ID and then remove the finalizer.
3. Resolve the connection Secret and construct a client with the configured timeout.
4. Resolve the Kubernetes dependency when applicable, and resolve auth-method Secrets without placing their contents in status.
5. Adopt if allowed and no external ID is recorded; otherwise create if allowed.
6. Read by external ID or identity, patch mutable fields when they drift, and reconcile declared identity memberships through Infisical’s identity-membership API. Project scope manages one project; organization scope resolves an explicit organization and optionally manages the listed project bindings after verifying they belong to it. A project template is selected by name only while a project is being created; later template edits do not mutate existing projects.
7. Write status and schedule a periodic drift check.
8. Set a dependency or external failure condition and requeue with a shorter dependency delay or longer external delay.

## Failure and security model

Bearer tokens are read only from Secret data and sent as Authorization headers. Kubernetes Auth token-review JWTs and CA certificates are also read only from Secret data. Error conditions include API status and a bounded response body, but neither logs nor status contain credentials. Same-namespace Secret references and explicit API URL validation limit accidental credential and endpoint confusion. Local Cilium policy tests allow only Kubernetes API, DNS, and the Infisical service egress from the manager.
