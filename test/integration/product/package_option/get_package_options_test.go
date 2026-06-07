package package_option

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestGetPackageOptions(t *testing.T) {
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
	client := helpers.NewAPIClient(server)

	t.Run("Get package options for product with seed data", func(t *testing.T) {
		productID := 5
		client.SetHeader("X-Seller-ID", "3")

		url := fmt.Sprintf("/api/product/%d/package-option", productID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := helpers.GetResponseData(t, response, "packageOptions")
		options := data["packageOptions"].([]any)

		assert.GreaterOrEqual(t, len(options), 1)

		first := options[0].(map[string]any)
		assert.NotNil(t, first["id"])
		assert.Equal(t, "T-Shirt 3-Pack", first["name"])
		assert.Equal(t, 75.0, first["price"])
		assert.Equal(t, float64(3), first["quantity"])
	})

	t.Run("Public access requires X-Seller-ID header", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		url := fmt.Sprintf("/api/product/%d/package-option", 5)
		w := client.Get(t, url)

		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("Get package options for non-existent product", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "3")

		url := fmt.Sprintf("/api/product/%d/package-option", 99999)
		w := client.Get(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Get package options with invalid product ID", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "3")

		url := "/api/product/invalid/package-option"
		w := client.Get(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})
}
