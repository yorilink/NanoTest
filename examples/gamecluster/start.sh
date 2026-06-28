#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
RUN_DIR="${SCRIPT_DIR}/run"
LOG_DIR="${SCRIPT_DIR}/logs"
BINARY="${RUN_DIR}/gamecluster"
MASTER_BINARY="${RUN_DIR}/masterserver"
GAME_BINARY="${RUN_DIR}/gameserver"
GATE_BINARY="${RUN_DIR}/gateserver"

MASTER_ADDR="${MASTER_ADDR:-127.0.0.1:34567}"
GATE_RPC_ADDR="${GATE_RPC_ADDR:-127.0.0.1:34570}"
GATE_CLIENT_ADDR="${GATE_CLIENT_ADDR:-127.0.0.1:34590}"
GAME_ADDR="${GAME_ADDR:-127.0.0.1:34680}"
REDIS_ADDR="${REDIS_ADDR:-127.0.0.1:6379}"

mkdir -p "${RUN_DIR}" "${LOG_DIR}"

cd "${REPO_ROOT}"

OUTPUT="${BINARY}" "${SCRIPT_DIR}/build.sh"

install_role_binary() {
	local target="$1"
	local tmp="${target}.tmp"

	cp "${BINARY}" "${tmp}"
	mv -f "${tmp}" "${target}"
}

install_role_binary "${MASTER_BINARY}"
install_role_binary "${GAME_BINARY}"
install_role_binary "${GATE_BINARY}"

is_running() {
	local pid="$1"
	kill -0 "${pid}" 2>/dev/null
}

start_process() {
	local name="$1"
	shift
	local pid_file="${RUN_DIR}/${name}.pid"
	local log_file="${LOG_DIR}/${name}.log"

	if [[ -f "${pid_file}" ]]; then
		local old_pid
		old_pid="$(cat "${pid_file}")"
		if [[ -n "${old_pid}" ]] && is_running "${old_pid}"; then
			echo "${name} already running, pid=${old_pid}"
			return
		fi
		rm -f "${pid_file}"
	fi

	echo "starting ${name}, log=${log_file}"
	nohup "$@" >>"${log_file}" 2>&1 &
	local pid="$!"
	echo "${pid}" >"${pid_file}"
	sleep 0.2
	if ! is_running "${pid}"; then
		echo "${name} failed to start, see ${log_file}"
		echo "last ${name} log lines:"
		tail -n 40 "${log_file}" || true
		rm -f "${pid_file}"
		return 1
	fi
	echo "${name} started, pid=${pid}"
}

start_process master "${MASTER_BINARY}" master \
	--listen "${MASTER_ADDR}"

sleep 1

start_process game "${GAME_BINARY}" game \
	--master "${MASTER_ADDR}" \
	--listen "${GAME_ADDR}" \
	--redis "${REDIS_ADDR}"

sleep 1

start_process gate "${GATE_BINARY}" gate \
	--master "${MASTER_ADDR}" \
	--listen "${GATE_RPC_ADDR}" \
	--gate-address "${GATE_CLIENT_ADDR}" \
	--redis "${REDIS_ADDR}"

cat <<EOF

gamecluster started
master: ${MASTER_ADDR}
game:   ${GAME_ADDR}
gate:   ${GATE_RPC_ADDR}
client: ws://${GATE_CLIENT_ADDR}/nano
redis:  ${REDIS_ADDR}

logs: ${LOG_DIR}
stop: ${SCRIPT_DIR}/stop.sh
EOF
