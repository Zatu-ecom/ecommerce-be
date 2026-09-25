# Research: Payment Gateway Platform

**Feature**: `010-payment-gateway-platform`  
**Date**: 2026-09-06  
**Sources**: `spec.md`, `pre-spec.md`, `plans/010-payment-seller-dashboard-gaps/`, `plans/payment-backend-handoff.md`, current `payment/` code on `009-razorpay-payment-gateway`.

All Technical Context items are resolved. No `[NEEDS CLARIFICATION]` remain.

---

## R-01 — Document hierarchy when older research conflicts

**Decision:** `spec.md` + `pre-spec.md` are binding. `phased-todos.md` / original `plans/010-payment-seller-dashboard-gaps/plan.md` are a checklist only.

**Superseded items in phased-todos (do not implement):**

| Old text | Binding replacement |
|----------|---------------------|
| `GET /transactions/:id/events` dedicated route | Events **only** on `GET /transactions/:id` (spec FR-020, pre-spec §10). List never includes events. |
| `payment_gateway_config.country` → `country_id` FK | **Drop country on config entirely** (pre-spec §8). Geo lives on catalog join tables + seller_settings. |
| Optional PATCH seller settings | Existing **PUT** ` /api/user/seller/settings` gains `paymentsEnvironment`. |
| Keep `/webhooks/razorpay` alias | Same path as `/:code` with `code=razorpay`. **No** Razorpay-named handler. |
| `VerifyWebhook` + `ParseWebhook` as two interface methods | One `NormalizeWebhook` after locate (pre-spec §4). |

**Rationale:** Implementing both designs would split flash-model implementers.  
**Alternatives considered:** Merge dedicated events route “for completeness” — rejected (N+1, extra surface, spec forbids).

---

## R-02 — In-place migrations (no 032)

**Decision:** Rewrite `migrations/009_create_payment_tables.sql`, slim `031` to event table + `event_id NOT NULL` unique, add `payments_environment` on `seller_settings` in `008`, update seed `004`. Recreate local/test DBs. No backfill.

**Rationale:** Payment not in prod. Additive ALTER would leave arrays, `webhook_url`, `payment_method`, `config.country`.  
**Alternatives:** `032` drop columns — rejected by FR-032.

---

## R-03 — One `PaymentGateway` interface per provider folder

**Decision:** Move Razorpay into `payment/service/payment_gateway/razorpay/`. Shared package holds `contract.go` + `crypto.go` only. Factory `map[code]PaymentGateway`. Base services call **only** this interface.

**Rationale:** Today initiate/refund/configure/webhook all leak Razorpay. OCP requires closed orchestrators.  
**Alternatives:** Separate `CredentialCodec` + `WebhookVerifier` interfaces — rejected (same methods, extra types, pre-spec §1.1).

---

## R-04 — Locate then verify (never `configs[0]`)

**Decision:** `PeekLocators(rawBody)` (untrusted) → find txn by session/payment/our txn id → load config for `txn.seller_id` + `txn.gateway_id` + **`txn.environment`** → `Decrypt` → `NormalizeWebhook` (HMAC on **raw bytes**, `hmac.Equal`). Wrong secret → 401, no state change. No txn → **do not persist body**; return 200 (initiate may still be in flight). Cron heals.

**Rationale:** Current `FindAllForGateway()[0]` is a cross-tenant defect. Brute-force HMAC across sellers does not scale.  
**Alternatives:** Per-seller webhook path tokens — later escape hatch, not v1.

---

## R-05 — Dual sandbox/production + frozen txn environment

**Decision:** Use existing unique `(seller_id, gateway_id, environment)`. Store mode `seller_settings.payments_environment` (`sandbox` default). Copy store mode onto `payment_transaction.environment` at initiate. Refunds/webhooks/cron use **txn environment**, not the current toggle.

**Rationale:** Mixing live charges with test keys is a product/security failure.  
**Alternatives:** Single config row + a boolean — rejected (unique key already exists; dashboard needs both saved).

---

## R-06 — Geo join tables, not VARCHAR arrays

**Decision:** `payment_gateway_country` and `payment_gateway_currency` with FKs to `country.id` / `currency.id`. Drop `supported_countries`, `supported_currencies`, `webhook_url`. Initiate: `EXISTS` seller `business_country_id` and `base_currency_id` on those joins.

**Rationale:** Indexed FK membership vs GIN array + string `IN` hardcoded in configure.  
**Alternatives:** Keep arrays for “simplicity” — rejected (does not join seller_settings; does not scale).

---

## R-07 — Checkout payload

**Decision:** HTTP initiate returns `checkout: { gatewayCode, fields }`. Razorpay adapter sets `fields.keyId` and `fields.orderId`. **Delete** top-level `keyId` on `InitiatePaymentResponse`. No compatibility field.

**Rationale:** Stripe needs `clientSecret`; orchestrator must not copy Razorpay key id. FE updates in the same cycle (spec assumption).  
**Alternatives:** Keep `keyId` alias — rejected by FR-004 / FR-032.

---

## R-08 — Shared `applyNormalized` + reconciliation cron

**Decision:** Webhook and cron call the same apply function. Cron every 5 minutes (`common/cron` `RegisterIntervalJob`), name `payment.reconcile_pending`. Pending older than `PAYMENT_PENDING_TTL_MINUTES` (default 45): `FetchRemoteStatus`; if still open → fail `EXPIRED_UNPAID` + fail order. Empty `gateway_session_id` after 5 minutes → fail. Refunds pending/processing older than `REFUND_STUCK_TTL_MINUTES` (default 30). Batch 100, `FOR UPDATE SKIP LOCKED`, skip overlapping run. `FetchRemoteStatus` GET retries 2× on 429/5xx; POST never retries.

**Rationale:** Webhooks are best-effort; abandoned Checkout leaves `pending` forever today. Second webhook-replay cron doubles provider QPS.  
**Alternatives:** Only webhooks — rejected (P1 stuck payments). Poll every second — rejected (cost).

---

## R-09 — Refunds and list filters

**Decision:** Allow refunds when status is `completed` **or** `partially_refunded`, up to remaining = amount − (pending+processing+completed refunds). List `?status=` validated enum, SQL filter, `pageSize` cap 100.

**Rationale:** Current code blocks a second partial refund; FE chips filter client-side on one page.  
**Alternatives:** Dedicated `GET /refunds` — out of scope.

---

## R-10 — Test connection

**Decision:** `POST /api/payment/gateways/:code/test`. Optional credentials; merge empty secrets from stored row; **do not persist** unsaved values. Razorpay: authenticated GET orders `count=1`. 401 → 400 invalid credentials. Webhook secret: length/format only.

**Rationale:** Saving encrypted JSON does not prove keys work. Mirror file-module storage test.  
**Alternatives:** Test only after save — worse UX for “try keys first.”

---

## R-11 — GET transaction by id for seller

**Decision:** `GET /api/payment/transactions/:transactionId` must work for **customer owner and seller owner**. Today the route is `CustomerAuth` only, which blocks the dashboard detail page.

**Rationale:** Spec story 4 is seller-centric.  
**Implementation:** One route; middleware that accepts customer **or** seller JWT (same pattern as checking `roleName` in handler). Service: customer → `user_id`; seller → `seller_id`. 404 otherwise (no leak).

**Alternatives:** Separate `/seller/transactions/:id` — extra surface, not requested.

---

## R-12 — Drop unused artifacts

**Decision:** Delete `payment/entity/payment_method.go` + table from 009. Delete `payment/model/gateway_response.go`. Delete package-level Razorpay encrypt helpers after move. Do not persist unverified webhook bodies. Do not store full webhook header dumps.

**Rationale:** Dead code and unauthenticated writes hurt security and scale.

---

## R-13 — Config and env

**Decision:**

| Env | Default | Use |
|-----|---------|-----|
| `PUBLIC_API_BASE_URL` | `http://localhost:8080` | Compose `webhookUrl` |
| `PAYMENT_PENDING_TTL_MINUTES` | `45` | Cron unpaid |
| `PAYMENT_SESSIONLESS_TTL_MINUTES` | `5` | Fail if no session |
| `REFUND_STUCK_TTL_MINUTES` | `30` | Cron refunds |
| `RAZORPAY_BASE_URL` | Razorpay API | Existing; tests override |
| `ENCRYPTION_KEY` | existing app config | Fail configure/decrypt if empty in non-dev if you cannot encrypt |

**Rationale:** Pre-spec §5.4, §7.1.  
**Note:** `common/config` currently defaults a local encryption key — configure must still fail if encryption cannot run; never store plaintext secrets.

---

## R-14 — Constitution VIII vs rewrite 009

**Decision:** Justified exception. See plan.md Complexity Tracking.

---

## R-15 — Adding a second provider (definition of done, not this feature’s code)

**Decision:** Stripe/Cashfree **code is out of scope**. The extension point **is** in scope: no `switch code` in `payment_service`, `webhook_service`, `payment_gateway_service`, handlers, or routes (except `/:code` path param).

Checklist for a future provider (do not do now): folder implementing interface, seed + join rows, one factory line, httptest suite.
