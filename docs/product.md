# Product scope

## Problem

Kubernetes platform definitions often need an Infisical project and one or more machine identities before workloads can consume secrets. Managing those control-plane entities manually creates an ownership gap: the Kubernetes resource can exist while the corresponding Infisical object is missing, changed, or attached to the wrong project.

## Initial outcome

This project makes the free-tier control-plane relationship declarative and observable:

1. A same-namespace `InfisicalConnection` supplies a host and bearer token or Universal Auth credentials.
2. An `InfisicalOrganization` creates or adopts an explicit top-level organization boundary.
3. An `InfisicalProject` creates or adopts a `secret-manager` or `cert-manager` project, optionally validates its organization boundary, and records its external identity and environments.
4. An `InfisicalEnvironment` waits for its project, then creates or adopts a project environment.
5. An `InfisicalIdentity` creates or adopts a project- or organization-scoped machine identity, manages its mutable metadata, and optionally manages permanent membership in Infisical’s built-in organization and project roles.
6. An `InfisicalKubernetesAuth` waits for its identity, then configures direct API-server Kubernetes service-account authentication for that identity.
7. An `InfisicalUniversalAuth` manages a machine identity’s Universal Auth configuration, rotates its one-time client secret, and publishes tenant credentials to a same-namespace Secret for a connection to exchange.

Each resource reports a Kubernetes `Ready` condition, requeues after external drift checks, and watches the local dependencies that can change its result.

This scope targets Infisical’s Free plan and does not attempt to expose paid or enterprise-only features. The operator does not enforce Infisical account quotas, so the account’s plan limits still apply.

## Safety boundaries

Creation is explicit through `creationPolicy`, while deletion is safe by default through `deletionPolicy: Orphan`. Setting `Delete` opts into external deletion. The operator never copies bearer tokens, token-review JWTs, or certificates into status or generated Secrets.

## Non-goals for the first version

- Synchronizing secrets into workloads; the official Infisical Kubernetes operator already owns that integration.
- Managing Infisical folders, secrets, dynamic secrets, groups, custom roles, project templates, gateway-backed authentication, LDAP/OIDC identity templates, KMS, PAM, secret scanning, or other paid product surfaces.
- Cross-namespace references or cross-tenant credential sharing.
- Replacing the Infisical CLI or SDK for secret retrieval.
