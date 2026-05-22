# Pre-Spec: File Read & Delete APIs (004)

**Feature Branch**: `004-file-read-delete-apis`
**Created**: 2026-05-14
**Status**: Pre-Spec (Draft)
**Depends On**: `003-upload-apis` (file_object, FileStatus, BlobAdapter, error patterns)

---

## Overview

This feature delivers the **read and delete** half of the file module:

1. **`GET /api/files`** — Batch list/filter files; primary call from Product Service to resolve multiple file IDs in one request.
2. **`GET /api/files/{fileId}`** — Returns file metadata + an optional short-lived signed download URL when the file is private.
3. **`GET /api/files/{fileId}/download-url`** — Generates a dedicated short-lived presigned download URL (5–15 min TTL, configurable per request).
4. **`DELETE /api/files/{fileId}`** — Hard-deletes the file: removes the blob from storage and the row from the DB.

These APIs complete the CRUD lifecycle that began with `003-upload-apis`. All endpoints use `sellerAuth` middleware and share the same tenant-scoping, blob adapter layer, and structured error envelope.

---

## Inter-Service Call Design Decision

### Problem
Product Service needs file metadata when serving the consumer-facing Get Products (home page) API. The consumer's JWT has role `BUYER`, which is explicitly blocked (`403`) on all File Service endpoints.

### Decision: In-Process Service Call (No HTTP, No Token Forwarding)

Since this is a **modular monolith** (single Go binary), Product Service MUST call File Service via **in-process Go interface** — not over HTTP. This means:

- No JWT or auth middleware involved; service-layer calls are trusted within the process boundary.
- Product Service injects `FileReadService` interface (same DI pattern used elsewhere in the monolith).
- The consumer's JWT never reaches File Service code.

### What NOT to Do
- ❌ Do NOT forward the consumer's JWT to File Service over HTTP.
- ❌ Do NOT add `BUYER` role to the allow-list on file endpoints (consumers should never read raw storage metadata).
- ❌ Do NOT create a separate HTTP call from Product Service to File Service in the monolith — use Go interfaces.

---

## Actors & Auth

All four HTTP endpoints use **`sellerAuth` middleware** (same as upload APIs). There is no public or buyer-accessible route.

| Caller | Auth | Allowed |
|---|---|---|
| Seller | `SELLER` JWT | Own files only (scoped by `seller_id`) |
| Admin | `ADMIN` JWT | Platform-owned files only (`owner_type=PLATFORM`) |
| Buyer/Customer | any other role | **403 FORBIDDEN** |
| Unauthenticated | — | **401 UNAUTHORIZED** |

> Admin cannot access Seller files and vice versa — always `404`, never `403` (prevents enumeration).

### Why not `PublicAPIAuth` (like product GET)?

The product module uses `PublicAPIAuth` because consumers browse product listings directly in the browser. File metadata (object keys, storage provider, ETags) is **internal infrastructure data** — consumers never need it. When Product Service needs file URLs for a product listing, it calls `FileReadService` **in-process** (Go interface injection, same binary), bypassing all HTTP middleware. The consumer's JWT never reaches the file module.

---

## API 1 — `GET /api/files` (Batch List / Filter)

### Purpose

Fetch metadata for multiple files in a single request. Used by:
- **Seller dashboard**: list own uploaded files with filters (purpose, status, mime).
- **Product Service** (in-process): resolve file IDs for a product listing page in one query instead of N separate calls.

### Route

```
GET /api/files
```

Auth: `sellerAuth` middleware (same as all other file endpoints).

### Request Headers

| Header | Required | Notes |
|---|---|---|
| `Authorization` | **Yes** | Bearer JWT (seller or admin) |
| `X-Correlation-ID` | **Yes** | Required; propagated to DB queries and logs |

### Query Parameters

Follows the same **`Param` → `Filter` + `ToFilter()`** pattern used in `inventory/model/inventory_model.go` (`GetInventoriesParam`, `GetInventoriesFilter`). Array-valued filters are received as a single comma-separated `*string` query param, then parsed into a typed slice in `ToFilter()`.

| Param | Wire Type | Required | Default | Notes |
|---|---|---|---|---|
| `fileIds` | comma-separated string | No | — | Batch lookup; max 100 entries. Results silently omit IDs not found or out of scope. |
| `purposes` | comma-separated string | No | — | Multi-value filter; e.g. `PRODUCT_IMAGE,SELLER_LOGO`. All listed purposes are ORed. |
| `statuses` | comma-separated string | No | `ACTIVE` | Multi-value filter; e.g. `ACTIVE,FAILED`. Omit param to default to `ACTIVE` only. Pass `statuses=UPLOADING,ACTIVE,FAILED` to get all. |
| `mimeTypes` | comma-separated string | No | — | Multi-value MIME filter; e.g. `image/jpeg,image/webp`. |
| `includeVariants` | bool | No | `false` | When `true`, includes `file_variant` rows nested per file item. |
| `page` | int ≥ 1 | No | `1` | 1-indexed page number. |
| `pageSize` | int [1,100] | No | `20` | Items per page. |
| `sortBy` | enum | No | `createdAt` | `createdAt` \| `sizeBytes` \| `originalFilename` |
| `sortOrder` | enum | No | `desc` | `asc` \| `desc` |

### Go Model Structure (Implementation Reference)

Follow the inventory module pattern exactly:

```go
// GetFilesBase holds scalar filter fields shared by Param and Filter.
type GetFilesBase struct {
    common.BaseListParams
    IncludeVariants bool   `form:"includeVariants" binding:"omitempty"`
    SortBy          string `form:"sortBy"          binding:"omitempty,oneof=createdAt sizeBytes originalFilename"`
    SortOrder       string `form:"sortOrder"       binding:"omitempty,oneof=asc desc"`
}

// GetFilesParam is bound directly from the HTTP query string.
// Array-valued filters use comma-separated *string (same as GetInventoriesParam).
type GetFilesParam struct {
    GetFilesBase
    FileIDs   *string `form:"fileIds"   binding:"omitempty"`
    Purposes  *string `form:"purposes"  binding:"omitempty"`
    Statuses  *string `form:"statuses"  binding:"omitempty"`
    MimeTypes *string `form:"mimeTypes" binding:"omitempty"`
}

// GetFilesFilter is the typed filter passed to the repository layer.
type GetFilesFilter struct {
    GetFilesBase
    FileIDs   []string
    Purposes  []string // []FilePurpose cast at query time
    Statuses  []string // []FileStatus cast at query time; defaults to ["ACTIVE"] if empty
    MimeTypes []string
}

// ToFilter converts the raw query param struct to a typed filter.
// Mirrors GetInventoriesParam.ToFilter() — uses helper.ParseCommaSeparatedPtr.
func (p *GetFilesParam) ToFilter() GetFilesFilter {
    filter := GetFilesFilter{GetFilesBase: p.GetFilesBase}

    if p.FileIDs != nil {
        filter.FileIDs = helper.ParseCommaSeparatedPtr[string](p.FileIDs)
    }
    if p.Purposes != nil {
        filter.Purposes = helper.ParseCommaSeparatedPtr[string](p.Purposes)
    }
    if p.Statuses != nil {
        filter.Statuses = helper.ParseCommaSeparatedPtr[string](p.Statuses)
    } else {
        filter.Statuses = []string{"ACTIVE"} // default
    }
    if p.MimeTypes != nil {
        filter.MimeTypes = helper.ParseCommaSeparatedPtr[string](p.MimeTypes)
    }

    return filter
}
```

> Validation of enum values (`FilePurpose`, `FileStatus`) happens in the repository/service layer after parsing, not in the `binding` tag, since the values are strings inside the comma-separated blob. Unknown enum values return `400 VALIDATION_FAILED`.

### Success Response — `200 OK`

```json
{
  "success": true,
  "message": "Files retrieved",
  "data": {
    "items": [
      {
        "fileId": "018f2c1a-7a3e-7b2c-b4e2-c2a9d3e80001",
        "status": "ACTIVE",
        "purpose": "PRODUCT_IMAGE",
        "visibility": "PRIVATE",
        "originalFilename": "hero.jpg",
        "mimeType": "image/jpeg",
        "sizeBytes": 524288,
        "etag": "d41d8cd98f00b204e9800998ecf8427e",
        "objectKey": "seller/42/PRODUCT_IMAGE/2026/05/018f2c1a-hero.jpg",
        "storageProvider": "S3",
        "createdAt": "2026-05-14T10:00:00Z",
        "completedAt": "2026-05-14T10:01:00Z",
        "variants": []
      }
    ],
    "pagination": {
      "page": 1,
      "pageSize": 20,
      "totalItems": 45,
      "totalPages": 3
    }
  }
}
```

> `variants` is an empty array `[]` when `includeVariants=false` or no variants exist.
> `objectKey` and `storageProvider` are internal fields; never expose bucket credentials.

### Business Logic

1. **Auth**: extract principal from JWT → derive `(ownerType, ownerId, sellerId)`.
2. **Scope**: all queries are scoped to the caller's `(owner_type, owner_id)`. A seller cannot retrieve another seller's files even if they know the `fileId`.
3. **`ToFilter()`**: handler calls `param.ToFilter()` to produce a typed `GetFilesFilter`. Enum values in `Purposes` and `Statuses` are validated against allowed constants; any unknown value → `400 VALIDATION_FAILED`.
4. **`fileIds` batch mode**: if `FileIDs` slice is non-empty, query `WHERE file_id = ANY($fileIds) AND owner_type=$ownerType AND owner_id=$ownerId`. Max 100 entries enforced before DB call. IDs not found or out of scope are silently omitted.
5. **Filter composition**: `Purposes`, `Statuses`, `MimeTypes` are ANDed across dimensions; within each array the values are ORed (`IN ($1,$2,...)`). `Statuses` defaults to `[ACTIVE]` when omitted.
6. **Variants**: if `includeVariants=true`, bulk LEFT JOIN `file_variant` on `file_object.id` and group into each item's `variants` array. Avoids N+1.
7. **Pagination**: standard offset pagination. `totalItems` reflects the filtered count before page slicing.
8. **No audit log** for reads; structured request log only (`action=listFiles`, `sellerId`, `filterCount`).

### Error Responses

| Status | Code | When |
|---|---|---|
| `400` | `VALIDATION_FAILED` | `fileIds` count > 100; invalid `pageSize`; unknown enum value in `purposes`/`statuses`; unknown `sortBy`/`sortOrder`; missing `X-Correlation-ID` |
| `401` | `UNAUTHORIZED` | Missing/invalid JWT |
| `403` | `FORBIDDEN` | Role is buyer/customer |

---

## API 2 — `GET /api/files/{fileId}`

### Purpose

Returns canonical metadata for a file and, when the file is `PRIVATE`, includes a short-lived signed download URL so the caller doesn't need a separate round-trip for display/download.

### Path Parameter

| Param | Type | Required | Notes |
|---|---|---|---|
| `fileId` | UUIDv7 string | Yes | Matches `file_object.file_id` |

### Query Parameters

| Param | Type | Required | Default | Notes |
|---|---|---|---|---|
| `includeDownloadUrl` | bool | No | `false` | When `true`, generates a presigned download URL (same TTL as API 2 default, 15 min) |
| `urlTtlMinutes` | int [5,60] | No | `15` | TTL for the embedded download URL; only used when `includeDownloadUrl=true` |

### Request Headers

| Header | Required | Notes |
|---|---|---|
| `Authorization` | **Yes** | Bearer JWT |
| `X-Correlation-ID` | **Yes** | Required; propagated to logs, blob adapter, and audit |

### Success Response — `200 OK`

```json
{
  "success": true,
  "message": "File retrieved",
  "data": {
    "fileId": "018f2c1a-7a3e-7b2c-b4e2-c2a9d3e80001",
    "status": "ACTIVE",
    "purpose": "PRODUCT_IMAGE",
    "visibility": "PRIVATE",
    "originalFilename": "hero.jpg",
    "mimeType": "image/jpeg",
    "sizeBytes": 524288,
    "etag": "d41d8cd98f00b204e9800998ecf8427e",
    "objectKey": "seller/42/PRODUCT_IMAGE/2026/05/018f2c1a-hero.jpg",
    "storageProvider": "S3",
    "createdAt": "2026-05-14T10:00:00Z",
    "completedAt": "2026-05-14T10:01:00Z",
    "variants": [
      {
        "variantCode": "thumb_200",
        "mimeType": "image/webp",
        "sizeBytes": 12800,
        "width": 200,
        "height": 200,
        "status": "READY"
      }
    ],
    "downloadUrl": "https://s3.amazonaws.com/bucket/key?X-Amz-Expires=900&...",
    "downloadUrlExpiresAt": "2026-05-14T10:16:00Z"
  }
}
```

> `downloadUrl` and `downloadUrlExpiresAt` are only present when `includeDownloadUrl=true` AND `visibility=PRIVATE`. For `PUBLIC` or `INTERNAL` files the field is omitted in v1 (public delivery via CDN is a later feature).
> `variants` array is empty `[]` when no `file_variant` rows exist yet.

### Business Logic

1. **Auth + tenant check**: extract principal from JWT → derive `(ownerType, ownerId, sellerId)` → query `file_object WHERE file_id=$1 AND owner_type=$2 AND owner_id=$3`. No match → `404 FILE_NOT_FOUND`.
2. **Status guard**: return metadata regardless of status (`UPLOADING`, `ACTIVE`, `FAILED`). Callers should check `status` in the response.
3. **Variants**: LEFT JOIN `file_variant` on `file_object.id` and include all rows.
4. **Presign (conditional)**: if `includeDownloadUrl=true` AND `status=ACTIVE` AND `visibility=PRIVATE` → call `BlobAdapter.PresignDownload` with resolved TTL. On provider error → log warning, return metadata without `downloadUrl` (non-fatal degraded mode).
5. **Storage provider label**: derive human-readable `storageProvider` from the resolved `storage_config.provider_type` (e.g. `S3`, `GCS`, `AZURE`). Never expose bucket name, credentials, or internal config to API response.
6. **Audit log**: write structured entry `{action: "file.get", fileId, actorId, ip, userAgent}`.

### Error Responses

| Status | Code | When |
|---|---|---|
| `400` | `VALIDATION_FAILED` | `urlTtlMinutes` out of `[5,60]` range |
| `401` | `UNAUTHORIZED` | Missing/invalid JWT |
| `403` | `FORBIDDEN` | Role is buyer/customer |
| `404` | `FILE_NOT_FOUND` | fileId not found OR cross-tenant attempt |
| `503` | `STORAGE_UNAVAILABLE` | PresignDownload fails (only if `includeDownloadUrl=true`; see degraded mode) |

---

## API 3 — `GET /api/files/{fileId}/download-url`

### Purpose

Generates a fresh short-lived presigned download URL for a specific file. Used when the client needs a standalone URL for downloading, rendering in `<img>`, sharing, etc.

### Path Parameter

| Param | Type | Required | Notes |
|---|---|---|---|
| `fileId` | UUIDv7 string | Yes | Must exist and be `ACTIVE` |

### Query Parameters

| Param | Type | Required | Default | Notes |
|---|---|---|---|---|
| `ttlMinutes` | int [5,60] | No | `15` | Lifetime of the generated URL |
| `variantCode` | string | No | *(none)* | If specified, generates URL for a `file_variant` instead of the original. Must be a valid variant code for this file. |
| `disposition` | enum | No | `inline` | `inline` or `attachment`; maps to `Content-Disposition` hint in presigned URL where supported. |

### Request Headers

| Header | Required | Notes |
|---|---|---|
| `Authorization` | **Yes** | Bearer JWT |
| `X-Correlation-ID` | **Yes** | Required; propagated to logs, presign call, and audit |

### Note on `disposition` Parameter

The `Content-Disposition` HTTP response header controls how a browser handles a file response:
- `inline` → browser renders/displays it in place (e.g. `<img>` tag shows the image, PDF opens in viewer)
- `attachment` → browser triggers a "Save As" / download dialog with the original filename

Providers like S3/MinIO support embedding `response-content-disposition=attachment; filename=hero.jpg` as a **signed query parameter** in the presigned URL itself, so the disposition is enforced by the provider's response to the browser — not by our API server. GCS and Azure also support this mechanism. The File Service encodes the chosen disposition into `BlobPresignDownloadInput` so the adapter builds the correct signed URL.

### Success Response — `200 OK`

```json
{
  "success": true,
  "message": "Download URL generated",
  "data": {
    "fileId": "018f2c1a-7a3e-7b2c-b4e2-c2a9d3e80001",
    "variantCode": null,
    "downloadUrl": "https://s3.amazonaws.com/bucket/key?X-Amz-Expires=900&...",
    "expiresAt": "2026-05-14T10:16:00Z",
    "ttlMinutes": 15,
    "mimeType": "image/jpeg",
    "sizeBytes": 524288
  }
}
```

> When `variantCode` is provided, `mimeType` and `sizeBytes` reflect the variant's values.

### Business Logic

1. **Auth + tenant check**: same as API 1.
2. **Status guard**: file MUST be `ACTIVE`. If `UPLOADING` → `409 FILE_NOT_ACTIVE`. If `FAILED` → `409 FILE_NOT_ACTIVE`.
3. **Variant resolution**: if `variantCode` is present → look up `file_variant WHERE file_object_id=$foid AND variant_code=$code`. Not found → `404 VARIANT_NOT_FOUND`. Variant status must be `READY`; if `PENDING` or `FAILED` → `409 VARIANT_NOT_READY`.
4. **Visibility check (v1)**: for `PUBLIC` files in v1 return `501 NOT_IMPLEMENTED` with message "Public URL delivery via CDN is not yet supported; use PRIVATE visibility for presigned access." This is intentional — no silent URL generation for "public" objects until CDN is wired.
5. **PresignDownload**: call `BlobAdapter.PresignDownload` with `{Bucket, Key, TTL}`. Map blob errors to `502/503`.
6. **Rate limiting hook (future)**: Add a TODO comment — per-caller per-file URL generation rate limit should be added before GA to prevent URL farming.
7. **Audit log**: `{action: "file.download-url", fileId, variantCode, ttlMinutes, actorId}`.

### Error Responses

| Status | Code | When |
|---|---|---|
| `400` | `VALIDATION_FAILED` | `ttlMinutes` out of range; unknown `disposition` value |
| `401` | `UNAUTHORIZED` | Missing/invalid JWT |
| `403` | `FORBIDDEN` | Wrong role |
| `404` | `FILE_NOT_FOUND` | fileId not found or cross-tenant |
| `404` | `VARIANT_NOT_FOUND` | `variantCode` doesn't exist for this file |
| `409` | `FILE_NOT_ACTIVE` | File is `UPLOADING` or `FAILED` |
| `409` | `VARIANT_NOT_READY` | Variant status is `PENDING` or `FAILED` |
| `501` | `NOT_IMPLEMENTED` | `visibility=PUBLIC` in v1 |
| `502` | `STORAGE_PERMISSION_DENIED` | Provider rejects presign request |
| `503` | `STORAGE_UNAVAILABLE` | Provider unreachable |

---

## API 4 — `DELETE /api/files/{fileId}`

### Purpose

Permanently deletes a file — removes the `file_object` row from the database **and** deletes the blob object (+ all variant objects) from the storage provider. This is a **synchronous hard delete**: the API does not return until the blob is deleted. There is no soft-delete, no `DELETED` status, and no async queue.

> **Why always hard-delete?** There is no compelling reason to keep orphan blobs around. Storage costs money. The blob adapter already exposes `DeleteObject`; calling it synchronously is simple, cheap, and leaves no cleanup debt.

### Path Parameter

| Param | Type | Required | Notes |
|---|---|---|---|
| `fileId` | UUIDv7 string | Yes | |

### Request Headers

| Header | Required | Notes |
|---|---|---|
| `Authorization` | **Yes** | Bearer JWT |
| `X-Correlation-ID` | **Yes** | Required; propagated to DB transaction, scheduler cancel, and blob delete calls |

### Success Response — `200 OK`

```json
{
  "success": true,
  "message": "File deleted",
  "data": {
    "fileId": "018f2c1a-7a3e-7b2c-b4e2-c2a9d3e80001",
    "deletedAt": "2026-05-14T11:00:00Z"
  }
}
```

### Business Logic

1. **Auth + tenant check**: same as API 2. Cross-tenant → `404`.
2. **Status guards**:
   - `UPLOADING` → **Allow**; also cancel the scheduled expiry job in Redis before proceeding.
   - `ACTIVE` → **Allow**.
   - `FAILED` → **Allow** (cleanup path; blob may or may not exist).
3. **Scheduler cleanup** (UPLOADING only): read cached Redis job ID key → call `scheduler.Cancel(jobId)`. Cancellation failure is logged as warning, NOT a blocker — continue with delete.
4. **Blob delete — original object**: call `BlobAdapter.DeleteObject(bucket, objectKey)`. Provider returns `nil` when object does not exist (idempotent per adapter contract). On non-idempotent provider errors → `503 STORAGE_UNAVAILABLE`; abort without touching DB.
5. **Blob delete — variants**: for each `file_variant` row associated with this file, call `BlobAdapter.DeleteObject(bucket, variantObjectKey)`. Best-effort: individual variant delete failures are logged as warnings but do NOT abort the overall delete (the DB row will still be hard-deleted; the orphan blob is a known trade-off logged for manual cleanup).
6. **Hard DB delete**: `DELETE FROM file_object WHERE id = $1`. `file_variant` rows are cascade-deleted by the FK `ON DELETE CASCADE`. `file_job` rows are also cascade-deleted.
7. **Audit log**: `{action: "file.delete", fileId, objectKey, variantCount, actorId, ip}`.

> **Idempotency**: if the `file_object` row does not exist (already deleted), step 1's tenant check returns `404 FILE_NOT_FOUND`. There is no idempotent `200` on repeat — the resource is gone.

### Error Responses

| Status | Code | When |
|---|---|---|
| `400` | `VALIDATION_FAILED` | Missing `X-Correlation-ID` |
| `401` | `UNAUTHORIZED` | Missing/invalid JWT |
| `403` | `FORBIDDEN` | Wrong role |
| `404` | `FILE_NOT_FOUND` | Not found or cross-tenant |
| `409` | `FILE_DELETE_CONFLICT` | File referenced by active product (future FK guard; stub `409` for now) |
| `503` | `STORAGE_UNAVAILABLE` | Original blob delete failed; DB row NOT deleted |

---

## Data Model Changes

**No new columns or enum values are required for this feature.**

- `file_object.status` remains `UPLOADING | ACTIVE | FAILED` — no `DELETED` status since the row is hard-deleted.
- No `deleted_at` column needed.
- No new migration for status changes.

### `file_object` lifecycle — updated

### `file_object.status` extended state machine
```
(nil) --> UPLOADING
              |
   complete OK|    scheduler / mismatch
              v               v
           ACTIVE  --DELETE--> (row gone, blob gone)
           FAILED  --DELETE--> (row gone, blob best-effort)
         UPLOADING --DELETE--> (scheduler cancelled, row gone)
```

### Cascade on hard-delete

Existing FK constraints handle all DB cleanup automatically:

| Table | FK | Behaviour |
|---|---|---|
| `file_variant` | `file_object_id -> file_object.id ON DELETE CASCADE` | All variant rows deleted automatically |
| `file_job` | `file_object_id -> file_object.id ON DELETE CASCADE` | All job rows deleted automatically |

---

---

## Cross-Cutting Requirements

| # | Requirement |
|---|---|
| CC-001 | All four endpoints MUST propagate `X-Correlation-ID` through context to DB, blob adapter, and log calls. Missing `X-Correlation-ID` header MUST return `400 VALIDATION_FAILED` before any processing. |
| CC-002 | No raw credentials, bucket names, or provider endpoints in any response body. `storageProvider` label only. |
| CC-003 | All responses MUST be `application/json; charset=utf-8`. |
| CC-004 | Context cancellation MUST abort in-flight provider calls and DB queries. |
| CC-005 | Structured audit log entry for every mutating call (DELETE) and presign call (GET download-url). |
| CC-006 | New error codes (`FILE_NOT_ACTIVE`, `VARIANT_NOT_FOUND`, `VARIANT_NOT_READY`, `FILE_DELETE_CONFLICT`) to be added to `file/error/` and `file/utils/constant/`. |
| CC-007 | DELETE MUST delete the original blob before deleting the DB row. If blob delete fails with a non-idempotent error, the DB row MUST be left intact and `503` returned. |

---

## Testing Philosophy

All test scenarios in this document MUST be implemented as **integration tests** using the project's Testcontainers infrastructure (Postgres, MinIO, RabbitMQ — same as `003-upload-apis`). This is a hard constraint, not a preference.

### Rules

| Rule | Detail |
|---|---|
| **Integration-first** | Every scenario that involves DB, HTTP handler, blob storage, or Redis MUST be an integration test. |
| **No mocks across process boundaries** | Do NOT mock `BlobAdapter`, `GORM`, or `RedisClient`. Use real MinIO, real Postgres, real Redis via Testcontainers. |
| **Unit tests: last resort** | A unit test is acceptable ONLY when the logic is purely computational and has zero I/O (e.g. `ToFilter()` parsing, enum validation helper, `ParseCommaSeparatedPtr` behavior). |
| **Testcontainers setup** | Reuse the shared `test/integration/setup` package established in `003-upload-apis`. No new container setup from scratch. |
| **API-first** | Tests call the real HTTP handler via `httptest.NewRecorder` + Gin router. Do NOT call service or repository methods directly in tests. |
| **DB seeding** | Seed test data through the upload API (`POST /api/files/init-upload` + `POST /api/files/complete-upload`) rather than inserting rows directly, to stay aligned with real production flows. Direct DB inserts are acceptable only for states unreachable through the API (e.g. corrupted rows). |

### Acceptable unit test cases (exhaustive list for this feature)

- `GetFilesParam.ToFilter()` — verify comma-separated parsing, default `Statuses=["ACTIVE"]`, nil handling
- Enum validation helper: unknown `purpose`/`status` string returns error
- `DeleteFileResponse` JSON serialisation

All other scenarios in the T00–T67 range MUST be integration tests.

---


## Test Scenarios

### API 1 — `GET /api/files` (Batch List / Filter)

#### Happy Path

| ID | Scenario | Expected | Test Type |
|---|---|---|---|
| T00 | Seller lists own ACTIVE files, no filters, default pagination | `200`, items scoped to seller, pagination metadata correct | Integration |
| T00a | `purposes=PRODUCT_IMAGE` (single value) | Only PRODUCT_IMAGE files returned | Integration |
| T00b | `purposes=PRODUCT_IMAGE,SELLER_LOGO` (multi-value) | Both purpose types returned | Integration |
| T00c | `statuses=FAILED` (single value) | Only FAILED files returned | Integration |
| T00d | `statuses=ACTIVE,FAILED` (multi-value) | Both status types returned | Integration |
| T00e | `mimeTypes=image/jpeg,image/webp` | Files matching either MIME type returned | Integration |
| T00f | `fileIds=id1,id2,id3` (all own) | Exactly those 3 returned | Integration |
| T00g | `fileIds` where one ID belongs to another seller | Cross-tenant ID silently omitted; own IDs returned | Integration |
| T00h | `includeVariants=true` | `variants` array populated per file | Integration |
| T00i | Seller passes 20 `fileIds` (mix of ACTIVE+FAILED) with `statuses=ACTIVE,FAILED` | All 20 returned | Integration |
| T00j | `pageSize=5`, 12 total files | First page has 5, `totalPages=3` | Integration |
| T00k | `sortBy=sizeBytes&sortOrder=asc` | Correct ordering verified | Integration |
| T00l | `purposes=PRODUCT_IMAGE` + `mimeTypes=image/jpeg` combined | AND across dimensions — only PRODUCT_IMAGE jpegs | Integration |

#### Auth / Tenant

| ID | Scenario | Expected | Test Type |
|---|---|---|---|
| T00m | Unauthenticated request | `401 UNAUTHORIZED` | Integration |
| T00n | Buyer role | `403 FORBIDDEN` | Integration |
| T00o | Missing `X-Correlation-ID` header | `400 VALIDATION_FAILED` | Integration |

#### Validation

| ID | Scenario | Expected | Test Type |
|---|---|---|---|
| T00p | `fileIds` count = 101 | `400 VALIDATION_FAILED` | Integration |
| T00q | `pageSize=0` or `pageSize=101` | `400 VALIDATION_FAILED` | Integration |
| T00r | Unknown `statuses=PENDING` | `400 VALIDATION_FAILED` | Integration |
| T00s | Unknown `purposes=SELFIE` | `400 VALIDATION_FAILED` | Integration |
| T00t | `GetFilesParam.ToFilter()` nil/default/multi-value parsing | Correct `GetFilesFilter` produced | **Unit** |

---

### API 2 — `GET /api/files/{fileId}`

#### Happy Path

| ID | Scenario | Expected |
|---|---|---|
| T01 | Seller fetches own ACTIVE PRODUCT_IMAGE without `includeDownloadUrl` | `200` with metadata, `downloadUrl` absent, `variants` populated |
| T02 | Seller fetches with `includeDownloadUrl=true&urlTtlMinutes=10` | `200`, valid presigned URL present, `downloadUrlExpiresAt` ≈ now+10min |
| T03 | Admin fetches own PLATFORM-owned ACTIVE file | `200` with metadata |
| T04 | File is `UPLOADING` — fetch returns metadata with `status=UPLOADING` and no download URL | `200`, no `downloadUrl` |
| T05 | File has no variants yet (just uploaded) | `200`, `variants: []` |
| T06 | File has multiple variants (thumb_200, webp_1600) all READY | `200`, all variants in array |

#### Auth / Tenant

| ID | Scenario | Expected |
|---|---|---|
| T07 | Unauthenticated request | `401 UNAUTHORIZED` |
| T08 | Buyer role | `403 FORBIDDEN` |
| T09 | Seller B fetches Seller A's fileId | `404 FILE_NOT_FOUND` |
| T10 | Admin fetches a Seller-owned fileId | `404 FILE_NOT_FOUND` |

#### Validation

| ID | Scenario | Expected |
|---|---|---|
| T11 | `urlTtlMinutes=4` (below minimum) | `400 VALIDATION_FAILED` |
| T12 | `urlTtlMinutes=61` (above maximum) | `400 VALIDATION_FAILED` |
| T13 | `fileId` is not a valid UUID format | `404 FILE_NOT_FOUND` (no row match) |

#### Provider Degraded

| ID | Scenario | Expected |
|---|---|---|
| T14 | `includeDownloadUrl=true` but MinIO unreachable | `200` metadata returned, no `downloadUrl` field, warning logged |

---

### API 3 — `GET /api/files/{fileId}/download-url`

#### Happy Path

| ID | Scenario | Expected |
|---|---|---|
| T20 | Seller requests URL for ACTIVE PRIVATE file, default TTL | `200`, valid presigned URL, `expiresAt` ≈ now+15min |
| T21 | `ttlMinutes=5` | URL valid for 5 min |
| T22 | `ttlMinutes=60` | URL valid for 60 min |
| T23 | `variantCode=thumb_200`, variant is READY | `200`, URL for variant key, variant `mimeType`/`sizeBytes` in response |
| T24 | `disposition=attachment` | `200`, presign includes attachment hint (where provider supports it) |

#### Auth / Tenant

| ID | Scenario | Expected |
|---|---|---|
| T25 | Unauthenticated | `401` |
| T26 | Buyer role | `403` |
| T27 | Cross-tenant fileId | `404 FILE_NOT_FOUND` |

#### Status Guards

| ID | Scenario | Expected |
|---|---|---|
| T28 | File is `UPLOADING` | `409 FILE_NOT_ACTIVE` |
| T29 | File is `FAILED` | `409 FILE_NOT_ACTIVE` |
| T30 | File is `DELETED` | `404 FILE_NOT_FOUND` (treat deleted as not found) |

#### Variant Edge Cases

| ID | Scenario | Expected |
|---|---|---|
| T31 | `variantCode=thumb_200` but no such variant for this file | `404 VARIANT_NOT_FOUND` |
| T32 | Variant exists but status is `PENDING` | `409 VARIANT_NOT_READY` |
| T33 | Variant exists but status is `FAILED` | `409 VARIANT_NOT_READY` |

#### Visibility

| ID | Scenario | Expected |
|---|---|---|
| T34 | File `visibility=PUBLIC` | `501 NOT_IMPLEMENTED` (v1 limitation) |
| T35 | File `visibility=INTERNAL`, seller requests | `200`, presigned URL returned (treated same as PRIVATE in v1) |

#### Validation

| ID | Scenario | Expected |
|---|---|---|
| T36 | `ttlMinutes=0` | `400 VALIDATION_FAILED` |
| T37 | `ttlMinutes=61` | `400 VALIDATION_FAILED` |
| T38 | Unknown `disposition=stream` | `400 VALIDATION_FAILED` |

#### Provider Errors

| ID | Scenario | Expected |
|---|---|---|
| T39 | Provider credentials revoked at presign time | `502 STORAGE_PERMISSION_DENIED` |
| T40 | Provider unreachable (network timeout) | `503 STORAGE_UNAVAILABLE` |

---

### API 4 — `DELETE /api/files/{fileId}`

#### Happy Path

| ID | Scenario | Expected |
|---|---|---|
| T50 | Seller deletes own ACTIVE file (no variants) | `200`, blob gone from MinIO, `file_object` row gone from DB |
| T51 | Seller deletes ACTIVE file WITH variants (thumb_200, webp_1600) | `200`, original blob gone, all variant blobs gone, all DB rows cascade-deleted |
| T52 | Delete UPLOADING file | `200`, scheduler expiry job cancelled, DB row gone, no blob to delete (object not uploaded yet — DeleteObject returns nil) |
| T53 | Delete FAILED file | `200`, DB row gone; DeleteObject is best-effort (blob may not exist — that's fine) |
| T54 | Seller deletes ACTIVE DOCUMENT (no variants) | `200`, blob gone, DB row gone |

#### Auth / Tenant

| ID | Scenario | Expected |
|---|---|---|
| T55 | Unauthenticated | `401` |
| T56 | Buyer | `403` |
| T57 | Seller B tries to delete Seller A's file | `404 FILE_NOT_FOUND`, A's record untouched |
| T58 | Admin tries to delete Seller-owned file | `404 FILE_NOT_FOUND` |
| T59 | Missing `X-Correlation-ID` | `400 VALIDATION_FAILED` |

#### Storage Failures

| ID | Scenario | Expected |
|---|---|---|
| T60 | Original blob delete fails (provider error) | `503 STORAGE_UNAVAILABLE`; DB row NOT deleted (verified by subsequent GET returning 200) |
| T61 | Variant blob delete fails for one of three variants | `200` still returned (best-effort); original + DB row deleted; failed variant key logged as warning |
| T62 | Provider returns "not found" for blob delete (already gone) | `200`; DB row deleted (DeleteObject is idempotent per adapter contract) |

#### Scheduler Cancellation (UPLOADING path)

| ID | Scenario | Expected |
|---|---|---|
| T63 | Delete UPLOADING file, Redis up | `200`, Redis scheduler job key removed, DB row gone |
| T64 | Delete UPLOADING file, Redis down | `200` still succeeds (warning logged), DB row gone |
| T65 | Scheduler expiry fires after DELETE completed | Expiry handler looks up row → not found → exits as no-op |

#### DB Cascade

| ID | Scenario | Expected |
|---|---|---|
| T66 | Delete ACTIVE file with 2 variants and 1 file_job | DB: `file_object`, both `file_variant`, and `file_job` rows all gone (verified by direct DB query) |
| T67 | Concurrent DELETE calls for same fileId | First succeeds, second gets `404 FILE_NOT_FOUND` (row already gone) |

---

## Open Questions / Decisions Required

| # | Question | Default Assumption |
|---|---|---|
| OQ-1 | Should `GET /api/files/{fileId}` for a non-existent (hard-deleted) file return `404` or `410 Gone`? | `404` — row is gone, same as never-existed. |
| OQ-2 | Should `DELETE` be allowed for files referenced by a product listing? | Stub `409 FILE_DELETE_CONFLICT` now; real guard in the product integration feature. |
| OQ-3 | Should `GET /api/files/{fileId}` include `uploadUrl` when `status=UPLOADING`? | No. Upload URL is a one-time secret; re-issue via `init-upload` idempotency key. |
| OQ-4 | Max number of variants per file in GET response? | No cap in v1. |
| OQ-5 | `disposition` param — query param or header? | Query param (header causes CDN caching issues). |
| OQ-8 | Should batch `GET /api/files` (seller public route) support `fileIds` param? | Yes — seller can filter by owned fileIds AND by purpose/status/mime. |
| OQ-9 | Variant blob delete best-effort vs. all-or-nothing? | **Best-effort** — variant delete failures log warnings but don't block the delete response. Orphan variant blobs are acceptable trade-off vs. blocking the user. |

---

## Implementation Notes

### New Files Expected

| File | Purpose |
|---|---|
| `file/handler/file_handler.go` | Implement `GetAllFiles`, `GetFile`, `GetDownloadURL`, `DeleteFile` (stubs already exist for latter 3) |
| `file/service/file_read_service.go` | `GetAllFiles`, `GetFile`, `GetDownloadURL` service logic |
| `file/service/file_delete_service.go` | `DeleteFile` — blob delete + DB hard-delete logic |
| `file/model/file_model.go` | `GetAllFilesRequest`, `GetAllFilesResponse`, `GetFileResponse`, `DownloadURLResponse`, `DeleteFileResponse` DTOs |
| `file/error/file_errors.go` | `ErrFileNotActive`, `ErrVariantNotFound`, `ErrVariantNotReady`, `ErrFileDeleteConflict` |
| `test/integration/file/get_all_files_test.go` | Integration tests for batch list API |
| `test/integration/file/get_file_test.go` | Integration tests for single get API |
| `test/integration/file/download_url_test.go` | Integration tests for download-url API |
| `test/integration/file/delete_file_test.go` | Integration tests for delete API |

> **No new migration needed** for the delete feature itself. The existing `ON DELETE CASCADE` FKs on `file_variant` and `file_job` handle all DB cleanup.

### Reused Patterns

- Auth principal extraction: `utils.ExtractPrincipal(c)` (same as upload handler)
- Tenant scoping query: `WHERE file_id=$1 AND owner_type=$2 AND owner_id=$3`
- Blob adapter resolution: existing `BlobAdapterFactory` via `StorageConfigID` stored on row
- Structured error envelope: `common.ErrorWithCode` / `HandleError`
- Scheduler cancellation: `common/scheduler.Cancel(jobId)` + cached Redis key pattern from `upload_expiry_scheduler.go`
- Async publish: `upload_variant_publisher.go` pattern (new command, same exchange)
- Audit log: `log.InfoWithContext` structured fields pattern

### Error Code Constants to Add

```
FILE_NOT_ACTIVE_CODE          = "FILE_NOT_ACTIVE"
FILE_NOT_ACTIVE_MSG           = "file is not in ACTIVE status"

VARIANT_NOT_FOUND_CODE        = "VARIANT_NOT_FOUND"
VARIANT_NOT_FOUND_MSG         = "file variant not found"

VARIANT_NOT_READY_CODE        = "VARIANT_NOT_READY"
VARIANT_NOT_READY_MSG         = "file variant is not ready"

FILE_DELETE_CONFLICT_CODE     = "FILE_DELETE_CONFLICT"
FILE_DELETE_CONFLICT_MSG      = "file cannot be deleted while referenced by active resources"
```

> **Removed**: `FILE_DELETED_CODE` / `FILE_DELETED_MSG` — no `DELETED` status.

---

## Success Criteria

| # | Criterion |
|---|---|
| SC-001 | All test scenarios T00–T67 pass as integration tests (Testcontainers: Postgres + MinIO + RabbitMQ). |
| SC-002 | p95 latency ≤ 100ms for `GET /api/files` batch with 20 `fileIds` (no variants) on local stack. |
| SC-003 | p95 latency ≤ 200ms for `GET /api/files/{fileId}` (no `includeDownloadUrl`) on local stack. |
| SC-004 | p95 latency ≤ 500ms for `GET /api/files/{fileId}/download-url` (includes presign round-trip). |
| SC-005 | p95 latency ≤ 300ms for `DELETE /api/files/{fileId}`. |
| SC-006 | Zero cross-tenant reads or deletes succeed across all isolation tests. |
| SC-007 | `GET /api/files` with `fileIds` returns only files owned by the authenticated seller; cross-tenant IDs are silently omitted. |
| SC-008 | Hard-delete is safe: if blob delete fails, DB row is NOT deleted (verified by subsequent GET returning 200). |
| SC-009 | Variant blob deletes are best-effort: variant delete failure does not block the main delete response. |
| SC-010 | Provider credentials never appear in any error response (scanned by shared helper). |
| SC-011 | Code coverage for new handler + service methods ≥ 85% line coverage driven by integration tests. |
| SC-012 | Every request missing `X-Correlation-ID` returns `400 VALIDATION_FAILED` across all four endpoints. |
