package package_option

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestBulkUpdatePackageOptions(t *testing.T) {
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
	client := helpers.NewAPIClient(server)

	createPackageOption := func(productID int, name string, price float64, quantity int) map[string]any {
		requestBody := map[string]any{
			"name":        name,
			"description": "Bulk test bundle",
			"price":       price,
			"quantity":    quantity,
		}

		url := fmt.Sprintf("/api/product/%d/package-option", productID)
		w := client.Post(t, url, requestBody)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		return helpers.GetResponseData(t, response, "packageOption")
	}

	t.Run("Bulk update multiple package options successfully", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5
		opt1 := createPackageOption(productID, "Bundle A", 50.0, 2)
		opt2 := createPackageOption(productID, "Bundle B", 60.0, 3)

		opt1ID := int(opt1["id"].(float64))
		opt2ID := int(opt2["id"].(float64))

		bulkUpdateBody := map[string]any{
			"packageOptions": []map[string]any{
				{
					"packageOptionId": opt1ID,
					"name":            "Updated Bundle A",
					"description":     "Updated A",
					"price":           55.0,
					"quantity":        4,
				},
				{
					"packageOptionId": opt2ID,
					"name":            "Updated Bundle B",
					"description":     "Updated B",
					"price":           65.0,
					"quantity":        5,
				},
			},
		}

		url := fmt.Sprintf("/api/product/%d/package-option/bulk", productID)
		w := client.Put(t, url, bulkUpdateBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		result := helpers.GetResponseData(t, response, "result")

		assert.Equal(t, float64(2), result["updatedCount"])

		packageOptions := result["packageOptions"].([]any)
		assert.Len(t, packageOptions, 2)

		for _, opt := range packageOptions {
			optMap := opt.(map[string]any)
			optID := int(optMap["id"].(float64))

			switch optID {
			case opt1ID:
				assert.Equal(t, "Updated Bundle A", optMap["name"])
				assert.Equal(t, 55.0, optMap["price"])
				assert.Equal(t, float64(4), optMap["quantity"])
			case opt2ID:
				assert.Equal(t, "Updated Bundle B", optMap["name"])
				assert.Equal(t, 65.0, optMap["price"])
				assert.Equal(t, float64(5), optMap["quantity"])
			}
		}
	})

	t.Run("Bulk update skips package options from another product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 6
		opt := createPackageOption(productID, "Dress Bundle", 40.0, 1)
		optID := int(opt["id"].(float64))

		bulkUpdateBody := map[string]any{
			"packageOptions": []map[string]any{
				{
					"packageOptionId": optID,
					"name":            "Wrong Product Update",
					"price":           99.0,
					"quantity":        1,
				},
			},
		}

		url := fmt.Sprintf("/api/product/%d/package-option/bulk", 5)
		w := client.Put(t, url, bulkUpdateBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		result := helpers.GetResponseData(t, response, "result")
		assert.Equal(t, float64(0), result["updatedCount"])
	})

	t.Run("Bulk update without authentication", func(t *testing.T) {
		client.SetToken("")

		bulkUpdateBody := map[string]any{
			"packageOptions": []map[string]any{
				{
					"packageOptionId": 3,
					"name":            "Unauthorized",
					"price":           10.0,
					"quantity":        1,
				},
			},
		}

		url := fmt.Sprintf("/api/product/%d/package-option/bulk", 5)
		w := client.Put(t, url, bulkUpdateBody)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})
}
