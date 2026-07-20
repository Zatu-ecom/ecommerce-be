# Research: Recently Viewed Products

**Feature**: 006-recently-viewed-products
**Date**: 2026-07-15
**Purpose**: Resolve all technical unknowns before Phase 1 design.

## Research Tasks

### 1. Where to inject the save logic?

**Decision**: Inject `RecentlyViewedService` into `ProductHandler` (not `ProductQueryService`).

**Rationale**:

- `ProductQueryService` is a pure query service — adding side-effect writes violates its contract and the constitution's Clean Architecture principle.
- The handler already orchestrates the flow: it extracts `userIDPtr` from context, calls the query service, and returns the response. Adding a side-effect recording call fits naturally in this orchestration.
- Wishlist status check already follows this pattern — the query service gets `userID *uint` as optional context for wishlist status, but wishlist mutations happen through a separate `WishlistHandler`.

**Alternatives considered**:

- **Inject into ProductQueryService**: Rejected — violates SRP, makes the query service impure, harder to test.
- **Middleware-based recording**: Rejected — middleware would need to duplicate query logic or parse the request body, breaking the middleware's responsibility boundary.

### 2. Synchronous or async save?

**Decision**: Synchronous inline call (no goroutine), with errors logged but not returned (fire-and-forget).

**Rationale**:

- Trimming to 10 entries must be synchronous to prevent race conditions where two concurrent inserts both count 10 and skip trimming, resulting in 11+ entries.
- The DB operation is a single-row upsert + conditional delete, which is fast (<5ms).
- Goroutine adds complexity (context cancellation, DB connection pooling from goroutine) with minimal benefit.
- The fire-and-forget pattern means errors are logged but not propagated — the product response is always returned successfully.

**Alternatives considered**:

- **Goroutine with channel-based trim**: Rejected — adds complexity, concurrency issues are harder to test.
- **Deferred trim via cron job**: Rejected — introduces delay where user could have >10 entries until the next cron run.

### 3. How to get `seller_id`?

**Decision**: Add a `SellerID uint` field (with `json:"-"`) to `model.ProductResponse`. The existing `BuildProductResponse` factory sets it from the product entity.

**Rationale**:

- The handler receives `productResponse` from `GetProductByID()`, which already contains all product data including `seller_id`.
- Adding an internal-only field avoids an extra DB query.
- `json:"-"` ensures it does not leak into the API response — consumers see no change.

**Alternatives considered**:

- **Query the product entity separately in the handler**: Rejected — extra DB query, wasteful.
- **Extract from ProductResponse data map**: Rejected — fragile, relies on map keys.

### 4. `viewed_at` vs relying on GORM's `UpdatedAt`?

**Decision**: Use a dedicated `viewed_at` column, updating it explicitly on upsert.

**Rationale**:

- `UpdatedAt` is managed by GORM and could change for reasons unrelated to "last viewed" (e.g., schema migrations, admin operations).
- A dedicated column provides semantic clarity.
- Required by the recommendation architecture plan which references `viewed_at` for the "Recently Viewed" recommendation strategy.

**Alternatives considered**:

- **Use UpdatedAt only**: Rejected — semantic ambiguity, GORM may mutate it on unrelated operations.

### 5. Role-based recording restriction — how?

**Decision**: Check `GetUserRoleLevelFromContext(c)` in the handler, only recording when `roleLevel == constants.CUSTOMER_ROLE_LEVEL` (3).

**Rationale**:

- The `PublicAPIAuth` middleware routes all authenticated roles (customer, seller, admin) through the same product endpoint.
- The role level is already extracted from JWT claims and stored in the Gin context by the auth middleware.
- Checking in the handler keeps the filtering explicit and testable.
- Sellers/admins browsing the storefront for testing/preview should not pollute customer-facing recently viewed data.

**Alternatives considered**:

- **Separate route with CustomerAuth middleware**: Rejected — product endpoints use `PublicAPIAuth` to allow unauthenticated browsing. Splitting routes would break the existing API design.
- **Check in service layer**: Rejected — role checks belong at the handler/authorization boundary, not in business logic.

### 6. Upsert strategy — INSERT vs SELECT-then-INSERT?

**Decision**: Use GORM's `clause.OnConflict` for a single-statement upsert.

**Rationale**:

- `ON CONFLICT (user_id, product_id) DO UPDATE SET viewed_at = NOW(), seller_id = EXCLUDED.seller_id` is atomic, fast, and standard PostgreSQL.
- Avoids the race condition inherent in SELECT-then-INSERT (two concurrent requests could both SELECT (find no row) and both INSERT, creating a unique violation).
- Fits GORM's pattern well via `clause.OnConflict{...}.Create()`.

**Alternatives considered**:

- **SELECT, then INSERT or UPDATE**: Rejected — race condition, two round trips.
- **Raw SQL**: Considered but rejected — GORM's `OnConflict` clause provides equivalent functionality with better readability.

### 7. Trimming strategy — DELETE after INSERT?

**Decision**: After upsert, count rows for the user. If count > 10, delete the oldest N rows (by `viewed_at ASC`).

**Rationale**:

- Simple, correct, and easy to verify.
- The count+delete is fast since `user_id` and `(user_id, viewed_at DESC)` are indexed.
- Occasional race between concurrent requests may briefly leave 11 entries, but the next view trims back to 10. This is acceptable for a fire-and-forget feature.

**Alternatives considered**:

- **DELETE before INSERT when at limit**: Rejected — risk of deleting the entry being re-viewed.
- **Window function to keep top 10**: Rejected — more complex SQL, minimal benefit.
- **Trigger-based trim**: Rejected — moves business logic to DB layer, harder to maintain.

## Technology Choices Confirmed

| Technology     | Version | Confirmed Usage                          |
| -------------- | ------- | ---------------------------------------- |
| Go             | 1.25+   | Required by constitution                 |
| Gin            | Latest  | HTTP framework, handler orchestration    |
| GORM           | Latest  | ORM for entity, upsert, count, delete    |
| PostgreSQL     | 16      | Storage for `user_recently_viewed` table |
| testify/suite  | Latest  | Integration test suite pattern           |
| Testcontainers | Latest  | Isolated test environment                |

## Risks & Mitigations

| Risk                                                              | Likelihood | Impact | Mitigation                                                                                                                                                                                                                                                                                                               |
| ----------------------------------------------------------------- | ---------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Concurrent upserts cause brief over-limit (11 entries)            | Medium     | Low    | Acceptable; next view trims. Fire-and-forget feature.                                                                                                                                                                                                                                                                    |
| Trim DELETE misses rows due to race                               | Low        | Low    | Delete uses subquery with LIMIT; any orphaned rows trimmed on next view.                                                                                                                                                                                                                                                 |
| Migration conflicts with existing migration numbers               | Low        | Medium | Use next available number (025). Check `migrations/` directory before creating.                                                                                                                                                                                                                                          |
| `SellerID` field in ProductResponse breaks existing serialization | None       | High   | `json:"-"` tag prevents serialization. Existing tests continue to pass.                                                                                                                                                                                                                                                  |
| Recording overhead exceeds <5ms target (SC-001)                   | Low        | Medium | Single-row upsert + conditional DELETE with indexes is inherently fast. Validated via code review of query plan (index usage on `user_id`, `(user_id, viewed_at DESC)`) rather than runtime benchmark. If profiling later reveals issues, addition of a Redis write-through cache or async goroutine is straightforward. |
