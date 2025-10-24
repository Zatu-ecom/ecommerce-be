package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ecommerce-be/test/helpers"

	"github.com/stretchr/testify/assert"
)

func TestLogin(t *testing.T) {
	// Setup test containers
	containers := helpers.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run migrations and seeds
	containers.RunMigrations(t, "migrations/001_create_user_tables.sql")
	containers.RunSeeds(t, "migrations/seeds/001_seed_user_data.sql")

	// Setup test server
	server := helpers.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Test case: Successful login
	t.Run("successful login", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"email":    "jane.merchant@example.com",
			"password": "seller123",
		}

		bodyBytes, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		assert.True(t, response["success"].(bool))
		assert.Equal(t, "Login successful", response["message"])

		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		token, tokenOk := data["token"].(string)
		assert.True(t, tokenOk)
		assert.NotEmpty(t, token)
	})
}
