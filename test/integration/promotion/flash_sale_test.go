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

const promoTypeFlashSale = "flash_sale"

type flashSaleOpts struct {
	promotionCreateOptions
	maxDiscountCents *int64
	minOrderCents    *int64
	stockLimit       *int
	soldCount        *int
}

type FlashSalePromotionTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler

	sellerClient   *helpers.APIClient
	customerClient *helpers.APIClient
}

func (s *FlashSalePromotionTestSuite) SetupSuite() {
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

func (s *FlashSalePromotionTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func (s *FlashSalePromotionTestSuite) SetupTest() {
	s.cleanupPromotions()
}

func TestFlashSalePromotionStrategy(t *testing.T) {
	suite.Run(t, new(FlashSalePromotionTestSuite))
}

// ---------------------------------------------------------------------------
// Core Discount Logic (percentage type)
// ---------------------------------------------------------------------------

func (s *FlashSalePromotionTestSuite) TestBasicPercentageFlashSale() {
	s.createFlashSalePromotion("30% Flash Sale", "percentage", 30, flashSaleOpts{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// 30% of 100000 = 30000
	assertPromotionSummary(s.T(), summary, 100000, 30000, 70000)
}

func (s *FlashSalePromotionTestSuite) TestPercentageFlashSaleWithMaxDiscountCap() {
	s.createFlashSalePromotion("50% Flash Capped", "percentage", 50, flashSaleOpts{
		maxDiscountCents: helpers.Int64Ptr(10000),
	})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// 50% of 100000 = 50000, but capped at 10000
	assertPromotionSummary(s.T(), summary, 100000, 10000, 90000)
}

func (s *FlashSalePromotionTestSuite) TestPercentageFlashSaleMultipleItems() {
	s.createFlashSalePromotion("30% Flash Multi", "percentage", 30, flashSaleOpts{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 80000, 1),
			cartItem("2", 2, 5, 40000, 1),
		),
	)
	// 30% of 80000 = 24000; 30% of 40000 = 12000; total = 36000
	assertPromotionSummary(s.T(), summary, 120000, 36000, 84000)

	item1 := findItemSummaryByID(s.T(), summary, "1")
	item2 := findItemSummaryByID(s.T(), summary, "2")
	s.Require().Equal(
		int64(36000),
		item1.TotalDiscountCents+item2.TotalDiscountCents,
		"item-level discounts must sum to total discount",
	)
}

func (s *FlashSalePromotionTestSuite) TestPercentageFlashSaleMultiQuantitySingleItem() {
	s.createFlashSalePromotion("20% Flash Qty", "percentage", 20, flashSaleOpts{})

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

// ---------------------------------------------------------------------------
// Core Discount Logic (fixed_amount type)
// ---------------------------------------------------------------------------

func (s *FlashSalePromotionTestSuite) TestBasicFixedAmountFlashSale() {
	s.createFlashSalePromotion("200 Off Flash", "fixed_amount", 20000, flashSaleOpts{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 80000, 1),
		),
	)
	// 20000 off the single item
	assertPromotionSummary(s.T(), summary, 80000, 20000, 60000)
}

func (s *FlashSalePromotionTestSuite) TestFixedAmountFlashSaleExceedsItemPrice() {
	s.createFlashSalePromotion("Huge Flash", "fixed_amount", 100000, flashSaleOpts{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 30000, 1),
		),
	)
	// 100000 > 30000, clamped to 30000
	assertPromotionSummary(s.T(), summary, 30000, 30000, 0)
}

func (s *FlashSalePromotionTestSuite) TestFixedAmountFlashSaleMultipleItems() {
	s.createFlashSalePromotion("150 Off Per Item Flash", "fixed_amount", 15000, flashSaleOpts{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 80000, 1),
			cartItem("2", 2, 5, 40000, 1),
		),
	)
	// 15000 per item * 2 items = 30000 total
	assertPromotionSummary(s.T(), summary, 120000, 30000, 90000)

	item1 := findItemSummaryByID(s.T(), summary, "1")
	item2 := findItemSummaryByID(s.T(), summary, "2")
	s.Require().Equal(int64(15000), item1.TotalDiscountCents)
	s.Require().Equal(int64(15000), item2.TotalDiscountCents)
}

// ---------------------------------------------------------------------------
// Flash-Sale-Specific: Stock Limit
// ---------------------------------------------------------------------------

func (s *FlashSalePromotionTestSuite) TestFlashSaleStockLimitReached() {
	stockLimit := 50
	soldCount := 50
	s.createFlashSalePromotion("Exhausted Flash", "percentage", 30, flashSaleOpts{
		stockLimit: &stockLimit,
		soldCount:  &soldCount,
	})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	assertPromotionSummary(s.T(), summary, 100000, 0, 100000)
	s.Require().Empty(summary.AppliedPromotions, "stock-exhausted flash sale should not apply")
}

func (s *FlashSalePromotionTestSuite) TestFlashSaleStockAvailable() {
	stockLimit := 50
	soldCount := 10
	s.createFlashSalePromotion("Stock OK Flash", "percentage", 30, flashSaleOpts{
		stockLimit: &stockLimit,
		soldCount:  &soldCount,
	})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// 30% of 100000 = 30000
	assertPromotionSummary(s.T(), summary, 100000, 30000, 70000)
}

func (s *FlashSalePromotionTestSuite) TestFlashSaleNoStockLimit() {
	s.createFlashSalePromotion("No Limit Flash", "percentage", 25, flashSaleOpts{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// 25% of 100000 = 25000
	assertPromotionSummary(s.T(), summary, 100000, 25000, 75000)
}

// ---------------------------------------------------------------------------
// Flash-Sale-Specific: Min Order
// ---------------------------------------------------------------------------

func (s *FlashSalePromotionTestSuite) TestFlashSaleMinOrderMet() {
	s.createFlashSalePromotion("Min Order Flash", "percentage", 20, flashSaleOpts{
		minOrderCents: helpers.Int64Ptr(50000),
	})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 60000, 1),
		),
	)
	// 60000 >= 50000 threshold; 20% of 60000 = 12000
	assertPromotionSummary(s.T(), summary, 60000, 12000, 48000)
}

func (s *FlashSalePromotionTestSuite) TestFlashSaleMinOrderNotMet() {
	s.createFlashSalePromotion("Min Order Flash", "percentage", 20, flashSaleOpts{
		minOrderCents: helpers.Int64Ptr(50000),
	})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 30000, 1),
		),
	)
	// 30000 < 50000 threshold; no discount
	assertPromotionSummary(s.T(), summary, 30000, 0, 30000)
}

// ---------------------------------------------------------------------------
// Scope and Eligibility
// ---------------------------------------------------------------------------

func (s *FlashSalePromotionTestSuite) TestFlashSaleScopedToSpecificProducts() {
	promoID := s.createFlashSalePromotion(
		"Product Flash 30%",
		"percentage", 30,
		flashSaleOpts{
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
	// 30% of 100000 = 30000 (only item 1)
	assertPromotionSummary(s.T(), summary, 180000, 30000, 150000)

	item1 := findItemSummaryByID(s.T(), summary, "1")
	item2 := findItemSummaryByID(s.T(), summary, "2")
	s.Require().Equal(int64(30000), item1.TotalDiscountCents, "eligible item should get discount")
	s.Require().
		Equal(int64(0), item2.TotalDiscountCents, "non-eligible item should have zero discount")
}

func (s *FlashSalePromotionTestSuite) TestFlashSaleScopedToCategory() {
	promoID := s.createFlashSalePromotion(
		"Category Flash 20%",
		"percentage", 20,
		flashSaleOpts{
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

func (s *FlashSalePromotionTestSuite) TestFlashSaleNewCustomerOnly() {
	s.createFlashSalePromotion("New Cust Flash", "percentage", 25, flashSaleOpts{
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
	// 25% of 100000 = 25000
	assertPromotionSummary(s.T(), newCustSummary, 100000, 25000, 75000)

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

func (s *FlashSalePromotionTestSuite) TestNonStackableFlashSaleVsPercentageBestWins() {
	s.createFlashSalePromotion("Flash 40%", "percentage", 40, flashSaleOpts{})

	s.createGenericPromotion(
		"Percentage 10%",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 10},
		promotionCreateOptions{},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// flash 40000 > percentage 10000, flash wins
	assertPromotionSummary(s.T(), summary, 100000, 40000, 60000)
}

func (s *FlashSalePromotionTestSuite) TestStackableFlashSaleThenPercentage() {
	s.createFlashSalePromotion("Flash 20% Stackable", "percentage", 20, flashSaleOpts{
		promotionCreateOptions: promotionCreateOptions{canStack: boolPtr(true)},
	})

	s.createGenericPromotion(
		"Percentage 10% Stackable",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 10},
		promotionCreateOptions{canStack: boolPtr(true)},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// flash first: 20% of 100000 = 20000 => effective 80000
	// percentage second: 10% of 80000 = 8000
	// total: 28000, final: 72000
	assertPromotionSummary(s.T(), summary, 100000, 28000, 72000)
}

func (s *FlashSalePromotionTestSuite) TestHigherPriorityFlashSaleBlocksLower() {
	s.createFlashSalePromotion("High Priority Flash", "percentage", 15, flashSaleOpts{
		promotionCreateOptions: promotionCreateOptions{priority: intPtr(10)},
	})

	s.createGenericPromotion(
		"Low Priority Percentage 50%",
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
	// non-stackable at priority 10 blocks everything else; 15% of 100000 = 15000
	assertPromotionSummary(s.T(), summary, 100000, 15000, 85000)
}

func (s *FlashSalePromotionTestSuite) TestSamePriorityNonStackableMaxDiscountWins() {
	s.createFlashSalePromotion("Flash 40%", "percentage", 40, flashSaleOpts{})
	s.createFlashSalePromotion("Flash 20%", "percentage", 20, flashSaleOpts{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// 40000 > 20000, so 40% flash wins
	assertPromotionSummary(s.T(), summary, 100000, 40000, 60000)
}

// ---------------------------------------------------------------------------
// Negative Paths
// ---------------------------------------------------------------------------

func (s *FlashSalePromotionTestSuite) TestFlashSaleNoEligibleItems() {
	promoID := s.createFlashSalePromotion(
		"Scoped Flash 30%",
		"percentage", 30,
		flashSaleOpts{
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
	s.Require().Empty(summary.AppliedPromotions, "no promo should apply when no items match scope")
}

func (s *FlashSalePromotionTestSuite) TestStackableSecondPromoMinOrderNotMetAfterFlashSale() {
	s.createFlashSalePromotion("Flash 50% Stackable", "percentage", 50, flashSaleOpts{
		promotionCreateOptions: promotionCreateOptions{canStack: boolPtr(true)},
	})

	s.createFlashSalePromotion("Flash 200 Off Min 60000", "fixed_amount", 20000, flashSaleOpts{
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
	// first flash: 50% of 100000 = 50000 => effective 50000
	// second flash: min_order=60000 but effective=50000 => skipped
	assertPromotionSummary(s.T(), summary, 100000, 50000, 50000)
	s.Require().Len(summary.AppliedPromotions, 1, "only first promo should apply")
}

func (s *FlashSalePromotionTestSuite) TestNonStackableLoserSkipped() {
	s.createFlashSalePromotion("Flash 40%", "percentage", 40, flashSaleOpts{})
	s.createFlashSalePromotion("Flash 10%", "percentage", 10, flashSaleOpts{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// 40% wins (40000 > 10000)
	assertPromotionSummary(s.T(), summary, 100000, 40000, 60000)
	s.Require().Len(summary.AppliedPromotions, 1, "exactly one promo should apply")
	s.Require().NotEmpty(summary.SkippedPromotions, "losing promo should be in skipped list")
}

func (s *FlashSalePromotionTestSuite) TestFlashSaleStockLimitAndMinOrderBothFail() {
	stockLimit := 10
	soldCount := 10
	s.createFlashSalePromotion("Exhausted+MinOrder Flash", "percentage", 30, flashSaleOpts{
		stockLimit:    &stockLimit,
		soldCount:     &soldCount,
		minOrderCents: helpers.Int64Ptr(50000),
	})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 30000, 1),
		),
	)
	// stock exhausted => fails immediately before min order check
	assertPromotionSummary(s.T(), summary, 30000, 0, 30000)
	s.Require().Empty(summary.AppliedPromotions)
}

// ---------------------------------------------------------------------------
// Validation and Security
// ---------------------------------------------------------------------------

func (s *FlashSalePromotionTestSuite) TestCreateValidFlashSalePromotion() {
	payload := buildFlashSalePayload("Valid Flash Sale", "percentage", 30, nil, nil, nil, nil)
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code)
}

func (s *FlashSalePromotionTestSuite) TestCreateInvalidFlashSalePercentageOver100() {
	payload := buildFlashSalePayload("Bad Percentage", "percentage", 150, nil, nil, nil, nil)
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

func (s *FlashSalePromotionTestSuite) TestCrossTenantFlashSaleDoesNotLeak() {
	s.createFlashSalePromotion("Seller2 Flash 30%", "percentage", 30, flashSaleOpts{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller4UserID,
			helpers.Customer3UserID,
			cartItem("1", 8, 16, 100000, 1),
		),
	)
	assertPromotionSummary(s.T(), summary, 100000, 0, 100000)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (s *FlashSalePromotionTestSuite) createFlashSalePromotion(
	name string,
	discountType string,
	discountValue float64,
	opts flashSaleOpts,
) uint {
	config := model.FlashSaleConfig{
		DiscountType:     discountType,
		DiscountValue:    discountValue,
		MaxDiscountCents: opts.maxDiscountCents,
		MinOrderCents:    opts.minOrderCents,
		StockLimit:       opts.stockLimit,
		SoldCount:        opts.soldCount,
	}
	return s.createGenericPromotion(name, promoTypeFlashSale, config, opts.promotionCreateOptions)
}

func (s *FlashSalePromotionTestSuite) createGenericPromotion(
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

func (s *FlashSalePromotionTestSuite) createPromotionFromPayload(
	payload map[string]interface{},
) uint {
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code, "promotion creation should succeed")

	respData := helpers.ParseResponse(s.T(), res.Body)
	promo := respData["data"].(map[string]interface{})["promotion"].(map[string]interface{})
	return uint(promo["id"].(float64))
}

func (s *FlashSalePromotionTestSuite) applyPromotions(
	cart *model.CartValidationRequest,
) *model.AppliedPromotionSummary {
	promotionService := singleton.GetInstance().GetPromotionService()
	summary, err := promotionService.ApplyPromotionsToCart(context.Background(), cart)
	s.Require().NoError(err)
	return summary
}

func (s *FlashSalePromotionTestSuite) linkPromotionProducts(
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

func (s *FlashSalePromotionTestSuite) linkPromotionCategories(
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

func buildFlashSalePayload(
	name string,
	discountType string,
	discountValue float64,
	maxDiscountCents *int64,
	minOrderCents *int64,
	stockLimit *int,
	soldCount *int,
) map[string]interface{} {
	config := model.FlashSaleConfig{
		DiscountType:     discountType,
		DiscountValue:    discountValue,
		MaxDiscountCents: maxDiscountCents,
		MinOrderCents:    minOrderCents,
		StockLimit:       stockLimit,
		SoldCount:        soldCount,
	}
	return map[string]interface{}{
		"name":           name,
		"promotionType":  promoTypeFlashSale,
		"discountConfig": config,
		"appliesTo":      appliesAllProducts,
		"eligibleFor":    eligibleEveryone,
		"startsAt":       "2023-01-01T00:00:00Z",
		"endsAt":         "2029-12-31T23:59:59Z",
		"status":         promoStatusActive,
	}
}

func (s *FlashSalePromotionTestSuite) cleanupPromotions() {
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
