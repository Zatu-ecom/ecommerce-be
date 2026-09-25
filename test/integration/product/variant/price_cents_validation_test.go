package variant

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
)

// ─── Excess-precision rejection (FR-002) ────────────────────────────────────
// Scenario A3: JPY seller sends price 100.5 → 400 validation error.
func TestPriceCents_RejectExcessPrecision_JPY(t *testing.T) {
	containers, client := seedPriceCentsServer(t)
	defer containers.Cleanup(t)
	loginSeller2(t, client)

	// Seller 2 → JPY (0 decimal digits).
	switchSellerBaseCurrency(t, client, "", 5)

	productID := 2
	requestBody := map[string]any{
		"sku":   "PRICE-CENTS-JPY-REJECT",
		"price": 100.5,
		"options": []map[string]any{
			{"optionName": "Color", "value": "Marble Gray"},
			{"optionName": "Storage", "value": "256GB"},
		},
	}
	url := fmt.Sprintf("/api/product/%d/variant", productID)
	w := client.Post(t, url, requestBody)

	// Expect a 400 with the standard validation envelope.
	assert.Equal(t, http.StatusBadRequest, w.Code, "JPY excess precision must be rejected")
	resp := helpers.ParseResponse(t, w.Body)
	assert.False(t, resp["success"].(bool), "failure response")
	assert.Equal(t, "VALIDATION_ERROR", resp["code"])
}

// Scenario A4: INR seller sends price 99.999 → 400 validation error.
func TestPriceCents_RejectExcessPrecision_INR(t *testing.T) {
	containers, client := seedPriceCentsServer(t)
	defer containers.Cleanup(t)
	loginSeller2(t, client) // Seller 2 = INR (2dp)

	productID := 2
	requestBody := map[string]any{
		"sku":   "PRICE-CENTS-INR-REJECT",
		"price": 99.999,
		"options": []map[string]any{
			{"optionName": "Color", "value": "Cobalt Violet"},
			{"optionName": "Storage", "value": "128GB"},
		},
	}
	url := fmt.Sprintf("/api/product/%d/variant", productID)
	w := client.Post(t, url, requestBody)

	assert.Equal(t, http.StatusBadRequest, w.Code, "INR excess precision must be rejected")
	resp := helpers.ParseResponse(t, w.Body)
	assert.False(t, resp["success"].(bool), "failure response")
	assert.Equal(t, "VALIDATION_ERROR", resp["code"])
}

// ─── minPrice/maxPrice filters use major units → cents query (A6) ───────────
// Seed variants: product 1 (seller 2) has iPhone at 999.00 (99900) and
// product 2 has Samsung at 799.00 (79900). Filtering minPrice=900 should
// return only the 99900-cents variant.
func TestPriceCents_MinMaxPriceFiltersInMajorUnits(t *testing.T) {
	containers, client := seedPriceCentsServer(t)
	defer containers.Cleanup(t)
	loginSeller2(t, client)
	client.SetHeader("X-Seller-ID", "2")

	// List variants for seller 2 with minPrice=900.00 (major units).
	w := client.Get(t, "/api/product/variant?minPrice=900&maxPrice=1200&allowPurchase=true")
	response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	data := response["data"].(map[string]any)
	variants, ok := data["variants"].([]any)
	assert.True(t, ok, "data.variants should be an array")
	assert.Greater(t, len(variants), 0, "should find variants in 900-1200 range")

	for _, v := range variants {
		variant := v.(map[string]any)
		price := variant["price"].(map[string]any)
		cents := price["amountCents"].(float64)
		// 900.00 → 90000 cents, 1200.00 → 120000 cents.
		assert.GreaterOrEqual(t, cents, float64(90000))
		assert.LessOrEqual(t, cents, float64(120000))
	}
}
