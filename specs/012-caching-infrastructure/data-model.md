# Data Model: Caching Infrastructure

Binding: [pre-spec.md](./pre-spec.md) §5 (keys/TTLs — normative). No Postgres schema changes in this feature. "Entities" here are key-space records: name, encoding, lifetime, placement, and transitions.

Conventions: wire TTL is always `JitteredTTL(base)` (§5.6). `seller = seller_id` tenant scope. Values are JSON DTO bytes (never GORM entities), ≤256KB or the SET is skipped + counted.

## Volatile role (evictable, fail-open)

| Record | Key | Value | Base TTL | Transitions |
|---|---|---|---|---|
| Seller validation | `seller:{id}:seller:complete` | `SellerValidationResult` JSON | `min(5m, until SubscriptionEndDate)` if set else 5m; 60s tombstone | DB/user/subscription/plan write → exact `Del`; hard delete → `Del` |
| Seller currency | `seller:{id}:currency:default`, `seller:{id}:currency:user:{uid}` | `CurrencyResponse` JSON | 1h | settings/currency write → exact `Del` (no glob) |
| Geo entity | `geo:country:{id}`, `geo:currency:{id}` (public active-only) | country/currency JSON | 12h | admin geo/mapping write → `Del` |
| Geo active list | `geo:countries:active:v{ver}`, `geo:currencies:active:v{ver}` | full active set JSON (paged in process) | 12h | admin write → durable `INCR` version |
| Seller settings | `seller:{id}:settings` | settings DTO JSON | 5m | settings write → `Del` |
| Gateway catalog | `gateway:catalog:{code}`, `gateway:catalog:all:v{ver}` (public metadata only) | catalog JSON | 12h | admin/seed → `Del` + version bump |
| File refs | `file:providers`, `file:schema:{adapter}` | provider list / static schema JSON | 24h | seed/deploy → `Del` |
| Product detail | `seller:{id}:product:{pid}` | stable storefront DTO (§4.3: no stock, no signed URLs, no personalization) | 10m + 60s tombstone | product/nested write (§5.7) → `Del` + `productlist:ver` bump |
| Variant | `seller:{id}:variant:{vid}` | variant DTO JSON | 10m | variant/media write → `Del` + parent product `Del` |
| Category/attribute | `seller:{id}:category:{cid}`, `…:categories:all:v{ver}`, `…:categories:parent:{pid}:v{ver}`, `…:attribute:{aid}`, `…:attributes:all:v{ver}` | DTO/list JSON | 60m / 30m | write/link/unlink → `Del` + version bump |
| Collection by id | `seller:{id}:collection:{cid}` (optional P1) | collection DTO JSON | 10m | collection CRUD → `Del` |
| Product list (P2) | `seller:{id}:productlist:v{ver}:{hash}` | assembled page JSON ≤256KB | 1–2m | any product/nested write bumps `ver` |
| Collection products (P2) | `seller:{id}:collectionlist:v{ver}:{cid}:{hash}` | page JSON ≤256KB | 1–2m | membership write bumps `collectionlist:ver` |
| Admission marker (P2) | `seller:{id}:seen:{hash}` | `1` | ~60s | `SETNX` on first sight; repeat earns full SET |
| Availability (P2) | `seller:{id}:inv:avail:{hash}` | quantity summary JSON | 3–5s | TTL only; writes bypass |
| Tombstone | same key as the entry | `{"_neg":true}` (+ reason code) | 30–60s | any write `Del`s it (same key, uniform); create `Del`s it so refill can proceed |

List-query hash input: canonical params only (sorted keys, stable types, seller id included) — never the raw URL string.

## Durable role (non-evicting, snapshotted)

| Record | Key | Value | Lifetime | Transitions |
|---|---|---|---|---|
| Version counter | `seller:{id}:productlist:ver`, `…:collectionlist:ver`, `geo:*:active` versions, `gateway:catalog:all` version, category/attribute versions | integer | none (noeviction) | `INCR` on owning write |
| Denylist | `bl:{sha256(token)}` | `revoked` | remaining token `exp` | logout writes; never deleted early |
| Coupon limiter | `coupon:apply:rate:{user}` | integer counter | 1m window (Lua `INCR+EXPIRE`) | TTL |
| File idempotency | `file:init:idem:{owner}:{id}:{sha256}` | init record JSON | init TTL + buffer | claim (`SETNX`) → replay → delete on failure |
| Scheduler queue | `delayed_jobs` (ZSET, score = exec unix) | job JSON | until claimed (`ZRem==1`) | poll → claim → dispatch → `Del(jobKey)` |
| Scheduler index | `scheduled_job:{uuid}` | job JSON | until processed/cancelled | `Del` after run/cancel |

## State-transition notes

- Fill: `miss → (singleflight) → DB → return → async SET` (dropped if pool full, shed active, over size, or generation changed).
- Write: `DB commit → Del entity (+tombstone, same key) → INCR version → async derived cleanup`.
- Version loss is impossible by placement (durable, noeviction) — never by TTL.
- Tombstone decode: readers treat the sentinel as a documented miss/404, never as a DTO (T11-type assertion per module).
