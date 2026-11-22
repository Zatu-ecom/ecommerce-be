# Zatu-ecom/Config

Non-secret configuration for the Zatu e-commerce services, one file per service
per environment. **Secrets never live here** — they are injected at deploy time
from the environment / a secret store. Placeholder `__REQUIRED_SECRET__` marks a
value the deploy must supply.

## Layout

```text
envs/<env>/<service>.env     # dev, prod
required/<service>.keys      # required key names (incl. secret names)
scripts/validate.sh          # check an env file covers its key list
scripts/render-backend-env.sh# merge env file + secrets -> server .env
```

## Services

| File | Consumed by |
|---|---|
| `envs/<env>/backend.env` | Go backend (`ecommerce-be`) |
| `envs/<env>/storefront.env` | Vercel storefront (build-time `NEXT_PUBLIC_*`) |
| `envs/<env>/seller-dashboard.env` | Vercel seller dashboard (build-time `NEXT_PUBLIC_*`) |

## Secret names (values never committed)

`DB_PASSWORD`, `JWT_SECRET`, `ENCRYPTION_KEY`, `CACHE_PASSWORD`, `KV_PASSWORD`,
`RABBITMQ_PASSWORD` / `RABBITMQ_URL` (only when messaging is enabled).

## Quoting rules

- Plain `KEY=value`. Quote when the value has spaces, `#`, or `$`: `KEY='a b#c$d'`.
- `PUBLIC_API_BASE_URL` is origin-only, no trailing slash.
- `NEXT_PUBLIC_API_URL` **must include the `/api` prefix**.

## Commands

```bash
scripts/validate.sh dev                    # keys covered?
scripts/validate.sh prod
DB_PASSWORD=... JWT_SECRET=... ENCRYPTION_KEY=... \
  CACHE_PASSWORD=... scripts/render-backend-env.sh prod ~/ecommerce-app/.env
```

After rendering, the output `.env` must contain no `__REQUIRED_SECRET__` and be
mode `600`. Record the Config commit SHA in the deployment record.
