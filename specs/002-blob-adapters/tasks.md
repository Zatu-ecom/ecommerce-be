# Tasks: BlobAdapter Layer for Multi-Cloud File Storage

**Input**: Design documents from `/specs/002-blob-adapters/` (`plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/blob-adapter-contract.md`, `quickstart.md`)

**Tests**: **REQUIRED** by spec (`FR-014`..`FR-015`) — integration tests per provider + unit tests for factory dispatch/decryption.

**Organization**: Tasks are grouped by user story so each story can be implemented and tested independently.

## Format: `- [ ] T### [P?] [US?] Description (with file paths)`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[US#]**: Which user story this task belongs to

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Ensure repo has required deps and test scaffolding paths

- [X] T001 Confirm module paths exist (`file/`, `test/integration/`) and create missing directories for this feature (`file/service/blob_adapter/`, `test/integration/file/`, `test/integration/setup/`)
- [X] T002 Add/verify Go dependencies in `go.mod` for AWS SDK v2 S3, GCS client, Azure Blob SDK, and Testcontainers (as required by `specs/002-blob-adapters/plan.md`)
- [X] T003 [P] Add feature-local README notes for running integration tests in `specs/002-blob-adapters/quickstart.md` if any repo-specific env vars/commands differ

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared contracts, models, error taxonomy, and testcontainer helpers used by all stories

**⚠️ CRITICAL**: No user story implementation should begin until this phase is complete.

- [X] T004 Create `BlobAdapter` interface in `file/service/blob_adapter/adapter.go` (exact 7 methods per `contracts/blob-adapter-contract.md`)
- [X] T005 Create operation DTOs in `file/model/blob_adapter_model.go` (`BlobPutObjectInput/Output`, `BlobObjectMeta`, `BlobPresign*`, `BlobCopyObjectInput`)
- [X] T006 Implement categorized error sentinels in `file/error/blob_adapter_errors.go` using `commonError.AppError` pattern (`ErrBlobNotFound|ErrBlobPermissionDenied|ErrBlobNetwork|ErrBlobValidation|ErrBlobInternal|ErrBlobFactoryInit`); constants in `file/utils/constant/config_constants.go`; `IsBlobError` helper for test assertions
- [X] T007 Add factory skeleton + public constructor API in `file/service/blob_adapter/factory.go` (no provider logic yet; just exported entrypoint + parameter validation surface; import `StorageConfig` from `ecommerce-be/file/entity` — NOT from `file/model` which is the HTTP DTO layer)
- [X] T008 [P] Add MinIO testcontainer helper in `test/integration/file/minio_container.go` (start/stop, endpoint/creds, bucket bootstrap)
- [X] T009 [P] Add Fake-GCS-Server testcontainer helper in `test/integration/file/fake_gcs_container.go` (start/stop, endpoint, bucket bootstrap)
- [X] T010 [P] Add Azurite testcontainer helper in `test/integration/file/azurite_container.go` (start/stop, connection string/account key bootstrap, container bootstrap)
- [X] T011 Add shared blob integration utilities in `test/integration/file/blob_test_helpers.go` (random key helpers, content generators, TTL helpers, context deadline helpers)

**Checkpoint**: Foundation ready — user story work can now proceed.

---

## Phase 3: User Story 1 — S3-Compatible Blob Operations (Priority: P1) 🎯 MVP

**Goal**: Provide a working S3-compatible adapter implementing all seven `BlobAdapter` methods.

**Independent Test**: `go test ./test/integration/file/... -run S3 -v -count=1` passes against a MinIO container (no GCS/Azure required).

### Tests for User Story 1 (REQUIRED)

- [X] T012 [US1] Create S3 integration suite in `test/integration/file/blob_adapter_s3_integration_test.go` covering all 7 methods against MinIO
- [X] T013 [US1] Add invalid-credential test cases in `test/integration/file/blob_adapter_s3_integration_test.go` (factory/adapter returns categorized provider error, no secrets leaked)
- [X] T014 [US1] Add context-deadline tests for IO-heavy methods in `test/integration/file/blob_adapter_s3_integration_test.go` (`PutObject` and `GetObjectStream`: cancel mid-operation returns deadline/cancel error; stream body closed without goroutine/connection leak)
- [X] T014b [US1] Add context-deadline tests for metadata/presign methods in `test/integration/file/blob_adapter_s3_integration_test.go` (already-cancelled context passed to `HeadObject`, `PresignUpload`, `PresignDownload`, `CopyObject`, `DeleteObject` each return error before SDK call)

### Implementation for User Story 1

- [X] T015 [US1] Implement S3 adapter struct + constructor in `file/service/blob_adapter/s3_compatible_adapter.go` (AWS SDK v2, endpoint override, force-path-style, region)
- [X] T016 [US1] Implement `PutObject` in `file/service/blob_adapter/s3_compatible_adapter.go` (content-type/length, return `ETag` + key)
- [X] T017 [US1] Implement `HeadObject` in `file/service/blob_adapter/s3_compatible_adapter.go` (map metadata to `ObjectMeta`, not-found mapping)
- [X] T018 [US1] Implement `GetObjectStream` in `file/service/blob_adapter/s3_compatible_adapter.go` (return `io.ReadCloser` + `ObjectMeta`, ensure caller closes)
- [X] T019 [US1] Implement `PresignUpload` in `file/service/blob_adapter/s3_compatible_adapter.go` (validate TTL > 0, presign PUT)
- [X] T020 [US1] Implement `PresignDownload` in `file/service/blob_adapter/s3_compatible_adapter.go` (validate TTL > 0, presign GET)
- [X] T021 [US1] Implement `CopyObject` in `file/service/blob_adapter/s3_compatible_adapter.go` (intra-account copy, not-found mapping)
- [X] T022 [US1] Implement `DeleteObject` in `file/service/blob_adapter/s3_compatible_adapter.go` (idempotent semantics + categorized errors)
- [X] T023 [US1] Implement S3 error mapping helpers in `file/service/blob_adapter/s3_compatible_adapter.go` (map SDK errors to categories without leaking credentials)

**Checkpoint**: US1 complete — S3 adapter passes integration suite and can be used directly.

---

## Phase 4: User Story 2 — Factory Resolves Adapter by Provider Type (Priority: P1)

**Goal**: Factory returns correct adapter instance based on `StorageConfig.Provider.AdapterType`.

**Independent Test**: Unit tests prove dispatch behavior for `s3_compatible|gcs|azure|unknown` without requiring integration containers.

### Tests for User Story 2 (REQUIRED)

- [X] T024 [US2] Create factory dispatch unit tests in `file/service/blob_adapter/factory_test.go` (s3_compatible -> S3 adapter, gcs -> GCS adapter, azure -> Azure adapter, unknown -> error)
- [X] T025 [US2] Add factory input validation unit tests in `file/service/blob_adapter/factory_test.go` (missing provider/type/config returns `validation` error)

### Implementation for User Story 2

- [X] T026 [US2] Implement adapter-type dispatch in `file/service/blob_adapter/factory.go` (`s3_compatible|gcs|azure`, unknown -> categorized error)
- [X] T027 [US2] Define internal credential schema structs in `file/service/blob_adapter/factory.go` (S3, GCS, Azure decrypted payloads per `data-model.md`)
- [X] T028 [US2] Implement per-adapter credential validation in `file/service/blob_adapter/factory.go` (fail fast before any network call)

**Checkpoint**: US2 complete — factory dispatch works and is fully unit-tested.

---

## Phase 5: User Story 5 — Credential Decryption Integration (Priority: P1)

**Goal**: Factory decrypts `StorageConfig.CredentialsEncrypted` before adapter construction; callers never handle raw secrets.

**Independent Test**: Given an encrypted credential payload, factory can construct S3 adapter; wrong key returns decryption error (no providers required for wrong-key test).

### Tests for User Story 5 (REQUIRED)

- [X] T029 [US5] Add decryption success/failure unit tests in `test/integration/file/blob_adapter_factory_test.go` (success path produces validated credentials; wrong key -> categorized decryption error)
- [X] T030 [US5] Add no-secret-leak assertions in `test/integration/file/blob_adapter_factory_test.go` (error strings must not contain access keys/service-account JSON/account keys)

### Implementation for User Story 5

- [X] T031 [US5] Wire decryption into factory in `file/service/blob_adapter/factory.go` using existing AES envelope helper referenced in `specs/002-blob-adapters/spec.md` assumptions (`common/helper/crypto.go`)
- [X] T032 [US5] Ensure factory returns categorized `validation/internal` error on decrypt/parse failure in `file/service/blob_adapter/factory.go`
- [X] T033 [US5] Add JSON parsing + strict field extraction for decrypted credential blobs in `file/service/blob_adapter/factory.go` (no logging of raw payload)

**Checkpoint**: US5 complete — decryption is centralized and safe.

---

## Phase 6: User Story 3 — GCS Blob Operations (Priority: P2)

**Goal**: Provide a working GCS adapter implementing all seven `BlobAdapter` methods.

**Independent Test**: `go test ./test/integration/file/... -run GCS -v -count=1` passes against Fake-GCS-Server container.

### Tests for User Story 3 (REQUIRED)

- [X] T034 [US3] Create GCS integration suite in `test/integration/file/blob_adapter_gcs_integration_test.go` covering all 7 methods against Fake-GCS-Server
- [X] T035 [US3] Add invalid-credential test cases in `test/integration/file/blob_adapter_gcs_integration_test.go` (categorized provider error, no secrets leaked)
- [X] T036 [US3] Add context-deadline tests for IO-heavy methods in `test/integration/file/blob_adapter_gcs_integration_test.go` (`PutObject` and `GetObjectStream`: cancel mid-operation returns deadline/cancel error; stream body closed without goroutine/connection leak)
- [X] T036b [US3] Add context-deadline tests for metadata/presign methods in `test/integration/file/blob_adapter_gcs_integration_test.go` (already-cancelled context passed to `HeadObject`, `PresignUpload`, `PresignDownload`, `CopyObject`, `DeleteObject` each return error before SDK call)

### Implementation for User Story 3

- [X] T037 [US3] Implement GCS adapter struct + constructor in `file/service/blob_adapter/gcs_adapter.go` (client init from decrypted service-account JSON, bucket handling)
- [X] T038 [US3] Implement `PutObject` in `file/service/blob_adapter/gcs_adapter.go`
- [X] T039 [US3] Implement `HeadObject` in `file/service/blob_adapter/gcs_adapter.go`
- [X] T040 [US3] Implement `GetObjectStream` in `file/service/blob_adapter/gcs_adapter.go`
- [X] T041 [US3] Implement `PresignUpload` in `file/service/blob_adapter/gcs_adapter.go` (validate TTL > 0, signed URL strategy)
- [X] T042 [US3] Implement `PresignDownload` in `file/service/blob_adapter/gcs_adapter.go` (validate TTL > 0)
- [X] T043 [US3] Implement `CopyObject` in `file/service/blob_adapter/gcs_adapter.go`
- [X] T044 [US3] Implement `DeleteObject` in `file/service/blob_adapter/gcs_adapter.go`
- [X] T045 [US3] Implement GCS error mapping helpers in `file/service/blob_adapter/gcs_adapter.go` (categories, no secret leakage)

**Checkpoint**: US3 complete — GCS adapter passes integration suite.

---

## Phase 7: User Story 4 — Azure Blob Operations (Priority: P2)

**Goal**: Provide a working Azure Blob adapter implementing all seven `BlobAdapter` methods.

**Independent Test**: `go test ./test/integration/file/... -run Azure -v -count=1` passes against Azurite container.

### Tests for User Story 4 (REQUIRED)

- [X] T046 [US4] Create Azure integration suite in `test/integration/file/blob_adapter_azure_integration_test.go` covering all 7 methods against Azurite
- [X] T047 [US4] Add invalid-credential test cases in `test/integration/file/blob_adapter_azure_integration_test.go` (categorized provider error, no secrets leaked)
- [X] T048 [US4] Add context-deadline tests for IO-heavy methods in `test/integration/file/blob_adapter_azure_integration_test.go` (`PutObject` and `GetObjectStream`: cancel mid-operation returns deadline/cancel error; stream body closed without goroutine/connection leak)
- [X] T048b [US4] Add context-deadline tests for metadata/presign methods in `test/integration/file/blob_adapter_azure_integration_test.go` (already-cancelled context passed to `HeadObject`, `PresignUpload`, `PresignDownload`, `CopyObject`, `DeleteObject` each return error before SDK call)

### Implementation for User Story 4

- [X] T049 [US4] Implement Azure adapter struct + constructor in `file/service/blob_adapter/azure_blob_adapter.go` (account-name + key/SAS from factory)
- [X] T050 [US4] Implement `PutObject` in `file/service/blob_adapter/azure_blob_adapter.go`
- [X] T051 [US4] Implement `HeadObject` in `file/service/blob_adapter/azure_blob_adapter.go`
- [X] T052 [US4] Implement `GetObjectStream` in `file/service/blob_adapter/azure_blob_adapter.go`
- [X] T053 [US4] Implement `PresignUpload` in `file/service/blob_adapter/azure_blob_adapter.go` (validate TTL > 0, SAS URL generation)
- [X] T054 [US4] Implement `PresignDownload` in `file/service/blob_adapter/azure_blob_adapter.go` (validate TTL > 0, SAS URL generation)
- [X] T055 [US4] Implement `CopyObject` in `file/service/blob_adapter/azure_blob_adapter.go`
- [X] T056 [US4] Implement `DeleteObject` in `file/service/blob_adapter/azure_blob_adapter.go`
- [X] T057 [US4] Implement Azure error mapping helpers in `file/service/blob_adapter/azure_blob_adapter.go` (categories, no secret leakage)

**Checkpoint**: US4 complete — Azure adapter passes integration suite.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Wiring, regressions, and hardening across all stories

- [X] T058 [P] Add package-level compile-only assertions in `file/service/blob_adapter/adapter_test.go` (each concrete type implements `BlobAdapter`) (validates SC-001)
- [X] T058b [P] Add godoc comments to all exported symbols per constitution commenting standard: `BlobAdapter` interface + all 7 method signatures in `adapter.go`; all input/output structs in `file/model/blob_adapter_model.go`; all sentinel errors + `IsBlobError` helper in `file/error/blob_adapter_errors.go`; `NewAdapterFromConfig` constructor in `factory.go` explaining preconditions (`cfg.Provider` must be preloaded) and error categories returned
- [X] T059 Update file module wiring to construct adapters via factory in `file/factory/singleton/service_factory.go` (no handler-layer dependency) (satisfies SC-003 factory wiring)
- [X] T060 [P] Ensure no import cycles / layering violations by running `go test ./...` and fixing any package boundary issues (satisfies SC-006)
- [X] T061 [P] Run full integration suite `go test ./test/integration/file/... -v -count=1` and address flakiness/timeouts in `test/integration/file/*_container.go` (validates SC-001, SC-002, SC-005)
- [X] T062 [P] Ensure all adapter/factory errors are sanitized (no secrets) by adding string-scan test helpers in `test/integration/file/blob_test_helpers.go` (validates SC-004)
- [X] T063 [P] Update feature docs with final run commands + env var list in `specs/002-blob-adapters/quickstart.md` (general hygiene)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: start immediately
- **Foundational (Phase 2)**: depends on Setup; **blocks all user stories**
- **User Stories (Phases 3–7)**: all depend on Foundational
- **Polish (Phase 8)**: depends on desired user stories being complete

### User Story Dependencies

- **US1 (P1)**: depends on Foundational only; can ship as MVP
- **US2 (P1)**: depends on Foundational; independent of US1 (but will reuse S3 constructor once implemented)
- **US5 (P1)**: depends on US2 factory surface + Foundational crypto dependency
- **US3 (P2)**: depends on US2+US5 for real factory construction paths; adapter can still be implemented independently
- **US4 (P2)**: depends on US2+US5 for real factory construction paths; adapter can still be implemented independently

---

## Parallel Opportunities

- Setup tasks `T002` and doc updates can proceed while directories are created (`T001`)
- Testcontainer helpers `T008`–`T010` are parallelizable
- Within each user story, method-level work can be parallelized by splitting tests/implementation into additional files (only mark `[P]` when tasks touch different files)

---

## Parallel Example: Provider Adapters (after Phase 2 completes)

These three touch **different files** and can genuinely run in parallel:

```bash
# Developer A
Task T015–T023: "Implement S3 adapter in file/service/blob_adapter/s3_compatible_adapter.go"

# Developer B
Task T037–T045: "Implement GCS adapter in file/service/blob_adapter/gcs_adapter.go"

# Developer C
Task T049–T057: "Implement Azure adapter in file/service/blob_adapter/azure_blob_adapter.go"
```

> **Note**: Tasks within a single provider story (e.g. T012–T014b for S3 tests) all write to the **same file** and must be done sequentially — do not mark single-file task groups as `[P]`.

---

## Implementation Strategy

### MVP First

1. Phase 1 + Phase 2
2. Phase 3 (US1 S3 adapter + integration tests)
3. Validate: `go test ./test/integration/file/... -run S3 -v -count=1`
4. Then proceed with factory (US2/US5) and remaining providers (US3/US4)

