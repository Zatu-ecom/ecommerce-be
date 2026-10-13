# Quickstart: Caching Infrastructure (local verification)

Audience: engineers implementing or reviewing `012-caching-infrastructure`. Full behavior contract: [pre-spec.md](./pre-spec.md).

## 1. Start the stack

```bash
# Postgres + volatile cache + durable KV (two roles; Redis until cutover).
# CACHE_PASSWORD must be non-empty: an empty value breaks the roles'
# --requirepass flag and boot-loops them (deploy always injects a secret).
CACHE_PASSWORD=devpass docker compose up -d postgres cache-volatile cache-durable
docker compose ps   # both KV roles healthy, app NOT depending on volatile
```

## 2. Run the conformance suite (works from Phase 0 on)

```bash
# Against Redis profiles (default). Per-story suites carry the conformance
# cases (T7/T9/T11 in US1CachesSuite, T3/T12 in DurableCorrectnessSuite,
# T13 in AdmissionSuite, T14 in DegradedBackendSuite, T15–T17 in
# WriteProtectionSuite, T8 in InvalidationSuite, lists in
# ProductListCacheSuite); TestConformanceSuite skips unimplemented T1–T17
# skeletons until later phases wire them.
go test ./test/integration/cachekit/ -run TestConformanceSuite -v

# Against Dragonfly profiles (same suites, second backend; needs ~3 GiB
# free for the v1.40.0 image or the container exits on boot)
KV_BACKEND=dragonfly go test ./test/integration/cachekit/ -run TestConformanceSuite -v
```

Green on both backends is the cutover gate (T1–T17). Never float image tags — compose, Testcontainers, and CI pin identical tags.

> Resource note: Testcontainers **default** is Postgres + **one** Redis
> (`CACHE_ADDR` and `KV_ADDR` share it). Production compose stays two
> processes (`cache-volatile` / `cache-durable`). Isolation proofs that
> stop one role (`TestFailOpen_VolatileDown`, file upload outage split,
> T10 eviction) skip unless `TEST_KV_DUAL=1`.
>
> Dual-KV full suite: `make test-pretty-dual`. On constrained Docker
> hosts also run packages sequentially (`go test -p 1 ./test/...`).

## 3. Exercise flags locally (all default off)

```bash
CACHE_ENABLED=true CACHE_PRODUCT_DETAIL=true go run .
# Same product twice → second response served without DB query (check debug logs/metrics).
# Update the product as seller → next read reflects the write (miss + refill).
CACHE_SET_WRITES=false go run .   # serve reads, skip all domain SETs (write-shed drill)
```

Enable one flag at a time; staging before any shared environment.

## 4. Simulate the incident modes

```bash
# Dead volatile backend → reads still 200 from DB (T7/T14 behavior)
docker compose stop cache-volatile && <browse catalog> && docker compose start cache-volatile
# Flood one-off queries → markers only, no big SETs, memory flat (T13 behavior)
```

## 5. Observe

Per-`(module, op, store)` hit/miss/error/latency counters; `cache_set_async_dropped`, `cache_set_generation_dropped`, `cache_write_shed_active`, `cache_fill_inflight`. Pollution signature to watch: miss ≈100% + climbing SET rate on one namespace → shed writes, investigate query shapes.

## 6. Cutover drill (staging)

Flip `CACHE_ADDR`/`KV_ADDR` to the Dragonfly pair, bake, compare error rate + p99 against baseline; revert is the same two variables back. No deploy in either direction.
