#!/usr/bin/env bash
set -euo pipefail

release_tag="${1:-}"
chart_file="${2:-charts/infisical-entity-operator/Chart.yaml}"
semver_re='^[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+)?$'

if [[ -z "$release_tag" ]]; then
  echo "usage: $0 vX.Y.Z[-rc.N] [chart-file]" >&2
  exit 2
fi

case "$release_tag" in
  v*) version="${release_tag#v}" ;;
  *)
    echo "release tag must start with v: $release_tag" >&2
    exit 1
    ;;
esac

if [[ ! "$version" =~ $semver_re ]]; then
  echo "release tag must use vX.Y.Z or vX.Y.Z-rc.N: $release_tag" >&2
  exit 1
fi

if [[ ! -f "$chart_file" ]]; then
  echo "chart metadata file does not exist: $chart_file" >&2
  exit 1
fi

chart_version="$(awk '$1 == "version:" { value=$2; gsub(/^"/, "", value); gsub(/"$/, "", value); print value; exit }' "$chart_file")"
app_version="$(awk '$1 == "appVersion:" { value=$2; gsub(/^"/, "", value); gsub(/"$/, "", value); print value; exit }' "$chart_file")"

if [[ "$chart_version" != "$version" ]]; then
  echo "Chart.yaml version $chart_version does not match release tag $release_tag" >&2
  exit 1
fi

if [[ "$app_version" != "$version" ]]; then
  echo "Chart.yaml appVersion $app_version does not match release tag $release_tag" >&2
  exit 1
fi

echo "Release metadata is valid: $release_tag (chart and image version $version)"
