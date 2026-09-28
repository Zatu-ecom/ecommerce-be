# 013 — Courier Fulfillment Platform · Service / Logic Design (Phase 2)

> Companion to `data-model.md` (tables T1–T10). This doc designs the **adapter (courier gateway) contract, factory, orchestrator services, and cross-module boundaries**. No code yet — signatures below are the implementation blueprint.
> Reference implementation mirrored: `payment/service/payment_gateway/contract.go`, `payment/factory/payment_gateway_factory.go`, `payment/service/apply_normalized.go`, `payment/service/webhook_service.go`, `fulfillment/shiprocket/*` (Shiprocket adapter home).

## 1. Principles (non-negotiable, copied from 010)

1. **Single provider contract.** All orchestration speaks only to `CourierPartner`. Orchestrators never switch on provider codes and never import provider subpackages (grep-enforced, like Razorpay confinement).
2. **Provider names live in one folder.** All Shiprocket specifics — endpoints, `sr-status` ids, auth headers, webhook shape — live only in `fulfillment/service/courier/shiprocket/`. Adding Delhivery = new folder, zero orchestrator edits.
3. **Frozen action set.** Webhook/cron apply paths switch only on `ShipmentAction`. New providers map their statuses inside their adapter; the base switch never grows.
4. **One apply path.** Webhook (`source=webhook`) and reconciler (`source=system`) share `NormalizedApplier.Apply`. They can never diverge on state machine, idempotency, or order hooks.
5. **Service-to-service, never cross-repo** (CODING_STANDARDS Rule #1). Fulfillment calls order/inventory through narrow hook interfaces, never their repositories.
6. **Fail-closed secrets, fail-open reads.** No encryption key → refuse to store/decrypt. Cache outage → serve from PG, never fail the shipment.

```
┌──────── handlers / routes ────────┐
│  thin: parse DTO → call service   │
└──────────────┬────────────────────┘
┌──────────────▼────────────────────┐
│  orchestrator services (provider- │  ◄── ONLY sees CourierPartner iface
│  agnostic): shipment │ rate │     │
│  tracking │ webhook │ reconcile │  │
│  ndr │ return │ cod │ config      │
└──┬──────────┬──────────┬──────────┘
   │          │          │
   ▼          ▼          ▼
 repositories  CourierPartner  narrow hooks
 (PG T4–T10)   iface ──► shiprocket/   (order/inventory/
               ▲        delhivery/…     notify — decoupled)
               │
        CourierPartnerFactory
        (registry + hybrid creds)
```

## 2. File layout (follows `payment/` shape)

```
fulfillment/
├── service/
│   ├── courier/
│   │   ├── contract.go            # CourierPartner iface + ShipmentAction + NormalizedShipmentEvent + TrackLocators
│   │   ├── crypto.go              # ResolveEncryptionKey (shared; mirrors gateway.ResolveEncryptionKey)
│   │   └── shiprocket/            # ALL Shiprocket specifics confined here
│   │       ├── adapter.go         # implements CourierPartner (Code()="shiprocket")
│   │       ├── credentials.go     # typed creds + Validate/Encrypt/Decrypt/MaskHints/MergePartial
│   │       ├── http.go            # pooled client, retry policy, log redaction
│   │       ├── rates.go           # serviceability → []RateOption
│   │       ├── shipment.go        # create/awb/pickup/cancel/label/return calls
│   │       ├── track.go           # FetchTracking + bulk track
│   │       ├── ndr.go             # NDR get/act
│   │       └── webhook.go         # PeekLocators + NormalizeWebhook + status map
│   ├── shipment_service.go / _impl.go
│   ├── rate_service.go
│   ├── tracking_service.go
│   ├── webhook_service.go
│   ├── apply_normalized.go        # NormalizedApplier (shared webhook+cron path)
│   ├── reconcile_service.go       # cron: stale shipments, stuck NDR, COD
│   ├── ndr_service.go
│   ├── return_service.go
│   ├── cod_service.go
│   └── provider_config_service.go # hybrid resolve, configure, test-connection
├── factory/
│   └── courier_partner_factory.go # registry map[code]CourierPartner
├── model/                         # provider-agnostic DTOs (below)
└── errors/                        # FULFILLMENT_* AppError codes
```

## 3. The contract — `CourierPartner` (mirrors `PaymentGateway`)

```go
// Code identifies the provider: "shiprocket" | "delhivery" | "bluedart" ...
// Orchestrators must never branch on it — the factory resolves, the iface executes.
type CourierPartner interface {
    Code() string

    // -- Credentials (dashboard + decrypt for outbound calls) --
    Validate(raw map[string]any, partial bool) error
    Encrypt(raw map[string]any) (map[string]any, error)
    Decrypt(stored map[string]any) (map[string]any, error)
    MaskHints(stored map[string]any) map[string]any
    MergePartial(existingStored, incoming map[string]any) (map[string]any, error)

    // -- Rate shopping --
    GetRates(ctx context.Context, in RateInput, creds map[string]any) ([]RateOption, error)

    // -- Shipment lifecycle --
    CreateOrder(ctx context.Context, in CreateShipmentInput, creds map[string]any) (*CreateShipmentOutput, error)
    AssignAWB(ctx context.Context, in AssignAWBInput, creds map[string]any) (*AWBOutput, error)
    SchedulePickup(ctx context.Context, in PickupInput, creds map[string]any) error
    Cancel(ctx context.Context, in CancelInput, creds map[string]any) error
    GetLabel(ctx context.Context, in LabelInput, creds map[string]any) ([]byte, error)

    // -- Tracking (single + bulk for cron) --
    FetchTracking(ctx context.Context, awb string, creds map[string]any) (*NormalizedShipmentEvent, error)
    FetchTrackingBulk(ctx context.Context, awbs []string, creds map[string]any) ([]*NormalizedShipmentEvent, error)

    // -- Webhook: peek (untrusted) then verify+normalize --
    PeekLocators(rawBody []byte) TrackLocators
    NormalizeWebhook(rawBody []byte, headers http.Header, creds map[string]any) (*NormalizedShipmentEvent, error)

    TestConnection(ctx context.Context, creds map[string]any) error
}

// Optional capabilities — reconciler/services type-assert and skip when absent
// (mirrors gateway.RefundStatusFetcher). Direct couriers often lack NDR.
type NDRHandler interface {
    GetNDR(ctx context.Context, awb string, creds map[string]any) (*NDRDetail, error)
    ActNDR(ctx context.Context, in NDRActionInput, creds map[string]any) error
}
type ReturnHandler interface {
    CreateReturn(ctx context.Context, in CreateReturnInput, creds map[string]any) (*CreateShipmentOutput, error)
}
```

Provider-agnostic DTOs (`fulfillment/model/`, mirrors `payment/model/gateway_models.go`):

```go
type RateInput struct { PickupPincode, DeliveryPincode string; WeightGrams int; LengthCm, BreadthCm, HeightCm float64; CodCents int64; OrderValueCents int64 }
type RateOption struct { CourierName, ServiceCode string; RateCents int64; ETD *time.Time; PickupCapable bool }
type ShipmentItemInput struct { OrderItemID uint; Quantity int }
type CreateShipmentInput struct { OrderID, SellerID uint; Items []ShipmentItemInput; PickupAlias string; WeightGrams *int; // override
    LengthCm, BreadthCm, HeightCm *float64; CourierHint string; IdempotencyKey string }
type CreateShipmentOutput struct { ProviderOrderID, ProviderShipmentID string; Status string; RawResponse map[string]any }
type AssignAWBInput struct { ShipmentID uint; ProviderOrderID, ProviderShipmentID string; ServiceCode, CourierID string }
type AWBOutput struct { AWB, CourierName, ServiceCode string; ETD *time.Time; RawResponse map[string]any }
type PickupInput struct { ShipmentID uint; AWB string; PickupDate *time.Time }
type CancelInput struct { ShipmentID uint; AWB, ProviderOrderID string }
type LabelInput struct { ShipmentID uint; AWB, ProviderShipmentID string }
type NDRActionInput struct { AWB string; Action string /* reattempt|rto */; AddressNote string }
type CreateReturnInput struct { OrigShipmentID uint; Reason string; Items []ShipmentItemInput }
```

### 3.1 Frozen `ShipmentAction` + normalized event (mirrors `WebhookAction` / `NormalizedWebhook`)

```go
type ShipmentAction string
const (
    ShipmentActionIgnore          ShipmentAction = "ignore"           // open/non-terminal observation
    ShipmentActionCreated         ShipmentAction = "created"
    ShipmentActionAWBAssigned     ShipmentAction = "awb_assigned"
    ShipmentActionPickupScheduled ShipmentAction = "pickup_scheduled"
    ShipmentActionPicked          ShipmentAction = "picked"
    ShipmentActionInTransit       ShipmentAction = "in_transit"
    ShipmentActionOutForDelivery  ShipmentAction = "out_for_delivery"
    ShipmentActionDelivered       ShipmentAction = "delivered"        // terminal-ok
    ShipmentActionFailed          ShipmentAction = "failed"           // terminal-bad (lost/damaged)
    ShipmentActionNDR             ShipmentAction = "ndr"              // needs seller action
    ShipmentActionNDRResolved     ShipmentAction = "ndr_resolved"
    ShipmentActionRTO             ShipmentAction = "rto"              // return-to-origin leg
    ShipmentActionReturned        ShipmentAction = "returned"         // terminal-returned
    ShipmentActionCancelled       ShipmentAction = "cancelled"        // terminal-cancelled
)

type NormalizedShipmentEvent struct {
    Action        ShipmentAction
    EventID       string // idempotency key, never empty (fallback {event}:{awb}:{status}:{ts})
    ProviderEvent string // raw label, stored on event ledger for audit only
    AWB           string
    ProviderOrderID, ProviderShipmentID string
    CourierName   string
    ETD           *time.Time
    FailureCode, FailureMessage string
    Payload       map[string]any // sanitized for storage (no phones/addresses)
}

type TrackLocators struct { AWB, ProviderOrderID string } // untrusted hints only
```

Terminal states (`delivered/failed/returned/cancelled`) never regress — `Apply` refuses backward transitions (compare against precedence map), so out-of-order webhooks are safe.

## 4. Factory + hybrid credential resolution (mirrors `PaymentGatewayFactory`)

```go
type CourierPartnerFactory struct {
    providerRepo repository.CourierProviderRepository
    registry     map[string]courier.CourierPartner
}
func NewCourierPartnerFactory(repo ..., adapters ...courier.CourierPartner) *CourierPartnerFactory
func (f *...) GetByCode(code string) (courier.CourierPartner, error) // unknown → FULFILLMENT_PROVIDER_NOT_SUPPORTED
```

Credential resolution (`provider_config_service.ResolveForSeller(ctx, sellerID, code)`):

1. Load `courier_provider_config` seller row (seller_id, code, environment) — if active, use it.
2. Else fall back to platform row (`seller_id IS NULL`, same code/env).
3. None / inactive → `FULFILLMENT_PROVIDER_NOT_CONFIGURED`.
4. `adapter.Decrypt(stored)` in memory only; decrypted map travels as `creds` param, never logged, never persisted.

Configure flow: `Validate(incoming, partial) → MergePartial(stored, incoming) → Validate(merged, false) → Encrypt → save`. Test-connection uses supplied-or-stored creds, persists nothing.

## 5. Orchestrator services

### 5.1 `ShipmentService` — create/AWB/pickup/label/cancel/return-request

- `CreateShipment(ctx, sellerID, in CreateShipmentInput)`: validate order ownership (via order hook read, not repo) + Σqty guard; PG tx (T4 draft + T5 items + T6 `created` event); idempotency via durable `fulfill:init:idem:{key}` 24h replay; then `adapter.CreateOrder` (POST, **no retry** — provider dedupes on `provider_order_id` UNIQUE); store ids → status `created` + event; `Del` detail + `BumpVersion` list; emit `FulfillmentEvent{Created}`.
- `AssignAWB / SchedulePickup / Cancel`: load shipment (seller-scoped), resolve adapter+creds, call provider (POST no retry), `UpdateStatusIfCurrent` (expected-from → to; stale/cancelled-after-pickup → `FULFILLMENT_INVALID_STATE`), append T6 event, emit.
- `GetLabel`: volatile URL cache 24h; bytes streamed to client, never cached as JSON (>256KB guard).
- Method budget: ≤50 lines each — split validation / tx / provider-call / emit helpers (CODING_STANDARDS).

### 5.2 `RateService` — `GetRates`

Volatile read-through `seller:{id}:fulfill:rate:{hash}` Jittered 90s + singleflight; provider `GetRates` on miss; rate-limit outbound via durable Lua `fulfill:rate:limit:{seller}` (mirrors coupon limiter). Never persists rates to PG.

### 5.3 `TrackingService` — `GetTrack(shipmentID)`

Merge PG status + live `FetchTracking` (volatile 45s); on drift (provider ahead), route through `NormalizedApplier` so ledger stays canonical. Customer variant strips creds/raw PII and checks `order.user_id` ownership.

### 5.4 `WebhookService.HandleWebhook(ctx, providerCode, rawBody, headers, ip)` — locate→verify→apply

Direct mirror of payment's `HandleWebhook` (§reference), adapted: locate by AWB → provider_order_id (miss → 200, persist nothing); creds resolved from the **shipment's seller** (seller override else platform); `NormalizeWebhook` verifies `x-api-key`/HMAC on raw bytes (bad sig → 401, persist nothing); `webhook_log` UNIQUE(provider,event_id) dedupes replays; `Apply` via shared applier; verified-but-failed → `MarkFailed` + return 200 (cron heals).

### 5.5 `NormalizedApplier.Apply(ctx, shipment, event, source)` — the ONE apply path

Switches **only** on `ShipmentAction`: `ignore` → ledger observation row; movement actions → `UpdateStatusIfCurrent` + T6 event + timestamp patch (`shipped_at/delivered_at/cancelled_at`, `courier_name`, `etd`); `delivered` → upsert T10 COD-pending (if `cod_cents>0`) + `FulfillmentOrderHooks.MarkDelivered`; `ndr` → upsert T8 + notify seller; `rto/returned` → T9 link + `MarkReturned`; `cancelled/failed` → terminal + `MarkFailed`. Already-terminal rows → idempotent no-op (no event spam). Amount guard analogue: AWB mismatch (event.AWB ≠ shipment.awb and both set) → refuse, mark failed.

### 5.6 `ReconcileService` — cron jobs via `common/cron`

- `reconcile_pending` every 5m: `ListStaleInFlight(SKIP LOCKED)` (no event in N hours by status) → `FetchTrackingBulk` 100/batch → `Apply(source=system)`; system `EventID = "system:{awb}:{action}"` keeps cron applies idempotent.
- `ndr_sweep` every 15m: re-pull open NDRs (adapter `GetNDR` when capable), escalate un-actioned >24h.
- `cod_remit` every 1h: match T10 pending against provider statement → `remitted(+utr)` / `disputed`.
- `token_refresh` every 12h: proactively refresh Shiprocket 10-day JWT into durable KV (fail-closed: no token → 503 on outbound, never ship on stale token).

### 5.7 `ProviderConfigService`, `NDRService`, `ReturnService`, `CODService`

Thin wrappers over T2/T3/T8/T9/T10 + capability type-asserts (`NDRHandler`/`ReturnHandler` absent → `FULFILLMENT_CAPABILITY_UNSUPPORTED`, UI already gates via T1 flags).

## 6. Decoupled order/inventory boundary (user decision: decoupled events)

Mirror of payment's `PaymentOrderHooks` narrow interface — fulfillment never imports order/inventory repos:

```go
// FulfillmentOrderHooks is the narrow surface fulfillment may call.
// Implemented by the order module; keeps fulfillment DB-free in tests.
type FulfillmentOrderHooks interface {
    MarkShippedByOrderID(ctx context.Context, orderID uint) error
    MarkDeliveredByOrderID(ctx context.Context, orderID uint) error
    MarkFulfillmentFailed(ctx context.Context, orderID uint, reason string) error
    MarkReturnedByOrderID(ctx context.Context, orderID uint) error
    GetOrderForFulfillment(ctx context.Context, orderID uint) (*FulfillmentOrderView, error) // id, seller, user, items, address-pins, cod
}
type FulfillmentOrderView struct { OrderID, SellerID, UserID uint; Items []FulfillmentOrderItemView; DeliveryPincode string; CodCents int64 }
```

Compile-time guard `var _ FulfillmentOrderHooks = (orderService.OrderService)(nil)` lives in fulfillment (same pattern as `apply_normalized.go:31`). Inventory moves (reserve on create, FULFILLED on pickup, restock on return) happen inside order's implementation, not fulfillment. Notifications read T6/the emitted `FulfillmentEvent`.

## 7. Shiprocket adapter mapping (confined to `shiprocket/`)

| Contract method | Shiprocket API | Notes |
|---|---|---|
| login (internal, `auth.go`) | `POST /v1/external/auth/login` (email+password → 10-day JWT) | Token in durable KV + mem; refresh job |
| `GetRates` | `POST /v1/external/courier/serviceability/` | Rate-limit, cache 90s |
| `CreateOrder` | `POST /v1/external/orders/create/adhoc` | Our order id in `order_id` field for locate-back |
| `AssignAWB` | `POST /v1/external/courier/assign/awb` | Returns AWB + courier |
| `SchedulePickup` | `POST /v1/external/courier/generate/pickup` | |
| `Cancel` | `POST /v1/external/orders/cancel` (+ `/cancel/shipment/awbs` variant) | Pre-pickup only |
| `GetLabel` | label/manifest/invoice GETs | Stream bytes |
| `FetchTracking` | `GET /v1/external/courier/track/awb/{awb}` | Cron uses `POST /courier/track/awbs` bulk |
| `GetNDR/ActNDR` | `GET /ndr/all\|/{awb}`, `POST /ndr/{awb}/action` | Optional capability |
| `CreateReturn` | `POST /v1/external/orders/create/return` | |
| `PeekLocators` | scrape `awb` / `order_id` / `sr_order_id` (untrusted) | Discarded if verify fails |
| `NormalizeWebhook` | verify `x-api-key`/secret on raw body (constant-time), map `sr-status`/`shipment_status_id` → `ShipmentAction` | Provider names never leave this folder |
| `TestConnection` | `GET` lightweight authed call (e.g. pickup-address list) | 401 → validation error (400, not 500); persist nothing |

HTTP policy (copies `razorpay/http.go`): one pooled client per adapter (GET 5s / POST 10s timeout); **POST never retried** (provider-side dedupe via our unique keys; retry risks double AWB/orders); GET retried ≤2× with jitter on 429/5xx only, 4xx fails fast; 401 maps to `FULFILLMENT_CREDENTIALS_INVALID`; logs carry status+path+512B truncated body only — never `Authorization`, secrets, phones, or full bodies; SSRF allowlist = `apiv2.shiprocket.in` (+ per-provider base URLs).

Credentials (`credentials.go`): fields `api_email` (plaintext id), `api_password` + `webhook_secret` (AES-256-GCM via `common/helper`, key from `crypto.go`); `MaskHints` returns masked email only — never secret blobs; `MergePartial` decrypts-then-overlays so updates never double-encrypt (exact Razorpay semantics).

## 8. Errors (`fulfillment/errors/`, `AppError` codes)

`FULFILLMENT_NOT_FOUND · INVALID_STATE · PROVIDER_NOT_SUPPORTED · PROVIDER_NOT_CONFIGURED · CREDENTIALS_INVALID · ENCRYPTION_KEY_MISSING · RATE_FAILED · CREATE_FAILED · AWB_FAILED · PICKUP_FAILED · LABEL_FAILED · CAPABILITY_UNSUPPORTED · WEBHOOK_UNVERIFIED · WEBHOOK_UNLOCATED · IDEMPOTENT_REPLAY · APPLY_MISMATCH`. Provider 401 → `CREDENTIALS_INVALID` (400-class); provider 429/5xx → typed retryable (service maps to 502/503); everything else wraps with `awb/shipmentId` context.

## 9. Testing (mirrors payment + repo TDD bar)

- **Contract tests per adapter** (run against wiremock Shiprocket): every `CourierPartner` method happy-path + 401 + 429 + malformed body; status-map unit table (`sr-status` → action, incl. unknown → `ignore` + alert).
- **Webhook matrix**: valid, bad signature (401, no rows), unknown AWB (200, no rows), replay (200, single apply), out-of-order (no regression), verified-apply-fail (200 + `failed` log + cron heals).
- **Reconciler**: stale sweep with `SKIP LOCKED`, bulk batching, system-EventID idempotency.
- **Isolation**: cross-seller shipment access must 404; customer sees only own orders.
- Integration tests under `test/integration/fulfillment/` via Testcontainers (pattern: `test/integration/product/...`); seeded `courier_provider` + field rows.

## 10. Adding a new courier — the AI playbook (goal: <1 day)

1. Paste provider API docs → generate `fulfillment/service/courier/<code>/` (`adapter, credentials, http, track, webhook` + optional `ndr`) implementing `CourierPartner` (+ `NDRHandler`/`ReturnHandler` asserts as capable).
2. Confine ALL provider names/URLs/status codes to that folder (grep check in review).
3. Seed: `courier_provider` row (+ capability flags) + `courier_provider_field` rows + optional platform-default config.
4. One factory registry line. **No orchestrator, migration, or contract change.**
5. Contract + webhook-matrix tests green. Ship behind per-provider `is_active` flag.

## 11. What Phase 3 covers (not this doc)

Endpoints + request/response DTOs, cache keys/TTLs, cron schedules, rate limits, dashboard masking — built on top of this contract without changing it.
