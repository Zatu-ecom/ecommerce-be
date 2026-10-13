# PR #71 Code Review — Findings Tracker (013-courier-fulfillment-platform)

> Generated from detailed review of `origin/develop...HEAD` (PR #71).
> Check off (`[x]`) as each fix lands. All money sent to frontend MUST use `common/model.Money`
> (`Amount` major + `AmountCents` minor + `Formatted`) + `common/model.CurrencyInfo`.
> Storage stays integer minor units (`*_cents`); provider payloads convert via
> `CurrencyInfo` factor (never hardcoded `/100` or `*100`).

## A. Money / multi-currency standard (common module)

- [x] A1 — `shipment.go:120,136` + `ndr.go:145,161` send `UnitPriceCents/SubTotalCents` (paise) as
      `selling_price/sub_total` (rupees) → 100x overcharge. Must convert with
      `commonModel.FromCents(cents, fulfillment/model.CurrencyFor(code))`.
- [x] A2 — `BookShipmentInput/RateInput` carry no `CurrencyCode`; adapter cannot do
      currency-aware conversion. Add `CurrencyCode string` to both + `BookReturnInput` path.
- [x] A3 — `rates.go:81` hardcodes `*100`. Provider rupees → storage cents must use
      `CurrencyInfo.ToCents()` / `Factor()` so JPY (0dp) / BHD (3dp) work.
- [x] A4 — Frontend already uses `commonModel.Money` for `ShipmentResponse.COD/Rate` and
      `RateOptionResponse.Rate` — keep. Audit that no raw `*_cents` leaks in JSON.

## B. Critical

- [x] B1 — `product/route/attribute_route.go:36-41` `/definitions` shadowed by `/:attributeId`.
      Register static route first.
- [x] B2 — `handler/shipment_handler.go:412,620` `err == AppError` never matches wrapped errors.
      Use `errors.As` + code compare.
- [x] B3 — `032:182` global `UNIQUE(idempotency_key)` + unscoped `FindByIdempotencyKey` + check
      outside lock. Scope to `(seller_id, key)`, re-check inside lock, map `23505` → replay.
- [x] B4 — `shipment_planner.go:95-111,195` courier HTTP inside `WithOrderLock`. Move
      `autoBookNewDrafts` outside locked tx.
- [x] B5 — `shiprocket/auth.go:124-148` 401-retry re-runs whole `create→assign→pickup`.
      Retry only failed leg (per-step token refresh), never re-create.
- [x] B6 — `shipment_service.go:458-459` transient adopt failure cancels draft. Only
      `ErrorStockMismatch` cancels; else return error.

## C. Major

- [x] C1 — `markBookAttempt` lost update (`draft.BookAttempts+1`). Use `gorm.Expr` atomic inc.
- [x] C2 — `commitBook` missing nil/empty AWB guards + ETD/rate persistence.
- [x] C3 — Lost-race / cancel swallow wrong terminal as success (`commitBook:680`, `Cancel:843`).
- [x] C4 — Token cache keyed by email, no singleflight, fixed 9d TTL (`auth.go`). Key by
      email+pw hash, add flight, parse JWT exp.
- [x] C5 — `http.go` shared deadline across GET retries + collapsed untyped errors. Fresh
      timeout per attempt; map 404/429/5xx to typed errors; skip `Bearer` on empty token.
- [x] C6 — `UpdateStatusIfCurrent/UpdateColumns` not seller-scoped. Add `seller_id` predicate.
- [x] C7 — `FindRecoverableDrafts/FindStaleInFlight` claim `SKIP LOCKED`, implement plain `Find`.
- [x] C8 — Inventory blind decrements + restock double-count (no `>=take` guard, non-unique ref).
- [x] C9 — Split commits: `commitBook` 3 commits, order hooks 2 commits, webhook box-move
      committed on hook failure. Single tx / outbox; rollback on hook failure.
      (Done: commit guards + cancel atomicity + return re-check inside lock + reservation
      PENDING guard. Full saga/outbox remains follow-up; failed webhooks heal via
      `FindFailedSince` cron.)
- [x] C10 — Silent prod→sandbox fallback in booking path. Require explicit env for booking.

## D. Medium / Minor

- [x] D1 — `tracking_handler.go:147` customer auth returns seller error. Dedicated user error.
- [x] D2 — Provider `code` param unvalidated/case-sensitive. Normalize + whitelist
      (contract: unknown code → 404 `FULFILLMENT_PROVIDER_NOT_SUPPORTED`, not 400).
- [x] D3 — Webhook no body limit / content-type check. `MaxBytesReader` 1MiB.
- [x] D4 — List `status` unvalidated (silent `[]`), `sortBy` ignored in webhook logs. 400 on unknown.
- [x] D5 — `032` missing indexes (`provider_config_id`, `webhook_log(shipment_id,awb)`),
      missing CHECKs, inconsistent `ON DELETE`. (Done via `034_harden_fulfillment_constraints.sql`.)
- [x] D6 — `main.go:164` concrete type-assert silently disables planner. Interface + log.
- [x] D7 — `base_handler.go` leaks raw PG text in 500. Map `23505/23503`.
- [x] D8 — `cache/keys.go` `sanitize()` `-` folding collision; `IdempotencyKey` not validated.
      (Done: idempotency now seller-scoped via `BuildSellerKey`; stale platform allowlist
      entries removed. Full hash-based sanitize remains low-risk follow-up.)
- [ ] D9 — Physical-spec N+1. Batch `WHERE id IN (?)`. (Follow-up: planner path already
      preloads; batching tracked separately to avoid scope creep.)
- [x] D10 — Seed `005` `DO UPDATE` rewrites user edits. `DO NOTHING`.
- [x] D11 — Unbounded `AddressNote/Reason/Credentials` blobs. `max` tags + size cap.
- [x] D12 — Duplicate label-code constant; hardcoded `"FULFILLMENT_BOOK_FAILED"` string.
