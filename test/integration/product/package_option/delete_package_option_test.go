package package_option

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestDeletePackageOption(t *testing.T) {
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
	client := helpers.NewAPIClient(server)

	createPackageOption := func(productID int) map[string]any {
		requestBody := map[string]any{
			"name":        "Temp Bundle",
			"description": "To be deleted",
			"price":       49.99,
			"quantity":    2,
		}

		url := fmt.Sprintf("/api/product/%d/package-option", productID)
		w := client.Post(t, url, requestBody)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		return helpers.GetResponseData(t, response, "packageOption")
	}

	t.Run("Seller deletes package option from own product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5
		packageOption := createPackageOption(productID)
		packageOptionID := int(packageOption["id"].(float64))

		url := fmt.Sprintf("/api/product/%d/package-option/%d", productID, packageOptionID)
		w := client.Delete(t, url)

		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("Delete already deleted package option returns 404", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 6
		packageOption := createPackageOption(productID)
		packageOptionID := int(packageOption["id"].(float64))

		url := fmt.Sprintf("/api/product/%d/package-option/%d", productID, packageOptionID)
		w1 := client.Delete(t, url)
		helpers.AssertSuccessResponse(t, w1, http.StatusOK)

		w2 := client.Delete(t, url)
		helpers.AssertErrorResponse(t, w2, http.StatusNotFound)
	})

	t.Run("Delete package option that belongs to another product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5
		packageOption := createPackageOption(productID)
		packageOptionID := int(packageOption["id"].(float64))

		url := fmt.Sprintf("/api/product/%d/package-option/%d", 6, packageOptionID)
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Delete without authentication", func(t *testing.T) {
		client.SetToken("")

		url := fmt.Sprintf("/api/product/%d/package-option/%d", 5, 3)
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Admin can delete package option from any product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		packageOption := createPackageOption(5)
		packageOptionID := int(packageOption["id"].(float64))

		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		url := fmt.Sprintf("/api/product/%d/package-option/%d", 5, packageOptionID)
		w := client.Delete(t, url)

		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("Seller cannot delete package option on another seller's product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		url := fmt.Sprintf("/api/product/%d/package-option/%d", 1, 1)
		w := client.Delete(t, url)

		assert.True(
			t,
			w.Code == http.StatusForbidden || w.Code == http.StatusNotFound,
			"Expected 403 or 404, got %d",
			w.Code,
		)
	})
}
