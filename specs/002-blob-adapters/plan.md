# Implementation Plan: BlobAdapter Layer for Multi-Cloud File Storage

**Branch**: `002-blob-adapters` | **Date**: 2026-04-12 | **Spec**: [/home/kushal/Work/Personal Codes/Ecommerce/ecommerce-be/specs/002-blob-adapters/spec.md](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/specs/002-blob-adapters/spec.md)
**Input**: Feature specification from `/specs/002-blob-adapters/spec.md`

## Summary

Implement a provider-agnostic `BlobAdapter` abstraction under `file/service/blob_adapter` with three concrete adapter families (S3-compatible, GCS, Azure Blob) plus an adapter factory that decrypts `StorageConfig.CredentialsEncrypted`, resolves provider adapter type, constructs the correct adapter, and returns standardized categorized errors. Deliver integration tests for all adapter methods with real/emulated providers (MinIO required via Testcontainers, plus emulator-backed GCS and Azure in CI-capable setup).

## Technical Context

**Language/Version**: Go 1.25+  
**Primary Dependencies**: Gin, GORM, validator, testify/suite, Testcontainers infrastructure (`test/integration/setup`), AWS SDK v2 (`service/s3`), GCS storage client, Azure Blob SDK  
**Storage**: PostgreSQL 16 for `storage_config`/`storage_provider` source records; external blob storage providers (S3-compatible, GCS, Azure Blob) for object data  
**Testing**: Go `testing` + `testify`, integration suites under `test/integration/file/` with Testcontainers-managed provider containers/emulators  
**Target Platform**: Linux server (containerized backend service and CI runners)  
**Project Type**: Modular monolith web-service backend  
**Performance Goals**: Adapter operations honor context deadlines; presign and metadata operations complete within existing service p95 targets (<200ms for metadata/presign paths in normal network conditions). **This is an aspirational design constraint, not a gated CI requirement** — container-based integration tests run in noisy environments and cannot reliably assert wall-clock latency. The 200ms target applies to real cloud/on-prem provider deployments and will be enforced by load testing in the future endpoint feature that wires file upload/download API handlers.  
**Constraints**: Strict handler->service->repository layering; no credential leakage in logs/errors; factory must own decryption; no dependency from adapter package to HTTP route/handler layers  
**Scale/Scope**: New `file/service/blob_adapter` package (interface, models, factory, provider implementations), dependency wiring updates, and integration coverage for all seven methods across three adapter families

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Modular Monolith boundary**: PASS. Changes remain inside `file` module plus shared crypto/helper usage.
- **Layered Architecture**: PASS. Blob adapter package is service-layer infrastructure; handlers/repositories are not bypassed.
- **Factory-Singleton DI**: PASS. Adapter factory will be consumed via file service/factory wiring without global mutable state.
- **TDD & Integration-first**: PASS WITH SCOPE EXCEPTION. This feature adds no HTTP endpoints; there is no handler or route to exercise. Integration-first testing is satisfied by real provider container tests (MinIO via Testcontainers for S3-compatible, Fake-GCS-Server for GCS, Azurite for Azure) covering all seven `BlobAdapter` interface methods end-to-end. The `blob_adapter` package is service-layer infrastructure (analogous to a database driver); HTTP lifecycle coverage (`HTTP → Middleware → Handler → Service`) will be provided by the future feature that wires file upload/download endpoints and calls the adapter factory. Direct database querying in tests is not applicable here — provider side-effects are verified via the adapter return values (ETags, ObjectMeta, stream content). This exception is explicitly approved; scope boundary stated in `spec.md:L168`.
- **Multi-tenant seller isolation**: PASS. Tenant scope continues to be resolved before adapter creation through existing config resolution path.
- **Correlation ID & tracing**: PASS. Adapter methods accept context and preserve trace propagation.
- **Backward compatibility**: PASS. No existing API contracts changed; this introduces internal adapter capability.
- **Performance & scalability**: PASS. Context-aware IO and bounded presign operations align with constitution requirements.

## Project Structure

### Documentation (this feature)

```text
specs/002-blob-adapters/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── blob-adapter-contract.md
└── tasks.md
```

### Source Code (repository root)

```text
file/
├── factory/
│   └── singleton/
│       └── service_factory.go
├── service/
│   ├── config_service.go
│   └── blob_adapter/
│       ├── adapter.go
│       ├── models.go
│       ├── errors.go
│       ├── factory.go
│       ├── s3_compatible_adapter.go
│       ├── gcs_adapter.go
│       └── azure_blob_adapter.go
└── utils/
    └── (existing crypto/config helpers reused)

test/integration/file/
├── blob_adapter_s3_integration_test.go
├── blob_adapter_gcs_integration_test.go
└── blob_adapter_azure_integration_test.go

test/integration/setup/
└── (existing shared testcontainer/bootstrap utilities extended as needed)
```

**Structure Decision**: Keep all new adapter abstractions and implementations within `file/service/blob_adapter` to preserve module ownership and allow higher-level file services to stay provider-agnostic. Integration tests live in `test/integration/file` with shared setup extensions for MinIO, Fake-GCS, and Azurite.

## Phase 0: Research Summary

See `research.md`. All technical ambiguities are resolved, including provider SDK choice, signed URL strategy, credential decryption boundary, error normalization categories, and containerized integration testing approach.

## Phase 1: Design Summary

- Interface/domain model and validation contracts are documented in `data-model.md`.
- Adapter behavioral contract is documented in `contracts/blob-adapter-contract.md`.
- Execution and verification path is documented in `quickstart.md`.
- Agent context update executed via `.specify/scripts/bash/update-agent-context.sh codex`.

## Post-Design Constitution Check

- **Architecture boundaries**: PASS. Adapter package remains internal to file module and does not cross module repository boundaries.
- **Layer compliance**: PASS. Business orchestration remains in service layer; adapters are infrastructure dependencies.
- **TDD requirement**: PASS BY PLAN WITH SCOPE EXCEPTION. Quickstart enforces writing integration tests for each provider adapter path against real/emulated provider containers. HTTP lifecycle tests are deferred to the future file upload/download endpoint feature (see C1 exception rationale in Constitution Check above).
- **Tenant isolation & RBAC**: PASS. Storage scope/ownership remains enforced before adapter construction.
- **Backward compatibility**: PASS. Existing endpoints and contracts stay intact while internal storage capability expands.
- **Observability/error standards**: PASS. Categorized error mapping and context propagation are built into adapter/factory contracts.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
