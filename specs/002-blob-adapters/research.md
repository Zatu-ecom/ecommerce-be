# Phase 0 Research: BlobAdapter Layer for Multi-Cloud File Storage

## Decision 1: Define one stable `BlobAdapter` contract and keep provider SDK details private
- **Decision**: Expose exactly seven provider-agnostic methods from `file/service/blob_adapter` and keep all SDK-specific request/response types internal to concrete adapters.
- **Rationale**: Preserves interface stability across providers and avoids leaking provider-specific coupling into file services.
- **Alternatives considered**:
  - Provider-specific interfaces per adapter: rejected because callers would need branching logic by provider.
  - Generic map-based operations: rejected due to type-safety and maintainability loss.

## Decision 2: Use AWS SDK v2 S3 client for all S3-compatible providers
- **Decision**: Build one `S3CompatibleAdapter` using AWS SDK v2 with endpoint override and path-style toggles for AWS S3, Cloudflare R2, MinIO, and Backblaze B2.
- **Rationale**: One mature client supports all target S3-compatible endpoints with minimal branching and robust signing/presign support.
- **Alternatives considered**:
  - Separate clients per S3-compatible provider: rejected due to duplicated logic and inconsistent behavior.
  - MinIO Go client as primary: rejected because AWS SDK v2 better aligns with wider provider compatibility and existing ecosystem practices.

## Decision 3: Keep credential decryption in adapter factory only
- **Decision**: `AdapterFactory` decrypts `StorageConfig.CredentialsEncrypted` using existing AES envelope helper, validates required credential fields for the selected adapter type, then constructs the adapter.
- **Rationale**: Centralizing decryption prevents raw secret handling from spreading across services and adapter constructors.
- **Alternatives considered**:
  - Decrypt in caller service and pass plain credentials: rejected for secret sprawl risk.
  - Lazy decrypt inside each adapter method call: rejected for repetitive overhead and larger leakage surface.

## Decision 4: Standardize error taxonomy at adapter boundary
- **Decision**: Map provider errors into consistent categories: `not_found`, `permission_denied`, `network`, `validation`, `internal`, with provider context included but secrets excluded.
- **Rationale**: Service and API layers need predictable error semantics regardless of provider.
- **Alternatives considered**:
  - Return raw provider errors: rejected as unstable contract and potential credential leakage risk.
  - Collapse all errors to internal: rejected because callers need actionable behavior (retry/not-found/permission handling).

## Decision 5: Enforce TTL validation before presign execution
- **Decision**: `PresignUpload` and `PresignDownload` reject zero or negative TTL with validation errors before provider SDK call.
- **Rationale**: Explicit contract behavior for edge cases and prevention of provider-dependent inconsistencies.
- **Alternatives considered**:
  - Let providers enforce TTL semantics: rejected due to inconsistent error messages and behavior by SDK/provider.

## Decision 6: Implement copy semantics as intra-account operation per provider
- **Decision**: `CopyObject` supports cross-bucket/container copy within the same authenticated provider account/config; cross-provider copy is out-of-scope.
- **Rationale**: Matches FR-012 while keeping API and implementation bounded.
- **Alternatives considered**:
  - Cross-provider copy via read/write streaming: rejected as out-of-scope complexity for adapter-only feature.

## Decision 7: Integration strategy uses real/emulated providers via Testcontainers
- **Decision**: Required S3 integration tests run against MinIO Testcontainer; GCS tests run against Fake-GCS Server container; Azure tests run against Azurite container.
- **Rationale**: Meets integration-first constitution rule and validates actual protocol behavior without requiring external cloud accounts in CI.
- **Alternatives considered**:
  - Unit tests with mocks only: rejected because presign, streaming, and error mapping behavior need end-to-end validation.
  - Real cloud account integration in CI: rejected for cost/secret management instability.

## Decision 8: Keep adapter package independent of HTTP and repository layers
- **Decision**: `blob_adapter` package depends only on context, io, provider SDKs, and shared crypto/error helpers; it does not import handlers/routes/repositories.
- **Rationale**: Satisfies clean architecture and SC-006 (no HTTP-layer dependency).
- **Alternatives considered**:
  - Build adapters in handler/service files directly: rejected due to coupling and reduced testability.
