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
	promoTypeBuyXGetY = "buy_x_get_y"

	promotionProductsEndpoint   = "/api/promotion/scope/product"
	promotionCategoriesEndpoint = "/api/promotion/scope/category"
)

type BuyXGetYPromotionTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler

	sellerClient   *helpers.APIClient
	customerClient *helpers.APIClient
}

func (s *BuyXGetYPromotionTestSuite) SetupSuite() {
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

func (s *BuyXGetYPromotionTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func (s *BuyXGetYPromotionTestSuite) SetupTest() {
	s.cleanupPromotions()
}

func TestBuyXGetYPromotionStrategy(t *testing.T) {
	suite.Run(t, new(BuyXGetYPromotionTestSuite))
}

// TestCreateBuyXGetYValidation validates the supported config contract:
// same-reward mode defaults to same_product, cross-reward mode requires get_product_id,
// and invalid field combinations must be rejected with HTTP 400.
func (s *BuyXGetYPromotionTestSuite) TestCreateBuyXGetYValidation() {
	validSameRewardPayload := buildBuyXGetYPayload(
		"Valid Same Reward",
		model.BuyXGetYConfig{
			BuyQuantity: 2,
			GetQuantity: 1,
		},
	)
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, validSameRewardPayload)
	s.Require().
		Equal(http.StatusCreated, res.Code, "default same-reward payload should be accepted")

	validCrossRewardPayload := buildBuyXGetYPayload(
		"Valid Cross Reward",
		model.BuyXGetYConfig{
			BuyQuantity:  1,
			GetQuantity:  1,
			IsSameReward: boolPtr(false),
			GetProductID: uintPtr(4),
		},
	)
	res = s.sellerClient.Post(s.T(), PromotionAPIEndpoint, validCrossRewardPayload)
	s.Require().
		Equal(http.StatusCreated, res.Code, "cross-reward payload with get_product_id should be accepted")

	invalidSameRewardPayload := buildBuyXGetYPayload(
		"Invalid Same Reward",
		model.BuyXGetYConfig{
			BuyQuantity:  1,
			GetQuantity:  1,
			IsSameReward: boolPtr(true),
			ScopeType:    model.BuyXGetYScopeSameProduct,
			GetProductID: uintPtr(4),
		},
	)
	res = s.sellerClient.Post(s.T(), PromotionAPIEndpoint, invalidSameRewardPayload)
	s.Require().
		Equal(http.StatusBadRequest, res.Code, "same-reward payload must reject get_product_id")

	invalidCrossRewardPayload := buildBuyXGetYPayload(
		"Invalid Cross Reward",
		model.BuyXGetYConfig{
			BuyQuantity:  1,
			GetQuantity:  1,
			IsSameReward: boolPtr(false),
			ScopeType:    model.BuyXGetYScopeSameProduct,
		},
	)
	res = s.sellerClient.Post(s.T(), PromotionAPIEndpoint, invalidCrossRewardPayload)
	s.Require().
		Equal(http.StatusBadRequest, res.Code, "cross-reward payload must reject scope_type and require get_product_id")
}

// TestBuyTwoGetOneFreeSameProduct validates the default same-product reward behavior.
// The third unit of the same product should become free, and the line's effective price
// must carry that discount forward for summaries and later stacked promotions.
func (s *BuyXGetYPromotionTestSuite) TestBuyTwoGetOneFreeSameProduct() {
	s.createBuyXGetYPromotion(
		"Buy 2 Get 1 Free",
		model.BuyXGetYConfig{
			BuyQuantity: 2,
			GetQuantity: 1,
		},
		promotionCreateOptions{},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("phone-line", 1, 1, 99900, 3),
		),
	)

	assertPromotionSummary(s.T(), summary, 299700, 99900, 199800)
	require.Len(s.T(), summary.AppliedPromotions, 1, "exactly one BxGy promotion should apply")

	phoneLine := findItemSummaryByID(s.T(), summary, "phone-line")
	require.Equal(s.T(), int64(99900), phoneLine.TotalDiscountCents, "one unit should be free")
	require.Equal(
		s.T(),
		int64(199800),
		phoneLine.FinalPriceCents,
		"effective line total should be reduced for later stacking",
	)
	require.Len(s.T(), phoneLine.AppliedPromotions, 1, "line should show one applied BxGy detail")
	require.Equal(
		s.T(),
		1,
		phoneLine.AppliedPromotions[0].FreeQuantity,
		"free quantity should be tracked at item level",
	)
}

// TestBuyTwoGetOneFreeSameVariant validates variant-level grouping.
// Different variants of the same product must not combine when scope_type is same_variant.
func (s *BuyXGetYPromotionTestSuite) TestBuyTwoGetOneFreeSameVariant() {
	s.createBuyXGetYPromotion(
		"Buy 2 Get 1 Same Variant",
		model.BuyXGetYConfig{
			BuyQuantity:  2,
			GetQuantity:  1,
			IsSameReward: boolPtr(true),
			ScopeType:    model.BuyXGetYScopeSameVariant,
		},
		promotionCreateOptions{},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("s24-128", 2, 5, 79900, 2),
			cartItem("s24-256", 2, 6, 89900, 1),
		),
	)

	assertPromotionSummary(s.T(), summary, 249700, 0, 249700)
	require.Empty(
		s.T(),
		summary.AppliedPromotions,
		"different variants must not pool under same_variant scope",
	)
}

// TestBuyTwoGetOneFreeSameProductAcrossVariants validates product-level pooling.
// Variants of the same product should combine when scope_type is same_product, and the cheapest
// eligible unit should become free to protect seller margin.
func (s *BuyXGetYPromotionTestSuite) TestBuyTwoGetOneFreeSameProductAcrossVariants() {
	s.createBuyXGetYPromotion(
		"Buy 2 Get 1 Same Product",
		model.BuyXGetYConfig{
			BuyQuantity:  2,
			GetQuantity:  1,
			IsSameReward: boolPtr(true),
			ScopeType:    model.BuyXGetYScopeSameProduct,
		},
		promotionCreateOptions{},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("s24-128", 2, 5, 79900, 1),
			cartItem("s24-256", 2, 6, 89900, 2),
		),
	)

	assertPromotionSummary(s.T(), summary, 259700, 79900, 179800)
	cheapVariant := findItemSummaryByID(s.T(), summary, "s24-128")
	require.Equal(
		s.T(),
		int64(79900),
		cheapVariant.TotalDiscountCents,
		"cheapest eligible variant should be free under same_product",
	)
	require.Equal(
		s.T(),
		int64(0),
		cheapVariant.FinalPriceCents,
		"free single-quantity line should drop to zero",
	)
}

// TestBuyTwoGetOneFreeSameCategory validates category-level pooling.
// Eligible lines in the same category should combine, and the cheapest category item should be free.
func (s *BuyXGetYPromotionTestSuite) TestBuyTwoGetOneFreeSameCategory() {
	promoID := s.createBuyXGetYPromotion(
		"Phones Buy 2 Get 1",
		model.BuyXGetYConfig{
			BuyQuantity:  2,
			GetQuantity:  1,
			IsSameReward: boolPtr(true),
			ScopeType:    model.BuyXGetYScopeSameCategory,
		},
		promotionCreateOptions{
			appliesTo: "specific_categories",
		},
	)
	s.linkPromotionCategories(promoID, 4)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItemWithCategory("iphone-line", 1, 1, 4, 99900, 1),
			cartItemWithCategory("s24-128-line", 2, 5, 4, 79900, 1),
			cartItemWithCategory("s24-256-line", 2, 6, 4, 89900, 1),
		),
	)

	assertPromotionSummary(s.T(), summary, 269700, 79900, 189800)
	cheapestLine := findItemSummaryByID(s.T(), summary, "s24-128-line")
	require.Equal(
		s.T(),
		int64(79900),
		cheapestLine.TotalDiscountCents,
		"cheapest category item should be discounted",
	)
}

// TestBuyOneGetSpecificProductFree validates cross-product reward mode.
// The buy side comes from the scoped product, and the configured reward product becomes free only
// if it is actually present in the cart.
func (s *BuyXGetYPromotionTestSuite) TestBuyOneGetSpecificProductFree() {
	promoID := s.createBuyXGetYPromotion(
		"Buy Phone Get Headphones Free",
		model.BuyXGetYConfig{
			BuyQuantity:  1,
			GetQuantity:  1,
			IsSameReward: boolPtr(false),
			GetProductID: uintPtr(4),
		},
		promotionCreateOptions{
			appliesTo: "specific_products",
		},
	)
	s.linkPromotionProducts(promoID, 1)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("phone-line", 1, 1, 99900, 1),
			cartItem("headphone-line", 4, 19, 39999, 1),
		),
	)

	assertPromotionSummary(s.T(), summary, 139899, 39999, 99900)
	headphones := findItemSummaryByID(s.T(), summary, "headphone-line")
	require.Equal(
		s.T(),
		int64(39999),
		headphones.TotalDiscountCents,
		"configured reward product should be free",
	)
	require.Equal(
		s.T(),
		int64(0),
		headphones.FinalPriceCents,
		"free reward line should drop to zero",
	)
}

// TestRewardProductMissingDoesNotDiscount validates that the engine does not auto-add or auto-reward
// a missing cross-product reward item. The customer must explicitly have the reward product in cart.
func (s *BuyXGetYPromotionTestSuite) TestRewardProductMissingDoesNotDiscount() {
	promoID := s.createBuyXGetYPromotion(
		"Buy Phone Get Headphones Free",
		model.BuyXGetYConfig{
			BuyQuantity:  1,
			GetQuantity:  1,
			IsSameReward: boolPtr(false),
			GetProductID: uintPtr(4),
		},
		promotionCreateOptions{
			appliesTo: "specific_products",
		},
	)
	s.linkPromotionProducts(promoID, 1)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("phone-line", 1, 1, 99900, 1),
		),
	)

	assertPromotionSummary(s.T(), summary, 99900, 0, 99900)
	require.Empty(
		s.T(),
		summary.AppliedPromotions,
		"missing reward product should produce no discount",
	)
}

// TestBuyXGetYHonorsMaxSetsLimit validates max_sets at both summary and item level.
// With quantity 6 and max_sets=2 on Buy 1 Get 1, only 2 units should be free.
func (s *BuyXGetYPromotionTestSuite) TestBuyXGetYHonorsMaxSetsLimit() {
	maxSets := 2
	s.createBuyXGetYPromotion(
		"Buy 1 Get 1 Max 2",
		model.BuyXGetYConfig{
			BuyQuantity: 1,
			GetQuantity: 1,
			MaxSets:     &maxSets,
		},
		promotionCreateOptions{},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("phone-line", 1, 1, 99900, 6),
		),
	)

	assertPromotionSummary(s.T(), summary, 599400, 199800, 399600)
	phoneLine := findItemSummaryByID(s.T(), summary, "phone-line")
	require.Equal(
		s.T(),
		int64(199800),
		phoneLine.TotalDiscountCents,
		"max_sets should limit the total free quantity",
	)
	require.Equal(
		s.T(),
		2,
		phoneLine.AppliedPromotions[0].FreeQuantity,
		"free quantity should reflect the cap",
	)
}

// TestStackableBuyXGetYThenPercentageUsesReducedPrices validates sequential stacking.
// After BxGy applies, the later percentage promotion must calculate from the reduced effective price.
func (s *BuyXGetYPromotionTestSuite) TestStackableBuyXGetYThenPercentageUsesReducedPrices() {
	s.createBuyXGetYPromotion(
		"Stackable Buy 2 Get 1",
		model.BuyXGetYConfig{
			BuyQuantity: 2,
			GetQuantity: 1,
		},
		promotionCreateOptions{
			canStack: boolPtr(true),
		},
	)
	s.createPromotion(
		"Stackable 10 Percent",
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
			cartItem("phone-line", 1, 1, 99900, 3),
		),
	)

	assertPromotionSummary(s.T(), summary, 299700, 119880, 179820)
	require.Len(s.T(), summary.AppliedPromotions, 2, "both stackable promotions should apply")

	phoneLine := findItemSummaryByID(s.T(), summary, "phone-line")
	require.Equal(
		s.T(),
		int64(179820),
		phoneLine.FinalPriceCents,
		"later promotions should see the reduced effective line total",
	)
	require.Equal(
		s.T(),
		int64(119880),
		phoneLine.TotalDiscountCents,
		"line discount should accumulate both promotions",
	)
}

// TestBuyXGetYAndPercentageBestDiscountWins validates non-stackable conflict resolution.
// When both BxGy and a percentage discount are eligible, only the better one should apply.
func (s *BuyXGetYPromotionTestSuite) TestBuyXGetYAndPercentageBestDiscountWins() {
	s.createBuyXGetYPromotion(
		"Buy 2 Get 1 Free",
		model.BuyXGetYConfig{
			BuyQuantity: 2,
			GetQuantity: 1,
		},
		promotionCreateOptions{},
	)
	s.createPromotion(
		"Global 15 Percent",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 15},
		promotionCreateOptions{},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("phone-line", 1, 1, 99900, 3),
		),
	)

	assertPromotionSummary(s.T(), summary, 299700, 99900, 199800)
	assertSingleAppliedPromotionName(s.T(), summary, "Buy 2 Get 1 Free")
}

// TestLowerNumberPriorityWinsAcrossNonStackablePromotions validates the updated priority contract:
// lower numeric value means higher priority (P0 > P1 > P2). Even when a lower-priority promotion
// yields a larger discount, a higher-priority non-stackable promotion must win and stop evaluation.
func (s *BuyXGetYPromotionTestSuite) TestLowerNumberPriorityWinsAcrossNonStackablePromotions() {
	s.createBuyXGetYPromotion(
		"P0 Buy 2 Get 1 Free",
		model.BuyXGetYConfig{
			BuyQuantity: 2,
			GetQuantity: 1,
		},
		promotionCreateOptions{
			priority: intPtr(0),
		},
	)
	s.createPromotion(
		"P1 Global 50 Percent",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 50},
		promotionCreateOptions{
			priority: intPtr(1),
		},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("phone-line", 1, 1, 99900, 3),
		),
	)

	assertPromotionSummary(s.T(), summary, 299700, 99900, 199800)
	assertSingleAppliedPromotionName(s.T(), summary, "P0 Buy 2 Get 1 Free")
}

// TestCrossProductRewardHonorsMaxSetsWithMultipleQuantities validates cross-product reward capping.
// When both buy and reward quantities can form multiple sets, max_sets must cap free rewards exactly.
func (s *BuyXGetYPromotionTestSuite) TestCrossProductRewardHonorsMaxSetsWithMultipleQuantities() {
	maxSets := 1
	promoID := s.createBuyXGetYPromotion(
		"Buy Phone Get Headphones Free Max1",
		model.BuyXGetYConfig{
			BuyQuantity:  1,
			GetQuantity:  1,
			MaxSets:      &maxSets,
			IsSameReward: boolPtr(false),
			GetProductID: uintPtr(4),
		},
		promotionCreateOptions{
			appliesTo: "specific_products",
		},
	)
	s.linkPromotionProducts(promoID, 1)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("phone-line", 1, 1, 99900, 3),
			cartItem("headphone-line", 4, 19, 39999, 2),
		),
	)

	assertPromotionSummary(s.T(), summary, 379698, 39999, 339699)
	headphones := findItemSummaryByID(s.T(), summary, "headphone-line")
	require.Equal(
		s.T(),
		int64(39999),
		headphones.TotalDiscountCents,
		"max_sets=1 must cap reward discount to one unit",
	)
	require.Equal(
		s.T(),
		1,
		headphones.AppliedPromotions[0].FreeQuantity,
		"free quantity should match capped reward sets",
	)
}

// TestContestedRewardItemDoesNotDoubleDiscountAcrossStackablePromotions validates contested target handling.
// Two stackable promotions can be eligible, but once the shared reward line reaches effective price zero,
// a later promotion must not discount that same reward unit again.
func (s *BuyXGetYPromotionTestSuite) TestContestedRewardItemDoesNotDoubleDiscountAcrossStackablePromotions() {
	phoneRewardPromoID := s.createBuyXGetYPromotion(
		"P0 Buy Phone Get Headphone",
		model.BuyXGetYConfig{
			BuyQuantity:  1,
			GetQuantity:  1,
			IsSameReward: boolPtr(false),
			GetProductID: uintPtr(4),
		},
		promotionCreateOptions{
			appliesTo: "specific_products",
			canStack:  boolPtr(true),
			priority:  intPtr(0),
		},
	)
	s.linkPromotionProducts(phoneRewardPromoID, 1)

	tabletRewardPromoID := s.createBuyXGetYPromotion(
		"P1 Buy Tablet Get Headphone",
		model.BuyXGetYConfig{
			BuyQuantity:  1,
			GetQuantity:  1,
			IsSameReward: boolPtr(false),
			GetProductID: uintPtr(4),
		},
		promotionCreateOptions{
			appliesTo: "specific_products",
			canStack:  boolPtr(true),
			priority:  intPtr(1),
		},
	)
	s.linkPromotionProducts(tabletRewardPromoID, 2)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("phone-line", 1, 1, 99900, 1),
			cartItem("tablet-line", 2, 5, 79900, 1),
			cartItem("headphone-line", 4, 19, 39999, 1),
		),
	)

	assertPromotionSummary(s.T(), summary, 219799, 39999, 179800)
	assertSingleAppliedPromotionName(s.T(), summary, "P0 Buy Phone Get Headphone")
	headphones := findItemSummaryByID(s.T(), summary, "headphone-line")
	require.Equal(
		s.T(),
		int64(0),
		headphones.FinalPriceCents,
		"shared reward item should be fully consumed after first qualifying promotion",
	)
}

// TestBuyXGetYSameCategoryMultipleSetsPickCheapestEligibleUnits validates repeated-set behavior for grouped pools.
// For same-category pools with mixed prices, the implementation should consume expensive units as "buy" and
// discount the cheapest eligible units when multiple sets are formed in one pass.
func (s *BuyXGetYPromotionTestSuite) TestBuyXGetYSameCategoryMultipleSetsPickCheapestEligibleUnits() {
	promoID := s.createBuyXGetYPromotion(
		"Phones Buy 2 Get 1 Multi-Set",
		model.BuyXGetYConfig{
			BuyQuantity:  2,
			GetQuantity:  1,
			IsSameReward: boolPtr(true),
			ScopeType:    model.BuyXGetYScopeSameCategory,
		},
		promotionCreateOptions{
			appliesTo: "specific_categories",
		},
	)
	s.linkPromotionCategories(promoID, 4)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItemWithCategory("iphone-line", 1, 1, 4, 99900, 2),
			cartItemWithCategory("s24-256-line", 2, 6, 4, 89900, 2),
			cartItemWithCategory("s24-128-line", 2, 5, 4, 79900, 2),
		),
	)

	// 6 units total => 2 complete sets of (buy2+get1), so 2 units should be free.
	// Pricing rule: expensive units satisfy "buy", cheapest remaining units are free => 2x79900.
	assertPromotionSummary(s.T(), summary, 539400, 159800, 379600)
	cheapestLine := findItemSummaryByID(s.T(), summary, "s24-128-line")
	require.Equal(
		s.T(),
		int64(159800),
		cheapestLine.TotalDiscountCents,
		"cheapest eligible units should carry the free-unit discount",
	)
}

// TestCrossTenantBuyXGetYDoesNotLeak validates seller isolation during promotion application.
func (s *BuyXGetYPromotionTestSuite) TestCrossTenantBuyXGetYDoesNotLeak() {
	s.createBuyXGetYPromotion(
		"Seller 2 Buy 1 Get 1",
		model.BuyXGetYConfig{
			BuyQuantity: 1,
			GetQuantity: 1,
		},
		promotionCreateOptions{},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller4UserID,
			helpers.Customer3UserID,
			cartItem("sofa-line", 8, 16, 129900, 2),
		),
	)

	assertPromotionSummary(s.T(), summary, 259800, 0, 259800)
	require.Empty(s.T(), summary.AppliedPromotions, "promotion must not leak across sellers")
}

// TestUnauthorizedBuyXGetYCreation validates RBAC on the create endpoint.
func (s *BuyXGetYPromotionTestSuite) TestUnauthorizedBuyXGetYCreation() {
	res := s.customerClient.Post(
		s.T(),
		PromotionAPIEndpoint,
		buildBuyXGetYPayload(
			"Unauthorized Buy X Get Y",
			model.BuyXGetYConfig{
				BuyQuantity: 1,
				GetQuantity: 1,
			},
		),
	)
	s.Require().
		Equal(http.StatusForbidden, res.Code, "customer token must not be allowed to create promotions")
}

// TestCreateBuyXGetYValidationRejectsRawInvalidValues covers invalid JSON shapes that typed Go structs
// cannot represent. We intentionally use map payloads here to verify server-side validation against:
// - cross-reward mode with get_product_id=0
// - same-reward mode with unsupported scope_type value
func (s *BuyXGetYPromotionTestSuite) TestCreateBuyXGetYValidationRejectsRawInvalidValues() {
	invalidZeroGetProductIDPayload := buildBuyXGetYPayload(
		"Invalid Cross Reward Zero Product",
		map[string]interface{}{
			"buy_quantity":   1,
			"get_quantity":   1,
			"is_same_reward": false,
			"get_product_id": 0,
		},
	)
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, invalidZeroGetProductIDPayload)
	s.Require().Equal(
		http.StatusBadRequest,
		res.Code,
		"cross-reward payload must reject get_product_id=0",
	)

	invalidScopePayload := buildBuyXGetYPayload(
		"Invalid Same Reward Scope",
		map[string]interface{}{
			"buy_quantity":   1,
			"get_quantity":   1,
			"is_same_reward": true,
			"scope_type":     "same_collection",
		},
	)
	res = s.sellerClient.Post(s.T(), PromotionAPIEndpoint, invalidScopePayload)
	s.Require().Equal(
		http.StatusBadRequest,
		res.Code,
		"same-reward payload must reject unsupported scope_type",
	)
}

func (s *BuyXGetYPromotionTestSuite) createBuyXGetYPromotion(
	name string,
	discountConfig interface{},
	opts promotionCreateOptions,
	extraFields ...map[string]interface{},
) uint {
	payload := buildBuyXGetYPayload(name, discountConfig)
	payload["appliesTo"] = defaultString(opts.appliesTo, appliesAllProducts)
	payload["eligibleFor"] = defaultString(opts.eligibleFor, eligibleEveryone)
	if opts.canStack != nil {
		payload["canStackWithOtherPromotions"] = *opts.canStack
	}
	if opts.priority != nil {
		payload["priority"] = *opts.priority
	}
	if len(extraFields) > 0 {
		for key, value := range extraFields[0] {
			payload[key] = value
		}
	}
	return s.createPromotionFromPayload(payload)
}

func (s *BuyXGetYPromotionTestSuite) createPromotion(
	name string,
	promoType string,
	discountConfig interface{},
	opts promotionCreateOptions,
	extraFields ...map[string]interface{},
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
	if len(extraFields) > 0 {
		for key, value := range extraFields[0] {
			payload[key] = value
		}
	}
	return s.createPromotionFromPayload(payload)
}

func (s *BuyXGetYPromotionTestSuite) createPromotionFromPayload(
	payload map[string]interface{},
) uint {
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().
		Equal(http.StatusCreated, res.Code, "promotion creation should succeed for valid payloads")

	respData := helpers.ParseResponse(s.T(), res.Body)
	promo := respData["data"].(map[string]interface{})["promotion"].(map[string]interface{})
	return uint(promo["id"].(float64))
}

func (s *BuyXGetYPromotionTestSuite) applyPromotions(
	cart *model.CartValidationRequest,
) *model.AppliedPromotionSummary {
	promotionService := singleton.GetInstance().GetPromotionService()
	summary, err := promotionService.ApplyPromotionsToCart(context.Background(), cart)
	s.Require().NoError(err, "applying promotions should not return an unexpected error")
	return summary
}

func (s *BuyXGetYPromotionTestSuite) linkPromotionProducts(promotionID uint, productIDs ...uint) {
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

func (s *BuyXGetYPromotionTestSuite) linkPromotionCategories(
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

func (s *BuyXGetYPromotionTestSuite) cleanupPromotions() {
	sellerIDs := []uint{helpers.SellerUserID, helpers.Seller2UserID, helpers.Seller4UserID}

	var promoIDs []uint
	err := s.container.DB.
		Model(&promotionEntity.Promotion{}).
		Where("seller_id IN ?", sellerIDs).
		Pluck("id", &promoIDs).Error
	s.Require().NoError(err, "promotion cleanup should fetch candidate IDs")

	if len(promoIDs) == 0 {
		return
	}

	s.Require().NoError(
		s.container.DB.Where("promotion_id IN ?", promoIDs).
			Delete(&promotionEntity.PromotionUsage{}).
			Error,
		"promotion usages should be removable during cleanup",
	)
	s.Require().NoError(
		s.container.DB.Where("promotion_id IN ?", promoIDs).
			Delete(&promotionEntity.PromotionProductVariant{}).
			Error,
		"promotion variant links should be removable during cleanup",
	)
	s.Require().NoError(
		s.container.DB.Where("promotion_id IN ?", promoIDs).
			Delete(&promotionEntity.PromotionProduct{}).
			Error,
		"promotion product links should be removable during cleanup",
	)
	s.Require().NoError(
		s.container.DB.Where("promotion_id IN ?", promoIDs).
			Delete(&promotionEntity.PromotionCategory{}).
			Error,
		"promotion category links should be removable during cleanup",
	)
	s.Require().NoError(
		s.container.DB.Where("promotion_id IN ?", promoIDs).
			Delete(&promotionEntity.PromotionCollection{}).
			Error,
		"promotion collection links should be removable during cleanup",
	)
	s.Require().NoError(
		s.container.DB.Unscoped().
			Where("id IN ?", promoIDs).
			Delete(&promotionEntity.Promotion{}).
			Error,
		"promotions should be removable during cleanup",
	)
}

func buildBuyXGetYPayload(name string, discountConfig interface{}) map[string]interface{} {
	return map[string]interface{}{
		"name":           name,
		"promotionType":  promoTypeBuyXGetY,
		"discountConfig": discountConfig,
		"appliesTo":      appliesAllProducts,
		"eligibleFor":    eligibleEveryone,
		"startsAt":       "2023-01-01T00:00:00Z",
		"endsAt":         "2029-12-31T23:59:59Z",
		"status":         promoStatusActive,
	}
}

func cartItemWithCategory(
	itemID string,
	productID uint,
	variantID uint,
	categoryID uint,
	priceCents int64,
	quantity int,
) model.CartItem {
	item := cartItem(itemID, productID, variantID, priceCents, quantity)
	item.CategoryID = categoryID
	return item
}

func assertSingleAppliedPromotionName(
	t *testing.T,
	summary *model.AppliedPromotionSummary,
	name string,
) {
	t.Helper()
	require.Len(t, summary.AppliedPromotions, 1, "exactly one promotion should apply")
	require.NotNil(
		t,
		summary.AppliedPromotions[0].Promotion,
		"applied promotion should include promotion details",
	)
	require.Equal(
		t,
		name,
		summary.AppliedPromotions[0].Promotion.Name,
		"unexpected promotion won the conflict",
	)
}

func findItemSummaryByID(
	t *testing.T,
	summary *model.AppliedPromotionSummary,
	itemID string,
) model.CartItemSummary {
	t.Helper()
	for _, item := range summary.Items {
		if item.ItemID == itemID {
			return item
		}
	}
	t.Fatalf("cart item summary %q not found", itemID)
	return model.CartItemSummary{}
}

func uintPtr(v uint) *uint {
	return &v
}
