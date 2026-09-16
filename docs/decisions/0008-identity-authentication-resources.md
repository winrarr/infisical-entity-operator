# 0008: Keep identity authentication resources typed and credential-safe

Status: accepted

Date: 2026-09-16

## Decision

Represent Infisical identity authentication templates and Universal Auth as separate namespaced resources:

- `InfisicalIdentityTemplate` owns organization-scoped LDAP, Kubernetes, or OIDC template configuration and reads sensitive fields from same-namespace Secrets.
- `InfisicalKubernetesAuth` references a local ready `InfisicalIdentityTemplate` when template-managed fields are desired, while keeping allowlists and token lifetime settings on the auth resource.
- `InfisicalUniversalAuth` owns the Universal Auth configuration for an `InfisicalIdentity`, publishes the one-time client secret to a same-namespace Secret, and supports explicit rotation and revocation.
- `InfisicalConnection` accepts either the existing bearer-token Secret or a Universal Auth credential Secret and performs the short-lived token exchange in the typed client.

## Rationale

These are lifecycle entities rather than secret-consumer configuration. Local references express the dependency graph and allow organization ownership checks before the operator calls Infisical. Infisical returns a Universal Auth client secret only when it is created, so explicit publication and rotation are required for recoverable declarative management.

Secret values never enter status, logs, events, or generated manifests. Write-only identity-template values use referenced Secret resource versions as a non-sensitive change signal. Universal Auth output Secrets are owned by the custom resource, and a replacement is published before the previous remote secret is revoked.

The typed client owns authentication mechanics and response compatibility. Controllers only select credentials and declare desired resources, so bearer-token and Universal Auth connections do not create provider-specific branches in every controller.

## Consequences

Universal Auth client credentials are held in the client instance while a reconciliation uses them; the short-lived access token is cached in that client and refreshed before expiry. A connection Secret change requeues dependent resources. Removing or recreating an output Secret causes a new remote client secret to be issued because the original value cannot be retrieved from Infisical.

The API surface intentionally covers only the identity-template variants and Universal Auth operations needed by these resources. Other Infisical authentication methods remain outside the operator until their lifecycle needs are concrete.

## External evidence

- [Infisical identity-template OpenAPI operations](https://app.infisical.com/api/docs/json)
- [Infisical Universal Auth guide](https://infisical.com/docs/documentation/platform/identities/universal-auth)
- [Create a Universal Auth client secret](https://infisical.com/docs/api-reference/endpoints/universal-auth/create-client-secret)
- [Revoke a Universal Auth client secret](https://infisical.com/docs/api-reference/endpoints/universal-auth/revoke-client-secret)
