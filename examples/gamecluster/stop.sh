#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUN_DIR="${SCRIPT_DIR}/run"
BINARY="${RUN_DIR}/gamecluster"

is_running() {
	local pid="$1"
	kill -0 "${pid}" 2>/dev/null
}

binary_for_name() {
	local name="$1"
	case "${name}" in
	master)
		printf "%s/masterserver" "${RUN_DIR}"
		;;
	game)
		printf "%s/gameserver" "${RUN_DIR}"
		;;
	gate)
		printf "%s/gateserver" "${RUN_DIR}"
		;;
	*)
		printf "%s" "${BINARY}"
		;;
	esac
}

find_pids() {
	local name="$1"
	local binary
	binary="$(binary_for_name "${name}")"
	binary="$(readlink -f "${binary}" 2>/dev/null || printf "%s" "${binary}")"
	ps -eo pid=,args= | awk -v binary="${binary}" -v name="${name}" '
		index($0, binary " " name " ") > 0 || $0 ~ binary " " name "$" {
			print $1
		}
	'
}

stop_pid() {
	local name="$1"
	local pid="$2"

	if ! is_running "${pid}"; then
		echo "${name} not running, stale pid=${pid}"
		return
	fi

	echo "stopping ${name}, pid=${pid}"
	kill "${pid}" 2>/dev/null || true

	for _ in $(seq 1 20); do
		if ! is_running "${pid}"; then
			echo "${name} stopped"
			return
		fi
		sleep 0.2
	done

	echo "${name} did not stop in time, killing pid=${pid}"
	kill -9 "${pid}" 2>/dev/null || true
	echo "${name} stopped"
}

stop_process() {
	local name="$1"
	local pid_file="${RUN_DIR}/${name}.pid"

	if [[ ! -f "${pid_file}" ]]; then
		local pids
		pids="$(find_pids "${name}")"
		if [[ -z "${pids}" ]]; then
			echo "${name} not running, pid file missing"
			return
		fi
		echo "${name} pid file missing, found running process by command line"
		for pid in ${pids}; do
			stop_pid "${name}" "${pid}"
		done
		return
	fi

	local pid
	pid="$(cat "${pid_file}")"
	if [[ -z "${pid}" ]]; then
		echo "${name} pid file is empty"
		rm -f "${pid_file}"
		return
	fi

	stop_pid "${name}" "${pid}"
	rm -f "${pid_file}"
}

stop_process gate
stop_process game
stop_process master
