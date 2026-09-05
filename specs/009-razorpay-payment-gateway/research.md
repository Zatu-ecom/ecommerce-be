# Research: Razorpay Payment Gateway Integration

## Unknowns & Decisions

### 1. Payment ↔ Order coupling

**Decision**: Payments stay independent of orders via a polymorphic reference
(`reference_type` + `reference_id`, no FK). The order→payment link remains the existing
`order.transaction_id = payment_transaction.transaction_id`.

**Rationale**: The payment ledger must also serve future subscription and other payment types
without a schema change per owner. The order module already owns a `transaction_id` column, so
reusing it avoids a hard FK while preserving the link.

**Alternatives considered**: An `order_id` FK on `payment_transaction` (rejected — couples the
ledger to one owner type and blocks subscription payments); a separate junction table (rejected —
unnecessary indirection for a 1:1 reference).

### 2. Provider abstraction (SOLID / OCP)

**Decision**: A gateway-agnostic `PaymentGateway` interface
(`InitiatePayment`, `FetchPayment`, `Refund`, `VerifyWebhook`, `ParseWebhook`) resolved by a factory
registry keyed on `payment_gateway.code`.

**Rationale**: Adding a provider becomes additive only (adapter + registry entry + seed row), with
zero changes to `payment_service`/`webhook_service`. The previous stub interface leaked gateway
vocabulary and took a GORM entity directly.

**Alternatives considered**: A `switch` on provider code inside the service (rejected — violates OCP
and spreads provider knowledge into business logic); a single fat interface with gateway-specific
methods (rejected — violates ISP).

### 3. Razorpay client library

**Decision**: stdlib `net/http` with HTTP basic auth, configurable `RAZORPAY_BASE_URL`.

**Rationale**: Razorpay's API is simple REST; avoids a new `go.mod` dependency and makes integration
tests trivial via `httptest.Server`. The base URL override is the cleanest test seam.

**Alternatives considered**: The official `razorpay-go` SDK (rejected — heavier, less test-friendly,
adds a dependency for little gain).

### 4. Webhook authenticity & idempotency

**Decision**: Verify `X-Razorpay-Signature` = `HMAC-SHA256(raw body, webhook_secret)` in hex using
constant-time comparison; dedupe via a unique partial index on
`payment_webhook_log(gateway_id, event_id)`.

**Rationale**: Razorpay signs the raw body, so re-parsing before verification breaks validation. A
DB unique constraint is the only reliable exactly-once guard across concurrent duplicate deliveries.

**Alternatives considered**: Redis `SetNX` dedupe (rejected — adds a second store and still needs a
DB write for audit); trusting `event_id` without a unique index (rejected — races allow double
processing).

### 5. Concurrency / state regression

**Decision**: Conditional updates (`UPDATE ... WHERE id = ? AND status = ?`, checking
`rowsAffected`) inside `common/db.WithTransaction`.

**Rationale**: A late or duplicate webhook must never regress a completed/failed payment. Conditional
writes make transitions atomic and idempotent without explicit row locks.

**Alternatives considered**: `SELECT ... FOR UPDATE` (rejected — heavier than needed here since we
already transition a single status field); optimistic version column (rejected — overkill for this
state machine).

### 6. Credentials encryption

**Decision**: Reuse `common/helper/crypto.go` AES-256-GCM with the `ResolveEncryptionKey` pattern;
encrypt `key_secret` and `webhook_secret` field-level inside the `credentials` JSONB. `key_id` is
stored plaintext (it is a public identifier).

**Rationale**: Matches the existing `file/service/blobAdapter` config encryption pattern and keeps
secrets unreadable at rest while preserving the public key identifier for the client checkout.

**Alternatives considered**: Encrypting the whole `credentials` blob (rejected — would also hide the
non-secret `key_id` and complicate validation); a dedicated secrets store (rejected — out of scope,
and the existing helper is sufficient).

### 7. Currency / amount source

**Decision**: Derive currency from the seller's base currency and country from seller settings; take
order amount from `order.total_cents`. Client never supplies amount/currency.

**Rationale**: Prevents amount/currency spoofing and guarantees provider support validation (Razorpay
↔ INR/IN). Aligns with the codebase's `*_cents` money standardization.

**Alternatives considered**: Accepting amount/currency from the request (rejected — security risk);
hardcoding INR (rejected — non-portable for future providers).

### 8. Testing strategy

**Decision**: Testcontainers for Postgres/Redis + a local `httptest.Server` faking the Razorpay API,
with the webhook signature computed from a known seeded `webhook_secret`.

**Rationale**: Matches the project's integration-first constitution and lets the full
initiate → webhook → order-confirm → refund flow run against real DB behavior without a live
provider account.

**Alternatives considered**: Go mocks for the adapter (rejected — integration tests are the project
standard and catch real wiring issues); live sandbox calls (rejected — non-deterministic).
