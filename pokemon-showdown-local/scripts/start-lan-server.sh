#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUNTIME_DIR="$ROOT_DIR/tmp/lan"
PID_FILE="$RUNTIME_DIR/server.pid"
LOG_FILE="$ROOT_DIR/logs/lan-server.log"
source "$ROOT_DIR/scripts/lib-lan-process.sh"

mkdir -p "$RUNTIME_DIR" "$ROOT_DIR/logs"

if PID="$(read_pid_file "$PID_FILE" 2>/dev/null)"; then
	if lan_marker_matches "$PID" "server" "$ROOT_DIR"; then
		echo "LAN server already running on PID $PID"
		exit 0
	fi
	rm -f "$PID_FILE"
fi

cd "$ROOT_DIR"

: "${PS_PORT:=8000}"
: "${PS_BIND_ADDRESS:=0.0.0.0}"
: "${PS_CLIENT_PORT:=8080}"

if [[ ! -f "$ROOT_DIR/dist/server/index.js" ]]; then
	echo "Building server bundle..."
	node build
fi

setsid bash -lc "
	cd \"$ROOT_DIR\"
	exec env \
		SHOWDOWN_SUITE_LAN_ROLE=server \
		SHOWDOWN_SUITE_ROOT=\"$ROOT_DIR\" \
		PS_PORT=\"$PS_PORT\" \
		PS_BIND_ADDRESS=\"$PS_BIND_ADDRESS\" \
		PS_CLIENT_PORT=\"$PS_CLIENT_PORT\" \
		./pokemon-showdown start --skip-build \"$PS_PORT\" \
		>>\"$LOG_FILE\" 2>&1
" >/dev/null 2>&1 &

echo $! > "$PID_FILE"

HOST_TO_CHECK="$(lan_check_host "$PS_BIND_ADDRESS")"
if ! wait_for_tcp "$HOST_TO_CHECK" "$PS_PORT" 20; then
	PID="$(cat "$PID_FILE")"
	if lan_marker_matches "$PID" "server" "$ROOT_DIR"; then
		PGID="$(lan_process_group "$PID")"
		if [[ -n "$PGID" ]]; then
			kill -- "-$PGID" 2>/dev/null || true
		fi
	fi
	rm -f "$PID_FILE"
	echo "LAN server failed to start. Check $LOG_FILE"
	exit 1
fi

echo "LAN server started on ${PS_BIND_ADDRESS}:${PS_PORT}"
