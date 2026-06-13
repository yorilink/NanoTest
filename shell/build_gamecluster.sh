#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

OUTPUT="${OUTPUT:-${ROOT_DIR}/bin/gamecluster}"
GOCACHE="${GOCACHE:-/tmp/nano-go-build-cache}"
DEBUG_BUILD="${DEBUG_BUILD:-1}"

mkdir -p "$(dirname "${OUTPUT}")" "${GOCACHE}"

cd "${ROOT_DIR}"

echo "building examples/gamecluster -> ${OUTPUT}"
build_args=()
if [[ "${DEBUG_BUILD}" != "0" ]]; then
	build_args+=("-gcflags" "all=-N -l")
	echo "debug build enabled: -gcflags 'all=-N -l'"
fi

GOCACHE="${GOCACHE}" go build "${build_args[@]}" -o "${OUTPUT}" ./examples/gamecluster
echo "build ok: ${OUTPUT}"
