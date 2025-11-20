package product

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/product/entity"
	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeleteProduct validates product deletion functionality
// including validation, cascading deletes, authorization, and database integrity
//
// Test Requirements:
// - migrations/seeds/001_seed_user_data.sql (for authentication)
// - migrations/seeds/002_seed_product_data.sql (for test products)
func TestDeleteProduct(t *testing.T) {
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
	// HAPPY PATH SCENARIOS
	// ============================================================================

	t.Run("001 - Seller Successfully Deletes Own Product", func(t *testing.T) {
		// Given: Seller is authenticated and owns product
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller (Jane = seller_id 3)
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		productID := product.ID

		// Count related entities before deletion
		var variantCount, optionCount, attributeCount int64
		containers.DB.Model(&entity.ProductVariant{}).
			Where("product_id = ?", productID).
			Count(&variantCount)
		containers.DB.Model(&entity.ProductOption{}).
			Where("product_id = ?", productID).
			Count(&optionCount)
		containers.DB.Model(&entity.ProductAttribute{}).
			Where("product_id = ?", productID).
			Count(&attributeCount)

		// When: Seller sends DELETE request
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Delete(t, url)

		// Then: Validate response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.NotNil(t, response["message"], "Response should contain message")

		// Validate product is deleted from database
		var productCount int64
		containers.DB.Model(&entity.Product{}).Where("id = ?", productID).Count(&productCount)
		assert.Equal(t, int64(0), productCount, "Product should be deleted from database")

		// Validate all variants are deleted
		var deletedVariantCount int64
		containers.DB.Model(&entity.ProductVariant{}).
			Where("product_id = ?", productID).
			Count(&deletedVariantCount)
		assert.Equal(t, int64(0), deletedVariantCount, "All variants should be deleted")

		// Validate all options are deleted
		var deletedOptionCount int64
		containers.DB.Model(&entity.ProductOption{}).
			Where("product_id = ?", productID).
			Count(&deletedOptionCount)
		assert.Equal(t, int64(0), deletedOptionCount, "All product options should be deleted")

		// Validate all product attributes are deleted
		var deletedAttributeCount int64
		containers.DB.Model(&entity.ProductAttribute{}).
			Where("product_id = ?", productID).
			Count(&deletedAttributeCount)
		assert.Equal(t, int64(0), deletedAttributeCount, "All product attributes should be deleted")

		// Subsequent GET request should return 404
		wGet := client.Get(t, url)
		assert.Equal(t, http.StatusNotFound, wGet.Code, "GET should return 404 after deletion")
	})

	t.Run("002 - Admin Successfully Deletes Any Product", func(t *testing.T) {
		// Given: Admin is authenticated
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Find a product owned by different seller (seller_id 2)
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.Seller2UserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller 2")

		productID := product.ID

		// When: Admin sends DELETE request
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Delete(t, url)

		// Then: Validate response
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Validate product is deleted
		var productCount int64
		containers.DB.Model(&entity.Product{}).Where("id = ?", productID).Count(&productCount)
		assert.Equal(t, int64(0), productCount, "Product should be deleted by admin")

		// Validate no authorization error
		assert.Equal(t, http.StatusOK, w.Code, "Admin should be able to delete any product")
	})

	t.Run("003 - Delete Product with Multiple Variants", func(t *testing.T) {
		// Given: Product with multiple variants
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		// Find a product with multiple variants owned by seller 2
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.Seller2UserID).
			Joins("JOIN product_variant ON product_variant.product_id = product.id").
			Group("product.id").
			Having("COUNT(product_variant.id) > 1").
			First(&product).Error
		require.NoError(t, err, "Should find a product with multiple variants")

		productID := product.ID

		// Count variants before deletion
		var variantCount int64
		containers.DB.Model(&entity.ProductVariant{}).
			Where("product_id = ?", productID).
			Count(&variantCount)
		require.Greater(t, variantCount, int64(1), "Product should have multiple variants")

		// When: Owner sends DELETE request
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Delete(t, url)

		// Then: Validate all variants are deleted
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		var deletedVariantCount int64
		containers.DB.Model(&entity.ProductVariant{}).
			Where("product_id = ?", productID).
			Count(&deletedVariantCount)
		assert.Equal(t, int64(0), deletedVariantCount, "All variants should be deleted")

		// Verify no orphaned variant_option_values
		var orphanedVOVCount int64
		containers.DB.Table("variant_option_value").
			Joins("LEFT JOIN product_variant ON variant_option_value.variant_id = product_variant.id").
			Where("product_variant.id IS NULL").
			Count(&orphanedVOVCount)
		assert.Equal(
			t,
			int64(0),
			orphanedVOVCount,
			"No orphaned variant option values should exist",
		)
	})

	t.Run("004 - Delete Product with Attributes and Package Options", func(t *testing.T) {
		// Given: Product with attributes
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := uint(3) // MacBook Pro (has attributes)

		// Verify product has attributes
		var attributeCount int64
		containers.DB.Model(&entity.ProductAttribute{}).
			Where("product_id = ?", productID).
			Count(&attributeCount)
		require.Greater(t, attributeCount, int64(0), "Product should have attributes")

		// When: Owner sends DELETE request
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Delete(t, url)

		// Then: Validate deletion
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Validate all product attributes are deleted
		var deletedAttributeCount int64
		containers.DB.Model(&entity.ProductAttribute{}).
			Where("product_id = ?", productID).
			Count(&deletedAttributeCount)
		assert.Equal(t, int64(0), deletedAttributeCount, "All product attributes should be deleted")

		// Validate attribute definitions remain (not deleted)
		var attrDefCount int64
		containers.DB.Model(&entity.AttributeDefinition{}).Count(&attrDefCount)
		assert.Greater(t, attrDefCount, int64(0), "Attribute definitions should remain in database")
	})

	t.Run(
		"005 - Delete Product with Only Default Variant (No Options)",
		func(t *testing.T) {
			// Given: Simple product with one variant
			seller4Token := helpers.Login(t, client, helpers.Seller4Email, helpers.Seller4Password)
			client.SetToken(seller4Token)

			// Find product owned by seller 4 (Home & Living)
			var product entity.Product
			err := containers.DB.Where("seller_id = ?", helpers.Seller4UserID).First(&product).Error
			require.NoError(t, err, "Should find product for seller 4")

			productID := product.ID

			// When: Owner sends DELETE request
			url := fmt.Sprintf("/api/products/%d", productID)
			w := client.Delete(t, url)

			// Then: Validate deletion
			helpers.AssertSuccessResponse(t, w, http.StatusOK)

			// Validate product is deleted
			var productCount int64
			containers.DB.Model(&entity.Product{}).Where("id = ?", productID).Count(&productCount)
			assert.Equal(t, int64(0), productCount, "Product should be deleted")
		},
	)

	// ============================================================================
	// NEGATIVE SCENARIOS - AUTHENTICATION
	// ============================================================================

	t.Run("NEG_001 - Delete Product Without Authentication", func(t *testing.T) {
		// Given: No authentication token
		client.SetToken("")

		// Find any existing product
		var product entity.Product
		err := containers.DB.First(&product).Error
		require.NoError(t, err, "Should find a product")

		productID := product.ID

		// When: Request sent without token
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Delete(t, url)

		// Then: Request is rejected with 401
		assert.Equal(t, http.StatusUnauthorized, w.Code, "Should return 401 Unauthorized")

		// Validate product is NOT deleted
		var productCount int64
		containers.DB.Model(&entity.Product{}).Where("id = ?", productID).Count(&productCount)
		assert.Equal(t, int64(1), productCount, "Product should not be deleted")
	})

	t.Run("NEG_002 - Delete Product with Invalid Token", func(t *testing.T) {
		// Given: Invalid JWT token
		client.SetToken("invalid.jwt.token")

		// Find any existing product
		var product entity.Product
		err := containers.DB.First(&product).Error
		require.NoError(t, err, "Should find a product")

		productID := product.ID

		// When: Request sent with invalid token
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Delete(t, url)

		// Then: Request is rejected
		assert.Equal(t, http.StatusUnauthorized, w.Code, "Should return 401 Unauthorized")

		// Validate product is NOT deleted
		var productCount int64
		containers.DB.Model(&entity.Product{}).Where("id = ?", productID).Count(&productCount)
		assert.Equal(t, int64(1), productCount, "Product should not be deleted")
	})

	t.Run("NEG_003 - Delete Product with Malformed Token", func(t *testing.T) {
		// Given: Malformed token (random string)
		client.SetToken("Bearer invalidtoken123")

		// Find any existing product
		var product entity.Product
		err := containers.DB.First(&product).Error
		require.NoError(t, err, "Should find a product")

		productID := product.ID

		// When: Request sent with malformed token
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Delete(t, url)

		// Then: Request is rejected
		assert.Equal(t, http.StatusUnauthorized, w.Code, "Should return 401 Unauthorized")

		// Validate product remains unchanged
		var productCount int64
		containers.DB.Model(&entity.Product{}).Where("id = ?", productID).Count(&productCount)
		assert.Equal(t, int64(1), productCount, "Product should not be deleted")
	})

	// ============================================================================
	// NEGATIVE SCENARIOS - AUTHORIZATION
	// ============================================================================

	t.Run("NEG_004 - Seller Tries to Delete Another Seller's Product", func(t *testing.T) {
		// Given: Seller Jane (seller_id = 3) is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find product owned by seller 2 (not Jane)
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.Seller2UserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller 2")

		productID := product.ID

		// When: Jane tries to delete product owned by seller 2
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Delete(t, url)

		// Then: Request is rejected with 403 Forbidden
		assert.Equal(t, http.StatusForbidden, w.Code, "Should return 403 Forbidden")

		// Validate product is NOT deleted
		var productCount int64
		containers.DB.Model(&entity.Product{}).Where("id = ?", productID).Count(&productCount)
		assert.Equal(t, int64(1), productCount, "Product should not be deleted")
	})

	t.Run("NEG_005 - Customer Role Tries to Delete Product", func(t *testing.T) {
		// Given: Customer is authenticated
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		// Find any existing product
		var product entity.Product
		err := containers.DB.First(&product).Error
		require.NoError(t, err, "Should find a product")

		productID := product.ID

		// When: Customer tries to delete product
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Delete(t, url)

		// Then: Request is rejected with 403 Forbidden
		assert.Equal(t, http.StatusForbidden, w.Code, "Should return 403 Forbidden")

		// Validate product is NOT deleted
		var productCount int64
		containers.DB.Model(&entity.Product{}).Where("id = ?", productID).Count(&productCount)
		assert.Equal(t, int64(1), productCount, "Product should not be deleted")
	})

	// ============================================================================
	// NEGATIVE SCENARIOS - VALIDATION
	// ============================================================================

	t.Run("NEG_006 - Delete Non-Existent Product", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Non-existent product ID
		productID := uint(99999)

		// When: Seller tries to delete non-existent product
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Delete(t, url)

		// Then: Request is rejected with 404
		assert.Equal(t, http.StatusNotFound, w.Code, "Should return 404 Not Found")
	})

	t.Run("NEG_007 - Delete Product with Invalid Product ID Format", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// When: Seller sends request with non-numeric ID
		url := "/api/products/abc"
		w := client.Delete(t, url)

		// Then: Request is rejected with 400
		assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")
	})

	t.Run("NEG_008 - Delete Product with Negative Product ID", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// When: Seller sends request with negative ID
		url := "/api/products/-5"
		w := client.Delete(t, url)

		// Then: Request is rejected with 400
		assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")
	})

	t.Run("NEG_009 - Delete Product with Zero Product ID", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// When: Seller sends request with zero ID
		url := "/api/products/0"
		w := client.Delete(t, url)

		// Then: Request is rejected with 404 (product with ID 0 doesn't exist)
		assert.Equal(t, http.StatusNotFound, w.Code, "Should return 404 Not Found")
	})

	t.Run(
		"NEG_010 - Delete Product with Product ID Exceeding Maximum Integer",
		func(t *testing.T) {
			// Given: Seller is authenticated
			sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
			client.SetToken(sellerToken)

			// When: Seller sends request with very large number
			url := "/api/products/99999999999999999999"
			w := client.Delete(t, url)

			// Then: Request is rejected with 400
			assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")
		},
	)

	// ============================================================================
	// EDGE CASE SCENARIOS - SECURITY
	// ============================================================================

	t.Run(
		"EDGE_001 - Delete Product with SQL Injection Attempt in Product ID",
		func(t *testing.T) {
			// Given: Seller is authenticated
			sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
			client.SetToken(sellerToken)

			// When: Seller sends request with SQL injection attempt
			url := "/api/products/1; DROP TABLE product--"
			w := client.Delete(t, url)

			// Then: Request is safely rejected
			assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")

			// Verify product table still exists
			var tableExists bool
			err := containers.DB.Raw(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_name = 'product'
			)
		`).Scan(&tableExists).Error
			require.NoError(t, err, "Should be able to query database")
			assert.True(t, tableExists, "Product table should still exist")
		},
	)

	t.Run("EDGE_002 - Delete Product with XSS Attempt in Product ID", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// When: Seller sends request with XSS payload
		url := "/api/products/<script>alert('xss')</script>"
		w := client.Delete(t, url)

		// Then: Request doesn't match route pattern - returns 404
		// Note: This doesn't reach our handler as the router rejects the malformed path
		assert.Equal(t, http.StatusNotFound, w.Code, "Should return 404 Not Found")

		// Verify response doesn't echo the script
		responseBody := w.Body.String()
		assert.NotContains(t, responseBody, "<script>", "Response should not contain script tag")
	})

	t.Run("EDGE_003 - Delete Product with Path Traversal Attempt", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// When: Seller sends request with path traversal
		url := "/api/products/../../etc/passwd"
		w := client.Delete(t, url)

		// Then: Request doesn't match route pattern - returns 404
		// Note: Gin's router handles path traversal and rejects the malformed path
		assert.Equal(t, http.StatusNotFound, w.Code, "Should return 404 Not Found")
	})

	t.Run(
		"EDGE_004 - Delete Product with Unicode Characters in Product ID",
		func(t *testing.T) {
			// Given: Seller is authenticated
			sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
			client.SetToken(sellerToken)

			// When: Seller sends request with Unicode characters
			url := "/api/products/产品123"
			w := client.Delete(t, url)

			// Then: Request is rejected
			assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")
		},
	)

	t.Run(
		"EDGE_005 - Delete Product with Special Characters in Product ID",
		func(t *testing.T) {
			// Given: Seller is authenticated
			sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
			client.SetToken(sellerToken)

			// When: Seller sends request with special characters (URL encoded)
			// Note: Special characters need to be URL encoded: !@#$%^&*() -> %21%40%23%24%25%5E%26%2A%28%29
			url := "/api/products/%21%40%23%24%25%5E%26%2A%28%29"
			w := client.Delete(t, url)

			// Then: Request is rejected as invalid product ID format
			assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")
		},
	)

	// ============================================================================
	// SECURITY SCENARIOS
	// ============================================================================

	t.Run(
		"SEC_006 - Privilege Escalation - Seller Tries to Delete Admin's Product",
		func(t *testing.T) {
			// Given: Seller is authenticated
			sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
			client.SetToken(sellerToken)

			// Find product owned by different seller
			var product entity.Product
			err := containers.DB.Where("seller_id = ?", helpers.Seller2UserID).First(&product).Error
			require.NoError(t, err, "Should find product")

			productID := product.ID

			// When: Seller tries to delete another seller's product
			url := fmt.Sprintf("/api/products/%d", productID)
			w := client.Delete(t, url)

			// Then: Authorization check prevents deletion
			assert.Equal(t, http.StatusForbidden, w.Code, "Should return 403 Forbidden")

			// Validate product is NOT deleted
			var productCount int64
			containers.DB.Model(&entity.Product{}).Where("id = ?", productID).Count(&productCount)
			assert.Equal(t, int64(1), productCount, "Product should not be deleted")
		},
	)

	// ============================================================================
	// DATABASE TRANSACTION SCENARIOS
	// ============================================================================

	t.Run("DB_002 - Concurrent Deletion Attempts", func(t *testing.T) {
		// Given: Product exists
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)

		// Find product for deletion
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.Seller2UserID).
			Order("id DESC").
			First(&product).Error
		require.NoError(t, err, "Should find product")

		productID := product.ID

		// Create two clients with same token
		client1 := helpers.NewAPIClient(server)
		client1.SetToken(seller2Token)

		client2 := helpers.NewAPIClient(server)
		client2.SetToken(seller2Token)

		// When: Both clients try to delete simultaneously
		url := fmt.Sprintf("/api/products/%d", productID)

		// Execute deletions (one should succeed, one should fail)
		done := make(chan *int, 2)
		go func() {
			w1 := client1.Delete(t, url)
			code1 := w1.Code
			done <- &code1
		}()
		go func() {
			w2 := client2.Delete(t, url)
			code2 := w2.Code
			done <- &code2
		}()

		// Collect results
		result1 := <-done
		result2 := <-done

		// Then: At least one should succeed (200)
		// Note: Due to transaction isolation, both might succeed if they read before delete commits
		// The important validation is that the product ends up deleted only once
		results := []int{*result1, *result2}
		assert.Contains(t, results, http.StatusOK, "At least one deletion should succeed")

		// Validate product is deleted exactly once (not duplicated)
		var productCount int64
		containers.DB.Model(&entity.Product{}).Where("id = ?", productID).Count(&productCount)
		assert.Equal(t, int64(0), productCount, "Product should be deleted exactly once")
	})

	t.Run("DB_004 - Foreign Key Constraint Violation Prevention", func(t *testing.T) {
		// Given: Product has relationship with category
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find any remaining product for seller 3
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product")

		productID := product.ID
		categoryID := product.CategoryID

		// When: Product is deleted
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Delete(t, url)

		// Then: Product deletion succeeds (category has ON DELETE RESTRICT, but from category to product)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Validate product is deleted
		var productCount int64
		containers.DB.Model(&entity.Product{}).Where("id = ?", productID).Count(&productCount)
		assert.Equal(t, int64(0), productCount, "Product should be deleted")

		// Validate category still exists (not deleted)
		var categoryCount int64
		containers.DB.Model(&entity.Category{}).Where("id = ?", categoryID).Count(&categoryCount)
		assert.Equal(t, int64(1), categoryCount, "Category should still exist")
	})

	// ============================================================================
	// IDEMPOTENCY SCENARIOS
	// ============================================================================

	t.Run("IDEM_001 - Retry Same Delete Request", func(t *testing.T) {
		// Given: Find any remaining product for this test
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find product owned by seller 3
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).
			Order("id DESC").
			First(&product).Error
		require.NoError(t, err, "Should find product")

		productID := product.ID

		// First deletion
		url := fmt.Sprintf("/api/products/%d", productID)
		w1 := client.Delete(t, url)
		helpers.AssertSuccessResponse(t, w1, http.StatusOK)

		// When: Same DELETE request is sent again
		w2 := client.Delete(t, url)

		// Then: Second request returns 404 (idempotent behavior)
		assert.Equal(t, http.StatusNotFound, w2.Code, "Second deletion should return 404")

		// Validate product is still deleted (not duplicated)
		var productCount int64
		containers.DB.Model(&entity.Product{}).Where("id = ?", productID).Count(&productCount)
		assert.Equal(t, int64(0), productCount, "Product should remain deleted")
	})

	t.Run("IDEM_002 - Network Failure and Retry", func(t *testing.T) {
		// Given: Product exists - use seller 4 to avoid conflicts with other tests
		seller4Token := helpers.Login(t, client, helpers.Seller4Email, helpers.Seller4Password)
		client.SetToken(seller4Token)

		// Find last remaining product owned by seller 4
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.Seller4UserID).
			Order("id DESC").
			First(&product).Error
		require.NoError(t, err, "Should find product")

		productID := product.ID
		url := fmt.Sprintf("/api/products/%d", productID)

		// Simulate: First request completes on server
		w1 := client.Delete(t, url)
		helpers.AssertSuccessResponse(t, w1, http.StatusOK)

		// When: Client retries (simulating network failure scenario)
		w2 := client.Delete(t, url)

		// Then: Retry returns 404 (product already deleted)
		assert.Equal(t, http.StatusNotFound, w2.Code, "Retry should return 404")

		// Validate idempotent behavior
		var productCount int64
		containers.DB.Model(&entity.Product{}).Where("id = ?", productID).Count(&productCount)
		assert.Equal(t, int64(0), productCount, "Product should be deleted only once")
	})
}
