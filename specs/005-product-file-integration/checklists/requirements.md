# Manual UAT / Release-Gate Checklist — Product File Integration (005)

**Feature:** Product File Integration (includes Variant Media — US4)  
**Last updated:** 2026-05-21  
**Release gate:** All items below must pass before this feature is promoted to production.

---

## SC-001 — Product Detail Media Correctness (target ≥ 95 %)

Verify that `GET /api/product/:id` returns accurate media information for a product with attached files.

- [ ] Product detail response always includes the `media` field as a JSON array (never `null`)
- [ ] Media items are ordered by `displayOrder ASC, id ASC`
- [ ] Each media item contains `fileId`, `url`, `isPrimary`, and `displayOrder`
- [ ] Exactly one item has `isPrimary: true` when at least one item is attached
- [ ] Product loads successfully (200) when a referenced file is missing or inaccessible (graceful degradation)
- [ ] Product loads successfully (200) when `media` is empty (`[]`)
- [ ] Verified on ≥ 20 distinct products with varied media counts (0, 1, 5, 10+)

**Result:** `[ ] PASS  [ ] FAIL`  
**Notes:**

---

## SC-002 — Product Listing Media Accuracy (target ≥ 95 %)

Verify that `GET /api/product` returns accurate media summaries for all products in the list.

- [ ] Every product in the listing response includes the `media` field as a JSON array
- [ ] Media resolution is batched — no N+1 query pattern observed in DB query logs
- [ ] Products with no media show `media: []`
- [ ] Products with media show ordered items matching DB state
- [ ] Listing remains performant under 500 ms for a page of 20 products with media
- [ ] Verified with a mix of products with 0, 1, and multiple media items on the same page

**Result:** `[ ] PASS  [ ] FAIL`  
**Notes:**

---

## SC-003 — Attach / Reorder / Primary / Remove Cycle (target: < 2 min for 10-media product)

Walk through the full management lifecycle for a 10-media product:

### Attach (POST /api/product/:productId/media)

- [ ] Attach 10 files to a single product with incrementing `displayOrder` values (0–9)
- [ ] First attachment with `isPrimary: true` is reflected in the product detail
- [ ] Subsequent attachment with `isPrimary: true` clears the previous primary
- [ ] Duplicate attachment attempt returns `409 Conflict`
- [ ] Attachment with an invalid/inaccessible `fileId` returns `422 Unprocessable Entity`
- [ ] Unauthenticated attachment attempt returns `401 Unauthorized`
- [ ] Customer (non-seller) attachment attempt returns `403 Forbidden`
- [ ] Wrong-seller attachment returns `404 Not Found`

### Reorder / Update (PATCH /api/product/:productId/media/:fileId)

- [ ] Update `displayOrder` of 3 items to verify reordering is reflected in subsequent GET
- [ ] Set `isPrimary: true` on a non-primary item — previous primary is automatically demoted
- [ ] Empty body update attempt returns `400 Bad Request`
- [ ] `displayOrder` below 0 returns `400 Bad Request`

### Remove (DELETE /api/product/:productId/media/:fileId)

- [ ] Remove a non-primary item — product still loads and remaining items are intact
- [ ] Remove the primary item — the lowest-order remaining item is automatically promoted to primary
- [ ] Remove the last item — product loads with `media: []`
- [ ] Remove on a non-existent link returns `404 Not Found`
- [ ] Unauthenticated remove returns `401 Unauthorized`
- [ ] Customer (non-seller) remove returns `403 Forbidden`
- [ ] Wrong-seller remove returns `404 Not Found`
- [ ] Remove returns `204 No Content` regardless of whether file asset cleanup succeeds
- [ ] After all 10 removals the product responds with `media: []`

### Timing

- [ ] Full 10-media attach → reorder → remove cycle completes in under 2 minutes

**Result:** `[ ] PASS  [ ] FAIL`  
**Notes:**

---

---

## SC-004 — Variant Detail Media Correctness (target ≥ 95 %)

Verify that `GET /api/product/:productId/variant/:variantId` returns accurate media information.

- [ ] Variant detail response always includes the `media` field as a JSON array (never `null`)
- [ ] Media items are ordered by `displayOrder ASC, id ASC`
- [ ] Each media item contains `fileId`, `url`, `isPrimary`, and `displayOrder`
- [ ] Exactly one item has `isPrimary: true` when at least one item is attached
- [ ] Variant loads successfully (200) when a referenced file is missing or inaccessible
- [ ] Variant loads successfully (200) when `media` is empty (`[]`)
- [ ] `media` field is also present on `GET /api/product/:productId/variant/find` (find-by-options) responses
- [ ] Verified on variants across multiple products

**Result:** `[ ] PASS  [ ] FAIL`  
**Notes:**

---

## SC-005 — Variant Attach / Reorder / Primary / Remove Cycle

Walk through the full variant media management lifecycle for a single variant with 5 media items:

### Attach (POST /api/product/:productId/variant/:variantId/media)

- [ ] Attach 5 files to a single variant with incrementing `displayOrder` values (0–4)
- [ ] First attachment with `isPrimary: true` is reflected in the variant detail
- [ ] Subsequent attachment with `isPrimary: true` clears the previous primary
- [ ] Duplicate attachment attempt returns `409 Conflict`
- [ ] Attachment with an invalid/inaccessible `fileId` returns `422 Unprocessable Entity`
- [ ] Unauthenticated attachment attempt returns `401 Unauthorized`
- [ ] Customer (non-seller) attachment attempt returns `403 Forbidden`
- [ ] Wrong-seller attachment returns `404 Not Found`

### Reorder / Update (PATCH /api/product/:productId/variant/:variantId/media/:fileId)

- [ ] Update `displayOrder` of 2 items — order is reflected in subsequent GET
- [ ] Set `isPrimary: true` on a non-primary item — previous primary is automatically demoted
- [ ] Empty body update attempt returns `400 Bad Request`

### Remove (DELETE /api/product/:productId/variant/:variantId/media/:fileId)

- [ ] Remove a non-primary item — variant still loads and remaining items are intact
- [ ] Remove the primary item — the lowest-order remaining item is automatically promoted to primary
- [ ] Remove the last item — variant loads with `media: []`
- [ ] Remove on a non-existent link returns `404 Not Found`
- [ ] Unauthenticated remove returns `401 Unauthorized`
- [ ] Wrong-seller remove returns `404 Not Found`
- [ ] Remove returns `204 No Content` regardless of whether file asset cleanup succeeds

**Result:** `[ ] PASS  [ ] FAIL`  
**Notes:**

---

## SC-006 — Architecture Boundary

Verify the module boundary contract holds throughout the implementation:

- [ ] No product module Go file imports a File module repository or File entity package
- [ ] `product_variant` table has no `images` column (dropped by migration 020)
- [ ] Sending `"images": [...]` in `POST /variant` or `PUT /variant/:id` request body returns `201`/`200` (field silently ignored)
- [ ] `media` field is present and is a JSON array on all variant responses (create, update, get by ID, find by options, bulk update)

**Result:** `[ ] PASS  [ ] FAIL`  
**Notes:**

---

## Sign-Off

| Reviewer | Role | Date | Result |
|----------|------|------|--------|
|          |      |      |        |

All gates must show **PASS** before production deployment.
