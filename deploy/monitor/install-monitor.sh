#!/usr/bin/env bash
# Install the health-check cron on a host (spec 014 / T048).
# Usage: ./install-monitor.sh          # every 5 minutes
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
DEST_DIR="$HOME/monitor"
mkdir -p "$DEST_DIR"
if [ "$HERE/health-check.sh" != "$DEST_DIR/health-check.sh" ]; then
  install -m 700 "$HERE/health-check.sh" "$DEST_DIR/health-check.sh"
else
  chmod 700 "$DEST_DIR/health-check.sh"
fi

CRON_PATH="PATH=/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
CRON_LINE="*/5 * * * * $DEST_DIR/health-check.sh >> $HOME/monitor/health.log 2>&1"

# PATH goes first — cron's default PATH omits /usr/local/bin where docker lives.
{
  echo "$CRON_PATH"
  crontab -l 2>/dev/null | grep -vE '^PATH=|monitor/health-check.sh' || true
  echo "$CRON_LINE"
} | crontab -

echo "Installed cron:"; crontab -l | grep health-check.sh
echo "Run one now: $DEST_DIR/health-check.sh"
