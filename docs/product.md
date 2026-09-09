# Product scope

## Problem

Kubernetes platform definitions often need an Infisical project and one or more machine identities before workloads can consume secrets. Managing those control-plane entities manually creates an ownership gap: the Kubernetes resource can exist while the corresponding Infisical object is missing, changed, or attached to the wrong project.

## Initial outcome

This project makes the control-plane relationship declarative and observable:

1. A same-namespace `InfisicalConnection` supplies a host and bearer token.
2. An `InfisicalProject` creates or adopts a project and records its external identity and environments.
3. An `InfisicalEnvironment` waits for its project, then creates or adopts a project environment.
4. An `InfisicalProjectRole` waits for its project, then creates or adopts a typed project permission role.
5. An `InfisicalIdentity` waits for its project, then creates or adopts a machine identity and manages its mutable metadata.
6. An `InfisicalKubernetesAuth` waits for its identity, then configures Kubernetes service-account authentication for that identity.

Each resource reports a Kubernetes `Ready` condition, requeues after external drift checks, and watches the local dependencies that can change its result.

## Safety boundaries

Creation is explicit through `creationPolicy`, while deletion is safe by default through `deletionPolicy: Orphan`. Setting `Delete` opts into external deletion. The operator never copies bearer tokens, token-review JWTs, or certificates into status or generated Secrets.

## Non-goals for the first version

- Synchronizing secrets into workloads; the official Infisical Kubernetes operator already owns that integration.
- Managing organizations, folders, secrets, dynamic secrets, RBAC groups, identity role assignments, or authentication methods other than Kubernetes Auth.
- Cross-namespace references or cross-tenant credential sharing.
- Replacing the Infisical CLI or SDK for secret retrieval.
