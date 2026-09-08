package CartTest

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// JPYE2ESuite exercises JPY zero-decimal cart→order continuity (US5, Scenario
// B): seeded JPY-priced variant → cart → order → ¥ amount stays exactly the
// same with no *100 scaling anywhere in the chain.
type JPYE2ESuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler
	seller    *helpers.APIClient
	customer  *helpers.APIClient
}

func (s *JPYE2ESuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())
	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)

	// Switch seller 2's base currency to JPY (id 5, 0dp).
	s.seller = helpers.NewAPIClient(s.server)
	token := helpers.Login(s.T(), s.seller, helpers.Seller2Email, helpers.Seller2Password)
	s.seller.SetToken(token)
	sw := s.seller.Put(s.T(), "/api/user/seller/settings", map[string]any{"baseCurrencyId": 5})
	helpers.AssertSuccessResponse(s.T(), sw, http.StatusOK)

	s.customer = helpers.NewAPIClient(s.server)
	cToken := helpers.Login(s.T(), s.customer, helpers.CustomerEmail, helpers.CustomerPassword)
	s.customer.SetToken(cToken)
}

func (s *JPYE2ESuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func TestJPYE2E(t *testing.T) {
	suite.Run(t, new(JPYE2ESuite))
}

// ─── JPY cart→order E2E: ¥79,900 stays ¥79,900 ─────────────────────────────
// Seeded variant 5 (Samsung, 79900 cents). After switching seller to JPY (0dp),
// this becomes ¥79,900. Add qty 1 → cart → order → verify NO *100 scaling.
func (s *JPYE2ESuite) TestJPY_CartToOrderNoScaling() {
	deleteCartByID(s.T(), s.customer)

	// Add seeded variant 5 (79900 cents) qty 1.
	cartBody := map[string]any{
		"items": []map[string]any{
			{"variantId": 5, "quantity": 1},
		},
	}
	w := s.customer.Post(s.T(), CartItemAPIEndpoint, cartBody)
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	w = s.customer.Get(s.T(), CartAPIEndpoint)
	gr := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	cart := gr["data"].(map[string]any)

	helpers.AssertCurrency(s.T(), cart["currency"], "JPY", 0)

	items := cart["items"].([]any)
	assert.Len(s.T(), items, 1)
	item := items[0].(map[string]any)
	helpers.AssertMoneyCents(s.T(), item["unitPrice"], 79900)
	helpers.AssertMoneyAmount(s.T(), item["unitPrice"], 79900)
	helpers.AssertMoneyCents(s.T(), item["lineTotal"], 79900)

	// Place order.
	orderBody := map[string]any{
		"shippingAddressId": 1,
		"billingAddressId":  1,
		"fulfillmentType":   "delivery",
	}
	w = s.customer.Post(s.T(), "/api/order", orderBody)
	or := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	order := or["data"].(map[string]any)

	helpers.AssertCurrency(s.T(), order["currency"], "JPY", 0)
	helpers.AssertMoneyCents(s.T(), order["subtotal"], 79900)
	helpers.AssertMoneyAmount(s.T(), order["subtotal"], 79900)

	orderItems := order["items"].([]any)
	assert.Len(s.T(), orderItems, 1)
	orderItem := orderItems[0].(map[string]any)
	helpers.AssertMoneyCents(s.T(), orderItem["unitPrice"], 79900)
	helpers.AssertMoneyCents(s.T(), orderItem["lineTotal"], 79900)

	// Cart → order cents continuity.
	cartSubtotalCents := helpers.MoneyCents(cart["summary"].(map[string]any)["subtotal"])
	orderSubtotalCents := helpers.MoneyCents(order["subtotal"])
	assert.Equal(s.T(), cartSubtotalCents, orderSubtotalCents,
		"cart→order subtotal cents must match (JPY zero-decimal)")

	// Sanity: ¥79,900 never becomes ¥7,990,000.
	assert.Equal(s.T(), int64(79900), helpers.MoneyCents(orderItem["unitPrice"]),
		"JPY unitPrice must stay 79900, not 7990000")
}
