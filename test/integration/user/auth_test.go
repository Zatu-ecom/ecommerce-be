package user

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestAuth(t *testing.T) {
	// Setup test containers
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run migrations and seeds
	containers.RunMigrations(t, "migrations/001_create_user_tables.sql")
	containers.RunSeeds(t, "migrations/seeds/001_seed_user_data.sql")

	// Setup test server
	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Create API client
	client := helpers.NewAPIClient(server)

	// Test case: Successful login
	t.Run("successful login", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"email":    "jane.merchant@example.com",
			"password": "seller123",
		}

		w := client.Post(t, "/api/auth/login", requestBody)

		assert.Equal(t, http.StatusOK, w.Code)

		response := helpers.ParseResponse(t, w.Body)

		assert.True(t, response["success"].(bool))
		assert.Equal(t, "Login successful", response["message"])

		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		token, tokenOk := data["token"].(string)
		assert.True(t, tokenOk)
		assert.NotEmpty(t, token)
	})

	// Test case: Invalid credentials
	t.Run("invalid credentials", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"email":    "jane.merchant@example.com",
			"password": "wrongpassword",
		}

		w := client.Post(t, "/api/auth/login", requestBody)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		response := helpers.ParseResponse(t, w.Body)
		assert.False(t, response["success"].(bool))
		assert.Equal(t, "Invalid email or password", response["message"])
	})

	// Test case: Missing email
	t.Run("missing email", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"password": "seller123",
		}

		w := client.Post(t, "/api/auth/login", requestBody)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		response := helpers.ParseResponse(t, w.Body)
		assert.False(t, response["success"].(bool))
	})

	// Test case: Missing password
	t.Run("missing password", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"email": "jane.merchant@example.com",
		}

		w := client.Post(t, "/api/auth/login", requestBody)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		response := helpers.ParseResponse(t, w.Body)
		assert.False(t, response["success"].(bool))
	})

	// Test case: Invalid email format
	t.Run("invalid email format", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"email":    "not-an-email",
			"password": "seller123",
		}

		w := client.Post(t, "/api/auth/login", requestBody)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		response := helpers.ParseResponse(t, w.Body)
		assert.False(t, response["success"].(bool))
	})

	// Test case: Non-existent user
	t.Run("non-existent user", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"email":    "nonexistent@example.com",
			"password": "somepassword",
		}

		w := client.Post(t, "/api/auth/login", requestBody)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		response := helpers.ParseResponse(t, w.Body)
		assert.False(t, response["success"].(bool))
		assert.Equal(t, "Invalid email or password", response["message"])
	})
}
