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

func TestUpdateProductNegativePath(t *testing.T) {
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
	// NEGATIVE SCENARIOS - AUTHENTICATION
	// ============================================================================

	t.Run("UPD_PROD_NEG_001 - Update Without Authentication", func(t *testing.T) {
		// Given: No authentication token
		client.SetToken("")

		// Find any product
		var product entity.Product
		err := containers.DB.First(&product).Error
		require.NoError(t, err, "Should find a product")

		// When: Request sent without token
		updateRequest := map[string]interface{}{
			"name": "Unauthorized Update",
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should return 401
		assert.Equal(t, http.StatusUnauthorized, w.Code, "Should return 401 Unauthorized")

		// Validate product is NOT updated
		var dbProduct entity.Product
		err = containers.DB.First(&dbProduct, product.ID).Error
		require.NoError(t, err, "Should find product in database")
		assert.Equal(t, product.Name, dbProduct.Name, "Product name should not be updated")
	})

	t.Run("UPD_PROD_NEG_002 - Update with Invalid Token", func(t *testing.T) {
		// Given: Invalid JWT token
		client.SetToken("invalid.jwt.token")

		// Find any product
		var product entity.Product
		err := containers.DB.First(&product).Error
		require.NoError(t, err, "Should find a product")

		// When: Request sent with invalid token
		updateRequest := map[string]interface{}{
			"name": "Invalid Token Update",
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should return 401
		assert.Equal(t, http.StatusUnauthorized, w.Code, "Should return 401 Unauthorized")

		// Validate product is NOT updated
		var dbProduct entity.Product
		err = containers.DB.First(&dbProduct, product.ID).Error
		require.NoError(t, err, "Should find product in database")
		assert.Equal(t, product.Name, dbProduct.Name, "Product name should not be updated")
	})

	t.Run("UPD_PROD_NEG_003 - Update with Expired Token", func(t *testing.T) {
		// Note: Testing expired tokens requires generating an expired token
		// This test validates the token validation logic
		// In a real scenario, you would generate a token with past expiry

		// Given: Malformed token (simulating expired)
		client.SetToken("Bearer expiredtoken123")

		// Find any product
		var product entity.Product
		err := containers.DB.First(&product).Error
		require.NoError(t, err, "Should find a product")

		// When: Request sent with malformed/expired token
		updateRequest := map[string]interface{}{
			"name": "Expired Token Update",
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should return 401
		assert.Equal(t, http.StatusUnauthorized, w.Code, "Should return 401 Unauthorized")
	})

	// ============================================================================
	// NEGATIVE SCENARIOS - AUTHORIZATION
	// ============================================================================

	t.Run("UPD_PROD_NEG_004 - Seller Updates Another Seller's Product", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find product owned by different seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.Seller2UserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller 2")

		productID := product.ID
		originalName := product.Name

		// When: Seller tries to update another seller's product
		updateRequest := map[string]interface{}{
			"name": "Unauthorized Update Attempt",
		}
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Put(t, url, updateRequest)

		// Then: Should return 403 Forbidden
		assert.Equal(t, http.StatusForbidden, w.Code, "Should return 403 Forbidden")

		// Validate product is NOT updated
		var dbProduct entity.Product
		err = containers.DB.First(&dbProduct, productID).Error
		require.NoError(t, err, "Should find product in database")
		assert.Equal(t, originalName, dbProduct.Name, "Product name should not be updated")
	})

	t.Run("UPD_PROD_NEG_005 - Customer Role Tries to Update Product", func(t *testing.T) {
		// Given: Customer is authenticated
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		// Find any product
		var product entity.Product
		err := containers.DB.First(&product).Error
		require.NoError(t, err, "Should find a product")

		productID := product.ID

		// When: Customer tries to update product
		updateRequest := map[string]interface{}{
			"name": "Customer Update Attempt",
		}
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Put(t, url, updateRequest)

		// Then: Should return 403 Forbidden
		assert.Equal(t, http.StatusForbidden, w.Code, "Should return 403 Forbidden")

		// Validate product is NOT updated
		var dbProduct entity.Product
		err = containers.DB.First(&dbProduct, productID).Error
		require.NoError(t, err, "Should find product in database")
		assert.Equal(t, product.Name, dbProduct.Name, "Product name should not be updated")
	})

	// ============================================================================
	// NEGATIVE SCENARIOS - VALIDATION
	// ============================================================================

	t.Run("UPD_PROD_NEG_006 - Update Non-Existent Product", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// When: Seller tries to update non-existent product
		updateRequest := map[string]interface{}{
			"name": "Non-Existent Product",
		}
		url := "/api/products/99999"
		w := client.Put(t, url, updateRequest)

		// Then: Should return 404 Not Found
		assert.Equal(t, http.StatusNotFound, w.Code, "Should return 404 Not Found")
	})

	t.Run("UPD_PROD_NEG_007 - Update with Invalid Product ID Format", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// When: Request sent with non-numeric ID
		updateRequest := map[string]interface{}{
			"name": "Invalid ID Format",
		}
		url := "/api/products/abc"
		w := client.Put(t, url, updateRequest)

		// Then: Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")
	})

	t.Run("UPD_PROD_NEG_008 - Update with Negative Product ID", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// When: Request sent with negative ID
		updateRequest := map[string]interface{}{
			"name": "Negative ID",
		}
		url := "/api/products/-5"
		w := client.Put(t, url, updateRequest)

		// Then: Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")
	})

	t.Run("UPD_PROD_NEG_009 - Update with Zero Product ID", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// When: Request sent with zero ID
		updateRequest := map[string]interface{}{
			"name": "Zero ID",
		}
		url := "/api/products/0"
		w := client.Put(t, url, updateRequest)

		// Then: Should return 404 Not Found (product with ID 0 doesn't exist)
		assert.Equal(t, http.StatusNotFound, w.Code, "Should return 404 Not Found")
	})

	t.Run("UPD_PROD_NEG_010 - Update with Empty Request Body (All Fields Null)", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		originalName := product.Name

		// When: Request sent with empty body (all fields null = no fields provided)
		updateRequest := map[string]interface{}{}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should return 400 Bad Request (at least one field required)
		// API requires at least one field to be provided for update
		assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")

		// Validate product is NOT updated in database
		var dbProduct entity.Product
		err = containers.DB.First(&dbProduct, product.ID).Error
		require.NoError(t, err, "Should find product in database")
		assert.Equal(t, originalName, dbProduct.Name, "Product name should not be updated")
	})

	t.Run("UPD_PROD_NEG_011 - Update with Wrong Data Type", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		// When: Request sent with wrong data type (number for name)
		updateRequest := map[string]interface{}{
			"name": 12345, // Should be string
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")
	})

	t.Run("UPD_PROD_NEG_012 - Update Name with Too Short String (Min 3 chars)", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		// When: Update with name < 3 characters (validation still applies when field is provided)
		updateRequest := map[string]interface{}{
			"name": "AB",
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should return 400 Bad Request (min=3 validation)
		assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")
	})

	t.Run("UPD_PROD_NEG_013 - Update Name with Too Long String", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		// When: Update with name > 200 characters
		longName := strings.Repeat("A", 201)
		updateRequest := map[string]interface{}{
			"name": longName,
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")
	})

	t.Run("UPD_PROD_NEG_014 - Update Brand with Too Long String", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		// When: Update with brand > 100 characters
		longBrand := strings.Repeat("B", 101)
		updateRequest := map[string]interface{}{
			"brand": longBrand,
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")
	})

	t.Run("UPD_PROD_NEG_015 - Update Short Description Exceeding Max Length", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		// When: Update with shortDescription > 500 characters
		longDesc := strings.Repeat("D", 501)
		updateRequest := map[string]interface{}{
			"shortDescription": longDesc,
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")
	})

	t.Run("UPD_PROD_NEG_016 - Update Long Description Exceeding Max Length", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		// When: Update with longDescription > 5000 characters
		longDesc := strings.Repeat("L", 5001)
		updateRequest := map[string]interface{}{
			"longDescription": longDesc,
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")
	})

	t.Run("UPD_PROD_NEG_017 - Update Tags Exceeding Maximum Count", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		// When: Update with > 20 tags
		tags := make([]string, 21)
		for i := 0; i < 21; i++ {
			tags[i] = fmt.Sprintf("tag%d", i+1)
		}
		updateRequest := map[string]interface{}{
			"tags": tags,
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")
	})

	t.Run("UPD_PROD_NEG_018 - Update with Invalid Category ID", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		// When: Update with non-existent category ID
		updateRequest := map[string]interface{}{
			"categoryId": 99999,
		}
		url := fmt.Sprintf("/api/products/%d", product.ID)
		w := client.Put(t, url, updateRequest)

		// Then: Should return 400 or 404
		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound,
			"Should return 400 Bad Request or 404 Not Found")
	})
}
