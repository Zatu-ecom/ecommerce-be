# Feature Specification: Central Configuration Repository (`Zatu-ecom/Config`)

**Feature Branch**: `016-config-repository`
**Created**: 2026-11-22
**Updated**: 2026-11-22
**Status**: Bootstrapped and pushed — `Zatu-ecom/Config` holds `local/` (laptop) and `dev/` (deployed Oracle VMs) plus `required/`, `scripts/`, README, `.gitignore`. Remaining: add a secret-scan hook and switch the deploy path to Config (014 T071).
**Input**: User description: "We have a shared Config repo — all non-secret `.env` values should live there per environment; passwords/secrets stay in environment variables or a secret store. Deployment should take values from the Config repo."
**Related docs**: `specs/014-backend-cloud-deployment/`, `specs/015-frontend-cloud-deployment/`, `https://github.com/Zatu-ecom/Config`

## Background

Configuration is currently duplicated across local `.env` files and the two production servers (`~/ecommerce-data/.env`, `~/ecommerce-app/.env`). The app reads a large set of environment variables (`common/config`), and each frontend adds its own build-time variables. Without a single source of truth:

- `local` and `dev` drift silently;
- a new environment requires copying values by hand;
- nobody can see *what configuration a running deployment used*;
- secrets risk being pasted into the wrong file or committed by accident.

A dedicated repository, `Zatu-ecom/Config`, holds **non-secret** configuration per service and per environment. Secrets remain outside the repo and are injected at deploy time from environment variables or a secret store.

**Approach decision (2026-11-22):**

| Concern | Decision |
|---|---|
| Non-secret values | Committed in `Zatu-ecom/Config`, one file per service per environment |
| Secrets | Never committed; injected from the environment/secret store at deploy time |
| Local (laptop) | `local/{backend,storefront,seller-dashboard}.env` |
| Deployed (`dev`) | `dev/{backend,storefront,seller-dashboard}.env` — the Oracle VMs; no `prod` environment exists |
| Local developer overrides | Allowed via each developer's own `.env`; never committed |
| Backend runtime | Deploy renders `~/ecommerce-data/.env` and `~/ecommerce-app/.env` from Config + secrets |
| Frontend runtime | Vercel environment variables mirror the Config values (manual first, API automation later) |

## Repository Layout (as built)

```text
Config/
├── README.md                     # layout, key reference, quoting rules, secret names
├── local/                        # a developer laptop (localhost values)
│   ├── backend.env
│   ├── storefront.env
│   └── seller-dashboard.env
├── dev/                          # the deployed Oracle environment
│   ├── backend.env               # non-secret backend values + secret placeholders
│   ├── storefront.env            # NEXT_PUBLIC_* only
│   └── seller-dashboard.env      # NEXT_PUBLIC_* only
├── required/
│   ├── backend.keys              # required-key list for validation
│   ├── storefront.keys
│   └── seller-dashboard.keys
├── scripts/
│   ├── validate.sh               # checks required keys + placeholder rules
│   └── render-backend-env.sh     # merges <env>/backend.env + secrets → .env
└── .gitignore                    # ignores *.local, *.secret, .env, .env.*
```

Rules:

- The file path is always `<env>/<service>.env`; adding an environment means adding one folder, not changing scripts.
- `required/<service>.keys` lists every key the service needs (including secret names); validation compares the env file against it.
- `scripts/` is tooling, not configuration; it must work with the current manual deploy and the future GitHub Actions deploy.
- The repo is **private**; deployment credentials are read-only.

## Configuration Inventory

This inventory is derived from the local `.env`, `.env.test`, `common/config/*.go`, the provider factories, and `deploy/.env.example`. It is the authoritative list the Config repo must cover.

### Backend — runtime

| Key | Secret? | Dev value | Notes |
|---|---|---|---|
| `APP_ENV` | No | `local` | `dev` on the Oracle VMs; `prod` is not used |
| `GIN_MODE` | No | `debug` | `release` on the VMs (cheaper on 1 GB hosts) |
| `PORT` | No | `8080` | App listen port; nginx proxies to it |
| `LOG_LEVEL` | No | `debug` | `info` in production |
| `EXTENDED_LOGGING` | No | `false` | Verbose request logging |

### Backend — PostgreSQL

| Key | Secret? | Dev value | Notes |
|---|---|---|---|
| `DB_HOST` | No | `localhost` | `10.0.0.208` (VM-1 private IP) in production |
| `DB_PORT` | No | `5432` | |
| `DB_NAME` | No | `ecommerce_test` | `ecommerce` in production |
| `DB_USER` | No | `postgres` | |
| `DB_PASSWORD` | **Yes** | placeholder | Injected from environment/secret store |
| `DB_SSLMODE` | No | `disable` | Same-VCN Postgres in production |
| `DB_MAX_OPEN_CONNS` | No | `25` | Lower on 1 GB hosts (`20` on the VMs) |
| `DB_MAX_IDLE_CONNS` | No | `25` | Lower on 1 GB hosts (`5` on the VMs) |
| `DB_CONN_MAX_LIFETIME_MINUTES` | No | `5` | `30` on the VMs |
| `DB_CONN_MAX_IDLE_TIME_MINUTES` | No | `5` | |

### Backend — auth

| Key | Secret? | Dev value | Notes |
|---|---|---|---|
| `JWT_SECRET` | **Yes** | placeholder | Fresh 32-byte value per environment; never the dev default |
| `JWT_EXPIRY_HOURS` | No | `72` | `24` in production |

### Backend — volatile cache (Redis)

| Key | Secret? | Dev value | Notes |
|---|---|---|---|
| `CACHE_HOST` | No | `localhost` | `cache-volatile` (compose service) in production |
| `CACHE_PORT` | No | `6379` | |
| `CACHE_PASSWORD` | **Yes** | placeholder | Authenticates the volatile role |
| `CACHE_ADDR` | No | unset | Optional full `host:port` override; compose sets it |
| `CACHE_MAXMEMORY` | No | unset | Compose-only (`64mb` on the VMs); not read by the app |
| `REDIS_POOL_SIZE` | No | `20` | |
| `REDIS_MIN_IDLE_CONNS` | No | `5` | |
| `REDIS_DIAL_TIMEOUT_MS` | No | `5000` | |
| `REDIS_READ_TIMEOUT_MS` | No | `500` | |
| `REDIS_WRITE_TIMEOUT_MS` | No | `500` | |
| `REDIS_POOL_TIMEOUT_MS` | No | `1000` | |
| `REDIS_MAX_RETRIES` | No | `1` | |
| `CACHE_ASYNC_SET_BUDGET_MS` | No | `100` | |
| `CACHE_ASYNC_SET_MAX_INFLIGHT` | No | `64` | |

Per-area cache flags (all non-secret booleans; dev values from the local `.env`):

| Key | Dev | Prod (current) |
|---|---|---|
| `CACHE_ENABLED` | `true` | `true` |
| `CACHE_SELLER_VALIDATION` | `true` | `true` |
| `CACHE_CURRENCY` | `true` | `true` |
| `CACHE_PRODUCT_DETAIL` | `true` | `true` |
| `CACHE_CATEGORY` | `false` | `false` |
| `CACHE_GEO` | `true` | `true` |
| `CACHE_SELLER_SETTINGS` | `true` | `true` |
| `CACHE_GATEWAY_CATALOG` | `true` | `true` |
| `CACHE_FILE_REF` | `true` | `true` |
| `CACHE_INVENTORY_AVAIL` | `true` | `true` |
| `CACHE_PRODUCT_LIST` | `true` | `true` |
| `CACHE_SET_WRITES` | `true` | `true` |

### Backend — durable KV (Redis)

| Key | Secret? | Dev value | Notes |
|---|---|---|---|
| `KV_HOST` | No | `localhost` | `10.0.0.208` (VM-1 private IP) in production |
| `KV_PORT` | No | `6379` | `6381` in production |
| `KV_PASSWORD` | **Yes** | placeholder | Falls back to `CACHE_PASSWORD` if unset |
| `KV_ADDR` | No | unset | Optional full `host:port` override; compose sets it |
| `KV_MAXMEMORY` | No | unset | Compose-only (`128mb` on the VMs); not read by the app |

### Backend — application behaviour

| Key | Secret? | Dev value | Notes |
|---|---|---|---|
| `PUBLIC_API_BASE_URL` | No | `http://localhost:8080` | Origin only, no trailing slash; on the VMs it is `http://130.210.26.64` until TLS lands (`https://api.vitrapay.io`); used for courier webhook URLs |
| `ENCRYPTION_KEY` | **Yes** | placeholder | 32-byte key protecting stored gateway/courier credentials; never the dev default |
| `WORKER_POOL_SIZE` | No | `1` | `5` in production |
| `MAX_WISHLIST_ITEMS` | No | `100` | |
| `MAX_WISHLISTS_PER_USER` | No | `10` | |
| `PAYMENT_PENDING_TTL_MINUTES` | No | `45` | |
| `PAYMENT_SESSIONLESS_TTL_MINUTES` | No | `5` | |
| `REFUND_STUCK_TTL_MINUTES` | No | `30` | |
| `RECENTLY_VIEWED_QUEUE_SIZE` | No | `10` | |
| `RESET_TOKEN_EXPIRY_HOURS` | No | `1` | |
| `RESET_TOKEN_BYTE_LENGTH` | No | `32` | |

### Backend — messaging (RabbitMQ)

| Key | Secret? | Dev value | Notes |
|---|---|---|---|
| `MESSAGING_ENABLED` | No | `true` | `false` in the current production deploy (Noop publisher/consumer) |
| `MESSAGING_QUEUE_TYPE` | No | `rabbitmq` | |
| `MESSAGING_COMMANDS_EXCHANGE` | No | `ecom.commands` | |
| `MESSAGING_EVENTS_EXCHANGE` | No | `ecom.events` | |
| `MESSAGING_PREFETCH` | No | `10` | |
| `MESSAGING_CONSUMER_CONCURRENCY` | No | `5` | |
| `MESSAGING_RETRY_DELAY_MS` | No | `10000` | |
| `MESSAGING_MAX_RETRIES` | No | `5` | |
| `RABBITMQ_HOST` | No | `localhost` | Used when `RABBITMQ_URL` is not set |
| `RABBITMQ_PORT` | No | `5672` | |
| `RABBITMQ_VHOST` | No | `/` | |
| `RABBITMQ_TLS` | No | `false` | |
| `RABBITMQ_USER` | No | `guest` | Treat as sensitive in production |
| `RABBITMQ_PASSWORD` | **Yes** | placeholder | |
| `RABBITMQ_URL` | **Yes** | placeholder | Embeds credentials; prefer discrete fields in Config, use URL only as a secret override |

### Backend — provider endpoints

| Key | Secret? | Dev value | Notes |
|---|---|---|---|
| `RAZORPAY_BASE_URL` | No | `https://api.razorpay.com/v1` | Optional override; API keys/secrets are stored encrypted in the DB, not env |
| `SHIPROCKET_BASE_URL` | No | `https://apiv2.shiprocket.in` | Optional override; courier credentials are stored encrypted in the DB, not env |

Provider credentials (Razorpay key/secret, Shiprocket email/password/token) are **seller-configured rows encrypted with `ENCRYPTION_KEY`**; they are not environment variables and must never be committed to Config.

### Frontend — storefront

| Key | Secret? | Dev value | Production value |
|---|---|---|---|
| `NEXT_PUBLIC_API_URL` | No | `http://localhost:8080/api` | `https://api.vitrapay.io/api` |
| `NEXT_PUBLIC_SELLER_ID` | No | `2` | Real seller ID for the storefront tenant |
| `NEXT_PUBLIC_SITE_NAME` | No | `E-Shop` | Brand name |

### Frontend — seller dashboard

| Key | Secret? | Dev value | Production value |
|---|---|---|---|
| `NEXT_PUBLIC_API_URL` | No | `http://localhost:8080/api` | `https://api.vitrapay.io/api` |

The seller dashboard derives `X-Seller-ID` from the logged-in seller, so it has no seller-ID build variable. Frontends contain **no secrets**; only `NEXT_PUBLIC_*` values are allowed.

### Secret inventory (names only)

| Secret | Used by | Dev source | Production source |
|---|---|---|---|
| `DB_PASSWORD` | Backend | Developer `.env` | Server `.env` / secret store |
| `JWT_SECRET` | Backend | Developer `.env` | Server `.env` / secret store |
| `ENCRYPTION_KEY` | Backend | Developer `.env` | Server `.env` / secret store |
| `CACHE_PASSWORD` | Backend | Developer `.env` | Server `.env` / secret store |
| `KV_PASSWORD` | Backend | Developer `.env` (optional; falls back to cache password) | Server `.env` / secret store |
| `RABBITMQ_PASSWORD` / `RABBITMQ_URL` | Backend (only when messaging is enabled) | Developer `.env` | Server `.env` / secret store |
| Provider API keys/tokens | Backend (seller-configured) | DB rows encrypted with `ENCRYPTION_KEY` | DB rows encrypted with `ENCRYPTION_KEY` |

### Excluded keys

- `TEST_*` (`TEST_PG_DSN`, `TEST_USE_EXTERNAL`, `TEST_KV_DUAL`, Testcontainers variables) and `KV_BACKEND` (test integration setup) — test-only, never in Config.
- Vendor/CI variables (`AWS_*`, `AZURE_*`, `GOOGLE_*`, `DOCKER_*`, `SONIC_*`, `CLICOLOR*`) — not application configuration for this deployment; the normal storage path keeps provider credentials in the DB.
- `CACHE_MAXMEMORY`, `KV_MAXMEMORY`, `DRAGONFLY_CACHE_PORT`, `DRAGONFLY_KV_PORT` — listed above for completeness, but they are compose-level only; they belong in the deployment compose files, not in `backend.env`.

## Local Environment (`local`)

The `local` files mirror the current laptop setup so a developer can compare and switch without guesswork. Secret lines use an explicit placeholder that the renderer must resolve; an unresolved placeholder fails validation. `local` uses `APP_ENV=local`; the deployed environment is `dev` (`APP_ENV=dev`, VM private IPs).

```dotenv
# local/backend.env — non-secret values + secret placeholders
APP_ENV=local
GIN_MODE=debug
PORT=8080
LOG_LEVEL=debug
EXTENDED_LOGGING=false

DB_HOST=localhost
DB_PORT=5432
DB_NAME=ecommerce_test
DB_USER=postgres
DB_SSLMODE=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=25
DB_CONN_MAX_LIFETIME_MINUTES=5
DB_CONN_MAX_IDLE_TIME_MINUTES=5

JWT_EXPIRY_HOURS=72

CACHE_HOST=localhost
CACHE_PORT=6379
# CACHE_ADDR=localhost:6379        # optional full host:port override
KV_HOST=localhost
KV_PORT=6379
# KV_ADDR=localhost:6379           # optional full host:port override
REDIS_POOL_SIZE=20
REDIS_MIN_IDLE_CONNS=5
REDIS_DIAL_TIMEOUT_MS=5000
REDIS_READ_TIMEOUT_MS=500
REDIS_WRITE_TIMEOUT_MS=500
REDIS_POOL_TIMEOUT_MS=1000
REDIS_MAX_RETRIES=1
CACHE_ASYNC_SET_BUDGET_MS=100
CACHE_ASYNC_SET_MAX_INFLIGHT=64

CACHE_ENABLED=true
CACHE_SELLER_VALIDATION=true
CACHE_CURRENCY=true
CACHE_PRODUCT_DETAIL=true
CACHE_CATEGORY=false
CACHE_GEO=true
CACHE_SELLER_SETTINGS=true
CACHE_GATEWAY_CATALOG=true
CACHE_FILE_REF=true
CACHE_INVENTORY_AVAIL=true
CACHE_PRODUCT_LIST=true
CACHE_SET_WRITES=true

WORKER_POOL_SIZE=1
PUBLIC_API_BASE_URL=http://localhost:8080

MAX_WISHLIST_ITEMS=100
MAX_WISHLISTS_PER_USER=10
PAYMENT_PENDING_TTL_MINUTES=45
PAYMENT_SESSIONLESS_TTL_MINUTES=5
REFUND_STUCK_TTL_MINUTES=30
RECENTLY_VIEWED_QUEUE_SIZE=10
RESET_TOKEN_EXPIRY_HOURS=1
RESET_TOKEN_BYTE_LENGTH=32

# Provider endpoints (optional overrides; provider credentials live encrypted in the DB)
RAZORPAY_BASE_URL=https://api.razorpay.com/v1
SHIPROCKET_BASE_URL=https://apiv2.shiprocket.in

MESSAGING_ENABLED=true
MESSAGING_QUEUE_TYPE=rabbitmq
MESSAGING_COMMANDS_EXCHANGE=ecom.commands
MESSAGING_EVENTS_EXCHANGE=ecom.events
MESSAGING_PREFETCH=10
MESSAGING_CONSUMER_CONCURRENCY=5
MESSAGING_RETRY_DELAY_MS=10000
MESSAGING_MAX_RETRIES=5
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_VHOST=/
RABBITMQ_TLS=false
RABBITMQ_USER=guest

# Secrets are injected from the developer's environment; never commit values.
DB_PASSWORD=__REQUIRED_SECRET__
JWT_SECRET=__REQUIRED_SECRET__
ENCRYPTION_KEY=__REQUIRED_SECRET__
CACHE_PASSWORD=__REQUIRED_SECRET__
KV_PASSWORD=__REQUIRED_SECRET__
RABBITMQ_PASSWORD=__REQUIRED_SECRET__
# RABBITMQ_URL=__REQUIRED_SECRET__   # optional; embeds credentials — prefer the discrete fields
```

> For a passwordless local Redis, set `CACHE_PASSWORD=` and `KV_PASSWORD=` empty instead of the placeholder; production always requires values. `KV_PASSWORD` may be omitted locally because it falls back to `CACHE_PASSWORD`.

Local database setup: `make migrate` / `./run_migrations.sh --all` loads schema + CORE + MOCK seeds (demo users/products). If you do not want mock data, use `--migrate` (schema only) or `--production` (schema + CORE seeds); production always uses `--production`.

```dotenv
# local/storefront.env
NEXT_PUBLIC_API_URL=http://localhost:8080/api
NEXT_PUBLIC_SELLER_ID=2
NEXT_PUBLIC_SITE_NAME=E-Shop
```

```dotenv
# local/seller-dashboard.env
NEXT_PUBLIC_API_URL=http://localhost:8080/api
```

Developer workflow:

1. Copy the relevant file to a local `.env`/`.env.local`.
2. Replace `__REQUIRED_SECRET__` with local values (generated or low-risk dev values).
3. Never commit the local file; `.gitignore` covers `.env*` in the application repos.

## Deployed Environment (`dev`)

The `dev` files hold the Oracle VM values (`APP_ENV=dev`, VM private IPs) with the **same key set** as `local`; the differences are visible in git, not hidden on a server.

| Key group | Production values (non-secret) |
|---|---|
| Runtime | `APP_ENV=dev`, `GIN_MODE=release`, `PORT=8080`, `LOG_LEVEL=info` |
| PostgreSQL | `DB_HOST=10.0.0.208`, `DB_PORT=5432`, `DB_NAME=ecommerce`, `DB_USER=postgres`, `DB_SSLMODE=disable`, `DB_MAX_OPEN_CONNS=20`, `DB_MAX_IDLE_CONNS=5`, `DB_CONN_MAX_LIFETIME_MINUTES=30` |
| Auth | `JWT_EXPIRY_HOURS=24`; `JWT_SECRET` from secrets |
| Volatile cache | `CACHE_HOST=cache-volatile`, `CACHE_PORT=6379`; `CACHE_PASSWORD` from secrets |
| Durable KV | `KV_HOST=10.0.0.208`, `KV_PORT=6381`; `KV_PASSWORD` from secrets |
| App | `PUBLIC_API_BASE_URL=https://api.vitrapay.io`, `WORKER_POOL_SIZE=5`; `ENCRYPTION_KEY` from secrets |
| Messaging | `MESSAGING_ENABLED=false` (current deploy) |
| Frontends | `NEXT_PUBLIC_API_URL=https://api.vitrapay.io/api`, real `NEXT_PUBLIC_SELLER_ID`, brand `NEXT_PUBLIC_SITE_NAME` |

Until the deployed files are adopted (014 T071), the server-side `.env` files remain the source of truth. Their key set matches `local/backend.env` so adoption is a value move, not a redesign.

## Composition & Deployment

### Backend

```text
Zatu-ecom/Config @ <commit>
        │  <env>/backend.env
        ▼
  render-backend-env.sh ── + secrets from environment/secret store
        │  validate required/backend.keys
        ▼
~/ecommerce-data/.env  +  ~/ecommerce-app/.env   (mode 600)
        │
        ├── VM-1: migrations (run_migrations.sh --production)
        └── VM-2: docker compose up -d
```

- The deployment pins a Config commit SHA, so a running environment is traceable to an exact config revision.
- Missing required key or unresolved secret → **fail before touching the service**; never fall back to stale values.
- Precedence: secret value > Config non-secret value. Emergency environment overrides are allowed but must be recorded and reconciled in Config.
- After rendering, verify the server `.env` contains no `__REQUIRED_SECRET__` placeholders and mode is `600`.

### Frontends (Vercel)

- Read `<env>/storefront.env` and `<env>/seller-dashboard.env`.
- Set the values in each Vercel project under **Production** and **Preview** scopes.
- First deploy mirrors values manually; a later automation can call the Vercel API.
- Record the Config commit SHA in the deployment notes/record.
- Never set a server-side secret in a `NEXT_PUBLIC_*` variable; frontends only receive public values.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - All non-secret configuration is in one reviewable place (Priority: P1)

A developer opens `Zatu-ecom/Config` and finds every non-secret value for the `dev` environment — backend, storefront, and seller dashboard — in clearly named files, with the full key inventory covered.

**Why this priority**: This is the entire point of the repository; without it nothing else matters.

**Independent Test**: Pick any service and environment; every non-secret key the service reads is present exactly once, with a comment describing it.

**Acceptance Scenarios**:

1. **Given** a service and environment, **When** the developer opens the corresponding config file, **Then** all non-secret keys from the inventory are present.
2. **Given** a backend key, **When** it is searched in the repo, **Then** exactly one authoritative definition exists (no copy-paste duplicates).
3. **Given** the README, **When** a newcomer reads it, **Then** they can identify which file feeds which service/environment, which keys are secret, and how to run validation.

---

### User Story 2 - Deployments compose config + secrets (Priority: P1)

At deploy time, the deployment process reads the Config repo for the target environment, merges in secret values from the environment/secret store, and produces the runtime configuration consumed by the service — without manual edits on the server.

**Why this priority**: A config repo that still requires hand-copying values defeats its purpose.

**Independent Test**: Deploy an environment from scratch using only the Config repo plus a defined set of secret names; the service starts with no missing-variable errors.

**Acceptance Scenarios**:

1. **Given** the Config repo and the required secrets, **When** the deployment runs, **Then** the backend starts with all keys resolved and the rendered `.env` has no placeholders.
2. **Given** a missing required secret, **When** the deployment runs, **Then** it fails before starting the service and names the missing key.
3. **Given** a config change merged to the repo, **When** the next deployment runs, **Then** the new value is applied without editing any file on the server.
4. **Given** a frontend deployment, **When** Vercel builds, **Then** it consumes the Config's `NEXT_PUBLIC_*` values for the target environment.

---

### User Story 3 - Secrets never enter the repository (Priority: P1)

The Config repo contains only non-secret values and placeholders. Real passwords, JWT keys, encryption keys, API tokens, and credentials are provided through the environment or a secret store.

**Why this priority**: A leaked repo must not leak production credentials.

**Independent Test**: Scan the repository and its git history for known secret patterns; only placeholders/references are found.

**Acceptance Scenarios**:

1. **Given** a file in the Config repo, **When** it is inspected, **Then** it contains no real secret value — only `__REQUIRED_SECRET__` or the *name* of the secret.
2. **Given** a deployment, **When** secrets are injected, **Then** their source is an environment/secret store, never the repo.
3. **Given** an accidental secret commit, **When** detected, **Then** the value is rotated and removed from history.

---

### User Story 4 - Adding a new environment needs no code changes (Priority: P2)

A new environment is added by adding config files (and secret values), without modifying application code or deployment scripts.

**Why this priority**: The layout must scale if more than the current `dev` environment is ever needed.

**Independent Test**: Create a new env folder by copying `dev`, change values, deploy — the service starts with the new environment.

**Acceptance Scenarios**:

1. **Given** a new environment name, **When** its config files exist and secrets are provided, **Then** deployment succeeds with no code change.
2. **Given** two environments, **When** they are compared, **Then** differences are visible in git, not hidden in server files.
3. **Given** a new required key, **When** it is added to the key list, **Then** every environment's file must define it before deploy.

---

### User Story 5 - Changes are reviewable and reversible (Priority: P2)

Every configuration change goes through a pull request, is reviewed, and can be reverted like code. Deployments record which Config revision they used.

**Why this priority**: Config mistakes cause outages; review plus traceability shortens recovery.

**Independent Test**: Change a value, open a PR, deploy, inspect the deployment record — the Config commit is recorded; revert it and redeploy to restore the previous value.

**Acceptance Scenarios**:

1. **Given** a config PR, **When** it is merged and deployed, **Then** the deployment record includes the Config commit SHA.
2. **Given** a bad value, **When** it is reverted and redeployed, **Then** the previous behavior is restored.
3. **Given** a deployment record, **When** the operator inspects it, **Then** they can identify the exact config revision used.

---

### User Story 6 - Drift is detectable (Priority: P3)

The operator can compare the running service's effective configuration with the Config repo and see differences.

**Why this priority**: Nice-to-have once the process is stable; manual inspection is acceptable initially.

**Independent Test**: Change a value on a server manually, run the drift check, and observe a reported difference.

**Acceptance Scenarios**:

1. **Given** a running service, **When** the drift check runs, **Then** keys that differ from the Config repo are reported (values masked for secrets).
2. **Given** a secret key, **When** drift is reported, **Then** only presence/absence is shown, never the value.

---

### Edge Cases

- Missing key in one environment only (caught by validation).
- Secret name typo or unresolved `__REQUIRED_SECRET__` placeholder (fail-fast with the key name).
- Values containing spaces, `#`, `$`, quotes, or multi-line content (quoting rules documented).
- Frontend build-time vs runtime variables (`NEXT_PUBLIC_*` must be present at build time).
- Trailing slashes in URLs (`PUBLIC_API_BASE_URL` must not end with `/`; `NEXT_PUBLIC_API_URL` must include `/api`).
- A service deployed from a branch that expects a config key not yet added (deploy fails early, not at runtime).
- Config repo temporarily unreachable during deploy (deploy fails; never fall back to stale values silently).
- A developer's local `.env` has keys that no longer exist in Config (warn and reconcile; don't silently ignore).
- Secrets in git history from an earlier mistake (rotation + history rewrite policy).
- `NEXT_PUBLIC_SELLER_ID` points at the wrong tenant (visible data mismatch; validate during smoke test).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The Config repo MUST organize values **per service** (`backend`, `storefront`, `seller-dashboard`) and **per environment** (`local` for laptops, `dev` for the deployed Oracle VMs) using `<env>/<service>.env`.
- **FR-002**: All **non-secret** values consumed by a service MUST be committed in its environment file, covering the inventory in this spec.
- **FR-003**: Secret values MUST NOT be committed; files MAY reference secret **names/placeholders** only (`__REQUIRED_SECRET__`).
- **FR-004**: The backend deployment MUST compose the environment file + secrets into the server-side `.env` (or equivalent) automatically, with no manual server edits.
- **FR-005**: Frontend deployments MUST consume only public (`NEXT_PUBLIC_*`) non-secret values from Config (or mirrored platform settings); any other secrets MUST come from the platform secret store.
- **FR-006**: Each service/environment MUST have a validation list of required keys; deploys MUST fail early and name missing keys.
- **FR-007**: A secret scan MUST run on the repo (and ideally in CI) to prevent accidental secret commits.
- **FR-008**: The deployment record MUST include the Config commit SHA used (together with the application build revision).
- **FR-009**: The repo MUST support at least `local` and `dev` with identical key sets (values may differ); adding an environment MUST NOT require code changes.
- **FR-010**: Rollback MUST be possible by reverting a Config commit (or promoting a previous tag) and redeploying.
- **FR-011**: Access to the Config repo MUST be restricted; deployment credentials MUST be read-only.
- **FR-012**: A README MUST document file layout, required keys, secret names, quoting rules, and the validation/render commands.
- **FR-013**: `local/backend.env` MUST cover every non-secret key from the local `.env`, `.env.test`, `common/config/*.go`, and `deploy/.env.example` inventory in this spec.
- **FR-014**: The secret inventory MUST list every required secret by name; values MUST come from the environment/secret store.
- **FR-015**: `dev/*` (the deployed environment) MUST keep the same key set as `local/*`; adding an environment is a folder, not a redesign.
- **FR-016**: `NEXT_PUBLIC_API_URL` MUST include the `/api` path prefix; `PUBLIC_API_BASE_URL` MUST be origin-only with no trailing slash.
- **FR-017**: The storefront MUST define `NEXT_PUBLIC_SELLER_ID` per environment; the seller dashboard MUST derive seller ID from the login session.
- **FR-018**: Test-only (`TEST_*`) and vendor/CI variables MUST NOT be added to the Config repo.
- **FR-019**: Compose-level variables (`CACHE_MAXMEMORY`, `KV_MAXMEMORY`, Dragonfly ports) MUST be documented separately from `backend.env`.
- **FR-020**: A deployment MUST record both the application revision and the Config revision; neither alone is sufficient.
- **FR-021**: Provider credentials (Razorpay, Shiprocket, storage) MUST remain encrypted DB rows or secret-store values — never Config repo values.

### Non-Functional Requirements

- **NFR-001**: No secret value may appear in git history, build logs, or deploy output (mask on display).
- **NFR-002**: Configuration must be reproducible: same Config commit + same secrets ⇒ same runtime configuration.
- **NFR-003**: Config changes take effect on the next deploy; no hot reload is required.
- **NFR-004**: The repo must stay small and human-readable (plain env files; no generated blobs).
- **NFR-005**: The process must work with the current deployment (manual image ship) and the future GitHub Actions deploy without redesign.
- **NFR-006**: Validation and rendering must be deterministic and fail closed; a missing key is a deploy blocker, not a runtime surprise.

## Out of Scope

- Choosing/implementing the secret store (GitHub Secrets vs OCI Vault vs platform env vars) — this spec requires a source, not a specific product.
- Dynamic/hot configuration reload at runtime.
- A feature-flag service (LaunchDarkly-style).
- Per-developer local overrides in the repo — developers may keep local `.env` files; the repo is the shared source.
- Database schema/migrations (see spec 014).
- Application code changes to read YAML/JSON instead of env vars.
- Provider credential storage redesign (already DB-encrypted with `ENCRYPTION_KEY`).

## Dependencies & Blockers

| Dependency | Status | Impact |
|---|---|---|
| `Zatu-ecom/Config` repository access | ✅ exists (private) | Layout, inventory, and scripts must be created; deploy needs a read-only credential |
| Backend deployment procedure (spec 014) | ✅ live | Must be extended to pull Config at deploy time (T071) |
| Frontend repos + Vercel Hobby (spec 015) | ✅ resolved | Vercel env settings must mirror Config |
| Secret source for backend | ❓ undecided | Server `.env` today; GitHub Secrets/OCI Vault later |
| Secret scan tooling | ❓ undecided | Pre-commit hook and/or CI job |
| Config repo owner/reviewer | ❓ undecided | PR review before production config changes |

## Bootstrap Checklist (initial `dev` commit)

- [x] Create `local/backend.env` from the inventory/sample above (non-secret values + `__REQUIRED_SECRET__` placeholders).
- [x] Create `local/storefront.env` and `local/seller-dashboard.env` with `NEXT_PUBLIC_*` values only.
- [x] Create `required/backend.keys`, `required/storefront.keys`, and `required/seller-dashboard.keys` (backend includes secret names).
- [x] Add `scripts/validate.sh` (required keys + placeholder check) and `scripts/render-backend-env.sh` (Config + secrets → `.env`) — both validated locally.
- [x] Add `README.md` documenting layout, key reference, secret names, quoting rules, and validation/render commands.
- [x] Add `.gitignore` (ignores `*.local`, `*.secret`, `.env`, `.env.*`); **secret-scan pre-commit hook / CI job still pending**.
- [ ] Commit the initial config and record the commit SHA for 014 T071 — **pending (commit + push the Config repo)**.
- [x] `dev/` (deployed) files added with the same key set as `local/` (parity verified).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of non-secret keys used by backend + both frontends are defined once in the Config repo (verified against the inventory).
- **SC-002**: A fresh deploy of any supported environment succeeds using only Config + named secrets (no hand edits).
- **SC-003**: Zero real secret values exist in the repo or its history (verified by scan).
- **SC-004**: Any deployment can be traced to an application revision **and** a Config commit SHA.
- **SC-005**: A configuration rollback (revert + redeploy) restores previous behavior in under 10 minutes.
- **SC-006**: `local` and `dev` have identical key sets: `comm -3 <(cut -d= -f1 local/backend.env | sort) <(cut -d= -f1 dev/backend.env | sort)` is empty (same check for each frontend file).
- **SC-007**: A deploy with a missing required key fails before the service starts and names the key.

## Open Questions

Resolved on 2026-11-22:

1. **File layout** — `<env>/<service>.env` (one folder per environment).
2. **Secret source for the backend** — server `.env`/secret store at deploy time; the Config repo only names secrets.
3. **Vercel sync** — mirror values manually for the first deploy; automate via the Vercel API later.
4. **Dev secrets** — each developer supplies local values in their own `.env`; the repo uses `__REQUIRED_SECRET__` placeholders.
5. **History policy** — rotate any leaked secret and rewrite history if it was pushed.

Still open:

1. **Ownership/approvals** — who reviews Config PRs before a production deploy?
2. **Secret store product** — GitHub Secrets vs OCI Vault vs server-side `.env` for the first production phase?
3. **Drift tooling** — shell script vs CI job vs a small Go command?
4. **Prod template timing** — commit `prod` now with placeholders, or only when the production environment is formalized?

## Assumptions

- The Config repo is **private** and readable by deployments with a read-only credential.
- Secrets are generated/rotated independently and stored in a secret manager or environment configuration.
- Configuration changes are applied by redeploying; no runtime reload is needed.
- The backend remains a single production environment for now; `prod` files are added when that environment is formalized.
- Plain POSIX env files are sufficient (no YAML/JSON templating required initially).
- Local `.env` files remain developer overrides; they are never the shared source of truth.
- The current server `.env` key set will be reconciled to `local/backend.env` before T071 switches the deploy to Config.
