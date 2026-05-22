# Implementation Plan: Product File Integration

**Branch**: `[005-product-file-integration]` | **Date**: 2026-05-20 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/005-product-file-integration/spec.md`

**Setup Note**: `.specify/scripts/bash/setup-plan.sh --json` was executed with `SPECIFY_FEATURE=005-product-file-integration` because the current git branch is `feature/report-and-files`, which does not match Speckit's required feature-branch naming pattern. The generated paths target this feature folder.

## Summary

Integrate Product with the File module so products can store ordered media references, return storefront-ready product media summaries on detail/list responses, and support seller/admin media attachment, metadata updates, and removal. The implementation extends the existing Go modular monolith by adding a Product-owned media association, product service/repository/handler paths for media management, and cross-module service interfaces to the File module for validation, URL-enriched reads, and delete-on-detach cleanup.

## Technical Context

**Language/Version**: Go 1.25+  
**Primary Dependencies**: Gin, GORM, validator, testify/suite, Testcontainers, existing File module services  
**Storage**: PostgreSQL 16 for `product_media` association rows; external blob providers remain owned by File module  
**Testing**: Go testing with testify/suite and Testcontainers integration infrastructure  
**Target Platform**: Linux web-service backend  
**Project Type**: Modular monolith web service  
**Performance Goals**: Product list pages with 20 products include media without per-product visible loading; list media resolution uses a single batched file lookup for the returned page  
**Constraints**: Preserve Product/File module boundaries; all seller-scoped operations enforce seller isolation; no hard database foreign key to File module tables; no N+1 media resolution; route responses use existing standardized response helpers  
**Scale/Scope**: Product detail/list reads, media attach/update/remove endpoints, and integration tests for seller-admin product media management

**Integration test execution (repository standard)**:

- Tests live under `test/integration/product/product_media/` and use `test/integration/setup` (containers, migrations, seeds, `SetupTestServer`) plus `test/integration/helpers.APIClient` to call the real Gin `http.Handler`, matching `test/integration/product/product/create_product/create_product_test.go` and related files.
- Requests use the same JSON, headers, and auth flow as production clients (`Authorization`, `X-Seller-ID`, `X-Correlation-ID`). No mocks for router, middleware, or DB in primary scenarios.
- Optional unit tests only for isolated pure logic when HTTP coverage is impractical; feature behavior must still be fully covered per `spec.md` Verification & Testing.

## Constitution Check

**GATE**: Must pass before Phase 0 research. Re-check after Phase 1 design.

- **Modular Monolith Boundary**: PASS. Product owns `product_media` and stores File module `fileId` values only; Product accesses File behavior through service interfaces, not repositories or internal tables.
- **Clean Architecture / Layering**: PASS. Planned changes keep HTTP parsing in handlers, business orchestration in services, and persistence in repositories.
- **Factory-Singleton DI Pattern**: PASS. Product service dependencies will be wired through the existing singleton factory structure.
- **TDD & Integration-First Testing**: PASS. Integration tests are planned before implementation for all new endpoints and product response changes.
- **Multi-Tenant Seller Isolation**: PASS. Product media reads and writes remain product/seller scoped, and file validation/deletion uses caller principal context.
- **Correlation ID & Tracing**: PASS. New endpoints use existing route middleware and logging conventions.
- **RBAC**: PASS. Media management is seller/admin protected; product reads remain public with seller context.
- **Backward Compatibility**: PASS. Existing product routes remain; response additions are additive.
- **SOLID / Practical Design**: PASS. Add focused repository/service interfaces for media rather than bloating unrelated variant logic.
- **Performance & Scalability**: PASS. List reads explicitly batch product media and file metadata resolution.

## Project Structure

### Documentation (this feature)

```text
specs/005-product-file-integration/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── product-media.openapi.yaml
├── checklists/
│   └── requirements.md
├── spec.md
├── pre-spec.md
└── pre-spec-v2.md
```

### Source Code (repository root)

```text
migrations/
└── 019_create_product_media_table.sql

product/
├── entity/
│   └── product_media.go
├── model/
│   └── product_model.go
├── repository/
│   ├── product_media_repository.go
│   └── product_repository.go
├── service/
│   ├── product_media_service.go
│   ├── product_file_gateway.go
│   ├── product_query_service.go
│   └── product_service.go
├── handler/
│   └── product_handler.go
├── route/
│   └── product_route.go
├── factory/singleton/
│   ├── repository_factory.go
│   ├── service_factory.go
│   └── handler_factory.go
└── error/
    └── product_media_error.go

test/integration/product/product_media/
├── setup_suite_test.go
├── attach_media_test.go
├── update_media_test.go
├── remove_media_test.go
├── get_product_media_test.go
└── list_products_media_test.go
```

**Structure Decision**: Product media belongs inside the existing Product module because ordering and primary selection are product-specific business concerns. File storage and file metadata remain owned by the File module and are accessed only through service interfaces.

## Phase 0: Research

Research completed in [research.md](./research.md). All technical context items are resolved with no remaining `NEEDS CLARIFICATION` entries.

## Phase 1: Design & Contracts

Design artifacts generated:

- [data-model.md](./data-model.md)
- [quickstart.md](./quickstart.md)
- [contracts/product-media.openapi.yaml](./contracts/product-media.openapi.yaml)

## Post-Design Constitution Check

- **Boundary Compliance**: PASS. The data model avoids File table foreign keys and documents service-interface access only.
- **Layering Compliance**: PASS. Contracts map to handler/service/repository responsibilities without layer bypass.
- **Testing Compliance**: PASS. Quickstart and design call for integration-first tests covering happy paths, auth, validation, duplicates, not found, seller isolation, correlation ID, and file cleanup degradation.
- **Performance Compliance**: PASS. Data model and quickstart require batched media mapping for product lists and an index on `product_id`.
- **Backward Compatibility Compliance**: PASS. Product response changes are additive and media management uses new endpoints.

## Complexity Tracking

No constitution violations require justification.
