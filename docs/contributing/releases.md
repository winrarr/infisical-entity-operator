# Releases

Releases are paired operator and Helm chart artifacts published to GitHub Container Registry. The publication interface is one annotated `vX.Y.Z` or `vX.Y.Z-rc.N` tag; chart-only releases are intentionally not supported.

The tag version must match both `version` and `appVersion` in `charts/infisical-entity-operator/Chart.yaml`. A release publishes:

- `ghcr.io/winrarr/infisical-entity-operator:X.Y.Z` and `:vX.Y.Z` for `linux/amd64` and `linux/arm64`;
- `oci://ghcr.io/winrarr/charts/infisical-entity-operator:X.Y.Z`;
- a GitHub Release with the packaged chart and generated release notes.

The chart defaults its image tag to `.Chart.AppVersion`, so installing a released chart selects the matching released image. Local or development images can still be selected with `--set image.tag=dev` or another explicit tag.

GHCR package visibility is managed separately from the workflow. Before the first public release, set the operator image and Helm chart packages to public in the repository’s Packages settings so users can install them without registry credentials.

## Create a release

Update `version` and `appVersion` together in `Chart.yaml`, then open and merge the change through the normal pull request checks. From the resulting commit on `main`, create and push an annotated tag:

```sh
git switch main
git pull --ff-only
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

The metadata check can also be run locally with `make validate-release RELEASE_TAG=v0.1.0`.

The `Publish Release` workflow validates the tag and chart metadata, confirms that the tagged commit is contained in `main`, verifies generated artifacts and the chart, builds the multi-platform image, packages and publishes the chart, and creates the GitHub Release.

Install a released chart with:

```sh
helm upgrade --install infisical-entity-operator \
  oci://ghcr.io/winrarr/charts/infisical-entity-operator \
  --version 0.1.0 \
  --namespace infisical-entity-operator-system \
  --create-namespace \
  --wait
```

## Recovery and dry runs

If publication fails after the tag exists, rerun the failed workflow. The workflow also supports manual recovery with an existing tag:

```sh
gh workflow run publish-release.yaml -f tag_name=v0.1.0
```

Validate the tag, generated artifacts, image build, and chart package without publishing anything:

```sh
gh workflow run publish-release.yaml -f tag_name=v0.1.0 -f dry_run=true
```

Do not move or recreate a release tag. If a chart version already exists in GHCR, the publisher accepts it only when its `appVersion` matches the release version; otherwise publication fails for review.
