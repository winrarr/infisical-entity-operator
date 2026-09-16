# Local development

Use the repository Makefile for the canonical loop:

```sh
make manifests generate
make test
make lint-config lint helm-lint
make build
```

The disposable integration environment is:

```sh
make kind-e2e
make kind-down
```

It creates a named Kind cluster with the default CNI and the pinned Infisical standalone chart, builds and deploys the operator from committed artifacts, then exercises reconciliation with the standard egress policy manifest. Independent setup tasks are parallelized by the E2E target. Use `make kind-up KIND_CNI=cilium` when a local scenario needs Cilium. The [local Kind operations](../operations/local-kind.md) guide covers prerequisites, mode switching, cleanup, and known plan limitations.

Do not commit `bin/`, `dist/`, `tmp/`, coverage output, kubeconfigs, local bootstrap credentials, or Infisical tokens.
