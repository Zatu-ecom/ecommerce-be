# Cutover Drill: Redis → DragonflyDB (T049)

Binding: [pre-spec.md](./pre-spec.md) §7, [quickstart.md](./quickstart.md) §6,
[baselines.md](./baselines.md). Backend choice is unconstrained (pre-prod):
no migration, no backward-compat shims — cutover is an addr flip, rollback
is the same two variables back.

## 1. Topology (already in compose)

| Role | Redis profile | Dragonfly profile | Ports |
|---|---|---|---|
| volatile (fail-open) | `cache-volatile` (`allkeys-lru`, no snapshot) | `dragonfly-volatile` (`--cache_mode=true`, image `.../dragonfly:v1.40.0`) | 6380 / 6390 |
| durable (noeviction) | `cache-durable` (RDB+AOF) | `dragonfly-durable` (snapshots, no `cache_mode`) | 6381 / 6391 |

Image tags are pinned identically in compose, Testcontainers setup, and CI
— never float `latest` (research R7). App `depends_on`: Postgres only.

## 2. Evidence from this environment (2026-09-19)

| Check | Result |
|---|---|
| `docker compose config` | valid |
| Redis KV roles boot + `PING` (`CACHE_PASSWORD` set — empty default breaks `--requirepass`, see §4) | PONG / PONG |
| T-suite on Redis profiles | green: US1, US4, US2 (admission/degraded/write-protection), US3 (invalidation/erasure), P2 lists |
| T-suite on Dragonfly profiles (`KV_BACKEND=dragonfly`) | **blocked by host**: `v1.40.0` exits demanding 3.00 GiB (12 threads); host has 2.89 GiB (`proactor_pool.cc:149 … Exiting…`). Not a code signal — Lua scripts are KEYS-only per R2. |
| Revert drill | addr-only by construction (no code path branches on backend); exercised in-suite via dead-backend fail-open tests |

**Verdict: cutover NOT certified here — staging (≥4 GiB free) must run
`KV_BACKEND=dragonfly go test ./test/integration/...` green before any
addr flip.** The §3 runbook is the staging procedure.

## 3. Staging runbook

1. `docker compose --profile dragonfly up -d dragonfly-volatile dragonfly-durable`
2. `KV_BACKEND=dragonfly go test ./test/integration/...` — all green, notably
   T10 (durable jobs survive volatile pressure) and T13–T17.
3. Flip `CACHE_ADDR`/`KV_ADDR` to the Dragonfly pair (6390/6391 or service
   DNS). No deploy in either direction.
4. Bake one release: compare error rate + p99 domain-read latency + detail
   hit rate against `baselines.md`. Roll back on: volatile error rate above
   pre-cutover baseline, p99 regression, or any T-suite failure live.
5. Keep the Redis profiles one release, then retire or keep as fallback.

## 4. Local gaps found by this drill (fixed in T050)

- Compose KV roles boot-loop on a fresh checkout: empty password turns
  `--requirepass` into a flag-swallowing fatal config error. Deploy always
  injects a secret, so the fix is procedural — quickstart step 1 now
  requires `CACHE_PASSWORD` to be set (renamed from `REDIS_PASSWORD` in the
  legacy-cache retirement).
- quickstart §2 invoked `-run TestCachekitConformance`, which matches no
  test (real suite: `TestConformanceSuite`). Fixed.
- `cache_fill_inflight` had no readable source (FR-021): added
  `Flight.Inflight()` (+ unit test); shed/async gauges already exposed via
  `Adapter.Stats()`.
