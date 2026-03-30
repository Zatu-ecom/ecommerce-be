# Customer Segments for Promotions

## 🎯 Design Decision

**Customer eligibility is managed through SEGMENTS, not hardcoded types.**

### Why Segments?

✅ **Flexible** - Create any customer type (new, VIP, loyal, etc.)  
✅ **Scalable** - Add new segments without code changes  
✅ **Business-driven** - Marketing team can define segments  
✅ **Reusable** - Same segments used across promotions, email campaigns, etc.

---

## 📊 Customer Segment Table Structure

### Recommended Schema

```sql
CREATE TABLE customer_segments (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    segment_type VARCHAR(50) NOT NULL, -- 'new_customer', 'vip', 'loyal', 'inactive', etc.
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE customer_segment_members (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    customer_id BIGINT NOT NULL,
    segment_id BIGINT NOT NULL,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (customer_id) REFERENCES customers(id),
    FOREIGN KEY (segment_id) REFERENCES customer_segments(id),
    UNIQUE KEY unique_customer_segment (customer_id, segment_id)
);

CREATE INDEX idx_customer_segment_members_customer ON customer_segment_members(customer_id);
CREATE INDEX idx_customer_segment_members_segment ON customer_segment_members(segment_id);
```

---

## 🏗️ Entity Structure (Go)

### Customer Segment Entity

```go
// customer/entity/customer_segment.go
package entity

import "ecommerce-be/common/db"

type CustomerSegment struct {
    db.BaseEntity
    
    Name        string  `json:"name" gorm:"column:name;size:255;not null"`
    Description *string `json:"description" gorm:"column:description;type:text"`
    SegmentType string  `json:"segmentType" gorm:"column:segment_type;size:50;not null"`
    IsActive    bool    `json:"isActive" gorm:"column:is_active;default:true"`
}

type CustomerSegmentMember struct {
    db.BaseEntity
    
    CustomerID uint `json:"customerId" gorm:"column:customer_id;not null;index"`
    SegmentID  uint `json:"segmentId" gorm:"column:segment_id;not null;index"`
}
```

---

## 🎯 Promotion Eligibility Types

### Simplified to 2 Types

```go
type EligibilityType string

const (
    EligibleEveryone        EligibilityType = "everyone"
    EligibleSpecificSegment EligibilityType = "specific_segment"
)
```

### How It Works

| Promotion Setting | Behavior |
|-------------------|----------|
| `eligible_for: "everyone"` | All customers can use |
| `eligible_for: "specific_segment"` + `customer_segment_id: 5` | Only customers in segment 5 |

---

## 📋 Example Segments

### Predefined Segments (Created by Admin)

```json
[
  {
    "id": 1,
    "name": "New Customers",
    "segment_type": "new_customer",
    "description": "Customers with 0 orders"
  },
  {
    "id": 2,
    "name": "VIP Customers",
    "segment_type": "vip",
    "description": "Customers with lifetime spend > $10,000"
  },
  {
    "id": 3,
    "name": "Loyal Customers",
    "segment_type": "loyal",
    "description": "Customers with 10+ orders"
  },
  {
    "id": 4,
    "name": "Inactive Customers",
    "segment_type": "inactive",
    "description": "No purchase in last 90 days"
  },
  {
    "id": 5,
    "name": "Birthday Month",
    "segment_type": "birthday",
    "description": "Customers with birthday this month"
  }
]
```

---

## 🔄 Customer Service Responsibility

### Customer Service Should Provide

```go
// customer/service/customer_service.go

// GetCustomerSegments returns all segment IDs for a customer
func (s *CustomerService) GetCustomerSegments(
    ctx context.Context,
    customerID uint,
) ([]uint, error) {
    members, err := s.segmentMemberRepo.GetByCustomerID(ctx, customerID)
    if err != nil {
        return nil, err
    }
    
    segmentIDs := make([]uint, len(members))
    for i, member := range members {
        segmentIDs[i] = member.SegmentID
    }
    
    return segmentIDs, nil
}

// UpdateCustomerSegments automatically assigns/removes segments based on rules
func (s *CustomerService) UpdateCustomerSegments(
    ctx context.Context,
    customerID uint,
) error {
    customer, _ := s.customerRepo.FindByID(ctx, customerID)
    orderCount, _ := s.orderRepo.GetOrderCountByCustomer(ctx, customerID)
    lifetimeSpend, _ := s.orderRepo.GetLifetimeSpend(ctx, customerID)
    
    // Auto-assign to "New Customers" segment
    if orderCount == 0 {
        s.segmentMemberRepo.AssignToSegment(ctx, customerID, 1) // Segment ID 1
    } else {
        s.segmentMemberRepo.RemoveFromSegment(ctx, customerID, 1)
    }
    
    // Auto-assign to "VIP" segment
    if lifetimeSpend > 1000000 { // $10,000 in cents
        s.segmentMemberRepo.AssignToSegment(ctx, customerID, 2)
    }
    
    // Auto-assign to "Loyal" segment
    if orderCount >= 10 {
        s.segmentMemberRepo.AssignToSegment(ctx, customerID, 3)
    }
    
    return nil
}
```

---

## 🛒 Checkout Flow

### How Segments Are Passed to Promotion Validation

```go
// checkout/handler/checkout_handler.go

func (h *CheckoutHandler) ValidatePromotions(c *gin.Context) {
    var req CheckoutRequest
    c.ShouldBindJSON(&req)
    
    // Get customer segments from customer service
    var segmentIDs []uint
    if req.CustomerID != nil {
        segmentIDs, _ = h.customerService.GetCustomerSegments(c, *req.CustomerID)
    }
    
    // Build cart validation request
    cartValidation := &model.CartValidationRequest{
        Items:              req.Items,
        SubtotalCents:      req.SubtotalCents,
        ShippingCents:      req.ShippingCents,
        CustomerID:         req.CustomerID,
        CustomerSegmentIDs: segmentIDs, // ⭐ From customer service
    }
    
    // Validate each promotion
    for _, promoID := range req.PromotionIDs {
        result, _ := h.promotionService.ValidatePromotionForCart(c, promoID, cartValidation)
        // ... apply discount if valid
    }
}
```

---

## 🎯 Real-World Examples

### Example 1: New Customer Welcome Offer

```json
Promotion:
{
  "name": "Welcome 20% Off",
  "eligible_for": "specific_segment",
  "customer_segment_id": 1,  // "New Customers" segment
  "discount_config": {
    "percentage": 20
  }
}

Customer:
{
  "id": 123,
  "order_count": 0  // First order
}

Segments Assigned: [1]  // Auto-assigned to "New Customers"

Result: ✅ Eligible
```

### Example 2: VIP Exclusive Sale

```json
Promotion:
{
  "name": "VIP 30% Off",
  "eligible_for": "specific_segment",
  "customer_segment_id": 2,  // "VIP Customers" segment
  "discount_config": {
    "percentage": 30
  }
}

Customer:
{
  "id": 456,
  "lifetime_spend_cents": 1500000  // $15,000
}

Segments Assigned: [2, 3]  // VIP + Loyal

Result: ✅ Eligible (has segment 2)
```

### Example 3: Birthday Special

```json
Promotion:
{
  "name": "Birthday Month 15% Off",
  "eligible_for": "specific_segment",
  "customer_segment_id": 5,  // "Birthday Month" segment
  "discount_config": {
    "percentage": 15
  }
}

Customer:
{
  "id": 789,
  "birthday": "1990-02-15",
  "current_month": "February"
}

Segments Assigned: [5]  // Auto-assigned in February

Result: ✅ Eligible
```

---

## 🔧 Segment Assignment Strategies

### 1. **Automatic (Rule-Based)**
```go
// Triggered after order completion
func AfterOrderComplete(customerID uint) {
    customerService.UpdateCustomerSegments(ctx, customerID)
}
```

### 2. **Manual (Admin Assignment)**
```go
// Admin manually adds customer to segment
func AssignCustomerToSegment(customerID, segmentID uint) {
    segmentMemberRepo.AssignToSegment(ctx, customerID, segmentID)
}
```

### 3. **Scheduled (Cron Job)**
```go
// Daily job to update segments
func DailySegmentUpdate() {
    // Update "Inactive" segment
    inactiveCustomers := GetCustomersWithNoOrdersInDays(90)
    for _, customerID := range inactiveCustomers {
        AssignToSegment(customerID, 4) // Inactive segment
    }
    
    // Update "Birthday Month" segment
    birthdayCustomers := GetCustomersWithBirthdayThisMonth()
    for _, customerID := range birthdayCustomers {
        AssignToSegment(customerID, 5) // Birthday segment
    }
}
```

---

## ✅ Summary

### Responsibilities

| Service | Responsibility |
|---------|---------------|
| **Customer Service** | Manage segments, assign customers to segments |
| **Promotion Service** | Validate if customer's segments match promotion requirement |
| **Cart Service** | Just provide cart items and totals |
| **Checkout Handler** | Fetch customer segments and pass to promotion validation |

### Benefits

✅ **No hardcoded customer types** - All managed via segments  
✅ **Flexible** - Create unlimited segment types  
✅ **Automatic** - Segments update based on customer behavior  
✅ **Reusable** - Same segments for promotions, emails, analytics  
✅ **Business-friendly** - Marketing team can create/manage segments  

### Migration Path

Instead of:
- ❌ `new_customers` (hardcoded)
- ❌ `returning_customers` (hardcoded)

Use:
- ✅ Segment ID 1: "New Customers" (0 orders)
- ✅ Segment ID 3: "Loyal Customers" (10+ orders)
- ✅ Any custom segment you create!
