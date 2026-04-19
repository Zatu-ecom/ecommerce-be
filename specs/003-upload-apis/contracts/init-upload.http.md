# Contract: `POST /api/files/init-upload`

(Admin equivalent: `POST /api/admin/files/init-upload` — same body, auth header from admin JWT.)

## Headers

| Header | Required | Purpose |
|---|---|---|
| `Authorization: Bearer <jwt>` | Yes | Seller or Admin role |
| `X-Correlation-ID` | Yes | Rejected 400 if missing |
| `Content-Type: application/json` | Yes | |
| `Idempotency-Key` | Optional | See US5a / FR-030..FR-035 |

## Request body

```json
{
  "purpose": "PRODUCT_IMAGE",
  "visibility": "PRIVATE",
  "filename": "Hero Shot.JPG",
  "mimeType": "image/jpeg",
  "sizeBytes": 842317,
  "uploadExpiryMinutes": 15
}
```

| Field | Type | Required | Rules |
|---|---|---|---|
| `purpose` | enum (§4.1) | Yes | |
| `visibility` | enum (§4.2) | No (default `PRIVATE`) | |
| `filename` | string | Yes | 1..255 chars |
| `mimeType` | string | Yes | Must match purpose policy |
| `sizeBytes` | int64 | Yes | 1..purpose.maxSize |
| `uploadExpiryMinutes` | int | No (default 15) | 5..60 |

No `metadata` field is accepted; unknown fields are rejected by `ShouldBindJSON` with a 400.

## 201 Created (happy path)

```json
{
  "success": true,
  "message": "Upload initialised",
  "data": {
    "fileId": "018f2c1a-7a3e-7b2c-b4e2-c2a9d3e80001",
    "status": "UPLOADING",
    "uploadUrl": "https://minio.local/my-bucket/seller/42/PRODUCT_IMAGE/2026/04/018f2c1a-7a3e-7b2c-b4e2-c2a9d3e80001-hero-shot.jpg?X-Amz-...",
    "uploadMethod": "PUT",
    "uploadHeaders": {
      "Content-Type": "image/jpeg"
    },
    "objectKey": "seller/42/PRODUCT_IMAGE/2026/04/018f2c1a-7a3e-7b2c-b4e2-c2a9d3e80001-hero-shot.jpg",
    "expiresAt": "2026-04-18T10:15:00Z"
  }
}
```

Guarantees:
- `fileId` is stable across idempotent retries.
- `uploadUrl` is a single-part `PUT`. Client must echo `uploadHeaders` on the PUT.
- `expiresAt = now + uploadExpiryMinutes`.

## 200 OK (idempotent replay, same fingerprint)

Same body shape as 201; `uploadUrl` is **regenerated fresh** so the client's retry still has enough lifetime, but `fileId`, `objectKey`, and `expiresAt` are the ones stored under the idempotency record.

## Error responses

| HTTP | `code` | Trigger |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Bad JSON, missing `purpose`/`filename`/etc., invalid enum |
| 400 | `VALIDATION_ERROR` | `Idempotency-Key` malformed |
| 401 | `FILE_UPLOAD_UNAUTHORIZED` | JWT missing/invalid |
| 403 | `FILE_UPLOAD_FORBIDDEN` | Customer role, or role missing from token context |
| 409 | `FILE_UPLOAD_CONFLICT` | Same `Idempotency-Key`, different fingerprint |
| 412 | `FILE_UPLOAD_NO_STORAGE_CONFIG` | Seller has no active binding and no platform default |
| 422 | `FILE_UPLOAD_POLICY_VIOLATION` | Size > purpose max, or mime not allowed, or filename empty after sanitisation |
| 503 | `FILE_UPLOAD_STORAGE_UNAVAILABLE` | Blob adapter factory or presign failed |

All error bodies use `common.ErrorWithCode(...)` shape:

```json
{
  "success": false,
  "message": "File exceeds maximum size for PRODUCT_IMAGE",
  "code": "FILE_UPLOAD_POLICY_VIOLATION"
}
```

No provider credentials, bucket names, or internal paths appear in error messages (SC-006).

## Side effects on success

1. Row inserted into `file_object` with `status='UPLOADING'` and `upload_expires_at` computed.
2. Redis sorted set `delayed_jobs` gains a `ScheduledJob{command=file.upload.expiry, payload={fileObjectId}}` (FR-025).
3. Redis hash at `file:init:idem:{...}` set with `{fileId, fingerprint}` and TTL (only when header supplied).
4. A single presign call is issued to the provider (no GET; no HEAD).

## Side effects on error

- Validation/policy/config errors: no DB write, no Redis write, no provider call (except config resolution which is read-only).
- Presign failure: DB insert is rolled back (single-statement transaction).
