# 0003: Use a narrow typed Infisical client

Status: accepted

## Decision

Implement a small typed HTTP client for project and project-identity administration rather than depend on the official Go SDK.

## Rationale

The official SDK is a good fit for secret-oriented consumers but does not expose the entity administration operations required by this operator. A narrow client makes the supported API contract visible in Go types, limits accidental scope expansion, and supports deterministic HTTP contract tests. Revisit this decision if the SDK adds complete, maintained project and identity management support.
