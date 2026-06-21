package user

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestListUsers(t *testing.T) {
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	containers.RunAllMigrations(t)
	containers.RunAllSeeds(t)

	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
	client := helpers.NewAPIClient(server)

	t.Run("Success - Seller lists users scoped to own seller", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(token)

		w := client.Get(t, "/api/user?page=1&pageSize=20")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]any)
		users := data["users"].([]any)
		pagination := data["pagination"].(map[string]any)

		assert.NotEmpty(t, users, "Seller should see users in their scope")
		assert.Equal(t, float64(1), pagination["currentPage"])
		assert.Equal(t, float64(20), pagination["itemsPerPage"])

		for _, u := range users {
			user := u.(map[string]any)
			assert.Equal(t, float64(3), user["sellerId"], "Seller should only see seller 3 users")
		}
	})

	t.Run("Success - Admin lists all users", func(t *testing.T) {
		adminClient := helpers.NewAPIClient(server)
		token := helpers.Login(t, adminClient, helpers.AdminEmail, helpers.AdminPassword)
		adminClient.SetToken(token)

		w := adminClient.Get(t, "/api/user?page=1&pageSize=50")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		users := response["data"].(map[string]any)["users"].([]any)

		assert.GreaterOrEqual(t, len(users), 7, "Admin should see users across sellers")

		sellerIDs := make(map[float64]bool)
		for _, u := range users {
			user := u.(map[string]any)
			if sellerID, ok := user["sellerId"].(float64); ok && sellerID > 0 {
				sellerIDs[sellerID] = true
			}
		}
		assert.True(t, sellerIDs[2], "Admin should see seller 2 users")
		assert.True(t, sellerIDs[3], "Admin should see seller 3 users")
	})

	t.Run("Filter - By roleNames CUSTOMER for seller scope", func(t *testing.T) {
		sellerClient := helpers.NewAPIClient(server)
		token := helpers.Login(t, sellerClient, helpers.Seller2Email, helpers.Seller2Password)
		sellerClient.SetToken(token)

		w := sellerClient.Get(t, "/api/user?roleNames=CUSTOMER")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		users := response["data"].(map[string]any)["users"].([]any)

		assert.NotEmpty(t, users, "Should return customers for seller 2")
		for _, u := range users {
			user := u.(map[string]any)
			role := user["role"].(map[string]any)
			assert.Equal(t, "CUSTOMER", role["name"])
			assert.Equal(t, float64(2), user["sellerId"])
		}
	})

	t.Run("Filter - Admin by sellerIds", func(t *testing.T) {
		adminClient := helpers.NewAPIClient(server)
		token := helpers.Login(t, adminClient, helpers.AdminEmail, helpers.AdminPassword)
		adminClient.SetToken(token)

		w := adminClient.Get(t, "/api/user?sellerIds=3&roleNames=CUSTOMER")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		users := response["data"].(map[string]any)["users"].([]any)

		assert.Len(t, users, 1, "Seller 3 has one customer in seed data")
		user := users[0].(map[string]any)
		assert.Equal(t, "michael.s@example.com", user["email"])
	})

	t.Run("Filter - Search by name partial match", func(t *testing.T) {
		adminClient := helpers.NewAPIClient(server)
		token := helpers.Login(t, adminClient, helpers.AdminEmail, helpers.AdminPassword)
		adminClient.SetToken(token)

		w := adminClient.Get(t, "/api/user?name=Alice")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		users := response["data"].(map[string]any)["users"].([]any)

		assert.NotEmpty(t, users, "Should find Alice by name")
		found := false
		for _, u := range users {
			user := u.(map[string]any)
			if user["email"] == "alice.j@example.com" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should include Alice Johnson")
	})

	t.Run("Unauthorized - Customer cannot list users", func(t *testing.T) {
		customerClient := helpers.NewAPIClient(server)
		token := helpers.Login(t, customerClient, helpers.CustomerEmail, helpers.CustomerPassword)
		customerClient.SetToken(token)

		w := customerClient.Get(t, "/api/user")

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}
