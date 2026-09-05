# API Contracts: Razorpay Payment Gateway Integration

Base path: `/api/payment`. All responses use the standard envelope
`{ "success": bool, "message": string, "data": ... }` (errors add `code` / `errors`).
All authenticated endpoints require `X-Correlation-ID` (enforced by middleware).

## Customer & Seller Payment APIs

### POST /api/payment/initiate

Initiates payment for a pending order. Authenticated as the customer who owns the order.

Request:

```json
{
  "orderId": 123,
  "paymentMethodType": "card"
}
```

Response (200):

```json
{
  "success": true,
  "message": "Payment initiated",
  "data": {
    "transactionId": "TXN_...",
    "gatewaySessionId": "order_...",
    "keyId": "rzp_test_...",
    "amountCents": 199900,
    "currency": "INR",
    "status": "pending"
  }
}
```

Errors: order not found / not owned / not payable; provider not configured; unsupported
country/currency; duplicate payment for the order.

### GET /api/payment/transactions/:transactionId

Returns the current payment state for the authenticated owner/seller.

### GET /api/payment/transactions

Lists the authenticated seller's payments (paginated), scoped to `seller_id`.

### POST /api/payment/refunds

Seller requests a full or partial refund of a completed payment.

Request:

```json
{
  "transactionId": "TXN_...",
  "amountCents": 50000,
  "reason": "customer_request",
  "notes": "damaged item"
}
```

Response (200): refund record with `refundId`, `status`, `amountCents`, `currency`.

Errors: payment not completed; amount exceeds remaining refundable amount; provider not configured.

## Seller Gateway Dashboard APIs

### GET /api/payment/gateways

Lists all active providers, each with support info and whether this seller has an active config.

```json
{
  "success": true,
  "message": "Gateways fetched",
  "data": [
    {
      "code": "razorpay",
      "name": "Razorpay",
      "logo": { "fileId": "...", "url": "..." },
      "supportedCountries": ["IN"],
      "supportedCurrencies": ["INR"],
      "supportedPaymentMethods": ["card", "upi", "wallet"],
      "configured": false
    }
  ]
}
```

### GET /api/payment/gateways/:code

Full detail for one provider, including its configuration fields and validation rules.

### PUT /api/payment/gateways/:code/configure

Upsert (encrypt + activate) the seller's credentials for the provider.

Request:

```json
{
  "credentials": {
    "key_id": "rzp_test_...",
    "key_secret": "...",
    "webhook_secret": "..."
  },
  "environment": "sandbox",
  "priority": 1
}
```

Response (200): the saved configuration (secrets never echoed back).

### DELETE /api/payment/gateways/:code/configure

Deactivates the seller's config for the provider (does not delete saved credentials).

## Webhook APIs

### POST /api/payment/webhooks/razorpay

Public callback from Razorpay (no auth middleware; authenticity via signature).

Headers:

- `X-Razorpay-Signature`: `HMAC-SHA256(raw body, webhook_secret)` hex-encoded.

Body: raw JSON as sent by Razorpay.

Response:

- `200 OK` on a verified notification (processed, duplicate, or ignored).
- `401 Unauthorized` on missing/invalid signature (no state changes).

Handled events:

- `payment.captured` → payment `completed`; order `confirmed`.
- `payment.authorized` → recorded as an observation (append `authorized` event); does **not**
  confirm the order, since authorization precedes capture.
- `payment.failed` → payment `failed`; order `failed`.
- `refund.created` → refund `pending`; `refund.processed` → refund `completed` (and payment
  `refunded`/`partially_refunded`); `refund.failed` → refund `failed` (payment unchanged).

Idempotency: a repeated `event_id` for the same gateway is ignored and returns `200` without
re-applying any state change.
