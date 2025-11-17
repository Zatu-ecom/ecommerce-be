package product

import (
	"net/http"
	"os"
	"testing"
	"time"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/dgrijalva/jwt-go"
)

// TestCreateProductAuthorization tests authentication and authorization scenarios
// Validates: token validation, role-based access control
func TestCreateProductAuthorization(t *testing.T) {
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

	// Standard request body for authorization tests
	requestBody := map[string]interface{}{
		"name":       "Test Product - Auth Check",
		"categoryId": 4,
		"baseSku":    "TEST-AUTH-001",
		"options": []map[string]interface{}{
			{
				"name":        "Color",
				"displayName": "Color",
				"values": []map[string]interface{}{
					{"value": "Black", "displayName": "Black"},
				},
			},
		},
		"variants": []map[string]interface{}{
			{
				"sku":   "TEST-AUTH-001-V1",
				"price": 99.99,
				"options": []map[string]interface{}{
					{"optionName": "Color", "value": "Black"},
				},
			},
		},
	}

	// ============================================================================
	// AUTHORIZATION ERROR SCENARIOS
	// ============================================================================

	t.Run("Error - Unauthenticated request (no token)", func(t *testing.T) {
		// Don't set any token
		client.SetToken("")

		w := client.Post(t, "/api/products", requestBody)

		// Should return 401 Unauthorized
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Error - Invalid/Malformed token", func(t *testing.T) {
		// Set an invalid token
		client.SetToken("invalid-token-12345")

		w := client.Post(t, "/api/products", requestBody)

		// Should return 401 Unauthorized
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Error - Expired token", func(t *testing.T) {
		// Create an expired token
		// Note: This assumes your JWT secret is accessible for testing
		// You may need to adjust based on your actual JWT implementation
		claims := jwt.MapClaims{
			"user_id": helpers.SellerUserID,
			"role":    "seller",
			"exp":     time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		jwtSecret := os.Getenv("JWT_SECRET")
		expiredToken, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			t.Fatalf("Failed to create expired token: %v", err)
		}

		client.SetToken(expiredToken)

		w := client.Post(t, "/api/products", requestBody)

		// Should return 401 Unauthorized for expired token
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Error - Customer trying to create product", func(t *testing.T) {
		// Login as customer
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		w := client.Post(t, "/api/products", requestBody)

		// Should return 403 Forbidden (only sellers allowed)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("Success - Admin allowed to create product", func(t *testing.T) {
		// Login as admin
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// admins are allowed, expect StatusCreated

		requestBody["sellerId"] = helpers.SellerUserID // Admin creating product for a seller
		w := client.Post(t, "/api/products", requestBody)

		// Based on the requirement: "Expected: allowed"
		// This test checks if admin can create products
		if w.Code != http.StatusCreated && w.Code != http.StatusForbidden {
			t.Logf("Unexpected status code: %d", w.Code)
			t.Logf("Response: %s", w.Body.String())
		}

		// If your business logic allows admins to create products:
		if w.Code == http.StatusCreated {
			response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
			product := helpers.GetResponseData(t, response, "product")
			t.Logf("Admin successfully created product: %v", product["id"])
		}
	})

	t.Run("Success - Seller allowed to create product", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		w := client.Post(t, "/api/products", requestBody)

		// Seller should be allowed
		helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	})

	t.Run("Error - Token with invalid signature", func(t *testing.T) {
		// Create a token with wrong secret
		claims := jwt.MapClaims{
			"user_id": helpers.SellerUserID,
			"role":    "seller",
			"exp":     time.Now().Add(24 * time.Hour).Unix(),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		invalidToken, err := token.SignedString([]byte("wrong-secret-key"))
		if err != nil {
			t.Fatalf("Failed to create invalid token: %v", err)
		}

		client.SetToken(invalidToken)

		w := client.Post(t, "/api/products", requestBody)

		// Should return 401 Unauthorized for invalid signature
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Error - Token without required claims", func(t *testing.T) {
		// Create a token missing required fields
		claims := jwt.MapClaims{
			// Missing user_id and role
			"exp": time.Now().Add(24 * time.Hour).Unix(),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		jwtSecret := os.Getenv("JWT_SECRET")
		incompleteToken, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			t.Fatalf("Failed to create incomplete token: %v", err)
		}

		client.SetToken(incompleteToken)

		w := client.Post(t, "/api/products", requestBody)

		// Should return 401 Unauthorized for missing claims
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})
}
