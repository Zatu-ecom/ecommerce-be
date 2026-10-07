# Tasks: Courier Fulfillment Platform (013)

**Input**: Design documents from `/specs/013-courier-fulfillment-platform/`
**Prerequisites**: plan.md, spec.md (6 stories), research.md (R1–R10), data-model.md (T1–T8), contracts/courier-partner.md, api-contracts.md, physical-specs.md, quickstart.md

**Tests**: INCLUDED — constitution mandates TDD + integration-first (Testcontainers, suite pattern). **Test matrix**: every test task below covers the full constitution matrix (happy path, authN/authZ failures, validation errors, edge cases, correlation-ID enforcement, seller isolation) unless it says otherwise.

**Organization**: One phase per user story (P1 → P2). TDD order inside each story: tests FAIL first, then entities → repos → services → handlers/routes.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Parallelizable (different files, no pending dependencies)
- **[Story]**: US1–US6 maps to spec.md stories
- Exact file paths in every task

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Migrations, seeds, module skeleton, test harness — no business logic

- [x] T001 Write migration `migrations/032_create_fulfillment_tables.sql` (T1–T8 per data-model.md §3, FK order, partial uniques, comments)
- [x] T002 Write migration `migrations/033_create_physical_spec_unit.sql` per physical-specs.md §2 (lands after 032)
- [x] T003 Write core seed `migrations/seeds/core/005_seed_physical_specs.sql` per physical-specs.md §3 (ids 100–107, self-check block)
- [x] T004 [P] Create `fulfillment/` skeleton: `container.go`, `factory/singleton/` wiring stub, `error/fulfillment_errors.go` (all FULFILLMENT_* codes per service-design §8)
- [x] T005 [P] Create `fulfillment/cache/keys.go` (volatile/durable key builders + allowlist additions per data-model §2b2/2c)
- [x] T006 [P] Create integration test harness `test/integration/fulfillment/setup_suite_test.go` (suite, endpoint constants, Wiremock Shiprocket base URL override)
- [x] T007 Verify from scratch: dropdb/createdb + full migration run + seed NOTICE output (4 families, 8 units) + `go build ./...`

**Checkpoint**: Schema + seeds green, skeleton compiles, harness boots.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Entities, repositories, provider contract, factory, crypto, hook interfaces — MUST complete before ANY story

**⚠️ CRITICAL**: No user story work begins until this phase is complete.

- [x] T008 [P] Implement entities `fulfillment/entity/shipment.go` (extend stub: T4 + items T5 + `TableName()`, status vocabulary per data-model §2a)
- [x] T009 [P] Implement entities `fulfillment/entity/provider.go` (T1+T2), `provider_config.go` (T3 + auto_book/rate_preference/default_weight_grams), `shipment_event.go` (T6), `webhook_log.go` (T7), `ndr.go` (T8)
- [x] T010 [P] Implement repositories `fulfillment/repository/*_repository.go + *_repository_impl.go` (shipment, items, events, provider, config, webhook_log, ndr — all seller-scoped, `UpdateStatusIfCurrent`, `SKIP LOCKED` sweep queries)
- [x] T011 Implement `CourierPartner` contract + frozen `ShipmentAction` + `NormalizedShipmentEvent` + DTOs in `fulfillment/service/courier/contract.go` (verbatim from contracts/courier-partner.md)
- [x] T012 [P] Implement `fulfillment/service/courier/crypto.go` (ResolveEncryptionKey, fail-closed) + `fulfillment/factory/courier_partner_factory.go` (registry + hybrid `ResolveForSeller`)
- [x] T013 [P] Implement Shiprocket base files `fulfillment/service/courier/shiprocket/{credentials.go,http.go,auth.go}` (typed creds, Validate/Encrypt/Decrypt/MaskHints/MergePartial, pooled client GET-retry-only + log redaction, JWT in durable KV + 401 refresh)
- [x] T014 Define cross-module hook interfaces `FulfillmentOrderHooks` + `FulfillmentInventoryHooks` + `FulfillmentProductHooks` in `fulfillment/service/hooks.go` (per contracts/courier-partner.md; compile-time guards)
- [x] T015 Register module in `main.go` (`fulfillment.NewContainer(router)`) + webhook route exempt from mandatory-correlation-ID middleware (generate + echo)
- [x] T016 Foundational integration test: API smoke on migrated + seeded DB (empty-catalog responses) in `test/integration/fulfillment/foundation_test.go`

**Checkpoint**: Foundation ready — entities persist, factory resolves, hooks compile, module serves traffic.

---

## Phase 2B: Cross-Module Hook Implementations (Blocking)

**Purpose**: Order/inventory-side code that fulfillment consumes via hooks. Resolves analysis C1 — fulfillment-side tasks only *call* these; without this phase US2/US3/US5 have nothing behind their interfaces. Owners: order + inventory teams. Execute with Phase 2; MUST complete before user stories.

- [x] T017 [P] Order view: implement `GetOrderForFulfillment` in `order/service/order_fulfillment_view.go` (lines with item/variant/qty, fulfillment_type, delivery address id + updated_at + pincode, COD cents) + integration test `test/integration/order/order_fulfillment_view_test.go`
- [x] T018 Order aggregation: implement `OnShipmentsPlanned/Booked/Delivered/Failed/Returned` in `order/service/order_fulfillment_hooks.go` (order status derived from all boxes; compile-time guard asserting `FulfillmentOrderHooks`) + integration test `test/integration/order/order_fulfillment_hooks_test.go`
- [x] T019 Order stock paths: reservation release (cancel, `cancelled_pre_pickup` fails) + restock-once guard per return-shipment-id in `order/service/order_fulfillment_stock.go` + integration test (double-observed return restocks exactly once)
- [x] T020 [P] Inventory surface: implement `GetAvailability`/`ReserveForShipment`/`ReleaseReservation` in `inventory/service/inventory_fulfillment_hooks.go` + integration test `test/integration/inventory/inventory_fulfillment_hooks_test.go` (concurrent orders never plan the same units)
- [x] T021 Inventory TTL: verify confirmed reservations carry no TTL (service-design E10); remove/adjust expiry if present + regression test proving long-lived drafts keep their holds
- [x] T022 Cross-module conformance test `test/integration/fulfillment/hooks_conformance_test.go` (guards compile; plan → adopt → book → deliver emits the expected hook calls in order)

**Checkpoint**: Fulfillment's hooks resolve to real implementations; stories can now build on both sides of each interface.

---

## Phase 3: User Story 1 — Seller connects courier account (P1) 🎯 MVP

**Goal**: Sellers connect Shiprocket (or platform default), test creds, toggle auto-book — masked secrets throughout.
**Independent Test**: Connect test creds → test OK → masked config visible; bad creds → 400, nothing saved; no creds → platform default applies.

### Tests for US1 (write FIRST, FAIL before implementation)

- [x] T023 [P] [US1] Integration tests `test/integration/fulfillment/courier_config_test.go` (catalog, detail, configure happy + validation + authN/authZ + correlation-ID + seller isolation)
- [x] T024 [P] [US1] Integration tests for test-connection + unknown-code 404 in `test/integration/fulfillment/courier_test_test.go`

### Implementation for US1

- [x] T025 [P] [US1] Implement `fulfillment/model/provider_model.go` (configure/test DTOs, pointer updates, `dive`) + `fulfillment/service/provider_config_service.go` (resolve/configure/test/mask)
- [x] T026 [US1] Implement handlers + routes `fulfillment/handler/provider_handler.go`, `fulfillment/route/provider_route.go` (`GET /couriers`, `GET /couriers/:code`, `PUT /couriers/:code/configure`, `POST /couriers/:code/test` with webhook URL from `PUBLIC_API_BASE_URL`)
- [x] T027 [US1] Environment semantics: sandbox vs production rows, frozen-via-`provider_config_id`, per-env test/deactivate (service-design §4)

**Checkpoint**: US1 fully functional and independently testable — a shippable seller with zero shipments.

---

## Phase 4: User Story 2 — Confirmed orders become shippable boxes (P1)

**Goal**: Auto-plan one draft per warehouse on `OrderConfirmed`; manual draft creation + draft edits.
**Independent Test**: Confirm multi-warehouse order → one draft per warehouse, correct lines/ids; replay → no dupes; single warehouse → exactly one draft.

### Tests for US2 (write FIRST)

- [x] T028 [P] [US2] Planner tests `test/integration/fulfillment/plan_test.go` (grouping, priority split, Σ-guard, concurrent double-plan, replay no-op, pending order → 409)
- [x] T029 [P] [US2] Draft CRUD tests `test/integration/fulfillment/shipment_draft_test.go` (manual create, PATCH weight/dims, wrong-owner 404, validation)

### Implementation for US2

- [x] T030 [P] [US2] Implement `fulfillment/service/shipment_planner.go` (guard → availability hook → greedy priority allocate → reserve → group-by-location tx with id stamping + `OnShipmentsPlanned`; shortfall → `STOCK_MISMATCH`, never partial)
- [x] T031 [US2] Implement draft paths in `fulfillment/service/shipment_service.go` (Tx1 create + AdoptReservation + idempotency guard) + `POST /orders/:orderId/plan`, `POST /shipments`, `PATCH /shipments/:id`, `GET /shipments`, `GET /shipments/:id` handlers/routes
- [x] T032 [US2] Wire `PlanForOrder` invocation on order `pending → confirmed` commit (order module, narrow interface, post-commit only)

**Checkpoint**: US1 + US2 work — confirmed orders yield correct drafts with no courier contact or money spent.

---

## Phase 5: User Story 3 — Book a box, courier picks it up (P1)

**Goal**: Draft → booked + AWB + pickup (+ label); seller book-all; opt-in auto-book; draft crash recovery.
**Independent Test**: Book draft → AWB + pickup state; missing weight → 400; changed address → 409 until confirmed; double-click → single booking; process-death → recover job finishes it.

### Tests for US3 (write FIRST)

- [x] T033 [P] [US3] Adapter contract tests (Wiremock) `fulfillment/service/courier/shiprocket/adapter_wiremock_test.go` (book/cancel/label/track, 401, 429, bad JSON, shipment-id-as-order-id, unknown status → ignore)
- [x] T034 [P] [US3] Booking flow tests `test/integration/fulfillment/book_test.go` (precondition order, idempotent replay, 502→resume, over-attempt cap, seller isolation)
- [x] T035 [P] [US3] Rate + label + pickup + cancel tests `test/integration/fulfillment/rate_label_test.go` (cache hit, limiter, PDF stream, pre/post-pickup cancel paths)

### Implementation for US3

- [x] T036 [P] [US3] Implement `fulfillment/service/courier/shiprocket/{shipment.go,rates.go}` (create-adhoc → assign-AWB → optional pickup; serviceability; `order_id` = our shipment id; derived nickname `S{seller}L{location}` always sent)
- [x] T037 [US3] Implement book path in `fulfillment/service/shipment_service.go` (preconditions → pre-HTTP mark tx → HTTP → Tx2; `BookAll` per-draft precedence; `ConfirmAddress` + serviceability re-check; `default_weight_grams` fallback; `RequestRTO` stays in US5)
- [x] T038 [US3] Implement `fulfillment/service/rate_service.go` (volatile 90s + singleflight + Lua limiter; pincodes hook-resolved, never stored) + `POST /rates`
- [x] T039 [US3] Implement book/pickup/cancel/label/refresh-seller handlers + routes (`POST /shipments/:id/book`, `/orders/:orderId/book`, `/pickup`, `/cancel`, `/label` PDF stream, `/confirm-address`)
- [x] T040 [US3] Implement `recover_drafts` cron (2m, `SKIP LOCKED`, attempt cap 5 + alert, seller-retry reset) via `common/cron`; wire tenant `auto_book` + `rate_preference` + `AUTO_BOOK_AMBIGUOUS` path in planner step 7

**Checkpoint**: US1–US3 work — money moves correctly, crashes heal, humans stay in control unless opted in.

---

## Phase 6: User Story 4 — Tracking arrives on its own; buyers follow along (P1)

**Goal**: Courier pushes advance boxes via webhook; cron heals gaps; buyers track from our records only.
**Independent Test**: Push updates → status/history advance; replay/out-of-order safe; terminal never regresses; stale box healed by cron; buyer view instant + isolated.

### Tests for US4 (write FIRST)

- [x] T041 [P] [US4] Webhook matrix tests `test/integration/fulfillment/webhook_test.go` (valid apply, bad-sig 401/no-rows, unknown AWB 200/no-rows, replay single-apply, OFD→in-transit allowed, delivered never regresses, verified-apply-fail 200 + cron heals)
- [x] T042 [P] [US4] Tracking tests `test/integration/fulfillment/track_test.go` (customer GET PG-only, seller refresh writes via Apply, singleflight, webhook-logs list, cross-seller/customer 404s, reconcile sweep: stale selection, bulk-per-config, system-EventID idempotency)

### Implementation for US4

- [x] T043 [P] [US4] Implement `fulfillment/service/courier/shiprocket/{webhook.go,track.go}` (PeekLocators, HMAC verify on raw bytes, sr-status → action map, single + bulk track)
- [x] T044 [US4] Implement `fulfillment/service/apply_normalized.go` (single switch on `ShipmentAction`, allow-list, idempotent no-ops, AWB-mismatch refuse, per-action order hooks with lines)
- [x] T045 [US4] Implement `fulfillment/service/webhook_service.go` (locate → verify → dedupe → apply; 200/401 matrix) + public `POST /webhooks/:code` route
- [x] T046 [US4] Implement `fulfillment/service/tracking_service.go` (`GetTrack` PG-only, `RefreshTrack` seller-only) + customer `GET /my/orders/:orderId/shipments` + `GET /webhook-logs` routes
- [x] T047 [US4] Implement `reconcile_pending` cron (5m, stale via `last_synced_at`, bulk/100 per config, system EventIDs) via `common/cron`

**Checkpoint**: US1–US4 work — live visibility with no seller action, gaps self-heal.

---

## Phase 7: User Story 5 — Failed deliveries and returns (P2)

**Goal**: NDR rounds with re-attempt/RTO, proactive mid-transit RTO requests, post-delivery return boxes with exactly-once restock.
**Independent Test**: Failed attempt → seller notified → re-attempt resumes / RTO returns; delivered box → return box → restock once; double-observed return restocks once.

### Tests for US5 (write FIRST)

- [x] T048 [P] [US5] NDR/return tests `test/integration/fulfillment/ndr_return_test.go` (round lifecycle, one-open-round guard, same-reason second round, return-of-return rejected, over-quantity rejected, restock-once)

### Implementation for US5

- [x] T049 [P] [US5] Implement `fulfillment/service/courier/shiprocket/ndr.go` (GetNDR/ActNDR, RequestRTO via RTO surface) behind `NDRHandler`/`ReturnHandler`/`RTORequester`
- [x] T050 [US5] Implement `ActNDR`/`RequestRTO`/`RequestReturn` in `fulfillment/service/shipment_service.go` + routes (`POST /shipments/:id/ndr`, `/rto`, `/returns`) + `ndr_sweep` cron (15m, 24h escalation via dashboard-visible flag; push/email channels deferred) + `OnShipmentFailed` reason vocabulary enforcement

**Checkpoint**: US1–US5 work — delivery failures become revenue recovered or controlled cost.

---

## Phase 8: User Story 6 — Shippable product specs (P2)

**Goal**: Sellers enter weight/dims in familiar units; planner/rates/book compute in base units; missing specs degrade to human action.
**Independent Test**: Create product with kg/m specs → planner uses correct g/cm; unknown unit rejected at write; spec-less product → draft flagged, book refused without weight.

### Tests for US6 (write FIRST)

- [x] T051 [P] [US6] Spec catalog + endpoint tests `test/integration/product/product_attribute/physical_spec_test.go` (grouped response shape, per-family rejection, non-numeric rejected, inactive hidden) — 4/4 green

### Implementation for US6

- [x] T052 [P] [US6] Product module: `GET /api/product/attribute/definitions?scope=fulfillment` (grouped from `physical_spec_unit`), write-path validation (numeric > 0, one key per family), badge `GET /api/product/:productId/attribute/shipping-specs` — done (route lives under `/api/product/attribute/...` per codebase convention, not `/api/products/...`)
- [x] T053 [US6] Fulfillment: `GetPhysicalSpecs` reader (join by key, × factor) wired into planner weight fill + dims heuristic (max L, max B, stacked H); verified by `TestPlan_WeightFillFromCatalogSpecs` — done (no boot self-check cache; reader queries per plan, catalog is 8 rows)
- [x] T054 [US6] Dashboard handoff: `specs/013-courier-fulfillment-platform/contracts/dashboard-handoff.md` written (endpoint map for the spec block + unit dropdowns from the specs API); seller-frontend team sign-off + FE repo work out of scope here

**Checkpoint**: All six stories independently functional.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Ship-ready hardening across all stories

- [x] T055 Run quickstart.md end-to-end validation (migrate → seed → configure → plan → book → webhook → track) — scratch DB migrated+seeded clean, SQL checks green (shiprocket row, 4 families × 2 units), full fulfillment suite `ok` (66s)
- [x] T056 [P] Update `postman/` collection with all 24 endpoints + webhook sample bodies — done: `Fulfillment` folder, 27 requests (25 endpoints incl. US6 pair + 3 webhook samples: in-transit/delivered/NDR)
- [x] T057 [P] Format/lint gate: `gofumpt`/`golines` formatting, method ≤ 50 lines, file ≤ 500 lines, no provider imports outside adapter folders (grep gate) — done: gofumpt clean on fulfillment/ + US6 files; new code extracted to `shipment_measures.go` (planner 458 lines, all methods ≤ 50); provider gate added to `make arch-check` and passing (pre-existing violations in untouched files documented, out of scope)
- [x] T058 Full suite green: `go test ./test/integration/... -v` + migration-from-scratch re-run + seed idempotency (re-run `005` twice, counts unchanged) — done with notes: 005 twice → 8 units/4 families/8 defs; per-package fresh-DB runs: fulfillment/inventory/order/product-root/category/attribute/payment(fixed)/report + 9 product subpackages green; fixed payment-suite cleanup to clear fulfillment_* first (RESTRICT FK vs auto-planned drafts); Docker-dependent suites (file/variant_media/collection-upload/user-logo/order+cart images/promotion-image/cachekit-restart) cannot run here — Docker socket not accessible; one product sort assertion fails on local PG18 collation (byte-order vs en_US collation, pre-existing, passes on PG16 image)
- [x] T059 [P] Structured-log audit (shipmentId/awb/action/sellerId/correlationId, no secrets/PII) + reconcile/webhook metrics wiring — done: audit clean (field names only in errors, courier bodies redacted, raw webhook payloads never logged, api_email public-by-design); new `fulfillment/metrics` recorder (log sink, Prometheus-ready shape, mirrors cachekit.Recorder) wired into webhook (7 outcomes) + reconcile/NDR sweeps via nil-safe setters; verified live (applied/duplicate/ignored/unknown_awb/unverified + sweep summaries in test logs)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately (T001→T002→T003 migrations/seeds in order; T004–T006 parallel)
- **Foundational (Phase 2 + 2B)**: BLOCKS all stories (T008–T010 entities/repos parallel; then T011–T014; T015–T016 last; 2B covers order/inventory hook implementations T017–T022)
- **Stories (Phase 3+)**: All depend on Phase 2; P1 stories (US1→US2→US3→US4) sequential (each builds on the last); US5/US6 after US4 (need applier + planner respectively)
- **Polish (Phase 9)**: After all desired stories

### User Story Dependencies

- **US1 (P1)**: After Phases 2/2B only — **MVP slice 1** (configured seller, no shipments)
- **US2 (P1)**: After US1 (needs provider catalog for planner picks) — **MVP slice 2** (+ auto-drafts)
- **US3 (P1)**: After US2 (books drafts) — **MVP complete** (money moves, boxes fly)
- **US4 (P1)**: After US3 (advances booked boxes) — production-grade visibility
- **US5 (P2)**: After US4 (needs applier + webhook paths)
- **US6 (P2)**: After US2 (plugs weight into planner); product-module tasks (T046) parallel-safe anytime after Phase 2

### Within Each Story

- Tests FAIL first → entities/models → services → handlers/routes → cron → checkpoint demo

### Parallel Opportunities

- T004/T005/T006; T008/T009/T010; T012/T013/T014; all `* _test.go` tasks per phase; T046 alongside any story phase; T050/T051/T053 in polish

---

## Parallel Example: Phase 5 (US3)

```bash
# Tests first, together:
Task: "Adapter contract tests (Wiremock) fulfillment/service/courier/shiprocket/adapter_wiremock_test.go"   # T033
Task: "Booking flow tests test/integration/fulfillment/book_test.go"                                          # T034
Task: "Rate + label + pickup + cancel tests test/integration/fulfillment/rate_label_test.go"                  # T035
# Then entities/services in parallel where files differ:
Task: "Shiprocket shipment+rates implementation"  # T030  (after T027 red)
Task: "Rate service + POST /rates"                # T032  (after T029 red)
```

---

## Implementation Strategy

### MVP First (US1 → US2 → US3)

> Scope note: US4 is P1 but post-MVP — manual refresh covers tracking until auto-heal lands. MVP demo = confirm → drafts → book → AWB → label.

1. Phase 1 + Phase 2 (foundation)
2. US1 (configured sellers) → validate
3. US2 (auto-drafts) → validate
4. US3 (booking + pickup) → **STOP, MVP demo**: confirm order → drafts → book → AWB → label
5. Deploy behind seller opt-in; US4–US6 incrementally

### Incremental Delivery

Each story adds value without breaking previous stories (webhook/cron in US4 only accelerate what US3 already does manually via refresh).

---

## Notes

- [P] = different files, no pending dependencies; [USn] traces to spec.md stories
- **Alerting**: every "alert once" in this file = structured ERROR log (correlationId, sellerId, shipmentId, code), emitted once per entity; paging/notification channels are out of scope
- Commit after each task or logical group; stop at any checkpoint for independent validation
- Product-module task T052 lives outside `fulfillment/` — coordinate, don't duplicate
- Dashboard (seller frontend) is a separate repo concern — T054 is a contract handoff note only
- **59 tasks total**: Setup 7 · Foundational 9 · XMOD 6 · US1 5 · US2 5 · US3 8 · US4 7 · US5 3 · US6 4 · Polish 5
