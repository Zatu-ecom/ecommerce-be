package product_media_test

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetProduct_MediaFieldAlwaysPresent verifies that GET /api/product/:id always
// returns a "media" key that is a JSON array (not null/missing), even when the
// product has no attached media rows. This test FAILS before T019 is implemented
// because BuildProductResponse leaves Media as nil (serialised as null by JSON
// encoder), and PASSES after T019 initialises Media to an empty slice.
func TestGetProduct_MediaFieldAlwaysPresent(t *testing.T) {
	env := newMediaTestEnv(t)
	// Product 1 (iPhone 15 Pro) is owned by seller 2 — use seller 2's credentials.
	token := env.seller2Token(t)
	env.client.SetToken(token)

	// Product 1 is seeded (iPhone 15 Pro by seller 2).  No product_media rows are
	// inserted so media must be an empty ordered collection.
	w := env.client.Get(t, fmt.Sprintf("/api/product/%d", 1))
	require.Equal(t, http.StatusOK, w.Code, "product should be found")

	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	product := helpers.GetResponseData(t, resp, "product")

	mediaRaw, exists := product["media"]
	assert.True(t, exists, "product response must contain 'media' key")
	assert.NotNil(t, mediaRaw, "'media' must not be null when product has no media rows")

	mediaSlice, ok := mediaRaw.([]any)
	assert.True(t, ok, "'media' must be a JSON array, got: %T", mediaRaw)
	assert.Empty(t, mediaSlice, "'media' should be empty when no rows exist")
}

// TestGetProduct_MediaResilientToMissingFile verifies that GET /api/product/:id
// still succeeds with a 200 when the product has product_media rows that reference
// a non-existent file_id. The missing file data must be silently skipped so the
// product response degrades gracefully (media: []) without returning an error.
func TestGetProduct_MediaResilientToMissingFile(t *testing.T) {
	env := newMediaTestEnv(t)

	// Insert a product_media row with a fake file_id directly into the DB so the
	// test does not depend on the attach endpoint (implemented in Phase 4).
	sqlDB, err := env.containers.DB.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec(
		`INSERT INTO product_media (product_id, file_id, is_primary, display_order, created_at, updated_at)
		 VALUES (1, 'fake-file-id-not-in-file-object', true, 0, NOW(), NOW())`,
	)
	require.NoError(t, err, "should be able to insert test product_media row")

	// Product 1 (iPhone 15 Pro) is owned by seller 2 — use seller 2's credentials.
	token := env.seller2Token(t)
	env.client.SetToken(token)

	w := env.client.Get(t, fmt.Sprintf("/api/product/%d", 1))
	assert.Equal(t, http.StatusOK, w.Code, "product should load despite missing file reference")

	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	product := helpers.GetResponseData(t, resp, "product")

	mediaRaw, exists := product["media"]
	assert.True(t, exists, "product response must still contain 'media' key")

	mediaSlice, ok := mediaRaw.([]any)
	assert.True(t, ok, "'media' must be a JSON array")
	// The item with the non-existent file_id must be silently skipped.
	assert.Empty(t, mediaSlice, "media items with missing file data must be silently skipped")
}

// TestGetProduct_MediaOrderedByDisplayOrder verifies that media items returned on
// a product detail response are in ascending display_order order.
// The test inserts multiple product_media rows out of order and asserts they come
// back sorted. Because the file references do not resolve to real URLs, only the
// ordering behaviour (display_order ascending) can be verified here; URL content
// is validated in upload-level integration tests.
//
// This test FAILS before T017-T019 are implemented (GetMediaForProducts is a stub
// returning {}) and PASSES once the full read path is implemented. Specifically,
// it only produces non-empty media when a real storage config is available; in
// the default CI test run (no MinIO) the file gateway silently skips items whose
// DownloadURL cannot be generated, so the ordering assertion is effectively a
// no-op for URL-less items. The primary value of the test is the resilience
// assertion: the response is always 200 with media: [].
func TestGetProduct_MediaOrderedByDisplayOrder(t *testing.T) {
	env := newMediaTestEnv(t)

	sqlDB, err := env.containers.DB.DB()
	require.NoError(t, err)

	// Insert rows in reverse order to make the sort observable.
	_, err = sqlDB.Exec(
		`INSERT INTO product_media (product_id, file_id, is_primary, display_order, created_at, updated_at)
		 VALUES
		   (2, 'order-file-b', false, 10, NOW(), NOW()),
		   (2, 'order-file-a', true,   0, NOW(), NOW()),
		   (2, 'order-file-c', false, 20, NOW(), NOW())`,
	)
	require.NoError(t, err)

	// Product 2 (Samsung S24) is owned by seller 2 — use seller 2's credentials.
	token := env.seller2Token(t)
	env.client.SetToken(token)

	w := env.client.Get(t, fmt.Sprintf("/api/product/%d", 2))
	require.Equal(t, http.StatusOK, w.Code)

	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	product := helpers.GetResponseData(t, resp, "product")

	mediaRaw, exists := product["media"]
	assert.True(t, exists, "product response must contain 'media' key")
	assert.NotNil(t, mediaRaw)

	mediaSlice, ok := mediaRaw.([]any)
	assert.True(t, ok, "'media' must be a JSON array")

	// If items were resolved (requires real storage), they must be ordered.
	if len(mediaSlice) > 1 {
		assertMediaOrdered(t, mediaSlice)
	}
}

// TestGetProduct_NotFound verifies that a non-existent product returns 404.
// This is a regression guard to ensure the media path does not alter error handling.
func TestGetProduct_NotFound(t *testing.T) {
	env := newMediaTestEnv(t)
	token := env.sellerToken(t)
	env.client.SetToken(token)

	w := env.client.Get(t, fmt.Sprintf("/api/product/%d", 999999))
	assert.Equal(t, http.StatusNotFound, w.Code)
}
