# 0003: Use a narrow typed Infisical client

Status: accepted

## Decision

Implement a small typed HTTP client for project, environment, project-role, project-identity, and Kubernetes Auth administration rather than depend on the official Go SDK.

## Rationale

The official SDK is a good fit for secret-oriented consumers but does not expose the entity administration operations required by this operator. A narrow client makes the supported API contract visible in Go types, limits accidental scope expansion, and supports deterministic HTTP contract tests. Revisit this decision if the SDK adds complete, maintained support for the operator’s project, environment, role, identity, and Kubernetes Auth surface.
