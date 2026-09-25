# API Contracts: Payment Gateway Platform

Base path payment: `/api/payment`.  
Seller settings: `/api/user/seller/settings` (existing **PUT/GET**, not a new PATCH).

Envelope: `{ "success": bool, "message": string, "data": ... }` (errors add `code` / `errors`).

**Headers (authenticated):** `Authorization: Bearer …`, `X-Correlation-ID` required (400 `CORRELATION_ID_REQUIRED` if missing).  
**Webhooks:** `X-Correlation-ID` **not** required; server generates and may echo `X-Correlation-Id`.

**Breaking vs 009 (intentional, no aliases):**

- Initiate: no top-level `keyId`.
- Webhook: `POST /api/payment/webhooks/:code` (Razorpay `code=razorpay`). No `HandleRazorpay`.
- Gateway list: no string-array-only countries; no single `configured` as the only badge.

---

## POST /api/payment/initiate

**Auth:** Customer.  
**Purpose:** Start checkout for a **pending** order owned by the caller.

Request:

```json
{
  "orderId": 123,
  "paymentMethodType": "card"
}
```

`paymentMethodType` optional. If set and not in gateway `supported_payment_methods` → 400. If omitted, skip that check.

Response `200`:

```json
{
  "success": true,
  "message": "Payment initiated",
  "data": {
    "transactionId": "TXN_...",
    "gatewaySessionId": "order_...",
    "amountCents": 199900,
    "currency": "INR",
    "status": "pending",
    "checkout": {
      "gatewayCode": "razorpay",
      "fields": {
        "keyId": "rzp_test_...",
        "orderId": "order_..."
      }
    }
  }
}
```

**Forbidden:** `"keyId"` as a sibling of `checkout`.

**Server algorithm (must match pre-spec §9.1):**

1. Load payable order; 404 if not owner / not pending.
2. Reject duplicate if pending or completed payment exists for that order (`409 DUPLICATE_PAYMENT`). Failed prior payment does **not** block.
3. Amount/currency from order + seller base currency; ignore any client money fields (there are none).
4. Load `seller_settings.payments_environment`.
5. Select **active** configs for `(seller_id, environment)` whose gateway is active and `EXISTS` country/currency join for seller `business_country_id` / `base_currency_id`. Order by `priority DESC`. Factory `GetByCode`. If none → `GATEWAY_NOT_CONFIGURED` or unsupported geo (`GATEWAY_UNSUPPORTED_CURRENCY` / new country code). **Do not** fall back to the other environment.
6. Decrypt via **adapter**. Insert txn `pending` with `environment` + event `initiated`.
7. `adapter.InitiatePayment`. On error: mark txn **failed**, return error (user may retry).
8. Persist `gateway_session_id` + event `gateway_session_created`.
9. `orderService.AttachTransactionID`. If attach fails: **fail the HTTP initiate** (txn stays pending until TTL).
10. Return checkout from adapter.

---

## GET /api/payment/transactions/:transactionId

**Auth:** Customer **or** seller JWT.  
Customer: `user_id` must match. Seller: `seller_id` must match. Else 404 (no leak).

Response includes:

- All current txn fields plus `environment`
- `refundableAmountCents` (amount − pending/processing/completed refunds)
- `events`: array ordered by `created_at` ASC (full history; payments have few events)

List endpoint must **not** include `events`.

---

## GET /api/payment/transactions

**Auth:** Seller.  
Query: `page`, `pageSize` (max 100), `status` optional enum:

`pending` | `completed` | `failed` | `refunded` | `partially_refunded`

Unknown `status` → 400. Filter in SQL. `pagination.totalItems` is the **filtered** count.

---

## POST /api/payment/refunds

**Auth:** Seller.

```json
{
  "transactionId": "TXN_...",
  "amountCents": 50000,
  "reason": "customer_request",
  "notes": "damaged item"
}
```

Allowed when txn status is `completed` **or** `partially_refunded` and `amountCents` ≤ remaining refundable. Pending txn → `REFUND_NOT_ALLOWED`.

Load credentials for **`txn.environment`**, not store toggle. Adapter `Refund` (amount only; no RefundType enum). Persist refund `processing` + event `refund_initiated`.

---

## GET /api/payment/gateways

**Auth:** Seller.

Each item:

```json
{
  "code": "razorpay",
  "name": "Razorpay",
  "description": "...",
  "logo": { "fileId": "...", "url": "..." },
  "supportedCountries": [{ "id": 21, "code": "IN", "name": "India" }],
  "supportedCurrencies": [{ "id": 4, "code": "INR", "symbol": "₹", "decimalDigits": 2 }],
  "supportedPaymentMethods": ["card", "upi", "wallet", "netbanking", "emi", "cardless_emi", "paylater"],
  "configuredSandbox": true,
  "configuredProduction": false,
  "paymentsEnvironment": "sandbox",
  "webhookUrl": "http://localhost:8080/api/payment/webhooks/razorpay"
}
```

`webhookUrl` = `{PUBLIC_API_BASE_URL}/api/payment/webhooks/{code}`. Default base `http://localhost:8080`. Logo omitted if no file.

Remove exclusive reliance on boolean `configured` + string country arrays. (You may keep `configured` as OR of both envs only if FE still needs it — prefer the two flags; spec list story uses dual flags.)

---

## GET /api/payment/gateways/:code

**Auth:** Seller. 404 unknown code.

Includes `fields[]` (schema) plus:

```json
"configs": {
  "sandbox": {
    "configured": true,
    "isActive": true,
    "priority": 1,
    "configHints": { "keyId": "rzp_test_xxxxCgd", "accountId": "acc_..." }
  },
  "production": {
    "configured": false,
    "isActive": false,
    "priority": 0,
    "configHints": null
  }
}
```

Never return secrets or encrypted blobs.

---

## PUT /api/payment/gateways/:code/configure

**Auth:** Seller.

```json
{
  "environment": "sandbox",
  "credentials": {
    "key_id": "rzp_test_...",
    "key_secret": "...",
    "webhook_secret": "...",
    "account_id": "acc_..."
  },
  "priority": 1,
  "isActive": true
}
```

`environment` **required** (`sandbox`|`production`). Upsert unique (seller, gateway, env). Partial: empty `key_secret` / `webhook_secret` keep stored secrets. Adapter `MergePartial` → `Validate` → `Encrypt`. Missing encryption key → 500, nothing stored plaintext. Do not write country on config.

---

## DELETE /api/payment/gateways/:code/configure

**Auth:** Seller.  
Query: `environment=sandbox|production` (required, or default to store `payments_environment` if omitted — prefer **required** for flash-model clarity: **require query `environment`**).

Deactivates **that row only** (`is_active=false`). Does not delete the other environment.

---

## POST /api/payment/gateways/:code/test

**Auth:** Seller.

```json
{
  "environment": "sandbox",
  "credentials": {
    "key_id": "rzp_test_...",
    "key_secret": "...",
    "webhook_secret": "..."
  }
}
```

`credentials` optional. If omitted, decrypt saved row for that environment. If provided, merge empty secrets from saved row; **do not persist**.

Success:

```json
{
  "success": true,
  "message": "Credentials accepted",
  "data": { "ok": true, "environment": "sandbox" }
}
```

Invalid keys → 400 (not 500). Unknown gateway → 404.

---

## GET /api/payment/webhook-logs

**Auth:** Seller.  
Query: `page`, `pageSize` (max 100), optional `status`, optional `eventType`.

Only logs whose `transaction_id` points at this seller’s payments. Unmatched logs not returned.

Not on the checkout hot path.

---

## POST /api/payment/webhooks/:code

**Auth:** none (signature). Unknown `code` → 404.

Headers (Razorpay): `X-Razorpay-Signature` required for verify; `X-Razorpay-Event-Id` optional.

See [gateway-adapter.md](./gateway-adapter.md) for locate-then-verify and status codes:

| Situation | HTTP |
|-----------|------|
| Unknown code | 404 |
| Txn found, bad HMAC | 401 |
| No txn | 200, no persist |
| Verified (apply ok, duplicate, ignore, apply failed internally) | 200 |

---

## GET/PUT /api/user/seller/settings

Add `paymentsEnvironment`: `sandbox` | `production`. Default `sandbox` on create.

Toggle to `production` without an **active** production payment config does not error on settings save; **checkout** then returns `GATEWAY_NOT_CONFIGURED`.

---

## Error codes (existing + use)

| Code | When |
|------|------|
| `DUPLICATE_PAYMENT` | Pending/completed already exists for order |
| `GATEWAY_NOT_CONFIGURED` | No active config for store mode |
| `GATEWAY_UNSUPPORTED_CURRENCY` | Currency not in join table |
| `GATEWAY_VALIDATION` | Bad credentials / fields |
| `REFUND_NOT_ALLOWED` | Wrong status or amount |
| `PAYMENT_TRANSACTION_NOT_FOUND` | Wrong owner or missing |
| `INVALID_WEBHOOK_SIGNATURE` | HMAC fail |
| `PAYMENT_GATEWAY_NOT_FOUND` | Unknown catalog code |

Add `GATEWAY_UNSUPPORTED_COUNTRY` if country join fails independently (400). Add `ENCRYPTION_KEY_MISSING` or reuse 500 with clear message for configure fail-closed.

---

## Out of contract

- `GET /refunds`
- `GET /transactions/:id/events`
- `POST /webhooks/razorpay` as a second named handler
- Cancel-payment API
- Top-level `keyId`
