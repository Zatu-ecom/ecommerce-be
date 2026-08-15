# Quickstart: Money & Currency Standardization

**Branch**: `008-money-currency-standardization`  
**Spec**: [spec.md](../spec.md) | **Plan**: [plan.md](../plan.md)

## Prerequisites

- Go 1.25+, Docker (Testcontainers)
- Checkout feature branch
- Read [contracts/money-currency.md](./contracts/money-currency.md)

## Breaking-change note

Client payloads for money-bearing APIs change shape. Coordinate admin + storefront before enabling in shared environments.

## Local verify (after implementation)

### 1. Unit — common model

```bash
go test ./test/common/model/ -count=1
```

Cover: INR 2dp, JPY 0dp, KWD 3dp, excess-precision reject, format.

### 2. Integration — module-scoped suites

Tests live under `test/integration/<module>/` (product, order, promotion, report), matching the existing convention — no cross-module `money/` suite.

```bash
go test ./test/integration/product/... -count=1 -timeout 20m
go test ./test/integration/order/... -count=1 -timeout 20m
go test ./test/integration/promotion/... -count=1 -timeout 20m
go test ./test/integration/report/... -count=1 -timeout 20m
```

Must cover (per owning module):

- Product suite: INR/JPY seller price write → DB cents → Money response; excess-precision reject; package option rules (pre-spec §14.3 A)
- Order suite: seeded INR variant → cart → order round-trip; JPY no `*100` bug; guest cart same Money contract; fixed coupon cents coherent (pre-spec §14.3 B)
- Promotion suite: fixed promo/coupon major input → cents; JPY `50` ≠ `5000`; thresholds (pre-spec §14.3 C)
- Every suite: response shape assertions (`amount`, `amountCents`, `formatted`, parent `currency`), no legacy bare ints (pre-spec §14.3 E)

### 3. Regression modules

Re-run the touched suites to confirm no regressions:

```bash
go test ./test/integration/product/... -count=1 -timeout 20m
go test ./test/integration/order/... -count=1 -timeout 20m
go test ./test/integration/promotion/... -count=1 -timeout 20m
go test ./test/integration/report/... -count=1 -timeout 20m
```

### 4. Import cycle check

```bash
go list -f '{{.ImportPath}} {{.Imports}}' ./common/model | grep -E 'ecommerce-be/(user|product|order|promotion)' && exit 1 || echo OK
```

## Manual smoke (optional)

1. Set seller base currency INR; create variant `price: 99.99`; GET → Money `9999` cents.
2. Add to cart qty 2; confirm line `19998` cents / amounts.
3. Place order; confirm order totals match cart cents.
4. Repeat with JPY seller `price: 100`.

## Implementation order reminder

1. `common/model` + `test/common/model/` unit tests  
2. Product migration + APIs  
3. Cart/order  
4. Promotion/discount/report  
5. Remove aliases; CI gate  

Do not ship product `price_cents` without cart reading `PriceCents` in the same release.
