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

const promoTypeTiered = "tiered"

type tieredCreateOptions struct {
	promotionCreateOptions
}

type TieredPromotionTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler

	sellerClient   *helpers.APIClient
	customerClient *helpers.APIClient
}

func (s *TieredPromotionTestSuite) SetupSuite() {
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

func (s *TieredPromotionTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func (s *TieredPromotionTestSuite) SetupTest() {
	s.cleanupPromotions()
}

func TestTieredPromotionStrategy(t *testing.T) {
	suite.Run(t, new(TieredPromotionTestSuite))
}

// ---------------------------------------------------------------------------
// Core Discount Logic — Quantity Tiers
// ---------------------------------------------------------------------------

func (s *TieredPromotionTestSuite) TestTieredQuantityBelowAllTiers() {
	s.createTieredPromotion("Qty Tiers", model.TierTypeQuantity, []model.TierConfig{
		{Min: 2, Max: intPtr(4), DiscountType: model.DiscountTypePercentage, DiscountValue: 5},
		{Min: 5, Max: intPtr(9), DiscountType: model.DiscountTypePercentage, DiscountValue: 10},
		{Min: 10, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 15},
	}, tieredCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1), // qty 1, below all tiers
		),
	)
	assertPromotionSummary(s.T(), summary, 100000, 0, 100000)
	s.Require().Empty(summary.AppliedPromotions)
}

func (s *TieredPromotionTestSuite) TestTieredQuantityFirstTier() {
	s.createTieredPromotion("Qty Tiers", model.TierTypeQuantity, []model.TierConfig{
		{Min: 2, Max: intPtr(4), DiscountType: model.DiscountTypePercentage, DiscountValue: 5},
		{Min: 5, Max: intPtr(9), DiscountType: model.DiscountTypePercentage, DiscountValue: 10},
		{Min: 10, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 15},
	}, tieredCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 3), // qty 3, matches [2-4]: 5%
		),
	)
	// 5% of 300000 = 15000
	assertPromotionSummary(s.T(), summary, 300000, 15000, 285000)
}

func (s *TieredPromotionTestSuite) TestTieredQuantityMiddleTier() {
	s.createTieredPromotion("Qty Tiers", model.TierTypeQuantity, []model.TierConfig{
		{Min: 2, Max: intPtr(4), DiscountType: model.DiscountTypePercentage, DiscountValue: 5},
		{Min: 5, Max: intPtr(9), DiscountType: model.DiscountTypePercentage, DiscountValue: 10},
		{Min: 10, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 15},
	}, tieredCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 6), // qty 6, matches [5-9]: 10%
		),
	)
	// 10% of 600000 = 60000
	assertPromotionSummary(s.T(), summary, 600000, 60000, 540000)
}

func (s *TieredPromotionTestSuite) TestTieredQuantityTopTier() {
	s.createTieredPromotion("Qty Tiers", model.TierTypeQuantity, []model.TierConfig{
		{Min: 2, Max: intPtr(4), DiscountType: model.DiscountTypePercentage, DiscountValue: 5},
		{Min: 5, Max: intPtr(9), DiscountType: model.DiscountTypePercentage, DiscountValue: 10},
		{Min: 10, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 15},
	}, tieredCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 12), // qty 12, matches [10+]: 15%
		),
	)
	// 15% of 1200000 = 180000
	assertPromotionSummary(s.T(), summary, 1200000, 180000, 1020000)
}

func (s *TieredPromotionTestSuite) TestTieredQuantityExactBoundary() {
	s.createTieredPromotion("Qty Tiers", model.TierTypeQuantity, []model.TierConfig{
		{Min: 2, Max: intPtr(4), DiscountType: model.DiscountTypePercentage, DiscountValue: 5},
		{Min: 5, Max: intPtr(9), DiscountType: model.DiscountTypePercentage, DiscountValue: 10},
		{Min: 10, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 15},
	}, tieredCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 5), // qty 5, exact boundary of [5-9]
		),
	)
	// 10% of 500000 = 50000
	assertPromotionSummary(s.T(), summary, 500000, 50000, 450000)
}

func (s *TieredPromotionTestSuite) TestTieredQuantityFixedAmountTier() {
	s.createTieredPromotion("Qty Fixed", model.TierTypeQuantity, []model.TierConfig{
		{
			Min:           3,
			Max:           intPtr(5),
			DiscountType:  model.DiscountTypeFixedAmount,
			DiscountValue: 25000,
		}, // 250 off
	}, tieredCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 4), // qty 4, matches tier
		),
	)
	// Fixed 25000 off 400000
	assertPromotionSummary(s.T(), summary, 400000, 25000, 375000)
}

// ---------------------------------------------------------------------------
// Core Discount Logic — Spend Tiers
// ---------------------------------------------------------------------------

func (s *TieredPromotionTestSuite) TestTieredSpendBelowThreshold() {
	s.createTieredPromotion("Spend Tiers", model.TierTypeSpend, []model.TierConfig{
		{Min: 200000, Max: nil, DiscountType: model.DiscountTypeFixedAmount, DiscountValue: 20000},
	}, tieredCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 150000, 1), // spend 150000 < 200000
		),
	)
	assertPromotionSummary(s.T(), summary, 150000, 0, 150000)
	s.Require().Empty(summary.AppliedPromotions)
}

func (s *TieredPromotionTestSuite) TestTieredSpendFirstTier() {
	s.createTieredPromotion("Spend Tiers", model.TierTypeSpend, []model.TierConfig{
		{Min: 200000, Max: nil, DiscountType: model.DiscountTypeFixedAmount, DiscountValue: 20000},
	}, tieredCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 250000, 1), // spend 250000 >= 200000
		),
	)
	assertPromotionSummary(s.T(), summary, 250000, 20000, 230000)
}

func (s *TieredPromotionTestSuite) TestTieredSpendSecondTier() {
	// Higher tier first so 550000 matches [500000+] not [200000+]
	s.createTieredPromotion("Spend Tiers", model.TierTypeSpend, []model.TierConfig{
		{Min: 500000, Max: nil, DiscountType: model.DiscountTypeFixedAmount, DiscountValue: 70000},
		{Min: 200000, Max: nil, DiscountType: model.DiscountTypeFixedAmount, DiscountValue: 20000},
	}, tieredCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 550000, 1), // spend 550000
		),
	)
	assertPromotionSummary(s.T(), summary, 550000, 70000, 480000)
}

func (s *TieredPromotionTestSuite) TestTieredSpendMultiItemCrossesTier() {
	s.createTieredPromotion("Spend Tiers", model.TierTypeSpend, []model.TierConfig{
		{Min: 500000, Max: nil, DiscountType: model.DiscountTypeFixedAmount, DiscountValue: 70000},
	}, tieredCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 300000, 1),
			cartItem("2", 2, 5, 300000, 1), // total 600000
		),
	)
	assertPromotionSummary(s.T(), summary, 600000, 70000, 530000)
}

func (s *TieredPromotionTestSuite) TestTieredSpendPercentageTier() {
	s.createTieredPromotion("Spend %", model.TierTypeSpend, []model.TierConfig{
		{Min: 200000, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 15},
	}, tieredCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 300000, 1), // spend 300000
		),
	)
	// 15% of 300000 = 45000
	assertPromotionSummary(s.T(), summary, 300000, 45000, 255000)
}

// ---------------------------------------------------------------------------
// Scope
// ---------------------------------------------------------------------------

func (s *TieredPromotionTestSuite) TestTieredScopedToSpecificProducts() {
	promoID := s.createTieredPromotion(
		"Qty Scoped",
		model.TierTypeQuantity,
		[]model.TierConfig{
			{Min: 2, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 10},
		},
		tieredCreateOptions{
			promotionCreateOptions: promotionCreateOptions{appliesTo: appliesSpecificProducts},
		},
	)
	s.linkPromotionProducts(promoID, 1)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 2), // eligible, qty 2
			cartItem("2", 2, 5, 80000, 2),  // not eligible
		),
	)
	// Only product 1: qty 2, 10% of 200000 = 20000
	assertPromotionSummary(s.T(), summary, 360000, 20000, 340000)
	item1 := findItemSummaryByID(s.T(), summary, "1")
	item2 := findItemSummaryByID(s.T(), summary, "2")
	s.Require().Equal(int64(20000), item1.TotalDiscountCents)
	s.Require().Equal(int64(0), item2.TotalDiscountCents)
}

func (s *TieredPromotionTestSuite) TestTieredScopedToCategory() {
	promoID := s.createTieredPromotion(
		"Spend Cat",
		model.TierTypeSpend,
		[]model.TierConfig{
			{
				Min:           150000,
				Max:           nil,
				DiscountType:  model.DiscountTypeFixedAmount,
				DiscountValue: 20000,
			},
		},
		tieredCreateOptions{
			promotionCreateOptions: promotionCreateOptions{appliesTo: appliesSpecificCategories},
		},
	)
	s.linkPromotionCategories(promoID, 4)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItemWithCategory("1", 1, 1, 4, 100000, 2), // cat 4, 200000 eligible
			cartItemWithCategory("2", 2, 5, 99, 80000, 1), // cat 99 excluded
		),
	)
	// Eligible spend = 200000 >= 150000 => 20000 off
	assertPromotionSummary(s.T(), summary, 280000, 20000, 260000)
	item1 := findItemSummaryByID(s.T(), summary, "1")
	item2 := findItemSummaryByID(s.T(), summary, "2")
	s.Require().Equal(int64(20000), item1.TotalDiscountCents)
	s.Require().Equal(int64(0), item2.TotalDiscountCents)
}

func (s *TieredPromotionTestSuite) TestTieredScopeEligibleSpendBelowThreshold() {
	promoID := s.createTieredPromotion(
		"Spend Cat",
		model.TierTypeSpend,
		[]model.TierConfig{
			{
				Min:           200000,
				Max:           nil,
				DiscountType:  model.DiscountTypeFixedAmount,
				DiscountValue: 20000,
			},
		},
		tieredCreateOptions{
			promotionCreateOptions: promotionCreateOptions{appliesTo: appliesSpecificCategories},
		},
	)
	s.linkPromotionCategories(promoID, 4)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItemWithCategory("1", 1, 1, 4, 150000, 1),  // cat 4: 150000
			cartItemWithCategory("2", 2, 5, 99, 100000, 1), // cat 99 excluded
		),
	)
	// Eligible spend = 150000 < 200000 => no discount
	assertPromotionSummary(s.T(), summary, 250000, 0, 250000)
	s.Require().Empty(summary.AppliedPromotions)
}

// ---------------------------------------------------------------------------
// Eligibility
// ---------------------------------------------------------------------------

func (s *TieredPromotionTestSuite) TestTieredNewCustomerOnly() {
	s.createTieredPromotion(
		"Qty New",
		model.TierTypeQuantity,
		[]model.TierConfig{
			{Min: 5, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 10},
		},
		tieredCreateOptions{
			promotionCreateOptions: promotionCreateOptions{eligibleFor: eligibleNewCustomers},
		},
	)

	newCustSummary := s.applyPromotions(
		buildCartRequestWithFirstOrder(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			true,
			cartItem("1", 1, 1, 100000, 6), // qty 6, new customer
		),
	)
	// 10% of 600000 = 60000
	assertPromotionSummary(s.T(), newCustSummary, 600000, 60000, 540000)

	returningSummary := s.applyPromotions(
		buildCartRequestWithFirstOrder(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			false,
			cartItem("1", 1, 1, 100000, 6), // returning
		),
	)
	assertPromotionSummary(s.T(), returningSummary, 600000, 0, 600000)
}

// ---------------------------------------------------------------------------
// Stacking and Priority
// ---------------------------------------------------------------------------

func (s *TieredPromotionTestSuite) TestStackableTieredWithFixedAmount() {
	s.createTieredPromotion(
		"Qty Tier Stackable",
		model.TierTypeQuantity,
		[]model.TierConfig{
			{Min: 2, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 10},
		},
		tieredCreateOptions{
			promotionCreateOptions: promotionCreateOptions{canStack: boolPtr(true)},
		},
	)
	s.createGenericPromotion(
		"Fixed Stackable",
		promoTypeFixedAmount,
		model.FixedAmountConfig{AmountCents: 10000},
		promotionCreateOptions{canStack: boolPtr(true)},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 2), // qty 2
		),
	)
	// Tiered: 10% of 200000 = 20000; Fixed: 10000; total 30000
	assertPromotionSummary(s.T(), summary, 200000, 30000, 170000)
	s.Require().Len(summary.AppliedPromotions, 2)
}

func (s *TieredPromotionTestSuite) TestStackableTieredUsesReducedPrices() {
	s.createGenericPromotion(
		"Fixed First",
		promoTypeFixedAmount,
		model.FixedAmountConfig{AmountCents: 50000},
		promotionCreateOptions{canStack: boolPtr(true)},
	)
	s.createTieredPromotion(
		"Spend Tier Second",
		model.TierTypeSpend,
		[]model.TierConfig{
			{Min: 50000, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 10},
		},
		tieredCreateOptions{
			promotionCreateOptions: promotionCreateOptions{canStack: boolPtr(true)},
		},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1), // 100000 - 50000 = 50000 effective
		),
	)
	// Fixed 50000 first => effective 50000. Tiered: 10% of 50000 = 5000. Total 55000
	assertPromotionSummary(s.T(), summary, 100000, 55000, 45000)
}

func (s *TieredPromotionTestSuite) TestNonStackableTieredVsPercentageBestWins() {
	s.createTieredPromotion(
		"Qty Tier",
		model.TierTypeQuantity,
		[]model.TierConfig{
			{Min: 2, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 5},
		},
		tieredCreateOptions{},
	)
	s.createGenericPromotion(
		"20% Off",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 20},
		promotionCreateOptions{},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 3), // tiered 5% = 15000, percentage 20% = 60000
		),
	)
	// 20% wins (60000 > 15000)
	assertPromotionSummary(s.T(), summary, 300000, 60000, 240000)
	s.Require().Len(summary.AppliedPromotions, 1)
}

func (s *TieredPromotionTestSuite) TestHigherPriorityTieredBlocksLower() {
	s.createTieredPromotion(
		"Qty High",
		model.TierTypeQuantity,
		[]model.TierConfig{
			{Min: 1, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 15},
		},
		tieredCreateOptions{promotionCreateOptions: promotionCreateOptions{priority: intPtr(10)}},
	)
	s.createGenericPromotion(
		"50% Low",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 50},
		promotionCreateOptions{canStack: boolPtr(true), priority: intPtr(100)},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	assertPromotionSummary(s.T(), summary, 100000, 15000, 85000)
	s.Require().Len(summary.AppliedPromotions, 1)
}

func (s *TieredPromotionTestSuite) TestSamePriorityNonStackableTieredLoserSkipped() {
	s.createTieredPromotion(
		"Qty 5%",
		model.TierTypeQuantity,
		[]model.TierConfig{
			{Min: 2, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 5},
		},
		tieredCreateOptions{},
	)
	s.createTieredPromotion(
		"Qty 15%",
		model.TierTypeQuantity,
		[]model.TierConfig{
			{Min: 2, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 15},
		},
		tieredCreateOptions{},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 3), // 5% = 15000, 15% = 45000
		),
	)
	assertPromotionSummary(s.T(), summary, 300000, 45000, 255000)
	s.Require().Len(summary.AppliedPromotions, 1)
	s.Require().NotEmpty(summary.SkippedPromotions)
}

// ---------------------------------------------------------------------------
// Negative Paths
// ---------------------------------------------------------------------------

func (s *TieredPromotionTestSuite) TestTieredNoEligibleItems() {
	promoID := s.createTieredPromotion(
		"Qty Scoped",
		model.TierTypeQuantity,
		[]model.TierConfig{
			{Min: 2, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 10},
		},
		tieredCreateOptions{
			promotionCreateOptions: promotionCreateOptions{appliesTo: appliesSpecificProducts},
		},
	)
	s.linkPromotionProducts(promoID, 1)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 2, 5, 80000, 2), // product 2, not linked
		),
	)
	assertPromotionSummary(s.T(), summary, 160000, 0, 160000)
	s.Require().Empty(summary.AppliedPromotions)
}

func (s *TieredPromotionTestSuite) TestTieredDoesNotMeetMinimumTier() {
	s.createTieredPromotion("Qty Min", model.TierTypeQuantity, []model.TierConfig{
		{Min: 2, Max: intPtr(5), DiscountType: model.DiscountTypePercentage, DiscountValue: 10},
	}, tieredCreateOptions{})

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1), // qty 1 < min 2
		),
	)
	assertPromotionSummary(s.T(), summary, 100000, 0, 100000)
	s.Require().Empty(summary.AppliedPromotions)
}

func (s *TieredPromotionTestSuite) TestTieredInvalidConfigRejected() {
	payload := buildTieredPayload(model.TierTypeQuantity, []model.TierConfig{
		{
			Min:           2,
			Max:           intPtr(5),
			DiscountType:  model.DiscountTypePercentage,
			DiscountValue: 150,
		}, // > 100
	})
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

// ---------------------------------------------------------------------------
// Validation and Security
// ---------------------------------------------------------------------------

func (s *TieredPromotionTestSuite) TestCreateValidTieredPromotion() {
	payload := buildTieredPayload(model.TierTypeQuantity, []model.TierConfig{
		{Min: 2, Max: intPtr(4), DiscountType: model.DiscountTypePercentage, DiscountValue: 5},
		{Min: 5, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 10},
	})
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code)
}

func (s *TieredPromotionTestSuite) TestCreateValidTieredLastTierNoMax() {
	payload := buildTieredPayload(model.TierTypeQuantity, []model.TierConfig{
		{Min: 2, Max: intPtr(4), DiscountType: model.DiscountTypePercentage, DiscountValue: 5},
		{Min: 5, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 15},
	})
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code)
}

func (s *TieredPromotionTestSuite) TestCrossTenantTieredDoesNotLeak() {
	s.createTieredPromotion(
		"Seller2 Spend",
		model.TierTypeSpend,
		[]model.TierConfig{
			{Min: 100000, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 10},
		},
		tieredCreateOptions{},
	)

	summary := s.applyPromotions(
		buildCartRequest(
			helpers.Seller4UserID,
			helpers.Customer3UserID,
			cartItem("1", 8, 16, 100000, 1),
		),
	)
	assertPromotionSummary(s.T(), summary, 100000, 0, 100000)
}

func (s *TieredPromotionTestSuite) TestUnauthorizedTieredCreation() {
	payload := buildTieredPayload(model.TierTypeQuantity, []model.TierConfig{
		{Min: 2, Max: nil, DiscountType: model.DiscountTypePercentage, DiscountValue: 10},
	})
	res := s.customerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusForbidden, res.Code)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (s *TieredPromotionTestSuite) createTieredPromotion(
	name string,
	tierType model.TierType,
	tiers []model.TierConfig,
	opts tieredCreateOptions,
) uint {
	config := model.TieredConfig{
		TierType: tierType,
		Tiers:    tiers,
	}
	return s.createGenericPromotion(name, promoTypeTiered, config, opts.promotionCreateOptions)
}

func (s *TieredPromotionTestSuite) createGenericPromotion(
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

func (s *TieredPromotionTestSuite) createPromotionFromPayload(payload map[string]interface{}) uint {
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code, "promotion creation should succeed")

	respData := helpers.ParseResponse(s.T(), res.Body)
	promo := respData["data"].(map[string]interface{})["promotion"].(map[string]interface{})
	return uint(promo["id"].(float64))
}

func (s *TieredPromotionTestSuite) applyPromotions(
	cart *model.CartValidationRequest,
) *model.AppliedPromotionSummary {
	promotionService := singleton.GetInstance().GetPromotionService()
	summary, err := promotionService.ApplyPromotionsToCart(context.Background(), cart)
	s.Require().NoError(err)
	return summary
}

func (s *TieredPromotionTestSuite) linkPromotionProducts(promotionID uint, productIDs ...uint) {
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

func (s *TieredPromotionTestSuite) linkPromotionCategories(promotionID uint, categoryIDs ...uint) {
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

func buildTieredPayload(tierType model.TierType, tiers []model.TierConfig) map[string]interface{} {
	config := model.TieredConfig{
		TierType: tierType,
		Tiers:    tiers,
	}
	return map[string]interface{}{
		"name":           "Tiered Test",
		"promotionType":  promoTypeTiered,
		"discountConfig": config,
		"appliesTo":      appliesAllProducts,
		"eligibleFor":    eligibleEveryone,
		"startsAt":       "2023-01-01T00:00:00Z",
		"endsAt":         "2029-12-31T23:59:59Z",
		"status":         promoStatusActive,
	}
}

func (s *TieredPromotionTestSuite) cleanupPromotions() {
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
