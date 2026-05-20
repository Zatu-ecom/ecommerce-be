# Research: Product File Integration

## Decision: Product owns media links; File owns file records and blobs

**Rationale**: Product-specific concerns such as primary media and display order belong to the Product module. File-specific concerns such as storage provider, object keys, variants, download URLs, and blob deletion remain in the File module. This keeps the modular monolith boundary intact and preserves future service extraction.

**Alternatives considered**:

- Add hard foreign keys from Product to File tables. Rejected because it couples module schemas and violates the constitution's module boundary rule.
- Store full file metadata inside Product. Rejected because it duplicates File module state and risks stale data.

## Decision: Use a Product Media association table

**Rationale**: A dedicated `product_media` association supports multiple media items per product, duplicate prevention, product-specific ordering, and primary selection. It also supports images, videos, documents, and future file-backed media without introducing separate tables for each type.

**Alternatives considered**:

- Store media IDs as an array on `product`. Rejected because ordering, uniqueness, primary selection, and updates are harder to validate and query.
- Add image-only fields to Product. Rejected because the feature explicitly includes videos and future media types.

## Decision: Return hybrid product media summaries

**Rationale**: Product responses should expose fields consumers need to render and manage product media: `fileId`, `url`, `thumbnailUrl`, `isPrimary`, and `displayOrder`. This avoids leaking generic storage metadata while preserving enough information for storefront and admin workflows.

**Alternatives considered**:

- Return only URLs. Rejected because admins need stable media identifiers for reorder, update, and removal.
- Embed raw File module response objects. Rejected because it exposes unrelated storage metadata and makes Product response contracts depend too tightly on File internals.

## Decision: Batch media resolution for product list reads

**Rationale**: Product listing responses can include up to 20 products by default. Resolving files product-by-product would create N+1 behavior. Product list reads should collect all product IDs, fetch all media links for the page, collect unique file IDs, and call File read once for the full set.

**Alternatives considered**:

- Resolve media independently per product. Rejected because it violates performance guidance and scales poorly.
- Omit media from product lists. Rejected because the spec requires product cards to show media without additional user-visible retrieval.

## Decision: Delete underlying file on product-media removal, best effort after unlink

**Rationale**: The prespec chooses delete-on-detach. Product correctness should take priority: once the Product media link is removed, the product no longer references stale media. If File deletion fails, the API still reports product media removal success and logs the cleanup failure for operational follow-up.

**Alternatives considered**:

- Leave detached files for background cleanup only. Rejected because it delays intended asset cleanup and leaves more orphaned files.
- Fail the removal if File deletion fails. Rejected because it can leave product pages showing incorrect media due to external storage problems.

## Decision: Product media management routes live under Product

**Rationale**: Attaching, ordering, primary selection, and removal are product-management actions. Routes should be grouped under the existing Product base path and protected with seller/admin middleware, while product detail/list reads remain public with seller context.

**Alternatives considered**:

- Add Product-specific routes under File. Rejected because Product owns the relationship and business rules.
- Reuse generic Product update for media mutation. Rejected because media operations have distinct validation, conflict, and cleanup behavior.

## Decision: Integration tests are the primary verification strategy

**Rationale**: The feature crosses HTTP handlers, middleware, Product persistence, Product services, and File service integration. Integration tests are required by constitution and provide the highest confidence for tenant isolation, correlation ID enforcement, and response contracts.

**Alternatives considered**:

- Unit-only service tests. Rejected because they would miss middleware, route, response, transaction, and database behavior.

## Alignment with existing repository integration tests

**Decision**: Product media tests follow the same harness as existing Product and File integration tests.

**Concrete pattern** (already used across the repo):

- `test/integration/setup` for Testcontainers (PostgreSQL, Redis), migrations, seeds, and `SetupTestServer` to obtain the real application `http.Handler`.
- `test/integration/helpers.APIClient` to perform HTTP requests (`Get`, `Post`, `Patch`, `Delete`, etc.) against that handler so each test mimics an actual API call without starting a separate network listener.
- Seller/customer login via `helpers.Login` and JWT + `X-Seller-ID` headers consistent with other modules.
- Assertions through standardized response helpers and follow-up API reads where appropriate, not by bypassing the API layer for primary state checks.

**Mocking**: The integration test does not mock the HTTP stack or database. Optional **unit tests** supplement coverage for pure logic (for example mapping `FileItem` fields to product media DTOs) when an HTTP-level test would be impractical.
