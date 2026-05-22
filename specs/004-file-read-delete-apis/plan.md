# Implementation Plan: File Read & Delete APIs (004)

**Branch**: `004-file-read-delete-apis` | **Date**: 2026-05-16 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `specs/004-file-read-delete-apis/spec.md`

---

## Summary

Implement the **read and delete** half of the file module, completing the CRUD lifecycle started by `003-upload-apis`. Four HTTP endpoints are delivered:

1. `GET /api/files` — Batch list/filter files (seller dashboard + in-process product listing resolution)
2. `GET /api/files/{fileId}` — Single file metadata with optional embedded presigned download URL
3. `GET /api/files/{fileId}/download-url` — Generate standalone short-lived presigned download URL
4. `DELETE /api/files/{fileId}` — Synchronous hard-delete: blob + DB row

Additionally, a `FileReadService` in-process Go interface is defined for Product Service injection (no HTTP, no auth middleware involved).

All implementation follows the existing `file` module architecture: Clean Architecture layers, Factory-Singleton DI, Testcontainers integration-first TDD. **Zero new migrations** are required.

---

## Technical Context

**Language/Version**: Go 1.25+
**HTTP Framework**: Gin (`github.com/gin-gonic/gin`)
**Database**: PostgreSQL 16 via GORM — existing `file_object`, `file_variant`, `file_job` tables
**Cache**: Redis 7 — existing scheduler key pattern for expiry cancellation on delete
**Blob Storage**: MinIO (S3-compatible) / GCS / Azure via existing `BlobAdapter` interface
**Testing**: Testcontainers (PostgreSQL + MinIO + RabbitMQ) — reuse `003-upload-apis` suite
**Target Platform**: Linux server (single binary modular monolith)
**Performance Goals**: p95 ≤ 100ms batch list (20 items), ≤ 200ms single get, ≤ 500ms presign, ≤ 300ms delete
**Constraints**: No new migrations; no `GET /internal/files` HTTP endpoint; no new container setup
**Scale/Scope**: Extends existing file module; adds ~8 new Go files + 4 test files

---

## Constitution Check

*GATE: Must pass before implementation. Re-check after Phase 1 design.*

| Principle | Status | Evidence |
|---|---|---|
| I. Modular Monolith — no cross-module repo access | ✅ PASS | `FileReadService` interface used for Product Service; no direct repo injection |
| II. Clean Architecture — Handler → Service → Repository → DB | ✅ PASS | New `FileReadService`, `FileDeleteService`, extended `FileRepository` follow layered pattern |
| III. Factory-Singleton DI | ✅ PASS | New services registered in `ServiceFactory.initialize()`; new handler in `HandlerFactory` |
| IV. TDD Integration-First | ✅ PASS | All test scenarios T00–T67 are integration tests; Testcontainers reused from 003 |
| V. Multi-Tenant Seller Isolation | ✅ PASS | All queries scoped by `(owner_type, owner_id)`; cross-tenant → 404 |
| VI. Correlation ID Propagation | ✅ PASS | Missing `X-Correlation-ID` → 400 on all 4 endpoints |
| VII. RBAC | ✅ PASS | `sellerAuth` middleware on all routes; buyer → 403 |
| VIII. Backward Compatibility | ✅ PASS | No existing endpoints modified; no migration changes |
| IX. SOLID | ✅ PASS | `FileReadService` and `FileDeleteService` are narrow, SRP-compliant interfaces |
| X. Performance | ✅ PASS | Batch-then-stitch variant fetch prevents N+1; GORM Preload for single-file |

**No constitution violations. Ready for implementation.**

---

## Project Structure

### Documentation (this feature)

```text
specs/004-file-read-delete-apis/
├── plan.md              # This file
├── research.md          # Phase 0 output (all NEEDS CLARIFICATION resolved)
├── data-model.md        # Phase 1 output (no new migrations; query contracts)
├── contracts/
│   └── api-contracts.md # Phase 1 output (all 4 endpoint contracts + in-process interface)
├── pre-spec.md          # Original pre-spec (reference)
├── spec.md              # Feature specification (source of truth)
└── tasks.md             # Phase 2 output (/speckit.tasks — NOT created by /speckit.plan)
```

### Source Code Layout

```text
file/
├── container.go                         # [EXISTING] — no changes
├── entity/
│   └── file.go                          # [EXISTING] — no changes; FileObject, FileVariant, FileJob
├── error/
│   ├── upload_errors.go                 # [EXISTING] — no changes
│   ├── blob_adapter_errors.go           # [EXISTING] — no changes
│   └── file_read_delete_errors.go       # [NEW] ErrFileNotFound, ErrFileNotActive, ErrVariantNotFound,
│                                        #       ErrVariantNotReady, ErrFileDeleteConflict,
│                                        #       ErrStoragePermissionDenied, ErrStorageUnavailable
├── model/
│   ├── upload_model.go                  # [EXISTING] — no changes
│   ├── blob_adapter_model.go            # [MODIFY] Add Disposition field to BlobPresignDownloadInput
│   └── file_read_delete_model.go        # [NEW] GetFilesBase, GetFilesParam, GetFilesFilter,
│                                        #       GetFilesResponse, FileItem, FileVariantItem,
│                                        #       GetFileResponse, DownloadURLResponse, DeleteFileResponse
├── repository/
│   ├── file_repository.go               # [MODIFY] Extend interface: FindManyScoped,
│   │                                    #   FindVariantsByFileObjectIDs, FindVariantByCode, DeleteFileObject
│   └── file_repository_impl.go         # [MODIFY] Implement new repo methods
├── service/
│   ├── upload_service.go                # [EXISTING] — no changes
│   ├── file_read_service.go             # [NEW] FileReadService interface + fileReadService impl
│   │                                    #       (GetAllFiles, GetFile, GetDownloadURL, GetFilesByIDs)
│   └── file_delete_service.go           # [NEW] FileDeleteService interface + fileDeleteService impl
│                                        #       (DeleteFile)
├── handler/
│   ├── file_upload_handler.go           # [EXISTING] — no changes
│   └── file_handler.go                  # [MODIFY] Implement GetAllFiles, GetFile, GetDownloadURL,
│                                        #          DeleteFile (stubs → real implementations)
├── factory/singleton/
│   ├── repository_factory.go            # [EXISTING] — no changes (FileUploadRepository covers all)
│   ├── service_factory.go               # [MODIFY] Add FileReadService and FileDeleteService singletons
│   ├── handler_factory.go               # [MODIFY] Wire FileHandler with read/delete services
│   └── singleton_factory.go             # [MODIFY] Expose GetFileReadService, GetFileDeleteService
├── route/
│   └── file_operation_route.go          # [MODIFY] Add GET /api/files route (GetAllFiles)
├── utils/constant/
│   ├── upload_constants.go              # [EXISTING] — no changes
│   └── read_delete_constants.go         # [NEW] Error codes/messages, TTL bounds, success messages
└── service/blobAdapter/
    ├── adapter.go                        # [EXISTING] — no changes to interface
    ├── s3_compatible_adapter.go         # [MODIFY] Encode Disposition in PresignDownload
    ├── gcs_adapter.go                   # [MODIFY] Encode Disposition in PresignDownload
    └── azure_blob_adapter.go            # [MODIFY] Encode Disposition in PresignDownload

test/integration/file/
├── setup_upload_suite_test.go           # [EXISTING] — reused; may add GetAllFiles endpoint const
├── minio_container.go                   # [EXISTING] — reused
├── blob_test_helpers.go                 # [EXISTING] — reused
├── upload_test_helpers_test.go          # [EXISTING] — reused for seeding via API
├── get_all_files_test.go                # [NEW] T00–T00t (batch list + auth + validation)
├── get_file_test.go                     # [NEW] T01–T14 (single get + presign + auth + degraded)
├── download_url_test.go                 # [NEW] T20–T40 (download-url + variants + errors)
└── delete_file_test.go                  # [NEW] T50–T67 (delete + cascade + scheduler + concurrency)
```

---

## Phase 0: Research — COMPLETE

All unknowns resolved. See [research.md](./research.md) for full rationale.

**Key decisions:**
- Batch list uses `GetFilesParam → ToFilter()` → `FindManyScoped` (inventory module pattern)
- Variant N+1 prevented by batch-then-stitch (list) and GORM Preload (single)
- Delete sequence: blob first, DB row second; abort on blob failure
- `BlobPresignDownloadInput` extended with `Disposition string` field
- `FileReadService` in-process interface for Product Service (no HTTP)
- New error file `file_read_delete_errors.go`; no changes to upload errors
- Zero new migrations

---

## Phase 1: Design & Contracts — COMPLETE

### Data Model
See [data-model.md](./data-model.md):
- No new entities or migrations
- 4 new repository methods: `FindManyScoped`, `FindVariantsByFileObjectIDs`, `FindVariantByCode`, `DeleteFileObject`
- Filter model: `GetFilesParam`, `GetFilesFilter` following inventory pattern

### API Contracts
See [contracts/api-contracts.md](./contracts/api-contracts.md):
- All 4 HTTP endpoints fully specified with request/response shapes
- In-process `FileReadService` interface defined
- Full error code reference

### Constitution Check (Post-Design)
All 10 principles verified above. ✅ No violations.

---

## Complexity Tracking

No constitution violations to justify. Design is straightforward extension of existing patterns.

---

## Next Step

Run `/speckit.tasks` to generate the dependency-ordered `tasks.md` implementation checklist.
