# Data Model: Money & Currency Standardization

**Feature**: `008-money-currency-standardization`  
**Date**: 2026-08-02

---

## 1. Shared DTOs (`common/model`)

### CurrencyInfo

| Field | JSON | Type | Rules |
| ----- | ---- | ---- | ----- |
| Code | `code` | string | ISO 4217, len 3 |
| Symbol | `symbol` | string | display symbol |
| DecimalDigits | `decimalDigits` | int | 0–4 |

**Behavior (same file `currency.go`)**: `Factor() int64`, digit validation helpers as needed.

### Money

| Field | JSON | Type | Rules |
| ----- | ---- | ---- | ----- |
| Amount | `amount` | float64 | major units; derived from cents + decimalDigits |
| AmountCents | `amountCents` | int64 | minor units; source of truth outbound |
| Formatted | `formatted` | string | `symbol` + amount with `decimalDigits` |

**Behavior (same file `money.go`)**: `NewMoney(cents, CurrencyInfo)`, `FromCents`, `ToCents` / validate major amount (reject excess precision), `Format`, money-specific errors. Encapsulated — no split helper files.

### Relocated contracts (from `common/response.go`)

| Type | File | Behavior in same file |
| ---- | ---- | --------------------- |
| `Response`, `ErrorResponse` | `api_response.go` | `SuccessResponse`, `ErrorWithValidation`, `ErrorWithCode`, `ErrorResp` |
| `BaseListParams`, `PaginationResponse` | `pagination.go` | `SetDefaults`, `NewPaginationResponse` |
| `ValidationError` | `validation_error.go` | constructors if any |

---

## 2. Persistence changes

### product_variant

| Column | Before | After |
| ------ | ------ | ----- |
| `price` | `DOUBLE PRECISION NOT NULL` | **dropped** |
| `price_cents` | — | `BIGINT NOT NULL` |

Indexes: replace price-based indexes with `(product_id, price_cents)` as needed.  
Entity: `PriceCents int64` `gorm:"column:price_cents"`.

### package_option

| Column | Before | After |
| ------ | ------ | ----- |
| `price` | `DOUBLE PRECISION NOT NULL` | **dropped** |
| `price_cents` | — | `BIGINT NOT NULL` |

### Related products

SQL function/aggregates: `min_price`/`max_price` → minor-unit equivalents (`min_price_cents` / `max_price_cents` or compute from `price_cents`).

### Unchanged (already cents)

- `order`: `subtotal_cents`, `discount_cents`, `shipping_cents`, `tax_cents`, `total_cents`
- `order_item`: `unit_price_cents`, `line_total_cents`
- Promo/discount usage and snapshot tables with `*_cents`
- `discount_code.max_discount_amount_cents`, `min_purchase_amount_cents`
- `discount_code.value` — still BIGINT; semantics by `discount_type` (percent vs fixed cents)

### discount_config JSONB (internal)

Keep cents keys for strategies (`amount_cents`, `min_order_cents`, …). API accepts major names; mapper converts at boundary.

---

## 3. Domain request fields (major units in)

| Area | Request field (major) | Stored |
| ---- | --------------------- | ------ |
| Product/variant/package | `price` | `price_cents` |
| List filters | `minPrice`, `maxPrice` | compared as cents |
| Promotion | `minPurchaseAmount`, `maxDiscountAmount` | `*_cents` columns |
| Promotion config | `amount`, `minOrder`, `maxDiscount`, `bundlePrice`, … | JSONB `*_cents` |
| Discount code | `value` (fixed = major), `minPurchaseAmount`, `maxDiscountAmount` | cents in DB |

Percentage fields unchanged (not money).

---

## 4. Domain response embedding

Money-bearing parents include:

```text
currency: CurrencyInfo
<moneyField>: Money
```

Examples:
- Variant/product: `price: Money`
- Cart item: `unitPrice`, `lineTotal`, discounts, `discountedLineTotal`
- Cart summary: `subtotal`, `tax`, `shipping`, `total`, discount aggregates
- Order: `subtotal`, `discount`, `shipping`, `tax`, `total`; item `unitPrice`, `lineTotal`
- Coupons/promos: fixed `value` / thresholds / discounts as Money when monetary

Remove wire-only ambiguous ints (`unitPrice` as raw cents) and bare `subtotalCents` without Money wrapper.

---

## 5. Validation rules

1. `amount` must be finite; `> 0` or `>= 0` per field business rules (prices typically `> 0`).
2. Fractional digit count ≤ `CurrencyInfo.DecimalDigits` or reject.
3. `ToCents` uses half-up after scale only when precision valid.
4. Seller currency must exist and be active.
5. Calculations never re-enter float major units except at response mapping via `NewMoney`.

---

## 6. Migration sketch

`migrations/030_money_currency_standardization.sql` (number may adjust):

1. Add `price_cents` nullable.
2. Backfill from `price` with `ROUND(price * 100)::BIGINT` (post-audit).
3. Set NOT NULL; drop `price`; recreate indexes.
4. Same for `package_option`.
5. Replace/update related-products function.

Deploy product + cart consumers in same release.
