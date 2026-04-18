# Blob Adapter Contract

## Scope

Contract for `file/service/blob_adapter` package that defines:
- provider-agnostic blob operations interface
- factory resolution contract from `StorageConfig`
- standardized error behavior

This is an internal module contract (non-HTTP).

## Interface Signature Contract

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

## Behavioral Guarantees

1. `PutObject`
- Requires non-empty bucket/key/contentType and non-nil body.
- Returns non-empty `Key`; returns `ETag` when provider emits one.

2. `DeleteObject`
- Deletes target object key in provided bucket/container.
- Not-found semantics may be idempotent by provider; adapter must still return categorized errors for hard failures.

3. `HeadObject`
- Returns metadata (`ContentType`, `SizeBytes`, `LastModified`, `ETag`) for existing object.
- Missing object returns `not_found` category.

4. `GetObjectStream`
- Returns stream + metadata for existing object.
- Stream must remain readable until caller closes it.

5. `PresignUpload`
- Requires `TTL > 0`; invalid TTL returns `validation` category.
- Returns expiring upload URL without exposing backend credentials.

6. `PresignDownload`
- Requires `TTL > 0`; invalid TTL returns `validation` category.
- Returns expiring download URL (SAS URL for Azure).

7. `CopyObject`
- Performs source->destination copy within same provider account/config.
- Cross-provider copy is out-of-scope for this interface.

## Factory Contract

Factory API behavior:
- Input: resolved `StorageConfig` + linked provider metadata (`adapter_type`).
- Flow:
  1. decrypt `CredentialsEncrypted`
  2. validate credential payload for adapter type
  3. create provider client and adapter instance
- Dispatch:
  - `s3_compatible` -> `S3CompatibleAdapter`
  - `gcs` -> `GCSAdapter`
  - `azure` -> `AzureBlobAdapter`
  - unknown type -> error category `validation` or `internal` with explicit unsupported-type message

## Error Contract

Every returned error from adapters/factory must include:
- category: `not_found | permission_denied | network | validation | internal`
- provider context (`s3_compatible|gcs|azure`) where applicable
- operation context (`put`, `head`, `get_stream`, `presign_upload`, etc.)

Prohibitions:
- no raw credentials in error strings
- no decrypted credential blobs in logs/test output

## Integration Test Contract

Minimum integration assertions per adapter family:
- all seven interface methods are exercised end-to-end
- invalid credential path returns categorized failure
- deadline/cancellation path returns context/network style failure without goroutine leak indicators

Provider targets:
- S3-compatible: MinIO via Testcontainers (mandatory)
- GCS: Fake-GCS-Server container (preferred CI path)
- Azure: Azurite container
