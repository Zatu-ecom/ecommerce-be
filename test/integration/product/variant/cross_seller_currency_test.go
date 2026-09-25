package variant

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
)

// ─── Cross-seller currency isolation (US5, Scenario D/E) ────────────────────
// Variants created by Seller 2 (INR) always return INR currency data,
// independent of which seller reads them or what currency is active.

// TestCrossSeller_INRCreateAndReadback creates a variant in INR and reads
// it back — verifying currency is consistent on write and read.
func TestCrossSeller_INRCreateAndReadback(t *testing.T) {
	containers, client := seedPriceCentsServer(t)
	defer containers.Cleanup(t)
	loginSeller2(t, client)

	body := map[string]any{
		"sku":   "XSELL-INR-001",
		"price": 149.99,
		"options": []map[string]any{
			{"optionName": "Color", "value": "Marble Gray"},
			{"optionName": "Storage", "value": "128GB"},
		},
	}
	w := client.Post(t, fmt.Sprintf("/api/product/%d/variant", 2), body)
	resp := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	variant := helpers.GetResponseData(t, resp, "variant")
	variantID := int(variant["id"].(float64))

	price := variant["price"].(map[string]any)
	assertMoney(t, price)
	assert.InDelta(t, 149.99, price["amount"].(float64), 0.0001)
	assert.Equal(t, float64(14999), price["amountCents"].(float64))
	assertCurrency(t, variant["currency"].(map[string]any), "INR", 2)

	// Re-read the same variant — currency must stay INR.
	readUrl := fmt.Sprintf("/api/product/%d/variant/%d", 2, variantID)
	w = client.Get(t, readUrl)
	resp = helpers.AssertSuccessResponse(t, w, http.StatusOK)
	vRead := helpers.GetResponseData(t, resp, "variant")

	priceReRead := vRead["price"].(map[string]any)
	assertMoney(t, priceReRead)
	assert.InDelta(t, 149.99, priceReRead["amount"].(float64), 0.0001)
	assertCurrency(t, vRead["currency"].(map[string]any), "INR", 2)
}

// TestCrossSeller_JPYCreateAndReadback creates a JPY variant and reads it
// back — verifying ¥500 stays ¥500 (not ¥50000).
func TestCrossSeller_JPYCreateAndReadback(t *testing.T) {
	containers, client := seedPriceCentsServer(t)
	defer containers.Cleanup(t)
	loginSeller2(t, client)
	switchSellerBaseCurrency(t, client, "", 5) // Switch to JPY

	body := map[string]any{
		"sku":   "XSELL-JPY-001",
		"price": 500,
		"options": []map[string]any{
			{"optionName": "Color", "value": "Cobalt Violet"},
			{"optionName": "Storage", "value": "256GB"},
		},
	}
	w := client.Post(t, fmt.Sprintf("/api/product/%d/variant", 2), body)
	resp := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	v := helpers.GetResponseData(t, resp, "variant")

	price := v["price"].(map[string]any)
	assertMoney(t, price)
	assert.InDelta(t, 500, price["amount"].(float64), 0.0001)
	assert.Equal(t, float64(500), price["amountCents"].(float64), "JPY 500 stays 500")
	assertCurrency(t, v["currency"].(map[string]any), "JPY", 0)
}
