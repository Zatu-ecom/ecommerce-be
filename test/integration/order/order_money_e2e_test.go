package order_test

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// OrderMoneyE2ESuite exercises the seeded product→cart→order money continuity
// for an INR seller (US5, Scenario A): add seeded INR variant to cart →
// place order → verify cents match end-to-end with no drift.
type OrderMoneyE2ESuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler
	customer  *helpers.APIClient
}

func (s *OrderMoneyE2ESuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())
	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)

	s.customer = helpers.NewAPIClient(s.server)
	cToken := helpers.Login(s.T(), s.customer, helpers.CustomerEmail, helpers.CustomerPassword)
	s.customer.SetToken(cToken)
}

func (s *OrderMoneyE2ESuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func TestOrderMoneyE2E(t *testing.T) {
	suite.Run(t, new(OrderMoneyE2ESuite))
}

// ─── INR seeded-variant cart→order continuity ───────────────────────────────
// Uses pre-seeded variant 1 (iPhone 999.00 → 99900 cents) to verify
// the read path produces correct Money in both cart and order responses.
func (s *OrderMoneyE2ESuite) TestINR_SeededVariantCartToOrderContinuity() {
	s.destroyTestCart()

	// Add seeded variant 1 (99900 cents) qty 1.
	cartBody := map[string]any{
		"items": []map[string]any{
			{"variantId": 1, "quantity": 1},
		},
	}
	w := s.customer.Post(s.T(), "/api/order/cart/item", cartBody)
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	orderBody := map[string]any{
		"shippingAddressId": 1,
		"billingAddressId":  1,
		"fulfillmentType":   "delivery",
	}
	w = s.customer.Post(s.T(), "/api/order", orderBody)
	or := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	order := or["data"].(map[string]any)

	helpers.AssertMoney(s.T(), order["subtotal"])
	helpers.AssertMoneyCents(s.T(), order["subtotal"], 99900)
	helpers.AssertMoneyAmount(s.T(), order["subtotal"], 999.00)
	helpers.AssertCurrency(s.T(), order["currency"], "INR", 2)

	items := order["items"].([]any)
	assert.Len(s.T(), items, 1)
	item := items[0].(map[string]any)
	helpers.AssertMoneyCents(s.T(), item["unitPrice"], 99900)
	helpers.AssertMoneyCents(s.T(), item["lineTotal"], 99900)
}

// destroyTestCart cleans the customer's cart so each sub-test starts fresh.
func (s *OrderMoneyE2ESuite) destroyTestCart() {
	w := s.customer.Get(s.T(), "/api/order/cart")
	gr := helpers.ParseResponse(s.T(), w.Body)
	if cart, ok := gr["data"].(map[string]any); ok {
		if id, ok := cart["id"].(float64); ok && id > 0 {
			s.customer.Delete(s.T(), fmt.Sprintf("/api/order/cart/%d", int(id)))
		}
	}
}
