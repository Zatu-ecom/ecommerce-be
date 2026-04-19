# Implementation Plan: File Upload APIs (Init + Complete)

**Branch**: `003-upload-apis` | **Date**: 2026-04-18 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-upload-apis/spec.md`

## Summary

Deliver the two-step presigned upload flow for the `file` module:
`POST /api/files/init-upload` and `POST /api/files/complete-upload`, plus a Redis-scheduled abandoned-upload cleanup job and a RabbitMQ command emitted on completion for image-variant generation. Built on the existing blob adapter layer (`file/service/blob_adapter`), storage config resolution, RabbitMQ messaging (`common/messaging`), and the inventory-style Redis scheduler (`common/scheduler`). End-to-end integration tests (Testcontainers: Postgres + MinIO + RabbitMQ + Redis) are primary; unit tests only for pure helpers (key sanitiser, policy evaluator, envelope marshal).

## Technical Context

**Language/Version**: Go 1.25+ (per constitution)
**Primary Dependencies**:
- HTTP: Gin (`github.com/gin-gonic/gin`)
- DB: GORM + PostgreSQL 16 (`gorm.io/gorm`, `gorm.io/driver/postgres`)
- Cache + Scheduler: `github.com/go-redis/redis/v8`, `common/scheduler`
- Messaging: RabbitMQ (`common/messaging` + `common/messaging/rabbitmq`)
- Blob SDKs (indirectly via `file/service/blob_adapter`): AWS S3 v2, Azure Blob, GCS
- Logging: `common/log` (logrus)
- UUID / ULID: `github.com/google/uuid` (existing) + `oklog/ulid/v2` (new direct dep, see research)

**Storage**:
- Postgres 16 (`file_object`, `file_variant`, `file_job`, existing `storage_config` / `seller_storage_binding` / `storage_provider`)
- Redis 7 (scheduler sorted set `delayed_jobs`, cancellation cache, idempotency record)
- Blob providers (S3-compatible / GCS / Azure Blob) through existing adapters

**Testing**: Go `testing` + `stretchr/testify/suite` + Testcontainers (Postgres, Redis, MinIO, RabbitMQ). Extends `test/integration/setup` and `test/integration/file/`.
**Target Platform**: Linux container (Go HTTP service, monolith).
**Project Type**: Web service (modular monolith). Work isolated to the `file` module + tests; `common/messaging` gains one new constant; `common/scheduler` stays unchanged.
**Performance Goals**:
- p95 `init-upload` ≤ 300 ms (local stack, SC-007).
- p95 `complete-upload` ≤ 500 ms.
- 1 MB end-to-end journey ≤ 3 s on local stack (SC-001).

**Constraints**:
- Single-part upload only, ≤ 50 MB per request.
- Presigned URL TTL = `uploadExpiryMinutes` (5–60, default 15).
- Must not expose provider credentials or endpoints in any error body (SC-006).
- Zero cross-tenant finalisations (SC-005).
- Must reuse existing storage config resolver (from feature 001) and blob adapter factory (feature 002); no forks.

**Scale/Scope**:
- Phase 1 scope: seller + admin clients, existing volume (~hundreds of uploads/day/seller).
- Horizontal scaling safe: services stateless; scheduler is Redis-backed and multi-worker safe by design.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Verdict | Notes |
|---|---|---|
| I. Modular Monolith with Microservices DNA (NON-NEGOTIABLE) | ✅ PASS | All code lives in `file/`. Only approved cross-module touchpoints: `common/messaging`, `common/scheduler`, `common/cache`, `common/auth`, `common/middleware`, `common/response`, `common/error`. No direct access to other modules' repositories. |
| II. Clean Architecture / Layered Architecture (NON-NEGOTIABLE) | ✅ PASS | Request flow: Handler → Service → Repository → DB. Service owns storage resolver + blob adapter + scheduler calls; never DB directly. Scheduler handler (`file.upload.expiry`) calls back into service to mutate state. |
| III. Factory-Singleton Dependency Injection (NON-NEGOTIABLE) | ✅ PASS | Extend existing `file/factory/singleton/{repository_factory,service_factory,handler_factory}.go` to expose `FileUploadService`, `FileUploadHandler`, `FileUploadRepository`. No ad-hoc wiring. |
| IV. TDD & Integration-First Testing (NON-NEGOTIABLE) | ✅ PASS | Phase 1 emits `quickstart.md` and contracts so tests are written before impl. Integration tests in `test/integration/file/upload_*_test.go` drive production code. Unit tests only for pure helpers. |
| V. Multi-Tenant Seller Isolation (NON-NEGOTIABLE) | ✅ PASS | Every `file_object` query scoped by `(owner_type, owner_id, seller_id)`. FR-011, US4 enforce cross-tenant negative cases. Redis keys include tenant prefix (`seller:{sellerId}:...` or `platform:...`). |
| VI. Correlation ID & Distributed Tracing (NON-NEGOTIABLE) | ✅ PASS | Reuse existing `middleware.CorrelationID()`; propagate `X-Correlation-ID` into Redis job (`ScheduledJob.CorrelationId`) and into RabbitMQ envelope (`correlationId` field). Tests assert header requirement. |
| VII. RBAC | ✅ PASS | Routes registered under `SellerAuth()` for sellers and a separate admin-mounted route group under `AdminAuth()` (both delegate to the same service with explicit role-based owner resolution). Customers rejected by middleware chain → `403`. |
| VIII. Backward Compatibility (NON-NEGOTIABLE) | ⚠️ JUSTIFIED EDIT | `metadata JSONB` will be removed from `FileObject` / `FileVariant` entities and dropped from `migrations/018_create_file_storage_tables.sql` **only because** that migration has not been merged to `develop` yet (see Complexity Tracking). No downstream caller references `metadata` (verified via grep). No breaking change to any released API contract. |
| IX. SOLID | ✅ PASS | New interfaces kept ≤ 10 methods: `FileUploadService`, `FileUploadRepository`, `UploadExpiryScheduler`. Blob adapter and storage resolver reused via DIP. |
| X. Performance & Scalability | ✅ PASS | All DB ops in single transactions; no N+1 (single SELECT by `file_id`, single UPDATE, single INSERT for `file_job`). Redis scheduler backed by sorted set with O(log N). Streaming not needed in this slice (no body through the API). Stateless handlers. |

**Gate**: PASS (with one justified edit tracked in Complexity Tracking).

## Project Structure

### Documentation (this feature)

```text
specs/003-upload-apis/
├── plan.md                 # This file (/speckit.plan output)
├── research.md             # Phase 0 output
├── data-model.md           # Phase 1 output
├── quickstart.md           # Phase 1 output (integration-test playbook)
├── contracts/              # Phase 1 output
│   ├── init-upload.http.md
│   ├── complete-upload.http.md
│   ├── file.upload.expiry.job.md
│   └── file.image.process.requested.event.md
├── spec.md                 # Feature specification (input)
└── checklists/requirements.md
```

### Source Code (repository root)

Work is confined to the `file` module plus test fixtures. Directories with ➕ are new files to be created; others are existing and will be edited.

```text
file/
├── container.go                                  # (edit) mount new admin-auth route group; no other change
├── entity/
│   └── file.go                                   # (edit) remove Metadata field from FileObject & FileVariant
├── model/
│   ├── blob_adapter_model.go                     # existing
│   ├── config_model.go                           # existing
│   └── upload_model.go                           # ➕ init/complete request + response DTOs
├── repository/
│   ├── config_repository.go                      # existing
│   ├── file_repository.go                        # (edit) populate the currently-empty interface + impl
│   └── file_repository_impl.go                   # ➕ (split if file_repository.go > 500 lines)
├── service/
│   ├── blob_adapter/*                            # existing, untouched
│   ├── config_service.go                         # existing
│   ├── file_service.go                           # (edit) populate; delegate to sub-services below
│   ├── upload_service.go                         # ➕ init + complete orchestration
│   ├── upload_policy.go                          # ➕ pure: size/mime/purpose policy evaluator
│   ├── upload_key_builder.go                     # ➕ pure: deterministic object-key builder + sanitiser
│   ├── upload_expiry_scheduler.go                # ➕ Redis scheduler wrapper (mirrors inventory pattern)
│   ├── upload_expiry_handler.go                  # ➕ scheduler.Handler for "file.upload.expiry"
│   └── upload_variant_publisher.go               # ➕ RabbitMQ publisher for file.image.process.requested
├── handler/
│   ├── config_handler.go                         # existing
│   ├── file_handler.go                           # (edit) implement InitUpload + CompleteUpload stubs
│   └── (no new file)
├── route/
│   ├── file_operation_route.go                   # (edit) add admin-auth variant + keep seller routes
│   ├── import_export_routes.go                   # existing
│   └── storage_config_routes.go                  # existing
├── factory/singleton/
│   ├── repository_factory.go                     # (edit) expose FileUploadRepository
│   ├── service_factory.go                        # (edit) expose FileUploadService + dependencies
│   ├── handler_factory.go                        # (edit) expose FileUploadHandler
│   └── singleton_factory.go                      # (edit) facade
├── utils/constant/
│   └── upload_constants.go                       # ➕ error codes, scheduler command, routing key, cache keys
├── error/
│   └── upload_errors.go                          # ➕ AppError definitions for all upload 4xx/5xx codes
└── messaging/
    └── contracts.go                              # ➕ ImageProcessRequested payload struct (per RabbitMQ design doc §6)

common/messaging/
└── (no code change; `constants/messaging_constants.go` gains "file.image.process.requested" routing key + "ecom.commands" exchange constants if not present)

migrations/
└── 018_create_file_storage_tables.sql            # (edit) append CREATE TABLE file_object / file_variant / file_job + indexes; drop metadata column from entity design

test/integration/setup/
├── container.go                                  # (edit) add MinIO + RabbitMQ testcontainers bootstrap helpers reused from file/
├── database.go                                   # (edit) ensure migration 018 runs (already does by listing order)
└── server.go                                     # (edit) register file module messaging/scheduler bindings during test server init

test/integration/helpers/
└── upload_helper.go                              # ➕ seller/admin upload journey helpers (init → PUT → complete, retries, expiry waits)

test/integration/file/
├── upload_init_test.go                           # ➕ US1 (P1) init happy path + policy variants
├── upload_complete_test.go                       # ➕ US1/US2/US5 complete happy + mismatch + idempotency
├── upload_policy_test.go                         # ➕ US3 policy enforcement
├── upload_tenant_isolation_test.go               # ➕ US4 + admin-vs-seller scoping
├── upload_expiry_test.go                         # ➕ US6a scheduled cleanup (short TTL)
├── upload_outage_test.go                         # ➕ US6 storage outage + RabbitMQ-outage (edge case)
├── upload_idempotency_test.go                    # ➕ US5a Idempotency-Key
├── upload_variant_publish_test.go                # ➕ end-to-end variant message assertions via test consumer
└── setup_upload_suite_test.go                    # ➕ testify.Suite with MinIO + RabbitMQ + Redis + Postgres
```

**Structure Decision**: Continue the modular-monolith layout. All new source files live under `file/`; all new tests live under `test/integration/file/` and `test/integration/helpers/`. No sibling module is touched. Existing patterns (factory singleton, constants, error codes, suite-based tests) are reused.

## Phase 0: Research

Research is captured in `research.md`. Key open questions at the start of Phase 0:

1. ULID library choice for `fileId` generation (or reuse UUIDv7).
2. RabbitMQ Testcontainer wiring (image, wait strategy, integration with existing `common/messaging` connection manager).
3. MinIO Testcontainer reuse (confirm `test/integration/file/minio_container.go` exposes what we need).
4. How the existing `common/scheduler` handler registry dispatches — do workers need to be started by the test harness?
5. Admin-auth route group policy — is there an existing admin-mount pattern for the file module or do we create one?
6. Exact DB column set required (we remove `metadata` but must still match entity for GORM auto-migrate safety).
7. RabbitMQ exchange declaration responsibility — publisher-side or consumer-side? (Affects first-time test runs.)

All decisions and rationale: [research.md](./research.md).

## Phase 1: Design & Contracts

**Prerequisites**: `research.md` complete (it is).

Generated artifacts:

1. **Data model** — [data-model.md](./data-model.md)
   - Final columns for `file_object`, `file_variant`, `file_job` (aligned with entity, `metadata` removed).
   - Status state machine with guards (init → UPLOADING; complete+verify → ACTIVE; complete-not-found → stays UPLOADING; size/mime mismatch → FAILED; scheduler → FAILED/UPLOAD_EXPIRED; idempotent re-complete).
   - Indexes: `(seller_id, created_at DESC)`, `(purpose, status)`, `(owner_type, owner_id)`, `UNIQUE(file_id)`.
   - Redis keys and TTL table.

2. **Contracts** — [contracts/](./contracts/)
   - `init-upload.http.md` — request/response/error JSON examples for every branch (FR-001…FR-009, FR-025, FR-030…FR-035).
   - `complete-upload.http.md` — request/response/error JSON examples for every branch (FR-010…FR-019, FR-026–FR-029, FR-033 fall-through).
   - `file.upload.expiry.job.md` — scheduled job payload, handler semantics, idempotency guarantee (FR-027, FR-029).
   - `file.image.process.requested.event.md` — RabbitMQ envelope + payload (FR-018), routing key, exchange.

3. **Quickstart** — [quickstart.md](./quickstart.md)
   - Exact commands to run the upload test suite locally.
   - Seed steps for `storage_provider` / `storage_config` / `seller_storage_binding` inside tests.
   - How to assert: DB state, MinIO object, Redis job presence/absence, RabbitMQ message delivery.

4. **Agent context update**
   - Run `.specify/scripts/bash/update-agent-context.sh codex` at the end of Phase 1 to patch `AGENTS.md` with the new capabilities (Redis scheduler reuse, MinIO + RabbitMQ + Redis containers in the file integration suite, new DTOs and constants).

Re-evaluate Constitution Check after Phase 1 design: all gates still PASS; the sole `⚠️ JUSTIFIED EDIT` (Principle VIII) remains bounded to the unmerged migration and the entity struct.

## Complexity Tracking

> Fill ONLY if Constitution Check has violations that must be justified.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| Editing `migrations/018_create_file_storage_tables.sql` in place (normally principle VIII forbids editing applied migrations). | The migration has **not** been merged to `develop` yet; it was introduced in feature 002 on a branch that is still open. Creating a "drop `metadata` column" follow-up migration would pollute history for a change that logically belongs to the original table design. | A new `019_drop_file_metadata.sql` was rejected because (a) it creates a DO/UNDO churn on an unreleased schema, (b) the column has never been read/written by any code or test, and (c) no prod DB has applied migration 018 yet. If migration 018 has already landed on `develop` by the time this plan executes, this plan must be revised to add a forward migration instead. |

---

## Stop Point

Phase 2 (tasks generation) is **not** produced by this command; it is produced by `/speckit.tasks`. This plan, `research.md`, `data-model.md`, `quickstart.md`, and `contracts/*` are the full output of `/speckit.plan`.
