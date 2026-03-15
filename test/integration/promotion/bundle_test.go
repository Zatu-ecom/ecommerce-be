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

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	promoTypeBundle     = "bundle"
	promoTypePercentage = "percentage_discount"

	bundleDiscountTypePercentage  = "percentage"
	bundleDiscountTypeFixedAmount = "fixed_amount"
	bundleDiscountTypeFixedPrice  = "fixed_price"

	appliesAllProducts = "all_products"

	eligibleEveryone     = "everyone"
	eligibleNewCustomers = "new_customers"

	promoStatusActive = "active"
)

type promotionCreateOptions struct {
	appliesTo   string
	eligibleFor string
	canStack    *bool
	priority    *int
}

type BundlePromotionTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler

	sellerClient   *helpers.APIClient
	customerClient *helpers.APIClient
}

func (s *BundlePromotionTestSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())

	// Keep tests runnable while migration 005 is out of sync with entity.Promotion.
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

func (s *BundlePromotionTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func (s *BundlePromotionTestSuite) SetupTest() {
	s.cleanupPromotions()
}

func TestBundlePromotionStrategy(t *testing.T) {
	suite.Run(t, new(BundlePromotionTestSuite))
}

// TestScenario1_ExactBundleMatchPercentage validates the baseline bundle behavior:
// Given an active 20% bundle for exact required variants, when the cart has both
// items in required quantities, then discount is applied to bundle subtotal only.
func (s *BundlePromotionTestSuite) TestScenario1_ExactBundleMatchPercentage() {
	s.createBundlePromotion(
		"Phone + Headphones 20% Off",
		bundleDiscountTypePercentage,
		helpers.Float64Ptr(20),
		nil,
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(4, 19, 1),
		},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
			cartItem("2", 4, 19, 39999, 1),
		),
	)
	assertPromotionSummary(s.T(), summary, 139899, 27979, 111920)
}

// TestScenario2_ExactBundleMatchFixedPrice validates fixed bundle pricing:
// Given a fixed-price bundle, when all required variants are present, then
// final subtotal equals bundle price and discount equals (standalone - bundle price).
func (s *BundlePromotionTestSuite) TestScenario2_ExactBundleMatchFixedPrice() {
	s.createBundlePromotion(
		"Phone + Samsung Fixed Price",
		bundleDiscountTypeFixedPrice,
		nil,
		helpers.Int64Ptr(150000),
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(2, 5, 1),
		},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
			cartItem("2", 2, 5, 79900, 1),
		),
	)
	assertPromotionSummary(s.T(), summary, 179800, 29800, 150000)
}

// TestScenario2A_FixedPriceMultipleSets validates that the fixed-price discount scales
// correctly with the number of complete bundle sets. Each additional set should reduce
// the subtotal by exactly one bundle_price_cents.
func (s *BundlePromotionTestSuite) TestScenario2A_FixedPriceMultipleSets() {
	s.createBundlePromotion(
		"Phone + Headphones Fixed Price 100k",
		bundleDiscountTypeFixedPrice,
		nil,
		helpers.Int64Ptr(100000),
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(4, 19, 1),
		},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 2),
			cartItem("2", 4, 19, 39999, 2),
		),
	)
	// subtotal: (99900*2) + (39999*2) = 279798
	// completeSets: 2
	// discount: 279798 - (100000 * 2) = 79798
	// final: 200000
	assertPromotionSummary(s.T(), summary, 279798, 79798, 200000)
}

// TestScenario2B_FixedPriceBundlePriceExceedsItemsNoDiscount validates the edge case
// where the configured bundle_price_cents is higher than the actual item total.
// No discount should apply since the "discount" would be negative.
func (s *BundlePromotionTestSuite) TestScenario2B_FixedPriceBundlePriceExceedsItemsNoDiscount() {
	s.createBundlePromotion(
		"Overpriced Fixed Price Bundle",
		bundleDiscountTypeFixedPrice,
		nil,
		helpers.Int64Ptr(200000), // higher than the 139899 item total
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(4, 19, 1),
		},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
			cartItem("2", 4, 19, 39999, 1),
		),
	)
	// bundleTotalCents: 99900 + 39999 = 139899
	// discount: 139899 - 200000 = -60101 → clamped to 0, no promotion applied
	assertPromotionSummary(s.T(), summary, 139899, 0, 139899)
}

// TestScenario2C_FixedAmountSingleSet validates the fixed_amount bundle discount type
// for a single complete set. A flat amount is deducted from the bundle regardless of
// how individual item prices are split.
func (s *BundlePromotionTestSuite) TestScenario2C_FixedAmountSingleSet() {
	s.createBundlePromotion(
		"Phone + Headphones Fixed Amount 200 Off",
		bundleDiscountTypeFixedAmount,
		helpers.Float64Ptr(20000),
		nil,
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(4, 19, 1),
		},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
			cartItem("2", 4, 19, 39999, 1),
		),
	)
	// subtotal: 99900 + 39999 = 139899
	// discount: 20000 * 1 set = 20000
	// final: 119899
	assertPromotionSummary(s.T(), summary, 139899, 20000, 119899)
}

// TestScenario2D_FixedAmountMultipleSets validates that the fixed_amount discount
// multiplies with the number of complete bundle sets found in the cart.
func (s *BundlePromotionTestSuite) TestScenario2D_FixedAmountMultipleSets() {
	s.createBundlePromotion(
		"Phone + Headphones Fixed Amount 200 Off",
		bundleDiscountTypeFixedAmount,
		helpers.Float64Ptr(20000),
		nil,
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(4, 19, 1),
		},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 2),
			cartItem("2", 4, 19, 39999, 2),
		),
	)
	// subtotal: (99900*2) + (39999*2) = 279798
	// completeSets: 2
	// discount: 20000 * 2 sets = 40000
	// final: 239798
	assertPromotionSummary(s.T(), summary, 279798, 40000, 239798)
}

// TestScenario3_BundleItemsPlusOutsideItems validates bundle-target isolation:
// Given bundle + non-bundle items in same cart, only bundle item totals should be
// discounted while outside items remain full price.
func (s *BundlePromotionTestSuite) TestScenario3_BundleItemsPlusOutsideItems() {
	s.createBundlePromotion(
		"Phone + Headphones 20% Off",
		bundleDiscountTypePercentage,
		helpers.Float64Ptr(20),
		nil,
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(4, 19, 1),
		},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
			cartItem("2", 4, 19, 39999, 1),
			cartItem("3", 2, 5, 79900, 1),
		),
	)
	assertPromotionSummary(s.T(), summary, 219799, 27979, 191820)
}

// TestScenario4_MissingBundleItemNoDiscount validates strict completeness:
// If any required bundle item is missing, no bundle discount should be applied.
func (s *BundlePromotionTestSuite) TestScenario4_MissingBundleItemNoDiscount() {
	s.createBundlePromotion(
		"Incomplete Bundle",
		bundleDiscountTypeFixedPrice,
		nil,
		helpers.Int64Ptr(100000),
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(4, 19, 1),
		},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
		),
	)
	assertPromotionSummary(s.T(), summary, 99900, 0, 99900)
}

// TestScenario5_MultipleCompleteBundlesInOneCart validates repeated bundle matching.
// This test intentionally asserts the mathematically expected value for 2 complete sets.
// Current implementation applies bundle only once, so this test fails and exposes the bug.
func (s *BundlePromotionTestSuite) TestScenario5_MultipleCompleteBundlesInOneCart() {
	s.createBundlePromotion(
		"Phone + Headphones 20% Off",
		bundleDiscountTypePercentage,
		helpers.Float64Ptr(20),
		nil,
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(4, 19, 1),
		},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 2),
			cartItem("2", 4, 19, 39999, 2),
		),
	)
	assertPromotionSummary(s.T(), summary, 279798, 55959, 223839)
}

// TestScenario5A_TwoCompleteSetsPlusOneExtraBundleItem validates partial-overflow handling:
// with 3 phones and 2 headphones, only 2 complete sets should be discounted and the extra
// phone should remain full price.
func (s *BundlePromotionTestSuite) TestScenario5A_TwoCompleteSetsPlusOneExtraBundleItem() {
	s.createBundlePromotion(
		"Phone + Headphones 20% Off",
		bundleDiscountTypePercentage,
		helpers.Float64Ptr(20),
		nil,
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(4, 19, 1),
		},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 3),
			cartItem("2", 4, 19, 39999, 2),
		),
	)
	// subtotal: (99900*3) + (39999*2) = 379698
	// discounted sets: 2 => bundle base 279798, 20% discount = 55959
	// final: 379698 - 55959 = 323739
	assertPromotionSummary(s.T(), summary, 379698, 55959, 323739)
}

// TestScenario5B_TwoCompleteSetsPlusOutsideItem validates mixed-cart handling at higher quantities:
// with 2 full bundle sets and one unrelated product, discount must apply only to bundle sets.
func (s *BundlePromotionTestSuite) TestScenario5B_TwoCompleteSetsPlusOutsideItem() {
	s.createBundlePromotion(
		"Phone + Headphones 20% Off",
		bundleDiscountTypePercentage,
		helpers.Float64Ptr(20),
		nil,
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(4, 19, 1),
		},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 2),
			cartItem("2", 4, 19, 39999, 2),
			cartItem("3", 2, 5, 79900, 1), // outside bundle
		),
	)
	// subtotal: 279798 + 79900 = 359698
	// discount on 2 complete sets only: 55959
	// final: 303739
	assertPromotionSummary(s.T(), summary, 359698, 55959, 303739)
}

// TestScenario6_NewVsReturningCustomerEligibility validates eligibleFor=new_customers.
// Note: cart HTTP flow currently hardcodes IsFirstOrder=false; therefore this test
// validates at PromotionService level by controlling CartValidationRequest.IsFirstOrder.
func (s *BundlePromotionTestSuite) TestScenario6_NewVsReturningCustomerEligibility() {
	s.createBundlePromotionWithOptions(
		"New Customers Bundle",
		bundleDiscountTypePercentage,
		helpers.Float64Ptr(20),
		nil,
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(4, 19, 1),
		},
		promotionCreateOptions{
			eligibleFor: eligibleNewCustomers,
		},
	)

	newCustomerSummary := s.applyPromotions(
		buildCartRequestWithFirstOrder(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			true,
			cartItem("1", 1, 1, 99900, 1),
			cartItem("2", 4, 19, 39999, 1),
		),
	)
	assertPromotionSummary(s.T(), newCustomerSummary, 139899, 27979, 111920)

	returningCustomerSummary := s.applyPromotions(
		buildCartRequestWithFirstOrder(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			false,
			cartItem("1", 1, 1, 99900, 1),
			cartItem("2", 4, 19, 39999, 1),
		),
	)
	assertPromotionSummary(s.T(), returningCustomerSummary, 139899, 0, 139899)
}

// TestScenario7_AllProductsScopeStillRequiresBundleItems validates edge behavior:
// even when appliesTo=all_products, bundle logic should still require configured
// bundle_items and must not become a global cart-wide discount.
func (s *BundlePromotionTestSuite) TestScenario7_AllProductsScopeStillRequiresBundleItems() {
	s.createBundlePromotionWithOptions(
		"All Products Scoped Bundle",
		bundleDiscountTypePercentage,
		helpers.Float64Ptr(20),
		nil,
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(4, 19, 1),
		},
		promotionCreateOptions{
			appliesTo: appliesAllProducts,
		},
	)

	// Missing one required bundle item => no discount.
	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
			cartItem("2", 2, 5, 79900, 1),
		),
	)
	assertPromotionSummary(s.T(), summary, 179800, 0, 179800)
}

// TestStackingScenario1_NonStackableBestDiscountWins validates stacking with same priority:
// two non-stackable promotions should result in only one applied (the best discount).
func (s *BundlePromotionTestSuite) TestStackingScenario1_NonStackableBestDiscountWins() {
	s.createBundlePromotion(
		"Bundle 20 NonStack",
		bundleDiscountTypePercentage,
		helpers.Float64Ptr(20),
		nil,
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(4, 19, 1),
		},
	)

	s.createPromotion(
		"Global 10 NonStack",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 10},
		promotionCreateOptions{},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
			cartItem("2", 4, 19, 39999, 1),
		),
	)
	// Best single discount should be bundle(20%) = 27979, not percentage(10%) = 13989.
	assertPromotionSummary(s.T(), summary, 139899, 27979, 111920)
}

// TestStackingScenario1A_SamePriorityNonStackableMaxDiscountWins validates that
// when multiple non-stackable promotions share the same priority, only the best
// discount candidate should apply.
func (s *BundlePromotionTestSuite) TestStackingScenario1A_SamePriorityNonStackableMaxDiscountWins() {
	s.createPromotion(
		"Global 10 NonStack",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 10},
		promotionCreateOptions{},
	)

	s.createPromotion(
		"Global 20 NonStack",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 20},
		promotionCreateOptions{},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
		),
	)
	// 20% should win over 10% at same priority when non-stackable.
	assertPromotionSummary(s.T(), summary, 99900, 19980, 79920)
}

// TestStackingScenario2_StackablePromotionsApplySequentially validates additive stacking:
// when both promotions are stackable, second promotion should compute on effective prices.
func (s *BundlePromotionTestSuite) TestStackingScenario2_StackablePromotionsApplySequentially() {
	s.createBundlePromotionWithOptions(
		"Bundle 20 Stackable",
		bundleDiscountTypePercentage,
		helpers.Float64Ptr(20),
		nil,
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(4, 19, 1),
		},
		promotionCreateOptions{
			canStack: boolPtr(true),
		},
	)

	s.createPromotion(
		"Global 10 Stackable",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 10},
		promotionCreateOptions{
			canStack: boolPtr(true),
		},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 99900, 1),
			cartItem("2", 4, 19, 39999, 1),
		),
	)
	// 20% bundle first => 27979; then percentage strategy applies per-item with integer floor.
	// Current engine result: second-step discount = 11191, so total = 39170, final = 100729.
	assertPromotionSummary(s.T(), summary, 139899, 39170, 100729)
}

// TestStackingScenario2A_DifferentPriorityStackableCarriesForward validates cross-group
// sequential behavior: lower-priority stackable promotions must calculate from prices
// already reduced by higher-priority group promotions.
func (s *BundlePromotionTestSuite) TestStackingScenario2A_DifferentPriorityStackableCarriesForward() {
	s.createPromotion(
		"High Priority 20 Stackable",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 20},
		promotionCreateOptions{
			canStack: boolPtr(true),
			priority: intPtr(10),
		},
	)

	s.createPromotion(
		"Low Priority 10 Stackable",
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
	// 20% first: 19980 => 79920
	// then 10% on 79920: 7992
	// total: 27972, final: 71928
	assertPromotionSummary(s.T(), summary, 99900, 27972, 71928)
}

// TestStackingScenario3_HigherPriorityNonStackableBlocksLowerPriority validates that
// a higher-priority non-stackable promotion terminates further group processing.
func (s *BundlePromotionTestSuite) TestStackingScenario3_HigherPriorityNonStackableBlocksLowerPriority() {
	s.createPromotion(
		"High Priority 5 NonStack",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 5},
		promotionCreateOptions{
			priority: intPtr(10),
		},
	)

	s.createBundlePromotionWithOptions(
		"Low Priority Bundle 20 Stackable",
		bundleDiscountTypePercentage,
		helpers.Float64Ptr(20),
		nil,
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(4, 19, 1),
		},
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
			cartItem("2", 4, 19, 39999, 1),
		),
	)
	// Only high-priority 5% should apply: floor(139899*5/100)=6994.
	assertPromotionSummary(s.T(), summary, 139899, 6994, 132905)
}

// TestStackingScenario3A_DifferentPriorityBothNonStackableFirstPriorityWins validates
// explicit "first priority wins" behavior when both promotions are non-stackable.
func (s *BundlePromotionTestSuite) TestStackingScenario3A_DifferentPriorityBothNonStackableFirstPriorityWins() {
	s.createPromotion(
		"High Priority 5 NonStack",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 5},
		promotionCreateOptions{
			priority: intPtr(10),
		},
	)

	s.createPromotion(
		"Low Priority 20 NonStack",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 20},
		promotionCreateOptions{
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
	// High-priority non-stackable should block lower-priority promotion.
	assertPromotionSummary(s.T(), summary, 99900, 4995, 94905)
}

// TestScenario8_CreateBundleValidation validates create API payload checks.
func (s *BundlePromotionTestSuite) TestScenario8_CreateBundleValidation() {
	validPayload := buildBundlePayload(
		"Valid Bundle",
		bundleDiscountTypePercentage,
		helpers.Float64Ptr(10),
		nil,
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
		},
	)
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, validPayload)
	s.Require().Equal(http.StatusCreated, res.Code)

	invalidPayload := buildBundlePayload(
		"Invalid Bundle Missing Items",
		bundleDiscountTypePercentage,
		helpers.Float64Ptr(10),
		nil,
		[]model.BundleItemConfig{},
	)
	res = s.sellerClient.Post(s.T(), PromotionAPIEndpoint, invalidPayload)
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

// TestScenario10_UnauthorizedCreationRbac validates seller-only promotion creation.
func (s *BundlePromotionTestSuite) TestScenario10_UnauthorizedCreationRbac() {
	payload := buildBundlePayload(
		"Unauthorized Bundle",
		bundleDiscountTypePercentage,
		helpers.Float64Ptr(10),
		nil,
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
		},
	)
	res := s.customerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusForbidden, res.Code)
}

// TestScenario11_SQLInjectionVariantIDsValidation validates request type safety for
// variantIds payload at promotion scope endpoint.
func (s *BundlePromotionTestSuite) TestScenario11_SQLInjectionVariantIDsValidation() {
	promoID := s.createBundlePromotion(
		"Injection Target Promo",
		bundleDiscountTypePercentage,
		helpers.Float64Ptr(10),
		nil,
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
		},
	)

	res := s.sellerClient.Post(
		s.T(),
		PromotionVariantsEndpoint,
		map[string]interface{}{
			"promotionId": promoID,
			"variantIds":  "1 OR 1=1",
		},
	)
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

func (s *BundlePromotionTestSuite) createBundlePromotion(
	name string,
	discountType string,
	discountValue *float64,
	bundlePrice *int64,
	bundleItems []model.BundleItemConfig,
) uint {
	return s.createBundlePromotionWithOptions(
		name,
		discountType,
		discountValue,
		bundlePrice,
		bundleItems,
		promotionCreateOptions{},
	)
}

func (s *BundlePromotionTestSuite) createBundlePromotionWithOptions(
	name string,
	discountType string,
	discountValue *float64,
	bundlePrice *int64,
	bundleItems []model.BundleItemConfig,
	opts promotionCreateOptions,
) uint {
	payload := buildBundlePayload(name, discountType, discountValue, bundlePrice, bundleItems)
	payload["appliesTo"] = defaultString(opts.appliesTo, appliesAllProducts)
	payload["eligibleFor"] = defaultString(opts.eligibleFor, eligibleEveryone)
	if opts.canStack != nil {
		payload["canStackWithOtherPromotions"] = *opts.canStack
	}
	if opts.priority != nil {
		payload["priority"] = *opts.priority
	}

	return s.createPromotionFromPayload(payload)
}

func (s *BundlePromotionTestSuite) createPromotion(
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

func (s *BundlePromotionTestSuite) createPromotionFromPayload(payload map[string]interface{}) uint {
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code)

	respData := helpers.ParseResponse(s.T(), res.Body)
	promo := respData["data"].(map[string]interface{})["promotion"].(map[string]interface{})
	return uint(promo["id"].(float64))
}

func buildBundlePayload(
	name string,
	discountType string,
	discountValue *float64,
	bundlePrice *int64,
	bundleItems []model.BundleItemConfig,
) map[string]interface{} {
	return map[string]interface{}{
		"name":          name,
		"promotionType": promoTypeBundle,
		"discountConfig": model.BundleConfig{
			BundleItems:         bundleItems,
			BundleDiscountType:  model.DiscountType(discountType),
			BundleDiscountValue: discountValue,
			BundlePriceCents:    bundlePrice,
		},
		"appliesTo":   appliesAllProducts,
		"eligibleFor": eligibleEveryone,
		"startsAt":    "2023-01-01T00:00:00Z",
		"endsAt":      "2029-12-31T23:59:59Z",
		"status":      promoStatusActive,
	}
}

func bundleItem(productID, variantID uint, quantity int) model.BundleItemConfig {
	v := variantID
	return model.BundleItemConfig{
		ProductID: productID,
		VariantID: &v,
		Quantity:  quantity,
	}
}

func cartItem(
	itemID string,
	productID uint,
	variantID uint,
	priceCents int64,
	quantity int,
) model.CartItem {
	v := variantID
	total := priceCents * int64(quantity)
	return model.CartItem{
		ItemID:     itemID,
		ProductID:  productID,
		VariantID:  &v,
		Quantity:   quantity,
		PriceCents: priceCents,
		TotalCents: total,
	}
}

func buildCartRequest(
	sellerID uint,
	customerID uint,
	items ...model.CartItem,
) *model.CartValidationRequest {
	var subtotal int64
	for _, item := range items {
		subtotal += item.TotalCents
	}
	return &model.CartValidationRequest{
		SellerID:      sellerID,
		CustomerID:    &customerID,
		IsFirstOrder:  false,
		Items:         items,
		SubtotalCents: subtotal,
	}
}

func buildCartRequestWithFirstOrder(
	sellerID uint,
	customerID uint,
	isFirstOrder bool,
	items ...model.CartItem,
) *model.CartValidationRequest {
	req := buildCartRequest(sellerID, customerID, items...)
	req.IsFirstOrder = isFirstOrder
	return req
}

func (s *BundlePromotionTestSuite) applyPromotions(
	cart *model.CartValidationRequest,
) *model.AppliedPromotionSummary {
	promotionService := singleton.GetInstance().GetPromotionService()
	summary, err := promotionService.ApplyPromotionsToCart(context.Background(), cart)
	s.Require().NoError(err)
	return summary
}

func assertPromotionSummary(
	t *testing.T,
	summary *model.AppliedPromotionSummary,
	subtotal int64,
	discount int64,
	finalSubtotal int64,
) {
	t.Helper()
	require.Equal(t, subtotal, summary.OriginalSubtotal)
	require.Equal(t, discount, summary.TotalDiscountCents)
	require.Equal(t, finalSubtotal, summary.FinalSubtotal)
}

func boolPtr(v bool) *bool {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func (s *BundlePromotionTestSuite) cleanupPromotions() {
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
			Delete(&promotionEntity.PromotionUsage{}).
			Error,
	)
	s.Require().NoError(
		s.container.DB.Where("promotion_id IN ?", promoIDs).
			Delete(&promotionEntity.PromotionProductVariant{}).
			Error,
	)
	s.Require().NoError(
		s.container.DB.Where("promotion_id IN ?", promoIDs).
			Delete(&promotionEntity.PromotionProduct{}).
			Error,
	)
	s.Require().NoError(
		s.container.DB.Where("promotion_id IN ?", promoIDs).
			Delete(&promotionEntity.PromotionCategory{}).
			Error,
	)
	s.Require().NoError(
		s.container.DB.Where("promotion_id IN ?", promoIDs).
			Delete(&promotionEntity.PromotionCollection{}).
			Error,
	)
	s.Require().NoError(
		s.container.DB.Unscoped().
			Where("id IN ?", promoIDs).
			Delete(&promotionEntity.Promotion{}).
			Error,
	)
}
