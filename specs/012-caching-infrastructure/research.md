# Research: Caching Infrastructure

Binding: [spec.md](./spec.md), [pre-spec.md](./pre-spec.md). All unknowns below were resolved before design; nothing remains NEEDS CLARIFICATION.

## R1. Go client: `go-redis/v8` → `github.com/redis/go-redis/v9`

- **Decision**: Upgrade to `github.com/redis/go-redis/v9` (recent maintained line, v9.2x in 2026) in Phase 0; confine it to `common/cachekit/provider/`.
- **Rationale**: v8 (`v8.11.5`, frozen since 2022) lacks modern timeout/pool/retry tuning the design depends on (per-op timeouts, fine-grained pool config, retry backoff hooks). v9's `NewClient(Options{...})` shape is API-compatible for every command this codebase uses (`GET/SET/DEL/SETNX/INCR/EXPIRE/TTL/ZADD/ZRANGEBYSCORE/ZREM/EVAL/SCAN/PING`), so migration is a module-path change plus config expansion, not a rewrite.
- **Alternatives considered**: rueidis (higher throughput, auto-pipelining) — rejected for 011: bigger API churn, and the conformance suite gates a revisit only on measured pipeline saturation. Keeping v8 — rejected: cannot express the timeout/pool policy §8 requires.

## R2. DragonflyDB RESP compatibility for our command set

- **Decision**: DragonflyDB is fully compatible with every command/pattern in this design; no Dragonfly-specific code paths.
- **Rationale** (verified against Dragonfly docs, compat matrix v1.40 / Jul–Aug 2026): `EVAL` fully supported (Lua 5.4); full ZSET family (`ZADD/ZRANGE/ZRANGEBYSCORE/ZREM/ZCARD/COUNT`) supported; `SCAN` family fully supported; `SETNX/INCR/EXPIRE/TTL/PING` standard.
- **Constraint found**: Dragonfly **forbids Lua access to undeclared keys by default** (`script tried accessing undeclared key`). All our scripts (limiter `INCR+EXPIRE`, generation CAS) MUST pass every touched key via `KEYS`, never generate key names inside the script. This is already our pattern — recorded here as a hard implementation rule and a T3/T17 assertion.
- **Alternatives considered**: Redis-only — retained as one-release rollback, not as target (pre-spec §7).

## R3. Dragonfly process flags (two roles)

- **Decision**: Volatile: `--cache_mode=true --maxmemory=<computed> --requirepass=…` (no data dir). Durable: `--dir <vol> --dbfilename <name> --snapshot_cron … --requirepass=…`, `cache_mode` left `false` (default = no eviction) with a memory ceiling so pressure fails loud (`denyoom`) instead of dropping jobs.
- **Rationale**: Verified against Dragonfly flag docs — `cache_mode` defaults to `false` (unbounded growth if omitted: the volatile instance MUST set it explicitly); `--snapshot_cron` uses cron syntax; `--dir/--dbfilename` control snapshot location (Docker defaults `/data`).
- **Alternatives considered**: Single process with `noeviction` — accepted only as documented fallback, not target (catalog would share fate with jobs).

## R4. Observability: backend INFO is second priority

- **Decision**: Dashboards/alerts key off our own `cachekit` metrics first; backend `INFO` is corroboration only.
- **Rationale**: Dragonfly's `INFO` fields and eviction accounting differ from Redis (e.g., no `maxmemory_policy` semantics in cache mode). Depending on backend-specific fields would fork every dashboard at cutover.
- **Alternatives considered**: Backend-specific exporters per role — rejected as cutover friction.

## R5. Testcontainers for two backends

- **Decision**: Extend `test/integration/setup` with a second KV container profile using Testcontainers generic-container support (Redis profile keeps the existing image; Dragonfly profile pins the same tag as compose). T1–T17 run against both; PR CI runs both (Dragonfly nightly minimum if runtime hurts).
- **Rationale**: No stable official Dragonfly Testcontainers module to depend on; generic containers with pinned tags + `PING` wait strategy give identical coverage without a new dependency.
- **Alternatives considered**: Docker-Compose-based test env — rejected (breaks the repo's Testcontainers-first TDD standard).

## R6. In-process singleflight + jitter as the stampede baseline

- **Decision**: `golang.org/x/sync/singleflight` (already an indirect dependency) for per-pod miss collapse; `JitteredTTL(base, ±15%)` on every SET including negatives/markers; no distributed locks, no probabilistic early refresh in 011.
- **Rationale**: Matches the threat model — identical-key herds collapse per pod (bounded by pod count), synchronized expiry is structurally prevented by jitter, and random-query floods are handled by admission (§4.4), a class of problem locks cannot solve.
- **Alternatives considered**: Distributed SETNX leases, X-Fetch probabilistic refresh, L1 near-cache — all rejected for 011 per pre-spec §8.4/§11 (complexity without a measured trigger).

## R7. Image and version pins (exact tags in tasks/implementation)

- **Decision**: Pin `redis:7-alpine` patch tag and `docker.dragonflydb.io/dragonflydb/dragonfly:<v1.x>` tag identically in `docker-compose.yml`, Testcontainers setup, and CI; never float `latest`.
- **Rationale**: Backend behavior (Lua sandbox, eviction accounting, INFO fields) is version-sensitive; the T-suite certifies exact images.
- **Alternatives considered**: Floating tags — rejected (silent behavior drift under a passing suite).
