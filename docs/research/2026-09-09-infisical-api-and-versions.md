# Infisical API and dependency research

Research date: 2026-09-09.

## API contract

The client is based on the official Infisical API documentation:

- [Create project](https://infisical.com/docs/api-reference/endpoints/projects/create-project), [get project](https://infisical.com/docs/api-reference/endpoints/projects/get-project), [update project](https://infisical.com/docs/api-reference/endpoints/projects/update-project), and [delete project](https://infisical.com/docs/api-reference/endpoints/projects/delete-project).
- [Create identity](https://infisical.com/docs/api-reference/endpoints/project-identities/add-identity), [list identities](https://infisical.com/docs/api-reference/endpoints/project-identities/list-identity), [get identity](https://infisical.com/docs/api-reference/endpoints/project-identities/get-by-id), [update identity](https://infisical.com/docs/api-reference/endpoints/project-identities/update-identity), and [delete identity](https://infisical.com/docs/api-reference/endpoints/project-identities/delete-identity).
- [Environment endpoints](https://infisical.com/docs/api-reference/endpoints/environments/create) cover environment creation and lifecycle, including soft-delete restoration.
- [Project identity membership](https://infisical.com/docs/api-reference/endpoints/project-identities-membership/add-identity-membership) defines assignment of Infisical’s built-in project roles.
- [Project identity membership create](https://infisical.com/docs/api-reference/endpoints/project-identities-membership/add-identity-membership), [get](https://infisical.com/docs/api-reference/endpoints/project-identities-membership/get-by-id), [update](https://infisical.com/docs/api-reference/endpoints/project-identities-membership/update-identity-membership), and [delete](https://infisical.com/docs/api-reference/endpoints/project-identities-membership/delete-identity-membership) define role assignment by project and identity ID.
- [Kubernetes Auth attach](https://infisical.com/docs/api-reference/endpoints/kubernetes-auth/attach) defines identity-scoped Kubernetes Auth configuration.
- [Automated bootstrapping](https://infisical.com/docs/self-hosting/guides/automated-bootstrapping) documents the in-cluster bootstrap Secret and its instance-admin token.
- [Organization identity creation](https://infisical.com/docs/api-reference/endpoints/identities/create), [organization identity listing](https://infisical.com/docs/api-reference/endpoints/identities/list), [identity lookup](https://infisical.com/docs/api-reference/endpoints/identities/get-by-id), [identity update](https://infisical.com/docs/api-reference/endpoints/identities/update), and [identity deletion](https://infisical.com/docs/api-reference/endpoints/identities/delete) define the organization-scoped machine identity lifecycle.
- [Organization structure](https://infisical.com/docs/documentation/guides/organization-structure), [projects](https://infisical.com/docs/documentation/platform/project), and [machine identities](https://infisical.com/docs/documentation/platform/identities/machine-identities) document organizations as the outer boundary, projects as isolated workspaces, and organization identities as principals assignable to multiple projects.

The public entity endpoints use `/api/v1/projects`, `/api/v1/projects/{projectId}/environments`, `/api/v1/projects/{projectId}/identities`, `/api/v1/identities`, `/api/v1/identities/{identityId}`, `/api/v1/projects/{projectId}/memberships/identities/{identityId}`, and `/api/v1/auth/kubernetes-auth/identities/{identityId}`. Organization lifecycle uses `/api/v2/organizations`, `/api/v1/organization`, `/api/v2/organizations/{organizationId}/memberships`, and `/api/v2/organizations/{organizationId}/workspaces`; the organization membership endpoint is the strongest machine-token-compatible boundary check, while a non-empty workspace response and project visibility provide compatibility fallbacks. The checked-in client deliberately models only the fields and response envelopes needed by the CRDs. It sends bearer authentication, bounds response error bodies, classifies 404s, and applies a per-client timeout. Identity membership role requests use the current `roles` array with `isTemporary: false`; responses normalize built-in roles to one comparison key. Organization identity listing uses `orgId` and normalizes Infisical’s organization-membership list envelope to the identity model used by the controller.

## Current OpenAPI comparison

The live [Infisical OpenAPI document](https://app.infisical.com/api/docs/json) was checked again on 2026-09-16. It is an OpenAPI 3.0.3 document with 1,486 paths and an informational version of `0.0.1`; that informational value is not used as a server-version pin. The project creation enum includes several paid product types, but this operator intentionally supports only `secret-manager` and `cert-manager`, which are available in the free-tier product paths. The operator also omits project templates, custom roles, gateway-backed Kubernetes Auth, and identity-authentication templates. The reviewed free-tier operation subset and automated check are documented in the [API compatibility policy](../compatibility.md).

The OpenAPI document also contains many product areas outside this operator’s free-tier control-plane scope, including secret synchronization, dynamic secrets, PKI, integrations, app connections, groups, authentication templates, and paid product features. Those are intentionally not represented as CRDs; adding a resource should be driven by a concrete free-tier lifecycle use case rather than by mirroring every API tag. Universal Auth remains represented by `InfisicalUniversalAuth`; the excluded product areas stay out of scope.

## Tenant-principal design

The operator uses an organization-scoped machine identity as the Infisical-side tenant principal. The organization identity is created with `no-access` by default and receives explicit project roles through the same project identity-membership API used by project-scoped identities. An organization-admin machine identity that creates an organization-bound project is granted permanent project-admin membership for that project, because Infisical does not create that membership automatically. The controller obtains the organization ID from an `InfisicalOrganization` status reference and verifies the organization ID of every project binding before reconciliation.

This design deliberately does not make the core controller aware of vCluster, Capsule, Flux, or Kyverno. A platform installation can issue the resulting machine identity’s credential to a tenant-specific operator deployment, while native Kubernetes RBAC and the deployment boundary decide who can create the CRs. A shared cluster-wide manager remains a trusted deployment and is not presented as tenant isolation.

The current OpenAPI document covers the public project, environment, identity, membership, Kubernetes Auth, Universal Auth, and organization-access paths used by the client. It still does not describe the organization lifecycle paths used by this project (`/api/v2/organizations` and the user-oriented `/api/v1/organization` endpoints). Those paths remain covered by their official endpoint pages and the client’s HTTP contract tests. The project does not check the entire upstream document; it records and verifies only the supported free-tier subset in the [contract manifest](../../hack/infisical-api-contract.json).

## SDK assessment

The official [Infisical Go SDK](https://github.com/Infisical/go-sdk) is maintained and mature enough for secret-oriented integrations, but its documented and exported packages focus on authentication, secret retrieval, folders, dynamic secrets, KMS, and SSH. It does not provide the project and identity administration operations required here. A small typed client keeps the unsupported surface explicit and makes request/response behavior testable without pulling in an unrelated SDK abstraction.

## Version pins

The project tracks current upstream releases identified during this research:

| Component | Pin | Source |
| --- | --- | --- |
| Go toolchain | `1.27.1` | [Go release history](https://go.dev/doc/devel/release) |
| Kubernetes Go modules | `v0.37.0` | [Kubernetes releases](https://kubernetes.io/releases/) |
| controller-runtime | `v0.25.0` | [controller-runtime releases](https://github.com/kubernetes-sigs/controller-runtime/releases) |
| controller-gen | `v0.22.0` | [controller-tools releases](https://github.com/kubernetes-sigs/controller-tools/releases) |
| Cilium | `1.20.1` | [Cilium releases](https://github.com/cilium/cilium/releases) |
| Infisical server image | `v0.165.8` | [Infisical releases](https://github.com/Infisical/infisical/releases/tag/v0.165.8) |
| Infisical standalone chart | `1.10.0` | [Infisical Helm repository](https://infisical.com/docs/self-hosting/deployment-options/kubernetes-helm) |
| Kustomize | `v5.8.1` | [Kustomize releases](https://github.com/kubernetes-sigs/kustomize/releases) |
| golangci-lint | `v2.13.2` | [golangci-lint releases](https://github.com/golangci/golangci-lint/releases) |

Kubernetes compatibility between the operator modules, Kind node image, and Cilium is an operational assumption worth rechecking when any of these pins changes.
