# Tasks: Caching Infrastructure

**Input**: Design documents from `/specs/012-caching-infrastructure/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md, pre-spec.md

**Tests**: Included — TDD is constitution-mandated (integration-first, Testcontainers) and every spec story defines independent test criteria. Write tests FIRST, ensure they FAIL before implementation.

**Organization**: Grouped by user story. Flags default off throughout; each story is independently testable with its flags on and everything else off.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1)
- Exact file paths in every description

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Dependencies, two-role backend topology in compose/CI, config plumbing, baselines

- [X] T001 Swap `github.com/go-redis/redis/v8` → `github.com/redis/go-redis/v9` in `go.mod`/`go.sum` (`go get`, `go mod tidy`, `go build ./...`)
- [X] T002 [P] Add volatile + durable KV services named `cache-volatile`/`cache-durable` to `docker-compose.yml` (Redis profiles now; Dragonfly profiles alongside, pinned tags, `cache_mode`/snapshot flags per research.md R3; app depends on Postgres only)
- [X] T003 [P] Extend `test/integration/setup/container.go` + `server.go` with a second KV container profile (generic container, pinned tags, `PING` wait; select backend via env per quickstart.md)
- [X] T004 Add `CACHE_ADDR`/`KV_ADDR`, pool/timeout/retry settings to `common/config/redis.go` (+ loader validation, `.env`/`.env.test` entries)
- [X] T005 [P] Add all `CACHE_*` feature flags (default off) to config in `common/config/` per pre-spec §8.5
- [X] T006 Capture DB QPS + latency baselines for auth/catalog paths and record in `specs/012-caching-infrastructure/baselines.md` (Phase 0 exit criterion)
- [X] T007 [P] Add provider-blindness CI grep (pre-spec §12 Phase 0) to `.github/workflows/` (fail on provider imports/types outside `common/cachekit/provider/` and test files)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: `cachekit` infrastructure, scheduler transport, conformance harness. MUST complete before ANY user story.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T008 Define `Cache`, `Durable`, `DelayQueue` interfaces + `ErrMiss`/`ErrUnavailable` in `common/cachekit/interface.go` (contracts/cachekit-interfaces.md; ≤10 methods each)
- [ ] T009 [P] Implement v9 adapter in `common/cachekit/provider/go_redis.go` (only file importing the client lib; KEYS-only Lua discipline per research.md R2)
- [ ] T010 Implement two interfaced clients in `common/cachekit/client.go` (pool, per-op timeouts, `Ping`, metrics hook; holds interfaces, never `*redis.Client`) + wire both clients' `Close` into `main.go` graceful shutdown (existing shutdown sequence: HTTP → requests → DB → cache → cron)
- [ ] T011 [P] Implement JSON codec + 256KB guard in `common/cachekit/codec.go`
- [ ] T012 [P] Implement tenant-scoped key builder + platform allowlist in `common/cachekit/key.go` (reject unscoped; T9 unit tests)
- [ ] T013 [P] Implement `JitteredTTL` in `common/cachekit/ttl.go` (±15%, bounds-tested)
- [ ] T014 [P] Implement singleflight wrapper in `common/cachekit/singleflight.go` + tombstone helpers in `common/cachekit/negative.go` (same-key sentinel)
- [ ] T015 [P] Implement metrics hooks in `common/cachekit/metrics.go` (logrus sink; per-module/op/store labels)
- [ ] T016 Rewire `common/scheduler/scheduler.go` + `common/scheduler/worker.go` to `DelayQueue` (adapt dispatcher loop to the `Poll()` claim API; remove raw `*redis.Client` incl. `inventory/factory/singleton/service_factory.go:46` nil-deref; keep ZSET poll + `ZRem==1` claim semantics)
- [ ] T017 [P] Scaffold `test/integration/cachekit/` conformance harness running T1–T17 skeletons against both backend profiles (red: all fail)
- [ ] T018 Remove dead cache code: `product/utils/cache_constants.go`, dead keys/invalidators in `common/cache/cache_invalidation.go` + `common/constants/cache_constants.go` (keep shims only if callers remain)

**Checkpoint**: Foundation ready — `go build ./...`, blindness grep green, harness red. User stories can now begin (US1+US4+US5 in parallel; US2+US3 integrate after US1).

---

## Phase 3: User Story 1 - Browse catalog quickly and correctly (Priority: P1) 🎯 MVP

**Goal**: P1 safe caches — product/variant/category/attribute detail, seller validation, currency, geo, seller settings, gateway catalog slice, file refs — each behind its flag.

**Independent Test**: Seed staging; same-resource repeat reads served without DB queries; writer updates visible within bounds; payloads contain no stock/signed-URLs/personalization; cross-seller reads fail closed.

### Tests for User Story 1 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T019 [P] [US1] Conformance T7 subset (domain reads succeed with volatile down) in `test/integration/cachekit/cache_down_test.go`
- [ ] T020 [P] [US1] Conformance T9 cross-seller isolation in `test/integration/cachekit/tenant_isolation_test.go`
- [ ] T021 [P] [US1] Payload-contract test (no stock/URLs/personalization, 256KB skip) in `test/integration/product/product_cache_test.go`
- [ ] T022 [P] [US1] Seller-validation + currency + geo + settings + gateway-catalog read tests in `test/integration/user/cache_reads_test.go` and `test/integration/payment/gateway_cache_test.go`

### Implementation for User Story 1

- [ ] T023 [P] [US1] Product/variant/category/attribute strategies in `product/cache/` (detail keys, tombstones, §5.7 nested-invalidation set; incl. optional collection-by-id `seller:{id}:collection:{cid}`)
- [ ] T024 [P] [US1] Seller-validation + currency + geo + settings strategies in `user/cache/` (subscription-end-capped TTL, exact-`Del`)
- [ ] T025 [P] [US1] Gateway catalog-slice strategy in `payment/cache/` (public metadata only; seller overlay stays live in `payment/service/payment_gateway_service.go`)
- [ ] T026 [P] [US1] Providers/schema strategy in `file/cache/` (no URLs/secrets)
- [ ] T027 [US1] Wire strategies into query services (`product/service/product_query_service.go`, `category_service.go`, `user/service/user_service.go`, `common/auth/seller_validation.go`, gateway/file handlers) behind flags (depends on T023–T026)
- [ ] T028 [US1] Wire invalidation hooks into all product/nested, category/attribute, settings/currency, geo, gateway-admin writers (depends on T027)

**Checkpoint**: US1 flags on in staging → detail hit rate climbs, writes reflect within bounds, T7/T9/T11 green. MVP shippable.

---

## Phase 4: User Story 4 - Logout holds; rate limits hold (Priority: P1)

**Goal**: Denylist keyed by token hash with remaining-`exp` TTL + fail-closed semantics; atomic Lua coupon limiter. (Scheduled with US1 — depends only on Foundational.)

**Independent Test**: Replay logged-out tokens past the old 24h window (always rejected); concurrent multi-instance coupon-apply never exceeds the limit and never permanently blocks.

### Tests for User Story 4 ⚠️

- [ ] T029 [P] [US4] Conformance T3 (Lua atomicity, concurrent + two clients) + T12 (denylist hash/TTL/fail-closed/logout-503) in `test/integration/cachekit/durable_correctness_test.go`

### Implementation for User Story 4

- [ ] T030 [P] [US4] Hashed denylist helpers (`bl:{sha256}`, remaining-`exp` via `ParseToken`) in `user/cache/` or `common/cachekit/` Durable path; logout returns 503 on SET failure in `user/handler/user_handler.go`; denylist-miss-on-KV-down fails closed in `common/auth/auth_middleware.go`
- [ ] T031 [P] [US4] Lua `INCR+EXPIRE` limiter in `order/utils/coupon_apply_rate_limit.go` on Durable (KEYS-only script; keep in-process fallback)

**Checkpoint**: US4 green on both backends; logout replay window closed.

---

## Phase 5: User Story 5 - Checkout inventory stays correct (Priority: P2)

**Goal**: Atomic reservation guard first, then 3–5s availability micro-cache for internal reads only. (Depends only on Foundational; parallel with US1/US4.)

**Independent Test**: Race concurrent checkouts for last units (cache hot and cold) — confirmed reservations never exceed stock; reserve path never consults cache.

### Tests for User Story 5 ⚠️

- [ ] T032 [P] [US5] Atomic-guard test (concurrent reserves, exactly-one-wins, no oversell without cache) in `test/integration/inventory/reserve_atomicity_test.go`
- [ ] T033 [P] [US5] Micro-cache race test (hot-cache concurrent checkout, ≤5s gate lag accepted, zero oversell) in `test/integration/inventory/availability_cache_test.go`

### Implementation for User Story 5

- [ ] T034 [US5] Conditional atomic decrement guard in `inventory/repository/inventory_repository.go` (reuse `IncrementReservedQuantity` semantics; `WHERE quantity - reserved >= ?`) + use in `inventory/service/inventory_reservation_service.go` (depends on T032)
- [ ] T035 [US5] Availability micro-cache strategy in `inventory/cache/` (TTL-only 3–5s, singleflight, writes bypass) behind `CACHE_INVENTORY_AVAIL`, wired into internal `GetTotalAvailableQuantities` callers only (depends on T034)

**Checkpoint**: US5 green; oversell impossible in all cache states.

---

## Phase 6: User Story 2 - Survive traffic spikes (Priority: P1)

**Goal**: Async bounded population, generation-guarded SETs, write-shed breaker, retry discipline, admission markers. (Builds on Foundational client; integrates US1 paths transparently.)

**Independent Test**: Unique-query flood (flat p99, markers only, memory flat) + dead-backend drill (reads 200 from DB, SETs dropped + counted) + error-storm drill (breaker pauses writes, GETs continue, auto-recovery).

### Tests for User Story 2 ⚠️

- [ ] T036 [P] [US2] Conformance T13 (admission: one-off marker-only, repeat fills) in `test/integration/cachekit/admission_test.go`
- [ ] T037 [P] [US2] Conformance T14 (dead-backend latency + drop counting) in `test/integration/cachekit/degraded_backend_test.go`
- [ ] T038 [P] [US2] Conformance T15 (breaker trip/recover) + T16 (jitter bounds) + T17 (no stale resurrection) in `test/integration/cachekit/write_protection_test.go`

### Implementation for User Story 2

- [ ] T039 [US2] Async bounded SET pool + drop metrics in `common/cachekit/client.go` (detached ctx with IDs, 50–100ms budget, default 64 in-flight)
- [ ] T040 [P] [US2] Generation-guarded SET (Lua CAS, KEYS-only) + `cache_set_generation_dropped` in `common/cachekit/client.go` / `version.go`
- [ ] T041 [P] [US2] Write-shed breaker + `CACHE_SET_WRITES` manual lever in `common/cachekit/client.go` (cooldown + trickle recovery, `cache_write_shed_active`)
- [ ] T042 [US2] Admission marker helper (`SETNX seller:{id}:seen:{hash}`) in `common/cachekit/` for P2 list strategies (depends on T039)

**Checkpoint**: US2 green on both backends; §0.4 chain broken at every link by test.

---

## Phase 7: User Story 3 - Changes take effect promptly (Priority: P1)

**Goal**: Complete invalidation fabric — durable version counters, remaining writer hooks, erasure deletes. (Depends on US1 strategies.)

**Independent Test**: Every §5.7 row + settings/currency/geo/gateway writers: write → immediate read reflects or misses-then-refills; list versions advance; deleted-user keys gone exactly.

### Tests for User Story 3 ⚠️

- [ ] T043 [P] [US3] Conformance T8 full matrix (entity miss + version bump per write incl. all §5.7 nested rows) in `test/integration/product/invalidation_test.go`
- [ ] T044 [P] [US3] Erasure test (hard delete → exact keys gone, no globs/tombstones) in `test/integration/user/cache_erasure_test.go`

### Implementation for User Story 3

- [ ] T045 [US3] Durable version-counter helpers in `common/cachekit/version.go` (durable KV placement) + `productlist:ver`/`collectionlist:ver`/geo/gateway/category version wiring in write paths
- [ ] T046 [US3] Remaining hooks: seller-validation invalidation on `user.is_active`/subscription/plan writers (incl. non-HTTP writers), currency/settings/geo/gateway exact-`Del`s, user-delete purge (depends on T045)

**Checkpoint**: US3 green; staleness bounds hold per resource; no cross-seller leakage.

---

## Phase 8: P2 lists (US1 browsing value, US2 flood hardening) + User Story 6 - Operate and roll back (Priority: P2)

**Goal**: Versioned list/search/filter/collection-product caching with admission (P2b); observability, baselines doc, cutover drill with rollback.

**Independent Test**: Repeated lists fast within version; product write retires lists; one-off floods bypass (T13 regression); backend flip by config with revert.

### Tests + Implementation

- [ ] T047 [P] [US1] List/admission strategy + canonical-hash tests in `product/cache/` list module + `test/integration/product/product_list_cache_test.go` (uses T042 markers, T045 versions)
- [ ] T048 [US6] Dashboard/alert assertions doc + alert-rule stubs per pre-spec §8.2 in `specs/012-caching-infrastructure/alerts.md` (hit/miss/error/latency, shed-active, pollution signature, durable-eviction page; log-sink metrics are the source — no metrics-backend deploy in 011)
- [ ] T049 [US6] Cutover drill on staging: T1–T17 green on Dragonfly, flip `CACHE_ADDR`/`KV_ADDR`, bake vs baselines, revert drill (addr-only), record in `specs/012-caching-infrastructure/cutover.md`
- [ ] T050 [US6] Run `quickstart.md` end-to-end validation (compose, both suites, flag drills, incident simulations) and fix gaps

**Checkpoint**: P2b green; cutover proven reversible on staging.

---

## Phase 9: Coverage remediation (analysis findings)

**Purpose**: Close the gaps found in the specification analysis report — each task traces to a finding ID

- [ ] T056 [P] [US4] Negative-guard test: checkout, payment, and coupon-apply paths emit zero volatile `Cache` SETs in `test/integration/order/no_cache_guard_test.go` + `test/integration/payment/no_cache_guard_test.go` (FR-012; finding H1)
- [ ] T057 [P] [US6] Durable-path conformance fill-in: full T2/T4/T5/T6/T10 implementations in `test/integration/cachekit/durable_paths_test.go` (T017 provided skeletons; finding H3)
- [ ] T058 [P] [US2] Retry-discipline unit test (single attempt on GET/SET then fail-open/drop) in `common/cachekit/client_test.go` + upstream jittered-backoff guidance note in `specs/012-caching-infrastructure/alerts.md` (finding H4; T048 owns the doc)
- [ ] T059 [P] [US1] Load/flood script: unique-query flood (memory flat, markers only) + repeat-query flood (hit rate climbs) with p99 assertions in `test/integration/cachekit/flood_test.go` (feeds SC-001 and the P2b flood regression; finding M1)
- [ ] T060 [P] [US1] Unit tests for pure `cachekit` logic in `common/cachekit/key_test.go`, `common/cachekit/ttl_test.go`, `common/cachekit/codec_test.go`, `common/cachekit/negative_test.go` (tombstone store-as-miss/`Del`-on-write/create-clears), `common/cachekit/singleflight_test.go` (collapse) (findings M2/M3)

**Checkpoint**: Analysis report shows zero open findings; coverage re-verified (FR-012/H1, FR-018/H4, SC-001/M1 all green)

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Release hygiene

- [ ] T061 [P] Final blindness + dead-code grep pass (`go-redis` outside provider, `KEYS` outside janitor/tests, `Background()` without IDs) and fix strays
- [ ] T062 [P] Constitution amendment PR: update §X cache clause + client standard to v9/two-store (pre-spec documents rationale)
- [ ] T063 [P] Mermaid/diagram render check for pre-spec + fix `\n`→`<br/>` cosmetics
- [ ] T064 Full suite green (`make test`), `go vet`, `gofmt -l` clean; update `ARCHITECTURE.md` cache section if it contradicts the design
- [ ] T065 Promotion sweep multi-pod audit (`promotion/container.go:45` idempotency/locking; payment already SKIP LOCKED) + stale-PENDING reconciler decision recorded

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS all stories
- **Stories**: US1/US4/US5 depend only on Foundational (parallelizable); US2 needs Foundational (integrates US1 paths, no US1 edits); US3 needs US1; Phase 8 needs US1+US2+US3 (+US5 guard exists for 2a); Phase 9 remediation needs its target phases' tests green (T056→US4/US5 paths, T057→Foundational, T058→US2 client, T059/T060→US1); Polish needs all
- Recommended order: Setup → Foundational → [US1 + US4 + US5 in parallel] → US2 → US3 → Phase 8 → Phase 9 remediation → Polish (US4/US5 may land before US1; US2/US3/P2 must not precede their deps)

### Within Each Story

- Tests FIRST (red) → interfaces/helpers → strategies → wiring → invalidation → green → refactor (TDD per constitution)

### Parallel Opportunities

- [P] Setup tasks (T002/T003, T005) and Foundational files (T009, T011–T015) run in parallel — different files
- US1 strategies (T023–T026), US4 tasks (T030/T031), US5 tests (T032/T033) parallelizable across developers once Foundational lands
- Per-story test files run in parallel; list strategy (T047) parallel with US6 docs (T048)

---

## Parallel Example: Foundational

```bash
# Different files, no interdependencies — launch together:
Task: "Implement v9 adapter in common/cachekit/provider/go_redis.go"
Task: "Implement JSON codec + 256KB guard in common/cachekit/codec.go"
Task: "Implement tenant-scoped key builder + platform allowlist in common/cachekit/key.go"
Task: "Implement JitteredTTL in common/cachekit/ttl.go"
Task: "Implement singleflight wrapper in common/cachekit/singleflight.go + tombstone helpers in common/cachekit/negative.go"
Task: "Implement metrics hooks in common/cachekit/metrics.go"
```

---

## Implementation Strategy

### MVP First (US1 + US4)

1. Complete Phase 1: Setup (incl. baselines T006)
2. Complete Phase 2: Foundational (harness red, greps green)
3. Complete US1 (catalog/reference caches, one flag at a time in staging) + US4 (logout/limiter correctness)
4. **STOP and VALIDATE**: detail hit rate, invalidation bounds, denylist replay, limiter exactness
5. Deploy/demo behind flags (all default off; zero behavior change when off)

### Incremental Delivery

1. Foundation → US1+US4+US5 → US2 (hardening) → US3 (invalidation fabric) → P2 lists + US6 (operate/cutover) → Polish
2. Each phase independently testable; flags allow per-area rollout and instant revert

### Parallel Team Strategy

1. Team completes Setup + Foundational together
2. Then: Developer A → US1 strategies; Developer B → US4 + US5 tests/guard; Developer C → US2 client hardening
3. Converge on US3 invalidation, then Phase 8 lists + cutover drill together

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to spec user story for traceability (US5/US6 have no product-surface dependency on US1; US3/P2 extend US1)
- T-suite IDs (T1–T17) referenced are conformance cases from pre-spec §9.1, not these task IDs
- Commit after each task or logical group; stop at any checkpoint to validate
- Avoid: vague tasks, same-file conflicts, prefix deletes on request paths, provider imports outside `cachekit/provider/`
