# Razorpay Payment Gateway Integration — Implementation Plan

## Summary

Integrate Razorpay into the payment module as the first real gateway adapter, following the existing
multi-gateway strategy design (`payment/PAYMENT_MODULE_DESIGN_V2.md`). The flow is
**order → pay → auto-confirm**: a customer places a pending order, the backend initiates a payment
session at the gateway, the gateway webhook marks `payment_transaction` captured and auto-confirms
the order via the order service. Credentials are **per-seller** (each seller's own
`key_id`/`key_secret`/`webhook_secret`), and webhook handling covers both payment and refund events.

The current `payment/` module is scaffolding only: 7 tables exist in
`migrations/009_create_payment_tables.sql`, GORM entities exist, but the gateway interface is a
`CashfreeGateway` stub, the factory has a nil-repository bug, and there are **no** handlers, routes,
repositories (beyond one lookup), or services. This plan fills in the full stack for Razorpay while
keeping the abstraction open for future gateways (Stripe, PayU, Cashfree).

---

## Razorpay API Contract (verified against official docs)

| Operation | Method + Path | Auth | Notes |
| --- | --- | --- | --- |
| Create order | `POST {base}/orders` | HTTP Basic (`key_id:key_secret`) | Body: `amount`, `currency`, `receipt`, `notes`. `receipt` is an **idempotency key** on the account — reuse it to dedupe session creation. We send our internal `transaction_id` as `receipt`. Response: `order.id` (stored generically as `gateway_session_id`). |
| Fetch payment | `GET {base}/payments/{id}` | HTTP Basic | Used for status reconciliation; returns payment entity incl. `amount`, `fee`, `status`, `method`. |
| Create refund | `POST {base}/payments/{id}/refund` | HTTP Basic | Body: `amount`, `notes`, `speed`. Response: `refund.id`. |
| Webhook signature | `X-Razorpay-Signature` header | `webhook_secret` | `HMAC-SHA256(raw request body, webhook_secret)` **hex-encoded**. Must verify the **raw body** (never the parsed JSON). The secret is the dashboard webhook secret, **not** the API key. |
| Payment webhook events | `payment.authorized`, `payment.captured`, `payment.failed` | — | Payload exposes `payment.entity` (id, amount, fee, status, method). |
| Refund webhook events | `refund.created`, `refund.processed`, `refund.failed` | — | Payload exposes `refund.entity` (id, payment_id, amount, status). |

- Base URL is configurable (`RAZORPAY_BASE_URL`, default `https://api.razorpay.com/v1`) so
  integration tests can point at a local fake server.
- Amounts are in the currency's **minor unit** (paise for INR), which matches the codebase's
  existing `*_cents` standardization.
- Razorpay is INR/India-only; our adapter + gateway record declare `INR` / `IN` support and the
  service validates the seller's currency/country against it.

---

## Key Architecture Decisions

### 1. Transaction history table — YES, add it

`payment_transaction` is the **aggregate** (current state, one mutable row per payment).
`payment_webhook_log` is the **raw inbound event** audit. Neither captures the full internal
state-machine history (initiated → gateway_session_created → authorized → captured → refunded, who
triggered it, and the gateway request/response at each step). An append-only
**`payment_transaction_event`** ledger is warranted for reconciliation, disputes, and debugging.

### 2. Payments stay independent of orders — polymorphic reference, no FK

`payment_transaction` gets **no `order_id` FK**. Payments are a reusable ledger for orders,
subscriptions, and future use cases. Instead:

- `payment_transaction.reference_type` (`order`, `subscription`) + `payment_transaction.reference_id`
  (`BIGINT`, **no foreign key**) — a polymorphic owner reference.
- The order→payment link already exists and stays as-is: `order.transaction_id` holds our internal
  `payment_transaction.transaction_id`. The webhook resolves the order through that value (via the
  order service, never the order repository).

### 3. Currency is derived from the seller, never from the client

The initiate request does **not** carry `amount` or `currency`. The service resolves the seller's
**base currency** (`UserService.GetSellerDefaultCurrency` → `CurrencyInfo.Code`) and business country
(`seller_settings.business_country_id`), and for order payments takes the amount from the order's
`total_cents`. This prevents amount/currency spoofing and guarantees gateway support validation
(Razorpay ↔ INR/IN).

### 4. No redundant columns between the aggregate and the event ledger

`payment_transaction` is the **current-state aggregate**; `payment_transaction_event` is the
**append-only history**. Each field lives in exactly one place:

- **Gateway references — two distinct columns, drop `gateway_transaction_id`:**
  - `gateway_session_id` — the checkout/order/intent reference returned by `InitiatePayment`
    (Razorpay `order.id`, Stripe `PaymentIntent.id`, Cashfree `order_id`). Needed to match incoming
    webhooks back to our transaction.
  - `gateway_payment_id` — the actual payment/charge reference returned on capture
    (Razorpay `payment.id`, Stripe `charge.id`). Needed to issue refunds and reconcile.
  - The existing `gateway_transaction_id` is dropped: it was a vague alias for the payment reference
    and overlapped with `gateway_payment_id`.
- **`gateway_request` / `gateway_response` live only on `payment_transaction_event`**, not on
  `payment_transaction`. The aggregate doesn't need raw call payloads — the latest request/response
  is simply the most recent matching event.
- **`initiated_at` is dropped** from `payment_transaction`: the row is created at initiation, so it
  duplicates `created_at`.
- **`failure_code` / `failure_message` stay on the aggregate** (current terminal failure reason for
  quick display) *and* on each event (the failure at that step). These serve different reads, so
  they are not history duplication on the aggregate.

### 5. `payment_transaction.metadata` is removed

The catch-all `metadata` column is unnecessary — everything it was intended to hold now has an
explicit column (`payment_method_details`, `gateway_fee_cents`) or belongs in the event ledger
(`gateway_request`, `gateway_response`, which live only on `payment_transaction_event`). The new
migration drops `metadata`.

### 6. Gateway logo uses the file service, not a raw URL

`payment_gateway.logo_url` is replaced with `logo_file_id` (`VARCHAR(80)`, nullable), matching the
established pattern (`seller_profile.business_logo_file_id`, `collection.image_file_id`). Gateway
list/detail responses resolve it through `common/filegateway.FileDisplayGateway` +
`filegateway.ResolveOptional` into a `FileAssetResponse`.

### 7. Gateway catalog + seller dashboard is part of this iteration

`payment_gateway` + `payment_gateway_field` are **core** seed data (available on backend bootstrap),
and seller-facing endpoints expose a dashboard:

- `GET /api/payment/gateways` → all active gateways, each showing: gateway info, resolved logo,
  config fields, supported countries/currencies/methods, and whether this seller has an active
  config (`configured`/`notConfigured`).
- `GET /api/payment/gateways/:code` → full gateway detail (fields, validation rules, support).
- `PUT /api/payment/gateways/:code/configure` → upsert seller credentials (encrypted) + activate.
- `DELETE /api/payment/gateways/:code/configure` → deactivate a seller's gateway config.

### 8. Gateway abstraction — gateway-agnostic interface (SOLID / OCP / ISP)

The `PaymentGateway` interface is the **only contract** the payment service knows about. It must be
applicable to every gateway, so its method names and DTOs are provider-neutral — no Razorpay,
Cashfree, or Stripe vocabulary leaks into it:

```go
// PaymentGateway is the gateway-agnostic contract every adapter implements.
// The payment service never references a concrete gateway type — it only
// depends on this interface and the gateway `code`.
type PaymentGateway interface {
    Code() string
    InitiatePayment(ctx context.Context, in InitiatePaymentInput) (*InitiatePaymentOutput, error)
    FetchPayment(ctx context.Context, gatewayPaymentID string) (*PaymentStatusOutput, error)
    Refund(ctx context.Context, in RefundInput) (*RefundOutput, error)
    VerifyWebhook(rawBody []byte, signature string) (bool, error)
    ParseWebhook(rawBody []byte) (*WebhookEvent, error)
}
```

- **`InitiatePayment`** (not "CreateOrder"): starts a provider checkout session. Razorpay maps this
  to `POST /orders`, Stripe to a PaymentIntent, Cashfree to its order/session call. The returned
  provider reference is normalized into `gateway_session_id`.
- `FetchPayment`, `Refund`, `VerifyWebhook`, `ParseWebhook` are already provider-neutral.

**Open/Closed Principle — how a new gateway is added with zero changes to the main service:**

1. Write a new adapter implementing `PaymentGateway` (e.g. `cashfree_gateway.go`).
2. Register it once in the factory's `map[code]PaymentGateway`.
3. Add its `payment_gateway` + `payment_gateway_field` catalog rows (seed).

That's it. `payment_service.go` and `webhook_service.go` are untouched — they resolve the adapter by
`payment_gateway.code` through the factory and call the same interface. No `switch`/`if gateway ==
"razorpay"` branches in business logic.

Adapters do **not** depend on GORM entities; they take **decrypted** credentials and typed DTOs. The
existing `factory/payment_gateway_factory.go` nil-injection bug gets fixed.

### 9. Razorpay client — stdlib HTTP, configurable base URL

Use `net/http` with HTTP basic auth rather than a third-party SDK: Razorpay's API is simple REST,
this avoids a new `go.mod` dependency, and a configurable base URL lets tests point at a local
`httptest.Server`. Webhook verification uses stdlib `crypto/hmac` + `crypto/sha256` with
constant-time comparison.

### 10. Credentials encryption

Reuse `common/helper/crypto.go` (`Encrypt`/`Decrypt`, AES-256-GCM) and the key-resolution pattern
from `file/service/blobAdapter/helper.go` (`ResolveEncryptionKey`). Sensitive fields
(`key_secret`, `webhook_secret`) are field-level encrypted inside
`payment_gateway_config.credentials` JSONB. `key_id` is a public identifier, stored plaintext.

### 11. Webhook security, idempotency, concurrency

- Verify `X-Razorpay-Signature` (HMAC-SHA256 over raw body) before any DB work.
- Dedupe via unique `(gateway_id, event_id)` partial index on `payment_webhook_log`.
- Apply transitions atomically with `common/db.WithTransaction` and **conditional UPDATEs**
  (`WHERE status = ?`) so a duplicate/late webhook cannot regress a completed transaction.
- Cross-module order updates go through the order **service interface** (never order repository).

---

## Database Changes

### New migration: `migrations/031_create_razorpay_payment_integration.sql`

1. Extend and reshape `payment_transaction` (existing table, created in `009`):

```sql
ALTER TABLE payment_transaction
    DROP COLUMN IF EXISTS metadata,
    DROP COLUMN IF EXISTS initiated_at,
    DROP COLUMN IF EXISTS gateway_transaction_id,
    ADD COLUMN IF NOT EXISTS reference_type       VARCHAR(20),
    ADD COLUMN IF NOT EXISTS reference_id         BIGINT,          -- no FK (polymorphic)
    ADD COLUMN IF NOT EXISTS gateway_session_id   VARCHAR(255),
    ADD COLUMN IF NOT EXISTS gateway_payment_id   VARCHAR(255),
    ADD COLUMN IF NOT EXISTS payment_method_details JSONB;

ALTER TABLE payment_transaction ALTER COLUMN gateway_fee_cents DROP NOT NULL;
ALTER TABLE payment_transaction ALTER COLUMN gateway_fee_cents SET DEFAULT 0;

DROP INDEX IF EXISTS idx_payment_transaction_gateway_transaction_id;

CREATE INDEX IF NOT EXISTS idx_payment_transaction_reference
    ON payment_transaction(reference_type, reference_id);
CREATE INDEX IF NOT EXISTS idx_payment_transaction_gateway_session_id
    ON payment_transaction(gateway_session_id);
CREATE INDEX IF NOT EXISTS idx_payment_transaction_gateway_payment_id
    ON payment_transaction(gateway_payment_id);
```

2. Replace `payment_gateway.logo_url` with a file reference:

```sql
ALTER TABLE payment_gateway
    ADD COLUMN IF NOT EXISTS logo_file_id VARCHAR(80),
    DROP COLUMN IF EXISTS logo_url;
```

3. New append-only history table:

```sql
CREATE TABLE IF NOT EXISTS payment_transaction_event (
    id               BIGSERIAL PRIMARY KEY,
    transaction_id   BIGINT NOT NULL REFERENCES payment_transaction(id) ON DELETE CASCADE,
    event_type       VARCHAR(50)  NOT NULL,  -- initiated, gateway_session_created,
                                             -- authorized, captured, completed, failed,
                                             -- refund_initiated, refund_completed, refund_failed
    from_status      VARCHAR(30),
    to_status        VARCHAR(30)  NOT NULL,
    gateway_event_id VARCHAR(255),
    gateway_request  JSONB,
    gateway_response JSONB,
    failure_code     VARCHAR(100),
    failure_message  TEXT,
    source           VARCHAR(20)  NOT NULL,  -- api, webhook, system, admin
    actor_id         BIGINT,
    actor_type       VARCHAR(20),            -- customer, seller, admin, system
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_txn_event_transaction_id ON payment_transaction_event(transaction_id);
CREATE INDEX IF NOT EXISTS idx_payment_txn_event_created_at     ON payment_transaction_event(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payment_txn_event_type           ON payment_transaction_event(event_type);
```

4. Webhook idempotency:

```sql
CREATE UNIQUE INDEX IF NOT EXISTS uq_payment_webhook_log_gateway_event
    ON payment_webhook_log (gateway_id, event_id)
    WHERE event_id IS NOT NULL;
```

### New core seed: `migrations/seeds/core/004_seed_payment_gateways.sql`

Core (bootstrap) seed — NOT mock, because the gateway catalog must exist whenever the backend runs:

- `payment_gateway`: `code='razorpay'`, `name='Razorpay'`, `supported_countries={IN}`,
  `supported_currencies={INR}`,
  `supported_payment_methods={card,upi,wallet,netbanking,emi,cardless_emi,paylater}`,
  `logo_file_id=NULL` (logo uploaded later via file service).
- `payment_gateway_field` (idempotent upsert): `key_id` (not sensitive), `key_secret` (sensitive),
  `webhook_secret` (sensitive), `account_id` (optional).

> Follow-up (out of scope): migration `009` names the `payment_method` column `details` while the
> entity uses `metadata`. Saved payment methods are deferred; reconcile this in a later migration
> when that feature is built.

---

## Backend Changes by File

### Entities (`payment/entity/`)

- **`payment_gateway.go`** — replace `LogoURL string` with `LogoFileID *string`
  (`column:logo_file_id;size:80`).
- **`payment_transaction.go`** — remove `Metadata`, `GatewayTransactionID`, `InitiatedAt`; add
  `ReferenceType`, `ReferenceID`, `GatewaySessionID`, `GatewayPaymentID`,
  `PaymentMethodDetails db.JSONMap`; make `GatewayFeeCents` a `*int64` (nullable, unknown at
  initiation). Add `ReferenceType` string constants (`order`, `subscription`). No
  `gateway_request`/`gateway_response` on the aggregate — those live only on the event entity.
- **`payment_transaction_event.go` (new)** — `PaymentTransactionEvent` (columns above; defines its
  own `ID` + `CreatedAt`, **no `UpdatedAt`** — mirror `payment_webhook_log`, do NOT embed
  `db.BaseEntity`; `TableName() = "payment_transaction_event"`).

### Models (`payment/model/`)

- **`gateway_models.go` (new)** — gateway-agnostic DTOs: `InitiatePaymentInput/Output`,
  `RefundInput/Output`, `PaymentStatusOutput`, `WebhookEvent`, `RazorpayCredentials`.
- **`payment_request.go` (new)** — `InitiatePaymentRequest` (orderID, paymentMethodType),
  `RefundRequest` (amountCents, reason, notes), `ConfigureGatewayRequest` (credentials map,
  isActive, priority).
- **`payment_response.go` (new)** — `PaymentTransactionResponse`, `InitiatePaymentResponse`
  (transactionId, gatewaySessionId, keyId, amountCents, currency, status), `RefundResponse`,
  `GatewaySummaryResponse` (id, code, name, logo `*FileAssetResponse`, supported lists,
  `configured bool`), `GatewayDetailResponse` (adds fields + validation rules).

### Gateway layer (`payment/service/payment_gateway/`)

- **`payment_gateway.go`** — replace stub interface with the gateway-agnostic `PaymentGateway`
  interface above.
- **`razorpay_gateway.go` (new)** — `RazorpayGateway` implements `PaymentGateway`:
  - `InitiatePayment` → `POST {base}/orders` (amount, currency, receipt=transaction_id); maps
    `order.id` → `gateway_session_id`.
  - `FetchPayment` → `GET {base}/payments/{id}`.
  - `Refund` → `POST {base}/payments/{id}/refund`.
  - `VerifyWebhook` → HMAC-SHA256 hex over raw body, constant-time compare.
  - `ParseWebhook` → map Razorpay `event` + payload to a gateway-agnostic `WebhookEvent`.
  - Inject `*http.Client` + `baseURL`; basic auth from decrypted credentials.
- **Delete `cashfree_gateway.go`** (stub, superseded).

### Credentials (`payment/service/payment_gateway/`)

- **`razorpay_credentials.go` (new)** — `ParseRazorpayCredentials(map) (*RazorpayCredentials, error)`
  + `EncryptSensitive()` / `DecryptSensitive()` (mirror `file/service/blobAdapter` config types).

### Factory (`payment/factory/`)

- **`payment_gateway_factory.go`** — fix constructor to inject `PaymentGatewayRepository`; registry
  `map[string]gateway.PaymentGateway` seeded with the Razorpay adapter; resolve by
  `payment_gateway.code` (unknown → `ErrorPaymentGatewayNotSupported`). Adding a gateway is purely
  additive: new adapter + one registry entry.

### Repositories (`payment/repository/`)

- **`payment_gateway_repository.go`** — add `FindByCode(ctx, code)`, `FindAllActive(ctx)`.
- **`payment_gateway_field_repository.go` (new)** — `FindByGatewayID(ctx, gatewayID)` (ordered by
  `display_order`).
- **`payment_gateway_config_repository.go` (new)** — `FindBySellerAndGateway(ctx, sellerID, gatewayID)`,
  `FindActiveBySeller(ctx, sellerID)`, `UpsertConfig(ctx, config)` (uses the
  `(seller_id, gateway_id, environment)` unique constraint), `DeactivateConfig(ctx, id)`.
- **`payment_transaction_repository.go` (new)** — `Create`, `FindByTransactionID`,
  `FindByGatewaySessionID`, `FindByGatewayPaymentID`, `UpdateStatusIfCurrent(ctx, id, from, to, patch)`
  (conditional UPDATE, returns `rowsAffected`), `FindByID`, `FindBySellerID` (paginated).
- **`payment_refund_repository.go` (new)** — `Create`, `FindByRefundID`, `UpdateStatusIfCurrent`.
- **`payment_webhook_log_repository.go` (new)** — `Create`, `MarkProcessed`, `MarkIgnored`.
- **`payment_transaction_event_repository.go` (new)** — `Create` (append-only).

All repositories use `db.DB(ctx)` (transaction-aware) and return entities.

### Services (`payment/service/`)

- **`payment_gateway_service.go` (new)** — seller dashboard catalog:
  `ListForSeller(ctx, sellerID)`, `GetByCode(ctx, code)`, `Configure(ctx, sellerID, code, req)`,
  `Deactivate(ctx, sellerID, code)`. Uses `FileDisplayGateway` to resolve logos.
- **`payment_service.go`** — replace stub with interface + impl. It depends only on the
  `PaymentGateway` interface and the factory (no concrete gateway type):
  - `InitiatePayment(ctx, userID, sellerID, req)`:
    1. Load order via order service (ownership/status check).
    2. Resolve seller base currency + business country.
    3. Select gateway config by code, decrypt credentials, validate country/currency support.
    4. Create `payment_transaction` (status `pending`, `reference_type=order`,
       `reference_id=orderID`, `amount_cents=order.TotalCents`, `currency=sellerCurrency`).
    5. Call adapter `InitiatePayment`; persist `gateway_session_id` on the transaction and write a
       `gateway_session_created` event carrying `gateway_request` + `gateway_response`.
    6. Write `order.transaction_id = our transaction_id` via order service.
    7. Append `payment_transaction_event`.
    8. Return `gatewaySessionId` + `keyId` + amount for client checkout.
  - `GetPaymentStatus`, `ListSellerTransactions`, `InitiateRefund`.
- **`webhook_service.go` (new)** — `HandleWebhook(ctx, gatewayCode, rawBody, signature, ip)`:
  1. Verify signature; 401 on failure.
  2. Parse event; insert `payment_webhook_log` (unique index dedupes replays → `ignored`).
  3. `payment.captured` → conditional mark transaction `completed`, set
     `gateway_payment_id`, `completed_at`, `gateway_fee_cents`; append event; auto-confirm order via
     order service (`transaction_id` → `pending → confirmed`).
  4. `payment.authorized` → append an `authorized` observation event only; do **not** complete the
     payment or confirm the order (authorization precedes capture).
  5. `payment.failed` → mark transaction `failed`; append event; mark order `failed` (if `pending`).
  6. `refund.created`/`refund.processed`/`refund.failed` → create/update `payment_refund`; set
     transaction `refunded`/`partially_refunded`.
  7. Every transition is a conditional update inside one `db.WithTransaction`.

### Handlers (`payment/handler/`)

- **`payment_handler.go` (new)** — `InitiatePayment`, `GetPaymentStatus`, `ListSellerTransactions`,
  `Refund` (customer/seller scoped). Embeds `common/handler.BaseHandler`.
- **`gateway_handler.go` (new)** — seller dashboard: `ListGateways`, `GetGateway`, `ConfigureGateway`,
  `DeactivateGateway`.
- **`webhook_handler.go` (new)** — reads raw body via `io.ReadAll(c.Request.Body)`, extracts
  `X-Razorpay-Signature`, calls `WebhookService.HandleWebhook`; returns 200 on verified receipt,
  401 on signature failure.

### Routes (`payment/route/`)

- **`payment_route.go` (new)** — `PaymentModule` under `constants.APIBasePayment` (`/api/payment`)
  with `CustomerAuth`/`SellerAuth`.
- **`gateway_route.go` (new)** — seller gateway dashboard endpoints under `/api/payment/gateways`.
- **`webhook_route.go` (new)** — `WebhookModule`: `POST /api/payment/webhooks/razorpay` with **no
  auth middleware** (signature is the auth).

### Container & singleton wiring

- **`payment/container.go`** — `addModules` registers `PaymentModule`, `GatewayModule`, `WebhookModule`.
- **`payment/factory/singleton/`** (new) — `singleton_factory.go`, `repository_factory.go`,
  `service_factory.go`, `handler_factory.go` (mirror `product/factory/singleton/`). The service
  factory wires in `orderSingleton.GetInstance().GetOrderService()`,
  `userSingleton.GetInstance().GetUserService()`, and
  `fileSingleton.GetInstance().GetFileReadService()` (or its `FileDisplayGateway` adapter) — all via
  service interfaces, no import cycles.
- **`test/integration/setup/singletons.go`** — add `paymentSingleton.ResetInstance()`.

### Order module (cross-module service hook — no import of payment)

- **`order/service/order_service.go`** — add to `OrderService` interface:
  - `AttachTransactionID(ctx, orderID, sellerID, transactionID string) error`
  - `ConfirmPaymentByTransactionID(ctx, transactionID string) error` (pending → confirmed, sets
    `paid_at`).
  - `FailPaymentByTransactionID(ctx, transactionID, reason string) error` (pending → failed).
- **`order/repository/order_repository.go`** — add `FindOrderByTransactionID(ctx, transactionID)`.
- These reuse the existing status-transition helpers (`applyUpdateOrderStatusTx`,
  `utils.IsValidTransition`) and keep the payment module out of order internals.

### Errors & constants

- **`payment/error/payment_gateway_error.go`** — keep existing sentinels; add
  `ErrorPaymentTransactionNotFound`, `ErrorPaymentOrderNotFound`, `ErrorInvalidWebhookSignature`,
  `ErrorDuplicateWebhook`, `ErrorRefundNotAllowed`, `ErrorGatewayNotConfigured`,
  `ErrorGatewayUnsupportedCurrency`.
- **`payment/utils/constant/`** — add error codes/messages + gateway constants (`GATEWAY_RAZORPAY`,
  status/event values, `reference_type` values).

---

## Webhook Processing Flow (Razorpay)

```
POST /api/payment/webhooks/razorpay
  → read raw body
  → VerifyWebhook(rawBody, X-Razorpay-Signature)   [HMAC-SHA256 hex, constant-time]
  → ParseWebhook(rawBody)                           [event + payload]
  → db.WithTransaction:
      insert payment_webhook_log (received)         [unique (gateway_id,event_id) dedupes]
      lookup payment_transaction by gateway_session_id/gateway_payment_id (or receipt)
      conditional UPDATE status (WHERE status = expected)
      append payment_transaction_event (request + response)
      if payment.captured:
          orderService.ConfirmPaymentByTransactionID(transaction_id)
      if payment.authorized:
          append authorized observation event (no order change)
      if payment.failed:
          orderService.FailPaymentByTransactionID(transaction_id, failure_reason)
      if refund.*:
          create/update payment_refund + adjust transaction refund status
      mark webhook_log processed
  → 200 OK
```

Replay/delayed duplicate → unique index conflict → mark `ignored`, return 200 (idempotent).

---

## Testing (TDD, integration-first)

New suite: `test/integration/payment/setup_payment_suite_test.go` (+ per-feature `_test.go` files),
package `payment_test`, using `testify/suite` + Testcontainers.

- Spin up a local `httptest.Server` acting as the fake Razorpay API, set `RAZORPAY_BASE_URL` via env
  (before `config.Reset()`), and compute the webhook signature with the seeded `webhook_secret`.
- Seed a seller `payment_gateway_config` (test key/secret) and a seller base currency/country.
- Cover:
  - initiate payment (creates transaction with `reference_type=order`, initiates gateway session,
    writes `order.transaction_id`, returns gatewaySessionId/keyId),
  - webhook `payment.captured` → transaction `completed` + order `confirmed`,
  - webhook `payment.failed` → transaction `failed` + order `failed`,
  - refund `created`/`processed`/`failed` updates,
  - invalid signature → 401,
  - duplicate webhook → idempotent (single event, no double order transition),
  - gateway dashboard list shows configured vs not-configured state,
  - seller isolation on listing transactions and gateway configs.
- Reuse `helpers.NewAPIClient`, `helpers.Login*`, `helpers.AssertSuccess/Error`,
  `setup.SetupTestContainers`, `RunAllMigrations`, `RunAllSeeds`.

---

## Files to Create / Modify (summary)

**Create**
- `migrations/031_create_razorpay_payment_integration.sql`
- `migrations/seeds/core/004_seed_payment_gateways.sql`
- `payment/entity/payment_transaction_event.go`
- `payment/model/{gateway_models,payment_request,payment_response}.go`
- `payment/service/payment_gateway/razorpay_gateway.go`
- `payment/service/payment_gateway/razorpay_credentials.go`
- `payment/repository/{payment_gateway_field,payment_gateway_config,payment_transaction,payment_refund,payment_webhook_log,payment_transaction_event}_repository.go`
- `payment/service/{payment_service,payment_gateway_service,webhook_service}.go`
- `payment/handler/{payment_handler,gateway_handler,webhook_handler}.go`
- `payment/route/{payment_route,gateway_route,webhook_route}.go`
- `payment/factory/singleton/{singleton,repository,service,handler}_factory.go`
- `test/integration/payment/*_test.go`

**Modify**
- `payment/entity/payment_gateway.go`
- `payment/entity/payment_transaction.go`
- `payment/service/payment_gateway/payment_gateway.go`
- `payment/factory/payment_gateway_factory.go`
- `payment/repository/payment_gateway_repository.go`
- `payment/error/payment_gateway_error.go`
- `payment/utils/constant/{error_code,payment_gateway_constant}.go`
- `payment/container.go`
- `order/service/order_service.go`
- `order/repository/order_repository.go`
- `test/integration/setup/singletons.go`

**Delete**
- `payment/service/payment_gateway/cashfree_gateway.go`

---

## Verification

1. `go build ./...` (clean build gate).
2. `go vet ./...` and `gofmt`/`gofumpt`.
3. Apply migrations + seeds against local Postgres (`migrations/run_migrations.sh --migrate --seed`);
   confirm the Razorpay `payment_gateway`/`payment_gateway_field` core seed rows, the new
   `payment_transaction_event` table + indexes, and `payment_transaction` reshaped columns
   (`reference_type/reference_id`, `gateway_session_id`/`gateway_payment_id`, no `metadata`,
   no `gateway_transaction_id`, no `initiated_at`, no `gateway_request`/`gateway_response`).
4. `go test ./test/integration/payment/... -v` — full Razorpay initiate/webhook/refund/dashboard/
   isolation coverage passes against the fake gateway.
5. `go test ./test/integration/... -v` (or `make test`) — no regressions in order/user/file suites.

---

## Sources

- [Create an Order — Razorpay Docs](https://razorpay.com/docs/api/orders/create/)
- [API Authentication — Razorpay Docs](https://razorpay.com/docs/api/authentication/)
- [Validate and Test Webhooks — Razorpay Docs](https://razorpay.com/docs/webhooks/validate-test/)
- [All Webhook Events — Razorpay Docs](https://razorpay.com/docs/webhooks/all/)
- [Refunds Webhook Events — Razorpay Docs](https://razorpay.com/docs/webhooks/refunds/)
