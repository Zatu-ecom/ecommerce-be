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

type percentageCreateOptions struct {
	promotionCreateOptions
	maxDiscountCents *int64
	minOrderCents    *int64
}

type PercentageDiscountPromotionTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler

	sellerClient   *helpers.APIClient
	customerClient *helpers.APIClient
}

func (s *PercentageDiscountPromotionTestSuite) SetupSuite() {
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

func (s *PercentageDiscountPromotionTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func (s *PercentageDiscountPromotionTestSuite) SetupTest() {
	s.cleanupPromotions()
}

func TestPercentageDiscountPromotionStrategy(t *testing.T) {
	suite.Run(t, new(PercentageDiscountPromotionTestSuite))
}

// ---------------------------------------------------------------------------
// Core Discount Logic (positive)
// ---------------------------------------------------------------------------

func (s *PercentageDiscountPromotionTestSuite) TestBasicPercentageDiscount() {
	s.createPercentagePromotion("15% Off", 15, percentageCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// 15% of 100000 = 15000
	assertPromotionSummary(s.T(), summary, 100000, 15000, 85000)
}

func (s *PercentageDiscountPromotionTestSuite) TestPercentageWithMaxDiscountCap() {
	s.createPercentagePromotion("15% Capped", 15, percentageCreateOptions{
		maxDiscountCents: helpers.Int64Ptr(10000),
	})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 200000, 1), // raw 15% = 30000, capped at 10000
		),
	)
	assertPromotionSummary(s.T(), summary, 200000, 10000, 190000)
}

func (s *PercentageDiscountPromotionTestSuite) TestPercentageMaxCapExceedsRaw() {
	s.createPercentagePromotion("10% Big Cap", 10, percentageCreateOptions{
		maxDiscountCents: helpers.Int64Ptr(50000),
	})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1), // raw 10% = 10000, cap 50000 has no effect
		),
	)
	assertPromotionSummary(s.T(), summary, 100000, 10000, 90000)
}

func (s *PercentageDiscountPromotionTestSuite) TestPercentageMultipleItemsProportional() {
	s.createPercentagePromotion("20% Off", 20, percentageCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 80000, 1),
			cartItem("2", 2, 5, 40000, 1),
		),
	)
	// 20% of 80000 = 16000; 20% of 40000 = 8000; total = 24000
	assertPromotionSummary(s.T(), summary, 120000, 24000, 96000)

	item1 := findItemSummaryByID(s.T(), summary, "1")
	item2 := findItemSummaryByID(s.T(), summary, "2")
	s.Require().Equal(
		int64(24000),
		item1.TotalDiscountCents+item2.TotalDiscountCents,
		"item-level discounts must sum to total discount",
	)
}

func (s *PercentageDiscountPromotionTestSuite) TestPercentageMultiQuantitySingleItem() {
	s.createPercentagePromotion("20% Off", 20, percentageCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 50000, 3), // line total = 150000
		),
	)
	// 20% of 150000 = 30000
	assertPromotionSummary(s.T(), summary, 150000, 30000, 120000)
}

func (s *PercentageDiscountPromotionTestSuite) TestPercentageMinOrderMet() {
	s.createPercentagePromotion("10% Min 500", 10, percentageCreateOptions{
		minOrderCents: helpers.Int64Ptr(50000),
	})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 60000, 1),
		),
	)
	// 10% of 60000 = 6000
	assertPromotionSummary(s.T(), summary, 60000, 6000, 54000)
}

func (s *PercentageDiscountPromotionTestSuite) TestPercentageMinOrderNotMet() {
	s.createPercentagePromotion("10% Min 500", 10, percentageCreateOptions{
		minOrderCents: helpers.Int64Ptr(50000),
	})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 30000, 1),
		),
	)
	assertPromotionSummary(s.T(), summary, 30000, 0, 30000)
	s.Require().Empty(summary.AppliedPromotions)
}

func (s *PercentageDiscountPromotionTestSuite) TestPercentageMinOrderExactBoundary() {
	s.createPercentagePromotion("10% Min 500", 10, percentageCreateOptions{
		minOrderCents: helpers.Int64Ptr(50000),
	})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 50000, 1), // exactly at threshold
		),
	)
	// 10% of 50000 = 5000
	assertPromotionSummary(s.T(), summary, 50000, 5000, 45000)
}

// ---------------------------------------------------------------------------
// Scope
// ---------------------------------------------------------------------------

func (s *PercentageDiscountPromotionTestSuite) TestPercentageScopedToSpecificProducts() {
	promoID := s.createPercentagePromotion(
		"20% Specific Product",
		20,
		percentageCreateOptions{
			promotionCreateOptions: promotionCreateOptions{appliesTo: appliesSpecificProducts},
		},
	)
	s.linkPromotionProducts(promoID, 1)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1), // eligible
			cartItem("2", 2, 5, 80000, 1),  // not eligible
		),
	)
	// 20% of 100000 = 20000 (only item 1)
	assertPromotionSummary(s.T(), summary, 180000, 20000, 160000)

	item1 := findItemSummaryByID(s.T(), summary, "1")
	item2 := findItemSummaryByID(s.T(), summary, "2")
	s.Require().Equal(int64(20000), item1.TotalDiscountCents)
	s.Require().Equal(int64(0), item2.TotalDiscountCents)
}

func (s *PercentageDiscountPromotionTestSuite) TestPercentageScopedToCategory() {
	promoID := s.createPercentagePromotion(
		"20% Category 4",
		20,
		percentageCreateOptions{
			promotionCreateOptions: promotionCreateOptions{appliesTo: appliesSpecificCategories},
		},
	)
	s.linkPromotionCategories(promoID, 4)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItemWithCategory("1", 1, 1, 4, 100000, 1), // eligible (cat 4)
			cartItemWithCategory("2", 2, 5, 99, 80000, 1), // not eligible (cat 99)
		),
	)
	// 20% of 100000 = 20000 (only category-4 item)
	assertPromotionSummary(s.T(), summary, 180000, 20000, 160000)

	item1 := findItemSummaryByID(s.T(), summary, "1")
	item2 := findItemSummaryByID(s.T(), summary, "2")
	s.Require().Equal(int64(20000), item1.TotalDiscountCents)
	s.Require().Equal(int64(0), item2.TotalDiscountCents)
}

// ---------------------------------------------------------------------------
// Eligibility
// ---------------------------------------------------------------------------

func (s *PercentageDiscountPromotionTestSuite) TestPercentageNewCustomerOnly() {
	s.createPercentagePromotion("15% New Customer", 15, percentageCreateOptions{
		promotionCreateOptions: promotionCreateOptions{eligibleFor: eligibleNewCustomers},
	})

	newCustSummary := s.applyPromotions(
		buildCartRequestWithFirstOrder(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			true,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	assertPromotionSummary(s.T(), newCustSummary, 100000, 15000, 85000)

	returningSummary := s.applyPromotions(
		buildCartRequestWithFirstOrder(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			false,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	assertPromotionSummary(s.T(), returningSummary, 100000, 0, 100000)
}

// ---------------------------------------------------------------------------
// Stacking and Priority
// ---------------------------------------------------------------------------

func (s *PercentageDiscountPromotionTestSuite) TestStackablePercentageThenFixedAmount() {
	s.createPercentagePromotion("20% Stackable", 20, percentageCreateOptions{
		promotionCreateOptions: promotionCreateOptions{canStack: boolPtr(true)},
	})

	s.createGenericPromotion(
		"Fixed 100 Stackable",
		promoTypeFixedAmount,
		model.FixedAmountConfig{AmountCents: 10000},
		promotionCreateOptions{canStack: boolPtr(true)},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// Ranking: percentage 20000 vs fixed 10000 => percentage wins first
	// Then fixed on reduced: 100000 - 20000 = 80000, fixed 10000 off => 70000
	// Total discount: 20000 + 10000 = 30000
	assertPromotionSummary(s.T(), summary, 100000, 30000, 70000)
	s.Require().Len(summary.AppliedPromotions, 2)
}

func (s *PercentageDiscountPromotionTestSuite) TestStackablePercentageUsesReducedPrices() {
	s.createGenericPromotion(
		"Fixed 200 Stackable",
		promoTypeFixedAmount,
		model.FixedAmountConfig{AmountCents: 20000},
		promotionCreateOptions{canStack: boolPtr(true)},
	)

	s.createPercentagePromotion("10% Stackable", 10, percentageCreateOptions{
		promotionCreateOptions: promotionCreateOptions{canStack: boolPtr(true)},
	})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
		),
	)
	// Fixed 20000 wins first (20000 > 9990). Effective = 79900
	// Percentage second: 10% of 79900 = 7990
	// Total: 27990, final: 71910
	assertPromotionSummary(s.T(), summary, 99900, 27990, 71910)
}

func (s *PercentageDiscountPromotionTestSuite) TestNonStackablePercentageVsFixedBestWins() {
	s.createPercentagePromotion("15% Off", 15, percentageCreateOptions{})

	s.createGenericPromotion(
		"Fixed 500 Off",
		promoTypeFixedAmount,
		model.FixedAmountConfig{AmountCents: 50000},
		promotionCreateOptions{},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
		),
	)
	// Fixed 50000 > percentage 14985 => fixed wins
	assertPromotionSummary(s.T(), summary, 99900, 50000, 49900)
}

func (s *PercentageDiscountPromotionTestSuite) TestStackableSecondPromoMinOrderNotMetAfterFirst() {
	s.createPercentagePromotion("50% Stackable", 50, percentageCreateOptions{
		promotionCreateOptions: promotionCreateOptions{canStack: boolPtr(true)},
	})

	s.createPercentagePromotion("10% Min 600 Stackable", 10, percentageCreateOptions{
		promotionCreateOptions: promotionCreateOptions{canStack: boolPtr(true)},
		minOrderCents:          helpers.Int64Ptr(60000),
	})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// First: 50% of 100000 = 50000 => effective 50000
	// Second: min_order=60000, effective=50000 => skipped
	assertPromotionSummary(s.T(), summary, 100000, 50000, 50000)
	s.Require().Len(summary.AppliedPromotions, 1)
}

func (s *PercentageDiscountPromotionTestSuite) TestNonStackableLoserSkipped() {
	s.createPercentagePromotion("30% Off", 30, percentageCreateOptions{})
	s.createPercentagePromotion("10% Off", 10, percentageCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// 30% (30000) > 10% (10000) => 30% wins
	assertPromotionSummary(s.T(), summary, 100000, 30000, 70000)
	s.Require().Len(summary.AppliedPromotions, 1)
	s.Require().NotEmpty(summary.SkippedPromotions, "losing promo should be in skipped list")
}

func (s *PercentageDiscountPromotionTestSuite) TestBothPromosMinOrderNotMet() {
	s.createPercentagePromotion("10% Min 500", 10, percentageCreateOptions{
		promotionCreateOptions: promotionCreateOptions{canStack: boolPtr(true)},
		minOrderCents:          helpers.Int64Ptr(50000),
	})

	s.createPercentagePromotion("20% Min 800", 20, percentageCreateOptions{
		promotionCreateOptions: promotionCreateOptions{canStack: boolPtr(true)},
		minOrderCents:          helpers.Int64Ptr(80000),
	})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 30000, 1),
		),
	)
	assertPromotionSummary(s.T(), summary, 30000, 0, 30000)
	s.Require().Empty(summary.AppliedPromotions)
}

func (s *PercentageDiscountPromotionTestSuite) TestHigherPriorityNonStackableBlocksLower() {
	s.createPercentagePromotion("15% High Priority", 15, percentageCreateOptions{
		promotionCreateOptions: promotionCreateOptions{priority: intPtr(10)},
	})

	s.createGenericPromotion(
		"50% Low Priority",
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
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// Non-stackable at priority 10 blocks lower-priority group
	assertPromotionSummary(s.T(), summary, 100000, 15000, 85000)
	s.Require().Len(summary.AppliedPromotions, 1)
}

func (s *PercentageDiscountPromotionTestSuite) TestDifferentPriorityStackableCarriesForward() {
	s.createGenericPromotion(
		"Fixed 200 High Priority",
		promoTypeFixedAmount,
		model.FixedAmountConfig{AmountCents: 20000},
		promotionCreateOptions{
			canStack: boolPtr(true),
			priority: intPtr(10),
		},
	)

	s.createPercentagePromotion("10% Low Priority", 10, percentageCreateOptions{
		promotionCreateOptions: promotionCreateOptions{
			canStack: boolPtr(true),
			priority: intPtr(100),
		},
	})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
		),
	)
	// Priority 10: fixed 20000 => effective 79900
	// Priority 100: 10% of 79900 = 7990
	// Total: 27990, final: 71910
	assertPromotionSummary(s.T(), summary, 99900, 27990, 71910)
}

func (s *PercentageDiscountPromotionTestSuite) TestSamePriorityNonStackableMaxWins() {
	s.createPercentagePromotion("30% Off", 30, percentageCreateOptions{})
	s.createPercentagePromotion("10% Off", 10, percentageCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// 30000 > 10000 => 30% wins
	assertPromotionSummary(s.T(), summary, 100000, 30000, 70000)
}

func (s *PercentageDiscountPromotionTestSuite) TestNullPriorityDefaults() {
	s.createPercentagePromotion("15% No Priority", 15, percentageCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	assertPromotionSummary(s.T(), summary, 100000, 15000, 85000)
}

// ---------------------------------------------------------------------------
// Negative Paths
// ---------------------------------------------------------------------------

func (s *PercentageDiscountPromotionTestSuite) TestPercentageNoEligibleItems() {
	promoID := s.createPercentagePromotion(
		"20% Scoped",
		20,
		percentageCreateOptions{
			promotionCreateOptions: promotionCreateOptions{appliesTo: appliesSpecificProducts},
		},
	)
	s.linkPromotionProducts(promoID, 1)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 2, 5, 80000, 1), // product 2, not linked
		),
	)
	assertPromotionSummary(s.T(), summary, 80000, 0, 80000)
	s.Require().Empty(summary.AppliedPromotions)
}

func (s *PercentageDiscountPromotionTestSuite) TestPercentageExceeds100Rejected() {
	payload := buildPercentagePayload("Bad Percentage", 150, nil, nil)
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

// ---------------------------------------------------------------------------
// Validation and Security
// ---------------------------------------------------------------------------

func (s *PercentageDiscountPromotionTestSuite) TestCreateValidPercentagePromotion() {
	payload := buildPercentagePayload("Valid 15%", 15, nil, nil)
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code)
}

func (s *PercentageDiscountPromotionTestSuite) TestCreateInvalidPercentageNegativeMinOrder() {
	config := model.PercentageDiscountConfig{
		Percentage:    15,
		MinOrderCents: helpers.Int64Ptr(-1),
	}
	payload := map[string]interface{}{
		"name":           "Negative Min",
		"promotionType":  promoTypePercentage,
		"discountConfig": config,
		"appliesTo":      appliesAllProducts,
		"eligibleFor":    eligibleEveryone,
		"startsAt":       "2023-01-01T00:00:00Z",
		"endsAt":         "2029-12-31T23:59:59Z",
		"status":         promoStatusActive,
	}
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

func (s *PercentageDiscountPromotionTestSuite) TestCrossTenantPercentageDoesNotLeak() {
	s.createPercentagePromotion("Seller2 20%", 20, percentageCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller4UserID,
			helpers.Customer3UserID,
			cartItem("1", 8, 16, 100000, 1),
		),
	)
	assertPromotionSummary(s.T(), summary, 100000, 0, 100000)
}

func (s *PercentageDiscountPromotionTestSuite) TestUnauthorizedPercentageCreation() {
	payload := buildPercentagePayload("Unauthorized", 15, nil, nil)
	res := s.customerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusForbidden, res.Code)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (s *PercentageDiscountPromotionTestSuite) createPercentagePromotion(
	name string,
	percentage float64,
	opts percentageCreateOptions,
) uint {
	config := model.PercentageDiscountConfig{
		Percentage:       percentage,
		MaxDiscountCents: opts.maxDiscountCents,
		MinOrderCents:    opts.minOrderCents,
	}
	return s.createGenericPromotion(name, promoTypePercentage, config, opts.promotionCreateOptions)
}

func (s *PercentageDiscountPromotionTestSuite) createGenericPromotion(
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

func (s *PercentageDiscountPromotionTestSuite) createPromotionFromPayload(
	payload map[string]interface{},
) uint {
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code, "promotion creation should succeed")

	respData := helpers.ParseResponse(s.T(), res.Body)
	promo := respData["data"].(map[string]interface{})["promotion"].(map[string]interface{})
	return uint(promo["id"].(float64))
}

func (s *PercentageDiscountPromotionTestSuite) applyPromotions(
	cart *model.CartValidationRequest,
) *model.AppliedPromotionSummary {
	promotionService := singleton.GetInstance().GetPromotionService()
	summary, err := promotionService.ApplyPromotionsToCart(context.Background(), cart)
	s.Require().NoError(err)
	return summary
}

func (s *PercentageDiscountPromotionTestSuite) linkPromotionProducts(
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

func (s *PercentageDiscountPromotionTestSuite) linkPromotionCategories(
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

func buildPercentagePayload(
	name string,
	percentage float64,
	maxDiscountCents *int64,
	minOrderCents *int64,
) map[string]interface{} {
	config := model.PercentageDiscountConfig{
		Percentage:       percentage,
		MaxDiscountCents: maxDiscountCents,
		MinOrderCents:    minOrderCents,
	}
	return map[string]interface{}{
		"name":           name,
		"promotionType":  promoTypePercentage,
		"discountConfig": config,
		"appliesTo":      appliesAllProducts,
		"eligibleFor":    eligibleEveryone,
		"startsAt":       "2023-01-01T00:00:00Z",
		"endsAt":         "2029-12-31T23:59:59Z",
		"status":         promoStatusActive,
	}
}

func (s *PercentageDiscountPromotionTestSuite) cleanupPromotions() {
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
