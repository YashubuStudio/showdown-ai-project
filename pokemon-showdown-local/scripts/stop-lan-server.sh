#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PID_FILE="$ROOT_DIR/tmp/lan/server.pid"
source "$ROOT_DIR/scripts/lib-lan-process.sh"

if [[ ! -f "$PID_FILE" ]]; then
	echo "LAN server is not running"
	exit 0
fi

PID="$(cat "$PID_FILE")"
if lan_marker_matches "$PID" "server" "$ROOT_DIR"; then
	PGID="$(lan_process_group "$PID")"
	if [[ -n "$PGID" ]]; then
		kill -- "-$PGID"
		echo "Stopped LAN server process group (PGID $PGID)"
	else
		echo "LAN server exited before it could be signaled"
	fi
else
	echo "LAN server pid file was stale"
fi
rm -f "$PID_FILE"
