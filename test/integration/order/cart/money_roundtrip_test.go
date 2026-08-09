package CartTest

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// MoneyRoundTripSuite exercises the cart → order money continuity for an INR
// seller (US2). Seed: seller 2 = INR (2dp), variant 1 = iPhone 999.00 → 99900 cents.
type MoneyRoundTripSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler
	customer  *helpers.APIClient
}

func (s *MoneyRoundTripSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())
	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)
	s.customer = helpers.NewAPIClient(s.server)
	token := helpers.Login(s.T(), s.customer, helpers.CustomerEmail, helpers.CustomerPassword)
	s.customer.SetToken(token)
}

func (s *MoneyRoundTripSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func TestMoneyRoundTrip(t *testing.T) {
	suite.Run(t, new(MoneyRoundTripSuite))
}

// ─── Cart Money shape + currency (FR-005/FR-006) ────────────────────────────
// Variant 1 = iPhone 999.00 (99900 cents). Add qty 2 → unitPrice 99900,
// lineTotal 199800, summary subtotal 199800.
func (s *MoneyRoundTripSuite) TestCartMoneyShapeAndCurrency() {
	// Clean any prior cart for the customer.
	s.cleanupCarts()

	// Add 2 × variant 1.
	body := map[string]any{"items": []map[string]any{{"variantId": 1, "quantity": 2}}}
	w := s.customer.Post(s.T(), CartItemAPIEndpoint, body)
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	// Get cart.
	gw := s.customer.Get(s.T(), CartAPIEndpoint)
	gr := helpers.AssertSuccessResponse(s.T(), gw, http.StatusOK)
	cart := gr["data"].(map[string]any)

	// Parent currency metadata (INR, 2dp).
	helpers.AssertCurrency(s.T(), cart["currency"], "INR", 2)

	items := cart["items"].([]any)
	assert.Len(s.T(), items, 1, "one cart line")
	item := items[0].(map[string]any)

	// Money objects on the line.
	helpers.AssertMoney(s.T(), item["unitPrice"])
	helpers.AssertMoney(s.T(), item["lineTotal"])
	helpers.AssertMoneyCents(s.T(), item["unitPrice"], 99900)
	helpers.AssertMoneyCents(s.T(), item["lineTotal"], 199800)
	helpers.AssertMoneyAmount(s.T(), item["unitPrice"], 999.00)
	helpers.AssertMoneyAmount(s.T(), item["lineTotal"], 1998.00)

	// Summary money shape.
	summary := cart["summary"].(map[string]any)
	helpers.AssertMoney(s.T(), summary["subtotal"])
	helpers.AssertMoneyCents(s.T(), summary["subtotal"], 199800)
	helpers.AssertMoney(s.T(), summary["total"])
	helpers.AssertMoneyCents(s.T(), summary["total"], 199800)
}

// ─── Cart → Order cents continuity (FR-004) ─────────────────────────────────
// After the cart above, place an order and confirm order item/total cents
// match the cart snapshot.
func (s *MoneyRoundTripSuite) TestCartToOrderCentsContinuity() {
	s.cleanupCarts()

	body := map[string]any{"items": []map[string]any{{"variantId": 1, "quantity": 2}}}
	w := s.customer.Post(s.T(), CartItemAPIEndpoint, body)
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	// Create order (address IDs from seed; seller 2 context).
	orderBody := map[string]any{
		"shippingAddressId": 1,
		"billingAddressId":  1,
		"fulfillmentType":   "delivery",
	}
	ow := s.customer.Post(s.T(), "/api/order", orderBody)
	or := helpers.AssertSuccessResponse(s.T(), ow, http.StatusCreated)
	order := or["data"].(map[string]any)

	// Parent currency on order.
	helpers.AssertCurrency(s.T(), order["currency"], "INR", 2)

	// Order totals match cart (199800 cents subtotal).
	helpers.AssertMoney(s.T(), order["subtotal"])
	helpers.AssertMoneyCents(s.T(), order["subtotal"], 199800)
	helpers.AssertMoney(s.T(), order["total"])

	// Order item money matches cart line.
	items := order["items"].([]any)
	assert.Len(s.T(), items, 1)
	item := items[0].(map[string]any)
	helpers.AssertMoney(s.T(), item["unitPrice"])
	helpers.AssertMoneyCents(s.T(), item["unitPrice"], 99900)
	helpers.AssertMoney(s.T(), item["lineTotal"])
	helpers.AssertMoneyCents(s.T(), item["lineTotal"], 199800)
}

// cleanupCarts deletes the customer's cart so each test starts fresh.
func (s *MoneyRoundTripSuite) cleanupCarts() {
	deleteCartByID(s.T(), s.customer)
}
