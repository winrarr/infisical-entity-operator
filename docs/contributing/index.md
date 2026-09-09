# Development

The project uses a Makefile as its task entrypoint. It keeps Go generation, Helm rendering, documentation generation, and the Cilium-backed Kind environment in one discoverable workflow.

Before submitting a change, run:

```sh
make check
```

Read [local development](local-development.md) for the code and test loop, and [documentation](documentation.md) when changing the published site.
