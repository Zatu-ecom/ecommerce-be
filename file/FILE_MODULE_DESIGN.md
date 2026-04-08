# 📁 File Module - Multi-Tenant Storage & Processing Design

> **Purpose**: Unified file upload, processing, import/export, and delivery module for admin + sellers  
> **Last Updated**: April 5, 2026  
> **Status**: Ready for implementation (Phase 1)

---

## 📋 Table of Contents

1. [Goals](#goals)
2. [Why Separate Module](#why-separate-module)
3. [Core Requirements](#core-requirements)
4. [High-Level Architecture](#high-level-architecture)
5. [Storage Provider Strategy](#storage-provider-strategy)
6. [Provider Configuration Contract](#provider-configuration-contract)
7. [Database Schema](#database-schema)
8. [API Design](#api-design)
9. [File Processing Pipeline](#file-processing-pipeline)
10. [Import/Export Flow](#importexport-flow)
11. [Security & Privacy](#security--privacy)
12. [Validation Rules](#validation-rules)
13. [Caching, CDN, and Performance](#caching-cdn-and-performance)
14. [Observability & Auditing](#observability--auditing)
15. [Failure Handling](#failure-handling)
16. [Implementation Plan](#implementation-plan)
17. [Open Decisions](#open-decisions)

---

## 🎯 Goals

- Support a central file system for:
  - Product/storefront media
  - Seller documents
  - Import files (CSV/XLSX)
  - Export files (reports, catalogs, orders)
- Allow dual storage model:
  - **Platform default storage** (admin configured)
  - **Seller-owned storage** (optional seller override for privacy/compliance)
- Keep design extensible to new storage providers without changing business logic.
- Support async processing and large file workflows.

---

## ✅ Why Separate Module

Create a dedicated `file` module now (inside same monolith), with boundaries that let it become a microservice later.

Benefits:

- Single place for upload policy, ACL, storage routing, and file metadata.
- Reusable across `product`, `user`, `report`, `order`, and future modules.
- Prevents duplicate upload logic in each module.
- Easier compliance controls (PII isolation, audit trails, retention policies).

---

## 📌 Core Requirements

1. Upload/download/delete with signed URLs.
2. Support private and public file visibility.
3. Seller-level storage override with fallback to platform storage.
4. Async processing:
   - image resize/thumbnail
   - metadata extraction
   - virus scan
5. Import/export jobs with progress tracking and downloadable output.
6. Strong tenant isolation (seller A must never access seller B files).
7. Provider-agnostic abstraction for blob storage.

---

## 🧱 High-Level Architecture

```text
[API Client]
    |
    v
[File Handler Layer]
    |
    v
[File Service]
  |      |        |
  |      |        +--> [Storage Resolver] ---> [Storage Adapter Factory] ---> [Provider Client]
  |      |
  |      +--> [File Registry Repository (Postgres)]
  |
  +--> [Job Publisher] ---> [Queue] ---> [File Processor Worker]
                                          |
                                          +--> [Variant Generator / Virus Scan / Import-Export Worker]
```

### Subcomponents

- `file/handler`: HTTP handlers
- `file/service`: Business logic (upload, access, policies)
- `file/storage`: Provider adapters + resolver
- `file/repository`: DB operations
- `file/worker`: Async processing/import-export
- `file/model`: request/response DTOs
- `file/entity`: DB entities

---

## ☁️ Storage Provider Strategy

## Popular providers to support

1. **AWS S3**
2. **Google Cloud Storage (GCS)**
3. **Azure Blob Storage**
4. **Cloudflare R2** (S3-compatible)
5. **MinIO** (self-hosted, S3-compatible)
6. **Backblaze B2** (S3-compatible)

### Adapter Groups

Implement three adapter families:

1. `s3_compatible_adapter`
   - AWS S3, Cloudflare R2, MinIO, Backblaze B2
2. `gcs_adapter`
3. `azure_blob_adapter`

### Unified Adapter Interface

```go
type BlobAdapter interface {
   PutObject(ctx context.Context, in PutObjectInput) (PutObjectOutput, error)
   DeleteObject(ctx context.Context, bucket, key string) error
   HeadObject(ctx context.Context, bucket, key string) (ObjectMeta, error)
   GetObjectStream(ctx context.Context, bucket, key string) (io.ReadCloser, ObjectMeta, error)
   PresignUpload(ctx context.Context, in PresignUploadInput) (PresignOutput, error)
   PresignDownload(ctx context.Context, in PresignDownloadInput) (PresignOutput, error)
   CopyObject(ctx context.Context, in CopyObjectInput) error
}
```

### Storage Resolution (important)

Request flow:

1. Determine seller/tenant from auth context.
2. Check active seller storage binding.
3. If enabled and valid: use seller provider credentials.
4. Else fallback to admin platform default storage.

This gives privacy choice without breaking default behavior.

---

## 🔐 Provider Configuration Contract

Store provider settings as encrypted JSONB (`credentials_encrypted` + `config_json`).

### Common Fields

- `provider`: `aws_s3 | gcs | azure_blob | r2 | minio | b2`
- `bucket_or_container`
- `region` (if applicable)
- `endpoint` (for custom S3-compatible)
- `base_path` (tenant prefix root, optional)
- `is_active`
- `is_default` (only for admin config)

### Provider-Specific Required Fields

1. **AWS S3 / S3-Compatible**
   - `access_key_id`
   - `secret_access_key`
   - `region`
   - `bucket`
   - optional: `endpoint`, `force_path_style`

2. **GCS**
   - `project_id`
   - `bucket`
   - `service_account_json` (encrypted)

3. **Azure Blob**
   - `account_name`
   - `account_key` or SAS policy
   - `container`

### Connection Test API

Before saving seller config:

- Validate credentials by lightweight `HeadBucket`/container check.
- Store only if validation passes.

---

## 🗄️ Database Schema

## 1) `storage_provider`

Master list of providers.

```sql
CREATE TABLE storage_provider (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE, -- aws_s3, gcs, azure_blob, r2, minio, b2
    name VARCHAR(100) NOT NULL,
    adapter_type VARCHAR(50) NOT NULL, -- s3_compatible, gcs, azure
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
```

## 2) `storage_config`

Admin and seller-specific storage credentials.

```sql
CREATE TABLE storage_config (
    id BIGSERIAL PRIMARY KEY,
    owner_type VARCHAR(20) NOT NULL, -- PLATFORM | SELLER
    owner_id BIGINT,                 -- NULL for PLATFORM default
    provider_id BIGINT NOT NULL REFERENCES storage_provider(id),
    display_name VARCHAR(150) NOT NULL,

    bucket_or_container VARCHAR(255) NOT NULL,
    region VARCHAR(100),
    endpoint VARCHAR(500),
    base_path VARCHAR(500),
    force_path_style BOOLEAN DEFAULT false,

    credentials_encrypted BYTEA NOT NULL,  -- envelope encrypted secrets
    config_json JSONB,                     -- non-secret settings

    is_default BOOLEAN NOT NULL DEFAULT false, -- true only for PLATFORM default
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_validated_at TIMESTAMPTZ,
    validation_status VARCHAR(30) NOT NULL DEFAULT 'PENDING',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_storage_config_owner ON storage_config(owner_type, owner_id);
CREATE INDEX idx_storage_config_provider_id ON storage_config(provider_id);
```

## 3) `seller_storage_binding`

Binding table to choose which seller config is active.

```sql
CREATE TABLE seller_storage_binding (
    id BIGSERIAL PRIMARY KEY,
    seller_id BIGINT NOT NULL REFERENCES "user"(id),
    storage_config_id BIGINT NOT NULL REFERENCES storage_config(id),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (seller_id, is_active) DEFERRABLE INITIALLY IMMEDIATE
);
```

## 4) `file_object`

Main file registry.

```sql
CREATE TABLE file_object (
    id BIGSERIAL PRIMARY KEY,
    file_id VARCHAR(80) NOT NULL UNIQUE,     -- public-safe identifier
    owner_type VARCHAR(20) NOT NULL,         -- SELLER | PLATFORM | USER
    owner_id BIGINT NOT NULL,
    seller_id BIGINT,                        -- nullable for platform/global files

    storage_config_id BIGINT NOT NULL REFERENCES storage_config(id),
    provider_code VARCHAR(50) NOT NULL,      -- denormalized for debug/audit

    bucket_or_container VARCHAR(255) NOT NULL,
    object_key VARCHAR(1000) NOT NULL,

    original_file_name VARCHAR(500) NOT NULL,
    extension VARCHAR(20),
    mime_type VARCHAR(120),
    size_bytes BIGINT NOT NULL,
    checksum_sha256 VARCHAR(64),
    e_tag VARCHAR(200),

    visibility VARCHAR(20) NOT NULL DEFAULT 'PRIVATE',  -- PRIVATE | PUBLIC | INTERNAL
    purpose VARCHAR(50) NOT NULL,                       -- PRODUCT_IMAGE | IMPORT_FILE | EXPORT_FILE | DOCUMENT
    status VARCHAR(30) NOT NULL DEFAULT 'UPLOADING',    -- UPLOADING | ACTIVE | FAILED | DELETED

    metadata JSONB,
    tags TEXT[],

    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_file_object_seller ON file_object(seller_id, created_at DESC);
CREATE INDEX idx_file_object_purpose ON file_object(purpose, status);
CREATE INDEX idx_file_object_owner ON file_object(owner_type, owner_id);
```

## 5) `file_variant`

Derived files (thumbnails/webp/optimized exports).

```sql
CREATE TABLE file_variant (
    id BIGSERIAL PRIMARY KEY,
    file_object_id BIGINT NOT NULL REFERENCES file_object(id),
    variant_type VARCHAR(50) NOT NULL, -- THUMBNAIL_SM, THUMBNAIL_MD, WEBP, PREVIEW
    bucket_or_container VARCHAR(255) NOT NULL,
    object_key VARCHAR(1000) NOT NULL,
    mime_type VARCHAR(120),
    size_bytes BIGINT,
    width INT,
    height INT,
    status VARCHAR(30) NOT NULL DEFAULT 'PROCESSING',
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(file_object_id, variant_type)
);
```

## 6) `file_job`

Async processing/import/export jobs.

```sql
CREATE TABLE file_job (
    id BIGSERIAL PRIMARY KEY,
    job_id VARCHAR(80) NOT NULL UNIQUE,
    seller_id BIGINT,
    initiated_by BIGINT,
    job_type VARCHAR(50) NOT NULL,  -- FILE_PROCESS | IMPORT | EXPORT | VIRUS_SCAN
    status VARCHAR(30) NOT NULL,    -- QUEUED | RUNNING | PARTIAL_SUCCESS | SUCCESS | FAILED
    progress_percent INT DEFAULT 0,
    input_file_id BIGINT REFERENCES file_object(id),
    output_file_id BIGINT REFERENCES file_object(id),
    error_code VARCHAR(100),
    error_message TEXT,
    payload JSONB,
    result_json JSONB,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_file_job_seller_status ON file_job(seller_id, status, created_at DESC);
```

---

## 🌐 API Design

Base path proposal: `/api/files`

## A) Upload/Download APIs

1. `POST /api/files/init-upload`
   - Returns presigned URL + `fileId` + required headers
   - Request:
     - `fileName`, `mimeType`, `sizeBytes`, `purpose`, `visibility`
   - Validates policy (max size, allowed mime, purpose)

2. `POST /api/files/complete-upload`
   - Verifies object exists, checksum/eTag, marks `ACTIVE`
   - Triggers async processing if needed

3. `GET /api/files/{fileId}`
   - Returns metadata + signed download URL (if private)

4. `GET /api/files/{fileId}/download-url`
   - Generates short-lived signed URL (e.g., 5-15 minutes)

5. `DELETE /api/files/{fileId}`
   - Soft delete + optional async hard-delete object

6. `POST /api/files/{fileId}/variants`
   - Request specific derived outputs

## B) Provider Config (Seller/Admin)

1. `GET /api/files/storage/providers`
   - Return supported provider list and required fields

2. `POST /api/files/storage-config/test`
   - Validate credentials + bucket/container access

3. `POST /api/files/storage-config`
   - Create/update storage config (seller or admin scope)

4. `POST /api/files/storage-config/{id}/activate`
   - Activate seller config as primary routing target

5. `GET /api/files/storage-config/active`
   - Return active resolved config summary (masked)

## C) Import/Export APIs

1. `POST /api/files/imports`
   - Input: source `fileId`, import type (products/orders/etc)
   - Creates job and returns `jobId`

2. `GET /api/files/imports/{jobId}`
   - Job status, errors, success rows

3. `POST /api/files/exports`
   - Input: export type + filters
   - Creates async job

4. `GET /api/files/exports/{jobId}`
   - Returns status + output `fileId` when complete

---

## ⚙️ File Processing Pipeline

## Upload Flow (presigned)

1. Client asks `init-upload`
2. Backend resolves storage config (seller override or admin default)
3. Backend returns presigned upload URL + object key
4. Client uploads directly to blob provider
5. Client calls `complete-upload`
6. Backend verifies object and creates `file_object`
7. Worker processes variants/scans and updates status

## Async Worker Tasks

- `virus_scan`
- `image_optimize`
- `thumbnail_generate`
- `document_preview_generate`
- `import_execute`
- `export_execute`

Queue topics:

- `file.process.requested`
- `file.variant.requested`
- `file.import.requested`
- `file.export.requested`

---

## 📦 Import/Export Flow

## Import

1. Upload import file (`purpose=IMPORT_FILE`)
2. Create import job with mapping options
3. Worker reads file stream from storage
4. Parse and validate row-by-row (batched)
5. Apply transactional upserts in chunks
6. Produce result summary file (optional CSV with row errors)
7. Mark job final state

## Export

1. Create export job with filters
2. Worker queries paginated data
3. Streams CSV/XLSX to temporary object
4. Registers output in `file_object` (`purpose=EXPORT_FILE`)
5. Return output `fileId` for download

---

## 🛡️ Security & Privacy

1. Encrypt credentials in DB (KMS/Vault-backed envelope encryption).
2. Never expose raw secrets in API responses.
3. Use short-lived presigned URLs.
4. Keep bucket private by default.
5. Enforce tenant ACL in every read/write.
6. Virus scan before marking file safe for storefront.
7. Optional content moderation for public media.
8. Audit every access:
   - who uploaded/downloaded/deleted
   - timestamp
   - IP/user-agent
9. Data retention and hard-delete policy for compliance.

---

## ✅ Validation Rules

- Max file size by purpose:
  - product image: 10 MB
  - document: 25 MB
  - import: 50 MB
  - export: system generated
- Allowed MIME per purpose (whitelist, no extension-only check)
- Reject dangerous extensions (if policy requires)
- Verify checksum from client and provider metadata
- Deduplicate optional: `(seller_id, checksum_sha256, size_bytes)`

---

## 🚀 Caching, CDN, and Performance

1. Public storefront assets can be CDN-served with long cache headers.
2. Private assets served through short-lived signed URLs.
3. Use deterministic object keys:
   - `seller/{sellerId}/{purpose}/{yyyy}/{mm}/{uuid}-{sanitizedName}`
4. Generate optimized variants (`webp`, thumbnails) for storefront performance.
5. Stream large import/export files; avoid loading full file in memory.

---

## 📈 Observability & Auditing

Metrics:

- upload init count / success rate
- upload complete latency
- processing job latency and failure rate
- provider error rate by provider code
- import/export throughput and row failure rates

Logs:

- include `trace_id`, `seller_id`, `file_id`, `job_id`, `provider`
- redact secrets

Audit table suggestion:

- `file_audit_log(file_id, seller_id, actor_id, action, metadata, created_at)`

---

## 🔁 Failure Handling

1. **Provider unavailable**
   - fail fast with retryable error
   - keep file status as `UPLOADING` or `FAILED`
2. **Worker crash**
   - idempotent jobs with retry count + dead-letter queue
3. **Partial variant failure**
   - keep original file `ACTIVE`
   - mark failed variants with reason
4. **Import partial data errors**
   - return error report file + row numbers
5. **Seller storage misconfiguration**
   - config activation blocked until successful test
   - fallback to default only if policy allows

---

## 🧭 Implementation Plan

## Phase 1 (MVP)

1. Build core file registry + default platform storage (S3-compatible only)
2. Implement init/complete upload + signed download
3. Implement product image upload usage
4. Implement export job with downloadable file
5. Add audit logs and basic metrics

## Phase 2

1. Add seller-owned storage configs + activation flow
2. Add GCS and Azure adapters
3. Add import pipeline with row-level error report
4. Add image variants and virus scanning

## Phase 3

1. Extract file worker (or full module) as separate service if needed
2. Add lifecycle/retention policies
3. Add advanced deduplication and cross-region replication options

---

## ❓ Open Decisions

1. Should seller fallback to platform storage be mandatory or optional policy?
2. Which queue infrastructure will be used first (`Redis`, `SQS`, or `RabbitMQ`)?
3. Do we require virus scanning for all file purposes or only storefront/public?
4. For imports, allow synchronous small-file mode or always async?
5. For hard-delete, immediate delete vs delayed retention window?

---

## 🔚 Final Recommendation

Yes, create a **separate `file` module now** in the monolith with clean interfaces.  
Start with **S3-compatible adapter + platform default storage**, then add seller BYOS and other providers in Phase 2.
