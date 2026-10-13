# Data Model: Payment Gateway Platform

Binding: [pre-spec.md](./pre-spec.md) §8. Applied by **editing** `009` / `031` / `008` / seed `004` in place (no `032`). Recreate local databases after the rewrite.

Money is always integer minor units (`*_cents`). GORM `SingularTable: true`.

---

## seller_settings (008 — add column)

| Column | Type | Notes |
|--------|------|-------|
| *(existing columns unchanged)* | | |
| **payments_environment** | `VARCHAR(20) NOT NULL DEFAULT 'sandbox'` | `sandbox` \| `production`. Store checkout mode. |

Entity: `user/entity/seller_settings.go`. Seed mock sellers inherit default `sandbox`.

---

## payment_gateway (009 rewrite)

Catalog row. **Removed:** `supported_countries`, `supported_currencies`, `webhook_url`, `logo_url`.

| Column | Type | Notes |
|--------|------|-------|
| id | BIGSERIAL PK | |
| code | VARCHAR(50) UNIQUE NOT NULL | e.g. `razorpay` |
| name | VARCHAR(100) NOT NULL | |
| description | TEXT | |
| logo_file_id | VARCHAR(80) NULL | File module; resolve via `FileDisplayGateway` |
| is_active | BOOLEAN NOT NULL DEFAULT TRUE | |
| supported_payment_methods | TEXT[] NOT NULL | Optional filter if client sends `paymentMethodType` |
| created_at / updated_at | TIMESTAMPTZ | |

Indexes: `code`; GIN on `supported_payment_methods` optional (small array).

---

## payment_gateway_country (NEW in 009)

| Column | Type | Notes |
|--------|------|-------|
| gateway_id | BIGINT NOT NULL FK → payment_gateway(id) ON DELETE CASCADE | |
| country_id | BIGINT NOT NULL FK → country(id) | |

`UNIQUE (gateway_id, country_id)`. Indexes on both FKs.

Seed Razorpay: `country.code = 'IN'`.

---

## payment_gateway_currency (NEW in 009)

| Column | Type | Notes |
|--------|------|-------|
| gateway_id | BIGINT NOT NULL FK → payment_gateway(id) ON DELETE CASCADE | |
| currency_id | BIGINT NOT NULL FK → currency(id) | |

`UNIQUE (gateway_id, currency_id)`. Indexes on both FKs.

Seed Razorpay: `currency.code = 'INR'`.

---

## payment_gateway_field (unchanged idea)

Per-provider credential schema for the dashboard.

Razorpay fields (seed): `key_id` (not sensitive), `key_secret` (sensitive), `webhook_secret` (sensitive), `account_id` (optional, not sensitive).

---

## payment_gateway_config (009 rewrite)

| Column | Type | Notes |
|--------|------|-------|
| id | BIGSERIAL PK | |
| seller_id | BIGINT NOT NULL FK → seller_profile(user_id) | |
| gateway_id | BIGINT NOT NULL FK → payment_gateway | |
| environment | VARCHAR(20) NOT NULL | `sandbox` \| `production` |
| credentials | JSONB NOT NULL | Encrypted sensitive fields; never returned |
| is_active | BOOLEAN DEFAULT TRUE | |
| priority | INT DEFAULT 0 | Higher wins on initiate select |
| created_at / updated_at | TIMESTAMPTZ | |

**Removed:** `country`.  
`UNIQUE (seller_id, gateway_id, environment)`.

Repos **must** query by environment. Never `First()` without environment.

---

## payment_method — DELETE

Drop table from 009. Delete `payment/entity/payment_method.go`. No APIs.

---

## payment_transaction (009 CREATE already includes 031 columns)

Do **not** create then ALTER for session ids if 009 is rewritten as the final CREATE.

| Column | Type | Notes |
|--------|------|-------|
| id | BIGSERIAL PK | |
| transaction_id | VARCHAR(50) UNIQUE NOT NULL | Public id `TXN_…`; also on `order.transaction_id` |
| user_id | BIGINT NOT NULL | Customer |
| seller_id | BIGINT NOT NULL | Tenant |
| gateway_id | BIGINT NULL FK | |
| reference_type | VARCHAR(20) | `order` (subscription column kept, no flow) |
| reference_id | BIGINT | Polymorphic; no FK |
| gateway_session_id | VARCHAR(255) | Provider order/intent id |
| gateway_payment_id | VARCHAR(255) | Provider capture id |
| currency | VARCHAR(3) NOT NULL | From order/seller base |
| amount_cents | BIGINT NOT NULL | From order; **never** client |
| gateway_fee_cents | BIGINT NULL | Set on capture |
| status | VARCHAR(30) NOT NULL | See state machine |
| failure_code / failure_message | | e.g. `EXPIRED_UNPAID` |
| payment_method_type | VARCHAR(50) | |
| payment_method_details | JSONB | Fill on capture from webhook payload when present |
| **environment** | VARCHAR(20) NOT NULL | Frozen copy of store mode at initiate |
| completed_at | TIMESTAMPTZ | |
| created_at / updated_at | TIMESTAMPTZ | |

**Removed vs original 009:** `gateway_transaction_id`, `initiated_at`, `metadata`.

**Indexes:** `(seller_id)`, `(user_id)`, `(status, created_at)` for cron, `(gateway_session_id)`, `(gateway_payment_id)`, `(reference_type, reference_id)`, `(transaction_id)`.

Duplicate initiate: reject if another row for same reference is `pending` or `completed`. `failed` does **not** block.

---

## payment_transaction_event (031 CREATE, or fold into 009)

Append-only. No `updated_at`. Never UPDATE rows.

| Column | Type | Notes |
|--------|------|-------|
| id | BIGSERIAL PK | |
| transaction_id | BIGINT FK → payment_transaction(id) ON DELETE CASCADE | PK of txn, not public string |
| event_type | VARCHAR(50) NOT NULL | `initiated`, `gateway_session_created`, `authorized`, `captured`, `completed`, `failed`, `refund_initiated`, `refund_completed`, `refund_failed` |
| from_status / to_status | VARCHAR(30) | |
| gateway_event_id | VARCHAR(255) | |
| gateway_request / gateway_response | JSONB | Sanitized; no secrets |
| failure_code / failure_message | | |
| source | VARCHAR(20) NOT NULL | `api` \| `webhook` \| `system` \| `admin` |
| actor_id / actor_type | | |
| created_at | TIMESTAMPTZ | |

Returned **only** on GET transaction by id, ordered `created_at ASC`.

---

## payment_refund (009, keep)

Statuses: `pending`, `processing`, `completed`, `failed`.  
Refundable remaining = txn.amount_cents − SUM(refunds where status in pending, processing, completed).

---

## payment_webhook_log (009 + 031 unique)

| Column | Change |
|--------|--------|
| event_id | **VARCHAR NOT NULL** (always set by adapter, composite fallback `{providerEvent}:{entityId}`) |
| unique | `(gateway_id, event_id)` **without** `WHERE event_id IS NOT NULL` |
| payload | Store **after** verify only |
| headers | Do not dump full headers (skip PII) |
| transaction_id | Set when txn located |
| status | `received` / `processed` / `failed` / `ignored` |

Index `(created_at DESC)`. Seller list: join `payment_transaction.seller_id`. Unmatched logs (null txn) are **not** returned to sellers.

---

## State machines

### payment_transaction.status

```
pending → completed | failed
completed → partially_refunded | refunded
partially_refunded → refunded | partially_refunded (another partial)
failed → (terminal for that row; new initiate allowed)
```

- `authorized` webhook: **stay pending**, append event only.
- Cron expire: pending → failed (`EXPIRED_UNPAID`).
- Amount/currency mismatch on complete: **do not** transition; log failed.

Conditional updates: `UPDATE … WHERE status = expected`. Loser is no-op.

### payment_refund.status

```
pending → processing → completed | failed
```

Provider may skip pending. Failed refund does not change txn to failed.

---

## Validation rules

- `environment` / `payments_environment`: only `sandbox` | `production`.
- List `status` query: only the five txn statuses; else 400.
- `pageSize` default 20, max 100 (`common.BaseListParams` pattern).
- Secrets: AES-256-GCM via existing helper; empty key → fail closed.
- Initiate amount/currency from order, not request body (body has `orderId` + optional `paymentMethodType` only).

---

## Relationships (implementer)

- Gateway 1—* configs, 1—* country links, 1—* currency links, 1—* fields.
- Txn *—1 gateway, *—1 seller, 1—* events, 1—* refunds.
- Webhook log *—0..1 txn.

Cross-module: `order.transaction_id` string; payment never imports order repositories.

---

## 031 file after slim

If 009 CREATE already has txn session columns, logo_file_id, events, webhook unique:

`031` should **only** `CREATE TABLE IF NOT EXISTS payment_transaction_event` and unique index on webhook log — **or** fold events into 009 and leave 031 as a no-op comment + `SELECT 1` so numbering stays. Prefer: events in 009; 031 becomes comments explaining folded ALTERs so `RunAllMigrations` still applies the file without error.

**Do not** leave 031 ALTERs that drop columns 009 no longer creates (`DROP COLUMN metadata` on a table that never had it is OK/idempotent but confusing). Prefer clean 009 CREATE + slim 031.
