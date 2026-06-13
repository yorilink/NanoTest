#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

OUTPUT="${OUTPUT:-${ROOT_DIR}/bin/gamecluster}"
GOCACHE="${GOCACHE:-/tmp/nano-go-build-cache}"

mkdir -p "$(dirname "${OUTPUT}")" "${GOCACHE}"

cd "${ROOT_DIR}"

echo "building examples/gamecluster -> ${OUTPUT}"
GOCACHE="${GOCACHE}" go build -o "${OUTPUT}" ./examples/gamecluster
echo "build ok: ${OUTPUT}"
