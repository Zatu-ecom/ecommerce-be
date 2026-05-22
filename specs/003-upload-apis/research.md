# Phase 0 Research: File Upload APIs

All NEEDS CLARIFICATION items from the plan's Technical Context are resolved below. Each entry follows the
Decision / Rationale / Alternatives format.

---

## R1. `fileId` identifier format

- **Decision**: Use UUIDv7 (time-ordered) via `github.com/google/uuid` (`uuid.NewV7()`), already an indirect dependency, promoted to direct.
- **Rationale**:
  - Existing codebase already imports `github.com/google/uuid` for correlation IDs and other IDs.
  - UUIDv7 preserves insertion locality on B-trees (`file_object.file_id` UNIQUE index) and keeps the object-key suffix sortable, which helps S3 listing.
  - Avoids pulling in a second ID library (`oklog/ulid/v2`).
- **Alternatives considered**:
  - **ULID** — rejected because it introduces a new direct dependency with no concrete benefit over UUIDv7 given Postgres `UUID` type support is not in play (column stays `VARCHAR(36)` per entity).
  - **Snowflake-style int64** — rejected because it needs a coordinator; the module does not justify one.

---

## R2. RabbitMQ Testcontainer

- **Decision**: Use `testcontainers.GenericContainer` with image `rabbitmq:3.13-management-alpine`, wait strategy `wait.ForLog("Server startup complete")`, exposed port `5672/tcp`. Wire it in a new helper `test/integration/setup/rabbitmq_container.go` that returns an AMQP URI and a ready `*amqp091.Connection`.
- **Rationale**:
  - The existing `common/messaging/rabbitmq` package already accepts a URI; no new abstraction needed.
  - Management image adds ~20 MB but gives the HTTP API for debug inspection in tests (queue peek), worth the cost for this first end-to-end RabbitMQ test.
  - Matches the testcontainers v0.42.0 upgrade already done for Docker security remediation.
- **Alternatives considered**:
  - **`testcontainers-go/modules/rabbitmq`** — rejected because it pins image tags in a way that conflicts with our existing version policy, and the generic container gives us direct control of the healthcheck log line.
  - **In-process AMQP mock** — rejected; violates "integration-first" (constitution §IV).

---

## R3. MinIO reuse

- **Decision**: Reuse `test/integration/file/minio_container.go` unchanged. Lift the helper into `test/integration/setup/minio_container.go` so the upload suite and the existing blob-adapter suite both consume the same container constructor; add `CreateBucket(ctx, name)` convenience.
- **Rationale**: Avoids container duplication, keeps bucket seeding deterministic, and the file already exposes the minimal surface needed (host, port, access key, secret key, endpoint URL).
- **Alternatives considered**: Stand up a second instance inside the suite → rejected (slow, resource-heavy).

---

## R4. `common/scheduler` handler registration and worker lifecycle

- **Decision**:
  - Register an `UploadExpiryHandler` with command key `file.upload.expiry` via the existing `scheduler.Registry` facility at module bootstrap (inside `file/container.go` when the module is constructed, mirroring inventory).
  - In production the scheduler workers are started by `main.go`'s existing scheduler bootstrap.
  - In integration tests the test server must call `scheduler.StartWorkers(ctx, redisClient, 1 /*workers*/)` once the module registers handlers; a new helper `setup.StartSchedulerWorkers(t, redisClient)` wraps that with `t.Cleanup` for graceful shutdown.
- **Rationale**: Mirrors `inventory` module verbatim; no new abstraction.
- **Alternatives considered**: Synchronous in-process timer → rejected because it forces tests to `time.Sleep` without cancellability and it doesn't represent production behaviour.

---

## R5. Admin-auth route group

- **Decision**: Add a **second route group** in `file/route/file_operation_route.go`:
  - `/api/files` group guarded by `middleware.SellerAuth()` → existing behaviour.
  - `/api/admin/files` group guarded by `middleware.AdminAuth()` → same handlers, which read the caller role from the Gin context to derive `owner_type` (`PLATFORM` for admin).
  - No route duplication in code; both groups register the same `FileUploadHandler.InitUpload`/`CompleteUpload` methods.
- **Rationale**: Matches how other modules (`product`, `order`) expose admin variants; keeps RBAC enforcement at middleware boundary (constitution §VII).
- **Alternatives considered**:
  - Single route + in-handler role switch → rejected because customers could hit the endpoint and reach handler code before a 403 is rendered.
  - Separate admin controller → rejected (duplicate code; violates SRP of handler).

---

## R6. DB schema source of truth and the `metadata` column

- **Decision**:
  - `file_object`, `file_variant`, `file_job` tables are added to **the same migration file** `migrations/018_create_file_storage_tables.sql` because that file has not been merged to `develop`.
  - The `metadata JSONB` column is **removed** from both `file_object` and `file_variant` in the entity structs (`file/entity/file.go`) and not declared in the migration.
  - GORM is **not** used in auto-migrate mode in this project (migrations drive the schema), so the only consistency obligation is: entity fields must match table columns. We will run the integration test suite which performs real migrations, to catch drift.
- **Rationale**: Keeps schema history linear for an unreleased migration and avoids creating a follow-up "drop column" migration for a schema no one is running.
- **Alternatives considered**:
  - New migration `019_drop_file_metadata.sql` → rejected (see plan's Complexity Tracking).
  - Keep the column but nil from the API → rejected per user decision (no untyped fields).

---

## R7. RabbitMQ exchange / queue declaration ownership

- **Decision**:
  - The **publisher side** (this feature) declares the exchange `ecom.commands` (durable, topic) with a passive-declare-first fallback, and does **not** declare consumer queues or bindings.
  - Consumer queues (`file.image.process.requested.q` + DLQ) and bindings are declared by the variant-worker feature that will consume them; the integration test for this feature sets them up inside the test harness only (mirroring what the consumer would do in production) so we can assert message receipt.
- **Rationale**: Matches `file/RABBITMQ_FILE_MODULE_DESIGN.md` §6: publishers own exchanges, consumers own queues.
- **Alternatives considered**: Publisher declares queue too → rejected because it couples this feature to downstream worker topology.

---

## R8. Idempotency cache layout

- **Decision**:
  - Key: `file:init:idem:{ownerType}:{ownerId}:{sha256(key)}`. For **Admin** callers `{ownerId}` is the admin's authenticated user_id (the DB `owner_id` column is NULL for PLATFORM; the Redis key uses the user_id purely for per-admin namespace isolation).
  - Value: `fileId` string.
  - TTL: `uploadExpiryMinutes * 60 + 300` (5-minute buffer, covered by FR-034).
  - Write semantics: `SETNX` on the key → if lost, GET the winning `fileId`, compare the request fingerprint, reuse or return 409 per FR-033.
- **Rationale**: Prevents duplicate `file_object` insertion under concurrent retries; hashing the raw key prevents header-value leakage into Redis keys and caps key length.
- **Alternatives considered**:
  - Store full request body hash under the key → rejected (too large); store a separate "fingerprint" field in a hash instead.
  - DB-level unique index on `(owner_id, idempotency_key)` → rejected (persistent noise; Redis TTL is enough given client semantics).
- **Implementation detail**: The value is stored as a small JSON `{"fileId":"...","fingerprint":"sha256:..."}` in a Redis hash to let us cheaply validate fingerprint on repeat.

---

## R9. Object-key template

- **Decision**:
  - Template: `seller/{sellerId}/{purpose}/{yyyy}/{mm}/{fileId}-{sanitizedFilename}`.
  - For admins: `platform/{purpose}/{yyyy}/{mm}/{fileId}-{sanitizedFilename}`.
  - Sanitiser: lowercase, strip/replace characters outside `[A-Za-z0-9._-]`, collapse whitespace to `-`, truncate to 120 bytes, prefix with `file` if empty after sanitisation.
- **Rationale**: Matches the design doc, makes cross-tenant scanning in S3 trivially auditable, and guarantees URL safety for the presigned PUT.
- **Alternatives considered**: Use only `fileId` in the key → rejected because humans inspecting the bucket benefit from the original filename hint.

---

## R10. Testcontainer orchestration per suite

- **Decision**:
  - One shared `UploadSuite` (testify) per package that boots Postgres + Redis + MinIO + RabbitMQ **once** via `SetupSuite`, with per-test DB table truncation rather than container reuse across packages.
  - Suite seeds: one seller, one admin, one customer, one platform storage config (MinIO-backed), one seller storage config + binding, one product (used as `owner_id` if needed later).
- **Rationale**: Container boot is the slowest operation (~5–15 s per container); amortise across all upload tests. Truncate-per-test preserves isolation and is already the pattern in existing suites.
- **Alternatives considered**: Per-test containers → rejected (10× slowdown).

---

## R11. Error-code inventory (pulled from spec)

The following `AppError` values are finalised:

| Code | HTTP | Trigger |
|---|---|---|
| `FILE_UPLOAD_UNAUTHORIZED` | 401 | Missing/expired token |
| `FILE_UPLOAD_FORBIDDEN` | 403 | Customer role or role missing |
| `FILE_UPLOAD_INVALID_INPUT` | 400 | Validation failure on request body |
| `FILE_UPLOAD_POLICY_VIOLATION` | 422 | Size/mime/purpose policy rejected |
| `FILE_UPLOAD_STORAGE_UNAVAILABLE` | 503 | Blob adapter factory or presign failed |
| `FILE_UPLOAD_NO_STORAGE_CONFIG` | 412 | Seller has no binding and no platform default |
| `FILE_UPLOAD_NOT_FOUND` | 404 | `fileId` not found OR cross-tenant access |
| `FILE_UPLOAD_CONFLICT` | 409 | Idempotency fingerprint mismatch |
| `FILE_UPLOAD_OBJECT_MISSING` | 409 | `complete-upload` before provider has the object |
| `FILE_UPLOAD_OBJECT_MISMATCH` | 422 | Etag/size/mime mismatch on head |
| `FILE_UPLOAD_ALREADY_FINALIZED` | 200 (idempotent) | Re-complete on `ACTIVE` row → return current state, no error |
| `FILE_UPLOAD_INTERNAL` | 500 | Unhandled errors; always secret-stripped |

All codes live in `file/utils/constant/upload_constants.go` as typed constants and in `file/error/upload_errors.go` as `*AppError` singletons.

---

## R12. Performance envelope

- **Decision**: Track SC-001/SC-007 by adding a benchmark-ish integration test `upload_performance_test.go` (opt-in via `-short=false` + build tag `perf`) that times a 1 MB happy-path round-trip and asserts ≤ 3 s on CI hardware. No additional tooling (no profiling harness) is introduced in this slice.
- **Rationale**: Keeps the CI cost bounded while still exercising the success criterion.
- **Alternatives considered**: Add pprof endpoints and a load test → rejected as out of scope.

---

## R13. Existing code already in place (do not reinvent)

| Capability | Existing asset | How we consume it |
|---|---|---|
| Blob adapter factory | `file/service/blob_adapter/*` + storage config resolver in `file/service/config_service.go` | Inject into `FileUploadService` via service factory |
| Correlation ID middleware | `common/middleware/correlation_id.go` | Already applied by test server |
| RBAC middleware | `common/middleware/auth.go` (`SellerAuth`, `AdminAuth`) | Applied at route registration |
| Structured response helpers | `common/response.go` | Used by handler for 2xx + 4xx |
| AppError / IsAppError | `common/error/` | Service returns, handler maps to HTTP |
| Redis scheduler | `common/scheduler` | Direct reuse: `Schedule`, `Cancel`, `RegisterHandler` |
| Messaging publisher | `common/messaging/rabbitmq/publisher.go` + `common/messaging/envelope.go` | New `upload_variant_publisher.go` wraps it |
| Integration setup | `test/integration/setup/*` + `test/integration/file/*` | Extended with MinIO/RabbitMQ helpers |

No duplicated functionality.
