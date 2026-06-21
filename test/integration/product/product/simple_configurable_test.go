package product

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/product/entity"
	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleConfigurableProducts(t *testing.T) {
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
	client := helpers.NewAPIClient(server)

	loginSeller := func() {
		token := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(token)
	}

	createSimpleProduct := func(name, sku string, price float64) int {
		t.Helper()
		loginSeller()
		body := map[string]any{
			"name":       name,
			"categoryId": 4,
			"baseSku":    sku,
			"price":      price,
		}
		resp := helpers.AssertSuccessResponse(t, client.Post(t, "/api/product", body), http.StatusCreated)
		product := helpers.GetResponseData(t, resp, "product")
		return int(product["id"].(float64))
	}

	addColorOption := func(productID int) {
		t.Helper()
		body := map[string]any{
			"name":        "color",
			"displayName": "Color",
			"position":    1,
			"values": []map[string]any{
				{"value": "black", "displayName": "Black", "position": 1},
				{"value": "white", "displayName": "White", "position": 2},
			},
		}
		url := fmt.Sprintf("/api/product/%d/option", productID)
		helpers.AssertSuccessResponse(t, client.Post(t, url, body), http.StatusCreated)
	}

	createOptionVariant := func(productID int, sku string, price float64, color string) map[string]any {
		t.Helper()
		body := map[string]any{
			"sku":   sku,
			"price": price,
			"options": []map[string]any{
				{"optionName": "color", "value": color},
			},
		}
		url := fmt.Sprintf("/api/product/%d/variant", productID)
		resp := helpers.AssertSuccessResponse(t, client.Post(t, url, body), http.StatusCreated)
		return helpers.GetResponseData(t, resp, "variant")
	}

	getProduct := func(productID int) map[string]any {
		t.Helper()
		client.SetHeader("X-Seller-ID", fmt.Sprintf("%d", helpers.SellerUserID))
		resp := helpers.AssertSuccessResponse(t, client.Get(t, fmt.Sprintf("/api/product/%d", productID)), http.StatusOK)
		return helpers.GetResponseData(t, resp, "product")
	}

	t.Run("A - Simple listing shape", func(t *testing.T) {
		productID := createSimpleProduct("Simple Listing Product", "TEST-SIMPLE-LIST-001", 29.99)

		client.SetHeader("X-Seller-ID", fmt.Sprintf("%d", helpers.SellerUserID))
		listResp := helpers.AssertSuccessResponse(t, client.Get(t, "/api/product?pageSize=50"), http.StatusOK)
		products := listResp["data"].(map[string]any)["products"].([]any)

		var found map[string]any
		for _, p := range products {
			product := p.(map[string]any)
			if int(product["id"].(float64)) == productID {
				found = product
				break
			}
		}
		require.NotNil(t, found, "Simple product should appear in listing")

		assert.Equal(t, false, found["hasVariants"])
		assert.Equal(t, 29.99, found["price"])
		assert.Nil(t, found["variantPreview"], "Simple products should not expose variantPreview")
	})

	t.Run("B - POST simple product with top-level price only", func(t *testing.T) {
		loginSeller()
		body := map[string]any{
			"name":       "Simple POST Product",
			"categoryId": 4,
			"baseSku":    "TEST-SIMPLE-POST-001",
			"price":      49.99,
		}
		resp := helpers.AssertSuccessResponse(t, client.Post(t, "/api/product", body), http.StatusCreated)
		product := helpers.GetResponseData(t, resp, "product")

		assert.Equal(t, false, product["hasVariants"])
		assert.Equal(t, 49.99, product["price"])
		variants, ok := product["variants"].([]any)
		assert.True(t, ok)
		assert.Empty(t, variants)
	})

	t.Run("C - Simple to configurable migration", func(t *testing.T) {
		productID := createSimpleProduct("Migrate Simple Product", "TEST-MIGRATE-SC-001", 59.99)
		addColorOption(productID)
		createOptionVariant(productID, "TEST-MIGRATE-SC-001-BLK", 64.99, "black")

		client.SetHeader("X-Seller-ID", fmt.Sprintf("%d", helpers.SellerUserID))
		listResp := helpers.AssertSuccessResponse(t, client.Get(t, fmt.Sprintf("/api/product?pageSize=50")), http.StatusOK)
		products := listResp["data"].(map[string]any)["products"].([]any)

		var found map[string]any
		for _, p := range products {
			product := p.(map[string]any)
			if int(product["id"].(float64)) == productID {
				found = product
				break
			}
		}
		require.NotNil(t, found)
		assert.Equal(t, true, found["hasVariants"])

		preview := found["variantPreview"].(map[string]any)
		assert.Equal(t, float64(1), preview["totalVariants"])
	})

	t.Run("D - Second option variant has no empty selectedOptions", func(t *testing.T) {
		productID := createSimpleProduct("Two Variant Product", "TEST-TWO-VAR-001", 19.99)
		addColorOption(productID)
		createOptionVariant(productID, "TEST-TWO-VAR-001-BLK", 24.99, "black")
		createOptionVariant(productID, "TEST-TWO-VAR-001-WHT", 26.99, "white")

		product := getProduct(productID)
		variants := product["variants"].([]any)
		assert.Len(t, variants, 2)

		for _, v := range variants {
			variant := v.(map[string]any)
			opts := variant["selectedOptions"].([]any)
			assert.NotEmpty(t, opts, "Public variants must have selected options")
		}
	})

	t.Run("E - OR semantics for allowPurchase and isPopular", func(t *testing.T) {
		loginSeller()
		body := map[string]any{
			"name":       "OR Semantics Product",
			"categoryId": 4,
			"baseSku":    "TEST-OR-001",
			"options": []map[string]any{
				{
					"name": "color", "displayName": "Color",
					"values": []map[string]any{
						{"value": "black", "displayName": "Black"},
						{"value": "white", "displayName": "White"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku": "TEST-OR-001-BLK", "price": 10.0,
					"allowPurchase": true, "isPopular": false,
					"options": []map[string]any{{"optionName": "color", "value": "black"}},
				},
				{
					"sku": "TEST-OR-001-WHT", "price": 12.0,
					"allowPurchase": false, "isPopular": true,
					"options": []map[string]any{{"optionName": "color", "value": "white"}},
				},
			},
		}
		resp := helpers.AssertSuccessResponse(t, client.Post(t, "/api/product", body), http.StatusCreated)
		productID := int(helpers.GetResponseData(t, resp, "product")["id"].(float64))

		product := getProduct(productID)
		assert.True(t, product["allowPurchase"].(bool), "allowPurchase OR: one variant allows purchase")
		assert.True(t, product["isPopular"].(bool), "isPopular OR: one variant is popular")
	})

	t.Run("F - Default reassigned when first option variant created without isDefault", func(t *testing.T) {
		productID := createSimpleProduct("Default Reassign SC", "TEST-DEFAULT-SC-001", 39.99)
		addColorOption(productID)
		variant := createOptionVariant(productID, "TEST-DEFAULT-SC-001-BLK", 44.99, "black")
		assert.True(t, variant["isDefault"].(bool))
	})

	t.Run("G - PUT price on simple product updates placeholder only", func(t *testing.T) {
		productID := createSimpleProduct("PUT Price Simple", "TEST-PUT-PRICE-001", 100.0)

		loginSeller()
		putURL := fmt.Sprintf("/api/product/%d", productID)
		putResp := helpers.AssertSuccessResponse(t, client.Put(t, putURL, map[string]any{"price": 150.0}), http.StatusOK)
		updated := helpers.GetResponseData(t, putResp, "product")
		assert.Equal(t, 150.0, updated["price"])

		product := getProduct(productID)
		assert.Equal(t, 150.0, product["price"])
		variants := product["variants"].([]any)
		assert.Empty(t, variants)

		var dbVariant entity.ProductVariant
		require.NoError(t, containers.DB.Where("product_id = ?", productID).First(&dbVariant).Error)
		assert.Equal(t, 150.0, dbVariant.Price)
	})

	t.Run("H - PUT allowPurchase false on simple product", func(t *testing.T) {
		productID := createSimpleProduct("PUT AllowPurchase Simple", "TEST-PUT-AP-001", 80.0)

		loginSeller()
		putURL := fmt.Sprintf("/api/product/%d", productID)
		putResp := helpers.AssertSuccessResponse(
			t,
			client.Put(t, putURL, map[string]any{"allowPurchase": false}),
			http.StatusOK,
		)
		updated := helpers.GetResponseData(t, putResp, "product")
		assert.Equal(t, false, updated["allowPurchase"])

		var dbVariant entity.ProductVariant
		require.NoError(t, containers.DB.Where("product_id = ?", productID).First(&dbVariant).Error)
		assert.False(t, dbVariant.AllowPurchase)
	})

	t.Run("I - PUT price on configurable updates default variant only", func(t *testing.T) {
		loginSeller()
		body := map[string]any{
			"name": "PUT Price Configurable", "categoryId": 4, "baseSku": "TEST-PUT-CFG-001",
			"options": []map[string]any{
				{
					"name": "color", "displayName": "Color",
					"values": []map[string]any{
						{"value": "black", "displayName": "Black"},
						{"value": "white", "displayName": "White"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku": "TEST-PUT-CFG-001-BLK", "price": 30.0, "isDefault": true,
					"options": []map[string]any{{"optionName": "color", "value": "black"}},
				},
				{
					"sku": "TEST-PUT-CFG-001-WHT", "price": 35.0, "isDefault": false,
					"options": []map[string]any{{"optionName": "color", "value": "white"}},
				},
			},
		}
		resp := helpers.AssertSuccessResponse(t, client.Post(t, "/api/product", body), http.StatusCreated)
		productID := int(helpers.GetResponseData(t, resp, "product")["id"].(float64))

		putURL := fmt.Sprintf("/api/product/%d", productID)
		putResp := helpers.AssertSuccessResponse(t, client.Put(t, putURL, map[string]any{"price": 39.99}), http.StatusOK)
		assert.Equal(t, 39.99, helpers.GetResponseData(t, putResp, "product")["price"])

		var defaultVariant, otherVariant entity.ProductVariant
		require.NoError(t, containers.DB.Where("product_id = ? AND sku = ?", productID, "TEST-PUT-CFG-001-BLK").First(&defaultVariant).Error)
		require.NoError(t, containers.DB.Where("product_id = ? AND sku = ?", productID, "TEST-PUT-CFG-001-WHT").First(&otherVariant).Error)
		assert.Equal(t, 39.99, defaultVariant.Price)
		assert.Equal(t, 35.0, otherVariant.Price)
	})

	t.Run("J - PUT isPopular true on configurable updates all variants", func(t *testing.T) {
		loginSeller()
		body := map[string]any{
			"name": "PUT Popular Configurable", "categoryId": 4, "baseSku": "TEST-PUT-POP-001",
			"options": []map[string]any{
				{
					"name": "color", "displayName": "Color",
					"values": []map[string]any{
						{"value": "black", "displayName": "Black"},
						{"value": "white", "displayName": "White"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku": "TEST-PUT-POP-001-BLK", "price": 20.0, "isPopular": false,
					"options": []map[string]any{{"optionName": "color", "value": "black"}},
				},
				{
					"sku": "TEST-PUT-POP-001-WHT", "price": 22.0, "isPopular": false,
					"options": []map[string]any{{"optionName": "color", "value": "white"}},
				},
			},
		}
		resp := helpers.AssertSuccessResponse(t, client.Post(t, "/api/product", body), http.StatusCreated)
		productID := int(helpers.GetResponseData(t, resp, "product")["id"].(float64))

		putURL := fmt.Sprintf("/api/product/%d", productID)
		putResp := helpers.AssertSuccessResponse(t, client.Put(t, putURL, map[string]any{"isPopular": true}), http.StatusOK)
		assert.True(t, helpers.GetResponseData(t, putResp, "product")["isPopular"].(bool))

		var variants []entity.ProductVariant
		require.NoError(t, containers.DB.Where("product_id = ?", productID).Find(&variants).Error)
		require.Len(t, variants, 2)
		for _, v := range variants {
			assert.True(t, v.IsPopular)
		}
	})
}
