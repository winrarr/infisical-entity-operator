# 0004: Keep project-role permissions typed

Status: accepted

Date: 2026-09-09

## Decision

Represent `InfisicalProjectRole` permissions as a structural Kubernetes API with explicit subjects, actions, and condition fields. Model actions as a list in the CRD and accept both string and list forms when decoding Infisical responses.

## Rationale

Permission rules are part of the operator’s desired state and should be reviewable, diffable, and validated as Kubernetes data. Arbitrary JSON would make malformed rules and unsafe future expansion easy to hide. Infisical responses observed during implementation use more than one action representation, so the client preserves compatibility at the API boundary while exposing one stable CRD shape.

## Constraints

- Keep the model limited to project-role permissions; organization users, groups, and identity membership remain separate scope.
- Add stricter subject/action/operator validation only when it is backed by the supported Infisical API contract.
- Treat malformed or unsupported rules as reconciliation errors until admission validation can reject them earlier.

## Consequences

The CRD is easier to review and can evolve with explicit fields, but the current free-form strings leave a known validation gap recorded in [TD-003](../project/tech-debt.md#td-003-tighten-project-role-permission-validation). The client’s response normalization is intentionally part of the compatibility boundary rather than a second user-facing schema.
