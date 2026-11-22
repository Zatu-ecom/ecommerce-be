# Feature Specification: Backend Cloud Deployment (Oracle Cloud, Phase 1)

**Feature Branch**: `014-backend-cloud-deployment`
**Created**: 2026-10-13
**Status**: Draft
**Input**: User description: "Deploy the backend with all its dependencies on Oracle Cloud first; frontends come later as a separate phase."
**Related docs**: `ecommerce-be/docker-compose.yml`, `ecommerce-be/Dockerfile`, `ecommerce-be/Makefile`, `common/config/*` (env contract)

## Background

The backend (`ecommerce-be`, Go/Gin monolith) currently runs locally via `docker compose` (`make quickstart: build up migrate`). Production needs the same stack on an always-on host with a public HTTPS endpoint, because the browser frontends (Vercel-hosted Next.js, Phase 2) can only call an `https://` API (mixed-content blocking) and courier webhooks (`{PUBLIC_API_BASE_URL}/api/fulfillment/webhooks/{code}`) must reach us from the public internet.

Runtime dependency inventory (verified from `docker-compose.yml` + `common/config`):

| Dependency | Image / form | Required? | Notes |
|---|---|---|---|
| Postgres 16 | `postgres:16-alpine` + `./migrations` init | Yes | System of record; app boots on Postgres only |
| Redis volatile (cache) | `redis:7-alpine`, LRU, 256 MB cap | Yes | Fail-open cache; data loss on restart is by design |
| Redis durable (KV) | `redis:7-alpine`, noeviction + AOF | Yes | Scheduler queue, idempotency, limiter — pressure must fail loud |
| Go app | built from `Dockerfile` (`ecommerce-backend:latest`) | Yes | Single binary, `:8080` |
| Caddy 2 (TLS reverse proxy) | `caddy:2-alpine` — **new, to be added** | Yes (prod) | Auto-HTTPS, hides `:8080` |
| RabbitMQ | none in compose; `MESSAGING_ENABLED` defaults `false` | No | Noop publisher/consumer unless explicitly enabled |
| Object storage / SMTP / Razorpay keys | seller-configured rows in DB, encrypted with `ENCRYPTION_KEY` | No host-level setup | No new infra; key must be strong (see US2) |

Out of scope for this spec: frontend deployments (Phase 2), CI/CD pipelines, multi-region/HA, managed DB (Autonomous DB) migration, Kafka.

## Target Architecture (decision)

**Primary: single Ampere A1 VM (2 OCPU / 12 GB RAM, Always Free)**, one VCN, `docker compose` deployment as-is plus a Caddy service. One box, ~$0, ~10x headroom over the stack's ~1 GB steady state.

**Fallback (only if A1 capacity is unavailable in the home region): two E2.1.Micro VMs (1 OCPU / 1 GB each)** — VM-1 data (Postgres + durable Redis), VM-2 app (Go + volatile Redis + Caddy), private-subnet traffic between them. Same compose files split per host with tuned caps (`CACHE_MAXMEMORY=64mb`, `KV_MAXMEMORY=128mb`, Postgres `shared_buffers=64MB`). Accepted only as fallback: two boxes to patch/back up, no HA either way.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Cloud network and VM exist (Priority: P1)

The operator creates (console or CLI) a VCN with one public subnet, an internet gateway, and a VM (Ampere A1, Oracle Linux 9, 50 GB boot volume) with a public IP. The security list allows inbound **only** `80/443` from `0.0.0.0/0` and `22` from the operator's IP; DB/Redis ports are never public. Docker Engine + Compose plugin are installed.

**Why this priority**: Nothing else can proceed without a reachable host.

**Independent Test**: `ssh` into the VM and `docker info` succeeds; `nmap`/console shows only 22/80/443 reachable.

**Acceptance Scenarios**:

1. **Given** a fresh tenancy, **When** the VM and security list are created, **Then** ports 80/443 are open to the world, 22 is restricted to the operator IP, and 5432/6379/6380/6381/8080 are unreachable from the internet.
2. **Given** the Micro fallback, **When** two VMs are created in the same VCN/subnet, **Then** they reach each other over private IPs and the same port rules hold per role (VM-1: no public ports at all except 22; VM-2: 80/443).

---

### User Story 2 - Production secrets and environment are set (Priority: P1)

The operator provisions a server-side `.env` (never committed) with production values. Fresh secrets are generated; dev defaults are never reused.

**Why this priority**: Wrong secrets silently break auth, encryption, and webhooks; the dev `ENCRYPTION_KEY` default would leave stored courier credentials recoverable by anyone with the repo.

**Independent Test**: `docker compose config` renders with all required vars; no value equals a known dev default.

**Acceptance Scenarios**:

1. **Given** the secret checklist, **When** `.env` is written, **Then** `DB_PASSWORD`, `CACHE_PASSWORD`/`KV_PASSWORD`, `JWT_SECRET`, and `ENCRYPTION_KEY` (32-byte, freshly generated, e.g. `openssl rand -hex 32`) are all set and differ from dev values.
2. **Given** production, **When** the app boots, **Then** `APP_ENV=prod`, `GIN_MODE=release`, `PUBLIC_API_BASE_URL=https://<api-domain>` (no trailing path), `DB_SSLMODE=disable` (same-host Postgres over bridge network), and `MESSAGING_ENABLED=false`.
3. **Given** the Micro fallback, **When** configured, **Then** `DB_HOST`/`KV_HOST`/`CACHE_HOST` point at the correct private IPs (or compose service names on single-host) and Redis caps are lowered (`CACHE_MAXMEMORY=64mb`, `KV_MAXMEMORY=128mb`).

---

### User Story 3 - Stack deploys and migrates cleanly (Priority: P1)

The operator ships the backend image (built in CI or on a build machine for x86, or on the A1 itself for ARM — never `go build` on a 1 GB Micro), copies `docker-compose.yml` + `Caddyfile` + `.env` to the VM, and runs the equivalent of `make up` + `make migrate` (compose `app` + `migrate` profile). All containers become healthy.

**Why this priority**: This is the deployment itself.

**Independent Test**: `docker compose ps` shows `postgres`, `cache-volatile`, `cache-durable`, `app`, `caddy` healthy; `GET /` on localhost returns 200.

**Acceptance Scenarios**:

1. **Given** the image and compose files on the VM, **When** `docker compose up -d` runs, **Then** Postgres passes its `pg_isready` healthcheck, both Redis instances answer `PING`, and `app` passes its `/` healthcheck.
2. **Given** a fresh Postgres volume, **When** the `migrate` profile runs `run_migrations.sh --all`, **Then** it exits 0 and the fulfillment/product/order tables exist.
3. **Given** a repeat deploy, **When** compose is re-applied, **Then** migrations are idempotent and the Postgres/KV volumes persist (only `cache-volatile` is allowed to be empty).

---

### User Story 4 - Public HTTPS serves the API (Priority: P1)

DNS `A` record `api.<domain>` points at the VM public IP; Caddy terminates TLS (Let's Encrypt, auto-renew) and reverse-proxies to `app:8080`. Gin trusts the proxy so the Redis-backed IP limiter sees real client IPs (`X-Forwarded-For`, not `127.0.0.1`).

**Why this priority**: Frontends and courier webhooks both require a public HTTPS endpoint.

**Independent Test**: `curl https://api.<domain>/` returns the app health response with a valid cert; `curl http://<vm-ip>:8080/` from the internet times out.

**Acceptance Scenarios**:

1. **Given** DNS propagates, **When** Caddy starts, **Then** it obtains a certificate on first request and `https://api.<domain>/` responds 200.
2. **Given** a client request through Caddy, **When** the app logs/handles it, **Then** the observed client IP is the real public IP (trusted-proxy configured), not the proxy's.
3. **Given** renewal time (90-day LE cert), **When** Caddy renews, **Then** no operator action or downtime is needed (`caddy_data` volume persists).

---

### User Story 5 - Backend is callable from the future frontends (Priority: P2)

CORS on the API permits the (future) Vercel origins, and a browser-equivalent preflight from an allowed origin succeeds. No frontend is deployed in this phase — verification uses `curl` with `Origin` headers.

**Why this priority**: Catches the most common "deployed but the UI can't reach it" failure before Phase 2 starts.

**Independent Test**: `curl -H 'Origin: https://placeholder.vercel.app' -X OPTIONS` returns the appropriate CORS headers per backend policy.

**Acceptance Scenarios**:

1. **Given** the CORS allowlist, **When** a preflight arrives from a listed origin, **Then** it succeeds; from an unlisted origin it is rejected.
2. **Given** a seller login request over HTTPS, **When** credentials are valid, **Then** a JWT is issued and an authenticated fulfillment read (e.g. `GET /api/fulfillment/couriers`) returns 200.

---

### User Story 6 - Backups and day-2 ops exist (Priority: P2)

Boot-volume backup policy (weekly full + daily incremental, Always Free allowances), Postgres logical dump cron (`pg_dump` nightly to block volume / object storage), log rotation (`docker compose logs` JSON limits), and 2 GB swap on Micro fallback only.

**Why this priority**: A store with no backup story is one disk failure from data loss.

**Independent Test**: Backup policy attached in console; a manual `pg_dump` restore into a scratch container succeeds.

**Acceptance Scenarios**:

1. **Given** the policy, **When** a boot volume fails, **Then** the VM can be rebuilt from the latest backup and the Postgres volume restored.
2. **Given** nightly dumps, **When** a dump is taken, **Then** the previous 7 days are retained and the newest dump restores cleanly.

---

## Requirements *(mandatory)*

### Functional

- FR1: Single-host `docker compose` deployment: `postgres`, `cache-volatile`, `cache-durable`, `app`, `caddy` (new service), `migrate` (tools profile).
- FR2: `Caddyfile` with `reverse_proxy` to the app and persistent `data`/`config` volumes.
- FR3: Server-side `.env` per the US2 checklist; `docker compose config` must render without missing-variable warnings.
- FR4: Migrations run via the existing `migrate` profile (`run_migrations.sh --all`), idempotent on re-deploy.
- FR5: Micro fallback uses lowered Redis caps and Postgres `shared_buffers=64MB`, `max_connections=20`.
- FR6: Gin trusted-proxy configuration covers the Caddy/bridge-network source address.

### Non-Functional / Constraints

- NFR1: Cost target $0/mo (Always Free resources only; Pro/paid tiers out of scope).
- NFR2: No secret material in git, shell history screenshots, or chat — `.env` lives on the server (and optionally OCI Vault later).
- NFR3: ARM (Ampere) images must resolve for `linux/arm64` (`postgres:16-alpine`, `redis:7-alpine`, `caddy:2-alpine` all multi-arch; Go image built with `GOARCH=arm64`).
- NFR4: Public attack surface is exactly 80/443 (+22 restricted); DB/Redis/app ports are localhost/V CN-only.

## Rollback

Each deploy tags the previous `ecommerce-backend` image (`:prev`); rollback is `docker compose` re-point + restart (<2 min). Data rollback is the nightly `pg_dump` + volume backup (RPO ≤ 24 h).

## What the operator (you) must provide

1. **OCI access**: tenancy OCID, home region, compartment choice (new `ecommerce-prod` compartment recommended), and either (a) OCI CLI config (user OCID + API private key + fingerprint) if you want agent/CLI-driven provisioning, or (b) console access where you create the VCN/VM yourself.
2. **SSH keypair**: you generate (`ssh-keygen -t ed25519`); the public key goes into the VM create step; the private key never leaves your machine.
3. **Domain + DNS**: a domain you control and where its DNS is managed (registrar / Cloudflare / OCI DNS), so we can add the `api` A record.
4. **Secret values** (generated fresh, shared securely, never in chat if avoidable): `DB_PASSWORD`, `CACHE_PASSWORD` (= `KV_PASSWORD` may reuse), `JWT_SECRET`, `ENCRYPTION_KEY` (`openssl rand -hex 32`).
5. **Shape decision**: Ampere A1 (recommended) vs 2× Micro fallback — confirm A1 capacity in your region, else we split.
6. **Image delivery**: confirm how the image reaches the VM — build on the A1 itself (simplest, `docker build` from repo) or ship a tarball (`docker save` → `scp` → `docker load`).

## Phase 2 preview (not this spec)

Vercel Hobby projects for `storefront` + `seller-dashboard` with `NEXT_PUBLIC_API_URL=https://api.<domain>` (build-time), backend CORS allowlist update, and end-to-end login → courier → shipment smoke test.
