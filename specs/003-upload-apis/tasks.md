# Tasks: File Upload APIs (Init + Complete)

**Input**: Design documents from `/specs/003-upload-apis/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md, contracts/

**Tests**: INCLUDED — the spec explicitly mandates integration tests (TR-001..TR-011, SC-002, SC-008). Integration tests are primary drivers; unit tests are authored only for pure helpers (policy evaluator, key sanitiser, envelope marshal).

**Organization**: Tasks are grouped by user story (US1..US6, plus US5a, US6a). Each story can be implemented and verified independently once Phases 1 & 2 are complete.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4, US5, US5a, US6, US6a)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare module scaffolding, constants, and error catalogue before any business logic is written.

- [ ] T001 Edit `file/entity/file.go` to remove `Metadata db.JSONMap` field from both `FileObject` and `FileVariant` structs; ensure `FileObject` has all columns from `data-model.md §1.1` (`FileID`, `SellerID`, `UploaderUserID`, `OwnerType`, `OwnerID`, `Purpose`, `Visibility`, `StorageConfigID`, `BucketOrContainer`, `ObjectKey`, `OriginalFilename`, `SanitizedFilename`, `MimeType`, `SizeBytes`, `Etag`, `Status`, `FailureReason`, `UploadExpiresAt`, `CompletedAt`) with correct GORM tags, and verify `FileJob` struct carries `CorrelationID` (add if missing) plus `Command`, `Status`, `Attempts`, `LastError` fields.
- [ ] T002 Edit `migrations/018_create_file_storage_tables.sql` in place (no new migration needed — migration 018 is on the feature branch and has not been merged to `develop`). **Pre-flight check before editing** (CA1): run `git log origin/develop -- migrations/018_create_file_storage_tables.sql`; if any commits are returned the migration has already landed on develop and you MUST create `019_file_object_tables.sql` instead and update this plan accordingly. If the output is empty, proceed: (a) drop the `metadata JSONB` column from `file_object` and `file_variant` DDL, (b) add `CREATE TABLE file_object`, `file_variant`, `file_job` with columns, CHECK constraints, and indexes exactly as listed in `data-model.md §1.1–§1.3`, and (c) keep the statement idempotent (`CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`).
- [ ] T003 [P] Create `file/utils/constant/upload_constants.go` with: error code string constants (all codes from `research.md R11`), scheduler command `SchedulerCommandUploadExpiry = "file.upload.expiry"`, RabbitMQ constants `ExchangeEcomCommands = "ecom.commands"` and `RoutingKeyFileImageProcessRequested = "file.image.process.requested"`, Redis key prefixes (`RedisKeyInitIdempotencyPrefix = "file:init:idem:"`, `RedisKeySchedulerFileUploadExpiryPrefix`), cache buffer (`CacheBufferDuration = 5 * time.Minute`), and default/min/max upload expiry minutes (15/5/60).
- [ ] T004 [P] Create `file/error/upload_errors.go` declaring `*AppError` singletons for every error code in `research.md R11` (`ErrFileUploadUnauthorized` 401, `ErrFileUploadForbidden` 403, `ErrFileUploadInvalidInput` 400, `ErrFileUploadPolicyViolation` 422, `ErrFileUploadStorageUnavailable` 503, `ErrFileUploadNoStorageConfig` 412, `ErrFileUploadNotFound` 404, `ErrFileUploadConflict` 409, `ErrFileUploadObjectMissing` 409, `ErrFileUploadObjectMismatch` 422, `ErrFileUploadExpired` 410, `ErrFileUploadInternal` 500) using the same factory pattern as `file/error/config_errors.go`.
- [ ] T005 [P] Create `file/messaging/contracts.go` with struct `ImageProcessRequested` (fields: `FileID string`, `FileObjectID uint64`, `StorageConfigID uint64`, `BucketOrContainer string`, `ObjectKey string`, `MimeType string`, `SizeBytes int64`, `Purpose string`, `VariantsRequested []string`) with JSON tags per `contracts/file.image.process.requested.event.md`.
- [ ] T006 [P] Create `file/model/upload_model.go` with request DTOs `InitUploadRequest` (fields: `Purpose`, `Visibility`, `Filename`, `MimeType`, `SizeBytes`, `UploadExpiryMinutes`) with binding tags + strict-JSON via `binding:"required"` and custom validators rejecting unknown fields, `CompleteUploadRequest` (`FileID`, `ClientEtag *string`, `ActualSizeBytes *int64`), and response DTOs `InitUploadData` and `CompleteUploadData` per `contracts/init-upload.http.md` and `contracts/complete-upload.http.md`.
- [ ] T007 Ensure `common/messaging/constants/messaging_constants.go` (or its closest equivalent) exports the required exchange and routing-key constants with the exact values used by this feature. Steps: (1) Search the file for `ExchangeEcomCommands` — if found, assert its value is `"ecom.commands"`; if the value differs, update it in place. (2) Search for the routing-key constant for `"file.image.process.requested"` — if found, assert the value matches exactly; update if not. (3) If neither constant exists in any file, add both beside the existing exchange/routing-key constants. Acceptance: `grep -r 'ecom.commands' common/` and `grep -r 'file.image.process.requested' common/` each return exactly one declaration with the correct string value.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build shared services, repository layer, factories, routes, and the scheduler + publisher wrappers that every user story depends on. **NO user story task may begin until this phase checkpoint is reached.**

- [ ] T008 Populate `file/repository/file_repository.go` `FileUploadRepository` interface (≤10 methods): `InsertUploading(ctx, *entity.FileObject) error`, `FindByFileIDScoped(ctx, fileID, ownerType, ownerID, sellerID) (*FileObject, error)`, `FindByID(ctx, id uint64) (*FileObject, error)`, `MarkActive(ctx, id, etag, sizeBytes, completedAt) error` (conditional on `status = entity.FileStatusUploading`), `MarkFailed(ctx, id, reason) error` (conditional on `status = entity.FileStatusUploading`), `InsertFileJob(ctx, *entity.FileJob) error`, `FindFileJobByFileObjectID(ctx, id) (*FileJob, error)`. Create `file/repository/file_repository_impl.go` with GORM implementation using single-statement transactions where FR-007/FR-017 demand atomicity; enforce `(owner_type, owner_id, seller_id)` scoping in `FindByFileIDScoped`. **CA4**: all status comparisons in GORM queries MUST use entity constants (`entity.FileStatusUploading`, `entity.FileStatusActive`, `entity.FileStatusFailed`) — never inline string literals like `"UPLOADING"` (Constitution §Magic Values).
- [ ] T009 [P] Create `file/service/upload_policy.go` (pure, no I/O) exposing `Policy struct{MaxSize int64; AllowedMimes []string; HasVariants bool; VariantCodes []string}` and `func Evaluate(purpose entity.FilePurpose, mime string, size int64) (*Policy, *AppError)`. Table-driven per `data-model.md §4.1`; return `ErrFileUploadPolicyViolation` on violation.
- [ ] T010 [P] Create `file/service/upload_key_builder.go` (pure, no I/O) with `SanitizeFilename(raw string) string` (lowercase, strip non `[A-Za-z0-9._-]`, collapse whitespace to `-`, truncate 120 bytes, default `file` if empty) and `BuildObjectKey(ownerType, sellerID, purpose, now time.Time, fileID, sanitized) string` implementing `research.md R9` template.
- [ ] T011 Create `file/service/upload_expiry_scheduler.go` exposing `UploadExpiryScheduler` interface with `Schedule(ctx, fileObjectID uint64, fileID string, sellerID *uint64, runAt time.Time, correlationID string) (jobID string, err error)` and `Cancel(ctx, fileObjectID uint64, sellerID *uint64) error`; implementation wraps `common/scheduler.Scheduler.Schedule/Cancel`, caches `jobID` under `seller:{sellerId}:file.upload.expiry:{fileObjectId}` (or `platform:...` for admin) with TTL `delay + CacheBufferDuration`, mirroring `inventory/service/reservation_scheduler_service.go` verbatim. **CA3**: the `correlationID` parameter MUST be embedded in the scheduled job's payload (not discarded) so `UploadExpiryHandler` can extract it for structured log context on execution — mirror the same pattern used in the inventory scheduler payload.
- [ ] T012 Create `file/service/upload_expiry_handler.go` implementing `scheduler.Handler` for command `file.upload.expiry`; signature `Handle(ctx, job ScheduledJob) error` performs the pseudocode in `contracts/file.upload.expiry.job.md §Handler semantics` (idempotent no-op if `ACTIVE`/`FAILED`/missing; otherwise best-effort `DeleteObject` + `MarkFailed(reason=UPLOAD_EXPIRED)`).
- [ ] T013 Create `file/service/upload_variant_publisher.go` exposing `VariantPublisher` interface with `Publish(ctx, msg ImageProcessRequested, correlationID string) error`; implementation performs passive-declare of `ecom.commands` (topic, durable), publishes persistent message with publisher confirms (2s timeout), using `common/messaging/envelope.go` and `common/messaging/rabbitmq/publisher.go`. **CA2**: before calling `publisher.Publish`, set the envelope's `CorrelationId` field to the `correlationID` parameter — this is required by Constitution §VI (Correlation IDs MUST be propagated to message queues); the field name is defined in `common/messaging/envelope.go`.
- [ ] T014 Create `file/service/upload_service.go` exposing `FileUploadService` interface with `InitUpload(ctx, caller Principal, req InitUploadRequest, idempotencyKey *string) (*InitUploadData, error)` and `CompleteUpload(ctx, caller Principal, req CompleteUploadRequest) (*CompleteUploadData, error)`; leave method bodies as `TODO` returning `ErrFileUploadInternal` — they will be implemented per-story. Wire dependencies: repository, storage config resolver (from `config_service.go`), blob adapter factory, scheduler wrapper, variant publisher, Redis client (for idempotency).
- [ ] T015 Edit `file/service/file_service.go` to remove any placeholder that collides with the new `FileUploadService` and to delegate legacy methods unchanged. If the file is empty / stub, leave it untouched.
- [ ] T016 Edit `file/handler/file_handler.go` to add `FileUploadHandler` struct with methods `InitUpload(c *gin.Context)` and `CompleteUpload(c *gin.Context)`; both bind request, extract `Principal` (role + id + sellerId) from context (`common/auth`), call the service, and render response via `common.Success*`/`common.ErrorWithCode`. Reject unknown JSON fields by using `json.NewDecoder(...).DisallowUnknownFields()` (or equivalent Gin binding). Enforce `X-Correlation-ID` header presence → 400 if missing. **Sub-item (U2)**: Before wiring the handler, create `file/service/principal.go` defining `type Principal struct { Role string; UserID uint64; SellerID *uint64; OwnerType entity.FileOwnerType }` and `func ExtractPrincipal(c *gin.Context) (Principal, *AppError)` that calls `common/auth.GetUserIDFromContext`, `GetSellerIDFromContext`, `GetUserRoleFromContext` and maps them into the struct — returns `ErrFileUploadForbidden` if role is missing or unrecognised. This type is consumed by `FileUploadService` interface (T014) and role extraction tests (T042).
- [ ] T017 Edit `file/route/file_operation_route.go` to mount (a) `/api/files` group guarded by `middleware.SellerAuth()` with `POST /init-upload` and `POST /complete-upload`, and (b) `/api/admin/files` group guarded by `middleware.AdminAuth()` with the same two routes, both bound to the same `FileUploadHandler` methods (no handler duplication). Customers reach neither group.
- [ ] T018 Edit `file/factory/singleton/repository_factory.go`, `service_factory.go`, `handler_factory.go`, and `singleton_factory.go` to expose `FileUploadRepository()`, `FileUploadService()`, `UploadExpiryScheduler()`, `UploadExpiryHandler()`, `VariantPublisher()`, and `FileUploadHandler()` via the factory-singleton pattern. Register `UploadExpiryHandler` with `scheduler.Registry` inside `file/container.go` at module bootstrap (mirroring inventory).
- [ ] T019 Edit `file/container.go` to call the new route registration (done in T017) and to register the scheduler handler (done in T018); no other behavioural change.

**Checkpoint**: After T019, the module compiles, routes are mounted (returning 501/TODO), scheduler handler is registered, and factories expose all upload dependencies. User story phases may now begin in parallel.

---

## Phase 3: User Story 1 — Seller uploads a product image (happy path) (Priority: P1) 🎯 MVP

**Goal**: End-to-end presigned upload for `PRODUCT_IMAGE`: init returns fileId + URL, client PUTs bytes, complete transitions row to `ACTIVE` and publishes a `file.image.process.requested` message.

**Independent Test**: Seller with MinIO-backed storage config calls init with a valid JPEG envelope, PUTs bytes to returned URL, calls complete, and asserts: `file_object.status=ACTIVE`, object present at deterministic key, RabbitMQ message received with matching `fileObjectId`.

### Tests for User Story 1

- [ ] T020 [P] [US1] Create `test/integration/file/setup_upload_suite_test.go` declaring `UploadSuite` (testify Suite) that boots Postgres + Redis + MinIO + RabbitMQ via Testcontainers once per package, runs migrations, seeds admin/seller/customer/platform config + seller binding per `quickstart.md §2, §4`, starts `common/scheduler` workers, declares the test-only `file.image.process.requested.q` queue bound to `ecom.commands`, and exposes a buffered channel of consumed variant messages.
- [ ] T021 [P] [US1] Create `test/integration/setup/rabbitmq_container.go` with a Testcontainers helper using `rabbitmq:3.13-management-alpine`, wait strategy `wait.ForLog("Server startup complete")`, and returning AMQP URI + `*amqp091.Connection` per `research.md R2`.
- [ ] T022 [P] [US1] Lift/create `test/integration/setup/minio_container.go` (reuse logic from `test/integration/file/minio_container.go`) adding `CreateBucket(ctx, name)` per `research.md R3`.
- [ ] T023 [P] [US1] Create `test/integration/helpers/upload_helper.go` implementing `UploadJourney` per `quickstart.md §3`: `Init`, `PutBytes`, `Complete`, `RunHappyPath`, `AssertFileObject`, `NextVariantMessage`, `AssertNoVariantMessage`.
- [ ] T024 [P] [US1] Create `test/integration/helpers/scheduler_helper.go` with `AssertSchedulerJobExists(t, fileObjectID)`, `AssertNoSchedulerJob(t, fileObjectID)`, and `FastForwardExpiry(t, fileObjectID)` (rewrites ZSET score per `quickstart.md §7`).
- [ ] T025 [US1] Create `test/integration/file/upload_init_test.go` with `TestInitUpload_ProductImage_HappyPath` asserting US1 Acceptance #1: 201 response shape per contract, `file_object` row in `UPLOADING` with correct columns, scheduler job exists, idempotency hash NOT set (no header supplied).
- [ ] T026 [US1] Create `test/integration/file/upload_complete_test.go` with `TestCompleteUpload_ProductImage_HappyPath` (US1 Acceptance #2 + #3): seller performs init→PUT→complete, asserts 200 `ACTIVE`, `file_job{status=PUBLISHED}` row, scheduler job cancelled, RabbitMQ message delivered on `file.image.process.requested` with envelope fields matching `contracts/file.image.process.requested.event.md`. **CA2**: additionally assert that the consumed envelope's `CorrelationId` field equals the `X-Correlation-ID` header sent with the HTTP request (Constitution §VI — correlation ID propagation to message queues).
- [ ] T027 [P] [US1] Create `file/service/upload_policy_test.go` unit tests covering every purpose/mime/size combination in `data-model.md §4.1` (table-driven).
- [ ] T028 [P] [US1] Create `file/service/upload_key_builder_test.go` unit tests covering sanitiser edge cases (path separators, NUL, RTL override, 500-char filename, empty result).

### Implementation for User Story 1

- [ ] T029 [US1] Implement `FileUploadService.InitUpload` happy path in `file/service/upload_service.go`: resolve caller role/owner, call `upload_policy.Evaluate`, resolve storage config (seller binding → platform default, else `ErrFileUploadNoStorageConfig`), generate `fileId` via `uuid.NewV7()`, build object key via `upload_key_builder`, open a single DB transaction: insert `file_object{status=UPLOADING}` + call `blob_adapter.PresignUpload` (rollback on any error → `ErrFileUploadStorageUnavailable`), schedule expiry job via `UploadExpiryScheduler.Schedule`, commit, and return `InitUploadData` per contract. (Idempotency-Key branch is NOT implemented here; handled in US5a.)
- [ ] T030 [US1] Implement `FileUploadService.CompleteUpload` happy path in `file/service/upload_service.go`: `FindByFileIDScoped` (404 on miss), if already `ACTIVE` return idempotent 200 with stored state + re-read `file_job` for `variantsQueued` flag (FR-012), else `HeadObject` via adapter (map `ErrBlobNotFound` → `ErrFileUploadObjectMissing`); run three independent hint verifications against `HeadObject` output: (a) verify reported size equals init `sizeBytes` (FR-015 → `MarkFailed` + `ErrFileUploadObjectMismatch` / `422`), (b) verify `HeadObject.ContentType` matches init-declared `mimeType` — mismatch → `MarkFailed` + `ErrFileUploadObjectMismatch` / `422` (FR-015, C2), (c) if request `ClientEtag` is non-nil verify it equals `HeadObject.ETag` — mismatch → `MarkFailed` + `ErrFileUploadObjectMismatch` / `422` (FR-016, C1); `MarkActive` atomically (persists `etag`, `size_bytes`, `completed_at`), `scheduler.Cancel` best-effort, call `upload_policy.EvaluateVariants(purpose, mimeType)` — if `HasVariants=true` call `VariantPublisher.Publish` and insert `file_job{status=PUBLISHED}` (on publish failure → `file_job{status=FAILED_TO_PUBLISH}` but return 200). Propagate `ctx` through every provider and DB call so request cancellation unwinds all I/O (FR-023, C4).
- [ ] T031 [US1] Verify error-mapping behavior in `file/handler/file_handler.go` (behavior test, not setup — setup is T016): assert that binding errors (unknown JSON fields, missing required fields, wrong types) produce `400 VALIDATION_ERROR` with a structured envelope; assert that `X-Correlation-ID` is forwarded into the service `ctx` so it appears in audit log entries (FR-020) and propagates to downstream calls. Add assertions to `upload_init_test.go` or `upload_complete_test.go` — no new production code required if T016 was implemented correctly.
- [ ] T032 [US1] Add structured audit log at handler entry/exit (FR-020): action, actor, file_id, provider_code (derived), bucket, key, ip, user-agent — no credentials.

**Checkpoint**: US1 integration tests pass. Seller can upload a PRODUCT_IMAGE end-to-end. This is the MVP.

---

## Phase 4: User Story 2 — Seller finalises a non-image document (Priority: P2)

**Goal**: Same flow as US1 but for `DOCUMENT` / `IMPORT_FILE` purposes: variant publishing is suppressed.

**Independent Test**: Seller inits+completes a PDF (`purpose=DOCUMENT`, `mimeType=application/pdf`) and a CSV (`purpose=IMPORT_FILE`), both land as `ACTIVE`, no `file.image.process.requested` message is published, no `file_job` row created.

### Tests for User Story 2

- [ ] T033 [P] [US2] Extend `test/integration/file/upload_complete_test.go` with `TestCompleteUpload_Document_PDF_NoVariants` (US2 Acceptance #1) asserting `ACTIVE`, `variantsQueued=false`, and `AssertNoVariantMessage` within 500ms window.
- [ ] T034 [P] [US2] Add `TestCompleteUpload_ImportFile_CSV_NoVariants` to the same file (US2 Acceptance #2) asserting the same for `IMPORT_FILE + text/csv`.

### Implementation for User Story 2

- [ ] T035 [US2] Verify branch in `FileUploadService.CompleteUpload` (`file/service/upload_service.go`) keys variant publishing off `policy.HasVariants` not off mime string; adjust `upload_policy.go` if needed so `DOCUMENT`, `IMPORT_FILE`, `INVOICE_PDF` all return `HasVariants=false`, and `USER_AVATAR` / `SELLER_LOGO` return `HasVariants=true` with raster-only guard (SVG skips variant publish for `SELLER_LOGO`). No new code path in handler.

**Checkpoint**: US1 and US2 both pass. Image and non-image purposes both finalise correctly.

---

## Phase 5: User Story 3 — Quota / policy enforcement on init (Priority: P2)

**Goal**: Reject policy violations before any DB write or storage call.

**Independent Test**: Init with oversized (`PRODUCT_IMAGE`, 12 MB), disallowed mime (`application/x-msdownload` for `PRODUCT_IMAGE`), empty filename, and `sizeBytes <= 0` all return structured 400/422 with no DB row and no MinIO call.

### Tests for User Story 3

- [ ] T036 [P] [US3] Create `test/integration/file/upload_policy_test.go` with subtests: `Reject_OversizedProductImage` (422 `FILE_UPLOAD_POLICY_VIOLATION`), `Reject_DisallowedMime` (422), `Reject_EmptyFilename` (400 `VALIDATION_ERROR`), `Reject_ZeroSize` (400), `Reject_ExportFilePurpose` (422). Assert no `file_object` rows, no scheduler jobs, and no provider calls (inspect MinIO object count).
- [ ] T037 [P] [US3] Extend `file/service/upload_policy_test.go` with negative cases for each branch in T036.

### Implementation for User Story 3

- [ ] T038 [US3] Ensure `FileUploadService.InitUpload` (`file/service/upload_service.go`) calls `upload_policy.Evaluate` **before** storage resolution and DB write; return the policy `*AppError` verbatim.
- [ ] T039 [US3] Ensure handler binding rejects `sizeBytes<=0`, empty `filename`, unknown `purpose`, unknown `visibility` via DTO validators in `file/model/upload_model.go`.

**Checkpoint**: US3 tests pass. Policy enforcement is pre-flight.

---

## Phase 6: User Story 4 — Tenant isolation on complete-upload (Priority: P2)

**Goal**: Cross-tenant `complete-upload` must return 404 (no enumeration) and leave the victim row untouched. Admin cannot complete a seller's upload.

**Independent Test**: Seller A inits; Seller B calls complete with A's fileId → 404. Unauthenticated → 401. Customer → 403. Admin → 404 for a `SELLER`-owned row.

### Tests for User Story 4

- [ ] T040 [P] [US4] Create `test/integration/file/upload_tenant_isolation_test.go` with subtests per US4 Acceptance #1–#4: `SellerB_Cannot_Complete_SellerA_File` (404, A's row untouched), `Unauthenticated_Returns_401`, `Customer_Returns_403`, `Admin_Cannot_Complete_Seller_File` (404).

### Implementation for User Story 4

- [ ] T041 [US4] Verify `FileUploadRepository.FindByFileIDScoped` in `file/repository/file_repository_impl.go` enforces `(owner_type, owner_id, seller_id)` predicates matching the caller; write unit-coverage via table test if needed. Service returns `ErrFileUploadNotFound` on miss (no 403).
- [ ] T042 [US4] Verify role extraction in handler (`file/handler/file_handler.go`) sets `owner_type=SELLER` for seller callers and `owner_type=PLATFORM` for admin callers; `Principal` type in helper never blends scopes.

**Checkpoint**: US4 tests pass. Cross-tenant enumeration is impossible.

---

## Phase 7: User Story 5 — Complete called before the object exists (Priority: P3)

**Goal**: Graceful handling of premature / mismatched / duplicate complete calls.

**Independent Test**: Call init, skip PUT, call complete → 409 `OBJECT_MISSING`, row stays `UPLOADING`. Size mismatch → 422 `OBJECT_MISMATCH`, row `FAILED`. Duplicate complete on `ACTIVE` → idempotent 200, no duplicate variant message.

### Tests for User Story 5

- [ ] T043 [P] [US5] Extend `test/integration/file/upload_complete_test.go` with `TestCompleteUpload_ObjectMissing` (US5 #1): init without PUT → 409, row `UPLOADING`, scheduler job still present, no variant message.
- [ ] T044 [P] [US5] Add `TestCompleteUpload_SizeMismatch` (US5 #2): PUT body whose byte-length differs from init `sizeBytes` → 422 `FILE_UPLOAD_OBJECT_MISMATCH`, row `FAILED`, no variant message.
- [ ] T045 [P] [US5] Add `TestCompleteUpload_Idempotent_OnActive` (US5 #3): init → PUT → complete (success) → complete again → idempotent 200, exactly one RabbitMQ message across both calls.
- [ ] T046 [P] [US5] Add `TestCompleteUpload_RabbitMQOutage_Still200` (TR-005, edge case): stop RabbitMQ container (or toggle flaky publisher via test double env var), complete succeeds with 200 and `file_job{status=FAILED_TO_PUBLISH}`.

### Implementation for User Story 5

- [ ] T047 [US5] [REVIEW] Code-review gate: verify `FileUploadService.CompleteUpload` (T030) covers all mismatch branches — `ErrBlobNotFound` (→ `ErrFileUploadObjectMissing`, 409, no status change), size mismatch, mime ContentType mismatch, `clientEtag` hint mismatch (all three → `MarkFailed` + `ErrFileUploadObjectMismatch` 422). Confirm every `MarkFailed` path carries `reason=OBJECT_MISMATCH` and does NOT call `VariantPublisher.Publish`. No new production code; integration tests T044/T065/T066 provide executable coverage.
- [ ] T048 [US5] [REVIEW] Code-review gate: verify the idempotent path in `FileUploadService.CompleteUpload` (T030) re-reads `file_job` to populate `variantsQueued` without calling `VariantPublisher.Publish` a second time (FR-012), and that the publish-failure path sets `file_job{status=FAILED_TO_PUBLISH}` but still returns HTTP 200 (FR-019). No new production code; T045/T046 provide executable coverage.
- [ ] T065 [P] [US5] Add `TestCompleteUpload_EtagHintMismatch` (C1 / FR-016): PUT the correct bytes, then call complete-upload with a `clientEtag` value that does NOT match the provider-returned ETag; assert `422 FILE_UPLOAD_OBJECT_MISMATCH`, row transitions to `FAILED`, no variant message published (`AssertNoVariantMessage`). Add this subtest to `test/integration/file/upload_complete_test.go`.
- [ ] T066 [P] [US5] Add `TestCompleteUpload_MimeMismatch` (C2 / FR-015): init with `mimeType=image/jpeg`; PUT bytes that are actually a PDF (so `HeadObject.ContentType` returns `application/pdf`); call complete-upload; assert `422 FILE_UPLOAD_OBJECT_MISMATCH`, row `FAILED`, no variant message. Add to `test/integration/file/upload_complete_test.go`.
- [ ] T067 [P] [US5] Add `TestCompleteUpload_ConcurrentCalls_SingleVariantMessage` (C3 / SC-004): after init+PUT, fire two complete-upload calls in parallel via `errgroup`; assert both return 200, the row is `ACTIVE`, and exactly one `file.image.process.requested` message was published (use `AssertNoVariantMessage` with a 500ms window after consuming the first). Add to `test/integration/file/upload_complete_test.go`.

**Checkpoint**: US5 tests pass. Retry and outage semantics are correct.

---

## Phase 8: User Story 5a — Init-upload is idempotent against client retries (Priority: P2)

**Goal**: `Idempotency-Key` header dedupes init retries within `uploadExpiryMinutes + cacheBufferDuration` window.

**Independent Test**: Two back-to-back inits with same header + same body → same `fileId`, one DB row, one scheduler job, one idempotency hash.

### Tests for User Story 5a

- [ ] T049 [P] [US5a] Create `test/integration/file/upload_idempotency_test.go` with subtests per US5a Acceptance: `SameKey_SameBody_ReturnsSameFileId` (#1), `SameKey_AfterUploadUrlExpired_ReissuesUrl` (#2), `SameKey_AfterActive_Returns409Conflict` (#3), `MalformedKey_Returns400` (#4), `DifferentSellers_SameKey_Distinct` (#5 / FR-035).

### Implementation for User Story 5a

- [ ] T050 [US5a] Implement `Idempotency-Key` handling in `FileUploadService.InitUpload` (`file/service/upload_service.go`): validate regex `^[A-Za-z0-9._~-]+$` and length 8..128, SHA-256 hash, `SETNX` on Redis key `file:init:idem:{ownerType}:{ownerId}:{sha256(key)}` with value `{fileId, fingerprint, uploadUrl, headers, expiresAt}`, TTL = `uploadExpiryMinutes*60 + cacheBufferDuration`. If key exists: fetch record, if fingerprint matches and row still `UPLOADING` reuse `fileId` (re-issue presigned URL if cached `expiresAt <= now`), else if row status moved on → `ErrFileUploadConflict` (409, code `FILE_UPLOAD_CONFLICT`), else if fingerprint mismatches → `ErrFileUploadConflict`.
- [ ] T051 [US5a] Ensure handler passes `c.GetHeader("Idempotency-Key")` (`*string`, nil when absent) into the service call and maps malformed key → `VALIDATION_ERROR` with field `Idempotency-Key` via validator.

**Checkpoint**: US5a tests pass. Client retries are safe.

---

## Phase 9: User Story 6a — Abandoned upload is auto-cleaned (Priority: P2)

**Goal**: Scheduler handler transitions expired `UPLOADING` rows to `FAILED/UPLOAD_EXPIRED` and best-effort deletes stray objects.

**Independent Test**: Init with short `uploadExpiryMinutes`, skip PUT, fast-forward scheduler, assert row `FAILED` with reason `UPLOAD_EXPIRED`, Redis job key gone, stray object deleted, subsequent complete → 410 `FILE_UPLOAD_EXPIRED`.

### Tests for User Story 6a

- [ ] T052 [P] [US6a] Create `test/integration/file/upload_expiry_test.go` with subtests: `ExpiryHandler_TransitionsToFailed` (US6a #1 + TR-009) using `FastForwardExpiry` helper, `CompleteCancelsExpiryJob` (US6a #2 + TR-010), `ExpiryFires_AfterActive_IsNoOp` (FR-029), `Reject_UploadExpiryOutOfRange` (US6a #3: values 0 and 61 → 400 `VALIDATION_ERROR`), `ReCompleteAfterExpiry_Returns409Expired` (US6a #4 → 410 `FILE_UPLOAD_EXPIRED`). **CA3**: in `ExpiryHandler_TransitionsToFailed`, capture the structured log output from `UploadExpiryHandler` and assert it contains a `correlationId` field matching the value stored in the scheduled job payload (verifying Constitution §VI end-to-end propagation through the scheduler).

### Implementation for User Story 6a

- [ ] T053 [US6a] Verify `UploadExpiryHandler.Handle` (`file/service/upload_expiry_handler.go`, from T012) matches pseudocode in contract: idempotent guard, best-effort delete, conditional `MarkFailed`. Add logging per contract §Observability.
- [ ] T054 [US6a] Verify `InitUpload` validates `UploadExpiryMinutes ∈ [5,60]` (default 15) via DTO validator in `file/model/upload_model.go`; out-of-range → 400 `VALIDATION_ERROR` with detail code `UPLOAD_EXPIRY_OUT_OF_RANGE`.
- [ ] T055 [US6a] Verify `CompleteUpload` returns `ErrFileUploadExpired` (410) when row is already `FAILED` with reason `UPLOAD_EXPIRED` (FR-028), distinct from generic `FILE_STATE_INVALID`.

**Checkpoint**: US6a tests pass. No row lingers in `UPLOADING` past its deadline.

---

## Phase 10: User Story 6 — Storage outage during init (Priority: P3)

**Goal**: Provider failures at init time map to 502/503 with no orphan rows.

**Independent Test**: Point seller storage config at a non-existent bucket → init returns 503 `FILE_UPLOAD_STORAGE_UNAVAILABLE`, no DB row. Invalid credentials → 502 `STORAGE_PERMISSION_DENIED` (mapped through same `ErrFileUploadStorageUnavailable` family or distinct error — follow research R11 table).

### Tests for User Story 6

- [ ] T056 [P] [US6] Create `test/integration/file/upload_outage_test.go` with subtests: `NonexistentBucket_Returns503_NoDbRow` (US6 #1), `InvalidCredentials_Returns502_NoSecretInMessage` (US6 #2 — also cover SC-006 by scanning response body for known secret substrings), `NoStorageConfig_Returns412_NoDbRow` (C5 — seed a seller with no active binding and disable the platform default, call init, assert `412 FILE_UPLOAD_NO_STORAGE_CONFIG`, no `file_object` row inserted).

### Implementation for User Story 6

- [ ] T057 [US6] Verify `InitUpload` single-transaction rollback path (T029): any non-nil error from `PresignUpload` or `scheduler.Schedule` rolls back the `file_object` insert (FR-009); provider errors map to `ErrFileUploadStorageUnavailable` (or distinct codes if research R11 mandates separate) and never echo provider error bodies. Error-message sanitiser lives in `file/error/upload_errors.go`.
- [ ] T058 [US6] Add integration subtest `RedisUnavailable_InitUpload_Returns503_NoDbRow` (C6) to `upload_outage_test.go`: pause the Redis container via `tc.Container.Stop(ctx)` before calling init-upload; assert `503 FILE_UPLOAD_STORAGE_UNAVAILABLE`, no `file_object` row inserted (SC-003), then restart Redis for teardown. This replaces the earlier "verify" stub and provides concrete evidence of the rollback guarantee under FR-009.

**Checkpoint**: US6 tests pass. Provider outages do not create orphan state.

---

## Phase 11: Polish & Cross-Cutting Concerns

- [ ] T059 [P] Run `.specify/scripts/bash/update-agent-context.sh codex` to patch `AGENTS.md` with the new upload surface area per plan §Phase 1 step 4.
- [ ] T060 [P] Create `test/integration/file/upload_performance_test.go` (build tag `//go:build perf`) executing a 1 MB JPEG round-trip and asserting p95 ≤ 3 s (SC-001).
- [ ] T061 [P] Add a response-body scanning helper in `test/integration/helpers/assertions.go` (`AssertNoSecretsInBody(t, body)`) and invoke it across all `4xx`/`5xx` error assertions to enforce SC-006.
- [ ] T062 Run the full test suite and verify SC-002 and SC-008. Use the exact commands below: (1) `go test ./file/... ./test/integration/file/...` — all tests must pass (SC-002). (2) Generate a coverage profile and enforce the ≥ 85% threshold (SC-008): `go test -coverprofile=coverage.out -covermode=count ./file/handler/... ./file/service/... ./file/repository/...` then `go tool cover -func=coverage.out | awk '/^total:/{val=strtonum($3); if (val < 85.0) {printf "FAIL: coverage %.1f%% < 85%%\n", val; exit 1} else {printf "OK: coverage %.1f%%\n", val}}'`. Build fails if the threshold is not met. Fix uncovered lines before merging.
- [ ] T063 Run the quickstart playbook commands (`quickstart.md §1`) end-to-end on a clean workspace to validate no undocumented setup steps remain.
- [ ] T064 [P] Review logs emitted by `FileUploadService` / `UploadExpiryHandler` / `VariantPublisher` to ensure FR-020 audit fields are present and FR-021 redaction holds (no `endpoint`, `accessKey`, or stack trace content).

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: T001 and T002 are sequential (entity changes drive migration shape); T003–T007 can run in parallel after T001.
- **Phase 2 (Foundational)**: Depends on Phase 1 completion. Within Phase 2:
  - T008 after T001/T002.
  - T009, T010 are [P] (pure, no dependency on T008).
  - T011 after T003 (constants) and exists independently of T008.
  - T012 after T008 (needs repo) and T011 (needs scheduler wrapper indirectly via job payload type).
  - T013 after T005 (envelope payload struct) and T003 (constants).
  - T014 after T008, T009, T010, T011, T013.
  - T015, T016 after T014.
  - T017 after T016.
  - T018 after T008, T011, T012, T013, T014, T016.
  - T019 after T017, T018.
- **Phase 3+ (User Stories)**: All depend on Phase 2 checkpoint (T019). Individual stories can then proceed in parallel if staffed; otherwise follow priority order (P1 → P2 → P3).
- **Polish (Phase 11)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **US1 (P1)**: No story dependency. Blocks MVP.
- **US2 (P2)**: Independent of US1 once `HasVariants` branching exists (T030 from US1 provides it, but can be reimplemented first). Can run after Foundational.
- **US3 (P2)**: Independent — pre-flight policy check.
- **US4 (P2)**: Independent — tenant scoping lives in repository layer.
- **US5 (P3)**: Benefits from US1 being done (same file `upload_complete_test.go`), but logic is additive.
- **US5a (P2)**: Independent of other stories; touches only `InitUpload` idempotency branch and a new test file.
- **US6a (P2)**: Independent — exercises scheduler handler + a new test file.
- **US6 (P3)**: Independent — exercises init error-mapping + a new test file.

### Within Each User Story

- Test tasks (integration + unit) MUST be written and fail before implementation tasks.
- Models (`file/model/upload_model.go`) → repository (`file_repository_impl.go`) → service (`upload_service.go`) → handler (`file_handler.go`) → route (already mounted in Foundational).
- Within a story, all test files marked `[P]` touching different files can run in parallel.

### Parallel Opportunities

- Phase 1: T003, T004, T005, T006 all touch distinct new files → run in parallel after T001.
- Phase 2: T009, T010 are pure helpers in distinct new files → parallel. T013 and T011 touch distinct new files → parallel.
- Phase 3 tests: T020–T024 and T027, T028 all create distinct files → parallel.
- US2/US3/US4/US5/US5a/US6/US6a integration tests are all in distinct files → can be authored in parallel by different contributors.

---

## Parallel Example: User Story 1 kickoff

```bash
# After Foundational (T019), launch all US1 test scaffolding in parallel:
Task: "Create test/integration/file/setup_upload_suite_test.go (T020)"
Task: "Create test/integration/setup/rabbitmq_container.go (T021)"
Task: "Create test/integration/setup/minio_container.go (T022)"
Task: "Create test/integration/helpers/upload_helper.go (T023)"
Task: "Create test/integration/helpers/scheduler_helper.go (T024)"
Task: "Create file/service/upload_policy_test.go (T027)"
Task: "Create file/service/upload_key_builder_test.go (T028)"

# Then serialize the two suite-attached integration tests (same package, suite state):
Task: "Create test/integration/file/upload_init_test.go (T025)"
Task: "Create test/integration/file/upload_complete_test.go (T026)"

# Finally implement service + handler:
Task: "Implement InitUpload happy path in file/service/upload_service.go (T029)"
Task: "Implement CompleteUpload happy path in file/service/upload_service.go (T030)"
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Complete Phase 1 (Setup) — entity, migration, constants, errors, DTOs.
2. Complete Phase 2 (Foundational) — repository, services, scheduler wrapper, publisher, factory wiring, routes.
3. Complete Phase 3 (US1) — full seller → MinIO → RabbitMQ journey with integration tests.
4. **STOP and VALIDATE**: run `go test ./test/integration/file/... -run '^TestUploadSuite$/TestInitUpload_ProductImage_HappyPath|^TestUploadSuite$/TestCompleteUpload_ProductImage_HappyPath'`.
5. MVP is demoable: a seller can upload a product image end-to-end.

### Incremental Delivery (recommended)

1. MVP: Setup + Foundational + US1.
2. Add US2 (non-image docs) — low risk, one branch addition.
3. Add US3 (policy) and US4 (tenant) in parallel — independent.
4. Add US5 + US5a + US6a (retry / idempotency / expiry) — all touch failure semantics.
5. Add US6 (storage outage) — final resilience test.
6. Polish phase (perf test, AGENTS.md refresh, coverage audit).

### Parallel Team Strategy

With multiple developers, after T019 (Foundational checkpoint):

- Dev A: US1 (MVP) → US5.
- Dev B: US3 + US4 (policy + tenant).
- Dev C: US5a + US6a (idempotency + expiry).
- Dev D: US2 + US6 (non-image + outage) + Polish.

---

## Format Validation Checklist

All tasks above conform to: `- [ ] T### [P?] [Story?] Description with exact file path`:

- Every task starts with `- [ ]`.
- Every task carries a sequential `T001..T067` ID.
- Every user-story task carries a `[US1]..[US6a]` label; Setup, Foundational, and Polish tasks carry no story label.
- Every task names at least one exact file path to be created or edited.

Total tasks: **67**. Tasks per phase: Setup 7 (T001–T007), Foundational 12 (T008–T019), US1 13 (T020–T032), US2 3 (T033–T035), US3 4 (T036–T039), US4 3 (T040–T042), US5 9 (T043–T048, T065–T067), US5a 3 (T049–T051), US6a 4 (T052–T055), US6 3 (T056–T058), Polish 6 (T059–T064).

Suggested MVP scope: **T001–T032** (Phase 1 + Phase 2 + US1).
