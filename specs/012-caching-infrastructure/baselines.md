# Cache Baselines (012 Phase 0 exit criterion)

P1/P2 "DB QPS down" claims are measured against this file. Values below are
captured on **staging** with all `CACHE_*` flags off, seeded catalog, and steady
synthetic traffic. Do not turn any flag on before this table is filled.

## Method

1. Start staging on the pre-012 image (or `CACHE_ENABLED=false`).
2. Warm up 10 minutes with the load script (`test/integration/cachekit/flood_test.go`, T059).
3. Record, per path: p50/p99 latency, requests/s, Postgres `tup_fetched` + active connections (`pg_stat_database`, `pg_stat_activity`), volatile/durable backend `INFO` (memory, ops/s).
4. Repeat after each flag enable; attach deltas to the flag's rollout note.

## Baseline table (staging, flags off)

| Path | p50 | p99 | rps | DB reads/s | Notes |
|---|---|---|---|---|---|
| Auth seller check (middleware) | TBD | TBD | TBD | TBD | per-request overhead of `seller_complete` query |
| `GET /api/product/:id` | TBD | TBD | TBD | TBD | detail + variants + options + media |
| `GET /api/product/category*` | TBD | TBD | TBD | TBD | tree + attributes |
| `GET /api/product` list + `/search` | TBD | TBD | TBD | TBD | P2 comparison basis |
| `GET /api/user/country`, `/currency` | TBD | TBD | TBD | TBD | — |
| `GET /api/payment/gateways` | TBD | TBD | TBD | TBD | `1+1+2N` query shape |
| `GET /api/inventory/summary/available` | TBD | TBD | TBD | TBD | dashboard; never cached (reference only) |
| Internal `GetTotalAvailableQuantities` | TBD | TBD | TBD | TBD | P2a comparison basis |

## Snappiness budget (SC-001)

Fill after the first capture: catalog detail p99 budget = _____ ms. The budget is this table's p99 with headroom, not an aspirational number.
