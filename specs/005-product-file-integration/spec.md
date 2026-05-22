# Feature Specification: Product File Integration

**Feature Branch**: `[005-product-file-integration]`  
**Created**: 2026-05-19  
**Last Updated**: 2026-05-21  
**Status**: Complete  
**Input**: User description: "Create the Product/File Integration specification from the prespec files in `specs/005-product-file-integration`, with the spec in the same folder."

> **Architectural note (added post-initial-spec):** During implementation it was confirmed that variant images must also flow through the File module rather than being stored as raw URLs on the `product_variant` table. A fourth user story (US4) was added to capture this requirement. The `images TEXT[]` column on `product_variant` was removed via migration `020_create_variant_media_table.sql` and replaced with a `variant_media` join table that mirrors the `product_media` pattern.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View Product Media (Priority: P1)

Storefront shoppers and admin users need product detail and listing views to include the product's associated media, such as primary images, gallery images, videos, and thumbnails, so products can be evaluated without extra lookups or missing visual context.

**Why this priority**: Product media is central to the shopping experience and is the primary user-facing value of the integration.

**Independent Test**: Can be tested with product-media records already present in the system (seeded or created by a prior step) by viewing the product detail and product list experiences and confirming each product displays the correct ordered media with usable display URLs and thumbnails where available. Attaching new media is User Story 2 scope; this story can be validated independently once media associations exist.

**Acceptance Scenarios**:

1. **Given** a product has multiple active media items, **When** a user views the product detail, **Then** the product includes all associated media in display order with the primary item identifiable.
2. **Given** a product has media with thumbnail or preview variants, **When** a user views a product listing, **Then** the listing shows the thumbnail or preview URL (preferring a dedicated thumbnail or poster variant over the full-size asset; falling back to the main display URL when no such variant exists) without needing to load the full-size asset.
3. **Given** a product has no attached media, **When** a user views the product detail or listing, **Then** the product still loads successfully with an empty media collection.

---

### User Story 2 - Manage Product Media Links (Priority: P2)

Admin users need to attach already uploaded files to products, choose the primary media item, and control display order so product pages present media in the intended merchandising sequence.

**Why this priority**: Product teams need management controls before media can be kept accurate and useful over time.

**Independent Test**: Can be tested by linking an uploaded file to a product, updating primary status and order, and confirming the product views immediately reflect those changes.

**Acceptance Scenarios**:

1. **Given** an admin has an existing uploaded media file, **When** they attach it to a product with a display order, **Then** the product includes that media item in the requested position.
2. **Given** an admin marks a media item as primary, **When** the change is saved, **Then** that item becomes the only primary media item for the product.
3. **Given** an admin attempts to attach the same media file to the same product more than once, **When** the duplicate attachment is submitted, **Then** the system rejects the duplicate and leaves the existing link unchanged.

---

### User Story 3 - Remove Product Media (Priority: P3)

Admin users need to remove media from a product when it is outdated, incorrect, or no longer relevant, so customers do not see stale product imagery or documents.

**Why this priority**: Removal is required for product data hygiene, but it depends on the core ability to display and manage media links.

**Independent Test**: Can be tested by removing an attached media item and confirming the product no longer references it while the removal completes successfully for the admin.

**Acceptance Scenarios**:

1. **Given** a product has an attached media item, **When** an admin removes that media item, **Then** the media item no longer appears on product detail or listing views.
2. **Given** the removed media item was the product's primary media item and other media remains, **When** the removal completes, **Then** the remaining item with the earliest display order becomes primary.
3. **Given** cleanup of the removed underlying asset cannot be completed immediately, **When** the admin removes the media link, **Then** the product state remains correct and the cleanup problem is recorded for follow-up.

---

### User Story 4 - Manage Variant Media (Priority: P2)

Sellers need to attach already uploaded files to specific product variants so that variant-specific images (e.g., a red shoe, a black phone) are managed through the File module rather than stored as raw URLs, giving variants the same file lifecycle guarantees as products.

**Why this priority**: Variants are the purchasable unit in this system. Color/finish/style variants require their own images to show customers exactly what they are buying. Products where variants differ only on non-visual specs (RAM, storage, CPU) are served by product-level media instead.

**Independent Test**: Can be tested by attaching an uploaded file to a variant, reading back the variant detail and confirming the `media` field is a JSON array with the expected item, and verifying other variants on the same product are unaffected.

**Acceptance Scenarios**:

1. **Given** a seller uploads a file and attaches it to a variant, **When** the variant is retrieved, **Then** the `media` field contains the attached file with correct `fileId`, `url`, `isPrimary`, and `displayOrder`.
2. **Given** a variant's primary media item is removed and at least one other media item remains, **When** the removal completes, **Then** the remaining item with the lowest `displayOrder` is automatically promoted to primary.
3. **Given** a variant has no attached media, **When** the variant is retrieved, **Then** the `media` field is an empty JSON array (`[]`), never `null`.
4. **Given** a referenced file is missing or inaccessible, **When** the variant is retrieved, **Then** the variant still loads successfully and that item is silently omitted from the `media` array.

---

### Edge Cases

- Products and variants with no media must remain readable and return an empty media collection (`[]`, never `null`).
- Missing, inactive, or inaccessible media files must not prevent the product or variant from loading.
- Product lists containing many products with media must avoid repeated per-product media lookups from the user's perspective by returning complete pages efficiently.
- Duplicate media attachments to the same product or the same variant must be rejected.
- Only one media item may be primary for a product at a time; only one media item may be primary for a variant at a time.
- Removing a non-existent product-media or variant-media link must return a clear not-found outcome.
- Media display order ties must produce a stable and predictable order.
- Variant media operations must be scoped to the owning seller; a different seller must receive 404.
- The `images TEXT[]` column on `product_variant` has been removed; all variant images are managed exclusively through the File module via `variant_media`.

## Requirements *(mandatory)*

### Functional Requirements

#### Product Media (US1–US3)

- **FR-001**: The system MUST allow an authorized seller (or an admin acting on behalf of a seller) to attach an already uploaded media file to an existing product.
- **FR-002**: The system MUST verify that a media file exists and is accessible before linking it to a product.
- **FR-003**: The system MUST prevent the same media file from being attached to the same product more than once.
- **FR-004**: The system MUST store product-specific media attributes, including whether the media is primary and its display order.
- **FR-005**: The system MUST ensure each product has no more than one primary media item.
- **FR-006**: The system MUST return product media alongside product detail and listing views; detail responses include media identifier, display URL, thumbnail or preview URL when available, primary status, and display order; listing responses include enough media information to render product cards without additional user-visible retrieval steps. Media within each product response is ordered by display order ascending, with a stable secondary sort when display order values are equal.
- **FR-007**: The system MUST support updating a product media item's primary status and display order.
- **FR-008**: The system MUST support removing a media item from a product.
- **FR-009**: The system MUST attempt to remove the underlying media asset when a product media item is removed.
- **FR-010**: The system MUST keep the product-media removal successful when underlying asset cleanup fails, while recording the cleanup failure for operational follow-up.
- **FR-011**: The system MUST assign a new primary media item after primary media removal when other media remains.
- **FR-012**: The system MUST maintain product ownership of product-media ordering and primary selection, independent of generic file metadata.

#### Variant Media (US4)

- **FR-013**: The system MUST NOT store variant images as raw URL strings on the `product_variant` table; all variant visual assets MUST be managed through the File module via the `variant_media` join table.
- **FR-014**: The system MUST allow an authorized seller to attach an already uploaded media file to an existing product variant.
- **FR-015**: The system MUST verify that a media file exists and is accessible before linking it to a variant.
- **FR-016**: The system MUST prevent the same media file from being attached to the same variant more than once.
- **FR-017**: The system MUST store variant-specific media attributes, including whether the media is primary and its display order.
- **FR-018**: The system MUST ensure each variant has no more than one primary media item.
- **FR-019**: The system MUST return variant media alongside variant detail and variant find-by-options responses; the `media` field is always a JSON array, never `null`, and is ordered by display order ascending.
- **FR-020**: The system MUST support updating a variant media item's primary status and display order.
- **FR-021**: The system MUST support removing a media item from a variant, with best-effort file cleanup and primary fallback promotion when applicable.
- **FR-022**: The system MUST scope all variant media operations to the owning seller; a different seller receives 404.
- **FR-023**: Products where variants differ only on non-visual specifications (e.g., RAM, storage size) SHOULD use product-level media (`product_media`) instead of duplicating the same file across every variant.
- **FR-024**: The system MUST return an empty `media` array for variants that have no attached media items.

### Key Entities

- **Product**: A sellable catalog item that can have zero or more associated media items via `product_media`, and zero or more variants each with their own media.
- **Product Media**: A product-owned association to an uploaded media file, including display order and primary status. Used for variant-agnostic images (e.g., packaging, lifestyle shots, products where variants differ only on non-visual specs like RAM).
- **Product Variant**: A purchasable SKU defined by a combination of option values (e.g., Red + Size M). Carries its own `variant_media` collection for variant-specific visuals.
- **Variant Media**: A variant-owned association to an uploaded media file, including display order and primary status. Used when variants are visually distinct (e.g., color/finish).
- **Uploaded Media File**: An existing file asset managed by the File module. May represent an image, video, document, thumbnail, poster, or other media type.
- **Product Media Summary / Variant Media Summary**: The user-facing media representation embedded in product/variant responses, containing only the information needed to render and manage media items.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 95% of product detail views with attached media show the correct ordered media collection on first load during acceptance testing. *(UAT / release gate — verified by manual QA or acceptance test run, not a CI assertion.)*
- **SC-002**: 95% of product listing views with attached media show the expected primary or first available media item without requiring users to manually refresh. *(UAT / release gate — same as SC-001.)*
- **SC-003**: Authorized sellers (or admins acting on behalf of a seller) can attach, reorder, mark primary, and remove product media in under 2 minutes for a typical product with up to 10 media items. *(UX benchmark — verified by manual walkthrough before release, not automated.)*
- **SC-004**: Duplicate media attachments are rejected 100% of the time in validation testing for both product and variant media. *(Automated — covered by integration tests.)*
- **SC-005**: Product and variant pages remain available 100% of the time in validation testing when a referenced media file is missing or inaccessible. *(Automated — covered by integration tests.)*
- **SC-006**: Product listing pages containing 20 products with attached media remain usable without visible step-by-step loading of each product's media. *(Automated — covered by integration tests asserting batched media resolution.)*
- **SC-007**: Variant detail and find-by-options responses always include `media` as a JSON array (never `null`), regardless of whether any media is attached. *(Automated — covered by variant media integration tests.)*
- **SC-008**: Variant media operations are fully seller-isolated; a seller cannot attach, update, or remove media for another seller's variants. *(Automated — covered by variant media integration tests.)*

## Assumptions

- The feature uses the hybrid product media representation described in the v2 prespec: product responses expose product-relevant media fields rather than raw file-module metadata.
- Media files are uploaded before they are attached to products; this feature does not add direct product media upload.
- Product media may include images, videos, documents, and future media types that can be represented by an uploaded file.
- Removing media from a product also attempts to remove the underlying file asset, while product correctness takes priority if asset cleanup fails.
- Thumbnail or preview URLs are included when available; if no thumbnail or preview exists, consumers may use the main display URL as a fallback.
- Existing authorization rules for product administration and file access apply.
- Throughout this document, "admin" in user-facing descriptions refers to an authorized seller or a system admin acting on behalf of a seller, consistent with the project's RBAC model (Seller role manages own products; Admin role has full access). Route-level enforcement uses the existing seller and admin authentication middleware.

## Verification & Testing

This section defines how the feature must be verified so delivery matches real client usage and project quality bars.

- **End-to-end API verification (primary)**: Automated tests must call the same public HTTP API that clients use (product detail/list and product media management), exercising the full request path including authentication context, tenant headers, correlation identifiers, validation, and persistence. Assertions should reflect what an API consumer would observe (status codes, response bodies, and follow-up reads via the API where applicable).

- **Realistic stack**: Primary automated tests must not replace the live application wiring with mocks for HTTP routing, middleware, or the database. Storage and file behavior follow the same rules as in other Product and File features unless an external dependency is explicitly brought up by the shared test harness (for example object storage in file upload tests).

- **Scenario completeness**: Automated verification must cover the acceptance scenarios and edge cases in this document, including happy paths, authorization and tenant isolation failures, validation and duplicate conflicts, missing resources, correlation-ID requirements, and degraded file cleanup.

- **Unit tests (supplement only)**: Where an integration-style API test is impractical for a narrow, isolated behavior (for example pure mapping rules with no meaningful HTTP surface), targeted unit tests may be added so that no specified behavior ships without automated coverage.

- **Coverage expectation**: Every functional requirement and acceptance scenario in this specification must map to at least one automated test (integration-first, with unit tests only where justified above). Existing Product and File automated suites must remain passing so regressions are caught.
