# Quickstart: Product File Integration

## Goal

Implement Product media management so products can attach existing uploaded files, expose ordered media in product responses, update primary/order metadata, and remove linked media with best-effort File module cleanup.

## Prerequisites

- Go 1.25+
- Docker available for Testcontainers
- PostgreSQL 16 integration test container support
- Existing Product and File modules available in the application container

## Implementation Order

1. Add migration `019_create_product_media_table.sql`.
2. Add `product/entity/product_media.go`.
3. Add Product media request/response DTOs to `product/model`.
4. Add Product media repository with product-scoped CRUD and batch lookup methods.
5. Add Product File gateway interfaces in Product service layer for File read/delete operations.
6. Add Product media service methods for attach, update, remove, and response mapping.
7. Extend Product detail/list query responses to include ordered media summaries.
8. Add Product media routes under `/api/product/{productId}/media`.
9. Wire repositories/services/handlers through Product singleton factories.
10. Write integration tests before implementation and keep them green.

## Integration test approach (matches existing modules)

Follow the same pattern as `test/integration/product/product/create_product/create_product_test.go` and other Product/File integration tests:

1. **Containers and database**: `setup.SetupTestContainers(t)`, then `RunAllMigrations`, `RunAllCoreSeeds`, and any mock seeds needed (for example `migrations/seeds/mock/001_seed_users.sql`, `002_seed_products.sql`) so sellers and products exist for realistic calls.
2. **Application server**: `setup.SetupTestServer(t, containers.DB, containers.RedisClient)` returns an `http.Handler` — the full Gin app, not a stub.
3. **HTTP client**: `helpers.NewAPIClient(server)` issues `GET`/`POST`/`PATCH`/`DELETE` against that handler (`httptest.ResponseRecorder` + `ServeHTTP`). This mimics real API usage; default `X-Correlation-ID` is set on the client — override or omit only when testing middleware rejection.
4. **Auth and headers**: Use `helpers.Login` for JWTs, `client.SetToken`, and set `X-Seller-ID` via `client.SetHeader` for public catalog routes. Match the constants and emails used elsewhere in `test/integration/helpers`.
5. **Assertions**: Prefer `helpers.AssertSuccessResponse`, `helpers.GetResponseData`, and follow-up GETs to assert state through the API (API-first rule from project standards), rather than asserting by reaching into repositories from tests.
6. **Suite vs package tests**: Use either per-test `SetupTestContainers` (common in Product CRUD tests) or `testify/suite` with `SetupSuite` when setup is heavy (see `test/integration/file/setup_upload_suite_test.go` for MinIO/Rabbit-dependent flows). Product media tests can start with the lightweight per-file pattern unless file upload requires blob infra.
7. **No mocking policy**: Do not mock the Gin router, handlers, or GORM/DB for primary integration scenarios. File and Product modules run for real; cross-module calls happen through wired services. Use **unit tests** only for small pure helpers (for example DTO mapping edge cases) when an HTTP test adds no value.

## API Verification

Use the contract in `contracts/product-media.openapi.yaml` as the expected behavior.

### Attach media

```bash
curl -X POST "$BASE_URL/api/product/101/media" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -H "X-Correlation-ID: product-media-attach-1" \
  -H "Content-Type: application/json" \
  -d '{
    "fileId": "018e6b00-0000-7000-8000-000000000001",
    "isPrimary": true,
    "displayOrder": 0
  }'
```

Expected: `201 Created` with a Product media summary.

### Update media metadata

```bash
curl -X PATCH "$BASE_URL/api/product/101/media/018e6b00-0000-7000-8000-000000000001" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -H "X-Correlation-ID: product-media-update-1" \
  -H "Content-Type: application/json" \
  -d '{
    "isPrimary": true,
    "displayOrder": 2
  }'
```

Expected: `200 OK` with updated primary/order values.

### Read product with media

```bash
curl "$BASE_URL/api/product/101" \
  -H "X-Seller-ID: $SELLER_ID" \
  -H "X-Correlation-ID: product-media-read-1"
```

Expected: existing Product response plus additive `media` collection ordered by `displayOrder`.

### List products with media

```bash
curl "$BASE_URL/api/product?page=1&pageSize=20" \
  -H "X-Seller-ID: $SELLER_ID" \
  -H "X-Correlation-ID: product-media-list-1"
```

Expected: product list response includes media summaries without per-product follow-up calls.

### Remove media

```bash
curl -X DELETE "$BASE_URL/api/product/101/media/018e6b00-0000-7000-8000-000000000001" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -H "X-Correlation-ID: product-media-remove-1"
```

Expected: `204 No Content`. Product no longer includes the removed media.

## Integration Test Coverage

Required tests:

- Attach media happy path returns `201` and Product detail includes the media.
- Attach media rejects missing correlation ID.
- Attach media rejects unauthenticated or unauthorized caller.
- Attach media rejects invalid product ID, missing product, invalid file ID, inaccessible file, and duplicate link.
- Attach media with `isPrimary=true` clears existing primary media.
- Update media changes `displayOrder`.
- Update media with `isPrimary=true` clears existing primary media.
- Update media rejects missing link and invalid payload.
- Remove media deletes the Product media link and Product response no longer includes it.
- Remove primary media promotes the remaining lowest-order media item.
- Remove media returns success when File cleanup fails after unlink and logs cleanup failure.
- Product detail returns an empty media collection when no media exists.
- Product detail remains available when a referenced file is missing or inaccessible.
- Product list batches media resolution for all returned products.
- Seller A cannot attach, update, remove, or read Seller B private product media.

## Done Criteria

- All new integration tests pass (`go test ./test/integration/product/product_media/...`).
- Existing Product and File integration tests still pass.
- Every scenario in **Integration Test Coverage** above is covered; any gap filled with a **unit** test is documented in code comments and justified (narrow pure logic only).
- Product response changes are additive.
- No Product code imports File repositories or File entities for persistence.
- Product list media resolution is batched.
- Product media operations preserve seller isolation and correlation ID enforcement.
