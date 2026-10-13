# Dashboards & Alerts: Caching Infrastructure

Binding: [pre-spec.md](./pre-spec.md) §8.2, [quickstart.md](./quickstart.md) §5.
Status: log-sink metrics are the source — **no metrics-backend deploy in 011**.
Every rule below ships as a Prometheus stub (same label shape, for the day
the sink lands) plus the log query that fires it today.

## 1. Signal source

`cachekit.Recorder` (`common/cachekit/metrics.go`, `LogRecorder` default)
emits one structured event per cache op with fields:

| Field | Values |
|---|---|
| `module` | `product`, `user`, `payment`, `file`, `cachekit`, `inventory` |
| `op` | per-area op (`detail`, `seller-validation`, `set`, `compare-and-set`, …) |
| `store` | `cache` (volatile) \| `kv` (durable) |
| `latency_ms` | per-op latency |
| level/msg | `error`/`cache op error`, `warning`/`cache write dropped`, `debug`/`cache op <hit\|miss>` |

Gauges are polled, not emitted: `Flight.Inflight()` (cache_fill_inflight),
`Adapter.Stats()` (`Inflight`, `ShedActive`, drop counters). Backend `INFO`
(memory, evicted keys, clients, per store) is corroboration only — Dragonfly
field names differ from Redis (research R4).

## 2. Signal catalog

| Signal | Kind | Labels | Emitted where |
|---|---|---|---|
| `cache_hit` | counter (debug) | module, op, store | `FetchBytes`, `ListCache.Get`, strategies |
| `cache_miss` | counter (debug, incl. tombstone) | module, op, store | fill paths |
| `cache_error` | counter (error) | module, op, store | GET/SET backend failures |
| `cache_negative_hit` | counter (debug) | module, op | tombstone serves |
| `cache_value_too_large` | counter (warning) | module, op | size-guard skips (via dropped) |
| `cache_set_async_dropped` | counter (warning, op=`set`) | module=`cachekit`, store=`cache` | pool-full / shed / oversize skips |
| `cache_set_generation_dropped` | counter (warning, op=`compare-and-set`) | module=`cachekit` | stale guarded SETs |
| `cache_op_latency_ms` | histogram | module, op, result, store | every recorded op |
| `cache_fill_inflight` | gauge (poll `Flight.Inflight()`) | per strategy | singleflight pressure |
| `cache_write_shed_active` | gauge (poll `Adapter.ShedActive()`) | per process | 1 = manual lever off or breaker open |

## 3. Dashboard (minimum viable)

1. Hit rate by `(module, op, store)` — post-warmup catalog detail > 80%.
2. Miss rate by namespace — spike on a stable key = invalidation bug.
3. Error rate by store — spike = backend down (fail-open working if HTTP 200s hold).
4. p99 `cache_op_latency_ms` by `(op, store)` — volatile GET p99 < 20ms when up.
5. `cache_set_async_dropped` rate — climbing = pool saturation or shed.
6. `cache_write_shed_active` — must be 0; 1 pages §4.
7. `cache_fill_inflight` — sustained high = stampede pressure.
8. Backend `INFO` memory/evicted/clients per store — durable evictions must be 0.

## 4. Alert rules

Thresholds are bake-window starting points — tune against `baselines.md`.

| # | Alert | Severity | Condition (PromQL stub) | Log query today |
|---|---|---|---|---|
| A1 | CacheBackendDown | page | `rate(cache_error[5m]) > 0.05` by store | `msg="cache op error"` grouped by `store`, rate jump |
| A2 | DurableEvictionPage | **page immediately** | backend `evicted_keys > 0` on durable KV | backend `INFO` poll per deploy check |
| A3 | InvalidationBug | ticket | `miss rate ≈100%` on a stable key + flat SET rate | `msg="cache op miss"` by `(module,op)` sustained |
| A4 | DomainP99Breach | ticket | `histogram_quantile(0.99, cache_op_latency_ms) > 200ms` | `latency_ms` p99 by `(op,store)` |
| A5 | WriteStormPollution | page | `miss ≈100%` **and** climbing `set` rate on one namespace | miss flood + `cache write dropped` flood, same namespace |
| A6 | WriteShedActive | notify | `cache_write_shed_active == 1` | `ShedActive()` / shed-skip warnings |
| A7 | DetailHitRateLow | ticket | `hit/(hit+miss) < 0.8` 10m after warmup, `op=detail` | hit/miss ratio by op |

```yaml
# Prometheus stubs (011: documentation only — no exporter deployed).
# Labels mirror the log fields so queries survive the sink swap.
groups:
  - name: cachekit
    rules:
      - alert: CacheBackendDown
        expr: sum by (store) (rate(cache_error[5m])) > 0.05
        for: 2m
        labels: {severity: page}
      - alert: WriteStormPollution
        expr: sum by (module) (rate(cache_miss[5m])) > 100 and sum by (module) (rate(cache_set_async_dropped[5m])) > 10
        for: 2m
        labels: {severity: page}
      - alert: WriteShedActive
        expr: cache_write_shed_active == 1
        for: 1m
        labels: {severity: notify}
```

## 5. Pollution playbook (A5)

Signature: miss ≈100% + climbing SET rate on one namespace = random-query
flood (§0.4). Response, in order:

1. `CACHE_SET_WRITES=false` on the volatile role (serve GETs, skip all
   domain SETs — the manual lever; instant, no deploy).
2. Confirm `cache_write_shed_active == 1` and p99 flat.
3. Investigate query shapes (admission markers show cardinality, not content).
4. Fix capacity/indexes; re-enable writes; watch hit rate recover.
5. Do NOT "fix" with longer TTLs or more cache memory — caching cannot
   absorb traffic that never repeats (pre-spec §4.4).

## 6. Upstream retry discipline (FR-018)

- Cache GET/SET take exactly one fast attempt (fail open / drop and count).
  Retrying cache ops adds tail latency for zero benefit — the DB fallback
  *is* the retry. (Durable-KV ops keep bounded retries; they have no DB
  fallback.)
- Upstream request retries (gateway/clients) MUST use bounded attempts with
  exponential backoff and full jitter. Unjittered fixed-backoff retries
  re-synchronize into waves that grow instead of decay (§0.4 link 5→6).
