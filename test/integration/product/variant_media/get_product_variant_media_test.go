package variant_media_test

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testVariantID = uint(1)

func findVariantInProduct(t *testing.T, product map[string]any, variantID uint) map[string]any {
	t.Helper()
	variants, ok := product["variants"].([]any)
	require.True(t, ok, "product must have variants array")
	for _, raw := range variants {
		v, ok := raw.(map[string]any)
		require.True(t, ok)
		if uint(v["id"].(float64)) == variantID {
			return v
		}
	}
	t.Fatalf("variant %d not found in product response", variantID)
	return nil
}

func assertVariantMediaResolved(t *testing.T, variant map[string]any, fileID string) {
	t.Helper()
	media, ok := variant["media"].([]any)
	require.True(t, ok, "variant media must be a JSON array")
	require.NotEmpty(t, media, "variant media must not be empty when variant_media row and file exist")

	found := false
	for _, raw := range media {
		item, ok := raw.(map[string]any)
		require.True(t, ok)
		if item["fileId"] == fileID {
			found = true
			assert.NotEmpty(t, item["url"], "resolved media must include a non-empty url")
		}
	}
	assert.True(t, found, "expected fileId %s in variant media", fileID)
}

func seedVariantWithUploadedMedia(t *testing.T, env *variantMediaStorageEnv) string {
	t.Helper()
	token := env.sellerToken(t)
	fileID := uploadFileAsSeller(t, env, token)
	seedVariantMediaRow(t, env.variantMediaTestEnv, int(testVariantID), fileID, true, 0)
	return fileID
}

// TestGetProductByID_VariantMediaResolved verifies GET /api/product/:id returns
// non-empty variant media when variant_media rows reference resolvable seller files.
func TestGetProductByID_VariantMediaResolved(t *testing.T) {
	env := newVariantMediaStorageTestEnv(t)
	fileID := seedVariantWithUploadedMedia(t, env)

	env.client.SetToken(env.sellerToken(t))
	env.client.SetHeader("X-Seller-ID", "2")

	w := env.client.Get(t, "/api/product/1")
	require.Equal(t, http.StatusOK, w.Code)

	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	product := helpers.GetResponseData(t, resp, "product")
	variant := findVariantInProduct(t, product, testVariantID)
	assertVariantMediaResolved(t, variant, fileID)
}

// TestGetVariantByID_VariantMediaResolved verifies GET /api/product/:id/variant/:id
// returns resolved variant media for seller-scoped files.
func TestGetVariantByID_VariantMediaResolved(t *testing.T) {
	env := newVariantMediaStorageTestEnv(t)
	fileID := seedVariantWithUploadedMedia(t, env)

	env.client.SetToken(env.sellerToken(t))
	env.client.SetHeader("X-Seller-ID", "2")

	w := env.client.Get(t, "/api/product/1/variant/1")
	require.Equal(t, http.StatusOK, w.Code)

	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	variant := helpers.GetResponseData(t, resp, "variant")
	assertVariantMediaResolved(t, variant, fileID)
}

// TestGetProductByID_VariantMediaForAnonymousCustomer verifies unauthenticated
// public browsing (X-Seller-ID header, no JWT) still resolves variant media.
func TestGetProductByID_VariantMediaForAnonymousCustomer(t *testing.T) {
	env := newVariantMediaStorageTestEnv(t)
	fileID := seedVariantWithUploadedMedia(t, env)

	anonClient := helpers.NewAPIClient(env.client.Handler)
	anonClient.SetHeader("X-Seller-ID", "2")
	w := anonClient.Get(t, "/api/product/1")
	require.Equal(t, http.StatusOK, w.Code)

	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	product := helpers.GetResponseData(t, resp, "product")
	variant := findVariantInProduct(t, product, testVariantID)
	assertVariantMediaResolved(t, variant, fileID)
}
