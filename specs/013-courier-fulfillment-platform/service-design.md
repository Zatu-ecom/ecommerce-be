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
type RTORequester interface {
    RequestRTO(ctx context.Context, in RTORequestInput, creds map[string]any) error
}
type RTORequestInput struct { ShipmentID uint; AWB string; Reason string }
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
    PickupLocationID uint // adapter derives the pickup nickname: S{seller}L{location}
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

Location rule: `RateInput` pincodes are resolved in memory from `pickup_location_id` / `delivery_address_id` via hooks at call time. The shipment persists ids only. The courier pickup nickname is derived at book time as `S{seller_id}L{location_id}` from the shipment's own ids — stored nowhere, identical path for single- and multi-pickup sellers.

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

1. **Tx1** (short): lock the order row, check seller ownership via `GetOrderForFulfillment`, check Σ quantity on non-cancelled shipments, insert `draft` + items + a ledger row with `event_type=draft`. `provider_code` stays NULL. Set `idempotency_key` when the client sent one. If that key already exists, return the existing shipment. Do not insert a second row and do not return a unique-violation error. Commit. A Redis replay key is optional and is not the source of truth. `delivery_address_revised_at` is the address `updated_at` from the order hook.
2. **Mark, then HTTP.** A short update sets `book_requested_at` (if NULL) and increments `book_attempts`, then commits. Only then call `BookShipment`, outside any transaction. The body uses `shipment_id` as the courier order id so a second box of the same order does not collide. POST is not retried by the HTTP client. A return box (`return_of_shipment_id` set) calls `BookReturn` instead.
3. **Tx2** (short): `UpdateStatusIfCurrent` from `draft` only. The winner saves provider code, provider ids, AWB, `provider_config_id`, status `booked` (or `pickup_scheduled` when pickup was included), ledger row, `last_synced_at`. Then drop volatile keys, bump the list version, emit `OnShipmentBooked` **once**. The loser sees the status already moved and emits nothing.

If the process dies after the courier accepts and before Tx2, the draft has `book_requested_at` set and no `provider_order_id`. `recover_drafts` (section 5.7) books again with the same shipment id. The adapter treats “already exists” as success and returns the existing AWB. A draft whose `book_requested_at` is NULL is waiting for a person. The job must not book it.

Before any courier call, refuse with no retry loop when `weight_grams` is missing or not positive (`FULFILLMENT_WEIGHT_REQUIRED`). The planner fills weight from the sum of catalog weights on the lines. The seller can override it on the draft. Auto-book that still lacks a weight falls back to the config's `default_weight_grams`; with neither, it leaves the draft, alerts once, and does not set `book_requested_at`.

Re-read the address at book time. If its `updated_at` is newer than `delivery_address_revised_at`, refuse (`FULFILLMENT_ADDRESS_CHANGED`) and do not set `book_requested_at`. The seller confirms, which re-stamps the timestamp, and then book continues. Auto-book does not confirm a changed address by itself.

Reservation: checkout already holds units at **variant** level (`CONFIRMED` on the order). Planning and manual create do not create another reservation. Inside Tx1, `AdoptReservation` checks that the confirmed hold still covers these lines. If it does not, Tx1 rolls back and nothing is left half-saved. The book path checks the same hold again before the courier call. A dead hold refuses booking with `FULFILLMENT_STOCK_MISMATCH`, cancels that draft, and runs the planner again for the quantity that is still uncovered. See section 7.

`Cancel`: allowed only from `draft` (local, no HTTP), `booked`, or `pickup_scheduled`. A local draft cancel needs no courier call. For `booked` or `pickup_scheduled`, call the courier first. If the courier refuses (already picked up, or any error), leave the local status unchanged and do not release stock. Release stock only after the courier cancel succeeds, or when the draft never left our database. If this box carries the order’s `cod_cents` and it is cancelled before pickup, move that amount to the next box still in `draft` or `booked`. If no such box exists, leave the amount on the cancelled row and alert. Never copy it onto a second live box.

`SchedulePickup`: only if the adapter implements `PickupScheduler`. Otherwise `FULFILLMENT_CAPABILITY_UNSUPPORTED` (pickup already happened inside `BookShipment` when `PickupAt` was set).

`GetLabel`: stream bytes to the client. Do not write them to PostgreSQL or to a JSON cache.

`ActNDR`: close the open round (`action_taken` set) or insert the next `attempt_no`. A repeat of the same reason is a new round after the previous one is closed. Requires `NDRHandler`. Local status stays `ndr_pending` until a webhook or refresh applies the next status. `reattempt` does not by itself move the shipment.

`RequestRTO`: for `picked`, `in_transit`, `out_for_delivery`, or `ndr_pending`. This is how a seller asks for the box back after pickup. Cancel is not used here. Call `ActNDR` with `rto` when an NDR round is open; otherwise call the courier’s in-transit return if the adapter has one. If it has neither, return `FULFILLMENT_CAPABILITY_UNSUPPORTED` and change nothing. Do not set `rto_in_transit` locally until `Apply` sees that status from the courier.

`RequestReturn`: only when the original shipment is `delivered`, and only when that shipment’s own `return_of_shipment_id` is NULL (no return of a return). Each line’s quantity, summed across non-cancelled return boxes, must be ≤ the quantity on the original box, and each line must be on that box. Insert a new shipment with `return_of_shipment_id` set, then `ReturnHandler.BookReturn` under the same `book_requested_at` rule as a forward book. Tracking for that box uses the same applier. There is no second return-status column.

Location stamping: every create path (manual or planner) stamps `pickup_location_id` + `delivery_address_id`. The adapter derives the pickup nickname (`S{seller}L{location}`) from those ids at book time and always sends it.

`BookAll`: runs the §5.1 book path per remaining draft on the order. Body `providerCode`/`serviceCode` are optional batch overrides; omitted fields resolve per draft (planner pick / seller config). Failures collect per draft and never fail the batch.

`ConfirmAddress`: re-reads `order_address`; unchanged since `delivery_address_revised_at` → no-op. Changed → serviceability-check the new pincode first (refuse with `FULFILLMENT_INVALID_STATE` when unserviceable), then re-stamp. Does not book.

### 5.2 `ShipmentPlanner` — auto-plan on `OrderConfirmed`

Runs on the order-confirmed event, before any human acts. The existence check and the inserts run inside the order-row lock, so a replay and a manual create cannot both pass the guard.

Only `directship` orders are planned. `bopis` (pickup in store) and `transfer` (stock move) stop here. They have no courier box. `delivery` is the local-delivery type and is also out of this module.

1. Guard: if non-cancelled shipments already cover every order line, skip (replay-safe). Cancelled rows do not count. A replan after a failed book only allocates quantity that is still uncovered.
2. Load `FulfillmentOrderView` (lines, weights, `fulfillment_type`, `delivery_address_id`, address `updated_at`) and availability per location via `FulfillmentInventoryHooks.GetAvailability` (variant → `[{location_id, available_qty, priority, pincode}]`).
3. Greedy allocate each still-uncovered line by warehouse priority until covered. Full coverage of what is still uncovered is required — a shortfall raises `FULFILLMENT_STOCK_MISMATCH` (exception + alert), never a partial plan. No backorder until PO.
4. Group allocations by `location_id` → one `draft` per warehouse in a single short tx (same Tx1 shape, Σ-guard, order-row lock). Stamp `pickup_location_id`, `delivery_address_id`, and `delivery_address_revised_at`. `provider_code` stays NULL. Inside this same transaction, `AdoptReservation` checks the order’s existing CONFIRMED hold. It does not insert a reservation. If the check fails, the whole transaction rolls back.
5. Stamp `cod_cents` on exactly one new draft: the one with the best warehouse priority (lowest priority number, then lowest id). Other new drafts get 0. Do not stamp COD onto a box that already carries it.
6. Emit `OnShipmentsPlanned(orderID, draftIDs)`. Single-warehouse sellers collapse to exactly one draft with no special case.
7. Auto-book, only when it is unambiguous. Read the seller’s own active config rows. If the seller has none, the platform row may apply. If exactly one active row has `auto_book` TRUE, book each new draft with that provider. `rate_preference` NULL means `cheapest`. If two or more rows have `auto_book` TRUE, leave the drafts and alert `FULFILLMENT_AUTO_BOOK_AMBIGUOUS`. A platform row with `auto_book` TRUE does not override a seller row that is FALSE. Default is drafts for the seller to review. Missing weight or a changed address skips the courier call, as in §5.1. Zero rate options do not call `BookShipment`; alert once and leave `book_requested_at` NULL.

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
- `ndr_pending` → next NDR round + notify seller. A new scan is a new round even when the reason text is the same. The cron event id includes the provider scan time, so the second “customer not home” is not dropped as a replay. The same scan time still dedupes.
- `returned` → `OnShipmentReturned` with lines. The order module restocks those quantities.
- `failed` / `cancelled` → `OnShipmentFailed` with lines.
- Terminal row, or same status, or a pair not in the allow-list → no status change. A verified disallowed pair is logged as `ignore`.
- AWB set on both sides and different → do not apply (`FULFILLMENT_APPLY_MISMATCH`).

### 5.7 `ReconcileService`

Jobs use `common/cron`. They group work by `provider_config_id` so one bulk call uses one account.

- `recover_drafts` every 2m: `status = draft`, `book_requested_at` set, `provider_order_id` NULL, `book_attempts` < 5, requested more than 2 minutes ago. `SKIP LOCKED`. Call `BookReturn` when `return_of_shipment_id` is set, otherwise `BookShipment`. Seller drafts with `book_requested_at` NULL are not selected. On the 5th failure, stop and alert once (`FULFILLMENT_BOOK_FAILED`). A seller retry resets `book_attempts` to 0. A missing pickup nickname, a 4xx, and a 5xx all count toward the 5. They do not retry forever.
- `reconcile_pending` every 5m: in-flight rows whose `last_synced_at` is older than the threshold for that status (not `updated_at`). `FetchTrackingBulk` in batches of 100 **per config**. `Apply(source=system)`. System `EventID` is `system:{provider}:{awb}:{action}:{scan_unix}`. `scan_unix` is the provider scan time. A repeat of that scan dedupes. A later scan does not. `delivered` from `booked` or `ndr_pending` is applied, not ignored.
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
    FulfillmentType string // directship is the only type this module plans
    Items []FulfillmentOrderItemView
    DeliveryAddressID uint // logical ref to order_address; pincode resolved from it
    DeliveryAddressUpdatedAt time.Time
    DeliveryPincode string // in memory only, never stored on the shipment
    CodCents int64
}
type FulfillmentOrderItemView struct { OrderItemID, VariantID uint; Quantity int; WeightGrams int }

type FulfillmentProgress struct {
    OrderID, ShipmentID uint
    Items []FulfillmentLine // order item id + quantity in this box
    Reason string
}

// Narrow inventory surface. No repo imports.
// Checkout already created the CONFIRMED hold at variant level. Fulfillment does not create another one.
type FulfillmentInventoryHooks interface {
    GetAvailability(ctx context.Context, sellerID uint, variantIDs []uint) ([]AvailabilityRow, error)
    AdoptReservation(ctx context.Context, sellerID uint, lines []ReservationLine) error // inside the draft tx; check only
    ReleaseReservation(ctx context.Context, sellerID uint, lines []ReservationLine) error // cancel path, this box's qty only
}
type AvailabilityRow struct { VariantID, LocationID uint; AvailableQty int; Priority int; Pincode string }
type ReservationLine struct { VariantID, LocationID uint; Quantity int; ShipmentID uint }
```

The order module decides whether the **order** is partially shipped, shipped, or delivered by reading every box. Inventory reserve/release go through `FulfillmentInventoryHooks`; fulfill and restock happen inside order's hook implementations. Fulfillment never imports order or inventory repositories.

`OnShipmentFailed` reason vocabulary (the string decides stock movement, so it is pinned, not free text): `cancelled_pre_pickup` → release reservation back to stock; `lost` / `damaged` → write-off via inventory transaction, no restock. Anything else → treat as `lost` (fail safe, alert).

Compile-time guard lives next to the applier: `var _ FulfillmentOrderHooks = (orderService.OrderService)(nil)`.

## 7. Inventory seam — edge cases

Ownership: order/inventory own stock truth. Checkout holds units at variant level (`PENDING` with TTL → `CONFIRMED` on capture). The warehouse is chosen here, at plan time. Fulfillment checks that confirmed hold and does not create a second one. It never decrements stock itself. Every disagreement below surfaces as a loud exception (before money moves) or a defined manual fallback (after) — never a silent wrong box.

- **E1 — reservation expires before confirm.** Slow payment → `PENDING` lapses → stock taken by another order. Detection: planner availability check at confirm. Handling: `FULFILLMENT_STOCK_MISMATCH` + alert; order stays confirmed with zero drafts; seller manually creates the shipment when stock returns.
- **E2 — planner runs behind (delay/crash after confirm).** Confirmed order with no drafts. Handling: planner is idempotent — dashboard exposes a **"plan shipments" retry** running the same function and guards. (A "confirmed-without-drafts" sweep job may come later.)
- **E3 — stock evaporates between plan and book.** Cycle-count correction, damage, or theft after the draft sits. Handling: book-time check refuses the booking, cancels that draft, and runs the planner again. The planner ignores cancelled rows and plans only the uncovered quantity.
- **E4 — cancel while drafts exist (pre-book).** Drafts → `cancelled` locally, zero courier cost, this box’s quantity released via `ReleaseReservation`. Pure local tx. COD, if this box held it, moves as in §5.1.
- **E5 — cancel after booking, pre-pickup.** Courier cancel first. Stock is released only when the courier accepts. A refused cancel leaves status and stock as they are.
- **E6 — lost/damaged vs cancelled pre-pickup.** Different stock fates behind one hook: reason vocabulary pinned in section 6 (`cancelled_pre_pickup` → release; `lost`/`damaged` → write-off, no restock).
- **E7 — double restock on return.** Webhook + cron both observe the return. Handling: fulfillment emits exactly once (same-status no-op + event-id idempotency); the **order module** guards restock per return-shipment-id.
- **E8 — manual shipment bypassing the planner.** No separate code path: manual create calls the same allocate/verify/stamp functions with human-supplied lines. Identical guards.
- **E9 — oversell across concurrent orders.** Serialized at checkout by inventory's own tx. The planner checks that CONFIRMED hold inside the draft transaction. It does not reserve the same units again.
- **E11 — seller has not pressed book.** `book_requested_at` is NULL. `recover_drafts` does not call the courier.
- **E12 — crash after the courier accepted.** `book_requested_at` is set, `provider_order_id` is NULL. The job books again with the same shipment id. A return box uses `BookReturn`.
- **E13 — courier keeps failing.** After 5 attempts the job stops and alerts once. It does not call forever. Missing nickname, no rates, and bad login are attempts, not infinite retries. Missing weight and a changed address never start the attempt loop.
- **E14 — two book clicks.** Tx2 moves `draft` only once. `OnShipmentBooked` fires once.
- **E15 — only a delivered scan arrives.** `delivered` applies from `booked`, `pickup_scheduled`, `picked`, `in_transit`, `out_for_delivery`, and `ndr_pending`.
- **E16 — second NDR from the checker.** The system event id includes the scan time. A later “customer not home” opens the next round.
- **E17 — two boxes and cash on delivery.** One box carries the full amount. The others stay 0. Cancelling the cash box before pickup moves the amount once.
- **E18 — pickup in store, transfer, local delivery.** The planner does not create a courier draft.
- **E19 — two couriers both set to auto-book.** Drafts stay unbooked and an alert is raised. NULL rate preference means cheapest. The platform flag does not override a seller flag that is off.
- **E20 — in-transit “send it back”.** `RequestRTO`. Local status changes only when the courier confirms. A return after delivery follows the quantity rules in §5.1.
- **E10 — confirmed-reservation TTL.** Confirmed reservations must live until consumed or released (no TTL); otherwise every long-lived draft is a time bomb and E3 becomes routine. Verify in the inventory module during implementation — if a TTL exists there, book-time revalidation becomes load-bearing rather than belt-and-braces.

Future (info only): order routing later swaps planner step 3 (greedy-by-priority) for cost/distance-aware assignment. Availability in, allocations out, and the existing confirmed hold stay. Only the algorithm changes. Backorder/PO later turns today's `STOCK_MISMATCH` call sites into backorder entry points; keep them centralized for that reason.

## 8. Shiprocket mapping (this folder only)

| Contract | Shiprocket | Notes |
|---|---|---|
| login (`auth.go`) | `POST /v1/external/auth/login` | 10-day JWT in durable KV. Refresh inside the adapter |
| `GetRates` | `POST /v1/external/courier/serviceability/` | Cache 90s |
| `BookShipment` | create adhoc, then assign AWB, then pickup if `PickupAt` is set | `order_id` = our shipment id; pickup nickname derived as `S{seller}L{location}`, always sent |
| `SchedulePickup` | `POST /v1/external/courier/generate/pickup` | Optional interface, when pickup was not part of book |
| `Cancel` | cancel order / cancel AWBs | Pre-pickup only |
| `GetLabel` | label GET | Stream bytes, no DB column |
| `FetchTracking` | `GET /v1/external/courier/track/awb/{awb}` | Bulk: `POST /courier/track/awbs`, one account per call |
| `GetNDR` / `ActNDR` | NDR get + action | Optional |
| `BookReturn` | `POST /v1/external/orders/create/return` | New shipment row points at the original |
| `PeekLocators` | `awb`, `order_id` | Discarded when verify fails |
| `NormalizeWebhook` | verify secret from the **booked** config row; map `sr-status` to `ShipmentAction` | Unknown status → `ignore` |
| `TestConnection` | small authed GET | 401 → 400-class, persist nothing |

Pickup nickname rule (all couriers, implemented per adapter): the nickname is `S{seller_id}L{location_id}`, derived at book time — never stored, never entered. Onboarding computes the seller's expected nicknames and checks them against the courier's registered pickup-address list; the seller is not marked shippable until every nickname exists. A nickname that disappears later fails booking loudly (no default fallback).

HTTP: one pooled client. GET timeout 5s, POST 10s. POST is not retried. GET retries at most twice on 429/5xx. 4xx fails fast. 401 → `FULFILLMENT_CREDENTIALS_INVALID`. Logs: status, path, 512-byte body. No `Authorization`, secrets, phones, or full bodies. Allowed host: `apiv2.shiprocket.in` only, compiled into this package.

Credentials: `api_email` plaintext, `api_password` and `webhook_secret` AES-256-GCM. `MaskHints` returns a masked email. `MergePartial` decrypts, overlays, then encrypts once.

## 9. Errors

`FULFILLMENT_NOT_FOUND`, `INVALID_STATE`, `PROVIDER_NOT_SUPPORTED`, `PROVIDER_NOT_CONFIGURED`, `CREDENTIALS_INVALID`, `ENCRYPTION_KEY_MISSING`, `RATE_FAILED`, `BOOK_FAILED`, `LABEL_FAILED`, `CAPABILITY_UNSUPPORTED`, `WEBHOOK_UNVERIFIED`, `APPLY_MISMATCH`, `STOCK_MISMATCH` (plan-time shortfall: checkout/inventory disagree), `WEIGHT_REQUIRED`, `ADDRESS_CHANGED`, `AUTO_BOOK_AMBIGUOUS`.

Provider 401 → `CREDENTIALS_INVALID`. Provider 429/5xx → retryable, mapped to 502/503.

## 10. Testing

- Adapter contract tests (Wiremock): book, cancel, label, track, 401, 429, bad JSON. Status table includes unknown → `ignore`.
- Book sends shipment id, not order id. Two boxes of one order get two provider orders.
- Webhook: valid apply; bad signature 401 and no rows; unknown AWB **200** and no rows; replay 200 and one apply; out-for-delivery then in-transit is allowed; delivered does not move backward; verified apply failure returns 200 and the cron heals.
- Reconciler: only drafts with `book_requested_at` set are recovered; a review draft is not booked; a return draft uses `BookReturn`; the 6th try does not call the courier; bulk track is split by `provider_config_id`; stale uses `last_synced_at`; a second NDR scan opens a new round; `booked` → `delivered` applies.
- Quantity: two concurrent creates cannot exceed `order_item.quantity` (order row lock).
- Planner: multi-warehouse order yields one draft per warehouse with correct location ids and a NULL courier; single warehouse collapses to one draft; priority split honors the Σ-guard; cancelled rows do not block a replan; no second reservation row is created; missing stock raises, never partial-plans; `OrderConfirmed` replay is a no-op; `bopis`, `transfer`, and `delivery` create no draft; one auto-book config books, two auto-book configs do not; COD is on one box only.
- Pickup nickname: derived `S{seller}L{location}` matches the registered nickname (onboarding verify); a missing nickname fails booking loudly, never falls back to a default.
- NDR: two rounds with the same reason succeed after the first is actioned; a second open round does not.
- Seller A cannot read seller B. Customer track does not call the courier.
- Inventory seam: book refuses on a dead hold and the draft transaction rolls back with the check; courier-refused cancel does not release stock; failed-reason vocabulary honored (release vs write-off); restock guarded once per return box on the order side.
- Book: same idempotency key returns the existing shipment; two overlapping books emit `OnShipmentBooked` once; missing weight and a newer address do not call the courier; return qty cannot exceed the delivered box; a return of a return is refused.
- Integration: `test/integration/fulfillment/` with Testcontainers.

## 11. Adding a courier

1. New folder `fulfillment/service/courier/<code>/` implementing `CourierPartner`, plus optional pickup / NDR / return interfaces.
2. Provider names stay in that folder.
3. Seed `courier_provider`, `courier_provider_field`, and optional platform config.
4. One factory line. No orchestrator change and no new status string.
5. Contract tests green. Ship behind `is_active`.

## 12. Later, not this contract

HTTP routes and JSON bodies are in `api-contracts.md`. Cache key details stay out of that file. Backorder/PO and COD remittance stay out until their own designs. Planner weight sums need `weight_grams` on product variants — no such column exists in the product module today, so confirm the source (or add it there) before implementation.
