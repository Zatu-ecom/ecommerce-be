# Data Model: Recently Viewed Products

**Feature**: 006-recently-viewed-products
**Date**: 2026-07-15

## Entity: RecentlyViewed

Represents a single product view event by a customer. Each `(user_id, product_id)` pair is unique — re-viewing the same product updates the `viewed_at` timestamp rather than creating a new row.

### Table: `user_recently_viewed`

| Column       | Type          | Constraints                                        | Description                                 |
| ------------ | ------------- | -------------------------------------------------- | ------------------------------------------- |
| `id`         | `BIGSERIAL`   | PRIMARY KEY                                        | Auto-incrementing row ID                    |
| `user_id`    | `BIGINT`      | NOT NULL                                           | Customer who viewed the product             |
| `seller_id`  | `BIGINT`      | NOT NULL                                           | Seller who owns the product at time of view |
| `product_id` | `BIGINT`      | NOT NULL, REFERENCES product(id) ON DELETE CASCADE | Product that was viewed                     |
| `viewed_at`  | `TIMESTAMPTZ` | NOT NULL, DEFAULT CURRENT_TIMESTAMP                | When the product was last viewed            |
| `created_at` | `TIMESTAMPTZ` | NOT NULL, DEFAULT CURRENT_TIMESTAMP                | Row creation timestamp                      |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL, DEFAULT CURRENT_TIMESTAMP                | Row last update timestamp                   |

### Constraints

| Constraint                        | Type        | Columns                 | Purpose                                                        |
| --------------------------------- | ----------- | ----------------------- | -------------------------------------------------------------- |
| `uq_user_recently_viewed_product` | UNIQUE      | `(user_id, product_id)` | One row per user-product pair; re-views update via ON CONFLICT |
| `pk_user_recently_viewed`         | PRIMARY KEY | `(id)`                  | Standard row identifier                                        |

### Indexes

| Index                             | Columns                     | Purpose                                             |
| --------------------------------- | --------------------------- | --------------------------------------------------- |
| `idx_recently_viewed_user_id`     | `(user_id)`                 | Fast count and lookup by user                       |
| `idx_recently_viewed_user_viewed` | `(user_id, viewed_at DESC)` | Retrieve most recent N entries; trim oldest entries |
| `idx_recently_viewed_product_id`  | `(product_id)`              | Cascade delete efficiency                           |

### Relationships

```
User ──< user_recently_viewed >── Product
         (user_id)        (product_id)
```

- **User to RecentlyViewed**: One-to-many (a user has up to 10 recently viewed records). No foreign key constraint on `user_id` to allow for eventual user module independence.
- **Seller to RecentlyViewed**: One-to-many. No foreign key constraint on `seller_id` — the seller ID is captured as a snapshot at the time of viewing and does not need referential integrity. If a seller is deleted, stale `seller_id` values in recently viewed records are harmless (they serve only as audit/metadata and are not used for join queries).
- **Product to RecentlyViewed**: One-to-many with CASCADE delete. When a product is deleted from the catalog, its associated recently viewed records are automatically removed.

### State Transitions

```
[NEW VIEW] ──upsert──> [RECORDED] ──re-view──> [RECORDED (updated timestamp)]
                            │
                            ├── < 10 entries: [ACTIVE]
                            │
                            └── ≥ 11 entries: [ACTIVE] + oldest [TRIMMED]
```

- **NEW VIEW**: A customer views a product they haven't viewed before → INSERT new row.
- **RE-VIEW**: A customer views a product they've already viewed → UPDATE `viewed_at` and `seller_id` via ON CONFLICT.
- **TRIMMED**: When the user exceeds 10 entries, the oldest entry (by `viewed_at ASC`) is deleted. Trimming is not reversible — the deleted record is permanently removed.

### Validation Rules

1. `user_id` must be a valid authenticated customer (role level 3). Enforced by handler-level role check.
2. `product_id` must reference an existing product. Enforced by foreign key constraint.
3. `seller_id` is captured from the product entity at the time of viewing. No validation beyond NOT NULL.
4. `viewed_at` defaults to `CURRENT_TIMESTAMP` on insert and is explicitly set on upsert.

### Migration

**File**: [`migrations/025_create_recently_viewed_table.sql`](../../migrations/025_create_recently_viewed_table.sql)

Migration includes:

- CREATE TABLE with all columns, constraints, and indexes
- Trigger for auto-updating `updated_at` on row modification (reuses existing `update_updated_at_column()` function)
