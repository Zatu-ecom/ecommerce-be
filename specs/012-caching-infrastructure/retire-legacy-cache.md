# Retire `common/cache/` — Migration Plan

**Status**: Complete (A–F executed and verified; package deleted).
**Goal**: Delete `common/cache/` (`redis.go`, `cache_invalidation.go`) plus dead
weight, with zero behavior change. The `cachekit` two-role platform
(volatile `Cache` + durable `Durable`) already carries every other path.

**Scope decision (confirmed)**: Remove all — including the legacy `redis`
compose service, `REDIS_HOST/PORT/PASSWORD/DB` env, legacy `cfg.Redis` addr
fields, and dead constants. Rollback targets the role services
(`cache-volatile`/`cache-durable`), never the single Redis.

## 1. Why it still exists (verified)

```
common/cache/redis.go  (legacy client, REDIS_HOST:6379)
    ▲ used by ONLY:
    ├── file/service/upload_service.go   ← raw *redis.Client (SET/GET/TTL/SETNX/DEL)
    ├── file/factory/singleton/service_factory.go:49  (GetRedisClient → ctor)
    ├── main.go (ConnectRedis/CloseRedis)
    ├── setup/server.go:81 (SetRedisClient per suite)
    └── helpers/file_storage_env.go:55-60 (save/restore dance)
        + scheduler fallbacks (reservation_scheduler:154, upload_expiry:99+)

common/cache/cache_invalidation.go  (already cachekit-backed inside)
    └── user/service/user_service.go (UpdateProfile, Ctx variant)
```

## 2. Workstreams

### A. Upload idempotency → `Durable` (only real work)

File: `file/service/upload_service.go`.

- Struct: `redisClient *redis.Client` → `durable cachekit.Durable`; the
  factory passes `cachekit.DefaultDurable()`. Nil-safe: nil Durable keeps
  today's fail-closed `ErrFileUploadStorageUnavailable`, mirroring the
  current nil-client guards.
- Keys byte-identical — `file:init:idem:{owner}:{id}:{sha256}` is already in
  the key allowlist. No data migration.
- Op mapping:

  | Site | Today | After |
  |---|---|---|
  | replay nil-client / Get | `redis.Nil`→Conflict, other→Unavailable | `ErrMiss`→Conflict, `ErrUnavailable`→Unavailable |
  | `TTL` + fallback | same fallback | identical via `Durable.TTL` |
  | `SetNX` claim | err→Unavailable, `!claimed`→replay | identical via `Durable.SetNX` |
  | `Set` record | raw err → caller maps to Unavailable | `ErrUnavailable` → same mapping |
  | `Del(context.Background())` (failure path) | no IDs (stray) | `Del(ctx, …)` — fixes it |
- Drop the `go-redis` import (blindness grep win).

### B. Scheduler fallbacks → mandatory durable

Files: `inventory/service/reservation_scheduler_service.go`,
`file/service/upload_expiry_scheduler.go`.

- Delete the `else cache.Set/Get/Del` legacy branches added during 012.
- Rationale: boot gates already forbid running the scheduler without
  durable KV; silent fallback to an evictable store reintroduces the exact
  bug 012 fixed. Nil durable on the schedule path returns the error
  (fail-closed) instead of caching nowhere.

### C. Invalidators relocate

- Move `InvalidateSeller*Ctx` into `user/cache/` (new `validation.go`:
  seller-validation keys are user-module rules). Update the single caller
  (`user/service/user_service.go` → new import). Drop the legacy no-ctx
  wrappers if no caller remains (verify first).

### D. Wiring removal

- `main.go`: delete `ConnectRedis`/`CloseRedis` calls (keep `CloseDefaults`).
- `test/integration/setup/server.go`: delete `SetRedisClient` (keep cachekit
  wiring).
- `file/factory/singleton/service_factory.go`: pass
  `cachekit.DefaultDurable()` to `NewFileUploadService`; delete the
  `GetRedisClient` lookup.
- `test/integration/helpers/file_storage_env.go`: delete the save/restore
  dance (nothing overwrites the client anymore).

### E. Infra + config cleanup

- `docker-compose.yml`: delete the legacy `redis` service (+ `redis_data`
  volume). Role services and Dragonfly profiles stay.
- `.env` / `.env.test` (git-ignored locals — devs must refresh): delete
  `REDIS_HOST/PORT/DB`; rename the shared secret `REDIS_PASSWORD` →
  `CACHE_PASSWORD` (role services + healthchecks + app config + deploy
  workflow; GCP secret names unchanged); keep `CACHE_*`/`KV_*`
  (`KV_PASSWORD` fallback stays). Client-tuning vars (`REDIS_POOL_*`,
  `REDIS_*_TIMEOUT_MS`, `REDIS_MAX_RETRIES`) keep their names: they
  configure the v9 client for both roles, not the removed instance.
- `common/config/redis.go`: delete legacy `Host/Port/Password/DB/Addr`
  fields + loader lines; drop the `REDIS_HOST` required check in
  `Validate()`. Update deploy workflow lines that write `REDIS_*` into
  `.env` (deploy-gcp.yml).
- Delete dead `common/constants/cache_constants.go` + `redis_constants.go`
  (zero importers).

### F. Delete + verify

- `rm common/cache/` (both files).
- `go build ./...`; blindness greps must show **zero** `go-redis` outside
  `cachekit/provider/` (non-test) and zero `common/cache` importers;
  `KEYS`/detached-ctx greps clean.
- Suites: `TestUploadSuite` (outage + replay/conflict fail-closed paths),
  order + payment no-cache guards, inventory reservation, file config.
- Known pre-existing failures to compare against, not fix:
  `TestInitUpload_StorageOutage`, Minio pull-denied suites.

## 3. Risks

- Upload replay/conflict branching must stay exact (the
  `ErrMiss`↔`redis.Nil` mapping is the subtle one; existing conflict-path
  tests cover it).
- Deploy workflow `.env` edits need a matching infra change outside code.
- Devs must refresh local `.env` (note in the commit message).

## 4. Execution order

A → B → C → D → E → F, verifying each workstream (build + targeted
suites) before the next. A is the only behavior-sensitive step; B–E are
mechanical deletions.
