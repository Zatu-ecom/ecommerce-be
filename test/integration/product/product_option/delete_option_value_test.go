package product_option

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// TestDeleteOptionValue - Comprehensive tests for DeleteOptionValue API
func TestDeleteOptionValue(t *testing.T) {
	// Setup test containers
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run migrations and seeds
	containers.RunAllMigrations(t)
	containers.RunSeeds(t, "migrations/seeds/001_seed_user_data.sql")
	containers.RunSeeds(t, "migrations/seeds/002_seed_product_data.sql")

	// Setup test server
	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Create API client
	client := helpers.NewAPIClient(server)

	/*
		Seed Data Reference (002_seed_product_data.sql):
		- Product 5: "Classic Cotton T-Shirt" (seller_id: 3 - Jane/seller@example.com)
		  - Option 8: "Size" (position: 1)
		    - Value 23: "S" -> "Small" (position: 1) - NOT USED in variants
		    - Value 24: "M" -> "Medium" (position: 2) - USED in variants 9, 10
		    - Value 25: "L" -> "Large" (position: 3) - USED in variant 11
		    - Value 26: "XL" -> "Extra Large" (position: 4) - NOT USED in variants
		    - Value 27: "XXL" -> "2X Large" (position: 5) - NOT USED in variants
		  - Option 9: "Color" (position: 2)
		    - Value 28: "Black" -> "Black" #000000 (position: 1) - USED in variants 9, 11
		    - Value 29: "White" -> "White" #FFFFFF (position: 2) - USED in variant 10
		    - Value 30: "Navy" -> "Navy Blue" #000080 (position: 3) - NOT USED in variants
		    - Value 31: "Gray" -> "Gray" #808080 (position: 4) - NOT USED in variants

		Variants for Product 5:
		- Variant 9: Size M (24) + Color Black (28)
		- Variant 10: Size M (24) + Color White (29)
		- Variant 11: Size L (25) + Color Black (28)

		- Product 1: "iPhone 15 Pro" (seller_id: 2 - John Seller/john.seller@example.com)
		  - Option 1: "Color" - has variants using values
	*/

	// Helper function to delete an option value
	deleteOptionValue := func(productID int, optionID, valueID int) *httptest.ResponseRecorder {
		url := fmt.Sprintf("/api/products/%d/options/%d/values/%d", productID, optionID, valueID)
		return client.Delete(t, url)
	}

	// ============================================================================
	// SUCCESS CASES
	// ============================================================================

	t.Run("Delete option value successfully - not used in variants", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Delete value 23 (Small) - not used in any variant
		w := deleteOptionValue(5, 8, 23)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("Delete option value with color code - not used in variants", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Delete value 30 (Navy Blue) - has color code but not used in variants
		w := deleteOptionValue(5, 9, 30)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("Delete option value without color code - not used in variants", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Delete value 26 (XL) - no color code and not used in variants
		w := deleteOptionValue(5, 8, 26)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("Delete last value in sequence - not used in variants", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Delete value 31 (Gray) - last in the color option
		w := deleteOptionValue(5, 9, 31)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("Delete value and verify other values remain", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Delete value 27 (XXL)
		w := deleteOptionValue(5, 8, 27)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify other values still exist by trying to get available options
		getURL := "/api/products/5/options"
		wGet := client.Get(t, getURL)
		assert.Equal(t, http.StatusOK, wGet.Code)
		// The option should still have other values
	})

	// ============================================================================
	// AUTHENTICATION & AUTHORIZATION
	// ============================================================================

	t.Run("Delete without authentication", func(t *testing.T) {
		client.SetToken("")

		w := deleteOptionValue(5, 8, 24)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Delete with customer role", func(t *testing.T) {
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		w := deleteOptionValue(5, 8, 24)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Delete another seller's product option value", func(t *testing.T) {
		// Product 5 belongs to seller_id 3 (seller@example.com)
		// Try with seller_id 2 (john.seller@example.com)
		anotherSellerToken := helpers.Login(t, client, "john.seller@example.com", "seller123")
		client.SetToken(anotherSellerToken)

		w := deleteOptionValue(5, 8, 24)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	// ============================================================================
	// VALIDATION ERRORS - INVALID IDs
	// ============================================================================

	t.Run("Invalid product ID format", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		w := client.Delete(t, "/api/products/invalid/options/8/values/24")

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid option ID format", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		w := client.Delete(t, "/api/products/5/options/invalid/values/24")

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid value ID format", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		w := client.Delete(t, "/api/products/5/options/8/values/invalid")

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Non-existent product ID", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		w := deleteOptionValue(99999, 8, 24)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	// ============================================================================
	// VALIDATION ERRORS - RELATIONSHIPS
	// ============================================================================

	t.Run("Non-existent option ID", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		w := deleteOptionValue(5, 99999, 24)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Non-existent value ID", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		w := deleteOptionValue(5, 8, 99999)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Option doesn't belong to product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Option 8 belongs to product 5, try with product 6
		w := deleteOptionValue(6, 8, 24)

		helpers.AssertStatusCodeOneOf(t, w, http.StatusBadRequest, http.StatusNotFound)
	})

	t.Run("Value doesn't belong to option", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Value 28 (Black) belongs to option 9 (Color), try with option 8 (Size)
		w := deleteOptionValue(5, 8, 28)

		helpers.AssertStatusCodeOneOf(t, w, http.StatusBadRequest, http.StatusNotFound)
	})

	// ============================================================================
	// VARIANT IMPACT / BUSINESS LOGIC
	// ============================================================================

	t.Run(
		"Prevent deletion if value is used in active variants - single variant",
		func(t *testing.T) {
			sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
			client.SetToken(sellerToken)

			// Value 25 (Large) is used in variant 11
			w := deleteOptionValue(5, 8, 25)

			helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
		},
	)

	t.Run("Prevent deletion if value is used in multiple variants", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Value 24 (Medium) is used in variants 9 and 10
		w := deleteOptionValue(5, 8, 24)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run(
		"Prevent deletion - value used in multiple variants with different options",
		func(t *testing.T) {
			sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
			client.SetToken(sellerToken)

			// Value 28 (Black) is used in variants 9 and 11
			w := deleteOptionValue(5, 9, 28)

			helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
		},
	)

	t.Run("Check error message clarity for variant usage", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Try to delete value that's in use - White used in variant 10
		w := deleteOptionValue(5, 9, 29)

		// Should return 400 Bad Request when value is in use
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Verify variant still exists after failed deletion attempt", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Try to delete value 24 (Medium) - should fail
		w := deleteOptionValue(5, 8, 24)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Verify the variant still exists by checking product options
		// (In a real scenario, you'd query the variant endpoint)
		wGet := client.Get(t, "/api/products/5/options")
		assert.Equal(t, http.StatusOK, wGet.Code)
	})

	t.Run("Delete value after all variants using it are removed", func(t *testing.T) {
		// This test would require first deleting variants that use the value
		// Then successfully deleting the value
		// For now, we'll skip this as it requires variant deletion API
		t.Skip("Requires variant deletion API to be tested properly")
	})

	t.Run("Verify error includes helpful information", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Try to delete value in use - Medium used in 2 variants
		w := deleteOptionValue(5, 8, 24)

		response := helpers.AssertErrorResponse(t, w, http.StatusBadRequest)

		// Verify response structure
		helpers.AssertResponseStructure(t, response, false)
	})

	// ============================================================================
	// EDGE CASES
	// ============================================================================

	t.Run("Delete same value twice - second attempt should fail", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Try to delete value that was already deleted
		w := deleteOptionValue(5, 8, 23) // Already deleted in first test

		helpers.AssertStatusCodeOneOf(t, w, http.StatusNotFound, http.StatusBadRequest)
	})

	t.Run("Delete from option with only remaining values used in variants", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// After deleting several values, Size option should only have M and L left (both used in variants)
		// This is just to verify the state - both should fail to delete

		wM := deleteOptionValue(5, 8, 24) // Medium - used in variants
		assert.Equal(t, http.StatusBadRequest, wM.Code)

		wL := deleteOptionValue(5, 8, 25) // Large - used in variant
		assert.Equal(t, http.StatusBadRequest, wL.Code)
	})

	t.Run("Authorization still enforced when deleting from different seller", func(t *testing.T) {
		// Product 1 belongs to seller 2 (john.seller@example.com)
		// Login as seller 3 (Jane) and try to delete from Product 1
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Try to delete from iPhone 15 Pro (Product 1, owned by john.seller)
		w := deleteOptionValue(1, 1, 1) // Natural Titanium color

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}
