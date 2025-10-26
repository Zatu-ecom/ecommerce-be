package category

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/common/constants"
	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestGetCategoryByID(t *testing.T) {
	// Setup test containers
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run migrations and seeds
	containers.RunAllMigrations(t)
	containers.RunSeeds(t, "migrations/seeds/001_seed_user_data.sql")

	// Setup test server
	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Create API client
	client := helpers.NewAPIClient(server)

	// ============================================================================
	// PUBLIC ACCESS WITH SELLER ID - (P0)
	// ============================================================================

	t.Run("Public access with seller ID - get global category", func(t *testing.T) {
		// Create global category as admin
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		createReq := map[string]interface{}{
			"name":        "Global Test Category",
			"description": "Global category for testing",
		}
		createW := client.Post(t, "/api/categories", createReq)
		createResponse := helpers.AssertSuccessResponse(
			t,
			createW,
			http.StatusCreated,
			"",
		)
		category := helpers.GetResponseData(t, createResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Access as public with X-Seller-ID
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "3") // Jane Merchant's seller ID

		getW := client.Get(t, fmt.Sprintf("/api/categories/%d", categoryID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
			"",
		)

		// Verify response
		returnedCategory := helpers.GetResponseData(t, response, "category")
		assert.Equal(
			t,
			categoryID,
			uint(returnedCategory["id"].(float64)),
			"Should return correct category",
		)
		assert.True(t, returnedCategory["isGlobal"].(bool), "Should be global category")
	})

	t.Run("Public access with seller ID - get own seller-specific category", func(t *testing.T) {
		// Create seller-specific category
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		createReq := map[string]interface{}{
			"name":        "Seller Specific GetByID Test",
			"description": "Seller's own category",
		}
		createW := client.Post(t, "/api/categories", createReq)
		createResponse := helpers.AssertSuccessResponse(
			t,
			createW,
			http.StatusCreated,
			"",
		)
		category := helpers.GetResponseData(t, createResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Access as public with same seller ID
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "3") // Same seller (Jane Merchant)

		getW := client.Get(t, fmt.Sprintf("/api/categories/%d", categoryID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
			"",
		)

		// Verify response
		returnedCategory := helpers.GetResponseData(t, response, "category")
		assert.Equal(
			t,
			categoryID,
			uint(returnedCategory["id"].(float64)),
			"Should return correct category",
		)
		assert.False(t, returnedCategory["isGlobal"].(bool), "Should be seller-specific category")
	})

	t.Run("Public access with seller ID - cannot get other seller's category", func(t *testing.T) {
		// Create seller-specific category for seller 1 (seller_id=3)
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		createReq := map[string]interface{}{
			"name":        "Seller1 Private Category",
			"description": "Seller 1's private category",
		}
		createW := client.Post(t, "/api/categories", createReq)
		createResponse := helpers.AssertSuccessResponse(
			t,
			createW,
			http.StatusCreated,
			"",
		)
		category := helpers.GetResponseData(t, createResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Try to access with different seller ID (seller 2 = seller_id=4)
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "4") // Seller 2's ID

		getW := client.Get(t, fmt.Sprintf("/api/categories/%d", categoryID))
		helpers.AssertErrorResponse(
			t,
			getW,
			http.StatusNotFound,
			"",
		)
	})

	t.Run("Public access without seller ID fails", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "")

		getW := client.Get(t, "/api/categories/1")
		helpers.AssertErrorResponse(
			t,
			getW,
			http.StatusBadRequest,
			"",
		)
	})

	// ============================================================================
	// AUTHENTICATED ACCESS - (P0)
	// ============================================================================

	t.Run("Seller authenticated - get global category", func(t *testing.T) {
		// Create global category as admin
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		createReq := map[string]interface{}{
			"name":        "Global For Seller Auth Test",
			"description": "Global category",
		}
		createW := client.Post(t, "/api/categories", createReq)
		createResponse := helpers.AssertSuccessResponse(
			t,
			createW,
			http.StatusCreated,
			"",
		)
		category := helpers.GetResponseData(t, createResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Access as authenticated seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)
		client.SetHeader(constants.SELLER_ID_HEADER, "") // Clear header

		getW := client.Get(t, fmt.Sprintf("/api/categories/%d", categoryID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
			"",
		)

		// Verify response
		returnedCategory := helpers.GetResponseData(t, response, "category")
		assert.Equal(
			t,
			categoryID,
			uint(returnedCategory["id"].(float64)),
			"Should return correct category",
		)
		assert.True(t, returnedCategory["isGlobal"].(bool), "Should be global category")
	})

	t.Run("Seller authenticated - get own seller-specific category", func(t *testing.T) {
		// Create seller-specific category
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		createReq := map[string]interface{}{
			"name":        "Own Category Auth Test",
			"description": "Seller's own category with auth",
		}
		createW := client.Post(t, "/api/categories", createReq)
		createResponse := helpers.AssertSuccessResponse(
			t,
			createW,
			http.StatusCreated,
			"",
		)
		category := helpers.GetResponseData(t, createResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Get the same category
		getW := client.Get(t, fmt.Sprintf("/api/categories/%d", categoryID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
			"",
		)

		// Verify response
		returnedCategory := helpers.GetResponseData(t, response, "category")
		assert.Equal(
			t,
			categoryID,
			uint(returnedCategory["id"].(float64)),
			"Should return correct category",
		)
		assert.False(t, returnedCategory["isGlobal"].(bool), "Should be seller-specific")
	})

	t.Run("Seller authenticated - cannot get other seller's category", func(t *testing.T) {
		// Create category as seller 1 (Jane Merchant - seller_id=3)
		seller1Token := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(seller1Token)

		createReq := map[string]interface{}{
			"name":        "Seller1 Auth Private Category",
			"description": "Seller 1's private category",
		}
		createW := client.Post(t, "/api/categories", createReq)
		createResponse := helpers.AssertSuccessResponse(
			t,
			createW,
			http.StatusCreated,
			"",
		)
		category := helpers.GetResponseData(t, createResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Try to access as Seller 2 (Bob Store - seller_id=4)
		seller2Token := helpers.Login(t, client, "bob.store@example.com", "seller123")
		client.SetToken(seller2Token)

		getW := client.Get(t, fmt.Sprintf("/api/categories/%d", categoryID))
		helpers.AssertErrorResponse(
			t,
			getW,
			http.StatusNotFound,
			"",
		)
	})

	t.Run("Admin can get any category", func(t *testing.T) {
		// Create seller-specific category
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		createReq := map[string]interface{}{
			"name":        "Seller Category For Admin Test",
			"description": "Seller's category that admin will access",
		}
		createW := client.Post(t, "/api/categories", createReq)
		createResponse := helpers.AssertSuccessResponse(
			t,
			createW,
			http.StatusCreated,
			"",
		)
		category := helpers.GetResponseData(t, createResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Access as admin
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		getW := client.Get(t, fmt.Sprintf("/api/categories/%d", categoryID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
			"",
		)

		// Verify admin can access seller-specific category
		returnedCategory := helpers.GetResponseData(t, response, "category")
		assert.Equal(
			t,
			categoryID,
			uint(returnedCategory["id"].(float64)),
			"",
		)
		assert.False(t, returnedCategory["isGlobal"].(bool), "Should be seller-specific")
	})

	// ============================================================================
	// ERROR CASES - (P0)
	// ============================================================================

	t.Run("Get non-existent category", func(t *testing.T) {
		// Use public access with seller ID
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "3")

		getW := client.Get(t, "/api/categories/99999")
		helpers.AssertErrorResponse(
			t,
			getW,
			http.StatusNotFound,
			"",
		)
	})

	t.Run("Invalid category ID format", func(t *testing.T) {
		// Use public access with seller ID
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "3")

		getW := client.Get(t, "/api/categories/invalid")
		helpers.AssertErrorResponse(
			t,
			getW,
			http.StatusBadRequest,
			"",
		)
	})

	// ============================================================================
	// RESPONSE VALIDATION - (P1)
	// ============================================================================

	t.Run("Verify category response fields", func(t *testing.T) {
		// Create category as admin
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		createReq := map[string]interface{}{
			"name":        "Response Fields Validation Category",
			"description": "Category for field validation",
		}
		createW := client.Post(t, "/api/categories", createReq)
		createResponse := helpers.AssertSuccessResponse(
			t,
			createW,
			http.StatusCreated,
			"",
		)
		category := helpers.GetResponseData(t, createResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Get category
		getW := client.Get(t, fmt.Sprintf("/api/categories/%d", categoryID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
			"",
		)

		// Verify all required fields
		returnedCategory := helpers.GetResponseData(t, response, "category")

		assert.NotNil(t, returnedCategory["id"], "Should have id")
		assert.NotNil(t, returnedCategory["name"], "Should have name")
		assert.NotNil(t, returnedCategory["description"], "Should have description")
		assert.NotNil(t, returnedCategory["isGlobal"], "Should have isGlobal")
		assert.NotNil(t, returnedCategory["createdAt"], "Should have createdAt")
		assert.NotNil(t, returnedCategory["updatedAt"], "Should have updatedAt")

		// Verify timestamp format (UTC with Z suffix)
		createdAt := returnedCategory["createdAt"].(string)
		updatedAt := returnedCategory["updatedAt"].(string)
		assert.Contains(t, createdAt, "Z", "createdAt should be in UTC format")
		assert.Contains(t, updatedAt, "Z", "updatedAt should be in UTC format")

		// Verify data types
		assert.IsType(t, float64(0), returnedCategory["id"], "id should be number")
		assert.IsType(t, "", returnedCategory["name"], "name should be string")
		assert.IsType(t, "", returnedCategory["description"], "description should be string")
		assert.IsType(t, true, returnedCategory["isGlobal"], "isGlobal should be boolean")
	})

	t.Run("Category with parent reference", func(t *testing.T) {
		// Create parent and child as admin
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create parent
		parentReq := map[string]interface{}{
			"name":        "Parent For Reference Test",
			"description": "Parent category",
		}
		parentW := client.Post(t, "/api/categories", parentReq)
		parentResponse := helpers.AssertSuccessResponse(
			t,
			parentW,
			http.StatusCreated,
			"",
		)
		parentCategory := helpers.GetResponseData(t, parentResponse, "category")
		parentID := uint(parentCategory["id"].(float64))

		// Create child
		childReq := map[string]interface{}{
			"name":        "Child For Reference Test",
			"description": "Child category",
			"parentId":    parentID,
		}
		childW := client.Post(t, "/api/categories", childReq)
		childResponse := helpers.AssertSuccessResponse(
			t,
			childW,
			http.StatusCreated,
			"",
		)
		childCategory := helpers.GetResponseData(t, childResponse, "category")
		childID := uint(childCategory["id"].(float64))

		// Get child category and verify parentId
		getW := client.Get(t, fmt.Sprintf("/api/categories/%d", childID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
			"",
		)

		returnedCategory := helpers.GetResponseData(t, response, "category")
		assert.NotNil(t, returnedCategory["parentId"], "Child should have parentId")
		assert.Equal(
			t,
			float64(parentID),
			returnedCategory["parentId"].(float64),
			"parentId should match",
		)
	})
}
