# Contract: Scheduled job `file.upload.expiry`

Produced by `POST /api/files/init-upload`. Consumed by the `UploadExpiryHandler` registered in the `file` module.

## Envelope (as stored by `common/scheduler`)

```json
{
  "id": "01HKX...-job-ulid",
  "command": "file.upload.expiry",
  "runAt": "2026-04-18T10:15:00Z",
  "correlationId": "7f1c...-original-request-cid",
  "attempt": 0,
  "payload": {
    "fileObjectId": 12345,
    "fileId": "018f2c1a-7a3e-7b2c-b4e2-c2a9d3e80001"
  }
}
```

The scheduler guarantees at-least-once delivery and honours `scheduler:cancelled:{id}` fast-path cancellation.

## Handler semantics

Pseudo-code (maps 1:1 to `file/service/upload_expiry_handler.go`):

```
Handle(ctx, job):
  row = repo.FindByID(ctx, job.payload.fileObjectId)
  if row == nil:
      return nil                              # already deleted; nothing to do (FR-029)
  if row.Status == ACTIVE or row.Status == FAILED:
      return nil                              # idempotent no-op (FR-029)
  if row.Status == UPLOADING and row.UploadExpiresAt <= now():
      # Best-effort: delete the stray object if present; ignore not-found
      adapter, _ := storageResolver.Resolve(ctx, row.StorageConfigID)
      if adapter != nil:
          _ = adapter.DeleteObject(ctx, row.BucketOrContainer, row.ObjectKey)
      repo.MarkFailed(ctx, row.ID, failureReason="UPLOAD_EXPIRED")
      return nil
  return nil                                  # race: caller completed between dispatch and handler
```

## Guarantees

- **Idempotency** (FR-029): handler safe to invoke N times for the same `fileObjectId` with no extra side effects after the first terminal transition.
- **Cancellation path**: on successful `complete-upload`, service calls `scheduler.Cancel(jobID)`. The handler also self-guards against the race above.
- **Blast radius**: one failed file_object row per job; no batch fan-out.
- **Deletion of stray object**: best-effort only; provider errors are logged but do not fail the job (the row moves to `FAILED` regardless).

## Failure modes

| Condition | Outcome |
|---|---|
| Redis unavailable at dispatch | `common/scheduler` retries with backoff; see scheduler contract |
| DB unavailable | Job retries; no mutation happens |
| Provider delete fails (network, 403) | Logged; row still moves to `FAILED` |
| Row already `ACTIVE` | Job no-ops (handler idempotent) |

## Observability

Every handler invocation emits a log line with `correlationId`, `fileId`, `fileObjectId`, `oldStatus`, `newStatus`, and `deletedObject` boolean. Counter metric `file_upload_expiry_handled_total{outcome}` is incremented (future metrics integration — not required in this feature, but left in log form now).
