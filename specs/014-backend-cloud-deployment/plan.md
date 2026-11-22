# Implementation Plan: Backend Cloud Deployment (Oracle Cloud, Phase 1)

**Branch**: `014-backend-cloud-deployment` | **Created**: 2026-10-13 | **Updated**: 2026-11-22 | **Spec**: [spec.md](./spec.md)
**Status**: **Implemented — as-built plan** (backend live at `http://130.210.26.64`)

## Summary

Deploy the `ecommerce-be` monolith and its runtime dependencies on Oracle Cloud Always Free using two `VM.Standard.E2.1.Micro` hosts split by role: **VM-1 data** (PostgreSQL 16 + durable Redis) and **VM-2 app** (Go backend + volatile Redis + nginx edge). Both hosts run Docker Engine 29.7.2 installed from static tarballs (no `dnf` transactions — package operations OOM-storm on 1 GB Micros), with 2 GB additional swap and OS background package scans disabled. The database/cache plane is reachable only over the VCN private subnet (`10.0.0.0/24`), enforced twice (OCI security list + firewalld). The application image is built on the operator's `amd64` laptop and shipped to VM-2 via `docker save | gzip | ssh | docker load`; deploys are manual. The `vitrapay.io` domain is owned and the backend target is `api.vitrapay.io`; TLS is pending the DNS record + certbot.

## Technical Context

**Language/Version**: no new Go code in this feature (deployment/config only); app built with the repo's Go 1.25 toolchain
**Primary Dependencies**: Docker 29.7.2 (static tarball), Docker Compose v5.7.1 plugin, `postgres:16-alpine`, `redis:7-alpine` ×2, `nginx:1.27-alpine`, `ecommerce-backend:latest` (built from `Dockerfile`)
**Storage**: VM-1 named volumes `ecommerce-postgres-data`, `ecommerce-kv-data`; VM-2 has no persistent Docker volume (cache is restart-losable by design)
**Host OS**: Oracle Linux Server 9.8 (UEK), firewalld active, SELinux per image defaults
**Host sizes**: VM-2 ~945 MB RAM / 2 vCPU; VM-1 reports ~498 MB RAM / 2 vCPU (observed post-reboot) + 2.99 GB total swap on both
**Network**: one VCN/subnet `10.0.0.0/24`; VM-1 `10.0.0.208` / `140.238.240.2`; VM-2 `10.0.0.88` / `130.210.26.64`; public IPs reserved
**Testing**: external port scan, container health checks, `pg_isready`, curl smoke tests, `pg_tables`/`schema_migration` queries
**Target Platform**: Linux `amd64` Docker hosts
**Project Type**: deployment/ops (no application module changes)
**Constraints**: ~1 GB RAM/host; no `dnf`; manual image delivery; domain decided (`api.vitrapay.io`), TLS pending; no CI/CD
**Scale/Scope**: single production environment, single host per role (no HA)

## Constitution Check

This feature adds no Go module code, migrations, or API contracts. It does not change layer boundaries, DI, seller isolation, correlation-ID behavior, or RBAC. Deployment artifacts live under `deploy/`; the only app-adjacent change is the compose-level healthcheck (TCP probe) that replaces the image's unusable `/` HTTP probe. **No violations; no Complexity Tracking entries.**

## Architecture (as-built)

```
                        INTERNET
                           │  :80 (public)
                    ┌──────▼──────┐
                    │   nginx     │  VM-2 130.210.26.64 (reserved)
                    │  1.27-alpine│  proxy_pass http://app:8080
                    └──────┬──────┘
                           │ compose network
                    ┌──────▼──────┐        ┌────────────────┐
                    │  Go app     │───────►│ cache-volatile │  redis:7, 64mb LRU
                    │  :8080      │        └────────────────┘  (no volume)
                    └──────┬──────┘
                           │ private subnet 10.0.0.0/24
                    ┌──────▼──────────────────────────────┐
                    │  VM-1 10.0.0.208 / 140.238.240.2    │
                    │  ┌───────────┐  ┌─────────────────┐ │
                    │  │ postgres  │  │ cache-durable   │ │
                    │  │ 16-alpine │  │ redis:7, 6381   │ │
                    │  └───────────┘  └─────────────────┘ │
                    │  volumes: postgres-data, kv-data     │
                    └──────────────────────────────────────┘
```

**Port matrix**

| Port | VM-1 | VM-2 | Enforcement |
|---|---|---|---|
| 22 | open (key-only) | open (key-only) | OCI security list; world-open by operator choice (deferred hardening) |
| 80 | closed | **open** | OCI `0.0.0.0/0:80` + firewalld `http` |
| 5432 | subnet-only | — | OCI `10.0.0.0/24:5432` + firewalld rich rule |
| 6381 | subnet-only | — | OCI `10.0.0.0/24:6381` + firewalld rich rule |
| 6379 / 6380 / 8080 | closed | closed | never exposed; app publishes `127.0.0.1:8080`, nginx is the only edge |

## Components & Files

| File | Purpose |
|---|---|
| `deploy/bootstrap.sh` | Per-host bootstrap: static Docker + compose plugin, 2 GB `/swapfile2`, docker group, firewalld rules per role, disables `dnf-makecache.timer` |
| `deploy/ship-image.sh` | Builds `linux/amd64` image on the laptop and loads it on VM-2 over SSH |
| `deploy/setup-secrets.sh` | Hidden-input prompt for the 4 secrets; writes both `.env` files and copies them (mode 600) |
| `deploy/vm1-data/docker-compose.yml` | Postgres (tuned) + durable Redis + `migrate` tools profile |
| `deploy/vm2-app/docker-compose.yml` | App + volatile Redis + nginx; TCP healthcheck for the app |
| `deploy/vm2-app/nginx-ecommerce.conf` | Port 80 edge, `proxy_pass http://app:8080`, 100 MB uploads, gzip, forwarded headers |
| `deploy/.env.example` | Env contract template for VM-2 (placeholders only) |
| `deploy/ORACLE-SETUP-TODO.md` | Operator runbook: exact console steps, current deployment record, master checklist |
| `specs/014-backend-cloud-deployment/` | This spec, plan, tasks |

## Configuration & Secrets

- Secrets live **only** in `~/ecommerce-data/.env` (VM-1) and `~/ecommerce-app/.env` (VM-2), mode `600`, owned by `opc`; both verified placeholder-free.
- VM-1 `.env` keys: `DB_HOST=postgres` (compose service for the migration runner), `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `CACHE_PASSWORD`, `KV_MAXMEMORY`.
- VM-2 `.env` keys: `APP_ENV=prod`, `GIN_MODE=release`, `PORT`, `LOG_LEVEL`, `DB_*` (host = VM-1 private IP), `JWT_*`, `CACHE_*`, `KV_*` (host = VM-1 private IP), `PUBLIC_API_BASE_URL=http://130.210.26.64` (no trailing slash), `ENCRYPTION_KEY`, `WORKER_POOL_SIZE=5`, `MESSAGING_ENABLED=false`.
- `CACHE_PASSWORD` and `KV_PASSWORD` share one generated value; the same `DB_PASSWORD` is used on both hosts by construction of the setup script.
- No secret ever appears in git, shell history, or chat; the setup script wipes its temp directory on exit.
- **Future (spec 016):** non-secret configuration will be sourced from `Zatu-ecom/Config` per environment and merged with secrets at deploy time; until then, the server-side `.env` files remain the source of truth. Secrets always come from the environment/secret store, never the Config repo. The Config repo's `dev/backend.env` must contain the full non-secret key set (see spec 016), and `prod/backend.env` is added when the production environment is formalized.
- **Domain target:** `api.vitrapay.io` (`A → 130.210.26.64`); `PUBLIC_API_BASE_URL` switches to `https://api.vitrapay.io` when TLS lands (T052). The API path prefix is `/api`; frontends use `NEXT_PUBLIC_API_URL=https://api.vitrapay.io/api`.

## Deployment Procedure (as-built)

1. **Host bootstrap** (one-time, per VM): copy `bootstrap.sh`, run `sudo bash bootstrap.sh data|app`. Produces Docker + compose, 2 GB swap, `docker` group, firewalld rules, disabled package scans.
2. **Secrets**: run `./deploy/setup-secrets.sh` from the laptop; it detects VM-1's private IP, builds both `.env` files, copies them, `chmod 600`.
3. **Data plane**: copy `deploy/vm1-data/docker-compose.yml` + `migrations/` (including `seeds/`) to `~/ecommerce-data/`; `docker compose up -d`.
4. **Migrations**: `docker compose --profile tools run --rm migrate` → `run_migrations.sh --production` (all schema migrations + CORE seeds only; no mock seeds).
5. **Application image**: `docker buildx build --platform linux/amd64 --load -t ecommerce-backend:latest .` on the laptop; ship with `deploy/ship-image.sh` (or the equivalent `docker save | gzip | ssh ... docker load`).
6. **App plane**: copy `deploy/vm2-app/docker-compose.yml` + `nginx-ecommerce.conf` to `~/ecommerce-app/`; `docker compose up -d`.
7. **Expose**: OCI security list `0.0.0.0/0:80`; verify externally.
8. **Verify**: container health, `pg_isready`, external port scan, `curl http://130.210.26.64/` (expect the `CORRELATION_ID_REQUIRED` JSON).

## Migration Strategy

- Production uses `run_migrations.sh --production` — **all schema migrations + CORE seeds only**. The default `--all` mode also loads `seeds/mock` (test users, sample products) and **must never run in prod**; no mock seed has been executed on the production database.
- **CORE seeds** (`migrations/seeds/core/`): plans, roles, geo/currency, storage providers, payment gateways, physical-spec units, courier providers. These are production-safe reference rows.
- **MOCK seeds** (`migrations/seeds/mock/`): demo users (`admin@ecommerce.com`, `*@example.com`), sample products, inventory, wishlists. Never run in production; `--seed`, `--seed-mock`, `--force-seed`, and `--reset` are also forbidden.
- The root `docker-compose.yml` `migrate` service runs `--all` for local development; production uses `deploy/vm1-data/docker-compose.yml`, whose migrate service runs `--production`.
- The runner records applied files in `schema_migration` (41 rows observed) and is idempotent; repeat deploys skip applied files.
- **Post-migration verification**: `schema_migration` rows are all `SUCCESS`; reference tables are populated; `SELECT COUNT(*) FROM "user" WHERE email = 'admin@ecommerce.com' OR email LIKE '%@example.com'` returns 0; no mock seller profiles (`Tech Gadgets Pro`, `Fashion Forward`, `Home & Living Store`) exist.
- Migrations execute inside the `postgres:16-alpine` `migrate` container on the compose network (`DB_HOST=postgres`), not on the host.

## Security Posture

- **Open to the internet**: `22` (key-only, world-open by decision — revisit) and `80` (intended).
- **Private only**: Postgres `5432`, durable Redis `6381` — allowed strictly from `10.0.0.0/24` at both the OCI security list and firewalld.
- **Never exposed**: volatile Redis `6379`, app `8080` (published to `127.0.0.1` only).
- **Secrets**: mode-600 `.env`, no git, hidden-input generation; dev defaults never reused.
- **Gaps (tracked)**: no TLS yet; no backups yet; SSH not source-restricted; `latest` image tag lacks commit traceability.

## Rollback

- **App**: rebuild/reship the previous image and `docker compose up -d` (today), or re-point a `:<branch>-<sha>` tag once manual deploy lands. Target < 2 minutes.
- **Config**: `.env` changes are reversible from the operator's password manager; re-run `setup-secrets.sh`.
- **Data**: volumes persist across deploys; no automated logical backup yet (US6 / Phase 7). RPO is currently undefined — accepted risk until backups land.

## Verification (executed)

| Check | Result |
|---|---|
| External scan VM-1 (`22,80,443,5432,6379,6380,6381,8080`) | only 22 reachable ✅ |
| External scan VM-2 | 22 + 80 reachable; DB/cache/app closed ✅ |
| `docker ps` VM-1 | `ecommerce-postgres` healthy, `ecommerce-cache-durable` healthy ✅ |
| `docker ps` VM-2 | `ecommerce-app` healthy, `ecommerce-cache-volatile` healthy, `ecommerce-nginx` up ✅ |
| App → Postgres | connected; 41 migrations, 82 tables ✅ |
| `curl http://130.210.26.64/` | 400 `CORRELATION_ID_REQUIRED` in ~50–60 ms (app answering) ✅ |
| API smoke (`/api/product/category`, `/api/fulfillment/couriers`) | 400 `INVALID_SELLER` / 401 — middleware working ✅ |

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| 1 GB-class Micros thrash and freeze sshd | management outage (app unaffected) | 2 GB swap; `dnf-makecache.timer` disabled; reboot via console if needed |
| Single host per role | DB or app downtime on failure | Accepted for Phase 1; backups (Phase 7) + reserved IPs reduce recovery friction |
| SSH `22` world-open | brute-force surface | key-only auth (password login disabled); restrict later / Tailscale |
| No TLS | plaintext API, blocks frontends | spec 015 makes HTTPS a hard dependency; add name + certbot |
| No backups | data loss on disk failure | Phase 7 tasks: nightly `pg_dump`, boot-volume policy, restore drill |
| `latest` tag only | cannot prove which build runs | future manual GitHub deploy: branch+SHA tags, deploy record, Actions history |
| VM-1 only ~498 MB RAM | Postgres memory pressure | tuned `shared_buffers=64MB`, `max_connections=20`; consider A1 migration later |

## Remaining Work

Tracked in [tasks.md](./tasks.md) Phase 7–9: backups + retention, Docker log rotation, boot-volume backup policy, TLS rollout (`api.vitrapay.io` + certbot), SSH hardening, repo/docs commit, central-config adoption (spec 016; bootstrap `dev` first, then `prod`), and the future manual GitHub deploy. `/version` is intentionally dropped from scope.
