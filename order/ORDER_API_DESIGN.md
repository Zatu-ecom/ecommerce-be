# Order API Design

## Overview

Order APIs for the Zatu e-commerce platform. All order endpoints are **role-aware** — a single unified route adapts its query scope and response shape based on the authenticated user's role from the JWT token.

**Existing Infrastructure:**
- DB tables: `order`, `order_item`, `order_address`, `order_applied_promotion`, `order_applied_coupon`, `order_item_applied_promotion`
- Entities: All defined in `order/entity/`
- Cart API: Fully built (cart → order conversion path ready)

---

## Cart Lifecycle

Carts are **not deleted** after order creation. Instead, they transition through statuses to preserve history and prevent race conditions during checkout.

### Cart Statuses

| Status | Description | Constraint |
|--------|------------|------------|
| `active` | User is browsing/adding items. **Only one active cart per user.** | Partial unique index: `UNIQUE(user_id) WHERE status = 'active'` |
| `checkout` | Cart is locked — order entity created, payment in progress. Cart cannot be modified. | No unique constraint (user can retry checkout) |
| `converted` | Order successfully placed. Cart kept for history/audit. | Cleaned up by cron after configurable period |

### Cart Status Transitions

```
┌────────┐    Initiate Checkout     ┌──────────┐    Payment Success     ┌───────────┐
│ active │ ─────────────────────→ │ checkout │ ──────────────────→  │ converted │
└────────┘                         └─────┬────┘                     └───────────┘
     ▲                                   │
     │          Payment Failed /         │
     └───────── Checkout Timeout ────────┘
```

| From | To | Trigger |
|------|----|---------|
| `active` | `checkout` | User initiates order creation |
| `checkout` | `converted` | Payment succeeds / order confirmed |
| `checkout` | `active` | Payment fails / checkout times out |

### DB Schema Change

The existing `uniqueIndex` on `user_id` must change to a **partial unique index** so only one `active` cart exists per user, while allowing multiple `converted` carts for history:

```sql
-- Remove old unique index
DROP INDEX IF EXISTS idx_cart_user_id;

-- Add status column
ALTER TABLE cart ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'active';

-- Partial unique index: only one active cart per user
CREATE UNIQUE INDEX idx_cart_user_id_active ON cart(user_id) WHERE status = 'active';

-- Index for cron cleanup queries
CREATE INDEX idx_cart_status_updated_at ON cart(status, updated_at);
```

### Cart Entity Update

```go
type CartStatus string

const (
    CART_STATUS_ACTIVE    CartStatus = "active"
    CART_STATUS_CHECKOUT  CartStatus = "checkout"
    CART_STATUS_CONVERTED CartStatus = "converted"
)

type Cart struct {
    db.BaseEntity
    UserID   uint       `json:"userId"   gorm:"column:user_id;not null;index"`
    Status   CartStatus `json:"status"   gorm:"column:status;size:20;not null;default:'active'"`
    OrderID  *uint      `json:"orderId"  gorm:"column:order_id;index"`  // linked after conversion
    Metadata db.JSONMap `json:"metadata" gorm:"column:metadata;type:jsonb;default:'{}'"`
}
```

> **Note:** `OrderID` links the converted cart to its order — useful for debugging and audit trails.

---

## API Endpoints

### 1. Create Order

```
POST /api/order
Auth: CustomerAuth (customers only)
```

Converts the authenticated user's cart into an order. Snapshots all prices, promotions, and addresses at the time of placement.

#### Request Body

```json
{
  "shippingAddressId": 12,
  "billingAddressId": 12,
  "fulfillmentType": "directship",
  "metadata": {}
}
```

> **Why no `cartId` or `items`?**
> The service finds the user's `active` cart automatically from the JWT token's `userId`. Cart items are already populated via the existing Add-to-Cart API.

#### Flow

```
Find Active Cart → Cart: active → checkout (lock)
→ Validate variants belong to seller
→ Re-evaluate promotions
→ Generate order number
→ [DB Transaction]
   → Create Order (status: pending)
   → Create Order Items (price snapshot)
   → Create Order Addresses (address snapshot)
   → Create Order Applied Promotions (snapshot)
   → Create Order Item Applied Promotions (snapshot)
   → Create order_history entry (null → pending)
   → CreateReservation(orderID, cartItems, expiresInMinutes)
      ↳ Validates stock by location priority
      ↳ Moves qty: available → reserved
      ↳ Schedules Redis TTL for auto-expiry
   → Cart: checkout → converted (set order_id)
   → Create new active cart for user
→ Return Order
```

If any step fails after the cart is locked:
```
Cart: checkout → active (unlock, user can retry)
```

#### Business Rules
- User must have an `active` cart with items
- All variants must belong to the seller (validated by reservation service)
- Promotions are re-evaluated at order time (not from cart cache)
- Address IDs must belong to the authenticated user
- Cart is **locked** (`checkout` status) during order creation — prevents concurrent modifications
- On success: cart marked `converted`, a **new empty `active` cart** is created for the user
- On failure: cart reverted to `active`
- `fulfillmentType` defaults to `directship` if not provided
- Order number is auto-generated (e.g., `ORD-20260329-XXXX`)
- Initial order status: always `pending` (cannot be overridden in request)
- `placed_at` is set to current timestamp
- Inventory is **reserved** (not deducted) via `CreateReservation` — actual deduction happens on shipment
- Reservation auto-expires via Redis TTL if payment is not completed in time

> **Note:** The `status` field is NOT accepted in the Create Order request — orders always start as `pending`. Status changes happen via the Update Status API or payment webhooks.

#### Success Response `201 Created`

```json
{
  "success": true,
  "message": "Order placed successfully",
  "data": {
    "id": 1,
    "orderNumber": "ORD-20260329-A1B2",
    "status": "pending",
    "subtotalCents": 279798,
    "discountCents": 27979,
    "shippingCents": 0,
    "taxCents": 0,
    "totalCents": 251819,
    "fulfillmentType": "directship",
    "placedAt": "2026-03-29T00:00:00Z",
    "items": [
      {
        "id": 1,
        "productId": 10,
        "variantId": 25,
        "productName": "iPhone 15 Pro",
        "variantName": "256GB Space Black",
        "sku": "IPH15P-256-BLK",
        "imageUrl": "https://...",
        "quantity": 1,
        "unitPriceCents": 99900,
        "lineTotalCents": 99900,
        "appliedPromotionBreakdown": [
          {
            "promotionId": 5,
            "promotionName": "10% off Electronics",
            "promotionType": "percentage",
            "discountCents": 9990,
            "originalCents": 99900,
            "finalCents": 89910,
            "freeQuantity": 0
          }
        ]
      }
    ],
    "addresses": [
      {
        "type": "shipping",
        "address": "123 Main St",
        "landmark": "Near Central Mall",
        "city": "Mumbai",
        "state": "Maharashtra",
        "zipCode": "400001",
        "countryId": 1
      }
    ],
    "appliedPromotions": [
      {
        "promotionId": 5,
        "promotionName": "10% off Electronics",
        "promotionType": "percentage",
        "discountCents": 27979,
        "shippingDiscountCents": 0,
        "isStackable": false,
        "priority": 1
      }
    ]
  }
}
```

> **TODO:** `appliedCoupons` will be added once the coupon/discount-code engine is implemented.

#### Error Cases
| Status | Condition |
|--------|-----------|
| `400` | Missing/invalid shipping address, invalid fulfillment type |
| `404` | Cart not found (empty cart) |
| `409` | Insufficient stock for one or more items |
| `401` | Unauthorized |

---

### 2. Get Order by ID

```
GET /api/order/:id
Auth: Auth (any authenticated user)
```

Returns full order details including items, addresses, and promotion snapshots.

#### Role-Based Scoping

| Role | Access Rule |
|------|------------|
| Customer | Can only view their own orders (`user_id = token.userId`) |
| Seller | Can only view orders for their store (`seller_id = token.sellerId`) |
| Admin | Can view any order |

#### Success Response `200 OK`

```json
{
  "success": true,
  "data": {
    "id": 1,
    "orderNumber": "ORD-20260329-A1B2",
    "status": "confirmed",
    "subtotalCents": 279798,
    "discountCents": 27979,
    "shippingCents": 0,
    "taxCents": 0,
    "totalCents": 251819,
    "fulfillmentType": "directship",
    "placedAt": "2026-03-29T00:00:00Z",
    "paidAt": null,
    "customer": {
      "id": 5,
      "firstName": "Kushal",
      "lastName": "Patel",
      "email": "kushal@example.com",
      "phone": "+91-9876543210"
    },
    "items": [
      {
        "id": 1,
        "productId": 10,
        "variantId": 25,
        "productName": "iPhone 15 Pro",
        "variantName": "256GB Space Black",
        "sku": "IPH15P-256-BLK",
        "imageUrl": "https://...",
        "quantity": 1,
        "unitPriceCents": 99900,
        "lineTotalCents": 99900,
        "attributes": {},
        "appliedPromotionBreakdown": [
          {
            "promotionId": 5,
            "promotionName": "10% off Electronics",
            "promotionType": "percentage",
            "discountCents": 9990,
            "originalCents": 99900,
            "finalCents": 89910,
            "freeQuantity": 0
          }
        ]
      }
    ],
    "addresses": [
      {
        "type": "shipping",
        "address": "123 Main St",
        "landmark": "Near Central Mall",
        "city": "Mumbai",
        "state": "Maharashtra",
        "zipCode": "400001",
        "countryId": 1,
        "latitude": 19.076,
        "longitude": 72.877
      },
      {
        "type": "billing",
        "address": "123 Main St",
        "landmark": "Near Central Mall",
        "city": "Mumbai",
        "state": "Maharashtra",
        "zipCode": "400001",
        "countryId": 1
      }
    ],
    "appliedPromotions": [
      {
        "promotionId": 5,
        "promotionName": "10% off Electronics",
        "promotionType": "percentage",
        "discountCents": 27979,
        "shippingDiscountCents": 0,
        "isStackable": false,
        "priority": 1
      }
    ]
  }
}
```

> **TODO:** `appliedCoupons` array will be added to this response once the coupon/discount-code engine is implemented.

#### Error Cases
| Status | Condition |
|--------|-----------|
| `404` | Order not found or forbidden for user's role scope |
| `401` | Unauthorized |

---

### 3. List Orders (Unified, Role-Aware)

```
GET /api/order
Auth: Auth (any authenticated user)
```

Single endpoint that returns orders scoped by the authenticated user's role.

#### Role-Based Query Scope

| Role | Query Scope | Use Case |
|------|------------|----------|
| Customer | `WHERE user_id = ?` | "My Orders" |
| Seller | `WHERE seller_id = ?` | "Store Orders" |
| Admin | No scope filter | "All Orders" |
| Support (future) | Read-only, no scope filter | "Customer lookup" |

#### Query Parameters

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `pageSize` | int | 20 | Items per page (max 100) |
| `status` | string | — | Filter by status: `pending`, `confirmed`, `cancelled`, etc. |
| `sortBy` | string | `created_at` | Sort field: `created_at`, `total_cents`, `order_number` |
| `sortOrder` | string | `desc` | `asc` or `desc` |
| `fromDate` | string | — | Filter orders placed after this ISO date |
| `toDate` | string | — | Filter orders placed before this ISO date |
| `search` | string | — | Search by order number |

#### Success Response `200 OK`

```json
{
  "success": true,
  "data": {
    "orders": [
      {
        "id": 1,
        "orderNumber": "ORD-20260329-A1B2",
        "status": "confirmed",
        "totalCents": 251819,
        "discountCents": 27979,
        "itemCount": 2,
        "fulfillmentType": "directship",
        "placedAt": "2026-03-29T00:00:00Z"
      }
    ],
    "pagination": {
      "currentPage": 1,
      "totalPages": 5,
      "totalItems": 47,
      "itemsPerPage": 20,
      "hasNext": true,
      "hasPrev": false
    }
  }
}
```

> **Note:** List response returns a lightweight summary per order (no items/addresses). Use `GET /api/order/:id` for full details.

#### Customer Info (Seller/Admin View)

When the authenticated user is a **seller** or **admin**, each order in the list includes customer info for fulfillment:

```json
{
  "id": 1,
  "orderNumber": "ORD-20260329-A1B2",
  "status": "confirmed",
  "totalCents": 251819,
  "discountCents": 27979,
  "itemCount": 2,
  "fulfillmentType": "directship",
  "placedAt": "2026-03-29T00:00:00Z",
  "customer": {
    "id": 5,
    "firstName": "Kushal",
    "lastName": "Patel",
    "email": "kushal@example.com",
    "phone": "+91-9876543210"
  }
}
```

> **Note:** `customer` field is **omitted** for customer role (they already know who they are). This is handled in the service layer's response mapper — no route changes needed.

---

### 4. Update Order Status

```
PATCH /api/order/:id/status
Auth: SellerAuth (sellers only)
```

Seller updates the order through its lifecycle. Every status change is recorded in the `order_history` table for full audit trail.

#### Request Body

```json
{
  "status": "confirmed",
  "transactionId": "pay_RAZORPAY_TXN_123",
  "note": "Payment verified, preparing for shipment",
  "failureReason": null,
  "metadata": {}
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | ✅ | Target status |
| `transactionId` | string | ❌ | Payment gateway reference (required for `pending → confirmed`) |
| `note` | string | ❌ | Internal note about the transition |
| `failureReason` | string | ❌ | Reason for failure (required for `pending → failed`) |
| `metadata` | object | ❌ | Additional context (tracking number, refund ID, etc.) |

#### Valid Status Transitions

```
pending ──→ confirmed ──→ completed
   │            │
   │            └──→ cancelled
   │
   └──→ cancelled
   └──→ failed

completed ──→ returned
```

| From | Allowed To | Required Fields |
|------|-----------|----------------|
| `pending` | `confirmed` | `transactionId` |
| `pending` | `cancelled` | — |
| `pending` | `failed` | `failureReason` |
| `confirmed` | `completed` | — |
| `confirmed` | `cancelled` | — |
| `completed` | `returned` | — |

#### Business Rules
- Seller can only update orders for their store (`seller_id = token.sellerId`)
- Invalid transitions return `400`
- `cancelled`, `failed`, and `returned` are terminal states
- `transactionId` is required when confirming payment (`pending → confirmed`)
- `failureReason` is required when marking as failed
- Every transition is logged in `order_history` with actor, timestamp, and context
- `paidAt` is auto-set when status changes to `confirmed`

#### Inventory Reservation Side Effects

| Order Transition | Reservation Action | Inventory Effect |
|---|---|---|
| Create Order (`→ pending`) | `CreateReservation(PENDING)` | `available → reserved` |
| `pending → confirmed` | `UpdateReservationStatus(CONFIRMED)` | No change (stays reserved) |
| `pending → failed` | `UpdateReservationStatus(CANCELLED)` | `reserved → available` (released) |
| `pending → cancelled` | `UpdateReservationStatus(CANCELLED)` | `reserved → available` (released) |
| `confirmed → completed` | `UpdateReservationStatus(FULFILLED)` | `reserved → outbound` (actual deduction at shipment) |
| `confirmed → cancelled` | `UpdateReservationStatus(CANCELLED)` | `reserved → available` (released) |
| `completed → returned` | No reservation change | Handle via return/refund flow (future) |
| Reservation TTL expires | Auto-triggered by Redis scheduler | `reserved → available` (auto-release) |

> **Key principle:** Stock is only **permanently deducted** (`FULFILLED`/`outbound`) when the shipment leaves. Until then it stays `reserved`, protecting availability for this order while keeping it visible for restocking decisions.

#### Success Response `200 OK`

```json
{
  "success": true,
  "message": "Order status updated to confirmed",
  "data": {
    "id": 1,
    "orderNumber": "ORD-20260329-A1B2",
    "status": "confirmed",
    "previousStatus": "pending",
    "transactionId": "pay_RAZORPAY_TXN_123"
  }
}
```

#### Error Cases
| Status | Condition |
|--------|-----------|
| `400` | Invalid status value, invalid transition, or missing required fields |
| `404` | Order not found for seller |
| `401/403` | Unauthorized / wrong role |

---

### 5. Cancel Order (Customer-Initiated)

```
POST /api/order/:id/cancel
Auth: CustomerAuth (customers only)
```

Customer cancels their own order. Only allowed for `pending` or `confirmed` orders.

#### Request Body (Optional)

```json
{
  "reason": "Changed my mind"
}
```

#### Business Rules
- Customer can only cancel their own orders (`user_id = token.userId`)
- Only `pending` and `confirmed` orders can be cancelled
- Cancellation restores inventory (stock reversal)
- Cancellation reason is stored in `metadata`

#### Success Response `200 OK`

```json
{
  "success": true,
  "message": "Order cancelled successfully",
  "data": {
    "id": 1,
    "orderNumber": "ORD-20260329-A1B2",
    "status": "cancelled"
  }
}
```

#### Error Cases
| Status | Condition |
|--------|-----------|
| `400` | Order not in cancellable state |
| `404` | Order not found for user |

---

## Order History (Audit Log)

Every status change is recorded in the `order_history` table for full traceability.

### DB Schema

```sql
CREATE TABLE IF NOT EXISTS order_history (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES "order"(id) ON DELETE CASCADE,
    from_status VARCHAR(32),
    to_status VARCHAR(32) NOT NULL,
    changed_by_user_id BIGINT,
    changed_by_role VARCHAR(32),
    transaction_id VARCHAR(255),
    failure_reason TEXT,
    note TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_order_history_order_id ON order_history(order_id);
CREATE INDEX idx_order_history_created_at ON order_history(created_at);
```

### Entity

```go
type OrderHistory struct {
    db.BaseEntity
    OrderID         uint       `json:"orderId"         gorm:"column:order_id;not null;index"`
    FromStatus      *string    `json:"fromStatus"      gorm:"column:from_status;size:32"`
    ToStatus        string     `json:"toStatus"        gorm:"column:to_status;size:32;not null"`
    ChangedByUserID *uint      `json:"changedByUserId" gorm:"column:changed_by_user_id"`
    ChangedByRole   *string    `json:"changedByRole"   gorm:"column:changed_by_role;size:32"`
    TransactionID   *string    `json:"transactionId"   gorm:"column:transaction_id;size:255"`
    FailureReason   *string    `json:"failureReason"   gorm:"column:failure_reason"`
    Note            *string    `json:"note"            gorm:"column:note"`
    Metadata        db.JSONMap `json:"metadata"        gorm:"column:metadata;type:jsonb;default:'{}'"`
}
```

### What Gets Logged

| Event | `fromStatus` | `toStatus` | Extra Fields |
|-------|-------------|-----------|-------------|
| Order created | `null` | `pending` | — |
| Payment confirmed | `pending` | `confirmed` | `transactionId` |
| Payment failed | `pending` | `failed` | `failureReason` |
| Seller confirms | `pending` | `confirmed` | `note` |
| Order completed | `confirmed` | `completed` | — |
| Customer cancels | `pending`/`confirmed` | `cancelled` | `note` (reason) |
| Seller cancels | `confirmed` | `cancelled` | `note` |
| Return initiated | `completed` | `returned` | `note`, `metadata` |

> **Note:** `changedByUserId` and `changedByRole` track WHO made the change — useful for distinguishing customer cancellations from seller cancellations, or system-triggered changes (cron timeout).

---

## Architecture

### Service Layer Pattern

```go
type OrderService interface {
    CreateOrder(ctx, userCtx, req CreateOrderRequest) (*Order, error)
    GetOrderByID(ctx, userCtx, orderID uint) (*Order, error)
    ListOrders(ctx, userCtx, filters OrderFilters) (*PaginatedOrders, error)
    UpdateOrderStatus(ctx, userCtx, orderID uint, req UpdateStatusRequest) (*Order, error)
    CancelOrder(ctx, userCtx, orderID uint, req CancelOrderRequest) (*Order, error)
}
```

The `userCtx` contains `UserID`, `SellerID`, and `Role` — extracted from the JWT by middleware. The service layer uses this to scope every query.

### Role Extensibility

To add a new role (e.g., `warehouse_manager`):
1. Add the role constant
2. Add a `case` in the service's query scoper
3. Optionally add role-specific response fields in the mapper

No route, handler, or middleware changes needed.

### Order Number Generation

Format: `ORD-<epoch_ms>-<seller_b36>-<random>`

```
Example: ORD-1774766744711-24B-X9K4
         │   │             │   │
         │   │             │   └── 4 random alphanumeric (collision avoidance)
         │   │             └────── Seller ID in base36 (seller_id=2887 → "24B")
         │   └──────────────────── Epoch milliseconds (compact, sortable, unique)
         └──────────────────────── Prefix
```

| Segment | Purpose | Example |
|---------|---------|---------|
| `ORD` | Fixed prefix for order identification | `ORD` |
| `epoch_ms` | Unix timestamp in milliseconds (13 digits) | `1774766744711` |
| `seller_b36` | Seller ID encoded in base36 (compact, reversible) | `24B` (seller 2887) |
| `random` | 4-char alphanumeric random (collision avoidance) | `X9K4` |

```go
// Pseudo-code
func GenerateOrderNumber(sellerID uint) string {
    epochMs := time.Now().UnixMilli()
    sellerB36 := strconv.FormatUint(uint64(sellerID), 36)
    random := generateRandomAlphanumeric(4)
    
    return fmt.Sprintf("ORD-%d-%s-%s",
        epochMs, strings.ToUpper(sellerB36), random)
}
```

- Unique constraint in DB prevents collisions
- Epoch ms is **naturally sortable** — orders sort chronologically by order number
- Base36 seller ID is **reversible** (`strconv.ParseUint("24B", 36, 64)` → `2887`) for debugging
- Total length: ~30 chars (fits in `VARCHAR(255)`)

### Cart Cleanup (Cron / DB Trigger)

Converted carts are retained for audit/debugging and cleaned up automatically:

| Option | Description |
|--------|------------|
| **Cron Job** (recommended) | Runs daily, deletes `converted` carts older than `N` days (configurable, default: 30 days) |
| **DB Trigger** | `pg_cron` or a Postgres trigger that fires on a schedule |

```go
// Cron job pseudo-code
func (s *CartCleanupService) CleanupConvertedCarts() {
    retentionDays := config.GetInt("cart.retention_days", 30)
    cutoff := time.Now().AddDate(0, 0, -retentionDays)
    
    db.Where("status = ? AND updated_at < ?", "converted", cutoff).
       Delete(&Cart{})
}
```

> **Note:** This follows the same pattern as the existing `PromotionCronService`. The retention period should be configurable via environment variable.

---

## Order Status Lifecycle Diagram

```
┌──────────┐
│ pending  │
└────┬─────┘
     │
     ├──────────────────┐──────────────┐
     ▼                  ▼              ▼
┌───────────┐     ┌───────────┐  ┌──────────┐
│ confirmed │     │ cancelled │  │  failed  │
└─────┬─────┘     └───────────┘  └──────────┘
      │
      ├──────────────────┐
      ▼                  ▼
┌───────────┐     ┌───────────┐
│ completed │     │ cancelled │
└─────┬─────┘     └───────────┘
      │
      ▼
┌───────────┐
│ returned  │
└───────────┘
```

---

## TDD Test Scenarios

> **Approach:** Write tests FIRST, then implement. Each test scenario below maps to a suite method in the integration tests. Tests should verify not just the API response, but also the **side effects** — cart status, inventory levels, order history entries, and promotion snapshots.

### 1. Create Order (`POST /api/order`)

#### Happy Path
| # | Scenario | Verify |
|---|----------|--------|
| 1.1 | Create order with valid cart and shipping address | Order created with `pending` status, correct `orderNumber` format |
| 1.2 | Order items match cart items | `order_item` rows match cart items (product, variant, quantity, price) |
| 1.3 | Cart status transitions to `checkout` then `converted` | Cart `status = 'converted'`, cart `order_id` links to new order |
| 1.4 | New active cart is created for user | A fresh `active` cart exists for the user after order creation |
| 1.5 | Inventory is deducted | Variant stock reduced by ordered quantities |
| 1.6 | Promotions are snapshotted | `order_applied_promotion` rows match active promotions at order time |
| 1.7 | Item-level promotion breakdown is snapshotted | `order_item_applied_promotion` rows have correct `discountCents`, `originalCents`, `finalCents` |
| 1.8 | Order totals are correct | `subtotalCents`, `discountCents`, `totalCents` math is accurate |
| 1.9 | Order addresses are snapshotted | `order_address` rows match user's address (shipping + billing) |
| 1.10 | `placedAt` is set | `placed_at` is not null and is recent timestamp |
| 1.11 | Order history entry created | `order_history` row: `from_status = null`, `to_status = 'pending'` |
| 1.12 | Default fulfillment type | When `fulfillmentType` is omitted, defaults to `directship` |
| 1.13 | Custom fulfillment type | When `fulfillmentType = 'bopis'`, order has `bopis` |

#### Cart & Stock Validation
| # | Scenario | Expected |
|---|----------|----------|
| 1.14 | No active cart (never added items) | `404` — cart not found |
| 1.15 | Active cart exists but is empty (0 items) | `400` — cart has no items |
| 1.16 | Variant out of stock | `409` — insufficient stock, order NOT created, cart stays `active` |
| 1.17 | Partial stock (qty 5 requested, only 3 available) | `409` — insufficient stock with details |
| 1.18 | Cart already in checkout (concurrent request) | `409` — cart already locked |

#### Address Validation
| # | Scenario | Expected |
|---|----------|----------|
| 1.19 | Missing shipping address ID | `400` — shipping address required |
| 1.20 | Shipping address belongs to different user | `404` — address not found |
| 1.21 | Invalid fulfillment type | `400` — invalid fulfillment type |

#### Promotion Edge Cases
| # | Scenario | Verify |
|---|----------|--------|
| 1.22 | Promotion expired between cart and checkout | Promotion NOT applied, order totals reflect full price |
| 1.23 | Bundle promotion — exact match | Bundle discount snapshotted correctly in `order_applied_promotion` |
| 1.24 | Multiple stackable promotions | All promotions snapshotted, combined `discountCents` is correct |
| 1.25 | Non-stackable promotion — highest priority wins | Only highest-priority promotion applied and snapshotted |

#### Rollback on Failure
| # | Scenario | Verify |
|---|----------|--------|
| 1.26 | Error after cart locked (e.g., DB error during order insert) | Cart reverted to `active`, inventory NOT deducted, no order created |

---

### 2. Get Order by ID (`GET /api/order/:id`)

#### Happy Path
| # | Scenario | Verify |
|---|----------|--------|
| 2.1 | Customer gets their own order | Full order returned with items, addresses, promotions |
| 2.2 | Seller gets order for their store | Full order returned with `customer` info (name, email, phone) |
| 2.3 | Response includes item-level promotion breakdown | `appliedPromotionBreakdown` is populated on each item |
| 2.4 | Response includes both shipping and billing addresses | Both address types present |

#### Access Control
| # | Scenario | Expected |
|---|----------|----------|
| 2.5 | Customer tries to access another user's order | `404` — not found |
| 2.6 | Seller tries to access order from different store | `404` — not found |
| 2.7 | Unauthenticated request | `401` — unauthorized |

#### Customer Info Visibility
| # | Scenario | Verify |
|---|----------|--------|
| 2.8 | Customer role — `customer` field is absent | Response does NOT contain `customer` object |
| 2.9 | Seller role — `customer` field is present | Response contains `customer` with `firstName`, `lastName`, `email`, `phone` |

---

### 3. List Orders (`GET /api/order`)

#### Role-Based Scoping
| # | Scenario | Verify |
|---|----------|--------|
| 3.1 | Customer sees only their own orders | All returned orders have `userId` matching token |
| 3.2 | Seller sees only their store's orders | All returned orders have `sellerId` matching token |
| 3.3 | Seller list includes `customer` info | Each order in response has `customer` object |
| 3.4 | Customer list does NOT include `customer` info | `customer` field absent from response |

#### Pagination
| # | Scenario | Verify |
|---|----------|--------|
| 3.5 | Default pagination (page 1, pageSize 20) | Correct defaults, pagination metadata present |
| 3.6 | Custom pagination (page 2, pageSize 5) | Correct `currentPage`, `itemsPerPage`, `hasPrev = true` |
| 3.7 | Page beyond total | Empty `orders` array, `totalItems` still accurate |
| 3.8 | Very large pageSize (1000) | Capped at 100 |

#### Filters
| # | Scenario | Verify |
|---|----------|--------|
| 3.9 | Filter by `status=confirmed` | Only confirmed orders returned |
| 3.10 | Filter by date range (`fromDate`, `toDate`) | Only orders within range |
| 3.11 | Search by order number | Exact/partial match works |
| 3.12 | Sort by `total_cents` ascending | Orders sorted correctly |
| 3.13 | Combined filters (status + date + sort) | All filters applied simultaneously |
| 3.14 | No orders exist | Empty array, `totalItems = 0` |

---

### 4. Update Order Status (`PATCH /api/order/:id/status`)

#### Valid Transitions
| # | Scenario | Verify |
|---|----------|--------|
| 4.1 | `pending → confirmed` with `transactionId` | Status updated, `paidAt` set, `transactionId` stored on order |
| 4.2 | `pending → cancelled` | Status updated, inventory restored |
| 4.3 | `pending → failed` with `failureReason` | Status updated, `failureReason` recorded |
| 4.4 | `confirmed → completed` | Status updated |
| 4.5 | `confirmed → cancelled` | Status updated, inventory restored |
| 4.6 | `completed → returned` | Status updated |

#### Order History Audit
| # | Scenario | Verify |
|---|----------|--------|
| 4.7 | Any valid transition | `order_history` row created with correct `from_status`, `to_status`, `changed_by_user_id`, `changed_by_role` |
| 4.8 | Transition with `note` | `note` saved in `order_history` |
| 4.9 | Transition with `metadata` (tracking number) | `metadata` saved in `order_history` |
| 4.10 | Multiple transitions on same order | Multiple `order_history` rows in chronological order |

#### Cart Status Side Effects
| # | Scenario | Verify |
|---|----------|--------|
| 4.11 | `pending → confirmed` | Cart stays `converted` |
| 4.12 | `pending → failed` | Cart reverted to `active` (user can retry), order's linked cart `order_id` cleared |

#### Inventory Side Effects
| # | Scenario | Verify |
|---|----------|--------|
| 4.13 | `pending → cancelled` | Stock is restored (variant quantity incremented) |
| 4.14 | `confirmed → cancelled` | Stock is restored |
| 4.15 | `pending → confirmed` | Stock stays deducted (was already deducted at create) |

#### Invalid Transitions
| # | Scenario | Expected |
|---|----------|----------|
| 4.16 | `pending → completed` (skip confirmed) | `400` — invalid transition |
| 4.17 | `cancelled → confirmed` (terminal state) | `400` — invalid transition |
| 4.18 | `failed → confirmed` (terminal state) | `400` — invalid transition |
| 4.19 | `returned → completed` (terminal state) | `400` — invalid transition |

#### Validation
| # | Scenario | Expected |
|---|----------|----------|
| 4.20 | `pending → confirmed` without `transactionId` | `400` — transactionId required |
| 4.21 | `pending → failed` without `failureReason` | `400` — failureReason required |
| 4.22 | Invalid status value (`pending → shipped`) | `400` — invalid status |
| 4.23 | Seller updating order from different store | `404` — not found |

---

### 5. Cancel Order (`POST /api/order/:id/cancel`)

#### Happy Path
| # | Scenario | Verify |
|---|----------|--------|
| 5.1 | Cancel `pending` order | Status → `cancelled`, inventory restored, `order_history` entry created |
| 5.2 | Cancel `confirmed` order | Status → `cancelled`, inventory restored |
| 5.3 | Cancel with reason | Reason stored in `order_history.note` |
| 5.4 | Cancel without reason | Succeeds, `note` is null |

#### Cart Side Effects
| # | Scenario | Verify |
|---|----------|--------|
| 5.5 | Cancel pending order | Cart reverted to `active` (checkout items available for retry) |
| 5.6 | Cancel confirmed order | Cart stays `converted` (user needs new cart) |

#### Access Control
| # | Scenario | Expected |
|---|----------|----------|
| 5.7 | Customer cancels another user's order | `404` — not found |
| 5.8 | Seller tries to use customer cancel endpoint | `401/403` — wrong role |
| 5.9 | Unauthenticated request | `401` |

#### Invalid State
| # | Scenario | Expected |
|---|----------|----------|
| 5.10 | Cancel `completed` order | `400` — order cannot be cancelled |
| 5.11 | Cancel already `cancelled` order | `400` — already cancelled |
| 5.12 | Cancel `failed` order | `400` — order cannot be cancelled |
| 5.13 | Cancel `returned` order | `400` — order cannot be cancelled |

---

### Cross-Cutting Test Scenarios

#### End-to-End Flow
| # | Scenario | Verify |
|---|----------|--------|
| E2E.1 | Full happy path: Add to cart → Create order → Confirm → Complete | All statuses, all side effects, all history entries correct |
| E2E.2 | Cart → Order → Payment fails → Cart active → Retry → Success | Cart transitions: `active → checkout → active → checkout → converted` |
| E2E.3 | Cart with promotion → Order → Cancel → Re-add to cart → Reorder | Promotions re-evaluated fresh on second order |

#### Concurrency
| # | Scenario | Verify |
|---|----------|--------|
| C.1 | Two simultaneous create-order requests for same user | Only one succeeds (cart lock prevents double order) |
| C.2 | Update status and cancel at the same time | Only one transition succeeds (DB-level row lock) |

#### Data Integrity
| # | Scenario | Verify |
|---|----------|--------|
| D.1 | Product price changed after order created | `order_item.unit_price_cents` still has original price (snapshot) |
| D.2 | Promotion deleted after order created | `order_applied_promotion` still has promotion name and discount (snapshot) |
| D.3 | User address changed after order created | `order_address` still has original address (snapshot) |

