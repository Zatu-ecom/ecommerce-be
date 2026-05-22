# Feature Specification: File Upload APIs (Init + Complete)

**Feature Branch**: `003-upload-apis`
**Created**: 2026-04-18
**Status**: Draft
**Input**: User description: "Implement `POST /api/files/init-upload` and `POST /api/files/complete-upload` end-to-end with integration tests (minimal mocking). Include async variant generation where applicable. Align with `file/FILE_MODULE_DESIGN.md` and `file/RABBITMQ_FILE_MODULE_DESIGN.md` and existing code state."

---

## Overview

This feature delivers the **two-step presigned upload flow** for the file module:

1. **Init upload**: the seller's client asks the platform for a _short-lived, provider-signed URL_ and receives a server-issued `fileId`. The client uploads bytes **directly to the blob provider** (S3 / MinIO / GCS / Azure Blob).
2. **Complete upload**: the client tells the platform the upload finished. The platform **verifies the object with the provider**, finalises the `file_object` record as `ACTIVE`, and — for eligible purposes — **publishes an async job** to generate image variants (thumbnails, webp).

This is the first end-to-end slice on top of the existing blob adapter layer and storage config/activation work. Out of scope: download URL API, delete API, variants re-request API, import/export workers, virus scanning (tracked in later slices but referenced here for correct state modeling).

## Clarifications

### Session 2026-04-18

- Q: Who may call `init-upload` / `complete-upload`? → A: Seller and Admin only; buyers/customers cannot upload. `owner_type` is derived from the caller's role (`SELLER` for sellers, `PLATFORM` for admins); both roles reuse the same endpoints.
- Q: What does `visibility = PUBLIC` do at upload time in v1? → A: Logical flag only; bucket and object stay private, no public-read ACL on the presigned PUT, no public URL returned. Public delivery is deferred to a later download/CDN feature.
- Q: How are never-completed `UPLOADING` rows cleaned up? → A: **Build the cleanup in this feature** by reusing the existing Redis-sorted-set scheduler pattern from the inventory reservation module (`common/scheduler` + `inventory/service/reservation_scheduler_service.go`). `init-upload` schedules a per-`file_object` expiry job; `complete-upload` cancels it on success. The delay is **configurable per upload** (request parameter with server-enforced bounds) and the worker marks the row `FAILED` and best-effort deletes any stray object.
- Q: How is `init-upload` deduplicated against client retries? → A: **Optional `Idempotency-Key` HTTP header** (Stripe-style). A Redis mapping `file:init:idem:{ownerType}:{ownerId}:{sha256(Idempotency-Key)} → fileId` (TTL = `uploadExpiryMinutes + cacheBufferDuration`) ensures duplicate calls with the same key return the same `fileId`. The raw header value is SHA-256 hashed into the Redis key to cap key length and avoid exposing the client-supplied value in keyspace. If the header is absent, each call is treated as a new upload.
- Q: Shape of the `metadata` field on `init-upload` / `file_object`? → A: **Removed entirely.** No free-form/untyped blob is accepted or stored. The `Metadata db.JSONMap` column is removed from both `FileObject` and `FileVariant` in `file/entity/file.go`. Because this feature's migration has not been merged to `develop` yet, we **do not create a new migration**; the existing `file_object` / `file_variant` migration SQL is updated in place to drop the column. Any future caller that legitimately needs a typed field must add a dedicated column with its own type.

## Scope

**In scope**

- `POST /api/files/init-upload` handler, service, repository, validation, integration tests.
- `POST /api/files/complete-upload` handler, service, repository, variant publishing, integration tests.
- Storage resolution using existing seller/platform storage configs and blob adapter factory.
- Async variant generation triggered at complete-upload for `PRODUCT_IMAGE` (raster images only).
- End-to-end integration tests using Testcontainers (Postgres + MinIO + RabbitMQ) with minimal mocking.

**Out of scope (handled in later features)**

- Download URL endpoint, file metadata endpoint, delete endpoint, variant re-request endpoint.
- Import / export job execution.
- Virus scanning worker.
- CDN / public URL publishing.
- Multipart upload for > 100 MB files.

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Seller uploads a product image (happy path) (Priority: P1)

A seller uploading a product image from the admin panel initiates a single-part upload, uploads the bytes directly to blob storage using the signed URL, and then finalises the upload so the image is visible in the product editor and queued for thumbnail generation.

**Why this priority**: This is the core workflow blocking every downstream feature (product media, seller assets, future imports). Without it the `file` module has no usable write path.

**Independent Test**: Using a seller account bound to a MinIO-backed storage config, call `init-upload` with a valid JPEG envelope, `PUT` the bytes to the returned URL, call `complete-upload`, and assert:

- `fileId` exists with status `ACTIVE` in the DB,
- the object is present in the provider bucket at the deterministic object key,
- a RabbitMQ message for image processing has been published for this `fileId`.

**Acceptance Scenarios**:

1. **Given** an authenticated seller with an active storage config, **When** the seller calls `init-upload` with `{fileName: "hero.jpg", mimeType: "image/jpeg", sizeBytes: 524288, purpose: "PRODUCT_IMAGE", visibility: "PRIVATE"}`, **Then** the response contains a non-empty `fileId`, a non-empty presigned PUT URL, required upload headers, an `expiresAt` in the future, and a `file_object` row exists in the DB with status `UPLOADING` and `purpose = PRODUCT_IMAGE`.
2. **Given** a valid `fileId` returned by `init-upload` and an object successfully `PUT` to the provider at the expected key with matching size and content-type, **When** the seller calls `complete-upload` with that `fileId`, **Then** the response contains `{fileId, status: "ACTIVE", size, mimeType, eTag, checksum?}`, the `file_object` row transitions to `ACTIVE`, provider-reported `size_bytes` / `e_tag` are persisted, and a message is published on `ecom.commands` with routing key `file.image.process.requested` carrying the `fileId`/`fileObjectId`.
3. **Given** a `complete-upload` request for a `PRODUCT_IMAGE` whose mime type is `image/jpeg`, **When** the platform verifies the object, **Then** image variant publishing is invoked; **Given** the same for `purpose = DOCUMENT`, **Then** no image variant message is published.

---

### User Story 2 - Seller finalises a non-image document (Priority: P2)

A seller uploads a PDF invoice or CSV import file. The same two-step flow finalises the file as `ACTIVE` but does **not** queue image variants.

**Why this priority**: Required for the import/export pipeline and seller documents, and it validates that variant-publishing is correctly gated by purpose/mime.

**Independent Test**: Using a seller account, `init-upload` with `purpose=DOCUMENT` and `mimeType=application/pdf`, upload the bytes, call `complete-upload`, and assert the record is `ACTIVE` and **no** `file.image.process.requested` message is published.

**Acceptance Scenarios**:

1. **Given** an authenticated seller, **When** they initiate and complete an upload with `purpose = DOCUMENT, mimeType = application/pdf, sizeBytes <= 25 MB`, **Then** the final status is `ACTIVE` and no image variant message is published.
2. **Given** an authenticated seller, **When** they initiate and complete an upload with `purpose = IMPORT_FILE, mimeType = text/csv, sizeBytes <= 50 MB`, **Then** the final status is `ACTIVE` and no image variant message is published.

---

### User Story 3 - Quota / policy enforcement on init (Priority: P2)

The platform must reject uploads that violate per-purpose size or mime policy **before** issuing a presigned URL, so that storage is never contacted for requests that would later be rejected.

**Why this priority**: Prevents abuse of signed URLs, limits blob storage cost, and is a prerequisite for seller trust and audit.

**Independent Test**: Call `init-upload` with oversized or disallowed mime payloads and assert `422` with structured error codes; verify that no row is inserted and no presigned URL is returned. Assert `400` for structurally malformed inputs (empty filename, zero size).

**Acceptance Scenarios**:

1. **Given** an authenticated seller, **When** they call `init-upload` with `purpose=PRODUCT_IMAGE, sizeBytes = 12 * 1024 * 1024`, **Then** the response is `422` with error code `FILE_UPLOAD_POLICY_VIOLATION`, no `file_object` row is created, and no presigned URL is issued.
2. **Given** an authenticated seller, **When** they call `init-upload` with `purpose=PRODUCT_IMAGE, mimeType = application/x-msdownload`, **Then** the response is `422` with error code `FILE_UPLOAD_POLICY_VIOLATION`.
3. **Given** an authenticated seller, **When** they call `init-upload` with `sizeBytes <= 0` or `fileName = ""`, **Then** the response is `400` with a validation error and no row is created.

---

### User Story 4 - Tenant isolation on complete-upload (Priority: P2)

Seller B must not be able to finalise seller A's upload even if they somehow learn the `fileId`. Storage resolution and DB scoping must be tenant-aware end to end.

**Why this priority**: Multi-tenant correctness is a security baseline; a bug here leaks files across sellers.

**Independent Test**: Have seller A call `init-upload`, have seller B authenticate and call `complete-upload` with A's `fileId`; assert `404` (not `403`, to avoid enumeration) and that A's record is untouched.

**Acceptance Scenarios**:

1. **Given** seller A owns a `file_object` in `UPLOADING` state, **When** seller B calls `complete-upload` with that `fileId`, **Then** the response is `404 FILE_NOT_FOUND` and A's record remains `UPLOADING`.
2. **Given** seller A owns a file, **When** an unauthenticated client calls either endpoint, **Then** the response is `401 UNAUTHORIZED`.
3. **Given** a customer/buyer-authenticated principal, **When** they call either endpoint, **Then** the response is `403 FORBIDDEN` and no `file_object` row is created or modified.
4. **Given** an Admin-authenticated principal and a `file_object` with `owner_type = SELLER`, **When** the admin calls `complete-upload` with that `fileId`, **Then** the response is `404 FILE_NOT_FOUND` (admins do not inherit seller scope).

---

### User Story 5 - Complete called before the object exists (Priority: P3)

A client calls `complete-upload` while the `PUT` to the provider has not finished (or failed). The platform must not mark the record `ACTIVE`.

**Why this priority**: Prevents broken-image records from polluting the catalog; well-defined retry semantics let the client recover.

**Independent Test**: Call `init-upload` but skip the `PUT`, then call `complete-upload`; assert `409` with a specific error, and that status remains `UPLOADING`.

**Acceptance Scenarios**:

1. **Given** a `fileId` in `UPLOADING` state with no object in the bucket, **When** the seller calls `complete-upload`, **Then** the response is `409 FILE_NOT_UPLOADED_YET`, status remains `UPLOADING`, and no variant message is published.
2. **Given** a `fileId` whose uploaded object has `sizeBytes` different from the `sizeBytes` promised at init, **When** the seller calls `complete-upload`, **Then** the response is `409 FILE_SIZE_MISMATCH`, status is set to `FAILED`, and no variant message is published.
3. **Given** a `fileId` already in `ACTIVE`, **When** the seller calls `complete-upload` again, **Then** the response is idempotent `200` with the current `file_object` state and **no duplicate** variant message is published.

---

### User Story 5a - Init-upload is idempotent against client retries (Priority: P2)

A flaky mobile network causes the client to retry `init-upload` with the same body. The platform must return the **same** `fileId` (not create a duplicate row + scheduler job) when the client supplies an `Idempotency-Key` header.

**Why this priority**: Protects DB and Redis from amplification under normal retry behaviour; failing to handle this causes scheduler-key collisions and unnecessary `UPLOADING` rows.

**Independent Test**: Call `init-upload` twice with the same `Idempotency-Key` header and body; assert the same `fileId` is returned, exactly one `file_object` row exists, exactly one scheduled expiry job exists in Redis, and the presigned URL is valid (either replayed from cache or re-issued if the cached one expired).

**Acceptance Scenarios**:

1. **Given** an authenticated seller and a valid `Idempotency-Key`, **When** they call `init-upload` twice in quick succession with identical bodies, **Then** both responses carry the same `fileId`, exactly one `file_object` row is in `UPLOADING`, and exactly one scheduled expiry job exists in Redis.
2. **Given** an authenticated seller, **When** the second call uses the same `Idempotency-Key` after the originally-issued presigned URL has passed its `expiresAt` but the idempotency record has not yet expired, **Then** the response returns the same `fileId` with a **freshly-issued** presigned URL (and the cache is updated).
3. **Given** an authenticated seller whose previous upload under this key has reached `ACTIVE`, **When** they retry `init-upload` with the same `Idempotency-Key`, **Then** the response is `409 IDEMPOTENCY_KEY_CONFLICT` (FR-033).
4. **Given** an `Idempotency-Key` that is empty / too long / contains disallowed characters, **When** `init-upload` is called, **Then** the response is `400 VALIDATION_FAILED` with code `IDEMPOTENCY_KEY_INVALID`.
5. **Given** two different sellers using the same string as their `Idempotency-Key`, **When** each calls `init-upload`, **Then** each receives their own distinct `fileId` and neither record is linked to the other (FR-035).

---

### User Story 6a - Abandoned upload is auto-cleaned (Priority: P2)

A client calls `init-upload` but never completes (tab closed, network dropped, app crashed). The platform must not leave that row in `UPLOADING` forever, and must reuse the existing Redis-scheduler pattern from the inventory reservation module rather than introducing a new cron or polling.

**Why this priority**: Prevents silent bloat of `file_object` and stray bytes in the bucket; tests exercise the *integration* between the new upload flow and `common/scheduler`.

**Independent Test**: Call `init-upload` with `uploadExpiryMinutes = 1` (or override via test configuration to a few seconds), do not `PUT`, wait the bounded delay, and assert (a) the row is `FAILED` with reason `UPLOAD_EXPIRED`, (b) a subsequent `complete-upload` returns `409 UPLOAD_EXPIRED`, and (c) if a partial object existed the adapter's `HeadObject` now returns not-found.

**Acceptance Scenarios**:

1. **Given** a `file_object` in `UPLOADING` with a scheduled expiry in the past, **When** the scheduler worker dispatches `file.upload.expiry`, **Then** the row transitions to `FAILED` with reason `UPLOAD_EXPIRED`, the stored Redis job key is absent, and any stray object at the target key has been best-effort deleted.
2. **Given** a `file_object` in `UPLOADING` with a scheduled expiry in the future, **When** `complete-upload` succeeds, **Then** the cached Redis job id MUST be cancelled and the scheduled job MUST not fire afterwards; if it does fire (race), **Then** it observes `ACTIVE` and exits without side effects (FR-029).
3. **Given** the caller supplies `uploadExpiryMinutes = 0` or `61`, **When** `init-upload` is called, **Then** the response is `400 VALIDATION_FAILED` with code `UPLOAD_EXPIRY_OUT_OF_RANGE`.
4. **Given** a `file_object` already transitioned to `FAILED` with reason `UPLOAD_EXPIRED`, **When** the caller re-submits `complete-upload` for that `fileId`, **Then** the response is `409 UPLOAD_EXPIRED` and the row is not re-activated.

---

### User Story 6 - Storage outage during init (Priority: P3)

If the resolved storage provider is unreachable (network error, invalid credentials, bucket missing), `init-upload` must fail fast with a retryable error shape and must not leave orphan rows.

**Why this priority**: Ensures the DB and the provider cannot drift; protects against zombie `UPLOADING` rows during provider incidents.

**Independent Test**: Point the seller's storage config at a non-existent MinIO bucket (or a bad endpoint) and call `init-upload`; assert `503` (or provider-mapped error), no row in DB, and a structured error code.

**Acceptance Scenarios**:

1. **Given** the resolved storage config points at a non-existent bucket, **When** the seller calls `init-upload`, **Then** the response is `503 STORAGE_UNAVAILABLE` with no `file_object` row inserted.
2. **Given** the resolved storage config has invalid credentials, **When** the seller calls `init-upload`, **Then** the response is `502 STORAGE_PERMISSION_DENIED` and secrets are not echoed in the message.

---

### Edge Cases

- Client sends `mimeType` that disagrees with the uploaded object's content-type returned by `HeadObject`; `complete-upload` MUST fail with `422 FILE_UPLOAD_OBJECT_MISMATCH` and mark the row `FAILED` (covered by T066).
- Client supplies a `clientEtag` hint that does not match the ETag returned by `HeadObject`; `complete-upload` MUST fail with `422 FILE_UPLOAD_OBJECT_MISMATCH` and mark the row `FAILED` (covered by T065).
- Seller has no active storage binding and platform default is disabled: `init-upload` returns `412 FILE_UPLOAD_NO_STORAGE_CONFIG` (not `500`).
- RabbitMQ is unreachable at complete-upload time: `file_object` status transitions to `ACTIVE` (the bytes are safe), the response is `200`, and a background reconciler / retry is responsible for publishing the variant message. `complete-upload` MUST NOT fail end-users for a queue outage.
- Two parallel `complete-upload` calls for the same `fileId`: only one succeeds, the other observes `ACTIVE` and returns `200` idempotently; exactly one variant message is published.
- `fileName` contains path separators, NUL bytes, unicode RTL override, or is longer than 500 chars: sanitised by the server and the stored `original_file_name` matches the sanitised value; the object key never contains unsanitised user input.
- Presigned URL TTL expires before the client finishes the `PUT`: `complete-upload` will see `FILE_NOT_UPLOADED_YET` if the scheduler has not yet fired; once the scheduled expiry job runs, the row is `FAILED / UPLOAD_EXPIRED` and any subsequent `complete-upload` returns `409 UPLOAD_EXPIRED`. The client is expected to re-initiate either way.
- Redis unavailable at `init-upload`: scheduling the expiry job MUST fail the init request with `503 SCHEDULER_UNAVAILABLE` and roll back the `file_object` insert and the presigned URL issuance (no orphan rows without a cleanup path).
- Redis unavailable at `complete-upload` (cannot cancel the scheduled expiry): the request MUST still succeed (`200 ACTIVE`); FR-029 guarantees the later-firing expiry job is a no-op when the row is already `ACTIVE`.
- Scheduled expiry fires while `complete-upload` is mid-flight: both paths are serialised through the `file_object` row's status transition (`UPLOADING` → `ACTIVE` or `UPLOADING` → `FAILED`); whichever wins the conditional update is authoritative, the other is a no-op.

## Requirements _(mandatory)_

### Functional Requirements

**Init upload**

- **FR-001**: `POST /api/files/init-upload` MUST accept authenticated principals with role **Seller** or **Admin** (buyers/customers MUST be rejected with `403 FORBIDDEN`; unauthenticated requests MUST be rejected with `401`). `owner_type` on the resulting `file_object` MUST be `SELLER` when the caller is a seller and `PLATFORM` when the caller is an admin. For **Seller** callers, `owner_id` MUST be the seller's user id and `seller_id` MUST also be set. For **Admin** callers, `owner_id` MUST be **NULL** and `seller_id` MUST be **NULL** in the DB (the admin's user_id is **not** stored in `owner_id`; it is used only as the `{ownerId}` segment in the idempotency Redis key to prevent cross-admin collisions — see FR-031).
- **FR-002**: The init request MUST accept `fileName`, `mimeType`, `sizeBytes`, `purpose`, `visibility`, and optional `uploadExpiryMinutes`. **No `checksumSha256` or checksum field is accepted at init time** (checksum/etag verification is a complete-time operation; see FR-016). **No free-form `metadata` field is accepted.** Any request body key not in the closed set above MUST be rejected with `400 VALIDATION_FAILED` (strict binding; no unknown-field passthrough). `purpose` and `visibility` are **closed enumerations**, not free-form strings: `purpose ∈ {PRODUCT_IMAGE, IMPORT_FILE, EXPORT_FILE, DOCUMENT, USER_AVATAR, SELLER_LOGO, INVOICE_PDF}` (matches `entity.FilePurpose`), `visibility ∈ {PRIVATE, PUBLIC, INTERNAL}` (matches `entity.FileVisibility`). Any value outside these sets MUST return `400 VALIDATION_FAILED` before the request is processed. `uploadExpiryMinutes` is an integer in the closed range **[5, 60]**; when omitted it defaults to **15**; values outside the bounds MUST return `400 VALIDATION_FAILED`.
- **FR-003**: The platform MUST enforce per-purpose policy before contacting the provider: `PRODUCT_IMAGE` ≤ 10 MB and mime in {`image/jpeg`, `image/png`, `image/webp`}; `DOCUMENT` ≤ 25 MB and mime in {`application/pdf`, `image/jpeg`, `image/png`}; `IMPORT_FILE` ≤ 50 MB and mime in {`text/csv`, `application/vnd.ms-excel`, `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`}; `USER_AVATAR` ≤ 2 MB and mime in {`image/jpeg`, `image/png`, `image/webp`}; `SELLER_LOGO` ≤ 3 MB and mime in {`image/jpeg`, `image/png`, `image/webp`, `image/svg+xml`}; `INVOICE_PDF` ≤ 10 MB and mime in {`application/pdf`}; `EXPORT_FILE` is not accepted by init-upload (system-generated only) — MUST return `422 FILE_UPLOAD_POLICY_VIOLATION`.
- **FR-004**: The platform MUST resolve the target storage config using (a) active seller binding if present, else (b) platform default. If neither resolves to an active, validated config, return `412 FILE_UPLOAD_NO_STORAGE_CONFIG`.
- **FR-005**: The platform MUST generate a deterministic object key of the form `seller/{sellerId}/{purpose}/{yyyy}/{mm}/{uuid}-{sanitizedFileName}` (platform-owned files use `platform/...`). Sanitisation lowercases, strips path separators, removes non-ASCII except letters/digits/`.-_`, and truncates to 120 characters.
- **FR-006**: The platform MUST generate a `fileId` as a **UUIDv7 string** (e.g. `018f2c1a-7a3e-7b2c-b4e2-c2a9d3e80001`) via `github.com/google/uuid` (`uuid.NewV7()`), exposed to the client as the canonical upload reference, distinct from the DB primary key. UUIDv7 is time-ordered, preserves B-tree insertion locality on the `UNIQUE(file_id)` index, and requires no additional dependency (already an indirect dep; promoted to direct per research R1).
- **FR-007**: The platform MUST insert a `file_object` row with `status = UPLOADING` before returning; this row is the source of truth for complete-upload.
- **FR-008**: The platform MUST call the resolved blob adapter's `PresignUpload` with a TTL equal to the request's effective `uploadExpiryMinutes` (default 15, range [5, 60]; see FR-002) and MUST return `{fileId, uploadUrl, method: "PUT", headers, expiresAt, maxSizeBytes}` on success. `expiresAt` in the response MUST match the scheduled abandonment expiry in FR-025. The presigned PUT MUST NOT include any provider public-read ACL header regardless of `visibility`; all uploads land as private objects in v1.
- **FR-008a**: `visibility` MUST be persisted on `file_object` exactly as supplied, but in v1 it is a **logical flag only**: it does not alter the presigned PUT, does not alter object ACL at complete time, and does not cause a public URL to be returned. Public delivery (object ACL / CDN / public URL) is explicitly deferred to a later feature.
- **FR-009**: If `PresignUpload` returns any error, the platform MUST NOT persist a `file_object` row (use a single DB transaction that only commits on success), and MUST map provider errors to `502/503` with structured codes `STORAGE_UNAVAILABLE`, `STORAGE_PERMISSION_DENIED`, `STORAGE_NOT_FOUND`.

**Abandoned-upload lifecycle (Redis-scheduled, reuses inventory reservation pattern)**

- **FR-025**: After persisting the `UPLOADING` row and generating the presigned URL, the platform MUST schedule a single delayed expiry job using `common/scheduler.Scheduler` (Redis sorted-set backed) with `delay = uploadExpiryMinutes` and command `file.upload.expiry`. The payload MUST contain the `file_object.id` and `fileId`. The returned Redis job id MUST be cached under a per-file key (e.g. `seller:{sellerId}:file.upload.expiry:{fileObjectId}` for sellers, `platform:file.upload.expiry:{fileObjectId}` for admins) with a TTL of `delay + cacheBufferDuration` — mirroring `inventory/service/reservation_scheduler_service.go`.
- **FR-026**: `complete-upload` MUST cancel the scheduled expiry job via the cached job id on successful verification (i.e. before or in the same transaction that sets `status = ACTIVE`). Cancellation failure MUST NOT fail the HTTP request (logged as warning); the expiry handler MUST be idempotent (see FR-029).
- **FR-027**: The `file.upload.expiry` handler, when invoked by the scheduler worker, MUST transition the `file_object` from `UPLOADING` → `FAILED` with reason `UPLOAD_EXPIRED` **only if** the row is still in `UPLOADING`. It MUST then best-effort call `DeleteObject` on the resolved blob adapter to remove any partially-uploaded stray object (failures are logged, not retried in v1).
- **FR-028**: After expiry (row is `FAILED` with reason `UPLOAD_EXPIRED`), any subsequent `complete-upload` for that `fileId` MUST return `409 UPLOAD_EXPIRED` and MUST NOT re-activate the record; the client is expected to re-initiate.
- **FR-029**: The expiry job MUST be idempotent: if it runs after `complete-upload` has already succeeded (cancellation race / missed cancel), it MUST observe `status = ACTIVE` and exit without side effects (no status change, no provider delete, no queue publish).

**Request idempotency (`init-upload`)**

- **FR-030**: `POST /api/files/init-upload` MUST honour an optional HTTP header `Idempotency-Key` (length 8–128 ASCII characters, regex `^[A-Za-z0-9._~-]+$`; values outside this range MUST return `400 VALIDATION_FAILED` with code `IDEMPOTENCY_KEY_INVALID`). The header MUST NOT be required; requests without it are processed without dedupe.
- **FR-031**: When `Idempotency-Key` is present, the platform MUST look up Redis key `file:init:idem:{ownerType}:{ownerId}:{sha256(Idempotency-Key)}` (the raw header value is SHA-256 hashed before being embedded in the Redis key — this caps key length at a constant size and prevents header-value leakage in keyspace monitoring; see research R8). For **Admin** callers `{ownerId}` is the admin's authenticated user_id even though `owner_id` is NULL in the DB (this scopes admin idempotency keys per-admin without storing extra data; see FR-001). If it maps to an existing `fileId` whose `file_object` status is **`UPLOADING`**, the platform MUST NOT insert a new row, MUST NOT schedule a new expiry job, MUST NOT call `PresignUpload` if a valid presigned URL record is still cached (see FR-032), and MUST return the original `fileId` with an equivalent response shape.
- **FR-032**: The cached idempotency record MUST include at minimum `{fileId, expiresAt}`; if the cached `expiresAt` is still in the future, the platform MUST return the originally-issued `uploadUrl` + headers (cached alongside the record); otherwise it MUST re-issue a fresh presigned URL against the same `fileId` and update the cache. Under no circumstance may a duplicate `file_object` row be inserted for the same idempotency key within its TTL.
- **FR-033**: If `Idempotency-Key` is present but the cached record maps to a `file_object` whose status is no longer `UPLOADING` (e.g. `ACTIVE`, `FAILED`), the platform MUST return `409 IDEMPOTENCY_KEY_CONFLICT` and MUST NOT create a new upload; the client is expected to use a different key.
- **FR-034**: The idempotency record's TTL MUST equal `uploadExpiryMinutes + cacheBufferDuration` (matches the scheduler's cache TTL from FR-025), ensuring the window closes no later than the abandonment window.
- **FR-035**: The Idempotency-Key namespace MUST be scoped to `(ownerType, ownerId)`; the same key used by a different caller MUST NOT collide or expose information (lookups are strictly scoped by the authenticated caller).

**Complete upload**

- **FR-010**: `POST /api/files/complete-upload` MUST accept authenticated principals with role **Seller** or **Admin** (buyers/customers rejected with `403`; unauthenticated with `401`), accept required `{fileId}` and optional hint fields `{clientEtag}` (string — the ETag the client observed from the provider PUT response) and `{actualSizeBytes}` (int64 — the byte count reported by the provider). The caller's role MUST match the owning `owner_type` of the target record (an Admin cannot complete a Seller's upload and vice versa, returning `404 FILE_NOT_FOUND`).
- **FR-011**: The platform MUST look up `file_object` by `fileId` scoped by `(owner_type, owner_id, seller_id)` derived from the caller; non-matching records MUST return `404 FILE_NOT_FOUND` (no cross-tenant enumeration).
- **FR-012**: If the record is already `ACTIVE`, the platform MUST return `200` idempotently with the current state and MUST NOT publish a duplicate variant message.
- **FR-013**: If the record is `FAILED`, the platform MUST return `409 FILE_STATE_INVALID` and MUST NOT touch storage. (File objects are not soft-deleted in this feature; there is no `DELETED` status.)
- **FR-014**: For records in `UPLOADING`, the platform MUST call `HeadObject` on the resolved adapter; on `ErrBlobNotFound` it MUST return `409 FILE_NOT_UPLOADED_YET` and leave status as `UPLOADING`.
- **FR-015**: On successful `HeadObject` the platform MUST verify (a) reported size equals the `sizeBytes` declared at init; mismatch MUST transition status to `FAILED` and return `409 FILE_SIZE_MISMATCH`.
- **FR-016**: If the client supplied `clientEtag` or `actualSizeBytes` in the complete-upload request, the platform MUST compare each against the corresponding value returned by `HeadObject` on the resolved adapter. If `clientEtag` is present and does not match the provider-reported ETag, OR `actualSizeBytes` is present and does not match the provider-reported size, the platform MUST transition status to `FAILED` and return `422 FILE_UPLOAD_OBJECT_MISMATCH`. Both fields are optional trust-but-verify hints; the platform always performs `HeadObject` regardless (FR-014).
- **FR-017**: On successful verification the platform MUST persist `size_bytes`, `mime_type`, `e_tag`, provider `last_modified`, and set `status = ACTIVE` atomically (single DB transaction).
- **FR-018**: The platform evaluates variant generation via `upload_policy.EvaluateVariants(purpose, mimeType)` at complete time (not init time), because `SELLER_LOGO` accepts both raster and SVG but only raster uploads produce variants. The rules are: if `purpose ∈ {PRODUCT_IMAGE, USER_AVATAR}` → `HasVariants = true`; if `purpose = SELLER_LOGO` AND `mimeType ≠ image/svg+xml` → `HasVariants = true` (raster variants `[thumb_200, webp_400]`); if `purpose = SELLER_LOGO` AND `mimeType = image/svg+xml` → `HasVariants = false` (SVG passthrough — no `file_job` row inserted, no variant message published); for all other purposes → `HasVariants = false`. When `HasVariants = true` the platform MUST publish `file.image.process.requested` on exchange `ecom.commands` with payload `{fileId, fileObjectId, storageConfigId, bucketOrContainer, objectKey, mimeType, sizeBytes, purpose, variantsRequested}` (variant codes per purpose from `upload_policy.go`; e.g. `["thumb_200","thumb_600","webp_1600"]` for `PRODUCT_IMAGE`) and insert a corresponding `file_job` row with `status = PUBLISHED` on broker confirm, or `status = FAILED_TO_PUBLISH` on confirm timeout / nack.
- **FR-019**: Variant-message publish failures MUST NOT fail the HTTP request; the platform MUST log the error and the `file_object` MUST remain `ACTIVE`. (Reconciler retry is a later feature.)
- **FR-020**: Both endpoints MUST write a structured audit log entry (action, actor, file_id, provider_code, bucket, key, ip, user_agent) without exposing raw credentials.

**Cross-cutting**

- **FR-021**: All error responses MUST use the existing structured error envelope (`code`, `message`, `details`) and MUST NOT leak provider credentials, endpoints, or stack traces.
- **FR-022**: All responses MUST be JSON with `application/json; charset=utf-8`.
- **FR-023**: The platform MUST support `ctx` propagation (request cancellation → cancels provider and DB calls).
- **FR-024**: Migrations MUST be idempotent and match the columns declared in `file/entity/file.go`. Because the `file_object` / `file_variant` migration from the predecessor features has not yet been merged to `develop`, **no new migration file is created by this feature**; the existing `migrations/018_create_file_storage_tables.sql` is edited in place to (a) drop the `metadata JSONB` column from both tables, and (b) keep all other columns aligned with the entity. Corresponding `Metadata db.JSONMap` fields in `file/entity/file.go` MUST be removed.

### Testing Requirements _(mandatory, per user's explicit direction)_

- **TR-001**: All acceptance scenarios above MUST be covered by **integration tests** using Testcontainers: real Postgres, real MinIO (S3-compatible), real RabbitMQ. Unit-level mocking is allowed only for pure-logic helpers (e.g., key sanitiser, policy evaluator), not for DB, storage, or queue.
- **TR-002**: Tests MUST exercise the full HTTP path through the Gin router (same bootstrap as production), including the seller-auth middleware with a real JWT.
- **TR-003**: Tests MUST verify the presigned URL works end-to-end: they MUST `PUT` bytes using the returned URL and headers, not via the SDK bypass.
- **TR-004**: Tests MUST assert on the published RabbitMQ message: queue binding, routing key `file.image.process.requested`, envelope fields (`messageId`, `eventType`, `tenantId`, `payload.fileObjectId`), using a test consumer that drains with a bounded timeout.
- **TR-005**: A dedicated test MUST cover the RabbitMQ-outage path (publisher returns error) and verify that the HTTP response is still `200 ACTIVE` and the record is persisted.
- **TR-006**: Tenant isolation and idempotency scenarios MUST each have a dedicated test.
- **TR-007**: Error-mapping tests MUST verify the exact structured error code returned for every `4xx`/`5xx` branch listed in FR-009, FR-014–FR-016.
- **TR-008**: Tests MUST NOT rely on `time.Sleep` for message delivery; use a blocking receive with a bounded deadline from the test consumer.
- **TR-009**: An integration test MUST cover the abandoned-upload path using the real Redis scheduler worker (Testcontainer): call `init-upload` with a short `uploadExpiryMinutes`, start the scheduler worker pool, and poll the DB/Redis with a bounded deadline until the row reaches `FAILED / UPLOAD_EXPIRED` and the Redis job key is gone; assert a subsequent `complete-upload` returns `409 UPLOAD_EXPIRED`.
- **TR-010**: An integration test MUST cover the scheduler cancellation path: `init-upload` → `complete-upload` success → assert the Redis job id is no longer in the sorted set (`delayed_jobs`) and the cache key is deleted. A follow-up test MUST exercise FR-029 by forcing the job to fire against an already-`ACTIVE` row and asserting no side effects.
- **TR-011**: An integration test MUST cover the `Idempotency-Key` happy path (two back-to-back calls with the same key return the same `fileId`, a single DB row, and a single scheduled expiry job) and the `IDEMPOTENCY_KEY_CONFLICT` branch (retry against an already-`ACTIVE` record). Cross-tenant isolation of the key namespace (FR-035) MUST also be covered.

### Key Entities _(include if feature involves data)_

- **`file_object`** (already in `file/entity/file.go`): canonical row for a user-uploaded file. Written by init (`UPLOADING`), updated by complete (`ACTIVE`/`FAILED`). Scoped by `owner_type/owner_id/seller_id` and bound to exactly one `storage_config`.
- **Enumerations** (reused from `file/entity/file.go`, not redeclared):
  - `FilePurpose`: `PRODUCT_IMAGE | IMPORT_FILE | EXPORT_FILE | DOCUMENT | USER_AVATAR | SELLER_LOGO | INVOICE_PDF`
  - `FileVisibility`: `PRIVATE | PUBLIC | INTERNAL`
  - `FileStatus`: `UPLOADING | ACTIVE | FAILED` (no soft-delete in this feature)
  Request DTOs and validators for init/complete MUST use these typed constants; binding layer MUST reject any other value with `400 VALIDATION_FAILED`. No new enum values are introduced by this feature.
- **`file_job`** (already in `file/entity/file.go`): row inserted on complete-upload for image-variant purposes. `status = PUBLISHED` on broker confirm (or `FAILED_TO_PUBLISH` on timeout/nack), `input_file_id = file_object.id`. Status transitions to `DONE` by the future variant-worker consumer.
- **Presigned URL contract** (no DB): `{uploadUrl, method, headers[], expiresAt, maxSizeBytes}` returned by init; consumed only by the client, never persisted.
- **Variant command envelope** (existing `common/messaging.Envelope`): published on `ecom.commands`, routing key `file.image.process.requested`, payload `ImageProcessRequested` from `file/RABBITMQ_FILE_MODULE_DESIGN.md §6`.
- **Upload-expiry scheduled job** (reuses `common/scheduler.ScheduledJob`): Redis sorted-set entry with `command = "file.upload.expiry"` and payload `{fileObjectId, fileId}`. Scheduled at `init-upload`, cancelled at `complete-upload`. Job id is cached in Redis under a per-file key for cancellation, exactly mirroring the inventory reservation pattern.
- **Init-upload idempotency record** (Redis, no DB): key `file:init:idem:{ownerType}:{ownerId}:{sha256(Idempotency-Key)}`, value `{fileId, fingerprint, uploadUrl, headers, expiresAt}`, TTL = `uploadExpiryMinutes + cacheBufferDuration`. Consumed only by `init-upload` retries; never persisted.

## Flow Diagrams

### Init upload → client PUT → complete upload

```text
Seller Client             Platform API             Postgres           Blob Provider        RabbitMQ
     |                         |                        |                     |                  |
     | 1. POST /init-upload    |                        |                     |                  |
     |------------------------>|                        |                     |                  |
     |                         | 2. validate policy     |                     |                  |
     |                         |    (size/mime/purpose) |                     |                  |
     |                         |                        |                     |                  |
     |                         | 3. resolve storage cfg |                     |                  |
     |                         |    (seller → platform) |                     |                  |
     |                         |                        |                     |                  |
     |                         | 4. BEGIN TX            |                     |                  |
     |                         |----------------------->|                     |                  |
     |                         | 5. INSERT file_object  |                     |                  |
     |                         |    status=UPLOADING    |                     |                  |
     |                         |----------------------->|                     |                  |
     |                         |                        |                     |                  |
     |                         | 6. PresignUpload       |                     |                  |
     |                         |----------------------------------------------| (signing only)   |
     |                         |<---------------------------------------------|                  |
     |                         | 7. COMMIT              |                     |                  |
     |                         |----------------------->|                     |                  |
     | 8. 200 {fileId, url, …} |                        |                     |                  |
     |<------------------------|                        |                     |                  |
     |                                                                                           |
     | 9. PUT <presigned URL>                                                                    |
     |------------------------------------------------------------------------->                 |
     |                                                                                           |
     | 10. POST /complete-upload {fileId}                                                        |
     |------------------------>|                        |                     |                  |
     |                         | 11. SELECT file_object |                     |                  |
     |                         |----------------------->|                     |                  |
     |                         | 12. tenant check +     |                     |                  |
     |                         |     state check        |                     |                  |
     |                         |                        |                     |                  |
     |                         | 13. HeadObject         |                     |                  |
     |                         |---------------------------------------------->                  |
     |                         |<----------------------------------------------                  |
     |                         | 14. size/mime/etag verify                    |                  |
     |                         |                        |                     |                  |
     |                         | 15. BEGIN TX           |                     |                  |
     |                         |     UPDATE file_object |                     |                  |
     |                         |     status=ACTIVE      |                     |                  |
     |                         |     INSERT file_job    |                     |                  |
     |                         |     COMMIT             |                     |                  |
     |                         |----------------------->|                     |                  |
     |                         |                        |                     |                  |
     |                         | 16. Publish (best-effort)                    |                  |
     |                         |--------------------------------------------------------------->|
     |                         |                                                                 |
     | 17. 200 {fileId, ACTIVE}|                                                                 |
     |<------------------------|                                                                 |
```

### State machine for `file_object.status`

```text
              init-upload OK
              (schedules expiry job)
    (nil) ───────────────────────▶ UPLOADING
                                      │
            complete: HeadObject      │ ┌──── size/mime/checksum mismatch ───────────┐
            not found                 │ │                                             │
                                      ▼ ▼                                             ▼
                                 (stays UPLOADING) ── scheduler fires ──▶ FAILED (reason=*)
                                      │                (reason=UPLOAD_EXPIRED,
            complete: Head OK &       │                 best-effort DeleteObject)
            verification OK           │
            (cancels expiry job)      │
                                      ▼
                                   ACTIVE (terminal — no soft-delete in this feature)
```

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: A seller can complete the full upload journey (init → PUT → complete) for a 1 MB JPEG in **under 3 seconds** on the local Testcontainers stack on a developer laptop.
- **SC-002**: **100%** of the acceptance scenarios (US1–US6) pass as integration tests, each finishing in **≤ 20 seconds** on CI.
- **SC-003**: **Zero** orphan `file_object` rows are created when the provider is unavailable at init time (asserted by the storage-outage integration test).
- **SC-004**: **Exactly one** image-variant message is published per successfully completed `PRODUCT_IMAGE` upload, including under duplicate `complete-upload` calls (asserted by test consumer count).
- **SC-005**: **Zero** cross-tenant finalisations succeed in the isolation test (US4).
- **SC-006**: **No** provider credentials, endpoints, or stack traces appear in any `4xx`/`5xx` response body across all integration tests (asserted by a shared response-scanning helper).
- **SC-007**: p95 of `init-upload` ≤ **300 ms** and p95 of `complete-upload` ≤ **500 ms** on the local Testcontainers stack, measured across the integration suite.
- **SC-008**: Code coverage for `file/handler`, `file/service` (upload paths), and `file/repository` upload methods ≥ **85%** line coverage, driven by integration tests (not unit stubs).
- **SC-009**: **Every** `init-upload` that reaches `HTTP 200` has a corresponding scheduled expiry job in Redis; **every** `complete-upload` that reaches `HTTP 200 ACTIVE` has no remaining expiry job in Redis (asserted in TR-010). Zero rows may remain in `UPLOADING` past `uploadExpiryMinutes + cacheBufferDuration` when the scheduler worker is running.

## Assumptions

- Seller authentication middleware (`middleware.SellerAuth`) and JWT issuance already work and are reused as-is.
- The blob adapter layer and factory (`file/service/blob_adapter`) from feature 002 are stable and provide `PresignUpload`, `HeadObject`, and the structured error set used here.
- Storage config activation (feature 001) is available: each seller test uses an active storage config pointing at MinIO; a platform default config exists for non-seller cases.
- RabbitMQ infra in `common/messaging/rabbitmq` is already wired; only new routing keys / queues for `file.image.process.requested` are declared by this feature.
- Image variant generation **worker** itself is out of scope; this feature only **publishes** the command. A stub consumer (or test-only consumer) asserts delivery.
- Maximum file size via this flow is 50 MB; multipart uploads for larger files are out of scope.
- Clock source is `time.Now().UTC()`; presigned URL expiry is expressed in UTC in responses.
- All new tests live under `test/integration/file/` and reuse existing container helpers (`minio_container.go`, RabbitMQ test helper, Postgres container helper).
