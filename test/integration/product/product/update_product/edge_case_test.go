package updateproduct

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"ecommerce-be/product/entity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateProductEdgeCases(t *testing.T) {
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
	// EDGE CASE SCENARIOS
	// ============================================================================

	t.Run("UPD_PROD_EDGE_001 - Update with Only Whitespace in Name", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		// When: Update with whitespace-only name
		updateRequest := map[string]interface{}{
			"name": "   ",
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: API accepts whitespace (200 OK)
		// Note: Validation checks length but not whitespace trimming
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedProduct := helpers.GetResponseData(t, response, "product")
		assert.Equal(t, "   ", updatedProduct["name"], "Name should be updated to whitespace")
	})

	t.Run("UPD_PROD_EDGE_002 - Update with Name Containing Special Characters", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		// When: Update with special characters
		updateRequest := map[string]interface{}{
			"name": "Product & Co. <Premium> Edition™",
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should accept (200 OK) or reject (400)
		// If accepted, special characters should be properly handled
		if w.Code == http.StatusOK {
			response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
			updatedProduct := helpers.GetResponseData(t, response, "product")
			assert.Contains(t, updatedProduct["name"], "&", "Should contain special characters")
		} else {
			assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 if special chars not allowed")
		}
	})

	t.Run("UPD_PROD_EDGE_003 - Update with Unicode Characters", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		// When: Update with unicode characters
		updateRequest := map[string]interface{}{
			"name":  "高级产品 Premium Product 😀",
			"brand": "العلامة التجارية",
			"tags":  []string{"中文", "العربية", "emoji😀"},
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should accept unicode characters
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedProduct := helpers.GetResponseData(t, response, "product")
		assert.Contains(
			t,
			updatedProduct["name"].(string),
			"高级产品",
			"Should contain Chinese characters",
		)
		assert.Contains(t, updatedProduct["name"].(string), "😀", "Should contain emoji")
	})

	t.Run("UPD_PROD_EDGE_004 - Update with Maximum Valid Length Strings", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		// When: Update with maximum allowed lengths
		updateRequest := map[string]interface{}{
			"name":             strings.Repeat("A", 200),  // max 200
			"brand":            strings.Repeat("B", 100),  // max 100
			"shortDescription": strings.Repeat("S", 500),  // max 500
			"longDescription":  strings.Repeat("L", 5000), // max 5000
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should accept boundary values
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedProduct := helpers.GetResponseData(t, response, "product")
		assert.Len(t, updatedProduct["name"].(string), 200, "Name should be 200 characters")
		assert.Len(t, updatedProduct["brand"].(string), 100, "Brand should be 100 characters")
	})

	t.Run("UPD_PROD_EDGE_006 - Update with SQL Injection Attempt", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		// When: Attempt SQL injection
		updateRequest := map[string]interface{}{
			"name": "Test'; DROP TABLE product; --",
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should treat as literal string (200 OK)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedProduct := helpers.GetResponseData(t, response, "product")
		assert.Equal(
			t,
			"Test'; DROP TABLE product; --",
			updatedProduct["name"],
			"Should store as literal string",
		)

		// Verify product table still exists
		var tableExists bool
		err = containers.DB.Raw(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_name = 'product'
			)
		`).Scan(&tableExists).Error
		require.NoError(t, err, "Should be able to query database")
		assert.True(t, tableExists, "Product table should still exist")
	})

	t.Run("UPD_PROD_EDGE_007 - Update with XSS Payload", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		// When: Attempt XSS attack
		updateRequest := map[string]interface{}{
			"name":             "<script>alert('XSS')</script>",
			"shortDescription": "<img src=x onerror=alert('XSS')>",
			"tags":             []string{"<script>", "alert('xss')", "</script>"},
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should handle XSS payloads safely
		// API may accept (200) and sanitize, or reject (400)
		if w.Code == http.StatusOK {
			response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
			_ = helpers.GetResponseData(t, response, "product")
			// XSS payloads should be stored but not executable
			// Response should not contain unescaped scripts
			responseBody := w.Body.String()
			assert.NotContains(
				t,
				responseBody,
				"<script>alert",
				"Response should not contain executable script",
			)
		} else {
			assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 if XSS not allowed")
		}
	})

	t.Run("UPD_PROD_EDGE_009 - Update Product with Exactly 20 Tags (Boundary)", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		originalName := product.Name

		// When: Update with exactly 20 tags (boundary value)
		tags := make([]string, 20)
		for i := 0; i < 20; i++ {
			tags[i] = fmt.Sprintf("tag%d", i+1)
		}
		updateRequest := map[string]interface{}{
			"tags": tags,
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should accept exactly 20 tags
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedProduct := helpers.GetResponseData(t, response, "product")

		responseTags, ok := updatedProduct["tags"].([]interface{})
		require.True(t, ok, "Tags should be an array")
		assert.Len(t, responseTags, 20, "Should have exactly 20 tags")

		// Name should remain unchanged (not provided = null)
		assert.Equal(t, originalName, updatedProduct["name"], "Name should remain unchanged")
	})

	t.Run("UPD_PROD_EDGE_010 - Update with Leading/Trailing Spaces", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		// When: Update with leading/trailing spaces
		updateRequest := map[string]interface{}{
			"name":  "  Product Name  ",
			"brand": "  BrandName  ",
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should accept (spaces may be trimmed or preserved)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedProduct := helpers.GetResponseData(t, response, "product")

		// Validate that name exists and is not empty after trimming
		name := updatedProduct["name"].(string)
		assert.NotEmpty(t, strings.TrimSpace(name), "Name should not be empty after trimming")
	})

	t.Run("UPD_PROD_EDGE_011 - Null vs Empty Distinction Test", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller with brand and tags
		var product entity.Product
		err := containers.DB.Where("seller_id = ? AND brand != ''", helpers.SellerUserID).
			First(&product).
			Error
		require.NoError(t, err, "Should find product with brand")

		productID := product.ID
		originalName := product.Name
		originalBrand := product.Brand
		require.NotEmpty(t, originalBrand, "Product should have brand initially")

		// When: Send request with brand=null (omitted) and shortDescription="" (empty)
		// brand field NOT in request = null = don't update
		// shortDescription field in request with empty = update to empty
		updateRequest := map[string]interface{}{
			"shortDescription": "", // Empty = clear field
			// brand NOT provided = null = keep existing value
		}
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Put(t, url, updateRequest)

		// Then: Validate response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedProduct := helpers.GetResponseData(t, response, "product")

		// brand should remain unchanged (null = no update)
		assert.Equal(
			t,
			originalBrand,
			updatedProduct["brand"],
			"Brand should remain unchanged (null)",
		)

		// shortDescription should be cleared (empty string = update to empty)
		assert.Empty(
			t,
			updatedProduct["shortDescription"],
			"Short description should be cleared (empty)",
		)

		// name should remain unchanged (not provided)
		assert.Equal(t, originalName, updatedProduct["name"], "Name should remain unchanged")

		// Validate in database
		var dbProduct entity.Product
		err = containers.DB.First(&dbProduct, productID).Error
		require.NoError(t, err, "Should find product in database")
		assert.Equal(t, originalBrand, dbProduct.Brand, "Brand should remain in database")
		assert.Empty(t, dbProduct.ShortDescription, "Short description should be empty in database")
	})

	// ============================================================================
	// SECURITY SCENARIOS
	// ============================================================================

	t.Run("UPD_PROD_SEC_001 - Concurrent Updates to Same Product", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).
			Order("id DESC").
			First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		productID := product.ID

		// Create two clients with same token
		client1 := helpers.NewAPIClient(server)
		client1.SetToken(sellerToken)

		client2 := helpers.NewAPIClient(server)
		client2.SetToken(sellerToken)

		// When: Both clients try to update simultaneously
		url := fmt.Sprintf("/api/products/%d", productID)

		done := make(chan *int, 2)
		go func() {
			updateRequest := map[string]interface{}{"name": "Concurrent Update A"}
			w := client1.Put(t, url, updateRequest)
			code := w.Code
			done <- &code
		}()
		go func() {
			updateRequest := map[string]interface{}{"name": "Concurrent Update B"}
			w := client2.Put(t, url, updateRequest)
			code := w.Code
			done <- &code
		}()

		// Collect results
		result1 := <-done
		result2 := <-done

		// Then: Both should succeed
		assert.Equal(t, http.StatusOK, *result1, "First update should succeed")
		assert.Equal(t, http.StatusOK, *result2, "Second update should succeed")

		// Final state should be consistent
		var dbProduct entity.Product
		err = containers.DB.First(&dbProduct, productID).Error
		require.NoError(t, err, "Should find product in database")
		assert.True(
			t,
			dbProduct.Name == "Concurrent Update A" || dbProduct.Name == "Concurrent Update B",
			"Final name should be one of the concurrent updates",
		)
	})

	t.Run("UPD_PROD_SEC_002 - Mass Assignment Vulnerability Test", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		productID := product.ID
		originalSellerID := product.SellerID
		originalID := product.ID

		// When: Attempt mass assignment of protected fields
		updateRequest := map[string]interface{}{
			"name":      "Updated Name",
			"sellerId":  999,          // Attempt to change ownership
			"id":        888,          // Attempt to change ID
			"createdAt": "2020-01-01", // Attempt to change creation date
		}
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Put(t, url, updateRequest)

		// Then: Should update only allowed fields
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedProduct := helpers.GetResponseData(t, response, "product")

		// Validate name is updated (allowed)
		assert.Equal(t, "Updated Name", updatedProduct["name"], "Name should be updated")

		// Validate protected fields are NOT updated
		assert.Equal(
			t,
			float64(originalSellerID),
			updatedProduct["sellerId"],
			"Seller ID should remain unchanged",
		)
		assert.Equal(
			t,
			float64(originalID),
			updatedProduct["id"],
			"Product ID should remain unchanged",
		)

		// Validate in database
		var dbProduct entity.Product
		err = containers.DB.First(&dbProduct, productID).Error
		require.NoError(t, err, "Should find product in database")
		assert.Equal(
			t,
			originalSellerID,
			dbProduct.SellerID,
			"Seller ID should remain unchanged in DB",
		)
		assert.Equal(t, originalID, dbProduct.ID, "Product ID should remain unchanged in DB")
	})

	// ============================================================================
	// INTEGRATION SCENARIOS
	// ============================================================================

	t.Run("UPD_PROD_INT_001 - Update Product with Existing Variants", func(t *testing.T) {
		// Given: Seller is authenticated and product has variants
		sellerToken := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(sellerToken)

		// Find a product with variants
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.Seller2UserID).
			Joins("JOIN product_variant ON product_variant.product_id = product.id").
			Group("product.id").
			Having("COUNT(product_variant.id) > 0").
			First(&product).Error
		require.NoError(t, err, "Should find product with variants")

		productID := product.ID

		// Count variants before update
		var variantCountBefore int64
		containers.DB.Model(&entity.ProductVariant{}).
			Where("product_id = ?", productID).
			Count(&variantCountBefore)
		require.Greater(t, variantCountBefore, int64(0), "Product should have variants")

		// When: Seller updates product details
		updateRequest := map[string]interface{}{
			"name":             "Updated Product with Variants",
			"shortDescription": "Updated description",
		}
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Put(t, url, updateRequest)

		// Then: Product should be updated and variants remain intact
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedProduct := helpers.GetResponseData(t, response, "product")
		assert.Equal(
			t,
			"Updated Product with Variants",
			updatedProduct["name"],
			"Name should be updated",
		)

		// Validate variants still exist
		var variantCountAfter int64
		containers.DB.Model(&entity.ProductVariant{}).
			Where("product_id = ?", productID).
			Count(&variantCountAfter)
		assert.Equal(
			t,
			variantCountBefore,
			variantCountAfter,
			"Variant count should remain unchanged",
		)
	})
}
