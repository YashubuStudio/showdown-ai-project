#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUNTIME_DIR="$ROOT_DIR/tmp/lan"
source "$ROOT_DIR/scripts/lib-lan-process.sh"

show_status() {
	local name="$1"
	local pid_file="$2"
	local role="$3"
	local pid=""

	if pid="$(read_pid_file "$pid_file" 2>/dev/null)" && lan_marker_matches "$pid" "$role" "$ROOT_DIR"; then
		echo "$name: running (PID $pid)"
	elif pid="$(lan_find_pid_by_marker "$role" "$ROOT_DIR" 2>/dev/null)"; then
		echo "$pid" > "$pid_file"
		echo "$name: running (PID $pid)"
	else
		echo "$name: stopped"
	fi
}

show_status "LAN server" "$RUNTIME_DIR/server.pid" "server"
show_status "LAN client" "$RUNTIME_DIR/client.pid" "client"
