# Contracts: Module Cache Strategy

Binding: [pre-spec.md](./pre-spec.md) §§3–4. Every module that caches owns a `cache/` package following this contract. Strategies contain business caching rules only — no client code, no provider imports.

## Shape (per module, e.g. `product/cache/`)

```go
// Reader: cache-aside fill for one resource. On hit returns the DTO bytes.
// On miss uses singleflight → loads via the owning service/repository path,
// stores async (never blocking), and returns. Any cache failure → DB path.
GetProduct(ctx, sellerID, productID uint) ([]byte, error)

// Writer hook: called by the service AFTER DB commit. Deletes entity keys
// (+tombstones, same keys) and bumps list versions. Best-effort; failures
// are logged + counted, never returned as request errors.
InvalidateProduct(ctx, sellerID, productID uint, affectedVariantIDs []uint) error
```

Conventions every strategy follows:

1. Keys built only via `cachekit` key builder (tenant scope enforced; platform keys from the closed allowlist).
2. TTLs passed as base durations; jitter applied inside the client on every SET.
3. Stored payloads are module DTO bytes honoring the payload contracts (pre-spec §4.3 for catalog: no stock, no signed URLs, no personalization).
4. Reads: hit → return; miss → singleflight fill → return first, SET async.
5. Writes: `Del` exact keys + `IncrVersion` where lists exist; no prefix deletes on the request path.
6. Negative results stored as same-key tombstones with short TTL; invalidated by the same `Del`.
7. Metrics labeled with the owning module name on every path.

## Registry binding

No strategy may introduce a key, TTL, or cached resource not named in pre-spec §4.5 (consistency rule). Adding a resource = one §4.5 row + matching key-table row + flag + tests, in the same change.

## Per-module checklist (applied at implementation per phase)

- product: detail/variant/category/attribute/option strategies + §5.7 nested-invalidation matrix + P2 list/admission strategy.
- user: seller-validation (subscription-end-capped TTL), currency, geo, settings strategies + exact-`Del` on all writers incl. delete.
- payment: catalog-slice strategy (public metadata only; seller overlay stays live in the service).
- file: providers/schema strategies (no URLs/secrets).
- inventory (P2a, post-guard): availability micro-cache strategy, TTL-only, bypass on writes.
