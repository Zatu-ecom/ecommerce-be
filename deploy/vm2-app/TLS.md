# TLS for `api.dev.ecom.vitrapay.io` (spec 014 / T052)

**Status: live on VM-2.** Let's Encrypt certificate issued 2026-11-26, expires
2027-02-24, renewed daily by cron. The only remaining step is the OCI ingress
rule for port 443.

## What is in place

- DNS `A api.dev.ecom` -> `130.210.26.64` (Hostinger).
- nginx serves HTTP **and** HTTPS (`deploy/vm2-app/nginx-ecommerce.conf`).
- certbot is a one-shot compose service (`--profile tools`): webroot
  `~/ecommerce-app/certbot/www`, certs `~/ecommerce-app/certbot/conf`.
- Renewal: daily cron on VM-2 at 04:15 -> `~/ecommerce-app/renew-certs.sh`
  (certbot renew is a no-op until ~30 days before expiry).
- firewalld allows http/https; compose publishes 80 and 443.

## Remaining steps

1. **operator** - OCI console -> Networking -> VCN -> Subnet -> Security List ->
   Add Ingress Rule: source `0.0.0.0/0`, IP protocol TCP, destination port `443`.
2. Verify: `curl -sS -o /dev/null -w "%{http_code}\n" https://api.dev.ecom.vitrapay.io/` -> `400`.
3. Point the app at HTTPS: set in `Zatu-ecom/Config -> dev/backend.env`
   `PUBLIC_API_BASE_URL=https://api.dev.ecom.vitrapay.io`, then re-run the
   deploy workflow.
4. Optional once 1-3 are verified: replace the HTTP `location /` block with
   `return 301 https://$host$request_uri;` so HTTP redirects to HTTPS.

## Fresh host / re-issue

nginx refuses to start if the 443 block references a certificate that does not
exist yet, so the rollout is two-stage:

1. Install the bootstrap variant as the live config (HTTP + ACME challenge only):

       cp nginx-ecommerce-bootstrap.conf nginx-ecommerce.conf
       docker compose up -d nginx

2. Issue the certificate:

       docker compose --profile tools run --rm certbot certonly \
         --webroot -w /var/www/certbot -d api.dev.ecom.vitrapay.io \
         --agree-tos --register-unsafely-without-email --non-interactive

3. Install the TLS variant as the live config and reload:

       cp nginx-ecommerce.conf.tls nginx-ecommerce.conf
       docker compose exec nginx nginx -t && docker compose exec nginx nginx -s reload

   (In this repo `nginx-ecommerce.conf` *is* the TLS variant; keep the bootstrap
   file alongside it so a rebuild can repeat the order above.)

## Why not one config

A `listen 443 ssl` block with missing `fullchain.pem` aborts nginx startup. Two
files keep first-time issuance working without hand-editing.
