# API Contracts: Coupon / Discount Codes

**Feature**: 007-coupon-discount-codes  
**Date**: 2026-08-01  
**Auth headers (all)**: `Authorization: Bearer …`, `X-Correlation-ID` (required)

Paths use live bases: `/api/promotion`, `/api/order` (cart mounted at `/api/order/cart`).

Detailed Go struct sketches also live in [pre-spec.md](../pre-spec.md) §5.

---

## A. Seller — Discount Code CRUD

**Middleware**: `SellerAuth`

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/api/promotion/discount-code` | Create |
| GET | `/api/promotion/discount-code` | List (`isActive`, `discountType`, `appliesTo`, page, pageSize) |
| GET | `/api/promotion/discount-code/:id` | Get one |
| PUT | `/api/promotion/discount-code/:id` | Partial update (code immutable) |
| PATCH | `/api/promotion/discount-code/:id/status` | `{ "isActive": bool }` |
| DELETE | `/api/promotion/discount-code/:id` | Hard delete if unused |

### Create request (JSON)

| Field | Required | Notes |
| --- | --- | --- |
| code | yes | Normalized trim+uppercase; unique per seller |
| discountType | yes | percentage \| fixed_amount \| free_shipping \| buy_x_get_y |
| value | yes | Type-specific validation |
| appliesTo | yes | all_products \| specific_products \| specific_categories \| specific_collections \| specific_variant |
| startsAt | yes | RFC3339 |
| title, description, maxDiscountAmountCents, mins, eligibility, limits, reset, canCombine, endsAt, isActive, metadata | no | |

### Response envelope

Standard `{ success, message, data: { discountCode: DiscountCodeResponse } }` (list uses `discountCodes` + `pagination`).

### Seller errors

| Code | HTTP | When |
| --- | --- | --- |
| DISCOUNT_CODE_NOT_FOUND | 404 | Missing / wrong seller |
| DISCOUNT_CODE_EXISTS | 409 | Duplicate code |
| INVALID_DISCOUNT_CODE_VALUE | 400 | Bad value for type |
| INVALID_DISCOUNT_CODE_DATE_RANGE | 400 | ends ≤ starts |
| DISCOUNT_CODE_HAS_USAGE | 409 | Delete blocked |
| UNAUTHORIZED / VALIDATION / missing correlation | 401/400 | Standard |

---

## B. Seller — Scopes

**Base**: `/api/promotion/discount-code/scope`  
**Middleware**: `SellerAuth`

Per resource (`product`, `variant`, `category`, `collection`):

| Method | Path pattern |
| --- | --- |
| POST | `/scope/{resource}` — add IDs |
| DELETE | `/scope/{resource}` — remove IDs |
| DELETE | `/scope/:discountCodeId/{resource}` — remove all |
| GET | `/scope/:discountCodeId/{resource}` — list (paginated) |

**Rules**: Code must belong to seller; `appliesTo` must match resource; `all_products` rejects mutations.

---

## C. Customer — Cart coupons

**Base**: `/api/order/cart`  
**Middleware**: `CustomerAuth` (no guest)

| Method | Path | Response |
| --- | --- | --- |
| GET | `` (existing) | Full `CartResponse` **with** coupons + availableCoupons |
| POST | `/coupon` | Body `{ "code" }` → full `CartResponse` |
| DELETE | `/coupon/:code` | Full `CartResponse` |
| DELETE | `/coupon` | Remove all → full `CartResponse` |
| GET | `/available-coupon` | `{ applicable[], notApplicable[] }` |

### CartResponse coupon fields (contract)

```json
{
  "appliedCoupons": [
    {
      "id": 1,
      "discountCodeId": 50,
      "code": "SAVE20",
      "title": "20% Off",
      "discountType": "percentage",
      "discount": 50000,
      "discountFormatted": "₹500.00",
        "shippingDiscount": 0,
      "shippingDiscountFormatted": "₹0.00"
    }
  ],
  "availableCoupons": {
    "applicable": [ { "id": 60, "code": "…", "potentialDiscount": 1000, "potentialDiscountFormatted": "…" } ],
    "notApplicable": [ { "id": 70, "code": "…", "reason": "…" } ]
  },
  "summary": {
    "promotionDiscount": 0,
    "couponCount": 1,
    "couponDiscount": 50000,
    "couponDiscountFormatted": "₹500.00",
    "totalDiscount": 50000,
    "afterDiscount": 0,
    "total": 0
  }
}
```

(Other existing cart fields unchanged.)

### Customer coupon errors

| Code | HTTP |
| --- | --- |
| INVALID_COUPON | 400 |
| COUPON_EXPIRED | 400 |
| COUPON_NOT_STARTED | 400 |
| COUPON_USAGE_LIMIT_REACHED | 400 |
| COUPON_ALREADY_USED | 400 |
| COUPON_MIN_PURCHASE_NOT_MET | 400 |
| COUPON_MIN_QUANTITY_NOT_MET | 400 |
| COUPON_NOT_ELIGIBLE | 400 |
| COUPON_CANNOT_COMBINE | 400 |
| COUPON_ALREADY_APPLIED | 400 |
| COUPON_NOT_APPLICABLE | 400 |
| COUPON_NOT_ON_CART | 404 |
| GUEST_COUPON_NOT_ALLOWED | 403 |

### Buy-X-Get-Y discount code metadata (locked)

Stored on `discount_code.metadata` (JSONB). Required when `discountType = buy_x_get_y`:

| Key | Type | Rule |
| --- | --- | --- |
| `buyQuantity` | int | ≥ 1 |
| `getQuantity` | int | ≥ 1 |
| `getDiscountPercent` | number | 0–100 inclusive; `100` means free get items |

Create/Update MUST reject missing/invalid keys with `INVALID_DISCOUNT_CODE_VALUE` (or validation error). Calculator applies to eligible scoped lines after promotions.

---

### AppliedCouponInfo (required fields)

| Field | Notes |
| --- | --- |
| id | `cart_applied_coupon.id` |
| discountCodeID | |
| code, title, discountType | |
| discount, discountFormatted | Merchandise discount cents |
| shippingDiscount, shippingDiscountFormatted | Required; set for `free_shipping` (0 otherwise) |

---

## D. Internal service contracts (order → promotion)

Not HTTP. Order builds `CouponCartRequest` (seller, customer, items after promo, shipping, applied code IDs, promotionsAllowCoupons) and calls:

- `ApplyCouponsToCart` / `ListAvailableCouponsForCart` via **`CouponApplyService`** (`promotion/service/coupon_apply_service.go`) → `AppliedCouponSummary` / available list (order formats currency)
- Calculators under **`promotion/service/discountStrategy/`** only
- Seller CRUD remains on **`DiscountCodeService`**
- `RecordCouponUsages` + `IncrementUsageAtomically` (promotion usage repos/services) inside create-order transaction

---

## E. Checkout behavior contract

**Existing** create-order flow MUST additionally:

1. Persist `order_applied_coupon` rows from cart snapshot  
2. Insert `discount_code_usage` and atomically increment `current_usage_count`  
3. Fail the whole order transaction if usage increment loses the race  

Order GET continues to preload `AppliedCoupons`.

---

## F. Storefront integration guarantees

1. Apply / Remove / Get Cart return enough data to render totals without client-side discount math.  
2. Invalid applied coupons are self-healed on Get Cart.  
3. Promotions applied before coupons in total calculation.  
4. No second cart base path (`/api/cart`); use `/api/order/cart`.
