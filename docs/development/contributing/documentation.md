# Documentation development

Markdown under `docs/` is the source of truth for the published site. `zensical.toml` defines navigation, repository links, and strict validation. The generated CRD reference at `docs/usage/reference/api.md` comes from the Go API definitions and `hack/crd-ref-docs.yaml`; edit the API comments and regenerate it instead of editing the generated file.

Build the site locally with the pinned container image:

```sh
make docs-build
```

Serve it while writing:

```sh
make docs-serve
```

The docs workflow runs on pull requests and pushes to `main`. Pull requests validate the generated reference and strict site build. A push to `main` uploads the `site/` artifact and deploys it to [GitHub Pages](https://winrarr.github.io/infisical-entity-operator/).

Keep credentials, kubeconfigs, generated tokens, private endpoints, and local-cluster state out of documentation. Link to source files or external references when the detail should remain operational rather than public site content.
