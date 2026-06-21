package package_option

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestUpdatePackageOption(t *testing.T) {
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
	client := helpers.NewAPIClient(server)

	t.Run("Seller updates own product package option", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5
		packageOptionID := 3

		requestBody := map[string]any{
			"name":        "T-Shirt 5-Pack",
			"description": "Buy 5 t-shirts and save more",
			"price":       110.0,
			"quantity":    5,
		}

		url := fmt.Sprintf("/api/product/%d/package-option/%d", productID, packageOptionID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		packageOption := helpers.GetResponseData(t, response, "packageOption")

		assert.Equal(t, float64(packageOptionID), packageOption["id"])
		assert.Equal(t, float64(productID), packageOption["productId"])
		assert.Equal(t, "T-Shirt 5-Pack", packageOption["name"])
		assert.Equal(t, 110.0, packageOption["price"])
		assert.Equal(t, float64(5), packageOption["quantity"])
	})

	t.Run("Update non-existent package option", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":     "Missing",
			"price":    10.0,
			"quantity": 1,
		}

		url := fmt.Sprintf("/api/product/%d/package-option/%d", 5, 99999)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Update package option that belongs to another product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":     "Wrong Product",
			"price":    10.0,
			"quantity": 1,
		}

		url := fmt.Sprintf("/api/product/%d/package-option/%d", 6, 3)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Update without authentication", func(t *testing.T) {
		client.SetToken("")

		requestBody := map[string]any{
			"name":     "Unauthorized",
			"price":    10.0,
			"quantity": 1,
		}

		url := fmt.Sprintf("/api/product/%d/package-option/%d", 5, 3)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})
}
