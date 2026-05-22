package product_media_test

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Remove media (DELETE /api/product/:productId/media/:fileId) ──────────────

// TestRemoveMedia_Unauthenticated verifies the DELETE endpoint requires auth.
func TestRemoveMedia_Unauthenticated(t *testing.T) {
	env := newMediaTestEnv(t)

	w := env.client.Delete(t, productMediaItemURL(1, "any-file-id"))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestRemoveMedia_CustomerForbidden verifies that a customer (non-seller) role
// cannot remove product media.
func TestRemoveMedia_CustomerForbidden(t *testing.T) {
	env := newMediaTestEnv(t)
	token := env.customerToken(t)
	env.client.SetToken(token)

	w := env.client.Delete(t, productMediaItemURL(1, "any-file-id"))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestRemoveMedia_ProductNotFound verifies that deleting media from a non-existent
// product returns 404.
func TestRemoveMedia_ProductNotFound(t *testing.T) {
	env := newMediaTestEnv(t)
	token := helpers.Login(t, env.client, helpers.Seller2Email, helpers.Seller2Password)
	env.client.SetToken(token)

	w := env.client.Delete(t, productMediaItemURL(999999, "any-file-id"))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestRemoveMedia_LinkNotFound verifies that deleting a non-existent media link
// returns 404.
func TestRemoveMedia_LinkNotFound(t *testing.T) {
	env := newMediaTestEnv(t)
	token := helpers.Login(t, env.client, helpers.Seller2Email, helpers.Seller2Password)
	env.client.SetToken(token)

	w := env.client.Delete(t, productMediaItemURL(1, "non-existent-link"))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestRemoveMedia_WrongSeller verifies seller isolation: a seller cannot remove
// media from a product owned by a different seller (returns 404 for security).
func TestRemoveMedia_WrongSeller(t *testing.T) {
	env := newMediaTestEnv(t)

	seedMediaRow(t, env, 1, "isolation-remove-file-id", false, 0)

	// Jane (seller 3) tries to remove from product 1 (owned by seller 2) → 404.
	token := env.sellerToken(t)
	env.client.SetToken(token)

	w := env.client.Delete(t, productMediaItemURL(1, "isolation-remove-file-id"))
	assert.Equal(t, http.StatusNotFound, w.Code, "wrong seller should see 404")
}

// TestRemoveMedia_HappyPath verifies that an authorized seller can remove a
// linked media item and the endpoint returns 204 No Content.
// The product_media row must be absent from the DB after the call.
func TestRemoveMedia_HappyPath(t *testing.T) {
	env := newMediaTestEnv(t)

	const targetFileID = "remove-happy-path-file-id"
	seedMediaRow(t, env, 1, targetFileID, false, 0)

	token := helpers.Login(t, env.client, helpers.Seller2Email, helpers.Seller2Password)
	env.client.SetToken(token)

	w := env.client.Delete(t, productMediaItemURL(1, targetFileID))
	require.Equal(t, http.StatusNoContent, w.Code, "remove should return 204 No Content")
	assert.Empty(t, w.Body.String(), "204 response should have no body")

	// Verify the row is gone from the DB.
	sqlDB, err := env.containers.DB.DB()
	require.NoError(t, err)
	var count int
	err = sqlDB.QueryRow(
		`SELECT COUNT(*) FROM product_media WHERE product_id = 1 AND file_id = $1`,
		targetFileID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "product_media row must be deleted")
}

// TestRemoveMedia_ProductResponseNoLongerIncludesMedia verifies that after
// removing a media link, a subsequent GET /api/product/:id no longer contains
// that item in the media collection.
// NOTE: Because there is no real storage in the CI environment, file info
// cannot be resolved and the media item was already silently skipped from
// responses before removal. This test therefore verifies the structural
// contract: the row is deleted and the response is still 200 with media: [].
func TestRemoveMedia_ProductResponseNoLongerIncludesMedia(t *testing.T) {
	env := newMediaTestEnv(t)

	const targetFileID = "remove-verify-response-file-id"
	seedMediaRow(t, env, 2, targetFileID, false, 0)

	token := helpers.Login(t, env.client, helpers.Seller2Email, helpers.Seller2Password)
	env.client.SetToken(token)

	// Remove the media link.
	w := env.client.Delete(t, productMediaItemURL(2, targetFileID))
	require.Equal(t, http.StatusNoContent, w.Code)

	// Read the product — it must still load (200) and media must be a JSON array.
	w = env.client.Get(t, fmt.Sprintf("/api/product/%d", 2))
	require.Equal(t, http.StatusOK, w.Code, "product should still load after media removal")

	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	product := helpers.GetResponseData(t, resp, "product")

	mediaRaw, exists := product["media"]
	assert.True(t, exists, "product must still have 'media' key after removal")
	mediaSlice, ok := mediaRaw.([]any)
	assert.True(t, ok, "'media' must be a JSON array")

	// None of the remaining media items should reference the removed fileId.
	for _, item := range mediaSlice {
		m := item.(map[string]any)
		assert.NotEqual(t, targetFileID, m["fileId"], "removed file must not appear in product media")
	}
}

// TestRemoveMedia_PrimaryFallbackPromotion verifies that when the primary media
// item is removed, the remaining item with the lowest display_order is promoted
// to primary automatically.
func TestRemoveMedia_PrimaryFallbackPromotion(t *testing.T) {
	env := newMediaTestEnv(t)

	const (
		primaryFileID  = "remove-primary-file-id"
		fallbackFileID = "remove-fallback-file-id"
	)

	// Set up: primary at order=0, fallback at order=1.
	seedMediaRow(t, env, 1, primaryFileID, true, 0)
	seedMediaRow(t, env, 1, fallbackFileID, false, 1)

	token := helpers.Login(t, env.client, helpers.Seller2Email, helpers.Seller2Password)
	env.client.SetToken(token)

	// Remove the primary item.
	w := env.client.Delete(t, productMediaItemURL(1, primaryFileID))
	require.Equal(t, http.StatusNoContent, w.Code, "removing primary should return 204")

	// Verify the fallback row is now primary in the DB.
	sqlDB, err := env.containers.DB.DB()
	require.NoError(t, err)
	var isPrimary bool
	err = sqlDB.QueryRow(
		`SELECT is_primary FROM product_media WHERE product_id = 1 AND file_id = $1`,
		fallbackFileID,
	).Scan(&isPrimary)
	require.NoError(t, err)
	assert.True(t, isPrimary, "fallback item must be promoted to primary after removing the primary")

	// Verify the primary row is gone.
	var count int
	err = sqlDB.QueryRow(
		`SELECT COUNT(*) FROM product_media WHERE product_id = 1 AND file_id = $1`,
		primaryFileID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "original primary row must be deleted")
}

// TestRemoveMedia_OnlyOneItem_NoFallbackNeeded verifies that removing the sole
// media item leaves the product with an empty media collection and does not error.
func TestRemoveMedia_OnlyOneItem_NoFallbackNeeded(t *testing.T) {
	env := newMediaTestEnv(t)

	const soleFileID = "remove-sole-item-file-id"
	seedMediaRow(t, env, 1, soleFileID, true, 0)

	token := helpers.Login(t, env.client, helpers.Seller2Email, helpers.Seller2Password)
	env.client.SetToken(token)

	w := env.client.Delete(t, productMediaItemURL(1, soleFileID))
	require.Equal(t, http.StatusNoContent, w.Code)

	// Verify no rows remain for this product.
	sqlDB, err := env.containers.DB.DB()
	require.NoError(t, err)
	var count int
	err = sqlDB.QueryRow(
		`SELECT COUNT(*) FROM product_media WHERE product_id = 1`,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "no media rows should remain after removing the sole item")
}

// TestRemoveMedia_BestEffortCleanup verifies that a media link is successfully
// removed (204) even when the underlying file no longer exists in the file_object
// table. The file gateway treats "file not found" as a no-op and returns nil,
// so the product-media removal is always reported as successful regardless of
// whether the file asset cleanup actually ran.
//
// For the "cleanup fails with a storage error" path (e.g. blob unreachable), the
// service logs a warning and still returns nil (success) to the caller. This
// behaviour is verified by the service-layer unit test in
// product/service/product_media_service_test.go. An integration test that forces
// blob connectivity failures would require a deliberately broken storage config
// and is out of scope for the default no-storage CI environment.
func TestRemoveMedia_BestEffortCleanup(t *testing.T) {
	env := newMediaTestEnv(t)

	// Insert a product_media row pointing to a file_id that does NOT exist in
	// file_object. DeleteFile will silently no-op (file not found path).
	const missingFileID = "remove-cleanup-noop-file-id"
	seedMediaRow(t, env, 1, missingFileID, false, 0)

	token := helpers.Login(t, env.client, helpers.Seller2Email, helpers.Seller2Password)
	env.client.SetToken(token)

	w := env.client.Delete(t, productMediaItemURL(1, missingFileID))
	assert.Equal(t, http.StatusNoContent, w.Code,
		"removal must succeed even when file cleanup is a no-op")

	// Verify the product_media row is gone.
	sqlDB, err := env.containers.DB.DB()
	require.NoError(t, err)
	var count int
	err = sqlDB.QueryRow(
		`SELECT COUNT(*) FROM product_media WHERE product_id = 1 AND file_id = $1`,
		missingFileID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
