package promotion_test

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromotionScopeProductImageUrlFromPrimaryMedia(t *testing.T) {
	env := helpers.SetupFileStorageEnv(t, helpers.DefaultFileStorageEnvConfig())

	sellerToken := helpers.Login(t, env.Client, helpers.Seller2Email, helpers.Seller2Password)
	env.Client.SetToken(sellerToken)

	fileID := helpers.UploadProductImage(t, env.Server, sellerToken)
	helpers.SeedProductMediaRow(t, env.Containers, 1, fileID, true, 0)

	promoPayload := map[string]any{
		"name":          "Scope Image Promo",
		"promotionType": "percentage_discount",
		"discountConfig": map[string]any{
			"percentage": 10,
		},
		"appliesTo":   "specific_products",
		"eligibleFor": "everyone",
		"startsAt":    "2023-01-01T00:00:00Z",
		"endsAt":      "2029-12-31T23:59:59Z",
		"status":      "active",
	}
	promoW := env.Client.Post(t, PromotionAPIEndpoint, promoPayload)
	promoResp := helpers.AssertSuccessResponse(t, promoW, http.StatusCreated)
	promotion := helpers.GetResponseData(t, promoResp, "promotion")
	promoID := uint(promotion["id"].(float64))

	linkW := env.Client.Post(t, "/api/promotion/scope/product", map[string]any{
		"promotionId": promoID,
		"productIds":  []uint{1},
	})
	helpers.AssertSuccessResponse(t, linkW, http.StatusOK)

	scopeW := env.Client.Get(t, fmt.Sprintf("/api/promotion/scope/%d/product", promoID))
	scopeResp := helpers.AssertSuccessResponse(t, scopeW, http.StatusOK)
	rawData, ok := scopeResp["data"].(map[string]any)
	require.True(t, ok)

	var products []any
	switch payload := rawData["products"].(type) {
	case map[string]any:
		products, ok = payload["products"].([]any)
	case []any:
		products = payload
		ok = true
	}
	require.True(t, ok, "scope response must include products list")
	require.NotEmpty(t, products)

	product := products[0].(map[string]any)
	imageURL, ok := product["imageUrl"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, imageURL, "promotion scope product must include imageUrl from primary product media")
}
