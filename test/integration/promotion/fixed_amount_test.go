package promotion_test

import (
	"context"
	"net/http"
	"testing"

	promotionEntity "ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/factory/singleton"
	"ecommerce-be/promotion/model"
	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/suite"
)

const (
	promoTypeFixedAmount = "fixed_amount"

	appliesSpecificProducts   = "specific_products"
	appliesSpecificCategories = "specific_categories"
)

type FixedAmountPromotionTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler

	sellerClient   *helpers.APIClient
	customerClient *helpers.APIClient
}

func (s *FixedAmountPromotionTestSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())

	s.Require().NoError(
		s.container.DB.Exec("ALTER TABLE promotion ADD COLUMN IF NOT EXISTS sale_id BIGINT").Error,
	)

	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)

	s.sellerClient = helpers.NewAPIClient(s.server)
	sellerToken := helpers.Login(
		s.T(),
		s.sellerClient,
		helpers.Seller2Email,
		helpers.Seller2Password,
	)
	s.sellerClient.SetToken(sellerToken)

	s.customerClient = helpers.NewAPIClient(s.server)
	customerToken := helpers.Login(
		s.T(),
		s.customerClient,
		helpers.CustomerEmail,
		helpers.CustomerPassword,
	)
	s.customerClient.SetToken(customerToken)
}

func (s *FixedAmountPromotionTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func (s *FixedAmountPromotionTestSuite) SetupTest() {
	s.cleanupPromotions()
}

func TestFixedAmountPromotionStrategy(t *testing.T) {
	suite.Run(t, new(FixedAmountPromotionTestSuite))
}

// ---------------------------------------------------------------------------
// Core Discount Logic
// ---------------------------------------------------------------------------

// TestBasicFixedAmountDiscount validates that a flat fixed amount is subtracted
// from the cart subtotal when the single item exceeds the discount.
func (s *FixedAmountPromotionTestSuite) TestBasicFixedAmountDiscount() {
	s.createFixedAmountPromotion("500 Off", 50000, nil, promotionCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
		),
	)
	// subtotal: 99900, discount: 50000, final: 49900
	assertPromotionSummary(s.T(), summary, 99900, 50000, 49900)
}

// TestFixedAmountExceedsCartTotal validates that when the discount is larger than
// the cart total, the discount is clamped to the cart value (no negative totals).
func (s *FixedAmountPromotionTestSuite) TestFixedAmountExceedsCartTotal() {
	s.createFixedAmountPromotion("1000 Off", 100000, nil, promotionCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 60000, 1),
		),
	)
	// discount clamped to 60000, final = 0
	assertPromotionSummary(s.T(), summary, 60000, 60000, 0)
}

// TestFixedAmountOnSpecificProducts validates that when scope is specific_products,
// the discount only applies to eligible items and non-eligible items stay full price.
func (s *FixedAmountPromotionTestSuite) TestFixedAmountOnSpecificProducts() {
	promoID := s.createFixedAmountPromotion(
		"200 Off Specific Product",
		20000,
		nil,
		promotionCreateOptions{appliesTo: appliesSpecificProducts},
	)
	s.linkPromotionProducts(promoID, 1)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1), // eligible
			cartItem("2", 2, 5, 79900, 1), // not eligible
		),
	)
	// total subtotal: 179800, discount: 20000 (only on item 1), final: 159800
	assertPromotionSummary(s.T(), summary, 179800, 20000, 159800)

	item1 := findItemSummaryByID(s.T(), summary, "1")
	item2 := findItemSummaryByID(s.T(), summary, "2")
	s.Require().Greater(item1.TotalDiscountCents, int64(0), "eligible item should receive discount")
	s.Require().
		Equal(int64(0), item2.TotalDiscountCents, "non-eligible item should have zero discount")
}

// TestFixedAmountMinOrderNotMet validates that the discount is skipped when the
// eligible cart total is below the configured min_order_cents threshold.
func (s *FixedAmountPromotionTestSuite) TestFixedAmountMinOrderNotMet() {
	s.createFixedAmountPromotion(
		"100 Off Min 1000",
		10000,
		helpers.Int64Ptr(100000),
		promotionCreateOptions{},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 80000, 1), // 80000 < 100000 threshold
		),
	)
	assertPromotionSummary(s.T(), summary, 80000, 0, 80000)
}

// TestFixedAmountMinOrderMet validates that the discount applies once the
// eligible cart total meets or exceeds min_order_cents.
func (s *FixedAmountPromotionTestSuite) TestFixedAmountMinOrderMet() {
	s.createFixedAmountPromotion(
		"100 Off Min 1000",
		10000,
		helpers.Int64Ptr(100000),
		promotionCreateOptions{},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 80000, 1),
			cartItem("2", 2, 5, 30000, 1), // total = 110000 >= 100000
		),
	)
	assertPromotionSummary(s.T(), summary, 110000, 10000, 100000)
}

// TestFixedAmountScopedToCategory validates that when scope is specific_categories,
// the discount is computed only from items in the linked category.
func (s *FixedAmountPromotionTestSuite) TestFixedAmountScopedToCategory() {
	promoID := s.createFixedAmountPromotion(
		"300 Off Category 4",
		30000,
		nil,
		promotionCreateOptions{appliesTo: appliesSpecificCategories},
	)
	s.linkPromotionCategories(promoID, 4)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItemWithCategory("1", 1, 1, 4, 99900, 1),  // eligible (cat 4)
			cartItemWithCategory("2", 2, 5, 99, 79900, 1), // not eligible (cat 99)
		),
	)
	// discount: 30000 (only on category-4 item), final: 179800 - 30000 = 149800
	assertPromotionSummary(s.T(), summary, 179800, 30000, 149800)

	item1 := findItemSummaryByID(s.T(), summary, "1")
	item2 := findItemSummaryByID(s.T(), summary, "2")
	s.Require().
		Equal(int64(30000), item1.TotalDiscountCents, "category-4 item should get full discount")
	s.Require().
		Equal(int64(0), item2.TotalDiscountCents, "non-category item should have zero discount")
}

// TestFixedAmountNewCustomerOnly validates eligibleFor=new_customers: first-time
// customers receive the discount, returning customers do not.
func (s *FixedAmountPromotionTestSuite) TestFixedAmountNewCustomerOnly() {
	s.createFixedAmountPromotion(
		"New Customer 200 Off",
		20000,
		nil,
		promotionCreateOptions{eligibleFor: eligibleNewCustomers},
	)

	newCustomerSummary := s.applyPromotions(
		buildCartRequestWithFirstOrder(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			true,
			cartItem("1", 1, 1, 99900, 1),
		),
	)
	assertPromotionSummary(s.T(), newCustomerSummary, 99900, 20000, 79900)

	returningCustomerSummary := s.applyPromotions(
		buildCartRequestWithFirstOrder(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			false,
			cartItem("1", 1, 1, 99900, 1),
		),
	)
	assertPromotionSummary(s.T(), returningCustomerSummary, 99900, 0, 99900)
}

// ---------------------------------------------------------------------------
// Proportional Distribution
// ---------------------------------------------------------------------------

// TestFixedAmountDistributesProportionally validates that a fixed discount is
// split across multiple items in proportion to their line totals, and no cents
// are lost to integer truncation.
func (s *FixedAmountPromotionTestSuite) TestFixedAmountDistributesProportionally() {
	s.createFixedAmountPromotion("300 Off Proportional", 30000, nil, promotionCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 90000, 1), // 90000 of 120000 = 75%
			cartItem("2", 2, 5, 30000, 1), // 30000 of 120000 = 25%
		),
	)
	assertPromotionSummary(s.T(), summary, 120000, 30000, 90000)

	item1 := findItemSummaryByID(s.T(), summary, "1")
	item2 := findItemSummaryByID(s.T(), summary, "2")
	s.Require().Equal(
		int64(30000),
		item1.TotalDiscountCents+item2.TotalDiscountCents,
		"item-level discounts must sum to total discount",
	)
}

// ---------------------------------------------------------------------------
// Multi-item and Quantity
// ---------------------------------------------------------------------------

// TestFixedAmountMultipleQuantitySameItem validates that a single line item with
// quantity > 1 is handled correctly: FinalPriceCents is a line total, so the
// discount applies against (price * quantity).
func (s *FixedAmountPromotionTestSuite) TestFixedAmountMultipleQuantitySameItem() {
	s.createFixedAmountPromotion("300 Off", 30000, nil, promotionCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 50000, 3), // line total = 150000
		),
	)
	// subtotal: 150000, discount: 30000, final: 120000
	assertPromotionSummary(s.T(), summary, 150000, 30000, 120000)
}

// ---------------------------------------------------------------------------
// Stacking and Priority
// ---------------------------------------------------------------------------

// TestNonStackableBestDiscountWins validates that when two non-stackable promos
// compete at the same default priority, the one yielding the largest discount wins.
func (s *FixedAmountPromotionTestSuite) TestNonStackableBestDiscountWins() {
	s.createFixedAmountPromotion("Fixed 500 Off", 50000, nil, promotionCreateOptions{})

	s.createGenericPromotion(
		"Percentage 10 Off",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 10},
		promotionCreateOptions{},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
		),
	)
	// fixed 50000 > percentage 9990, so fixed wins
	assertPromotionSummary(s.T(), summary, 99900, 50000, 49900)
}

// TestStackableFixedAmountThenPercentage validates additive stacking: the
// percentage promo computes on the effective prices after the fixed discount.
func (s *FixedAmountPromotionTestSuite) TestStackableFixedAmountThenPercentage() {
	s.createFixedAmountPromotion(
		"Fixed 200 Stackable",
		20000,
		nil,
		promotionCreateOptions{canStack: boolPtr(true)},
	)

	s.createGenericPromotion(
		"Percentage 10 Stackable",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 10},
		promotionCreateOptions{canStack: boolPtr(true)},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
		),
	)
	// fixed first: 20000 off => effective 79900
	// percentage second: 10% of 79900 = 7990
	// total: 27990, final: 71910
	assertPromotionSummary(s.T(), summary, 99900, 27990, 71910)
}

// TestHigherPriorityNonStackableBlocksLower validates that a higher-priority
// (lower number) non-stackable promotion prevents any lower-priority group
// from being evaluated.
func (s *FixedAmountPromotionTestSuite) TestHigherPriorityNonStackableBlocksLower() {
	s.createFixedAmountPromotion(
		"High Priority Fixed 100",
		10000,
		nil,
		promotionCreateOptions{priority: intPtr(10)},
	)

	s.createGenericPromotion(
		"Low Priority Percentage 50",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 50},
		promotionCreateOptions{
			canStack: boolPtr(true),
			priority: intPtr(100),
		},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
		),
	)
	// non-stackable at priority 10 blocks everything else
	assertPromotionSummary(s.T(), summary, 99900, 10000, 89900)
}

// TestDifferentPriorityStackableCarriesForward validates that a lower-priority
// stackable promotion calculates on effective prices reduced by the higher-priority one.
func (s *FixedAmountPromotionTestSuite) TestDifferentPriorityStackableCarriesForward() {
	s.createFixedAmountPromotion(
		"High Priority Fixed 200",
		20000,
		nil,
		promotionCreateOptions{
			canStack: boolPtr(true),
			priority: intPtr(10),
		},
	)

	s.createGenericPromotion(
		"Low Priority Percentage 10",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 10},
		promotionCreateOptions{
			canStack: boolPtr(true),
			priority: intPtr(100),
		},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
		),
	)
	// priority 10: fixed 20000 => effective 79900
	// priority 100: 10% of 79900 = 7990
	// total: 27990, final: 71910
	assertPromotionSummary(s.T(), summary, 99900, 27990, 71910)
}

// TestSamePriorityNonStackableMaxDiscountWins validates that among multiple
// non-stackable promos at the same priority, only the best discount is applied.
func (s *FixedAmountPromotionTestSuite) TestSamePriorityNonStackableMaxDiscountWins() {
	s.createFixedAmountPromotion("Fixed 500 Off", 50000, nil, promotionCreateOptions{})
	s.createFixedAmountPromotion("Fixed 300 Off", 30000, nil, promotionCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
		),
	)
	// 50000 > 30000, so 50000 wins
	assertPromotionSummary(s.T(), summary, 99900, 50000, 49900)
}

// TestNullPriorityDefaultsToZero validates that a promotion created without an
// explicit priority still applies correctly (defaults to 0).
func (s *FixedAmountPromotionTestSuite) TestNullPriorityDefaultsToZero() {
	s.createFixedAmountPromotion("No Priority Fixed 200", 20000, nil, promotionCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
		),
	)
	assertPromotionSummary(s.T(), summary, 99900, 20000, 79900)
}

// ---------------------------------------------------------------------------
// Negative-path Stacking
// ---------------------------------------------------------------------------

// TestStackableSecondPromoMinOrderNotMetAfterFirstReducesPrices validates that
// when two stackable promos exist but the first discount reduces the effective
// subtotal below the second promo's min_order_cents, the second is skipped.
func (s *FixedAmountPromotionTestSuite) TestStackableSecondPromoMinOrderNotMetAfterFirstReducesPrices() {
	s.createFixedAmountPromotion(
		"Fixed 500 Stackable",
		50000,
		nil,
		promotionCreateOptions{canStack: boolPtr(true)},
	)

	s.createFixedAmountPromotion(
		"Fixed 100 Min 600 Stackable",
		10000,
		helpers.Int64Ptr(60000),
		promotionCreateOptions{canStack: boolPtr(true)},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
		),
	)
	// First promo: 50000 off => effective 49900
	// Second promo: min_order=60000, but effective is 49900 => skipped
	// Only the first discount should apply.
	assertPromotionSummary(s.T(), summary, 99900, 50000, 49900)
	s.Require().Len(summary.AppliedPromotions, 1, "only first promo should apply")
}

// TestNonStackableLoserIsSkippedExplicitly validates that when two non-stackable
// promos compete at the same priority, the losing promo appears in SkippedPromotions.
func (s *FixedAmountPromotionTestSuite) TestNonStackableLoserIsSkippedExplicitly() {
	s.createFixedAmountPromotion("Fixed 500 Off", 50000, nil, promotionCreateOptions{})
	s.createFixedAmountPromotion("Fixed 300 Off", 30000, nil, promotionCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
		),
	)
	// 50000 wins
	assertPromotionSummary(s.T(), summary, 99900, 50000, 49900)
	s.Require().Len(summary.AppliedPromotions, 1, "exactly one promo should apply")
	s.Require().NotEmpty(summary.SkippedPromotions, "losing promo should be in skipped list")
}

// TestBothPromosMinOrderNotMetCartTooSmall validates that when the cart total
// is below the min_order_cents of every available promotion, nothing applies
// and both are skipped.
func (s *FixedAmountPromotionTestSuite) TestBothPromosMinOrderNotMetCartTooSmall() {
	s.createFixedAmountPromotion(
		"Fixed 200 Min 500",
		20000,
		helpers.Int64Ptr(50000),
		promotionCreateOptions{canStack: boolPtr(true)},
	)

	s.createFixedAmountPromotion(
		"Fixed 100 Min 800",
		10000,
		helpers.Int64Ptr(80000),
		promotionCreateOptions{canStack: boolPtr(true)},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 30000, 1), // 30000 < both thresholds
		),
	)
	assertPromotionSummary(s.T(), summary, 30000, 0, 30000)
	s.Require().Empty(summary.AppliedPromotions, "no promo should apply")
}

// ---------------------------------------------------------------------------
// Validation and Security
// ---------------------------------------------------------------------------

// TestCreateValidFixedAmountPromotion validates that a correctly formed payload
// returns HTTP 201 Created.
func (s *FixedAmountPromotionTestSuite) TestCreateValidFixedAmountPromotion() {
	payload := buildFixedAmountPayload("Valid Fixed Amount", 50000, nil)
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code)
}

// TestCreateInvalidFixedAmountNegativeAmount validates that a negative amount_cents
// is rejected with HTTP 400.
func (s *FixedAmountPromotionTestSuite) TestCreateInvalidFixedAmountNegativeAmount() {
	payload := buildFixedAmountPayload("Invalid Negative", -100, nil)
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

// TestCrossTenantFixedAmountDoesNotLeak validates that a promotion created by
// Seller2 does not apply to a cart containing only Seller4 items.
func (s *FixedAmountPromotionTestSuite) TestCrossTenantFixedAmountDoesNotLeak() {
	s.createFixedAmountPromotion("Seller2 Fixed 200", 20000, nil, promotionCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller4UserID,
			helpers.Customer3UserID,
			cartItem("1", 8, 16, 99900, 1),
		),
	)
	assertPromotionSummary(s.T(), summary, 99900, 0, 99900)
}

// TestUnauthorizedFixedAmountCreation validates RBAC: a customer token cannot
// create promotions.
func (s *FixedAmountPromotionTestSuite) TestUnauthorizedFixedAmountCreation() {
	payload := buildFixedAmountPayload("Unauthorized Fixed", 50000, nil)
	res := s.customerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusForbidden, res.Code)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (s *FixedAmountPromotionTestSuite) createFixedAmountPromotion(
	name string,
	amountCents int64,
	minOrderCents *int64,
	opts promotionCreateOptions,
) uint {
	config := model.FixedAmountConfig{
		AmountCents:   amountCents,
		MinOrderCents: minOrderCents,
	}
	return s.createGenericPromotion(name, promoTypeFixedAmount, config, opts)
}

func (s *FixedAmountPromotionTestSuite) createGenericPromotion(
	name string,
	promoType string,
	discountConfig interface{},
	opts promotionCreateOptions,
) uint {
	payload := map[string]interface{}{
		"name":           name,
		"promotionType":  promoType,
		"discountConfig": discountConfig,
		"appliesTo":      defaultString(opts.appliesTo, appliesAllProducts),
		"eligibleFor":    defaultString(opts.eligibleFor, eligibleEveryone),
		"startsAt":       "2023-01-01T00:00:00Z",
		"endsAt":         "2029-12-31T23:59:59Z",
		"status":         promoStatusActive,
	}
	if opts.canStack != nil {
		payload["canStackWithOtherPromotions"] = *opts.canStack
	}
	if opts.priority != nil {
		payload["priority"] = *opts.priority
	}

	return s.createPromotionFromPayload(payload)
}

func (s *FixedAmountPromotionTestSuite) createPromotionFromPayload(
	payload map[string]interface{},
) uint {
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code, "promotion creation should succeed")

	respData := helpers.ParseResponse(s.T(), res.Body)
	promo := respData["data"].(map[string]interface{})["promotion"].(map[string]interface{})
	return uint(promo["id"].(float64))
}

func (s *FixedAmountPromotionTestSuite) applyPromotions(
	cart *model.CartValidationRequest,
) *model.AppliedPromotionSummary {
	promotionService := singleton.GetInstance().GetPromotionService()
	summary, err := promotionService.ApplyPromotionsToCart(context.Background(), cart)
	s.Require().NoError(err)
	return summary
}

func (s *FixedAmountPromotionTestSuite) linkPromotionProducts(
	promotionID uint,
	productIDs ...uint,
) {
	res := s.sellerClient.Post(
		s.T(),
		promotionProductsEndpoint,
		map[string]interface{}{
			"promotionId": promotionID,
			"productIds":  productIDs,
		},
	)
	s.Require().Equal(http.StatusOK, res.Code, "product scope linking should succeed")
}

func (s *FixedAmountPromotionTestSuite) linkPromotionCategories(
	promotionID uint,
	categoryIDs ...uint,
) {
	categories := make([]map[string]interface{}, 0, len(categoryIDs))
	for _, categoryID := range categoryIDs {
		categories = append(categories, map[string]interface{}{
			"categoryId":           categoryID,
			"includeSubcategories": false,
		})
	}

	res := s.sellerClient.Post(
		s.T(),
		promotionCategoriesEndpoint,
		map[string]interface{}{
			"promotionId": promotionID,
			"categories":  categories,
		},
	)
	s.Require().Equal(http.StatusOK, res.Code, "category scope linking should succeed")
}

func buildFixedAmountPayload(
	name string,
	amountCents int64,
	minOrderCents *int64,
) map[string]interface{} {
	config := model.FixedAmountConfig{
		AmountCents:   amountCents,
		MinOrderCents: minOrderCents,
	}
	return map[string]interface{}{
		"name":           name,
		"promotionType":  promoTypeFixedAmount,
		"discountConfig": config,
		"appliesTo":      appliesAllProducts,
		"eligibleFor":    eligibleEveryone,
		"startsAt":       "2023-01-01T00:00:00Z",
		"endsAt":         "2029-12-31T23:59:59Z",
		"status":         promoStatusActive,
	}
}

func (s *FixedAmountPromotionTestSuite) cleanupPromotions() {
	sellerIDs := []uint{helpers.SellerUserID, helpers.Seller2UserID, helpers.Seller4UserID}

	var promoIDs []uint
	err := s.container.DB.
		Model(&promotionEntity.Promotion{}).
		Where("seller_id IN ?", sellerIDs).
		Pluck("id", &promoIDs).Error
	s.Require().NoError(err)

	if len(promoIDs) == 0 {
		return
	}

	s.Require().NoError(
		s.container.DB.Where("promotion_id IN ?", promoIDs).
			Delete(&promotionEntity.PromotionUsage{}).Error,
	)
	s.Require().NoError(
		s.container.DB.Where("promotion_id IN ?", promoIDs).
			Delete(&promotionEntity.PromotionProductVariant{}).Error,
	)
	s.Require().NoError(
		s.container.DB.Where("promotion_id IN ?", promoIDs).
			Delete(&promotionEntity.PromotionProduct{}).Error,
	)
	s.Require().NoError(
		s.container.DB.Where("promotion_id IN ?", promoIDs).
			Delete(&promotionEntity.PromotionCategory{}).Error,
	)
	s.Require().NoError(
		s.container.DB.Where("promotion_id IN ?", promoIDs).
			Delete(&promotionEntity.PromotionCollection{}).Error,
	)
	s.Require().NoError(
		s.container.DB.Unscoped().
			Where("id IN ?", promoIDs).
			Delete(&promotionEntity.Promotion{}).Error,
	)
}
