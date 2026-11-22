# TLS rollout for `api.vitrapay.io` (spec 014 / T052)

Today the API is HTTP-only on `http://130.210.26.64`. HTTPS is a hard
dependency for spec 015 (an HTTPS frontend cannot call an HTTP API).

## Prerequisites

- DNS: add `A  api  →  130.210.26.64` at the `vitrapay.io` registrar/zone.
  Confirm it resolves before running certbot (use the reserved public IP).
- VM-2 only. VM-1 needs no public name.

## Steps (VM-2, certbot + nginx)

```bash
# 0. DNS check from the laptop
dig +short api.vitrapay.io          # must print 130.210.26.64

# 1. On VM-2, point nginx at the name and open :443
#    Edit ~/ecommerce-app/nginx-ecommerce.conf: server_name _;  ->  server_name api.vitrapay.io;
sed -i 's/server_name _;/server_name api.vitrapay.io;/' ~/ecommerce-app/nginx-ecommerce.conf

# 2. Add a certbot service (one-shot) to ~/ecommerce-app/docker-compose.yml:
#      certbot:
#        image: certbot/certbot
#        volumes:
#          - ./certbot/conf:/etc/letsencrypt
#          - ./certbot/www:/var/www/certbot
#    and mount the same two dirs into nginx as read-only.
#
# 3. Issue the certificate (standalone webroot), then reload nginx:
docker compose run --rm certbot certonly --webroot -w /var/www/certbot \
  -d api.vitrapay.io --email <you>@vitrapay.io --agree-tos --no-eff-email
docker compose exec nginx nginx -s reload

# 4. Switch the backend's public URL so webhook links use HTTPS:
#    ~/ecommerce-app/.env  ->  PUBLIC_API_BASE_URL=https://api.vitrapay.io
docker compose up -d app

# 5. Verify
curl -sS https://api.vitrapay.io/ -o /dev/null -w '%{http_code}\n'   # expect 400
```

## Auto-renewal

Add a daily cron on VM-2:

```cron
0 4 * * * cd /home/opc/ecommerce-app && docker compose run --rm certbot renew --quiet && docker compose exec nginx nginx -s reload
```

Certs renew at ~30 days remaining; the command is a no-op otherwise.

## After TLS (spec 015)

- Update backend CORS allowlist to the real origins (`CORS_ALLOWED_ORIGINS`).
- Set `NEXT_PUBLIC_API_URL=https://api.vitrapay.io/api` in both Vercel projects.

> Do not open `:443` in the OCI security list until certbot has a cert, and
> never expose `:8080`.
