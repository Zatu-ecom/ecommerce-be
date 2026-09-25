package CartTest

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// JPYZeroDecimalSuite verifies zero-decimal currency handling in the cart
// (US2, B3): a JPY-priced variant must NOT be scaled as if it had 2 decimals.
// Seed: seller 2 owns variant 5 (Samsung, 79900 cents). After switching seller 2's
// base currency to JPY (0dp), 79900 cents must stay ¥79900 — NOT ¥799.00.
type JPYZeroDecimalSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler
	customer  *helpers.APIClient
}

func (s *JPYZeroDecimalSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())
	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)

	// Switch seller 2's base currency to JPY (currency id 5, 0dp).
	seller := helpers.NewAPIClient(s.server)
	token := helpers.Login(s.T(), seller, helpers.Seller2Email, helpers.Seller2Password)
	seller.SetToken(token)
	sw := seller.Put(s.T(), "/api/user/seller/settings", map[string]any{"baseCurrencyId": 5})
	helpers.AssertSuccessResponse(s.T(), sw, http.StatusOK)

	s.customer = helpers.NewAPIClient(s.server)
	cToken := helpers.Login(s.T(), s.customer, helpers.CustomerEmail, helpers.CustomerPassword)
	s.customer.SetToken(cToken)
}

func (s *JPYZeroDecimalSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func TestJPYZeroDecimal(t *testing.T) {
	suite.Run(t, new(JPYZeroDecimalSuite))
}

// ─── JPY cart money stays unscaled (B3) ─────────────────────────────────────
// Variant 5 = 79900 cents. With JPY (0dp), unitPrice must be ¥79900 (amountCents 79900,
// amount 79900), never scaled to 7990000.
func (s *JPYZeroDecimalSuite) TestJPYCartMoneyNotScaled() {
	deleteCartByID(s.T(), s.customer)

	body := map[string]any{"items": []map[string]any{{"variantId": 5, "quantity": 1}}}
	w := s.customer.Post(s.T(), CartItemAPIEndpoint, body)
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	gw := s.customer.Get(s.T(), CartAPIEndpoint)
	gr := helpers.AssertSuccessResponse(s.T(), gw, http.StatusOK)
	cart := gr["data"].(map[string]any)

	// Parent currency is JPY with 0 decimal digits.
	helpers.AssertCurrency(s.T(), cart["currency"], "JPY", 0)

	items := cart["items"].([]any)
	assert.Len(s.T(), items, 1)
	item := items[0].(map[string]any)

	// amountCents stays 79900 — no *100 scaling.
	helpers.AssertMoney(s.T(), item["unitPrice"])
	helpers.AssertMoneyCents(s.T(), item["unitPrice"], 79900)
	helpers.AssertMoneyAmount(s.T(), item["unitPrice"], 79900)
}
