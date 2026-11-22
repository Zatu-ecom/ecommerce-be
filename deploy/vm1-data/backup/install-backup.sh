#!/usr/bin/env bash
# Install the nightly pg_dump cron on VM-1 (spec 014 / T045).
# Idempotent: re-running replaces the managed cron line, never duplicates it.
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
DEST_DIR="$HOME/ecommerce-data/backup"
mkdir -p "$DEST_DIR"

if [ "$HERE/pg-backup.sh" != "$DEST_DIR/pg-backup.sh" ]; then
  install -m 700 "$HERE/pg-backup.sh" "$DEST_DIR/pg-backup.sh"
else
  chmod 700 "$DEST_DIR/pg-backup.sh"
fi

CRON_LINE="0 3 * * * $DEST_DIR/pg-backup.sh >> $HOME/ecommerce-data/backups/backup.log 2>&1"

# Drop any previous managed line, then append the current one.
{ crontab -l 2>/dev/null | grep -v 'ecommerce-data/backup/pg-backup.sh' || true; echo "$CRON_LINE"; } | crontab -

echo "Installed cron:"; crontab -l | grep pg-backup.sh
echo "Logs: $HOME/ecommerce-data/backups/backup.log"
echo "Run one now: $DEST_DIR/pg-backup.sh"
