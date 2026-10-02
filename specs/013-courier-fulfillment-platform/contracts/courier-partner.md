# Contract: `CourierPartner` (code-facing)

> Canonical behavior lives in `service-design.md` §§3–7. This file is the copy-paste-ready interface + DTO surface for implementation. HTTP route/JSON contracts live in `api-contracts.md`. Table contracts live in `data-model.md`.

## Interface

```go
// Code identifies the provider ("shiprocket" | "delhivery" | ...).
// Orchestrators MUST NOT branch on it.
type CourierPartner interface {
    Code() string

    // Credentials (dashboard + decrypt for outbound calls).
    Validate(raw map[string]any, partial bool) error
    Encrypt(raw map[string]any) (map[string]any, error)
    Decrypt(stored map[string]any) (map[string]any, error)
    MaskHints(stored map[string]any) map[string]any
    MergePartial(existingStored, incoming map[string]any) (map[string]any, error)

    // Rates (read-only; safe to retry).
    GetRates(ctx context.Context, in RateInput, creds map[string]any) ([]RateOption, error)

    // Booking (POST-equivalent; NEVER retried by the HTTP layer).
    // Sends our shipment id as the provider order id. Assigns the AWB.
    // Schedules pickup too when in.PickupAt is set.
    BookShipment(ctx context.Context, in BookShipmentInput, creds map[string]any) (*BookShipmentOutput, error)
    Cancel(ctx context.Context, in CancelInput, creds map[string]any) error
    GetLabel(ctx context.Context, in LabelInput, creds map[string]any) ([]byte, error)

    // Tracking (safe to retry; bulk variant is the cron path).
    FetchTracking(ctx context.Context, awb string, creds map[string]any) (*NormalizedShipmentEvent, error)
    FetchTrackingBulk(ctx context.Context, awbs []string, creds map[string]any) ([]*NormalizedShipmentEvent, error)

    // Webhook: PeekLocators scrapes UNTRUSTED hints; NormalizeWebhook verifies
    // the signature on raw bytes, then maps to a frozen ShipmentAction.
    PeekLocators(rawBody []byte) TrackLocators
    NormalizeWebhook(rawBody []byte, headers http.Header, creds map[string]any) (*NormalizedShipmentEvent, error)

    // No persistence. Provider 401 surfaces as a 400-class validation error.
    TestConnection(ctx context.Context, creds map[string]any) error
}

// Optional capabilities — callers type-assert and skip when absent.
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
```

## Frozen actions + normalized event

`ShipmentAction`: `ignore | booked | pickup_scheduled | picked | in_transit | out_for_delivery | ndr_pending | delivered | failed | rto_in_transit | returned | cancelled`.
Apply paths switch ONLY on these; provider status names never leave the adapter folder.

```go
type NormalizedShipmentEvent struct {
    Action        ShipmentAction
    EventID       string // idempotency key, never empty
    ProviderEvent string // raw label, audit only
    AWB           string
    ProviderOrderID, ProviderShipmentID string
    CourierName   string
    ETD           *time.Time
    FailureCode, FailureMessage string
    Payload       map[string]any // sanitized (no phones/addresses)
}
type TrackLocators struct { AWB, ProviderOrderID string }
```

## DTOs (provider-agnostic, `fulfillment/model/`)

`RateInput{pickup/delivery pincodes (hook-resolved), weight, dims, cod/order value}` →
`RateOption{CourierName, ServiceCode, RateCents, ETD, PickupCapable}` ·
`BookShipmentInput{ShipmentID, OrderID, SellerID, Items[{OrderItemID, Quantity}], PickupLocationID, PickupAt?, Weight/Dims?, CourierHint}` (nickname derived as `S{seller}L{location}`) →
`BookShipmentOutput{ProviderOrderID, ProviderShipmentID, AWB, CourierName, ServiceCode, ETD, RawResponse}` ·
`PickupInput{ShipmentID, AWB, PickupAt}` · `CancelInput{ShipmentID, AWB, ProviderOrderID}` ·
`LabelInput{ShipmentID, AWB, ProviderShipmentID}` · `NDRActionInput{AWB, Action(reattempt|rto), AddressNote}` ·
`BookReturnInput{OrigShipmentID, Reason, Items}` · `RTORequestInput{ShipmentID, AWB, Reason}`.

## Cross-module hook contracts

```go
type FulfillmentOrderHooks interface {
    GetOrderForFulfillment(ctx context.Context, orderID uint) (*FulfillmentOrderView, error)
    OnShipmentsPlanned(ctx context.Context, orderID uint, draftIDs []uint) error
    OnShipmentBooked(ctx context.Context, p FulfillmentProgress) error
    OnShipmentDelivered(ctx context.Context, p FulfillmentProgress) error
    OnShipmentFailed(ctx context.Context, p FulfillmentProgress) error  // Reason: cancelled_pre_pickup | lost | damaged
    OnShipmentReturned(ctx context.Context, p FulfillmentProgress) error
}
type FulfillmentInventoryHooks interface {
    GetAvailability(ctx context.Context, sellerID uint, variantIDs []uint) ([]AvailabilityRow, error)
    ReserveForShipment(ctx context.Context, sellerID uint, lines []ReservationLine) error
    ReleaseReservation(ctx context.Context, sellerID uint, lines []ReservationLine) error
}
type FulfillmentProductHooks interface {
    GetPhysicalSpecs(ctx context.Context, variantIDs []uint) ([]PhysicalSpec, error) // grams + cm, normalized
}
```

Fulfillment NEVER imports order/inventory/product repositories — hooks only (Constitution I+II).
