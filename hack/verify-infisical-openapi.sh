#!/usr/bin/env bash
set -euo pipefail

openapi_url="${INFISICAL_OPENAPI_URL:-https://app.infisical.com/api/docs/json}"
contract_file="${INFISICAL_API_CONTRACT_FILE:-hack/infisical-api-contract.json}"
spec_file="$(mktemp "${TMPDIR:-/tmp}/infisical-openapi.XXXXXX.json")"
trap 'rm -f "$spec_file"' EXIT

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required to verify the Infisical OpenAPI contract" >&2
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required to verify the Infisical OpenAPI contract" >&2
  exit 1
fi
if [[ ! -f "$contract_file" ]]; then
  echo "Infisical API contract manifest not found: $contract_file" >&2
  exit 1
fi

curl --fail --silent --show-error --location \
  --retry 3 --retry-delay 1 --connect-timeout 10 --max-time 60 \
  "$openapi_url" --output "$spec_file"

if ! jq -e '(.openapi | type == "string" and startswith("3.")) and (.paths | type == "object")' "$spec_file" >/dev/null; then
  echo "Infisical OpenAPI document is not a supported OpenAPI 3 document: $openapi_url" >&2
  exit 1
fi
if ! jq -e '(.formatVersion == 1) and (.operations | type == "array" and length > 0)' "$contract_file" >/dev/null; then
  echo "Infisical API contract manifest is invalid: $contract_file" >&2
  exit 1
fi

issues="$(jq -r --slurpfile contract "$contract_file" '
  def schema_properties:
    ((. // {}) | (.properties // {}) + ((.anyOf // []) | map(.properties // {}) | add // {}));
  . as $spec |
  $contract[0].operations[] as $expected |
  ($spec.paths[$expected.path][($expected.method | ascii_downcase)] // null) as $operation |
  if $operation == null then
    "missing operation: \($expected.method) \($expected.path)"
  elif ($operation.responses["200"] // null) == null then
    "missing 200 response: \($expected.method) \($expected.path)"
  else
    (
      [
        $expected.requestProperties[]? as $property
        | select(((($operation.requestBody.content["application/json"].schema // {}) | schema_properties | has($property)) | not))
        | "missing request property \($property): \($expected.method) \($expected.path)"
      ] +
      [
        $expected.requiredRequestProperties[]? as $property
        | select(((($operation.requestBody.content["application/json"].schema.required // []) | index($property)) | . == null))
        | "request property is no longer required: \($property): \($expected.method) \($expected.path)"
      ] +
      [
        $expected.responseProperties[]? as $property
        | select(((($operation.responses["200"].content["application/json"].schema // {}) | schema_properties | has($property)) | not))
        | "missing response property \($property): \($expected.method) \($expected.path)"
      ]
    )[]
  end
' "$spec_file")"

if [[ -n "$issues" ]]; then
  echo "Infisical OpenAPI contract mismatch ($openapi_url):" >&2
  printf ' - %s\n' "$issues" >&2
  exit 1
fi

operation_count="$(jq '.operations | length' "$contract_file")"
echo "Infisical OpenAPI contract verified: ${operation_count} operations from $openapi_url"
