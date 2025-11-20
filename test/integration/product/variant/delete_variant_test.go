package variant

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestDeleteVariant(t *testing.T) {
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

	// ============================================================================
	// SUCCESS SCENARIOS
	// ============================================================================

	t.Run("Success - Delete a non-default variant", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1 // iPhone with 4 variants
		variantID := 2 // Non-default variant

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Delete(t, url)

		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify variant is deleted - GET should return 404
		wGet := client.Get(t, url)
		assert.Equal(t, http.StatusNotFound, wGet.Code)
	})

	t.Run("Success - Delete a default variant (with other variants remaining)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5 // T-Shirt with 3 variants, variant 9 is default
		variantID := 9 // Default variant

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Delete(t, url)

		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify variant is deleted
		wGet := client.Get(t, url)
		assert.Equal(t, http.StatusNotFound, wGet.Code)

		// Verify other variants still exist
		variant10URL := fmt.Sprintf("/api/products/%d/variants/10", productID)
		w10 := client.Get(t, variant10URL)
		assert.Equal(t, http.StatusOK, w10.Code)

		variant11URL := fmt.Sprintf("/api/products/%d/variants/11", productID)
		w11 := client.Get(t, variant11URL)
		assert.Equal(t, http.StatusOK, w11.Code)
	})

	t.Run("Success - Delete variant with images", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 2 // Samsung with variants having images
		variantID := 5

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Delete(t, url)

		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify deletion
		wGet := client.Get(t, url)
		assert.Equal(t, http.StatusNotFound, wGet.Code)
	})

	t.Run("Success - Delete variant with option values", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1 // iPhone with color/storage options
		variantID := 3

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Delete(t, url)

		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify deletion
		wGet := client.Get(t, url)
		assert.Equal(t, http.StatusNotFound, wGet.Code)
	})

	t.Run("Success - Seller can delete own variant", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1 // Belongs to seller 2
		variantID := 4

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Delete(t, url)

		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("Success - Admin can delete any seller's variant", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		productID := 5 // Belongs to seller 3
		variantID := 10

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Delete(t, url)

		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("Success - Delete variant reduces total count", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 3 // MacBook with 2 variants

		// Verify we have 2 variants initially
		variant7URL := fmt.Sprintf("/api/products/%d/variants/7", productID)
		w7Before := client.Get(t, variant7URL)
		assert.Equal(t, http.StatusOK, w7Before.Code)

		variant8URL := fmt.Sprintf("/api/products/%d/variants/8", productID)
		w8Before := client.Get(t, variant8URL)
		assert.Equal(t, http.StatusOK, w8Before.Code)

		// Delete one variant
		deleteURL := fmt.Sprintf("/api/products/%d/variants/8", productID)
		wDelete := client.Delete(t, deleteURL)
		helpers.AssertSuccessResponse(t, wDelete, http.StatusOK)

		// Verify only 1 variant remains
		w7After := client.Get(t, variant7URL)
		assert.Equal(t, http.StatusOK, w7After.Code)

		w8After := client.Get(t, variant8URL)
		assert.Equal(t, http.StatusNotFound, w8After.Code)
	})

	t.Run("Success - Delete multiple variants sequentially (but not all)", func(t *testing.T) {
		seller4Token := helpers.Login(t, client, helpers.Seller4Email, helpers.Seller4Password)
		client.SetToken(seller4Token)

		productID := 8 // Sofa with 2 variants (16, 17)

		// Delete first variant - should succeed
		url16 := fmt.Sprintf("/api/products/%d/variants/16", productID)
		w16 := client.Delete(t, url16)
		helpers.AssertSuccessResponse(t, w16, http.StatusOK)

		// Try to delete last variant - should fail
		url17 := fmt.Sprintf("/api/products/%d/variants/17", productID)
		w17 := client.Delete(t, url17)
		helpers.AssertErrorResponse(t, w17, http.StatusBadRequest)
	})

	// ============================================================================
	// VALIDATION ERROR SCENARIOS
	// ============================================================================

	t.Run("Validation Error - Cannot delete last variant of product", func(t *testing.T) {
		seller4Token := helpers.Login(t, client, helpers.Seller4Email, helpers.Seller4Password)
		client.SetToken(seller4Token)

		productID := 9 // TV with only 1 variant (variant 18)
		variantID := 18

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)

		// Verify variant still exists
		wGet := client.Get(t, url)
		assert.Equal(t, http.StatusOK, wGet.Code)
	})

	t.Run(
		"Validation Error - Cannot delete all variants leaving product empty",
		func(t *testing.T) {
			sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
			client.SetToken(sellerToken)

			productID := 6 // Summer Dress with 2 variants (12, 13)

			// Delete first variant - should succeed
			url12 := fmt.Sprintf("/api/products/%d/variants/12", productID)
			w12 := client.Delete(t, url12)
			helpers.AssertSuccessResponse(t, w12, http.StatusOK)

			// Try to delete last remaining variant - should fail
			url13 := fmt.Sprintf("/api/products/%d/variants/13", productID)
			w13 := client.Delete(t, url13)
			helpers.AssertErrorResponse(t, w13, http.StatusBadRequest)

			// Verify last variant still exists
			wGet := client.Get(t, url13)
			assert.Equal(t, http.StatusOK, wGet.Code)
		},
	)

	t.Run(
		"Validation Error - Delete last non-default variant (default remains)",
		func(t *testing.T) {
			sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
			client.SetToken(sellerToken)

			productID := 7 // Running Shoes with 2 variants (14 is default, 15 is not)

			// Delete non-default variant - should succeed
			url15 := fmt.Sprintf("/api/products/%d/variants/15", productID)
			w15 := client.Delete(t, url15)
			helpers.AssertSuccessResponse(t, w15, http.StatusOK)

			// Verify default variant still exists
			url14 := fmt.Sprintf("/api/products/%d/variants/14", productID)
			w14 := client.Get(t, url14)
			assert.Equal(t, http.StatusOK, w14.Code)

			// Try to delete the last remaining (default) variant - should fail
			wDeleteLast := client.Delete(t, url14)
			helpers.AssertErrorResponse(t, wDeleteLast, http.StatusBadRequest)
		},
	)

	// ============================================================================
	// AUTHORIZATION SCENARIOS
	// ============================================================================

	t.Run("Authorization Error - Unauthenticated request", func(t *testing.T) {
		unauthClient := helpers.NewAPIClient(server)

		productID := 2
		variantID := 6

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := unauthClient.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Authorization Error - Customer cannot delete variant", func(t *testing.T) {
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		productID := 2
		variantID := 6

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run(
		"Authorization Error - Seller cannot delete another seller's variant",
		func(t *testing.T) {
			sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
			client.SetToken(sellerToken)

			productID := 2 // Belongs to seller 2
			variantID := 6

			url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
			w := client.Delete(t, url)

			helpers.AssertErrorResponse(t, w, http.StatusForbidden)
		},
	)

	// ============================================================================
	// NOT FOUND SCENARIOS
	// ============================================================================

	t.Run("Not Found - Non-existent product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 9999 // Doesn't exist
		variantID := 1

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Not Found - Non-existent variant", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 2    // Exists
		variantID := 9999 // Doesn't exist

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Not Found - Variant doesn't belong to product", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 2 // Samsung
		variantID := 1 // Belongs to product 1 (iPhone), not product 2

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Not Found - Already deleted variant", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 6  // Summer Dress - we already deleted variant 12 earlier
		variantID := 12 // This was deleted in a previous test

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)

		// Try to delete already deleted variant - should fail with 404
		w := client.Delete(t, url)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})
}
