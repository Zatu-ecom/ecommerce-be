# Data Model: BlobAdapter Layer for Multi-Cloud File Storage

## 1) Core Interface Model

### `BlobAdapter` (non-persisted contract)
Provider-agnostic interface used by file services.

Methods:
- `PutObject(ctx, in PutObjectInput) (PutObjectOutput, error)`
- `DeleteObject(ctx, bucket, key string) error`
- `HeadObject(ctx, bucket, key string) (ObjectMeta, error)`
- `GetObjectStream(ctx, bucket, key string) (io.ReadCloser, ObjectMeta, error)`
- `PresignUpload(ctx, in PresignUploadInput) (PresignOutput, error)`
- `PresignDownload(ctx, in PresignDownloadInput) (PresignOutput, error)`
- `CopyObject(ctx, in CopyObjectInput) error`

### Invariants
- `context.Context` is always first parameter and must be honored for cancellation/deadline.
- `GetObjectStream` caller owns `ReadCloser.Close()`.
- All method errors are normalized to shared adapter error categories.

## 2) Operation Input/Output Models

### `PutObjectInput`
- `Bucket string` (required)
- `Key string` (required)
- `ContentType string` (required)
- `ContentLength int64` (required, >= 0)
- `Body io.Reader` (required)

### `PutObjectOutput`
- `Key string`
- `ETag string`
- `VersionID *string` (optional provider value)

### `ObjectMeta`
- `ContentType string`
- `SizeBytes int64`
- `LastModified time.Time`
- `ETag string`

### `PresignUploadInput`
- `Bucket string` (required)
- `Key string` (required)
- `ContentType string` (required)
- `ContentLengthLimit int64` (optional, >0 when set)
- `TTL time.Duration` (required, >0)

### `PresignDownloadInput`
- `Bucket string` (required)
- `Key string` (required)
- `TTL time.Duration` (required, >0)

### `PresignOutput`
- `URL string`
- `ExpiresAt time.Time`

### `CopyObjectInput`
- `SourceBucket string` (required)
- `SourceKey string` (required)
- `DestinationBucket string` (required)
- `DestinationKey string` (required)

## 3) Factory and Configuration Models

### `AdapterFactory` (non-persisted service component)

**Pinned public function signature**:

```go
// NewAdapterFromConfig decrypts credentials from cfg, validates the
// credential schema for the resolved provider type, and returns the
// appropriate BlobAdapter implementation. Returns a categorized
// BlobAdapterError on decryption, validation, or initialisation failure.
// cfg.Provider must be pre-loaded (GORM Preload or equivalent) before calling.
func NewAdapterFromConfig(ctx context.Context, cfg entity.StorageConfig) (BlobAdapter, error)
```

**Import boundary**:
- `StorageConfig` is defined in `ecommerce-be/file/entity` (`file/entity/storage.go`)
- `factory.go` resides in `ecommerce-be/file/service/blob_adapter` — both are within the same `file` module, so this import is a **legal intra-module reference** and does not violate the constitution's cross-module boundary rule
- `factory.go` MUST NOT import from any other module's entity/repository/service packages

**Preconditions**:
- `cfg.Provider` must be loaded (non-zero `AdapterType`) before the call — callers are responsible for preloading
- `ctx` is used to bound any decryption or client-init I/O; pass a deadline-aware context

Input:
- `StorageConfig` (persisted entity loaded upstream, `ecommerce-be/file/entity`)
- required provider metadata (`storage_provider.adapter_type` via `cfg.Provider.AdapterType`)

Responsibilities:
- decrypt `StorageConfig.CredentialsEncrypted`
- validate credential schema for resolved adapter type
- build concrete adapter instance
- return unknown-type/decryption/validation/init errors with category mapping

### `StorageConfig` fields consumed by factory
- `Provider.AdapterType` (`s3_compatible` | `gcs` | `azure`) — loaded via `cfg.Provider`
- `BucketOrContainer`
- `Region`
- `Endpoint`
- `ForcePathStyle`
- `CredentialsEncrypted`
- `ConfigJSON` (optional non-secret provider config)

### Decrypted credential payload schemas (internal)

#### `S3CompatibleCredentials`
- `AccessKeyID string`
- `SecretAccessKey string`
- optional: `SessionToken string`

#### `GCSCredentials`
- `ServiceAccountJSON string` (or parsed JSON object)
- optional: `ProjectID string` override

#### `AzureCredentials`
- `AccountName string`
- one of:
  - `AccountKey string`
  - `SAS string`

Validation rules:
- each adapter type must have all mandatory keys before any provider network call.
- malformed/empty credential payload returns factory validation error.

## 4) Concrete Adapter Types

### `S3CompatibleAdapter`
- Uses endpoint + region + force-path-style configuration.
- Supports AWS S3, R2, MinIO, B2 through compatible API.

### `GCSAdapter`
- Uses service-account credentials for object operations and signed URL generation.

### `AzureBlobAdapter`
- Uses account-name + key/SAS for blob operations and SAS URL generation.

## 5) Error Model

### `BlobAdapterError`
- `Category`: `not_found | permission_denied | network | validation | internal`
- `Provider`: `s3_compatible | gcs | azure`
- `Operation`: `put | delete | head | get_stream | presign_upload | presign_download | copy | factory_init`
- `Cause error` (wrapped, never exposing secrets)

### Mapping expectations
- missing key/blob/bucket -> `not_found`
- auth/signature/permission failures -> `permission_denied`
- timeouts/connection/reset/dns -> `network`
- invalid TTL/inputs -> `validation`
- unexpected SDK/runtime failures -> `internal`

## 6) State and Transition Notes

### Object lifecycle (provider-side)
- `PutObject` creates/overwrites object state.
- `HeadObject` and `GetObjectStream` read current state.
- `CopyObject` creates destination object from source.
- `DeleteObject` removes object; subsequent `HeadObject` should return `not_found`.

### Factory lifecycle
1. Receive `StorageConfig`
2. Resolve adapter type from provider
3. Decrypt credentials
4. Validate credential schema
5. Construct provider client + adapter
6. Return ready adapter or categorized error

No shared mutable global adapter cache is required for this feature scope; each factory call returns an independent adapter instance.
