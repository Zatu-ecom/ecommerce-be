# Quickstart: Razorpay Payment Gateway Integration

## Running the feature locally

1. Apply the new migration and core seed (Razorpay gateway catalog):

   ```bash
   cd migrations && ./run_migrations.sh --migrate --seed
   ```

   This creates migration `031` changes (new `payment_transaction_event` table, reshaped
   `payment_transaction`, `logo_file_id` on `payment_gateway`) and seeds the `razorpay`
   `payment_gateway` + `payment_gateway_field` rows.

2. Configure a seller's Razorpay credentials. In the local sandbox, put the Razorpay test values in
   the seller's `payment_gateway_config` via the dashboard API:

   ```bash
   curl -X PUT http://localhost:8080/api/payment/gateways/razorpay/configure \
     -H "Authorization: Bearer <seller-jwt>" \
     -H "X-Correlation-ID: <uuid>" \
     -H "Content-Type: application/json" \
     -d '{
       "credentials": {
         "key_id": "rzp_test_...",
         "key_secret": "...",
         "webhook_secret": "..."
       },
       "environment": "sandbox",
       "priority": 1
     }'
   ```

3. Initiate a payment for a pending order:

   ```bash
   curl -X POST http://localhost:8080/api/payment/initiate \
     -H "Authorization: Bearer <customer-jwt>" \
     -H "X-Correlation-ID: <uuid>" \
     -H "Content-Type: application/json" \
     -d '{"orderId": 123, "paymentMethodType": "card"}'
   ```

4. The response returns `keyId`, `gatewaySessionId`, `amountCents`, and `currency` for the client
   checkout.

## Webhook simulation (local)

Point `RAZORPAY_BASE_URL` at a local fake server for outbound calls. For inbound webhooks, sign the
raw JSON body with the configured `webhook_secret`:

```text
signature = hex( HMAC_SHA256( raw_body, webhook_secret ) )
```

Then POST to `/api/payment/webhooks/razorpay` with header `X-Razorpay-Signature: <signature>`.
`X-Correlation-ID` is optional; the server generates one when it is omitted.

## Environment

- `RAZORPAY_BASE_URL` — defaults to `https://api.razorpay.com/v1`; override in tests.
- Encryption key — resolved from config/`ENCRYPTION_KEY` (same as the file module).

## Tests

```bash
go test ./test/integration/payment/... -v
```

The suite spins up Postgres/Redis Testcontainers and a fake Razorpay `httptest.Server`, covering
initiate → webhook capture/fail → order confirm/fail → refund, signature rejection, duplicate
webhook idempotency, and seller isolation.
