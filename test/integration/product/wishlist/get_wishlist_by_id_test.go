package wishlist

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// TestGetWishlistByID tests the Get Wishlist By ID (GET /api/product/wishlist/:id) API
// This endpoint retrieves a single wishlist with paginated items.
//
// Endpoint: GET /api/product/wishlist/:id
// Query Params: page (default: 1), pageSize (default: 20, max: 100)
// Authentication: Required (Customer Auth only)
//
// Response includes:
//   - Wishlist details (id, name, isDefault, timestamps)
//   - Paginated items array with per-item metadata (wishlistItemId, variantId, addedAt) + full product
//   - Top-level pagination
func TestGetWishlistByID(t *testing.T) {
	// Setup test containers
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run migrations and seeds
	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	// Setup test server
	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Create API client
	client := helpers.NewAPIClient(server)

	// ============================================================================
	// Setup: Create wishlists and add items for testing
	// ============================================================================

	// Login as customer (Alice - user 5)
	aliceToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
	client.SetToken(aliceToken)

	// Create first wishlist (will be default)
	createReq := map[string]any{
		"name": "Alice Tech Wishlist",
	}
	w := client.Post(t, "/api/product/wishlist", createReq)
	response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	aliceWishlist := helpers.GetResponseData(t, response, "wishlist")
	aliceWishlistID := uint(aliceWishlist["id"].(float64))

	// Create second wishlist (non-default)
	createReq = map[string]any{
		"name": "Alice Fashion Wishlist",
	}
	w = client.Post(t, "/api/product/wishlist", createReq)
	response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	aliceWishlist2 := helpers.GetResponseData(t, response, "wishlist")
	aliceWishlistID2 := uint(aliceWishlist2["id"].(float64))

	// Add items to first wishlist using known variant IDs from seed data
	// Seed data has variants: 1 (iPhone 15 Pro), 5 (Samsung S24), 9 (Nike T-Shirt)
	variantIDs := []uint{1, 5, 9}
	addedVariantIDs := make([]uint, 0, len(variantIDs))

	for _, variantID := range variantIDs {
		addReq := map[string]any{
			"variantId": variantID,
		}
		w = client.Post(
			t,
			fmt.Sprintf("/api/product/wishlist/%d/item", aliceWishlistID),
			addReq,
		)
		helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		addedVariantIDs = append(addedVariantIDs, variantID)
	}

	// Login as Michael (user 6) and create wishlist for authorization tests
	michaelToken := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
	client.SetToken(michaelToken)

	createReq = map[string]any{
		"name": "Michael Wishlist",
	}
	w = client.Post(t, "/api/product/wishlist", createReq)
	response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	michaelWishlist := helpers.GetResponseData(t, response, "wishlist")
	michaelWishlistID := uint(michaelWishlist["id"].(float64))

	// ============================================================================
	// Happy Path Scenarios
	// ============================================================================

	t.Run("HP-001: Get wishlist by ID with items", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Get wishlist with items
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))

		// Assert response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		// Validate wishlist fields
		assert.Equal(t, float64(aliceWishlistID), wishlist["id"].(float64), "ID should match")
		assert.Equal(t, "Alice Tech Wishlist", wishlist["name"], "Name should match")
		assert.True(t, wishlist["isDefault"].(bool), "First wishlist should be default")
		assert.NotNil(t, wishlist["createdAt"], "Should have createdAt")
		assert.NotNil(t, wishlist["updatedAt"], "Should have updatedAt")

		// Validate items array
		itemsData, ok := wishlist["items"].([]any)
		assert.True(t, ok, "Items field should be an array")
		assert.GreaterOrEqual(t, len(itemsData), 1, "Should have at least 1 item")

		// Validate each item has required metadata fields
		for _, itemRaw := range itemsData {
			item := itemRaw.(map[string]any)
			assert.NotNil(t, item["wishlistItemId"], "Item should have wishlistItemId")
			assert.NotNil(t, item["variantId"], "Item should have variantId")
			assert.NotNil(t, item["addedAt"], "Item should have addedAt")
			assert.NotNil(t, item["product"], "Item should have product object")

			product, ok := item["product"].(map[string]any)
			assert.True(t, ok, "Product should be a map")
			assert.NotNil(t, product["id"], "Product should have id")
			assert.NotNil(t, product["name"], "Product should have name")
		}

		// Validate pagination at top level
		pagination, ok := wishlist["pagination"].(map[string]any)
		assert.True(t, ok, "Pagination should be at top level")
		assert.NotNil(t, pagination["currentPage"], "Should have currentPage")
		assert.NotNil(t, pagination["totalItems"], "Should have totalItems")
		assert.NotNil(t, pagination["itemsPerPage"], "Should have itemsPerPage")
	})

	t.Run("HP-002: Get empty wishlist (no items)", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Get second wishlist (no items added)
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID2))

		// Assert response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		// Validate empty state
		assert.False(t, wishlist["isDefault"].(bool), "Second wishlist should not be default")

		// Validate empty items array
		itemsData, ok := wishlist["items"].([]any)
		assert.True(t, ok, "Items field should be an array")
		assert.Equal(t, 0, len(itemsData), "Items array should be empty")

		// Validate pagination for empty state at top level
		pagination, ok := wishlist["pagination"].(map[string]any)
		assert.True(t, ok, "Pagination should be at top level")
		assert.Equal(
			t,
			float64(0),
			pagination["totalItems"].(float64),
			"totalItems should be 0",
		)
		assert.Equal(
			t,
			float64(0),
			pagination["totalPages"].(float64),
			"totalPages should be 0",
		)
		assert.False(t, pagination["hasNext"].(bool), "hasNext should be false")
		assert.False(t, pagination["hasPrev"].(bool), "hasPrev should be false")
	})

	t.Run("HP-003: Get default wishlist by ID", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Get first wishlist (default)
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))

		// Assert response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		// Validate default flag
		assert.True(t, wishlist["isDefault"].(bool), "First wishlist should be default")
	})

	t.Run("HP-004: Get non-default wishlist by ID", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Get second wishlist (non-default)
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID2))

		// Assert response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		// Validate non-default flag
		assert.False(t, wishlist["isDefault"].(bool), "Second wishlist should not be default")
	})

	t.Run("HP-005: Get wishlist with custom pagination", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Get wishlist with custom page size
		w := client.Get(
			t,
			fmt.Sprintf("/api/product/wishlist/%d?page=1&pageSize=2", aliceWishlistID),
		)

		// Assert response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		// Validate pagination at top level
		pagination, ok := wishlist["pagination"].(map[string]any)
		assert.True(t, ok, "Pagination should be at top level")
		assert.Equal(
			t,
			float64(1),
			pagination["currentPage"].(float64),
			"currentPage should be 1",
		)
		assert.Equal(
			t,
			float64(2),
			pagination["itemsPerPage"].(float64),
			"itemsPerPage should be 2",
		)
	})

	t.Run("HP-006: Get wishlist with default pagination values", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Get wishlist without pagination params
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))

		// Assert response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		// Validate default pagination at top level
		pagination, ok := wishlist["pagination"].(map[string]any)
		assert.True(t, ok, "Pagination should be at top level")
		assert.Equal(
			t,
			float64(1),
			pagination["currentPage"].(float64),
			"Default currentPage should be 1",
		)
		assert.Equal(
			t,
			float64(20),
			pagination["itemsPerPage"].(float64),
			"Default itemsPerPage should be 20",
		)
	})

	// ============================================================================
	// Negative Scenarios - Authentication
	// ============================================================================

	t.Run("NEG-001: Get wishlist without authentication returns 401", func(t *testing.T) {
		// Clear token
		client.SetToken("")

		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("NEG-002: Get wishlist with invalid token returns 401", func(t *testing.T) {
		// Set invalid token
		client.SetToken("invalid-token-here")

		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	// ============================================================================
	// Negative Scenarios - Authorization
	// ============================================================================

	t.Run("NEG-003: Seller role cannot access wishlist endpoint", func(t *testing.T) {
		// Login as seller
		token := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(token)

		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))

		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("NEG-004: Admin role cannot access wishlist endpoint", func(t *testing.T) {
		// Login as admin
		token := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(token)

		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))

		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("NEG-005: User cannot access another user wishlist", func(t *testing.T) {
		// Login as Michael
		token := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
		client.SetToken(token)

		// Try to access Alice wishlist
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))

		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("NEG-006: User A accessing User B wishlist gets 403 not 404", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Try to access Michael wishlist
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", michaelWishlistID))

		// Should be 403 Forbidden not 404 - user exists but unauthorized
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	// ============================================================================
	// Negative Scenarios - Validation
	// ============================================================================

	t.Run("NEG-007: Get wishlist with non-existent ID returns 404", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Try to get non-existent wishlist
		w := client.Get(t, "/api/product/wishlist/99999")

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("NEG-008: Get wishlist with invalid ID format string", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		w := client.Get(t, "/api/product/wishlist/abc")

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("NEG-009: Get wishlist with invalid ID format negative", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		w := client.Get(t, "/api/product/wishlist/-1")

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("NEG-010: Get wishlist with ID zero returns 404", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		w := client.Get(t, "/api/product/wishlist/0")

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	// ============================================================================
	// Edge Case Scenarios
	// ============================================================================

	t.Run("EDGE-001: Get wishlist with very large ID returns 404", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Large but valid uint64 - should return 404 Not Found (valid ID format, doesn't exist)
		w := client.Get(t, "/api/product/wishlist/9999999999")

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("EDGE-002: Get wishlist with page beyond total pages", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Request page 100 when we only have a few items
		w := client.Get(
			t,
			fmt.Sprintf("/api/product/wishlist/%d?page=100&pageSize=20", aliceWishlistID),
		)

		// Should return 200 with empty items array
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		itemsData, ok := wishlist["items"].([]any)
		assert.True(t, ok, "Items field should be an array")
		assert.Equal(
			t,
			0,
			len(itemsData),
			"Items array should be empty for page beyond total",
		)

		pagination, ok := wishlist["pagination"].(map[string]any)
		assert.True(t, ok, "Pagination should be at top level")
		assert.Equal(
			t,
			float64(100),
			pagination["currentPage"].(float64),
			"currentPage should be 100",
		)
		assert.False(t, pagination["hasNext"].(bool), "hasNext should be false")
	})

	t.Run("EDGE-003: Get wishlist with pageSize exceeding max capped to 100", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Request pageSize 500 max should be 100
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d?pageSize=500", aliceWishlistID))

		// Should succeed with capped pageSize
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		pagination, ok := wishlist["pagination"].(map[string]any)
		assert.True(t, ok, "Pagination should be at top level")
		assert.Equal(
			t,
			float64(100),
			pagination["itemsPerPage"].(float64),
			"pageSize should be capped to 100",
		)
	})

	t.Run("EDGE-004: Get wishlist with pageSize zero uses default", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d?pageSize=0", aliceWishlistID))

		// Should succeed with default pageSize
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		pagination, ok := wishlist["pagination"].(map[string]any)
		assert.True(t, ok, "Pagination should be at top level")
		assert.Equal(
			t,
			float64(20),
			pagination["itemsPerPage"].(float64),
			"pageSize should default to 20",
		)
	})

	t.Run("EDGE-005: Get wishlist with negative page uses default", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d?page=-1", aliceWishlistID))

		// Should succeed with default page
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		pagination, ok := wishlist["pagination"].(map[string]any)
		assert.True(t, ok, "Pagination should be at top level")
		assert.Equal(
			t,
			float64(1),
			pagination["currentPage"].(float64),
			"page should default to 1",
		)
	})

	// ============================================================================
	// Security Scenarios
	// ============================================================================

	t.Run("SEC-001: SQL injection in wishlist ID is prevented", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		w := client.Get(t, "/api/product/wishlist/1;DROP TABLE wishlist;--")

		// Should return 400 Bad Request invalid ID format
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("SEC-002: SQL injection in pagination params is prevented", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// SQL injection attempt in page param - should be sanitized to default value
		w := client.Get(
			t,
			fmt.Sprintf("/api/product/wishlist/%d?page=1;DELETE FROM wishlist;--", aliceWishlistID),
		)

		// Gin's query param parsing will fail to convert to int, should use default page=1
		// and return success (SQL injection is inherently prevented by typed binding)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")
		assert.NotNil(t, wishlist, "Should return wishlist safely")
	})

	t.Run("SEC-003: User isolation - verify no data leakage", func(t *testing.T) {
		// Login as Alice
		aliceToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(aliceToken)

		// Get Alice wishlist
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		aliceData := helpers.GetResponseData(t, response, "wishlist")

		// Verify Alice data is returned
		assert.Equal(t, "Alice Tech Wishlist", aliceData["name"])

		// Now login as Michael
		michaelToken := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
		client.SetToken(michaelToken)

		// Try to access Alice wishlist - should fail
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)

		// Michael should only see his own wishlist
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", michaelWishlistID))
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		michaelData := helpers.GetResponseData(t, response, "wishlist")
		assert.Equal(t, "Michael Wishlist", michaelData["name"])
	})

	// ============================================================================
	// Business Logic Scenarios
	// ============================================================================

	t.Run("BL-001: pagination totalItems matches actual items in wishlist", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Get wishlist
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		itemsData, ok := wishlist["items"].([]any)
		assert.True(t, ok, "Items field should be an array")

		pagination, ok := wishlist["pagination"].(map[string]any)
		assert.True(t, ok, "Pagination should be at top level")

		totalItems := int(pagination["totalItems"].(float64))
		// pagination.totalItems equals total items in DB (not just current page)
		assert.GreaterOrEqual(t, totalItems, len(itemsData), "totalItems should be >= items on page")
	})

	t.Run("BL-002: Items have required fields", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Get wishlist with items
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		itemsData, ok := wishlist["items"].([]any)
		assert.True(t, ok, "Items field should be an array")
		if len(itemsData) > 0 {
			item := itemsData[0].(map[string]any)

			// Verify item has metadata fields
			assert.NotNil(t, item["wishlistItemId"], "Item should have wishlistItemId")
			assert.NotNil(t, item["variantId"], "Item should have variantId")
			assert.NotNil(t, item["addedAt"], "Item should have addedAt")

			// Verify nested product has required fields
			product, ok := item["product"].(map[string]any)
			assert.True(t, ok, "Product should be a map")
			assert.NotNil(t, product["id"], "Product should have id")
			assert.NotNil(t, product["name"], "Product should have name")
		}
	})

	t.Run("BL-003: Wishlist timestamps are valid", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		createdAt := wishlist["createdAt"].(string)
		updatedAt := wishlist["updatedAt"].(string)

		// Verify timestamps are non-empty strings ISO format
		assert.NotEmpty(t, createdAt, "createdAt should not be empty")
		assert.NotEmpty(t, updatedAt, "updatedAt should not be empty")
	})

	// ============================================================================
	// Integration Scenarios
	// ============================================================================

	t.Run("INT-001: Get wishlist after adding item reflects change", func(t *testing.T) {
		// Login as Michael
		token := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
		client.SetToken(token)

		// Get initial item count from pagination
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", michaelWishlistID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		pagination, ok := wishlist["pagination"].(map[string]any)
		assert.True(t, ok, "Pagination should be at top level")
		initialCount := int(pagination["totalItems"].(float64))

		// Add an item to Michael's wishlist using known variant ID from seed data
		addReq := map[string]any{
			"variantId": uint(14), // Running shoes variant
		}
		w = client.Post(
			t,
			fmt.Sprintf("/api/product/wishlist/%d/item", michaelWishlistID),
			addReq,
		)
		helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		// Get wishlist again - should have one more item
		w = client.Get(
			t,
			fmt.Sprintf("/api/product/wishlist/%d", michaelWishlistID),
		)
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist = helpers.GetResponseData(t, response, "wishlist")

		pagination, ok = wishlist["pagination"].(map[string]any)
		assert.True(t, ok, "Pagination should be at top level")
		newCount := int(pagination["totalItems"].(float64))

		assert.Equal(
			t,
			initialCount+1,
			newCount,
			"totalItems should increase by 1 after adding item",
		)
	})
}
