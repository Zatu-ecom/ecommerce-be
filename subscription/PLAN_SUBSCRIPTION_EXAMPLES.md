# 📊 Plan & Subscription System - Data Examples

> This document provides concrete examples to understand how the plan and subscription system works together.

---

## 🗂️ Table of Contents

1. [Entity Relationship Overview](#entity-relationship-overview)
2. [Complete Plan Example](#complete-plan-example)
3. [Subscription Lifecycle Example](#subscription-lifecycle-example)
4. [Usage Tracking Example](#usage-tracking-example)
5. [API Response Examples](#api-response-examples)
6. [Real-World Scenarios](#real-world-scenarios)

---

## 🔗 Entity Relationship Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              PLAN SYSTEM                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌──────────────┐         ┌──────────────┐         ┌──────────────┐       │
│   │     PLAN     │────────▶│  PLAN_LIMIT  │         │  PLAN_TIER   │       │
│   │              │    1:N  │              │    1:N  │              │       │
│   │  id: 2       │         │  plan_id: 2  │◀────────│  plan_id: 2  │       │
│   │  slug: pro   │         │  key: max_   │         │  min: 0      │       │
│   │  price: $79  │         │     products │         │  max: 100    │       │
│   └──────┬───────┘         └──────────────┘         └──────────────┘       │
│          │                                                                   │
│          │ 1:N             ┌──────────────┐                                 │
│          └────────────────▶│ PLAN_FEATURE │                                 │
│                            │              │                                 │
│                            │  plan_id: 2  │                                 │
│                            │  key: api_   │                                 │
│                            │     access   │                                 │
│                            └──────────────┘                                 │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                           SUBSCRIPTION SYSTEM                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌──────────────┐         ┌──────────────┐         ┌──────────────┐       │
│   │ SUBSCRIPTION │────────▶│  SUB_USAGE   │────────▶│SUB_USAGE_LOG │       │
│   │              │    1:N  │              │    1:N  │              │       │
│   │  id: 100     │         │  sub_id: 100 │         │  sub_id: 100 │       │
│   │  seller: 5   │         │  orders: 150 │         │  +1 order    │       │
│   │  plan: pro   │         │  products: 45│         │  ref: ORD123 │       │
│   └──────┬───────┘         └──────────────┘         └──────────────┘       │
│          │                                                                   │
│          │ 1:N             ┌──────────────┐                                 │
│          └────────────────▶│ SUB_HISTORY  │                                 │
│                            │              │                                 │
│                            │  sub_id: 100 │                                 │
│                            │  action:     │                                 │
│                            │   upgraded   │                                 │
│                            └──────────────┘                                 │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 📦 Complete Plan Example

### The "Pro" Plan - Full Data Structure

#### 1. Plan (Main Record)

```json
{
  "id": 3,
  "name": "Pro",
  "slug": "pro",
  "description": "Level up with professional features and more capacity",
  "shortDesc": "For growing businesses",
  "planType": "fixed",
  "price": 79.0,
  "yearlyPrice": 790.0,
  "currency": "USD",
  "billingCycle": "monthly",
  "trialDays": 14,
  "usageMetric": null,
  "includedUnits": 0,
  "overagePrice": null,
  "usageCap": null,
  "hasTiers": false,
  "isPopular": true,
  "isActive": true,
  "badgeText": "Most Popular",
  "sortOrder": 3,
  "createdAt": "2026-01-01T00:00:00Z",
  "updatedAt": "2026-01-01T00:00:00Z"
}
```

#### 2. Plan Limits (Feature Gating - What seller CAN do)

```json
[
  {
    "id": 1,
    "planId": 3,
    "limitKey": "max_products",
    "limitName": "Products",
    "limitType": "count",
    "numericValue": -1,
    "booleanValue": null,
    "stringValue": null,
    "displayValue": "Unlimited",
    "isHighlight": true,
    "sortOrder": 1
  },
  {
    "id": 2,
    "planId": 3,
    "limitKey": "max_orders_per_month",
    "limitName": "Orders per month",
    "limitType": "count",
    "numericValue": 2000,
    "booleanValue": null,
    "stringValue": null,
    "displayValue": "2,000",
    "isHighlight": true,
    "sortOrder": 2
  },
  {
    "id": 3,
    "planId": 3,
    "limitKey": "max_staff",
    "limitName": "Staff accounts",
    "limitType": "count",
    "numericValue": 5,
    "booleanValue": null,
    "stringValue": null,
    "displayValue": "5",
    "isHighlight": false,
    "sortOrder": 3
  },
  {
    "id": 4,
    "planId": 3,
    "limitKey": "max_locations",
    "limitName": "Inventory locations",
    "limitType": "count",
    "numericValue": 10,
    "booleanValue": null,
    "stringValue": null,
    "displayValue": "10",
    "isHighlight": false,
    "sortOrder": 4
  },
  {
    "id": 5,
    "planId": 3,
    "limitKey": "transaction_fee_percent",
    "limitName": "Transaction fee",
    "limitType": "count",
    "numericValue": 1,
    "booleanValue": null,
    "stringValue": null,
    "displayValue": "1%",
    "isHighlight": false,
    "sortOrder": 5
  },
  {
    "id": 6,
    "planId": 3,
    "limitKey": "has_api_access",
    "limitName": "API Access",
    "limitType": "boolean",
    "numericValue": null,
    "booleanValue": true,
    "stringValue": null,
    "displayValue": "Yes",
    "isHighlight": true,
    "sortOrder": 10
  },
  {
    "id": 7,
    "planId": 3,
    "limitKey": "has_custom_reports",
    "limitName": "Custom reports",
    "limitType": "boolean",
    "numericValue": null,
    "booleanValue": true,
    "stringValue": null,
    "displayValue": "Yes",
    "isHighlight": false,
    "sortOrder": 11
  },
  {
    "id": 8,
    "planId": 3,
    "limitKey": "support_level",
    "limitName": "Support",
    "limitType": "tier",
    "numericValue": null,
    "booleanValue": null,
    "stringValue": "priority",
    "displayValue": "Priority Support",
    "isHighlight": false,
    "sortOrder": 20
  }
]
```

#### 3. Plan Features (Marketing Display - What to show on pricing page)

```json
[
  {
    "id": 1,
    "planId": 3,
    "featureKey": "online_store",
    "featureText": "Online store with unlimited bandwidth",
    "category": "Sales",
    "isHighlight": true,
    "isIncluded": true,
    "sortOrder": 1,
    "iconName": "store"
  },
  {
    "id": 2,
    "planId": 3,
    "featureKey": "unlimited_products",
    "featureText": "Unlimited products",
    "category": "Products",
    "isHighlight": true,
    "isIncluded": true,
    "sortOrder": 2,
    "iconName": "package"
  },
  {
    "id": 3,
    "planId": 3,
    "featureKey": "staff_accounts",
    "featureText": "5 staff accounts",
    "category": "Team",
    "isHighlight": false,
    "isIncluded": true,
    "sortOrder": 3,
    "iconName": "users"
  },
  {
    "id": 4,
    "planId": 3,
    "featureKey": "inventory_locations",
    "featureText": "Up to 10 inventory locations",
    "category": "Inventory",
    "isHighlight": false,
    "isIncluded": true,
    "sortOrder": 4,
    "iconName": "warehouse"
  },
  {
    "id": 5,
    "planId": 3,
    "featureKey": "api_access",
    "featureText": "Full API access",
    "category": "Developer",
    "isHighlight": true,
    "isIncluded": true,
    "sortOrder": 5,
    "iconName": "code"
  },
  {
    "id": 6,
    "planId": 3,
    "featureKey": "custom_reports",
    "featureText": "Custom report builder",
    "category": "Analytics",
    "isHighlight": false,
    "isIncluded": true,
    "sortOrder": 6,
    "iconName": "chart"
  },
  {
    "id": 7,
    "planId": 3,
    "featureKey": "priority_support",
    "featureText": "Priority 24/7 support",
    "category": "Support",
    "isHighlight": false,
    "isIncluded": true,
    "sortOrder": 10,
    "iconName": "headset"
  },
  {
    "id": 8,
    "planId": 3,
    "featureKey": "advanced_analytics",
    "featureText": "Advanced analytics",
    "category": "Analytics",
    "isHighlight": false,
    "isIncluded": true,
    "sortOrder": 11,
    "iconName": "analytics"
  }
]
```

---

## 🔄 Subscription Lifecycle Example

### Scenario: Seller "TechGadgets" Journey

#### Day 1: Seller Registers (Trial Starts)

**Subscription Created:**

```json
{
  "id": 100,
  "sellerId": 5,
  "planId": 1,
  "planSlug": "free-trial",
  "status": "trialing",
  "billingCycle": "monthly",
  "startDate": "2026-01-01T10:00:00Z",
  "endDate": null,
  "trialEndDate": "2026-01-15T10:00:00Z",
  "currentPeriodStart": "2026-01-01T10:00:00Z",
  "currentPeriodEnd": "2026-01-15T10:00:00Z",
  "nextBillingDate": "2026-01-15T10:00:00Z",
  "baseAmount": 0,
  "discountPercent": 0,
  "currency": "USD",
  "autoRenew": true,
  "cancelAtPeriodEnd": false,
  "cancelledAt": null,
  "cancelReason": null,
  "graceperiodEnd": null,
  "paymentRetryCount": 0,
  "externalSubId": null,
  "paymentMethodId": null,
  "createdAt": "2026-01-01T10:00:00Z"
}
```

**History Entry:**

```json
{
  "id": 1,
  "subscriptionId": 100,
  "sellerId": 5,
  "action": "created",
  "fromPlanId": null,
  "toPlanId": 1,
  "fromStatus": null,
  "toStatus": "trialing",
  "amount": 0,
  "currency": "USD",
  "transactionId": null,
  "performedBy": 5,
  "performedByRole": "seller",
  "reason": "New seller registration",
  "notes": null,
  "metadata": { "source": "web_signup" },
  "createdAt": "2026-01-01T10:00:00Z"
}
```

**Initial Usage Records:**

```json
[
  {
    "id": 1,
    "subscriptionId": 100,
    "sellerId": 5,
    "periodStart": "2026-01-01T10:00:00Z",
    "periodEnd": "2026-01-15T10:00:00Z",
    "metricType": "orders",
    "usedCount": 0,
    "includedLimit": 50,
    "overageCount": 0,
    "overageAmount": 0,
    "overageRate": 0
  },
  {
    "id": 2,
    "subscriptionId": 100,
    "sellerId": 5,
    "periodStart": "2026-01-01T10:00:00Z",
    "periodEnd": "2026-01-15T10:00:00Z",
    "metricType": "products",
    "usedCount": 0,
    "includedLimit": 10,
    "overageCount": 0,
    "overageAmount": 0,
    "overageRate": 0
  },
  {
    "id": 3,
    "subscriptionId": 100,
    "sellerId": 5,
    "periodStart": "2026-01-01T10:00:00Z",
    "periodEnd": "2026-01-15T10:00:00Z",
    "metricType": "staff",
    "usedCount": 1,
    "includedLimit": 1,
    "overageCount": 0,
    "overageAmount": 0,
    "overageRate": 0
  }
]
```

---

#### Day 10: Seller Upgrades to Pro (During Trial)

**Updated Subscription:**

```json
{
  "id": 100,
  "sellerId": 5,
  "planId": 3,
  "planSlug": "pro",
  "status": "active",
  "billingCycle": "monthly",
  "startDate": "2026-01-01T10:00:00Z",
  "endDate": null,
  "trialEndDate": null,
  "currentPeriodStart": "2026-01-10T14:30:00Z",
  "currentPeriodEnd": "2026-02-10T14:30:00Z",
  "nextBillingDate": "2026-02-10T14:30:00Z",
  "baseAmount": 79.0,
  "discountPercent": 0,
  "currency": "USD",
  "autoRenew": true,
  "cancelAtPeriodEnd": false,
  "externalSubId": "sub_stripe_abc123",
  "paymentMethodId": "pm_card_visa_4242",
  "createdAt": "2026-01-01T10:00:00Z",
  "updatedAt": "2026-01-10T14:30:00Z"
}
```

**New History Entry:**

```json
{
  "id": 2,
  "subscriptionId": 100,
  "sellerId": 5,
  "action": "upgraded",
  "fromPlanId": 1,
  "toPlanId": 3,
  "fromStatus": "trialing",
  "toStatus": "active",
  "amount": 79.0,
  "currency": "USD",
  "transactionId": "txn_stripe_xyz789",
  "performedBy": 5,
  "performedByRole": "seller",
  "reason": "Seller upgraded during trial",
  "notes": "Charged immediately, trial ended",
  "metadata": {
    "payment_method": "card",
    "card_last4": "4242",
    "proration_amount": 0
  },
  "createdAt": "2026-01-10T14:30:00Z"
}
```

**New Usage Records (Reset for new period):**

```json
[
  {
    "id": 4,
    "subscriptionId": 100,
    "sellerId": 5,
    "periodStart": "2026-01-10T14:30:00Z",
    "periodEnd": "2026-02-10T14:30:00Z",
    "metricType": "orders",
    "usedCount": 0,
    "includedLimit": 2000,
    "overageCount": 0,
    "overageAmount": 0,
    "overageRate": 0
  },
  {
    "id": 5,
    "subscriptionId": 100,
    "sellerId": 5,
    "periodStart": "2026-01-10T14:30:00Z",
    "periodEnd": "2026-02-10T14:30:00Z",
    "metricType": "products",
    "usedCount": 8,
    "includedLimit": -1,
    "overageCount": 0,
    "overageAmount": 0,
    "overageRate": 0
  },
  {
    "id": 6,
    "subscriptionId": 100,
    "sellerId": 5,
    "periodStart": "2026-01-10T14:30:00Z",
    "periodEnd": "2026-02-10T14:30:00Z",
    "metricType": "staff",
    "usedCount": 1,
    "includedLimit": 5,
    "overageCount": 0,
    "overageAmount": 0,
    "overageRate": 0
  }
]
```

---

#### Month 3: Seller Cancels (But Continues Until Period End)

**Updated Subscription:**

```json
{
  "id": 100,
  "sellerId": 5,
  "planId": 3,
  "planSlug": "pro",
  "status": "active",
  "billingCycle": "monthly",
  "currentPeriodStart": "2026-03-10T14:30:00Z",
  "currentPeriodEnd": "2026-04-10T14:30:00Z",
  "nextBillingDate": null,
  "baseAmount": 79.0,
  "autoRenew": false,
  "cancelAtPeriodEnd": true,
  "cancelledAt": "2026-03-25T09:15:00Z",
  "cancelReason": "Too expensive",
  "cancelFeedback": "Love the product but need to cut costs. Will come back when business picks up.",
  "graceperiodEnd": null,
  "updatedAt": "2026-03-25T09:15:00Z"
}
```

**History Entry:**

```json
{
  "id": 5,
  "subscriptionId": 100,
  "sellerId": 5,
  "action": "cancel_scheduled",
  "fromPlanId": 3,
  "toPlanId": 3,
  "fromStatus": "active",
  "toStatus": "active",
  "amount": null,
  "currency": "USD",
  "transactionId": null,
  "performedBy": 5,
  "performedByRole": "seller",
  "reason": "Too expensive",
  "notes": "Cancellation scheduled for end of billing period",
  "metadata": {
    "feedback": "Love the product but need to cut costs. Will come back when business picks up.",
    "effective_date": "2026-04-10T14:30:00Z"
  },
  "createdAt": "2026-03-25T09:15:00Z"
}
```

---

#### April 10: Subscription Expires (System Job)

**Final Subscription State:**

```json
{
  "id": 100,
  "sellerId": 5,
  "planId": 3,
  "planSlug": "pro",
  "status": "cancelled",
  "billingCycle": "monthly",
  "currentPeriodStart": "2026-03-10T14:30:00Z",
  "currentPeriodEnd": "2026-04-10T14:30:00Z",
  "nextBillingDate": null,
  "endDate": "2026-04-10T14:30:00Z",
  "autoRenew": false,
  "cancelAtPeriodEnd": true,
  "cancelledAt": "2026-03-25T09:15:00Z",
  "updatedAt": "2026-04-10T14:30:00Z"
}
```

**Final History Entry:**

```json
{
  "id": 6,
  "subscriptionId": 100,
  "sellerId": 5,
  "action": "cancelled",
  "fromPlanId": 3,
  "toPlanId": null,
  "fromStatus": "active",
  "toStatus": "cancelled",
  "amount": null,
  "currency": "USD",
  "transactionId": null,
  "performedBy": null,
  "performedByRole": "system",
  "reason": "Scheduled cancellation executed",
  "notes": "Subscription ended as scheduled",
  "metadata": {
    "total_lifetime_value": 237.0,
    "months_subscribed": 3
  },
  "createdAt": "2026-04-10T14:30:00Z"
}
```

---

## 📈 Usage Tracking Example

### Scenario: Seller Makes Sales Throughout the Month

#### Initial State (Start of billing period)

```json
{
  "subscriptionId": 100,
  "sellerId": 5,
  "periodStart": "2026-02-10T14:30:00Z",
  "periodEnd": "2026-03-10T14:30:00Z",
  "metricType": "orders",
  "usedCount": 0,
  "includedLimit": 2000,
  "overageCount": 0,
  "overageAmount": 0.0,
  "overageRate": 0.15,
  "lastUpdated": "2026-02-10T14:30:00Z"
}
```

---

#### Feb 15: First Order Placed

**Usage Log Entry:**

```json
{
  "id": 101,
  "subscriptionId": 100,
  "sellerId": 5,
  "metricType": "orders",
  "delta": 1,
  "referenceId": "ORD-2026-00001",
  "referenceType": "order",
  "timestamp": "2026-02-15T11:23:45Z"
}
```

**Updated Usage:**

```json
{
  "usedCount": 1,
  "includedLimit": 2000,
  "overageCount": 0,
  "lastUpdated": "2026-02-15T11:23:45Z"
}
```

---

#### Feb 28: After a Busy Week (150 orders)

**Usage Log Entries (sample):**

```json
[
  {
    "delta": 1,
    "referenceId": "ORD-2026-00001",
    "timestamp": "2026-02-15T11:23:45Z"
  },
  {
    "delta": 1,
    "referenceId": "ORD-2026-00002",
    "timestamp": "2026-02-15T14:56:12Z"
  },
  {
    "delta": 1,
    "referenceId": "ORD-2026-00003",
    "timestamp": "2026-02-16T09:11:33Z"
  },
  // ... more entries ...
  {
    "delta": 1,
    "referenceId": "ORD-2026-00150",
    "timestamp": "2026-02-28T22:45:01Z"
  }
]
```

**Updated Usage:**

```json
{
  "usedCount": 150,
  "includedLimit": 2000,
  "overageCount": 0,
  "overageAmount": 0.0,
  "lastUpdated": "2026-02-28T22:45:01Z"
}
```

---

#### Usage Dashboard Display (What seller sees)

```
┌─────────────────────────────────────────────────────────────┐
│  📊 Usage This Period (Feb 10 - Mar 10)                     │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Orders                                                      │
│  ████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  150 / 2,000   │
│  7.5% used                                                   │
│                                                              │
│  Products                                                    │
│  ∞ Unlimited                                    45 active   │
│                                                              │
│  Staff Accounts                                              │
│  ████████████████████░░░░░░░░░░░░░░░░░░░░░░░░  2 / 5       │
│  40% used                                                    │
│                                                              │
│  API Calls                                                   │
│  ████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  1,234 / ∞    │
│  (Tracking only, no limit)                                   │
│                                                              │
│  ─────────────────────────────────────────────              │
│  💰 Current Charges                                          │
│  Base Plan: $79.00                                           │
│  Overage: $0.00                                              │
│  ─────────────────────────────────────────────              │
│  Estimated Total: $79.00                                     │
│                                                              │
│  Next billing: March 10, 2026                                │
└─────────────────────────────────────────────────────────────┘
```

---

#### Scenario: Seller Exceeds Order Limit (Usage-Based Plan)

If seller was on a **usage-based plan** with 500 included orders and $0.15 per additional order:

**After 650 orders:**

```json
{
  "subscriptionId": 200,
  "sellerId": 10,
  "periodStart": "2026-02-10T00:00:00Z",
  "periodEnd": "2026-03-10T00:00:00Z",
  "metricType": "orders",
  "usedCount": 650,
  "includedLimit": 500,
  "overageCount": 150,
  "overageAmount": 22.5,
  "overageRate": 0.15,
  "lastUpdated": "2026-02-28T23:59:59Z"
}
```

**Usage Dashboard for Usage-Based Plan:**

```
┌─────────────────────────────────────────────────────────────┐
│  📊 Usage This Period (Feb 10 - Mar 10)                     │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Orders                                                      │
│  ████████████████████████████████████████████ 650 / 500    │
│  ⚠️ 150 orders over limit                                    │
│                                                              │
│  ─────────────────────────────────────────────              │
│  💰 Current Charges                                          │
│                                                              │
│  Base Plan (500 orders included): $29.00                    │
│  Additional 150 orders @ $0.15:   $22.50                    │
│  ─────────────────────────────────────────────              │
│  Estimated Total: $51.50                                     │
│                                                              │
│  💡 Tip: Upgrade to Pro for 2,000 orders at $79/mo          │
│  Next billing: March 10, 2026                                │
└─────────────────────────────────────────────────────────────┘
```

---

## 🌐 API Response Examples

### GET /api/v1/plans - List All Plans

```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "name": "Free Trial",
      "slug": "free-trial",
      "shortDesc": "Perfect for getting started",
      "planType": "fixed",
      "price": 0,
      "yearlyPrice": 0,
      "currency": "USD",
      "billingCycle": "monthly",
      "trialDays": 14,
      "isPopular": false,
      "badgeText": null,
      "limits": {
        "max_products": { "value": 10, "display": "10" },
        "max_orders_per_month": { "value": 50, "display": "50" },
        "max_staff": { "value": 1, "display": "1" },
        "has_api_access": { "value": false, "display": "No" }
      },
      "features": [
        { "text": "Online store", "isHighlight": true, "isIncluded": true },
        { "text": "10 products", "isHighlight": true, "isIncluded": true },
        { "text": "50 orders/month", "isHighlight": false, "isIncluded": true },
        { "text": "Email support", "isHighlight": false, "isIncluded": true },
        { "text": "API access", "isHighlight": false, "isIncluded": false }
      ]
    },
    {
      "id": 2,
      "name": "Basic",
      "slug": "basic",
      "shortDesc": "For solo entrepreneurs",
      "planType": "fixed",
      "price": 29.0,
      "yearlyPrice": 290.0,
      "currency": "USD",
      "billingCycle": "monthly",
      "trialDays": 14,
      "isPopular": false,
      "badgeText": null,
      "limits": {
        "max_products": { "value": -1, "display": "Unlimited" },
        "max_orders_per_month": { "value": 500, "display": "500" },
        "max_staff": { "value": 2, "display": "2" },
        "max_locations": { "value": 4, "display": "4" },
        "transaction_fee_percent": { "value": 2, "display": "2%" },
        "has_api_access": { "value": false, "display": "No" }
      },
      "features": [
        {
          "text": "Online store with unlimited bandwidth",
          "isHighlight": true,
          "isIncluded": true
        },
        {
          "text": "Unlimited products",
          "isHighlight": true,
          "isIncluded": true
        },
        {
          "text": "500 orders/month",
          "isHighlight": false,
          "isIncluded": true
        },
        {
          "text": "2 staff accounts",
          "isHighlight": false,
          "isIncluded": true
        },
        {
          "text": "24/7 chat support",
          "isHighlight": false,
          "isIncluded": true
        }
      ]
    },
    {
      "id": 3,
      "name": "Pro",
      "slug": "pro",
      "shortDesc": "For growing businesses",
      "planType": "fixed",
      "price": 79.0,
      "yearlyPrice": 790.0,
      "currency": "USD",
      "billingCycle": "monthly",
      "trialDays": 14,
      "isPopular": true,
      "badgeText": "Most Popular",
      "limits": {
        "max_products": { "value": -1, "display": "Unlimited" },
        "max_orders_per_month": { "value": 2000, "display": "2,000" },
        "max_staff": { "value": 5, "display": "5" },
        "max_locations": { "value": 10, "display": "10" },
        "transaction_fee_percent": { "value": 1, "display": "1%" },
        "has_api_access": { "value": true, "display": "Yes" },
        "has_custom_reports": { "value": true, "display": "Yes" }
      },
      "features": [
        {
          "text": "Everything in Basic",
          "isHighlight": false,
          "isIncluded": true
        },
        {
          "text": "2,000 orders/month",
          "isHighlight": true,
          "isIncluded": true
        },
        {
          "text": "5 staff accounts",
          "isHighlight": false,
          "isIncluded": true
        },
        { "text": "Full API access", "isHighlight": true, "isIncluded": true },
        {
          "text": "Custom report builder",
          "isHighlight": false,
          "isIncluded": true
        },
        { "text": "Priority support", "isHighlight": false, "isIncluded": true }
      ]
    },
    {
      "id": 4,
      "name": "Enterprise",
      "slug": "enterprise",
      "shortDesc": "For large operations",
      "planType": "enterprise",
      "price": 0,
      "yearlyPrice": 0,
      "currency": "USD",
      "billingCycle": "custom",
      "trialDays": 30,
      "isPopular": false,
      "badgeText": "Custom Pricing",
      "limits": {
        "max_products": { "value": -1, "display": "Unlimited" },
        "max_orders_per_month": { "value": -1, "display": "Unlimited" },
        "max_staff": { "value": -1, "display": "Unlimited" },
        "max_locations": { "value": -1, "display": "Unlimited" },
        "transaction_fee_percent": { "value": 0, "display": "Negotiated" },
        "has_api_access": { "value": true, "display": "Yes" },
        "support_level": {
          "value": "dedicated",
          "display": "Dedicated Account Manager"
        }
      },
      "features": [
        {
          "text": "Everything in Pro",
          "isHighlight": false,
          "isIncluded": true
        },
        {
          "text": "Unlimited everything",
          "isHighlight": true,
          "isIncluded": true
        },
        {
          "text": "Custom integrations",
          "isHighlight": true,
          "isIncluded": true
        },
        {
          "text": "Dedicated account manager",
          "isHighlight": true,
          "isIncluded": true
        },
        { "text": "SLA guarantee", "isHighlight": false, "isIncluded": true },
        {
          "text": "On-premise option",
          "isHighlight": false,
          "isIncluded": true
        }
      ],
      "ctaText": "Contact Sales",
      "ctaUrl": "/contact-sales"
    }
  ],
  "message": "Plans fetched successfully"
}
```

---

### GET /api/v1/seller/subscription - Get Current Subscription

```json
{
  "success": true,
  "data": {
    "subscription": {
      "id": 100,
      "status": "active",
      "billingCycle": "monthly",
      "currentPeriodStart": "2026-02-10T14:30:00Z",
      "currentPeriodEnd": "2026-03-10T14:30:00Z",
      "nextBillingDate": "2026-03-10T14:30:00Z",
      "baseAmount": 79.0,
      "currency": "USD",
      "autoRenew": true,
      "cancelAtPeriodEnd": false
    },
    "plan": {
      "id": 3,
      "name": "Pro",
      "slug": "pro",
      "price": 79.0,
      "billingCycle": "monthly"
    },
    "usage": {
      "orders": {
        "used": 150,
        "limit": 2000,
        "percentUsed": 7.5,
        "remaining": 1850,
        "isUnlimited": false
      },
      "products": {
        "used": 45,
        "limit": -1,
        "percentUsed": 0,
        "remaining": null,
        "isUnlimited": true
      },
      "staff": {
        "used": 2,
        "limit": 5,
        "percentUsed": 40,
        "remaining": 3,
        "isUnlimited": false
      }
    },
    "billing": {
      "currentCharges": {
        "base": 79.0,
        "overage": 0.0,
        "total": 79.0
      },
      "paymentMethod": {
        "type": "card",
        "brand": "visa",
        "last4": "4242",
        "expiryMonth": 12,
        "expiryYear": 2028
      }
    }
  },
  "message": "Subscription fetched successfully"
}
```

---

### POST /api/v1/seller/subscription/change-plan - Upgrade/Downgrade

**Request:**

```json
{
  "newPlanSlug": "pro",
  "billingCycle": "yearly"
}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "subscription": {
      "id": 100,
      "planId": 3,
      "planSlug": "pro",
      "status": "active",
      "billingCycle": "yearly",
      "currentPeriodStart": "2026-02-15T10:00:00Z",
      "currentPeriodEnd": "2027-02-15T10:00:00Z",
      "nextBillingDate": "2027-02-15T10:00:00Z",
      "baseAmount": 790.0
    },
    "proration": {
      "unusedAmount": 45.23,
      "newPlanAmount": 790.0,
      "chargeAmount": 744.77,
      "explanation": "You'll be charged $744.77 today (yearly Pro plan $790 - $45.23 unused from current plan)"
    },
    "invoice": {
      "id": "inv_123456",
      "amount": 744.77,
      "status": "paid",
      "paidAt": "2026-02-15T10:00:05Z"
    }
  },
  "message": "Successfully upgraded to Pro (Yearly)"
}
```

---

## 🎯 Real-World Scenarios

### Scenario 1: Checking if Seller Can Create Product

**Service Logic:**

```go
func (s *ProductService) CanCreateProduct(ctx context.Context, sellerID uint) (bool, error) {
    // Get current subscription usage
    usage, err := s.subscriptionService.GetUsage(ctx, sellerID, "products")
    if err != nil {
        return false, err
    }

    // Check limit
    if usage.IncludedLimit == -1 {
        return true, nil // Unlimited
    }

    if usage.UsedCount >= usage.IncludedLimit {
        return false, errors.PlanLimitExceeded("products", usage.IncludedLimit)
    }

    return true, nil
}
```

**Error Response when Limit Exceeded:**

```json
{
  "success": false,
  "error": {
    "code": "PLAN_LIMIT_EXCEEDED",
    "message": "You've reached the maximum of 10 products on your current plan",
    "details": {
      "resource": "products",
      "limit": 10,
      "used": 10
    },
    "upgradeUrl": "/settings/billing/upgrade"
  }
}
```

---

### Scenario 2: Recording Order for Usage Tracking

**When order is created:**

```go
func (s *OrderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error) {
    // ... create order logic ...

    // Record usage
    err := s.usageService.RecordUsage(ctx, UsageRecord{
        SellerID:      req.SellerID,
        MetricType:    "orders",
        Delta:         1,
        ReferenceID:   order.ID,
        ReferenceType: "order",
    })

    return order, err
}
```

**When order is cancelled/refunded:**

```go
func (s *OrderService) CancelOrder(ctx context.Context, orderID string) error {
    // ... cancel order logic ...

    // Decrement usage (if within same billing period)
    if order.CreatedAt.After(subscription.CurrentPeriodStart) {
        err := s.usageService.RecordUsage(ctx, UsageRecord{
            SellerID:      order.SellerID,
            MetricType:    "orders",
            Delta:         -1,
            ReferenceID:   order.ID,
            ReferenceType: "order_cancelled",
        })
    }

    return nil
}
```

---

### Scenario 3: Displaying Pricing Page

**Frontend fetches plans:**

```javascript
const plans = await fetch("/api/v1/plans").then((r) => r.json());

// Render pricing cards
plans.data.forEach((plan) => {
  renderPricingCard({
    name: plan.name,
    price:
      plan.billingCycle === "monthly"
        ? `$${plan.price}/mo`
        : `$${plan.yearlyPrice}/yr`,
    badge: plan.badgeText,
    isPopular: plan.isPopular,
    features: plan.features.filter((f) => f.isIncluded),
    limits: Object.entries(plan.limits).map(([key, val]) => ({
      label: key.replace(/_/g, " "),
      value: val.display,
    })),
  });
});
```

---

### Scenario 4: End of Month Billing (Cron Job)

**System processes subscriptions:**

```go
func (j *BillingJob) ProcessMonthlyBilling(ctx context.Context) error {
    // Get subscriptions due for billing
    subscriptions, err := j.repo.GetDueForBilling(ctx, time.Now())

    for _, sub := range subscriptions {
        // Calculate total charge
        baseAmount := sub.BaseAmount

        // Get overage charges
        usages, _ := j.usageRepo.GetBySubscription(ctx, sub.ID, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)

        overageTotal := 0.0
        for _, usage := range usages {
            overageTotal += usage.OverageAmount
        }

        totalCharge := baseAmount + overageTotal

        // Charge customer
        payment, err := j.paymentService.Charge(ctx, ChargeRequest{
            SubscriptionID: sub.ID,
            Amount:         totalCharge,
            Currency:       sub.Currency,
            Description:    fmt.Sprintf("%s Plan - %s", sub.PlanName, sub.BillingCycle),
        })

        if err != nil {
            // Handle failed payment (retry, grace period, etc.)
            j.handleFailedPayment(ctx, sub, err)
            continue
        }

        // Renew subscription
        j.renewSubscription(ctx, sub, payment)
    }
}
```

---

## 📋 Summary: All Tables & Their Purpose

| Table                    | Purpose                      | Example Data                    |
| ------------------------ | ---------------------------- | ------------------------------- |
| `plan`                   | Main plan definition         | Pro plan, $79/mo                |
| `plan_limit`             | Feature gating/limits        | max_products=∞, max_orders=2000 |
| `plan_tier`              | Volume pricing tiers         | 0-100: $0.50, 101-500: $0.30    |
| `plan_feature`           | Marketing display            | "Full API access" ✓             |
| `subscription`           | Seller's active subscription | Seller #5 on Pro plan           |
| `subscription_usage`     | Current period metrics       | 150/2000 orders used            |
| `subscription_usage_log` | Audit trail per action       | +1 order, ref: ORD-123          |
| `subscription_history`   | Subscription lifecycle       | upgraded from Basic to Pro      |

---

## 🔑 Key Relationships

```
plan (1) ──────► (N) plan_limit       "A plan has many limits"
plan (1) ──────► (N) plan_tier        "A plan has many pricing tiers"
plan (1) ──────► (N) plan_feature     "A plan has many features"

seller (1) ────► (1) subscription     "A seller has one active subscription"
plan (1) ──────► (N) subscription     "A plan can have many subscribers"

subscription (1) ► (N) subscription_usage      "Track multiple metrics"
subscription (1) ► (N) subscription_usage_log  "Detailed audit log"
subscription (1) ► (N) subscription_history    "Lifecycle events"
```

---

**This completes the comprehensive example documentation!** 🎉
