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

It creates a named Kind cluster, installs Cilium and the pinned Infisical standalone chart, builds and deploys the operator, then exercises reconciliation and egress policy behavior. The [local Kind operations](../operations/local-kind.md) guide covers prerequisites, overrides, cleanup, and known plan limitations.

Do not commit `bin/`, `dist/`, `tmp/`, coverage output, kubeconfigs, local bootstrap credentials, or Infisical tokens.
