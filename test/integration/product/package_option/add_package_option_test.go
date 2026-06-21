package package_option

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestAddPackageOption(t *testing.T) {
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
	client := helpers.NewAPIClient(server)

	t.Run("Seller adds valid package option to own product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5
		requestBody := map[string]any{
			"name":        "Holiday Bundle",
			"description": "Seasonal t-shirt bundle",
			"price":       89.99,
			"quantity":    2,
		}

		url := fmt.Sprintf("/api/product/%d/package-option", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		packageOption := helpers.GetResponseData(t, response, "packageOption")

		assert.NotNil(t, packageOption["id"])
		assert.Equal(t, float64(productID), packageOption["productId"])
		assert.Equal(t, "Holiday Bundle", packageOption["name"])
		assert.Equal(t, "Seasonal t-shirt bundle", packageOption["description"])
		assert.Equal(t, 89.99, packageOption["price"])
		assert.Equal(t, float64(2), packageOption["quantity"])
		assert.NotNil(t, packageOption["createdAt"])
		assert.NotNil(t, packageOption["updatedAt"])
	})

	t.Run("Validation - price must be positive", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":     "Invalid Bundle",
			"price":    0,
			"quantity": 1,
		}

		url := fmt.Sprintf("/api/product/%d/package-option", 5)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Validation - quantity must be positive", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":     "Invalid Bundle",
			"price":    10.0,
			"quantity": 0,
		}

		url := fmt.Sprintf("/api/product/%d/package-option", 5)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Add without authentication", func(t *testing.T) {
		client.SetToken("")

		requestBody := map[string]any{
			"name":     "Unauthorized Bundle",
			"price":    10.0,
			"quantity": 1,
		}

		url := fmt.Sprintf("/api/product/%d/package-option", 5)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Seller cannot add package option to another seller's product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":     "Wrong Seller Bundle",
			"price":    10.0,
			"quantity": 1,
		}

		url := fmt.Sprintf("/api/product/%d/package-option", 1)
		w := client.Post(t, url, requestBody)

		assert.True(
			t,
			w.Code == http.StatusForbidden || w.Code == http.StatusNotFound,
			"Expected 403 or 404, got %d",
			w.Code,
		)
	})

	t.Run("Add to non-existent product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":     "Missing Product Bundle",
			"price":    10.0,
			"quantity": 1,
		}

		url := fmt.Sprintf("/api/product/%d/package-option", 99999)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})
}
