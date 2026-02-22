# New Customer Determination Logic

## 🎯 Definition

**New Customer = First-time buyer (no completed orders)**

A customer is considered "new" if they have **zero completed orders** in the system.

---

## 📊 How It Works

### CartValidationRequest Field

```go
type CartValidationRequest struct {
    CustomerID         *uint  // The customer ID
    HasCompletedOrders bool   // ⭐ Set by Order Service
    // ... other fields
}
```

### Validation Logic

```go
case entity.EligibleNewCustomers:
    // New customer = no completed orders (first-time buyer)
    return !cart.HasCompletedOrders
```

**Simple:**
- `HasCompletedOrders = false` → New customer ✅
- `HasCompletedOrders = true` → Not new customer ❌

---

## 🔄 Who Sets `HasCompletedOrders`?

### Order Service Responsibility

The **Order Service** determines if a customer has completed orders:

```go
// order/service/order_service.go

// HasCompletedOrders checks if customer has any completed orders
func (s *OrderService) HasCompletedOrders(
    ctx context.Context,
    customerID uint,
) (bool, error) {
    count, err := s.orderRepo.CountCompletedOrdersByCustomer(ctx, customerID)
    if err != nil {
        return false, err
    }
    return count > 0, nil
}
```

### Repository Method

```go
// order/repository/order_repository.go

func (r *OrderRepositoryImpl) CountCompletedOrdersByCustomer(
    ctx context.Context,
    customerID uint,
) (int64, error) {
    var count int64
    err := db.GetDB().WithContext(ctx).
        Model(&entity.Order{}).
        Where("customer_id = ?", customerID).
        Where("status IN (?)", []string{"completed", "delivered"}).
        Count(&count).Error
    
    return count, err
}
```

---

## 🛒 Checkout Handler Implementation

### How to Set the Field

```go
// checkout/handler/checkout_handler.go

func (h *CheckoutHandler) ApplyPromotions(c *gin.Context) {
    var req CheckoutRequest
    c.ShouldBindJSON(&req)
    
    // Build cart validation request
    cartValidation := &model.CartValidationRequest{
        Items:         req.Items,
        SubtotalCents: req.SubtotalCents,
        ShippingCents: req.ShippingCents,
        CustomerID:    req.CustomerID,
    }
    
    // Set HasCompletedOrders if customer is logged in
    if req.CustomerID != nil {
        hasOrders, err := h.orderService.HasCompletedOrders(c, *req.CustomerID)
        if err != nil {
            // Log error, default to false (treat as new customer)
            hasOrders = false
        }
        cartValidation.HasCompletedOrders = hasOrders
    } else {
        // Guest checkout - treat as new customer
        cartValidation.HasCompletedOrders = false
    }
    
    // Validate promotions
    for _, promoID := range req.PromotionIDs {
        result, _ := h.promotionService.ValidatePromotionForCart(c, promoID, cartValidation)
        // ... apply discount
    }
}
```

---

## 📋 Example Scenarios

### Scenario 1: First-Time Buyer (New Customer)

```json
Customer:
{
  "id": 123,
  "email": "john@example.com",
  "created_at": "2024-02-15"
}

Order History:
[]  // No orders

Cart Validation:
{
  "customer_id": 123,
  "has_completed_orders": false  // ⭐ No orders
}

Promotion:
{
  "eligible_for": "new_customers",
  "discount": "20% off first order"
}

Result: ✅ Eligible (first-time buyer)
```

### Scenario 2: Returning Customer

```json
Customer:
{
  "id": 456,
  "email": "jane@example.com",
  "created_at": "2024-01-10"
}

Order History:
[
  {"id": 1001, "status": "completed", "created_at": "2024-01-15"},
  {"id": 1002, "status": "completed", "created_at": "2024-02-01"}
]

Cart Validation:
{
  "customer_id": 456,
  "has_completed_orders": true  // ⭐ Has 2 completed orders
}

Promotion:
{
  "eligible_for": "new_customers",
  "discount": "20% off first order"
}

Result: ❌ Not Eligible (has previous orders)
```

### Scenario 3: Guest Checkout

```json
Customer:
null  // Guest user

Cart Validation:
{
  "customer_id": null,
  "has_completed_orders": false  // ⭐ Treat guest as new
}

Promotion:
{
  "eligible_for": "new_customers",
  "discount": "20% off first order"
}

Result: ✅ Eligible (guest = new customer)
```

### Scenario 4: Pending Orders Don't Count

```json
Customer:
{
  "id": 789,
  "email": "bob@example.com"
}

Order History:
[
  {"id": 2001, "status": "pending", "created_at": "2024-02-15"},
  {"id": 2002, "status": "processing", "created_at": "2024-02-15"}
]

Cart Validation:
{
  "customer_id": 789,
  "has_completed_orders": false  // ⭐ No COMPLETED orders
}

Promotion:
{
  "eligible_for": "new_customers",
  "discount": "20% off first order"
}

Result: ✅ Eligible (only completed orders count)
```

---

## 🎯 Order Statuses That Count

### Completed Statuses

Only these order statuses count as "completed":

```go
completedStatuses := []string{
    "completed",
    "delivered",
}
```

### Don't Count

These statuses do NOT count:
- ❌ `pending`
- ❌ `processing`
- ❌ `shipped`
- ❌ `cancelled`
- ❌ `refunded`
- ❌ `failed`

**Why?** Because the customer hasn't actually completed a purchase yet.

---

## 🔧 Edge Cases

### 1. **Customer with Cancelled Orders Only**

```json
Order History:
[
  {"id": 3001, "status": "cancelled"}
]

has_completed_orders: false  // ✅ Still a new customer
```

### 2. **Customer Created Account But Never Ordered**

```json
Customer:
{
  "id": 999,
  "created_at": "2023-01-01"  // Account 1 year old
}

Order History: []

has_completed_orders: false  // ✅ Still a new customer
```

### 3. **Multiple Promotions in Same Checkout**

```go
// First promotion applied
hasCompletedOrders = false  // New customer

// Order is created and completed
// ... order processing ...

// Second promotion applied in same session
hasCompletedOrders = false  // Still false (order not yet marked complete)
```

**Note:** The flag is set at checkout time, not updated during the transaction.

---

## ✅ Summary

### Definition
**New Customer = No completed orders (first-time buyer)**

### Responsibility
| Service | Does |
|---------|------|
| **Order Service** | Provides `HasCompletedOrders(customerID)` method |
| **Checkout Handler** | Calls order service and sets `cart.HasCompletedOrders` |
| **Promotion Service** | Just checks `!cart.HasCompletedOrders` |

### Completed Order Criteria
- Status must be `completed` or `delivered`
- Pending/processing/cancelled orders don't count

### Guest Users
- Treated as new customers (`HasCompletedOrders = false`)

### Benefits
✅ **Simple** - Clear definition (no completed orders)  
✅ **Accurate** - Based on actual order history  
✅ **Flexible** - Order service owns the logic  
✅ **Cacheable** - Can cache the result per customer  
✅ **Fair** - Only counts completed purchases
