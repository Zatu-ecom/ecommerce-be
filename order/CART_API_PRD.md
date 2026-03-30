# 🛒 Cart API - Product Requirements Document

> **Last Updated**: January 28, 2026  
> **Module**: Order Service  
> **Status**: Design Phase

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Data Models](#data-models)
3. [Cart APIs](#cart-apis)
4. [Coupon APIs](#coupon-apis)
5. [Pricing Calculation](#pricing-calculation)
6. [Validation Rules](#validation-rules)
7. [Error Codes](#error-codes)
8. [Future Enhancements](#future-enhancements)

---

## 📖 Overview

### Purpose

The Cart API provides functionality for authenticated users to manage their shopping cart, apply discount codes (coupons), and view real-time pricing with all applicable promotions.

### Key Principles

| Principle                | Description                                                        |
| ------------------------ | ------------------------------------------------------------------ |
| **One Cart Per User**    | Each user has exactly one active cart (user belongs to one seller) |
| **No SellerID on Cart**  | Seller is derived from `user.seller_id`                            |
| **Runtime Pricing**      | All prices calculated at runtime (not stored in cart)              |
| **Multiple Discounts**   | Multiple promotions and coupons can stack based on rules           |
| **Real-time Validation** | Coupons validated on every cart fetch                              |

### Authentication

All cart endpoints require **Customer Authentication**:

- `Authorization: Bearer <token>` (JWT)
- `X-Correlation-ID: <uuid>` (Required for all requests)

---

## 🏗️ Data Models

### Entities

```
┌─────────────────────────────────────────────────────────────┐
│                          CART                                │
│  - id                                                       │
│  - user_id (unique - one cart per user)                    │
│  - metadata (JSONB)                                        │
├─────────────────────────────────────────────────────────────┤
│                       CART_ITEM                             │
│  - id                                                       │
│  - cart_id (FK)                                            │
│  - variant_id (FK to product_variant)                      │
│  - quantity                                                │
├─────────────────────────────────────────────────────────────┤
│                  CART_APPLIED_COUPON                        │
│  - id                                                       │
│  - cart_id (FK)                                            │
│  - discount_code_id (FK)                                   │
│  - applied_at                                              │
└─────────────────────────────────────────────────────────────┘
```

### Notes

- **No prices stored in cart** - Calculated at runtime from `product_variant.price`
- **No promotion links in cart** - Promotions auto-applied at runtime
- **Coupons stored by reference** - Only `discount_code_id`, discount calculated at runtime

### Response Models

Two main response models are defined in `model/cart_model.go`:

| Model               | Used In                            | Description                                  |
| ------------------- | ---------------------------------- | -------------------------------------------- |
| `CartBasicResponse` | Add Item, Update Item, Remove Item | Cart + Items (no pricing/promotions/coupons) |
| `CartResponse`      | Get Cart                           | Full cart with pricing, promotions, summary  |

**CartBasicResponse Structure:**

```go
type CartBasicResponse struct {
    ID       uint                   `json:"id"`
    UserID   uint                   `json:"userId"`
    Currency CurrencyInfo           `json:"currency"`
    Items    []CartItemResponse     `json:"items"`     // Items without pricing
    Metadata map[string]interface{} `json:"metadata"`
}
```

**CartResponse Structure:**

```go
type CartResponse struct {
    ID                  uint                          `json:"id"`
    UserID              uint                          `json:"userId"`
    Currency            CurrencyInfo                  `json:"currency"`
    Items               []CartItemWithPricingResponse `json:"items"`           // Items with pricing
    AppliedCoupons      []AppliedCouponInfo           `json:"appliedCoupons"`
    Summary             CartSummary                   `json:"summary"`
    AvailablePromotions []AvailablePromotionInfo      `json:"availablePromotions,omitempty"`
    Metadata            map[string]interface{}        `json:"metadata"`
}
```

---

## 🛒 Cart APIs

### Base URL

```
/api/cart
```

---

### 1. Get Cart

**Retrieves the current user's cart with all items, applied promotions, coupons, and calculated totals.**

```
GET /api/cart
```

#### Headers

| Header             | Required | Description         |
| ------------------ | -------- | ------------------- |
| `Authorization`    | ✅       | Bearer token        |
| `X-Correlation-ID` | ✅       | Request tracking ID |

#### Response (200 OK)

```json
{
  "success": true,
  "data": {
    "id": 1,
    "userId": 3,
    "currency": {
      "code": "INR",
      "symbol": "₹",
      "decimalDigits": 2
    },
    "items": [
      {
        "id": 1,
        "variantId": 5,
        "quantity": 3,
        "variant": {
          "id": 5,
          "sku": "IPH15PRO-BLK-256",
          "images": ["https://..."],
          "allowPurchase": true,
          "product": {
            "id": 1,
            "name": "iPhone 15 Pro",
            "slug": "iphone-15-pro"
          },
          "options": [
            { "name": "Color", "value": "Black" },
            { "name": "Storage", "value": "256GB" }
          ]
        },
        "unitPrice": 9999900,
        "lineTotal": 29999700,
        "appliedPromotions": [
          {
            "promotionId": 101,
            "name": "Flash Sale - 15% Off",
            "type": "flash_sale",
            "discount": 4499955,
            "discountFormatted": "₹44,999.55",
            "badgeText": "FLASH SALE",
            "badgeColor": "#FF0000"
          }
        ],
        "totalPromotionDiscount": 4499955,
        "discountedLineTotal": 25499745,
        "pointsEquivalent": 2550
      }
    ],
    "appliedCoupons": [
      {
        "id": 1,
        "discountCodeId": 50,
        "code": "SAVE500",
        "title": "₹500 Off",
        "discountType": "fixed_amount",
        "discount": 50000,
        "discountFormatted": "₹500.00",
        "appliedAt": "2026-01-27T10:30:00Z"
      }
    ],
    "summary": {
      "itemCount": 3,
      "uniqueItems": 1,

      "subtotal": 29999700,
      "subtotalFormatted": "₹2,99,997.00",

      "promotionCount": 1,
      "promotionDiscount": 4499955,
      "promotionDiscountFormatted": "₹44,999.55",

      "couponCount": 1,
      "couponDiscount": 50000,
      "couponDiscountFormatted": "₹500.00",

      "totalDiscount": 4549955,
      "totalDiscountFormatted": "₹45,499.55",

      "afterDiscount": 25449745,
      "afterDiscountFormatted": "₹2,54,497.45",

      "tax": 0,
      "taxFormatted": "₹0.00",

      "shipping": null,
      "shippingFormatted": null,

      "total": 25449745,
      "totalFormatted": "₹2,54,497.45",

      "savings": {
        "amount": 4549955,
        "percentage": 15.17,
        "message": "You're saving ₹45,499.55 (15% off)!"
      }
    },
    "metadata": {}
  },
  "message": "Cart retrieved successfully"
}
```

#### Response (Empty Cart - 200 OK)

```json
{
  "success": true,
  "data": {
    "id": 1,
    "userId": 3,
    "currency": {
      "code": "INR",
      "symbol": "₹",
      "decimalDigits": 2
    },
    "items": [],
    "appliedCoupons": [],
    "summary": {
      "itemCount": 0,
      "uniqueItems": 0,
      "subtotal": 0,
      "subtotalFormatted": "₹0.00",
      "promotionCount": 0,
      "promotionDiscount": 0,
      "promotionDiscountFormatted": "₹0.00",
      "couponCount": 0,
      "couponDiscount": 0,
      "couponDiscountFormatted": "₹0.00",
      "totalDiscount": 0,
      "totalDiscountFormatted": "₹0.00",
      "total": 0,
      "totalFormatted": "₹0.00"
    },
    "metadata": {}
  },
  "message": "Cart retrieved successfully"
}
```

---

### 2. Get Cart Summary

**Lightweight summary for header/badge display.**

```
GET /api/cart/summary
```

#### Response (200 OK)

```json
{
  "success": true,
  "data": {
    "itemCount": 5,
    "uniqueItems": 3,
    "subtotal": 29999700,
    "total": 25449745,
    "totalDiscount": 4549955,
    "currency": {
      "code": "INR",
      "symbol": "₹"
    },
    "pointsEquivalent": 2545
  },
  "message": "Cart summary retrieved"
}
```

---

### 3. Add Item to Cart

**Adds a product variant to the cart (or increases quantity if exists).**

```
POST /api/cart/item
```

#### Request Body

```json
{
  "variantId": 5,
  "quantity": 2
}
```

| Field       | Type   | Required | Validation                      |
| ----------- | ------ | -------- | ------------------------------- |
| `variantId` | `uint` | ✅       | Must exist, must be purchasable |
| `quantity`  | `int`  | ✅       | `gt=0`, `lte=99`                |

#### Response (201 Created)

Returns the cart with all items (without promotions, coupons, and summary). Uses `CartBasicResponse` model.

```json
{
  "success": true,
  "data": {
    "id": 1,
    "userId": 3,
    "currency": {
      "code": "INR",
      "symbol": "₹",
      "decimalDigits": 2
    },
    "items": [
      {
        "id": 1,
        "cartId": 1,
        "variantId": 5,
        "quantity": 2,
        "variant": {
          "id": 5,
          "sku": "IPH15PRO-BLK-256",
          "images": ["https://..."],
          "allowPurchase": true,
          "product": {
            "id": 1,
            "name": "iPhone 15 Pro",
            "slug": "iphone-15-pro"
          },
          "options": [
            { "name": "Color", "value": "Black" },
            { "name": "Storage", "value": "256GB" }
          ]
        }
      },
      {
        "id": 2,
        "cartId": 1,
        "variantId": 10,
        "quantity": 1,
        "variant": {
          "id": 10,
          "sku": "CASE-IPH15-CLR",
          "images": ["https://..."],
          "allowPurchase": true,
          "product": {
            "id": 5,
            "name": "iPhone 15 Clear Case",
            "slug": "iphone-15-clear-case"
          },
          "options": []
        }
      }
    ],
    "metadata": {}
  },
  "message": "Item added to cart"
}
```

#### Error Responses

| Status | Code                      | Message                                    |
| ------ | ------------------------- | ------------------------------------------ |
| 400    | `INVALID_QUANTITY`        | Quantity must be greater than 0            |
| 404    | `VARIANT_NOT_FOUND`       | Product variant not found                  |
| 400    | `VARIANT_NOT_PURCHASABLE` | This variant is not available for purchase |
| 400    | `INSUFFICIENT_STOCK`      | Not enough stock available                 |

---

### 4. Update Cart Item Quantity

**Updates the quantity of an existing cart item.**

```
PUT /api/cart/item/:itemId
```

#### Path Parameters

| Parameter | Type   | Description  |
| --------- | ------ | ------------ |
| `itemId`  | `uint` | Cart item ID |

#### Request Body

```json
{
  "quantity": 3
}
```

#### Response (200 OK)

Returns the updated cart with all items. Uses `CartBasicResponse` model (same as Add Item).

```json
{
  "success": true,
  "data": {
    "id": 1,
    "userId": 3,
    "currency": {
      "code": "INR",
      "symbol": "₹",
      "decimalDigits": 2
    },
    "items": [
      {
        "id": 1,
        "cartId": 1,
        "variantId": 5,
        "quantity": 3,
        "variant": {
          "id": 5,
          "sku": "IPH15PRO-BLK-256",
          "images": ["https://..."],
          "allowPurchase": true,
          "product": {
            "id": 1,
            "name": "iPhone 15 Pro",
            "slug": "iphone-15-pro"
          },
          "options": [
            { "name": "Color", "value": "Black" },
            { "name": "Storage", "value": "256GB" }
          ]
        }
      }
    ],
    "metadata": {}
  },
  "message": "Cart item updated"
}
```

#### Error Responses

| Status | Code                       | Message                            |
| ------ | -------------------------- | ---------------------------------- |
| 400    | `INVALID_QUANTITY`         | Quantity must be greater than 0    |
| 404    | `CART_ITEM_NOT_FOUND`      | Cart item not found                |
| 403    | `UNAUTHORIZED_CART_ACCESS` | You don't have access to this cart |
| 400    | `INSUFFICIENT_STOCK`       | Not enough stock available         |

---

### 5. Remove Item from Cart

**Removes an item from the cart.**

```
DELETE /api/cart/item/:itemId
```

#### Path Parameters

| Parameter | Type   | Description  |
| --------- | ------ | ------------ |
| `itemId`  | `uint` | Cart item ID |

#### Response (200 OK)

Returns the updated cart with remaining items. Uses `CartBasicResponse` model.

```json
{
  "success": true,
  "data": {
    "id": 1,
    "userId": 3,
    "currency": {
      "code": "INR",
      "symbol": "₹",
      "decimalDigits": 2
    },
    "items": [
      {
        "id": 2,
        "cartId": 1,
        "variantId": 10,
        "quantity": 1,
        "variant": {
          "id": 10,
          "sku": "CASE-IPH15-CLR",
          "images": ["https://..."],
          "allowPurchase": true,
          "product": {
            "id": 5,
            "name": "iPhone 15 Clear Case",
            "slug": "iphone-15-clear-case"
          },
          "options": []
        }
      }
    ],
    "metadata": {}
  },
  "message": "Item removed from cart"
}
```

#### Error Responses

| Status | Code                       | Message                            |
| ------ | -------------------------- | ---------------------------------- |
| 404    | `CART_ITEM_NOT_FOUND`      | Cart item not found                |
| 403    | `UNAUTHORIZED_CART_ACCESS` | You don't have access to this cart |

---

### 6. Clear Cart

**Removes all items from the cart.**

```
DELETE /api/cart
```

#### Response (200 OK)

```json
{
  "success": true,
  "data": null,
  "message": "Cart cleared successfully"
}
```

---

## 🎟️ Coupon APIs

### 1. Apply Coupon

**Applies a discount code (coupon) to the cart.**

```
POST /api/cart/coupon
```

#### Request Body

```json
{
  "code": "SAVE20"
}
```

| Field  | Type     | Required | Validation        |
| ------ | -------- | -------- | ----------------- |
| `code` | `string` | ✅       | `min=1`, `max=50` |

#### Response (200 OK)

```json
{
  "success": true,
  "data": {
    "coupon": {
      "id": 50,
      "code": "SAVE20",
      "title": "20% Off (Max ₹500)",
      "discountType": "percentage",
      "value": 20,
      "maxDiscountAmountCents": 50000
    },
    "applied": true,
    "message": "Coupon applied! You saved ₹500.00"
  },
  "message": "Coupon applied successfully"
}
```

#### Error Responses

| Status | Code                          | Message                                                 |
| ------ | ----------------------------- | ------------------------------------------------------- |
| 400    | `INVALID_COUPON`              | Invalid or expired coupon code                          |
| 400    | `COUPON_EXPIRED`              | This coupon has expired                                 |
| 400    | `COUPON_NOT_STARTED`          | This coupon is not yet active                           |
| 400    | `COUPON_USAGE_LIMIT_REACHED`  | This coupon has reached its usage limit                 |
| 400    | `COUPON_ALREADY_USED`         | You've already used this coupon (based on reset period) |
| 400    | `COUPON_MIN_PURCHASE_NOT_MET` | Minimum purchase of ₹X required                         |
| 400    | `COUPON_NOT_ELIGIBLE`         | You are not eligible for this coupon                    |
| 400    | `COUPON_CANNOT_COMBINE`       | This coupon cannot be combined with other discounts     |
| 400    | `COUPON_ALREADY_APPLIED`      | This coupon is already applied to your cart             |
| 400    | `COUPON_NOT_APPLICABLE`       | Coupon not applicable to items in your cart             |

---

### 2. Remove Specific Coupon

**Removes a specific coupon from the cart.**

```
DELETE /api/cart/coupon/:code
```

#### Path Parameters

| Parameter | Type     | Description           |
| --------- | -------- | --------------------- |
| `code`    | `string` | Coupon code to remove |

#### Response (200 OK)

```json
{
  "success": true,
  "data": {
    "removedCoupon": "SAVE20",
    "remainingCoupons": [
      {
        "code": "FREESHIP",
        "discountType": "free_shipping"
      }
    ]
  },
  "message": "Coupon removed"
}
```

---

### 3. Remove All Coupons

**Removes all coupons from the cart.**

```
DELETE /api/cart/coupon
```

#### Response (200 OK)

```json
{
  "success": true,
  "data": {
    "removedCount": 2
  },
  "message": "All coupons removed"
}
```

---

### 4. Get Available Coupons

**Lists coupons available for the current cart.**

```
GET /api/cart/available-coupon
```

#### Response (200 OK)

```json
{
  "success": true,
  "data": {
    "applicable": [
      {
        "id": 60,
        "code": "NEWYEAR2026",
        "title": "New Year Sale - 25% Off",
        "discountType": "percentage",
        "value": 25,
        "maxDiscountAmountCents": 100000,
        "potentialDiscount": 63749,
        "potentialDiscountFormatted": "₹637.49",
        "minPurchaseAmountCents": 100000,
        "canCombineWithOtherDiscounts": true,
        "startsAt": "2026-01-01T00:00:00Z",
        "endsAt": "2026-01-31T23:59:59Z"
      }
    ],
    "notApplicable": [
      {
        "id": 70,
        "code": "FIRST50",
        "title": "50% Off for New Users",
        "reason": "Only for first-time customers"
      },
      {
        "id": 75,
        "code": "MIN5000",
        "title": "₹1000 Off on ₹5000+",
        "reason": "Minimum purchase ₹5,000 required (current: ₹2,549.97)"
      }
    ]
  },
  "message": "Available coupons retrieved"
}
```

---

## 💰 Pricing Calculation

### Calculation Flow

```
┌─────────────────────────────────────────────────────────────┐
│  1. FETCH CART ITEMS                                        │
│     - Get variants with current prices                     │
│     - Calculate line totals (price × quantity)             │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│  2. APPLY PROMOTIONS (Auto-applied)                         │
│     - Find active promotions for each item                 │
│     - Check eligibility, dates, usage limits               │
│     - Apply stackable promotions (by priority)             │
│     - Calculate per-item promotion discounts               │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│  3. CALCULATE AFTER-PROMOTION SUBTOTAL                      │
│     subtotal_after_promotions = subtotal - promotion_discounts │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│  4. VALIDATE & APPLY COUPONS                                │
│     - Re-validate all applied coupons                      │
│     - Remove invalid/expired coupons                       │
│     - Calculate coupon discounts (on after-promotion total)│
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│  5. CALCULATE FINAL TOTALS                                  │
│     after_discount = subtotal - promotions - coupons       │
│     tax = calculate_tax(after_discount)  // TODO           │
│     shipping = calculate_shipping()       // TODO          │
│     total = after_discount + tax + shipping                │
└─────────────────────────────────────────────────────────────┘
```

### Discount Calculation Examples

#### Percentage Discount with Cap

```
Cart Total: ₹10,000
Coupon: 20% off, max ₹500

Calculation:
  20% of ₹10,000 = ₹2,000
  Max cap = ₹500
  Final discount = ₹500 (capped)
```

#### Multiple Stackable Coupons

```
Cart Total: ₹10,000
Coupon 1: ₹500 off (fixed)
Coupon 2: 10% off (can combine)

Calculation:
  After Coupon 1: ₹10,000 - ₹500 = ₹9,500
  Coupon 2: 10% of ₹9,500 = ₹950
  Final total: ₹9,500 - ₹950 = ₹8,550
```

---

## ✅ Validation Rules

### Coupon Validation

| Rule                   | Field                                         | Description                                             |
| ---------------------- | --------------------------------------------- | ------------------------------------------------------- |
| **Active**             | `is_active`                                   | Must be `true`                                          |
| **Date Range**         | `starts_at`, `ends_at`                        | Current time must be within range                       |
| **Total Usage**        | `usage_limit_total`, `current_usage_count`    | Not exceeded total limit                                |
| **Per-Customer Usage** | `usage_limit_per_customer`                    | User hasn't exceeded limit in reset period              |
| **Reset Period**       | `usage_reset_time_type`, `usage_reset_amount` | Check usage within period                               |
| **Min Purchase**       | `min_purchase_amount_cents`                   | Cart total meets minimum                                |
| **Min Quantity**       | `min_quantity`                                | Cart has minimum items                                  |
| **Eligibility**        | `customer_eligibility`                        | User matches (everyone/new/returning/segment)           |
| **Scope**              | `applies_to`                                  | Items match scope (all/products/categories/collections) |
| **Combinability**      | `can_combine_with_other_discounts`            | Can stack with existing coupons                         |

### Usage Reset Period Examples

| `usage_reset_time_type` | `usage_reset_amount` | `usage_limit_per_customer` | Meaning           |
| ----------------------- | -------------------- | -------------------------- | ----------------- |
| `none`                  | `null`               | `1`                        | One-time use ever |
| `day`                   | `1`                  | `1`                        | Once per day      |
| `day`                   | `30`                 | `1`                        | Once per 30 days  |
| `week`                  | `2`                  | `1`                        | Once per 2 weeks  |
| `month`                 | `1`                  | `3`                        | 3 times per month |
| `year`                  | `1`                  | `1`                        | Once per year     |

---

## ❌ Error Codes

### Cart Errors

| Code                       | HTTP Status | Message                                    |
| -------------------------- | ----------- | ------------------------------------------ |
| `CART_NOT_FOUND`           | 404         | Cart not found                             |
| `CART_ITEM_NOT_FOUND`      | 404         | Cart item not found                        |
| `UNAUTHORIZED_CART_ACCESS` | 403         | You don't have access to this cart         |
| `INVALID_QUANTITY`         | 400         | Quantity must be greater than 0            |
| `VARIANT_NOT_FOUND`        | 404         | Product variant not found                  |
| `VARIANT_NOT_PURCHASABLE`  | 400         | This variant is not available for purchase |
| `INSUFFICIENT_STOCK`       | 400         | Not enough stock available                 |

### Coupon Errors

| Code                          | HTTP Status | Message                                             |
| ----------------------------- | ----------- | --------------------------------------------------- |
| `INVALID_COUPON`              | 400         | Invalid or expired coupon code                      |
| `COUPON_EXPIRED`              | 400         | This coupon has expired                             |
| `COUPON_NOT_STARTED`          | 400         | This coupon is not yet active                       |
| `COUPON_USAGE_LIMIT_REACHED`  | 400         | This coupon has reached its usage limit             |
| `COUPON_ALREADY_USED`         | 400         | You've already used this coupon                     |
| `COUPON_MIN_PURCHASE_NOT_MET` | 400         | Minimum purchase of ₹X required                     |
| `COUPON_NOT_ELIGIBLE`         | 400         | You are not eligible for this coupon                |
| `COUPON_CANNOT_COMBINE`       | 400         | This coupon cannot be combined with other discounts |
| `COUPON_ALREADY_APPLIED`      | 400         | This coupon is already applied to your cart         |
| `COUPON_NOT_APPLICABLE`       | 400         | Coupon not applicable to items in your cart         |

---

## 🛤️ Route Configuration

```go
// routes/cart_route.go
func (m *CartModule) RegisterRoutes(router *gin.Engine) {
    cart := router.Group("/api/cart")
    cart.Use(middleware.CorrelationID())
    cart.Use(middleware.AuthMiddleware())
    cart.Use(middleware.CustomerAuth())
    {
        // Cart operations
        cart.GET("", m.handler.GetCart)
        cart.GET("/summary", m.handler.GetCartSummary)
        cart.DELETE("", m.handler.ClearCart)

        // Cart items
        cart.POST("/item", m.handler.AddItem)
        cart.PUT("/item/:itemId", m.handler.UpdateItem)
        cart.DELETE("/item/:itemId", m.handler.RemoveItem)

        // Coupons
        cart.POST("/coupon", m.handler.ApplyCoupon)
        cart.DELETE("/coupon", m.handler.RemoveAllCoupons)
        cart.DELETE("/coupon/:code", m.handler.RemoveCoupon)
        cart.GET("/available-coupon", m.handler.GetAvailableCoupons)
    }
}
```

---

## 🔮 Future Enhancements

### TODO: Tax Calculation Module

```json
{
  "tax": 45899,
  "taxFormatted": "₹458.99",
  "taxBreakdown": [
    { "name": "CGST", "rate": "9%", "amount": 22949 },
    { "name": "SGST", "rate": "9%", "amount": 22950 }
  ]
}
```

### TODO: Shipping Calculation

```json
{
  "shipping": 9900,
  "shippingFormatted": "₹99.00",
  "freeShipping": false,
  "freeShippingThreshold": 50000,
  "amountToFreeShipping": 24550
}
```

### TODO: Loyalty Points

```json
{
  "pointsEquivalent": 2545,
  "pointsEarnedOnPurchase": 127,
  "pointsRedemptionAvailable": 500,
  "pointsRedemptionValue": 5000
}
```

---

## 📁 File Structure

```
order/
├── CART_API_PRD.md              ← This document
├── container.go
├── entity/
│   └── cart.go                  ✅ Implemented
├── model/
│   └── cart_model.go            ✅ Implemented
├── repository/
│   └── cart_repository.go       ✅ Partially implemented
├── service/
│   └── cart_service.go          (to create)
├── handler/
│   └── cart_handler.go          (to create)
├── route/
│   └── cart_route.go            (to create)
└── error/
    └── cart_error.go            (to create)
```

---

## 📝 Change Log

| Date       | Version | Changes                                     |
| ---------- | ------- | ------------------------------------------- |
| 2026-01-28 | 1.0     | Initial cart API design with coupon support |
