# Developing the Project

This section is for contributors and maintainers. The project uses a Makefile as its task entrypoint for Go generation, Helm rendering, documentation generation, and the Cilium-backed Kind environment.

Before submitting a change, run:

```sh
make check
```

Read [local development](contributing/local-development.md) for the code and test loop, [documentation](contributing/documentation.md) when changing the published site, and [architecture](architecture.md) before changing reconciliation or client behavior.

- [Architecture and decisions](architecture/index.md) explain component boundaries and the rationale behind consequential choices.
- [Contributing](contributing/local-development.md) covers local code, tests, and documentation workflows.
- [Operations](operations/index.md) covers the disposable Cilium-backed Kind environment.
- [Project records](project/index.md) collect product scope, verification evidence, backlog, and tech debt.
- [Research](research/2026-09-09-infisical-api-and-versions.md) records dated external evidence and dependency choices.
