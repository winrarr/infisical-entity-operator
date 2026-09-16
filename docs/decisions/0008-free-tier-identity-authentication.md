# 0008: Keep identity authentication free-tier and credential-safe

Status: accepted

Date: 2026-09-16

## Decision

Support only the Infisical identity authentication methods available through the free-tier control-plane workflow:

- `InfisicalKubernetesAuth` configures direct Kubernetes API-server token review for an `InfisicalIdentity`.
- `InfisicalUniversalAuth` owns Universal Auth configuration, publishes one-time client credentials to a same-namespace Secret, and supports explicit rotation and revocation.
- `InfisicalConnection` accepts either a bearer-token Secret or Universal Auth credentials and performs short-lived token exchange in the typed client.

Gateway-backed Kubernetes Auth and identity-authentication templates are intentionally not represented. They are paid or enterprise-oriented product surfaces, and omitting them keeps the operator’s API aligned with the free tier.

## Rationale

These resources manage lifecycle entities rather than secret-consumer configuration. Local references express the dependency graph, while Secret references keep credentials out of status, logs, events, and generated manifests.

Infisical returns a Universal Auth client secret only when it is created, so explicit publication and rotation are required for recoverable declarative management. The typed client owns authentication mechanics and response compatibility; controllers only select credentials and declare desired resources.

## Consequences

The operator does not provide a reusable authentication-template abstraction or gateway integration. Kubernetes Auth configuration is explicit per resource and must use an API endpoint reachable from Infisical. Universal Auth credentials are held only for reconciliation and short-lived access tokens are cached by the client.

Other authentication methods remain outside the operator until they have a documented free-tier lifecycle use case.

## External evidence

- [Infisical Kubernetes Auth](https://infisical.com/docs/documentation/platform/identities/kubernetes-auth)
- [Infisical Universal Auth](https://infisical.com/docs/documentation/platform/identities/universal-auth)
- [Create a Universal Auth client secret](https://infisical.com/docs/api-reference/endpoints/universal-auth/create-client-secret)
- [Revoke a Universal Auth client secret](https://infisical.com/docs/api-reference/endpoints/universal-auth/revoke-client-secret)
