# Research: Money & Currency Standardization

**Feature**: `008-money-currency-standardization`  
**Date**: 2026-08-02  
**Sources**: `spec.md`, `plans/009-money-currency-standardization/pre-spec.md`, constitution, existing cart/order/product code

---

## 1. Shared contract location

**Decision**: Put Money, CurrencyInfo, and relocated API envelope/pagination/validation types in `common/model` (package `model`). Do **not** create `common/money`.

**Rationale**: Money is a request/response contract like pagination; living beside other shared models matches team preference and Spec FR-012/FR-013. Encapsulation: one file per contract with type + methods.

**Alternatives considered**:
- `common/money` package — rejected (user/pre-spec).
- Keep types in `common` root — rejected (response.go already overloaded; poor encapsulation).

---

## 2. Money response shape

**Decision**: Nested object:

```json
{ "amount": 99.99, "amountCents": 9999, "formatted": "₹99.99" }
```

Parent currency once:

```json
{ "code": "INR", "symbol": "₹", "decimalDigits": 2 }
```

**Rationale**: Spec/pre-spec lock; same shape everywhere; cents remain visible for clients that need exact integers; formatted aids UI.

**Alternatives considered**:
- Flat `price` / `priceCents` / `priceFormatted` — workable but more rename noise per field; nested is consistent for every money slot.
- Amount as string decimal — safer for huge values; deferred; use float64 major + int64 cents with documented limits.

---

## 3. Write path currency & precision

**Decision**: Writes interpret major units in **seller base currency** (`seller_settings.base_currency_id`). Validate fractional digits ≤ `decimalDigits`; **reject** excess (no silent round). Convert via encapsulated methods using `10^decimalDigits`. Never accept client cents on write in v1.

**Rationale**: Spec FR-001/002/008; avoids JPY/`*100` bugs; seller does not send currency code per field.

**Alternatives considered**:
- Silent half-up round on excess digits — rejected (hides client bugs).
- Allow optional `amountCents` on write — rejected for v1 dual-input conflicts.

---

## 4. Storage & catalog rename

**Decision**: Migrate `product_variant.price` and `package_option.price` (`DOUBLE PRECISION`) → `price_cents BIGINT NOT NULL`. Backfill `ROUND(price * 100)::BIGINT` after auditing non-2dp sellers. Keep order/payment/promo amount columns as existing `*_cents`.

**Rationale**: Spec FR-003/011; aligns naming with order tables; removes float bridge in cart (`variant.Price * 100`).

**Alternatives considered**:
- Keep column name `price` as BIGINT — rejected (ambiguous forever).
- NUMERIC major units in DB — rejected (spec requires integer minor-unit calc domain).

---

## 5. Display currency until FX

**Decision**: All Money presentation uses **seller base currency**, even if buyer `currency_id` differs and `display_prices_in_buyer_currency` is true. No FX in this feature.

**Rationale**: Spec FR-010 / out of scope FX; avoids lying about converted amounts without rates.

**Alternatives considered**:
- Use GetPreferredCurrency for formatting digits only when codes match — optional optimization; still no FX.

---

## 6. Breaking change strategy

**Decision**: Coordinated breaking change in one release with frontend (admin + storefront). Document field/semantic changes in contracts + quickstart. Optional short-lived type aliases in `common/response.go` for Go compile migration only (not dual JSON keys).

**Rationale**: Spec assumption; Constitution VIII satisfied via explicit documentation + lead approval. Dual JSON keys prolong confusion (pre-spec).

**Alternatives considered**:
- `/api/v2` — larger operational cost.
- Dual keys indefinitely — rejected.

---

## 7. Promotion / discount-code money I/O

**Decision**: Request fields use major-unit names (`minPurchaseAmount`, `maxDiscountAmount`, config `amount`, etc.). Persist cents in DB columns / JSONB keys (`*_cents`). Responses use Money for monetary fields. Percentage values stay plain numbers. Keep discount_code overloaded `value` column interpreted by `discountType` (no column split in v1).

**Rationale**: Spec FR-009/015; minimal schema churn; strategies already cents-based.

**Alternatives considered**:
- Split `percentage_value` / `amount_cents` columns — cleaner long-term; defer unless needed.

---

## 8. Dependency / encapsulation rules

**Decision**:
- `common/model` is pure: no imports of user/product/order/promotion.
- User owns DB currency resolution; maps to `model.CurrencyInfo`.
- Domain services: resolve currency → call `CurrencyInfo`/`Money` methods → persist cents.
- All money methods for `Money` live in `money.go`; currency helpers in `currency.go`; no `money_convert.go` sprawl.

**Rationale**: Constitution I + pre-spec encapsulation; prevents cycles.

---

## 9. Testing strategy

**Decision**: All tests live under `test/` organized **by module** (existing convention — no cross-module suite). Pre-spec §14.3 A–E scenarios map to owning module suites:

| Scenario group | Owning suite |
| --- | --- |
| A (product write → DB → read; excess precision; package option; min/max filters) | `test/integration/product/` (`variant/`, `package_option/`) |
| B (product → cart → order round-trip; JPY no `*100`; guest cart; fixed coupon) | `test/integration/order/` (owns cart/order; reuses seeded variants as existing order suite does) |
| C (promo/discount-code seller currency; thresholds) | `test/integration/promotion/` |
| D (cross-seller isolation) | covered inside owning suites (product + order auth) |
| E (contract shape, no legacy bare ints) | schema assertions within each module's suite + `test/common/model/` unit tests |

Unit tests for shared contracts go in `test/common/model/` (`money_test.go`, `currency_test.go`, `pagination_test.go`), mirroring source paths — not co-located in `common/model/`. TDD: write failing tests first per constitution IV. Round-trip failures block merge.

**Rationale**: Spec FR-014 / User Story 5; pre-spec §14; matches repo test convention (module-scoped suites only, cross-module flows via seed data).

---

## 10. Report module

**Decision**: Return revenue/discount metrics as Money (or equivalent shared shape) using seller currency digits; stop assuming `/100.0` in SQL without going through money helpers (prefer aggregate cents then `NewMoney`).

**Rationale**: Spec FR-007; consistency.

---

## Resolved clarifications

No remaining NEEDS CLARIFICATION items for planning. Open pre-spec items mapped to decisions above.
