package variant

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
)

// ─── Seller-currency write → DB cents → Money response ─────────────────────
// Scenario A1 (pre-spec §14.3): Seller A (INR) creates variant price 99.99 →
// response Money amount 99.99 / amountCents 9999 / formatted ₹99.99 + currency INR.
func TestPriceCents_INR_VariantWriteAndRead(t *testing.T) {
	containers, client := seedPriceCentsServer(t)
	defer containers.Cleanup(t)
	loginSeller2(t, client) // Seller 2 = INR base currency

	// Product 2 (Samsung) belongs to seller 2 and has Color+Storage options.
	productID := 2
	requestBody := map[string]any{
		"sku":   "PRICE-CENTS-INR-01",
		"price": 99.99,
		"options": []map[string]any{
			{"optionName": "Color", "value": "Marble Gray"},
			{"optionName": "Storage", "value": "128GB"},
		},
	}
	url := fmt.Sprintf("/api/product/%d/variant", productID)
	w := client.Post(t, url, requestBody)
	response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	variant := helpers.GetResponseData(t, response, "variant")

	// Money object on response.
	price := variant["price"].(map[string]any)
	assertMoney(t, price)
	assert.InDelta(t, 99.99, price["amount"].(float64), 0.0001)
	assert.Equal(t, float64(9999), price["amountCents"].(float64))
	assert.Equal(t, "₹99.99", price["formatted"])

	// Parent currency metadata (FR-006).
	assertCurrency(t, variant["currency"].(map[string]any), "INR", 2)
}

// Scenario A2 (pre-spec §14.3): Seller B (JPY) creates variant price 100 →
// response amount 100 / amountCents 100 (not scaled) + currency JPY.
func TestPriceCents_JPY_VariantWriteZeroDecimal(t *testing.T) {
	containers, client := seedPriceCentsServer(t)
	defer containers.Cleanup(t)
	loginSeller2(t, client)

	// Switch seller 2's base currency to JPY (id 5) before pricing.
	switchSellerBaseCurrency(t, client, "", 5)

	// Seller 2 owns product 2 (Samsung, Color+Storage options).
	productID := 2
	requestBody := map[string]any{
		"sku":   "PRICE-CENTS-JPY-01",
		"price": 100,
		"options": []map[string]any{
			{"optionName": "Color", "value": "Cobalt Violet"},
			{"optionName": "Storage", "value": "256GB"},
		},
	}
	url := fmt.Sprintf("/api/product/%d/variant", productID)
	w := client.Post(t, url, requestBody)
	response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	variant := helpers.GetResponseData(t, response, "variant")

	price := variant["price"].(map[string]any)
	assertMoney(t, price)
	assert.InDelta(t, 100, price["amount"].(float64), 0.0001)
	assert.Equal(t, float64(100), price["amountCents"].(float64), "JPY 100 must stay 100, not 10000")
	assert.Equal(t, "¥100", price["formatted"])

	assertCurrency(t, variant["currency"].(map[string]any), "JPY", 0)
}

// ─── Package option write → Money response ──────────────────────────────────
// Scenario A5 (pre-spec §14.3): Package option follows the same rules as variant.
func TestPriceCents_INR_PackageOptionWriteAndRead(t *testing.T) {
	containers, client := seedPriceCentsServer(t)
	defer containers.Cleanup(t)
	loginSeller2(t, client)

	productID := 2
	requestBody := map[string]any{
		"name":        "INR Package",
		"description": "package option",
		"price":       75.00,
		"quantity":    1,
	}
	url := fmt.Sprintf("/api/product/%d/package-option", productID)
	w := client.Post(t, url, requestBody)
	response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	pkg := helpers.GetResponseData(t, response, "packageOption")

	price := pkg["price"].(map[string]any)
	assertMoney(t, price)
	assert.InDelta(t, 75.00, price["amount"].(float64), 0.0001)
	assert.Equal(t, float64(7500), price["amountCents"].(float64))
}
