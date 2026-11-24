package promotion_test

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/suite"
)

// PromotionMoneySuite verifies seller-currency money handling for promotion
// sale configs (US3). Seller 2 = INR (2dp). discountConfig monetary keys are
// entered as major units and stored as cents; responses expose Money.
type PromotionMoneySuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler
	seller2   *helpers.APIClient
}

func (s *PromotionMoneySuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())
	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)
	s.seller2 = helpers.NewAPIClient(s.server)
	s.seller2.SetToken(helpers.Login(
		s.T(), s.seller2, helpers.Seller2Email, helpers.Seller2Password,
	))
}

func (s *PromotionMoneySuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func TestPromotionMoneyCurrency(t *testing.T) {
	suite.Run(t, new(PromotionMoneySuite))
}

func (s *PromotionMoneySuite) createPromo(payload map[string]any) (map[string]any, int) {
	res := s.seller2.Post(s.T(), "/api/promotion", payload)
	if res.Code == http.StatusCreated {
		resp := helpers.ParseResponse(s.T(), res.Body)
		return resp["data"].(map[string]any)["promotion"].(map[string]any), res.Code
	}
	return nil, res.Code
}

func promoPayload(name, promoType string, config map[string]any) map[string]any {
	return map[string]any{
		"name":           name,
		"promotionType":  promoType,
		"discountConfig": config,
		"appliesTo":      "all_products",
		"eligibleFor":    "everyone",
		"startsAt":       helpers.PastRFC3339(1),
		"endsAt":         helpers.FutureRFC3339(30),
		"status":         "active",
	}
}

// ─── INR fixed_amount config amount 50.00 → stored 5000 (C1) ────────────────
func (s *PromotionMoneySuite) TestINRFixedAmountConfig() {
	payload := promoPayload("INR FIXED 50", "fixed_amount", map[string]any{
		"amount": 50.00,
	})
	promo, code := s.createPromo(payload)
	s.Require().Equal(http.StatusCreated, code)

	// discountConfig.amount is Money: 50.00 → 5000 cents.
	cfg := promo["discountConfig"].(map[string]any)
	helpers.AssertMoney(s.T(), cfg["amount"])
	helpers.AssertMoneyCents(s.T(), cfg["amount"], 5000)
	helpers.AssertMoneyAmount(s.T(), cfg["amount"], 50.00)
}

// ─── minPurchaseAmount/maxDiscountAmount major (C2) ─────────────────────────
func (s *PromotionMoneySuite) TestINRThresholdsMajor() {
	payload := promoPayload("INR THRESHOLD", "percentage_discount", map[string]any{
		"percentage":        10,
		"max_discount":      100,
		"min_order":         1000,
	})
	payload["minPurchaseAmount"] = 1000
	payload["maxDiscountAmount"] = 100
	promo, code := s.createPromo(payload)
	s.Require().Equal(http.StatusCreated, code)

	// Top-level thresholds are Money in major units.
	helpers.AssertMoney(s.T(), promo["minPurchaseAmount"])
	helpers.AssertMoneyCents(s.T(), promo["minPurchaseAmount"], 100000)
	helpers.AssertMoneyAmount(s.T(), promo["minPurchaseAmount"], 1000.00)

	helpers.AssertMoney(s.T(), promo["maxDiscountAmount"])
	helpers.AssertMoneyCents(s.T(), promo["maxDiscountAmount"], 10000)
	helpers.AssertMoneyAmount(s.T(), promo["maxDiscountAmount"], 100.00)
}

// ─── JPY fixed_amount config amount 50 → stored 50, not 5000 (C3) ───────────
func (s *PromotionMoneySuite) TestJPYFixedAmountNotScaled() {
	// Switch seller 2's base currency to JPY (0dp).
	sw := s.seller2.Put(s.T(), "/api/user/seller/settings", map[string]any{"baseCurrencyId": 5})
	s.Require().Equal(http.StatusOK, sw.Code, sw.Body.String())
	// GetSellerDefaultCurrency is cached for 1h; clear it so the switch takes effect.
	s.Require().NoError(s.container.RedisClient.Del(s.T().Context(), "seller_default_currency:2").Err())

	payload := promoPayload("JPY FIXED 50", "fixed_amount", map[string]any{
		"amount": 50,
	})
	promo, code := s.createPromo(payload)
	s.Require().Equal(http.StatusCreated, code)

	cfg := promo["discountConfig"].(map[string]any)
	helpers.AssertMoney(s.T(), cfg["amount"])
	helpers.AssertMoneyCents(s.T(), cfg["amount"], 50)
	helpers.AssertMoneyAmount(s.T(), cfg["amount"], 50)

	// Restore seller 2 to INR and clear the cache so later suites see INR again.
	sw2 := s.seller2.Put(s.T(), "/api/user/seller/settings", map[string]any{"baseCurrencyId": 4})
	s.Require().Equal(http.StatusOK, sw2.Code, sw2.Body.String())
	s.Require().NoError(s.container.RedisClient.Del(s.T().Context(), "seller_default_currency:2").Err())
}

// ─── Percentage stays a plain number (C4) ───────────────────────────────────
func (s *PromotionMoneySuite) TestPercentageStaysPlain() {
	payload := promoPayload("PCT PROMO", "percentage_discount", map[string]any{
		"percentage": 15,
	})
	promo, code := s.createPromo(payload)
	s.Require().Equal(http.StatusCreated, code)

	cfg := promo["discountConfig"].(map[string]any)
	s.Require().Equal(float64(15), cfg["percentage"], "percentage must stay a plain number")
}
