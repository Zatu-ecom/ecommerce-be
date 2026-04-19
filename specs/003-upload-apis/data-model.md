# Phase 1 Data Model: File Upload APIs

This document captures the authoritative data layout used by the init/complete upload flow. It reconciles the feature spec,
the `file/FILE_MODULE_DESIGN.md` design doc, and the current entities in `file/entity/file.go`.

> Clarification outcome: the `metadata JSONB` column on `file_object` and `file_variant` is **removed**. Because the
> owning migration (`migrations/018_create_file_storage_tables.sql`) has not been merged to `develop` yet, the column
> is dropped by editing that migration in place rather than adding a follow-up. See plan's Complexity Tracking.

---

## 1. Postgres tables

### 1.1 `file_object` (new in this feature's migration edit)

| Column | Type | Null | Notes |
|---|---|---|---|
| `id` | `BIGSERIAL` | No | PK |
| `file_id` | `VARCHAR(80)` | No | UUIDv7 string, `UNIQUE` |
| `seller_id` | `BIGINT` | Yes | `NULL` when `owner_type = 'PLATFORM'` (admin upload); FK → `user.id` |
| `uploader_user_id` | `BIGINT` | No | FK → `user.id` — the actor who called init-upload |
| `owner_type` | `VARCHAR(20)` | No | `SELLER` or `PLATFORM` (CHECK constraint) |
| `owner_id` | `BIGINT` | Yes | Mirrors `seller_id` for SELLER; `NULL` for PLATFORM |
| `purpose` | `VARCHAR(40)` | No | Enum (see §4.1) |
| `visibility` | `VARCHAR(20)` | No | `PRIVATE` / `PUBLIC` / `INTERNAL`; v1 treats `PUBLIC` as a logical flag |
| `storage_config_id` | `BIGINT` | No | FK → `storage_config.id` (resolved at init time; immutable thereafter) |
| `bucket_or_container` | `VARCHAR(255)` | No | Snapshot from `storage_config` at init |
| `object_key` | `VARCHAR(1000)` | No | Deterministic key (see research R9) |
| `original_filename` | `VARCHAR(255)` | No | Client-supplied raw filename |
| `sanitized_filename` | `VARCHAR(255)` | No | Used inside `object_key` |
| `mime_type` | `VARCHAR(150)` | No | Client-declared at init; re-validated at complete |
| `size_bytes` | `BIGINT` | No | Expected size from init; verified at complete |
| `etag` | `VARCHAR(200)` | Yes | Populated at complete from provider `HeadObject` |
| `status` | `VARCHAR(20)` | No | `UPLOADING` / `ACTIVE` / `FAILED` (CHECK) |
| `failure_reason` | `VARCHAR(150)` | Yes | Short machine code, no PII |
| `upload_expires_at` | `TIMESTAMPTZ` | No | Init time + `uploadExpiryMinutes` |
| `completed_at` | `TIMESTAMPTZ` | Yes | Set on successful complete |
| `created_at` | `TIMESTAMPTZ` | No | default `NOW()` |
| `updated_at` | `TIMESTAMPTZ` | No | default `NOW()` |

**Indexes**:

- `UNIQUE (file_id)`
- `INDEX (seller_id, created_at DESC)`
- `INDEX (owner_type, owner_id)`
- `INDEX (status, upload_expires_at)` — supports a potential sweeper fallback if Redis scheduler is lost
- `INDEX (purpose, status)`
- `UNIQUE (storage_config_id, object_key)` — defence against accidental key collisions

**CHECK constraints**:

- `owner_type IN ('SELLER','PLATFORM')`
- `status IN ('UPLOADING','ACTIVE','FAILED')`
- `(owner_type='SELLER' AND owner_id IS NOT NULL AND seller_id IS NOT NULL) OR (owner_type='PLATFORM' AND owner_id IS NULL AND seller_id IS NULL)`

### 1.2 `file_variant` (new in this migration edit, feature doesn't populate rows yet)

| Column | Type | Null | Notes |
|---|---|---|---|
| `id` | `BIGSERIAL` | No | PK |
| `file_object_id` | `BIGINT` | No | FK → `file_object.id`, `ON DELETE CASCADE` |
| `variant_code` | `VARCHAR(40)` | No | e.g. `thumb_200`, `thumb_600`, `webp_400`, `webp_1600` (see §4.1 per-purpose variant list) |
| `mime_type` | `VARCHAR(150)` | No | Variant mime |
| `bucket_or_container` | `VARCHAR(255)` | No | Usually same as parent |
| `object_key` | `VARCHAR(1000)` | No | Derived key |
| `size_bytes` | `BIGINT` | No | |
| `width` | `INT` | Yes | For images |
| `height` | `INT` | Yes | For images |
| `status` | `VARCHAR(20)` | No | `PENDING` / `READY` / `FAILED` |
| `created_at` / `updated_at` | `TIMESTAMPTZ` | No | |

No `metadata` column. No rows created by this feature — rows are created by the variant-worker feature. Table is declared so the worker has a place to write on day 1.

**Indexes**:

- `INDEX (file_object_id)`
- `UNIQUE (file_object_id, variant_code)`

### 1.3 `file_job` (new in this migration edit; written to by complete-upload)

| Column | Type | Null | Notes |
|---|---|---|---|
| `id` | `BIGSERIAL` | No | PK |
| `file_object_id` | `BIGINT` | No | FK → `file_object.id`, `ON DELETE CASCADE` |
| `command` | `VARCHAR(60)` | No | `file.image.process.requested` in this feature |
| `status` | `VARCHAR(20)` | No | `PUBLISHED` / `FAILED_TO_PUBLISH` / `DONE` (consumers transition to DONE) |
| `attempts` | `INT` | No | default 0 |
| `last_error` | `VARCHAR(300)` | Yes | |
| `correlation_id` | `VARCHAR(100)` | No | Inherited from the HTTP request |
| `created_at` / `updated_at` | `TIMESTAMPTZ` | No | |

**Indexes**: `INDEX (file_object_id)`, `INDEX (command, status)`.

This feature only inserts rows in `PUBLISHED` or `FAILED_TO_PUBLISH` and leaves transition-to-`DONE` to the consumer.

---

## 2. Status state machine (`file_object.status`)

```
          ┌─────────────────── complete-upload (object found + verified) ───────────────────┐
          ▼                                                                                 │
  ┌──────────────┐   init-upload   ┌──────────────┐   complete-upload (mismatch or          │
  │   (none)     │───────────────▶ │  UPLOADING   │   object missing → 409/422)             │
  └──────────────┘                 └──────┬───────┘                                         │
                                          │                                                 │
                                          │ upload_expires_at reached                       │
                                          │  OR complete-upload verification fail           │
                                          ▼                                                 │
                                   ┌──────────────┐                                         │
                                   │    FAILED    │                                         │
                                   └──────────────┘                                         │
                                                                                            │
                                                                                    ┌──────────────┐
                                                                                    │   ACTIVE     │
                                                                                    └──────┬───────┘
                                                                                           │
                                                              re-complete-upload (idempotent 200)
                                                                                           │
                                                                                           ▼
                                                                                        ACTIVE
```

**Guards** enforced in `FileUploadService`:

- `UPLOADING → ACTIVE`: only if provider HEAD returns size + content-type consistent with init-time declaration (within the mime family tolerance defined in `upload_policy.go`).
- `UPLOADING → FAILED`: by scheduler handler when `upload_expires_at < now()` and status is still `UPLOADING`; or by complete-upload when HEAD succeeds but size/mime mismatch.
- `ACTIVE → *`: terminal in this feature. Deletions handled by later feature.
- `FAILED → *`: terminal; client must call `init-upload` again.

---

## 3. Redis layout

| Purpose | Key | Value | TTL |
|---|---|---|---|
| Scheduler sorted set | `delayed_jobs` (managed by `common/scheduler`) | JSON `ScheduledJob{command=file.upload.expiry, payload={fileObjectId}}` | n/a (scheduler-managed) |
| Scheduler cancellation | `scheduler:cancelled:{jobId}` (managed by `common/scheduler`) | `1` | min(60s, job TTL) |
| Init-upload idempotency | `file:init:idem:{ownerType}:{ownerId}:{sha256(key)}` | Hash `{fileId, fingerprint}` | `uploadExpiryMinutes*60 + 300` |

Fingerprint = `sha256(canonicalJSON({purpose, visibility, mime, size, filename, uploadExpiryMinutes}))`.

---

## 4. Enumerations

### 4.1 `purpose`

Closed set (defined in `file/entity/file.go` as Go constants of type `FilePurpose`).
`init-upload` rejects any value outside this set with `400 VALIDATION_FAILED`.
`EXPORT_FILE` is additionally rejected by `init-upload` (system-generated only → `422 FILE_UPLOAD_POLICY_VIOLATION`).

| Constant | String value | Max size | Allowed mimes | Variants? | init-upload? |
|---|---|---|---|---|---|
| `FilePurposeProductImage` | `PRODUCT_IMAGE` | 10 MB | `image/jpeg`, `image/png`, `image/webp` | Yes — `[thumb_200, thumb_600, webp_1600]` | ✅ |
| `FilePurposeDocument` | `DOCUMENT` | 25 MB | `application/pdf`, `image/jpeg`, `image/png` | No | ✅ |
| `FilePurposeImportFile` | `IMPORT_FILE` | 50 MB | `text/csv`, `application/vnd.ms-excel`, `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` | No | ✅ |
| `FilePurposeExportFile` | `EXPORT_FILE` | — | — | No | ❌ system-generated only |
| `FilePurposeUserAvatar` | `USER_AVATAR` | 2 MB | `image/jpeg`, `image/png`, `image/webp` | Yes — `[thumb_200, webp_400]` | ✅ |
| `FilePurposeSellerLogo` | `SELLER_LOGO` | 3 MB | `image/jpeg`, `image/png`, `image/webp`, `image/svg+xml` | Raster only — `[thumb_200, webp_400]`; `image/svg+xml` → `HasVariants=false`, no `file_job` row inserted (SVG passthrough) | ✅ |
| `FilePurposeInvoicePDF` | `INVOICE_PDF` | 10 MB | `application/pdf` | No | ✅ |

Each row corresponds to a **policy entry** in `upload_policy.go` (T009).
The `Variants?` column drives `policy.HasVariants`; variant codes in that column are passed in `file.image.process.requested` payload.

### 4.2 `visibility`

- `PRIVATE` (default)
- `PUBLIC` (logical flag only in v1; does **not** affect ACLs or returned URLs)
- `INTERNAL`

### 4.3 `status`

- `UPLOADING`
- `ACTIVE`
- `FAILED`

---

## 5. Validation rules (summary)

- `filename`: 1..255 chars, not just whitespace; sanitised into `sanitized_filename`.
- `mimeType`: must match one of purpose-allowed mimes (case-insensitive).
- `sizeBytes`: `1 ≤ size ≤ purpose.maxSize`.
- `uploadExpiryMinutes`: optional; when omitted, default 15; range `[5, 60]`.
- `purpose`: must be in §4.1.
- `visibility`: must be in §4.2.
- `Idempotency-Key` header (optional): length 8..128 ASCII characters, regex `^[A-Za-z0-9._~-]+$` (aligned with spec FR-030 and research R8). Values outside this range MUST be rejected with `400 VALIDATION_FAILED` / code `IDEMPOTENCY_KEY_INVALID`.

---

## 6. Entity alignment (Go)

`file/entity/file.go` is edited to match:

```go
type FileObject struct {
    common.BaseEntity
    FileID            string           `gorm:"column:file_id;uniqueIndex;size:36"`
    SellerID          *uint64          `gorm:"column:seller_id;index"`
    UploaderUserID    uint64           `gorm:"column:uploader_user_id"`
    OwnerType         FileOwnerType    `gorm:"column:owner_type;size:20"`
    OwnerID           *uint64          `gorm:"column:owner_id"`
    Purpose           FilePurpose      `gorm:"column:purpose;size:40"`
    Visibility        FileVisibility   `gorm:"column:visibility;size:20"`
    StorageConfigID   uint64           `gorm:"column:storage_config_id"`
    BucketOrContainer string           `gorm:"column:bucket_or_container;size:255"`
    ObjectKey         string           `gorm:"column:object_key;size:1000"`
    OriginalFilename  string           `gorm:"column:original_filename;size:255"`
    SanitizedFilename string           `gorm:"column:sanitized_filename;size:255"`
    MimeType          string           `gorm:"column:mime_type;size:150"`
    SizeBytes         int64            `gorm:"column:size_bytes"`
    Etag              *string          `gorm:"column:etag;size:200"`
    Status            FileStatus       `gorm:"column:status;size:20"`
    FailureReason     *string          `gorm:"column:failure_reason;size:150"`
    UploadExpiresAt   time.Time        `gorm:"column:upload_expires_at"`
    CompletedAt       *time.Time       `gorm:"column:completed_at"`
}

type FileVariant struct {
    common.BaseEntity
    FileObjectID      uint64  `gorm:"column:file_object_id;index"`
    VariantCode       string  `gorm:"column:variant_code;size:40"`
    MimeType          string  `gorm:"column:mime_type;size:150"`
    BucketOrContainer string  `gorm:"column:bucket_or_container;size:255"`
    ObjectKey         string  `gorm:"column:object_key;size:1000"`
    SizeBytes         int64   `gorm:"column:size_bytes"`
    Width             *int    `gorm:"column:width"`
    Height            *int    `gorm:"column:height"`
    Status            string  `gorm:"column:status;size:20"`
}
```

No `Metadata` field on either struct. `FileJob` keeps its existing shape (gains `CorrelationID` if not already present; see entity file before edit).
