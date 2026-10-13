# Pre-Spec: Coupon / Discount Code Full Stack (007)

**Feature**: `007-coupon-discount-codes`  
**Created**: 2026-08-01  
**Status**: Pre-Spec (Draft) — no production code yet  
**Modules**: `promotion/` (domain owner), `order/` (cart + checkout integration)  
**Depends On**: Existing promotion CRUD + `ApplyPromotionsToCart`, cart Get/Add pricing flow, `order_applied_coupon` snapshot table  
**Constitution**: `.specify/memory/constitution.md` (layered architecture, factory-singleton DI, TDD/integration-first, seller isolation, no magic strings, SOLID)

---

## 1. Goal

Deliver a **backend-complete** coupon (discount code) feature so the storefront stays thin:

1. **Seller** creates/manages discount codes and scopes in the promotion module.
2. **Customer** applies/removes coupons on cart; every cart API returns full pricing (promotions + coupons + summary).
3. **Cart integration is mandatory**: persist applied coupon (and cart-item promotion) **references** only; **recalculate discounts at runtime** on every cart read/mutate — same model as promotions today. Existing cart entities are kept as designed.
4. **Checkout** snapshots applied coupons onto the order and records usage atomically.
5. Storefront only calls APIs and renders responses — **no stacking / eligibility / discount math on the client**.

---

## 2. Entity & Schema Audit (source of truth)

Audited against Go entities and SQL migrations on 2026-08-01.

### 2.1 `DiscountCode` — entity exists

**File**: [`promotion/entity/discount_code.go`](../../promotion/entity/discount_code.go)

| Field | In Go entity? | In `005` migration? | Action |
|---|---|---|---|
| `id`, `created_at`, `updated_at` (`BaseEntity`) | Yes | Yes | Keep |
| `seller_id` | Yes | Yes | Keep |
| `code`, `title` | Yes | Yes | Keep |
| `description` | **Yes** | **No** | **ALTER table** — add column |
| `discount_type`, `value` | Yes | Yes | Keep |
| `max_discount_amount_cents` | **Yes** | **No** | **ALTER table** — add column |
| `applies_to` | Yes | Yes | Keep |
| `min_purchase_amount_cents`, `min_quantity` | Yes | Yes | Keep |
| `customer_eligibility`, `customer_segment_id` | Yes | Yes | Keep |
| `usage_limit_total`, `usage_limit_per_customer` | Yes | Yes | Keep |
| `current_usage_count` | **No** | **Yes** | **Add to entity** |
| `usage_reset_time_type` | **Yes** | **No** | **ALTER table** — add column |
| `usage_reset_amount` | **Yes** | **No** | **ALTER table** — add column |
| `can_combine_with_other_discounts` | Yes | Yes | Keep |
| `starts_at`, `ends_at` | Yes | Yes | Keep |
| `is_active` | Yes | Yes | Keep (activation flag — **not** soft-delete) |
| `metadata` | Yes | Yes | Keep |
| `deleted_at` | No | No | **Do NOT add** — no soft delete |

**Delete policy (locked)**: No GORM soft-delete. Lifecycle uses `is_active`.  
- `PATCH .../status` toggles `is_active`.  
- `DELETE` **hard-deletes** the row (CASCADE scopes) **only if** there are zero `discount_code_usage` rows; otherwise return business error and require deactivate instead.

**Enums already on entity**:
- `DiscountType`: `percentage` | `fixed_amount` | `free_shipping` | `buy_x_get_y`
- `ResetTimeType`: `none` | `day` | `week` | `month` | `year`
- Reuses promotion `ScopeType` / `EligibilityType`

---

### 2.2 Scope entities — exist with one drift

**File**: [`promotion/entity/discount_code_scope.go`](../../promotion/entity/discount_code_scope.go)

| Entity / table | Go fields | DB (`005`) | Action |
|---|---|---|---|
| `DiscountCodeProduct` / `discount_code_product` | `discount_code_id`, `variant_id` only | `product_id` **NOT NULL**, `variant_id` optional, unique `(discount_code_id, product_id, variant_id)` | **Add `ProductID` to entity**; optional `updated_at` align like promotion scopes |
| `DiscountCodeCategory` | `discount_code_id`, `category_id` | Matches | Keep; add `updated_at` if missing |
| `DiscountCodeCollection` | `discount_code_id`, `collection_id` | Matches | Keep; add `updated_at` if missing |

---

### 2.3 Usage entity — exists

**File**: [`promotion/entity/usage.go`](../../promotion/entity/usage.go) → `DiscountCodeUsage` / table `discount_code_usage`

Fields: `discount_code_id`, `user_id`, `order_id`, `discount_amount_cents`, `original_amount_cents`, `used_at`, `metadata` + `BaseEntity`.  
Table exists in `005`. Add indexes for `(discount_code_id, user_id)` if missing (query path for per-customer limits).

---

### 2.4 Cart attach entities — **IN SCOPE** (entities exist, tables missing)

**File**: [`order/entity/cart.go`](../../order/entity/cart.go)

Cart ↔ discount integration is **mandatory** for this feature. Both junction entities already exist and are the correct design: they store **references only**; discount amounts are calculated at **runtime** on every Get Cart / Apply / checkout snapshot (same pattern as automatic promotions today).

#### `CartAppliedCoupon` (coupon ↔ cart)

```go
type CartAppliedCoupon struct {
    db.BaseEntity
    CartID         uint // cart_id
    DiscountCodeID uint // discount_code_id
}
```

| Rule | Decision |
|---|---|
| What is stored | `cart_id` + `discount_code_id` only |
| What is **not** stored | Discount cents, titles, percentages — always recomputed |
| Migration | **Create** `cart_applied_coupon` matching entity + `UNIQUE(cart_id, discount_code_id)` + FKs + indexes |
| Soft delete | No |
| Feature status | **In scope** — required for apply/remove/get cart |

#### `CartItemPromotion` (promotion ↔ cart item)

```go
type CartItemPromotion struct {
    db.BaseEntity
    CartItemID  uint // cart_item_id
    PromotionID uint // promotion_id
}
```

| Rule | Decision |
|---|---|
| What is stored | `cart_item_id` + `promotion_id` only |
| What is **not** stored | Promotion discount effects — always recomputed via `ApplyPromotionsToCart` |
| Migration | **Create** `cart_item_promotion` matching entity + `UNIQUE(cart_item_id, promotion_id)` + FKs + indexes |
| Soft delete | No |
| Write rule | **Sync on every cart price build**: after `ApplyPromotionsToCart`, replace each line’s stored promotion IDs with the applied set (delete stale, insert missing) |
| Feature status | **In scope** — mandatory cart discount integration (same runtime model as coupons) |

**Runtime pricing model (locked)**  
Cart tables never hold money. On every full cart build:

1. Load items + applied coupon IDs (+ cart-item promotion IDs when persisted).
2. Fetch current variant prices.
3. Call `ApplyPromotionsToCart` → call `ApplyCouponsToCart`.
4. Return computed `CartResponse` (promotions, coupons, summary).

This matches how promotions already work in [`order/service/cart_service.go`](../../order/service/cart_service.go) / [`order/factory/cart_response_builder.go`](../../order/factory/cart_response_builder.go). Existing entities stay as-is; we only add missing tables.

---

### 2.5 `OrderAppliedCoupon` — entity + table exist

**File**: [`order/entity/order_applied_coupon.go`](../../order/entity/order_applied_coupon.go)  
**Migration**: [`migrations/014_create_order_discount_snapshot_tables.sql`](../../migrations/014_create_order_discount_snapshot_tables.sql)

Immutable checkout snapshot (amounts **are** frozen here for audit/reporting — unlike cart). Preload already wired on order repo. Persist path not implemented yet.

---

### 2.6 Migration summary (implementation later)

New migration (e.g. `028_align_discount_code_and_cart_coupon.sql`):

1. `ALTER discount_code` ADD: `description`, `max_discount_amount_cents`, `usage_reset_time_type` (default `'none'`), `usage_reset_amount`.
2. `CREATE TABLE cart_applied_coupon` matching `CartAppliedCoupon` entity (`cart_id`, `discount_code_id`, timestamps, unique pair, FKs).
3. `CREATE TABLE cart_item_promotion` matching `CartItemPromotion` entity (`cart_item_id`, `promotion_id`, timestamps, unique pair, FKs).
4. `ALTER` discount-code scope tables ADD `updated_at` if missing (same pattern as `011` for promotion scopes).
5. Indexes: `discount_code (seller_id, is_active)`, `discount_code_usage (discount_code_id, user_id)`, `cart_applied_coupon (cart_id)`, `cart_item_promotion (cart_item_id)`.

Entity-only fixes (no migration): add `CurrentUsageCount` to `DiscountCode`; add `ProductID` to `DiscountCodeProduct`.  
**Do not** add discount-amount columns to cart junction entities — runtime calc stays.
---

## 3. Architecture & SOLID

### 3.1 Layering (constitution)

```
Handler → Service → Repository → DB
```

- Handlers: bind/validate HTTP, auth context, map AppErrors — **no business rules**.
- Services: validation, stacking, eligibility, orchestration, transactions.
- Repositories: GORM only — **no business rules**; always accept `sellerID` for seller-scoped queries.
- Models: request/response DTOs with embedding; Entities: DB only.

### 3.2 Module boundary

| Concern | Owner module | Cross-module |
|---|---|---|
| Discount code CRUD + scopes | `promotion` | — |
| Validate + calculate coupons | `promotion` (`DiscountCodeService`) | Order calls **service interface** only |
| Cart attach (`cart_applied_coupon`, `cart_item_promotion`) | `order` | References only; amounts computed at runtime |
| Cart HTTP coupon routes + Get Cart enrichment | `order` | **In scope / mandatory** |
| Order snapshot + call usage record | `order` | Calls `DiscountCodeService.RecordUsages` |
| Usage rows + atomic increment | `promotion` | — |

**Forbidden**: order repo reading `discount_code` table directly; promotion repo writing `cart` / `order` tables.

### 3.3 SOLID mapping

| Principle | Application |
|---|---|
| **SRP** | Separate `DiscountCodeService` (CRUD), scope services, `CouponApplyService` / methods for cart calc, usage recording |
| **OCP** | Discount calculators by `DiscountType` (strategy map) — add types without editing cart flow |
| **LSP** | Strategy implementations share one calculator interface |
| **ISP** | Split interfaces if > ~10 methods: `DiscountCodeCommandService`, `DiscountCodeQueryService`, `CouponCartService` as needed |
| **DIP** | Order injects `promotion/service.DiscountCodeService` via factories — never concrete repo |

### 3.4 DI pattern

Extend existing factories:

- `promotion/factory/singleton/{repository,service,handler}_factory.go`
- `order/factory/singleton/service_factory.go` already injects `PromotionService` — add `DiscountCodeService` the same way
- Register `NewDiscountCodeModule()` (+ scope module) in `promotion/container.go`

### 3.5 Pricing pipeline (backend-owned, runtime calculation)

Cart persistence stores **IDs only**. Money is always computed when building the response (same as current promotion flow).

```
1. Load cart items + variant prices (current catalog prices)
2. Load cart_item_promotion IDs (if any) + discover/apply promotions via ApplyPromotionsToCart
3. Load cart_applied_coupon IDs
4. ApplyCouponsToCart (new) on after-promotion totals
5. Self-heal: drop invalid cart_applied_coupon / cart_item_promotion rows when rules no longer pass
6. Build CartResponse (promos + coupons + availableCoupons + summary) — no stored discount amounts read from cart tables
```

`totalDiscount = promotionDiscount + couponDiscount`  
`afterDiscount / total` use combined discounts (+ shipping/tax as today).

**Do not** denormalize discount amounts onto `CartAppliedCoupon` or `CartItemPromotion`. Immutable amounts belong only on order snapshot tables at checkout.
### 3.6 Thin storefront contract

- All mutate coupon APIs return **full `CartResponse`** (same as Get Cart).
- Get Cart includes `appliedCoupons`, `availableCoupons`, coupon summary fields.
- Paths use **real mounts**: `/api/order/cart/...` and `/api/promotion/discount-code/...` (not outdated `/api/cart` in CART_API_PRD).

---

## 4. Constants & Errors (no magic strings)

### 4.1 New files (to create during impl)

- `promotion/error/discount_code_error.go`
- `promotion/utils/constant/discount_code_constants.go`
- Order messages in `order/utils/constant/` for cart coupon handlers

### 4.2 Error codes

| Code | HTTP | When |
|---|---|---|
| `DISCOUNT_CODE_NOT_FOUND` | 404 | Seller get/update unknown / wrong seller |
| `DISCOUNT_CODE_EXISTS` | 409 | Duplicate `(seller_id, code)` |
| `UNAUTHORIZED_DISCOUNT_CODE_ACCESS` | 403 | Seller isolation |
| `INVALID_DISCOUNT_CODE_VALUE` | 400 | Bad percentage/fixed/etc. |
| `INVALID_DISCOUNT_CODE_DATE_RANGE` | 400 | endsAt ≤ startsAt |
| `DISCOUNT_CODE_HAS_USAGE` | 409 | Hard delete blocked |
| `INVALID_COUPON` | 400 | Unknown code for seller / inactive |
| `COUPON_EXPIRED` | 400 | past endsAt |
| `COUPON_NOT_STARTED` | 400 | before startsAt |
| `COUPON_USAGE_LIMIT_REACHED` | 400 | global limit |
| `COUPON_ALREADY_USED` | 400 | per-customer / reset window |
| `COUPON_MIN_PURCHASE_NOT_MET` | 400 | min purchase |
| `COUPON_MIN_QUANTITY_NOT_MET` | 400 | min qty |
| `COUPON_NOT_ELIGIBLE` | 400 | eligibility / segment stub |
| `COUPON_CANNOT_COMBINE` | 400 | stacking rules |
| `COUPON_ALREADY_APPLIED` | 400 | duplicate on cart |
| `COUPON_NOT_APPLICABLE` | 400 | scope mismatch / zero eligible |
| `COUPON_NOT_ON_CART` | 404 | remove unknown code |
| `GUEST_COUPON_NOT_ALLOWED` | 403 | guest cart apply |

### 4.3 Success / failure message constants

Examples: `DISCOUNT_CODE_CREATED_MSG`, `COUPON_APPLIED_MSG`, `FAILED_TO_APPLY_COUPON_MSG`, field keys `DISCOUNT_CODE_FIELD`, `COUPONS_FIELD`, etc.

### 4.4 API path constants

Add under `common/constants` or promotion constants:

- Base: `/api/promotion/discount-code`
- Cart coupons: `/api/order/cart/coupon`, `/api/order/cart/available-coupon`

---

## 5. Request / Response Contracts

### 5.1 Shared embedding (new models)

```go
// promotion/model/discount_code_base_model.go
type BaseDiscountCodeScopeRequest struct {
    DiscountCodeID uint `json:"discountCodeId" binding:"required"`
}

type GetDiscountCodeScopeRequest struct {
    BaseDiscountCodeScopeRequest
    common.BaseListParams
}

type BaseDiscountCodeScopeResponse struct {
    DiscountCodeID uint `json:"discountCodeId"`
}
```

Reuse `common.PaginationResponse` alias like promotions.

---

### 5.2 Seller — Create

`POST /api/promotion/discount-code`  
Auth: `SellerAuth`

**Request** `CreateDiscountCodeRequest`:

```go
type CreateDiscountCodeRequest struct {
    Code                         string                 `json:"code" binding:"required,min=1,max=50"`
    Title                        *string                `json:"title" binding:"omitempty,max=255"`
    Description                  *string                `json:"description" binding:"omitempty"`
    DiscountType                 entity.DiscountType    `json:"discountType" binding:"required,oneof=percentage fixed_amount free_shipping buy_x_get_y"`
    Value                        int64                  `json:"value" binding:"required"`
    MaxDiscountAmountCents       *int64                 `json:"maxDiscountAmountCents" binding:"omitempty,min=0"`
    AppliesTo                    entity.ScopeType       `json:"appliesTo" binding:"required,oneof=all_products specific_products specific_categories specific_collections specific_variant"`
    MinPurchaseAmountCents       *int64                 `json:"minPurchaseAmountCents" binding:"omitempty,min=0"`
    MinQuantity                  *int                   `json:"minQuantity" binding:"omitempty,min=1"`
    CustomerEligibility          entity.EligibilityType `json:"customerEligibility" binding:"omitempty,oneof=everyone new_customers specific_segment"`
    CustomerSegmentID            *uint                  `json:"customerSegmentId" binding:"omitempty"`
    UsageLimitTotal              *int                   `json:"usageLimitTotal" binding:"omitempty,min=1"`
    UsageLimitPerCustomer        *int                   `json:"usageLimitPerCustomer" binding:"omitempty,min=1"`
    UsageResetTimeType           entity.ResetTimeType   `json:"usageResetTimeType" binding:"omitempty,oneof=none day week month year"`
    UsageResetAmount             *int                   `json:"usageResetAmount" binding:"omitempty,min=1"`
    CanCombineWithOtherDiscounts *bool                  `json:"canCombineWithOtherDiscounts"`
    StartsAt                     string                 `json:"startsAt" binding:"required"`
    EndsAt                       *string                `json:"endsAt" binding:"omitempty"`
    IsActive                     *bool                  `json:"isActive"`
    Metadata                     map[string]any         `json:"metadata"`
}
```

**Logic**:
- Normalize `code` → trim + uppercase; unique per seller.
- Validate value by type (percentage 1–100; fixed > 0; free_shipping value 0; buy_x_get_y via metadata schema).
- If `specific_segment` → require `customerSegmentID` (segment engine may still stub at apply time).
- Default `UsageResetTimeType = none`, `IsActive = true`, `CanCombine = false`.

**Response** `201`: `{ discountCode: DiscountCodeResponse }`

---

### 5.3 Seller — Update / Status / Delete / Get / List

| Method | Path | Body / query | Logic |
|---|---|---|---|
| `GET` | `/api/promotion/discount-code/:id` | — | Seller-scoped find |
| `GET` | `/api/promotion/discount-code` | `isActive`, `discountType`, `appliesTo`, `page`, `pageSize` | Paginated list |
| `PUT` | `/api/promotion/discount-code/:id` | `UpdateDiscountCodeRequest` (all pointer optional fields) | Partial update; re-validate value/dates; **do not** change `code` if usage > 0 (or forbid code change always — **locked: code immutable after create**) |
| `PATCH` | `/api/promotion/discount-code/:id/status` | `{ "isActive": true\|false }` | Toggle only |
| `DELETE` | `/api/promotion/discount-code/:id` | — | Hard delete if no usages; else `DISCOUNT_CODE_HAS_USAGE` |

**Response** `DiscountCodeResponse`:

```go
type DiscountCodeResponse struct {
    ID                           uint                   `json:"id"`
    SellerID                     uint                   `json:"sellerId"`
    Code                         string                 `json:"code"`
    Title                        *string                `json:"title,omitempty"`
    Description                  *string                `json:"description,omitempty"`
    DiscountType                 entity.DiscountType    `json:"discountType"`
    Value                        int64                  `json:"value"`
    MaxDiscountAmountCents       *int64                 `json:"maxDiscountAmountCents,omitempty"`
    AppliesTo                    entity.ScopeType       `json:"appliesTo"`
    MinPurchaseAmountCents       *int64                 `json:"minPurchaseAmountCents,omitempty"`
    MinQuantity                  *int                   `json:"minQuantity,omitempty"`
    CustomerEligibility          entity.EligibilityType `json:"customerEligibility"`
    CustomerSegmentID            *uint                  `json:"customerSegmentId,omitempty"`
    UsageLimitTotal              *int                   `json:"usageLimitTotal,omitempty"`
    UsageLimitPerCustomer        *int                   `json:"usageLimitPerCustomer,omitempty"`
    CurrentUsageCount            int                    `json:"currentUsageCount"`
    UsageResetTimeType           entity.ResetTimeType   `json:"usageResetTimeType"`
    UsageResetAmount             *int                   `json:"usageResetAmount,omitempty"`
    CanCombineWithOtherDiscounts *bool                  `json:"canCombineWithOtherDiscounts"`
    StartsAt                     string                 `json:"startsAt"`
    EndsAt                       *string                `json:"endsAt,omitempty"`
    IsActive                     bool                   `json:"isActive"`
    Metadata                     map[string]any         `json:"metadata,omitempty"`
    CreatedAt                    string                 `json:"createdAt"`
    UpdatedAt                    string                 `json:"updatedAt"`
}

type ListDiscountCodesResponse struct {
    DiscountCodes []DiscountCodeResponse   `json:"discountCodes"`
    Pagination    common.PaginationResponse `json:"pagination"`
}
```

---

### 5.4 Seller — Scopes (mirror promotion scopes)

Base: `/api/promotion/discount-code/scope`  
Auth: `SellerAuth`

| Resource | Routes |
|---|---|
| Product | `POST /product`, `DELETE /product`, `DELETE /:discountCodeId/product`, `GET /:discountCodeId/product` |
| Variant | same with `/variant` |
| Category | same with `/category` |
| Collection | same with `/collection` |

**Contracts** (embed base scope request/response):

```go
type AddDiscountCodeProductRequest struct {
    BaseDiscountCodeScopeRequest
    ProductIDs []uint `json:"productIds" binding:"required,min=1"`
}
// Remove / Get analogous to promotion product scope models
```

**Logic**:
- Verify discount code belongs to seller.
- `appliesTo` must match resource (`specific_products` / `specific_variant` / categories / collections); `all_products` rejects scope mutations.
- Product scope rows always set `product_id`; optional `variant_id` when narrowing.

---

### 5.5 Customer — Cart coupon APIs

Base: `/api/order/cart`  
Auth: `CustomerAuth` (authenticated only — **no guest coupons**)

#### Apply — `POST /api/order/cart/coupon`

```go
type ApplyCouponRequest struct {
    Code string `json:"code" binding:"required,min=1,max=50"`
}
```

**Logic**:
1. Resolve user cart; reject guest.
2. Normalize code; find active code for seller.
3. Run validation pipeline (dates, limits, eligibility, min purchase/qty, scope, combinability vs existing cart coupons + whether applied promotions allow coupons).
4. Insert `cart_applied_coupon` (unique constraint → already applied).
5. Rebuild full cart (promos → coupons → available).
6. Return **full `CartResponse`**.

#### Remove one — `DELETE /api/order/cart/coupon/:code`

Remove matching applied row; return full `CartResponse`.

#### Remove all — `DELETE /api/order/cart/coupon`

Clear all applied rows; return full `CartResponse`.

#### Available — `GET /api/order/cart/available-coupon`

```go
type AvailableCouponsResponse struct {
    Applicable    []AvailableCouponInfo    `json:"applicable"`
    NotApplicable []UnavailableCouponInfo  `json:"notApplicable"`
}

type AvailableCouponInfo struct {
    ID                           uint   `json:"id"`
    Code                         string `json:"code"`
    Title                        string `json:"title"`
    DiscountType                 string `json:"discountType"`
    Value                        int64  `json:"value"`
    MaxDiscountAmountCents       *int64 `json:"maxDiscountAmountCents,omitempty"`
    PotentialDiscount            int64  `json:"potentialDiscount"`
    PotentialDiscountFormatted   string `json:"potentialDiscountFormatted"`
    MinPurchaseAmountCents       *int64 `json:"minPurchaseAmountCents,omitempty"`
    CanCombineWithOtherDiscounts bool   `json:"canCombineWithOtherDiscounts"`
    StartsAt                     string `json:"startsAt"`
    EndsAt                       *string `json:"endsAt,omitempty"`
}

type UnavailableCouponInfo struct {
    ID     uint   `json:"id"`
    Code   string `json:"code"`
    Title  string `json:"title"`
    Reason string `json:"reason"` // constant message
}
```

#### Get Cart enrichment

Extend existing `CartResponse`:

```go
// Already exists:
AppliedCoupons []AppliedCouponInfo
Summary.CouponCount / CouponDiscount / CouponDiscountFormatted
// ADD:
AvailableCoupons *AvailableCouponsResponse `json:"availableCoupons,omitempty"`
```

Extend `AppliedCouponInfo` if needed:

```go
type AppliedCouponInfo struct {
    ID                uint   `json:"id"`              // cart_applied_coupon.id
    DiscountCodeID    uint   `json:"discountCodeId"`
    Code              string `json:"code"`
    Title             string `json:"title"`
    DiscountType      string `json:"discountType"`
    Discount          int64  `json:"discount"`
    DiscountFormatted string `json:"discountFormatted"`
    ShippingDiscount  int64  `json:"shippingDiscount,omitempty"`
}
```

**Apply/Remove MUST return the same `CartResponse` shape as Get Cart** so storefront never needs a second call.

---

### 5.6 Internal promotion ↔ order contracts (not HTTP)

```go
// Built by order, passed to promotion
type CouponCartRequest struct {
    SellerID               uint
    CustomerID             *uint
    IsFirstOrder           bool
    Items                  []CartItem // after-promo adjusted line totals preferred
    SubtotalCents          int64      // after promotions
    ShippingCents          int64
    AppliedDiscountCodeIDs []uint
    PromotionsAllowCoupons bool      // false if any applied promo CanStackWithCoupons=false
}

type AppliedCouponSummary struct {
    AppliedCoupons     []CouponValidationResult
    SkippedCoupons     []SkippedCouponResult
    TotalDiscountCents int64
    ShippingDiscount   int64
    // Optional: available split for embedding in cart response
}

type CouponUsageRecord struct {
    DiscountCodeID      uint
    DiscountAmountCents int64
    OriginalAmountCents int64
}
```

Service methods:

- `ApplyCouponsToCart(ctx, *CouponCartRequest) (*AppliedCouponSummary, error)`
- `ListAvailableCouponsForCart(ctx, *CouponCartRequest) (*AvailableCouponsResponse, error)` — order formats currency strings
- `RecordCouponUsages(ctx, orderID, userID uint, records []CouponUsageRecord) error`
- `IncrementUsageAtomically(ctx, discountCodeID uint, limit int) (bool, error)`

---

## 6. Validation & Calculation Logic (per API)

### 6.1 Validation pipeline (ordered)

1. Exists for `seller_id` + normalized code  
2. `is_active == true`  
3. `now >= starts_at` else `COUPON_NOT_STARTED`  
4. `ends_at` null or `now <= ends_at` else `COUPON_EXPIRED`  
5. Global usage: `current_usage_count < usage_limit_total` (if set)  
6. Per-customer usage within reset window (`UsageResetTimeType` + `UsageResetAmount`; `none` = lifetime)  
7. Eligibility: everyone / new_customers (`IsFirstOrder`); `specific_segment` → **stub not eligible** until segment phase  
8. Min purchase / min quantity against **eligible scoped items** (after-promo amounts)  
9. Scope match: at least one cart item in scope  
10. Combinability: vs other applied coupons; vs promotions (`PromotionsAllowCoupons`)  
11. Calculate discount; if 0 → `COUPON_NOT_APPLICABLE`

### 6.2 Calculators (strategy)

| Type | Rule |
|---|---|
| `percentage` | `%` of eligible after-promo subtotal; cap `MaxDiscountAmountCents` |
| `fixed_amount` | `min(value, eligible subtotal)` |
| `free_shipping` | shipping discount up to `ShippingCents` |
| `buy_x_get_y` | from `metadata` (same semantics as promo BXGY where feasible) |

### 6.3 Get Cart self-heal

On every full cart build: re-run apply for stored coupon IDs; if invalid → delete `cart_applied_coupon` row and omit from `appliedCoupons` (storefront just shows updated cart).

### 6.4 Create Order

Inside existing `persistOrderSnapshotGraph` transaction:

1. Build `[]OrderAppliedCoupon` from cart snapshot applied coupons.  
2. `CreateOrderAppliedCoupons`.  
3. `RecordCouponUsages` + `IncrementUsageAtomically` per code.  
4. If increment fails (race) → abort transaction → customer sees error; cart coupons remain for retry.

Also record promotion usages in same phase if promotion hooks are ready; minimum for this feature is **coupon** usage.

---

## 7. File / Package Map (implementation checklist)

### Promotion module

```
promotion/
  entity/discount_code.go              # + CurrentUsageCount
  entity/discount_code_scope.go        # + ProductID
  entity/usage.go                      # exists
  model/discount_code_*.go             # NEW contracts
  model/coupon_cart_model.go           # NEW internal cart contracts
  error/discount_code_error.go         # NEW
  utils/constant/discount_code_constants.go
  repository/discount_code_repository.go (+ impl)
  repository/discount_code_*_scope_repository.go
  repository/discount_code_usage_repository.go
  service/discount_code_service.go (+ impl CRUD)
  service/coupon_apply_service.go      # or methods on DiscountCodeService
  service/discount_code_service.go     # seller CRUD only
  service/coupon_apply_service.go      # ApplyCouponsToCart + ListAvailableCouponsForCart
  service/discountStrategy/*.go        # coupon type calculators only
  handler/discount_code_handler.go
  handler/discount_code_*_scope_handler.go
  route/discount_code_routes.go
  route/discount_code_scope_routes.go
  factory/discount_code_mapper.go
  factory/singleton/*                  # wire
  container.go                         # register modules
```

### Order module

```
order/
  entity/cart.go                       # CartAppliedCoupon + CartItemPromotion exist (keep as-is)
  repository/cart_repository.go        # + coupon/promotion attach methods
  repository/order_repository.go       # + CreateOrderAppliedCoupons
  service/cart_service.go              # integrate ApplyCouponsToCart; runtime recompute
  service/cart_operations.go           # summary builder
  service/order_lifecycle.go           # snapshots + usage
  handler/cart_handler.go              # Apply/Remove/Available
  route/cart_route.go                  # register coupon routes
  factory/cart_response_builder.go     # fill coupons + availableCoupons from runtime summary
  model/cart_model.go                  # + AvailableCoupons field
  error/ + utils/constant/             # coupon errors/messages if needed
```

### Migrations

```
migrations/028_align_discount_code_and_cart_coupon.sql
  # ALTER discount_code columns
  # CREATE cart_applied_coupon
  # CREATE cart_item_promotion
  # scope updated_at + indexes
```

### Tests

```
test/integration/promotion/discount_code_*_test.go
test/integration/order/cart_coupon_*_test.go
test/integration/order/order_coupon_checkout_test.go
```

---

## 8. API Catalog (complete)

### Seller (`SellerAuth` + `X-Correlation-ID`)

| # | Method | Path | Purpose |
|---|---|---|---|
| S1 | POST | `/api/promotion/discount-code` | Create |
| S2 | GET | `/api/promotion/discount-code` | List |
| S3 | GET | `/api/promotion/discount-code/:id` | Get |
| S4 | PUT | `/api/promotion/discount-code/:id` | Update |
| S5 | PATCH | `/api/promotion/discount-code/:id/status` | Activate/deactivate |
| S6 | DELETE | `/api/promotion/discount-code/:id` | Hard delete (no usage) |
| S7–S10 | CRUD | `/api/promotion/discount-code/scope/product`… | Product scope |
| S11–S14 | CRUD | `.../scope/variant` | Variant scope |
| S15–S18 | CRUD | `.../scope/category` | Category scope |
| S19–S22 | CRUD | `.../scope/collection` | Collection scope |

### Customer (`CustomerAuth` + `X-Correlation-ID`)

| # | Method | Path | Response |
|---|---|---|---|
| C1 | GET | `/api/order/cart` | Full `CartResponse` (now with coupons) |
| C2 | POST | `/api/order/cart/coupon` | Full `CartResponse` |
| C3 | DELETE | `/api/order/cart/coupon/:code` | Full `CartResponse` |
| C4 | DELETE | `/api/order/cart/coupon` | Full `CartResponse` |
| C5 | GET | `/api/order/cart/available-coupon` | `AvailableCouponsResponse` |
| C6 | POST | `/api/order` (existing create) | Order with `appliedCoupons` snapshot |

---

## 9. Test Coverage Scenarios

Constitution mandatory per endpoint: happy path, auth, authz, validation, edge cases, correlation ID, seller isolation.

### 9.1 Seller CRUD

| ID | Scenario | Expect |
|---|---|---|
| T-S1 | Create percentage code | 201, code uppercased, `currentUsageCount=0` |
| T-S2 | Create duplicate code same seller | 409 `DISCOUNT_CODE_EXISTS` |
| T-S3 | Create same code other seller | 201 (allowed) |
| T-S4 | Invalid percentage value 0/101 | 400 |
| T-S5 | endsAt before startsAt | 400 |
| T-S6 | Get other seller's code | 404/403 per pattern (prefer 404) |
| T-S7 | List filters `isActive` | only matching |
| T-S8 | Update title/value | 200; code unchanged |
| T-S9 | Attempt change code | ignored or 400 (immutable) |
| T-S10 | Deactivate via status | `isActive=false` |
| T-S11 | Delete unused | 200 hard delete |
| T-S12 | Delete after usage exists | 409 `DISCOUNT_CODE_HAS_USAGE` |
| T-S13 | Missing JWT / wrong role / missing correlation | 401/403/400 |
| T-S14 | Pagination bounds | pageSize max 100 |

### 9.2 Seller scopes

| ID | Scenario | Expect |
|---|---|---|
| T-SC1 | Add products when `appliesTo=specific_products` | 200 |
| T-SC2 | Add products when `appliesTo=all_products` | 400 |
| T-SC3 | Remove / remove-all / list | works; pagination |
| T-SC4 | Scope on other seller's code | 404/403 |
| T-SC5 | Variant/category/collection parity | same as product |

### 9.3 Customer cart coupons

| ID | Scenario | Expect |
|---|---|---|
| T-C1 | Apply valid fixed code | 200 full cart; `appliedCoupons` len 1; summary coupon fields set |
| T-C2 | Apply unknown code | 400 `INVALID_COUPON` |
| T-C3 | Apply expired / not started | matching codes |
| T-C4 | Apply twice | 400 `COUPON_ALREADY_APPLIED` |
| T-C5 | Min purchase not met | 400 + reason |
| T-C6 | Scope not matching cart items | 400 `COUPON_NOT_APPLICABLE` |
| T-C7 | Non-combinable + second coupon | 400 `COUPON_CANNOT_COMBINE` |
| T-C8 | Promo with `canStackWithCoupons=false` | reject coupon |
| T-C9 | Remove one / remove all | cart updated |
| T-C10 | Get cart after seller deactivates code | self-heal removes coupon; totals update |
| T-C11 | Available coupons lists applicable + notApplicable | reasons are constants |
| T-C12 | Guest apply | 401/403 `GUEST_COUPON_NOT_ALLOWED` |
| T-C13 | Seller isolation: code of seller A on seller B cart | invalid |
| T-C14 | Per-customer usage after prior order | `COUPON_ALREADY_USED` |
| T-C15 | Global usage exhausted | `COUPON_USAGE_LIMIT_REACHED` |
| T-C16 | Percentage with max cap | discount ≤ cap |
| T-C17 | Free shipping coupon | shipping discount reflected |
| T-C18 | Auth / correlation failures | 401/400 |

### 9.4 Checkout E2E

| ID | Scenario | Expect |
|---|---|---|
| T-E1 | Create order with applied coupon | `order_applied_coupon` row; `discount_code_usage` row; `current_usage_count++` |
| T-E2 | Order response includes applied coupon snapshot | code/title/discount cents |
| T-E3 | Second order exceeds per-customer limit | blocked at apply or at checkout race |
| T-E4 | Concurrent checkout hitting global limit | one succeeds, one fails; count correct |
| T-E5 | Cart converted; coupons not re-usable from converted cart | status converted |

### 9.5 Integration test style

- `testify/suite` + Testcontainers (existing setup).
- API-first: assert via HTTP Get Cart / Get Order, not raw DB except for usage counters when needed.
- Endpoint path constants in test files (no magic strings).
- Document each test method with scenario comments (constitution).

---

## 10. Security, Scalability, Ops

| Topic | Decision |
|---|---|
| Tenant isolation | Every discount query includes `seller_id` |
| Enumeration | Cross-seller get returns 404 |
| Code brute-force | Optional soft rate-limit on apply (Phase harden); always normalize uppercase |
| Concurrency | Atomic `UPDATE ... WHERE current_usage_count < limit` |
| Stateless services | Coupon state in Postgres only |
| Caching | Optional Redis cache of active codes per seller (invalidate on CRUD) — later harden |
| Guest | No coupons until login/merge |
| Segments | Stub `specific_segment` as not eligible; document deferred |
| Soft delete | **Never** — `is_active` + hard delete without usage |

---

## 11. Phased Implementation Order

| Phase | Deliverable | Exit criteria |
|---|---|---|
| **0** | Migration (discount_code columns + `cart_applied_coupon` + `cart_item_promotion`) + entity fixes + error/constant/model stubs + test suite scaffolds | Schema matches entity audit; cart junction tables exist |
| **1** | Seller CRUD APIs + tests T-S* | Green integration |
| **2** | Scope APIs + tests T-SC* | Green |
| **3** | `ApplyCouponsToCart` / available / usage service (unit + service tests) | Calculator correct |
| **4** | **Cart integration (mandatory)**: persist/load `cart_applied_coupon`; **sync `cart_item_promotion` on each price build**; C2–C5; Get Cart enrichment; runtime recompute; T-C* | Full `CartResponse` for storefront |
| **5** | Order snapshot + usage + T-E* | Checkout end-to-end green |
| **6** | Harden: races, rate-limit, docs path fix in CART_API_PRD, mark deferred coupon done | Production-ready |

**TDD**: write failing integration tests for the phase first, then implement to green.

---

## 12. Explicit Non-Goals

- Storefront UI / client-side discount math  
- Soft-delete columns on cart or discount-code tables  
- Storing discount **amounts** on cart junction tables (runtime calc only — entities stay reference-only)  
- Full customer-segment rule engine (stub only)  
- Guest-cart coupon apply  
- Changing auto-promotion calculator math beyond respecting `canStackWithCoupons` and cart-item promotion attach  
- Creating a second cart base path (`/api/cart`) — use `/api/order/cart`

**In scope (do not defer):** full cart ↔ coupon integration (`cart_applied_coupon`), cart ↔ promotion attach table (`cart_item_promotion`), Get Cart / apply / remove returning runtime-priced `CartResponse`.

---

## 13. Open Follow-ups (do not block Phases 0–5)

1. Optional Redis cache of active discount codes per seller.  
2. Optional apply rate-limit (`T069`).  

**Resolved**: BXGY metadata keys; `cart_item_promotion` sync-on-build; `CouponApplyService` + `discountStrategy/` paths; `promotion_usage` out of scope for this feature.

---

## 14. Acceptance Checklist

- [ ] Entity/DB aligned (description, max cap, reset fields on DB; `current_usage_count` + `product_id` on entities)  
- [ ] `cart_applied_coupon` table exists matching entity (reference-only, no soft delete, no stored amounts)  
- [ ] `cart_item_promotion` table exists matching entity (reference-only, runtime promotion effects)  
- [ ] Cart coupon apply/remove/get fully integrated — discounts recomputed at runtime like promotions  
- [ ] Seller CRUD + scopes complete with constants/AppErrors  
- [ ] Customer apply/remove/available return storefront-ready full `CartResponse` payloads  
- [ ] Get Cart returns promotions + coupons + available + correct summary  
- [ ] Order create writes `order_applied_coupon` + `discount_code_usage` + increments count  
- [ ] Integration tests cover T-S*, T-SC*, T-C*, T-E*  
- [ ] Order module only talks to promotion via service interfaces  
