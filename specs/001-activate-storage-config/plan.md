# Implementation Plan: Storage Config Activation and Listing

**Branch**: `001-activate-storage-config` | **Date**: 2026-04-11 | **Spec**: [/home/kushal/Work/Personal Codes/Ecommerce/ecommerce-be/specs/001-activate-storage-config/spec.md](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/specs/001-activate-storage-config/spec.md)
**Input**: Feature specification from `/specs/001-activate-storage-config/spec.md`

## Summary

Implement `POST /api/files/storage-config/{id}/activate` and `GET /api/files/storage-config` in the file module with strict token-driven scope rules: seller-context calls are seller-scoped, while no-seller-context calls are platform-scoped. Replace the old active-config stub behavior with filtered/paginated listing, introduce explicit list models and query parsing, extend repository/service contracts for scope-safe filtering and activation, and add comprehensive integration coverage for auth, validation, scope isolation, filter semantics, concurrency, and standardized error handling.

## Technical Context

**Language/Version**: Go 1.25+  
**Primary Dependencies**: Gin, GORM, validator, testify/suite, Testcontainers infrastructure (`test/integration/setup`)  
**Storage**: PostgreSQL 16 (primary), Redis 7 (existing app dependency, not central to this feature)  
**Testing**: Go `testing`, `testify/suite`, integration tests in `test/integration/file/`  
**Target Platform**: Linux server (containerized backend service)  
**Project Type**: Modular monolith web-service backend  
**Performance Goals**: p95 list and activate API latency under 200ms for typical page sizes (<=20) in integration environment parity expectations  
**Constraints**: Strict layered architecture; seller isolation; standardized response/error format; correlation ID required; backward compatibility with existing save/provider endpoints  
**Scale/Scope**: Single file module feature; 2 endpoints; repository+service+handler+route+model+error/constants updates; integration suite expansion in existing `config_test.go` and/or adjacent test files

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Modular Monolith boundary**: PASS. Changes remain within `file/` module and shared `common/` contracts only.
- **Layered Architecture**: PASS. Plan keeps handler -> service -> repository flow; no layer bypass.
- **Factory-Singleton DI**: PASS. Existing singleton factory wiring remains; only interfaces/implementations extended.
- **TDD & Integration-first**: PASS WITH ENFORCEMENT. Implement by writing/expanding integration tests first for both endpoints and all clarified scenarios.
- **Multi-tenant seller isolation**: PASS. Token-derived scope enforced in listing and activation.
- **Correlation ID & tracing**: PASS. Existing middleware already enforces; tests include correlation-id negative behavior continuity.
- **RBAC**: PASS. Endpoint access constrained to seller-and-higher via middleware and service checks.
- **Backward compatibility**: PASS. Existing endpoints/contracts unaffected except replacing old `GET /storage-config/active` behavior by listing endpoint requirement in this feature branch.
- **Performance & scalability**: PASS. Pagination/sorting/filter query design and indexed scope predicates are preserved.

## Project Structure

### Documentation (this feature)

```text
specs/001-activate-storage-config/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── storage-config.openapi.yaml
└── tasks.md
```

### Source Code (repository root)

```text
file/
├── container.go
├── entity/
│   └── storage.go
├── error/
│   └── config_errors.go
├── handler/
│   └── config_handler.go
├── model/
│   └── config_model.go
├── repository/
│   └── config_repository.go
├── route/
│   └── storage_config_routes.go
├── service/
│   └── config_service.go
└── utils/
    └── constant/
        └── config_constants.go

test/integration/file/
└── config_test.go
```

**Structure Decision**: Use existing `file` module structure and existing file integration suite. Add/extend models for list filters and list responses in `file/model/`, repository list/activation queries in `file/repository/`, orchestration/authorization in `file/service/`, endpoint parsing and response wiring in `file/handler/`, route replacement in `file/route/`, and feature coverage in `test/integration/file/`.

## Phase 0: Research Summary

See `research.md`. All previously ambiguous areas are resolved: token-driven scope, forbidden sellerId query filter, ownerType filter removal, activation idempotency and concurrency expectations, and error/validation behavior.

## Phase 1: Design Summary

- Data model and filter semantics documented in `data-model.md`.
- HTTP API contracts documented in `contracts/storage-config.openapi.yaml`.
- Execution and verification flow documented in `quickstart.md`.
- Agent context updated via `.specify/scripts/bash/update-agent-context.sh codex`.

## Post-Design Constitution Check

- **Architecture boundaries**: PASS. No cross-module repository access introduced.
- **Layer compliance**: PASS. Activation/listing business logic remains in service.
- **TDD requirement**: PASS BY PLAN. Quickstart mandates writing/adjusting integration tests before implementation completion.
- **Tenant isolation & RBAC**: PASS. Data scope derives from token context only; no sellerId filter allowed.
- **Backward compatibility**: PASS WITH NOTE. `GET /storage-config/active` is intentionally superseded in this feature and covered by contract updates/tests.
- **Observability/error standards**: PASS. Plan includes standardized AppError mapping and correlation-id behavior preservation.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
