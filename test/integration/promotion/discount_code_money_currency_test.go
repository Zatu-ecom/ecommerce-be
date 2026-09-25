package promotion_test

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/suite"
)

// DiscountCodeMoneySuite verifies seller-currency money handling for discount
// codes (US3). Seller 2 = INR (2dp) by default; seller 3 = USD. The JPY case
// switches seller 2's base currency to JPY (0dp) for zero-decimal assertions.
type DiscountCodeMoneySuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler
	seller2   *helpers.APIClient
}

func (s *DiscountCodeMoneySuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())
	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)

	s.seller2 = helpers.NewAPIClient(s.server)
	s.seller2.SetToken(helpers.Login(
		s.T(), s.seller2, helpers.Seller2Email, helpers.Seller2Password,
	))
}

func (s *DiscountCodeMoneySuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func TestDiscountCodeMoneyCurrency(t *testing.T) {
	suite.Run(t, new(DiscountCodeMoneySuite))
}

func (s *DiscountCodeMoneySuite) createCode(payload map[string]any) (map[string]any, int) {
	res := s.seller2.Post(s.T(), "/api/promotion/discount-code", payload)
	if res.Code == http.StatusCreated {
		resp := helpers.ParseResponse(s.T(), res.Body)
		return resp["data"].(map[string]any)["discountCode"].(map[string]any), res.Code
	}
	return nil, res.Code
}

// ─── INR fixed_amount 50.00 → stored 5000 cents, Money response (C1) ─────────
func (s *DiscountCodeMoneySuite) TestINRFixedAmountStoredAsCents() {
	payload := map[string]any{
		"code":         "INRFIX50",
		"discountType": "fixed_amount",
		"value":        50.00,
		"appliesTo":    "all_products",
		"startsAt":     helpers.PastRFC3339(1),
		"endsAt":       helpers.FutureRFC3339(30),
	}
	dc, code := s.createCode(payload)
	s.Require().Equal(http.StatusCreated, code)

	// Response value is a Money object (FR-005).
	helpers.AssertMoney(s.T(), dc["value"])
	helpers.AssertMoneyCents(s.T(), dc["value"], 5000)
	helpers.AssertMoneyAmount(s.T(), dc["value"], 50.00)
}

// ─── INR minPurchaseAmount 1000 → stored 100000 cents (C2) ──────────────────
func (s *DiscountCodeMoneySuite) TestINRMinPurchaseThreshold() {
	payload := map[string]any{
		"code":               "INRMIN1000",
		"discountType":       "percentage",
		"value":              10,
		"minPurchaseAmount":  1000,
		"appliesTo":          "all_products",
		"startsAt":           helpers.PastRFC3339(1),
		"endsAt":             helpers.FutureRFC3339(30),
	}
	dc, code := s.createCode(payload)
	s.Require().Equal(http.StatusCreated, code)

	// Threshold is Money: 1000.00 INR → 100000 cents.
	helpers.AssertMoney(s.T(), dc["minPurchaseAmount"])
	helpers.AssertMoneyCents(s.T(), dc["minPurchaseAmount"], 100000)
	helpers.AssertMoneyAmount(s.T(), dc["minPurchaseAmount"], 1000.00)
}

// ─── Percentage stays a plain number (C4) ───────────────────────────────────
func (s *DiscountCodeMoneySuite) TestPercentageStaysPlain() {
	payload := map[string]any{
		"code":         "PCT15",
		"discountType": "percentage",
		"value":        15,
		"appliesTo":    "all_products",
		"startsAt":     helpers.PastRFC3339(1),
		"endsAt":       helpers.FutureRFC3339(30),
	}
	dc, code := s.createCode(payload)
	s.Require().Equal(http.StatusCreated, code)

	// Percentage value stays a plain number (15), NOT a Money object (FR-009).
	s.Require().Equal(float64(15), dc["value"])
}

// ─── JPY fixed_amount 50 → stored 50 cents, not 5000 (C3) ───────────────────
func (s *DiscountCodeMoneySuite) TestJPYFixedAmountNotScaled() {
	// Switch seller 2's base currency to JPY (currency id 5, 0dp).
	sw := s.seller2.Put(s.T(), "/api/user/seller/settings", map[string]any{"baseCurrencyId": 5})
	s.Require().Equal(http.StatusOK, sw.Code, sw.Body.String())
	// GetSellerDefaultCurrency is cached for 1h; clear it so the switch takes effect.
	s.Require().NoError(s.container.RedisClient.Del(s.T().Context(), "seller_default_currency:2").Err())

	payload := map[string]any{
		"code":         "JPYFIX50",
		"discountType": "fixed_amount",
		"value":        50,
		"appliesTo":    "all_products",
		"startsAt":     helpers.PastRFC3339(1),
		"endsAt":       helpers.FutureRFC3339(30),
	}
	dc, code := s.createCode(payload)
	s.Require().Equal(http.StatusCreated, code)

	helpers.AssertMoney(s.T(), dc["value"])
	// JPY 50 → 50 cents, never 5000.
	helpers.AssertMoneyCents(s.T(), dc["value"], 50)
	helpers.AssertMoneyAmount(s.T(), dc["value"], 50)

	// Restore seller 2 to INR and clear the cache so later suites see INR again.
	sw2 := s.seller2.Put(s.T(), "/api/user/seller/settings", map[string]any{"baseCurrencyId": 4})
	s.Require().Equal(http.StatusOK, sw2.Code, sw2.Body.String())
	s.Require().NoError(s.container.RedisClient.Del(s.T().Context(), "seller_default_currency:2").Err())
}

// helper to silence unused fmt in case of future edits
var _ = fmt.Sprintf
