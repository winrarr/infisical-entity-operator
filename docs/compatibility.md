# Infisical API compatibility

This project supports a deliberately narrow Infisical control-plane API surface. It does not promise compatibility with every Infisical API tag or every self-hosted release.

## Compatibility policy

Infisical versions its public API per resource, rather than assigning one version to the whole API. The client therefore pins each endpoint path explicitly, currently using `v1` for projects, environments, roles, identities, memberships, project templates, identity templates, and authentication methods, and `v2` for organization membership and workspace access. A resource may also use different route shapes within the same API version; for example, project-role reads use `/v1/projects/roles/{roleId}`, while project-role updates and deletes remain under `/v1/projects/{projectId}/roles/{roleId}`.

The supported profiles are:

| Profile | Support statement |
| --- | --- |
| Infisical Cloud | The current public API contract exposed by the official OpenAPI document at the time of verification. |
| Self-hosted Infisical | The repository’s tested baseline, server image `v0.165.8` with standalone chart `1.10.0`. Other versions are compatibility candidates, not a blanket semver promise. |

The baseline does not mean that every feature is available on every plan. Plan-gated behavior, such as custom project roles, remains an acceptance limitation documented in the [tech-debt register](tech-debt.md).

## Contract sources

| Concern | Source of truth |
| --- | --- |
| Endpoint existence, HTTP method, request fields, and documented response envelope | The official [Infisical OpenAPI document](https://app.infisical.com/api/docs/json), checked by `make verify-infisical-api`. The reviewed subset is recorded in [`hack/infisical-api-contract.json`](../hack/infisical-api-contract.json). |
| Endpoint semantics, authentication, plan behavior, and lifecycle details | The relevant official [API reference](https://infisical.com/docs/api-reference/overview/introduction) and endpoint documentation. |
| Response normalization, alternate deployed shapes, and error behavior | The deterministic HTTP contract tests in [`internal/infisicalclient/client_test.go`](../internal/infisicalclient/client_test.go). |
| Cross-resource behavior against a server | The local Kind verification described in [verification](verification.md), with the tested server baseline from `Makefile`. |

The current OpenAPI document does not describe the organization lifecycle endpoints used by this project (`/api/v2/organizations` and the user-oriented `/api/v1/organization` endpoints). Those paths are consequently governed by their official endpoint documentation and HTTP contract tests rather than by the OpenAPI check. The machine-token organization boundary checks through `/api/v2/organizations/{organizationId}/memberships` and `/workspaces` are included in the manifest.

## Verification workflow

Run the live schema check with:

```sh
make verify-infisical-api
```

The URL can be overridden for a compatible Infisical deployment:

```sh
make verify-infisical-api INFISICAL_OPENAPI_URL=https://infisical.example.com/api/docs/json
```

The check fails when a recorded operation disappears, loses its `200` response, or no longer contains a recorded request or response property. It intentionally does not fail for unrelated additions to Infisical’s much larger OpenAPI document.

When an upstream change is intentional, update the typed client, HTTP contract tests, manifest, and this policy together. Do not update the manifest merely to make a failing check pass; first establish whether the change is backward-compatible, requires a new endpoint version, or should remain unsupported.
