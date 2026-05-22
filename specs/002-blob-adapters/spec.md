# Feature Specification: BlobAdapter Layer for Multi-Cloud File Storage

**Feature Branch**: `002-blob-adapters`  
**Created**: 2026-04-12  
**Status**: Draft  
**Input**: User description: "Create BlobAdapter interface, provider-specific implementations (S3-compatible, GCS, Azure Blob), and an adapter factory, organised under `file/service/blob_adapter`. Verify each adapter type with integration tests. Scope is BlobAdapters only."

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 — S3-Compatible Blob Operations (Priority: P1)

A developer integrating the file module can perform all standard blob operations (upload, head, download-stream, presign upload URL, presign download URL, copy, delete) against any S3-compatible provider (AWS S3, Cloudflare R2, MinIO, Backblaze B2) through a single uniform adapter.

**Why this priority**: S3-compatible providers cover the majority of planned deployments (AWS S3 as the platform default, MinIO for self-hosted, R2 for edge). This adapter unlocks all file upload/download flows.

**Independent Test**: The S3-compatible adapter can be tested end-to-end against a real MinIO container using integration tests covering every method on the `BlobAdapter` interface. No other adapter needs to exist for this test to pass.

**Acceptance Scenarios**:

1. **Given** a valid S3-compatible configuration (endpoint, bucket, access key, secret key), **When** `PutObject` is called with a file payload, **Then** the object is stored and the returned output contains a non-empty `ETag` and `Key`.
2. **Given** an object that exists in the bucket, **When** `HeadObject` is called, **Then** the returned `ObjectMeta` contains correct `ContentType`, `SizeBytes`, and `LastModified`.
3. **Given** an object that exists, **When** `GetObjectStream` is called, **Then** a readable stream is returned along with correct `ObjectMeta`, and the full content can be read without error.
4. **Given** a valid configuration, **When** `PresignUpload` is called with a key and TTL, **Then** a pre-signed URL is returned that allows a client to PUT the object directly to storage without backend credentials.
5. **Given** an object that exists, **When** `PresignDownload` is called with TTL, **Then** a pre-signed URL is returned that allows a client to GET the object for the configured duration.
6. **Given** a source object that exists, **When** `CopyObject` is called with a destination bucket and key, **Then** the object is copied and accessible at the destination.
7. **Given** an object that exists, **When** `DeleteObject` is called, **Then** the object no longer exists and subsequent `HeadObject` returns a not-found error.
8. **Given** an invalid credential configuration, **When** any adapter method is called, **Then** a structured error is returned that identifies the provider as the failure source.

---

### User Story 2 — Factory Resolves Adapter by Provider Type (Priority: P1)

A file service developer passes a resolved `StorageConfig` record to the adapter factory and receives the correct `BlobAdapter` implementation without needing to know which concrete provider it is.

**Why this priority**: The factory is the single seam between the storage config (already implemented) and the blob operations layer. Without it, higher-level service code cannot use adapters in a provider-agnostic way.

**Independent Test**: Given a `StorageConfig` with `AdapterType = "s3_compatible"`, the factory returns an S3-compatible adapter instance. Given `AdapterType = "gcs"`, it returns a GCS adapter. Given `AdapterType = "azure"`, it returns an Azure adapter. An unknown type returns an error.

**Acceptance Scenarios**:

1. **Given** a `StorageConfig` whose provider has `adapter_type = "s3_compatible"`, **When** the factory is called, **Then** the returned adapter implements the full `BlobAdapter` interface and is ready to use.
2. **Given** a `StorageConfig` whose provider has `adapter_type = "gcs"`, **When** the factory is called, **Then** a GCS adapter is returned.
3. **Given** a `StorageConfig` whose provider has `adapter_type = "azure"`, **When** the factory is called, **Then** an Azure Blob adapter is returned.
4. **Given** a `StorageConfig` with an unknown `AdapterType`, **When** the factory is called, **Then** a descriptive error is returned and no adapter instance is created.
5. **Given** a `StorageConfig` with malformed / missing credentials, **When** the factory attempts to build the adapter, **Then** an initialisation error is returned before any network call is made.

---

### User Story 3 — GCS Blob Operations (Priority: P2)

A developer can perform all standard blob operations against a Google Cloud Storage bucket through the same `BlobAdapter` interface.

**Why this priority**: GCS is planned for Phase 2 multi-cloud support. It is independent of S3 and can be integrated after the S3 adapter and factory are stable.

**Independent Test**: Integration tests against a real GCS bucket (using service account credentials from test environment secrets) or a GCS emulator cover every `BlobAdapter` method.

**Acceptance Scenarios**:

1. **Given** a valid GCS configuration (project ID, bucket, service-account JSON), **When** `PutObject` is called, **Then** the object is stored and a valid `ETag` is returned.
2. **Given** an object that exists in GCS, **When** `PresignDownload` is called, **Then** a signed URL is returned allowing the caller to download the object.
3. **Given** invalid GCS credentials, **When** any adapter method is called, **Then** a structured error is returned identifying the failure.

---

### User Story 4 — Azure Blob Operations (Priority: P2)

A developer can perform all standard blob operations against an Azure Blob Storage container through the same `BlobAdapter` interface.

**Why this priority**: Azure Blob is planned for Phase 2. It is independent of S3 and GCS and can be integrated last among the three adapter families.

**Independent Test**: Integration tests against a real Azure Blob container (using Azurite emulator in tests) cover every `BlobAdapter` method.

**Acceptance Scenarios**:

1. **Given** a valid Azure configuration (account name, account key, container), **When** `PutObject` is called, **Then** the blob is stored and a valid `ETag` is returned.
2. **Given** a blob that exists, **When** `PresignDownload` is called, **Then** a SAS URL is returned allowing the caller to download the blob.
3. **Given** invalid Azure credentials, **When** any adapter method is called, **Then** a structured error is returned identifying the failure.

---

### User Story 5 — Credential Decryption Integration (Priority: P1)

The factory decrypts the credentials stored in `StorageConfig.CredentialsEncrypted` before building the adapter, so that no caller ever handles raw secrets.

**Why this priority**: Credentials are stored encrypted (AES envelope encryption, already implemented in `SaveConfig`). The factory must own the decryption step to prevent credentials from leaking into other layers.

**Independent Test**: Given a `StorageConfig` with AES-encrypted credentials, the factory decrypts them and successfully constructs a working S3-compatible adapter. If the encryption key is wrong, the factory returns a decryption error.

**Acceptance Scenarios**:

1. **Given** a `StorageConfig` with properly encrypted credentials, **When** the factory is invoked, **Then** the adapter is constructed and can communicate with the storage provider.
2. **Given** a `StorageConfig` with corrupted or tampered `CredentialsEncrypted`, **When** the factory is invoked, **Then** a decryption error is returned and no adapter instance is created.

---

### Edge Cases

- What happens if the bucket or container does not exist when `PutObject` is called? → Adapter returns a structured "bucket not found" error.
- What happens if the object key contains special characters or path segments? → Keys are passed to the provider as-is; callers own sanitisation.
- What happens if `PresignUpload` TTL is zero or negative? → The factory returns a validation error before issuing the request.
- What happens if `GetObjectStream` is called for a key that does not exist? → A not-found error is returned; no stream is opened.
- What happens if the same adapter factory is called concurrently for the same config? → Each call returns an independent adapter; there is no shared mutable state.
- What happens if network times out during a `PutObject` call? → The context deadline causes the operation to fail with a timeout error that wraps the provider error.

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST define a single `BlobAdapter` interface with exactly seven methods: `PutObject`, `DeleteObject`, `HeadObject`, `GetObjectStream`, `PresignUpload`, `PresignDownload`, and `CopyObject`.
- **FR-002**: The system MUST provide an S3-compatible adapter implementing `BlobAdapter`, capable of connecting to any S3-compatible provider (AWS S3, Cloudflare R2, MinIO, Backblaze B2) via configurable endpoint.
- **FR-003**: The system MUST provide a GCS adapter implementing `BlobAdapter` using Google service-account credentials.
- **FR-004**: The system MUST provide an Azure Blob adapter implementing `BlobAdapter` using account-name / account-key or SAS-based credentials.
- **FR-005**: The system MUST provide an `AdapterFactory` (or equivalent constructor) that accepts a `StorageConfig` record and returns the appropriate `BlobAdapter` implementation or an error.
- **FR-006**: The factory MUST decrypt `StorageConfig.CredentialsEncrypted` using the same AES envelope encryption scheme used by `SaveConfig` before constructing the adapter.
- **FR-007**: All adapter methods MUST accept a `context.Context` as their first argument and respect context cancellation and deadline.
- **FR-008**: All adapter methods MUST return errors that clearly identify the failure category (not-found, permission-denied, network, validation, internal) without exposing raw credentials.
- **FR-009**: `PresignUpload` and `PresignDownload` MUST accept a TTL duration input and return a URL that expires at the specified time.
- **FR-010**: `PutObject` MUST accept a `ContentType`, `ContentLength`, and an `io.Reader` payload.
- **FR-011**: `GetObjectStream` MUST return an `io.ReadCloser`; callers are responsible for closing it.
- **FR-012**: `CopyObject` MUST support cross-bucket copy within the same provider account.
- **FR-013**: The factory MUST return a descriptive error for any unknown or unsupported `AdapterType`.
- **FR-014**: Integration tests MUST exist for each adapter implementation, validating all seven interface methods against a real (or emulated) provider instance.
- **FR-015**: Integration tests for the S3-compatible adapter MUST run against a MinIO container managed by Testcontainers (consistent with the existing integration test infrastructure).
- **FR-016**: The `blob_adapter` package MUST reside under `file/service/blob_adapter/` and export the `BlobAdapter` interface, all input/output model types, and the factory.

### Key Entities

- **BlobAdapter**: The provider-agnostic interface that service layers depend on for all blob storage operations.
- **PutObjectInput**: Carries bucket, key, content-type, content-length, and payload reader for upload operations.
- **PutObjectOutput**: Carries the `ETag`, storage key, and provider-assigned version (if any) after a successful put.
- **ObjectMeta**: Common metadata returned by `HeadObject` and `GetObjectStream` — content-type, size in bytes, last-modified timestamp, ETag.
- **PresignUploadInput**: Carries bucket, key, content-type, content-length limit, and TTL duration.
- **PresignDownloadInput**: Carries bucket, key, and TTL duration.
- **PresignOutput**: Carries the pre-signed URL and its expiry time.
- **CopyObjectInput**: Carries source bucket, source key, destination bucket, and destination key.
- **AdapterFactory**: Resolves and constructs the correct `BlobAdapter` from a `StorageConfig` entity after credential decryption.
- **S3CompatibleAdapter**: Concrete implementation covering AWS S3, Cloudflare R2, MinIO, and Backblaze B2 via configurable endpoint.
- **GCSAdapter**: Concrete implementation for Google Cloud Storage.
- **AzureBlobAdapter**: Concrete implementation for Azure Blob Storage.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All seven `BlobAdapter` methods are implemented for all three adapter families (S3-compatible, GCS, Azure Blob) — verified by zero compilation errors and 100% interface-method coverage in tests.
- **SC-002**: Integration tests for the S3-compatible adapter pass against a MinIO container on every CI run without manual setup.
- **SC-003**: The factory correctly resolves the right adapter for all known `AdapterType` values and returns a typed error for unknown types — verified by unit tests with 100% branch coverage of factory dispatch logic.
- **SC-004**: No raw credentials (access keys, service-account JSON, account keys) appear in any error message, log line, or test assertion output.
- **SC-005**: All adapter method calls that exceed their context deadline return a context-deadline error within the bounds of that deadline, with no goroutine leaks — verified by context-cancellation test cases.
- **SC-006**: The `blob_adapter` package compiles with zero import cycles and has no dependency on any HTTP handler or route layer.

---

## Assumptions

- The existing AES envelope encryption/decryption utility in `common/helper/crypto.go` will be reused as-is for credential decryption in the factory — no new crypto logic is needed.
- The `StorageConfig` entity already contains all fields the factory needs (`AdapterType` via the loaded `Provider`, `BucketOrContainer`, `Region`, `Endpoint`, `ForcePathStyle`, `CredentialsEncrypted`).
- Integration tests for GCS and Azure may run against emulators (Fake-GCS-Server via Testcontainers for GCS, Azurite for Azure) rather than real cloud accounts in the CI environment.
- The `BlobAdapter` interface is the authoritative definition for this feature; no adapter-specific public types are exposed to callers outside the package.
- Presign operations for Azure Blob use SAS tokens as the equivalent of S3 pre-signed URLs.
- The `ForcePathStyle` flag in `StorageConfig` applies only to S3-compatible adapters (MinIO, some R2 setups).
- Scope boundary: this feature does NOT include storage resolver logic, upload/download API handlers, file registry persistence, or async workers — only the adapter layer and its factory.
- All provider SDK dependencies needed (AWS SDK v2, GCS client library, Azure SDK) will be added to `go.mod` as part of this feature.
