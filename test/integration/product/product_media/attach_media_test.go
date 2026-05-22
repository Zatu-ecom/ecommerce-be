package product_media_test

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Attach media (POST /api/product/:productId/media) ───────────────────────

// TestAttachMedia_Unauthenticated verifies that the attach endpoint requires auth.
func TestAttachMedia_Unauthenticated(t *testing.T) {
	env := newMediaTestEnv(t)

	w := env.client.Post(t, productMediaURL(1), map[string]any{
		"fileId": "some-file-id",
	})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAttachMedia_CustomerForbidden verifies that a customer (non-seller) role
// cannot attach media.
func TestAttachMedia_CustomerForbidden(t *testing.T) {
	env := newMediaTestEnv(t)
	token := env.customerToken(t)
	env.client.SetToken(token)

	w := env.client.Post(t, productMediaURL(1), map[string]any{
		"fileId": "some-file-id",
	})
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestAttachMedia_ProductNotFound verifies that attaching to a non-existent
// product returns 404.
func TestAttachMedia_ProductNotFound(t *testing.T) {
	env := newMediaTestEnv(t)
	token := env.sellerToken(t)
	env.client.SetToken(token)

	w := env.client.Post(t, productMediaURL(999999), map[string]any{
		"fileId": "some-file-id",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAttachMedia_MissingFileId verifies that the request body must contain a
// non-empty fileId field.
func TestAttachMedia_MissingFileId(t *testing.T) {
	env := newMediaTestEnv(t)
	token := env.sellerToken(t)
	env.client.SetToken(token)

	w := env.client.Post(t, productMediaURL(1), map[string]any{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAttachMedia_InvalidFile verifies that attaching a file_id that does not
// exist in the file_object table returns 422 Unprocessable Entity.
func TestAttachMedia_InvalidFile(t *testing.T) {
	env := newMediaTestEnv(t)

	// Product 1 belongs to seller 2; jane (seller ID 3) doesn't own it so we use
	// seller 2 (john.seller) for this test.
	token := helpers.Login(t, env.client, helpers.Seller2Email, helpers.Seller2Password)
	env.client.SetToken(token)

	w := env.client.Post(t, productMediaURL(1), map[string]any{
		"fileId": "non-existent-file-00000000-0000-0000-0000-000000000000",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// TestAttachMedia_WrongSeller verifies that a seller cannot attach media to
// a product owned by a different seller (returns 404 for security).
func TestAttachMedia_WrongSeller(t *testing.T) {
	env := newMediaTestEnv(t)

	// Product 1 belongs to seller 2 (john.seller); jane (seller 3) must not access it.
	token := env.sellerToken(t) // jane.merchant → seller 3
	env.client.SetToken(token)

	w := env.client.Post(t, productMediaURL(1), map[string]any{
		"fileId": "some-file-id",
	})
	assert.Equal(t, http.StatusNotFound, w.Code, "wrong seller should see 404 not 403")
}

// TestAttachMedia_Duplicate verifies that attaching the same file_id to the same
// product twice returns 409 Conflict on the second attempt.
// The duplicate check happens before the file gateway call, so this test works
// without real storage by inserting a product_media row directly.
func TestAttachMedia_Duplicate(t *testing.T) {
	env := newMediaTestEnv(t)

	// Insert an existing product_media row for product 1 + a known file_id.
	sqlDB, err := env.containers.DB.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec(
		`INSERT INTO product_media (product_id, file_id, is_primary, display_order, created_at, updated_at)
		 VALUES (1, 'duplicate-file-id', true, 0, NOW(), NOW())`,
	)
	require.NoError(t, err)

	// john.seller owns product 1.
	token := helpers.Login(t, env.client, helpers.Seller2Email, helpers.Seller2Password)
	env.client.SetToken(token)

	w := env.client.Post(t, productMediaURL(1), map[string]any{
		"fileId": "duplicate-file-id",
	})
	assert.Equal(t, http.StatusConflict, w.Code, "duplicate file attachment should return 409")
}

// TestAttachMedia_InvalidDisplayOrder verifies that a negative displayOrder is
// rejected with 400.
func TestAttachMedia_InvalidDisplayOrder(t *testing.T) {
	env := newMediaTestEnv(t)
	token := helpers.Login(t, env.client, helpers.Seller2Email, helpers.Seller2Password)
	env.client.SetToken(token)

	w := env.client.Post(t, productMediaURL(1), map[string]any{
		"fileId":       "some-file-id",
		"displayOrder": -1,
	})
	assert.Equal(t, http.StatusBadRequest, w.Code, "negative displayOrder must be rejected")
}
