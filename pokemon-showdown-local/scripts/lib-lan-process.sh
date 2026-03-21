#!/usr/bin/env bash
set -euo pipefail

read_pid_file() {
	local pid_file="$1"
	if [[ ! -f "$pid_file" ]]; then
		return 1
	fi
	cat "$pid_file"
}

lan_marker_matches() {
	local pid="$1"
	local role="$2"
	local root_dir="$3"

	if [[ ! "$pid" =~ ^[0-9]+$ ]]; then
		return 1
	fi
	if ! kill -0 "$pid" 2>/dev/null; then
		return 1
	fi
	if [[ ! -r "/proc/$pid/environ" ]]; then
		return 1
	fi

	local environ
	environ="$(tr '\0' '\n' < "/proc/$pid/environ")"
	grep -Fxq "SHOWDOWN_SUITE_LAN_ROLE=$role" <<<"$environ" || return 1
	grep -Fxq "SHOWDOWN_SUITE_ROOT=$root_dir" <<<"$environ" || return 1
	return 0
}

lan_process_group() {
	local pid="$1"
	ps -o pgid= -p "$pid" 2>/dev/null | tr -d '[:space:]'
}

lan_check_host() {
	local bind_address="$1"
	case "$bind_address" in
	""|"0.0.0.0"|"::")
		echo "127.0.0.1"
		;;
	*)
		echo "$bind_address"
		;;
	esac
}

wait_for_tcp() {
	local host="$1"
	local port="$2"
	local timeout_seconds="${3:-15}"
	local deadline=$((SECONDS + timeout_seconds))

	while (( SECONDS < deadline )); do
		if (echo >"/dev/tcp/$host/$port") >/dev/null 2>&1; then
			return 0
		fi
		sleep 1
	done
	return 1
}
