#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

BIN="${BIN:-${ROOT_DIR}/bin/gamecluster}"
LOG_DIR="${LOG_DIR:-${ROOT_DIR}/logs/gamecluster}"
PID_DIR="${PID_DIR:-${LOG_DIR}}"

MASTER_ADDR="${MASTER_ADDR:-127.0.0.1:34567}"
GATE_RPC_ADDR="${GATE_RPC_ADDR:-127.0.0.1:34570}"
GATE_WS_ADDR="${GATE_WS_ADDR:-127.0.0.1:34590}"
REDIS_ADDR="${REDIS_ADDR:-127.0.0.1:6379}"
GAME_ADDRS="${GAME_ADDRS:-127.0.0.1:34680 127.0.0.1:34681}"
CLIENT_HTTP_ADDR="${CLIENT_HTTP_ADDR:-0.0.0.0:8080}"

mkdir -p "${LOG_DIR}" "${PID_DIR}"

redis_host="${REDIS_ADDR%:*}"
redis_port="${REDIS_ADDR##*:}"
if [[ -z "${redis_host}" || -z "${redis_port}" || "${redis_host}" == "${redis_port}" ]]; then
	echo "invalid REDIS_ADDR: ${REDIS_ADDR}" >&2
	exit 1
fi

if ! (echo >"/dev/tcp/${redis_host}/${redis_port}") >/dev/null 2>&1; then
	echo "redis is not reachable at ${REDIS_ADDR}" >&2
	echo "start redis first, or run with REDIS_ADDR=host:port" >&2
	exit 1
fi

"${SCRIPT_DIR}/build_gamecluster.sh"

if ! command -v python3 >/dev/null 2>&1; then
	echo "python3 is required to serve client files" >&2
	exit 1
fi

sanitize_name() {
	echo "$1" | tr '.:' '__'
}

start_proc() {
	local name="$1"
	shift

	local pid_file="${PID_DIR}/${name}.pid"
	local log_file="${LOG_DIR}/${name}.log"

	if [[ -f "${pid_file}" ]]; then
		local old_pid
		old_pid="$(cat "${pid_file}")"
		if [[ -n "${old_pid}" ]] && kill -0 "${old_pid}" >/dev/null 2>&1; then
			echo "${name} already running, pid=${old_pid}" >&2
			echo "log: ${log_file}" >&2
			return 0
		fi
		rm -f "${pid_file}"
	fi

	echo "starting ${name}"
	nohup "$@" >"${log_file}" 2>&1 &
	local pid="$!"
	echo "${pid}" >"${pid_file}"
	echo "  pid: ${pid}"
	echo "  log: ${log_file}"
}

start_proc "master" "${BIN}" master --listen "${MASTER_ADDR}"
sleep 1

for game_addr in ${GAME_ADDRS}; do
	name="game_$(sanitize_name "${game_addr}")"
	start_proc "${name}" "${BIN}" game --master "${MASTER_ADDR}" --listen "${game_addr}" --redis "${REDIS_ADDR}"
done
sleep 1

start_proc "gate" "${BIN}" gate \
	--master "${MASTER_ADDR}" \
	--listen "${GATE_RPC_ADDR}" \
	--gate-address "${GATE_WS_ADDR}" \
	--redis "${REDIS_ADDR}"

client_http_host="${CLIENT_HTTP_ADDR%:*}"
client_http_port="${CLIENT_HTTP_ADDR##*:}"
start_proc "client_http" python3 -m http.server "${client_http_port}" --bind "${client_http_host}" --directory "${ROOT_DIR}"

echo
echo "gamecluster started"
echo "client: http://${CLIENT_HTTP_ADDR}/client/"
echo "websocket: ws://${GATE_WS_ADDR}/nano"
echo "logs: ${LOG_DIR}"
echo "pids: ${PID_DIR}"
echo "stop: ${SCRIPT_DIR}/stop_gamecluster.sh"
