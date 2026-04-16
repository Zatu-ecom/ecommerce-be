# Quickstart: BlobAdapter Layer for Multi-Cloud File Storage

## 1. Preconditions

- Checkout branch: `002-blob-adapters`
- Docker available for Testcontainers-based integration tests.
- Test environment variables configured (encryption key and provider emulator configs if required by test harness).

## 2. Implement in TDD Order

1. Write/expand integration tests in `test/integration/file/`:
- S3-compatible adapter suite against MinIO container
- GCS adapter suite against Fake-GCS-Server container
- Azure adapter suite against Azurite container
- Factory dispatch + credential decryption failure/success paths

2. Create `file/service/blob_adapter` package skeleton:
- `adapter.go` (interface)
- `models.go` (input/output DTOs)
- `errors.go` (categorized adapter errors)
- `factory.go` (adapter resolution + decryption + validation)

3. Implement providers:
- `s3_compatible_adapter.go`
- `gcs_adapter.go`
- `azure_blob_adapter.go`

4. Wire consumption path:
- service factory or file-service integration points call adapter factory with resolved `StorageConfig`
- preserve existing module boundaries and DI patterns

5. Make tests pass and refactor error mapping/log hygiene.

## 3. Verification Commands

Run targeted integration tests:

```bash
go test ./test/integration/file/... -v -count=1
```

Run feature package tests:

```bash
go test ./file/service/blob_adapter/... -v -count=1
```

Run broader regression before merge:

```bash
go test ./... 
```

## 4. Expected Outcomes

- Factory returns correct concrete adapter for `s3_compatible`, `gcs`, `azure`.
- Unknown adapter type returns descriptive categorized error.
- Credential decryption failure blocks adapter construction.
- All adapter methods satisfy interface contract for each provider family.
- No raw secrets appear in errors/log output.
