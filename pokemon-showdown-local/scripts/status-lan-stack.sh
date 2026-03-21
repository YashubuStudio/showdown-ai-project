#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUNTIME_DIR="$ROOT_DIR/tmp/lan"
source "$ROOT_DIR/scripts/lib-lan-process.sh"

show_status() {
	local name="$1"
	local pid_file="$2"
	local role="$3"

	if PID="$(read_pid_file "$pid_file" 2>/dev/null)" && lan_marker_matches "$PID" "$role" "$ROOT_DIR"; then
		echo "$name: running (PID $PID)"
	else
		echo "$name: stopped"
	fi
}

show_status "LAN server" "$RUNTIME_DIR/server.pid" "server"
show_status "LAN client" "$RUNTIME_DIR/client.pid" "client"
