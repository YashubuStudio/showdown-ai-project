#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

: "${PS_PORT:=8000}"
: "${PS_CLIENT_PORT:=8080}"

"$ROOT_DIR/scripts/start-lan-server.sh"
"$ROOT_DIR/scripts/start-lan-client.sh"

LAN_IP="$(
	ip route get 1.1.1.1 2>/dev/null | awk '
		{
			for (i = 1; i <= NF; i++) {
				if ($i == "src") {
					print $(i + 1)
					exit
				}
			}
		}
	'
)"

if [[ -z "$LAN_IP" ]]; then
	LAN_IP="localhost"
fi

echo
echo "Landing page:"
echo "  http://${LAN_IP}:${PS_PORT}"
echo
echo "Direct client:"
echo "  http://${LAN_IP}:${PS_CLIENT_PORT}/play.pokemonshowdown.com/testclient.html?~~${LAN_IP}:${PS_PORT}"
