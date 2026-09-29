# 013 — Courier Fulfillment Platform · Service / Logic Design (Phase 2)

> Companion to `data-model.md`. Adapter contract, factory, orchestrators, and location-based auto-planning with a per-tenant auto-book flag. No code yet.
> Mirrored from `payment/service/payment_gateway/contract.go`, `payment/factory/payment_gateway_factory.go`, `payment/service/apply_normalized.go`, `payment/service/webhook_service.go`.

## 1. Principles

1. **Single provider contract.** Orchestrators call `CourierPartner`. They do not switch on provider codes and do not import provider packages.
2. **Provider names live in one folder.** Shiprocket URLs, status ids, auth, and webhook shape stay in `fulfillment/service/courier/shiprocket/`.
3. **One status vocabulary.** Webhook and cron switch only on `ShipmentAction`. Those strings are the shipment status (plus `ignore`). The ledger `event_type` uses the same strings.
4. **One apply path.** Webhook (`source=webhook`) and reconciler (`source=system`) both call `NormalizedApplier.Apply`.
5. **Service-to-service.** Fulfillment calls order through `FulfillmentOrderHooks`. It does not import order or inventory repositories. Progress includes shipment id and line quantities. Fulfillment does not mark a whole order delivered by order id alone.
6. **Fail-closed secrets, fail-open reads.** No encryption key → refuse to store or decrypt. Cache down → read PostgreSQL. A customer track read never fails because a courier is down, because it does not call the courier.
7. **Courier HTTP stays outside DB transactions.**
8. **Location-first auto-plan.** `OrderConfirmed` auto-creates one draft per warehouse. Booking stays seller-confirmed unless the tenant opted into auto-book.
9. **Ids on rows, details via hooks.** Shipments store `pickup_location_id` / `delivery_address_id`; pincodes, addresses, and stock live in their owning modules and are resolved in memory. No pincode columns on the shipment.

```
┌──────── handlers ─────────────────┐
│  parse DTO → service              │
└──────────────┬────────────────────┘
┌──────────────▼────────────────────┐
│  planner │ shipment │ rate │      │
│  tracking │ webhook │ reconcile │  │  ◄── CourierPartner only
│  config                           │
└──┬──────────┬──────────┬──────────┘
   │          │          │
   ▼          ▼          ▼
 repositories  CourierPartner     order + inventory
               └── shiprocket/    narrow hooks
        CourierPartnerFactory
```

NDR actions and return booking are methods on `ShipmentService`, not separate services.

## 2. File layout

```
fulfillment/
├── service/
│   ├── courier/
│   │   ├── contract.go
│   │   ├── crypto.go
│   │   └── shiprocket/
│   │       ├── adapter.go
│   │       ├── auth.go          # JWT in durable KV; refresh on 401 and before expiry
│   │       ├── credentials.go
│   │       ├── http.go
│   │       ├── rates.go
│   │       ├── shipment.go      # BookShipment does create + AWB + optional pickup
│   │       ├── track.go
│   │       ├── ndr.go
│   │       └── webhook.go
│   ├── shipment_service.go      # book, cancel, label, ndr act, return
│   ├── shipment_planner.go      # auto-plan on OrderConfirmed (location grouping)
│   ├── rate_service.go
│   ├── tracking_service.go      # customer read; seller refresh
│   ├── webhook_service.go
│   ├── apply_normalized.go
│   ├── reconcile_service.go     # stale poll, NDR sweep, draft recovery
│   └── provider_config_service.go
├── factory/
│   └── courier_partner_factory.go
├── model/
└── errors/
```

## 3. Contract — `CourierPartner`

Credential methods stay on this interface, same as `PaymentGateway`. A second credential interface is not worth it for one provider.

Booking is **one** method. Shiprocket’s create-order, assign-AWB, and generate-pickup calls happen inside the adapter. A courier that returns an AWB from a single API does not invent empty steps.

```go
type CourierPartner interface {
    Code() string

    Validate(raw map[string]any, partial bool) error
    Encrypt(raw map[string]any) (map[string]any, error)
    Decrypt(stored map[string]any) (map[string]any, error)
    MaskHints(stored map[string]any) map[string]any
    MergePartial(existingStored, incoming map[string]any) (map[string]any, error)

    GetRates(ctx context.Context, in RateInput, creds map[string]any) ([]RateOption, error)

    // BookShipment sends our shipment id as the provider order id.
    // The adapter assigns the AWB. If in.PickupAt is set, it schedules pickup too.
    BookShipment(ctx context.Context, in BookShipmentInput, creds map[string]any) (*BookShipmentOutput, error)
    Cancel(ctx context.Context, in CancelInput, creds map[string]any) error
    GetLabel(ctx context.Context, in LabelInput, creds map[string]any) ([]byte, error)

    FetchTracking(ctx context.Context, awb string, creds map[string]any) (*NormalizedShipmentEvent, error)
    FetchTrackingBulk(ctx context.Context, awbs []string, creds map[string]any) ([]*NormalizedShipmentEvent, error)

    PeekLocators(rawBody []byte) TrackLocators
    NormalizeWebhook(rawBody []byte, headers http.Header, creds map[string]any) (*NormalizedShipmentEvent, error)
    TestConnection(ctx context.Context, creds map[string]any) error
}

// Optional. Type-assert and skip when absent.
type PickupScheduler interface {
    SchedulePickup(ctx context.Context, in PickupInput, creds map[string]any) error
}
type NDRHandler interface {
    GetNDR(ctx context.Context, awb string, creds map[string]any) (*NDRDetail, error)
    ActNDR(ctx context.Context, in NDRActionInput, creds map[string]any) error
}
type ReturnHandler interface {
    BookReturn(ctx context.Context, in BookReturnInput, creds map[string]any) (*BookShipmentOutput, error)
}
```

```go
type RateInput struct {
    PickupPincode, DeliveryPincode string
    WeightGrams int
    LengthCm, BreadthCm, HeightCm float64
    CodCents, OrderValueCents int64
}
type RateOption struct {
    CourierName, ServiceCode string
    RateCents int64
    ETD *time.Time
    PickupCapable bool
}
type ShipmentItemInput struct { OrderItemID uint; Quantity int }
type BookShipmentInput struct {
    ShipmentID, OrderID, SellerID uint
    Items []ShipmentItemInput
    PickupAlias string // planner-resolved from T9; empty = single pickup, courier default
    PickupAt *time.Time
    WeightGrams *int
    LengthCm, BreadthCm, HeightCm *float64
    CourierHint string
}
type BookShipmentOutput struct {
    ProviderOrderID, ProviderShipmentID, AWB string
    CourierName, ServiceCode string
    ETD *time.Time
    RawResponse map[string]any
}
type PickupInput struct { ShipmentID uint; AWB string; PickupAt *time.Time }
type CancelInput struct { ShipmentID uint; AWB, ProviderOrderID string }
type LabelInput struct { ShipmentID uint; AWB, ProviderShipmentID string }
type NDRActionInput struct { AWB string; Action string /* reattempt|rto */; AddressNote string }
type BookReturnInput struct { OrigShipmentID uint; Reason string; Items []ShipmentItemInput }
```

Location rule: `RateInput` pincodes and `BookShipmentInput.PickupAlias` are resolved in memory from `pickup_location_id` / `delivery_address_id` via hooks at call time. The shipment persists ids only.

### 3.1 `ShipmentAction` = status strings, plus `ignore`

```go
type ShipmentAction string
const (
    ShipmentActionIgnore           ShipmentAction = "ignore"
    ShipmentActionBooked           ShipmentAction = "booked"
    ShipmentActionPickupScheduled  ShipmentAction = "pickup_scheduled"
    ShipmentActionPicked           ShipmentAction = "picked"
    ShipmentActionInTransit        ShipmentAction = "in_transit"
    ShipmentActionOutForDelivery   ShipmentAction = "out_for_delivery"
    ShipmentActionNDRPending       ShipmentAction = "ndr_pending"
    ShipmentActionDelivered        ShipmentAction = "delivered"
    ShipmentActionFailed           ShipmentAction = "failed"
    ShipmentActionRTOInTransit     ShipmentAction = "rto_in_transit"
    ShipmentActionReturned         ShipmentAction = "returned"
    ShipmentActionCancelled        ShipmentAction = "cancelled"
)

type NormalizedShipmentEvent struct {
    Action        ShipmentAction
    EventID       string
    ProviderEvent string
    AWB           string
    ProviderOrderID, ProviderShipmentID string
    CourierName   string
    ETD           *time.Time
    FailureCode, FailureMessage string
    Payload       map[string]any
}

type TrackLocators struct { AWB, ProviderOrderID string }
```

`Apply` follows the allow-list in `data-model.md` section 2a. Terminal rows do not move. A repeat of the current status does not write another ledger row.

## 4. Factory and credentials

```go
type CourierPartnerFactory struct {
    providerRepo repository.CourierProviderRepository
    registry     map[string]courier.CourierPartner
}
func NewCourierPartnerFactory(repo ..., adapters ...courier.CourierPartner) *CourierPartnerFactory
func (f *...) GetByCode(code string) (courier.CourierPartner, error)
```

`ResolveForSeller` is used **once**, when booking:

1. Active seller row for `(seller_id, code, environment)`, else platform row (`seller_id IS NULL`).
2. None → `FULFILLMENT_PROVIDER_NOT_CONFIGURED`.
3. Decrypt in memory. The decrypted map is a function argument. It is not logged or saved.
4. Persist `provider_config_id` on the shipment inside the post-book transaction.

Webhook verification and tracking load **that** config row. If the seller later adds their own credentials, old shipments still verify with the account that booked them. The same row carries the tenant automation flags (`auto_book`, `rate_preference`).

Configure: `Validate(incoming, partial)` → `MergePartial` → `Validate(merged, false)` → `Encrypt` → save. Test-connection persists nothing.

Shiprocket’s JWT lives in the adapter (`auth.go`): durable KV, refresh before the 10-day expiry, and refresh on 401. The reconciler does not know about tokens. No token → that outbound call returns `FULFILLMENT_CREDENTIALS_INVALID`. It does not ship on a stale token.

## 5. Orchestrators

### 5.1 `ShipmentService`

`CreateShipment`:

1. **Tx1** (short): lock the order row, check seller ownership via `GetOrderForFulfillment`, check Σ quantity, insert `draft` + items + a ledger row with `event_type=draft`. Set `idempotency_key` when the client sent one. Commit. `UNIQUE(idempotency_key)` is the retry guard. A Redis replay key is optional and is not the source of truth.
2. **HTTP**, no transaction: `BookShipment`. The body uses `shipment_id` as the courier order id so a second box of the same order does not collide. POST is not retried by the HTTP client.
3. **Tx2** (short): save provider ids, AWB, `provider_config_id`, status `booked` (or `pickup_scheduled` when pickup was included), ledger row, `last_synced_at`. Commit. Then drop volatile keys, bump the list version, emit progress.

If the process dies after the courier accepts and before Tx2, the draft has no `provider_order_id`. `recover_drafts` (section 5.6) books again with the same shipment id. The adapter treats “already exists” as success and returns the existing AWB.

`Cancel`: allowed only from `draft` (local, no HTTP), `booked`, or `pickup_scheduled`. Then the allow-list.

`SchedulePickup`: only if the adapter implements `PickupScheduler`. Otherwise `FULFILLMENT_CAPABILITY_UNSUPPORTED` (pickup already happened inside `BookShipment` when `PickupAt` was set).

`GetLabel`: stream bytes to the client. Do not write them to PostgreSQL or to a JSON cache.

`ActNDR`: upsert is not by reason. Close the open round (`action_taken` set) or insert the next `attempt_no`. Requires `NDRHandler`.

`RequestReturn`: insert a new shipment with `return_of_shipment_id` set, then `ReturnHandler.BookReturn`. Tracking for that box uses the same applier. There is no second return-status column.

Location stamping: every create path (manual or planner) stamps `pickup_location_id` + `delivery_address_id`. `pickup_alias` is planner-resolved from T9 (NULL when the seller has a single pickup) and read-only after booking. The adapter sends the alias only when set.

### 5.2 `ShipmentPlanner` — auto-plan on `OrderConfirmed`

Runs on the order-confirmed event, before any human acts:

1. Guard: shipments already exist for this order → skip (replay-safe).
2. Load `FulfillmentOrderView` (lines + `delivery_address_id`) and availability per location via `FulfillmentInventoryHooks.GetAvailability` (variant → `[{location_id, available_qty, priority, pincode}]`).
3. Greedy allocate each line by warehouse priority until covered. Full coverage is guaranteed by checkout — a shortfall raises `FULFILLMENT_STOCK_MISMATCH` (exception + alert), never a partial plan. No backorder until PO.
4. Reserve each allocation via `FulfillmentInventoryHooks.ReserveForShipment` in the same pass (two concurrent orders cannot plan the same units).
5. Group allocations by `location_id` → one `draft` per warehouse in a single short tx (same Tx1 shape, Σ-guard, order-row lock). Stamp `pickup_location_id`, `delivery_address_id`, and `pickup_alias` (T9 lookup; NULL when single pickup).
6. Emit `OnShipmentsPlanned(orderID, draftIDs)`. Single-warehouse sellers collapse to exactly one draft with no special case.
7. Tenant flag: if the seller's config has `auto_book` TRUE, immediately run the §5.1 book path per draft, picking the rate by `rate_preference` (`cheapest` / `fastest`). Default FALSE stops at drafts for seller review ("book all").

### 5.3 `RateService`

Volatile `seller:{id}:fulfill:rate:{hash}`, jittered 90s, singleflight. Outbound limit via durable Lua `fulfill:rate:limit:{seller}`. Rates are not stored in PostgreSQL. Pincodes arrive in `RateInput` from hook-resolved ids (availability rows carry warehouse pincodes; the order view carries the delivery pincode) — there are no stored pincode columns.

### 5.4 `TrackingService`

`GetTrack`: PostgreSQL status + ledger for a shipment the caller owns. No courier call. No `Apply`.

`RefreshTrack`: seller-only. `FetchTracking` with the shipment’s `provider_config_id`, then `Apply`. Singleflight per shipment id.

### 5.5 `WebhookService`

1. `PeekLocators` (untrusted).
2. Load shipment by `(provider_code, awb)`, then `(provider_code, provider_order_id)`.
3. Unknown → **200**, persist nothing.
4. Decrypt the shipment’s `provider_config_id`. `NormalizeWebhook` checks the signature on raw bytes. Bad signature → **401**, persist nothing.
5. `UNIQUE(provider_code, event_id)` dedupes replays.
6. `Apply`. Verified but apply failed → mark the log `failed`, still return 200. The reconciler heals.

### 5.6 `NormalizedApplier.Apply`

Switch only on `ShipmentAction`.

- `ignore` → one ledger observation, status unchanged.
- Any other action → allow-list (data model 2a). Legal move: `UpdateStatusIfCurrent`, ledger row, timestamp patch (`shipped_at`, `delivered_at`, `cancelled_at`, `last_synced_at`, courier name, ETD).
- `delivered` → `OnShipmentDelivered` with lines. If `cod_cents > 0`, that amount stays on the shipment. No remittance row.
- `ndr_pending` → next NDR round + notify seller.
- `returned` → `OnShipmentReturned` with lines. The order module restocks those quantities.
- `failed` / `cancelled` → `OnShipmentFailed` with lines.
- Terminal row, or same status, or a pair not in the allow-list → no status change. A verified disallowed pair is logged as `ignore`.
- AWB set on both sides and different → do not apply (`FULFILLMENT_APPLY_MISMATCH`).

### 5.7 `ReconcileService`

Jobs use `common/cron`. They group work by `provider_config_id` so one bulk call uses one account.

- `recover_drafts` every 2m: drafts older than 2 minutes with `provider_order_id` NULL. `BookShipment` again with the same shipment id. `SKIP LOCKED`. Manual and planner drafts share this path.
- `reconcile_pending` every 5m: in-flight rows whose `last_synced_at` is older than the threshold for that status (not `updated_at`). `FetchTrackingBulk` in batches of 100 **per config**. `Apply(source=system)`. System `EventID` is `system:{provider}:{awb}:{action}`.
- `ndr_sweep` every 15m: open NDR rounds. `GetNDR` when the adapter implements it. Notify again when `action_taken` is still NULL after 24h.

No token-refresh job. No COD statement job.

### 5.8 `ProviderConfigService`

Thin wrapper over T2/T3: resolve, configure, test-connection, mask.

## 6. Order boundary

```go
type FulfillmentOrderHooks interface {
    GetOrderForFulfillment(ctx context.Context, orderID uint) (*FulfillmentOrderView, error)
    OnShipmentsPlanned(ctx context.Context, orderID uint, draftIDs []uint) error
    OnShipmentBooked(ctx context.Context, p FulfillmentProgress) error
    OnShipmentDelivered(ctx context.Context, p FulfillmentProgress) error
    OnShipmentFailed(ctx context.Context, p FulfillmentProgress) error
    OnShipmentReturned(ctx context.Context, p FulfillmentProgress) error
}

type FulfillmentOrderView struct {
    OrderID, SellerID, UserID uint
    Items []FulfillmentOrderItemView
    DeliveryAddressID uint // logical ref to order_address; pincode resolved from it
    DeliveryPincode string // in memory only, never stored on the shipment
    CodCents int64
}
type FulfillmentOrderItemView struct { OrderItemID, VariantID uint; Quantity int }

type FulfillmentProgress struct {
    OrderID, ShipmentID uint
    Items []FulfillmentLine // order item id + quantity in this box
    Reason string
}

// Narrow inventory surface: availability, reserve, release. No repo imports.
type FulfillmentInventoryHooks interface {
    GetAvailability(ctx context.Context, sellerID uint, variantIDs []uint) ([]AvailabilityRow, error)
    ReserveForShipment(ctx context.Context, sellerID uint, lines []ReservationLine) error
    ReleaseReservation(ctx context.Context, sellerID uint, lines []ReservationLine) error // cancel path
}
type AvailabilityRow struct { VariantID, LocationID uint; AvailableQty int; Priority int; Pincode string }
type ReservationLine struct { VariantID, LocationID uint; Quantity int; ShipmentID uint }
```

The order module decides whether the **order** is partially shipped, shipped, or delivered by reading every box. Inventory reserve/release go through `FulfillmentInventoryHooks`; fulfill and restock happen inside order's hook implementations. Fulfillment never imports order or inventory repositories.

Compile-time guard lives next to the applier: `var _ FulfillmentOrderHooks = (orderService.OrderService)(nil)`.

## 7. Shiprocket mapping (this folder only)

| Contract | Shiprocket | Notes |
|---|---|---|
| login (`auth.go`) | `POST /v1/external/auth/login` | 10-day JWT in durable KV. Refresh inside the adapter |
| `GetRates` | `POST /v1/external/courier/serviceability/` | Cache 90s |
| `BookShipment` | create adhoc, then assign AWB, then pickup if `PickupAt` is set | `order_id` = our shipment id; pickup alias sent only when set (multi-pickup) |
| `SchedulePickup` | `POST /v1/external/courier/generate/pickup` | Optional interface, when pickup was not part of book |
| `Cancel` | cancel order / cancel AWBs | Pre-pickup only |
| `GetLabel` | label GET | Stream bytes, no DB column |
| `FetchTracking` | `GET /v1/external/courier/track/awb/{awb}` | Bulk: `POST /courier/track/awbs`, one account per call |
| `GetNDR` / `ActNDR` | NDR get + action | Optional |
| `BookReturn` | `POST /v1/external/orders/create/return` | New shipment row points at the original |
| `PeekLocators` | `awb`, `order_id` | Discarded when verify fails |
| `NormalizeWebhook` | verify secret from the **booked** config row; map `sr-status` to `ShipmentAction` | Unknown status → `ignore` |
| `TestConnection` | small authed GET | 401 → 400-class, persist nothing |

HTTP: one pooled client. GET timeout 5s, POST 10s. POST is not retried. GET retries at most twice on 429/5xx. 4xx fails fast. 401 → `FULFILLMENT_CREDENTIALS_INVALID`. Logs: status, path, 512-byte body. No `Authorization`, secrets, phones, or full bodies. Allowed host: `apiv2.shiprocket.in` only, compiled into this package.

Credentials: `api_email` plaintext, `api_password` and `webhook_secret` AES-256-GCM. `MaskHints` returns a masked email. `MergePartial` decrypts, overlays, then encrypts once.

## 8. Errors

`FULFILLMENT_NOT_FOUND`, `INVALID_STATE`, `PROVIDER_NOT_SUPPORTED`, `PROVIDER_NOT_CONFIGURED`, `CREDENTIALS_INVALID`, `ENCRYPTION_KEY_MISSING`, `RATE_FAILED`, `BOOK_FAILED`, `LABEL_FAILED`, `CAPABILITY_UNSUPPORTED`, `WEBHOOK_UNVERIFIED`, `APPLY_MISMATCH`, `STOCK_MISMATCH` (plan-time shortfall: checkout/inventory disagree).

Provider 401 → `CREDENTIALS_INVALID`. Provider 429/5xx → retryable, mapped to 502/503.

## 9. Testing

- Adapter contract tests (Wiremock): book, cancel, label, track, 401, 429, bad JSON. Status table includes unknown → `ignore`.
- Book sends shipment id, not order id. Two boxes of one order get two provider orders.
- Webhook: valid apply; bad signature 401 and no rows; unknown AWB **200** and no rows; replay 200 and one apply; out-for-delivery then in-transit is allowed; delivered does not move backward; verified apply failure returns 200 and the cron heals.
- Reconciler: drafts without provider id are recovered; bulk track is split by `provider_config_id`; stale uses `last_synced_at`.
- Quantity: two concurrent creates cannot exceed `order_item.quantity` (order row lock).
- Planner: multi-warehouse order yields one draft per warehouse with correct ids + alias; single warehouse collapses to one draft; priority split across warehouses honors the Σ-guard; concurrent orders cannot double-plan the same units (reservation); missing stock raises, never partial-plans; `OrderConfirmed` replay is a no-op; opt-in sellers auto-book with their rate preference while default sellers stop at drafts.
- Pickup mapping: unknown location + provider fails the plan loudly; single-pickup sellers book with an empty alias.
- NDR: two rounds with the same reason succeed after the first is actioned; a second open round does not.
- Seller A cannot read seller B. Customer track does not call the courier.
- Integration: `test/integration/fulfillment/` with Testcontainers.

## 10. Adding a courier

1. New folder `fulfillment/service/courier/<code>/` implementing `CourierPartner`, plus optional pickup / NDR / return interfaces.
2. Provider names stay in that folder.
3. Seed `courier_provider`, `courier_provider_field`, and optional platform config.
4. One factory line. No orchestrator change and no new status string.
5. Contract tests green. Ship behind `is_active`.

## 11. Later, not this contract

HTTP routes and DTOs, cache key details, dashboard masking. Backorder/PO and COD remittance stay out until their own designs.
