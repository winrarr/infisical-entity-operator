# Development

The project uses a Makefile as its task entrypoint. It keeps Go generation, Helm rendering, documentation generation, and the default-CNI Kind environment in one discoverable workflow, while retaining an explicit Cilium setup target for scenarios that need it.

Before submitting a change, run:

```sh
make check
```

Read [local development](local-development.md) for the code and test loop, and [documentation](documentation.md) when changing the published site.
