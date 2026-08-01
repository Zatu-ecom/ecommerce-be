# Data Model: Coupon / Discount Codes

**Feature**: 007-coupon-discount-codes  
**Date**: 2026-08-01  
**Source**: Entity audit + [pre-spec.md](./pre-spec.md) §2

## Overview

Most coupon tables already exist from migration `005`. This feature **aligns** schema with Go entities, **creates** missing cart junction tables, and uses existing `order_applied_coupon` for checkout snapshots.

**Runtime rule**: Cart junction tables store **IDs only**. Monetary discounts are computed when pricing the cart and frozen on order snapshot tables at checkout.

---

## Entity: DiscountCode

**Go**: `promotion/entity/discount_code.go`  
**Table**: `discount_code` (exists; needs ALTER)

| Column | Type | Notes |
| --- | --- | --- |
| id, created_at, updated_at | PK + timestamps | `BaseEntity` |
| seller_id | BIGINT NOT NULL | Tenant owner |
| code | VARCHAR(50) NOT NULL | Unique per `(seller_id, code)`; store normalized uppercase |
| title | VARCHAR(255) | Optional |
| description | TEXT | **ADD via migration** (on entity already) |
| discount_type | VARCHAR(50) NOT NULL | `percentage` \| `fixed_amount` \| `free_shipping` \| `buy_x_get_y` |
| value | BIGINT NOT NULL | % points or cents depending on type |
| max_discount_amount_cents | BIGINT | **ADD via migration** |
| applies_to | VARCHAR(50) | Scope type |
| min_purchase_amount_cents | BIGINT | Optional |
| min_quantity | INT | Optional |
| customer_eligibility | VARCHAR(50) | `everyone` \| `new_customers` \| `specific_segment` |
| customer_segment_id | INT FK | Optional |
| usage_limit_total | INT | Optional |
| usage_limit_per_customer | INT | Default 1 |
| current_usage_count | INT DEFAULT 0 | **On DB; add to Go entity** |
| usage_reset_time_type | VARCHAR(20) | **ADD**; default `none` |
| usage_reset_amount | INT | **ADD** |
| can_combine_with_other_discounts | BOOLEAN | Default false |
| starts_at / ends_at | TIMESTAMPTZ | ends_at nullable |
| is_active | BOOLEAN | Lifecycle (not soft-delete) |
| metadata | JSONB | BXGY and extras |

### Validation rules

1. Code required, max 50, unique per seller after trim+uppercase.
2. Percentage value 1–100; fixed_amount > 0; free_shipping value 0.
3. `ends_at` null or > `starts_at`.
4. Code immutable after create.
5. Hard delete only if no `discount_code_usage` rows; else deactivate via `is_active`.

### State

```
[created is_active=true|false]
    ├── PATCH status → active / inactive
    ├── DELETE (no usage) → removed
    └── DELETE (has usage) → rejected
```

---

## Entity: DiscountCode scopes

| Entity / table | Key fields | Notes |
| --- | --- | --- |
| `DiscountCodeProduct` / `discount_code_product` | discount_code_id, **product_id** (add to entity), variant_id? | Unique (code, product, variant) |
| `DiscountCodeCategory` / `discount_code_category` | discount_code_id, category_id | |
| `DiscountCodeCollection` / `discount_code_collection` | discount_code_id, collection_id | |

Add `updated_at` on scope tables if missing (pattern from migration `011`).

Scope mutations allowed only when `applies_to` matches the resource type; `all_products` rejects scope writes.

---

## Entity: DiscountCodeUsage

**Table**: `discount_code_usage` (exists)

| Column | Purpose |
| --- | --- |
| discount_code_id, user_id, order_id | Redemption identity |
| discount_amount_cents, original_amount_cents | Snapshot amounts |
| used_at | For reset-window queries |
| metadata | Optional |

**Indexes to ensure**: `(discount_code_id, user_id)`, `(user_id)`.

---

## Entity: CartAppliedCoupon

**Go**: exists in `order/entity/cart.go`  
**Table**: **CREATE** (missing)

| Column | Type | Notes |
| --- | --- | --- |
| id, created_at, updated_at | BaseEntity | |
| cart_id | BIGINT NOT NULL FK → cart | Indexed |
| discount_code_id | BIGINT NOT NULL FK → discount_code | Indexed |

**UNIQUE(cart_id, discount_code_id)**  
**No discount amount columns.**

---

## Entity: CartItemPromotion

**Go**: exists in `order/entity/cart.go`  
**Table**: **CREATE** (missing) — in scope for cart discount integration

| Column | Type | Notes |
| --- | --- | --- |
| id, created_at, updated_at | BaseEntity | |
| cart_item_id | BIGINT NOT NULL FK → cart_item | Indexed |
| promotion_id | BIGINT NOT NULL FK → promotion | Indexed |

**UNIQUE(cart_item_id, promotion_id)**  
**No discount amount columns** — effects via `ApplyPromotionsToCart`.

**Write rule (locked)**: On each full cart price build, after promotions are applied, sync rows so stored promotion IDs per cart line equal the applied set (delete stale, insert missing).
---

## Entity: OrderAppliedCoupon

**Table**: `order_applied_coupon` (exists, migration `014`)

Immutable checkout snapshot: order_id, discount_code_id, coupon_code, title, type, value, discount_cents, shipping_discount_cents, is_combinable, metadata.

Written on create-order; not updated afterward.

---

## Relationships

```
Seller ──< DiscountCode ──< DiscountCodeProduct|Category|Collection
                │
                ├──< DiscountCodeUsage >── User, Order
                │
Cart ──< CartAppliedCoupon >── DiscountCode

CartItem ──< CartItemPromotion >── Promotion

Order ──< OrderAppliedCoupon >── DiscountCode (nullable FK; code text snapshotted)
```

---

## Migration

**File**: `migrations/028_align_discount_code_and_cart_coupon.sql` (name may adjust to next free number)

1. ALTER `discount_code` ADD description, max_discount_amount_cents, usage_reset_time_type, usage_reset_amount  
2. CREATE `cart_applied_coupon`  
3. CREATE `cart_item_promotion`  
4. ALTER scope tables ADD updated_at if needed  
5. Indexes: `(seller_id, is_active)` on discount_code; usage + cart junction indexes  

**Entity-only (no SQL)**: `CurrentUsageCount` on DiscountCode; `ProductID` on DiscountCodeProduct.
