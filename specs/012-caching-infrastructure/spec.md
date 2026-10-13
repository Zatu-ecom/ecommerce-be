# Feature Specification: Caching Infrastructure

**Feature Branch**: `012-caching-infrastructure`  
**Created**: 2026-09-18  
**Status**: Draft  
**Input**: User description: "lets create a spec.md file from a prespec"
**Source**: `pre-spec.md` in this folder is the design source of truth (architecture, module registry §4.5, keys/TTLs §5). This spec states the WHAT (behavioral requirements); pre-spec states the WHY and the HOW-boundaries.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Shopper browses the catalog quickly and correctly (Priority: P1)

A shopper opens product pages, categories, and variant details. Repeated views are served fast from cache; when a seller edits a product, shoppers see the update within a bounded short time; stock figures shown are never frozen into catalog pages.

**Why this priority**: Catalog browsing is the highest-volume traffic on the platform. Every cached read is database load avoided; every stale write is a trust incident.

**Independent Test**: Enable only catalog-detail caching against staging seeded with products; request the same product repeatedly (fast responses), update the product as a seller, re-request (update visible within the staleness bound); assert no stock or personalized fields in cached responses.

**Acceptance Scenarios**:

1. **Given** a product exists and its detail was viewed once, **When** the same product is requested again, **Then** the response is served without querying the primary database and matches the stored data.
2. **Given** a cached product, **When** a seller updates its price or options, **Then** subsequent shopper requests reflect the update within the agreed staleness bound (minutes, not hours).
3. **Given** any cached catalog response, **When** inspected, **Then** it contains no live stock quantities, no expiring signed URLs, and no per-shopper personalization.

---

### User Story 2 - Platform survives traffic spikes without database overload (Priority: P1)

During high-traffic bursts — including floods of one-off random queries — the platform keeps responding quickly. Cache misses never queue behind cache writes; unrepeatable queries are not stored; the system sheds cache-write load automatically while continuing to serve reads.

**Why this priority**: This is the pre-launch write-storm incident class. If the caching layer amplifies load instead of absorbing it, every sale event becomes an outage.

**Independent Test**: Drive a flood of unique (non-repeating) product queries at staging, then a flood of repeated queries; assert p99 response latency stays flat, the database is not overwhelmed by duplicate regeneration work, one-off queries leave no stored entries, and repeated queries become fast.

**Acceptance Scenarios**:

1. **Given** a flood of unique queries, **When** each misses the cache, **Then** responses are still served from the database at normal latency and no large result entries accumulate in cache.
2. **Given** the cache backend is slow or down, **When** shoppers browse, **Then** all reads still succeed via the database with bounded extra latency (no timeouts caused by waiting on cache).
3. **Given** a sustained cache-write error storm, **When** the breaker trips, **Then** reads continue to be served while writes pause, and writes resume automatically after recovery.

---

### User Story 3 - Seller and admin changes take effect promptly (Priority: P1)

A seller updates a product, variant, category, price, or setting; an admin updates reference data (countries, currencies, gateway catalog). Shoppers and the seller see consistent results within the staleness bound — never a mix where the old price shows with the new options, and never one seller's data visible to another.

**Why this priority**: Stale commercial data (wrong price, wrong availability, cross-seller leakage) is a correctness and legal incident, not a performance issue.

**Independent Test**: For each cached resource type, perform the corresponding write, then immediately read; assert the read reflects the write (or a documented miss that refills), and assert no other seller's cached data is reachable.

**Acceptance Scenarios**:

1. **Given** a cached entity, **When** it is updated or deleted, **Then** the next read does not return the pre-write version beyond the staleness bound.
2. **Given** a cached listing, **When** any included product changes, **Then** the listing version advances so old assembled pages are never served as current.
3. **Given** cached data for seller A, **When** seller B requests the same resource type, **Then** seller B never receives seller A's data.

---

### User Story 4 - Logout actually logs out; rate limits hold across instances (Priority: P1)

A user logs out and the session token stops working immediately and permanently (no resurrection window). Coupon-apply rate limiting counts consistently no matter which instance serves the request.

**Why this priority**: Authentication bypass windows and over-redeemed coupons are security and revenue incidents.

**Independent Test**: Log in, log out, then replay the token at intervals past the old fixed window; assert rejection every time. Hammer coupon-apply concurrently across instances; assert the limit holds exactly.

**Acceptance Scenarios**:

1. **Given** a logged-out token, **When** it is presented at any time before its natural expiry, **Then** it is rejected.
2. **Given** concurrent coupon-apply attempts beyond the limit from multiple instances, **When** counted, **Then** the total accepted never exceeds the limit and the counter expires on schedule (no permanent blocks).

---

### User Story 5 - Checkout inventory decisions stay correct under load (Priority: P2)

A shopper adds to cart and checks out while stock is changing. Availability displays may lag by seconds, but the reserve/fulfill decision is always made against live atomic stock data — overselling is impossible regardless of cache state.

**Why this priority**: Selling what doesn't exist destroys trust and creates fulfillment chaos; correctness here outranks speed.

**Independent Test**: With availability micro-caching on, race concurrent checkouts against the last units of stock; assert total confirmed reservations never exceed available units, with or without cache, hot or cold.

**Acceptance Scenarios**:

1. **Given** cached availability data, **When** checkout reserves stock, **Then** the decision uses live guarded stock data, not the cached value.
2. **Given** concurrent checkouts for the last unit, **When** all complete, **Then** at most one succeeds and the others receive insufficient-stock errors.

---

### User Story 6 - Operators can run, observe, and roll back the cache layer (Priority: P2)

Operators enable caching per area with flags, watch hit/miss/error and latency signals, receive alerts on miss spikes, error spikes, and write-storm signatures, and can switch cache backends or disable caching without a deploy.

**Why this priority**: An unobservable cache is a liability; a non-rollback-able backend migration is a gamble.

**Independent Test**: Toggle each flag independently in staging; break the cache backend; flip backend addresses; assert documented behavior (fail-open reads, alerts fire, rollback is an address change).

**Acceptance Scenarios**:

1. **Given** any single caching flag off, **When** traffic flows, **Then** that area behaves exactly as before caching existed.
2. **Given** a cache backend outage, **When** on call reviews signals, **Then** error-rate alerts fire and reads continue from the database.
3. **Given** a backend migration, **When** the new backend regresses (errors/latency vs baseline), **Then** traffic is reverted by configuration change alone.

---

### Edge Cases

- What happens when a cache write for a just-updated entity is still in flight (miss → write → delete → late write)? The late write must be detected and dropped so it cannot resurrect stale data.
- How does the system handle a version-counter loss (e.g., eviction)? Version counters must live where eviction is impossible.
- What happens when a cached entry exceeds the size guard? It is never stored (no truncation); the database result is served and the event is counted.
- What happens when a subscription ends while its seller record is cached? The cached "active" verdict must not outlive the subscription end.
- What happens when the same cache key is missed simultaneously on many instances? Each instance regenerates at most once; the database sees bounded duplicate work, never a flood.
- What happens when a user account is hard-deleted? All cached entries keyed to that user/seller are removed exactly (no pattern deletes), with no tombstones retaining the data.
- What happens when listing queries arrive with permuted but identical parameters? They resolve to the same cache entry.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST serve repeated catalog, category, attribute, variant, geo-reference, gateway-catalog, and file-reference reads from cache per the module registry (pre-spec §4.5), falling back to the database on any cache miss or cache failure.
- **FR-002**: System MUST continue serving all cacheable reads from the database with bounded extra latency when the volatile cache is slow or unavailable (fail-open); responses MUST never wait for cache writes (population path: FR-006; storm path: FR-017).
- **FR-003**: System MUST expire every cached entry with jittered lifetimes (no fixed fleet-wide TTLs), including negative entries and admission markers.
- **FR-004**: System MUST record short-lived negative entries for not-found results so repeated lookups of missing data do not reach the database.
- **FR-005**: System MUST collapse concurrent identical misses within one instance so only one database fill proceeds per key.
- **FR-006**: System MUST populate cache asynchronously after returning the response, through a bounded pool; overflow MUST drop the write and count it, never slow the response (failure mode of the FR-002 guarantee).
- **FR-007**: System MUST detect a concurrent write that happened after a miss and drop the resulting stale write instead of storing it.
- **FR-008**: System MUST gate combinatorial query (list/search/filter) caching on a repeat-sighting admission check; one-off queries MUST NOT produce stored result entries.
- **FR-009**: System MUST invalidate cached entities with exact key deletes after the database commit (never inside the transaction), and advance list versions so old assembled pages are not served as current; pattern/prefix deletes on the request path are forbidden.
- **FR-010**: System MUST scope every tenant-owned cache entry to its seller and reject unscoped keys except an explicit platform allowlist; cross-tenant cache reads MUST be impossible (tested).
- **FR-011**: System MUST exclude live stock, expiring signed URLs, secrets, per-shopper personalization, and per-seller overlay flags from shared cached payloads.
- **FR-012**: System MUST keep reservation, checkout-commit, payment-state, coupon-evaluation, and audit-ledger reads uncached and always live.
- **FR-013**: System MUST back list-version and admission counters that correctness depends on with non-evictable storage.
- **FR-014**: System MUST store logout denylist entries keyed by token hash with a lifetime equal to the token's remaining validity, and MUST reject logged-out tokens until natural expiry; denylist unavailability MUST NOT be treated as a valid session.
- **FR-015**: System MUST enforce coupon-apply rate limits atomically across instances with self-expiring counters.
- **FR-016**: System MUST keep scheduled-job queue entries, file-upload idempotency records, and rate-limit counters on non-evicting durable storage with restart recovery.
- **FR-017**: System MUST pause cache writes automatically under sustained write-error storms while continuing reads, and resume after recovery; operators MUST also be able to pause writes manually without a deploy (automatic + manual forms of the FR-002 fail-open guarantee under backend distress).
- **FR-018**: System MUST NOT retry failed cache reads or writes (fail open / drop and count); upstream request retries MUST use bounded jittered backoff.
- **FR-019**: System MUST isolate all cache-backend client specifics behind internal interfaces so no business, strategy, scheduler, or job code depends on a cache vendor's types; swapping backends MUST require no changes outside the adapter, configuration, and deployment descriptors.
- **FR-020**: System MUST gate every cacheable area behind an independently toggleable flag defaulting to off, with flags having no effect on durable queue/lock/denylist behavior.
- **FR-021**: System MUST emit hit/miss/error/latency signals per module and operation, plus write-drop, shed-state, and in-flight-fill signals, suitable for dashboards and alerts.
- **FR-022**: System MUST remove all cache entries keyed to a user/seller on hard account deletion using exact deletes.
- **FR-023**: System MUST cap subscription-derived cached verdicts at the subscription end time so cached "active" cannot outlive the subscription.
- **FR-024**: Inventory availability micro-caching MUST apply only to internal availability reads with second-scale lifetimes, MUST be gated on an atomic stock-reservation guard being in place first, and MUST never influence the reserve/commit decision.

### Key Entities

- **Cached Entry**: A stored response DTO for one resource (product, category, reference datum) with a jittered lifetime; tenant-scoped except allowlisted platform keys.
- **Negative Entry (tombstone)**: A short-lived marker sharing the missing entry's key, meaning "confirmed absent"; indistinguishable-delete on writes.
- **List Version Counter**: A monotonic non-evictable number embedded in assembled-list keys; advancing it retires all lists at once without enumeration.
- **Admission Marker**: A tiny short-lived record of a query's first sighting; a repeat sighting earns the full result entry.
- **Denylist Entry**: A token-hash record living exactly as long as the token's remaining validity; absence of the store never equals permission.
- **Module Cache Strategy**: The per-business-module owner of what is cached, for how long, and what invalidates it; the only place business caching rules live.
- **Cache Backend Roles**: The volatile role (evictable, restart-losable, fail-open) and the durable role (non-evicting, snapshotted, fail-closed where required).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Shoppers see repeated catalog detail responses quickly (p99 within the snappiness budget recorded in `baselines.md` before any flag turns on) while the primary database carries measurably less catalog read load than that baseline at equal traffic.
- **SC-002**: During a burst mixing repeated and one-off queries, shopper-facing p99 latency stays flat and no timeouts are caused by cache writes.
- **SC-003**: Seller/admin edits are visible to shoppers within the per-resource bound in pre-spec §5 (entity minutes, lists ~1–2m), verified per resource type.
- **SC-004**: No logged-out session token is accepted at any point before its natural expiry; concurrent coupon-apply pressure never exceeds the published limit.
- **SC-005**: Under concurrent checkout races for the last units, confirmed reservations never exceed available stock, with caching on or off.
- **SC-006**: With the cache backend unavailable, all browsing and checkout-read flows keep working from the database (degraded latency only).
- **SC-007**: Operators can enable/disable each cacheable area independently without a deploy, and can move traffic between cache backends by configuration with a same-day revert path.
- **SC-008**: Zero incidents of one tenant's cached data served to another, and zero incidents of cached commercial data (price/availability) contradicting committed state beyond the agreed bounds, over the release bake window.

## Assumptions

- The platform is pre-production: cache key formats, stored payload shapes, and backend choice carry no migration constraints.
- Constitution §VIII justification (breaking-change process): the denylist fail-closed change (FR-014) ships unflagged as a security fix with zero production traffic to migrate; behavior is fully covered by conformance tests and instantly revertible by restoring the previous handler/middleware lines. No other behavior change in this feature ships outside a default-off flag.
- Existing repository tenant scoping (`seller_id` filtering) is correct and stays the enforcement point; cache never bypasses it.
- The atomic stock-reservation guard lands before any inventory availability caching is enabled.
- Reference data (countries, currencies, gateway catalog, storage providers/schemas) changes only via admin/seed/deploy paths where invalidation hooks can be added.
- Token lifetime remains environment-driven; denylist lifetimes always derive from remaining token validity.
- Single-instance deployment is not a target; multi-instance behavior in the acceptance scenarios is mandatory.
- Staleness bounds in pre-spec §5 are the agreed product contract (entity minutes, lists ~1–2m, availability seconds).

## Out of Scope

- Cart, order, payment-result, coupon-evaluation, and audit-ledger caching.
- Collections detail beyond by-id, wishlist, recently-viewed, promotion evaluation/dashboard caches, and report aggregates (registry marks each as deferred or never; see pre-spec §4.5/§11).
- Recommendation engine caching (binding contract reserved in pre-spec §11).
- Search-index infrastructure, write-behind caching, stale-while-revalidate, and cache warm-up jobs.
- Cache clustering/replication, TLS for cache links, per-tenant quotas, and distributed cross-instance locks.

## Dependencies

- Primary database as source of truth and fallback for every cached read.
- Two cache backend roles provisioned (volatile + durable) with the specified eviction/persistence behavior, in compose, CI, and target environments.
- Atomic stock-reservation guard (separate change) as a precondition for inventory availability caching.
- Invalidation hooks in all writers named by the registry (product/nested, category/attribute, settings, currency, admin geo/gateway, file seed, subscription/plan/user-active writers).
- Feature-flag and metrics/alert plumbing for per-area toggles and the signals in FR-021.
