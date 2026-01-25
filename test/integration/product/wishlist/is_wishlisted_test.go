package wishlist

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// TestIsWishlistedField tests the isWishlisted field across all product and variant GET APIs
// This test verifies that the wishlist status is correctly returned for logged-in users
//
// Test Data Setup (from 005_seed_wishlist_data.sql):
// - User 5 (alice.j@example.com) has variants 1,2,3,4,5,6,7 in wishlists
// - User 6 (michael.s@example.com) has variants 9,10,12,14 in wishlists
// - User 7 (sarah.w@example.com) has variants 16,17,18 in wishlists
//
// Product/Variant Mapping:
// - Product 1 (iPhone, seller 2): variants 1,2,3,4
// - Product 2 (Samsung, seller 2): variants 5,6
// - Product 3 (MacBook, seller 2): variants 7,8
// - Product 5 (T-Shirt, seller 3): variants 9,10,11

func TestIsWishlistedField(t *testing.T) {
	// Setup test containers
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run migrations and seeds
	containers.RunAllMigrations(t)
	containers.RunAllSeeds(t) // Runs core seeds first, then mock seeds

	// Setup test server
	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Create API client
	client := helpers.NewAPIClient(server)

	// ============================================================================
	// GetVariantByID API - isWishlisted Tests
	// ============================================================================

	t.Run("GetVariantByID - isWishlisted true when user has variant in wishlist", func(t *testing.T) {
		// Login as customer (user 5 - alice.j@example.com)
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)
		client.SetHeader("X-Seller-ID", "2")

		// Variant 1 is in Alice's wishlist
		productID := 1
		variantID := 1
		url := fmt.Sprintf("/api/product/%d/variant/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		// Assert isWishlisted is true
		isWishlisted, ok := variant["isWishlisted"].(bool)
		assert.True(t, ok, "isWishlisted field should exist")
		assert.True(t, isWishlisted, "Variant 1 should be wishlisted for user 5")
	})

	t.Run("GetVariantByID - isWishlisted false when user doesn't have variant in wishlist", func(t *testing.T) {
		// Login as customer (user 5 - alice.j@example.com)
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)
		client.SetHeader("X-Seller-ID", "2")

		// Variant 8 (MacBook Silver) is NOT in Alice's wishlist
		productID := 3
		variantID := 8
		url := fmt.Sprintf("/api/product/%d/variant/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		// Assert isWishlisted is false
		isWishlisted, ok := variant["isWishlisted"].(bool)
		assert.True(t, ok, "isWishlisted field should exist")
		assert.False(t, isWishlisted, "Variant 8 should NOT be wishlisted for user 5")
	})

	t.Run("GetVariantByID - isWishlisted false for unauthenticated user", func(t *testing.T) {
		// Clear token (unauthenticated)
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "2")

		// Variant 1 is in Alice's wishlist, but user is not logged in
		productID := 1
		variantID := 1
		url := fmt.Sprintf("/api/product/%d/variant/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		// Assert isWishlisted is false (default for unauthenticated)
		isWishlisted, ok := variant["isWishlisted"].(bool)
		assert.True(t, ok, "isWishlisted field should exist")
		assert.False(t, isWishlisted, "isWishlisted should be false for unauthenticated user")
	})

	t.Run("GetVariantByID - different users see different wishlist status", func(t *testing.T) {
		// Variant 9 (T-Shirt Black M) is in Michael's wishlist (user 6) but NOT Alice's (user 5)

		// Test with Alice (should be false)
		tokenAlice := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(tokenAlice)
		client.SetHeader("X-Seller-ID", "3")

		productID := 5
		variantID := 9
		url := fmt.Sprintf("/api/product/%d/variant/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")
		isWishlistedAlice := variant["isWishlisted"].(bool)
		assert.False(t, isWishlistedAlice, "Variant 9 should NOT be wishlisted for Alice (user 5)")

		// Test with Michael (should be true)
		tokenMichael := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
		client.SetToken(tokenMichael)
		client.SetHeader("X-Seller-ID", "3")

		w = client.Get(t, url)

		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant = helpers.GetResponseData(t, response, "variant")
		isWishlistedMichael := variant["isWishlisted"].(bool)
		assert.True(t, isWishlistedMichael, "Variant 9 should be wishlisted for Michael (user 6)")
	})

	// ============================================================================
	// ListVariants API - isWishlisted Tests
	// ============================================================================

	t.Run("ListVariants - returns correct isWishlisted for each variant", func(t *testing.T) {
		// Login as customer (user 5 - alice.j@example.com)
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)
		client.SetHeader("X-Seller-ID", "2")

		// Get variants for product 1 (iPhone) - Alice has variants 1,2,3,4 in wishlist
		url := "/api/variant?productId=1"
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		variants := data["variants"].([]interface{})

		assert.NotEmpty(t, variants, "Should return variants")

		// Check each variant's wishlist status
		wishlistedCount := 0
		for _, v := range variants {
			variant := v.(map[string]interface{})
			variantID := int(variant["id"].(float64))
			isWishlisted := variant["isWishlisted"].(bool)

			// Alice has variants 1,2,3,4 in wishlist
			if variantID >= 1 && variantID <= 4 {
				assert.True(t, isWishlisted, "Variant %d should be wishlisted for user 5", variantID)
				wishlistedCount++
			}
		}
		assert.Greater(t, wishlistedCount, 0, "Should find at least one wishlisted variant")
	})

	t.Run("ListVariants - isWishlisted false for unauthenticated user", func(t *testing.T) {
		// Clear token (unauthenticated)
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "2")

		url := "/api/variant?productId=1"
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		variants := data["variants"].([]interface{})

		// All variants should have isWishlisted = false for unauthenticated user
		for _, v := range variants {
			variant := v.(map[string]interface{})
			isWishlisted, ok := variant["isWishlisted"].(bool)
			assert.True(t, ok, "isWishlisted field should exist")
			assert.False(t, isWishlisted, "isWishlisted should be false for unauthenticated user")
		}
	})

	// ============================================================================
	// FindVariantByOptions API - isWishlisted Tests
	// ============================================================================

	t.Run("FindVariantByOptions - isWishlisted true when variant in wishlist", func(t *testing.T) {
		// Login as customer (user 5 - alice.j@example.com)
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)
		client.SetHeader("X-Seller-ID", "2")

		// Find variant 1 by options (Natural Titanium + 128GB)
		productID := 1
		url := fmt.Sprintf("/api/product/%d/variant/find?color=Natural%%20Titanium&storage=128GB", productID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		// Assert isWishlisted is true for variant 1
		isWishlisted, ok := variant["isWishlisted"].(bool)
		assert.True(t, ok, "isWishlisted field should exist")
		assert.True(t, isWishlisted, "Found variant should be wishlisted for user 5")
	})

	t.Run("FindVariantByOptions - isWishlisted false for unauthenticated", func(t *testing.T) {
		// Clear token (unauthenticated)
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "2")

		productID := 1
		url := fmt.Sprintf("/api/product/%d/variant/find?color=Natural%%20Titanium&storage=128GB", productID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		// Assert isWishlisted is false for unauthenticated user
		isWishlisted, ok := variant["isWishlisted"].(bool)
		assert.True(t, ok, "isWishlisted field should exist")
		assert.False(t, isWishlisted, "isWishlisted should be false for unauthenticated user")
	})

	// ============================================================================
	// GetProductByID API - isWishlisted Tests
	// ============================================================================

	t.Run("GetProductByID - isWishlisted true when any variant in wishlist", func(t *testing.T) {
		// Login as customer (user 5 - alice.j@example.com)
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)
		client.SetHeader("X-Seller-ID", "2")

		// Product 1 (iPhone) has variants 1,2,3,4 all in Alice's wishlist
		url := "/api/product/1"
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		product := helpers.GetResponseData(t, response, "product")

		// Assert isWishlisted is true for product (since at least one variant is wishlisted)
		isWishlisted, ok := product["isWishlisted"].(bool)
		assert.True(t, ok, "isWishlisted field should exist on product")
		assert.True(t, isWishlisted, "Product 1 should be wishlisted for user 5 (has wishlisted variants)")
	})

	t.Run("GetProductByID - isWishlisted false when no variant in wishlist", func(t *testing.T) {
		// Login as customer (user 5 - alice.j@example.com)
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)
		client.SetHeader("X-Seller-ID", "4")

		// Product 8 (Sofa) has variants 16,17 - Alice doesn't have these
		url := "/api/product/8"
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		product := helpers.GetResponseData(t, response, "product")

		// Assert isWishlisted is false (no variants in Alice's wishlist)
		isWishlisted, ok := product["isWishlisted"].(bool)
		assert.True(t, ok, "isWishlisted field should exist on product")
		assert.False(t, isWishlisted, "Product 8 should NOT be wishlisted for user 5")
	})

	t.Run("GetProductByID - isWishlisted false for unauthenticated user", func(t *testing.T) {
		// Clear token (unauthenticated)
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "2")

		url := "/api/product/1"
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		product := helpers.GetResponseData(t, response, "product")

		// Assert isWishlisted is false for unauthenticated user
		isWishlisted, ok := product["isWishlisted"].(bool)
		assert.True(t, ok, "isWishlisted field should exist on product")
		assert.False(t, isWishlisted, "isWishlisted should be false for unauthenticated user")
	})

	// ============================================================================
	// GetAllProducts API - isWishlisted Tests
	// ============================================================================

	t.Run("GetAllProducts - returns correct isWishlisted for each product", func(t *testing.T) {
		// Login as customer (user 5 - alice.j@example.com)
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)
		client.SetHeader("X-Seller-ID", "2")

		url := "/api/products"
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		products := data["products"].([]interface{})

		assert.NotEmpty(t, products, "Should return products")

		// Check that isWishlisted field exists on all products
		for _, p := range products {
			product := p.(map[string]interface{})
			_, ok := product["isWishlisted"].(bool)
			assert.True(t, ok, "isWishlisted field should exist on product %v", product["id"])
		}

		// Find product 1 (iPhone) - should be wishlisted
		var foundProduct1 bool
		for _, p := range products {
			product := p.(map[string]interface{})
			if int(product["id"].(float64)) == 1 {
				foundProduct1 = true
				isWishlisted := product["isWishlisted"].(bool)
				assert.True(t, isWishlisted, "Product 1 should be wishlisted for user 5")
				break
			}
		}
		assert.True(t, foundProduct1, "Should find product 1 in response")
	})

	t.Run("GetAllProducts - isWishlisted false for unauthenticated user", func(t *testing.T) {
		// Clear token (unauthenticated)
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "2")

		url := "/api/products"
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		products := data["products"].([]interface{})

		// All products should have isWishlisted = false for unauthenticated user
		for _, p := range products {
			product := p.(map[string]interface{})
			isWishlisted, ok := product["isWishlisted"].(bool)
			assert.True(t, ok, "isWishlisted field should exist")
			assert.False(t, isWishlisted, "isWishlisted should be false for unauthenticated user")
		}
	})

	// ============================================================================
	// SearchProducts API - isWishlisted Tests
	// ============================================================================

	t.Run("SearchProducts - returns correct isWishlisted for search results", func(t *testing.T) {
		// Login as customer (user 5 - alice.j@example.com)
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)
		client.SetHeader("X-Seller-ID", "2")

		// Search for "iPhone" - should find product 1 which is wishlisted
		url := "/api/product/search?q=iPhone"
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		products := data["products"].([]interface{})

		if len(products) > 0 {
			// Check first product has isWishlisted field
			product := products[0].(map[string]interface{})
			_, ok := product["isWishlisted"].(bool)
			assert.True(t, ok, "isWishlisted field should exist on search results")
		}
	})
}

// TestIsWishlistedWithDirectDBInsert tests isWishlisted by inserting wishlist data directly into DB
func TestIsWishlistedWithDirectDBInsert(t *testing.T) {
	// Setup test containers
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run migrations and seeds (WITHOUT wishlist seed - we'll insert directly)
	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t) // Core seeds (roles, geo, plans)
	// Run only specific mock seeds (exclude wishlist seed)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	// Setup test server
	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Create API client
	client := helpers.NewAPIClient(server)

	t.Run("Direct DB Insert - verify isWishlisted becomes true after DB insert", func(t *testing.T) {
		// Login as customer (user 5 - alice.j@example.com)
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)
		client.SetHeader("X-Seller-ID", "2")

		// First, verify variant 1 is NOT wishlisted (no wishlist data yet)
		productID := 1
		variantID := 1
		url := fmt.Sprintf("/api/product/%d/variant/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")
		isWishlistedBefore := variant["isWishlisted"].(bool)
		assert.False(t, isWishlistedBefore, "Variant should NOT be wishlisted before DB insert")

		// Insert wishlist data directly into DB
		// Create wishlist for user 5
		err := containers.DB.Exec(`
			INSERT INTO wishlist (id, user_id, name, is_default, created_at, updated_at)
			VALUES (100, 5, 'Test Wishlist', TRUE, NOW(), NOW())
			ON CONFLICT (id) DO NOTHING
		`).Error
		assert.NoError(t, err, "Failed to insert wishlist")

		// Add variant 1 to wishlist
		err = containers.DB.Exec(`
			INSERT INTO wishlist_item (wishlist_id, variant_id, created_at, updated_at)
			VALUES (100, 1, NOW(), NOW())
			ON CONFLICT (wishlist_id, variant_id) DO NOTHING
		`).Error
		assert.NoError(t, err, "Failed to insert wishlist item")

		// Now verify variant 1 IS wishlisted
		w = client.Get(t, url)

		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant = helpers.GetResponseData(t, response, "variant")
		isWishlistedAfter := variant["isWishlisted"].(bool)
		assert.True(t, isWishlistedAfter, "Variant should be wishlisted after DB insert")
	})

	t.Run("Direct DB Insert - verify isWishlisted false after removing from DB", func(t *testing.T) {
		// Login as customer (user 5)
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)
		client.SetHeader("X-Seller-ID", "2")

		// First ensure wishlist exists
		err := containers.DB.Exec(`
			INSERT INTO wishlist (id, user_id, name, is_default, created_at, updated_at)
			VALUES (101, 5, 'Test Wishlist 2', FALSE, NOW(), NOW())
			ON CONFLICT (id) DO NOTHING
		`).Error
		assert.NoError(t, err)

		// Add variant 2 to wishlist
		err = containers.DB.Exec(`
			INSERT INTO wishlist_item (wishlist_id, variant_id, created_at, updated_at)
			VALUES (101, 2, NOW(), NOW())
			ON CONFLICT (wishlist_id, variant_id) DO NOTHING
		`).Error
		assert.NoError(t, err)

		// Verify it's wishlisted
		productID := 1
		variantID := 2
		url := fmt.Sprintf("/api/product/%d/variant/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")
		isWishlisted := variant["isWishlisted"].(bool)
		assert.True(t, isWishlisted, "Variant should be wishlisted")

		// Remove from wishlist
		err = containers.DB.Exec(`
			DELETE FROM wishlist_item WHERE wishlist_id = 101 AND variant_id = 2
		`).Error
		assert.NoError(t, err)

		// Verify it's no longer wishlisted
		w = client.Get(t, url)

		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant = helpers.GetResponseData(t, response, "variant")
		isWishlistedAfter := variant["isWishlisted"].(bool)
		assert.False(t, isWishlistedAfter, "Variant should NOT be wishlisted after removal")
	})
}
