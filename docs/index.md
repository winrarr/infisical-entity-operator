---
title: Overview
---

# Infisical Entity Operator

Kubernetes-native lifecycle management for Infisical projects, environments, project roles, machine identities, and Kubernetes Auth.

Choose the documentation path that matches your task. The [README](https://github.com/winrarr/infisical-entity-operator/blob/main/README.md) remains the concise project summary, and the [source repository](https://github.com/winrarr/infisical-entity-operator) contains the chart, examples, tests, and implementation.

The operator reconciles a dependency-aware resource graph:

```text
InfisicalConnection → InfisicalProject → InfisicalEnvironment
                                      ├→ InfisicalProjectRole
                                      └→ InfisicalIdentity → InfisicalKubernetesAuth
```

Secret synchronization is intentionally outside this project. Use Infisical’s official Kubernetes operator for workloads that need secrets materialized into Kubernetes Secrets.

## Using the Operator

- [Usage overview](usage/index.md): install, configure, and use the operator.
- [Quickstart](usage/quickstart.md): create a connection and project.
- [Custom resource guides](usage/crds/index.md): configure each of the six CRDs.
- [Reference](usage/reference/index.md): lifecycle, security, troubleshooting, status, and generated API details.
- [Examples](usage/examples/index.md): complete credential-free manifests.

## Developing the Project

- [Development overview](development/index.md): the canonical code and documentation workflow.
- [Architecture](development/architecture.md): controllers, client boundary, ownership, reconciliation, and security model.
- [Decisions](development/decisions/index.md): accepted choices that future changes should respect or supersede.
- [Local Kind operations](development/operations/local-kind.md): Cilium, Infisical, e2e, troubleshooting, and cleanup.
- [Project records](development/project/index.md): product scope, verification, backlog, and tech debt.
- [Infisical API and dependency research](development/research/2026-09-09-infisical-api-and-versions.md): dated external evidence and version pins.

The repository Markdown is the documentation source of truth. The [published documentation site](https://winrarr.github.io/infisical-entity-operator/) is built from these files on every change to the documentation, API definitions, or documentation workflow.
