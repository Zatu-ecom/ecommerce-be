# Tasks: Product File Integration

**Input**: Design documents from `/specs/005-product-file-integration/`  
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/product-media.openapi.yaml`, `quickstart.md`

**Tests**: Required by `spec.md` (Verification & Testing), the feature plan, quickstart, and project constitution. Integration tests must be written first, fail before implementation, and use `test/integration/setup` + `test/integration/helpers.APIClient` against `SetupTestServer` (real HTTP stack and database — same pattern as `test/integration/product/product/create_product/create_product_test.go`). Do not mock Gin, middleware, or GORM for primary scenarios. Add **unit tests** only for pure helpers (for example mapping) when an API test cannot meaningfully cover the branch; keep overall coverage aligned with all acceptance scenarios. Assert via API responses and follow-up GETs per API-first testing rules.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches a different file and does not depend on incomplete tasks.
- **[Story]**: Which user story the task belongs to, using `US1`, `US2`, or `US3`.
- Every task includes an exact file path.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare database and shared product-media scaffolding needed by every user story.

- [ ] T001 Create migration for `product_media` table with `product_id`, `file_id`, `is_primary`, `display_order`, timestamps, product cascade foreign key, `product_id` index, and `(product_id, file_id)` unique constraint in `migrations/019_create_product_media_table.sql`
- [ ] T002 [P] Add Product Media entity with table mapping and timestamp fields in `product/entity/product_media.go`
- [ ] T003 [P] Add product media request/response DTOs and additive `Media []ProductMediaResponse` field on product responses in `product/model/product_model.go`
- [ ] T004 [P] Add product media application errors for duplicate link, missing link, invalid file reference, and cleanup degradation handling in `product/error/product_media_error.go`
- [ ] T005 [P] Add shared integration test suite scaffolding, endpoint constants, and helper methods for Product media tests in `test/integration/product/product_media/setup_suite_test.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core repository, service contracts, and wiring that must be complete before user stories can be implemented.

**Critical**: No user story work can begin until this phase is complete.

- [ ] T006 Define `ProductMediaRepository` interface and GORM implementation methods for create, find by product/file, find by product IDs, update metadata, unset primary, promote fallback primary, and delete in `product/repository/product_media_repository.go`
- [ ] T007 Add repository factory constructor/cache method for `ProductMediaRepository` in `product/factory/singleton/repository_factory.go`
- [ ] T008 Define Product-owned File gateway interfaces for read, batch read with download URLs/variants, and delete operations without exposing File repositories in `product/service/product_file_gateway.go`
- [ ] T009 Add product media service interface, constructor dependencies, and shared mapping helpers for Product Media plus File metadata to Product Media DTOs in `product/service/product_media_service.go`
- [ ] T010 Update service factory wiring to inject `ProductMediaRepository`, Product File gateway dependencies, and create `ProductMediaService` in `product/factory/singleton/service_factory.go`
- [ ] T011 Update handler factory wiring to expose a product handler with Product media service dependencies in `product/factory/singleton/handler_factory.go`
- [ ] T012 Update Product handler struct constructor signatures to accept Product media service without adding endpoint logic yet in `product/handler/product_handler.go`
- [ ] T013 Add Product media route path constants for `/media` and `/:fileId` segments in `product/utils/constants.go`

**Checkpoint**: Foundation ready. User story implementation can now begin.

---

## Phase 3: User Story 1 - View Product Media (Priority: P1) MVP

**Goal**: Product detail and listing responses include ordered storefront-ready media summaries, while products with no media or missing file data still load.

**Independent Test**: Attach or seed media links for products, call product detail and product list APIs, and confirm media is returned in display order with URLs/thumbnails where available and empty media when none exists.

### Tests for User Story 1

- [ ] T014 [P] [US1] Write failing integration tests for product detail returning ordered media, thumbnail fallback, empty media collection, and missing/inaccessible file resilience in `test/integration/product/product_media/get_product_media_test.go`
- [ ] T015 [P] [US1] Write failing integration tests for product list returning media for all products on the page and avoiding per-product media lookup behavior in `test/integration/product/product_media/list_products_media_test.go`

### Implementation for User Story 1

- [ ] T016 [US1] Implement batch lookup methods for Product Media by product IDs with stable `display_order ASC, id ASC` ordering in `product/repository/product_media_repository.go`
- [ ] T017 [US1] Implement Product media DTO mapping, thumbnail/poster variant selection, fallback URL behavior, and missing File data skipping in `product/service/product_media_service.go`
- [ ] T018 [US1] Extend Product query/detail service flow to load media for a single product and batches of listed products without N+1 File calls in `product/service/product_query_service.go`
- [ ] T019 [US1] Populate the additive `media` field while preserving existing response fields in Product response builders in `product/service/product_service.go`
- [ ] T020 [US1] Ensure Product detail and list handlers return the updated Product responses through existing response helpers in `product/handler/product_handler.go`
- [ ] T021 [US1] Run Product media read/list integration tests and fix failures in `test/integration/product/product_media/get_product_media_test.go` and `test/integration/product/product_media/list_products_media_test.go`

**Checkpoint**: User Story 1 is independently functional and testable as the MVP.

---

## Phase 4: User Story 2 - Manage Product Media Links (Priority: P2)

**Goal**: Authorized seller/admin users can attach already uploaded files to products, set the primary media item, and update display order.

**Independent Test**: Link an uploaded file to a product, update primary/order metadata, verify duplicate links are rejected, and confirm product reads reflect changes immediately.

### Tests for User Story 2

- [ ] T022 [P] [US2] Write failing integration tests for attach media happy path, missing correlation ID, auth failures, invalid product, invalid/inaccessible file, duplicate link, and primary reset in `test/integration/product/product_media/attach_media_test.go`
- [ ] T023 [P] [US2] Write failing integration tests for update media metadata, primary reset, missing link, invalid payload, and seller isolation in `test/integration/product/product_media/update_media_test.go`

### Implementation for User Story 2

- [ ] T024 [US2] Implement attach media service flow with product existence check, seller scope check, File gateway validation, duplicate handling, primary reset, create link, and DTO return in `product/service/product_media_service.go`
- [ ] T025 [US2] Implement update media metadata service flow with link verification, optional primary reset, optional display order update, seller isolation, and DTO return in `product/service/product_media_service.go`
- [ ] T026 [US2] Add handler methods for `POST /api/product/:productId/media` and `PATCH /api/product/:productId/media/:fileId` with request binding, path parsing, principal extraction, and standardized responses in `product/handler/product_handler.go`
- [ ] T027 [US2] Register seller-protected attach and update media routes in `product/route/product_route.go`
- [ ] T028 [US2] Add validation tags and helper validation for `fileId`, `isPrimary`, and `displayOrder` request fields in `product/model/product_model.go`
- [ ] T029 [US2] Run Product media attach/update integration tests and fix failures in `test/integration/product/product_media/attach_media_test.go` and `test/integration/product/product_media/update_media_test.go`

**Checkpoint**: User Stories 1 and 2 both work independently.

---

## Phase 5: User Story 3 - Remove Product Media (Priority: P3)

**Goal**: Authorized seller/admin users can remove product media, product responses stop showing it, primary fallback is assigned when needed, and underlying File cleanup is attempted without breaking product correctness.

**Independent Test**: Remove a linked media item, confirm Product no longer references it, confirm primary fallback promotion, and confirm File cleanup failure still returns successful product-media removal.

### Tests for User Story 3

- [ ] T030 [P] [US3] Write failing integration tests for remove media happy path, missing link, auth failures, seller isolation, primary fallback promotion, and product response cleanup in `test/integration/product/product_media/remove_media_test.go`
- [ ] T031 [US3] Write failing integration test for best-effort File cleanup failure after unlink while returning `204 No Content` in `test/integration/product/product_media/remove_media_test.go`

### Implementation for User Story 3

- [ ] T032 [US3] Implement remove media service flow with link verification, delete link, fallback primary promotion, File gateway delete attempt, and cleanup failure logging in `product/service/product_media_service.go`
- [ ] T033 [US3] Add handler method for `DELETE /api/product/:productId/media/:fileId` with path parsing, principal extraction, and `204 No Content` response in `product/handler/product_handler.go`
- [ ] T034 [US3] Register seller-protected delete media route in `product/route/product_route.go`
- [ ] T035 [US3] Ensure Product deletion still cascades Product Media rows (via the `ON DELETE CASCADE` FK added in the migration for `product_media.product_id → products.id`) and does not call File repositories directly in `product/service/product_service.go`
- [ ] T036 [US3] Run Product media remove integration tests and fix failures in `test/integration/product/product_media/remove_media_test.go`

**Checkpoint**: All user stories are independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Verify contracts, regression safety, and maintainability across all user stories.

- [ ] T037 [P] Update API contract examples if implementation response details changed in `specs/005-product-file-integration/contracts/product-media.openapi.yaml`
- [ ] T038 [P] Update quickstart verification notes with final route names, required headers, and expected status codes in `specs/005-product-file-integration/quickstart.md`
- [ ] T039 Add or update constants for Product media success messages, error codes, and route helper strings in `product/utils/success_constants.go` and `product/utils/constants.go`
- [ ] T040 Run `go test ./test/integration/product/product_media/...` and fix failures in `test/integration/product/product_media`
- [ ] T041 Run relevant regression tests for Product and File modules with `go test ./test/integration/product/... ./test/integration/file/...` and fix failures in affected files
- [ ] T042 Run formatting and static checks for touched Go files under `product` and `test/integration/product/product_media`, then fix issues in affected files
- [ ] T043 Verify no Product implementation imports File repositories or File persistence entities by reviewing imports in `product/service`, `product/repository`, and `product/handler`
- [ ] T044 Validate `tasks.md` implementation completion against the contract and quickstart in `specs/005-product-file-integration/tasks.md`
- [ ] T045 Execute manual UAT/release-gate checklist for SC-001 (product detail media correctness ≥95%), SC-002 (product listing media accuracy ≥95%), and SC-003 (attach/reorder/primary/remove cycle under 2 minutes for 10-media product); document pass/fail before release sign-off in `specs/005-product-file-integration/checklists/requirements.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; T002, T003, T004, and T005 can run in parallel after T001 is understood.
- **Foundational (Phase 2)**: Depends on Phase 1; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Phase 2; recommended MVP.
- **User Story 2 (Phase 4)**: Depends on Phase 2, but benefits from US1 mapping because attach/update responses reuse Product media DTO mapping.
- **User Story 3 (Phase 5)**: Depends on Phase 2, but benefits from US1 mapping and US2 link management behavior.
- **Polish (Phase 6)**: Depends on whichever user stories are implemented.

### User Story Dependencies

- **US1 - View Product Media**: Can start after Foundational and is the MVP.
- **US2 - Manage Product Media Links**: Can start after Foundational; for easiest validation, complete after US1 so product reads verify link changes.
- **US3 - Remove Product Media**: Can start after Foundational; for easiest validation, complete after US2 so media links can be created through the API first.

### Within Each User Story

- Write integration tests first and verify they fail.
- Implement repository/service behavior before handlers.
- Register routes after handler methods exist.
- Run story-specific integration tests before moving to the next story.

---

## Parallel Execution Examples

### User Story 1

```text
Task: T014 [US1] Write product detail media integration tests in test/integration/product/product_media/get_product_media_test.go
Task: T015 [US1] Write product list media integration tests in test/integration/product/product_media/list_products_media_test.go
```

### User Story 2

```text
Task: T022 [US2] Write attach media integration tests in test/integration/product/product_media/attach_media_test.go
Task: T023 [US2] Write update media integration tests in test/integration/product/product_media/update_media_test.go
```

### User Story 3

```text
Task: T030 [US3] Write remove media integration tests in test/integration/product/product_media/remove_media_test.go
Task: T031 [US3] Write cleanup failure integration test in test/integration/product/product_media/remove_media_test.go
```

---

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete Phase 3 for User Story 1.
3. Validate product detail/list media responses independently.
4. Stop and demo the read-only Product media experience if needed.

### Incremental Delivery

1. Deliver US1 to make media visible on product detail/list responses.
2. Deliver US2 to allow seller/admin media attachment and ordering.
3. Deliver US3 to complete media removal and cleanup behavior.
4. Run cross-module regression tests after each story.

### Validation Rules

- Every task follows `- [ ] T### [P?] [US?] Description with file path`.
- Setup, Foundational, and Polish tasks do not include story labels.
- Story tasks include `US1`, `US2`, or `US3`.
- Tests precede implementation tasks within each story.
- Product/File module boundary must remain intact.
