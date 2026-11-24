#!/usr/bin/env bash
# Nightly logical backup of the production database (spec 014 / T045).
# Runs ON VM-1. Dumps through the running postgres container so no psql is
# needed on the host, gzips the result, optionally encrypts it, optionally
# copies it off-host, and enforces a retention window.
#
# Install:  ./install-backup.sh        (adds the cron entry)
# Test:     ./pg-backup.sh             (writes one dump immediately)
# Restore:  see ./RESTORE.md
#
# Optional hardening (set in the cron environment or ~/.bashrc):
#   BACKUP_PASSPHRASE_FILE=~/ecommerce-data/backup/passphrase   # encrypts dumps
#   BACKUP_RCLONE_REMOTE=oci:ecommerce-backups                 # off-host copy
set -euo pipefail

# cron runs with a minimal PATH; the static Docker install lives in /usr/local/bin.
export PATH="/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:${PATH:-}"

CONTAINER="${PG_CONTAINER:-ecommerce-postgres}"
BACKUP_DIR="${BACKUP_DIR:-$HOME/ecommerce-data/backups}"
RETAIN_DAYS="${RETAIN_DAYS:-7}"
PASSPHRASE_FILE="${BACKUP_PASSPHRASE_FILE:-}"
RCLONE_REMOTE="${BACKUP_RCLONE_REMOTE:-}"

mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"

ts="$(date -u +%Y%m%d-%H%M%S)"
base="$BACKUP_DIR/ecommerce-$ts.sql.gz"

echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] dumping $CONTAINER -> $base"

# POSTGRES_USER / POSTGRES_DB live in the container env; local socket is trusted.
# -6 not -9: same ballpark size, much less CPU on a 1 GB host.
if ! docker exec "$CONTAINER" sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --no-owner --no-privileges' \
     | gzip -6 > "$base"; then
  echo "ERROR: pg_dump failed" >&2
  rm -f "$base"
  exit 1
fi

if [ ! -s "$base" ]; then
  echo "ERROR: dump is empty, removing $base" >&2
  rm -f "$base"
  exit 1
fi
gzip -t "$base"

out="$base"
if [ -n "$PASSPHRASE_FILE" ]; then
  [ -r "$PASSPHRASE_FILE" ] || { echo "ERROR: $PASSPHRASE_FILE not readable" >&2; exit 1; }
  echo "    encrypting with $PASSPHRASE_FILE"
  openssl enc -aes-256-cbc -pbkdf2 -salt -md sha256 \
    -pass "file:$PASSPHRASE_FILE" -in "$base" -out "$base.enc"
  rm -f "$base"   # never keep the plaintext copy
  out="$base.enc"
fi
echo "ok: $out ($(du -h "$out" | cut -f1))"

# Off-host copy. Without this the backups share the boot volume with the DB,
# so a volume loss takes both. See deploy/ORACLE-SETUP-TODO.md.
if [ -n "$RCLONE_REMOTE" ]; then
  if command -v rclone >/dev/null 2>&1; then
    rclone copy "$out" "$RCLONE_REMOTE" && echo "uploaded to $RCLONE_REMOTE"
  else
    echo "WARN: BACKUP_RCLONE_REMOTE is set but rclone is not installed" >&2
  fi
else
  echo "WARN: backup is LOCAL-ONLY (same disk as the DB); set BACKUP_RCLONE_REMOTE for an off-host copy" >&2
fi

# Retention: keep the newest N dumps, delete the rest (.sql.gz and .sql.gz.enc).
ls -1t "$BACKUP_DIR"/ecommerce-*.sql.gz* 2>/dev/null | tail -n "+$((RETAIN_DAYS + 1))" | xargs -r rm -f
