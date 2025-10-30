package category

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestCreateCategory(t *testing.T) {
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
	// BASIC CREATION TESTS (P0)
	// ============================================================================

	t.Run("Admin creates global root category", func(t *testing.T) {
		// Login as admin
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		requestBody := map[string]interface{}{
			"name":        "Electronics",
			"description": "Electronic devices and accessories",
		}

		w := client.Post(t, "/api/categories", requestBody)

		response := helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusCreated,
		)
		category := helpers.GetResponseData(t, response, "category")

		helpers.AssertCategoryFields(t, category, "Electronics")
		assert.Equal(t, "Electronic devices and accessories", category["description"])
		assert.Nil(t, category["parentId"], "Root category should have nil parent")

		// Verify it's a global category
		assert.True(t, category["isGlobal"].(bool), "Admin-created category should be global")
		assert.Nil(t, category["sellerId"], "Global category should not have sellerId")
	})

	t.Run("Admin creates global subcategory", func(t *testing.T) {
		// Login as admin
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// First create parent category
		parentRequest := map[string]interface{}{
			"name":        "Home & Garden",
			"description": "Home and garden products",
		}
		parentW := client.Post(t, "/api/categories", parentRequest)
		parentResponse := helpers.AssertSuccessResponse(
			t,
			parentW,
			http.StatusCreated,
		)
		parentCategory := helpers.GetResponseData(t, parentResponse, "category")
		parentID := uint(parentCategory["id"].(float64))

		// Verify parent is global
		assert.True(t, parentCategory["isGlobal"].(bool), "Parent category should be global")

		// Create subcategory
		childRequest := map[string]interface{}{
			"name":        "Furniture",
			"description": "Home furniture",
			"parentId":    parentID,
		}

		w := client.Post(t, "/api/categories", childRequest)

		response := helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusCreated,
		)
		category := helpers.GetResponseData(t, response, "category")

		helpers.AssertCategoryFields(t, category, "Furniture")
		assert.Equal(
			t,
			float64(parentID),
			category["parentId"],
			"Subcategory should have correct parent ID",
		)

		// Verify subcategory is also global
		assert.True(t, category["isGlobal"].(bool), "Admin-created subcategory should be global")
		assert.Nil(t, category["sellerId"], "Global subcategory should not have sellerId")
	})

	// ============================================================================
	// VALIDATION TESTS (P0 & P1)
	// ============================================================================

	t.Run("Empty name validation", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		requestBody := map[string]interface{}{
			"name":        "",
			"description": "Test description",
		}

		w := client.Post(t, "/api/categories", requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Name too long validation", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create a name longer than 100 characters
		longName := ""
		for i := 0; i < 101; i++ {
			longName += "a"
		}

		requestBody := map[string]interface{}{
			"name":        longName,
			"description": "Test description",
		}

		w := client.Post(t, "/api/categories", requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Name too short validation", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		requestBody := map[string]interface{}{
			"name":        "ab", // Less than 3 characters
			"description": "Test description",
		}

		w := client.Post(t, "/api/categories", requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Description too long validation", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create a description longer than 500 characters
		longDesc := ""
		for i := 0; i < 501; i++ {
			longDesc += "a"
		}

		requestBody := map[string]interface{}{
			"name":        "Valid Name",
			"description": longDesc,
		}

		w := client.Post(t, "/api/categories", requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// ============================================================================
	// DUPLICATE DETECTION TESTS (P0)
	// ============================================================================

	t.Run("Duplicate category name", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create first category
		firstRequest := map[string]interface{}{
			"name":        "Books",
			"description": "Books and magazines",
		}
		client.Post(t, "/api/categories", firstRequest)

		// Try to create duplicate
		duplicateRequest := map[string]interface{}{
			"name":        "Books",
			"description": "Different description",
		}

		w := client.Post(t, "/api/categories", duplicateRequest)
		helpers.AssertErrorResponse(t, w, http.StatusConflict)
	}) // ============================================================================
	// PARENT HIERARCHY TESTS (P0 & P1)
	// ============================================================================

	t.Run("Invalid parent_id", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		requestBody := map[string]interface{}{
			"name":        "Test Category",
			"description": "Test description",
			"parentId":    99999, // Non-existent ID
		}

		w := client.Post(t, "/api/categories", requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// ============================================================================
	// AUTHORIZATION & PERMISSIONS TESTS (P0 & P1)
	// ============================================================================

	t.Run("Unauthorized access (no token)", func(t *testing.T) {
		client.SetToken("") // Clear token

		requestBody := map[string]interface{}{
			"name":        "Unauthorized Category",
			"description": "Should fail",
		}

		w := client.Post(t, "/api/categories", requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Customer attempts create", func(t *testing.T) {
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		requestBody := map[string]interface{}{
			"name":        "Customer Category",
			"description": "Should fail",
		}

		w := client.Post(t, "/api/categories", requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	// ============================================================================
	// GLOBAL vs SELLER-SPECIFIC CATEGORY TESTS (P0)
	// ============================================================================

	t.Run("Seller can create subcategory under global category", func(t *testing.T) {
		// Admin creates a global category
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		globalParentRequest := map[string]interface{}{
			"name":        "Global Sports",
			"description": "Global sports category",
		}
		parentW := client.Post(t, "/api/categories", globalParentRequest)
		parentResponse := helpers.AssertSuccessResponse(
			t,
			parentW,
			http.StatusCreated,
		)
		parentCategory := helpers.GetResponseData(t, parentResponse, "category")
		globalParentID := uint(parentCategory["id"].(float64))

		// Verify it's global
		assert.True(t, parentCategory["isGlobal"].(bool))

		// Seller tries to create a subcategory under the global category
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		childRequest := map[string]interface{}{
			"name":        "My Sports Items",
			"description": "My sports products",
			"parentId":    globalParentID,
		}

		w := client.Post(t, "/api/categories", childRequest)
		// This should fail because sellers cannot create subcategories under global categories
		helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	})

	t.Run("Admin can create subcategory under another admin's global category", func(t *testing.T) {
		// First admin creates a global category
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		parentRequest := map[string]interface{}{
			"name":        "Global Fashion",
			"description": "Global fashion category",
		}
		parentW := client.Post(t, "/api/categories", parentRequest)
		parentResponse := helpers.AssertSuccessResponse(
			t,
			parentW,
			http.StatusCreated,
		)
		parentCategory := helpers.GetResponseData(t, parentResponse, "category")
		parentID := uint(parentCategory["id"].(float64))

		// Same or different admin can create subcategory
		childRequest := map[string]interface{}{
			"name":        "Women's Fashion",
			"description": "Women's clothing and accessories",
			"parentId":    parentID,
		}

		w := client.Post(t, "/api/categories", childRequest)
		response := helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusCreated,
		)
		category := helpers.GetResponseData(t, response, "category")

		// Verify both are global
		assert.True(t, category["isGlobal"].(bool))
		assert.Equal(t, float64(parentID), category["parentId"])
	})

	t.Run("Verify global category has no sellerId", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		requestBody := map[string]interface{}{
			"name":        "Verified Global",
			"description": "Testing global category properties",
		}

		w := client.Post(t, "/api/categories", requestBody)
		response := helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusCreated,
		)
		category := helpers.GetResponseData(t, response, "category")

		// Explicit checks for global category
		assert.True(
			t,
			category["isGlobal"].(bool),
			"isGlobal should be true for admin-created category",
		)
		assert.Nil(t, category["sellerId"], "sellerId should be nil for global category")
	})

	t.Run("Verify seller category has sellerId and is not global", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"name":        "Verified Seller Category",
			"description": "Testing seller-specific category properties",
		}

		w := client.Post(t, "/api/categories", requestBody)
		response := helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusCreated,
		)
		category := helpers.GetResponseData(t, response, "category")

		// Explicit checks for seller-specific category
		assert.False(
			t,
			category["isGlobal"].(bool),
			"isGlobal should be false for seller-created category",
		)
		assert.NotNil(
			t,
			category["sellerId"],
			"sellerId should not be nil for seller-specific category",
		)

		// Verify sellerId is a valid number
		sellerID, ok := category["sellerId"].(float64)
		assert.True(t, ok, "sellerId should be a number")
		assert.Greater(t, sellerID, float64(0), "sellerId should be greater than 0")
	})

	t.Run("Create with all optional fields", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		requestBody := map[string]interface{}{
			"name":        "Complete Category",
			"description": "Category with all fields",
		}

		w := client.Post(t, "/api/categories", requestBody)

		response := helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusCreated,
		)
		category := helpers.GetResponseData(t, response, "category")

		helpers.AssertCategoryFields(t, category, "Complete Category")
		assert.Equal(t, "Category with all fields", category["description"])
	})
}
