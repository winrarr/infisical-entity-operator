# Product scope

## Problem

Kubernetes platform definitions often need an Infisical project and one or more machine identities before workloads can consume secrets. Managing those control-plane entities manually creates an ownership gap: the Kubernetes resource can exist while the corresponding Infisical object is missing, changed, or attached to the wrong project.

## Initial outcome

This project makes the control-plane relationship declarative and observable:

1. A same-namespace `InfisicalConnection` supplies a host and bearer token.
2. An `InfisicalOrganization` creates or adopts an explicit top-level organization boundary.
3. An `InfisicalProjectTemplate` creates or adopts a reusable project blueprint, including environments, roles, and optional memberships.
4. An `InfisicalProject` creates or adopts a project, optionally validates its organization boundary, applies a project template at creation time, and records its external identity and environments.
5. An `InfisicalEnvironment` waits for its project, then creates or adopts a project environment.
6. An `InfisicalProjectRole` waits for its project, then creates or adopts a typed project permission role.
7. An `InfisicalIdentity` creates or adopts a project- or organization-scoped machine identity, manages its mutable metadata, and optionally manages permanent project-role membership. Organization scope can grant the identity an Infisical organization role and explicit roles in selected projects.
8. An `InfisicalIdentityTemplate` manages organization-owned LDAP, Kubernetes, or OIDC identity-authentication configuration, while keeping sensitive fields in same-namespace Secrets.
9. An `InfisicalKubernetesAuth` waits for its identity and an optional identity template, then configures Kubernetes service-account authentication for that identity.
10. An `InfisicalUniversalAuth` manages a machine identity’s Universal Auth configuration, rotates its one-time client secret, and publishes tenant credentials to a same-namespace Secret for a connection to exchange.

Each resource reports a Kubernetes `Ready` condition, requeues after external drift checks, and watches the local dependencies that can change its result.

## Safety boundaries

Creation is explicit through `creationPolicy`, while deletion is safe by default through `deletionPolicy: Orphan`. Setting `Delete` opts into external deletion. The operator never copies bearer tokens, token-review JWTs, or certificates into status or generated Secrets.

## Non-goals for the first version

- Synchronizing secrets into workloads; the official Infisical Kubernetes operator already owns that integration.
- Managing Infisical folders, secrets, dynamic secrets, standalone RBAC groups, temporary project-role assignments, or authentication methods other than the supported identity templates, Kubernetes Auth, and Universal Auth.
- Cross-namespace references or cross-tenant credential sharing.
- Replacing the Infisical CLI or SDK for secret retrieval.
