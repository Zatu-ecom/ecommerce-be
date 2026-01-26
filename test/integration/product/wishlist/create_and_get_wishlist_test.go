package wishlist

import (
	"net/http"
	"strings"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// TestCreateAndGetWishlist tests the Create Wishlist (POST) and Get All Wishlists (GET) APIs
//
// These are tested together to verify:
// 	1. Flow testing - create wishlist, then verify it appears in list
// 	2. Data consistency - created wishlist is correctly returned in GET response
// 	3. Default wishlist logic - first wishlist becomes default, subsequent don't
// 	4. ItemCount verification - new wishlists start with itemCount = 0
//
// Endpoints:
// 	- POST /api/product/wishlist - Create Wishlist
// 	- GET /api/product/wishlist - Get All Wishlists
//
// Authentication: Required (Customer Auth only)
func TestCreateAndGetWishlist(t *testing.T) {
	// Setup test containers
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run migrations and core seeds only (no mock wishlist data)
	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	// Run user seeds for test users
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	// Setup test server
	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Create API client
	client := helpers.NewAPIClient(server)

	// ============================================================================
	// Happy Path Scenarios
	// ============================================================================

	t.Run("HP-001: Create first wishlist becomes default", func(t *testing.T) {
		// Login as customer (Alice - user 5)
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Create first wishlist
		requestBody := map[string]interface{}{
			"name": "My First Wishlist",
		}

		w := client.Post(t, "/api/product/wishlist", requestBody)

		// Assert response
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		// Validate wishlist fields
		assert.NotNil(t, wishlist["id"], "Wishlist should have id")
		assert.Equal(t, "My First Wishlist", wishlist["name"], "Name should match")
		assert.True(t, wishlist["isDefault"].(bool), "First wishlist should be default")
		assert.Equal(
			t,
			float64(0),
			wishlist["itemCount"].(float64),
			"New wishlist should have 0 items",
		)
		assert.NotNil(t, wishlist["createdAt"], "Should have createdAt")
		assert.NotNil(t, wishlist["updatedAt"], "Should have updatedAt")
	})

	t.Run("HP-002: Create additional wishlist is not default", func(t *testing.T) {
		// Login as customer (Alice - user 5)
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Create second wishlist
		requestBody := map[string]interface{}{
			"name": "Gift Ideas",
		}

		w := client.Post(t, "/api/product/wishlist", requestBody)

		// Assert response
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		// Validate - second wishlist should NOT be default
		assert.NotNil(t, wishlist["id"], "Wishlist should have id")
		assert.Equal(t, "Gift Ideas", wishlist["name"], "Name should match")
		assert.False(t, wishlist["isDefault"].(bool), "Second wishlist should NOT be default")
		assert.Equal(
			t,
			float64(0),
			wishlist["itemCount"].(float64),
			"New wishlist should have 0 items",
		)
	})

	t.Run("HP-003: Get all wishlists returns created wishlists", func(t *testing.T) {
		// Login as customer (Alice - user 5)
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Get all wishlists
		w := client.Get(t, "/api/product/wishlist")

		// Assert response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		wishlists := data["wishlists"].([]interface{})

		// Should have at least 2 wishlists from previous tests
		assert.GreaterOrEqual(t, len(wishlists), 2, "Should have at least 2 wishlists")

		// Verify exactly one is default
		defaultCount := 0
		for _, wl := range wishlists {
			wishlist := wl.(map[string]interface{})
			if wishlist["isDefault"].(bool) {
				defaultCount++
			}
			// All wishlists should have required fields
			assert.NotNil(t, wishlist["id"], "Wishlist should have id")
			assert.NotNil(t, wishlist["name"], "Wishlist should have name")
			assert.NotNil(t, wishlist["itemCount"], "Wishlist should have itemCount")
			assert.NotNil(t, wishlist["createdAt"], "Wishlist should have createdAt")
			assert.NotNil(t, wishlist["updatedAt"], "Wishlist should have updatedAt")
		}
		assert.Equal(t, 1, defaultCount, "Exactly one wishlist should be default")
	})

	t.Run("HP-004: Flow test - Create then Get", func(t *testing.T) {
		// Login as different customer (Michael - user 6) to start fresh
		token := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
		client.SetToken(token)

		// First, get initial count
		w := client.Get(t, "/api/product/wishlist")
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		initialWishlists := data["wishlists"].([]interface{})
		initialCount := len(initialWishlists)

		// Create first wishlist for Michael
		requestBody := map[string]interface{}{
			"name": "Tech Gadgets",
		}
		w = client.Post(t, "/api/product/wishlist", requestBody)
		response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		techWishlist := helpers.GetResponseData(t, response, "wishlist")
		techWishlistID := techWishlist["id"]

		// Create second wishlist
		requestBody = map[string]interface{}{
			"name": "Fashion Items",
		}
		w = client.Post(t, "/api/product/wishlist", requestBody)
		response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		fashionWishlist := helpers.GetResponseData(t, response, "wishlist")
		fashionWishlistID := fashionWishlist["id"]

		// Get all wishlists and verify both appear
		w = client.Get(t, "/api/product/wishlist")
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data = response["data"].(map[string]interface{})
		wishlists := data["wishlists"].([]interface{})

		// Should have 2 more than initial
		assert.Equal(t, initialCount+2, len(wishlists), "Should have 2 more wishlists")

		// Find both created wishlists
		foundTech := false
		foundFashion := false
		for _, wl := range wishlists {
			wishlist := wl.(map[string]interface{})
			if wishlist["id"] == techWishlistID {
				foundTech = true
				assert.Equal(t, "Tech Gadgets", wishlist["name"])
			}
			if wishlist["id"] == fashionWishlistID {
				foundFashion = true
				assert.Equal(t, "Fashion Items", wishlist["name"])
			}
		}
		assert.True(t, foundTech, "Tech Gadgets wishlist should be in list")
		assert.True(t, foundFashion, "Fashion Items wishlist should be in list")
	})

	// ============================================================================
	// Negative Scenarios - Authentication
	// ============================================================================

	t.Run("NEG-001: Create wishlist without authentication returns 401", func(t *testing.T) {
		// Clear token (unauthenticated)
		client.SetToken("")

		requestBody := map[string]interface{}{
			"name": "Unauthorized Wishlist",
		}

		w := client.Post(t, "/api/product/wishlist", requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("NEG-002: Get wishlists without authentication returns 401", func(t *testing.T) {
		// Clear token (unauthenticated)
		client.SetToken("")

		w := client.Get(t, "/api/product/wishlist")

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("NEG-003: Seller can access wishlist endpoint", func(t *testing.T) {
		// Login as seller
		token := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(token)

		requestBody := map[string]interface{}{
			"name": "Seller Wishlist",
		}

		w := client.Post(t, "/api/product/wishlist", requestBody)

		helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	})

	t.Run("NEG-004: Admin can access wishlist endpoint", func(t *testing.T) {
		// Login as admin
		token := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(token)

		w := client.Get(t, "/api/product/wishlist")

		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	// ============================================================================
	// Negative Scenarios - Validation
	// ============================================================================

	t.Run("NEG-005: Create wishlist with missing name returns 400", func(t *testing.T) {
		// Login as customer
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Empty body
		requestBody := map[string]interface{}{}

		w := client.Post(t, "/api/product/wishlist", requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("NEG-006: Create wishlist with empty name returns 400", func(t *testing.T) {
		// Login as customer
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		requestBody := map[string]interface{}{
			"name": "",
		}

		w := client.Post(t, "/api/product/wishlist", requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run(
		"NEG-007: Create wishlist with name exceeding max length returns 400",
		func(t *testing.T) {
			// Login as customer
			token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
			client.SetToken(token)

			// Create name with 101 characters (max is 100)
			longName := strings.Repeat("a", 101)
			requestBody := map[string]interface{}{
				"name": longName,
			}

			w := client.Post(t, "/api/product/wishlist", requestBody)

			helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
		},
	)

	// ============================================================================
	// Edge Case Scenarios
	// ============================================================================

	t.Run("EDGE-001: Create wishlist with minimum valid name (1 char)", func(t *testing.T) {
		// Login as customer
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		requestBody := map[string]interface{}{
			"name": "A",
		}

		w := client.Post(t, "/api/product/wishlist", requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		wishlist := helpers.GetResponseData(t, response, "wishlist")
		assert.Equal(t, "A", wishlist["name"], "Single character name should be accepted")
	})

	t.Run("EDGE-002: Create wishlist with maximum valid name (100 chars)", func(t *testing.T) {
		// Login as customer
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Create name with exactly 100 characters
		maxName := strings.Repeat("b", 100)
		requestBody := map[string]interface{}{
			"name": maxName,
		}

		w := client.Post(t, "/api/product/wishlist", requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		wishlist := helpers.GetResponseData(t, response, "wishlist")
		assert.Equal(t, maxName, wishlist["name"], "100 character name should be accepted")
	})

	t.Run("EDGE-003: Create wishlist with Unicode characters", func(t *testing.T) {
		// Login as customer
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		requestBody := map[string]interface{}{
			"name": "🎁 Gift Ideas 礼物 هدايا",
		}

		w := client.Post(t, "/api/product/wishlist", requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		wishlist := helpers.GetResponseData(t, response, "wishlist")
		assert.Equal(
			t,
			"🎁 Gift Ideas 礼物 هدايا",
			wishlist["name"],
			"Unicode name should be preserved",
		)
	})

	t.Run("EDGE-004: Create wishlist with special characters", func(t *testing.T) {
		// Login as customer
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		requestBody := map[string]interface{}{
			"name": "Gift's & Ideas! (2024)",
		}

		w := client.Post(t, "/api/product/wishlist", requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		wishlist := helpers.GetResponseData(t, response, "wishlist")
		assert.Equal(
			t,
			"Gift's & Ideas! (2024)",
			wishlist["name"],
			"Special characters should be preserved",
		)
	})

	// ============================================================================
	// Security Scenarios
	// ============================================================================

	t.Run("SEC-001: SQL injection in wishlist name is safely stored", func(t *testing.T) {
		// Login as customer
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		sqlInjection := "'; DROP TABLE wishlist; --"
		requestBody := map[string]interface{}{
			"name": sqlInjection,
		}

		w := client.Post(t, "/api/product/wishlist", requestBody)

		// Should succeed and store the literal string
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		wishlist := helpers.GetResponseData(t, response, "wishlist")
		assert.Equal(
			t,
			sqlInjection,
			wishlist["name"],
			"SQL injection should be stored as literal string",
		)
	})

	t.Run("SEC-002: XSS payload in wishlist name is safely stored", func(t *testing.T) {
		// Login as customer
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		xssPayload := "<script>alert('XSS')</script>"
		requestBody := map[string]interface{}{
			"name": xssPayload,
		}

		w := client.Post(t, "/api/product/wishlist", requestBody)

		// Should succeed and store (escaping is frontend responsibility for display)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		wishlist := helpers.GetResponseData(t, response, "wishlist")
		assert.NotNil(t, wishlist["name"], "Wishlist should be created")
	})

	t.Run("SEC-003: User A cannot see User B's wishlists", func(t *testing.T) {
		// Login as customer A (Alice)
		tokenAlice := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(tokenAlice)

		// Create a wishlist for Alice
		requestBody := map[string]interface{}{
			"name": "Alice Private Wishlist",
		}
		w := client.Post(t, "/api/product/wishlist", requestBody)
		helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		// Get Alice's wishlists and count them
		w = client.Get(t, "/api/product/wishlist")
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		aliceWishlists := data["wishlists"].([]interface{})
		aliceCount := len(aliceWishlists)

		// Now login as customer B (Sarah - user 7)
		tokenSarah := helpers.Login(t, client, helpers.Customer3Email, helpers.Customer3Password)
		client.SetToken(tokenSarah)

		// Get Sarah's wishlists
		w = client.Get(t, "/api/product/wishlist")
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data = response["data"].(map[string]interface{})
		sarahWishlists := data["wishlists"].([]interface{})

		// Sarah should NOT see Alice's wishlists
		for _, wl := range sarahWishlists {
			wishlist := wl.(map[string]interface{})
			assert.NotEqual(t, "Alice Private Wishlist", wishlist["name"],
				"Sarah should not see Alice's wishlist")
		}

		// Counts should be different (Alice has more)
		assert.NotEqual(t, aliceCount, len(sarahWishlists),
			"Different users should have different wishlist counts")
	})

	// ============================================================================
	// Business Logic Scenarios
	// ============================================================================

	t.Run("BL-001: New wishlist always has itemCount of 0", func(t *testing.T) {
		// Login as customer
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		requestBody := map[string]interface{}{
			"name": "Empty Wishlist Test",
		}

		w := client.Post(t, "/api/product/wishlist", requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		wishlist := helpers.GetResponseData(t, response, "wishlist")
		assert.Equal(t, float64(0), wishlist["itemCount"].(float64),
			"New wishlist must have itemCount of 0")
	})

	t.Run("BL-002: Only one wishlist can be default at a time", func(t *testing.T) {
		// Login as new customer (Sarah - user 7) to start fresh
		token := helpers.Login(t, client, helpers.Customer3Email, helpers.Customer3Password)
		client.SetToken(token)

		// Create multiple wishlists
		for i := 1; i <= 3; i++ {
			requestBody := map[string]interface{}{
				"name": "Sarah's Wishlist " + string(rune('0'+i)),
			}
			w := client.Post(t, "/api/product/wishlist", requestBody)
			helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		}

		// Get all wishlists
		w := client.Get(t, "/api/product/wishlist")
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		wishlists := data["wishlists"].([]interface{})

		// Count defaults
		defaultCount := 0
		for _, wl := range wishlists {
			wishlist := wl.(map[string]interface{})
			if wishlist["isDefault"].(bool) {
				defaultCount++
			}
		}
		assert.Equal(t, 1, defaultCount, "Exactly one wishlist should be default")
	})
}

// TestGetEmptyWishlists tests getting wishlists for a fresh user with no wishlists
func TestGetEmptyWishlists(t *testing.T) {
	// Setup test containers
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run migrations and seeds
	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")

	// Setup test server
	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Create API client
	client := helpers.NewAPIClient(server)

	t.Run("HP-003: Get all wishlists returns empty array for fresh user", func(t *testing.T) {
		// Login as customer (Sarah - user 7, fresh user with no wishlists)
		token := helpers.Login(t, client, helpers.Customer3Email, helpers.Customer3Password)
		client.SetToken(token)

		// Get all wishlists
		w := client.Get(t, "/api/product/wishlist")

		// Assert response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		wishlists := data["wishlists"].([]interface{})

		// Should be empty array (not null)
		assert.NotNil(t, wishlists, "Wishlists should not be null")
		assert.Equal(t, 0, len(wishlists), "Fresh user should have 0 wishlists")
	})
}
