# Tasks: Coupon / Discount Codes

**Input**: Design documents from `/specs/007-coupon-discount-codes/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, pre-spec.md  
**Tests**: **MANDATORY** (constitution TDD + integration-first). Write failing integration tests before production code in each story phase.  
**Organization**: Phases follow user stories US1–US5 from spec.md. Shared schema/DI in Setup + Foundational.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Parallelizable (different files, no incomplete dependency)
- **[Story]**: `[US1]`…`[US5]` on story phases only
- Every task includes concrete file paths

## Path Conventions

- Promotion module: `promotion/`
- Order/cart module: `order/`
- Migrations: `migrations/`
- Tests: `test/integration/promotion/`, `test/integration/order/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Schema alignment and entity drift fixes that all stories need.

- [x] T001 Create migration aligning `discount_code` columns (`description`, `max_discount_amount_cents`, `usage_reset_time_type`, `usage_reset_amount`), creating `cart_applied_coupon` and `cart_item_promotion`, scope `updated_at` if missing, and indexes in `migrations/028_align_discount_code_and_cart_coupon.sql`
- [x] T002 [P] Add `CurrentUsageCount` to `DiscountCode` in `promotion/entity/discount_code.go`
- [x] T003 [P] Add `ProductID` to `DiscountCodeProduct` in `promotion/entity/discount_code_scope.go`
- [x] T004 [P] Add `APIBaseDiscountCode = "/api/promotion/discount-code"` in `common/constants/api_constants.go`
- [x] T005 Ensure test DB runner picks up new migration via existing `test/integration/setup/database.go` (verify include path / ordering only)

**Checkpoint**: Migration + entities ready; `make migrate` / Testcontainers can apply schema.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared errors, messages, DTOs, and DI hooks. **No user story HTTP until this completes.**

**⚠️ CRITICAL**: Blocks US1–US5

- [x] T006 [P] Define AppError codes/messages for discount codes and cart coupons in `promotion/error/discount_code_error.go`
- [x] T007 [P] Add success/failure/field message constants in `promotion/utils/constant/discount_code_constants.go`
- [x] T008 [P] Add cart coupon success/failure message constants in `order/utils/constant/cart_constants.go`
- [x] T009 [P] Create request/response DTOs (`CreateDiscountCodeRequest`, `UpdateDiscountCodeRequest`, `UpdateDiscountCodeStatusRequest`, `ListDiscountCodesRequest`, `DiscountCodeResponse`, `ListDiscountCodesResponse`) in `promotion/model/discount_code_request_model.go` and `promotion/model/discount_code_response_model.go`
- [x] T010 [P] Create scope base + product/variant/category/collection request/response models with embedding in `promotion/model/discount_code_base_model.go` and `promotion/model/discount_code_*_scope_model.go`
- [x] T011 [P] Create internal coupon cart contracts (`CouponCartRequest`, `AppliedCouponSummary`, usage records, available coupon DTOs) in `promotion/model/coupon_cart_model.go`
- [x] T012 [P] Extend `CartResponse` with `AvailableCoupons` and require `ShippingDiscount` (+ formatted if non-zero) on `AppliedCouponInfo` in `order/model/cart_model.go` (used by free_shipping coupons)
- [x] T013 Create `DiscountCodeRepository` interface + GORM impl (FindByID, FindByCode, List, Create, Update, UpdateActive, Delete, CountUsage, IncrementUsageAtomically) in `promotion/repository/discount_code_repository.go`
- [x] T014 [P] Create `DiscountCodeUsageRepository` interface + impl (Create, CountByUserInWindow) in `promotion/repository/discount_code_usage_repository.go`
- [x] T015 Wire discount-code repos into `promotion/factory/singleton/repository_factory.go` and getters in `promotion/factory/singleton/singleton_factory.go`
- [x] T016 Add empty test suite scaffolds + endpoint path constants in `test/integration/promotion/discount_code_constants_test.go` and `test/integration/order/cart_coupon_constants_test.go`

**Checkpoint**: Foundation ready — stories can start (US1 first for MVP).

---

## Phase 3: User Story 1 — Seller Creates and Manages Discount Codes (P1) 🎯 MVP

**Goal**: Seller CRUD + activate/deactivate + hard-delete-when-unused for discount codes.

**Independent Test**: Seller creates/lists/gets/updates/status/deletes codes; duplicate + cross-seller isolation verified via HTTP.

### Tests for US1 (write first — must FAIL)

- [x] T017 [P] [US1] Add discount-code suite setup/helpers in `test/integration/promotion/discount_code_setup_test.go`
- [x] T018 [P] [US1] Write happy-path CRUD tests (create, get, list, update, status) in `test/integration/promotion/discount_code_crud_test.go`
- [x] T019 [P] [US1] Write sad-path tests (duplicate code, invalid value/dates, auth, correlation, cross-seller 404) in `test/integration/promotion/discount_code_sad_path_test.go` — **do not** cover delete-after-usage here (that is US5 `T058`)
- [x] T020 [P] [US1] Write code-normalization / immutable-code tests in `test/integration/promotion/discount_code_validation_test.go`

### Implementation for US1

- [x] T021 [US1] Implement `DiscountCodeMapper` create/update mapping in `promotion/factory/discount_code_mapper.go`
- [x] T022 [US1] Implement `DiscountCodeService` CRUD + status + delete guards in `promotion/service/discount_code_service.go` (+ `_impl` if split)
- [x] T023 [US1] Implement `DiscountCodeHandler` methods in `promotion/handler/discount_code_handler.go`
- [x] T024 [US1] Register seller routes in `promotion/route/discount_code_routes.go` and module in `promotion/container.go`
- [x] T025 [US1] Wire service/handler in `promotion/factory/singleton/service_factory.go` and `promotion/factory/singleton/handler_factory.go`
- [x] T026 [US1] Run US1 integration tests until green for packages under `test/integration/promotion/` (`go test ./test/integration/promotion/ -run DiscountCode -v`)

**Checkpoint**: MVP — sellers can manage store-scoped discount codes.

---

## Phase 4: User Story 2 — Seller Catalog Scopes (P1)

**Goal**: Attach/remove/list product, variant, category, collection scopes; reject mismatches with `appliesTo`.

**Independent Test**: Seller scoped CRUD via HTTP; `all_products` rejects scope add; cross-seller denied.

### Tests for US2 (write first — must FAIL)

- [x] T027 [P] [US2] Write product-scope integration tests in `test/integration/promotion/discount_code_scope_product_test.go`
- [x] T028 [P] [US2] Write variant/category/collection scope tests in `test/integration/promotion/discount_code_scope_other_test.go`
- [x] T029 [P] [US2] Write appliesTo-mismatch and isolation tests in `test/integration/promotion/discount_code_scope_validation_test.go`

### Implementation for US2

- [x] T030 [P] [US2] Implement product/variant/category/collection scope repositories in `promotion/repository/discount_code_*_scope_repository.go`
- [x] T031 [US2] Implement scope services mirroring promotion scope patterns in `promotion/service/discount_code_*_scope_service.go`
- [x] T032 [US2] Implement scope handlers in `promotion/handler/discount_code_*_scope_handler.go`
- [x] T033 [US2] Register scope routes in `promotion/route/discount_code_scope_routes.go` and wire factories/`promotion/container.go`
- [x] T034 [US2] Run US2 integration tests until green for `test/integration/promotion/discount_code_scope_*.go`

**Checkpoint**: Sellers can limit codes to catalog subsets.

---

## Phase 5: User Story 3 — Customer Apply / Remove / Available Coupons (P1)

**Goal**: Authenticated customers apply/remove coupons; cart returns full priced `CartResponse`; available coupons listed.

**Independent Test**: Seed code + cart → apply → summary shows coupon discount → remove → totals revert; guest apply denied.

### Tests for US3 (write first — must FAIL)

- [x] T035 [P] [US3] Add cart-coupon suite helpers in `test/integration/order/cart_coupon_setup_test.go`
- [x] T036 [P] [US3] Write apply/remove/available happy-path tests (include free_shipping shippingDiscount field and one buy_x_get_y metadata coupon) in `test/integration/order/cart_coupon_happy_path_test.go`
- [x] T037 [P] [US3] Write validation error tests (invalid, expired, not started, already applied, min purchase, **min quantity**, not applicable, cannot combine, **specific_segment → COUPON_NOT_ELIGIBLE stub**) in `test/integration/order/cart_coupon_sad_path_test.go`
- [x] T038 [P] [US3] Write guest-deny and auth/correlation tests in `test/integration/order/cart_coupon_auth_test.go`

### Implementation for US3

- [x] T039 [US3] Implement discount-type calculators (percentage, fixed, free_shipping, buy_x_get_y) under `promotion/service/discountStrategy/` only; BXGY reads metadata keys `buyQuantity`, `getQuantity`, `getDiscountPercent` (0–100; 100 = free) per contracts
- [x] T040 [US3] Implement `CouponApplyService` with `ApplyCouponsToCart` + `ListAvailableCouponsForCart` in `promotion/service/coupon_apply_service.go` and register it in `promotion/factory/singleton/service_factory.go` / `singleton_factory.go` (cart orchestration here — not on `DiscountCodeService`)
- [x] T041 [US3] Add cart coupon repo methods (list/add/remove/remove-all) for `CartAppliedCoupon` in `order/repository/cart_repository.go`
- [x] T042 [US3] Inject `DiscountCodeService` (CRUD) and `CouponApplyService` (cart apply) into order DI in `order/factory/singleton/service_factory.go`
- [x] T043 [US3] Extend cart pricing to call coupons after promotions and fill coupons in `order/service/cart_service.go` and `order/service/cart_operations.go`
- [x] T044 [US3] Update `order/factory/cart_response_builder.go` to populate `AppliedCoupons`, coupon summary fields, and `AvailableCoupons`
- [x] T045 [US3] Implement Apply/Remove/RemoveAll/Available handlers in `order/handler/cart_handler.go`
- [x] T046 [US3] Register routes `POST/DELETE /coupon`, `DELETE /coupon/:code`, `GET /available-coupon` in `order/route/cart_route.go`
- [x] T047 [US3] Run US3 integration tests until green for `test/integration/order/cart_coupon_*.go`

**Checkpoint**: Storefront can apply coupons and render full cart totals from one response.

---

## Phase 6: User Story 4 — Runtime Recalc & Self-Heal (P1)

**Goal**: Get Cart always recomputes money; invalid applied coupons dropped; promos before coupons; stacking flags honored; scoped-code cart eligibility (from US2 scopes); `cart_item_promotion` synced on each price build.

**Independent Test**: Apply coupon → seller deactivates → Get Cart clears coupon and fixes totals; stacking verified; scoped code fails on ineligible cart / succeeds on eligible items; after pricing, `cart_item_promotion` rows match applied promotions per line.

### Tests for US4 (write first — must FAIL)

- [x] T048 [P] [US4] Write self-heal tests (deactivate/expire/min-purchase change) in `test/integration/order/cart_coupon_self_heal_test.go`
- [x] T049 [P] [US4] Write stacking tests (`canCombine`, `canStackWithCoupons`) in `test/integration/order/cart_coupon_stacking_test.go`
- [x] T050 [P] [US4] Write scoped-code eligibility-on-cart tests in `test/integration/order/cart_coupon_scope_apply_test.go` and `cart_item_promotion` sync tests (junction matches applied promo IDs; stale rows removed) in `test/integration/order/cart_item_promotion_sync_test.go`

### Implementation for US4

- [x] T051 [US4] Implement self-heal removal of invalid `cart_applied_coupon` rows inside cart build in `order/service/cart_service.go`
- [x] T052 [US4] Pass `PromotionsAllowCoupons` / after-promo item totals into `CouponCartRequest` from `order/service/cart_service.go`
- [x] T053 [US4] Enforce stacking rules inside `promotion/service/coupon_apply_service.go`
- [x] T054 [US4] After `ApplyPromotionsToCart`, sync `cart_item_promotion`: for each cart line, replace stored promotion IDs with the set just applied (delete stale, insert missing) via methods in `order/repository/cart_repository.go`; call from `order/service/cart_service.go` cart build — never store discount amounts
- [x] T055 [US4] Run US4 integration tests until green for `test/integration/order/cart_coupon_self_heal_test.go`, `cart_coupon_stacking_test.go`, `cart_coupon_scope_apply_test.go`, `cart_item_promotion_sync_test.go`

**Checkpoint**: Cart totals stay trustworthy under rule changes.

---

## Phase 7: User Story 5 — Checkout Snapshot & Usage (P2)

**Goal**: Order placement snapshots coupons and records usage atomically; limits enforced under concurrency.

**Independent Test**: Place order with coupon → order shows snapshot → usage count increments → second redemption beyond limit fails.

### Tests for US5 (write first — must FAIL)

- [x] T056 [P] [US5] Write checkout coupon snapshot + usage happy path in `test/integration/order/order_coupon_checkout_test.go`
- [x] T057 [P] [US5] Write per-customer / global limit, **usage-reset window** (within window blocks; after window allows), and race-safe tests in `test/integration/order/order_coupon_usage_limit_test.go`
- [x] T058 [P] [US5] Write delete-blocked-after-usage seller test (**authoritative** for FR-015 after redemption) in `test/integration/promotion/discount_code_usage_delete_test.go`

### Implementation for US5

- [x] T059 [US5] Add `CreateOrderAppliedCoupons` to `order/repository/order_repository.go`
- [x] T060 [US5] Add `BuildOrderAppliedCouponsFromCartSnapshot` in `order/factory/order_builder.go` (mirror `BuildOrderAppliedPromotionsFromCartSnapshot`)
- [x] T061 [US5] Persist snapshots + call `RecordCouponUsages` / `IncrementUsageAtomically` inside `order/service/order_lifecycle.go` transaction
- [x] T062 [US5] Implement `RecordCouponUsages` on promotion usage path (service method used by order) using `promotion/repository/discount_code_usage_repository.go` — **coupon usage only**; do not implement `promotion_usage` recording in this feature
- [x] T063 [US5] Ensure converted carts do not reuse old coupon attachments for new active carts in `order/service/cart_service.go`
- [x] T064 [US5] Run US5 integration tests until green for `test/integration/order/order_coupon_*.go` and `test/integration/promotion/discount_code_usage_delete_test.go`

**Checkpoint**: End-to-end redemption integrity complete.

---

## Phase 8: Polish & Cross-Cutting

**Purpose**: Hardening, docs, consistency.

- [x] T065 [P] Add unit tests for pure coupon calculators in `promotion/service/discountStrategy/*_test.go`
- [x] T066 [P] Update outdated `/api/cart` references to `/api/order/cart` in `order/CART_API_PRD.md`
- [x] T067 [P] Mark coupon deferred items done in `promotion/PROMOTION_DEFERRED.md` / `promotion/PROMOTION_API_TODO.md`
- [x] T068 Verify quickstart smoke path documented in `specs/007-coupon-discount-codes/quickstart.md` against running APIs
- [x] T069 Optional harden: soft rate-limit on apply coupon in `order/handler/cart_handler.go` (handler-level guard; no new middleware file required for v1)
- [x] T070 Run full related suites under `test/integration/promotion/` and `test/integration/order/` (`go test ./test/integration/promotion/ ./test/integration/order/ -count=1`) and fix regressions

---

## Dependencies & Execution Order

### Plan ↔ tasks phase map

| Plan preview       | tasks.md                                        |
| ------------------ | ----------------------------------------------- |
| 0                  | Phase 1 Setup + Phase 2 Foundational            |
| 1 Seller CRUD      | Phase 3 US1                                     |
| 2 Scopes           | Phase 4 US2                                     |
| 3 Engine           | Phase 5 US3 (`CouponApplyService` + strategies) |
| 4 Cart integration | Phase 5–6 US3–US4                               |
| 5 Checkout usage   | Phase 7 US5                                     |
| 6 Harden           | Phase 8 Polish                                  |

### Phase Dependencies

- **Phase 1 Setup** → no deps
- **Phase 2 Foundational** → after Setup; **blocks all stories**
- **US1 (Phase 3)** → after Foundational — **MVP**
- **US2 (Phase 4)** → after US1 (needs codes to attach scopes)
- **US3 (Phase 5)** → after US1 (needs codes); scopes optional but recommended after US2 for rich tests
- **US4 (Phase 6)** → after US3
- **US5 (Phase 7)** → after US3 (needs applied coupons on cart)
- **Polish (Phase 8)** → after desired stories

### User Story Dependencies

```text
US1 (Seller CRUD)
  └── US2 (Scopes)
US1 ──> US3 (Cart apply/remove/available)
          ├── US4 (Self-heal / stacking / runtime)
          └── US5 (Checkout usage)
```

### Within Each Story

1. Tests written and failing
2. Models/repos/services
3. Handlers/routes/DI
4. Tests green

### Parallel Opportunities

- T002–T004; T006–T012; T017–T020; T027–T029; T035–T038; T048–T050; T056–T058; T065–T067
- After Foundational: US1 must lead; then US2 and early US3 engine work can partially parallelize if staffed (different modules: promotion scopes vs coupon calculator)

---

## Parallel Example: User Story 1

```bash
# Tests in parallel (after T017 setup):
# T018, T019, T020 → test/integration/promotion/discount_code_*.go

# Then implement sequentially:
# T021 mapper → T022 service → T023 handler → T024 routes → T025 wire → T026 green
```

## Parallel Example: User Story 3

```bash
# Tests: T036, T037, T038 in parallel after T035
# Impl: T039 calculators [P] with T041 cart repo [P]
# Then T040 engine → T042–T046 cart wiring → T047 green
```

---

## Implementation Strategy

### MVP First (US1 only)

1. Phase 1 + 2
2. Phase 3 US1 → demo seller can manage codes
3. Stop and validate

### Incremental Delivery

1. US1 → seller campaigns
2. US2 → scoped campaigns
3. US3 → customer apply (storefront-ready cart)
4. US4 → trust/self-heal
5. US5 → checkout integrity
6. Polish

### Suggested MVP Scope

**US1 only** (seller CRUD). Minimum useful customer slice is **US1 + US3** (codes + apply). Full storefront-ready = **US1–US4**. Production-complete = **US1–US5**.

---

## Notes

- **Terminology**: **Discount code** = seller-managed definition (`/api/promotion/discount-code`). **Coupon** = customer apply/remove on cart (`/api/order/cart/coupon`). Avoid “promo code” in new docs/tasks.
- **FR-008 vs FR-009**: US3 delivers storefront cart surface (FR-008); US4 hardens runtime sync/self-heal/stacking (FR-009 + related). Both required; not duplicate work.
- Follow constitution: no magic strings; AppErrors; SellerAuth/CustomerAuth; seller_id on every code query
- Cart junctions remain reference-only — never store discount amounts on cart tables
- Order talks to promotion via service interfaces only (`CouponApplyService` for cart apply)
- Wire `CouponApplyService` + `discountStrategy/` calculators (not on `DiscountCodeService`)
- Prefer patterns from existing `promotion/` sale + scope modules and `order/` cart promotion integration
- Detailed contracts/logic: `pre-spec.md` and `contracts/api-contracts.md`
- **Out of scope this feature**: recording `promotion_usage` on order (automatic promotions); coupon usage only
