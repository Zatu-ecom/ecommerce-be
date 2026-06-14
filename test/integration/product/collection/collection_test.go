package collection

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCollectionTest(t *testing.T) *helpers.APIClient {
	t.Helper()
	containers := setup.SetupTestContainers(t)
	t.Cleanup(func() { containers.Cleanup(t) })

	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
	return helpers.NewAPIClient(server)
}

func loginSeller(t *testing.T, client *helpers.APIClient) {
	t.Helper()
	token := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
	client.SetToken(token)
}

func createCollection(t *testing.T, client *helpers.APIClient, name string) uint {
	t.Helper()
	body := map[string]any{
		"name":        name,
		"description": fmt.Sprintf("%s description", name),
	}
	w := client.Post(t, "/api/product/collection", body)
	response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	collection := helpers.GetResponseData(t, response, "collection")
	id, ok := collection["id"].(float64)
	require.True(t, ok)
	return uint(id)
}

func TestCollectionCRUD(t *testing.T) {
	client := setupCollectionTest(t)

	t.Run("Seller creates collection", func(t *testing.T) {
		loginSeller(t, client)

		body := map[string]any{
			"name":        "Summer Sale",
			"description": "Hot summer deals",
		}
		w := client.Post(t, "/api/product/collection", body)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		collection := helpers.GetResponseData(t, response, "collection")

		assert.Equal(t, "Summer Sale", collection["name"])
		assert.Equal(t, "summer-sale", collection["slug"])
		assert.Equal(t, float64(helpers.SellerUserID), collection["sellerId"])
		assert.True(t, collection["isActive"].(bool))
	})

	t.Run("Seller lists own collections", func(t *testing.T) {
		loginSeller(t, client)
		createCollection(t, client, "Best Sellers")

		w := client.Get(t, "/api/product/collection")
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]any)
		collections := data["collections"].([]any)
		assert.NotEmpty(t, collections)
	})

	t.Run("Seller gets collection by ID", func(t *testing.T) {
		loginSeller(t, client)
		collectionID := createCollection(t, client, "New Arrivals")

		w := client.Get(t, fmt.Sprintf("/api/product/collection/%d", collectionID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		collection := helpers.GetResponseData(t, response, "collection")

		assert.Equal(t, float64(collectionID), collection["id"])
		assert.Equal(t, "New Arrivals", collection["name"])
	})

	t.Run("Seller updates collection", func(t *testing.T) {
		loginSeller(t, client)
		collectionID := createCollection(t, client, "Flash Deals")

		body := map[string]any{
			"name":        "Flash Deals Updated",
			"description": "Updated description",
			"isActive":    false,
		}
		w := client.Put(t, fmt.Sprintf("/api/product/collection/%d", collectionID), body)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		collection := helpers.GetResponseData(t, response, "collection")

		assert.Equal(t, "Flash Deals Updated", collection["name"])
		assert.False(t, collection["isActive"].(bool))
	})

	t.Run("Seller deletes collection", func(t *testing.T) {
		loginSeller(t, client)
		collectionID := createCollection(t, client, "To Delete")

		w := client.Delete(t, fmt.Sprintf("/api/product/collection/%d", collectionID), nil)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		w = client.Get(t, fmt.Sprintf("/api/product/collection/%d", collectionID))
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Unauthorized seller cannot update another seller's collection", func(t *testing.T) {
		loginSeller(t, client)
		collectionID := createCollection(t, client, "Seller Three Collection")

		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		body := map[string]any{
			"name":        "Hijacked",
			"description": "Should fail",
		}
		w := client.Put(t, fmt.Sprintf("/api/product/collection/%d", collectionID), body)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})
}

func TestCollectionProducts(t *testing.T) {
	client := setupCollectionTest(t)

	t.Run("Bulk add products to collection", func(t *testing.T) {
		loginSeller(t, client)
		collectionID := createCollection(t, client, "Fashion Picks")

		// Products 5, 6, 7 belong to seller 3 (Jane Merchant)
		body := map[string]any{
			"productIds": []uint{5, 6, 7},
		}
		w := client.Post(t, fmt.Sprintf("/api/product/collection/%d/product", collectionID), body)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		w = client.Get(t, fmt.Sprintf("/api/product/collection/%d/product", collectionID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]any)
		products := data["products"].([]any)
		assert.Len(t, products, 3)

		pagination := data["pagination"].(map[string]any)
		assert.Equal(t, float64(3), pagination["totalItems"])
	})

	t.Run("Re-adding existing products is idempotent", func(t *testing.T) {
		loginSeller(t, client)
		collectionID := createCollection(t, client, "Idempotent Add")

		body := map[string]any{"productIds": []uint{5, 6}}
		w := client.Post(t, fmt.Sprintf("/api/product/collection/%d/product", collectionID), body)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		w = client.Post(t, fmt.Sprintf("/api/product/collection/%d/product", collectionID), body)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		w = client.Get(t, fmt.Sprintf("/api/product/collection/%d/product", collectionID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]any)
		products := data["products"].([]any)
		assert.Len(t, products, 2)
	})

	t.Run("Cannot add products from another seller", func(t *testing.T) {
		loginSeller(t, client)
		collectionID := createCollection(t, client, "Cross Seller Test")

		// Product 1 belongs to seller 2
		body := map[string]any{"productIds": []uint{1}}
		w := client.Post(t, fmt.Sprintf("/api/product/collection/%d/product", collectionID), body)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Bulk remove products from collection", func(t *testing.T) {
		loginSeller(t, client)
		collectionID := createCollection(t, client, "Remove Test")

		addBody := map[string]any{"productIds": []uint{5, 6, 7}}
		w := client.Post(t, fmt.Sprintf("/api/product/collection/%d/product", collectionID), addBody)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		removeBody := map[string]any{"productIds": []uint{5, 7}}
		w = client.Delete(t, fmt.Sprintf("/api/product/collection/%d/product", collectionID), removeBody)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		w = client.Get(t, fmt.Sprintf("/api/product/collection/%d/product", collectionID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]any)
		products := data["products"].([]any)
		assert.Len(t, products, 1)
		assert.Equal(t, float64(6), products[0].(map[string]any)["productId"])
	})

	t.Run("Reorder products in collection", func(t *testing.T) {
		loginSeller(t, client)
		collectionID := createCollection(t, client, "Reorder Test")

		addBody := map[string]any{"productIds": []uint{5, 6, 7}}
		w := client.Post(t, fmt.Sprintf("/api/product/collection/%d/product", collectionID), addBody)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		reorderBody := map[string]any{
			"items": []map[string]any{
				{"productId": 7, "position": 0},
				{"productId": 5, "position": 1},
				{"productId": 6, "position": 2},
			},
		}
		w = client.Put(
			t,
			fmt.Sprintf("/api/product/collection/%d/product/reorder", collectionID),
			reorderBody,
		)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		w = client.Get(t, fmt.Sprintf("/api/product/collection/%d/product", collectionID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]any)
		products := data["products"].([]any)

		assert.Equal(t, float64(7), products[0].(map[string]any)["productId"])
		assert.Equal(t, float64(0), products[0].(map[string]any)["position"])
		assert.Equal(t, float64(5), products[1].(map[string]any)["productId"])
	})

	t.Run("Cannot reorder product not in collection", func(t *testing.T) {
		loginSeller(t, client)
		collectionID := createCollection(t, client, "Invalid Reorder")

		addBody := map[string]any{"productIds": []uint{5}}
		w := client.Post(t, fmt.Sprintf("/api/product/collection/%d/product", collectionID), addBody)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		reorderBody := map[string]any{
			"items": []map[string]any{
				{"productId": 6, "position": 0},
			},
		}
		w = client.Put(
			t,
			fmt.Sprintf("/api/product/collection/%d/product/reorder", collectionID),
			reorderBody,
		)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})
}
