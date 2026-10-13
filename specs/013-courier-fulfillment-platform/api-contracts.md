# 013 — Courier Fulfillment Platform · API Contracts

> Companion to `data-model.md` and `service-design.md`. Routes and JSON only. Behavior (state machine, booking, planner) stays in those docs.
> Every JSON endpoint uses the shared envelope in section 1. Do not invent a second wrapper.

Base path: `/api/fulfillment`.

**Auth headers (seller and customer):** `Authorization: Bearer …`, `X-Correlation-ID` (required). Missing correlation id → 400 `CORRELATION_ID_REQUIRED`.

**Webhook:** no JWT. `X-Correlation-ID` is not required. If missing, the server generates one and echoes `X-Correlation-Id`.

Wrong owner → **404**, not 403. No existence leak across sellers or customers.

---

## 1. Shared envelope

Success (`common/model.Response`):

```json
{
  "success": true,
  "message": "Shipment booked",
  "data": {}
}
```

`data` is omitted when there is nothing to return.

Error (`common/model.ErrorResponse`):

```json
{
  "success": false,
  "message": "Weight is required before booking",
  "code": "FULFILLMENT_WEIGHT_REQUIRED",
  "errors": [
    { "field": "weightGrams", "message": "must be greater than 0" }
  ]
}
```

`errors` is only present for field validation (`common/model.ValidationError`: `field`, `message`). `code` is the app error code.

List payload inside `data` (`common/model.PaginationResponse`):

```json
{
  "shipments": [],
  "pagination": {
    "currentPage": 1,
    "totalPages": 3,
    "totalItems": 41,
    "itemsPerPage": 20,
    "hasNext": true,
    "hasPrev": false
  }
}
```

Query on every list: `page` (default 1), `pageSize` (default 20, max 100), `sortBy`, `sortOrder` (`asc` | `desc`, default `desc`). Unknown `sortBy` → 400. `pagination.totalItems` is the filtered count.

Money in responses is `common/model.Money`. Clients never send a money amount. COD and freight are server-stamped.

```json
{ "amount": 499, "amountCents": 49900, "formatted": "₹499.00" }
```

Times are RFC3339 UTC. Ids are numbers. Names are camelCase.

`Idempotency-Key` header, optional, on `POST /shipments` and `POST /shipments/:id/book`. Same key and same seller returns the existing shipment with the original success status. It does not create a second row. Keys are scoped per endpoint: the same string on `POST /shipments` and `POST /shipments/:id/book` are independent. Re-clicking book on a 502'd draft resumes the same shipment; success is never emitted twice.

---

## 2. Shared bodies

### Shipment

Used by get, list, plan, book, cancel, and return. List rows carry `items` (tiny, saves N+1 detail calls) but omit `events` and `ndr`.

```json
{
  "id": 90,
  "orderId": 15,
  "sellerId": 4,
  "status": "draft",
  "providerCode": null,
  "awb": null,
  "courierName": null,
  "serviceCode": null,
  "pickupLocationId": 8,
  "deliveryAddressId": 22,
  "weightGrams": 650,
  "lengthCm": 20,
  "breadthCm": 15,
  "heightCm": 10,
  "cod": { "amount": 499, "amountCents": 49900, "formatted": "₹499.00" },
  "rate": null,
  "insured": false,
  "etd": null,
  "shippedAt": null,
  "deliveredAt": null,
  "cancelledAt": null,
  "lastSyncedAt": null,
  "returnOfShipmentId": null,
  "bookRequestedAt": null,
  "bookAttempts": 0,
  "items": [
    { "orderItemId": 101, "quantity": 1 }
  ],
  "createdAt": "2026-10-01T06:00:00Z",
  "updatedAt": "2026-10-01T06:00:00Z"
}
```

Never returned: credentials, `providerConfigId`, raw provider payloads, phone, or street address. The address stays an id. The client loads it from the order API.

Customer track uses the same shape plus a short `events` list, and omits `bookRequestedAt`, `bookAttempts`, and `sellerId`.

### Event

Seller detail includes these. Customer track includes them without `failureCode` or `failureMessage`.

```json
{
  "eventType": "booked",
  "fromStatus": "draft",
  "toStatus": "booked",
  "providerEvent": "AWB Assigned",
  "source": "api",
  "failureCode": null,
  "failureMessage": null,
  "createdAt": "2026-10-01T06:05:00Z"
}
```

### NDR round

```json
{
  "attemptNo": 1,
  "ndrStatus": "consignee_not_available",
  "reason": "Customer not available",
  "actionTaken": null,
  "actedAt": null,
  "createdAt": "2026-10-01T08:00:00Z"
}
```

### Rate option

```json
{
  "courierName": "Delhivery Surface",
  "serviceCode": "delhivery_surface",
  "rate": { "amount": 80, "amountCents": 8000, "formatted": "₹80.00" },
  "etd": "2026-10-04T18:00:00Z",
  "pickupCapable": true
}
```

---

## 3. Seller — couriers

**Auth:** seller.

| Method | Path | Purpose |
|---|---|---|
| GET | `/couriers` | Active catalog plus whether this seller has a config |
| GET | `/couriers/:code` | Credential form and masked config |
| PUT | `/couriers/:code/configure` | Save credentials and auto-book flags |
| POST | `/couriers/:code/test` | Check credentials. Persist nothing |

### GET /couriers

`data.couriers[]`:

```json
{
  "code": "shiprocket",
  "name": "Shiprocket",
  "kind": "aggregator",
  "supportsPickup": true,
  "supportsNdr": true,
  "supportsReturn": true,
  "supportsCod": true,
  "webhookSupported": true,
  "configured": true,
  "isActive": true,
  "webhookUrl": "http://localhost:8080/api/fulfillment/webhooks/shiprocket"
}
```

`webhookUrl` is `{PUBLIC_API_BASE_URL}/api/fulfillment/webhooks/{code}`.

### GET /couriers/:code

404 unknown code.

```json
{
  "code": "shiprocket",
  "name": "Shiprocket",
  "fields": [
    {
      "fieldName": "api_email",
      "displayName": "API email",
      "fieldType": "email",
      "isRequired": true,
      "isSensitive": false,
      "displayOrder": 1
    }
  ],
  "config": {
    "configured": true,
    "isActive": true,
    "autoBook": false,
    "ratePreference": null,
    "environment": "production",
    "configHints": { "api_email": "s***@shop.com" }
  }
}
```

`config` is null when nothing is saved. `configHints` never contains a password or webhook secret.

### PUT /couriers/:code/configure

```json
{
  "environment": "production",
  "credentials": {
    "api_email": "seller@shop.com",
    "api_password": "secret",
    "webhook_secret": "whsec"
  },
  "autoBook": false,
  "ratePreference": "cheapest",
  "isActive": true
}
```

`environment` is required (`sandbox` | `production`). Empty sensitive fields keep the stored secret (`MergePartial`). `ratePreference` is `cheapest` or `fastest`, or null. Missing encryption key → 500, nothing stored in plaintext. Response `data` is the same object as GET `:code` `config`.

### POST /couriers/:code/test

```json
{
  "environment": "production",
  "credentials": {
    "api_email": "seller@shop.com",
    "api_password": "secret"
  }
}
```

`credentials` optional. If omitted, use the saved row. If sent, empty secrets are filled from the saved row. Nothing is written.

`data`: `{ "ok": true, "environment": "production" }`. Invalid keys → 400 `FULFILLMENT_CREDENTIALS_INVALID`, not 500.

---

## 4. Seller — plan, rates, shipments

**Auth:** seller. The order must belong to this seller.

| Method | Path | Purpose |
|---|---|---|
| POST | `/orders/:orderId/plan` | Run the planner again (confirmed order, missing or cancelled drafts) |
| POST | `/rates` | Live rates. Not stored |
| POST | `/shipments` | Manual draft for lines the planner did not cover |
| GET | `/shipments` | Seller list |
| GET | `/shipments/:id` | Detail, items, events, NDR |
| PATCH | `/shipments/:id` | Weight and dimensions while `draft` |
| POST | `/shipments/:id/confirm-address` | Accept a changed delivery address |
| POST | `/shipments/:id/book` | Book this draft |
| POST | `/orders/:orderId/book` | Book every remaining draft on the order |
| POST | `/shipments/:id/pickup` | Schedule pickup when it was not part of book |
| POST | `/shipments/:id/cancel` | Cancel before pickup |
| GET | `/shipments/:id/label` | PDF bytes, not the JSON envelope |
| POST | `/shipments/:id/refresh` | Seller poll. Writes through `Apply` |
| POST | `/shipments/:id/ndr` | Reattempt or RTO on an open NDR |
| POST | `/shipments/:id/rto` | Ask for the box back after pickup |
| POST | `/shipments/:id/returns` | Return after delivery |
| GET | `/webhook-logs` | This seller’s verified webhook rows |

`bopis`, `transfer`, and `delivery` orders: plan returns 400 `FULFILLMENT_INVALID_STATE`. This module plans `directship` only.

### POST /orders/:orderId/plan

No body. Creates drafts only for quantity not already on a non-cancelled shipment. Does not book unless auto-book rules in the service design say so. The order must be confirmed; a `pending` (or otherwise unconfirmed) order → 409 `FULFILLMENT_INVALID_STATE`. Drafts left unbooked with `FULFILLMENT_AUTO_BOOK_AMBIGUOUS` are resolved by setting a `ratePreference` or booking manually.

`201` when new drafts were created, `200` when the order was already covered (`data.shipments` is the existing live set). `data` is `{ "shipments": [ Shipment ] }`. No pagination.

### POST /rates

```json
{
  "providerCode": "shiprocket",
  "orderId": 15,
  "pickupLocationId": 8,
  "weightGrams": 650,
  "lengthCm": 20,
  "breadthCm": 15,
  "heightCm": 10
}
```

`providerCode` is required when the seller has more than one active courier. With one, it may be omitted. Pincodes are resolved from the order address and the location. They are not request fields.

`data`: `{ "options": [ RateOption ] }`. Empty `options` is 200, not an error. The client must not book from an empty list.

### POST /shipments

```json
{
  "orderId": 15,
  "pickupLocationId": 8,
  "items": [
    { "orderItemId": 101, "quantity": 1 }
  ],
  "weightGrams": 650,
  "lengthCm": 20,
  "breadthCm": 15,
  "heightCm": 10
}
```

Creates a `draft`. Does not call the courier. `providerCode` stays null. COD is stamped by the server, not this body. `201` and `data.shipment`.

Repeat `Idempotency-Key` → `200` and the same shipment.

### GET /shipments

Query: `status`, `orderId`, `pickupLocationId`, plus the shared page fields. `sortBy`: `created_at` | `updated_at` | `status`.

`data`: `{ "shipments": [ Shipment without items/events/ndr ], "pagination": Pagination }`.

### GET /shipments/:id

`data.shipment` is the full Shipment plus `events` and `ndr`.

### PATCH /shipments/:id

Draft only. Any other status → 409 `FULFILLMENT_INVALID_STATE`.

```json
{
  "weightGrams": 700,
  "lengthCm": 20,
  "breadthCm": 15,
  "heightCm": 10
}
```

At least one field. `weightGrams` must be > 0 when sent. `data.shipment`.

### POST /shipments/:id/confirm-address

No body. Re-stamps `delivery_address_revised_at` from the current address. Does not book. `data.shipment`.

### POST /shipments/:id/book

```json
{
  "providerCode": "shiprocket",
  "serviceCode": "delhivery_surface",
  "pickupAt": "2026-10-02T10:00:00Z"
}
```

`providerCode` required. `serviceCode` optional. `pickupAt` optional. When set, pickup is part of the same book.

Refused before any courier call, and `bookRequestedAt` stays null:

- missing weight (no PATCHed weight and no config `default_weight_grams`) → 400 `FULFILLMENT_WEIGHT_REQUIRED`
- address `updated_at` newer than the stamp → 409 `FULFILLMENT_ADDRESS_CHANGED`
- dead stock hold → 409 `FULFILLMENT_STOCK_MISMATCH` (draft is cancelled, planner runs for the gap)

`200` `data.shipment` when the book is accepted (status `booked` or `pickup_scheduled`). A second click with the same idempotency key, or a click that loses the race, returns that same shipment and does not emit booked twice.

Courier failure after the book was marked → 502 `FULFILLMENT_BOOK_FAILED`. The draft stays a draft with `bookRequestedAt` set so the retry job can finish it. The response still includes `data.shipment`.

### POST /orders/:orderId/book

Same body, but `providerCode`/`serviceCode` are optional batch overrides; omitted fields resolve per draft (planner pick / seller config). Runs that path on every `draft` for the order that has no `bookRequestedAt` yet.

`data`:

```json
{
  "shipments": [],
  "failures": [
    { "shipmentId": 91, "code": "FULFILLMENT_WEIGHT_REQUIRED", "message": "Weight is required before booking" }
  ]
}
```

`200` when at least one shipment booked. `400` when every draft failed. Already-booked boxes are left unchanged and listed in `shipments`.

### POST /shipments/:id/pickup

```json
{ "pickupAt": "2026-10-02T10:00:00Z" }
```

Adapter has no pickup call → 400 `FULFILLMENT_CAPABILITY_UNSUPPORTED`. `data.shipment`.

### POST /shipments/:id/cancel

No body. Draft cancels locally. `booked` and `pickup_scheduled` call the courier first. Courier refusal → 409 `FULFILLMENT_INVALID_STATE`, status and stock unchanged. `data.shipment`.

### GET /shipments/:id/label

Success is **not** the JSON envelope. `200`, `Content-Type: application/pdf`, body is the file bytes, `Content-Disposition: attachment`.

Errors use the JSON envelope: `404` unknown id; `400` `FULFILLMENT_LABEL_FAILED` when there is nothing to fetch (no AWB yet); `503` on transient courier failure. The file is not stored.

### POST /shipments/:id/refresh

No body. Seller only. Fetches the courier and runs `Apply`. `data.shipment` is the row after apply. Customer GET must not use this path.

### POST /shipments/:id/ndr

Open round required.

```json
{
  "action": "reattempt",
  "addressNote": "Call before delivery"
}
```

`action`: `reattempt` | `rto`. No open round, or adapter has no NDR → 400 `FULFILLMENT_CAPABILITY_UNSUPPORTED`. Status stays `ndr_pending` until a later webhook or refresh. `data` is the NDR round.

### POST /shipments/:id/rto

No body. Status must be `picked`, `in_transit`, `out_for_delivery`, or `ndr_pending`. This does not cancel. Local status does not change until the courier confirms `rto_in_transit`. Adapter cannot do it → 400 `FULFILLMENT_CAPABILITY_UNSUPPORTED`. `data.shipment`.

### POST /shipments/:id/returns

Original must be `delivered` and must not itself be a return.

```json
{
  "reason": "wrong_size",
  "items": [
    { "orderItemId": 101, "quantity": 1 }
  ]
}
```

Quantity above the original box, unknown line, or a return of a return → 400 `FULFILLMENT_INVALID_STATE`. `201` `data.shipment` is the new return draft after book is attempted. Book rules (weight, attempts) are the same as a forward book.

### GET /webhook-logs

Query: `status` (`received` | `applied` | `failed` | `ignored`), `awb`, plus page fields. `sortBy`: `created_at`.

Only rows linked to this seller’s shipments. Unknown AWB deliveries were not stored and do not appear.

```json
{
  "id": 500,
  "providerCode": "shiprocket",
  "eventId": "evt_1",
  "awb": "SR123",
  "action": "in_transit",
  "status": "applied",
  "shipmentId": 90,
  "errorMessage": null,
  "createdAt": "2026-10-01T07:00:00Z"
}
```

No raw payload in this list.

---

## 5. Customer — track

**Auth:** customer. The order’s `user_id` must match. Else 404.

| Method | Path | Purpose |
|---|---|---|
| GET | `/my/orders/:orderId/shipments` | Read tracking from PostgreSQL |

No courier call. No status change.

`data`: `{ "shipments": [ Shipment ] }`. Each shipment includes `events` as in section 2. No pagination (one order). No NDR notes, no book attempts.

---

## 6. Public — webhook

### POST /webhooks/:code

**Auth:** none. Body is the raw courier payload. Do not parse it in the handler.

| Situation | HTTP | Body |
|---|---|---|
| Unknown `:code` | 404 | error envelope, `FULFILLMENT_PROVIDER_NOT_SUPPORTED` |
| Shipment found, bad signature | 401 | error envelope, `FULFILLMENT_WEBHOOK_UNVERIFIED`. Nothing stored |
| No matching shipment | 200 | success envelope, `data` omitted. Nothing stored |
| Verified (applied, duplicate, ignore, or apply failed internally) | 200 | success envelope, `data` omitted |

---

## 7. Status and error codes

| HTTP | When |
|---|---|
| 200 | Read, book accepted, cancel, refresh, idempotent replay |
| 201 | Draft created, plan created new drafts, return created |
| 400 | Validation, capability missing, weight, bad credentials, unknown list filter |
| 401 | Missing JWT, or webhook signature failed |
| 404 | Missing id or wrong owner, unknown courier code |
| 409 | Illegal status, address changed, stock mismatch, courier refused cancel |
| 500 | Encryption key missing |
| 502 | Courier 5xx / 429 after a book was marked (`FULFILLMENT_BOOK_FAILED`) |
| 503 | Courier temporarily unavailable on a read-only call such as rates or label fetch |

| Code | HTTP |
|---|---|
| `FULFILLMENT_NOT_FOUND` | 404 |
| `FULFILLMENT_INVALID_STATE` | 409 |
| `FULFILLMENT_PROVIDER_NOT_SUPPORTED` | 404 |
| `FULFILLMENT_PROVIDER_NOT_CONFIGURED` | 409 |
| `FULFILLMENT_CREDENTIALS_INVALID` | 400 |
| `FULFILLMENT_ENCRYPTION_KEY_MISSING` | 500 |
| `FULFILLMENT_RATE_FAILED` | 502 |
| `FULFILLMENT_BOOK_FAILED` | 502 |
| `FULFILLMENT_LABEL_FAILED` | 400 or 503 |
| `FULFILLMENT_CAPABILITY_UNSUPPORTED` | 400 |
| `FULFILLMENT_WEBHOOK_UNVERIFIED` | 401 |
| `FULFILLMENT_APPLY_MISMATCH` | 409 |
| `FULFILLMENT_STOCK_MISMATCH` | 409 |
| `FULFILLMENT_WEIGHT_REQUIRED` | 400 |
| `FULFILLMENT_ADDRESS_CHANGED` | 409 |
| `FULFILLMENT_AUTO_BOOK_AMBIGUOUS` | 409 |
| `CORRELATION_ID_REQUIRED` | 400 |

`FULFILLMENT_AUTO_BOOK_AMBIGUOUS` is raised by the planner, not by a request field. It is listed so the seller UI can show why drafts were left unbooked.

---

## 8. Not in this API

- COD remittance and statement import
- Backorder / purchase orders
- Label storage or a label URL
- A customer call that hits the courier
- Admin override of a terminal status
