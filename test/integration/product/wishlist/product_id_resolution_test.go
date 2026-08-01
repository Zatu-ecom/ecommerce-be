package wishlist

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProductIDResolutionForWishlistItem verifies that when the wishlist AddItem API
// receives a product ID (instead of a variant ID) in the variantId field, it
// defensively resolves it to the product's default variant.
//
// Endpoints:
//   - POST /api/product/wishlist/:id/item - Add item to wishlist
//
// IMPORTANT: The standard seed data (002_seed_products.sql) has product IDs (1-9)
// that overlap with variant IDs (1-20), so FindVariantByID(X) succeeds for product
// IDs 1-9 without ever reaching the product-ID fallback. To properly test the
// resolution logic, we insert dedicated test products with IDs (100+) that cannot
// collide with any existing variant IDs.
//
// Test product data inserted by this test:
//   - Product 100 (ResolutionSingle) → variant 200 (is_default=true, only variant)
//   - Product 101 (ResolutionMulti)  → variants 201 (is_default=true), 202, 203
//   - Product 102 (ResolutionOther)  → variant 204 (is_default=true, only variant)
func TestProductIDResolutionForWishlistItem(t *testing.T) {
	// Setup test containers
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run migrations and seeds
	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	// Insert dedicated test products with IDs that cannot collide with any variant ID.
	// Standard seed variant IDs go up to 20, so using 100+ guarantees no collision.
	sqlDB, err := containers.DB.DB()
	require.NoError(t, err)

	_, err = sqlDB.Exec(`
		INSERT INTO product (id, name, category_id, brand, base_sku, short_description, long_description, seller_id, created_at, updated_at) VALUES
		(100, 'ResolutionTest Single Variant', 4, 'TestBrand', 'RES-SINGLE', 'Single variant product', 'Single variant for resolution testing', 2, NOW(), NOW()),
		(101, 'ResolutionTest Multi Variant', 4, 'TestBrand', 'RES-MULTI', 'Multi variant product', 'Multiple variants for resolution testing', 2, NOW(), NOW()),
		(102, 'ResolutionTest Other', 4, 'TestBrand', 'RES-OTHER', 'Other single variant', 'Another single variant product', 2, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING;

		INSERT INTO product_variant (id, product_id, sku, price, allow_purchase, is_popular, is_default) VALUES
		(200, 100, 'RES-SINGLE-200', 10.00, true, true, true),
		(201, 101, 'RES-MULTI-201', 10.00, true, true, true),
		(202, 101, 'RES-MULTI-202', 15.00, true, false, false),
		(203, 101, 'RES-MULTI-203', 20.00, true, false, false),
		(204, 102, 'RES-OTHER-204', 25.00, true, true, true)
		ON CONFLICT (id) DO NOTHING;

		SELECT setval('product_id_seq', GREATEST((SELECT COALESCE(MAX(id), 0) FROM product), 102));
		SELECT setval('product_variant_id_seq', GREATEST((SELECT COALESCE(MAX(id), 0) FROM product_variant), 204));
	`)
	require.NoError(t, err)

	// Setup test server
	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Create API client
	client := helpers.NewAPIClient(server)

	// Login as customer (Alice - user 5)
	token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
	client.SetToken(token)

	// Create wishlist for Alice
	w := client.Post(t, "/api/product/wishlist", map[string]any{
		"name": "Resolution Test Wishlist",
	})
	response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	wishlistData := helpers.GetResponseData(t, response, "wishlist")
	wishlistID := uint(wishlistData["id"].(float64))

	// Create a second wishlist for multi-wishlist tests
	w = client.Post(t, "/api/product/wishlist", map[string]any{
		"name": "Second Resolution Wishlist",
	})
	response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	wishlistData2 := helpers.GetResponseData(t, response, "wishlist")
	wishlistID2 := uint(wishlistData2["id"].(float64))

	// ============================================================================
	// HAPPY PATH: Product ID → Default Variant Resolution
	// ============================================================================

	t.Run(
		"HP-RESOLVE-001: Product ID resolves to its default variant (single variant product)",
		func(t *testing.T) {
			// Product 100 has only variant 200 (is_default=true)
			productID := uint(100)
			expectedVariantID := uint(200)

			w := client.Post(
				t,
				fmt.Sprintf("/api/product/wishlist/%d/item", wishlistID),
				map[string]any{
					"variantId": productID,
				},
			)
			response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)

			item := helpers.GetResponseData(t, response, "wishlistItem")
			assert.NotNil(t, item["id"], "Item should have ID")
			assert.Equal(t, float64(expectedVariantID), item["variantId"],
				"Should resolve product ID 100 to default variant 200")
		},
	)

	t.Run(
		"HP-RESOLVE-002: Product ID resolves to default when product has multiple variants",
		func(t *testing.T) {
			// Product 101 has variants 201-203, variant 201 is is_default=true
			productID := uint(101)
			expectedVariantID := uint(201)

			w := client.Post(
				t,
				fmt.Sprintf("/api/product/wishlist/%d/item", wishlistID),
				map[string]any{
					"variantId": productID,
				},
			)
			response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)

			item := helpers.GetResponseData(t, response, "wishlistItem")
			assert.Equal(t, float64(expectedVariantID), item["variantId"],
				"Should resolve product ID 101 to default variant 201")
		},
	)

	t.Run("HP-RESOLVE-003: Valid variant ID still works (backward compatible)", func(t *testing.T) {
		// Variant 202 is a valid variant ID for product 101 (not the default)
		variantID := uint(202)

		w := client.Post(
			t,
			fmt.Sprintf("/api/product/wishlist/%d/item", wishlistID),
			map[string]any{
				"variantId": variantID,
			},
		)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		item := helpers.GetResponseData(t, response, "wishlistItem")
		assert.Equal(t, float64(variantID), item["variantId"],
			"Valid variant ID should be used as-is without resolution")
	})

	t.Run(
		"HP-RESOLVE-004: Same product ID resolves to same default variant consistently",
		func(t *testing.T) {
			// Product 102 has variant 204 (is_default=true)
			productID := uint(102)
			expectedVariantID := uint(204)

			w := client.Post(
				t,
				fmt.Sprintf("/api/product/wishlist/%d/item", wishlistID2),
				map[string]any{
					"variantId": productID,
				},
			)
			response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)

			item := helpers.GetResponseData(t, response, "wishlistItem")
			assert.Equal(t, float64(expectedVariantID), item["variantId"],
				"Should resolve product ID 102 to default variant 204")
		},
	)

	// ============================================================================
	// HAPPY PATH: Mixed Scenarios
	// ============================================================================

	t.Run(
		"HP-RESOLVE-005: Add product ID then add one of its specific variant IDs",
		func(t *testing.T) {
			// Product 101: default = 201, other variant = 203
			// This tests that adding a specific variant of a product whose default was
			// already resolved works fine (they are different variant IDs in the wishlist).
			specificVariantID := uint(203)

			w := client.Post(
				t,
				fmt.Sprintf("/api/product/wishlist/%d/item", wishlistID),
				map[string]any{
					"variantId": specificVariantID,
				},
			)
			response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
			item := helpers.GetResponseData(t, response, "wishlistItem")
			assert.Equal(t, float64(specificVariantID), item["variantId"],
				"Specific variant ID should work alongside resolved product IDs")
		},
	)

	// ============================================================================
	// NEGATIVE SCENARIOS
	// ============================================================================

	t.Run("NEG-RESOLVE-001: Non-existent ID returns 404", func(t *testing.T) {
		w := client.Post(
			t,
			fmt.Sprintf("/api/product/wishlist/%d/item", wishlistID),
			map[string]any{
				"variantId": 99999,
			},
		)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("NEG-RESOLVE-002: Duplicate after product-ID resolution returns 409", func(t *testing.T) {
		// Product ID 100 was already added in HP-RESOLVE-001, which resolved to variant 200.
		// Adding product ID 100 again should detect the duplicate on variant 200.
		w := client.Post(
			t,
			fmt.Sprintf("/api/product/wishlist/%d/item", wishlistID),
			map[string]any{
				"variantId": 100,
			},
		)
		helpers.AssertErrorResponse(t, w, http.StatusConflict)
	})

	t.Run(
		"NEG-RESOLVE-003: Duplicate when adding the resolved variant directly returns 409",
		func(t *testing.T) {
			// Variant 200 was already added (via product ID 100 resolution in HP-RESOLVE-001).
			// Adding variant 200 directly should also be detected as duplicate.
			w := client.Post(
				t,
				fmt.Sprintf("/api/product/wishlist/%d/item", wishlistID),
				map[string]any{
					"variantId": 200,
				},
			)
			helpers.AssertErrorResponse(t, w, http.StatusConflict)
		},
	)

	// ============================================================================
	// EDGE CASES
	// ============================================================================

	t.Run("EDGE-RESOLVE-001: Product ID 0 returns error", func(t *testing.T) {
		w := client.Post(
			t,
			fmt.Sprintf("/api/product/wishlist/%d/item", wishlistID),
			map[string]any{
				"variantId": 0,
			},
		)
		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound,
			"Should reject variantId 0, got %d", w.Code)
	})

	t.Run(
		"EDGE-RESOLVE-002: Add product ID to same wishlist twice is duplicate",
		func(t *testing.T) {
			// Product 101 was already added in HP-RESOLVE-002
			w := client.Post(
				t,
				fmt.Sprintf("/api/product/wishlist/%d/item", wishlistID),
				map[string]any{
					"variantId": 101,
				},
			)
			helpers.AssertErrorResponse(t, w, http.StatusConflict)
		},
	)
}
