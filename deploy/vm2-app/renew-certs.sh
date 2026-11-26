#!/usr/bin/env bash
# Renew the api.dev.ecom.vitrapay.io certificate and reload nginx (idempotent).
# Install as a daily cron on VM-2:
#   15 4 * * * /home/opc/ecommerce-app/renew-certs.sh >> /home/opc/ecommerce-app/renew.log 2>&1
set -euo pipefail

# cron runs with a minimal PATH; the static Docker install lives in /usr/local/bin.
export PATH="/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:${PATH:-}"

cd "$HOME/ecommerce-app"

# certbot renew is a no-op until ~30 days before expiry.
docker compose --profile tools run --rm certbot renew --quiet --webroot -w /var/www/certbot < /dev/null

# Reload so nginx picks up a renewed cert (harmless if nothing changed).
docker compose exec -T nginx nginx -s reload < /dev/null
echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] renew check done"
