# 📋 Subscription Service - Product Requirements Document (PRD)

> **Version**: 1.0  
> **Last Updated**: January 3, 2026  
> **Status**: Draft  
> **Author**: Development Team

---

## 📑 Table of Contents

1. [Overview](#overview)
2. [Service Architecture](#service-architecture)
3. [Plan Architecture](#plan-architecture)
4. [Subscription Management](#subscription-management)
5. [Usage Tracking](#usage-tracking)
6. [Admin APIs](#admin-apis)
7. [Data Models](#data-models)
8. [API Specifications](#api-specifications)
9. [Business Rules](#business-rules)
10. [Implementation Priority](#implementation-priority)
11. [Database Migration](#database-migration)

---

## 🎯 Overview

### Purpose

The Subscription Service is a dedicated module responsible for managing:

- Plan definitions and pricing models
- Seller subscriptions and lifecycle
- Usage tracking and limit enforcement
- Admin tools for plan and subscription management

### Why a Separate Service?

Subscription management is a **cross-cutting concern** that:

1. Every API endpoint needs to check limits (middleware)
2. Multiple services depend on subscription status (Product, Order, Inventory)
3. Complex business logic (trials, upgrades, downgrades, grace periods)
4. Separate scaling requirements for limit checks (high-frequency reads)

### Scope

| In Scope                                      | Out of Scope                                    |
| --------------------------------------------- | ----------------------------------------------- |
| Plan definitions (fixed, usage-based, tiered) | Payment processing (handled by Payment Service) |
| Subscription lifecycle management             | User authentication (handled by User Service)   |
| Usage tracking per metric                     | Billing invoices generation                     |
| Limit enforcement middleware                  | Complex proration calculations                  |
| Admin plan & subscription management          | Multi-currency pricing (Phase 2)                |
| Grace period handling                         | Coupon/discount system (Phase 2)                |

### Goals

1. Enable flexible plan configuration for SaaS model
2. Track seller usage against plan limits
3. Provide fast limit checks for middleware (Redis cached)
4. Support plan upgrades/downgrades with proper transitions
5. Enable admin control over subscriptions

---

## 🏗️ Service Architecture

### Service Dependencies

```
┌─────────────────────────────────────────────────────────────────────┐
│                        OTHER SERVICES                                │
│  (Product, Order, Inventory, etc.)                                  │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │              LIMIT CHECK MIDDLEWARE                          │   │
│  │  • Called on every mutating API (create product, order, etc)│   │
│  │  • Reads from Redis cache (60s TTL)                         │   │
│  │  • Falls back to Subscription Service API                   │   │
│  └──────────────────────────┬──────────────────────────────────┘   │
└─────────────────────────────┼───────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    SUBSCRIPTION SERVICE                              │
│                                                                      │
│  ┌──────────────────┐  ┌────────────────────┐  ┌────────────────┐  │
│  │   Plan Module    │  │ Subscription Module│  │  Usage Module  │  │
│  │                  │  │                    │  │                │  │
│  │  • List plans    │  │  • Subscribe       │  │  • Track usage │  │
│  │  • Plan details  │  │  • Upgrade/Down    │  │  • Check limits│  │
│  │  • Admin CRUD    │  │  • Cancel          │  │  • Log events  │  │
│  │  • Pricing logic │  │  • Grace period    │  │  • Reset cycle │  │
│  └────────┬─────────┘  └─────────┬──────────┘  └───────┬────────┘  │
│           │                      │                     │            │
│           └──────────────────────┴─────────────────────┘            │
│                                  │                                   │
│                          ┌───────▼───────┐                          │
│                          │  Repository   │                          │
│                          │    Layer      │                          │
│                          └───────┬───────┘                          │
└──────────────────────────────────┼──────────────────────────────────┘
                                   │
                    ┌──────────────┼──────────────┐
                    │              │              │
               ┌────▼────┐   ┌────▼─────┐  ┌────▼────┐
               │  Redis  │   │PostgreSQL│  │ Payment │
               │  Cache  │   │ Database │  │ Service │
               └─────────┘   └──────────┘  └─────────┘
```

### Caching Strategy

```
┌─────────────────────────────────────────────────────────────────┐
│                    REDIS CACHE STRUCTURE                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  seller:{id}:limits                                             │
│  ├── max_products: 100                                          │
│  ├── max_orders_per_month: 500                                  │
│  ├── max_staff: 2                                               │
│  ├── has_api_access: false                                      │
│  └── TTL: 60 seconds                                            │
│                                                                  │
│  seller:{id}:usage:{period}                                     │
│  ├── products: 45                                               │
│  ├── orders: 123                                                │
│  ├── staff: 2                                                   │
│  └── TTL: 300 seconds                                           │
│                                                                  │
│  seller:{id}:subscription                                       │
│  ├── plan_slug: "pro"                                           │
│  ├── status: "active"                                           │
│  ├── ends_at: "2026-02-01T00:00:00Z"                           │
│  └── TTL: 60 seconds                                            │
│                                                                  │
│  plans:list (all active plans)                                  │
│  └── TTL: 3600 seconds (1 hour)                                 │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📦 Plan Architecture

### 3.1 Plan Types Overview

The system supports multiple plan types to accommodate different pricing strategies:

| Plan Type      | Description                          | Use Case                 |
| -------------- | ------------------------------------ | ------------------------ |
| **Fixed**      | Flat monthly/yearly fee              | Standard SaaS pricing    |
| **Usage**      | Pay per unit (orders, API calls)     | Transactional businesses |
| **Tiered**     | Different rates at volume thresholds | Volume discounts         |
| **Hybrid**     | Base fee + usage charges             | Mixed pricing models     |
| **Enterprise** | Custom pricing, contact sales        | Large customers          |

### 3.2 Plan Entity Design

```go
type PlanType string

const (
    PlanTypeFixed      PlanType = "fixed"
    PlanTypeUsage      PlanType = "usage"
    PlanTypeTiered     PlanType = "tiered"
    PlanTypeHybrid     PlanType = "hybrid"
    PlanTypeEnterprise PlanType = "enterprise"
)

type BillingCycle string

const (
    BillingMonthly  BillingCycle = "monthly"
    BillingYearly   BillingCycle = "yearly"
    BillingLifetime BillingCycle = "lifetime"
    BillingCustom   BillingCycle = "custom"
)

type Plan struct {
    db.BaseEntity

    // Basic Info
    Name        string `json:"name"        gorm:"size:100;not null"`
    Slug        string `json:"slug"        gorm:"size:100;uniqueIndex"`
    Description string `json:"description" gorm:"type:text"`
    ShortDesc   string `json:"shortDesc"   gorm:"size:200"`

    // Pricing Type
    PlanType PlanType `json:"planType" gorm:"size:20;default:'fixed'"`

    // Pricing - Fixed
    BasePrice   float64 `json:"basePrice"   gorm:"type:decimal(10,2)"`
    YearlyPrice float64 `json:"yearlyPrice" gorm:"type:decimal(10,2)"`
    Currency    string  `json:"currency"    gorm:"size:3;default:'USD'"`

    // Pricing - Usage Based
    UsageMetric  string  `json:"usageMetric"  gorm:"size:50"`  // "orders", "products", "api_calls"
    IncludedUnits int    `json:"includedUnits" gorm:"default:0"`
    OveragePrice float64 `json:"overagePrice" gorm:"type:decimal(10,4)"`
    UsageCap     *int    `json:"usageCap"`  // NULL = unlimited

    // Pricing - Tiered
    HasTiers bool `json:"hasTiers" gorm:"default:false"`

    // Trial & Billing
    BillingCycle BillingCycle `json:"billingCycle" gorm:"size:20;default:'monthly'"`
    TrialDays    int          `json:"trialDays"    gorm:"default:0"`

    // Display
    IsPopular bool   `json:"isPopular" gorm:"default:false"`
    IsActive  bool   `json:"isActive"  gorm:"default:true"`
    SortOrder int    `json:"sortOrder" gorm:"default:0"`
    BadgeText string `json:"badgeText" gorm:"size:50"` // "Most Popular", "Best Value"

    // Relationships
    Features []PlanFeature `json:"features" gorm:"foreignKey:PlanID"`
    Limits   []PlanLimit   `json:"limits"   gorm:"foreignKey:PlanID"`
    Tiers    []PlanTier    `json:"tiers"    gorm:"foreignKey:PlanID"`
}
```

### 3.3 Plan Limits (Feature Gating)

```go
type LimitType string

const (
    LimitTypeCount   LimitType = "count"   // Numeric limit (max_products: 100)
    LimitTypeBoolean LimitType = "boolean" // Feature flag (has_api_access: true/false)
    LimitTypeTier    LimitType = "tier"    // Tiered value (support_level: "basic"|"priority"|"dedicated")
)

type PlanLimit struct {
    db.BaseEntity

    PlanID       uint      `json:"planId"       gorm:"index;not null"`
    LimitKey     string    `json:"limitKey"     gorm:"size:50;not null"` // "max_products", "has_api_access"
    LimitName    string    `json:"limitName"    gorm:"size:100;not null"` // Display name
    LimitType    LimitType `json:"limitType"    gorm:"size:20;not null"`

    // Values (use appropriate one based on LimitType)
    NumericValue *int    `json:"numericValue"` // -1 = unlimited
    BooleanValue *bool   `json:"booleanValue"`
    StringValue  *string `json:"stringValue"`  // For tier values

    // Display
    DisplayValue string `json:"displayValue" gorm:"size:50"` // "Unlimited", "10", "Yes"
    IsHighlight  bool   `json:"isHighlight"  gorm:"default:false"`
    SortOrder    int    `json:"sortOrder"    gorm:"default:0"`
}
```

**Standard Limit Keys:**

| Key                       | Type    | Description                 |
| ------------------------- | ------- | --------------------------- |
| `max_products`            | count   | Maximum products in catalog |
| `max_orders_per_month`    | count   | Monthly order limit         |
| `max_staff`               | count   | Staff account limit         |
| `max_locations`           | count   | Inventory locations         |
| `max_api_calls_per_day`   | count   | API rate limit              |
| `has_api_access`          | boolean | Can use API                 |
| `has_custom_reports`      | boolean | Custom report builder       |
| `has_multi_currency`      | boolean | Multi-currency support      |
| `support_level`           | tier    | basic/priority/dedicated    |
| `transaction_fee_percent` | count   | Platform transaction fee    |

### 3.4 Plan Tiers (Volume Pricing)

```go
type PlanTier struct {
    db.BaseEntity

    PlanID       uint    `json:"planId"       gorm:"index;not null"`
    TierName     string  `json:"tierName"     gorm:"size:50"`
    MinUnits     int     `json:"minUnits"     gorm:"not null"`
    MaxUnits     *int    `json:"maxUnits"`    // NULL = unlimited
    Price        float64 `json:"price"        gorm:"type:decimal(10,2);not null"`
    PricePerUnit float64 `json:"pricePerUnit" gorm:"type:decimal(10,4)"`
    Description  string  `json:"description"  gorm:"size:200"`
    SortOrder    int     `json:"sortOrder"    gorm:"default:0"`
}
```

### 3.5 Plan Features (Marketing Display)

```go
type PlanFeature struct {
    db.BaseEntity

    PlanID      uint   `json:"planId"      gorm:"index;not null"`
    FeatureKey  string `json:"featureKey"  gorm:"size:100;not null"`
    FeatureText string `json:"featureText" gorm:"size:200;not null"`
    Category    string `json:"category"    gorm:"size:50"` // "Sales", "Support", "Analytics"
    IsHighlight bool   `json:"isHighlight" gorm:"default:false"`
    IsIncluded  bool   `json:"isIncluded"  gorm:"default:true"` // false = shows as "X" in comparison
    SortOrder   int    `json:"sortOrder"   gorm:"default:0"`
    IconName    string `json:"iconName"    gorm:"size:50"` // For UI icons
}
```

### 3.6 Plan APIs

#### List Plans (Public)

**Endpoint**: `GET /api/subscription/plans`

**Response**:

```json
{
  "success": true,
  "data": {
    "plans": [
      {
        "id": 1,
        "slug": "basic",
        "name": "Basic",
        "shortDesc": "For solo entrepreneurs",
        "description": "Everything you need to start selling online",
        "planType": "fixed",
        "basePrice": 29.0,
        "yearlyPrice": 290.0,
        "currency": "USD",
        "billingCycle": "monthly",
        "trialDays": 14,
        "isPopular": false,
        "badgeText": null,
        "limits": [
          {
            "limitKey": "max_products",
            "limitName": "Products",
            "displayValue": "Unlimited",
            "isHighlight": true
          },
          {
            "limitKey": "max_orders_per_month",
            "limitName": "Orders/month",
            "displayValue": "500",
            "isHighlight": true
          }
        ],
        "features": [
          {
            "featureText": "Online store",
            "category": "Sales",
            "isIncluded": true
          }
        ]
      }
    ]
  }
}
```

#### Get Plan Details

**Endpoint**: `GET /api/subscription/plans/:slug`

**Response**: Full plan with all limits, features, and tiers

---

## 📊 Subscription Management

### 4.1 Subscription Entity

```go
type SubscriptionStatus string

const (
    StatusTrialing       SubscriptionStatus = "trialing"
    StatusActive         SubscriptionStatus = "active"
    StatusPastDue        SubscriptionStatus = "past_due"
    StatusCancelled      SubscriptionStatus = "cancelled"
    StatusExpired        SubscriptionStatus = "expired"
    StatusPaused         SubscriptionStatus = "paused"
    StatusGracePeriod    SubscriptionStatus = "grace_period"
)

type Subscription struct {
    db.BaseEntity

    // Core
    SellerID uint               `json:"sellerId" gorm:"uniqueIndex;not null"`
    PlanID   uint               `json:"planId"   gorm:"index;not null"`
    PlanSlug string             `json:"planSlug" gorm:"size:100"`
    Status   SubscriptionStatus `json:"status"   gorm:"size:20;not null"`

    // Dates
    StartDate    time.Time  `json:"startDate"    gorm:"not null"`
    EndDate      *time.Time `json:"endDate"`
    TrialEndDate *time.Time `json:"trialEndDate"`

    // Billing
    BillingCycle       BillingCycle `json:"billingCycle"       gorm:"size:20"`
    CurrentPeriodStart time.Time    `json:"currentPeriodStart"`
    CurrentPeriodEnd   time.Time    `json:"currentPeriodEnd"`
    NextBillingDate    *time.Time   `json:"nextBillingDate"`

    // Pricing
    BaseAmount      float64 `json:"baseAmount"      gorm:"type:decimal(10,2)"`
    DiscountPercent float64 `json:"discountPercent" gorm:"type:decimal(5,2);default:0"`
    Currency        string  `json:"currency"        gorm:"size:3;default:'USD'"`

    // Cancellation
    AutoRenew         bool       `json:"autoRenew"         gorm:"default:true"`
    CancelAtPeriodEnd bool       `json:"cancelAtPeriodEnd" gorm:"default:false"`
    CancelledAt       *time.Time `json:"cancelledAt"`
    GracePeriodEnd    *time.Time `json:"gracePeriodEnd"`
    CancelReason      string     `json:"cancelReason"      gorm:"size:500"`
    CancelFeedback    string     `json:"cancelFeedback"    gorm:"type:text"`

    // Payment
    PaymentRetryCount int    `json:"paymentRetryCount" gorm:"default:0"`
    ExternalSubID     string `json:"externalSubId"     gorm:"size:100"`
    PaymentMethodID   string `json:"paymentMethodId"   gorm:"size:100"`

    // Admin
    AdminGrantedBy   *uint  `json:"adminGrantedBy"`
    AdminGrantReason string `json:"adminGrantReason" gorm:"size:500"`

    // Relationships
    Plan   Plan           `json:"plan"   gorm:"foreignKey:PlanID"`
    Seller entity.User    `json:"-"      gorm:"foreignKey:SellerID"`
    Usage  []SubscriptionUsage `json:"usage" gorm:"foreignKey:SubscriptionID"`
}
```

### 4.2 Subscription Lifecycle

```
                    ┌─────────────────┐
                    │   Registration   │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
            ┌───────│    TRIALING     │◄──────────┐
            │       └────────┬────────┘           │
            │                │                     │
            │    Trial End   │   ┌────────────────┘
            │                ▼   │    Trial Extended (Admin)
            │       ┌─────────────────┐
            │       │  Payment Due    │
            │       └────────┬────────┘
            │                │
            │    ┌───────────┴───────────┐
            │    │                       │
            │    ▼                       ▼
            │ ┌──────────┐        ┌─────────────┐
            │ │  ACTIVE  │◄──────►│  PAST_DUE   │
            │ └────┬─────┘        └──────┬──────┘
            │      │                     │
            │      │                     │ Max retries
            │      │                     ▼
            │      │              ┌─────────────────┐
            │      │              │  GRACE_PERIOD   │
            │      │              └────────┬────────┘
            │      │                       │
            │      │    ┌─────────────────┴─────────────────┐
            │      │    │                                   │
            │      │    ▼                                   ▼
            │      │ ┌──────────┐    Payment Recovered   ┌─────────┐
            │      │ │ EXPIRED  │ ─────────────────────► │ ACTIVE  │
            │      │ └──────────┘                        └─────────┘
            │      │
            │      │  Cancel Request
            │      ▼
            │ ┌─────────────────┐
            └─│   CANCELLED     │
              └─────────────────┘
```

### 4.3 Subscription APIs

#### Get Current Subscription

**Endpoint**: `GET /api/subscription/current`

**Auth**: Seller JWT

**Response**:

```json
{
  "success": true,
  "data": {
    "subscription": {
      "id": 1,
      "status": "active",
      "plan": {
        "slug": "pro",
        "name": "Pro",
        "basePrice": 79.0
      },
      "billingCycle": "monthly",
      "currentPeriodStart": "2026-01-01T00:00:00Z",
      "currentPeriodEnd": "2026-02-01T00:00:00Z",
      "nextBillingDate": "2026-02-01T00:00:00Z",
      "autoRenew": true
    },
    "limits": {
      "max_products": { "limit": -1, "used": 45, "remaining": "unlimited" },
      "max_orders_per_month": { "limit": 2000, "used": 123, "remaining": 1877 },
      "max_staff": { "limit": 5, "used": 2, "remaining": 3 },
      "has_api_access": true
    }
  }
}
```

#### Subscribe to Plan

**Endpoint**: `POST /api/subscription/subscribe`

**Request**:

```json
{
  "planSlug": "pro",
  "billingCycle": "monthly",
  "paymentMethodId": "pm_xxx"
}
```

#### Change Subscription

**Endpoint**: `PUT /api/subscription/change`

**Request**:

```json
{
  "planSlug": "enterprise",
  "billingCycle": "yearly",
  "effectiveDate": "immediate"
}
```

**Rules**:

- **Upgrade**: Immediate effect, prorated charges
- **Downgrade**: Effective at end of billing period

#### Cancel Subscription

**Endpoint**: `POST /api/subscription/cancel`

**Request**:

```json
{
  "cancelAtPeriodEnd": true,
  "reason": "too_expensive",
  "feedback": "Looking for a more affordable option"
}
```

---

## 📈 Usage Tracking

### 5.1 Usage Entity

```go
type SubscriptionUsage struct {
    db.BaseEntity

    SubscriptionID uint   `json:"subscriptionId" gorm:"index;not null"`
    SellerID       uint   `json:"sellerId"       gorm:"index;not null"`
    PeriodStart    time.Time `json:"periodStart" gorm:"not null"`
    PeriodEnd      time.Time `json:"periodEnd"   gorm:"not null"`

    // Metric
    MetricType    string `json:"metricType"    gorm:"size:30;not null"` // "orders", "products", "staff"
    UsedCount     int    `json:"usedCount"     gorm:"not null;default:0"`
    IncludedLimit int    `json:"includedLimit" gorm:"not null"` // -1 = unlimited

    // Overage (for usage-based plans)
    OverageCount  int     `json:"overageCount"  gorm:"default:0"`
    OverageAmount float64 `json:"overageAmount" gorm:"type:decimal(10,2);default:0"`
    OverageRate   float64 `json:"overageRate"   gorm:"type:decimal(10,4);default:0"`

    LastUpdated time.Time `json:"lastUpdated" gorm:"not null"`
}
```

### 5.2 Usage Log Entity

```go
type SubscriptionUsageLog struct {
    db.BaseEntity

    SubscriptionID uint   `json:"subscriptionId" gorm:"index;not null"`
    SellerID       uint   `json:"sellerId"       gorm:"index;not null"`
    MetricType     string `json:"metricType"     gorm:"size:30;not null"`
    Delta          int    `json:"delta"          gorm:"not null"` // +1 for add, -1 for remove
    ReferenceID    string `json:"referenceId"    gorm:"size:100"` // Product ID, Order ID
    ReferenceType  string `json:"referenceType"  gorm:"size:50"` // "product", "order"
    Timestamp      time.Time `json:"timestamp"   gorm:"not null"`
}
```

### 5.3 Usage Tracking Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                    USAGE INCREMENT FLOW                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  1. Product Service calls: POST /api/products                       │
│                                                                      │
│  2. Middleware intercepts:                                          │
│     ┌────────────────────────────────────────────────────────────┐ │
│     │  CheckLimit("max_products", sellerID)                      │ │
│     │    ├── Read from Redis: seller:{id}:limits                 │ │
│     │    ├── If cache miss → Query SubscriptionService          │ │
│     │    ├── Compare: current_usage < limit                      │ │
│     │    └── Return: allow/deny                                  │ │
│     └────────────────────────────────────────────────────────────┘ │
│                                                                      │
│  3. If allowed, Product Service creates product                     │
│                                                                      │
│  4. Product Service calls Subscription Service:                     │
│     ┌────────────────────────────────────────────────────────────┐ │
│     │  IncrementUsage(sellerID, "products", productID)           │ │
│     │    ├── Update subscription_usage.used_count += 1           │ │
│     │    ├── Insert subscription_usage_log record                │ │
│     │    └── Update Redis cache: seller:{id}:usage              │ │
│     └────────────────────────────────────────────────────────────┘ │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 5.4 Limit Check Service Interface

```go
type LimitCheckService interface {
    // Check if seller can perform action (returns error if limit reached)
    CheckLimit(ctx context.Context, sellerID uint, limitKey string) error

    // Check with specific increment (for bulk operations)
    CheckLimitWithIncrement(ctx context.Context, sellerID uint, limitKey string, increment int) error

    // Get current usage for a metric
    GetUsage(ctx context.Context, sellerID uint, metricType string) (*UsageInfo, error)

    // Get all limits and usage for seller
    GetAllLimitsAndUsage(ctx context.Context, sellerID uint) (*LimitsAndUsage, error)

    // Increment usage after successful operation
    IncrementUsage(ctx context.Context, sellerID uint, metricType string, delta int, referenceID string, referenceType string) error

    // Decrement usage (on delete operations)
    DecrementUsage(ctx context.Context, sellerID uint, metricType string, delta int, referenceID string, referenceType string) error
}

type UsageInfo struct {
    MetricType    string `json:"metricType"`
    UsedCount     int    `json:"usedCount"`
    IncludedLimit int    `json:"includedLimit"` // -1 = unlimited
    Remaining     int    `json:"remaining"`     // -1 = unlimited
    OverageCount  int    `json:"overageCount"`
}

type LimitsAndUsage struct {
    SellerID     uint                  `json:"sellerId"`
    PlanSlug     string                `json:"planSlug"`
    Status       SubscriptionStatus    `json:"status"`
    Limits       map[string]LimitInfo  `json:"limits"`
    Usage        map[string]UsageInfo  `json:"usage"`
    CachedAt     time.Time             `json:"cachedAt"`
}
```

---

## 👨‍💼 Admin APIs

### 6.1 Admin Plan Management

**Endpoints**:

```
GET    /api/subscription/admin/plans           - List all plans (including inactive)
POST   /api/subscription/admin/plans           - Create new plan
GET    /api/subscription/admin/plans/:id       - Get plan details
PUT    /api/subscription/admin/plans/:id       - Update plan
DELETE /api/subscription/admin/plans/:id       - Soft delete (set inactive)
```

**Create/Update Plan Request**:

```json
{
  "name": "Enterprise",
  "slug": "enterprise",
  "description": "For large businesses",
  "shortDesc": "Custom solutions",
  "planType": "enterprise",
  "basePrice": 0,
  "currency": "USD",
  "billingCycle": "custom",
  "trialDays": 30,
  "isPopular": false,
  "isActive": true,
  "sortOrder": 4,
  "badgeText": "Custom Pricing",
  "limits": [
    {
      "limitKey": "max_products",
      "limitName": "Products",
      "limitType": "count",
      "numericValue": -1,
      "displayValue": "Unlimited",
      "isHighlight": true
    }
  ],
  "features": [
    {
      "featureKey": "dedicated_support",
      "featureText": "Dedicated Account Manager",
      "category": "Support",
      "isHighlight": true
    }
  ]
}
```

### 6.2 Admin Subscription Management

**Endpoints**:

```
GET  /api/subscription/admin/subscriptions                    - List all subscriptions
GET  /api/subscription/admin/subscriptions/:id                - Get subscription details
PUT  /api/subscription/admin/subscriptions/:id/extend         - Extend subscription
POST /api/subscription/admin/subscriptions/:sellerId/grant    - Grant subscription (free)
PUT  /api/subscription/admin/subscriptions/:id/status         - Change status manually
```

**Grant Subscription** (for partnerships, compensation):

```json
{
  "planSlug": "pro",
  "durationMonths": 6,
  "reason": "Partner program",
  "adminNotes": "As per agreement dated 2025-12-15"
}
```

**Extend Subscription**:

```json
{
  "extensionDays": 30,
  "reason": "Service compensation",
  "adminNotes": "Compensation for downtime on 2025-12-20"
}
```

### 6.3 Admin Usage & Analytics

**Endpoints**:

```
GET /api/subscription/admin/analytics/overview        - Subscription metrics
GET /api/subscription/admin/analytics/usage/:sellerId - Seller usage details
GET /api/subscription/admin/analytics/revenue         - Revenue by plan
```

**Overview Response**:

```json
{
  "success": true,
  "data": {
    "totalSubscriptions": 1500,
    "byStatus": {
      "active": 1200,
      "trialing": 150,
      "cancelled": 100,
      "expired": 50
    },
    "byPlan": {
      "basic": 600,
      "pro": 500,
      "enterprise": 100
    },
    "mrr": 89500.0,
    "churnRate": 4.2
  }
}
```

---

## 📐 Data Models

### Entity Summary

| Entity                 | Purpose                            |
| ---------------------- | ---------------------------------- |
| `Plan`                 | Plan definitions and pricing       |
| `PlanLimit`            | Feature limits per plan            |
| `PlanTier`             | Volume pricing tiers               |
| `PlanFeature`          | Marketing features for display     |
| `Subscription`         | Seller's active subscription       |
| `SubscriptionUsage`    | Usage tracking per billing period  |
| `SubscriptionUsageLog` | Audit log for usage events         |
| `SubscriptionHistory`  | Audit log for subscription changes |

### Subscription History Entity

```go
type SubscriptionHistory struct {
    db.BaseEntity

    SubscriptionID  uint   `json:"subscriptionId"  gorm:"index;not null"`
    SellerID        uint   `json:"sellerId"        gorm:"index;not null"`
    Action          string `json:"action"          gorm:"size:30;not null"` // created, upgraded, downgraded, cancelled, renewed, etc.
    FromPlanID      *uint  `json:"fromPlanId"`
    ToPlanID        *uint  `json:"toPlanId"`
    FromStatus      string `json:"fromStatus"      gorm:"size:20"`
    ToStatus        string `json:"toStatus"        gorm:"size:20"`
    Amount          float64 `json:"amount"         gorm:"type:decimal(10,2)"`
    Currency        string `json:"currency"        gorm:"size:3"`
    TransactionID   string `json:"transactionId"   gorm:"size:100"`
    PerformedBy     *uint  `json:"performedBy"`
    PerformedByRole string `json:"performedByRole" gorm:"size:20"` // seller, admin, system
    Reason          string `json:"reason"          gorm:"size:500"`
    Notes           string `json:"notes"           gorm:"type:text"`
    Metadata        JSON   `json:"metadata"        gorm:"type:jsonb"`
}
```

---

## 🛣️ API Specifications

### Route Summary

```
/api/subscription/
│
├── plans/
│   ├── GET    /                      - List active plans (public)
│   └── GET    /:slug                 - Get plan details (public)
│
├── (Seller - Auth Required)
│   ├── GET    /current               - Get current subscription with usage
│   ├── POST   /subscribe             - Subscribe to plan
│   ├── PUT    /change                - Change plan (upgrade/downgrade)
│   ├── POST   /cancel                - Cancel subscription
│   ├── GET    /history               - Subscription history
│   └── GET    /usage                 - Current usage breakdown
│
├── internal/ (Service-to-Service)
│   ├── GET    /limits/:sellerId      - Get seller limits (cached)
│   ├── POST   /usage/increment       - Increment usage
│   └── POST   /usage/decrement       - Decrement usage
│
└── admin/
    ├── plans/
    │   ├── GET    /                  - List all plans
    │   ├── POST   /                  - Create plan
    │   ├── GET    /:id               - Get plan details
    │   ├── PUT    /:id               - Update plan
    │   └── DELETE /:id               - Deactivate plan
    │
    ├── subscriptions/
    │   ├── GET    /                  - List all subscriptions
    │   ├── GET    /:id               - Get subscription details
    │   ├── PUT    /:id/extend        - Extend subscription
    │   ├── PUT    /:id/status        - Change status
    │   └── POST   /:sellerId/grant   - Grant free subscription
    │
    └── analytics/
        ├── GET    /overview          - Subscription metrics
        ├── GET    /usage/:sellerId   - Seller usage details
        └── GET    /revenue           - Revenue analytics
```

---

## 📏 Business Rules

### Plan Rules

| Rule     | Description                                       |
| -------- | ------------------------------------------------- |
| PLAN-001 | Plan slug must be unique                          |
| PLAN-002 | Cannot delete plan with active subscriptions      |
| PLAN-003 | Price changes don't affect existing subscriptions |
| PLAN-004 | Enterprise plans require custom pricing setup     |

### Subscription Rules

| Rule    | Description                                            |
| ------- | ------------------------------------------------------ |
| SUB-001 | One active subscription per seller at a time           |
| SUB-002 | Upgrade: Effective immediately with proration          |
| SUB-003 | Downgrade: Effective at end of billing cycle           |
| SUB-004 | Cancelled subscription remains active until period end |
| SUB-005 | Grace period: 3 days after payment failure             |
| SUB-006 | Expired sellers: Read-only access, cannot create new   |
| SUB-007 | Admin-granted subscriptions bypass payment             |

### Usage Rules

| Rule      | Description                               |
| --------- | ----------------------------------------- |
| USAGE-001 | Usage resets at billing period start      |
| USAGE-002 | Limit -1 means unlimited                  |
| USAGE-003 | Products count doesn't reset (cumulative) |
| USAGE-004 | Orders count resets monthly               |
| USAGE-005 | Overage charges calculated at period end  |

### Limit Enforcement

| Rule      | Description                                      |
| --------- | ------------------------------------------------ |
| LIMIT-001 | Check limits before allowing create operations   |
| LIMIT-002 | Cache limits in Redis with 60s TTL               |
| LIMIT-003 | Boolean limits checked directly (no counting)    |
| LIMIT-004 | Return 403 with clear message when limit reached |

---

## 🎯 Implementation Priority

### Phase 1 - Core Infrastructure (Week 1-2)

| Priority | Feature                  | Effort |
| -------- | ------------------------ | ------ |
| 🔴 P0    | Plan entity & repository | 2 days |
| 🔴 P0    | Plan limits & features   | 2 days |
| 🔴 P0    | List/Get plans API       | 1 day  |
| 🔴 P0    | Subscription entity      | 2 days |
| 🔴 P0    | Get current subscription | 1 day  |

### Phase 2 - Subscription Lifecycle (Week 3-4)

| Priority | Feature              | Effort |
| -------- | -------------------- | ------ |
| 🔴 P0    | Subscribe to plan    | 2 days |
| 🔴 P0    | Start free trial     | 1 day  |
| 🟡 P1    | Change subscription  | 2 days |
| 🟡 P1    | Cancel subscription  | 1 day  |
| 🟡 P1    | Subscription history | 1 day  |

### Phase 3 - Usage & Limits (Week 5-6)

| Priority | Feature                   | Effort |
| -------- | ------------------------- | ------ |
| 🔴 P0    | Usage tracking entity     | 2 days |
| 🔴 P0    | Limit check service       | 2 days |
| 🔴 P0    | Redis caching layer       | 2 days |
| 🟡 P1    | Limit check middleware    | 2 days |
| 🟡 P1    | Usage increment/decrement | 1 day  |

### Phase 4 - Admin & Scheduler (Week 7-8)

| Priority | Feature                       | Effort |
| -------- | ----------------------------- | ------ |
| 🟡 P1    | Admin plan CRUD               | 2 days |
| 🟡 P1    | Admin subscription management | 2 days |
| 🟡 P1    | Grant subscription            | 1 day  |
| 🟢 P2    | Trial expiry scheduler        | 1 day  |
| 🟢 P2    | Grace period scheduler        | 1 day  |
| 🟢 P2    | Analytics APIs                | 2 days |

---

## 📊 Database Migration

### Migration 009: Subscription Service Tables

```sql
-- migrations/009_subscription_service.sql
-- Description: Complete subscription service with plans, subscriptions, and usage tracking
-- Author: Development Team
-- Date: 2026-01-03

-- =============================================
-- 1. PLAN TABLE ENHANCEMENTS
-- =============================================

-- Add new columns to existing plan table (if exists) or create new
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'plan' AND column_name = 'slug') THEN
        ALTER TABLE plan ADD COLUMN slug VARCHAR(100);
    END IF;
END $$;

ALTER TABLE plan ADD COLUMN IF NOT EXISTS short_desc VARCHAR(200);
ALTER TABLE plan ADD COLUMN IF NOT EXISTS plan_type VARCHAR(20) DEFAULT 'fixed';
ALTER TABLE plan ADD COLUMN IF NOT EXISTS yearly_price DECIMAL(10,2);
ALTER TABLE plan ADD COLUMN IF NOT EXISTS usage_metric VARCHAR(50);
ALTER TABLE plan ADD COLUMN IF NOT EXISTS included_units INT DEFAULT 0;
ALTER TABLE plan ADD COLUMN IF NOT EXISTS overage_price DECIMAL(10,4);
ALTER TABLE plan ADD COLUMN IF NOT EXISTS usage_cap INT;
ALTER TABLE plan ADD COLUMN IF NOT EXISTS has_tiers BOOLEAN DEFAULT FALSE;
ALTER TABLE plan ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE;
ALTER TABLE plan ADD COLUMN IF NOT EXISTS badge_text VARCHAR(50);

-- Ensure slug is unique
CREATE UNIQUE INDEX IF NOT EXISTS idx_plan_slug ON plan(slug) WHERE deleted_at IS NULL;

-- =============================================
-- 2. PLAN LIMITS (Feature Gating)
-- =============================================

CREATE TABLE IF NOT EXISTS plan_limit (
    id BIGSERIAL PRIMARY KEY,
    plan_id BIGINT NOT NULL REFERENCES plan(id) ON DELETE CASCADE,
    limit_key VARCHAR(50) NOT NULL,
    limit_name VARCHAR(100) NOT NULL,
    limit_type VARCHAR(20) NOT NULL,
    numeric_value INT,
    boolean_value BOOLEAN,
    string_value VARCHAR(100),
    display_value VARCHAR(50),
    is_highlight BOOLEAN DEFAULT FALSE,
    sort_order INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_plan_limit_plan ON plan_limit(plan_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_plan_limit_unique ON plan_limit(plan_id, limit_key) WHERE deleted_at IS NULL;

-- =============================================
-- 3. PLAN TIERS (For Usage-Based Pricing)
-- =============================================

CREATE TABLE IF NOT EXISTS plan_tier (
    id BIGSERIAL PRIMARY KEY,
    plan_id BIGINT NOT NULL REFERENCES plan(id) ON DELETE CASCADE,
    tier_name VARCHAR(50),
    min_units INT NOT NULL,
    max_units INT,
    price DECIMAL(10,2) NOT NULL,
    price_per_unit DECIMAL(10,4),
    description VARCHAR(200),
    sort_order INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_plan_tier_plan ON plan_tier(plan_id);

-- =============================================
-- 4. PLAN FEATURES (Marketing Display)
-- =============================================

CREATE TABLE IF NOT EXISTS plan_feature (
    id BIGSERIAL PRIMARY KEY,
    plan_id BIGINT NOT NULL REFERENCES plan(id) ON DELETE CASCADE,
    feature_key VARCHAR(100) NOT NULL,
    feature_text VARCHAR(200) NOT NULL,
    category VARCHAR(50),
    is_highlight BOOLEAN DEFAULT FALSE,
    is_included BOOLEAN DEFAULT TRUE,
    sort_order INT DEFAULT 0,
    icon_name VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_plan_feature_plan ON plan_feature(plan_id);

-- =============================================
-- 5. SUBSCRIPTION ENHANCEMENTS
-- =============================================

ALTER TABLE subscription ADD COLUMN IF NOT EXISTS plan_slug VARCHAR(100);
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS billing_cycle VARCHAR(20) DEFAULT 'monthly';
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS current_period_start TIMESTAMPTZ;
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS current_period_end TIMESTAMPTZ;
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS next_billing_date TIMESTAMPTZ;
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS trial_end_date TIMESTAMPTZ;
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS base_amount DECIMAL(10,2);
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS discount_percent DECIMAL(5,2) DEFAULT 0;
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS currency VARCHAR(3) DEFAULT 'USD';
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS auto_renew BOOLEAN DEFAULT TRUE;
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS cancel_at_period_end BOOLEAN DEFAULT FALSE;
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMPTZ;
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS cancel_reason VARCHAR(500);
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS cancel_feedback TEXT;
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS grace_period_end TIMESTAMPTZ;
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS payment_retry_count INT DEFAULT 0;
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS admin_granted_by BIGINT REFERENCES "user"(id);
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS admin_grant_reason VARCHAR(500);
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS external_sub_id VARCHAR(100);
ALTER TABLE subscription ADD COLUMN IF NOT EXISTS payment_method_id VARCHAR(100);

CREATE INDEX IF NOT EXISTS idx_subscription_status ON subscription(status);
CREATE INDEX IF NOT EXISTS idx_subscription_external ON subscription(external_sub_id);

-- =============================================
-- 6. SUBSCRIPTION USAGE TRACKING
-- =============================================

CREATE TABLE IF NOT EXISTS subscription_usage (
    id BIGSERIAL PRIMARY KEY,
    subscription_id BIGINT NOT NULL REFERENCES subscription(id) ON DELETE CASCADE,
    seller_id BIGINT NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    metric_type VARCHAR(30) NOT NULL,
    used_count INT NOT NULL DEFAULT 0,
    included_limit INT NOT NULL,
    overage_count INT DEFAULT 0,
    overage_amount DECIMAL(10,2) DEFAULT 0,
    overage_rate DECIMAL(10,4) DEFAULT 0,
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_subscription_usage_sub ON subscription_usage(subscription_id);
CREATE INDEX IF NOT EXISTS idx_subscription_usage_seller ON subscription_usage(seller_id);
CREATE INDEX IF NOT EXISTS idx_subscription_usage_period ON subscription_usage(period_start, period_end);
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscription_usage_unique ON subscription_usage(subscription_id, metric_type, period_start) WHERE deleted_at IS NULL;

-- =============================================
-- 7. SUBSCRIPTION USAGE LOG (Audit)
-- =============================================

CREATE TABLE IF NOT EXISTS subscription_usage_log (
    id BIGSERIAL PRIMARY KEY,
    subscription_id BIGINT NOT NULL,
    seller_id BIGINT NOT NULL,
    metric_type VARCHAR(30) NOT NULL,
    delta INT NOT NULL,
    reference_id VARCHAR(100),
    reference_type VARCHAR(50),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_subscription_usage_log_sub ON subscription_usage_log(subscription_id);
CREATE INDEX IF NOT EXISTS idx_subscription_usage_log_seller ON subscription_usage_log(seller_id);
CREATE INDEX IF NOT EXISTS idx_subscription_usage_log_time ON subscription_usage_log(timestamp);

-- =============================================
-- 8. SUBSCRIPTION HISTORY (Audit Log)
-- =============================================

CREATE TABLE IF NOT EXISTS subscription_history (
    id BIGSERIAL PRIMARY KEY,
    subscription_id BIGINT NOT NULL REFERENCES subscription(id) ON DELETE CASCADE,
    seller_id BIGINT NOT NULL,
    action VARCHAR(30) NOT NULL,
    from_plan_id BIGINT,
    to_plan_id BIGINT,
    from_status VARCHAR(20),
    to_status VARCHAR(20),
    amount DECIMAL(10,2),
    currency VARCHAR(3),
    transaction_id VARCHAR(100),
    performed_by BIGINT REFERENCES "user"(id),
    performed_by_role VARCHAR(20),
    reason VARCHAR(500),
    notes TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_subscription_history_sub ON subscription_history(subscription_id);
CREATE INDEX IF NOT EXISTS idx_subscription_history_seller ON subscription_history(seller_id);
CREATE INDEX IF NOT EXISTS idx_subscription_history_action ON subscription_history(action);
CREATE INDEX IF NOT EXISTS idx_subscription_history_created ON subscription_history(created_at);
```

### Seed Data

```sql
-- seeds/009_seed_plan_data.sql

-- Insert default plans
INSERT INTO plan (name, slug, description, short_desc, plan_type, price, yearly_price, currency, billing_cycle, trial_days, is_popular, is_active, sort_order, badge_text, created_at, updated_at)
VALUES
    ('Free Trial', 'free-trial', 'Try all features for 14 days', 'Perfect for getting started', 'fixed', 0, 0, 'USD', 'monthly', 14, false, true, 1, NULL, NOW(), NOW()),
    ('Basic', 'basic', 'Everything you need to start selling online', 'For solo entrepreneurs', 'fixed', 29.00, 290.00, 'USD', 'monthly', 14, false, true, 2, NULL, NOW(), NOW()),
    ('Pro', 'pro', 'Level up with professional features and more capacity', 'For growing businesses', 'fixed', 79.00, 790.00, 'USD', 'monthly', 14, true, true, 3, 'Most Popular', NOW(), NOW()),
    ('Enterprise', 'enterprise', 'Custom solutions for high-volume businesses', 'For large operations', 'enterprise', 0, 0, 'USD', 'custom', 30, false, true, 4, 'Custom Pricing', NOW(), NOW())
ON CONFLICT DO NOTHING;

-- Insert plan limits (example for Basic plan)
INSERT INTO plan_limit (plan_id, limit_key, limit_name, limit_type, numeric_value, display_value, is_highlight, sort_order, created_at, updated_at)
SELECT id, 'max_products', 'Products', 'count', -1, 'Unlimited', true, 1, NOW(), NOW() FROM plan WHERE slug = 'basic' AND NOT EXISTS (SELECT 1 FROM plan_limit WHERE plan_id = (SELECT id FROM plan WHERE slug = 'basic') AND limit_key = 'max_products')
UNION ALL
SELECT id, 'max_orders_per_month', 'Orders/month', 'count', 500, '500', true, 2, NOW(), NOW() FROM plan WHERE slug = 'basic' AND NOT EXISTS (SELECT 1 FROM plan_limit WHERE plan_id = (SELECT id FROM plan WHERE slug = 'basic') AND limit_key = 'max_orders_per_month')
UNION ALL
SELECT id, 'max_staff', 'Staff accounts', 'count', 2, '2', false, 3, NOW(), NOW() FROM plan WHERE slug = 'basic' AND NOT EXISTS (SELECT 1 FROM plan_limit WHERE plan_id = (SELECT id FROM plan WHERE slug = 'basic') AND limit_key = 'max_staff');
```

---

## ✅ Success Metrics

| Metric                                 | Target |
| -------------------------------------- | ------ |
| Subscription conversion (trial → paid) | > 15%  |
| Plan upgrade rate                      | > 10%  |
| Churn rate (monthly)                   | < 5%   |
| Limit check latency (p99)              | < 10ms |
| Cache hit rate                         | > 95%  |

---

## 📚 References

- [Architecture Documentation](../ARCHITECTURE.md)
- [Coding Standards](../CODING_STANDARDS.md)
- [User Service PRD](../user/USER_SERVICE_PRD.md)
- [Payment Module Design](../payment/PAYMENT_MODULE_DESIGN.md)
- [Plan & Subscription Examples](../user/PLAN_SUBSCRIPTION_EXAMPLES.md)
- [Shopify Pricing](https://www.shopify.com/pricing) - Reference for plan structure
- [Stripe Billing](https://stripe.com/docs/billing) - Reference for subscription lifecycle

---

**Document Status**: Ready for Review  
**Next Steps**:

1. Review with team
2. Create subscription folder structure
3. Implement Phase 1 (Core Infrastructure)
4. Integrate limit check middleware with other services
