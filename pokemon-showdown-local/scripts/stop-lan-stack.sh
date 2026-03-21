#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUNTIME_DIR="$ROOT_DIR/tmp/lan"
source "$ROOT_DIR/scripts/lib-lan-process.sh"

stop_pid() {
	local name="$1"
	local pid_file="$2"
	local role="$3"

	if [[ ! -f "$pid_file" ]]; then
		echo "$name is not running"
		return
	fi

	local pid
	pid="$(cat "$pid_file")"
	if lan_marker_matches "$pid" "$role" "$ROOT_DIR"; then
		local pgid
		pgid="$(lan_process_group "$pid")"
		if [[ -n "$pgid" ]]; then
			kill -- "-$pgid"
			echo "Stopped $name process group (PGID $pgid)"
		else
			echo "$name exited before it could be signaled"
		fi
	else
		echo "$name pid file was stale"
	fi
	rm -f "$pid_file"
}

stop_pid "LAN client" "$RUNTIME_DIR/client.pid" "client"
stop_pid "LAN server" "$RUNTIME_DIR/server.pid" "server"
