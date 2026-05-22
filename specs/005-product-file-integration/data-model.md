# Data Model: Product File Integration

> **Last updated**: 2026-05-21 — Variant Media architecture added (US4). The `images TEXT[]` column previously on `product_variant` has been removed and replaced with the `variant_media` join table.

---

## Product

Represents an existing catalog product.

**Existing fields relevant to this feature**:

- `id`: Product identifier.
- `seller_id`: Seller tenant owner.
- `name`, `category_id`, `brand`, descriptions, tags: Existing product catalog fields.

**Relationships**:

- Product has zero or more Product Media rows.
- Product has zero or more Product Variants, each with their own Variant Media rows.
- Product Media rows are deleted when the Product is deleted.

**Validation rules**:

- Product must exist before media can be attached.
- Product operations must remain seller-scoped.

---

## Product Variant

Represents a purchasable SKU defined by a combination of option values (e.g., Red + Size M).

**Fields relevant to this feature**:

- `id`: Variant identifier.
- `product_id`: Parent product.
- `sku`, `price`, `allow_purchase`, `is_popular`, `is_default`: Existing variant fields.

> **Removed field**: `images TEXT[]` — raw URL storage was removed by migration `020_create_variant_media_table.sql`. All variant visual assets are now managed through the File module via the `variant_media` join table.

**Relationships**:

- Belongs to Product.
- Has zero or more Variant Media rows.
- Variant Media rows are owned by the variant; cascading behavior follows the same pattern as Product Media.

**Validation rules**:

- Variant must exist and belong to the owning seller before media can be attached.
- Variant operations must remain seller-scoped.

## Product Media

Represents a Product-owned link to an uploaded File module asset.

**Fields**:

- `id`: Internal association identifier.
- `product_id`: Product identifier.
- `file_id`: Stable File module UUIDv7 identifier. No hard database foreign key to File tables.
- `is_primary`: Whether this media item is the Product's primary media.
- `display_order`: Sort value for Product media presentation.
- `created_at`: Association creation timestamp.
- `updated_at`: Association update timestamp.

**Relationships**:

- Belongs to Product.
- References an Uploaded Media File by `file_id` through the File service boundary.

**Validation rules**:

- `product_id` is required.
- `file_id` is required and must be a valid File module identifier.
- A Product cannot link the same `file_id` more than once.
- A Product can have at most one primary media item.
- `display_order` defaults to `0` when not provided.
- When `is_primary` is set to true, all other Product Media rows for that Product must be set to false.

**Indexes and constraints**:

- Index on `product_id` for product detail/list media lookups.
- Unique constraint on `(product_id, file_id)` for duplicate prevention.
- Product foreign key with cascade delete is allowed because Product owns Product Media.
- No database foreign key to File module tables.

## Uploaded Media File

Represents an existing File module asset that can be attached to a Product.

**Fields consumed by Product**:

- `fileId`: Stable media identifier.
- `status`: Must be accessible/active for normal rendering.
- `downloadUrl`: Main display URL when requested.
- `variants`: Optional generated variants used to choose thumbnails or previews.

**Relationships**:

- File module owns the file record and blob lifecycle.
- Product references files only by `fileId`.

**Validation rules**:

- File must exist and be accessible to the caller before it can be attached.
- Product must request download URLs and variants when building Product media summaries.

## Product Media Summary

The response shape embedded in Product API responses.

**Fields**:

- `fileId`: File identifier used for future management actions.
- `url`: Main display URL for full-size media.
- `thumbnailUrl`: Thumbnail/poster/preview URL when available. If unavailable, may fall back to `url`.
- `isPrimary`: Product-specific primary flag.
- `displayOrder`: Product-specific order.

**Relationships**:

- Built from Product Media plus File module metadata.
- Included in Product detail and Product list responses.

**Validation rules**:

- Returned media must be sorted by `displayOrder` ascending.
- Missing or inaccessible File data must not prevent Product responses from loading.
- Returned media summaries must not expose storage-provider internals.

---

## Variant Media

Represents a Variant-owned link to an uploaded File module asset. Mirrors the Product Media model but is scoped to a specific variant.

**Fields**:

- `id`: Internal association identifier.
- `variant_id`: Variant identifier.
- `file_id`: Stable File module UUIDv7 identifier. No hard database foreign key to File tables.
- `is_primary`: Whether this media item is the Variant's primary media.
- `display_order`: Sort value for Variant media presentation.
- `created_at`: Association creation timestamp.
- `updated_at`: Association update timestamp.

**Relationships**:

- Belongs to Product Variant.
- References an Uploaded Media File by `file_id` through the File service boundary.

**Validation rules**:

- `variant_id` is required.
- `file_id` is required and must be a valid File module identifier.
- A Variant cannot link the same `file_id` more than once.
- A Variant can have at most one primary media item.
- `display_order` defaults to `0` when not provided.
- When `is_primary` is set to true, all other Variant Media rows for that Variant must be set to false.

**Indexes and constraints**:

- Index on `variant_id` for variant detail media lookups.
- Unique constraint on `(variant_id, file_id)` for duplicate prevention.
- Variant foreign key with cascade delete.
- No database foreign key to File module tables.

---

## Variant Media Summary

The response shape embedded in Variant API responses (`VariantDetailResponse` and `VariantResponse`).

**Fields**:

- `fileId`: File identifier used for future management actions.
- `url`: Main display URL for full-size media.
- `thumbnailUrl`: Thumbnail/poster/preview URL when available.
- `isPrimary`: Variant-specific primary flag.
- `displayOrder`: Variant-specific order.

**Validation rules**:

- Returned media must be sorted by `displayOrder` ascending.
- Missing or inaccessible File data must not prevent Variant responses from loading.
- The `media` field is always a JSON array (`[]` when empty, never `null`).

---

## When to Use Product Media vs Variant Media

| Product type | Visual differences between variants? | Use |
|---|---|---|
| Running shoes (color variants) | Yes — each color looks different | `variant_media` |
| Laptop (RAM variants) | No — 16GB and 64GB look identical | `product_media` |
| T-shirt (color + size) | Yes for color, no for size | `variant_media` on color variants |

---

## State Transitions

### Attach Media

1. Validate Product exists and caller can manage it.
2. Validate File exists and is accessible to caller.
3. If requested as primary, unset other primary media for the Product.
4. Create Product Media row.
5. Return Product Media Summary.

### Update Media Metadata

1. Validate Product Media link exists for the Product.
2. If setting primary, unset other primary media for the Product.
3. Update `is_primary` and/or `display_order`.
4. Return updated Product Media Summary.

### Remove Media (Product or Variant)

1. Validate the media link exists for the Product / Variant.
2. Delete the media row.
3. If removed row was primary and other media remains, promote the lowest `display_order` row to primary.
4. Attempt File module deletion for the underlying file.
5. If File deletion fails, log cleanup failure and keep the media removal successful.

---

## Variant Media State Transitions

### Attach Variant Media

1. Validate Product exists and caller can manage it (seller scope).
2. Validate Variant exists and belongs to the Product.
3. Validate File exists and is accessible to caller.
4. If requested as primary, unset other primary media for the Variant.
5. Create Variant Media row.
6. Return Variant Media Summary.

### Update Variant Media Metadata

1. Validate Variant Media link exists for the Variant.
2. Validate caller owns the parent Product.
3. If setting primary, unset other primary media for the Variant.
4. Update `is_primary` and/or `display_order`.
5. Return updated Variant Media Summary.

### Remove Variant Media

1. Validate Variant Media link exists for the Variant.
2. Validate caller owns the parent Product.
3. Delete Variant Media row.
4. If removed row was primary and other media remains, promote the lowest `display_order` row to primary.
5. Attempt File module deletion for the underlying file.
6. If File deletion fails, log cleanup failure and keep Variant media removal successful.
