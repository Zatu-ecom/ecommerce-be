# Feature Specification: Frontend Cloud Deployment (Storefront + Seller Dashboard)

**Feature Branch**: `015-frontend-cloud-deployment`
**Created**: 2026-11-17
**Updated**: 2026-11-22
**Status**: Draft — inputs resolved; blocked on backend TLS (014 T052)
**Input**: User description: "Deploy both frontends on a free host outside Oracle Cloud — the two Oracle VMs do not have enough disk/RAM. Backend stays where it is."
**Related docs**: `specs/014-backend-cloud-deployment/spec.md` (backend, live at `http://130.210.26.64`), `deploy/ORACLE-SETUP-TODO.md`

## Background

The backend is live on Oracle Cloud (spec 014) at `http://130.210.26.64`. Two Next.js frontends remain:

1. **Storefront** — customer-facing shop (browse, cart, checkout, account, orders).
2. **Seller dashboard** — seller-facing admin (products, inventory, orders, fulfillment, payments).

They are **not** deployed to the Oracle VMs by decision: both are `VM.Standard.E2.1.Micro` hosts (~1 GB RAM, 50 GB boot each) already carrying the app/database stack, and a Node.js frontend build/runtime would not fit comfortably. Frontends go to a **free managed platform** instead.

**Resolved inputs (2026-11-22):**

| Item | Decision |
|---|---|
| Backend domain | **`api.vitrapay.io`** (subdomain of the existing `vitrapay.io` domain, A record → `130.210.26.64`) |
| Frontend repos | `https://github.com/Zatu-ecom/storefront`, `https://github.com/Zatu-ecom/seller-dashboard` (both cloned locally) |
| Hosting | **Vercel Hobby** account created |
| Auth transport | **Bearer token** in the `Authorization` header — verified locally with both frontends calling the API |
| Shared configuration | `https://github.com/Zatu-ecom/Config` — non-secret values per environment; secrets stay in environment/platform secrets (dedicated spec: `specs/016-config-repository`) |
| Optional frontend domains | `shop.vitrapay.io` (storefront), `seller.vitrapay.io` (dashboard) — P3, optional |

**Critical constraint — HTTPS:** free frontend hosts serve over HTTPS. Browsers block an HTTPS page from calling an HTTP API (mixed content). Therefore **the backend must serve TLS on `https://api.vitrapay.io` before either frontend can be used in a browser** — this spec depends on spec 014 task **T052** (DNS A record + certbot).

**Current backend facts relevant to frontends:**

| Fact | Value |
|---|---|
| API base URL (today) | `http://130.210.26.64/api` (HTTP only; not usable from an HTTPS page) |
| API base URL (target) | **`https://api.vitrapay.io/api`** (origin + `/api` prefix, after 014 T052) |
| API contract | Every request needs `X-Correlation-ID`; protected routes need `Authorization: Bearer <jwt>` |
| CORS | Wildcard `Access-Control-Allow-Origin: *` today (to be tightened) |
| Uploads | nginx allows 100 MB request bodies |
| Image hosting | Seller-configured object storage providers (no host-level CDN) |
| Config source | `Zatu-ecom/Config` (non-secret) + platform/env secrets (spec 016) |

**Frontend repositories (verified 2026-11-22):**

| App | Repository | Local clone | Stack | Package manager |
|---|---|---|---|---|
| Storefront | `https://github.com/Zatu-ecom/storefront` | ✅ cloned locally | Next.js 16.2 / React 19 | npm (`package-lock.json`) |
| Seller dashboard | `https://github.com/Zatu-ecom/seller-dashboard` | ✅ cloned locally | Next.js 16.2 / React 19 | npm (`package-lock.json`) |

**Vercel project setup (Hobby account created):**

| Setting | Storefront | Seller dashboard |
|---|---|---|
| Vercel project | `storefront` | `seller-dashboard` |
| Git repository | `Zatu-ecom/storefront` | `Zatu-ecom/seller-dashboard` |
| Framework preset | Next.js | Next.js |
| Root directory | repository root | repository root |
| Install command | `npm ci` | `npm ci` |
| Build command | `npm run build` | `npm run build` |
| Output | Next.js default (`.next`) | Next.js default (`.next`) |
| Node.js | 20.x or 22.x | 20.x or 22.x |
| Env scopes | Production + Preview | Production + Preview |

**Authentication flow (resolved 2026-11-22):** both frontends use bearer-token auth, verified locally with both apps calling the local backend:

1. `POST /api/user/auth/login` returns a JWT.
2. The frontend stores the JWT in its auth store (client-side); no cookies are used.
3. Every protected request sends `Authorization: Bearer <jwt>` plus the mandatory `X-Correlation-ID`; the storefront also sends `X-Seller-ID` and `X-Device-ID`, and the dashboard sends `X-Seller-ID` from the seller store.
4. On `401`, the client calls `POST /api/user/auth/refresh` with the current bearer token, retries the original request once, and on failure clears the session and redirects to login.
5. The backend CORS policy allows `Authorization`, `X-Correlation-ID`, `X-Seller-ID`, `X-Device-ID`, and `Idempotency-Key`; no `Access-Control-Allow-Credentials` is needed because no cookies are used.
6. Production changes only the API base URL (`https://api.vitrapay.io/api`) and the allowed origins; the auth contract is unchanged.

**Frontend configuration keys (non-secret; source: `Zatu-ecom/Config`, spec 016):**

| App | Key | Dev value | Production value |
|---|---|---|---|
| Storefront | `NEXT_PUBLIC_API_URL` | `http://localhost:8080/api` | `https://api.vitrapay.io/api` |
| Storefront | `NEXT_PUBLIC_SELLER_ID` | `2` (local mock seller) | real seller ID for the storefront tenant |
| Storefront | `NEXT_PUBLIC_SITE_NAME` | `E-Shop` | `E-Shop` (or brand name) |
| Seller dashboard | `NEXT_PUBLIC_API_URL` | `http://localhost:8080/api` | `https://api.vitrapay.io/api` |

The API base URL **must include the `/api` prefix** — the backend routes are `/api/user`, `/api/product`, etc. A URL without `/api` returns 404s. The seller dashboard derives the seller ID from the logged-in seller, so it needs no seller-ID build variable.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Storefront is publicly reachable (Priority: P1)

A shopper opens the storefront URL on the free host, browses products, and can log in. The site calls the backend API configured at build time.

**Why this priority**: The storefront is the customer-facing product; without it the backend delivers no business value.

**Independent Test**: Open the production storefront URL on a phone and a laptop; product listing loads from the API and a login attempt reaches the backend (valid or invalid credentials both prove connectivity).

**Acceptance Scenarios**:

1. **Given** the storefront is deployed, **When** a visitor opens the production URL, **Then** the landing page renders with products fetched from the API.
2. **Given** a branch preview deployment, **When** the branch URL is opened, **Then** it uses the same API base URL and renders the same data.
3. **Given** an invalid API URL, **When** the site loads, **Then** the failure is visible in logs and no mixed-content errors occur.

---

### User Story 2 - Seller dashboard is publicly reachable (Priority: P1)

A seller opens the dashboard URL, logs in with seller credentials, and sees products/orders/fulfillment data from the API.

**Why this priority**: Sellers cannot operate the store without the dashboard.

**Independent Test**: Log in with a seller account created through the normal registration/admin flow (never a mock seed), load the orders list and a fulfillment read (e.g. couriers), and confirm data matches the API.

**Acceptance Scenarios**:

1. **Given** the dashboard is deployed, **When** a seller logs in, **Then** a JWT is issued and authenticated reads succeed.
2. **Given** an unauthenticated visitor, **When** they open a protected page, **Then** they are redirected to login without exposing data.
3. **Given** the API is unreachable, **When** the dashboard loads, **Then** a clear error state appears (no blank screen).

---

### User Story 3 - Browser → API traffic is secure (Priority: P1, blocking)

Both frontends talk to the backend over HTTPS. The backend gets a TLS certificate for **`api.vitrapay.io`**. No mixed-content warnings; no plaintext credentials.

**Why this priority**: Without this, neither frontend works in a browser. It is the gating dependency.

**Independent Test**: Open the storefront/dashboard over HTTPS, watch the network tab — every API call is `https://` and succeeds; the certificate is valid.

**Acceptance Scenarios**:

1. **Given** the DNS `A` record `api.vitrapay.io → 130.210.26.64`, **When** certbot completes, **Then** `https://api.vitrapay.io/` serves the API with a valid certificate and auto-renew.
2. **Given** the frontends are configured with `https://api.vitrapay.io/api`, **When** they call the API, **Then** no mixed-content errors appear and CORS preflight succeeds.
3. **Given** certificate renewal, **When** it happens, **Then** no operator action or downtime is required.

---

### User Story 4 - Environments are configured safely (Priority: P1)

Each frontend has per-environment configuration (API base URL, public keys only). No server-side secret is ever shipped to the browser; environment variables are set in the hosting platform, not committed.

**Why this priority**: Wrong API URLs silently break deploys; leaking secrets to the client is a security incident.

**Independent Test**: Inspect the build output and platform settings — only public variables are present; changing the API URL requires a platform setting change, not a code edit.

**Acceptance Scenarios**:

1. **Given** a production deployment, **When** the build runs, **Then** `NEXT_PUBLIC_API_URL=https://api.vitrapay.io/api`, `NEXT_PUBLIC_SELLER_ID` is the real storefront seller ID, and `NEXT_PUBLIC_SITE_NAME` is the brand name; no other environment-dependent values are required.
2. **Given** a preview deployment, **When** it builds, **Then** it uses the same API (or a preview API if one exists later) and never a developer's localhost.
3. **Given** the repository, **When** scanned, **Then** no secret values are committed (`.env*` ignored).

---

### User Story 5 - A deploy can be rolled back quickly (Priority: P2)

If a frontend deploy breaks, the operator can revert to the previous deployment in the hosting platform with one action, without touching the backend.

**Why this priority**: Frontend regressions are common and should never require a backend redeploy.

**Independent Test**: Promote a known-good deployment after a bad one and confirm the production URL serves the previous build.

**Acceptance Scenarios**:

1. **Given** a bad production deploy, **When** the previous deployment is promoted, **Then** the production URL serves the previous build within minutes.
2. **Given** a rollback, **When** the frontend calls the API, **Then** version compatibility is preserved (same API contract).

---

### User Story 6 - Custom domains (Priority: P3, optional)

Each frontend can be attached to a friendly domain (`shop.vitrapay.io` for the storefront, `seller.vitrapay.io` for the dashboard) once DNS is available.

**Why this priority**: Cosmetic/branding; not required for functionality.

**Independent Test**: Add the domain in the hosting platform, create the DNS record, and confirm HTTPS issuance for the custom domain.

**Acceptance Scenarios**:

1. **Given** a domain and DNS control, **When** the custom domain is added, **Then** the frontend serves over HTTPS on that domain.
2. **Given** the API CORS allowlist, **When** the custom origin is used, **Then** preflight succeeds.

---

### User Story 7 - Shared configuration comes from the Config repo (Priority: P1)

Non-secret configuration (API URL, public keys, feature flags) lives in `Zatu-ecom/Config`, organized per service and per environment. A frontend build reads its environment's values from there (or from mirrored Vercel settings); secrets never live in the repo and are supplied only through Vercel/environment variables.

**Why this priority**: Today every value lives in a local `.env`; without a shared, reviewable source, dev/prod drift is inevitable and deployments become manual guesswork.

**Independent Test**: Change one non-secret value in the Config repo, deploy, and observe the new value in the running site — without touching application code or committing secrets.

**Acceptance Scenarios**:

1. **Given** the Config repo, **When** a frontend build runs for an environment, **Then** it consumes that environment's non-secret values.
2. **Given** a required secret, **When** the build runs, **Then** the value comes from Vercel/environment secrets and is absent from the repo.
3. **Given** a config change, **When** it is merged and the next deploy runs, **Then** the change takes effect and is traceable in git history.
4. **Given** a missing required key, **When** the deploy starts, **Then** it fails early with a clear error instead of shipping a broken build.

> Full design: `specs/016-config-repository/spec.md`.

---

### Edge Cases

- Free-tier build limits (minutes/concurrency) — a large Next.js build may queue; the spec assumes builds complete within free limits.
- Preview URLs are public by default — protect any environment that touches real data.
- API returns `400 CORRELATION_ID_REQUIRED` if the frontend omits the header — the API client must always send `X-Correlation-ID`.
- Backend restarts briefly during a deploy — frontends must show a retry/error state, not crash.
- `PUBLIC_API_BASE_URL` on the backend must match the HTTPS name so webhook URLs are correct.
- Cookie-based auth across origins is fragile; **not used here** — both frontends use bearer tokens in `Authorization` (verified locally). No `SameSite`/`Secure`/CORS-credentials design is required.
- The API base URL must include `/api`; using only the origin (`https://api.vitrapay.io`) makes every call 404 even though TLS and CORS succeed.
- `NEXT_PUBLIC_SELLER_ID` is a public tenant selector, not a secret; a wrong value silently shows another seller's catalog, so it must be validated during the smoke test.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Both frontends MUST be hosted on a free managed platform (Vercel Hobby recommended for Next.js; Cloudflare Pages or Netlify acceptable) — **not** on the Oracle VMs.
- **FR-002**: Each frontend MUST build and deploy independently from its own repository.
- **FR-003**: Production deployments MUST be manual/promoted; preview deployments MAY be automatic per branch/PR.
- **FR-004**: `NEXT_PUBLIC_API_URL` MUST be configurable per environment in the platform; production MUST use `https://api.vitrapay.io/api` (origin **plus** the `/api` prefix).
- **FR-005**: The API client MUST send `X-Correlation-ID` on every request and propagate `Authorization: Bearer <jwt>`, `X-Seller-ID`, and (storefront) `X-Device-ID` where required.
- **FR-006**: The backend MUST serve HTTPS for the API (dependency: 014 T052) before frontends are considered usable.
- **FR-007**: CORS MUST allow the deployed frontend origins; wildcard is acceptable only temporarily and MUST be replaced by an allowlist (014 T050). The allowlist MUST include `Authorization`, `X-Correlation-ID`, `X-Seller-ID`, `X-Device-ID`, `Idempotency-Key`, and `Content-Type`.
- **FR-008**: Every deploy MUST be reversible by promoting a previous deployment in the platform.
- **FR-009**: Errors from the API (401/403/404/5xx) MUST surface as user-visible states, not blank pages.
- **FR-010**: A smoke test MUST verify: storefront product list, seller login, and one authenticated read (e.g. couriers/orders).
- **FR-011**: Image/upload flows MUST respect the backend's 100 MB body limit and object-storage URLs returned by the API.
- **FR-012**: Neither frontend repository may contain backend secrets; only public build-time variables are permitted.
- **FR-013**: Both frontends MUST consume non-secret configuration from `Zatu-ecom/Config`, organized per service and per environment (`dev` now, `prod` when it exists).
- **FR-014**: The Config repo MUST contain placeholders/references only for secrets; real secret values MUST come from Vercel/environment secret stores.
- **FR-015**: Each deployment MUST record the Config revision used (commit SHA), so the running build is traceable to a config point.
- **FR-016**: Builds/deploys MUST fail early with a clear error when a required configuration key is missing.
- **FR-017**: The frontends MUST use bearer-token auth (`Authorization: Bearer <jwt>`) as verified locally: login stores the JWT client-side, protected calls attach it, and `401` triggers `POST /api/user/auth/refresh` with one retry before logout. No cookie-based cross-origin auth is used.
- **FR-018**: Each frontend MUST be a separate Vercel project built from its own repository, with `npm ci` + `npm run build`, Next.js preset, and Production/Preview environment scopes.
- **FR-019**: The storefront MUST set `NEXT_PUBLIC_SELLER_ID` per environment (the dashboard derives seller ID from the login session, not a build variable).
- **FR-020**: Preview deployments MUST NOT point at `localhost`; they use the same HTTPS API base URL (or a future preview API) and are protected from real-data writes if they expose preview URLs publicly.

### Non-Functional / Constraints

- **NFR-001**: Cost target `$0/mo` — free tiers only (hosting, build minutes, bandwidth within free limits).
- **NFR-002**: No additional load on the Oracle VMs (no Node runtime, no build jobs there).
- **NFR-003**: Frontend bundle must remain within free-tier limits; server-side rendering must not require a long-running self-hosted server.
- **NFR-004**: HTTPS end-to-end (browser → frontend and frontend → API).
- **NFR-005**: Deploys must not require downtime for the backend.

## Out of Scope

- Backend application changes beyond CORS allowlisting (014 T050) and TLS (014 T052).
- The `/version` endpoint (dropped in 014).
- Backend CI/CD or manual GitHub deploy for the API (tracked as future work in 014 tasks).
- Migrating the frontends onto Oracle, Kubernetes, or a paid CDN.
- Native mobile apps, PWA offline support, internationalization.
- Real payment-gateway go-live on the storefront (backend feature readiness is separate).

## Dependencies & Blockers

| Dependency | Status | Impact |
|---|---|---|
| Backend HTTPS on `api.vitrapay.io` (014 T052: DNS A record + certbot) | ⏳ pending | **Blocks browser use of both frontends** |
| CORS allowlist tightening (014 T050) | ⏳ pending | Wildcard `*` remains acceptable temporarily |
| Frontend repositories | ✅ resolved (`storefront`, `seller-dashboard`) | — |
| Vercel Hobby account | ✅ created | — |
| Domain `vitrapay.io` | ✅ owned | Add `A api → 130.210.26.64`; optional `shop`/`seller` records later |
| Config repo `Zatu-ecom/Config` | 📄 exists; layout + dev inventory defined in spec 016 | Bootstrap `envs/dev` files; mirror values into Vercel |
| Seller account for smoke test | ❌ none in prod | Create one real seller through the normal registration/admin flow (never via `seeds/mock`) |

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Storefront production URL loads over HTTPS and lists products from the API in under 3 seconds on a normal connection.
- **SC-002**: Seller dashboard production URL loads, and a seller can log in (bearer JWT), refresh the token on expiry, and read orders/couriers.
- **SC-003**: Zero mixed-content errors and zero `CORRELATION_ID_REQUIRED` errors in the browser console during the smoke test.
- **SC-004**: A bad deploy can be rolled back by promoting the previous deployment in under 5 minutes.
- **SC-005**: Both frontends consume zero Oracle VM disk/RAM (verified by host resource check before/after).
- **SC-006**: No secret values are present in either frontend build output.

## Open Questions

Resolved on 2026-11-22: hosting (**Vercel Hobby**), repositories (URLs above), API name (**`api.vitrapay.io`**, base URL `https://api.vitrapay.io/api`), auth (**bearer token**, verified locally), Config layout (**spec 016: `envs/<env>/<service>.env`**).

Still open:

1. **Preview data safety** — should Vercel preview deployments point at the production API, or be blocked from real data?
2. **Custom frontend domains** — add `shop.vitrapay.io` / `seller.vitrapay.io` now or after first production deploy?
3. **Test seller account** — who creates the first seller for the login smoke test, and in which environment?
4. **`PUBLIC_API_BASE_URL` switch** — schedule the VM-2 env change to `https://api.vitrapay.io` together with the certbot rollout.
5. **Vercel env sync** — mirror Config values into Vercel manually for the first deploy, or automate via the Vercel API later?

## Assumptions

- Both frontends are Next.js 16 / React 19 apps using npm, deployed on **Vercel Hobby** (account created).
- The backend API remains a single production environment for now; no separate staging API.
- The API contract is stable during the frontend rollout; breaking changes are out of scope.
- The `vitrapay.io` DNS is manageable (an `A` record can be added for `api`).
- Bearer-token auth continues to work as verified locally; no cookie-based cross-origin flow is introduced.
- The API base URL always includes the `/api` prefix; `PUBLIC_API_BASE_URL` on the backend stays origin-only.
- Free-tier hosting/build limits are acceptable for initial launch; paid tiers are out of scope.
- Vercel Hobby is used for the initial phase; if Vercel's commercial-use/fair-use terms require a paid plan once the store is live, upgrading is a billing decision outside this spec.
