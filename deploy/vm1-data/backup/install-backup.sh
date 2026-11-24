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

CRON_PATH="PATH=/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
ENV_PREFIX=""
[ -n "${BACKUP_PASSPHRASE_FILE:-}" ] && ENV_PREFIX="BACKUP_PASSPHRASE_FILE=$BACKUP_PASSPHRASE_FILE "
[ -n "${BACKUP_RCLONE_REMOTE:-}" ] && ENV_PREFIX="${ENV_PREFIX}BACKUP_RCLONE_REMOTE=$BACKUP_RCLONE_REMOTE "
CRON_LINE="0 3 * * * ${ENV_PREFIX}$DEST_DIR/pg-backup.sh >> $HOME/ecommerce-data/backups/backup.log 2>&1"

# PATH goes first — cron's default PATH omits /usr/local/bin where docker lives.
# Then keep existing lines minus stale managed ones, and append the current entry.
{
  echo "$CRON_PATH"
  crontab -l 2>/dev/null | grep -vE '^PATH=|ecommerce-data/backup/pg-backup.sh' || true
  echo "$CRON_LINE"
} | crontab -

echo "Installed cron:"; crontab -l | grep pg-backup.sh
echo "Logs: $HOME/ecommerce-data/backups/backup.log"
echo "Run one now: $DEST_DIR/pg-backup.sh"
