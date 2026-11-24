#!/usr/bin/env bash
# LEGACY bootstrap path — use only for the very first setup.
#
# The normal path is the GitHub Actions workflow (.github/workflows/deploy-oracle.yml),
# which renders ~/ecommerce-app/.env from Zatu-ecom/Config + repo secrets and ships
# it. Anything written here is overwritten by the next workflow run.
#
# Run from anywhere on your laptop:
#   ./deploy/setup-secrets.sh
#
# It asks for your 4 secret values WITHOUT echoing them, builds the .env files
# for VM-1 (data) and VM-2 (app), copies them over SSH, and wipes the local
# temporary copies. Secrets never appear on screen, in shell history, or in git.
set -euo pipefail

VM1="opc@140.238.240.2"
VM2="opc@130.210.26.64"
KEY1="${HOME}/.ssh/oci_vm1"
KEY2="${HOME}/.ssh/oci_vm2"
DEFAULT_PUBLIC_URL="http://130.210.26.64"

SSH_OPTS=(-o BatchMode=yes -o ConnectTimeout=15 -o StrictHostKeyChecking=accept-new)

say()  { printf '%s\n' "$*"; }
fail() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

[ -f "$KEY1" ] || fail "SSH key not found: $KEY1"
[ -f "$KEY2" ] || fail "SSH key not found: $KEY2"

# ---- hidden input -----------------------------------------------------------
ask_secret() {
  local prompt="$1" value=""
  while [ -z "$value" ]; do
    printf '%s: ' "$prompt" >&2
    if [ -r /dev/tty ]; then
      IFS= read -rs value < /dev/tty
    else
      IFS= read -rs value
    fi
    printf '\n' >&2
    [ -n "$value" ] || say "  (cannot be empty, try again)" >&2
  done
  printf '%s' "$value"
}

# ---- sanity checks -----------------------------------------------------------
say "Checking SSH access..."
ssh "${SSH_OPTS[@]}" -i "$KEY1" "$VM1" 'true' || fail "cannot SSH to VM-1 ($VM1)"
ssh "${SSH_OPTS[@]}" -i "$KEY2" "$VM2" 'true' || fail "cannot SSH to VM-2 ($VM2)"

VM1_PRIVATE_IP=$(ssh "${SSH_OPTS[@]}" -i "$KEY1" "$VM1" \
  "ip -4 -o addr show ens3 | awk '{print \$4}' | cut -d/ -f1")
[ -n "$VM1_PRIVATE_IP" ] || fail "could not detect VM-1 private IP"
say "VM-1 private IP: $VM1_PRIVATE_IP"
say ""

say "Paste each value when asked (input is hidden). Values are NOT saved anywhere except the VMs."
DB_PASSWORD=$(ask_secret "DB_PASSWORD")
CACHE_PASSWORD=$(ask_secret "CACHE_PASSWORD (also used as KV_PASSWORD)")
JWT_SECRET=$(ask_secret "JWT_SECRET")
ENCRYPTION_KEY=$(ask_secret "ENCRYPTION_KEY")

[ "${#JWT_SECRET}" -ge 32 ] || say "WARNING: JWT_SECRET is shorter than 32 chars." >&2
[ "${#ENCRYPTION_KEY}" -ge 32 ] || say "WARNING: ENCRYPTION_KEY is shorter than 32 chars." >&2

# PUBLIC_API_BASE_URL: default to the current IP-only endpoint.
printf 'PUBLIC_API_BASE_URL [%s]: ' "$DEFAULT_PUBLIC_URL" >&2
PUBLIC_URL=""
if [ -r /dev/tty ]; then
  IFS= read -r PUBLIC_URL < /dev/tty
else
  IFS= read -r PUBLIC_URL
fi
PUBLIC_URL="${PUBLIC_URL:-$DEFAULT_PUBLIC_URL}"

# ---- build files in a private temp dir ---------------------------------------
TMPD=$(mktemp -d)
chmod 700 "$TMPD"
trap 'rm -rf "$TMPD"' EXIT
umask 077

cat > "$TMPD/vm1.env" <<EOF
# VM-1 (data): Postgres + durable Redis. Generated $(date -u +%Y-%m-%dT%H:%M:%SZ)
# DB_HOST is the compose service name (migrations run inside the network).
DB_HOST=postgres
DB_PORT=5432
DB_NAME=ecommerce
DB_USER=postgres
DB_PASSWORD='${DB_PASSWORD}'
DB_SSLMODE=disable

CACHE_PASSWORD='${CACHE_PASSWORD}'
KV_MAXMEMORY=128mb
EOF

cat > "$TMPD/vm2.env" <<EOF
# VM-2 (app): Go backend + volatile Redis + nginx. Generated $(date -u +%Y-%m-%dT%H:%M:%SZ)
APP_ENV=dev
GIN_MODE=release
PORT=8080
LOG_LEVEL=info

# Postgres on VM-1 (private IP)
DB_HOST=${VM1_PRIVATE_IP}
DB_PORT=5432
DB_NAME=ecommerce
DB_USER=postgres
DB_PASSWORD='${DB_PASSWORD}'
DB_SSLMODE=disable
DB_MAX_OPEN_CONNS=20
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME_MINUTES=30
DB_CONN_MAX_IDLE_TIME_MINUTES=5

# Auth
JWT_SECRET='${JWT_SECRET}'
JWT_EXPIRY_HOURS=24

# Volatile cache (this VM, compose-internal)
CACHE_HOST=cache-volatile
CACHE_PORT=6379
CACHE_PASSWORD='${CACHE_PASSWORD}'
CACHE_MAXMEMORY=64mb
CACHE_ENABLED=true

# Durable KV on VM-1 (private IP)
KV_HOST=${VM1_PRIVATE_IP}
KV_PORT=6381
KV_PASSWORD='${CACHE_PASSWORD}'

# App
PUBLIC_API_BASE_URL=${PUBLIC_URL}
ENCRYPTION_KEY='${ENCRYPTION_KEY}'
WORKER_POOL_SIZE=5

# Messaging off (no RabbitMQ)
MESSAGING_ENABLED=false
EOF

# ---- copy ---------------------------------------------------------------------
say ""
say "Copying .env files to the VMs..."
scp "${SSH_OPTS[@]}" -q -i "$KEY1" "$TMPD/vm1.env" "$VM1:~/ecommerce-data/.env"
scp "${SSH_OPTS[@]}" -q -i "$KEY2" "$TMPD/vm2.env" "$VM2:~/ecommerce-app/.env"

ssh "${SSH_OPTS[@]}" -i "$KEY1" "$VM1" 'chmod 600 ~/ecommerce-data/.env'
ssh "${SSH_OPTS[@]}" -i "$KEY2" "$VM2" 'chmod 600 ~/ecommerce-app/.env'

say "Done. Both .env files are in place (mode 600)."
say "Tell the agent: \"secrets done\"."
