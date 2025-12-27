# Inventory Reservation API - PRD

> **Version**: 1.0  
> **Created**: December 13, 2025  
> **Status**: Ready for Implementation

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Business Flow](#business-flow)
3. [API Specifications](#api-specifications)
4. [Database Changes](#database-changes)
5. [Redis Expiry Implementation](#redis-expiry-implementation)
6. [Error Handling](#error-handling)
7. [Implementation Checklist](#implementation-checklist)

---

## 🎯 Overview

### Purpose

The Reservation API allows temporary stock holds during checkout, preventing overselling while customers complete payment.

### Key Features

- **Create Reservation**: Reserve stock for cart/checkout (15 min default)
- **Update Reservation**: Confirm or cancel with single API
- **Auto-Expiry**: Redis-based automatic expiration
- **Stock Validation**: Prevent reservation if insufficient stock
- **Location-agnostic**: Reserve at variant level, location decided at fulfillment

### API Summary

| Method  | Endpoint                                   | Purpose                    |
| ------- | ------------------------------------------ | -------------------------- |
| `POST`  | `/api/inventory/reservations`              | Create reservation         |
| `GET`   | `/api/inventory/reservations/:referenceId` | Get reservation details    |
| `PATCH` | `/api/inventory/reservations/:referenceId` | Confirm or Cancel (action) |

---

## 🔄 Business Flow

### Happy Path: Checkout → Payment → Order

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. CHECKOUT START                                                │
│    Customer clicks "Checkout"                                    │
│    └── POST /api/inventory/reservations                         │
│        ├── Check available stock (quantity - reserved_quantity) │
│        ├── If insufficient → Return error                        │
│        ├── Create InventoryReservation (status=PENDING)         │
│        ├── Update inventory.reserved_quantity += N              │
│        ├── Create InventoryTransaction (type=RESERVED)          │
│        ├── Set Redis key with 15 min TTL                        │
│        └── Return reservation details                            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼ (Within 15 minutes)
┌─────────────────────────────────────────────────────────────────┐
│ 2. PAYMENT SUCCESS                                               │
│    Payment gateway confirms payment                              │
│    └── PATCH /api/inventory/reservations/:referenceId           │
│        ├── Body: { "action": "confirm", "orderId": "ORD-456" } │
│        ├── Find reservations by referenceId (cart ID)           │
│        ├── Update status: PENDING → CONFIRMED                   │
│        ├── Update referenceId: CART-123 → ORD-456               │
│        ├── Delete Redis expiry key (prevent expiration)         │
│        └── Return success                                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼ (Later: Order fulfillment)
┌─────────────────────────────────────────────────────────────────┐
│ 3. ORDER SHIPPED                                                 │
│    Warehouse ships the order                                     │
│    └── (Existing API) POST /api/inventory/manage                │
│        ├── type: SALE                                           │
│        ├── quantity -= N                                        │
│        ├── reserved_quantity -= N                               │
│        └── Create InventoryTransaction (type=SALE)              │
└─────────────────────────────────────────────────────────────────┘
```

### Expiry Path: Checkout → Timeout → Stock Released

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. CHECKOUT START (Same as above)                                │
│    └── Creates reservation with 15 min expiry                   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼ (15 minutes pass, no payment)
┌─────────────────────────────────────────────────────────────────┐
│ 2. REDIS KEY EXPIRES                                             │
│    Redis notifies via keyspace events                            │
│    └── Expiry Handler triggered                                  │
│        ├── Find reservation by ID                                │
│        ├── If status != PENDING → Skip (already confirmed)      │
│        ├── Update status: PENDING → EXPIRED                     │
│        ├── Update inventory.reserved_quantity -= N              │
│        ├── Create InventoryTransaction (type=RELEASED)          │
│        └── Log expiration                                        │
└─────────────────────────────────────────────────────────────────┘
```

### Cancel Path: User Cancels Checkout

```
┌─────────────────────────────────────────────────────────────────┐
│ MANUAL CANCEL                                                    │
│    User cancels or closes checkout                               │
│    └── PATCH /api/inventory/reservations/:referenceId           │
│        ├── Body: { "action": "cancel", "reason": "..." }       │
│        ├── Find reservations by referenceId                      │
│        ├── Update status: PENDING → CANCELLED                   │
│        ├── Update inventory.reserved_quantity -= N              │
│        ├── Create InventoryTransaction (type=RELEASED)          │
│        ├── Delete Redis expiry key                               │
│        └── Return success                                        │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📡 API Specifications

### API 1: Create Reservation

**Endpoint:** `POST /api/inventory/reservations`  
**Auth:** Customer (JWT)  
**Purpose:** Reserve stock when customer starts checkout

#### Request

```json
{
  "referenceId": "CART-12345",
  "items": [
    {
      "variantId": 1,
      "quantity": 2
    },
    {
      "variantId": 3,
      "quantity": 1
    }
  ],
  "expiresInMinutes": 15
}
```

| Field             | Type   | Required | Description                        |
| ----------------- | ------ | -------- | ---------------------------------- |
| referenceId       | string | Yes      | Cart/Checkout session ID           |
| items             | array  | Yes      | Items to reserve                   |
| items[].variantId | uint   | Yes      | Product variant ID                 |
| items[].quantity  | int    | Yes      | Quantity to reserve                |
| expiresInMinutes  | int    | No       | Expiry time (default: 15, max: 60) |

> **Note:** Location is NOT specified during reservation. Stock is reserved at variant level across all locations. Location selection happens during order fulfillment based on priority, proximity, and availability.

#### Response (Success - 201)

```json
{
  "success": true,
  "message": "Stock reserved successfully",
  "data": {
    "referenceId": "CART-12345",
    "expiresAt": "2025-12-13T10:30:00Z",
    "reservations": [
      {
        "id": 101,
        "variantId": 1,
        "quantity": 2,
        "status": "PENDING",
        "totalAvailableAfterReserve": 48
      },
      {
        "id": 102,
        "variantId": 3,
        "quantity": 1,
        "status": "PENDING",
        "totalAvailableAfterReserve": 24
      }
    ]
  }
}
```

#### Response (Error - Insufficient Stock)

```json
{
  "success": false,
  "message": "Insufficient stock for reservation",
  "error": {
    "code": "INSUFFICIENT_STOCK",
    "details": [
      {
        "variantId": 1,
        "requested": 5,
        "totalAvailable": 3
      }
    ]
  }
}
```

#### Business Rules

1. **Stock Check**: `totalAvailable = SUM(quantity - reserved_quantity)` across ALL locations for variant
2. **All or Nothing**: If any item fails, entire reservation fails (atomic)
3. **Duplicate Check**: Same referenceId with PENDING status → Return existing
4. **Max Items**: Maximum 50 items per reservation
5. **Expiry Limits**: Min 5 minutes, Max 60 minutes, Default 15 minutes
6. **Location Selection**: Happens at fulfillment time, not reservation time

---

### API 2: Update Reservation (Confirm/Cancel)

**Endpoint:** `PATCH /api/inventory/reservations/:referenceId`  
**Auth:** System/Internal (or Customer JWT)  
**Purpose:** Confirm reservation (order created) or cancel (checkout abandoned)

#### Path Parameters

| Parameter   | Type   | Description                |
| ----------- | ------ | -------------------------- |
| referenceId | string | Cart/Checkout reference ID |

#### Request - Confirm

```json
{
  "action": "confirm",
  "orderId": "ORD-2024-001"
}
```

#### Request - Cancel

```json
{
  "action": "cancel",
  "reason": "Customer abandoned checkout"
}
```

| Field   | Type   | Required                 | Description             |
| ------- | ------ | ------------------------ | ----------------------- |
| action  | string | Yes                      | `confirm` or `cancel`   |
| orderId | string | Yes (if action=confirm)  | The created order ID    |
| reason  | string | No (optional for cancel) | Reason for cancellation |

#### Response - Confirm (Success - 200)

```json
{
  "success": true,
  "message": "Reservation confirmed",
  "data": {
    "referenceId": "ORD-2024-001",
    "previousReferenceId": "CART-12345",
    "status": "CONFIRMED",
    "itemsConfirmed": 2
  }
}
```

#### Response - Cancel (Success - 200)

```json
{
  "success": true,
  "message": "Reservation cancelled, stock released",
  "data": {
    "referenceId": "CART-12345",
    "status": "CANCELLED",
    "itemsReleased": 2
  }
}
```

#### Business Rules - Confirm

1. **Status Check**: Only PENDING reservations can be confirmed
2. **Update Reference**: Change referenceId from CART-xxx to ORD-xxx
3. **Remove Expiry**: Delete Redis key to prevent auto-expiration
4. **No Transaction**: Confirmation doesn't create inventory transaction (stock already reserved)

#### Business Rules - Cancel

1. **Status Check**: Only PENDING reservations can be cancelled
2. **Release Stock**: Decrease reserved_quantity for each item
3. **Create Transaction**: RELEASED transaction for audit
4. **Remove Expiry**: Delete Redis key

---

### API 3: Get Reservation

**Endpoint:** `GET /api/inventory/reservations/:referenceId`  
**Auth:** Customer/Seller (JWT)  
**Purpose:** Get reservation status and details

#### Path Parameters

| Parameter   | Type   | Description         |
| ----------- | ------ | ------------------- |
| referenceId | string | Cart ID or Order ID |

#### Response (Success - 200)

````json
{
  "success": true,
  "data": {
    "referenceId": "CART-12345",
    "status": "PENDING",
    "expiresAt": "2025-12-13T10:30:00Z",
    "remainingSeconds": 542,
    "items": [
      {
        "id": 101,
        "variantId": 1,
        "quantity": 2,
        "status": "PENDING"
      }
    ]
  }
}

---

## 🗄️ Database Changes

### Entity: InventoryReservation (Updated - Variant Level)

```go
type InventoryReservation struct {
    db.BaseEntity
    VariantID   uint              `gorm:"not null;index"`   // Reserve at variant level, NOT inventory/location
    ReferenceID string            `gorm:"not null;index"`   // CART-xxx or ORD-xxx
    Quantity    int               `gorm:"not null"`
    ExpiresAt   time.Time         `gorm:"not null;index"`
    Status      ReservationStatus `gorm:"default:'PENDING';index"`

    // Relations
    Variant     *ProductVariant   `gorm:"foreignKey:VariantID"`
}
````

### Status Values

```go
const (
    ResPending   ReservationStatus = "PENDING"   // Active, waiting for confirmation
    ResConfirmed ReservationStatus = "CONFIRMED" // Converted to order
    ResExpired   ReservationStatus = "EXPIRED"   // Auto-expired, stock released
    ResCancelled ReservationStatus = "CANCELLED" // Manually cancelled
)
```

### Migration: 008_create_inventory_reservation_table.sql

```sql
-- Create inventory_reservation table
-- Note: Reservation is at VARIANT level, not inventory/location level
-- Location selection happens at fulfillment time based on priority
CREATE TABLE IF NOT EXISTS inventory_reservation (
    id BIGSERIAL PRIMARY KEY,
    variant_id BIGINT NOT NULL REFERENCES product_variant(id),  -- Variant level, NOT inventory
    reference_id VARCHAR(100) NOT NULL,
    quantity INT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT chk_quantity_positive CHECK (quantity > 0),
    CONSTRAINT chk_status_valid CHECK (status IN ('PENDING', 'CONFIRMED', 'EXPIRED', 'CANCELLED'))
);

-- Indexes
CREATE INDEX idx_reservation_variant_id ON inventory_reservation(variant_id);
CREATE INDEX idx_reservation_reference_id ON inventory_reservation(reference_id);
CREATE INDEX idx_reservation_status ON inventory_reservation(status);
CREATE INDEX idx_reservation_expires_at ON inventory_reservation(expires_at) WHERE status = 'PENDING';

-- Composite index for expiry queries
CREATE INDEX idx_reservation_pending_expiry ON inventory_reservation(status, expires_at)
    WHERE status = 'PENDING';
```

### Stock Availability Query

When checking available stock for a variant across ALL locations:

```sql
-- Get total available stock for a variant (across all locations)
SELECT
    v.id as variant_id,
    COALESCE(SUM(i.quantity - i.reserved_quantity), 0) as total_available
FROM product_variant v
LEFT JOIN inventory i ON i.variant_id = v.id
WHERE v.id = ?
GROUP BY v.id;
```

---

## 🔴 Redis Expiry Implementation

### Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                     REDIS EXPIRY FLOW                            │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  1. CREATE RESERVATION                                           │
│     └── SET reservation:expire:{id} {reservationId} EX 900       │
│         (Key expires in 900 seconds = 15 minutes)                │
│                                                                   │
│  2. REDIS KEYSPACE NOTIFICATION                                  │
│     └── When key expires, Redis publishes to:                    │
│         __keyevent@0__:expired                                   │
│                                                                   │
│  3. GO LISTENER (Background Goroutine)                           │
│     └── Subscribes to expired events                             │
│     └── Filters for "reservation:expire:*" pattern              │
│     └── Calls ExpireReservation(id)                              │
│                                                                   │
│  4. EXPIRE HANDLER                                               │
│     └── Update reservation status → EXPIRED                     │
│     └── Release reserved_quantity                                │
│     └── Create RELEASED transaction                              │
│                                                                   │
└──────────────────────────────────────────────────────────────────┘
```

### Redis Key Format

```
reservation:expire:{reservationId}
```

**Example:**

```
reservation:expire:101 → "101" (value is reservation ID)
TTL: 900 seconds (15 minutes)
```

### Enable Redis Keyspace Notifications

**Option 1: Redis CLI**

```bash
redis-cli CONFIG SET notify-keyspace-events Ex
```

**Option 2: redis.conf**

```
notify-keyspace-events Ex
```

**Option 3: On Connection (Go)**

```go
func EnableKeyspaceNotifications(client *redis.Client) error {
    return client.ConfigSet(ctx, "notify-keyspace-events", "Ex").Err()
}
```

### Go Implementation

#### 1. Redis Keys (constants)

```go
// common/constants/redis_constants.go
const (
    ReservationExpireKeyPrefix = "reservation:expire:"
    ReservationExpireChannel   = "__keyevent@0__:expired"
)

func ReservationExpireKey(reservationID uint) string {
    return fmt.Sprintf("%s%d", ReservationExpireKeyPrefix, reservationID)
}
```

#### 2. Set Expiry on Create

```go
// inventory/service/reservation_service_impl.go
func (s *ReservationService) CreateReservation(req CreateReservationRequest) (*ReservationResponse, error) {
    // ... create reservation in DB ...

    // Set Redis expiry key
    key := constants.ReservationExpireKey(reservation.ID)
    ttl := time.Until(reservation.ExpiresAt)

    err := s.redis.Set(ctx, key, reservation.ID, ttl).Err()
    if err != nil {
        // Log warning but don't fail (fallback to cron)
        logger.Warn("Failed to set Redis expiry", "reservationId", reservation.ID, "error", err)
    }

    return response, nil
}
```

#### 3. Delete Expiry on Confirm/Cancel

```go
func (s *ReservationService) ConfirmReservation(referenceId, orderId string) error {
    // ... update reservation status ...

    // Remove Redis expiry key (prevent auto-expiration)
    for _, reservation := range reservations {
        key := constants.ReservationExpireKey(reservation.ID)
        s.redis.Del(ctx, key)
    }

    return nil
}
```

#### 4. Expiry Listener (Background)

```go
// inventory/scheduler/reservation_expiry_listener.go
package scheduler

import (
    "context"
    "strings"

    "github.com/redis/go-redis/v9"
)

type ReservationExpiryListener struct {
    redis              *redis.Client
    reservationService service.ReservationService
    logger             *log.Logger
}

func NewReservationExpiryListener(redis *redis.Client, svc service.ReservationService) *ReservationExpiryListener {
    return &ReservationExpiryListener{
        redis:              redis,
        reservationService: svc,
    }
}

// Start begins listening for expired reservation keys
func (l *ReservationExpiryListener) Start(ctx context.Context) {
    // Subscribe to keyspace notifications for expired keys
    pubsub := l.redis.PSubscribe(ctx, "__keyevent@0__:expired")
    defer pubsub.Close()

    l.logger.Info("Reservation expiry listener started")

    for {
        select {
        case <-ctx.Done():
            l.logger.Info("Reservation expiry listener stopped")
            return

        case msg := <-pubsub.Channel():
            l.handleExpiredKey(ctx, msg.Payload)
        }
    }
}

func (l *ReservationExpiryListener) handleExpiredKey(ctx context.Context, key string) {
    // Check if this is a reservation expiry key
    if !strings.HasPrefix(key, constants.ReservationExpireKeyPrefix) {
        return
    }

    // Extract reservation ID
    idStr := strings.TrimPrefix(key, constants.ReservationExpireKeyPrefix)
    reservationID, err := strconv.ParseUint(idStr, 10, 64)
    if err != nil {
        l.logger.Error("Invalid reservation ID in expired key", "key", key, "error", err)
        return
    }

    // Expire the reservation
    l.logger.Info("Expiring reservation", "reservationId", reservationID)

    if err := l.reservationService.ExpireReservation(uint(reservationID)); err != nil {
        l.logger.Error("Failed to expire reservation", "reservationId", reservationID, "error", err)
    }
}
```

#### 5. Expire Reservation Logic

```go
// inventory/service/reservation_service_impl.go
func (s *ReservationService) ExpireReservation(reservationID uint) error {
    return s.db.Transaction(func(tx *gorm.DB) error {
        // 1. Find reservation
        var reservation entity.InventoryReservation
        if err := tx.First(&reservation, reservationID).Error; err != nil {
            return err
        }

        // 2. Check if still PENDING (might have been confirmed already)
        if reservation.Status != entity.ResPending {
            s.logger.Info("Reservation already processed, skipping expiry",
                "reservationId", reservationID,
                "status", reservation.Status)
            return nil
        }

        // 3. Update reservation status
        reservation.Status = entity.ResExpired
        if err := tx.Save(&reservation).Error; err != nil {
            return err
        }

        // 4. Release reserved quantity
        var inventory entity.Inventory
        if err := tx.First(&inventory, reservation.InventoryID).Error; err != nil {
            return err
        }

        beforeQty := inventory.ReservedQuantity
        inventory.ReservedQuantity -= reservation.Quantity
        if inventory.ReservedQuantity < 0 {
            inventory.ReservedQuantity = 0
        }

        if err := tx.Save(&inventory).Error; err != nil {
            return err
        }

        // 5. Create RELEASED transaction
        transaction := &entity.InventoryTransaction{
            InventoryID:    reservation.InventoryID,
            Type:           entity.TXN_RELEASED,
            Quantity:       reservation.Quantity,
            BeforeQuantity: beforeQty,
            AfterQuantity:  inventory.ReservedQuantity,
            PerformedBy:    constants.SystemUserID, // System user
            ReferenceID:    ptr(fmt.Sprintf("%d", reservation.ID)),
            ReferenceType:  ptr("RESERVATION"),
            Reason:         "Reservation expired automatically",
        }

        return tx.Create(transaction).Error
    })
}
```

#### 6. Start Listener on App Boot

```go
// main.go
func main() {
    // ... existing setup ...

    // Start reservation expiry listener
    expiryListener := scheduler.NewReservationExpiryListener(
        redisClient,
        reservationService,
    )

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go expiryListener.Start(ctx)

    // ... start HTTP server ...
}
```

### Fallback: Cron Job (Belt + Suspenders)

In case Redis misses any expirations, run a cleanup cron every 5 minutes:

```go
// inventory/scheduler/reservation_cleanup_cron.go
func (s *ReservationCleanupCron) CleanupExpiredReservations() {
    // Find all PENDING reservations past their expiry time
    var expired []entity.InventoryReservation
    s.db.Where("status = ? AND expires_at < ?", entity.ResPending, time.Now()).Find(&expired)

    for _, reservation := range expired {
        s.reservationService.ExpireReservation(reservation.ID)
    }

    if len(expired) > 0 {
        s.logger.Info("Cleanup cron expired reservations", "count", len(expired))
    }
}
```

---

## ❌ Error Handling

### Error Codes

| Code                            | HTTP Status | Message                            | When                               |
| ------------------------------- | ----------- | ---------------------------------- | ---------------------------------- |
| `INSUFFICIENT_STOCK`            | 400         | Insufficient stock for reservation | Available < requested              |
| `RESERVATION_NOT_FOUND`         | 404         | Reservation not found              | Invalid referenceId                |
| `RESERVATION_ALREADY_CONFIRMED` | 400         | Reservation already confirmed      | Trying to confirm/cancel confirmed |
| `RESERVATION_EXPIRED`           | 400         | Reservation has expired            | Trying to confirm expired          |
| `RESERVATION_CANCELLED`         | 400         | Reservation was cancelled          | Trying to confirm cancelled        |
| `INVALID_VARIANT`               | 400         | Variant not found                  | Invalid variantId                  |
| `VARIANT_NO_INVENTORY`          | 404         | No inventory records for variant   | Variant has no stock anywhere      |
| `MAX_ITEMS_EXCEEDED`            | 400         | Maximum 50 items per reservation   | Too many items                     |
| `INVALID_EXPIRY_TIME`           | 400         | Expiry must be 5-60 minutes        | Invalid expiresInMinutes           |

### Error Response Format

```json
{
  "success": false,
  "message": "Insufficient stock for reservation",
  "error": {
    "code": "INSUFFICIENT_STOCK",
    "details": [
      {
        "variantId": 1,
        "requested": 5,
        "totalAvailable": 3
      }
    ]
  }
}
```

---

## ✅ Implementation Checklist

### Phase 1: Core Setup

- [ ] Add migration for `inventory_reservation` table
- [ ] Update `InventoryReservation` entity if needed
- [ ] Add Redis constants for reservation expiry keys
- [ ] Enable Redis keyspace notifications

### Phase 2: Repository Layer

- [ ] Create `ReservationRepository` interface
- [ ] Implement `ReservationRepositoryImpl`
  - [ ] Create
  - [ ] FindByID
  - [ ] FindByReferenceID
  - [ ] UpdateStatus
  - [ ] FindExpiredPending

### Phase 3: Service Layer

- [ ] Create `ReservationService` interface
- [ ] Implement `ReservationServiceImpl`
  - [ ] CreateReservation (with stock validation)
  - [ ] ConfirmReservation
  - [ ] CancelReservation
  - [ ] GetReservation
  - [ ] ExpireReservation

### Phase 4: Handler & Routes

- [ ] Create `ReservationHandler`
  - [ ] POST /reservations (Create)
  - [ ] GET /reservations/:referenceId (Get)
  - [ ] PATCH /reservations/:referenceId (Update - confirm/cancel)
- [ ] Register routes in `reservation_routes.go`
- [ ] Update `container.go`

### Phase 5: Redis Expiry

- [ ] Create `ReservationExpiryListener`
- [ ] Start listener in `main.go`
- [ ] Create fallback cron job

### Phase 6: Testing

- [ ] Unit tests for service methods
- [ ] Integration tests for APIs
- [ ] Test expiry flow (manual + Redis)
- [ ] Test edge cases (insufficient stock, already confirmed, etc.)

### Phase 7: Documentation

- [ ] Add to Postman collection
- [ ] Update API documentation

---

## 📊 Transaction Flow Summary

| Event               | Transaction Type | quantity | reserved_quantity | Reservation Status |
| ------------------- | ---------------- | -------- | ----------------- | ------------------ |
| Create Reservation  | `RESERVED`       | -        | **+N**            | `PENDING`          |
| Confirm Reservation | ❌ None          | -        | -                 | `CONFIRMED`        |
| Cancel Reservation  | `RELEASED`       | -        | **-N**            | `CANCELLED`        |
| Expire Reservation  | `RELEASED`       | -        | **-N**            | `EXPIRED`          |
| Order Shipped       | `SALE`           | **-N**   | **-N**            | -                  |

---

## 🔗 Related Documents

- [ARCHITECTURE.md](../ARCHITECTURE.md)
- [Inventory API PRD.md](./Inventory%20API%20PRD.md)
- [CODING_STANDARDS.md](../CODING_STANDARDS.md)
