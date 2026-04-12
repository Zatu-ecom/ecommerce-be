# Tasks: Storage Config Activation and Listing

**Input**: Design documents from `/specs/001-activate-storage-config/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are required for this feature because the specification explicitly mandates comprehensive scenario coverage and TDD-style implementation.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare feature workspace and test structure for incremental delivery.

- [x] T001 Create and baseline-check feature artifacts in `specs/001-activate-storage-config/plan.md`, `specs/001-activate-storage-config/spec.md`, and `specs/001-activate-storage-config/contracts/storage-config.openapi.yaml`
- [x] T002 [P] Split file-config integration coverage into focused sections (or companion files) under `test/integration/file/config_test.go`
- [x] T003 [P] Add endpoint constants for activation/listing test paths in `test/integration/file/config_test.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core plumbing that all user stories depend on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T004 Add/adjust file-config error codes and messages for activation/listing flows in `file/utils/constant/config_constants.go`
- [x] T005 Extend module AppError definitions for listing/activation validation and internal failures in `file/error/config_errors.go`
- [x] T006 Add list-query/list-filter/list-response and activation-response DTOs in `file/model/config_model.go`
- [x] T007 Extend repository interface contracts for scope-aware list and activation operations in `file/repository/config_repository.go`
- [x] T008 Extend config service interface contracts for activate/list operations in `file/service/config_service.go`
- [x] T009 Wire route updates for `GET /storage-config` and `POST /storage-config/:id/activate` in `file/route/storage_config_routes.go`
- [x] T010 Add handler method signatures and shared parsing/validation entry points in `file/handler/config_handler.go`

**Checkpoint**: Foundation ready - user story implementation can now begin.

---

## Phase 3: User Story 1 - Activate a storage configuration (Priority: P1) 🎯 MVP

**Goal**: Implement scope-safe activation with idempotency, authorization, and concurrency-safe single-active behavior.

**Independent Test**: `POST /api/files/storage-config/{id}/activate` passes success, invalid-id, forbidden, unauthenticated, not-found, idempotent, and concurrency scenarios.

### Tests for User Story 1

- [x] T011 [P] [US1] Add activation happy-path/not-found/forbidden/unauthenticated/invalid-id integration tests in `test/integration/file/config_test.go`
- [x] T012 [P] [US1] Add activation idempotency and concurrent-activation convergence tests in `test/integration/file/config_test.go`

### Implementation for User Story 1

- [x] T013 [US1] Implement scope-aware activation repository transaction (single-active convergence) in `file/repository/config_repository.go`
- [x] T014 [US1] Implement activation orchestration and scope authorization in `file/service/config_service.go`
- [x] T015 [US1] Implement `ActivateConfig` handler path-param parsing and response mapping in `file/handler/config_handler.go`
- [x] T016 [US1] Add activation response mapping helper(s) in `file/model/config_model.go`
- [x] T017 [US1] Align activation success/failure response messages and codes in `file/utils/constant/config_constants.go`

**Checkpoint**: User Story 1 is independently functional and testable (MVP candidate).

---

## Phase 4: User Story 2 - List storage configurations with role-aware scoping (Priority: P1)

**Goal**: Provide listing endpoint with token-derived seller/platform scope and strict access enforcement.

**Independent Test**: `GET /api/files/storage-config` returns seller-only data in seller context, platform-only data without seller context, empty-result success, and rejects below-seller access.

### Tests for User Story 2

- [x] T018 [P] [US2] Add list scope/auth coverage tests (seller, platform, empty, below-seller/unauthenticated) in `test/integration/file/config_test.go`

### Implementation for User Story 2

- [x] T019 [US2] Implement scope-aware list query (owner scope + pagination/sort baseline) in `file/repository/config_repository.go`
- [x] T020 [US2] Implement list-scope resolution and authorization in `file/service/config_service.go`
- [x] T021 [US2] Implement listing handler logic in `file/handler/config_handler.go`
- [x] T022 [US2] Replace legacy active-config GET route wiring with listing route wiring in `file/route/storage_config_routes.go`

**Checkpoint**: User Story 2 is independently functional and testable.

---

## Phase 5: User Story 3 - Filter storage configurations effectively (Priority: P1)

**Goal**: Implement complete filter semantics (multi-value + single-value), forbidden filter handling, and list validation behavior.

**Independent Test**: List endpoint correctly applies `ids/providerIds/validationStatuses/isActive/isDefault/adapterType/search`, rejects forbidden `sellerId`, and validates invalid pagination/sort/filter values.

### Tests for User Story 3

- [x] T023 [P] [US3] Add integration tests for multi-value, single-value, and combined filter behavior in `test/integration/file/config_test.go`
- [x] T024 [P] [US3] Add integration tests for invalid filter/sort/pagination and forbidden `sellerId` query in `test/integration/file/config_test.go`

### Implementation for User Story 3

- [x] T025 [US3] Implement query-param parsing and normalized filter conversion in `file/model/config_model.go`
- [x] T026 [US3] Implement reusable comma-separated filter parsing helpers in `file/utils/filter_utils.go`
- [x] T027 [US3] Implement repository filter predicates for list-capable and single-value filters in `file/repository/config_repository.go`
- [x] T028 [US3] Enforce forbidden `sellerId` query handling and validation mapping in `file/handler/config_handler.go`
- [x] T029 [US3] Finalize filter-level validation and error propagation in `file/service/config_service.go`

**Checkpoint**: User Story 3 is independently functional and testable.

---

## Phase 6: User Story 4 - Confidence through complete automated tests (Priority: P2)

**Goal**: Complete robust test coverage and deterministic error-shape validation for long-term regression safety.

**Independent Test**: Running the file config suite validates all required positive/negative scenarios and standardized error response contracts.

### Tests for User Story 4

- [x] T030 [P] [US4] Add standardized error-schema assertions for activation/listing failures in `test/integration/file/config_test.go`
- [x] T031 [P] [US4] Add service-level tests for internal dependency failure mapping in `file/service/config_service_test.go`

### Implementation for User Story 4

- [x] T032 [US4] Normalize unexpected-error translation paths in `file/service/config_service.go`
- [x] T033 [US4] Ensure handler error mapping remains consistent for activation/listing paths in `file/handler/config_handler.go`

**Checkpoint**: User Story 4 regression protection is independently verifiable.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final consistency, documentation alignment, and full verification.

- [x] T034 [P] Update API contract examples and response details in `specs/001-activate-storage-config/contracts/storage-config.openapi.yaml`
- [x] T035 [P] Reconcile implementation notes and verification steps in `specs/001-activate-storage-config/quickstart.md`
- [x] T036 Execute focused file integration suite and capture outcomes in `test/integration/file/config_test.go`
- [x] T037 Execute full regression run and resolve residual failures in `file/handler/config_handler.go`, `file/service/config_service.go`, and `file/repository/config_repository.go`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies.
- **Phase 2 (Foundational)**: Depends on Phase 1 completion; blocks all user stories.
- **Phase 3 (US1)**: Depends on Phase 2.
- **Phase 4 (US2)**: Depends on Phase 2; can proceed in parallel with US1 once foundation is complete.
- **Phase 5 (US3)**: Depends on Phase 2 and should integrate on top of US2 listing baseline.
- **Phase 6 (US4)**: Depends on Phases 3-5 behavior being implemented.
- **Phase 7 (Polish)**: Depends on all prior phases.

### User Story Dependencies

- **US1 (P1)**: Independent after foundation.
- **US2 (P1)**: Independent after foundation.
- **US3 (P1)**: Depends on US2 listing pipeline.
- **US4 (P2)**: Depends on US1-US3 for full coverage hardening.

### Within Each User Story

- Tests first, then repository/service/handler/model refinements.
- Repository and service changes precede final handler wiring for endpoint behavior.
- Complete story checkpoint before moving to lower-priority scope.

### Parallel Opportunities

- Phase 1 tasks T002-T003 can run in parallel.
- Phase 2 tasks T004-T006 can run in parallel; T007-T010 depend on their outputs.
- US1 tests T011/T012 can run in parallel.
- US3 tests T023/T024 can run in parallel.
- US4 tests T030/T031 can run in parallel.
- Polish docs tasks T034/T035 can run in parallel.

---

## Parallel Example: User Story 1

```bash
# Parallel test authoring for activation behavior
Task: "T011 [US1] Add activation happy-path/not-found/forbidden/unauthenticated/invalid-id integration tests in test/integration/file/config_test.go"
Task: "T012 [US1] Add activation idempotency and concurrent-activation convergence tests in test/integration/file/config_test.go"
```

## Parallel Example: User Story 2

```bash
# Parallelizable after foundational plumbing exists
Task: "T018 [US2] Add list scope/auth coverage tests in test/integration/file/config_test.go"
Task: "T019 [US2] Implement scope-aware list query in file/repository/config_repository.go"
```

## Parallel Example: User Story 3

```bash
# Parallel filter test authoring
Task: "T023 [US3] Add integration tests for multi-value/single-value/combined filters in test/integration/file/config_test.go"
Task: "T024 [US3] Add integration tests for invalid and forbidden filters in test/integration/file/config_test.go"
```

## Parallel Example: User Story 4

```bash
# Parallel robustness verification
Task: "T030 [US4] Add standardized error-schema assertions in test/integration/file/config_test.go"
Task: "T031 [US4] Add service-level internal failure mapping tests in file/service/config_service_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Finish Phase 1 and Phase 2.
2. Deliver Phase 3 (US1 activation end-to-end).
3. Validate US1 independently via activation test set.

### Incremental Delivery

1. Add US2 listing scope behavior.
2. Add US3 filter behavior.
3. Add US4 comprehensive regression hardening.
4. Execute Polish phase and full regression.

### Parallel Team Strategy

1. One engineer drives foundational plumbing (Phase 2).
2. After foundation:
   - Engineer A: US1 activation flow.
   - Engineer B: US2 listing flow.
3. Engineer C can prepare US3 filter tests in parallel once listing baseline is merged.
4. US4 coverage hardening runs after feature-complete behavior.
