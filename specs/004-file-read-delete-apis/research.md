# Research: File Read & Delete APIs (004)

**Feature**: `004-file-read-delete-apis`
**Branch**: `004-file-read-delete-apis`
**Date**: 2026-05-16
**Status**: Complete — all NEEDS CLARIFICATION resolved

---

## R1 — Batch List Query Pattern

**Decision**: Follow the inventory module `GetInventoriesParam` → `ToFilter()` → `GetInventoriesFilter` pipeline exactly. Use comma-separated `*string` query params, parse in `ToFilter()`, validate enums in the service layer.

**Rationale**: This pattern is already production-proven in the inventory module and constitutionally mandated (Pattern Consistency principle). Consistent filtering across modules reduces cognitive load for future maintainers.

**Alternatives considered**: JSON-encoded filter body in POST — rejected because REST convention for reads is GET; GraphQL-style query — out of scope for this monolith.

---

## R2 — Presigned Download URL TTL Range

**Decision**: Min 5 min, max 60 min, default 15 min. Enforced for both `GET /api/files/{fileId}` (`urlTtlMinutes`) and `GET /api/files/{fileId}/download-url` (`ttlMinutes`).

**Rationale**: Pre-spec OQ-5 decisions. Short minimum prevents URL farming; 60-minute max matches S3/MinIO STS session limits without requiring long-lived credentials. Consistent with upload expiry range (5–60 min).

**Alternatives considered**: Longer TTLs (1h+) — rejected due to URL farming risk; no TTL control — rejected because seller use cases vary (inline display vs. share links).

---

## R3 — Presigned URL `disposition` Parameter Encoding

**Decision**: `disposition` is a query param (`inline` | `attachment`), encoded into `BlobPresignDownloadInput.Disposition` field. Each adapter builds `response-content-disposition` as a signed query parameter in the presigned URL. No `Content-Disposition` header set by our server.

**Rationale**: Pre-spec OQ-5: query param avoids CDN caching issues. S3/MinIO, GCS, and Azure all support `response-content-disposition` in presigned URL parameters. This moves enforcement to the provider's response, not our server.

**Alternatives considered**: HTTP response header — rejected (breaks CDN); separate endpoint — unnecessary complexity.

---

## R4 — `BlobPresignDownloadInput` Extension

**Decision**: Add `Disposition string` field to the existing `model.BlobPresignDownloadInput` struct (currently only `Bucket`, `Key`, `TTL`). Update all three adapters (S3-compatible, GCS, Azure) to encode the disposition into the presigned URL parameters. Default is `inline` (no parameter needed for most providers).

**Rationale**: The existing model is used only internally; adding a field is backward-compatible. Each adapter already knows how to construct provider-specific presigned URL parameters.

**Alternatives considered**: New input type — unnecessary duplication.

---

## R5 — Tenant Scoping in Batch List

**Decision**: Reuse `FindByFileIDScoped` for single-ID lookups. For batch list (`GetAllFiles`), a new `FindManyScoped` repository method returns `[]FileObject` filtered by `(owner_type, owner_id)` with optional `IN` clauses for `file_id`, `purpose`, `status`, `mime_type`. Max 100 `fileId` entries enforced before DB call.

**Rationale**: Single unified query avoids N+1. `IN` clause is safe for up to 100 items per pre-spec constraint. Cross-tenant IDs are silently omitted — matching row-level security via scoped WHERE clause.

**Alternatives considered**: N individual FindByFileIDScoped calls — rejected (N+1, O(N) round-trips). Application-level filtering after fetching all seller files — rejected (unbounded result set).

---

## R6 — Variant Fetching Strategy (N+1 Prevention)

**Decision**: For `GetAllFiles` with `includeVariants=true`, use a single batch query: `SELECT * FROM file_variant WHERE file_object_id IN (...)` after fetching the main page, then stitch variants into each item in memory.

For `GetFile` (single), use GORM `Preload("Variants")` or a LEFT JOIN.

**Rationale**: Constitution §X mandates using `Preload()` or `Joins()` to avoid N+1. The batch-then-stitch approach for list is O(2) queries regardless of page size, consistent with GORM preload semantics.

**Alternatives considered**: N individual variant queries — rejected (N+1). Subquery — overly complex for this use case.

---

## R7 — Delete Atomicity: Blob Before DB Row

**Decision**: The delete sequence is strictly ordered:
1. Fetch file row + tenant check
2. Cancel scheduler job (UPLOADING path, best-effort)
3. Delete original blob (BlobAdapter.DeleteObject) — abort on non-idempotent error
4. Delete variant blobs (best-effort, log failures)
5. Hard DELETE the DB row (cascade deletes `file_variant` + `file_job`)
6. Write audit log

**Rationale**: Pre-spec CC-007 and FR-018. If blob delete fails, the DB row stays so the caller can retry or the blob can be manually cleaned. The reverse order (DB first) would leave orphan blobs with no registry entry, making cleanup impossible. This matches the "cleanup debt is worse than user-visible retry" principle.

**Alternatives considered**: Transactional outbox pattern — overkill for synchronous hard-delete; soft-delete to `DELETED` status — pre-spec explicitly rejects this (no DELETED status in v1).

---

## R8 — Storage Provider Label Derivation

**Decision**: Derive the human-readable `storageProvider` string from `storage_config.provider` → `Provider.AdapterType` (e.g., `S3`, `GCS`, `AZURE`). Never expose `bucket_or_container`, `config_data`, or any credential field in any API response.

**Rationale**: Pre-spec CC-002 and FR-025. The `storage_config` entity already carries `AdapterType`; this is read from the resolved config at query time.

**Alternatives considered**: Store provider label redundantly on `file_object` — rejected (denormalization; source of truth is `storage_config`).

---

## R9 — Error Code Namespace

**Decision**: Add a new file `file/error/file_read_delete_errors.go` with four new error singletons using the `FILE_` prefix namespace. New constant file `file/utils/constant/read_delete_constants.go`.

| Error | HTTP | Code |
|---|---|---|
| `ErrFileNotFound` | 404 | `FILE_NOT_FOUND` |
| `ErrFileNotActive` | 409 | `FILE_NOT_ACTIVE` |
| `ErrVariantNotFound` | 404 | `VARIANT_NOT_FOUND` |
| `ErrVariantNotReady` | 409 | `VARIANT_NOT_READY` |
| `ErrFileDeleteConflict` | 409 | `FILE_DELETE_CONFLICT` |
| `ErrStoragePermissionDenied` | 502 | `STORAGE_PERMISSION_DENIED` |
| `ErrStorageUnavailable` | 503 | `STORAGE_UNAVAILABLE` |

**Rationale**: Separate error file keeps upload and read/delete concerns isolated (SRP). Existing `ErrFileUploadNotFound` is upload-specific (upload flow terminology); reads/deletes get their own clean `FILE_NOT_FOUND`.

---

## R10 — `FileReadService` Interface for In-Process Calls

**Decision**: Define a `FileReadService` interface in `file/service/file_read_service.go` with a `GetFilesByIDs(ctx, []string) ([]*entity.FileObject, error)` method that bypasses tenant scoping. This is the contract Product Service injects (in-process DI). The HTTP `GET /api/files` handler uses a separate `GetAllFiles` method with tenant scoping.

**Rationale**: Pre-spec's modular monolith decision — Product Service injects `FileReadService` as a Go interface, no HTTP. The interface must be narrow (ISP) and not expose upload-specific concerns.

**Alternatives considered**: Expose the whole `FileUploadRepository` to Product Service — rejected (violates module boundary); HTTP call — explicitly rejected in pre-spec.

---

## R11 — Scheduler Cancellation on Delete (UPLOADING path)

**Decision**: Reuse the existing `UploadExpiryScheduler.Cancel(ctx, fileObjectID, sellerID)` method used by `CompleteUpload`. Failure is non-blocking — log warning and continue. The same Redis key pattern (`seller:{sellerId}:file.upload.expiry:{fileObjectId}`) is used.

**Rationale**: Reusing the existing scheduler cancel path avoids duplicating Redis key construction logic. Pre-spec §4 business logic step 3 matches exactly. Expiry handler already handles the "row not found" case gracefully (no-op).

---

## R12 — `GetFilesParam.ToFilter()` Filter Composition

**Decision**: All dimension filters (purpose, status, mime) are ANDed across dimensions; values within a dimension are ORed using GORM `IN (?)` clauses. `Statuses` defaults to `["ACTIVE"]` when omitted.

```go
// Pseudo-query
WHERE owner_type=? AND owner_id=?
  AND (purpose IN (?,?,...))         -- if Purposes non-empty
  AND (status IN (?,?,...))          -- always (default ACTIVE)
  AND (mime_type IN (?,?,...))       -- if MimeTypes non-empty
  AND (file_id IN (?,?,...))         -- if FileIDs non-empty (batch)
ORDER BY <sortBy> <sortOrder>
LIMIT <pageSize> OFFSET <offset>
```

**Rationale**: Matches pre-spec FR-008 and the inventory module's established filter composition. GORM's `IN` is safe for up to 100 values (pre-spec max).

---

## R13 — Integration Test Infrastructure

**Decision**: Reuse existing `test/integration/file/` suite infrastructure: `setup_upload_suite_test.go` (PostgreSQL + MinIO + RabbitMQ via Testcontainers), `minio_container.go`, `blob_test_helpers.go`, `upload_test_helpers_test.go`. New test files extend the existing `UploadSuite` or introduce a new `FileOpsSuite` within the same package.

**Rationale**: Constitution §IV mandates Testcontainers-based integration tests. The 003-upload-apis suite already has MinIO and Postgres wired; reusing it avoids duplicating container setup. Seeding happens via real `POST /api/files/init-upload` + `POST /api/files/complete-upload` API calls.

---

## R14 — No New DB Migration Required

**Decision**: Zero migration files needed for this feature. Existing schema (`file_object`, `file_variant`, `file_job`) and FK constraints (`ON DELETE CASCADE`) fully support read and hard-delete operations.

**Rationale**: Pre-spec data model section is explicit: no new columns, no new enum values, no new tables.

---

## Summary: All Clarifications Resolved

| Item | Resolution |
|---|---|
| Batch query pattern | Inventory module `Param → Filter` pattern |
| TTL range | 5–60 min, default 15 |
| disposition param | Query param, encoded in presigned URL |
| `BlobPresignDownloadInput` | Add `Disposition string` field |
| Batch tenant scoping | New `FindManyScoped` repo method |
| Variant N+1 prevention | Batch-then-stitch (list) / Preload (single) |
| Delete ordering | Blob before DB row; abort on blob failure |
| storageProvider label | From `storage_config.AdapterType` |
| Error namespace | New `file_read_delete_errors.go` |
| In-process interface | `FileReadService` with `GetFilesByIDs` |
| Scheduler cancel | Reuse `UploadExpiryScheduler.Cancel` |
| Filter composition | AND across dims, OR within dim |
| Test infra | Reuse existing upload suite containers |
| Migration | None required |
