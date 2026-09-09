# Infisical API and dependency research

Research date: 2026-09-09.

## API contract

The client is based on the official Infisical API documentation:

- [Create project](https://infisical.com/docs/api-reference/endpoints/projects/create-project), [get project](https://infisical.com/docs/api-reference/endpoints/projects/get-project), [update project](https://infisical.com/docs/api-reference/endpoints/projects/update-project), and [delete project](https://infisical.com/docs/api-reference/endpoints/projects/delete-project).
- [Create identity](https://infisical.com/docs/api-reference/endpoints/project-identities/add-identity), [list identities](https://infisical.com/docs/api-reference/endpoints/project-identities/list-identity), [get identity](https://infisical.com/docs/api-reference/endpoints/project-identities/get-by-id), [update identity](https://infisical.com/docs/api-reference/endpoints/project-identities/update-identity), and [delete identity](https://infisical.com/docs/api-reference/endpoints/project-identities/delete-identity).
- [Environment endpoints](https://infisical.com/docs/api-reference/endpoints/environments/create) cover environment creation and lifecycle, including soft-delete restoration.
- [Project role creation](https://infisical.com/docs/api-reference/endpoints/project-roles/create) and [project permissions](https://infisical.com/docs/internals/permissions/project-permissions) define the typed role and condition surface.
- [Kubernetes Auth attach](https://infisical.com/docs/api-reference/endpoints/kubernetes-auth/attach) defines identity-scoped Kubernetes Auth configuration.
- [Automated bootstrapping](https://infisical.com/docs/self-hosting/guides/automated-bootstrapping) documents the in-cluster bootstrap Secret and its instance-admin token.

The public entity endpoints use `/api/v1/projects`, `/api/v1/projects/{projectId}/environments`, `/api/v1/projects/{projectId}/roles`, `/api/v1/projects/{projectId}/identities`, and `/api/v1/auth/kubernetes-auth/identities/{identityId}`. The checked-in client deliberately models only the fields and response envelopes needed by the CRDs. It sends bearer authentication, bounds response error bodies, classifies 404s, and applies a per-client timeout. Project-role action responses are accepted in both the string and array forms currently described by the API.

Infisical’s repository contains an OpenAPI fragment for platform project endpoints, but that fragment describes `/api/v1/platform/projects`, not the public project and project-identity endpoints above. It was therefore useful for comparison but not adopted as the client contract; the official endpoint pages are the authoritative source for this slice.

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
