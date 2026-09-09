# Infisical Entity Operator

Kubernetes-native lifecycle management for Infisical projects, environments, project roles, machine identities, and Kubernetes Auth.

Start with the [quickstart](quickstart.md), or read the [README](https://github.com/winrarr/infisical-entity-operator/blob/main/README.md) for the concise product summary. The [source repository](https://github.com/winrarr/infisical-entity-operator) contains the chart, examples, tests, and implementation.

The operator reconciles a dependency-aware resource graph:

```text
InfisicalConnection → InfisicalProject → InfisicalEnvironment
                                      ├→ InfisicalProjectRole
                                      └→ InfisicalIdentity → InfisicalKubernetesAuth
```

Secret synchronization is intentionally outside this project. Use Infisical’s official Kubernetes operator for workloads that need secrets materialized into Kubernetes Secrets.

## Product and design

- [Product scope](product.md): supported outcome, safety boundaries, and non-goals.
- [Architecture](architecture.md): controllers, client boundary, ownership, reconciliation, and security model.
- [Decision records](decisions/index.md): accepted choices that future changes should respect or supersede.
- [Infisical API and dependency research](research/2026-09-09-infisical-api-and-versions.md): dated external evidence and version pins.

## Development and operations

- [Verification guide](verification.md): canonical checks, evidence coverage, and known limitations.
- [Local Kind operations](operations/local-kind.md): Cilium, Infisical, e2e, troubleshooting, and cleanup.
- [Backlog](backlog.md): planned outcomes that are not implemented.
- [Tech-debt register](tech-debt.md): known current shortcomings and exit criteria.

The repository Markdown is the documentation source of truth. The [published documentation site](https://winrarr.github.io/infisical-entity-operator/) is built from these files on every change to the documentation, API definitions, or documentation workflow.
