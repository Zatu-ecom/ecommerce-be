#!/usr/bin/env bash
# Render a server-side .env from Config + secrets (spec 016).
#
#   scripts/render-backend-env.sh <env> [output-path]
#
# Non-secret values come from envs/<env>/backend.env. Secret values come from
# the environment (exported by the deploy job / CI secrets), never from the repo.
# Fails before writing anything if a required secret is missing or a placeholder
# survives — a missing key must be a deploy blocker, not a runtime surprise.
set -euo pipefail

ENV_NAME="${1:?usage: render-backend-env.sh <env> [output]}"
OUT="${2:-.env}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$ROOT/envs/$ENV_NAME/backend.env"

[ -f "$SRC" ] || { echo "missing config file: $SRC" >&2; exit 1; }

REQUIRED_SECRETS=(DB_PASSWORD JWT_SECRET ENCRYPTION_KEY CACHE_PASSWORD KV_PASSWORD)
for k in "${REQUIRED_SECRETS[@]}"; do
  # KV_PASSWORD may fall back to CACHE_PASSWORD.
  if [ "$k" = "KV_PASSWORD" ] && [ -z "${KV_PASSWORD:-}" ]; then
    export KV_PASSWORD="${CACHE_PASSWORD:-}"
  fi
  [ -n "${!k:-}" ] || { echo "missing required secret: $k" >&2; exit 1; }
done

# The broker password is only required when messaging is enabled.
if grep -qE '^MESSAGING_ENABLED=true' "$SRC"; then
  [ -n "${RABBITMQ_PASSWORD:-}" ] || { echo "missing required secret: RABBITMQ_PASSWORD" >&2; exit 1; }
fi

tmp="$(mktemp)"; trap 'rm -f "$tmp"' EXIT; chmod 600 "$tmp"
cp "$SRC" "$tmp"

set_secret() {
  local key="$1" val="$2" esc
  esc="$(printf '%s' "$val" | sed -e 's/[\\&|]/\\&/g')"   # escape sed replacement chars
  sed -i "s|^${key}=.*|${key}='${esc}'|" "$tmp"
}

for k in "${REQUIRED_SECRETS[@]}"; do set_secret "$k" "${!k}"; done
# Broker password: real value when messaging is on, blanked (not a stale
# placeholder) when off, so no unresolved placeholder is ever emitted.
set_secret RABBITMQ_PASSWORD "${RABBITMQ_PASSWORD:-}"

if grep -q '__REQUIRED_SECRET__' "$tmp"; then
  echo "unresolved __REQUIRED_SECRET__ placeholder remains in $OUT" >&2
  exit 1
fi

install -m 600 "$tmp" "$OUT"
echo "rendered $OUT from $SRC (secrets injected)"
