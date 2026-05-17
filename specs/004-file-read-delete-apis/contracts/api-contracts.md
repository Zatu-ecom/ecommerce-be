# API Contracts: File Read & Delete APIs (004)

**Feature**: `004-file-read-delete-apis`
**Branch**: `004-file-read-delete-apis`
**Base Path**: `/api/files`
**Auth**: All endpoints require `sellerAuth` middleware (SELLER or ADMIN JWT + X-Correlation-ID)

---

## Shared Headers (All Endpoints)

| Header | Required | Notes |
|---|---|---|
| `Authorization` | Yes | `Bearer <JWT>` — SELLER or ADMIN role |
| `X-Correlation-ID` | Yes | Rejected with `400 VALIDATION_FAILED` if missing |

---

## Shared Error Envelope

```json
{
  "success": false,
  "message": "<human-readable>",
  "code": "<ERROR_CODE>"
}
```

---

## API 1 — GET /api/files

Batch list/filter files. Tenant-scoped to the authenticated caller.

### Query Parameters

| Param | Type | Required | Default | Constraints |
|---|---|---|---|---|
| `fileIds` | comma-separated string | No | — | Max 100 entries |
| `purposes` | comma-separated string | No | — | Valid `FilePurpose` values ORed |
| `statuses` | comma-separated string | No | `ACTIVE` | Valid `FileStatus` values ORed |
| `mimeTypes` | comma-separated string | No | — | MIME strings ORed |
| `includeVariants` | bool | No | `false` | |
| `page` | int ≥ 1 | No | `1` | |
| `pageSize` | int [1,100] | No | `20` | |
| `sortBy` | enum | No | `createdAt` | `createdAt \| sizeBytes \| originalFilename` |
| `sortOrder` | enum | No | `desc` | `asc \| desc` |

### Success Response — 200 OK

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

> `variants` is `[]` when `includeVariants=false` or no variants exist.

### Error Responses

| Status | Code | Trigger |
|---|---|---|
| `400` | `VALIDATION_FAILED` | `fileIds` > 100; invalid `pageSize`; unknown enum in `purposes`/`statuses`; unknown `sortBy`/`sortOrder`; missing `X-Correlation-ID` |
| `401` | `UNAUTHORIZED` | Missing/invalid JWT |
| `403` | `FORBIDDEN` | Buyer/customer role |

---

## API 2 — GET /api/files/{fileId}

Single file metadata with optional embedded presigned download URL.

### Path Parameters

| Param | Type | Notes |
|---|---|---|
| `fileId` | UUIDv7 string | |

### Query Parameters

| Param | Type | Required | Default | Constraints |
|---|---|---|---|---|
| `includeDownloadUrl` | bool | No | `false` | |
| `urlTtlMinutes` | int [5,60] | No | `15` | Only used when `includeDownloadUrl=true` |

### Success Response — 200 OK

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

> `downloadUrl` and `downloadUrlExpiresAt` present only when `includeDownloadUrl=true` AND `status=ACTIVE` AND `visibility=PRIVATE`.
> On provider failure with `includeDownloadUrl=true`, metadata is returned without these fields (degraded mode).

### Error Responses

| Status | Code | Trigger |
|---|---|---|
| `400` | `VALIDATION_FAILED` | `urlTtlMinutes` outside [5,60]; missing `X-Correlation-ID` |
| `401` | `UNAUTHORIZED` | Missing/invalid JWT |
| `403` | `FORBIDDEN` | Buyer/customer role |
| `404` | `FILE_NOT_FOUND` | Not found or cross-tenant |
| `503` | `STORAGE_UNAVAILABLE` | PresignDownload fails (degraded mode is non-fatal — returns 200) |

---

## API 3 — GET /api/files/{fileId}/download-url

Generate a fresh short-lived presigned download URL.

### Path Parameters

| Param | Type | Notes |
|---|---|---|
| `fileId` | UUIDv7 string | Must be `ACTIVE` |

### Query Parameters

| Param | Type | Required | Default | Constraints |
|---|---|---|---|---|
| `ttlMinutes` | int [5,60] | No | `15` | |
| `variantCode` | string | No | — | Must be a valid READY variant for this file |
| `disposition` | enum | No | `inline` | `inline \| attachment` |

### Success Response — 200 OK

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

### Error Responses

| Status | Code | Trigger |
|---|---|---|
| `400` | `VALIDATION_FAILED` | `ttlMinutes` out of range; unknown `disposition`; missing `X-Correlation-ID` |
| `401` | `UNAUTHORIZED` | Missing/invalid JWT |
| `403` | `FORBIDDEN` | Wrong role |
| `404` | `FILE_NOT_FOUND` | fileId not found or cross-tenant |
| `404` | `VARIANT_NOT_FOUND` | `variantCode` not found for this file |
| `409` | `FILE_NOT_ACTIVE` | File is `UPLOADING` or `FAILED` |
| `409` | `VARIANT_NOT_READY` | Variant status is `PENDING` or `FAILED` |
| `501` | `NOT_IMPLEMENTED` | `visibility=PUBLIC` in v1 |
| `502` | `STORAGE_PERMISSION_DENIED` | Provider rejects presign |
| `503` | `STORAGE_UNAVAILABLE` | Provider unreachable |

---

## API 4 — DELETE /api/files/{fileId}

Synchronous hard-delete: blob + DB row.

### Path Parameters

| Param | Type | Notes |
|---|---|---|
| `fileId` | UUIDv7 string | |

### Success Response — 200 OK

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

### Error Responses

| Status | Code | Trigger |
|---|---|---|
| `400` | `VALIDATION_FAILED` | Missing `X-Correlation-ID` |
| `401` | `UNAUTHORIZED` | Missing/invalid JWT |
| `403` | `FORBIDDEN` | Wrong role |
| `404` | `FILE_NOT_FOUND` | Not found or cross-tenant |
| `409` | `FILE_DELETE_CONFLICT` | File referenced by active product (stub; future FK guard) |
| `503` | `STORAGE_UNAVAILABLE` | Original blob delete failed — DB row NOT deleted |

---

## In-Process Service Interface (Product Service Integration)

This is **not an HTTP endpoint**. It is the Go interface injected into Product Service via DI.

```go
// FileReadService is the in-process interface for cross-module file resolution.
// No authentication is required — callers are trusted within the process boundary.
type FileReadService interface {
    // GetFilesByIDs returns file metadata for the given file IDs.
    // Missing or non-existent IDs are silently omitted.
    // No tenant scoping — all files across all owners may be resolved.
    GetFilesByIDs(ctx context.Context, fileIDs []string) ([]*entity.FileObject, error)
}
```

---

## Error Code Reference

| Code | HTTP | Description |
|---|---|---|
| `FILE_NOT_FOUND` | 404 | File not found or cross-tenant attempt |
| `FILE_NOT_ACTIVE` | 409 | File is not in ACTIVE status for presign operations |
| `VARIANT_NOT_FOUND` | 404 | Variant code not found for this file |
| `VARIANT_NOT_READY` | 409 | Variant is PENDING or FAILED |
| `FILE_DELETE_CONFLICT` | 409 | File referenced by active product (future guard) |
| `STORAGE_PERMISSION_DENIED` | 502 | Provider rejected presign (bad credentials) |
| `STORAGE_UNAVAILABLE` | 503 | Provider unreachable or blob delete failed |
| `VALIDATION_FAILED` | 400 | Request validation failure |
| `UNAUTHORIZED` | 401 | Missing or invalid JWT |
| `FORBIDDEN` | 403 | Wrong role (buyer/customer) |
| `NOT_IMPLEMENTED` | 501 | Public file download via CDN (v1 deferral) |
