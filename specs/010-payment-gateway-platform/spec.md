# Feature Specification: Payment Gateway Platform

**Feature Branch**: `010-payment-gateway-platform`  
**Created**: 2026-09-06  
**Status**: Draft  
**Base branch**: Created from `009-razorpay-payment-gateway` (existing checkout).  
**Input**: Platform-ize payments using research in `plans/010-payment-seller-dashboard-gaps/` (`pre-spec.md`, `plan.md`) and seller dashboard handoff `plans/payment-backend-handoff.md`.

**Research (must be read by implementers — do not invent a second design):**

- [`pre-spec.md`](./pre-spec.md) — architecture (copy of `plans/010-payment-seller-dashboard-gaps/pre-spec.md`)
- [`plan.md`](./plan.md) — speckit implementation plan
- [`research.md`](./research.md) — decisions including which older todos are superseded
- [`data-model.md`](./data-model.md) · [`contracts/`](./contracts/) · [`quickstart.md`](./quickstart.md)
- [`phased-todos.md`](./phased-todos.md) — checklist only; **spec + pre-spec win on conflicts** (no dedicated events route; no `config.country`)
- [`tasks.md`](./tasks.md) — executable implementation tasks (T001–T066)
- [`plans/payment-backend-handoff.md`](../../plans/payment-backend-handoff.md) — seller UI gaps

This spec states **what must be true for users and the business**. How to build it is in the pre-spec and speckit plan artifacts. `/speckit.implement` MUST follow them; it MUST NOT contradict them.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Customer pays a pending order (Priority: P1)

A customer with a pending order starts checkout. The store uses whichever payment provider the seller has enabled for the store’s current test-or-live mode, as long as that provider supports the seller’s country and currency. The customer receives a **provider-neutral checkout payload** (not a Razorpay-only key field). After they pay at the provider, the payment becomes completed and the order is confirmed without the customer calling a second “confirm” API. If they abandon checkout, the payment does not stay pending forever.

**Why this priority**: Revenue path. Nothing else matters if checkout is wrong or stuck.

**Independent Test**: Create pending order as customer of a seller with sandbox credentials; initiate payment; simulate successful provider capture; order confirmed and payment completed. Abandon a second initiate; after the unpaid timeout the payment is failed and the order is failed.

**Acceptance Scenarios**:

1. **Given** a pending order owned by the customer and an active sandbox config for a provider that supports the seller’s country and currency, **When** the customer initiates payment, **Then** a pending payment is created, a provider checkout session exists, the order is linked to that payment, and the response includes `checkout.gatewayCode` plus `checkout.fields` (for Razorpay: `keyId` and `orderId` inside `fields` — **no top-level `keyId`**).
2. **Given** a pending payment with a provider session, **When** the provider delivers a verified “payment completed” notification, **Then** the payment is completed, fees and payment method details are stored, an event is recorded, and the order is confirmed.
3. **Given** a pending payment, **When** the provider delivers a verified “payment failed” notification, **Then** the payment and order are failed with a failure reason.
4. **Given** a pending payment, **When** the provider delivers a verified “authorized only” notification, **Then** an observation event is stored and the payment stays pending (order is not confirmed).
5. **Given** a pending payment older than the unpaid timeout with no successful capture, **When** the background reconciler runs, **Then** the provider is queried and the payment is completed if paid, or failed (`EXPIRED_UNPAID`) if still unpaid, and the order follows.
6. **Given** initiate already created a pending or completed payment for that order, **When** the customer initiates again, **Then** the request is rejected as a duplicate (failed prior payments do **not** block a new initiate).
7. **Given** the customer is not the order owner, **When** they initiate, **Then** the payment is not created (not found).
8. **Given** store mode is production but only sandbox credentials exist, **When** the customer initiates, **Then** payment is not started (gateway not configured). Do not silently use sandbox keys.
9. **Given** the seller’s country or currency is not supported by any enabled provider in the current store mode, **When** the customer initiates, **Then** payment is not started with a clear unsupported geo/currency outcome.
10. **Given** the client sends a payment method type that the catalog does not list for that provider, **When** they initiate, **Then** the request is rejected. If they omit method type, initiate still proceeds.

---

### User Story 2 - Seller configures sandbox and live independently (Priority: P1)

A seller sees available providers, configures **sandbox and production separately**, updates keys without re-pasting every secret, tests that keys work, copies the webhook URL to paste into the provider dashboard, toggles the store between test and live, and deactivates one environment without deleting the other.

**Why this priority**: Without this, FE cannot ship a real payments settings UI; live vs test mix-ups charge real money on test keys.

**Independent Test**: As seller, configure sandbox, see masked key hint and webhook URL, test connection success, switch store to live without production keys (checkout blocked), configure production, toggle live, deactivate sandbox only.

**Acceptance Scenarios**:

1. **Given** an authenticated seller, **When** they list providers, **Then** each row shows name, logo if present, supported countries/currencies/methods, whether **sandbox** is configured, whether **production** is configured, current store payments mode, and the composed webhook URL for that provider code.
2. **Given** they open one provider, **When** they view detail, **Then** they see field schema (required, sensitive, validation) and **two** config slots (sandbox and production), each with masked non-secret hints (e.g. truncated key id, full account id). Secrets are never returned, even encrypted.
3. **Given** they save credentials for `environment=sandbox`, **When** save succeeds, **Then** only the sandbox row is upserted; production is unchanged.
4. **Given** sandbox is already saved, **When** they send an update with a new key id and empty secrets, **Then** stored secrets are kept and the key id is updated.
5. **Given** they call test connection with unsaved or saved credentials, **When** the provider accepts the keys, **Then** they see success without the test persisting secrets. Invalid keys return a validation-style failure, not a crash.
6. **Given** they deactivate sandbox, **When** production remains, **Then** only sandbox is inactive.
7. **Given** they set store payments mode to production, **When** checkout runs, **Then** only the production credential row is used.
8. **Given** encryption key is missing, **When** they save credentials, **Then** save fails closed (nothing stored in plaintext).

---

### User Story 3 - Provider notifications are tenant-safe and provider-agnostic (Priority: P1)

Payment providers post notifications to a **single URL pattern per provider code**. The platform finds the payment first, then verifies the signature with **that seller’s** secret. Completing a payment never uses another seller’s webhook secret. Unknown provider codes are rejected. Duplicate notifications do not double-complete a payment. Adding a future provider must not require a new named notification URL besides `{code}`.

**Why this priority**: Multi-tenant security bug exists today (first seller’s secret). This is a hard stop.

**Independent Test**: Two sellers, two secrets; notification for seller A’s session signed with seller B’s secret is rejected; A’s secret completes A only. Replay of the same event id is a no-op. Missing correlation header still accepted on this public URL.

**Acceptance Scenarios**:

1. **Given** a notification to `/api/payment/webhooks/{code}` with a valid signature for the payment’s seller, **When** it is a completed-payment event, **Then** that seller’s payment completes (see Story 1).
2. **Given** the same body signed with another seller’s secret, **When** posted, **Then** no status change and the caller is unauthorized.
3. **Given** locators in the body do not match any payment, **When** posted, **Then** the body is **not** stored and no money movement occurs (avoid unauthenticated storage). A later reconciler still heals pending payments.
4. **Given** the same provider event id is posted twice after success, **When** the second arrives, **Then** the second is ignored and the payment stays correctly completed once.
5. **Given** amount or currency on a completed event does not match our payment, **When** verified, **Then** the payment is **not** completed (mismatch is recorded as a failed processing of that notification).
6. **Given** an unknown `{code}`, **When** posted, **Then** not found.
7. **Given** no correlation-id header, **When** a valid notification is posted, **Then** it is still processed (providers do not send that header).
8. **Given** a future provider folder is registered, **When** they post to `/webhooks/{that-code}`, **Then** the same locate → verify → apply-action flow runs. Core payment/webhook orchestration MUST NOT contain provider-specific event names.

---

### User Story 4 - Seller manages payments, refunds, and history (Priority: P2)

A seller lists payments with **server-side status filters** that match dashboard chips (pending, completed, failed, refunded, partially refunded). They open one payment and see remaining refundable amount, which environment it was charged in, and the full event history. They issue a full or **second partial** refund on a completed or already partially refunded payment. They can page through inbound notification logs for their own payments (not the checkout hot path).

**Why this priority**: Unblocks seller dashboard; refunds and filters are wrong today.

**Independent Test**: Filter list by failed/refunded with correct totals across pages. Detail shows refundable amount and events. Two partial refunds allowed until remaining is zero. Other seller sees empty logs.

**Acceptance Scenarios**:

1. **Given** mixed statuses, **When** the seller lists with `status=failed` (and page size), **Then** only failed rows are returned and `total` matches that filter. Unknown status is rejected. Page size is capped (100).
2. **Given** a completed payment, **When** they view detail, **Then** they see `refundableAmountCents` equal to amount minus pending/processing/completed refunds, `environment`, and ordered `events`. **List** endpoints do not attach events.
3. **Given** a completed payment, **When** they refund part, **Then** a processing refund is created and an event is recorded. Provider refund uses credentials for **the payment’s environment**, not the current store toggle.
4. **Given** a partially refunded payment with remaining balance, **When** they refund again within remaining, **Then** it is allowed. Over remaining is rejected. Refund on pending is rejected.
5. **Given** a verified refund-completed notification, **When** applied, **Then** refund is completed and the payment becomes `refunded` or `partially_refunded` based on totals.
6. **Given** a verified refund-failed notification, **When** applied, **Then** the refund is failed and the payment stays completed (or previous refunded state unchanged for that failed attempt).
7. **Given** webhook logs, **When** the seller pages them, **Then** they only see logs tied to **their** payments. Another seller sees none of those rows.

---

### User Story 5 - Platform can add another provider without rewriting checkout (Priority: P2)

Engineering can add a second provider by supplying a provider pack (credentials encrypt/decrypt/mask, checkout session, refund, test, webhook normalize) plus catalog seed (countries/currencies/methods). Checkout, refunds, webhooks, and seller settings continue to work without rewriting those orchestrations. Catalog countries/currencies are the platform country and currency records (not free-text arrays).

**Why this priority**: Explicit product/architecture goal; Razorpay-only leaks block this today.

**Independent Test**: With only Razorpay registered, checkout works. Code review / test: orchestration contains no Razorpay event names or Razorpay encrypt helpers. Seed uses India + INR via country/currency records. (A second provider implementation is out of scope; the **extension point** is in scope.)

**Acceptance Scenarios**:

1. **Given** multiple active configs for the store mode, **When** checkout selects a provider, **Then** it picks the highest **priority** among those that support the seller’s country and currency.
2. **Given** a new provider pack is registered and seeded, **When** a seller configures it and a customer pays, **Then** initiate, webhook, refund, and test work through the same seller and customer journeys.
3. **Given** the catalog, **When** support countries/currencies are shown, **Then** they are platform country/currency records linked to the provider (not a disconnected string list).

---

### Edge Cases

- Initiate fails at the provider after a pending row was created: payment is marked **failed** so the customer can initiate again.
- Linking the payment to the order fails after a session exists: initiate **fails** to the client; payment stays pending until unpaid timeout.
- Provider notification arrives before initiate finishes: no unauthenticated store; reconciler completes later if paid.
- Duplicate webhook after already completed: no regression of status.
- Store toggle after a sandbox payment exists: refunds and notifications for that payment still use **sandbox** credentials (environment frozen on the payment).
- Abandoned checkout: reconciler; no cancel-payment API in this feature.
- Logo file missing: list/detail still succeed; logo omitted.
- Correlation id still required on authenticated seller/customer payment APIs.
- Two app instances running reconciler: rows are not processed twice in a harmful way (skip locked / conditional status).
- Provider GET rate limit during reconcile: limited retries then wait for next run (do not retry money-moving POSTs).

---

## Requirements *(mandatory)*

### Functional Requirements

**Orchestration / extension**

- **FR-001**: System MUST treat payment providers as interchangeable packs. Core checkout, refund, webhook apply, seller configure, and reconcile MUST NOT branch on a specific provider name or provider event string.
- **FR-002**: System MUST expose one public notification URL pattern keyed only by provider **code**.
- **FR-003**: Adding a provider MUST be limited to: the provider pack, catalog seed (including country/currency links), and a single registration of that pack. No new public URL per provider name.
- **FR-004**: Checkout response MUST be a generic `checkout` object (`gatewayCode` + `fields`). MUST NOT expose a Razorpay-only top-level `keyId`.

**Seller configuration**

- **FR-005**: System MUST store at most one credential set per seller + provider + environment (`sandbox` | `production`).
- **FR-006**: Store MUST have a payments mode (`sandbox` | `production`, default sandbox) on seller business settings. Checkout and new refunds for **new** payments use that mode; existing payments keep the environment they were created with.
- **FR-007**: Configure MUST support partial updates: omitted/empty sensitive fields keep previously stored secrets.
- **FR-008**: Configure and detail MUST never return secrets (plaintext or encrypted). Non-secret identifiers MAY be masked (e.g. key id prefix/suffix).
- **FR-009**: System MUST provide a test-connection action that validates keys with the provider without requiring a live payment. Test MUST NOT persist unsaved secrets unless the seller also saved them.
- **FR-010**: System MUST compose the webhook URL from the public site base + `/api/payment/webhooks/{code}` and return it on provider list/detail. MUST NOT store a redundant webhook URL on the provider catalog row.
- **FR-011**: Deactivate MUST be per environment (query/body environment), not both rows at once unless specified.
- **FR-012**: If the encryption key is missing, saving credentials MUST fail; secrets MUST NOT be stored in plaintext.
- **FR-013**: Provider catalog MUST link supported countries and currencies to the platform country and currency records. Seller config MUST NOT copy a country code field that duplicates business country.

**Checkout and refunds**

- **FR-014**: Initiate MUST NOT trust client amount or currency; use the order total and seller base currency/business country.
- **FR-015**: Initiate MUST select among the seller’s **active** configs for the **current store payments mode**, filtered by country/currency support, ordered by seller **priority** (higher wins).
- **FR-016**: Duplicate initiate while a pending or completed payment exists for that order MUST be rejected. Failed payments MUST NOT block retry.
- **FR-017**: Refunds MUST be allowed on `completed` and `partially_refunded` up to remaining refundable amount (amount minus pending, processing, and completed refunds).
- **FR-018**: Refunds MUST use credentials for the **payment’s** environment.
- **FR-019**: Transaction list MUST filter by status on the server (`pending`, `completed`, `failed`, `refunded`, `partially_refunded`). Invalid status MUST be rejected. Page size MUST be capped at 100.
- **FR-020**: Transaction detail MUST include remaining refundable amount, environment, and the full ordered event history. Transaction **list** MUST NOT include events.

**Notifications and security**

- **FR-021**: System MUST locate the payment from untrusted locator hints, load **that seller’s** credentials for the payment’s environment, then verify the notification. Verification MUST use a constant-time compare on the **raw body**.
- **FR-022**: A notification signed with the wrong tenant’s secret MUST NOT change payment state.
- **FR-023**: Verified notifications MUST be idempotent by provider+event id. Event id MUST always be set (composite fallback if the provider omits a header).
- **FR-024**: Unverified bodies MUST NOT be persisted. Unknown provider code MUST be not found.
- **FR-025**: After a verified notification, the provider MUST receive success so it stops retrying, even if later apply is recorded as failed internally; the reconciler is the backstop.
- **FR-026**: Completed-payment apply MUST refuse amount/currency mismatch vs the stored payment.
- **FR-027**: Public webhook URLs MUST work without a client correlation header; authenticated payment APIs MUST still require it.
- **FR-028**: Provider-specific event names (e.g. `payment.captured`) MUST be mapped inside the provider pack to a frozen set of actions: ignore, authorized, payment completed, payment failed, refund pending, refund completed, refund failed. Core apply MUST switch only on those actions.

**Stuck payments**

- **FR-029**: System MUST periodically reconcile pending payments older than a configurable unpaid timeout (default 45 minutes): ask the provider; complete if paid; fail as expired if still unpaid; fail sooner (default 5 minutes) if no provider session was ever created.
- **FR-030**: System MUST reconcile refunds stuck in pending/processing older than a configurable timeout (default 30 minutes) using the same apply actions as notifications.
- **FR-031**: Reconcile MUST be safe with multiple app instances (no double-complete). Provider GET calls MAY retry twice on throttle/server errors; money-moving POSTs MUST NOT retry blindly.

**Data and development constraints**

- **FR-032**: Payment schema for this feature MUST be applied by **editing existing development migrations and seeds** (not a new additive “drop leftover columns” migration). Payment is not in production; no backward-compatible API aliases.
- **FR-033**: Saved-card (tokenized payment method) storage is **out of scope**; that unused table MUST be removed from the development schema.
- **FR-034**: Payments remain independent of orders (polymorphic owner). Order confirm/fail/attach MUST go through the order module’s public capabilities, not order internals.
- **FR-035**: Seller-scoped webhook log listing MUST only include logs linked to that seller’s payments, paginated.
- **FR-036**: On payment completed, system MUST store provider payment id, fee if present, and payment method details from the notification when available.
- **FR-037**: Logos MUST resolve through the existing file display capability when a logo file id exists.
- **FR-038**: Logs MUST NOT print secrets or full provider error bodies containing credentials.

### Key Entities

- **Payment provider (catalog)**: Platform-level provider (code, name, methods, logo). Linked countries and currencies are platform geo records.
- **Provider field schema**: Per-provider definition of credential fields (name, required, sensitive, validation).
- **Seller provider config**: Encrypted credentials per seller, provider, and environment; active flag; priority.
- **Store payments mode**: Sandbox vs production on seller business settings.
- **Payment**: Current-state money movement (amount, currency, status, provider session/payment ids, frozen environment, owner type/id, user, seller).
- **Payment event**: Append-only history of a payment (type, from/to status, actor, provider payloads).
- **Refund**: Child of a payment; pending/processing/completed/failed; amounts; provider refund id.
- **Notification log**: Verified (or processing) inbound provider events; idempotent by provider+event id; optional link to payment/refund.
- **Country / currency**: Existing platform geo; used for support checks and display.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A customer of a correctly configured seller can start checkout and, after a successful provider capture, see the order confirmed without a second confirm step.
- **SC-002**: Two sellers on the same provider cannot complete each other’s payments using the other seller’s webhook secret (100% isolation in tests).
- **SC-003**: Abandoned checkouts do not remain pending indefinitely; they reach a terminal failed or completed state within the unpaid timeout plus one reconcile interval.
- **SC-004**: Seller dashboard status chips match list totals when filtering by status (no client-only filter of a single page).
- **SC-005**: A seller can keep sandbox and production credentials at once and switch store mode without losing the unused environment’s saved keys.
- **SC-006**: A seller can rotate a public key id without re-entering secrets, and can test keys before relying on checkout.
- **SC-007**: Duplicate provider notifications never double-capture or double-refund (exactly one successful apply per event id).
- **SC-008**: Adding a second provider does not require changing checkout/refund/webhook **orchestration** behavior—only a new pack + seed + register (verified by review checklist / automated grep of provider names in orchestration).
- **SC-009**: Authenticated payment screens still reject missing correlation ids; provider webhooks still succeed without them.
- **SC-010**: Refund of a remainder after a partial refund succeeds when the amount is within remaining refundable balance.

---

## Assumptions

- Razorpay remains the first provider pack; Stripe/Cashfree packs are **not** implemented in this feature, only the extension point.
- Seller dashboard and storefront will adopt the new checkout payload (`checkout.fields`) in the same development cycle; no compatibility fields.
- Local databases will be recreated after schema edits to 009/031/008/seeds.
- Order module already exposes attach transaction id, confirm by transaction id, and fail by transaction id.
- Encryption key is the same platform key used elsewhere (file module).
- Default unpaid timeout 45 minutes and refund stuck timeout 30 minutes are acceptable unless ops changes configuration.
- No cancel-payment API, no subscription billing, no dedicated refund ledger list, no per-seller webhook URL tokens, no admin global unmatched-notification browser.
- `GET /refunds` is not required; refunded/partially_refunded payment statuses plus events suffice.
- Events on payment detail (not a second events URL) are enough because a payment has few events.

---

## Out of Scope

- Implementing a second payment provider.
- Saved cards / tokenized payment methods.
- Cancel-payment customer API.
- Subscriptions (`reference_type` may exist; no subscription flow).
- Mixing sandbox and production on one checkout.
- Smart routing beyond priority + country + currency + store mode.

---

## Dependencies

- Existing customer/seller authentication and correlation-id middleware (webhook skip already in place for `/api/payment/webhooks/`).
- Existing Razorpay integration on branch `009-razorpay-payment-gateway` (this feature **refactors and completes** it; it does not start from empty).
- Platform `country`, `currency`, `seller_settings`, file display, order payment hooks.
- Research documents listed at the top of this spec (binding for `/speckit.plan` and `/speckit.implement`).
