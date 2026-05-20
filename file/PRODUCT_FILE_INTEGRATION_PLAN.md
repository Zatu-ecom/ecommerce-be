# Product + File Service Integration Plan

> **Purpose**: Exact changes needed in current codebase to integrate `file` module with product, variant, order snapshots, and seller-facing media fields.  
> **Last Updated**: April 5, 2026  
> **Status**: Analysis complete, implementation pending

---

## 1) Current State (What exists today)

### Product/Variant

- `product_variant.images TEXT[]` stores raw URLs directly.
- `product` table has no file reference column (images are variant-driven).

References:
- [migrations/002_create_product_tables.sql:129](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/migrations/002_create_product_tables.sql:129)
- [product/entity/product_variant.go:12](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/product/entity/product_variant.go:12)
- [product/model/variant_model.go:98](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/product/model/variant_model.go:98)

### Query/Aggregation

- Product listing/detail aggregation reads `images` from default variant directly.

References:
- [product/query/variant_queries.go:11](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/product/query/variant_queries.go:11)
- [product/repository/variant_repository.go:525](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/product/repository/variant_repository.go:525)

### Order Snapshot

- Order item stores `image_url` snapshot as plain string.
- Snapshot is currently taken from first variant image URL.

References:
- [order/entity/order_item.go:17](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/order/entity/order_item.go:17)
- [order/factory/order_builder.go:53](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/order/factory/order_builder.go:53)
- [migrations/013_create_order_tables.sql:36](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/migrations/013_create_order_tables.sql:36)

### Other image fields

- `collection.image` (single URL string)
- `seller_profile.business_logo` (single URL string)

References:
- [product/entity/collection.go:24](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/product/entity/collection.go:24)
- [migrations/006_create_collection_tables.sql:22](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/migrations/006_create_collection_tables.sql:22)
- [user/entity/seller_profile.go:13](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/user/entity/seller_profile.go:13)
- [migrations/001_create_user_tables.sql:110](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/migrations/001_create_user_tables.sql:110)

---

## 2) Recommended Target State

Use **`file_object_id` references** in domain tables and resolve URLs at response time from file service.

Principles:
- DB stores stable IDs (`file_object.id`), not unstable URLs.
- API returns `url` (resolved/signed/public) + file metadata.
- Keep backward compatibility during migration.

---

## 3) DB Changes Required

## 3.1 Product Variant (mandatory)

### Option A (Recommended): separate media table

Create `product_variant_media`:

- `id`
- `variant_id` FK -> `product_variant`
- `file_object_id` FK -> `file_object`
- `sort_order`
- `is_primary`
- `alt_text`
- timestamps
- unique constraints:
  - `(variant_id, file_object_id)`
  - `(variant_id, is_primary)` partial unique where `is_primary=true`

Why this is best:
- Supports gallery cleanly.
- Avoids array FK limitations.
- Easy ordering and metadata.

### Option B (faster but limited)

Add on `product_variant`:
- `primary_image_file_id BIGINT NULL`
- keep existing `images TEXT[]` temporarily

Good for quick rollout, but less scalable for multiple images/metadata.

## 3.2 Product-level primary image (optional)

Add to `product`:
- `primary_image_file_id BIGINT NULL`

Use for listing performance and fallback when variants are absent.

## 3.3 Order snapshot (mandatory)

Keep `image_url` for immutable order snapshots, but source it from resolved file URL at checkout time.

Optional additive column:
- `image_file_id BIGINT NULL` in `order_item`

This preserves historical audit while still linking to origin file.

## 3.4 Collection and Seller Profile (high priority)

Add:
- `collection.image_file_id BIGINT NULL`
- `seller_profile.business_logo_file_id BIGINT NULL`

Keep old text columns during transition.

---

## 4) API Contract Changes

## Variant create/update request

Current:
- `images: []string`

Target:
- `mediaFileIds: []uint` (ordered)
- or `primaryImageFileId: uint` + `galleryFileIds: []uint`

## Variant response

Add:
- `imageFileIds: []uint`
- `images: []string` (resolved URLs, backward compatible)

Migration strategy:
- Keep `images` response for existing clients.
- Internally resolve from `file_object_id`.

---

## 5) Image Variant + Video Workflow

This is the recommended end-to-end flow for supporting multiple images/videos per variant plus derived file variants.

## 5.1 Upload and Link Flow (Write path)

1. Seller uploads original media files using file module:
- `POST /api/files/init-upload`
- direct upload to blob
- `POST /api/files/complete-upload`
- each upload creates one `file_object` row

2. Seller saves variant with ordered media:
- request includes `mediaFileIds` in order
- backend inserts rows in `product_variant_media` with:
  - `variant_id`
  - `file_object_id`
  - `media_type` (`IMAGE` or `VIDEO`)
  - `sort_order`
  - `is_primary` (first media or explicitly set)

3. On link/save, backend enqueues processing jobs:
- For `IMAGE` originals:
  - generate `THUMBNAIL_SM` (listing card)
  - generate `THUMBNAIL_MD` (PDP gallery)
  - generate `WEBP` (optimized web delivery)
  - keep `ORIGINAL` for zoom/download
- For `VIDEO` originals:
  - generate `POSTER_IMAGE` (preview thumbnail)
  - optional transcode variants (`MP4_720P`, `MP4_1080P`)

4. File worker writes results to `file_variant` table and marks status.

## 5.2 Data Model Notes

Add fields in `product_variant_media`:
- `media_type VARCHAR(20) NOT NULL` (`IMAGE`, `VIDEO`)
- `role VARCHAR(30)` optional (`GALLERY`, `SIZE_CHART`, `SWATCH_VIDEO`)
- `video_poster_file_id BIGINT NULL` (optional direct pointer)

`file_variant` remains the canonical place for processed outputs per `file_object`.

## 5.3 Read Flow (Storefront resolution)

1. Product/variant query fetches media rows ordered by `sort_order`.
2. For each media row, call `FileQueryService.ResolveBestVariant(...)`:
- listing page image: prefer `THUMBNAIL_SM`, fallback `THUMBNAIL_MD`, then original
- PDP gallery image: prefer `WEBP`, fallback original
- zoom view: original
- video tile: poster (`POSTER_IMAGE`) + stream URL (`MP4_720P`/original)
3. API returns both stable IDs and resolved URLs:
- `fileObjectId`
- `mediaType`
- `url`
- `posterUrl` (video)
- `variants` (optional map for client optimizations)

## 5.4 Fallback Rules

1. If processed variant is not ready, return original URL.
2. If media row missing for current variant, fallback to default variant media.
3. If no media at all, return placeholder URL configured by frontend.
4. Never fail product API just because one derived variant generation failed.

## 5.5 Order Snapshot Rule

At checkout, for each order line:
1. pick variant primary media
2. resolve display URL (prefer processed image variant)
3. store snapshot into `order_item.image_url`
4. optionally store `order_item.image_file_id`

This keeps historical order data immutable even if media is changed later.

---

## 6) Repository/Service Changes Required

## Must update

1. Variant entity/model/factory:
- [product/entity/product_variant.go](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/product/entity/product_variant.go)
- [product/model/variant_model.go](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/product/model/variant_model.go)
- [product/factory/variant_factory.go](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/product/factory/variant_factory.go)

2. Variant aggregation queries:
- [product/query/variant_queries.go](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/product/query/variant_queries.go)
- [product/repository/variant_repository.go](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/product/repository/variant_repository.go)

3. Product response builder:
- [product/factory/product_factory.go](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/product/factory/product_factory.go)

4. Order snapshot image source:
- [order/factory/order_builder.go](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/order/factory/order_builder.go)

5. Collection/Seller profile DTO + entity:
- [product/entity/collection.go](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/product/entity/collection.go)
- [user/entity/seller_profile.go](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/user/entity/seller_profile.go)
- [user/model/seller_profile_model.go](/home/kushal/Work/Personal%20Codes/Ecommerce/ecommerce-be/user/model/seller_profile_model.go)

## New dependency

Introduce `FileQueryService` (read-side):
- `ResolveFileURL(fileID, purpose, variantType, tenantCtx) -> string`
- `ResolveFileURLs([]fileID, ...) -> map[fileID]url`

Use batch resolve for product list APIs to avoid N+1.

---

## 7) Backward-Compatible Rollout Plan

1. Add new file reference columns/tables (no breaking changes).
2. Write path:
- New uploads populate file references.
- Continue writing legacy URL fields for short transition.
3. Read path:
- Prefer file references.
- Fallback to legacy URL fields when references absent.
4. Backfill:
- Migrate old URL arrays to file objects (script/job).
5. Remove legacy URL columns only after all clients migrated.

---

## 8) Migration SQL Skeleton (example)

```sql
-- 1) Variant media table
CREATE TABLE IF NOT EXISTS product_variant_media (
    id BIGSERIAL PRIMARY KEY,
    variant_id BIGINT NOT NULL REFERENCES product_variant(id) ON DELETE CASCADE,
    file_object_id BIGINT NOT NULL REFERENCES file_object(id) ON DELETE RESTRICT,
    sort_order INT NOT NULL DEFAULT 0,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    alt_text VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (variant_id, file_object_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_variant_primary_image
ON product_variant_media(variant_id) WHERE is_primary = true;

CREATE INDEX IF NOT EXISTS idx_variant_media_variant_id ON product_variant_media(variant_id);
CREATE INDEX IF NOT EXISTS idx_variant_media_file_object_id ON product_variant_media(file_object_id);

-- 2) Product optional primary image
ALTER TABLE product
ADD COLUMN IF NOT EXISTS primary_image_file_id BIGINT REFERENCES file_object(id) ON DELETE SET NULL;

-- 3) Collection and seller profile file references
ALTER TABLE collection
ADD COLUMN IF NOT EXISTS image_file_id BIGINT REFERENCES file_object(id) ON DELETE SET NULL;

ALTER TABLE seller_profile
ADD COLUMN IF NOT EXISTS business_logo_file_id BIGINT REFERENCES file_object(id) ON DELETE SET NULL;

-- 4) Order snapshot linkage (optional additive)
ALTER TABLE order_item
ADD COLUMN IF NOT EXISTS image_file_id BIGINT REFERENCES file_object(id) ON DELETE SET NULL;
```

---

## 9) Final Answer to "What changes are needed?"

Minimum required for file service integration in your current product flow:

1. Replace variant `images TEXT[]` write-path with `file_object_id` based media mapping.
2. Update variant/product query aggregation to resolve URLs from file module.
3. Keep order `image_url` snapshot, but generate it from file service at order creation.
4. Add file reference columns for collection image and seller business logo.
5. Run dual-read/dual-write migration until old clients and old data are fully moved.
