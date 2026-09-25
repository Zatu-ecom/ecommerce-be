# Contract: PaymentGateway adapter

**Package:** `payment/service/payment_gateway`  
**Implementers:** `payment/service/payment_gateway/{code}/`  
**Consumers:** `payment_service`, `webhook_service`, `payment_gateway_service`, `reconcile_service` — **never** import `razorpay` except `factory/singleton/service_factory.go`.

Replace today’s interface (`InitiatePayment`, `FetchPayment`, `Refund(RefundType,…)`, `VerifyWebhook`, `ParseWebhook`). Delete `RefundType`. Cron uses `NormalizedWebhook` from `FetchRemoteStatus`, not `PaymentStatusOutput`. Delete unused `PaymentStatusOutput` if nothing else needs it.

Orchestrator DTOs (`InitiatePaymentInput`, `CheckoutPayload`, `NormalizedWebhook`) may live in `payment/model` **or** in `contract.go`. Provider credential structs **must** live in the provider folder, not `payment/model`.

---

## Interface (copy this)

```go
type PaymentGateway interface {
    Code() string

    Validate(raw map[string]any, partial bool) error
    Encrypt(raw map[string]any) (map[string]any, error)
    Decrypt(stored map[string]any) (map[string]any, error)
    MaskHints(stored map[string]any) map[string]any
    MergePartial(existingStored, incoming map[string]any) (map[string]any, error)

    InitiatePayment(ctx context.Context, in InitiatePaymentInput) (*InitiatePaymentOutput, error)
    Refund(ctx context.Context, in RefundInput) (*RefundOutput, error)
    TestConnection(ctx context.Context, creds map[string]any) error

    PeekLocators(rawBody []byte) Locators
    NormalizeWebhook(rawBody []byte, headers http.Header, creds map[string]any) (*NormalizedWebhook, error)
    FetchRemoteStatus(ctx context.Context, sessionID, paymentID string, creds map[string]any) (*NormalizedWebhook, error)
}
```

`crypto.go`: `ResolveEncryptionKey()` only (config then `ENCRYPTION_KEY`). Adapters call it internally. Base services must **not** call Razorpay parse helpers.

---

## Frozen webhook actions

Base `webhook_service` / `applyNormalized` may switch **only** on:

| Action | Effect |
|--------|--------|
| `ignore` | webhook_log ignored |
| `authorized` | append event; status unchanged |
| `payment_completed` | pending → completed; persist payment id/fee/method details; `ConfirmPaymentByTransactionID` |
| `payment_failed` | pending → failed; `FailPaymentByTransactionID` |
| `refund_pending` | refund → pending/processing |
| `refund_completed` | refund completed; txn `refunded` or `partially_refunded` |
| `refund_failed` | refund failed; txn money status unchanged for that attempt |

Razorpay mapping (**only** in `razorpay/`):

| Provider event | Action |
|----------------|--------|
| `payment.authorized` | `authorized` |
| `payment.captured` | `payment_completed` |
| `payment.failed` | `payment_failed` |
| `refund.created` | `refund_pending` |
| `refund.processed` | `refund_completed` |
| `refund.failed` | `refund_failed` |
| else | `ignore` |

`EventID`: prefer `X-Razorpay-Event-Id`; fallback `{event}:{entity.id}` so authorized vs captured do not collide.

HMAC: `X-Razorpay-Signature` = hex HMAC-SHA256 of **raw body** with `webhook_secret`. Compare with `hmac.Equal`. Never re-serialize JSON before verify.

`PeekLocators`: scrape `notes.transaction_id` / order id / payment id / refund id from JSON **without** verifying. Hints discarded if HMAC fails.

---

## CheckoutPayload

```go
type CheckoutPayload struct {
    GatewayCode string         `json:"gatewayCode"`
    Fields      map[string]any `json:"fields"`
}

type InitiatePaymentOutput struct {
    GatewaySessionID string
    Status           string
    AmountCents      int64
    Currency         string
    GatewayResponse  map[string]any
    Checkout         CheckoutPayload
}
```

Razorpay `Fields`: `"keyId"` → decrypted key_id, `"orderId"` → Razorpay order id (session).

---

## HTTP client rules (razorpay/http.go)

- Timeout 10–15s, pooled `http.Client`.
- Auth: Basic `key_id:key_secret`.
- **POST** create order / refund: **no retry**.
- **GET** (`FetchRemoteStatus`, `TestConnection`): up to 2 retries, jitter, only HTTP 429 and 5xx.
- Logs: truncate body; never log secrets or `Authorization`.

`TestConnection`: `GET /v1/orders?count=1`. HTTP 401 → invalid credentials error (map to 400 at handler).

`FetchRemoteStatus`: `GET /v1/orders/{sessionID}` when session set. `paid` → `payment_completed`; open → `ignore`; expired/cancelled → `payment_failed`. If only payment id (refund stuck), `GET /v1/payments/{id}` / refunds as needed. Always set `EventID` so apply is idempotent (`system:{txn}:{action}` acceptable for cron-generated events if provider has no event id).

---

## Credentials (razorpay/credentials.go)

- `Validate(raw, partial)`: required fields on full save; on partial, only validate provided keys.
- `Encrypt`: encrypt `key_secret`, `webhook_secret`; leave `key_id`, `account_id` plaintext in JSON.
- `Decrypt`: reverse.
- `MaskHints`: `key_id` first 8 + last 4; `account_id` full; **never** secrets.
- `MergePartial`: empty/omitted sensitive keys keep existing **stored encrypted** values.

---

## Factory

```go
razorpay := razorpay.New(os.Getenv("RAZORPAY_BASE_URL"))
factory.NewPaymentGatewayFactory(repo, razorpay)
```

`GetByCode` / `GetPaymentGateway(ctx, id)` unchanged idea. Initiate **selects** config then `GetByCode(selected.Code)` — never `GATEWAY_CODE_RAZORPAY` in `payment_service.go`.

---

## Webhook HTTP (handler)

```
POST /api/payment/webhooks/:code
```

1. `io.ReadAll(c.Request.Body)` — raw bytes.
2. Unknown code / unknown gateway → 404.
3. Service: peek → find txn → load **that** seller env config → `NormalizeWebhook`.
4. Invalid HMAC → 401, no DB money change.
5. No txn → 200, **no payload persist**.
6. After successful verify → always **200** even if apply later fails (persist `webhook_log.status=failed`). Provider must stop retrying.

Correlation-ID skip prefix: `/api/payment/webhooks/` (already implemented).

---

## applyNormalized (one function)

Used by webhook (source=`webhook`) and cron (source=`system`).

Before complete: if `AmountCents > 0` and ≠ txn, or currency non-empty and ≠ txn → mark log failed, **do not** complete.

Idempotency: unique `(gateway_id, event_id)` insert; duplicate → 200 no-op.
