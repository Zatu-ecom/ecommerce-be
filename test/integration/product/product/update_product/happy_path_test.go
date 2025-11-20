package updateproduct

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"ecommerce-be/product/entity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateProductHappyPath(t *testing.T) {
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

	t.Run("UPD_PROD_001 - Seller Successfully Updates Own Product Name", func(t *testing.T) {
		// Given: Seller is authenticated and owns product
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		productID := product.ID
		originalName := product.Name
		originalBrand := product.Brand

		// When: Seller updates product name
		updateRequest := map[string]interface{}{
			"name": "Updated Premium Product Name",
		}
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Put(t, url, updateRequest)

		// Then: Validate response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedProduct := helpers.GetResponseData(t, response, "product")

		// Validate updated fields
		assert.Equal(
			t,
			"Updated Premium Product Name",
			updatedProduct["name"],
			"Name should be updated",
		)
		assert.NotEqual(
			t,
			originalName,
			updatedProduct["name"],
			"Name should be different from original",
		)

		// Validate unchanged fields
		assert.Equal(t, originalBrand, updatedProduct["brand"], "Brand should remain unchanged")

		// Validate in database
		var dbProduct entity.Product
		err = containers.DB.First(&dbProduct, productID).Error
		require.NoError(t, err, "Should find product in database")
		assert.Equal(
			t,
			"Updated Premium Product Name",
			dbProduct.Name,
			"Name should be updated in database",
		)
	})

	t.Run("UPD_PROD_002 - Update Multiple Fields Simultaneously", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).
			Order("id DESC").
			First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		productID := product.ID

		// When: Seller updates multiple fields
		updateRequest := map[string]interface{}{
			"name":             "Multi-Field Updated Product",
			"brand":            "NewBrand",
			"shortDescription": "This is an updated short description",
			"longDescription":  "This is a much longer updated description with detailed information about the product",
			"tags":             []string{"updated", "multiple", "fields"},
		}
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Put(t, url, updateRequest)

		// Then: Validate response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedProduct := helpers.GetResponseData(t, response, "product")

		// Validate all updated fields
		assert.Equal(
			t,
			"Multi-Field Updated Product",
			updatedProduct["name"],
			"Name should be updated",
		)
		assert.Equal(t, "NewBrand", updatedProduct["brand"], "Brand should be updated")
		assert.Equal(
			t,
			"This is an updated short description",
			updatedProduct["shortDescription"],
			"Short description should be updated",
		)
		assert.Equal(
			t,
			"This is a much longer updated description with detailed information about the product",
			updatedProduct["longDescription"],
			"Long description should be updated",
		)

		// Validate tags
		tags, ok := updatedProduct["tags"].([]interface{})
		require.True(t, ok, "Tags should be an array")
		assert.Len(t, tags, 3, "Should have 3 tags")
	})

	t.Run("UPD_PROD_003 - Update Product Category Only", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).
			Order("id DESC").
			First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		productID := product.ID
		originalCategoryID := product.CategoryID
		originalName := product.Name

		// Find a different valid category
		var newCategory entity.Category
		err = containers.DB.Where("id != ?", originalCategoryID).First(&newCategory).Error
		require.NoError(t, err, "Should find different category")

		// When: Seller updates only category (other fields not provided = null)
		updateRequest := map[string]interface{}{
			"categoryId": newCategory.ID,
		}
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Put(t, url, updateRequest)

		// Then: Validate response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedProduct := helpers.GetResponseData(t, response, "product")

		// Validate category is updated
		assert.Equal(
			t,
			float64(newCategory.ID),
			updatedProduct["categoryId"],
			"Category ID should be updated",
		)
		assert.NotEqual(
			t,
			float64(originalCategoryID),
			updatedProduct["categoryId"],
			"Category should be different",
		)

		// Validate name remains unchanged (null = no update)
		assert.Equal(t, originalName, updatedProduct["name"], "Name should remain unchanged")

		// Validate in database
		var dbProduct entity.Product
		err = containers.DB.First(&dbProduct, productID).Error
		require.NoError(t, err, "Should find product in database")
		assert.Equal(
			t,
			newCategory.ID,
			dbProduct.CategoryID,
			"Category should be updated in database",
		)
		assert.Equal(t, originalName, dbProduct.Name, "Name should remain unchanged in database")
	})

	t.Run("UPD_PROD_005 - Update Product Tags Only", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		productID := product.ID
		originalName := product.Name

		// When: Seller updates only tags (other fields not provided = null)
		newTags := []string{"new", "updated", "tags", "for", "search"}
		updateRequest := map[string]interface{}{
			"tags": newTags,
		}
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Put(t, url, updateRequest)

		// Then: Validate response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedProduct := helpers.GetResponseData(t, response, "product")

		// Validate tags are updated
		tags, ok := updatedProduct["tags"].([]interface{})
		require.True(t, ok, "Tags should be an array")
		assert.Len(t, tags, 5, "Should have 5 tags")

		// Validate name remains unchanged (null = no update)
		assert.Equal(t, originalName, updatedProduct["name"], "Name should remain unchanged")

		// Validate in database
		var dbProduct entity.Product
		err = containers.DB.First(&dbProduct, productID).Error
		require.NoError(t, err, "Should find product in database")
		assert.Len(t, dbProduct.Tags, 5, "Should have 5 tags in database")
		assert.Equal(t, originalName, dbProduct.Name, "Name should remain unchanged in database")
	})

	t.Run("UPD_PROD_006 - Admin Updates Any Seller's Product", func(t *testing.T) {
		// Given: Admin is authenticated
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Find a product belonging to a seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.Seller2UserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller 2")

		productID := product.ID
		originalSellerID := product.SellerID

		// When: Admin updates product
		updateRequest := map[string]interface{}{
			"name": "Admin Updated Product Name",
		}
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Put(t, url, updateRequest)

		// Then: Validate response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedProduct := helpers.GetResponseData(t, response, "product")

		// Validate product is updated
		assert.Equal(
			t,
			"Admin Updated Product Name",
			updatedProduct["name"],
			"Name should be updated",
		)
		assert.Equal(
			t,
			float64(originalSellerID),
			updatedProduct["sellerId"],
			"Seller ID should remain unchanged",
		)

		// Validate in database
		var dbProduct entity.Product
		err = containers.DB.First(&dbProduct, productID).Error
		require.NoError(t, err, "Should find product in database")
		assert.Equal(
			t,
			"Admin Updated Product Name",
			dbProduct.Name,
			"Name should be updated in database",
		)
		assert.Equal(t, originalSellerID, dbProduct.SellerID, "Seller ID should remain unchanged")
	})

	t.Run("UPD_PROD_007 - Partial Update - Only Provided Fields Updated", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		productID := product.ID
		originalName := product.Name
		originalShortDescription := product.ShortDescription

		// When: Seller updates only brand (name field NOT provided = null)
		updateRequest := map[string]interface{}{
			"brand": "PartialUpdateBrand",
		}
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Put(t, url, updateRequest)

		// Then: Validate response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedProduct := helpers.GetResponseData(t, response, "product")

		// Validate only brand is updated, name remains unchanged (null = don't update)
		assert.Equal(t, "PartialUpdateBrand", updatedProduct["brand"], "Brand should be updated")
		assert.Equal(t, originalName, updatedProduct["name"], "Name should remain unchanged (null)")
		assert.Equal(
			t,
			originalShortDescription,
			updatedProduct["shortDescription"],
			"Short description should remain unchanged (null)",
		)
	})

	t.Run("UPD_PROD_008 - Update with Empty Brand String (Clear Field)", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller with a brand
		var product entity.Product
		err := containers.DB.Where("seller_id = ? AND brand != ''", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product with brand")

		productID := product.ID
		require.NotEmpty(t, product.Brand, "Product should have a brand initially")

		// When: Seller updates brand to empty string (clear field)
		updateRequest := map[string]interface{}{
			"brand": "", // Empty string = clear the field
		}
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Put(t, url, updateRequest)

		// Then: Validate response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedProduct := helpers.GetResponseData(t, response, "product")

		// Brand should be cleared (empty string, not null)
		assert.Empty(t, updatedProduct["brand"], "Brand should be cleared to empty string")

		// Validate in database
		var dbProduct entity.Product
		err = containers.DB.First(&dbProduct, productID).Error
		require.NoError(t, err, "Should find product in database")
		assert.Empty(t, dbProduct.Brand, "Brand should be empty in database")
	})

	t.Run("UPD_PROD_009 - Update with Empty Tags Array (Clear Tags)", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller with tags
		var product entity.Product
		err := containers.DB.Where("seller_id = ?", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product for seller")

		productID := product.ID

		// When: Seller updates with empty tags array (clear tags)
		updateRequest := map[string]interface{}{
			"tags": []string{}, // Empty array = clear all tags
		}
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Put(t, url, updateRequest)

		// Then: Validate response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedProduct := helpers.GetResponseData(t, response, "product")

		// Tags should be cleared (empty array)
		tags, ok := updatedProduct["tags"].([]interface{})
		require.True(t, ok, "Tags should be an array")
		assert.Empty(t, tags, "Tags should be empty array")

		// Validate in database
		var dbProduct entity.Product
		err = containers.DB.First(&dbProduct, productID).Error
		require.NoError(t, err, "Should find product in database")
		assert.Empty(t, dbProduct.Tags, "Tags should be empty in database")
	})

	t.Run("UPD_PROD_010 - Update with Empty Descriptions (Clear Fields)", func(t *testing.T) {
		// Given: Seller is authenticated
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Find a product owned by seller with descriptions
		var product entity.Product
		err := containers.DB.Where("seller_id = ? AND short_description != ''", helpers.SellerUserID).First(&product).Error
		require.NoError(t, err, "Should find product with descriptions")

		productID := product.ID
		require.NotEmpty(t, product.ShortDescription, "Product should have short description")

		// When: Seller clears descriptions with empty strings
		updateRequest := map[string]interface{}{
			"shortDescription": "", // Clear short description
			"longDescription":  "", // Clear long description
		}
		url := fmt.Sprintf("/api/products/%d", productID)
		w := client.Put(t, url, updateRequest)

		// Then: Validate response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedProduct := helpers.GetResponseData(t, response, "product")

		// Descriptions should be cleared
		assert.Empty(t, updatedProduct["shortDescription"], "Short description should be cleared")
		assert.Empty(t, updatedProduct["longDescription"], "Long description should be cleared")

		// Validate in database
		var dbProduct entity.Product
		err = containers.DB.First(&dbProduct, productID).Error
		require.NoError(t, err, "Should find product in database")
		assert.Empty(t, dbProduct.ShortDescription, "Short description should be empty in database")
		assert.Empty(t, dbProduct.LongDescription, "Long description should be empty in database")
	})
}
