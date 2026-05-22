package product_media_test

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedMediaRow inserts a product_media row directly and returns the fileID used.
// This helper is shared across update tests to avoid duplicating setup code.
func seedMediaRow(
	t *testing.T,
	env *mediaTestEnv,
	productID int,
	fileID string,
	isPrimary bool,
	displayOrder int,
) {
	t.Helper()
	sqlDB, err := env.containers.DB.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec(
		`INSERT INTO product_media (product_id, file_id, is_primary, display_order, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW())
		 ON CONFLICT (product_id, file_id) DO NOTHING`,
		productID, fileID, isPrimary, displayOrder,
	)
	require.NoError(t, err)
}

// ─── Update media metadata (PATCH /api/product/:productId/media/:fileId) ─────

// TestUpdateMedia_Unauthenticated verifies the PATCH endpoint requires auth.
func TestUpdateMedia_Unauthenticated(t *testing.T) {
	env := newMediaTestEnv(t)

	w := env.client.Patch(t, productMediaItemURL(1, "any-file-id"), map[string]any{
		"displayOrder": 5,
	})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestUpdateMedia_CustomerForbidden verifies that a customer (non-seller) cannot
// update product media metadata.
func TestUpdateMedia_CustomerForbidden(t *testing.T) {
	env := newMediaTestEnv(t)
	token := env.customerToken(t)
	env.client.SetToken(token)

	w := env.client.Patch(t, productMediaItemURL(1, "any-file-id"), map[string]any{
		"displayOrder": 5,
	})
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestUpdateMedia_LinkNotFound verifies that patching a non-existent product-media
// link returns 404.
func TestUpdateMedia_LinkNotFound(t *testing.T) {
	env := newMediaTestEnv(t)

	// Product 1 belongs to seller 2.
	token := helpers.Login(t, env.client, helpers.Seller2Email, helpers.Seller2Password)
	env.client.SetToken(token)

	w := env.client.Patch(t, productMediaItemURL(1, "non-existent-link-file-id"), map[string]any{
		"displayOrder": 5,
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestUpdateMedia_ProductNotFound verifies that patching media on a non-existent
// product returns 404.
func TestUpdateMedia_ProductNotFound(t *testing.T) {
	env := newMediaTestEnv(t)
	token := helpers.Login(t, env.client, helpers.Seller2Email, helpers.Seller2Password)
	env.client.SetToken(token)

	w := env.client.Patch(t, productMediaItemURL(999999, "some-file-id"), map[string]any{
		"displayOrder": 5,
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestUpdateMedia_WrongSeller verifies seller isolation: a seller cannot update
// media on a product owned by a different seller.
func TestUpdateMedia_WrongSeller(t *testing.T) {
	env := newMediaTestEnv(t)

	// Seed a product_media row on product 1 (owned by seller 2).
	seedMediaRow(t, env, 1, "isolation-file-id", false, 0)

	// Jane (seller 3) tries to update product 1 → should get 404.
	token := env.sellerToken(t)
	env.client.SetToken(token)

	w := env.client.Patch(t, productMediaItemURL(1, "isolation-file-id"), map[string]any{
		"displayOrder": 5,
	})
	assert.Equal(t, http.StatusNotFound, w.Code, "wrong seller should see 404")
}

// TestUpdateMedia_EmptyBody verifies that a request with no updatable fields is
// rejected with 400.
func TestUpdateMedia_EmptyBody(t *testing.T) {
	env := newMediaTestEnv(t)
	token := helpers.Login(t, env.client, helpers.Seller2Email, helpers.Seller2Password)
	env.client.SetToken(token)

	w := env.client.Patch(t, productMediaItemURL(1, "some-file-id"), map[string]any{})
	assert.Equal(t, http.StatusBadRequest, w.Code, "empty PATCH body should be rejected")
}

// TestUpdateMedia_InvalidDisplayOrder verifies that a negative displayOrder is
// rejected with 400.
func TestUpdateMedia_InvalidDisplayOrder(t *testing.T) {
	env := newMediaTestEnv(t)

	seedMediaRow(t, env, 1, "order-update-file-id", false, 0)

	token := helpers.Login(t, env.client, helpers.Seller2Email, helpers.Seller2Password)
	env.client.SetToken(token)

	w := env.client.Patch(t, productMediaItemURL(1, "order-update-file-id"), map[string]any{
		"displayOrder": -5,
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestUpdateMedia_SetDisplayOrder verifies that an authorized seller can update
// the display order of a seeded product-media link. The response code must be 200
// and the updated displayOrder must be reflected in the response.
// NOTE: The response body includes a media item. Because there is no real storage
// in the test environment, the file URL will be empty (item silently degraded) but
// the metadata fields (isPrimary, displayOrder) are always populated from the DB.
func TestUpdateMedia_SetDisplayOrder(t *testing.T) {
	env := newMediaTestEnv(t)

	const targetFileID = "update-display-order-file-id"
	seedMediaRow(t, env, 1, targetFileID, false, 0)

	token := helpers.Login(t, env.client, helpers.Seller2Email, helpers.Seller2Password)
	env.client.SetToken(token)

	w := env.client.Patch(t, productMediaItemURL(1, targetFileID), map[string]any{
		"displayOrder": 10,
	})
	require.Equal(t, http.StatusOK, w.Code, "update should succeed")

	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	media, ok := data["media"].(map[string]any)
	require.True(t, ok, "response data should contain a media item")

	assert.Equal(t, float64(10), media["displayOrder"], "displayOrder should be updated to 10")
	assert.Equal(t, targetFileID, media["fileId"])
}

// TestUpdateMedia_SetPrimary verifies that setting isPrimary=true on a non-primary
// media item changes it to primary and any previously primary item is demoted.
func TestUpdateMedia_SetPrimary(t *testing.T) {
	env := newMediaTestEnv(t)

	const (
		primaryFileID    = "update-was-primary-file-id"
		nonPrimaryFileID = "update-becomes-primary-file-id"
	)

	// Insert two rows: first is primary, second is not.
	seedMediaRow(t, env, 1, primaryFileID, true, 0)
	seedMediaRow(t, env, 1, nonPrimaryFileID, false, 1)

	token := helpers.Login(t, env.client, helpers.Seller2Email, helpers.Seller2Password)
	env.client.SetToken(token)

	// Promote the second row to primary.
	w := env.client.Patch(t, productMediaItemURL(1, nonPrimaryFileID), map[string]any{
		"isPrimary": true,
	})
	require.Equal(t, http.StatusOK, w.Code, "promoting to primary should succeed")

	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	media, ok := data["media"].(map[string]any)
	require.True(t, ok)

	assert.Equal(t, true, media["isPrimary"], "targeted item should now be primary")

	// Verify the former primary is no longer primary in the DB.
	sqlDB, err := env.containers.DB.DB()
	require.NoError(t, err)
	var prevPrimary bool
	err = sqlDB.QueryRow(
		`SELECT is_primary FROM product_media WHERE product_id = 1 AND file_id = $1`,
		primaryFileID,
	).Scan(&prevPrimary)
	require.NoError(t, err)
	assert.False(t, prevPrimary, "former primary must be demoted after new primary is set")
}
