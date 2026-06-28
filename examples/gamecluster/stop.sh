#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUN_DIR="${SCRIPT_DIR}/run"

is_running() {
	local pid="$1"
	kill -0 "${pid}" 2>/dev/null
}

stop_process() {
	local name="$1"
	local pid_file="${RUN_DIR}/${name}.pid"

	if [[ ! -f "${pid_file}" ]]; then
		echo "${name} not running, pid file missing"
		return
	fi

	local pid
	pid="$(cat "${pid_file}")"
	if [[ -z "${pid}" ]]; then
		echo "${name} pid file is empty"
		rm -f "${pid_file}"
		return
	fi

	if ! is_running "${pid}"; then
		echo "${name} not running, stale pid=${pid}"
		rm -f "${pid_file}"
		return
	fi

	echo "stopping ${name}, pid=${pid}"
	kill "${pid}" 2>/dev/null || true

	for _ in $(seq 1 20); do
		if ! is_running "${pid}"; then
			rm -f "${pid_file}"
			echo "${name} stopped"
			return
		fi
		sleep 0.2
	done

	echo "${name} did not stop in time, killing pid=${pid}"
	kill -9 "${pid}" 2>/dev/null || true
	rm -f "${pid_file}"
	echo "${name} stopped"
}

stop_process gate
stop_process game
stop_process master
