# Contract: `POST /api/files/complete-upload`

(Admin equivalent: `POST /api/admin/files/complete-upload`.)

## Headers

| Header | Required | Purpose |
|---|---|---|
| `Authorization: Bearer <jwt>` | Yes | Seller or Admin |
| `X-Correlation-ID` | Yes | |
| `Content-Type: application/json` | Yes | |

## Request body

```json
{
  "fileId": "018f2c1a-7a3e-7b2c-b4e2-c2a9d3e80001",
  "clientEtag": "\"b3c4d5e6f7...\"",
  "actualSizeBytes": 842317
}
```

| Field | Type | Required | Rules |
|---|---|---|---|
| `fileId` | string | Yes | Must match a row the caller owns |
| `clientEtag` | string | No | When present, must equal provider `HeadObject` etag |
| `actualSizeBytes` | int64 | No | When present, must equal provider-reported size |

The service always performs a `HeadObject` regardless of client-supplied values, so the values are trust-but-verify hints only.

## 200 OK (happy path, status transitioned to ACTIVE)

```json
{
  "success": true,
  "message": "Upload completed",
  "data": {
    "fileId": "018f2c1a-7a3e-7b2c-b4e2-c2a9d3e80001",
    "status": "ACTIVE",
    "mimeType": "image/jpeg",
    "sizeBytes": 842317,
    "etag": "\"b3c4d5e6f7...\"",
    "completedAt": "2026-04-18T10:04:12Z",
    "variantsQueued": true
  }
}
```

`variantsQueued`:
- `true` → a RabbitMQ `file.image.process.requested` message was published and a `file_job` row written.
- `false` → purpose has no variants (FR-019).

## 200 OK (idempotent replay on already-ACTIVE)

Same body. `variantsQueued` reflects whatever happened on the first successful complete (i.e. replays do **not** re-publish variant commands; FR-016). Service re-reads `file_job` to populate the flag.

## Error responses

| HTTP | `code` | Trigger |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Missing `fileId`, bad UUID |
| 401 | `FILE_UPLOAD_UNAUTHORIZED` | JWT missing/invalid |
| 403 | `FILE_UPLOAD_FORBIDDEN` | Customer role, or role missing |
| 404 | `FILE_UPLOAD_NOT_FOUND` | `fileId` unknown, OR row owned by a different tenant (cross-tenant for admin also 404 per FR-011 update) |
| 409 | `FILE_UPLOAD_OBJECT_MISSING` | `HeadObject` returns NotFound — client PUT never arrived or still in flight |
| 410 | `FILE_UPLOAD_EXPIRED` | Row already `FAILED` via scheduler; caller must re-init |
| 422 | `FILE_UPLOAD_OBJECT_MISMATCH` | Provider size/mime/etag mismatch — row moves to `FAILED` |
| 503 | `FILE_UPLOAD_STORAGE_UNAVAILABLE` | Adapter/head call failed for transient reasons (row stays `UPLOADING` so client can retry) |

## Side effects on success (first complete)

1. `file_object.status` → `ACTIVE`; `etag`, `completed_at` set.
2. Scheduler `Cancel(jobID)` invoked for the expiry job (FR-026). Cancellation is best-effort; if Redis is unreachable, we still return 200 and log a warning — the scheduler handler is itself idempotent against `ACTIVE` rows (FR-029).
3. If purpose has variants: publish envelope to exchange `ecom.commands` with routing key `file.image.process.requested` AND insert `file_job{status=PUBLISHED}`. If publish fails we still return 200 but mark the job `FAILED_TO_PUBLISH` with `last_error` (FR-017, edge case "RabbitMQ outage").

## Side effects on idempotent replay

No new scheduler cancellation, no new variant message. DB reads only.

## Side effects on error

- `OBJECT_MISSING`: row stays `UPLOADING`; scheduler job stays; client may retry.
- `OBJECT_MISMATCH`: row transitions `UPLOADING → FAILED`; scheduler job cancelled; no variant publish.
- `EXPIRED`: no mutation (row was already `FAILED`).
