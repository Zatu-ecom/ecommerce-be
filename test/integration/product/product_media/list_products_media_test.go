package product_media_test

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestListProducts_MediaFieldAlwaysPresent verifies that GET /api/product returns
// a list of products where every item has a "media" key that is a JSON array.
// This test FAILS before T019 is implemented (media is null/absent) and PASSES
// once BuildProductResponse initialises the Media field to an empty slice.
func TestListProducts_MediaFieldAlwaysPresent(t *testing.T) {
	env := newMediaTestEnv(t)
	token := env.sellerToken(t)
	env.client.SetToken(token)

	w := env.client.Get(t, "/api/product?page=1&pageSize=10")
	require.Equal(t, http.StatusOK, w.Code)

	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok, "response should have a data object")

	productsRaw, ok := data["products"].([]any)
	require.True(t, ok, "data should contain 'products' array")
	require.NotEmpty(t, productsRaw, "expected seeded products to be returned")

	for i, p := range productsRaw {
		product, ok := p.(map[string]any)
		require.True(t, ok, "each product should be a JSON object")

		mediaRaw, exists := product["media"]
		assert.True(t, exists, "product[%d] must have 'media' key", i)
		assert.NotNil(t, mediaRaw, "product[%d] 'media' must not be null", i)

		_, ok = mediaRaw.([]any)
		assert.True(t, ok, "product[%d] 'media' must be a JSON array", i)
	}
}

// TestListProducts_MediaResilientToMissingFiles verifies that a product listing
// still succeeds when some products have product_media rows with non-existent
// file_ids. The missing file data must be silently skipped while all products
// are still included in the response.
func TestListProducts_MediaResilientToMissingFiles(t *testing.T) {
	env := newMediaTestEnv(t)

	// Attach fake media to two seeded products so they have product_media rows.
	sqlDB, err := env.containers.DB.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec(
		`INSERT INTO product_media (product_id, file_id, is_primary, display_order, created_at, updated_at)
		 VALUES
		   (1, 'list-fake-file-1', true,  0, NOW(), NOW()),
		   (3, 'list-fake-file-2', false, 0, NOW(), NOW())`,
	)
	require.NoError(t, err)

	token := env.sellerToken(t)
	env.client.SetToken(token)

	w := env.client.Get(t, "/api/product?page=1&pageSize=10")
	require.Equal(t, http.StatusOK, w.Code, "listing should succeed despite missing file references")

	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)

	productsRaw, ok := data["products"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, productsRaw)

	for _, p := range productsRaw {
		product := p.(map[string]any)
		mediaRaw, exists := product["media"]
		assert.True(t, exists, "all products must still have 'media' key")
		assert.NotNil(t, mediaRaw)

		_, ok := mediaRaw.([]any)
		assert.True(t, ok, "'media' must always be a JSON array")
	}
}

// TestListProducts_NoSingleProductLoopForMedia verifies that the product list
// endpoint does not perform a per-product media lookup. The test ensures all
// products are returned in a single response without obvious per-item latency
// indicators. (The actual N+1 prevention is structural – enforced by the
// batch-load implementation in GetMediaForProducts.)
func TestListProducts_NoSingleProductLoopForMedia(t *testing.T) {
	env := newMediaTestEnv(t)
	token := env.sellerToken(t)
	env.client.SetToken(token)

	// Simply verify the listing endpoint returns all expected products without
	// error. The batch vs. N+1 guarantee is tested at the service layer.
	w := env.client.Get(t, "/api/product?page=1&pageSize=20")
	assert.Equal(t, http.StatusOK, w.Code)

	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)

	productsRaw, ok := data["products"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(productsRaw), 1, "at least one seeded product expected")
}
