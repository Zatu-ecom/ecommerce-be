#!/usr/bin/env bash
# Lightweight health check for the Oracle VMs (spec 014 / T048).
# Exits non-zero when any container is not running/healthy, disk is tight, or
# swap is exhausted. Intended to be run by cron; wire a mail/webhook if wanted.
#
# Not a replacement for a metrics stack (Prometheus + node_exporter +
# cAdvisor) — that stays a future option; this is a "is prod okay" tripwire.
set -uo pipefail

# cron runs with a minimal PATH; the static Docker install lives in /usr/local/bin.
export PATH="/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:${PATH:-}"

DISK_MAX_PCT="${DISK_MAX_PCT:-85}"
fail=0

# 1) Every container must be running (and healthy where a healthcheck exists).
if command -v docker >/dev/null 2>&1; then
  while IFS= read -r name; do
    [ -z "$name" ] && continue
    state="$(docker inspect -f '{{.State.Status}}{{if .State.Health}}/{{.State.Health.Status}}{{end}}' "$name" 2>/dev/null)"
    case "$state" in
      running|running/healthy) : ;;
      *) echo "CONTAINER $name: $state"; fail=1 ;;
    esac
  done < <(docker ps -a --format '{{.Names}}')
else
  # Never report healthy without checking containers.
  echo "docker not found on PATH"
  fail=1
fi

# 2) Root filesystem headroom.
use="$(df -P / | awk 'NR==2 {gsub("%","",$5); print $5}')"
[ "${use:-0}" -lt "$DISK_MAX_PCT" ] || { echo "DISK / at ${use}%"; fail=1; }

# 3) Swap must not be fully consumed (1 GB hosts lean on it).
swap_free="$(awk '/SwapFree/{print int($2/1024)}' /proc/meminfo)"
[ "${swap_free:-0}" -gt 50 ] || { echo "SWAP free ${swap_free}MB"; fail=1; }

if [ "$fail" -eq 0 ]; then
  echo "health=ok"
else
  echo "health=DEGRADED"
fi
exit "$fail"
