# Quickstart: Coupon / Discount Codes

**Feature**: 007-coupon-discount-codes  
**Date**: 2026-08-01

## Overview

Sellers manage discount codes and scopes. Customers apply coupons on authenticated carts. The backend recalculates promotions + coupons on every cart price and snapshots coupons at checkout.

## Prerequisites

- Go 1.25+
- Docker (Testcontainers for integration tests)
- PostgreSQL 16 / Redis via project docker-compose or test setup
- `go mod download`
- Branch: `007-coupon-discount-codes`

## How pricing works

```
Get Cart / Apply Coupon / Remove Coupon
  → load cart items + variant prices
  → ApplyPromotionsToCart (existing)
  → load cart_applied_coupon IDs
  → ApplyCouponsToCart (new)
  → self-heal invalid attachments
  → CartResponse (items, promos, coupons, availableCoupons, summary)
```

Cart tables store **references only**. Money is computed at runtime. Order tables store **immutable** coupon snapshots.

## Key API paths

| Actor | Examples |
| --- | --- |
| Seller | `POST/GET/PUT/PATCH/DELETE /api/promotion/discount-code` |
| Seller | `/api/promotion/discount-code/scope/{product\|variant\|category\|collection}` |
| Customer | `POST /api/order/cart/coupon` `{ "code": "SAVE20" }` |
| Customer | `DELETE /api/order/cart/coupon/:code`, `DELETE /api/order/cart/coupon` |
| Customer | `GET /api/order/cart`, `GET /api/order/cart/available-coupon` |

## Database

```bash
make migrate
```

Migration (planned): align `discount_code` columns; create `cart_applied_coupon`; create `cart_item_promotion`; indexes. See [data-model.md](./data-model.md).

## Manual smoke (after implementation)

1. Seller login → create percentage code `SAVE10`, activate, optionally add product scope.  
2. Customer login → add items → `POST /api/order/cart/coupon` with `SAVE10`.  
3. `GET /api/order/cart` → verify `appliedCoupons`, `summary.couponDiscount`, `totalDiscount`.  
4. Place order → verify order applied coupons + code usage incremented.  
5. Re-apply beyond per-customer limit → expect `COUPON_ALREADY_USED` / usage error.

## Running tests (planned locations)

```bash
# Seller + scope
go test ./test/integration/promotion/ -run DiscountCode -v

# Cart coupons + checkout
go test ./test/integration/order/ -run Coupon -v

# Or full suite
make test-pretty
```

Coverage targets are listed in [pre-spec.md](./pre-spec.md) §9 (T-S*, T-SC*, T-C*, T-E*).

## Docs map

| Doc | Use |
| --- | --- |
| [spec.md](./spec.md) | WHAT / WHY |
| [plan.md](./plan.md) | Implementation plan |
| [research.md](./research.md) | Decisions |
| [data-model.md](./data-model.md) | Schema |
| [contracts/api-contracts.md](./contracts/api-contracts.md) | API contracts |
| [pre-spec.md](./pre-spec.md) | Detailed HOW draft |

## Next command

```text
/speckit.tasks
```

Generates dependency-ordered `tasks.md` for phased TDD implementation.
