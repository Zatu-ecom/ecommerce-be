package variant

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Price-cents suite shared setup ─────────────────────────────────────────
// Seller 2 (helpers.Seller2Email) is seeded with base currency INR (2dp,
// currency id 4) and owns products in seller 2's catalog. Seller 3
// (helpers.SellerEmail) is seeded with base currency USD. For JPY (0dp)
// assertions we flip the seller's base currency via seller settings endpoint.

// seedPriceCentsServer spins up a fresh test server + client for the price-cents suite.
func seedPriceCentsServer(t *testing.T) (*setup.TestContainer, *helpers.APIClient) {
	containers := setup.SetupTestContainers(t)
	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
	client := helpers.NewAPIClient(server)
	return containers, client
}

// loginSeller2 logs in as seller 2 (INR base currency) and returns the token.
func loginSeller2(t *testing.T, client *helpers.APIClient) string {
	token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
	client.SetToken(token)
	return token
}

// loginSeller3 logs in as seller 3 (USD base currency).
func loginSeller3(t *testing.T, client *helpers.APIClient) string {
	token := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
	client.SetToken(token)
	return token
}

// switchSellerBaseCurrency updates a seller's base currency to a currency code.
// currencyIDs: 4=INR, 5=JPY (from migrations/seeds/core/002_seed_geo_data.sql).
// The caller's current auth token is preserved.
func switchSellerBaseCurrency(
	t *testing.T,
	client *helpers.APIClient,
	token string,
	currencyID uint,
) {
	if token != "" {
		client.SetToken(token)
	}
	url := "/api/user/seller/settings"
	body := map[string]any{
		"baseCurrencyId": currencyID,
	}
	w := client.Put(t, url, body)
	// Settings update is a seller-only endpoint; allow 200.
	assert.NotEqual(t, http.StatusNotFound, w.Code, "seller settings endpoint should exist")
	assert.Equal(t, http.StatusOK, w.Code, "seller settings update should succeed")
}

// assertMoney asserts a nested Money object shape (amount/amountCents/formatted).
func assertMoney(t *testing.T, node map[string]any) {
	require.NotNil(t, node, "money node should exist")
	_, hasAmount := node["amount"]
	_, hasCents := node["amountCents"]
	_, hasFormatted := node["formatted"]
	assert.True(t, hasAmount, "money.amount present")
	assert.True(t, hasCents, "money.amountCents present")
	assert.True(t, hasFormatted, "money.formatted present")
}

// assertCurrency asserts a parent currency object shape (code/symbol/decimalDigits).
func assertCurrency(t *testing.T, node map[string]any, wantCode string, wantDigits int) {
	require.NotNil(t, node, "currency node should exist")
	assert.Equal(t, wantCode, node["code"])
	assert.Equal(t, float64(wantDigits), node["decimalDigits"])
}

// moneyAmount returns the major-unit amount from a nested Money node.
// The response `price` is now a Money object: {amount, amountCents, formatted}.
func moneyAmount(node any) float64 {
	m, ok := node.(map[string]any)
	if !ok {
		return 0
	}
	amt, _ := m["amount"].(float64)
	return amt
}

// moneyCents returns the minor-unit (cents) value from a nested Money node.
func moneyCents(node any) int64 {
	m, ok := node.(map[string]any)
	if !ok {
		return 0
	}
	cents, _ := m["amountCents"].(float64)
	return int64(cents)
}
