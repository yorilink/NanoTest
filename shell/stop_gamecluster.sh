#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

LOG_DIR="${LOG_DIR:-${ROOT_DIR}/logs/gamecluster}"
PID_DIR="${PID_DIR:-${LOG_DIR}}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-8}"

if [[ ! -d "${PID_DIR}" ]]; then
	echo "pid directory not found: ${PID_DIR}"
	exit 0
fi

stop_pid_file() {
	local pid_file="$1"
	local name
	name="$(basename "${pid_file}" .pid)"

	local pid
	pid="$(cat "${pid_file}" 2>/dev/null || true)"
	if [[ -z "${pid}" ]]; then
		echo "${name}: empty pid file, removing"
		rm -f "${pid_file}"
		return 0
	fi

	if ! kill -0 "${pid}" >/dev/null 2>&1; then
		echo "${name}: not running, removing stale pid ${pid}"
		rm -f "${pid_file}"
		return 0
	fi

	echo "stopping ${name}, pid=${pid}"
	kill "${pid}" >/dev/null 2>&1 || true

	local waited=0
	while kill -0 "${pid}" >/dev/null 2>&1; do
		if [[ "${waited}" -ge "${TIMEOUT_SECONDS}" ]]; then
			echo "${name}: still running after ${TIMEOUT_SECONDS}s, killing"
			kill -9 "${pid}" >/dev/null 2>&1 || true
			break
		fi
		sleep 1
		waited=$((waited + 1))
	done

	rm -f "${pid_file}"
}

stop_by_pattern() {
	local pattern="$1"
	local found=0
	for pid_file in "${PID_DIR}"/${pattern}; do
		if [[ -f "${pid_file}" ]]; then
			found=1
			stop_pid_file "${pid_file}"
		fi
	done
	return "${found}"
}

stop_by_pattern "gate.pid" || true
stop_by_pattern "game_*.pid" || true
stop_by_pattern "master.pid" || true
stop_by_pattern "client_http.pid" || true

for pid_file in "${PID_DIR}"/*.pid; do
	if [[ -f "${pid_file}" ]]; then
		stop_pid_file "${pid_file}"
	fi
done

echo "gamecluster stopped"
