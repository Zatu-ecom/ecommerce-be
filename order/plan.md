# Order API — Phased Implementation Plan

> **Approach:** TDD — write tests first, then implement. Each phase builds on the previous one. Follow existing project patterns exactly.

---

## Project Structure Reference

```
order/
├── container.go                        # Module registration (add OrderModule)
├── entity/
│   ├── cart.go                         # [MODIFY] Add CartStatus, OrderID fields
│   ├── order.go                       # [EXISTS] Order entity
│   ├── order_item.go                  # [EXISTS] OrderItem entity
│   ├── order_address.go               # [EXISTS] OrderAddress entity
│   ├── order_applied_promotion.go     # [EXISTS]
│   ├── order_applied_coupon.go        # [EXISTS]
│   ├── order_item_applied_promotion.go# [EXISTS]
│   └── order_history.go               # [NEW] OrderHistory entity
├── error/
│   ├── cart_error.go                  # [EXISTS]
│   └── order_error.go                 # [NEW] Order-specific errors
├── handler/
│   ├── cart_handler.go                # [EXISTS]
│   └── order_handler.go              # [NEW] Order API handlers
├── model/
│   ├── cart_model.go                  # [EXISTS]
│   └── order_model.go                # [NEW] Request/Response DTOs
├── repository/
│   ├── cart_repository.go             # [MODIFY] Add status update methods
│   ├── order_repository.go            # [MODIFY] Expand with CRUD operations
│   └── order_history_repository.go    # [NEW]
├── route/
│   ├── cart_route.go                  # [EXISTS]
│   └── order_route.go                # [NEW] Order routes
├── service/
│   ├── cart_service.go                # [EXISTS]
│   └── order_service.go              # [NEW] Order business logic
├── factory/
│   ├── singleton/
│   │   ├── singleton_factory.go       # [MODIFY] Add order getters
│   │   ├── repository_factory.go      # [MODIFY] Add order repos
│   │   ├── service_factory.go         # [MODIFY] Add order service
│   │   └── handler_factory.go         # [MODIFY] Add order handler
│   └── cart_response_builder.go       # [EXISTS]
├── utils/
│   ├── constant/
│   │   ├── cart_constants.go          # [EXISTS]
│   │   └── order_constants.go         # [NEW] Order messages
│   └── order_number.go         # [NEW] Order number generator
├── ORDER_API_DESIGN.md                # [EXISTS] API design doc
└── plan.md                            # This file

common/
├── helper/
│   └── helper.go                      # [EXISTS] Cross-module helpers (put here if used by multiple modules)
├── error/
│   ├── app_error.go                   # [EXISTS] AppError struct
│   └── common_errors.go              # [EXISTS] Shared errors
└── db/
    └── transaction_manager.go         # [EXISTS] db.WithTransaction / db.WithTransactionResult
```

### Where to Put Things

| What | Where | Why |
|------|-------|-----|
| Used by **only** order module | `order/utils/` | Module-scoped |
| Used by **multiple** modules | `common/helper/` | Shared |
| Error codes | `order/error/order_error.go` | Module-scoped (mirrors `cart_error.go`) |
| Message strings | `order/utils/constant/order_constants.go` | Module-scoped (mirrors `cart_constants.go`) |
| Order number generation | `order/utils/order_number.go` | Order-specific utility |
| DB transactions | Use `db.WithTransaction(ctx, fn)` | Already in `common/db/transaction_manager.go` |

---

## Best Practices

### 1. Transactions
- **Always wrap Create Order in `db.WithTransactionResult`** — it touches order, order_items, order_address, order_applied_promotion, order_item_applied_promotion, order_history, cart status, and inventory. ALL must succeed or ALL must rollback.
- Use `db.DB(ctx)` inside the transaction — it automatically uses the transaction-scoped connection.
- Cancel Order and Update Order Status also need transactions (status + inventory + history).

### 2. Error Handling
- Define structured errors in `order/error/order_error.go` using `commonError.AppError`.
- Use error functions for dynamic messages: `ErrInvalidTransition(from, to string)`.
- Use static vars for fixed errors: `var ErrCartNotActive = &AppError{...}`.
- Handler calls `h.HandleError(c, err, fallbackMsg)` — it detects `AppError` and returns the correct HTTP status.

### 3. Repository Pattern
- Repository handles **only** DB queries — no business logic.
- Service handles **all** business rules, validations, and orchestration.
- Handler handles **only** HTTP concerns (binding, auth context, response formatting).

### 4. Factory Singleton Pattern
- Follow the existing pattern: `RepositoryFactory` → `ServiceFactory` → `HandlerFactory`.
- Add new getters in `singleton_factory.go` for order repos/services/handlers.
- Cross-module dependencies (promotion, inventory, product, user) are injected in `ServiceFactory.initialize()`.

### 5. Context Propagation
- Always pass `context.Context` (from `gin.Context`) through handler → service → repository.
- Auth context: use `auth.GetUserIDFromContext(c)` and `auth.GetSellerIDFromContext(c)`.

### 6. Snapshots (Immutable Data)
- Order items snapshot **current** product name, price, image — NOT a foreign key reference to mutable data.
- Applied promotions snapshot promotion name, type, discount — even if the promotion is later deleted.
- Addresses are copied to `order_address` — not linked to user's mutable address table.

---

## Phase 1: DB Migration & Entities

**Goal:** Create the order_history table, add status to cart, update entities.

### 1.1 — New Migration: `016_add_cart_status_and_order_history.sql`
- [ ] Add `status` column to `cart` (VARCHAR(20), NOT NULL, DEFAULT 'active')
- [ ] Add `order_id` column to `cart` (BIGINT, nullable, references order.id)
- [ ] Drop old unique index on `cart.user_id`
- [ ] Create partial unique index: `CREATE UNIQUE INDEX idx_cart_user_id_active ON cart(user_id) WHERE status = 'active'`
- [ ] Create index: `idx_cart_status_updated_at ON cart(status, updated_at)`
- [ ] Create `order_history` table (as designed in ORDER_API_DESIGN.md)
- [ ] Create indexes on `order_history`

### 1.2 — Update Cart Entity (`order/entity/cart.go`)
- [ ] Add `CartStatus` type with constants: `CART_STATUS_ACTIVE`, `CART_STATUS_CHECKOUT`, `CART_STATUS_CONVERTED`
- [ ] Add `Status` field to Cart struct
- [ ] Add `OrderID` field to Cart struct (nullable)

### 1.3 — New Entity (`order/entity/order_history.go`)
- [ ] Create `OrderHistory` struct matching the DB schema
- [ ] Fields: `OrderID`, `FromStatus`, `ToStatus`, `ChangedByUserID`, `ChangedByRole`, `TransactionID`, `FailureReason`, `Note`, `Metadata`

---

## Phase 2: Error Constants & Messages

**Goal:** Define all error types and message strings before writing any logic.

### 2.1 — Order Errors (`order/error/order_error.go`)
- [ ] `ErrCartNotActive` — cart not in active status
- [ ] `ErrCartEmpty` — cart has no items
- [ ] `ErrCartAlreadyInCheckout` — cart locked for another checkout
- [ ] `ErrOrderNotFound` — order not found (or forbidden)
- [ ] `ErrInvalidOrderStatus` — invalid status value
- [ ] `ErrInvalidStatusTransition(from, to)` — invalid transition
- [ ] `ErrTransactionIDRequired` — transactionId required for confirmed
- [ ] `ErrFailureReasonRequired` — failureReason required for failed
- [ ] `ErrOrderNotCancellable` — order not in cancellable state
- [ ] `ErrAddressNotFound` — shipping/billing address not found
- [ ] `ErrInvalidFulfillmentType` — invalid fulfillment type

### 2.2 — Order Constants (`order/utils/constant/order_constants.go`)
- [ ] Handler success messages: `ORDER_CREATED_MSG`, `ORDER_FETCHED_MSG`, `ORDERS_LISTED_MSG`, `ORDER_STATUS_UPDATED_MSG`, `ORDER_CANCELLED_MSG`
- [ ] Handler error messages: `FAILED_TO_CREATE_ORDER_MSG`, `FAILED_TO_FETCH_ORDER_MSG`, `FAILED_TO_LIST_ORDERS_MSG`, `FAILED_TO_UPDATE_ORDER_STATUS_MSG`, `FAILED_TO_CANCEL_ORDER_MSG`

---

## Phase 3: Models (Request/Response DTOs)

**Goal:** Define all request and response structs.

### 3.1 — Request Models (`order/model/order_model.go`)
- [ ] `CreateOrderRequest` — `ShippingAddressID`, `BillingAddressID`, `FulfillmentType`, `Metadata`
- [ ] `UpdateOrderStatusRequest` — `Status`, `TransactionID`, `Note`, `FailureReason`, `Metadata`
- [ ] `CancelOrderRequest` — `Reason`
- [ ] `ListOrdersRequest` — `Page`, `PageSize`, `Status`, `SortBy`, `SortOrder`, `FromDate`, `ToDate`, `Search`

### 3.2 — Response Models (`order/model/order_model.go`)
- [ ] `OrderResponse` — full order response with items, addresses, promotions, customer
- [ ] `OrderItemResponse` — item with promotion breakdown
- [ ] `OrderAddressResponse`
- [ ] `OrderPromotionResponse` — order-level promotion
- [ ] `ItemPromotionBreakdownResponse` — item-level promotion
- [ ] `OrderListResponse` — lightweight summary for listing
- [ ] `OrderCustomerResponse` — customer info (name, email, phone)
- [ ] `UpdateStatusResponse` — status change result
- [ ] `PaginatedOrdersResponse` — paginated list wrapper

---

## Phase 4: Repository Layer

**Goal:** Database operations only — no business logic.

### 4.1 — Update Cart Repository (`order/repository/cart_repository.go`)
- [ ] `UpdateCartStatus(ctx, cartID, status)` — update cart status
- [ ] `SetCartOrderID(ctx, cartID, orderID)` — link cart to order
- [ ] `FindActiveCartByUserID(ctx, userID)` — find active cart (WHERE status='active')
- [ ] `CreateNewActiveCart(ctx, userID)` — create new empty active cart

### 4.2 — Expand Order Repository (`order/repository/order_repository.go`)
- [ ] `CreateOrder(ctx, order *entity.Order) error`
- [ ] `CreateOrderItems(ctx, items []entity.OrderItem) error`
- [ ] `CreateOrderAddresses(ctx, addresses []entity.OrderAddress) error`
- [ ] `CreateOrderAppliedPromotions(ctx, promos []entity.OrderAppliedPromotion) error`
- [ ] `CreateOrderItemAppliedPromotions(ctx, promos []entity.OrderItemAppliedPromotion) error`
- [ ] `FindOrderByID(ctx, orderID uint) (*entity.Order, error)` — with preloads (items, addresses, promotions)
- [ ] `FindOrdersByUserID(ctx, userID uint, filters) ([]entity.Order, total int64, error)`
- [ ] `FindOrdersBySellerID(ctx, sellerID uint, filters) ([]entity.Order, total int64, error)`
- [ ] `FindAllOrders(ctx, filters) ([]entity.Order, total int64, error)` — admin
- [ ] `UpdateOrderStatus(ctx, orderID uint, status string) error`
- [ ] `UpdateOrderTransactionID(ctx, orderID uint, txnID string) error`
- [ ] `UpdateOrderPaidAt(ctx, orderID uint, paidAt time.Time) error`

### 4.3 — New Order History Repository (`order/repository/order_history_repository.go`)
- [ ] `CreateHistoryEntry(ctx, entry *entity.OrderHistory) error`
- [ ] `FindHistoryByOrderID(ctx, orderID uint) ([]entity.OrderHistory, error)`

---

## Phase 5: Utilities

**Goal:** Helper functions used by the service layer.

### 5.1 — Order Number Generator (`order/utils/order_number.go`)
- [ ] `GenerateOrderNumber(sellerID uint) string` — `ORD-<epoch_ms>-<seller_b36>-<random>`
- [ ] `generateRandomAlphanumeric(n int) string` — crypto/rand based random string
- [ ] `EncodeSellerID(sellerID uint) string` — base36 encode
- [ ] `DecodeSellerID(encoded string) (uint, error)` — base36 decode (for debugging)

### 5.2 — Status Transition Validator (`order/utils/status_transition.go`)
- [ ] `ValidTransitions` map — defines allowed `from → []to` transitions
- [ ] `IsValidTransition(from, to OrderStatus) bool`
- [ ] `RequiredFieldsForTransition(from, to) []string` — returns required fields for specific transitions

### 5.3 — Inventory Reservation Configuration
- [ ] Add `ORDER_RESERVATION_EXPIRY_MINUTES` constant (default: `30`) in `order/utils/constant/order_constants.go`
  - Read from env var at runtime: `config.GetInt("order.reservation_expiry_minutes", 30)`
  - Passed as `ExpiresInMinutes` in `CreateReservation` call

### 5.4 — Factory: Inject Reservation Service
- [ ] In `service_factory.go`, import `inventoryFactory "ecommerce-be/inventory/factory/singleton"` and inject `inventoryFactory.GetInstance().GetInventoryReservationService()` as a dependency of `OrderService`
  - Pattern already used for `promotionSvc`, `inventorySvc` in existing `service_factory.go`

---

## Phase 6: Integration Tests (TDD — Write First!)

**Goal:** Write ALL test scenarios from `ORDER_API_DESIGN.md` BEFORE implementing the service.

> **Important:** Tests will fail initially — that's the TDD approach. Implement service layer to make them pass.

> **⚠️ API-First Test Rule — Mimic Frontend Call Flow:**
> Integration tests MUST assert side effects **via API calls**, not direct DB queries.
> - ❌ `db.Where("status = ?", "converted").Find(&cart)` — do NOT query DB directly
> - ✅ `GET /api/cart` → assert the response shows a new empty active cart
> - ✅ `GET /api/order/:id` → assert items, status, promotions from the response
> - ✅ `GET /api/inventory/...` → assert available/reserved quantities via the inventory API
>
> Direct DB assertions are allowed **only** when no API exists to expose that data (e.g. `order_history` rows when there is no history-list endpoint yet).

### ⚠️ Test Pattern — MUST Follow

**All order integration tests MUST follow the `suite.Suite` pattern used in `test/integration/promotion/bundle_test.go`.** Do NOT use the flat `TestXxx(t *testing.T)` + `t.Run(...)` style.

Reference file: `test/integration/promotion/bundle_test.go`

Required skeleton:

```go
package order

import (
    "testing"

    "ecommerce-be/test/integration/helpers"
    "ecommerce-be/test/integration/setup"

    "github.com/stretchr/testify/suite"
)

// OrderSuite holds all shared state for order integration tests
type OrderSuite struct {
    suite.Suite
    containers *setup.TestContainers
    server     *setup.TestServer
    client     *helpers.APIClient
    // role-specific clients
    customerClient *helpers.APIClient
    sellerClient   *helpers.APIClient
}

// SetupSuite runs once before all tests — start containers, run migrations/seeds, boot server
func (s *OrderSuite) SetupSuite() {
    s.containers = setup.SetupTestContainers(s.T())
    s.containers.RunAllMigrations(s.T())
    s.containers.RunAllSeeds(s.T())
    s.server = setup.SetupTestServer(s.T(), s.containers.DB, s.containers.RedisClient)
    s.client = helpers.NewAPIClient(s.server)
    // setup auth tokens for customer and seller
}

// TearDownSuite runs once after all tests — cleanup containers
func (s *OrderSuite) TearDownSuite() {
    s.containers.Cleanup(s.T())
}

// SetupTest runs before EACH test — reset state (e.g. clear orders, reset cart)
func (s *OrderSuite) SetupTest() {
    // truncate orders, order_items, order_history, reset cart status between tests
}

// TestOrderSuite is the entry point — runs the suite
func TestOrderSuite(t *testing.T) {
    suite.Run(t, new(OrderSuite))
}

// Each test scenario is a method on the suite:
func (s *OrderSuite) TestCreateOrder_HappyPath() { ... }
func (s *OrderSuite) TestCreateOrder_CartEmpty() { ... }
func (s *OrderSuite) TestCreateOrder_InsufficientStock() { ... }
```

Key rules:
- One suite per logical API group (e.g. `order_create_test.go`, `order_status_test.go`)
- Shared helpers (`createTestOrder()`, `addItemToCart()`) as **methods on the suite**, not global functions
- Use `s.Assert()` / `s.Require()` instead of bare `assert`/`require`
- DB-level assertions (verify cart status, inventory count, order_history rows) go in each test — not just HTTP response checks

### 6.1 — Test Setup
- [ ] Create `test/integration/order/` directory
- [ ] `setup_suite_test.go` — suite struct, `SetupSuite`, `TearDownSuite`, `SetupTest`, helper methods
- [ ] `order_helpers_test.go` — `createTestOrder()`, `addItemToCart()`, `loginAsCustomer()`, `loginAsSeller()`, DB assertion helpers

### 6.2 — Create Order Tests (`order_create_test.go`) — Scenarios 1.1–1.26
- [ ] **Happy path** — assert via APIs:
  - `POST /api/order` returns 201 with `status: pending`
  - `GET /api/order/:id` shows correct items, prices, promotions, addresses
  - `GET /api/cart` → user now has a new empty `active` cart (old cart is `converted`)
  - `GET /api/inventory/...` → `availableQuantity` decreased, `reservedQuantity` increased
  - DB assertion: `order_history` row exists (`from_status: null`, `to_status: pending`)
- [ ] **Cart & stock validation** — empty cart, out-of-stock (reservation service rejects), concurrent checkout
- [ ] **Address validation** — missing address, address from different user
- [ ] **Promotion edge cases** — expired promo, bundle, stackable, non-stackable
- [ ] **Rollback** — on any failure, `GET /api/cart` still shows original `active` cart; inventory unchanged

### 6.3 — Get Order Tests (`order_get_test.go`) — Scenarios 2.1–2.9
- [ ] Happy path (customer view, seller view)
- [ ] Access control tests
- [ ] Customer info visibility tests

### 6.4 — List Orders Tests (`order_list_test.go`) — Scenarios 3.1–3.14
- [ ] Role-based scoping tests
- [ ] Pagination tests
- [ ] Filter tests

### 6.5 — Update Order Status Tests (`order_status_test.go`) — Scenarios 4.1–4.23
- [ ] **Valid transitions** — assert via APIs:
  - `PATCH /api/order/:id/status` returns 200
  - `GET /api/order/:id` shows new status
  - `GET /api/inventory/...` → check reservation status per transition:
    - `→ confirmed`: `reservedQuantity` unchanged (stays reserved)
    - `→ failed` / `→ cancelled`: `availableQuantity` restored, `reservedQuantity` decreased
    - `→ completed`: `reservedQuantity` reduced, outbound transaction created (fulfilled)
  - `GET /api/cart` → after `→ failed`: user's cart back to `active`
- [ ] Order history audit — DB assertion (no history-list API yet)
- [ ] Invalid transition tests
- [ ] Validation (missing transactionId / failureReason)

### 6.6 — Cancel Order Tests (`order_cancel_test.go`) — Scenarios 5.1–5.13
- [ ] **Happy path** — assert via APIs:
  - `POST /api/order/:id/cancel` returns 200
  - `GET /api/order/:id` shows `status: cancelled`
  - `GET /api/inventory/...` → `availableQuantity` restored (reservation `CANCELLED`)
  - `GET /api/cart` → if was `pending`, user's cart is back to `active`
  - DB assertion: `order_history` row with `to_status: cancelled` and `note`
- [ ] Access control tests
- [ ] Invalid state tests (completed, already cancelled, failed)

### 6.7 — Cross-Cutting Tests (`order_e2e_test.go`) — E2E, Concurrency, Data Integrity
- [ ] End-to-end flow tests
- [ ] Concurrency tests
- [ ] Data integrity tests (snapshot immutability)

---

## Phase 7: Service Layer (Make Tests Pass!)

**Goal:** Implement the business logic. This is where ALL the orchestration happens.

### 7.1 — Order Service Interface (`order/service/order_service.go`)
```go
type OrderService interface {
    CreateOrder(ctx context.Context, userID, sellerID uint, req model.CreateOrderRequest) (*model.OrderResponse, error)
    GetOrderByID(ctx context.Context, userID uint, role string, orderID uint) (*model.OrderResponse, error)
    ListOrders(ctx context.Context, userID uint, role string, filters model.ListOrdersRequest) (*model.PaginatedOrdersResponse, error)
    UpdateOrderStatus(ctx context.Context, sellerID uint, orderID uint, req model.UpdateOrderStatusRequest) (*model.UpdateStatusResponse, error)
    CancelOrder(ctx context.Context, userID uint, orderID uint, req model.CancelOrderRequest) (*model.UpdateStatusResponse, error)
}
```

### 7.2 — CreateOrder Logic
- [ ] Find active cart → validate not empty
- [ ] Lock cart (`active → checkout`)
- [ ] Re-evaluate promotions
- [ ] Generate order number
- [ ] **Wrap in `db.WithTransactionResult`:**
  - Create order entity (status: `pending`)
  - Create order items (snapshot prices, names, images)
  - Create order addresses (snapshot)
  - Create order applied promotions (snapshot)
  - Create order item applied promotions (snapshot)
  - Create `order_history` entry (`null → pending`)
  - Call `inventoryReservationSvc.CreateReservation(ctx, sellerID, ReservationRequest{ReferenceId: orderID, ExpiresInMinutes: config, Items: cartItems})`
    - Service validates variant ownership by seller
    - Service validates stock availability across locations by priority
    - Moves `available_quantity → reserved_quantity`
    - Schedules Redis TTL for auto-expiry (`ExpireScheduleReservation` is called automatically by the scheduler when TTL hits)
  - Update cart (`checkout → converted`, set `order_id`)
  - Create new empty `active` cart for user
- [ ] On error: revert cart (`checkout → active`)

> **Note:** If reservation fails due to insufficient stock, the transaction rolls back and cart reverts to `active`. No order is created.

### 7.3 — GetOrderByID Logic
- [ ] Fetch order with preloads (items, addresses, promotions)
- [ ] Apply role-based access check (customer: user_id match, seller: seller_id match)
- [ ] Build response (include customer info for seller/admin role)

### 7.4 — ListOrders Logic
- [ ] Apply role-based query scope
- [ ] Apply filters (status, date range, search)
- [ ] Apply pagination and sorting
- [ ] Build lightweight response (no items/addresses)
- [ ] Include customer info for seller/admin role

### 7.5 — UpdateOrderStatus Logic
- [ ] Fetch order, validate seller ownership
- [ ] Validate transition: `IsValidTransition(current, target)`
- [ ] Validate required fields (`transactionId`, `failureReason`)
- [ ] **Wrap in `db.WithTransaction`:**
  - Update order status
  - Set `paidAt` if → `confirmed`
  - Set `transactionId` if provided
  - Call `inventoryReservationSvc.UpdateReservationStatus(ctx, sellerID, {ReferenceId: orderID, Status: ...})`
    - `→ confirmed` → `CONFIRMED` (keeps reserved, no inventory change)
    - `→ failed` → `CANCELLED` (releases `reserved → available`)
    - `→ cancelled` → `CANCELLED` (releases `reserved → available`)
    - `→ completed` → `FULFILLED` (marks as outbound — permanent deduction at shipment)
  - Revert cart to `active` if → `failed`
  - Create `order_history` entry
- [ ] Return update result

### 7.6 — CancelOrder Logic
- [ ] Validate order belongs to user
- [ ] Validate order is cancellable (`pending` or `confirmed`)
- [ ] **Wrap in `db.WithTransaction`:**
  - Update status → `cancelled`
  - Call `inventoryReservationSvc.UpdateReservationStatus(ctx, sellerID, {ReferenceId: orderID, Status: CANCELLED})` → releases `reserved → available`
  - Revert cart to `active` if was `pending`
  - Create `order_history` entry with reason

---

## Phase 8: Handler Layer

**Goal:** HTTP layer — bind requests, call service, return responses.

### 8.1 — Order Handler (`order/handler/order_handler.go`)
- [ ] `CreateOrder(c *gin.Context)` — bind `CreateOrderRequest`, call service, return 201
- [ ] `GetOrderByID(c *gin.Context)` — parse `:id`, get role from context, call service
- [ ] `ListOrders(c *gin.Context)` — parse query params, get role, call service
- [ ] `UpdateOrderStatus(c *gin.Context)` — bind `UpdateOrderStatusRequest`, call service
- [ ] `CancelOrder(c *gin.Context)` — bind `CancelOrderRequest`, call service

### Pattern (mirrors `cart_handler.go`):
```go
func (h *OrderHandler) CreateOrder(c *gin.Context) {
    userID, _ := auth.GetUserIDFromContext(c)
    sellerID, _ := auth.GetSellerIDFromContext(c)
    var req model.CreateOrderRequest
    if err := h.BindJSON(c, &req); err != nil { ... }
    resp, err := h.orderService.CreateOrder(c, userID, sellerID, req)
    if err != nil { h.HandleError(c, err, constant.FAILED_TO_CREATE_ORDER_MSG); return }
    h.Success(c, http.StatusCreated, constant.ORDER_CREATED_MSG, resp)
}
```

---

## Phase 9: Routes & Wiring

**Goal:** Register routes and wire everything through the factory.

### 9.1 — Order Routes (`order/route/order_route.go`)
- [ ] `POST /api/order` — CustomerAuth → `CreateOrder`
- [ ] `GET /api/order` — Auth (any role) → `ListOrders`
- [ ] `GET /api/order/:id` — Auth (any role) → `GetOrderByID`
- [ ] `PATCH /api/order/:id/status` — SellerAuth → `UpdateOrderStatus`
- [ ] `POST /api/order/:id/cancel` — CustomerAuth → `CancelOrder`

### 9.2 — Factory Updates
- [ ] `repository_factory.go` — add `GetOrderRepository()`, `GetOrderHistoryRepository()`
- [ ] `service_factory.go` — add `GetOrderService()`
- [ ] `handler_factory.go` — add `GetOrderHandler()`
- [ ] `singleton_factory.go` — add delegate methods for all above

### 9.3 — Container Update (`order/container.go`)
- [ ] Register `route.NewOrderModule()` in `addModules()`

---

## Phase 10: Verification & Cleanup

- [ ] Run all integration tests: `make test`
- [ ] Verify all TDD test scenarios pass
- [ ] Verify no regressions in existing cart tests
- [ ] Check DB migration works on fresh database
- [ ] Review error messages for consistency
