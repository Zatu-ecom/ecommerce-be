# Implementation Plan: Recently Viewed Products

**Branch**: `006-recently-viewed-products` | **Date**: 2026-07-15 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/006-recently-viewed-products/spec.md`

## Summary

Implement automatic recently-viewed product tracking triggered by the existing `GET /api/product/:productId` endpoint. When an authenticated customer fetches product details, a `(user_id, product_id, seller_id)` record is upserted in a new `user_recently_viewed` table. Only the last 10 entries per user are retained — the oldest is trimmed when the limit is exceeded. Recording is fire-and-forget: failures are logged but never break the product response. Only customers (role level 3) trigger recording; sellers (level 2) and admins (level 1) are excluded.

The technical approach follows the existing wishlist reference pattern: a new entity, repository, service, and handler integration within the product module, wired through the factory-singleton dependency injection chain. Integration tests use testify/suite with Testcontainers.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: Gin (HTTP), GORM (ORM), testify/suite (testing), Testcontainers (integration tests)
**Storage**: PostgreSQL 16 — new `user_recently_viewed` table
**Testing**: `go test` + testify assertions + testify/suite pattern following [`flash_sale_test.go`](../../test/integration/promotion/flash_sale_test.go)
**Target Platform**: Linux server (Dockerized)
**Project Type**: Web service (REST API) — modular monolith, product module
**Performance Goals**: Recently-viewed recording adds <5ms overhead per request; product response must never be delayed by recording failures
**Constraints**: Fire-and-forget recording; max 10 entries per user; must not break existing product API contract
**Scale/Scope**: ~10k users; per-user limit of 10 records; 1 migration file; 5 new source files; 4 test files; 8 modified files

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle                            | Status  | Evidence                                                                                                                                                                                                                                                        |
| ------------------------------------ | ------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **I. Modular Monolith**              | ✅ PASS | All new code lives within `product/` module. No cross-module repository access. Only uses `common/auth` helpers and `common/constants` — both are allowed cross-cutting concerns.                                                                               |
| **II. Clean Architecture**           | ✅ PASS | Handler → Service → Repository → DB flow maintained. Handler orchestrates (no business logic). Service contains trim-to-10 logic. Repository performs pure DB operations. Entity is a pure data struct.                                                         |
| **III. Factory-Singleton DI**        | ✅ PASS | New RecentlyViewedRepository, RecentlyViewedService follow the exact 4-file singleton chain (RepositoryFactory → ServiceFactory → HandlerFactory → SingletonFactory). `sync.Once` pattern preserved.                                                            |
| **IV. TDD & Integration-First**      | ✅ PASS | 17 integration tests written first across 4 files (setup + happy-path + sad-path + edge-case), covering all scenarios. Tests use full request lifecycle via APIClient.                                                                                          |
| **V. Multi-Tenant Seller Isolation** | ✅ PASS | `seller_id` is captured from the product response (populated by the existing multi-tenant query). Recording is scoped to the seller who owns the product.                                                                                                       |
| **VI. Correlation ID**               | ✅ PASS | Feature uses existing handler infrastructure which already enforces correlation ID middleware.                                                                                                                                                                  |
| **VII. RBAC**                        | ✅ PASS | Explicit role check: only `CUSTOMER_ROLE_LEVEL` (3) triggers recording. Sellers (2) and admins (1) are skipped via `GetUserRoleLevelFromContext()`. `PublicAPIAuth` middleware routes all roles — handler filters.                                              |
| **VIII. Backward Compatibility**     | ✅ PASS | Zero changes to existing API response contracts. `SellerID` field added to `ProductResponse` with `json:"-"` (not serialized). Recording is a side effect — product API behavior unchanged.                                                                     |
| **IX. SOLID**                        | ✅ PASS | Single Responsibility: one service for recently-viewed concerns. Interface Segregation: focused `RecentlyViewedRepository` (4 methods) and `RecentlyViewedService` (2 methods). Dependency Inversion: handler depends on service interface, not implementation. |
| **X. Performance**                   | ✅ PASS | Single-row upsert + conditional delete (<5ms). ON CONFLICT clause avoids read-before-write. Indexes on `user_id` and `(user_id, viewed_at DESC)`. Fire-and-forget prevents product response delay.                                                              |

**Gate Result**: ALL GATES PASS — No violations, no complexity tracking needed.

## Project Structure

### Documentation (this feature)

```text
specs/006-recently-viewed-products/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── api-contracts.md # API behavior contract
└── tasks.md             # Phase 2 output (/speckit.tasks command — NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
migrations/
└── 025_create_recently_viewed_table.sql    # NEW: DB migration

product/
├── entity/
│   └── recently_viewed.go                  # NEW: Entity definition
├── repository/
│   └── recently_viewed_repository.go       # NEW: Repository layer
├── service/
│   └── recently_viewed_service.go          # NEW: Service layer
├── handler/
│   └── product_handler.go                  # MODIFY: Inject RecentlyViewedService, add recording call
├── model/
│   └── product_model.go                    # MODIFY: Add SellerID uint `json:"-"`
├── factory/
│   ├── product_factory.go                  # MODIFY: Set SellerID in BuildProductResponse
│   └── singleton/
│       ├── repository_factory.go           # MODIFY: Register RecentlyViewedRepo
│       ├── service_factory.go              # MODIFY: Register RecentlyViewedService
│       ├── handler_factory.go              # MODIFY: Pass new service to handler constructor
│       └── singleton_factory.go            # MODIFY: Add getter methods
├── utils/
│   ├── error_constants.go                  # MODIFY: Add 2 error codes
│   └── message_constants.go                # MODIFY: Add 2 message constants
└── route/
    └── product_route.go                    # MODIFY (Phase 7 optional): Add GET /recently-viewed route

test/integration/
└── product/product/get_product_by_id/
    ├── recently_viewed_setup_test.go       # NEW: Suite setup + shared helpers
    ├── recently_viewed_happy_path_test.go  # NEW: Happy path tests (4)
    ├── recently_viewed_sad_path_test.go    # NEW: Sad path tests (6)
    └── recently_viewed_edge_case_test.go   # NEW: Edge case tests (9)
```

**Structure Decision**: This is a single-project web service (Option 1). All new code lives within the existing `product/` module following the standard module layout defined in the constitution. Integration tests follow the existing `test/integration/product/` directory convention.

## Complexity Tracking

> No constitution violations — this section is intentionally empty.
