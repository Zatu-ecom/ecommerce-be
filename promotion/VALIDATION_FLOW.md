# Promotion Validation Flow

## 📋 Overview

The promotion validation happens in **two layers**:

1. **General Validation** - Common checks for all promotion types
2. **Type-Specific Validation** - Strategy pattern for each promotion type

---

## 🔍 General Validation (Before Strategy)

These checks happen in `ValidatePromotionForCart()` **before** delegating to type-specific strategies:

### 1. Promotion Status
```go
if promotion.Status != entity.StatusActive {
    return "Promotion is not active"
}
```

### 2. Date Range
```go
// Check if promotion has started
if promotion.StartsAt != nil && now.Before(*promotion.StartsAt) {
    return "Promotion has not started yet"
}

// Check if promotion has ended
if promotion.EndsAt != nil && now.After(*promotion.EndsAt) {
    return "Promotion has ended"
}
```

### 3. Usage Limits
```go
// Total usage limit
if promotion.UsageLimitTotal != nil && 
   promotion.CurrentUsageCount >= *promotion.UsageLimitTotal {
    return "Promotion usage limit reached"
}

// Per-customer usage limit (checked separately in handler)
```

### 4. Customer Eligibility (`EligibleFor`)
```go
switch promotion.EligibleFor {
case entity.EligibleEveryone:
    ✅ All customers eligible

case entity.EligibleNewCustomers:
    ✅ Only if cart.IsNewCustomer == true

case entity.EligibleReturningCustomers:
    ✅ Only if cart.IsNewCustomer == false

case entity.EligibleSpecificSegment:
    ✅ Only if customer belongs to promotion.CustomerSegmentID
    // Checks cart.CustomerSegmentIDs array
}
```

### 5. Minimum Purchase Amount
```go
if promotion.MinPurchaseAmountCents != nil && 
   cart.SubtotalCents < *promotion.MinPurchaseAmountCents {
    return "Minimum purchase amount not met"
}
```

### 6. Minimum Quantity
```go
if promotion.MinQuantity != nil {
    totalQuantity = sum of all cart item quantities
    if totalQuantity < *promotion.MinQuantity {
        return "Minimum quantity not met"
    }
}
```

### 7. **Promotion Scope (`AppliesTo`)** ⭐ NEW

This checks if cart items are eligible based on the promotion's scope:

```go
switch promotion.AppliesTo {
case entity.ScopeAllProducts:
    ✅ Always eligible - applies to all products

case entity.ScopeSpecificProducts:
    ✅ Check if ANY cart item is in promotion's product list
    // Fetches from promotion_products table
    // Returns true if at least one cart item matches

case entity.ScopeSpecificCategories:
    ✅ Check if ANY cart item's category is in promotion's category list
    // Fetches from promotion_categories table
    // Uses cart.Items[].CategoryID

case entity.ScopeSpecificCollections:
    ❌ Not yet supported (CartItem needs CollectionID field)
    // TODO: Add collection support
}
```

---

## 🎯 Type-Specific Validation (Strategy Pattern)

After all general checks pass, the validation is delegated to the appropriate strategy:

```go
strategy := promotionStrategy.GetPromotionStrategy(promotion.PromotionType)
return strategy.ValidateCart(ctx, promotion, cart)
```

Each strategy validates its specific requirements:

| Strategy | Validates |
|----------|-----------|
| **Percentage** | Max discount cap |
| **Fixed Amount** | Discount doesn't exceed cart total |
| **Free Shipping** | Shipping amount available |
| **Buy X Get Y** | Sufficient quantity per product |
| **Bundle** | All bundle items present in cart |
| **Tiered** | Which tier applies based on quantity/spend |
| **Flash Sale** | Stock limits, time windows |

---

## 🔄 Complete Validation Flow

```
ValidatePromotionForCart()
    │
    ├─ 1. Fetch Promotion
    │
    ├─ 2. Check Status (active?)
    │
    ├─ 3. Check Date Range (started? not ended?)
    │
    ├─ 4. Check Usage Limits (not exceeded?)
    │
    ├─ 5. Check Customer Eligibility (EligibleFor)
    │      ├─ Everyone
    │      ├─ New Customers (cart.IsNewCustomer)
    │      ├─ Returning Customers (!cart.IsNewCustomer)
    │      └─ Specific Segment (cart.CustomerSegmentIDs)
    │
    ├─ 6. Check Minimum Purchase (cart.SubtotalCents)
    │
    ├─ 7. Check Minimum Quantity (sum of cart quantities)
    │
    ├─ 8. Check Promotion Scope (AppliesTo) ⭐ NEW
    │      ├─ All Products → Always pass
    │      ├─ Specific Products → Check cart items
    │      ├─ Specific Categories → Check cart categories
    │      └─ Specific Collections → TODO
    │
    └─ 9. Delegate to Strategy
           └─ strategy.ValidateCart() → Type-specific logic
```

---

## 📊 How Customer Eligibility Works

### Data Required in `CartValidationRequest`

```go
type CartValidationRequest struct {
    Items              []CartItem
    SubtotalCents      int64
    ShippingCents      int64
    CustomerID         *uint   // Optional: logged-in customer
    IsNewCustomer      bool    // ⭐ Is this their first purchase?
    CustomerSegmentIDs []uint  // ⭐ Which segments they belong to
}
```

### Determining Customer Type

**Frontend/Handler Responsibility:**

1. **IsNewCustomer**
   ```go
   // Check if customer has any previous orders
   orderCount := orderService.GetOrderCountByCustomer(customerID)
   cart.IsNewCustomer = (orderCount == 0)
   ```

2. **CustomerSegmentIDs**
   ```go
   // Fetch customer's segments from customer service
   segments := customerService.GetCustomerSegments(customerID)
   cart.CustomerSegmentIDs = extractSegmentIDs(segments)
   ```

### Example Scenarios

#### Scenario 1: New Customer Promotion
```json
Promotion:
{
  "eligible_for": "new_customers",
  "discount": "20% off first order"
}

Cart:
{
  "customer_id": 123,
  "is_new_customer": true,  // ✅ First order
  "subtotal_cents": 50000
}

Result: ✅ Eligible
```

#### Scenario 2: VIP Segment Promotion
```json
Promotion:
{
  "eligible_for": "specific_segment",
  "customer_segment_id": 5,  // VIP segment
  "discount": "30% off"
}

Cart:
{
  "customer_id": 456,
  "customer_segment_ids": [3, 5, 7],  // ✅ Includes segment 5
  "is_new_customer": false
}

Result: ✅ Eligible
```

#### Scenario 3: Returning Customer Promotion
```json
Promotion:
{
  "eligible_for": "returning_customers",
  "discount": "15% off"
}

Cart:
{
  "customer_id": 789,
  "is_new_customer": false,  // ✅ Has previous orders
  "customer_segment_ids": []
}

Result: ✅ Eligible
```

---

## 🎯 Scope Validation Examples

### Example 1: Specific Products
```json
Promotion:
{
  "applies_to": "specific_products",
  "promotion_products": [101, 102, 103]  // T-shirt IDs
}

Cart:
{
  "items": [
    {"product_id": 101, "quantity": 2},  // ✅ Match
    {"product_id": 205, "quantity": 1}   // Not in scope
  ]
}

Result: ✅ Eligible (at least one item matches)
```

### Example 2: Specific Categories
```json
Promotion:
{
  "applies_to": "specific_categories",
  "promotion_categories": [5, 8]  // Electronics, Accessories
}

Cart:
{
  "items": [
    {"product_id": 301, "category_id": 5},  // ✅ Electronics
    {"product_id": 402, "category_id": 12}  // Clothing (not in scope)
  ]
}

Result: ✅ Eligible (at least one item in eligible category)
```

---

## ✅ Summary

### General Checks (All Promotions)
1. ✅ Status is active
2. ✅ Within date range
3. ✅ Usage limits not exceeded
4. ✅ Customer eligible (`EligibleFor`)
5. ✅ Minimum purchase met
6. ✅ Minimum quantity met
7. ✅ **Scope matches (`AppliesTo`)** ⭐

### Type-Specific Checks (Strategy)
8. ✅ Promotion type-specific validation

### Customer Eligibility Determined By
- **IsNewCustomer** - Set by checking order history
- **CustomerSegmentIDs** - Set by customer service/segments

All general validations must pass before type-specific strategy is invoked!
