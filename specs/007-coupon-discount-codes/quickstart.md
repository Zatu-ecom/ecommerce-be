# Quickstart: Coupon / Discount Codes

**Feature**: 007-coupon-discount-codes  
**Date**: 2026-08-01  
**Status**: Implemented (Phases 1–8)

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
  → sync cart_item_promotion (reference IDs only)
  → load cart_applied_coupon IDs
  → ApplyCouponsToCart
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
| Customer | `POST /api/order` (checkout snapshots coupons + records usage) |

## Database

```bash
make migrate
```

Migration `028_align_discount_code_and_cart_coupon.sql` aligns `discount_code` columns and creates `cart_applied_coupon` + `cart_item_promotion`. See [data-model.md](./data-model.md).

## Manual smoke

1. Seller login → `POST /api/promotion/discount-code` percentage code `SAVE10` → optionally add product scope.  
2. Customer login → `POST /api/order/cart/item` → `POST /api/order/cart/coupon` with `SAVE10`.  
3. `GET /api/order/cart` → verify `appliedCoupons`, `summary.couponDiscount`, `totalDiscount`.  
4. `POST /api/order` → verify `appliedCoupons` on order + `discount_code_usage` / `current_usage_count`.  
5. Re-apply beyond per-customer limit → expect `COUPON_ALREADY_USED` (or global `COUPON_USAGE_LIMIT_REACHED`).  
6. Seller `DELETE` used code → expect `409 DISCOUNT_CODE_HAS_USAGE`.

## Running tests

```bash
# Seller + scope
go test ./test/integration/promotion/ -run DiscountCode -count=1

# Cart coupons + checkout usage
go test ./test/integration/order/ -run 'CartCoupon|Coupon' -count=1

# Calculator unit tests
go test ./promotion/service/discountStrategy/ -count=1

# Full related suites
go test ./test/integration/promotion/ ./test/integration/order/ -count=1
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
| [tasks.md](./tasks.md) | Phased TDD task list |
| [pre-spec.md](./pre-spec.md) | Detailed HOW draft |
