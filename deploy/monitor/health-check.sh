#!/usr/bin/env bash
# Lightweight health check for the Oracle VMs (spec 014 / T048).
# Exits non-zero when any container is not running/healthy, disk is tight, or
# swap is exhausted. Run by cron; optionally POSTs to HEALTH_WEBHOOK on failure.
#
# Not a replacement for a metrics stack (Prometheus + node_exporter +
# cAdvisor) — that stays a future option; this is a "is prod okay" tripwire.
set -uo pipefail

# cron runs with a minimal PATH; the static Docker install lives in /usr/local/bin.
export PATH="/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:${PATH:-}"

DISK_MAX_PCT="${DISK_MAX_PCT:-85}"
HEALTH_WEBHOOK="${HEALTH_WEBHOOK:-}"
fail=0
detail=""
add() { detail="${detail}${detail:+; }$1"; fail=1; }

# 1) Every container must be running (and healthy where a healthcheck exists).
if command -v docker >/dev/null 2>&1; then
  while IFS= read -r name; do
    [ -z "$name" ] && continue
    state="$(docker inspect -f '{{.State.Status}}{{if .State.Health}}/{{.State.Health.Status}}{{end}}' "$name" 2>/dev/null)"
    case "$state" in
      running|running/healthy) : ;;
      *) add "container $name: $state" ;;
    esac
  done < <(docker ps -a --format '{{.Names}}')
else
  # Never report healthy without checking containers.
  add "docker not found on PATH"
fi

# 2) Root filesystem headroom.
use="$(df -P / | awk 'NR==2 {gsub("%","",$5); print $5}')"
[ "${use:-0}" -lt "$DISK_MAX_PCT" ] || add "disk / at ${use}%"

# 3) Swap must not be fully consumed (1 GB hosts lean on it).
swap_free="$(awk '/SwapFree/{print int($2/1024)}' /proc/meminfo)"
[ "${swap_free:-0}" -gt 50 ] || add "swap free ${swap_free}MB"

if [ "$fail" -eq 0 ]; then
  echo "health=ok"
  exit 0
fi

echo "health=DEGRADED: $detail"

# Optional alerting so a failure isn't just a line in a log file.
if [ -n "$HEALTH_WEBHOOK" ]; then
  if curl -fsS -m 5 -H 'Content-Type: application/json' \
       -d "{\"text\":\"$(hostname) health=DEGRADED: $detail\"}" "$HEALTH_WEBHOOK"; then
    echo "alert sent"
  else
    echo "WARN: webhook POST failed" >&2
  fi
fi
exit 1
