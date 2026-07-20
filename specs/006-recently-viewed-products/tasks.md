# Tasks: Recently Viewed Products

**Input**: Design documents from `/specs/006-recently-viewed-products/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are MANDATORY per constitution TDD requirement. All integration tests use `testify/suite` pattern with Testcontainers.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. US1, US2, and US3 share the same production code infrastructure and are tested together in Phase 4/5.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

---

## Phase 1: Setup — Database Migration

**Purpose**: Create the `user_recently_viewed` table and indexes. This is the foundational schema that ALL subsequent code depends on.

- [x] T001 Create database migration file with table definition, UNIQUE constraint on (user_id, product_id), indexes on user_id, (user_id, viewed_at DESC), product_id, CASCADE delete on product FK, and updated_at trigger in [`migrations/025_create_recently_viewed_table.sql`](migrations/025_create_recently_viewed_table.sql)

**Checkpoint**: Migration file exists — ready to apply with `make migrate` during test setup.

---

## Phase 2: Foundational — Entity, Constants, Repository & Service

**Purpose**: Core domain layer that ALL user stories depend on. Entity, error codes, message constants, repository interface+implementation, and service interface+implementation.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T002 [P] Define `RecentlyViewed` entity struct with BaseEntity embedding, UserID, SellerID, ProductID, ViewedAt fields, and TableName() override in [`product/entity/recently_viewed.go`](product/entity/recently_viewed.go)
- [x] T003 [P] Add `FAILED_TO_RECORD_RECENTLY_VIEWED_CODE` and `FAILED_TO_TRIM_RECENTLY_VIEWED_CODE` error code constants in [`product/utils/error_constants.go`](product/utils/error_constants.go)
- [x] T004 [P] Add `FAILED_TO_RECORD_RECENTLY_VIEWED_MSG` and `FAILED_TO_TRIM_RECENTLY_VIEWED_MSG` message constants in [`product/utils/message_constants.go`](product/utils/message_constants.go)
- [x] T005 [P] Add `SellerID uint` field — already exists in [`product/model/product_model.go`](product/model/product_model.go) with `json:"sellerId"` tag (no change needed)
- [x] T006 [P] Set `response.SellerID = product.SellerID` — already set in `BuildProductResponse` in [`product/factory/product_factory.go`](product/factory/product_factory.go) (no change needed)
- [x] T007 [P] Create `RecentlyViewedRepository` interface (4 methods: UpsertByUserAndProduct, CountByUserID, DeleteOldestByUserID, FindByUserID) and `RecentlyViewedRepositoryImpl` with GORM OnConflict upsert, count, subquery-based delete, and ordered find in [`product/repository/recently_viewed_repository.go`](product/repository/recently_viewed_repository.go)
- [x] T008 Create `RecentlyViewedService` interface (2 methods: RecordRecentlyViewed, GetRecentlyViewed) and `RecentlyViewedServiceImpl` with fire-and-forget recording (upsert → count → trim if >10) and product ID retrieval in [`product/service/recently_viewed_service.go`](product/service/recently_viewed_service.go) (depends on T007)

**Checkpoint**: Foundation ready — entity, repository, and service layers exist. User story implementation can now begin.

---

## Phase 3: Factory / Singleton Registration

**Purpose**: Wire the new repository and service into the 4-file singleton dependency injection chain so `ProductHandler` can receive the `RecentlyViewedService`.

- [x] T009 Register `RecentlyViewedRepo` field in `RepositoryFactory` struct and initialize in constructor in [`product/factory/singleton/repository_factory.go`](product/factory/singleton/repository_factory.go)
- [x] T010 Register `RecentlyViewedService` field in `ServiceFactory` struct and initialize via `service.NewRecentlyViewedService(repoFactory.GetRecentlyViewedRepository())` in [`product/factory/singleton/service_factory.go`](product/factory/singleton/service_factory.go) (depends on T009)
- [x] T011 Pass `sf.RecentlyViewedService` to `handler.NewProductHandler(...)` call in [`product/factory/singleton/handler_factory.go`](product/factory/singleton/handler_factory.go) (depends on T010)
- [x] T012 Add `GetRecentlyViewedRepository()` and `GetRecentlyViewedService()` getter methods in [`product/factory/singleton/singleton_factory.go`](product/factory/singleton/singleton_factory.go) (depends on T009, T010)

**Checkpoint**: Dependency injection chain wired — handler construction can now accept `RecentlyViewedService`.

---

## Phase 4: Tests for US1, US2, US3 (P1 — Automatic Recording, Role Restriction, Fire-and-Forget)

**Goal**: Write ALL integration tests FIRST (TDD Red phase). Tests cover: happy path recording (US1), role-based restriction (US2), and fire-and-forget error resilience (US3). These tests WILL FAIL until Phase 5 handler integration is complete.

**Independent Test**: Run `go test ./test/integration/product/product/get_product_by_id/ -run "RecentlyViewed" -v` — tests should FAIL (handler not modified yet).

### Test Infrastructure

- [x] T013 [P] Note: No static seed data file needed — all recently viewed data is inserted programmatically by test helpers in the setup suite.
- [x] T014 Create shared test suite setup: helper functions (`seedRecentlyViewed`, `countByUserAndProduct`, `countByUser`, `clearRecentlyViewed`, `getProductSellerID`) in [`test/integration/product/product/get_product_by_id/recently_viewed_setup_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_setup_test.go)

### Happy Path Tests (US1 — Automatic Recording)

- [x] T015 [P] [US1] Write `TestCustomerView_ProductRecorded` (HP-RV-01): customer views product → row inserted with correct user_id, seller_id, product_id, recent viewed_at in [`test/integration/product/product/get_product_by_id/recently_viewed_happy_path_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_happy_path_test.go)
- [x] T016 [P] [US1] Write `TestCustomerView_SellerIDCaptured` (HP-RV-02): customer views product → seller_id correctly captured from product in [`test/integration/product/product/get_product_by_id/recently_viewed_happy_path_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_happy_path_test.go)
- [x] T017 [P] [US1] Write `TestReView_SameProduct_UpdatesTimestamp` (HP-RV-03): re-view same product → viewed_at updated, no duplicate row in [`test/integration/product/product/get_product_by_id/recently_viewed_happy_path_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_happy_path_test.go)
- [x] T018 [P] [US1] Write `TestUnderLimit_CountIncrements` (HP-RV-04): view when under 10-entry limit → count increments normally in [`test/integration/product/product/get_product_by_id/recently_viewed_happy_path_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_happy_path_test.go)

### Sad Path Tests (US2 — Role Restriction + US3 — Fire-and-Forget)

- [x] T019 [P] [US2] Write `TestPublicUser_NoRecordCreated` (SP-RV-01): unauthenticated user → product returns 200, no recording in [`test/integration/product/product/get_product_by_id/recently_viewed_sad_path_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_sad_path_test.go)
- [x] T020 [P] [US2] Write `TestNonExistentProduct_NoRecordCreated` (SP-RV-02): non-existent product → 404, no recording in [`test/integration/product/product/get_product_by_id/recently_viewed_sad_path_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_sad_path_test.go)
- [x] T021 [P] [US2] Write `TestWrongSellerID_NoRecordCreated` (SP-RV-03): wrong X-Seller-ID → 404, no recording in [`test/integration/product/product/get_product_by_id/recently_viewed_sad_path_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_sad_path_test.go)
- [x] T022 [P] [US2] Write `TestInvalidProductID_NoRecordCreated` (SP-RV-04): non-numeric product ID → 400, no recording in [`test/integration/product/product/get_product_by_id/recently_viewed_sad_path_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_sad_path_test.go)
- [x] T023 [P] [US2] Write `TestMultipleDistinctProducts_NoDuplication` (SP-RV-05): 3 distinct products → 3 rows, correct counts in [`test/integration/product/product/get_product_by_id/recently_viewed_sad_path_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_sad_path_test.go)
- [x] T023a [P] [US3] Write `TestRecordingFailure_ProductResponseUnchanged` (SP-RV-06): product response always HTTP 200 with complete data (fire-and-forget resilience) in [`test/integration/product/product/get_product_by_id/recently_viewed_sad_path_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_sad_path_test.go)

### Edge Case Tests (US2 — Role Restriction + US1 — Limit Boundary)

- [x] T024 [P] [US1] Write `TestMaxLimit_OldestTrimmed` (EC-RV-01): 10 entries → 11th view trims oldest, keeps exactly 10 in [`test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go)
- [x] T025 [P] [US1] Write `TestReView_RefreshesPosition` (EC-RV-02): re-view refreshes position in trim ordering in [`test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go)
- [x] T026 [P] [US1] Write `TestUserIsolation_IndependentLists` (EC-RV-03): user A's views don't affect user B's list in [`test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go)
- [x] T027 [P] [US1] Write `TestMaxLimit_IdempotentTrim` (EC-RV-04): re-view at limit doesn't double-trim in [`test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go)
- [x] T028 [P] [US2] Write `TestSellerView_NotRecorded` (EC-RV-05): seller views product → NOT recorded (role level 2) in [`test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go)
- [x] T029 [P] [US2] Write `TestAdminView_NotRecorded` (EC-RV-06): admin views product → NOT recorded (role level 1) in [`test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go)
- [x] T030 [P] [US1] Write `TestZeroPriceProduct_Recorded` (EC-RV-07): zero-price product → view still recorded in [`test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go)
- [x] T031 [P] [US1] Write `TestUnicodeProduct_Recorded` (EC-RV-08): unicode product name → view still recorded in [`test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go)
- [x] T031a [P] [US1] Write `TestProductDeletion_CascadesRecord` (EC-RV-09): customer views product → record exists → delete the product via API → recently viewed record is removed (CASCADE) in [`test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go) (requires seller/admin auth to delete product, then verify customer's record count drops)

**Checkpoint**: All 19 tests written (T015-T031a) — they should FAIL because handler hasn't been modified yet. Confirm with: `go test ./test/integration/product/product/get_product_by_id/ -run "RecentlyViewed" -v`

---

## Phase 5: Handler Integration (US1 + US2 + US3 Implementation)

**Goal**: Modify `ProductHandler` to accept and call `RecentlyViewedService`, making ALL Phase 4 tests pass (TDD Green phase).

- [x] T032 [US1] [US2] [US3] Add `recentlyViewedService` field to `ProductHandler` struct and update constructor `NewProductHandler` to accept `service.RecentlyViewedService` parameter in [`product/handler/product_handler.go`](product/handler/product_handler.go)
- [x] T033 [US1] [US2] [US3] Add recently-viewed recording call in `GetProductByID` method: after successful query, check `userIDPtr != nil`, then check `roleLevel == constants.CUSTOMER_ROLE_LEVEL` via `auth.GetUserRoleLevelFromContext(c)`, then call `h.recentlyViewedService.RecordRecentlyViewed(ctx, *userIDPtr, productResponse.SellerID, productID)` in [`product/handler/product_handler.go`](product/handler/product_handler.go) (depends on T032)

**Checkpoint**: Run ALL 19 tests (US1+US2+US3) — they should now PASS. `go test ./test/integration/product/product/get_product_by_id/ -run "RecentlyViewed" -v`

---

## Phase 6: User Story 4 — Retrieve Recently Viewed Products (Priority: P2) 🎯 Optional

**Goal**: Add `GET /api/product/recently-viewed?limit={N}` endpoint so customers can retrieve their recently viewed product IDs.

**Independent Test**: View several products as a customer, then call `GET /api/product/recently-viewed` and verify product IDs returned in reverse chronological order.

### Tests for US4

- [x] T034 [P] [US4] Write `TestGetRecentlyViewed_ReturnsProductIDs` (RV-GET-01): customer with 5 views → GET returns 5 IDs, newest first in [`test/integration/product/product/get_product_by_id/recently_viewed_happy_path_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_happy_path_test.go)
- [x] T035 [P] [US4] Write `TestGetRecentlyViewed_Unauthenticated` (RV-GET-02): no auth → 401 in [`test/integration/product/product/get_product_by_id/recently_viewed_sad_path_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_sad_path_test.go)
- [x] T036 [P] [US4] Write `TestGetRecentlyViewed_EmptyList` (RV-GET-03): no views → empty array, not null in [`test/integration/product/product/get_product_by_id/recently_viewed_sad_path_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_sad_path_test.go)
- [x] T037 [P] [US4] Write `TestGetRecentlyViewed_RespectsLimit` (RV-GET-04): limit=5 → at most 5; limit=100 → capped at 50 in [`test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go`](test/integration/product/product/get_product_by_id/recently_viewed_edge_case_test.go)

### Implementation for US4

- [x] T038 [US4] Add `GetRecentlyViewedProducts` handler method: extract userID from context, parse optional limit param (default 10, max 50), call `h.recentlyViewedService.GetRecentlyViewed()`, return product IDs in [`product/handler/product_handler.go`](product/handler/product_handler.go) (depends on T034-T037 tests FAILING first)
- [x] T039 [US4] Register `GET /recently-viewed` route with `CustomerAuth` middleware in [`product/route/product_route.go`](product/route/product_route.go) (depends on T038)

**Checkpoint**: All US4 tests pass. Full feature complete including retrieval endpoint.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and cleanup across all user stories.

- [x] T040 Run full integration test suite and verify all 23 tests pass: `go test ./test/integration/product/product/get_product_by_id/ -run "RecentlyViewed" -v`
- [x] T041 Verify no existing product API tests regressed: `go test ./test/integration/product/product/get_product_by_id/ -v` (all 5 suites pass)
- [x] T042 Verify constitution compliance: modular monolith boundaries ✅, clean architecture layers ✅, factory-singleton DI ✅, RBAC enforcement ✅, backward compatibility ✅
- [x] T043 Migration `025_create_recently_viewed_table.sql` verified via Testcontainers integration tests (23/23 pass) & manual SQL syntax review

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup (migration file) — BLOCKS all user stories
- **Factory Registration (Phase 3)**: Depends on Foundational (entity, repo, service exist) — BLOCKS handler integration
- **Tests (Phase 4)**: Depends on Foundational + Factory (types must compile) — write FIRST, expect FAIL
- **Handler Integration (Phase 5)**: Depends on Tests written + Factory wired — makes tests PASS
- **US4 (Phase 6)**: Depends on Phase 5 (handler has recentlyViewedService field) — optional
- **Polish (Phase 7)**: Depends on all desired phases complete

### User Story Dependencies

- **US1 + US2 + US3 (P1)**: Implemented together in Phases 4-5. They share the same handler code change. No dependencies between them — they are test scenarios for the same implementation.
  - US1 tests: recording works, upsert, trim-to-10
  - US2 tests: role-based skip for seller/admin
  - US3: fire-and-forget is inherent in the service design (error logged, not returned)
- **US4 (P2)**: Depends on Phases 2-3 (service exists) + Phase 5 (handler has service field). Adds new endpoint, not modifying existing behavior.

### Within Each Phase

- Phase 2: T002-T006 are parallelizable [P] (different files). T007 runs after T002. T008 runs after T007.
- Phase 3: T009 → T010 → T011 (sequential chain). T012 runs after T009+T010.
- Phase 4: T014 must run first (setup). T015-T031 are all parallelizable [P] within their test files.
- Phase 5: T032 → T033 (sequential).
- Phase 6: T034-T037 parallel [P] (tests). T038 → T039 (sequential implementation).

### Parallel Opportunities

- All Phase 2 foundational tasks T002-T006 can run in parallel (different files)
- All Phase 4 test tasks T015-T023 can run in parallel (different test scenarios in same file type)
- All Phase 4 test tasks T024-T031 can run in parallel
- Phase 4 happy path, sad path, and edge case files can be written in parallel
- All Phase 6 test tasks T034-T037 can run in parallel

---

## Parallel Example: Phase 2 Foundational

```bash
# Launch all [P] foundational tasks together:
Task: "T002 Define RecentlyViewed entity in product/entity/recently_viewed.go"
Task: "T003 Add error code constants in product/utils/error_constants.go"
Task: "T004 Add message constants in product/utils/message_constants.go"
Task: "T005 Add SellerID field to ProductResponse in product/model/product_model.go"
Task: "T006 Set SellerID in BuildProductResponse in product/factory/product_factory.go"

# Then sequential:
Task: "T007 Create repository in product/repository/recently_viewed_repository.go"
Task: "T008 Create service in product/service/recently_viewed_service.go"
```

## Parallel Example: Phase 4 Tests

```bash
# After T014 (setup) is complete, launch all test files in parallel:
Task: "T015-T018: Happy path tests in recently_viewed_happy_path_test.go"
Task: "T019-T023: Sad path tests in recently_viewed_sad_path_test.go"
Task: "T024-T031: Edge case tests in recently_viewed_edge_case_test.go"
```

---

## Implementation Strategy

### MVP First (US1 + US2 + US3 Only — P1 Stories)

1. Complete Phase 1: Migration (T001)
2. Complete Phase 2: Foundational (T002-T008)
3. Complete Phase 3: Factory Wiring (T009-T012)
4. Complete Phase 4: Write Tests (T013-T031) → Tests FAIL ✓
5. Complete Phase 5: Handler Integration (T032-T033) → Tests PASS ✓
6. **STOP and VALIDATE**: All 17 tests green. Feature delivers automatic recording with role restriction.
7. Deploy/demo if ready.

### Incremental Delivery

1. Setup + Foundational + Factory → Foundation ready
2. Tests + Handler integration → US1/US2/US3 complete (MVP!)
3. Add US4 tests + implementation → Full feature with retrieval endpoint
4. Each phase adds value without breaking previous work

### Parallel Team Strategy

With multiple developers:

1. Team completes Phase 1-3 together (sequential chain)
2. Once Phase 3 is done:
   - Developer A: Phase 4 Happy Path tests (T015-T018)
   - Developer B: Phase 4 Sad Path tests (T019-T023)
   - Developer C: Phase 4 Edge Case tests (T024-T031)
3. Developer A: Phase 5 handler integration (T032-T033)
4. All: Phase 6 US4 tasks (if desired)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- US1, US2, US3 share the same production code — separated into test scenarios only
- Integration tests use Testcontainers — ensure Docker is running
- Tests follow `testify/suite` pattern with `SetupSuite`/`TearDownSuite`/`SetupTest`
- `SetupTest` clears `user_recently_viewed` table before each test for isolation
- API endpoints use constants for URLs (follow constitution pattern)
- `json:"-"` tag on `SellerID` field prevents leaking internal data to API responses
- Fire-and-forget: errors logged via `log.Errorf`, never returned to caller
- Commit after each phase or logical task group
- Stop at any checkpoint to validate independently
