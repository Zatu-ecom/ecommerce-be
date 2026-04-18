# Quickstart: BlobAdapter Layer for Multi-Cloud File Storage

## 1. Preconditions

- Docker available (Testcontainers pulls MinIO, Fake-GCS-Server, and Azurite on first run).
- Docker socket accessible to the test process:
  ```bash
  sudo usermod -aG docker $USER   # then re-login
  # or
  export DOCKER_HOST=unix:///run/user/$(id -u)/docker.sock   # rootless Docker
  ```

## 2. Required Environment Variables

| Variable         | Used By                       | Description                                                         |
|-----------------|-------------------------------|---------------------------------------------------------------------|
| `ENCRYPTION_KEY` | `blob_adapter.NewAdapterFromConfig` | AES-256 key (hex or raw string) for credential decryption. **Required** when `config.Get()` is not loaded (all integration tests set this via `t.Setenv`). |

No other environment variables are required for local development. Container endpoints
are assigned dynamically by Testcontainers.

## 3. Package Layout

```
file/
  service/blob_adapter/
    adapter.go              — BlobAdapter interface
    factory.go              — NewAdapterFromConfig (decrypt + dispatch)
    s3_compatible_adapter.go
    gcs_adapter.go
    azure_blob_adapter.go
    adapter_test.go         — compile-only interface assertions (T058)
  error/
    blob_adapter_errors.go  — ErrBlob* sentinels + IsBlobError helper
  model/
    blob_adapter_model.go   — DTOs (BlobPutObjectInput, BlobObjectMeta, …)
  factory/singleton/
    service_factory.go      — GetBlobAdapter(ctx, cfg) wiring point

test/integration/file/
  blob_test_helpers.go      — RandomHex, AssertNoSecretLeak, …
  blob_adapter_factory_test.go
  blob_adapter_s3_integration_test.go
  blob_adapter_gcs_integration_test.go
  blob_adapter_azure_integration_test.go
  minio_container.go
  fake_gcs_container.go
  azurite_container.go
```

## 4. Running Tests

### Full integration suite (requires Docker)

```bash
go test ./test/integration/file/... -v -count=1
```

### Provider-scoped runs

```bash
# S3-compatible (MinIO)
go test ./test/integration/file/... -run S3 -v -count=1

# GCS (fake-gcs-server)
go test ./test/integration/file/... -run GCS -v -count=1

# Azure Blob (Azurite)
go test ./test/integration/file/... -run Azure -v -count=1

# Factory + decryption unit tests
go test ./test/integration/file/... -run Factory -v -count=1
```

### Compile & vet (no Docker needed)

```bash
go build ./...
go vet ./...
```

### Adapter package only (unit tests + compile assertions)

```bash
go test ./file/service/blob_adapter/... -v -count=1
```

### Broader regression before merge

```bash
go test ./... -count=1
```

## 5. Wiring a BlobAdapter at Runtime

```go
// In a service or use-case that needs to perform blob operations:
//
// 1. Resolve the StorageConfig for the current tenant — Provider must be preloaded:
//    db.Preload("Provider").First(&cfg, tenantStorageConfigID)
//
// 2. Obtain the adapter via the service factory:
adapter, err := serviceFactory.GetBlobAdapter(ctx, cfg)
if err != nil {
    // err is a *commonError.AppError — safe to return to caller
    return err
}

// 3. Use the adapter:
out, err := adapter.PutObject(ctx, model.BlobPutObjectInput{
    Bucket:        cfg.BucketName,
    Key:           objectKey,
    ContentType:   "application/octet-stream",
    ContentLength: size,
    Body:          reader,
})
```

## 6. Expected Outcomes

- `NewAdapterFromConfig` returns correct concrete adapter for `s3_compatible`, `gcs`, `azure`.
- Unknown adapter type returns `ErrBlobValidation` with a descriptive message.
- Credential decryption failure returns `ErrBlobFactoryInit`; no secret appears in the error.
- All 7 `BlobAdapter` methods pass against provider emulators.
- `IsBlobError(err, fileError.ErrBlobNotFound)` is the correct way to inspect adapter errors.
