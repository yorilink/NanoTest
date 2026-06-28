#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
RUN_DIR="${SCRIPT_DIR}/run"
OUTPUT="${OUTPUT:-${RUN_DIR}/gamecluster}"

mkdir -p "$(dirname "${OUTPUT}")"

cd "${REPO_ROOT}"

echo "building gamecluster binary: ${OUTPUT}"
go build -o "${OUTPUT}" ./examples/gamecluster
echo "build complete: ${OUTPUT}"
