#!/usr/bin/env bash
# Validate envs/<env>/<service>.env against required/<service>.keys.
# Usage: scripts/validate.sh <env> [service]     (service default: backend)
set -euo pipefail

ENV_NAME="${1:?usage: validate.sh <env> [service]}"
SERVICE="${2:-backend}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FILE="$ROOT/envs/$ENV_NAME/$SERVICE.env"
KEYS="$ROOT/required/$SERVICE.keys"

[ -f "$FILE" ] || { echo "missing config file: $FILE" >&2; exit 1; }
[ -f "$KEYS" ] || { echo "missing key list: $KEYS" >&2; exit 1; }

fail=0
while IFS= read -r key; do
  case "$key" in ''|\#*) continue ;; esac
  if ! grep -qE "^${key}=" "$FILE"; then
    echo "MISSING key: $key (in $FILE)" >&2
    fail=1
  fi
done < "$KEYS"

if [ "$fail" -ne 0 ]; then
  echo "validation FAILED: $FILE" >&2
  exit 1
fi
echo "ok: $FILE has all $(grep -cvE '^$|^#' "$KEYS") required keys"
