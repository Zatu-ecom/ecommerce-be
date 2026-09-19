# Implementation Plan: Caching Infrastructure

**Branch**: `012-caching-infrastructure` | **Date**: 2026-09-18 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/012-caching-infrastructure/spec.md`
**Design source**: [pre-spec.md](./pre-spec.md) (architecture, module registry §4.5, keys/TTLs §5 — normative for behavior)

## Summary

Replace the thin inconsistent `common/cache` wrapper with a layered, provider-blind caching platform (`common/cachekit` + per-module strategies), split across two backend roles (volatile fail-open cache + durable non-evicting KV), and roll out per-area caches behind flags: Phase 0 foundation + bug fixes, Phase 1 safe reference/catalog caches, Phase 2 (flag-gated) inventory micro-cache + versioned list cache with admission control, then a config-only DragonflyDB cutover with Redis rollback. Primary requirements: write-storm survival (admission, async bounded writes, jitter, write-shed breaker), inventory correctness (atomic guard first), tenant isolation, and observability from day one.

## Technical Context

**Language/Version**: Go 1.25+ (module `ecommerce-be`)
**Primary Dependencies**: `github.com/redis/go-redis/v9` (new, replaces v8), `golang.org/x/sync` (singleflight — already indirect dep), Gin, GORM, testify/suite, Testcontainers
**Storage**: PostgreSQL 16 (source of truth, unchanged schema — no migrations in this feature); volatile cache role + durable KV role (Redis 7 now, DragonflyDB at cutover; RESP-compatible)
**Testing**: Go testing + testify (unit) + Testcontainers (integration, both Redis and Dragonfly images); existing `test/integration/setup` extended with a second KV profile
**Target Platform**: Linux server (Docker Compose local/dev; same container topology in target envs)
**Project Type**: Backend modular-monolith feature (shared `common/` infrastructure + per-module `cache/` strategies)
**Performance Goals**: Catalog detail p99 fast on hit with >80% hit rate after warmup (directional); cache-down reads succeed from DB with bounded extra latency; slow/dead backend never causes upstream timeouts (T14)
**Constraints**: Per-op cache timeouts (GET ~200ms, async SET 50–100ms budget); cached values ≤256KB (skip + count); no `KEYS` on request path; no provider types outside `cachekit/provider/`; interfaces ≤10 methods (constitution); files ≤500 lines, methods ≤50 lines; mandatory comments
**Scale/Scope**: 9 modules touched (reads in 6), ~17 new key shapes, T1–T17 conformance suite on two backends, single-node per backend role (no cluster/sentinel/replication in 011)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Modular monolith | ✅ PASS | `cachekit` lives in `common/` (cross-cutting, like auth/log); business rules live in owning module's `cache/` package; no cross-module repo access; strategies duplicated per module, never shared |
| II. Layered architecture | ✅ PASS | Handler→Service→(strategy)→Repository unchanged; strategies sit beside services, call repos only via services; no handler→cachekit, no repository→cachekit (forbidden directions §3.1) |
| III. Factory-singleton DI | ✅ PASS | `cachekit` clients constructed once (wired in `main.go`/container setup); module strategies get interfaces via existing `ServiceFactory` constructors (`NewXxx(deps...)`) |
| IV. TDD, integration-first | ✅ PASS | T-suite written first per phase (TDD Red); Testcontainers extended with Dragonfly profile; API-first assertions; suite pattern + endpoint constants + per-file cleanup |
| V. Seller isolation | ✅ PASS | Key builder enforces `seller:{id}` scope + closed platform allowlist; repos keep `WHERE seller_id=?`; T9 cross-seller test fails closed |
| VI. Correlation ID | ✅ PASS | Detached async contexts inherit correlation/seller/user IDs for logging (never `Background()` without IDs) |
| VII. RBAC | ✅ PASS | No change to auth paths; denylist hardens (not weakens) logout |
| VIII. Backward compat | ✅ PASS | All behavior changes opt-in via flags defaulting off; HTTP contracts unchanged; no migrations; denylist fail-closed change flagged + tested (T12) |
| IX. SOLID | ✅ PASS | `Cache`/`Durable`/`DelayQueue` small focused interfaces (≤10 methods); provider adapter = DIP/OCP (new backend = new adapter); strategies = SRP per module |
| X. Performance by design | ⚠️ SUPERSEDED (justified) | Constitution §X cache clause ("single Redis, LRU 256MB, pattern-based invalidation") is stale: replaced by two-store topology, version-counter invalidation, and write-storm protections. Rationale documented in pre-spec §§0.3, 5.4, 8.6. No other X clause violated (pagination, indexes, N+1 rules untouched) |
| Tech standards (go-redis v8) | ⚠️ SUPERSEDED (justified) | v8 frozen since 2022; v9 is the maintained path-change upgrade with compatible API. Noted for constitution amendment alongside §X cache clause |

**Stateless-services note (X):** in-process singleflight maps, shed flags, and limiter fallback are ephemeral per-pod state — safe to lose, never authoritative. No session affinity required.

## Project Structure

### Documentation (this feature)

```text
specs/012-caching-infrastructure/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   ├── cachekit-interfaces.md
│   └── module-strategy.md
├── checklists/
│   └── requirements.md  # Spec quality gate (passed)
├── pre-spec.md          # Design source of truth (prior artifact)
├── spec.md              # Behavioral requirements (prior artifact)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
common/cachekit/                 # NEW - infrastructure only
├── interface.go                 # Cache, Durable, DelayQueue + errors
├── provider/                    # NEW - sole provider-client importer
│   ├── go_redis.go              # v9 adapter (later: dragonfly needs no code change - same RESP)
│   └── construct.go             # NewCache/NewDurable (one-way dep, like sql drivers)
├── codec.go                     # JSON + 256KB guard
├── key.go                       # tenant-scoped builder + allowlist
├── ttl.go                       # JitteredTTL
├── singleflight.go              # miss-collapse wrapper
├── negative.go                  # tombstone helpers
├── version.go                   # durable-backed version counters
└── metrics.go                   # hooks (logrus now, Prometheus-ready labels)

product/cache/  user/cache/  payment/cache/  file/cache/      # NEW - one strategy pkg per module (Phase 1)
inventory/cache/                                             # NEW (Phase 2a)
common/config/  main.go  docker-compose.yml                  # MODIFIED - two addrs, pool/timeouts, two KV services
common/auth/  user/handler/  order/utils/  common/scheduler/ # MODIFIED - denylist, Lua limiter, DelayQueue wiring
test/integration/cachekit/      # NEW - T1-T17 conformance (both backends)
```

**Structure Decision**: Follows the repo's standard module layout (`cache/` beside `service/`/`repository/` inside each module; shared infra in `common/`). No new top-level projects; no new databases (Postgres schema untouched).

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| Two backend roles instead of constitution's single Redis | Today's single LRU instance can evict scheduler jobs/idempotency/denylist (proven contradiction: `allkeys-lru` + volume) | One process with `noeviction` forces catalog to share fate with jobs; eviction classes must not mix (§0.3) |
| Async cache population + generation guards | Sync SET on the response path caused the write-storm outage mode (§0.4) | Inline SET with timeout still couples response latency to backend health; drop-and-count is strictly safer |
| Admission markers for list caching | Random-query floods defeat all stampede tools (singleflight only coalesces identical keys) | Caching every list response = unbounded SET load + LRU pollution with ~zero hits |
| Superseding constitution §X cache clause + v8 client | Stale (pattern invalidation never implemented; v8 frozen 2022) | Staying would enshrine the exact defects this feature fixes; flagged for constitution amendment |
