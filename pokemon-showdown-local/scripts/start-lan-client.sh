#!/usr/bin/env bash
set -euo pipefail

SERVER_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLIENT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../pokemon-showdown-client-local" && pwd)"
RUNTIME_DIR="$SERVER_ROOT/tmp/lan"
PID_FILE="$RUNTIME_DIR/client.pid"
LOG_FILE="$SERVER_ROOT/logs/lan-client.log"
source "$SERVER_ROOT/scripts/lib-lan-process.sh"

mkdir -p "$RUNTIME_DIR" "$SERVER_ROOT/logs"

if PID="$(read_pid_file "$PID_FILE" 2>/dev/null)"; then
	if lan_marker_matches "$PID" "client" "$SERVER_ROOT"; then
		echo "LAN client already running on PID $PID"
		exit 0
	fi
	rm -f "$PID_FILE"
fi

if ! command -v python3 >/dev/null 2>&1; then
	echo "python3 is required to serve the local client"
	exit 1
fi

cd "$CLIENT_ROOT"

: "${PS_CLIENT_PORT:=8080}"
: "${PS_CLIENT_BIND_ADDRESS:=0.0.0.0}"

setsid bash -lc "
	cd \"$CLIENT_ROOT\"
	exec env \
		SHOWDOWN_SUITE_LAN_ROLE=client \
		SHOWDOWN_SUITE_ROOT=\"$SERVER_ROOT\" \
		python3 -m http.server \"$PS_CLIENT_PORT\" \
		--bind \"$PS_CLIENT_BIND_ADDRESS\" \
		>>\"$LOG_FILE\" 2>&1
" >/dev/null 2>&1 &

echo $! > "$PID_FILE"

HOST_TO_CHECK="$(lan_check_host "$PS_CLIENT_BIND_ADDRESS")"
if ! wait_for_tcp "$HOST_TO_CHECK" "$PS_CLIENT_PORT" 20; then
	PID="$(cat "$PID_FILE")"
	if lan_marker_matches "$PID" "client" "$SERVER_ROOT"; then
		PGID="$(lan_process_group "$PID")"
		if [[ -n "$PGID" ]]; then
			kill -- "-$PGID" 2>/dev/null || true
		fi
	fi
	rm -f "$PID_FILE"
	echo "LAN client failed to start. Check $LOG_FILE"
	exit 1
fi

echo "LAN client started on ${PS_CLIENT_BIND_ADDRESS}:${PS_CLIENT_PORT}"
