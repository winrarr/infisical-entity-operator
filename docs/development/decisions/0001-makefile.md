# 0001: Choose Make

Status: accepted

## Decision

Use a Makefile as the project task runner.

## Rationale

The project is a Kubebuilder Go operator and its ecosystem already uses Make targets for generators, manifests, tests, linting, image builds, chart deployment, and Kind workflows. Make also composes dependency targets and exposes variable overrides naturally in CI and local development. A Justfile would be pleasant for simple recipes, but would add a second convention to a workflow whose upstream tooling and operator examples are Make-oriented.
