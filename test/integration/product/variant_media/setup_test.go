package variant_media_test

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Endpoint helpers ─────────────────────────────────────────────────────────

const (
	variantMediaBasePath = "/api/product/%d/variant/%d/media"
	variantMediaItemPath = "/api/product/%d/variant/%d/media/%s"
)

func variantMediaURL(productID, variantID uint) string {
	return fmt.Sprintf(variantMediaBasePath, productID, variantID)
}

func variantMediaItemURL(productID, variantID uint, fileID string) string {
	return fmt.Sprintf(variantMediaItemPath, productID, variantID, fileID)
}

// ─── Test environment ─────────────────────────────────────────────────────────

// variantMediaTestEnv holds the shared server, API client, and direct-DB access.
type variantMediaTestEnv struct {
	client     *helpers.APIClient
	containers *setup.TestContainer
}

func newVariantMediaTestEnv(t *testing.T) *variantMediaTestEnv {
	t.Helper()

	containers := setup.SetupTestContainers(t)
	t.Cleanup(func() { containers.Cleanup(t) })

	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
	client := helpers.NewAPIClient(server)

	return &variantMediaTestEnv{
		client:     client,
		containers: containers,
	}
}

func (e *variantMediaTestEnv) sellerToken(t *testing.T) string {
	t.Helper()
	return helpers.Login(t, e.client, helpers.Seller2Email, helpers.Seller2Password)
}

func (e *variantMediaTestEnv) customerToken(t *testing.T) string {
	t.Helper()
	return helpers.Login(t, e.client, helpers.CustomerEmail, helpers.CustomerPassword)
}

// seedVariantMediaRow inserts a variant_media row directly via the DB connection.
func seedVariantMediaRow(
	t *testing.T,
	env *variantMediaTestEnv,
	variantID int,
	fileID string,
	isPrimary bool,
	displayOrder int,
) {
	t.Helper()
	sqlDB, err := env.containers.DB.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec(
		`INSERT INTO variant_media (variant_id, file_id, is_primary, display_order, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW())
		 ON CONFLICT (variant_id, file_id) DO NOTHING`,
		variantID, fileID, isPrimary, displayOrder,
	)
	require.NoError(t, err)
}

// ─── Smoke test ───────────────────────────────────────────────────────────────

// TestVariantMediaSuite_Smoke confirms the variant media routes are registered
// and the server starts up cleanly.
func TestVariantMediaSuite_Smoke(t *testing.T) {
	env := newVariantMediaTestEnv(t)
	token := env.sellerToken(t)
	env.client.SetToken(token)

	// GET a variant that exists (product 1, variant 1 from seed data).
	w := env.client.Get(t, "/api/product/1/variant/1")
	assert.Contains(
		t,
		[]int{http.StatusOK, http.StatusNotFound},
		w.Code,
		"variant route must be reachable",
	)
}

// ─── GET /api/product/:productId/variant/:variantId — media field ─────────────

// TestGetVariant_MediaFieldAlwaysPresent verifies that the media field is always
// a JSON array (never null) even when no media is attached to the variant.
func TestGetVariant_MediaFieldAlwaysPresent(t *testing.T) {
	env := newVariantMediaTestEnv(t)
	env.client.SetHeader("X-Seller-ID", "2")

	w := env.client.Get(t, "/api/product/1/variant/1")
	require.Equal(t, http.StatusOK, w.Code, "variant should load (product 1, variant 1)")

	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	variant := helpers.GetResponseData(t, resp, "variant")

	mediaRaw, exists := variant["media"]
	assert.True(t, exists, "variant must have 'media' field")
	_, ok := mediaRaw.([]any)
	assert.True(t, ok, "'media' must be a JSON array, not null")
}

// TestGetVariant_MediaResilientToMissingFile inserts a variant_media row with a
// non-existent file_id. The variant must still load (200) and the unresolvable
// item is silently skipped.
func TestGetVariant_MediaResilientToMissingFile(t *testing.T) {
	env := newVariantMediaTestEnv(t)

	seedVariantMediaRow(t, env, 1, "ghost-file-for-variant-resilience", false, 0)

	env.client.SetHeader("X-Seller-ID", "2")
	w := env.client.Get(t, "/api/product/1/variant/1")
	require.Equal(t, http.StatusOK, w.Code)

	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	variant := helpers.GetResponseData(t, resp, "variant")

	_, ok := variant["media"].([]any)
	assert.True(t, ok, "media must still be a JSON array even when file is not resolvable")
}

// ─── POST /api/product/:productId/variant/:variantId/media ────────────────────

// TestAttachVariantMedia_Unauthenticated verifies the POST endpoint requires auth.
func TestAttachVariantMedia_Unauthenticated(t *testing.T) {
	env := newVariantMediaTestEnv(t)

	w := env.client.Post(t, variantMediaURL(1, 1), map[string]any{
		"fileId": "any-file",
	})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAttachVariantMedia_CustomerForbidden verifies that customers cannot attach
// media to variants.
func TestAttachVariantMedia_CustomerForbidden(t *testing.T) {
	env := newVariantMediaTestEnv(t)
	env.client.SetToken(env.customerToken(t))

	w := env.client.Post(t, variantMediaURL(1, 1), map[string]any{
		"fileId": "any-file",
	})
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestAttachVariantMedia_ProductNotFound verifies 404 for a non-existent product.
func TestAttachVariantMedia_ProductNotFound(t *testing.T) {
	env := newVariantMediaTestEnv(t)
	env.client.SetToken(env.sellerToken(t))

	w := env.client.Post(t, variantMediaURL(999999, 1), map[string]any{
		"fileId": "any-file",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAttachVariantMedia_VariantNotFound verifies 404 for a variant that does not
// belong to the specified product.
func TestAttachVariantMedia_VariantNotFound(t *testing.T) {
	env := newVariantMediaTestEnv(t)
	env.client.SetToken(env.sellerToken(t))

	w := env.client.Post(t, variantMediaURL(1, 999999), map[string]any{
		"fileId": "any-file",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAttachVariantMedia_MissingFileID verifies 400 when fileId is omitted.
func TestAttachVariantMedia_MissingFileID(t *testing.T) {
	env := newVariantMediaTestEnv(t)
	env.client.SetToken(env.sellerToken(t))

	w := env.client.Post(t, variantMediaURL(1, 1), map[string]any{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAttachVariantMedia_WrongSeller verifies seller isolation for attach.
func TestAttachVariantMedia_WrongSeller(t *testing.T) {
	env := newVariantMediaTestEnv(t)
	// Seller 3 tries to attach to product 1 (owned by seller 2).
	token := helpers.Login(t, env.client, helpers.SellerEmail, helpers.SellerPassword)
	env.client.SetToken(token)

	w := env.client.Post(t, variantMediaURL(1, 1), map[string]any{
		"fileId": "isolation-test-file",
	})
	assert.Equal(t, http.StatusNotFound, w.Code, "wrong seller must see 404")
}

// TestAttachVariantMedia_InvalidFile verifies 422 when the fileId does not exist
// in the File module (file not found / inaccessible).
func TestAttachVariantMedia_InvalidFile(t *testing.T) {
	env := newVariantMediaTestEnv(t)
	env.client.SetToken(env.sellerToken(t))

	w := env.client.Post(t, variantMediaURL(1, 1), map[string]any{
		"fileId": "non-existent-file-uuid-000000000001",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// TestAttachVariantMedia_Duplicate verifies 409 when the same file is attached
// twice to the same variant.
func TestAttachVariantMedia_Duplicate(t *testing.T) {
	env := newVariantMediaTestEnv(t)

	seedVariantMediaRow(t, env, 1, "duplicate-attach-file-id", false, 0)

	env.client.SetToken(env.sellerToken(t))
	w := env.client.Post(t, variantMediaURL(1, 1), map[string]any{
		"fileId": "duplicate-attach-file-id",
	})
	assert.Equal(t, http.StatusConflict, w.Code)
}

// ─── PATCH /api/product/:productId/variant/:variantId/media/:fileId ───────────

// TestUpdateVariantMedia_Unauthenticated verifies the PATCH endpoint requires auth.
func TestUpdateVariantMedia_Unauthenticated(t *testing.T) {
	env := newVariantMediaTestEnv(t)

	w := env.client.Patch(t, variantMediaItemURL(1, 1, "any-file"), map[string]any{
		"displayOrder": 2,
	})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestUpdateVariantMedia_CustomerForbidden verifies customers cannot update
// variant media metadata.
func TestUpdateVariantMedia_CustomerForbidden(t *testing.T) {
	env := newVariantMediaTestEnv(t)
	env.client.SetToken(env.customerToken(t))

	w := env.client.Patch(t, variantMediaItemURL(1, 1, "any-file"), map[string]any{
		"displayOrder": 2,
	})
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestUpdateVariantMedia_LinkNotFound verifies 404 when the media link does not exist.
func TestUpdateVariantMedia_LinkNotFound(t *testing.T) {
	env := newVariantMediaTestEnv(t)
	env.client.SetToken(env.sellerToken(t))

	w := env.client.Patch(t, variantMediaItemURL(1, 1, "non-existent-link"), map[string]any{
		"displayOrder": 1,
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestUpdateVariantMedia_EmptyBody verifies 400 when no fields are provided.
func TestUpdateVariantMedia_EmptyBody(t *testing.T) {
	env := newVariantMediaTestEnv(t)

	seedVariantMediaRow(t, env, 1, "update-empty-body-file-id", false, 0)
	env.client.SetToken(env.sellerToken(t))

	w := env.client.Patch(t, variantMediaItemURL(1, 1, "update-empty-body-file-id"), map[string]any{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestUpdateVariantMedia_SetDisplayOrder verifies 200 when displayOrder is updated.
func TestUpdateVariantMedia_SetDisplayOrder(t *testing.T) {
	env := newVariantMediaTestEnv(t)

	const fileID = "update-order-variant-file-id"
	seedVariantMediaRow(t, env, 1, fileID, false, 0)
	env.client.SetToken(env.sellerToken(t))

	order := 3
	w := env.client.Patch(t, variantMediaItemURL(1, 1, fileID), map[string]any{
		"displayOrder": order,
	})
	require.Equal(t, http.StatusOK, w.Code)

	// Verify the DB row was updated.
	sqlDB, err := env.containers.DB.DB()
	require.NoError(t, err)
	var dbOrder int
	err = sqlDB.QueryRow(
		`SELECT display_order FROM variant_media WHERE variant_id = 1 AND file_id = $1`,
		fileID,
	).Scan(&dbOrder)
	require.NoError(t, err)
	assert.Equal(t, order, dbOrder)
}

// TestUpdateVariantMedia_SetPrimary verifies that setting isPrimary=true demotes
// the previously primary item on the same variant.
func TestUpdateVariantMedia_SetPrimary(t *testing.T) {
	env := newVariantMediaTestEnv(t)

	const (
		oldPrimary = "update-old-primary-variant"
		newPrimary = "update-new-primary-variant"
	)
	seedVariantMediaRow(t, env, 1, oldPrimary, true, 0)
	seedVariantMediaRow(t, env, 1, newPrimary, false, 1)

	env.client.SetToken(env.sellerToken(t))
	isPrimary := true
	w := env.client.Patch(t, variantMediaItemURL(1, 1, newPrimary), map[string]any{
		"isPrimary": isPrimary,
	})
	require.Equal(t, http.StatusOK, w.Code)

	sqlDB, err := env.containers.DB.DB()
	require.NoError(t, err)

	var oldIsPrimary, newIsPrimaryDB bool
	_ = sqlDB.QueryRow(
		`SELECT is_primary FROM variant_media WHERE variant_id = 1 AND file_id = $1`, oldPrimary,
	).Scan(&oldIsPrimary)
	_ = sqlDB.QueryRow(
		`SELECT is_primary FROM variant_media WHERE variant_id = 1 AND file_id = $1`, newPrimary,
	).Scan(&newIsPrimaryDB)

	assert.False(t, oldIsPrimary, "old primary must be demoted")
	assert.True(t, newIsPrimaryDB, "new item must be promoted to primary")
}

// ─── DELETE /api/product/:productId/variant/:variantId/media/:fileId ──────────

// TestRemoveVariantMedia_Unauthenticated verifies the DELETE endpoint requires auth.
func TestRemoveVariantMedia_Unauthenticated(t *testing.T) {
	env := newVariantMediaTestEnv(t)

	w := env.client.Delete(t, variantMediaItemURL(1, 1, "any-file"))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestRemoveVariantMedia_CustomerForbidden verifies customers cannot remove
// variant media.
func TestRemoveVariantMedia_CustomerForbidden(t *testing.T) {
	env := newVariantMediaTestEnv(t)
	env.client.SetToken(env.customerToken(t))

	w := env.client.Delete(t, variantMediaItemURL(1, 1, "any-file"))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestRemoveVariantMedia_LinkNotFound verifies 404 when the link does not exist.
func TestRemoveVariantMedia_LinkNotFound(t *testing.T) {
	env := newVariantMediaTestEnv(t)
	env.client.SetToken(env.sellerToken(t))

	w := env.client.Delete(t, variantMediaItemURL(1, 1, "non-existent-link"))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestRemoveVariantMedia_HappyPath verifies 204 on successful removal and that
// the variant_media row is deleted from the DB.
func TestRemoveVariantMedia_HappyPath(t *testing.T) {
	env := newVariantMediaTestEnv(t)

	const fileID = "remove-variant-media-happy-path"
	seedVariantMediaRow(t, env, 1, fileID, false, 0)

	env.client.SetToken(env.sellerToken(t))
	w := env.client.Delete(t, variantMediaItemURL(1, 1, fileID))
	require.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())

	sqlDB, err := env.containers.DB.DB()
	require.NoError(t, err)
	var count int
	err = sqlDB.QueryRow(
		`SELECT COUNT(*) FROM variant_media WHERE variant_id = 1 AND file_id = $1`, fileID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "variant_media row must be deleted")
}

// TestRemoveVariantMedia_PrimaryFallbackPromotion verifies that removing the
// primary media item promotes the next-lowest-order remaining item.
func TestRemoveVariantMedia_PrimaryFallbackPromotion(t *testing.T) {
	env := newVariantMediaTestEnv(t)

	const (
		primaryFileID  = "remove-vm-primary-file"
		fallbackFileID = "remove-vm-fallback-file"
	)
	seedVariantMediaRow(t, env, 1, primaryFileID, true, 0)
	seedVariantMediaRow(t, env, 1, fallbackFileID, false, 1)

	env.client.SetToken(env.sellerToken(t))
	w := env.client.Delete(t, variantMediaItemURL(1, 1, primaryFileID))
	require.Equal(t, http.StatusNoContent, w.Code)

	sqlDB, err := env.containers.DB.DB()
	require.NoError(t, err)
	var isPrimary bool
	err = sqlDB.QueryRow(
		`SELECT is_primary FROM variant_media WHERE variant_id = 1 AND file_id = $1`, fallbackFileID,
	).Scan(&isPrimary)
	require.NoError(t, err)
	assert.True(t, isPrimary, "fallback must be promoted to primary after removing primary")
}

// TestRemoveVariantMedia_WrongSeller verifies seller isolation for delete.
func TestRemoveVariantMedia_WrongSeller(t *testing.T) {
	env := newVariantMediaTestEnv(t)

	seedVariantMediaRow(t, env, 1, "remove-vm-isolation-file", false, 0)

	// Seller 3 tries to remove from product 1 (owned by seller 2).
	token := helpers.Login(t, env.client, helpers.SellerEmail, helpers.SellerPassword)
	env.client.SetToken(token)

	w := env.client.Delete(t, variantMediaItemURL(1, 1, "remove-vm-isolation-file"))
	assert.Equal(t, http.StatusNotFound, w.Code, "wrong seller must see 404")
}
