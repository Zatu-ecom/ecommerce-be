# Quickstart: Recently Viewed Products

**Feature**: 006-recently-viewed-products
**Date**: 2026-07-15

## Overview

The Recently Viewed Products feature automatically tracks which products each customer views. When a customer fetches product details via `GET /api/product/:productId`, the system silently records the `(user_id, product_id, seller_id)` tuple. Only the last 10 recently viewed products are kept per user.

## Prerequisites

- Go 1.25+ installed
- Docker & Docker Compose running (for Testcontainers)
- PostgreSQL 16 database accessible
- Existing project dependencies installed (`go mod download`)

## How It Works

### Recording Flow

```
Customer fetches GET /api/product/123
  → PublicAPIAuth middleware extracts user_id from JWT
  → ProductHandler.GetProductByID queries product data
  → If user is a CUSTOMER (role level 3):
      → RecentlyViewedService.RecordRecentlyViewed is called
        → Upsert (INSERT or UPDATE viewed_at) for (user_id, product_id)
        → If user has > 10 entries, delete oldest
  → Product response returned (HTTP 200)
```

### Key Behaviors

| Scenario                              | Behavior                                  |
| ------------------------------------- | ----------------------------------------- |
| Customer views product for first time | New row inserted                          |
| Customer re-views same product        | `viewed_at` updated, no duplicate         |
| Customer exceeds 10 entries           | Oldest entry deleted                      |
| Seller/admin views product            | NOT recorded                              |
| Unauthenticated user views product    | NOT recorded                              |
| Recording fails (DB error, etc.)      | Error logged; product response unaffected |
| Product deleted from catalog          | Associated records cascade-deleted        |

## Database

### Migration

```bash
# Apply the migration
make migrate
```

The migration creates:

- [`user_recently_viewed`](data-model.md) table with UNIQUE(user_id, product_id)
- Indexes on `user_id`, `(user_id, viewed_at DESC)`, `product_id`
- CASCADE delete on product foreign key

## Running Tests

```bash
# Run all recently-viewed integration tests
go test ./test/integration/product/product/get_product_by_id/ -v

# Run specific test suites
go test ./test/integration/product/product/get_product_by_id/ -run TestRecentlyViewedHappyPath -v
go test ./test/integration/product/product/get_product_by_id/ -run TestRecentlyViewedSadPath -v
go test ./test/integration/product/product/get_product_by_id/ -run TestRecentlyViewedEdgeCase -v
```

### Test Coverage

| File                                 | Tests | Category                                   |
| ------------------------------------ | ----- | ------------------------------------------ |
| `recently_viewed_happy_path_test.go` | 4     | Core recording works correctly             |
| `recently_viewed_sad_path_test.go`   | 5     | Error resilience, edge inputs              |
| `recently_viewed_edge_case_test.go`  | 8     | Limits, isolation, roles, special products |

## Files Summary

### New Files (9)

| File                                                      | Purpose                       |
| --------------------------------------------------------- | ----------------------------- |
| `migrations/025_create_recently_viewed_table.sql`         | Database migration            |
| `product/entity/recently_viewed.go`                       | Entity definition             |
| `product/repository/recently_viewed_repository.go`        | Repository (interface + impl) |
| `product/service/recently_viewed_service.go`              | Service (interface + impl)    |
| `test/integration/data/recently_viewed_seed_data.sql`     | Test seed placeholder         |
| `test/integration/.../recently_viewed_setup_test.go`      | Suite setup + helpers         |
| `test/integration/.../recently_viewed_happy_path_test.go` | Happy path tests              |
| `test/integration/.../recently_viewed_sad_path_test.go`   | Sad path tests                |
| `test/integration/.../recently_viewed_edge_case_test.go`  | Edge case tests               |

### Modified Files (8)

| File                                              | Change                             |
| ------------------------------------------------- | ---------------------------------- |
| `product/utils/error_constants.go`                | Add 2 error codes                  |
| `product/utils/message_constants.go`              | Add 2 message constants            |
| `product/model/product_model.go`                  | Add `SellerID uint` field          |
| `product/handler/product_handler.go`              | Inject service, add recording call |
| `product/factory/singleton/repository_factory.go` | Register repo                      |
| `product/factory/singleton/service_factory.go`    | Register service                   |
| `product/factory/singleton/handler_factory.go`    | Pass service to handler            |
| `product/factory/singleton/singleton_factory.go`  | Add getters                        |

## API Contract

See [`contracts/api-contracts.md`](contracts/api-contracts.md) for detailed behavior specification of the recording trigger and optional retrieval endpoint.
