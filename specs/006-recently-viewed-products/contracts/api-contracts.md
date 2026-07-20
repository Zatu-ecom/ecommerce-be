# API Contracts: Recently Viewed Products

**Feature**: 006-recently-viewed-products
**Date**: 2026-07-15

## Contract 1: Automatic Recording on Product View

### Trigger

**Existing endpoint**: `GET /api/product/:productId`
**Middleware**: `PublicAPIAuth` (allows all authenticated roles + public)
**No API contract change** — the recording is a silent side effect.

### Behavior Contract

| Input Condition                       | Recording Behavior                                       | Product Response                   |
| ------------------------------------- | -------------------------------------------------------- | ---------------------------------- |
| Authenticated customer (role level 3) | Record is upserted                                       | Unchanged (HTTP 200, product data) |
| Authenticated seller (role level 2)   | NOT recorded                                             | Unchanged (HTTP 200, product data) |
| Authenticated admin (role level 1)    | NOT recorded                                             | Unchanged (HTTP 200, product data) |
| Unauthenticated (no token)            | NOT recorded                                             | Unchanged (HTTP 200, product data) |
| Product not found                     | NOT recorded (handler returns 404 before recording code) | HTTP 404                           |
| Invalid product ID (non-numeric)      | NOT recorded (handler returns 400 before recording code) | HTTP 400                           |
| Recording fails (DB error)            | Error logged only                                        | Unchanged (HTTP 200, product data) |

### Guarantees

1. **Fire-and-forget**: Recording failures NEVER affect the product response. HTTP status, response body, and response time are unaffected by recording failures.
2. **No duplicate rows**: Re-viewing the same product updates the `viewed_at` timestamp. Exactly one row per `(user_id, product_id)` pair.
3. **Max 10 entries**: After any view, the user has at most 10 recently viewed records. Trimming is best-effort (may briefly have 11 under concurrent writes, stabilized on next view).
4. **Role isolation**: Only customers (role level 3) trigger recording. Sellers and admins do not.

### Non-Guarantees

1. **Exactly-once under concurrency**: Two simultaneous views may both insert before either trims. The user may briefly have 11 entries.
2. **Recording latency**: While target is <5ms, no hard SLA on recording completion time.
3. **Recording durability**: If the DB is unreachable, the recording is silently dropped. No retry mechanism.

---

## Contract 2: Retrieve Recently Viewed Products (Phase 7 — Optional)

### Endpoint

```
GET /api/product/recently-viewed?limit={N}
```

**Authentication**: Required — uses `CustomerAuth` middleware (not `PublicAPIAuth`).
**Authorization**: Customer role only.

### Request

| Parameter | Type        | Required | Default | Max | Description                  |
| --------- | ----------- | -------- | ------- | --- | ---------------------------- |
| `limit`   | query (int) | No       | 10      | 50  | Number of products to return |

### Response

**200 OK** — Success:

```json
{
  "success": true,
  "message": "Recently viewed products retrieved",
  "data": {
    "productIds": [15, 7, 3, 22, 8]
  }
}
```

**401 Unauthorized** — Missing/invalid token:

```json
{
  "success": false,
  "message": "Authentication required",
  "code": "UNAUTHORIZED"
}
```

**403 Forbidden** — Wrong role (non-customer):

```json
{
  "success": false,
  "message": "Access denied",
  "code": "FORBIDDEN"
}
```

### Behavior

| Condition                                  | Response                                          |
| ------------------------------------------ | ------------------------------------------------- |
| Customer with 5 recently viewed, limit=10  | `productIds` array with 5 IDs, newest first       |
| Customer with 15 recently viewed, limit=10 | `productIds` array with 10 IDs (oldest 5 trimmed) |
| Customer with 0 recently viewed            | `productIds` array is empty `[]`                  |
| limit=5                                    | Returns at most 5 IDs                             |
| limit=100                                  | Capped at 50                                      |
| limit not provided                         | Defaults to 10                                    |
| Unauthenticated                            | 401 Unauthorized                                  |
| Seller or admin                            | 403 Forbidden (middleware-enforced)               |

### Ordering

Product IDs are returned in reverse chronological order — **most recently viewed first**. The ordering is based on `viewed_at DESC`.

---

## Contract 3: Database Schema Contract

### Table: `user_recently_viewed`

This table is the source of truth for recently viewed data. External systems (recommendation engine, analytics) may read from this table but MUST NOT write to it directly.

| Guarantee  | Details                                                                      |
| ---------- | ---------------------------------------------------------------------------- |
| Uniqueness | One row per `(user_id, product_id)`                                          |
| Retention  | Max 10 rows per `user_id` (enforced by application logic, not DB constraint) |
| Cascade    | Rows are deleted when the referenced product is deleted                      |
| Timestamps | `viewed_at` reflects the most recent view time for that product              |
