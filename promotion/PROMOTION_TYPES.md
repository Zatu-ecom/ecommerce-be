# 🎯 Supported Promotion Types

> **Module**: Promotion Service  
> **Last Updated**: February 14, 2026

---

## Overview

Our promotion system supports **two distinct discount mechanisms**:

| Mechanism | Purpose | Applied |
|---|---|---|
| **Promotions** | Seller-created sales/offers | Auto-applied at runtime based on rules |
| **Discount Codes** (Coupons) | Manual coupon codes | Customer applies code to cart |

---

## 🏷️ Promotion Types

### 1. Percentage Discount (`percentage_discount`)

Apply a percentage off the item price. Optionally cap the maximum discount.

**`discount_config` shape:**
```json
{
  "percentage": 15,
  "max_discount_cents": 100000
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `percentage` | `float64` | ✅ | Percentage (1–100) |
| `max_discount_cents` | `int64` | ❌ | Max discount cap in cents/paise |

**Example**: *"15% off, max ₹1,000 discount"*

---

### 2. Fixed Amount (`fixed_amount`)

Flat amount off the item or order total.

**`discount_config` shape:**
```json
{
  "amount_cents": 50000
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `amount_cents` | `int64` | ✅ | Fixed discount in cents/paise |

**Example**: *"₹500 off"*

---

### 3. Buy X Get Y (`buy_x_get_y`)

Buy a certain quantity and get items free.

**`discount_config` shape:**
```json
{
  "buy_quantity": 2,
  "get_quantity": 1,
  "max_sets": 3,
  "is_same_reward": true,
  "scope_type": "same_product",
  "get_product_id": null
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `buy_quantity` | `int` | ✅ | Items customer must buy |
| `get_quantity` | `int` | ✅ | Items customer gets free |
| `max_sets` | `int` | ❌ | Max times offer can apply per order |
| `is_same_reward` | `bool` | ❌ | `true` means reward comes from the same pool; defaults to `true` |
| `scope_type` | `string` | Conditionally | Required when `is_same_reward=true`; one of `same_variant`, `same_product`, `same_category` |
| `get_product_id` | `uint` | Conditionally | Required when `is_same_reward=false`; the specific reward product |

**Examples**:
- *"Buy 2, Get 1 Free from the same product"* → `is_same_reward: true, scope_type: same_product`
- *"Buy 1 phone, Get 1 headphones Free"* → `is_same_reward: false, get_product_id: 4`

---

### 4. Free Shipping (`free_shipping`)

Waive shipping costs. Can be conditional on cart total.

**`discount_config` shape:**
```json
{
  "min_order_cents": 200000,
  "max_shipping_discount_cents": 15000
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `min_order_cents` | `int64` | ❌ | Min order to qualify (0 = no minimum) |
| `max_shipping_discount_cents` | `int64` | ❌ | Max shipping discount (null = full shipping waived) |

**Example**: *"Free shipping on orders above ₹2,000"*

---

### 5. Bundle Discount (`bundle`)

Discounted price when buying a specific set of products together.

**`discount_config` shape:**
```json
{
  "bundle_items": [
    { "product_id": 1, "variant_id": 5, "quantity": 1 },
    { "product_id": 2, "variant_id": 8, "quantity": 1 }
  ],
  "bundle_discount_type": "percentage",
  "bundle_discount_value": 20,
  "bundle_price_cents": null
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `bundle_items` | `[]object` | ✅ | Products/variants in the bundle |
| `bundle_discount_type` | `string` | ✅ | `"percentage"`, `"fixed_amount"`, or `"fixed_price"` |
| `bundle_discount_value` | `float64` | depends | % or fixed amount off |
| `bundle_price_cents` | `int64` | depends | Fixed bundle price (for `fixed_price` type) |

**Examples**:
- *"Phone + Case = 20% off"*
- *"Complete outfit bundle for ₹2,999"*

---

### 6. Tiered Discount (`tiered`)

Discounts that increase based on quantity purchased or amount spent.

**`discount_config` shape:**
```json
{
  "tier_type": "quantity",
  "tiers": [
    { "min": 2, "max": 4, "discount_type": "percentage", "discount_value": 5 },
    { "min": 5, "max": 9, "discount_type": "percentage", "discount_value": 10 },
    { "min": 10, "max": null, "discount_type": "percentage", "discount_value": 15 }
  ]
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `tier_type` | `string` | ✅ | `"quantity"` or `"spend"` |
| `tiers` | `[]object` | ✅ | Ordered tier breakpoints |
| `tiers[].min` | `int` | ✅ | Minimum quantity/amount for this tier |
| `tiers[].max` | `int` | ❌ | Maximum (null = unlimited) |
| `tiers[].discount_type` | `string` | ✅ | `"percentage"` or `"fixed_amount"` |
| `tiers[].discount_value` | `float64` | ✅ | Discount at this tier |

**Examples**:
- *"Buy 2–4: 5% off, Buy 5–9: 10% off, Buy 10+: 15% off"*
- *"Spend ₹2,000+: ₹200 off, ₹5,000+: ₹700 off"*

---

### 7. Flash Sale (`flash_sale`)

Time-limited deep discounts. Typically short duration (hours/days).

**`discount_config` shape:**
```json
{
  "discount_type": "percentage",
  "discount_value": 30,
  "max_discount_cents": 200000,
  "stock_limit": 50,
  "sold_count": 12
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `discount_type` | `string` | ✅ | `"percentage"` or `"fixed_amount"` |
| `discount_value` | `float64` | ✅ | Discount value |
| `max_discount_cents` | `int64` | ❌ | Cap for percentage type |
| `stock_limit` | `int` | ❌ | Max units at sale price |
| `sold_count` | `int` | ❌ | Counter for units sold (managed by system) |

**Example**: *"Flash Sale — 30% off (max ₹2,000) — only 50 units!"*

---

## 🎟️ Discount Code Types

Discount codes (coupons) use a simpler `discount_type` field:

| `discount_type` | `value` Field | Description |
|---|---|---|
| `percentage` | `15` → 15% | Percentage off cart/items |
| `fixed_amount` | `50000` → ₹500 | Flat amount off |
| `free_shipping` | `0` | Waive shipping |
| `buy_x_get_y` | — | Complex (uses metadata) |

---

## 📋 Scope Types (`applies_to`)

Both promotions and discount codes can be scoped:

| `applies_to` Value | Scope Tables Used | Description |
|---|---|---|
| `all_products` | — | Applies to entire catalog |
| `specific_products` | `promotion_product` / `discount_code_product` | Only listed products/variants |
| `specific_categories` | `promotion_category` / `discount_code_category` | All products in listed categories |
| `specific_collections` | `promotion_collection` / `discount_code_collection` | All products in listed collections |

---

## 🧑‍🤝‍🧑 Customer Eligibility (`eligible_for`)

| Value | Description |
|---|---|
| `everyone` | All customers |
| `new_customers` | First-time buyers only |
| `returning_customers` | Repeat purchasers only |
| `specific_segment` | Customers in a `customer_segment` (rule-based) |

---

## 🔄 Promotion Status Lifecycle

```
  ┌───────┐
  │ draft │ ←────── Created (default)
  └───┬───┘
      │ activate (manual or auto_start at starts_at)
  ┌───▼──────┐
  │ scheduled │ ──── if starts_at is in the future
  └───┬──────┘
      │ starts_at reached
  ┌───▼───┐
  │ active │ ←────── Promotion is live
  └─┬───┬─┘
    │   │ pause
    │ ┌─▼────┐
    │ │paused│ ──── temporarily stopped
    │ └─┬────┘
    │   │ resume (activate)
    │   │
  ┌─▼───▼───┐
  │  ended   │ ←────── ends_at reached / manual end
  └─────────┘
```

---

## 📝 Change Log

| Date | Changes |
|---|---|
| 2026-02-14 | Initial documentation of all supported promotion types |
