#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
	echo "Usage: $0 <username>"
	exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
USERGROUPS_FILE="$ROOT_DIR/config/usergroups.csv"
USERNAME="$1"

# Validate username: alphanumeric (and spaces/hyphens), max 18 chars, no commas or newlines.
if [[ ${#USERNAME} -gt 18 ]]; then
	echo "Error: username must be 18 characters or fewer"
	exit 1
fi
if [[ "$USERNAME" =~ [,] ]] || [[ "$USERNAME" =~ $'\n' ]]; then
	echo "Error: username must not contain commas or newlines"
	exit 1
fi

mkdir -p "$ROOT_DIR/config"

TMP_FILE="$(mktemp "$ROOT_DIR/config/usergroups.XXXXXX")"

if [[ -f "$USERGROUPS_FILE" ]]; then
	awk -F, -v username="$USERNAME" '
		BEGIN {
			OFS = ",";
			found = 0;
		}
		tolower($1) == tolower(username) {
			print username, "~";
			found = 1;
			next;
		}
		{ print }
		END {
			if (!found) print username, "~";
		}
	' "$USERGROUPS_FILE" > "$TMP_FILE"
else
	echo "${USERNAME},~" > "$TMP_FILE"
fi

mv "$TMP_FILE" "$USERGROUPS_FILE"

echo "Granted admin to ${USERNAME} in $USERGROUPS_FILE"
