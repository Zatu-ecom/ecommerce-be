# Tasks: Payment Gateway Platform

**Input**: Design documents from `/specs/010-payment-gateway-platform/`  
**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/), [quickstart.md](./quickstart.md), [pre-spec.md](./pre-spec.md)

**Binding (no consistency drift):** Follow spec + pre-spec + research **R-01**. Do **not** add `GET /transactions/:id/events`. Do **not** add `country` / `country_id` on `payment_gateway_config`. Do **not** keep `HandleRazorpay` or top-level `keyId`. Do **not** add migration `032`. Tests live under `test/`, never beside `payment/service`.

**Tests**: Included (constitution TDD + plan/quickstart matrix). Write failing tests first, then production code.

**Organization**: Setup → Foundational (blocks all stories) → US1–US5 → Polish.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Parallelizable (different files, no incomplete dependency)
- **[Story]**: [US1]–[US5] on story phases only

## Path Conventions

Repo-root modular monolith: `payment/`, `user/`, `migrations/`, `test/integration/payment/`, `test/payment/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Config knobs and dead-code removal so later tasks have a clean tree

- [ ] T001 Add `PublicAPIBaseURL` (default `http://localhost:8080`), `PaymentPendingTTLMinutes` (default 45), `PaymentSessionlessTTLMinutes` (default 5), and `RefundStuckTTLMinutes` (default 30) to `common/config/app.go` and document keys in `.env.example` if that file exists
- [ ] T002 [P] Delete unused DTOs in `payment/model/gateway_response.go` (types `CreatePaymentResponse`, `CancelPayment`, unused `PaymentStatusResponse`) and fix any remaining imports
- [ ] T003 [P] Remove Razorpay event/base-URL constants from `payment/utils/constant/payment_gateway_constant.go` that orchestrators import (`RAZORPAY_EVENT_*`, `RAZORPAY_BASE_URL`); they must live only under `payment/service/payment_gateway/razorpay/` after T012

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Final schema, entities, one `PaymentGateway` interface, Razorpay folder, dual-env repos. **No user story work until this phase is complete.**

**⚠️ CRITICAL**: Recreate local DB / Testcontainers after SQL edits (`RunAllMigrations`). Payment is not in production — edit `009`/`031`/`008`/seed `004` in place. No `032`.

- [ ] T004 Rewrite `migrations/009_create_payment_tables.sql` to the target CREATE in `data-model.md`: drop `supported_countries`, `supported_currencies`, `webhook_url`, `logo_url`; add `logo_file_id`; create `payment_gateway_country` and `payment_gateway_currency`; drop `payment_gateway_config.country`; drop entire `payment_method` table; `payment_transaction` includes `reference_*`, `gateway_session_id`, `gateway_payment_id`, `payment_method_details`, `environment`, nullable `gateway_fee_cents`; **no** `metadata`/`initiated_at`/`gateway_transaction_id`; `payment_webhook_log.event_id VARCHAR NOT NULL` + unique `(gateway_id, event_id)` without partial WHERE; index `payment_transaction (status, created_at)`
- [ ] T005 Slim `migrations/031_create_razorpay_payment_integration.sql` so it does not re-ALTER columns 009 already creates; keep or fold `payment_transaction_event` CREATE; unique webhook index without `WHERE event_id IS NOT NULL`
- [ ] T006 Add `payments_environment VARCHAR(20) NOT NULL DEFAULT 'sandbox'` on `seller_settings` in `migrations/008_create_geo_tables.sql`
- [ ] T007 Rewrite `migrations/seeds/core/004_seed_payment_gateways.sql`: Razorpay row without country/currency arrays and without `webhook_url`; insert join rows via `country.code='IN'` and `currency.code='INR'` subselects; keep field seed for `key_id`/`key_secret`/`webhook_secret`/`account_id`
- [ ] T008 [P] Update `user/entity/seller_settings.go`, `user/model/seller_settings_model.go`, `user/factory/seller_response_builder.go`, and `user/service/seller_settings_service.go` so GET/PUT `/api/user/seller/settings` read/write `paymentsEnvironment` (`sandbox`|`production`)
- [ ] T009 [P] Align `payment/entity/payment_gateway.go` (no arrays for countries/currencies, no `webhook_url`) and add `payment/entity/payment_gateway_country.go` plus `payment/entity/payment_gateway_currency.go`
- [ ] T010 [P] Remove `country` from `payment/entity/payment_gateway_config.go`; keep `UNIQUE(seller_id, gateway_id, environment)`
- [ ] T011 [P] Add `Environment` to `payment/entity/payment_transaction.go`; delete `payment/entity/payment_method.go`; set `payment/entity/payment_webhook_log.go` `EventID` as required (non-empty)
- [ ] T012 Create `payment/service/payment_gateway/contract.go` with the **exact** `PaymentGateway` interface, `Locators`, `WebhookAction` constants, and `NormalizedWebhook` from `contracts/gateway-adapter.md`; delete `RefundType`, `VerifyWebhook`, `ParseWebhook`, and `FetchPayment` from the old `payment/service/payment_gateway/payment_gateway.go` (replace that file)
- [ ] T013 Create `payment/service/payment_gateway/crypto.go` with `ResolveEncryptionKey()` only (move off `razorpay_credentials.go`)
- [ ] T014 Move Razorpay into `payment/service/payment_gateway/razorpay/` (`adapter.go`, `credentials.go`, `http.go`, `webhook.go`): implement full interface including `Validate`/`Encrypt`/`Decrypt`/`MaskHints`/`MergePartial`/`PeekLocators`/`NormalizeWebhook`/`FetchRemoteStatus`/`TestConnection`; HMAC on raw body; Razorpay event map only in this folder; GET retries 2× on 429/5xx; POST no retry; redact logs
- [ ] T015 Delete `payment/service/payment_gateway/razorpay_gateway.go` and `payment/service/payment_gateway/razorpay_credentials.go`; move `RazorpayCredentials` out of `payment/model/gateway_models.go`
- [ ] T016 Wire `razorpay.New(os.Getenv("RAZORPAY_BASE_URL"))` in `payment/factory/singleton/service_factory.go` as the **only** Razorpay import in payment factories; keep `payment/factory/payment_gateway_factory.go` registry by code
- [ ] T017 Replace environment-less `FindBySellerAndGateway` usage: add `FindBySellerGatewayAndEnvironment` and `FindAllBySellerAndGateway` in `payment/repository/payment_gateway_config_repository.go` (and clauses file); stop `First()` without environment
- [ ] T018 Add gateway geo membership queries (`EXISTS` country/currency) in `payment/repository/payment_gateway_repository.go` (or new `payment/repository/payment_gateway_geo_repository.go` wired in `payment/factory/singleton/repository_factory.go`)
- [ ] T019 Add `ListPendingForReconcile` / `ListStuckRefunds` using `FOR UPDATE SKIP LOCKED LIMIT 100` in `payment/repository/payment_transaction_repository.go` and `payment/repository/payment_refund_repository.go`
- [ ] T020 Extract shared `applyNormalized` into `payment/service/apply_normalized.go` (or webhook service private func used by both webhook + future cron): switch **only** on `WebhookAction`; amount/currency mismatch does not complete; conditional status updates; order confirm/fail via `order/service` only
- [ ] T021 Add `ENCRYPTION_KEY` fail-closed path (no plaintext secrets) in configure/decrypt using `payment/error/` AppErrors; add `GATEWAY_UNSUPPORTED_COUNTRY` in `payment/utils/constant/error_code.go` and `payment/error/payment_gateway_error.go` if missing
- [ ] T022 Write failing unit tests for Razorpay HMAC, event map, mask, merge-partial in `test/payment/payment_gateway/razorpay/` (`package razorpay_test`)
- [ ] T023 Write failing unit tests for `applyNormalized` with a fake adapter in `test/payment/apply_normalized_test.go`

**Checkpoint**: Schema + adapter compile; `rg` of `RAZORPAY_EVENT` in `payment/service/*.go` should already be moving toward zero. User stories can start.

---

## Phase 3: User Story 1 - Customer pays a pending order (Priority: P1) 🎯 MVP

**Goal**: Initiate uses store mode + geo + priority (not hardcoded Razorpay); checkout is `checkout.gatewayCode` + `checkout.fields`; capture/fail/authorized via normalized apply; abandoned pending expires via reconciler.

**Independent Test**: Sandbox seller; initiate; fake capture → order confirmed; abandon → after TTL reconciler fails `EXPIRED_UNPAID`; duplicate pending/completed initiate 409; failed prior does not block retry; production mode without prod keys does not use sandbox.

### Tests for User Story 1

> Write these FIRST; they MUST fail before implementation

- [ ] T024 [P] [US1] Update `test/integration/payment/initiate_payment_test.go` so success JSON has `checkout.fields.keyId` / `orderId` and **no** top-level `keyId`; add cases: production mode without production config → `GATEWAY_NOT_CONFIGURED`; optional `paymentMethodType` invalid → 400
- [ ] T025 [P] [US1] Extend `test/integration/payment/webhook_payment_test.go` for authorized (stay pending), failed payment, amount mismatch does not complete (may overlap US3 — keep one test here if webhook apply already shared)
- [ ] T026 [P] [US1] Add `test/integration/payment/reconcile_payment_test.go`: backdated pending with session still open → `EXPIRED_UNPAID` + order failed; pending with fake paid remote status → completed; sessionless pending older than 5 minutes → failed

### Implementation for User Story 1

- [ ] T027 [US1] Add `Checkout` (`gatewayCode` + `fields`) to `payment/model/gateway_models.go` `InitiatePaymentOutput` and replace `KeyID` with `Checkout` on `payment/model/payment_response.go` `InitiatePaymentResponse`
- [ ] T028 [US1] Rewrite initiate in `payment/service/payment_service.go`: no `FindByCode(GATEWAY_CODE_RAZORPAY)`; select active configs for `seller_settings.payments_environment` + geo `EXISTS` + `priority DESC`; decrypt via adapter; persist `environment` on txn; on adapter error mark txn **failed**; if `AttachTransactionID` fails return error (txn stays pending)
- [ ] T029 [US1] Map handler response in `payment/handler/payment_handler.go` only (no extra business logic)
- [ ] T030 [US1] Implement `payment/service/reconcile_service.go`: load adapter by txn gateway; decrypt **txn.environment** config; `FetchRemoteStatus`; call `applyNormalized` source=`system`; TTL fail `EXPIRED_UNPAID`; sessionless shorter TTL; stuck refunds; skip if previous run active
- [ ] T031 [US1] Register cron in `payment/container.go` via `common/cron.RegisterIntervalJob(5*time.Minute, "payment.reconcile_pending", …)` (ensure `cron.Init()` already runs in `main.go`)
- [ ] T032 [US1] Run `test/integration/payment/initiate_payment_test.go` and `test/integration/payment/reconcile_payment_test.go` until green

**Checkpoint**: Customer checkout + capture + expiry work without Razorpay named fields on the HTTP initiate contract.

---

## Phase 4: User Story 2 - Seller configures sandbox and live independently (Priority: P1)

**Goal**: Dual-env list/detail/configure/deactivate, masked hints, composed webhook URL, store toggle, test connection without persisting unsaved secrets.

**Independent Test**: Configure sandbox; see hints + webhook URL; test connection ok; switch store to production without prod keys (checkout blocked); configure production; deactivate sandbox only.

### Tests for User Story 2

- [ ] T033 [P] [US2] Expand `test/integration/payment/gateway_dashboard_test.go`: dual-row configure; list `configuredSandbox`/`configuredProduction`/`paymentsEnvironment`/`webhookUrl`; detail `configs.sandbox`/`production` + `configHints`; partial update keeps secrets; deactivate one env; missing encryption key fails closed if testable
- [ ] T034 [P] [US2] Add test-connection cases in `test/integration/payment/gateway_dashboard_test.go` or `test/integration/payment/gateway_test_connection_test.go`: httptest 200 → ok; 401 → 400; omitted credentials uses saved row; unsaved credentials not persisted

### Implementation for User Story 2

- [ ] T035 [US2] Extend DTOs in `payment/model/payment_response.go` and `payment/model/payment_request.go`: geo objects `{id,code,name}` / currency `{id,code,symbol,decimalDigits}`; `configs` map; `TestGatewayRequest`; require `environment` on configure
- [ ] T036 [US2] Implement list/detail/configure/deactivate in `payment/service/payment_gateway_service.go` using adapter codec methods only (no `ParseRazorpayCredentials`, no `country := "IN"`); compose webhook URL from `PublicAPIBaseURL` + `/api/payment/webhooks/{code}`; resolve logo via existing `FileDisplayGateway`
- [ ] T037 [US2] Add `POST /:code/test` in `payment/route/gateway_route.go` and `payment/handler/gateway_handler.go`; handler binds JSON and calls service `TestConnection` (no persist)
- [ ] T038 [US2] DELETE configure requires query `environment` in `payment/handler/gateway_handler.go` (deactivate that row only)
- [ ] T039 [US2] Run gateway dashboard + test-connection integration tests until green

**Checkpoint**: Seller can save both environments, mask secrets, test keys, copy webhook URL.

---

## Phase 5: User Story 3 - Provider notifications are tenant-safe and provider-agnostic (Priority: P1)

**Goal**: `POST /api/payment/webhooks/:code`; locate txn then HMAC with **that** seller’s secret; no unverified persist; unknown code 404; duplicates no-op; core service has no Razorpay event strings.

**Independent Test**: Two sellers two secrets; A’s payment cannot complete with B’s signature; replay event id no-op; missing correlation header still 200; unknown code 404.

### Tests for User Story 3

- [ ] T040 [P] [US3] Extend `test/integration/payment/webhook_payment_test.go`: `TestWebhookWorksWithoutCorrelationID` still 200; unknown code 404; wrong-seller signature 401; two-seller isolation; duplicate event id; no body stored when txn not found (assert via seller webhook-logs empty or DB helper only if unavoidable — prefer API-first after US4 logs exist; until then assert payment status unchanged)
- [ ] T041 [P] [US3] Confirm skip rule still `POST` prefix `/api/payment/webhooks/` in `common/middleware/skip_rules.go` (update tests in `test/common/middleware/` if path changes)

### Implementation for User Story 3

- [ ] T042 [US3] Change `payment/route/webhook_route.go` to `POST /:code` and rename handler to `HandleWebhook` in `payment/handler/webhook_handler.go` (read raw body with `io.ReadAll`; pass `http.Header`; **no** Razorpay signature header parsing in the handler)
- [ ] T043 [US3] Rewrite `payment/service/webhook_service.go`: PeekLocators → find txn → load config `seller_id` + gateway + **`txn.environment`** → adapter `Decrypt` + `NormalizeWebhook`; never `FindAllForGateway()[0]`; never switch on `payment.captured`; no persist if txn missing; 401 on HMAC fail; after verify always HTTP 200; apply errors set `webhook_log.status=failed`
- [ ] T044 [US3] Remove Razorpay-specific strings from `payment/handler/webhook_error.go` and webhook constants in `payment/utils/constant/`
- [ ] T045 [US3] Run webhook integration tests until green; `rg "HandleRazorpay|payment.captured" payment/service/webhook_service.go payment/handler payment/route` must be empty

**Checkpoint**: Multi-tenant webhook security + generic `:code` route.

---

## Phase 6: User Story 4 - Seller manages payments, refunds, and history (Priority: P2)

**Goal**: Server-side `?status=`; detail has `refundableAmountCents`, `environment`, **events**; second partial refund; seller GET-by-id; paginated seller `GET /webhook-logs`. No dedicated `/events` route.

**Independent Test**: Filter failed/refunded with correct `total`; detail shows events + remaining; two partials until remaining 0; other seller empty logs.

### Tests for User Story 4

- [ ] T046 [P] [US4] Extend `test/integration/payment/refund_test.go`: refund on `partially_refunded` allowed; over remaining rejected; pending rejected
- [ ] T047 [P] [US4] Add list filter tests in `test/integration/payment/initiate_payment_test.go` or `test/integration/payment/transaction_list_test.go`: `status=` each enum; unknown status 400; `pageSize` cap; list payload has **no** `events`
- [ ] T048 [P] [US4] Add GET-by-id as seller in `test/integration/payment/` : events present; `refundableAmountCents`; other seller 404
- [ ] T049 [P] [US4] Add `test/integration/payment/webhook_logs_test.go`: captured webhook appears for owner seller; other seller empty; unmatched (no txn) not listed

### Implementation for User Story 4

- [ ] T050 [US4] Pass `status` into `FindBySellerID` in `payment/repository/payment_transaction_repository.go`; validate enum in `payment/service/payment_service.go`
- [ ] T051 [US4] Compute `refundableAmountCents` and load events (`ListByTransactionID` already in `payment/repository/payment_transaction_event_repository.go`) only in GET-by-id mapper in `payment/model/payment_response.go` / `payment/service/payment_service.go`
- [ ] T052 [US4] Allow refund when status is `completed` **or** `partially_refunded` in `payment/service/payment_service.go`; load credentials by **txn.environment**; drop `RefundType` argument to adapter
- [ ] T053 [US4] Register `GET /transactions/:transactionId` for **customer or seller** in `payment/route/payment_route.go` (combined auth); keep list seller-only; ownership check in `payment/service/payment_service.go`
- [ ] T054 [US4] Add paginated `FindBySellerID` join in `payment/repository/payment_webhook_log_repository.go`; `GET /webhook-logs` in `payment/route/payment_route.go` + `payment/handler/payment_handler.go` (pageSize max 100)
- [ ] T055 [US4] Run refund, list, detail, webhook-log integration tests until green

**Checkpoint**: Dashboard chips, refunds, events, logs work. Still **no** `/events` URL.

---

## Phase 7: User Story 5 - Platform can add another provider without rewriting checkout (Priority: P2)

**Goal**: Extension point only (no Stripe adapter). Orchestrators have no provider switches; catalog geo is FK joins; factory is the only growth line.

**Independent Test**: Razorpay still works. Review/grep: no Razorpay event names or encrypt helpers in `payment/service/*.go` except factory singleton. Seed India+INR via country/currency rows.

### Tests for User Story 5

- [ ] T056 [P] [US5] Add a grep/CI-oriented test or documented assert in `test/payment/ocp_boundaries_test.go` that fails if `payment/service/*.go` (excluding comments if needed) contain `RAZORPAY_EVENT`, `ParseRazorpay`, `DecryptSensitive`, `GATEWAY_CODE_RAZORPAY` (factory file may be excluded like pre-spec)

### Implementation for User Story 5

- [ ] T057 [US5] Confirm initiate geo uses join tables only in `payment/service/payment_service.go` (no string `IN`/`INR` compares)
- [ ] T058 [US5] Confirm webhook/configure/refund/reconcile have **zero** `switch code` / Razorpay event names; fix any leftover in `payment/service/payment_gateway_service.go`, `payment/handler/*.go`, `payment/route/*.go`
- [ ] T059 [US5] Add a short comment block on `payment/factory/singleton/service_factory.go` documenting: new provider = folder + seed + one `NewPaymentGatewayFactory` argument
- [ ] T060 [US5] Run full `test/integration/payment/` suite and `test/payment/` unit tests

**Checkpoint**: Adding Stripe later does not require editing orchestrators.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Docs, leftover FE handoff, security/logging, quickstart validation

- [ ] T061 [P] Update `plans/payment-backend-handoff.md` “current API surface” to match `contracts/payment-api.md` (no `keyId`, dual env, webhook `:code`)
- [ ] T062 [P] Add a pointer at top of `specs/009-razorpay-payment-gateway/contracts/payment-api.md` that 010 supersedes checkout/webhook contracts (do not leave two conflicting sources of truth without a banner)
- [ ] T063 Redact provider error bodies in `payment/service/payment_gateway/razorpay/http.go`; confirm logs never print secrets
- [ ] T064 Verify `common/middleware/skip_rules.go` still skips only `/api/payment/webhooks/` and authenticated payment APIs still require correlation id (`test/integration/payment/initiate_payment_test.go`)
- [ ] T065 Run the forbidden `rg` from `quickstart.md` and fix remaining hits
- [ ] T066 Walk `quickstart.md` §8–§9 as a review checklist (schema recreate, initiate JSON, webhook without correlation id, seller detail events)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Start immediately
- **Foundational (Phase 2)**: Depends on Setup — **BLOCKS** all user stories
- **US1 (Phase 3)**: Depends on Foundational; needs adapter + `applyNormalized` + txn `environment`
- **US2 (Phase 4)**: Depends on Foundational; can proceed in parallel with US1 after T016–T017 if staffing allows (same `payment_gateway_service.go` as later polish — sequential preferred for one agent)
- **US3 (Phase 5)**: Depends on Foundational + US1 apply path; rewrite `webhook_service.go` after T020
- **US4 (Phase 6)**: Depends on US1 payments existing; webhook-logs tests depend on US3 pipeline
- **US5 (Phase 7)**: Depends on US1–US4 code existing to grep
- **Polish (Phase 8)**: After desired stories

### User Story Dependencies

- **US1**: After Phase 2 only (MVP checkout + reconcile)
- **US2**: After Phase 2; uses same config rows US1 initiate reads
- **US3**: After Phase 2; should land before treating webhooks as production-safe
- **US4**: After US1 (and US3 for log rows)
- **US5**: After orchestrators rewritten (US1–US3)

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Models/DTOs before services
- Services before routes/handlers
- Story green before next priority unless parallel staffing

### Parallel Opportunities

- T002 and T003
- T008–T011 entity files
- T022 and T023 unit tests
- T024–T026 US1 tests
- T033–T034 US2 tests
- T040–T041 US3 tests
- T046–T049 US4 tests
- T061–T062 docs

Same-file clusters (**do not** parallelize): `payment_service.go`, `webhook_service.go`, `payment_gateway_service.go`, `009` SQL.

---

## Parallel Example: User Story 1

```bash
# After Phase 2, launch US1 tests together:
Task: "T024 initiate_payment_test.go checkout.fields / no keyId"
Task: "T025 webhook authorized/fail/mismatch"
Task: "T026 reconcile_payment_test.go"

# Then sequential implementation T027 → T031, then T032 green.
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1 Setup
2. Phase 2 Foundational (schema + one interface + Razorpay folder)
3. Phase 3 US1 (initiate + apply + cron)
4. **STOP**: customer can pay and abandoned checkouts expire
5. Then US3 (security) before exposing webhooks as multi-tenant-safe — **do not ship US1 webhooks to multiple sellers without US3**

Practical MVP for this codebase: **Phase 2 + US1 + US3** (checkout + tenant-safe webhook). US2 unblocks the seller dashboard in parallel next.

### Incremental Delivery

1. Setup + Foundational → recreate DB
2. US1 → checkout + reconcile
3. US3 → locate-then-verify (hard security stop)
4. US2 → dashboard configure/test
5. US4 → list/refunds/events/logs
6. US5 + Polish → OCP grep + docs

### Parallel Team Strategy

After Phase 2:

- Dev A: US1 (`payment_service.go`, reconcile)
- Dev B: US2 (`payment_gateway_service.go`, settings) — avoid touching `payment_service.go` select until US1 lands
- Dev C: US3 (`webhook_service.go`) — depends on T020 `applyNormalized`

---

## Notes

- Task IDs T001–T066; every line is `- [ ]`, has ID, file path, and story label on US tasks
- [P] = different files only
- Independent tests are copied from spec.md
- Flash-model implementers: copy signatures from `contracts/gateway-adapter.md` and JSON from `contracts/payment-api.md`; algorithms from `pre-spec.md` §9
- Commit after each task or logical group if the user asks for commits
- Stop at checkpoints to validate the story

## Task count summary

| Phase | IDs | Count |
|-------|-----|-------|
| Setup | T001–T003 | 3 |
| Foundational | T004–T023 | 20 |
| US1 Customer pay | T024–T032 | 9 |
| US2 Seller configure | T033–T039 | 7 |
| US3 Webhooks | T040–T045 | 6 |
| US4 Dashboard ledger | T046–T055 | 10 |
| US5 Extension point | T056–T060 | 5 |
| Polish | T061–T066 | 6 |
| **Total** | T001–T066 | **66** |
