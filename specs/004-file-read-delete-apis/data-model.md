# Data Model: File Read & Delete APIs (004)

**Feature**: `004-file-read-delete-apis`
**Branch**: `004-file-read-delete-apis`
**Date**: 2026-05-16
**Status**: Final — no new migrations required

---

## Overview

This feature introduces **zero new database entities or migrations**. All read and delete operations work against the existing schema established by `003-upload-apis`. This document captures the entity reference, query contracts, and state-machine extension relevant to this feature.

---

## 1. Existing Entities (Reference Only)

### 1.1 `file_object` (primary entity)

Already defined in `file/entity/file.go`. Relevant fields for this feature:

| Field | Type | Notes |
|---|---|---|
| `id` | `uint64` (BIGSERIAL) | Internal PK; used in variant/job FKs |
| `file_id` | `string` (UUIDv7) | External handle; exposed in all API responses |
| `seller_id` | `*uint64` | NULL for PLATFORM-owned files |
| `owner_type` | `OwnerType` | `SELLER` or `PLATFORM` |
| `owner_id` | `*uint64` | Mirrors seller_id for SELLER; NULL for PLATFORM |
| `purpose` | `FilePurpose` | One of the FilePurpose constants |
| `visibility` | `FileVisibility` | `PRIVATE`, `PUBLIC`, `INTERNAL` |
| `storage_config_id` | `uint64` | FK → storage_config; needed for blob adapter resolution |
| `bucket_or_container` | `string` | Snapshotted at init; used for presign and delete |
| `object_key` | `string` | Full provider key; used for presign and delete |
| `original_filename` | `string` | Exposed in list/get responses |
| `mime_type` | `string` | Used in filter and response |
| `size_bytes` | `int64` | Used in filter (sortBy) and response |
| `etag` | `*string` | NULL until complete-upload |
| `status` | `FileStatus` | `UPLOADING`, `ACTIVE`, `FAILED` |
| `failure_reason` | `*string` | Set on FAILED transition |
| `upload_expires_at` | `time.Time` | Used by expiry scheduler |
| `completed_at` | `*time.Time` | Set on ACTIVE transition |

### 1.2 `file_variant` (derived entity)

Relevant fields for download-url and delete:

| Field | Type | Notes |
|---|---|---|
| `id` | `uint64` | Internal PK |
| `file_object_id` | `uint64` | FK → file_object.id (ON DELETE CASCADE) |
| `variant_code` | `string` | e.g. `thumb_200`, `webp_1600` |
| `mime_type` | `string` | Variant-specific MIME |
| `bucket_or_container` | `string` | May differ from parent |
| `object_key` | `string` | Used for variant presign and delete |
| `size_bytes` | `int64` | |
| `width` | `*int` | Nullable; set for image variants |
| `height` | `*int` | Nullable; set for image variants |
| `status` | `string` | `PENDING`, `READY`, `FAILED` |

> `file_variant.status` uses string (not typed const) in the current entity. The service layer must compare against `"READY"`, `"PENDING"`, `"FAILED"` string literals or define constants.

### 1.3 `file_job` (cascade target)

| Field | Type | Notes |
|---|---|---|
| `file_object_id` | `uint64` | FK → file_object.id (ON DELETE CASCADE) |

On hard-delete of `file_object`, all `file_job` rows are cascade-deleted automatically. No explicit query needed.

---

## 2. State Machine Extension

The `FileStatus` enum and state machine are unchanged. The delete path is now explicitly modelled:

```
(nil) ──► UPLOADING
              │
   complete OK│    scheduler / mismatch
              ▼               ▼
           ACTIVE  ──DELETE──► (row gone, blob gone)
           FAILED  ──DELETE──► (row gone, blob best-effort)
         UPLOADING ──DELETE──► (scheduler cancelled, row gone)
```

- **No `DELETED` status** is added. Hard-delete removes the row entirely.
- The FK cascade handles `file_variant` and `file_job` automatically.

---

## 3. New Query Contracts (Repository Layer)

These are new methods added to the **extended** `FileRepository` interface (extends `FileUploadRepository`):

### 3.1 `FindManyScoped` — Batch List with Filters

```go
// FindManyScoped returns a page of file_object rows scoped to (ownerType, ownerID).
// All filter slices are ORed within their dimension; dimensions are ANDed.
// If FileIDs is non-empty, an IN clause is added (max 100 enforced by service layer).
// Statuses defaults to ["ACTIVE"] when nil/empty (enforced by ToFilter()).
FindManyScoped(
    ctx context.Context,
    ownerType entity.OwnerType,
    ownerID *uint64,
    filter GetFilesFilter,
) ([]entity.FileObject, int64, error)
// Returns: (items, totalCount, error)
```

### 3.2 `FindVariantsByFileObjectIDs` — Batch Variant Fetch

```go
// FindVariantsByFileObjectIDs returns all file_variant rows for the given
// internal file_object IDs. Used for includeVariants=true batch list.
FindVariantsByFileObjectIDs(
    ctx context.Context,
    fileObjectIDs []uint64,
) ([]entity.FileVariant, error)
```

### 3.3 `FindVariantByCode` — Variant Lookup for Download URL

```go
// FindVariantByCode returns the file_variant row for the given
// (fileObjectID, variantCode) pair. Returns nil, nil when not found.
FindVariantByCode(
    ctx context.Context,
    fileObjectID uint64,
    variantCode string,
) (*entity.FileVariant, error)
```

### 3.4 `DeleteFileObject` — Hard Delete

```go
// DeleteFileObject hard-deletes the file_object row by primary key.
// The FK ON DELETE CASCADE propagates to file_variant and file_job rows.
// Called ONLY after blob deletion succeeds.
DeleteFileObject(ctx context.Context, id uint64) error
```

---

## 4. Filter Model

```go
// GetFilesBase holds scalar filter fields shared by Param and Filter.
type GetFilesBase struct {
    common.BaseListParams
    IncludeVariants bool   `form:"includeVariants" binding:"omitempty"`
    SortBy          string `form:"sortBy"          binding:"omitempty,oneof=createdAt sizeBytes originalFilename"`
    SortOrder       string `form:"sortOrder"       binding:"omitempty,oneof=asc desc"`
}

// GetFilesParam is bound directly from the HTTP query string.
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
    Statuses  []string // defaults to ["ACTIVE"] when empty
    MimeTypes []string
}
```

---

## 5. Cascade Constraints (Existing — No Change)

| Table | FK Column | References | ON DELETE |
|---|---|---|---|
| `file_variant` | `file_object_id` | `file_object.id` | CASCADE |
| `file_job` | `file_object_id` | `file_object.id` | CASCADE |

---

## 6. No New Migrations

This feature requires **zero migration files**. All DDL is in place from `003-upload-apis`. The state machine extension (deletes) operates at the application layer only.
