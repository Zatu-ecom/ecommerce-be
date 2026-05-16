# Feature Specification: File Read, Download URL & Delete APIs

**Feature Branch**: `004-file-read-delete-apis`  
**Created**: 2026-05-15  
**Status**: Draft  
**Input**: User description: "File Read, Download URL, and Delete APIs for the file module"

---

## Overview

This feature completes the file module's CRUD lifecycle by delivering the **read and delete** capabilities that the upload feature (003) established the write path for. Four new endpoints enable sellers to:

1. **List and filter** their uploaded files in batch — powering the seller dashboard and enabling efficient in-process file resolution for product listings.
2. **View detailed metadata** for a single file, optionally with a time-limited download link.
3. **Generate short-lived download links** for private files, with control over expiry and browser download behaviour.
4. **Permanently remove** a file and all its storage artefacts when no longer needed.

### Inter-Service Context

The Product Service needs file information when serving consumer-facing product listings. Since the platform is a modular monolith (single binary), the Product Service accesses file data through an **in-process service interface** — not over the network. Consumers never interact with file endpoints directly; their requests are served entirely through the product module.

---

## User Scenarios & Testing _(mandatory)_

### User Story 1 — Seller Lists and Filters Uploaded Files (Priority: P1)

A seller navigating the admin dashboard views a paginated list of their uploaded files. They can filter by file purpose (e.g. product images, documents), file status (active, failed), and file type (JPEG, PDF, etc.). They can also look up specific files by ID. The system ensures they only ever see their own files — never files belonging to another seller.

**Why this priority**: This is the primary read path used by both the seller dashboard and the internal product listing flow. Without batch retrieval, every product listing would require individual file lookups, creating a severe performance bottleneck.

**Independent Test**: As an authenticated seller with several previously uploaded files across different purposes and statuses, request the file list with various filter combinations and verify the returned items match the applied filters, respect tenant boundaries, and include correct pagination metadata.

**Acceptance Scenarios**:

1. **Given** an authenticated seller with 12 uploaded files (mix of PRODUCT_IMAGE and DOCUMENT, mix of ACTIVE and FAILED), **When** they request the file list with no filters, **Then** the response contains only their own files, default pagination (page 1, 20 per page), and total count reflects 12.
2. **Given** the same seller, **When** they filter by `purposes=PRODUCT_IMAGE`, **Then** only PRODUCT_IMAGE files are returned.
3. **Given** the same seller, **When** they filter by `purposes=PRODUCT_IMAGE,SELLER_LOGO` (multiple purposes), **Then** files matching either purpose are returned.
4. **Given** the same seller, **When** they filter by `statuses=ACTIVE,FAILED` (multiple statuses), **Then** files matching either status are returned.
5. **Given** the same seller, **When** they filter by `mimeTypes=image/jpeg,image/webp`, **Then** only files with matching file types are returned.
6. **Given** the same seller, **When** they combine `purposes=PRODUCT_IMAGE` and `mimeTypes=image/jpeg`, **Then** only PRODUCT_IMAGE files that are also JPEGs are returned (filters are ANDed across dimensions).
7. **Given** the same seller, **When** they request specific file IDs where one belongs to another seller, **Then** the cross-tenant ID is silently omitted; only owned files are returned.
8. **Given** the same seller with 12 files, **When** they request page size 5, **Then** page 1 shows 5 files and total pages equals 3.
9. **Given** the same seller, **When** they request files sorted by size ascending, **Then** files are returned in correct size order.
10. **Given** the same seller, **When** they request with `includeVariants=true`, **Then** each file item includes its associated variant information.

---

### User Story 2 — Seller Views a Single File's Details (Priority: P1)

A seller clicks on a specific file in the dashboard to see its full metadata — filename, size, type, status, storage provider label, upload timestamps, and any generated variants (thumbnails, etc.). Optionally, the seller can request a time-limited download link in the same call to avoid a separate round-trip.

**Why this priority**: The single-file detail view is the second most common read pattern and the only way to get comprehensive file information including variant details and an embedded download link.

**Independent Test**: As an authenticated seller, request metadata for a known active file with and without the download link option, and verify the response includes all expected fields and correctly handles the link generation.

**Acceptance Scenarios**:

1. **Given** an authenticated seller with an active PRODUCT_IMAGE file that has thumbnail variants, **When** they request the file's details, **Then** the response includes all metadata fields (ID, status, purpose, visibility, filename, type, size, ETag, storage provider, timestamps) and the variants array.
2. **Given** the same seller, **When** they request the file's details with `includeDownloadUrl=true`, **Then** the response includes a time-limited download link and its expiry timestamp.
3. **Given** the same seller requesting a file in UPLOADING status, **When** they view the details, **Then** metadata is returned with status UPLOADING and no download link (regardless of the download link flag).
4. **Given** a seller requesting another seller's file by ID, **When** they attempt to view details, **Then** the system returns a "not found" response (never reveals the file exists to a cross-tenant caller).
5. **Given** an admin user, **When** they request a platform-owned file's details, **Then** metadata is returned. **When** they request a seller-owned file, **Then** "not found" is returned.

---

### User Story 3 — Seller Generates a Download Link for a File (Priority: P2)

A seller needs a fresh, short-lived download link for a private file to embed in an email, share with a partner, or render in a page. They may also want a link for a specific variant (e.g. a thumbnail) and can control whether the browser should display the file inline or prompt a download.

**Why this priority**: Dedicated download link generation is needed when the embedded link in the metadata response has expired or when the seller needs links for specific variants or with specific download behaviour.

**Independent Test**: As an authenticated seller, generate download links for an active file with various TTLs, for variants, and with different browser behaviours. Verify the links are valid and expire as expected.

**Acceptance Scenarios**:

1. **Given** an authenticated seller with an active private file, **When** they request a download link with default settings, **Then** the response includes a valid link that expires in 15 minutes, along with the file type and size.
2. **Given** the same seller, **When** they request a download link with TTL of 5 minutes, **Then** the link expires in approximately 5 minutes.
3. **Given** the same seller, **When** they request a download link with TTL of 60 minutes, **Then** the link expires in approximately 60 minutes.
4. **Given** the same seller with a file that has a ready thumbnail variant, **When** they request a download link for that variant, **Then** the response returns a link for the variant's storage location and reports the variant's type and size.
5. **Given** the same seller, **When** they request a download link with `disposition=attachment`, **Then** the generated link includes a hint that causes the browser to prompt a download dialog.
6. **Given** the same seller requesting a download link for a file in UPLOADING status, **When** the request is made, **Then** the system responds with a conflict error indicating the file is not active.
7. **Given** the same seller requesting a download link for a non-existent variant code, **When** the request is made, **Then** the system responds with a "not found" error for the variant.
8. **Given** the same seller with a file whose visibility is set to PUBLIC, **When** they request a download link, **Then** the system responds with "not implemented" (public URL delivery via CDN is planned for a future release).

---

### User Story 4 — Seller Permanently Deletes a File (Priority: P2)

A seller decides to remove a file they no longer need. The system permanently removes the file record, all its variant records, and all associated storage blobs. The delete is synchronous — the seller sees confirmation only after the storage objects have been removed.

**Why this priority**: Cleanup capability is essential for storage cost control and data hygiene. The hard-delete approach avoids accumulating orphan storage objects that would require separate cleanup processes.

**Independent Test**: As an authenticated seller, delete files in various statuses (active, uploading, failed) and verify that the storage blobs are gone, all database records are removed (including variants and jobs via cascade), and subsequent requests for the deleted file return "not found."

**Acceptance Scenarios**:

1. **Given** an authenticated seller with an active file that has no variants, **When** they delete the file, **Then** the storage blob is removed, the database record is removed, and a subsequent request for this file returns "not found."
2. **Given** an authenticated seller with an active file that has two variants, **When** they delete the file, **Then** the original blob and both variant blobs are removed from storage, and all database records (file, variants, jobs) are cascade-deleted.
3. **Given** an authenticated seller with a file in UPLOADING status, **When** they delete it, **Then** the scheduled expiry job is cancelled, the database record is removed, and any partially-uploaded storage object is cleaned up.
4. **Given** an authenticated seller with a FAILED file, **When** they delete it, **Then** the database record is removed. The storage blob may or may not exist — the system handles both cases gracefully.
5. **Given** a seller attempting to delete another seller's file, **When** the request is made, **Then** "not found" is returned and the target file is untouched.
6. **Given** the storage provider is unreachable when deleting the primary blob, **When** the delete is attempted, **Then** the system returns a service unavailable error and the database record is NOT deleted (ensuring consistency).
7. **Given** a variant's storage blob fails to delete but the primary blob succeeds, **When** the overall delete completes, **Then** the delete is still reported as successful (best-effort for variants), a warning is logged, and the database records are removed.
8. **Given** two concurrent delete requests for the same file, **When** both execute, **Then** the first succeeds and the second returns "not found."

---

### User Story 5 — Product Service Resolves File Data for Consumer Listings (Priority: P1)

When a consumer browses the home page, the Product Service needs to resolve file metadata (URLs, thumbnails) for all products on the page. It does this through an in-process call to the File Service — not over the network. The consumer's authentication token never reaches the file module.

**Why this priority**: This is the most performance-sensitive path. Every product listing page load triggers this flow, and it must resolve multiple files efficiently in a single batch call.

**Independent Test**: Through the in-process service interface, request metadata for a batch of file IDs belonging to multiple sellers and verify all matching files are returned regardless of tenant boundaries (the service layer trusts its in-process callers).

**Acceptance Scenarios**:

1. **Given** the Product Service calling the File Service in-process with 20 file IDs, **When** all IDs exist, **Then** all 20 file metadata records are returned.
2. **Given** the Product Service calling with file IDs where some have been deleted, **When** the call completes, **Then** only existing files are returned; missing IDs are silently omitted.
3. **Given** a consumer browsing the home page, **When** the consumer's request triggers product listing, **Then** the consumer's authentication token is never forwarded to the File Service.

---

### Edge Cases

- What happens when a seller requests a file list with more than 100 file IDs? → The system rejects the request with a validation error.
- What happens when a seller requests a download link with a TTL outside the allowed range (below 5 or above 60 minutes)? → Validation error returned.
- What happens when the storage provider is temporarily unreachable during a metadata request with download link? → Metadata is returned successfully without the download link (degraded mode); a warning is logged.
- What happens when a file has been hard-deleted and a subsequent request arrives? → "Not found" is returned — identical to a file that never existed.
- What happens when a scheduled expiry fires for an already-deleted UPLOADING file? → The expiry handler finds no record and exits as a no-op.
- What happens when an unknown file purpose or status value is passed as a filter? → Validation error returned.
- What happens when the correlation tracking header is missing from any request? → The system rejects the request with a validation error before any processing occurs.

---

## Requirements _(mandatory)_

### Functional Requirements

**Batch List / Filter (GET files)**

- **FR-001**: System MUST allow sellers to list their own uploaded files with pagination (configurable page size 1–100, default 20).
- **FR-002**: System MUST support filtering files by multiple purposes simultaneously (e.g. PRODUCT_IMAGE and SELLER_LOGO in a single request), with values ORed within the filter.
- **FR-003**: System MUST support filtering files by multiple statuses simultaneously, with ACTIVE as the default when no status filter is provided.
- **FR-004**: System MUST support filtering files by multiple file types (MIME types) simultaneously.
- **FR-005**: System MUST support batch lookup of up to 100 specific file IDs in a single request, silently omitting IDs that do not exist or belong to another tenant.
- **FR-006**: System MUST support sorting results by creation date, file size, or filename in ascending or descending order.
- **FR-007**: System MUST optionally include variant information nested within each file item when requested.
- **FR-008**: Filters across different dimensions (purpose, status, file type) MUST be combined with AND logic; values within a single dimension MUST be combined with OR logic.

**Single File Metadata (GET file by ID)**

- **FR-009**: System MUST return complete file metadata for a single file, including all variants, regardless of the file's status (UPLOADING, ACTIVE, FAILED).
- **FR-010**: System MUST optionally generate and include a time-limited download link in the metadata response when requested, only for ACTIVE files with PRIVATE visibility.
- **FR-011**: When the download link cannot be generated due to a provider issue, the system MUST still return the metadata without the link (degraded mode) and log a warning.

**Download URL Generation (GET download-url)**

- **FR-012**: System MUST generate a short-lived download link for active, private files with configurable expiry (5–60 minutes, default 15).
- **FR-013**: System MUST support generating download links for specific file variants when a valid variant code is provided.
- **FR-014**: System MUST support controlling browser download behaviour (inline display vs. download prompt) through a disposition parameter.
- **FR-015**: System MUST return a "not implemented" response for files with PUBLIC visibility until CDN-based delivery is available.
- **FR-016**: System MUST reject download link requests for files that are not in ACTIVE status, and for variants that are not in READY status.

**Delete (DELETE file)**

- **FR-017**: System MUST permanently remove the file's storage blobs (original + all variants) and all database records in a single synchronous operation.
- **FR-018**: System MUST delete the primary storage blob before removing the database record. If the primary blob deletion fails, the database record MUST be preserved.
- **FR-019**: Variant blob deletions MUST be best-effort — individual failures are logged but do not block the overall delete.
- **FR-020**: System MUST support deleting files in any status (UPLOADING, ACTIVE, FAILED). For UPLOADING files, the scheduled expiry job MUST be cancelled first.
- **FR-021**: System MUST treat already-absent storage blobs as successful deletions (idempotent delete).

**Cross-Cutting**

- **FR-022**: All four endpoints MUST enforce seller-only authentication. Buyer/consumer roles MUST receive a "forbidden" response. Unauthenticated requests MUST receive an "unauthorized" response.
- **FR-023**: All requests MUST include a correlation tracking header. Requests missing this header MUST be rejected with a validation error before any processing.
- **FR-024**: Tenant isolation MUST be enforced on all endpoints. A seller MUST only see, download, and delete their own files. Cross-tenant attempts MUST return "not found" — never "forbidden" (to prevent enumeration).
- **FR-025**: Storage credentials, bucket names, and internal configuration MUST never appear in any response or error message.
- **FR-026**: All mutating operations (delete) and download link generations MUST produce structured audit log entries.

### Key Entities

- **File Object**: The primary record representing an uploaded file. Contains the file's unique ID, status (UPLOADING, ACTIVE, FAILED), purpose (PRODUCT_IMAGE, DOCUMENT, etc.), visibility (PRIVATE, PUBLIC, INTERNAL), original filename, MIME type, size, ETag, storage location reference, and timestamps. Owned by a seller or the platform.
- **File Variant**: A derived version of a file (e.g. thumbnail, webp conversion). Linked to a File Object. Contains variant code, MIME type, size, dimensions, status (PENDING, READY, FAILED), and storage location reference. Cascade-deleted when the parent File Object is removed.
- **File Job**: A record of an async processing command associated with a File Object. Cascade-deleted when the parent File Object is removed.

---

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: All test scenarios (67+ acceptance tests) pass as automated integration tests with real infrastructure (not mocked).
- **SC-002**: Batch file listing with 20 file IDs responds within 100ms at the 95th percentile.
- **SC-003**: Single file metadata retrieval responds within 200ms at the 95th percentile.
- **SC-004**: Download link generation (including provider round-trip) responds within 500ms at the 95th percentile.
- **SC-005**: File deletion (including storage cleanup) responds within 300ms at the 95th percentile.
- **SC-006**: Zero cross-tenant data leaks detected across all tenant isolation tests.
- **SC-007**: Batch listing with file IDs returns only files owned by the authenticated caller; cross-tenant IDs are never returned.
- **SC-008**: If the primary storage blob deletion fails, the file record is preserved in the database (verified by a subsequent successful read).
- **SC-009**: Variant storage blob failures during delete do not block the delete response or prevent database cleanup.
- **SC-010**: Storage provider credentials never appear in any error response across all endpoints.
- **SC-011**: Code coverage for new modules is at least 85%, driven by integration tests.
- **SC-012**: Every request missing the correlation tracking header returns a validation error across all four endpoints.

---

## Assumptions

- The file upload feature (003) is fully implemented and operational — file objects exist in the database with correct statuses and storage locations.
- Existing database cascade constraints (ON DELETE CASCADE) on variant and job tables are in place and functional.
- The blob storage adapter layer already provides presigned download URL generation and object deletion capabilities.
- The scheduler infrastructure from the upload feature is available for expiry job cancellation during UPLOADING file deletion.
- The shared integration test infrastructure (containerised database, storage, and message queue) established in the upload feature is available for reuse.
- Public file delivery via CDN is a future feature; download link generation for PUBLIC files is intentionally deferred.
- The in-process service interface pattern (dependency injection) already exists in the monolith and is used by other modules.
- Array-valued query parameters follow the established comma-separated string pattern used in the inventory module's filter APIs.
