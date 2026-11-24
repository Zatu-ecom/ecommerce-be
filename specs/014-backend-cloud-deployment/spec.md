# Feature Specification: Backend Cloud Deployment (Oracle Cloud, Phase 1)

**Feature Branch**: `014-backend-cloud-deployment`
**Created**: 2026-10-13
**Updated**: 2026-11-22
**Status**: **Implemented — live** (`dev` environment; HTTP, IP-based; `api.vitrapay.io` DNS + TLS pending)
**Input**: User description: "Deploy the backend with all its dependencies on Oracle Cloud first; frontends come later as a separate phase."
**Related docs**: `specs/014-backend-cloud-deployment/plan.md`, `tasks.md`, `deploy/ORACLE-SETUP-TODO.md`, `deploy/bootstrap.sh`, `deploy/setup-secrets.sh`, `deploy/ship-image.sh`, `deploy/vm1-data/`, `deploy/vm2-app/`, `deploy/monitor/`, `deploy/vm1-data/backup/`, `deploy/ROTATION.md`, `deploy/GITHUB-DEPLOY.md`, `.github/workflows/deploy-oracle.yml`, `Zatu-ecom/Config` (non-secret configuration)
**Sibling specs**: `specs/015-frontend-cloud-deployment/spec.md` (frontends, Phase 2), `specs/016-config-repository/spec.md` (non-secret configuration)

## Background

The backend (`ecommerce-be`, Go/Gin monolith) runs locally via `docker compose` (`make quickstart: build up migrate`). Production needs the same stack on an always-on host with a public endpoint, because:

- frontends (storefront + seller dashboard, Phase 2 / spec 015) can only call an `https://` API from an HTTPS page (mixed-content blocking), and
- courier webhooks (`{PUBLIC_API_BASE_URL}/api/fulfillment/webhooks/{code}`) must reach us from the public internet.

**As-built summary (2026-11-17):** two Oracle Always Free `VM.Standard.E2.1.Micro` instances in one VCN/subnet, split by role:

| Role | Public IP (reserved) | Private IP | Runs |
|---|---|---|---|
| VM-1 (data) | `140.238.240.2` | `10.0.0.208` | PostgreSQL 16, durable Redis (KV) |
| VM-2 (app) | `130.210.26.64` | `10.0.0.88` | Go app, volatile Redis (cache), nginx edge |

Public URL today: **`http://130.210.26.64`** (HTTP only; TLS deferred — see US4).

**Domain and DNS plan (2026-11-22):** `vitrapay.io` is owned. The backend target name is **`api.vitrapay.io`**; TLS rollout is pending (US4 / T052).

| Record | Type | Target | Purpose | Status |
|---|---|---|---|---|
| `api.vitrapay.io` | `A` | `130.210.26.64` | Backend API (required) | ⏳ create with TLS rollout (T052) |
| `shop.vitrapay.io` | `CNAME` | Vercel storefront project | Storefront (optional) | P3, spec 015 |
| `seller.vitrapay.io` | `CNAME` | Vercel seller-dashboard project | Seller dashboard (optional) | P3, spec 015 |

The API is served under the **`/api` path prefix** (`/api/user`, `/api/product`, `/api/fulfillment`, …). `PUBLIC_API_BASE_URL` is the origin only (`https://api.vitrapay.io`, no trailing slash); the frontends use `NEXT_PUBLIC_API_URL=https://api.vitrapay.io/api`.

The primary single-Ampere-A1 design from the original draft was not used; the two-Micro fallback became the deployed architecture because A1 capacity was unavailable in the home region. Nginx (not Caddy) was chosen as the edge for the IP-first phase; certbot is the planned TLS path when a name exists.

Runtime dependency inventory (verified from `deploy/vm1-data` + `deploy/vm2-app` + `common/config`):

| Dependency | Where | Required? | Notes |
|---|---|---|---|
| PostgreSQL 16 | VM-1, `postgres:16-alpine` | Yes | System of record; tuned for 1 GB-class hosts (`shared_buffers=64MB`, `max_connections=20`) |
| Redis durable (KV) | VM-1, `redis:7-alpine`, `noeviction` + AOF | Yes | Scheduler queue, idempotency, limiter — pressure must fail loud |
| Redis volatile (cache) | VM-2, `redis:7-alpine`, LRU, 64 MB cap | Yes | Fail-open cache; data loss on restart is by design |
| Go app | VM-2, image `ecommerce-backend:latest` | Yes | Single binary, `:8080`, published on `127.0.0.1` only |
| nginx 1.27 | VM-2, `nginx:1.27-alpine` | Yes | Public `:80` edge, reverse-proxies to `app:8080` |
| Docker 29.7.2 (static) + Compose plugin | both VMs | Yes | Installed from static tarball (no `dnf`, avoids OOM on Micros) |
| RabbitMQ | none | No | `MESSAGING_ENABLED=false`; Noop publisher/consumer |
| Object storage / SMTP / gateway keys | seller-configured rows in DB | No | Encrypted with `ENCRYPTION_KEY`; no host-level setup |

Out of scope for this spec: frontend deployments (see `specs/015-frontend-cloud-deployment`), CI/CD pipelines (manual GitHub deploy tracked as future work), multi-region/HA, managed DB (Autonomous DB), Kafka, the `/version` endpoint (dropped — see Out of Scope).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Cloud network and VMs exist (Priority: P1) — ✅ Implemented

The operator creates a VCN with one public subnet, an internet gateway, and two Micro VMs (Oracle Linux 9) with public IPs. The security list allows inbound only `80` from `0.0.0.0/0`, `22` from the operator (temporarily `0.0.0.0/0` — deferred hardening), and DB/Redis ports only from the private subnet `10.0.0.0/24`. Docker Engine + Compose are installed on both.

**Why this priority**: Nothing else can proceed without reachable hosts.

**Independent Test**: `ssh` into each VM and `docker info` succeeds; an external scan shows only 80/22 reachable on VM-2 and only 22 on VM-1.

**Acceptance Scenarios**:

1. **Given** the VCN and security list, **When** an external scan runs, **Then** 5432/6379/6380/6381/8080 are unreachable from the internet on both VMs. ✅
2. **Given** the two VMs in the same subnet, **When** VM-2 connects to VM-1's private IP on 5432/6381, **Then** the connection succeeds. ✅
3. **Given** the instances, **When** the public IPs are reserved in OCI, **Then** the addresses persist across instance lifecycle changes. ✅

---

### User Story 2 - Production secrets and environment are set (Priority: P1) — ✅ Implemented

The operator provisions server-side `.env` files (never committed) with production values. Fresh secrets are generated; dev defaults are never reused.

**Why this priority**: Wrong secrets silently break auth, encryption, and webhooks; the dev `ENCRYPTION_KEY` default would leave stored courier credentials recoverable by anyone with the repo.

**Independent Test**: `docker compose config` renders with all required vars; no value equals a known dev default; files are mode `600`.

**Acceptance Scenarios**:

1. **Given** the secret checklist, **When** `.env` is written, **Then** `DB_PASSWORD`, `CACHE_PASSWORD`/`KV_PASSWORD`, `JWT_SECRET`, and `ENCRYPTION_KEY` (32-byte, freshly generated) are all set and differ from dev values. ✅
2. **Given** the deployed VM, **When** the app boots, **Then** `APP_ENV=dev`, `GIN_MODE=release`, `PUBLIC_API_BASE_URL=http://130.210.26.64` (switch to `https://api.vitrapay.io` with TLS), `DB_SSLMODE=disable` (same-VCN Postgres), and `MESSAGING_ENABLED=false`. ✅
3. **Given** the split-host layout, **When** configured, **Then** `DB_HOST`/`KV_HOST` point at VM-1's **private** IP (`10.0.0.208`) and Redis caps are lowered (`CACHE_MAXMEMORY=64mb`, `KV_MAXMEMORY=128mb`). ✅

---

### User Story 3 - Stack deploys and migrates cleanly (Priority: P1) — ✅ Implemented

The operator ships a pre-built `linux/amd64` image to VM-2 (never builds Go on a 1 GB Micro), copies compose files + `.env` to the hosts, and runs the equivalent of `make up` + `make migrate` (compose `migrate` tools profile on VM-1).

**Why this priority**: This is the deployment itself.

**Independent Test**: `docker compose ps` shows Postgres, durable Redis, app, volatile cache, nginx healthy; `GET /` on VM-2 returns the app's JSON (400 correlation-ID requirement is a valid response).

**Acceptance Scenarios**:

1. **Given** the image and compose files, **When** `docker compose up -d` runs on VM-1 and VM-2, **Then** Postgres passes `pg_isready`, both Redis instances answer `PING`, and the app container is `healthy`. ✅
2. **Given** a fresh Postgres volume, **When** the `migrate` profile runs `run_migrations.sh --production`, **Then** it exits 0, all schema migrations are recorded (`schema_migration`; 41 rows observed), 82 application tables exist, and CORE reference seeds are loaded — with **zero** rows from `seeds/mock`. ✅
3. **Given** a repeat deploy, **When** compose is re-applied, **Then** migrations are idempotent and Postgres/KV volumes persist (only `cache-volatile` may be empty). ✅

---

### User Story 4 - Public HTTPS serves the API (Priority: P1) — ⚠️ Partial (HTTP live, TLS deferred)

The public IP serves the API over HTTP today. HTTPS (TLS termination, auto-renew) is pending the `api.vitrapay.io` DNS record + certbot; the domain name is decided (T052), and certbot takes over nginx once the record resolves.

**Why this priority**: Frontends and courier webhooks both ultimately require a public HTTPS endpoint; HTTPS is a launch blocker for Phase 2 (spec 015), not for backend smoke testing.

**Independent Test**: `curl http://130.210.26.64/` returns the app's JSON with a valid nginx hop; `curl http://<vm-ip>:8080/` from the internet times out.

**Acceptance Scenarios**:

1. **Given** the IP-based phase, **When** a request arrives through nginx, **Then** the app answers and the request is logged by nginx. ✅
2. **Given** the DNS `A` record `api.vitrapay.io → 130.210.26.64`, **When** certbot runs with the webroot, **Then** `https://api.vitrapay.io/` responds and renews without operator action. ⏳ deferred (T052)
3. **Given** proxied requests, **When** the app logs client IPs, **Then** the real client IP is visible via `X-Forwarded-For` (trusted-proxy configuration). ⏳ to verify with HTTPS rollout

---

### User Story 5 - Backend is callable from the future frontends (Priority: P2) — ✅ Implemented (verification via curl)

CORS on the API currently allows any origin (`Access-Control-Allow-Origin: *`), so browser-equivalent preflight requests succeed. No frontend is deployed in this phase — verification uses `curl` with `Origin` headers. Tightening the allowlist belongs to spec 015.

**Why this priority**: Catches the most common "deployed but the UI can't reach it" failure before Phase 2 starts.

**Independent Test**: `curl -H 'Origin: https://placeholder.vercel.app' -X OPTIONS` returns the appropriate CORS headers.

**Acceptance Scenarios**:

1. **Given** the current wildcard CORS, **When** a preflight arrives from any origin, **Then** it succeeds. ✅ (tighten in 015)
2. **Given** a real seller account created through the normal registration/admin flow (never via `seeds/mock`), **When** credentials are valid, **Then** a JWT is issued and an authenticated fulfillment read returns 200. ⏳ pending creation of the first production seller

---

### User Story 6 - Backups and day-2 ops exist (Priority: P2) — ❌ Not implemented

Boot-volume backup policy, nightly Postgres logical dumps, log rotation (compose JSON limits), and monitoring/alerts are still pending. Tracked in `tasks.md` Phase 7.

**Why this priority**: A store with no backup story is one disk failure from data loss.

**Independent Test**: Backup policy attached in console; a manual `pg_dump` restore into a scratch container succeeds.

**Acceptance Scenarios**:

1. **Given** the policy, **When** a boot volume fails, **Then** the VM can be rebuilt from the latest backup and the Postgres volume restored. ⏳
2. **Given** nightly dumps, **When** a dump is taken, **Then** the previous 7 days are retained and the newest dump restores cleanly. ⏳

---

### Edge Cases

- 1 GB-class Micro hosts can thrash under OS package scans, freezing sshd while HTTP keeps serving. Mitigation: 2 GB swap + `dnf-makecache.timer` disabled; observed and handled. ✅
- Ephemeral public IPs change if an instance is terminated/recreated → reserved public IPs now assigned. ✅
- The app container has no anonymous health route (all routes require `X-Correlation-ID` and/or auth) → Docker healthcheck uses a TCP check. ✅
- nginx must proxy to the compose service name (`app:8080`), not `127.0.0.1` (container-local). Fixed and verified. ✅
- Image is currently tagged `latest` → no commit traceability until the future GitHub deploy feature lands. ⏳
- No TLS yet → any credentials sent over `http://130.210.26.64` are plaintext. Accepted for smoke testing only. ⚠️
- `seeds/mock` accidentally run in production → demo users (`admin@ecommerce.com`, `*@example.com`), sample products, and fake sellers appear. Recovery: identify/remove the mock rows, fix the deploy command, and re-run only `--production`.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Single-role-per-host `docker compose` deployment: VM-1 `postgres`, `cache-durable`, `migrate` (tools profile); VM-2 `app`, `cache-volatile`, `nginx`. ✅
- **FR-002**: nginx reverse-proxies public `:80` to `app:8080` on the shared compose network, with `X-Forwarded-For`/`X-Forwarded-Proto`, 100 MB uploads, gzip. ✅
- **FR-003**: Server-side `.env` per the US2 checklist; `docker compose config` renders without missing-variable warnings; files mode `600`. ✅
- **FR-004**: Migrations run via the existing `migrate` profile (`run_migrations.sh --production`), idempotent on re-deploy. **All schema migrations and CORE seeds run; `seeds/mock` MUST never run in production** (the default `--all` mode is forbidden). ✅ (41 `schema_migration` rows: 34 schema + 7 CORE; mock seeds skipped)
- **FR-005**: DB/Redis reachable only over the VCN subnet (`10.0.0.0/24`), enforced by both the OCI security list and firewalld rich rules. ✅
- **FR-006**: Docker installed from static tarballs (no `dnf` transactions) on both hosts; compose plugin present; `opc` in the `docker` group. ✅
- **FR-007**: Two GB additional swap per host; OS background package scans disabled to prevent sshd starvation. ✅
- **FR-008**: Public IPs reserved in OCI so addresses survive instance lifecycle changes. ✅
- **FR-009**: Operator-facing runbook maintained in `deploy/ORACLE-SETUP-TODO.md`, including a current deployment record. ✅
- **FR-010**: Non-secret configuration MUST come from `Zatu-ecom/Config` (spec 016) and secrets from the environment/secret store; the server-side `.env` is rendered from these sources, never hand-edited. ⏳ (T071)
- **FR-011**: A fresh production database initialized with `--production` MUST be reproducible: same schema, same CORE reference rows, and zero mock/demo rows. ✅

### Non-Functional / Constraints

- **NFR-001**: Cost target `$0/mo` — Always Free resources only (2× E2.1.Micro, reserved IPs attached, no paid tiers).
- **NFR-002**: No secret material in git, shell history, screenshots, or chat — `.env` lives on the servers (mode `600`), generated via a hidden-input script.
- **NFR-003**: ARM (Ampere) images are not required by the deployed path; the current images are `linux/amd64`; all base images remain multi-arch for a future A1 migration.
- **NFR-004**: Public attack surface is exactly `80` (plus `22`, currently world-open and key-only — deferred hardening); DB/Redis/application ports are unreachable from the internet.
- **NFR-005**: The stack must run within ~1 GB RAM per host — Postgres tuned (`shared_buffers=64MB`, `max_connections=20`), Redis capped (64 MB / 128 MB).

### Database Migration Policy (production)

The production database is created and upgraded **only** through the migration runner's production mode:

```bash
# VM-1, from ~/ecommerce-data
docker compose --profile tools run --rm migrate
# the migrate service executes: ./run_migrations.sh --production
```

**Allowed in production**

1. **All schema migrations** present in `migrations/` (currently `001_…` through `034_…`) in lexicographic order. They are tracked in `schema_migration` and are idempotent on re-run.
2. **All CORE seeds** in `migrations/seeds/core/` — production-safe reference data only:
   - `000_seed_plans.sql`
   - `001_seed_roles.sql`
   - `002_seed_geo_data.sql` (countries + currencies)
   - `003_seed_storage_providers.sql`
   - `004_seed_payment_gateways.sql`
   - `005_seed_physical_specs.sql`
   - `006_seed_courier_providers.sql`

**Forbidden in production**

- `migrations/seeds/mock/` — demo users (`admin@ecommerce.com`, `*@example.com`), sample products, inventory, and wishlists.
- Runner modes `--all` / default, `--seed`, `--seed-mock`, `--force-seed`, and `--reset`.
- The local root `docker-compose.yml` `migrate` service, which runs `--all`; production uses `deploy/vm1-data/docker-compose.yml` (`--production`).

**Verification after every production migration**

- `schema_migration` has one `SUCCESS` row per schema + CORE file (41 observed on 2026-11-17).
- Reference tables are populated: `role`, `plan`, `country`, `currency`, `storage_provider`, `payment_gateway`, `physical_spec_unit`, `courier_provider`.
- No mock data:

```sql
SELECT COUNT(*) FROM "user"
WHERE email = 'admin@ecommerce.com' OR email LIKE '%@example.com'; -- must be 0

SELECT COUNT(*) FROM seller_profile
WHERE business_name IN ('Tech Gadgets Pro', 'Fashion Forward', 'Home & Living Store'); -- must be 0
```

- If a future migration needs new reference data, it goes into `migrations/seeds/core/` (never `seeds/mock/`) and is verified the same way.

### Configuration Sources (target — spec 016)

- **Non-secret values** come from `Zatu-ecom/Config` at `<env>/backend.env` (`local` for laptops, `dev` for the deployed Oracle VMs). The full key inventory and layout are defined in `specs/016-config-repository/spec.md`.
- **Secrets** (`DB_PASSWORD`, `JWT_SECRET`, `ENCRYPTION_KEY`, `CACHE_PASSWORD`, `KV_PASSWORD`, and any broker/provider credential) come from the environment or a secret store at deploy time; they are never committed to the Config repo.
- The deploy step renders `~/ecommerce-data/.env` and `~/ecommerce-app/.env` from Config + secrets, validates required keys, and records the Config commit SHA in the deployment record (T071).
- Until T071 lands, the current server-side `.env` files remain the source of truth, but their **key set must match** the Config repo's `backend.env` so the switch is a value move, not a redesign.

## Out of Scope

- **Frontend deployments** — storefront + seller dashboard are spec `015-frontend-cloud-deployment` (Vercel Hobby, outside Oracle).
- **Central configuration repository** — spec `016-config-repository` (`Zatu-ecom/Config`). The repo layout, full dev key inventory, and secret policy are defined there; backend adoption is tracked as T071. Until then, the current `.env`-on-server flow stays.
- **CI/CD pipelines / auto-deploy** — every deploy is manual today. A manual-trigger GitHub Actions deploy (`workflow_dispatch`, pick a branch, no push trigger) is tracked as future work in `tasks.md`.
- **`/version` endpoint** — dropped. Build visibility is provided by GitHub Actions run history and a server-side deploy record when the manual-deploy feature lands; no API endpoint is required for Phase 1.
- Multi-region/HA, managed DB (Autonomous Database), Kafka, A1 migration (kept as a future option).
- TLS rollout itself is deferred (blocked on the `api.vitrapay.io` DNS record + certbot), but the domain target is decided.

## Rollback

- **Application**: re-deploy the previous image. Today the image is `ecommerce-backend:latest`; once the manual-deploy feature lands, images will be tagged `:<branch>-<sha>` and the previous tag can be re-pointed and restarted in < 2 minutes.
- **Data**: Postgres/KV volumes persist across re-deploys. Until nightly dumps + boot-volume backups land (US6), the only rollback is volume-level and there is no point-in-time recovery (accepted risk, tracked).

## What the Operator Provided *(completed)*

1. **OCI access**: console-driven provisioning (no OCI CLI configured). ✅
2. **SSH keypair**: `~/.ssh/oci_vm1`, `~/.ssh/oci_vm2` (ed25519). ✅
3. **Domain + DNS**: `vitrapay.io` owned (2026-11-22); `A api → 130.210.26.64` planned; `shop`/`seller` CNAMEs optional for spec 015; HTTPS deferred. ⏳
4. **Secret values**: `DB_PASSWORD`, `CACHE_PASSWORD`/`KV_PASSWORD`, `JWT_SECRET`, `ENCRYPTION_KEY` — generated fresh, installed only on the VMs. ✅
5. **Shape decision**: 2× E2.1.Micro (A1 unavailable). ✅
6. **Image delivery**: built on the operator laptop (`linux/amd64`), shipped via `docker save | gzip | ssh | docker load`. ✅

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `http://130.210.26.64/` answers from the public internet in < 1 s. ✅ (~50–60 ms observed)
- **SC-002**: All five containers healthy (`postgres`, `cache-durable`, `app`, `cache-volatile`, `nginx`). ✅
- **SC-003**: All schema migrations applied (41 observed), 82 application tables present, CORE reference seeds loaded, and zero `seeds/mock` rows. ✅
- **SC-004**: External scan shows zero database/cache/application ports open to the internet. ✅
- **SC-005**: A repeat `docker compose up -d` + `migrate` succeeds without data loss. ✅ (idempotent paths verified)
- **SC-006**: Operator can restore service after a host failure once US6 backups land. ⏳ pending

## Assumptions

- The two Micro hosts remain on Always Free and are not resized; the 1 GB-class memory constraint is permanent for this phase.
- IP-based HTTP is acceptable for backend smoke testing; real users/frontends wait for HTTPS (spec 015 + TLS rollout).
- No OCI CLI is configured; console actions (security lists, reserved IPs, reboots) are performed by the operator.
- The repo default `--all` migration mode is never used in production (it loads mock/dev seeds); production uses `--production` (migrations + CORE seeds only), and `seeds/mock` is never executed.
- The Config repo (`Zatu-ecom/Config`) is the source of non-secret configuration (spec 016); local `.env` files remain developer overrides, and secrets are injected from the environment/secret store.
- Images are built on an `amd64` machine; the VMs do not build Go code.
