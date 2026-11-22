#!/usr/bin/env bash
# Nightly logical backup of the production database (spec 014 / T045).
# Runs ON VM-1. Dumps through the running postgres container so no psql is
# needed on the host, gzips the result, and enforces a retention window.
#
# Install:  ./install-backup.sh        (adds the cron entry)
# Test:     ./pg-backup.sh             (writes one dump immediately)
# Restore:  see ./RESTORE.md
set -euo pipefail

CONTAINER="${PG_CONTAINER:-ecommerce-postgres}"
BACKUP_DIR="${BACKUP_DIR:-$HOME/ecommerce-data/backups}"
RETAIN_DAYS="${RETAIN_DAYS:-7}"

mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"

ts="$(date -u +%Y%m%d-%H%M%S)"
out="$BACKUP_DIR/ecommerce-$ts.sql.gz"

echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] dumping $CONTAINER -> $out"

# POSTGRES_USER / POSTGRES_DB live in the container env; local socket is trusted.
if ! docker exec "$CONTAINER" sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --no-owner --no-privileges' \
     | gzip -9 > "$out"; then
  echo "ERROR: pg_dump failed" >&2
  rm -f "$out"
  exit 1
fi

# Never keep an empty/truncated dump.
if [ ! -s "$out" ]; then
  echo "ERROR: dump is empty, removing $out" >&2
  rm -f "$out"
  exit 1
fi

# Record a checksum so a corrupt archive is detectable at restore time.
gzip -t "$out"
echo "ok: $out ($(du -h "$out" | cut -f1))"

# Retention: keep the newest N dumps, delete the rest.
ls -1t "$BACKUP_DIR"/ecommerce-*.sql.gz 2>/dev/null | tail -n "+$((RETAIN_DAYS + 1))" | xargs -r rm -f
