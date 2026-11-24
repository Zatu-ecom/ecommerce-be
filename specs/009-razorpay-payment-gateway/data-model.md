# Data Model: Razorpay Payment Gateway Integration

This documents the resulting data model after migration `031`. Existing tables from migration `009`
that are unchanged are summarized only where relevant.

**Entity naming**: The business spec's "Payment" = `payment_transaction`, "Payment Event" =
`payment_transaction_event`, "Provider Configuration" = `payment_gateway_config`, "Payment Provider"
= `payment_gateway`, "Notification Log" = `payment_webhook_log`. Money is stored in minor units
(`*_cents`); for INR this is paise.

## payment_gateway (modified)

Catalog of supported payment providers.

| Column | Type | Notes |
|--------|------|-------|
| id | BIGSERIAL PK | |
| code | VARCHAR(50) UNIQUE NOT NULL | `razorpay` |
| name | VARCHAR(100) NOT NULL | `Razorpay` |
| description | TEXT | |
| logo_file_id | VARCHAR(80) NULL | Replaces `logo_url`; resolved via file service |
| is_active | BOOLEAN | |
| supported_countries | VARCHAR(2)[] | `{IN}` |
| supported_currencies | VARCHAR(3)[] NOT NULL | `{INR}` |
| supported_payment_methods | TEXT[] NOT NULL | card, upi, wallet, netbanking, emi, ... |
| webhook_url | VARCHAR(500) | |
| created_at / updated_at | TIMESTAMPTZ | |

## payment_gateway_field (existing, seeded)

Configuration fields a seller must supply for a provider.

| Column | Type | Notes |
|--------|------|-------|
| id | BIGSERIAL PK | |
| gateway_id | BIGINT FK → payment_gateway | |
| field_name | VARCHAR(100) NOT NULL | `key_id`, `key_secret`, `webhook_secret`, `account_id` |
| display_name | VARCHAR(200) NOT NULL | |
| field_type | VARCHAR(50) NOT NULL | |
| is_required | BOOLEAN | |
| is_sensitive | BOOLEAN | true for `key_secret`/`webhook_secret` |
| display_order | INT | |
| validation_rules | JSONB | |

Unique `(gateway_id, field_name)`.

## payment_gateway_config (existing)

A seller's credentials/settings for a provider.

| Column | Type | Notes |
|--------|------|-------|
| id | BIGSERIAL PK | |
| seller_id | BIGINT NOT NULL | FK → seller_profile |
| gateway_id | BIGINT NOT NULL | FK → payment_gateway |
| environment | VARCHAR(20) NOT NULL | `sandbox`/`production` |
| credentials | JSONB NOT NULL | field-level encrypted secrets; `key_id` plaintext |
| is_active | BOOLEAN | |
| priority | INT | |
| country | VARCHAR(2) NOT NULL | |
| created_at / updated_at | TIMESTAMPTZ | |

Unique `(seller_id, gateway_id, environment)`.

## payment_transaction (modified)

Current-state aggregate for a payment.

| Column | Type | Notes |
|--------|------|-------|
| id | BIGSERIAL PK | |
| transaction_id | VARCHAR(50) UNIQUE NOT NULL | our internal id; also stored on `order.transaction_id` |
| reference_type | VARCHAR(20) | `order` / `subscription` |
| reference_id | BIGINT | polymorphic owner id (no FK) |
| user_id | BIGINT NOT NULL | customer |
| seller_id | BIGINT NOT NULL | |
| gateway_id | BIGINT NULL | provider; NULL reserved for future COD |
| gateway_session_id | VARCHAR(255) | provider checkout/order/intent id |
| gateway_payment_id | VARCHAR(255) | provider captured payment/charge id |
| currency | VARCHAR(3) NOT NULL | seller base currency |
| amount_cents | BIGINT NOT NULL | order total (minor units) |
| gateway_fee_cents | BIGINT NULL | set on capture |
| status | VARCHAR(30) NOT NULL | see state machine below |
| failure_code | VARCHAR(100) | |
| failure_message | TEXT | |
| payment_method_type | VARCHAR(50) | |
| payment_method_details | JSONB | |
| completed_at | TIMESTAMPTZ | |
| created_at / updated_at | TIMESTAMPTZ | |

Removed: `metadata`, `initiated_at`, `gateway_transaction_id`. `gateway_request`/`gateway_response`
do **not** live here (see event table).

Indexes: `(reference_type, reference_id)`, `gateway_session_id`, `gateway_payment_id`,
`seller_id`, `user_id`, `status`, `transaction_id` (unique).

### payment_transaction state machine

```
pending → completed
pending → failed
completed → refunded
completed → partially_refunded
```

Transitions are conditional (`WHERE status = expected`); terminal states do not regress.

## payment_transaction_event (new)

Append-only history of every payment state change.

> The event entity has no `updated_at` (append-only, never mutated). It mirrors
> `payment_webhook_log`: only `id` + `created_at`. Do **not** embed `db.BaseEntity` (which adds an
> `UpdatedAt` column GORM would expect in the table).

| Column | Type | Notes |
|--------|------|-------|
| id | BIGSERIAL PK | |
| transaction_id | BIGINT NOT NULL FK | → payment_transaction (ON DELETE CASCADE) |
| event_type | VARCHAR(50) NOT NULL | initiated, gateway_session_created, authorized, captured, completed, failed, refund_initiated, refund_completed, refund_failed |
| from_status | VARCHAR(30) | |
| to_status | VARCHAR(30) NOT NULL | |
| gateway_event_id | VARCHAR(255) | provider event id |
| gateway_request | JSONB | outbound provider call |
| gateway_response | JSONB | provider response |
| failure_code | VARCHAR(100) | |
| failure_message | TEXT | |
| source | VARCHAR(20) NOT NULL | api / webhook / system / admin |
| actor_id | BIGINT | |
| actor_type | VARCHAR(20) | customer / seller / admin / system |
| created_at | TIMESTAMPTZ NOT NULL | |

Indexes: `transaction_id`, `created_at DESC`, `event_type`.

## payment_refund (existing)

Tracks full/partial refunds.

| Column | Type | Notes |
|--------|------|-------|
| id | BIGSERIAL PK | |
| refund_id | VARCHAR(50) UNIQUE NOT NULL | |
| transaction_id | BIGINT NOT NULL FK | → payment_transaction |
| gateway_refund_id | VARCHAR(255) | |
| currency | VARCHAR(3) NOT NULL | |
| amount_cents | BIGINT NOT NULL | |
| status | VARCHAR(30) NOT NULL | pending / processing / completed / failed |
| failure_reason | TEXT | |
| reason | VARCHAR(100) | |
| notes | TEXT | |
| initiated_by | BIGINT | |
| initiated_by_type | VARCHAR(20) | |
| completed_at | TIMESTAMPTZ | |
| metadata | JSONB | |
| created_at / updated_at | TIMESTAMPTZ | |

### payment_refund state machine

- `refund.created` → `pending`
- `refund.processed` → `completed`
- `refund.failed` → `failed`

`processing` is the state set after a successful outbound refund request but before the provider's
terminal webhook (`refund.processed`/`refund.failed`) arrives; it is a transitional state.

## payment_webhook_log (existing + new unique index)

Raw inbound provider notification audit.

| Column | Type | Notes |
|--------|------|-------|
| id | BIGSERIAL PK | |
| gateway_id | BIGINT | |
| event_type | VARCHAR(100) NOT NULL | |
| event_id | VARCHAR(255) | unique per `(gateway_id, event_id)` when not null |
| payload | JSONB NOT NULL | |
| headers | JSONB | |
| status | VARCHAR(30) NOT NULL | received / processed / failed / ignored |
| error_message | TEXT | |
| processed_at | TIMESTAMPTZ | |
| transaction_id | BIGINT | |
| refund_id | BIGINT | |
| ip_address | VARCHAR(50) | |
| created_at | TIMESTAMPTZ | |

New index: unique partial `(gateway_id, event_id) WHERE event_id IS NOT NULL` (idempotency).
