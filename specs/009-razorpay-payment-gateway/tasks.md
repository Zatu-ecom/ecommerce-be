# Tasks: Razorpay Payment Gateway Integration

**Input**: Design documents from `/specs/009-razorpay-payment-gateway/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Integration tests are REQUIRED for this feature per the project constitution
(TDD / integration-first) and the `pre-spec.md` testing section. Tests are written first within each
user-story phase and must FAIL before the corresponding implementation exists.

**Organization**: Tasks are grouped by user story so each story is independently implementable and
testable. The canonical technical detail source is `pre-spec.md`; `data-model.md` and `contracts/`
derive from it and must stay in sync.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)

## Path Conventions

- Payment module: `payment/` (mirrors `product/`/`order/` layout)
- Order module cross-module hooks: `order/`
- Migrations/seeds: `migrations/`
- Integration tests: `test/integration/payment/`

---

## Consistency Notes (resolving cross-doc ambiguity)

The following are intentionally aligned across `spec.md`, `pre-spec.md`, `data-model.md`, and
`contracts/payment-api.md`. Any deviation from these in an implementation task is a defect:

- **`payment_transaction` does NOT have `order_id`, `metadata`, `initiated_at`,
  `gateway_transaction_id`, `gateway_request`, or `gateway_response`.** The order link is
  `reference_type` + `reference_id` (polymorphic, no FK) plus the existing
  `order.transaction_id = payment_transaction.transaction_id`. Request/response payloads live only on
  `payment_transaction_event`.
- **Two distinct provider references**: `gateway_session_id` (checkout/order/intent) and
  `gateway_payment_id` (captured payment/charge). Never reuse the removed `gateway_transaction_id`.
- **Currency/amount** are derived server-side from the seller's base currency and the order's
  `total_cents`; the client never supplies amount or currency.
- **`payment_transaction_event` has no `updated_at`** (append-only). Its entity defines its own `ID`
  + `CreatedAt` and does **not** embed `db.BaseEntity` (mirror `payment_webhook_log`).
- **Status enums** (data-model.md is authoritative):
  - Payment: `pending → completed`, `pending → failed`, `completed → refunded`,
    `completed → partially_refunded`.
  - Refund: `refund.created → pending`, `refund.processed → completed`, `refund.failed → failed`;
    `processing` is a transitional state set after the outbound refund request before the terminal
    webhook arrives.
  - Webhook log: `received / processed / failed / ignored`.
- **Order confirmation happens only on `payment.captured`.** `payment.authorized` is recorded as an
  observation event and MUST NOT confirm the order (authorization precedes capture).
- **Endpoints** (contracts/payment-api.md is authoritative):
  - `POST /api/payment/initiate`
  - `GET /api/payment/transactions/:transactionId`
  - `GET /api/payment/transactions`
  - `POST /api/payment/refunds`
  - `GET /api/payment/gateways`
  - `GET /api/payment/gateways/:code`
  - `PUT /api/payment/gateways/:code/configure`
  - `DELETE /api/payment/gateways/:code/configure`
  - `POST /api/payment/webhooks/razorpay`
- **Razorpay specifics** (verified in `pre-spec.md`): `InitiatePayment` maps to `POST /orders` with
  `receipt` = our `transaction_id` (idempotency key); refunds to `POST /payments/{id}/refund`;
  webhook signature is `HMAC-SHA256(raw body, webhook_secret)` hex-encoded, verified over the raw
  body; handled events are `payment.authorized`, `payment.captured`, `payment.failed`,
  `refund.created`, `refund.processed`, `refund.failed`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Database schema, seed catalog, and entity definitions shared by all stories.

- [X] T001 Create migration `migrations/031_create_razorpay_payment_integration.sql` reshaping `payment_transaction` (drop `metadata`, `initiated_at`, `gateway_transaction_id`; add `reference_type`, `reference_id`, `gateway_session_id`, `gateway_payment_id`, `payment_method_details`; make `gateway_fee_cents` nullable; drop `idx_payment_transaction_gateway_transaction_id`; add reference/session/payment indexes), replacing `payment_gateway.logo_url` with `logo_file_id`, creating `payment_transaction_event`, and adding the unique partial webhook index `uq_payment_webhook_log_gateway_event` — exact SQL per pre-spec.md
- [X] T002 Create core seed `migrations/seeds/core/004_seed_payment_gateways.sql` (idempotent upserts for the `razorpay` `payment_gateway` row with `IN`/`INR`/payment-methods and `payment_gateway_field` rows: `key_id` non-sensitive, `key_secret` sensitive, `webhook_secret` sensitive, `account_id` optional)
- [X] T003 [P] Modify `payment/entity/payment_gateway.go` to replace `LogoURL string` with `LogoFileID *string` (`column:logo_file_id;size:80`)
- [X] T004 [P] Modify `payment/entity/payment_transaction.go` to remove `Metadata`, `GatewayTransactionID`, `InitiatedAt`; add `ReferenceType`, `ReferenceID`, `GatewaySessionID`, `GatewayPaymentID`, `PaymentMethodDetails db.JSONMap`; make `GatewayFeeCents` a `*int64`; add `ReferenceType` constants (`order`, `subscription`)
- [X] T005 [P] Create `payment/entity/payment_transaction_event.go` with `PaymentTransactionEvent` (own `ID` + `CreatedAt` only, **no** `UpdatedAt` — mirror `payment_webhook_log`, do NOT embed `db.BaseEntity`; `TableName() = "payment_transaction_event"`, columns per data-model.md)
- [X] T006 [P] Extend `payment/error/payment_gateway_error.go` with sentinels: `ErrorPaymentTransactionNotFound`, `ErrorPaymentOrderNotFound`, `ErrorInvalidWebhookSignature`, `ErrorDuplicateWebhook`, `ErrorRefundNotAllowed`, `ErrorGatewayNotConfigured`, `ErrorGatewayUnsupportedCurrency`; add matching codes/messages in `payment/utils/constant/error_code.go` and `payment/utils/constant/payment_gateway_constant.go`, plus gateway/status/event/reference-type constants

**Checkpoint**: Schema + entities + errors defined; `go build ./payment/entity/...` compiles.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Data access, gateway abstraction, DTOs, cross-module hooks, and the DI singleton
scaffold that every story depends on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T007 [P] Extend `payment/repository/payment_gateway_repository.go` with `FindByCode(ctx, code)` and `FindAllActive(ctx)`
- [X] T008 [P] Create `payment/repository/payment_gateway_field_repository.go` (interface + impl) with `FindByGatewayID(ctx, gatewayID)` ordered by `display_order`
- [X] T009 [P] Create `payment/repository/payment_gateway_config_repository.go` (interface + impl) with `FindBySellerAndGateway`, `FindActiveBySeller`, `UpsertConfig`, `DeactivateConfig`
- [X] T010 [P] Create `payment/repository/payment_transaction_repository.go` (interface + impl) with `Create`, `FindByTransactionID`, `FindByGatewaySessionID`, `FindByGatewayPaymentID`, `UpdateStatusIfCurrent` (conditional UPDATE returning rowsAffected), `FindByID`, `FindBySellerID` (paginated)
- [X] T011 [P] Create `payment/repository/payment_refund_repository.go` (interface + impl) with `Create`, `FindByRefundID`, `UpdateStatusIfCurrent`
- [X] T012 [P] Create `payment/repository/payment_webhook_log_repository.go` (interface + impl) with `Create`, `MarkProcessed`, `MarkIgnored`
- [X] T013 [P] Create `payment/repository/payment_transaction_event_repository.go` (interface + impl) with `Create` (append-only)
- [X] T014 [P] Create `payment/model/gateway_models.go` with gateway-agnostic DTOs: `InitiatePaymentInput/Output`, `RefundInput/Output`, `PaymentStatusOutput`, `WebhookEvent`, `RazorpayCredentials`
- [X] T015 [P] Create `payment/model/payment_request.go` with `InitiatePaymentRequest`, `RefundRequest`, `ConfigureGatewayRequest`
- [X] T016 [P] Create `payment/model/payment_response.go` with `PaymentTransactionResponse`, `InitiatePaymentResponse`, `RefundResponse`, `GatewaySummaryResponse`, `GatewayDetailResponse`
- [X] T017 Create `payment/service/payment_gateway/razorpay_credentials.go` with `ParseRazorpayCredentials`, `EncryptSensitive`, `DecryptSensitive` (reuse `common/helper` + `ResolveEncryptionKey` pattern)
- [X] T018 Replace `payment/service/payment_gateway/payment_gateway.go` with the gateway-agnostic `PaymentGateway` interface (`Code`, `InitiatePayment`, `FetchPayment`, `Refund`, `VerifyWebhook`, `ParseWebhook`)
- [X] T019 Delete `payment/service/payment_gateway/cashfree_gateway.go`
- [X] T020 Create `payment/service/payment_gateway/razorpay_gateway.go` implementing `PaymentGateway` (stdlib `net/http`, basic auth, configurable base URL, HMAC-SHA256 hex constant-time webhook verification, `ParseWebhook` → gateway-agnostic `WebhookEvent`) — depends on T014, T017, T018
- [X] T021 Fix `payment/factory/payment_gateway_factory.go` to inject `PaymentGatewayRepository`, seed registry `map[code]PaymentGateway` with the Razorpay adapter, resolve by `payment_gateway.code` (unknown → `ErrorPaymentGatewayNotSupported`) — depends on T007, T020
- [X] T022 Extend `order/service/order_service.go` interface + impl with `AttachTransactionID`, `ConfirmPaymentByTransactionID` (pending→confirmed + `paid_at`), `FailPaymentByTransactionID` (pending→failed), reusing `applyUpdateOrderStatusTx`/`utils.IsValidTransition`; add `FindOrderByTransactionID` to `order/repository/order_repository.go`
- [X] T023 Add `paymentSingleton.ResetInstance()` to `test/integration/setup/singletons.go`
- [X] T024 Create `payment/factory/singleton/` scaffold: `singleton_factory.go` (`GetInstance`/`ResetInstance`), `repository_factory.go` (wires all Phase 2 repositories), and `service_factory.go` wiring the gateway factory/adapters + `orderSingleton.GetInstance().GetOrderService()`, `userSingleton.GetInstance().GetUserService()`, `fileSingleton` (or its `FileDisplayGateway` adapter) — no handlers/services from later phases yet

**Checkpoint**: Foundation ready — repositories, gateway abstraction, DTOs, order hooks, and the DI
scaffold exist. User story implementation can begin.

---

## Phase 3: User Story 1 - Customer pays for an order online (Priority: P1) 🎯 MVP

**Goal**: Initiate a payment for a pending order, expose a checkout session, and on the provider
webhook mark the payment completed/failed and auto-confirm/fail the order.

**Independent Test**: A customer can initiate payment for their own pending order, receive a payment
session, and a signed provider webhook transitions the payment and order correctly without manual action.

### Tests for User Story 1 (write FIRST, ensure they FAIL) ⚠️

- [X] T025 [P] [US1] Create `test/integration/payment/setup_payment_suite_test.go` (testify suite + Testcontainers; fake Razorpay `httptest.Server`; `RAZORPAY_BASE_URL` override; seed seller gateway config + base currency/country)
- [X] T026 [P] [US1] Create `test/integration/payment/initiate_payment_test.go` covering happy path (transaction created with `reference_type=order`, order `transaction_id` written, returns gatewaySessionId/keyId) and failures (order not owned/not payable, no active provider, unsupported currency/country, duplicate payment for order)
- [X] T027 [P] [US1] Create `test/integration/payment/webhook_payment_test.go` covering `payment.captured` → payment completed + order confirmed, `payment.authorized` → observation event only (no order confirm), `payment.failed` → payment failed + order failed, invalid signature → 401, duplicate webhook idempotency

### Implementation for User Story 1

- [X] T028 [US1] Implement `payment/service/payment_service.go` interface + impl (depends on T010, T021, T022): `InitiatePayment` (load order via order service → resolve seller currency/country → select+decrypt gateway config → validate support → create `payment_transaction` → call adapter `InitiatePayment` → persist `gateway_session_id` + `gateway_session_created` event → write `order.transaction_id`), `GetPaymentStatus`, `ListSellerTransactions`
- [X] T029 [US1] Implement `payment/service/webhook_service.go` (depends on T010, T011, T012, T013, T020, T022, T028): verify signature → parse → insert `payment_webhook_log` (unique index dedupe → ignored) → `payment.captured` conditional complete + confirm order; `payment.authorized` append observation event only; `payment.failed` conditional fail + fail order; conditional updates inside one `db.WithTransaction`
- [X] T030 [US1] Implement `payment/handler/payment_handler.go` (`InitiatePayment`, `GetPaymentStatus`, `ListSellerTransactions`) embedding `common/handler.BaseHandler`
- [X] T031 [US1] Implement `payment/handler/webhook_handler.go` (read raw body, extract `X-Razorpay-Signature`, call `WebhookService.HandleWebhook`, return 200 verified / 401 invalid)
- [X] T032 [US1] Implement `payment/route/payment_route.go` and `payment/route/webhook_route.go` (customer/seller auth on payment routes; no auth on webhook route)
- [X] T033 [US1] Extend `payment/factory/singleton/` service + handler factories to wire `paymentService`/`webhookService` and `paymentHandler`/`webhookHandler`; register `PaymentModule` and `WebhookModule` in `payment/container.go`

**Checkpoint**: User Story 1 fully functional and testable independently (MVP).

---

## Phase 4: User Story 2 - Seller activates and manages a payment provider (Priority: P2)

**Goal**: Sellers list available providers, view each provider's required fields, and save/update/deactivate their own credentials.

**Independent Test**: A seller can list providers (configured vs not), save credentials for one, see it reflected as configured, and deactivate it.

### Tests for User Story 2 (write FIRST, ensure they FAIL) ⚠️

- [X] T034 [P] [US2] Create `test/integration/payment/gateway_dashboard_test.go` covering list (configured vs not, resolved logo), detail (fields + validation rules), configure (valid/invalid credentials), deactivate, and seller isolation

### Implementation for User Story 2

- [X] T035 [US2] Implement `payment/service/payment_gateway_service.go` (depends on T007, T008, T009, T017): `ListForSeller`, `GetByCode`, `Configure` (validate required fields, encrypt secrets), `Deactivate`; resolve logos via `FileDisplayGateway`
- [X] T036 [US2] Implement `payment/handler/gateway_handler.go` (`ListGateways`, `GetGateway`, `ConfigureGateway`, `DeactivateGateway`)
- [X] T037 [US2] Implement `payment/route/gateway_route.go`; extend `payment/factory/singleton/` service + handler factories to wire `paymentGatewayService` and `gatewayHandler`; register `GatewayModule` in `payment/container.go`

**Checkpoint**: User Stories 1 AND 2 both work independently.

---

## Phase 5: User Story 3 - Seller refunds a customer (Priority: P3)

**Goal**: Sellers issue full or partial refunds for completed payments; provider refund webhooks update refund and payment state.

**Independent Test**: A seller can request a full/partial refund for a completed payment and observe it reach a final state via provider notifications.

### Tests for User Story 3 (write FIRST, ensure they FAIL) ⚠️

- [X] T038 [P] [US3] Create `test/integration/payment/refund_test.go` covering full/partial refund happy path, refund-not-allowed for non-completed payment, `refund.processed` → refund completed + payment refunded/partially_refunded, `refund.failed` → refund failed + payment unchanged, and seller isolation

### Implementation for User Story 3

- [X] T039 [US3] Implement `InitiateRefund` in `payment/service/payment_service.go` (validate completed status + refundable amount, call adapter `Refund`, create `payment_refund` in `processing` + event)
- [X] T040 [US3] Extend `payment/service/webhook_service.go` to handle `refund.created`/`refund.processed`/`refund.failed` (create/update `payment_refund`, adjust transaction refund state via conditional updates)
- [X] T041 [US3] Implement refund endpoint wiring in `payment/handler/payment_handler.go` + `payment/route/payment_route.go` (seller auth)

**Checkpoint**: All user stories independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Verification and cleanup across all stories.

- [X] T042 Run `gofmt`/`gofumpt` and `go vet ./...`; resolve all findings
- [X] T043 Run `go build ./...` (clean build gate) and fix any compilation errors
- [X] T044 Apply migrations + seeds against local Postgres and confirm the reshaped `payment_transaction`, new `payment_transaction_event` + indexes, and Razorpay catalog rows per `quickstart.md`
- [X] T045 Run `go test ./test/integration/payment/... -v` and make all tests green
- [X] T046 Run `go test ./test/integration/... -v` (or `make test`) to confirm no regressions in order/user/file suites
- [X] T047 Validate the flow end-to-end against `quickstart.md` (initiate → webhook → order confirm/fail → refund)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Phase 1 — BLOCKS all user stories.
- **User Stories (Phase 3–5)**: All depend on Phase 2. US3 extends `payment_service.go` and
  `webhook_service.go` created in US1, so US3 depends on US1; US2 is independent of US1/US3.
- **Polish (Phase 6)**: Depends on all desired user stories.

### User Story Dependencies

- **US1 (P1)**: After Phase 2 — no other story dependency. (MVP)
- **US2 (P2)**: After Phase 2 — independent of US1 (the singleton scaffold is created in
  Foundational T024, so US2 only adds its own service/handler/route).
- **US3 (P3)**: After Phase 2 and US1 (extends its services/webhook handler).

### Within Each User Story

- Tests written first and must FAIL before implementation.
- Services → handlers → routes → DI wiring (DI scaffold already exists from Foundational T024).
- Conditional status updates use `UpdateStatusIfCurrent` (rowsAffected guard).

### Parallel Opportunities

- All `[P]` tasks in Setup and Foundational phases run in parallel.
- US1 and US2 can be implemented in parallel by different developers after Phase 2.
- Within US1, test files T026/T027 run in parallel; models T014–T016 run in parallel.

---

## Parallel Example: Foundational Repositories

```bash
# Launch all repository tasks together (different files, no dependencies):
Task: "Extend payment/repository/payment_gateway_repository.go ..."
Task: "Create payment/repository/payment_gateway_field_repository.go ..."
Task: "Create payment/repository/payment_gateway_config_repository.go ..."
Task: "Create payment/repository/payment_transaction_repository.go ..."
Task: "Create payment/repository/payment_refund_repository.go ..."
Task: "Create payment/repository/payment_webhook_log_repository.go ..."
Task: "Create payment/repository/payment_transaction_event_repository.go ..."
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1 Setup → Phase 2 Foundational.
2. Phase 3 User Story 1 (tests first, then implementation).
3. **STOP and VALIDATE**: US1 independently (initiate + webhook + auto-confirm).
4. Deploy/demo the MVP.

### Incremental Delivery

1. Setup + Foundational → foundation ready.
2. Add US1 → test independently → deploy (MVP).
3. Add US2 → test independently → deploy (seller dashboard).
4. Add US3 → test independently → deploy (refunds).

### Parallel Team Strategy

1. Team completes Setup + Foundational together.
2. After Foundational: Developer A → US1; Developer B → US2.
3. Developer A (or C) → US3 after US1 lands.

---

## Notes

- `[P]` = different files, no dependencies.
- `[Story]` maps each task to its user story for traceability.
- The "Consistency Notes" section is authoritative for cross-doc alignment — follow it to avoid
  ambiguity between `spec.md`, `pre-spec.md`, `data-model.md`, and `contracts/`.
- Write integration tests first and confirm they fail before implementation.
- Commit after each task or logical group; stop at checkpoints to validate independently.
