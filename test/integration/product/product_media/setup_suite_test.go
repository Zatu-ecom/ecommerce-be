package product_media_test

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// ─── Endpoint constants ───────────────────────────────────────────────────────

const (
	productMediaBasePath = "/api/product/%d/media"
	productMediaItemPath = "/api/product/%d/media/%s"
)

func productMediaURL(productID uint) string {
	return fmt.Sprintf(productMediaBasePath, productID)
}

func productMediaItemURL(productID uint, fileID string) string {
	return fmt.Sprintf(productMediaItemPath, productID, fileID)
}

// ─── Suite setup helper ───────────────────────────────────────────────────────

// mediaTestEnv holds the shared server, API client, and container references for
// a test function. The containers field gives tests direct DB access for seeding
// product_media and file_object rows without going through the HTTP API.
type mediaTestEnv struct {
	client     *helpers.APIClient
	containers *setup.TestContainer
}

// newMediaTestEnv spins up containers, runs migrations and seeds, and returns a
// ready-to-use test environment. Cleanup of containers is registered on t.Cleanup.
func newMediaTestEnv(t *testing.T) *mediaTestEnv {
	t.Helper()

	containers := setup.SetupTestContainers(t)
	t.Cleanup(func() { containers.Cleanup(t) })

	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
	client := helpers.NewAPIClient(server)

	return &mediaTestEnv{
		client:     client,
		containers: containers,
	}
}

// sellerToken logs in as seller 3 (Jane Merchant, owns products 5-7) and returns a bearer token.
func (e *mediaTestEnv) sellerToken(t *testing.T) string {
	t.Helper()
	return helpers.Login(t, e.client, helpers.SellerEmail, helpers.SellerPassword)
}

// seller2Token logs in as seller 2 (John Seller, owns products 1-4) and returns a bearer token.
func (e *mediaTestEnv) seller2Token(t *testing.T) string {
	t.Helper()
	return helpers.Login(t, e.client, helpers.Seller2Email, helpers.Seller2Password)
}

// adminToken logs in as the admin user and returns a bearer token.
func (e *mediaTestEnv) adminToken(t *testing.T) string {
	t.Helper()
	return helpers.Login(t, e.client, helpers.AdminEmail, helpers.AdminPassword)
}

// customerToken logs in as a regular customer and returns a bearer token.
func (e *mediaTestEnv) customerToken(t *testing.T) string {
	t.Helper()
	return helpers.Login(t, e.client, helpers.CustomerEmail, helpers.CustomerPassword)
}

// ─── Assertion helpers ────────────────────────────────────────────────────────

// assertMediaItem checks that a map decoded from JSON satisfies the required
// shape of a ProductMediaResponse.
func assertMediaItem(t *testing.T, item map[string]any) {
	t.Helper()
	assert.NotEmpty(t, item["fileId"], "media item must have fileId")
	assert.NotEmpty(t, item["url"], "media item must have url")
	_, hasPrimary := item["isPrimary"]
	assert.True(t, hasPrimary, "media item must have isPrimary")
	_, hasOrder := item["displayOrder"]
	assert.True(t, hasOrder, "media item must have displayOrder")
}

// assertMediaOrdered verifies that media items are in non-descending displayOrder.
func assertMediaOrdered(t *testing.T, items []any) {
	t.Helper()
	var prev float64 = -1
	for _, raw := range items {
		item := raw.(map[string]any)
		order := item["displayOrder"].(float64)
		assert.GreaterOrEqual(t, order, prev, "media must be ordered by displayOrder ASC")
		prev = order
	}
}

// ─── Smoke test to confirm the suite wires correctly ─────────────────────────

func TestProductMediaSuite_Smoke(t *testing.T) {
	env := newMediaTestEnv(t)
	token := env.sellerToken(t)
	env.client.SetToken(token)

	// A GET on a non-existent product should return 404 or 200 with empty media –
	// either confirms the route is registered and the server is healthy.
	w := env.client.Get(t, productMediaURL(999999))
	assert.Contains(
		t,
		[]int{http.StatusOK, http.StatusNotFound},
		w.Code,
		"product media route should be reachable",
	)
}
