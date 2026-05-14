#!/usr/bin/env bash
# Regenerate the Trello Go SDK from the upstream OpenAPI spec.
#
# Usage: scripts/generate.sh
#
# The script:
#   1. Downloads the latest Trello OpenAPI v3 spec to api/trello-swagger.json
#   2. Runs api/preprocess_spec.py to patch a handful of upstream bugs
#   3. Invokes oapi-codegen to regenerate trello/trello.gen.go
#
# Requirements: bash, curl, python3, go (>= 1.22).
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SPEC_URL="${TRELLO_SPEC_URL:-https://dac-static.atlassian.com/cloud/trello/swagger.v3.json?_v=1.957.0}"
SPEC_FILE="${REPO_ROOT}/api/trello-swagger.json"
CFG_FILE="${REPO_ROOT}/api/oapi-codegen.yaml"
OAPI_VERSION="${OAPI_CODEGEN_VERSION:-v2.4.1}"

echo ">> downloading Trello spec from ${SPEC_URL}"
curl -fsSL -o "${SPEC_FILE}" "${SPEC_URL}"

echo ">> preprocessing spec"
python3 "${REPO_ROOT}/api/preprocess_spec.py" "${SPEC_FILE}"

echo ">> ensuring oapi-codegen ${OAPI_VERSION} is installed"
GOBIN="$(mktemp -d)"
export GOBIN
go install "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@${OAPI_VERSION}"

echo ">> generating trello/trello.gen.go"
"${GOBIN}/oapi-codegen" --config="${CFG_FILE}" "${SPEC_FILE}"

echo ">> running go mod tidy"
( cd "${REPO_ROOT}" && go mod tidy )

echo ">> done"
