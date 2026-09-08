package CartTest

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/suite"
)

// MoneyContractShapeSuite asserts the one-money-language contract (US2/E):
// every money object has amount/amountCents/formatted; parents expose currency
// code/symbol/decimalDigits; no bare unlabeled cent ints remain on the wire.
type MoneyContractShapeSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler
	customer  *helpers.APIClient
}

func (s *MoneyContractShapeSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())
	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)
	s.customer = helpers.NewAPIClient(s.server)
	token := helpers.Login(s.T(), s.customer, helpers.CustomerEmail, helpers.CustomerPassword)
	s.customer.SetToken(token)
}

func (s *MoneyContractShapeSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func TestMoneyContractShape(t *testing.T) {
	suite.Run(t, new(MoneyContractShapeSuite))
}

// assertAllMoneyNodes walks a parsed JSON map and asserts every value that
// looks like a money node (has amountCents) also has amount + formatted.
func assertAllMoneyNodes(t *testing.T, node map[string]any) {
	for k, v := range node {
		switch val := v.(type) {
		case map[string]any:
			if _, hasCents := val["amountCents"]; hasCents {
				helpers.AssertMoney(t, val)
			}
			assertAllMoneyNodes(t, val)
		case []any:
			for _, item := range val {
				if m, ok := item.(map[string]any); ok {
					assertAllMoneyNodes(t, m)
				}
			}
		default:
			_ = k
		}
	}
}

// assertNoBareMoney walks for bare int fields named like money that lack the
// Money shape (unitPrice/lineTotal/subtotal/…Cents top-level ints).
func assertNoBareMoney(t *testing.T, node map[string]any, keys []string) {
	for k, v := range node {
		// The Money object's own sub-fields (amount/amountCents/formatted) are
		// legitimate; only flag bare ints under money-bearing key names.
		if k == "amount" || k == "amountCents" || k == "formatted" {
			continue
		}
		for _, bare := range keys {
			if k == bare {
				// A bare int under a money key is forbidden unless it's the Money object.
				if _, ok := v.(float64); ok {
					t.Errorf("bare money int %q found (expected Money object)", k)
				}
			}
		}
		if m, ok := v.(map[string]any); ok {
			assertNoBareMoney(t, m, keys)
		}
		if arr, ok := v.([]any); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					assertNoBareMoney(t, m, keys)
				}
			}
		}
	}
}

// ─── Cart response has only Money-shaped money fields ───────────────────────
func (s *MoneyContractShapeSuite) TestCartMoneyContractShape() {
	deleteCartByID(s.T(), s.customer)
	body := map[string]any{"items": []map[string]any{{"variantId": 1, "quantity": 1}}}
	w := s.customer.Post(s.T(), CartItemAPIEndpoint, body)
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	gw := s.customer.Get(s.T(), CartAPIEndpoint)
	gr := helpers.AssertSuccessResponse(s.T(), gw, http.StatusOK)
	cart := gr["data"].(map[string]any)

	helpers.AssertCurrency(s.T(), cart["currency"], "INR", 2)
	assertAllMoneyNodes(s.T(), cart)

	// No bare money ints anywhere in the cart payload.
	assertNoBareMoney(s.T(), cart, []string{
		"unitPrice", "lineTotal", "subtotal", "total", "discount",
		"tax", "shipping", "totalPromotionDiscount", "discountedLineTotal",
		"promotionDiscount", "couponDiscount", "totalDiscount", "afterDiscount",
		"potentialDiscount", "potentialSavings",
	})
}
