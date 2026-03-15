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

const promoTypeFreeShipping = "free_shipping"

type freeShippingOpts struct {
	promotionCreateOptions
	minOrderCents            *int64
	maxShippingDiscountCents *int64
}

type FreeShippingPromotionTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler

	sellerClient   *helpers.APIClient
	customerClient *helpers.APIClient
}

func (s *FreeShippingPromotionTestSuite) SetupSuite() {
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

func (s *FreeShippingPromotionTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func (s *FreeShippingPromotionTestSuite) SetupTest() {
	s.cleanupPromotions()
}

func TestFreeShippingPromotionStrategy(t *testing.T) {
	suite.Run(t, new(FreeShippingPromotionTestSuite))
}

// ---------------------------------------------------------------------------
// Core Discount Logic
// ---------------------------------------------------------------------------

// TestFreeShippingUnconditional validates that a free shipping promotion with no
// config waives the full shipping charge and does not touch item prices.
func (s *FreeShippingPromotionTestSuite) TestFreeShippingUnconditional() {
	s.createFreeShippingPromotion("Free Shipping Always", freeShippingOpts{})

	summary := s.applyPromotions(
		buildCartRequestWithShipping(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			15000, // shipping = $150
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// item subtotal unchanged, full shipping waived
	assertPromotionSummary(s.T(), summary, 100000, 0, 100000)
	s.Require().Equal(int64(15000), summary.ShippingDiscount)
	s.Require().Len(summary.AppliedPromotions, 1)
}

// TestFreeShippingWithMaxDiscountCap validates that when max_shipping_discount_cents
// is less than the actual shipping, only the cap is discounted.
func (s *FreeShippingPromotionTestSuite) TestFreeShippingWithMaxDiscountCap() {
	s.createFreeShippingPromotion("Capped Free Shipping", freeShippingOpts{
		maxShippingDiscountCents: helpers.Int64Ptr(8000),
	})

	summary := s.applyPromotions(
		buildCartRequestWithShipping(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			15000, // shipping = $150, cap = $80
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// shipping discount capped at 8000, not full 15000
	s.Require().Equal(int64(8000), summary.ShippingDiscount)
	s.Require().Equal(int64(0), summary.TotalDiscountCents)
}

// TestFreeShippingCapExceedsActualShipping validates that when the cap is larger
// than the actual shipping, the full shipping is waived (not the cap amount).
func (s *FreeShippingPromotionTestSuite) TestFreeShippingCapExceedsActualShipping() {
	s.createFreeShippingPromotion("Big Cap Shipping", freeShippingOpts{
		maxShippingDiscountCents: helpers.Int64Ptr(50000),
	})

	summary := s.applyPromotions(
		buildCartRequestWithShipping(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			8000, // shipping = $80, cap = $500 (cap is larger)
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// only actual shipping waived, not the cap
	s.Require().Equal(int64(8000), summary.ShippingDiscount)
}

// TestFreeShippingMinOrderMet validates that shipping is waived when the cart
// subtotal meets or exceeds the min_order_cents threshold.
func (s *FreeShippingPromotionTestSuite) TestFreeShippingMinOrderMet() {
	s.createFreeShippingPromotion("Free Shipping Min 500", freeShippingOpts{
		minOrderCents: helpers.Int64Ptr(50000),
	})

	summary := s.applyPromotions(
		buildCartRequestWithShipping(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			10000, // shipping = $100
			cartItem("1", 1, 1, 60000, 1), // subtotal = 60000 >= 50000
		),
	)
	s.Require().Equal(int64(10000), summary.ShippingDiscount)
}

// TestFreeShippingMinOrderNotMet validates that shipping is NOT waived when the
// cart subtotal is below the min_order_cents threshold.
func (s *FreeShippingPromotionTestSuite) TestFreeShippingMinOrderNotMet() {
	s.createFreeShippingPromotion("Free Shipping Min 500", freeShippingOpts{
		minOrderCents: helpers.Int64Ptr(50000),
	})

	summary := s.applyPromotions(
		buildCartRequestWithShipping(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			10000,
			cartItem("1", 1, 1, 30000, 1), // subtotal = 30000 < 50000
		),
	)
	s.Require().Equal(int64(0), summary.ShippingDiscount)
	s.Require().Empty(summary.AppliedPromotions, "threshold not met, should not apply")
}

// TestFreeShippingMinOrderExactBoundary validates the boundary condition where the
// cart subtotal exactly equals min_order_cents — it should qualify.
func (s *FreeShippingPromotionTestSuite) TestFreeShippingMinOrderExactBoundary() {
	s.createFreeShippingPromotion("Free Shipping Min 500", freeShippingOpts{
		minOrderCents: helpers.Int64Ptr(50000),
	})

	summary := s.applyPromotions(
		buildCartRequestWithShipping(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			10000,
			cartItem("1", 1, 1, 50000, 1), // subtotal exactly = threshold
		),
	)
	s.Require().Equal(int64(10000), summary.ShippingDiscount)
}

// TestFreeShippingZeroShipping validates that when the cart has no shipping cost,
// the promotion is skipped (nothing to waive).
func (s *FreeShippingPromotionTestSuite) TestFreeShippingZeroShipping() {
	s.createFreeShippingPromotion("Free Shipping Always", freeShippingOpts{})

	summary := s.applyPromotions(
		buildCartRequest( // ShippingCents = 0 by default
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	s.Require().Equal(int64(0), summary.ShippingDiscount)
	s.Require().Empty(summary.AppliedPromotions, "no shipping to waive, should not apply")
}

// ---------------------------------------------------------------------------
// Scope Behavior
// ---------------------------------------------------------------------------

// TestFreeShippingScopeIsIgnored validates the key invariant: free shipping is
// always cart-wide. Even when appliesTo=specific_products with a linked product,
// the strategy ignores eligibleItems and applies to the entire cart shipping cost.
func (s *FreeShippingPromotionTestSuite) TestFreeShippingScopeIsIgnored() {
	promoID := s.createFreeShippingPromotion("Scoped Free Shipping", freeShippingOpts{
		promotionCreateOptions: promotionCreateOptions{appliesTo: appliesSpecificProducts},
	})
	// Link only product 1 to the scope
	s.linkFreeShippingProducts(promoID, 1)

	summary := s.applyPromotions(
		buildCartRequestWithShipping(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			12000,
			cartItem("1", 1, 1, 50000, 1), // product 1 (in scope)
			cartItem("2", 2, 5, 50000, 1), // product 2 (out of scope)
		),
	)
	// Shipping fully waived regardless — scope is not applied for free_shipping
	s.Require().Equal(int64(12000), summary.ShippingDiscount)
	s.Require().Len(summary.AppliedPromotions, 1)
}

// ---------------------------------------------------------------------------
// Customer Eligibility
// ---------------------------------------------------------------------------

// TestFreeShippingNewCustomerOnly validates that when eligible_for=new_customers,
// only first-time buyers receive free shipping; returning customers pay full shipping.
func (s *FreeShippingPromotionTestSuite) TestFreeShippingNewCustomerOnly() {
	s.createFreeShippingPromotion("New Customer Free Shipping", freeShippingOpts{
		promotionCreateOptions: promotionCreateOptions{eligibleFor: eligibleNewCustomers},
	})

	newCustomerSummary := s.applyPromotions(
		buildCartRequestFirstOrderWithShipping(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			10000, // shipping = $100
			true,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	s.Require().Equal(int64(10000), newCustomerSummary.ShippingDiscount)

	returningCustomerSummary := s.applyPromotions(
		buildCartRequestFirstOrderWithShipping(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			10000,
			false,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	s.Require().Equal(int64(0), returningCustomerSummary.ShippingDiscount)
}

// ---------------------------------------------------------------------------
// Stacking and Priority
// ---------------------------------------------------------------------------

// TestStackableFreeShippingWithItemDiscount validates that a stackable item
// discount (fixed_amount) and a stackable free shipping promotion both apply
// independently — item prices reduce AND shipping is waived.
func (s *FreeShippingPromotionTestSuite) TestStackableFreeShippingWithItemDiscount() {
	s.createGenericFreeShippingPromotion(
		"Fixed 200 Stackable",
		promoTypeFixedAmount,
		model.FixedAmountConfig{AmountCents: 20000},
		promotionCreateOptions{canStack: boolPtr(true)},
	)

	s.createFreeShippingPromotion("Free Shipping Stackable", freeShippingOpts{
		promotionCreateOptions: promotionCreateOptions{canStack: boolPtr(true)},
	})

	summary := s.applyPromotions(
		buildCartRequestWithShipping(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			10000,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// item discount: 20000, shipping discount: 10000
	assertPromotionSummary(s.T(), summary, 100000, 20000, 80000)
	s.Require().Equal(int64(10000), summary.ShippingDiscount)
	s.Require().Len(summary.AppliedPromotions, 2, "both promotions should apply")
}

// TestFreeShippingMinOrderUsesReducedSubtotal validates that when a preceding
// stackable item promotion reduces FinalSubtotal below the free shipping
// min_order_cents threshold, the free shipping is skipped.
//
// This exercises the key behavior: free shipping uses FinalSubtotal (post-discount)
// not OriginalSubtotal for the threshold check.
func (s *FreeShippingPromotionTestSuite) TestFreeShippingMinOrderUsesReducedSubtotal() {
	// Item discount at higher priority: reduces subtotal from 80000 to 20000
	s.createGenericFreeShippingPromotion(
		"Fixed 600 Off",
		promoTypeFixedAmount,
		model.FixedAmountConfig{AmountCents: 60000},
		promotionCreateOptions{
			canStack: boolPtr(true),
			priority: intPtr(10),
		},
	)

	// Free shipping at lower priority requires min_order = 50000
	s.createFreeShippingPromotion("Free Shipping Min 500", freeShippingOpts{
		promotionCreateOptions: promotionCreateOptions{
			canStack: boolPtr(true),
			priority: intPtr(100),
		},
		minOrderCents: helpers.Int64Ptr(50000),
	})

	summary := s.applyPromotions(
		buildCartRequestWithShipping(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			10000,
			cartItem("1", 1, 1, 80000, 1), // original subtotal = 80000
		),
	)
	// Item discount applied first: 80000 - 60000 = 20000 (FinalSubtotal)
	// Free shipping min_order = 50000, FinalSubtotal = 20000 < 50000 => skipped
	assertPromotionSummary(s.T(), summary, 80000, 60000, 20000)
	s.Require().Equal(int64(0), summary.ShippingDiscount, "free shipping should be skipped")
	s.Require().Len(summary.AppliedPromotions, 1, "only item discount should apply")
}

// TestNonStackableFreeShippingLosesToLargerItemDiscount validates that when a
// non-stackable free shipping promotion competes against a non-stackable item
// discount at the same priority, the one with the higher total benefit wins.
// The ranking sums TotalDiscountCents + ShippingDiscount.
func (s *FreeShippingPromotionTestSuite) TestNonStackableFreeShippingLosesToLargerItemDiscount() {
	// Item discount: 50000 off items (ranked as 50000)
	s.createGenericFreeShippingPromotion(
		"Fixed 500 Off",
		promoTypeFixedAmount,
		model.FixedAmountConfig{AmountCents: 50000},
		promotionCreateOptions{},
	)

	// Free shipping: 5000 shipping discount (ranked as 5000)
	s.createFreeShippingPromotion("Free Shipping", freeShippingOpts{})

	summary := s.applyPromotions(
		buildCartRequestWithShipping(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			5000,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// Item discount 50000 > shipping 5000 => item discount wins
	assertPromotionSummary(s.T(), summary, 100000, 50000, 50000)
	s.Require().Equal(int64(0), summary.ShippingDiscount, "free shipping should lose")
	s.Require().Len(summary.AppliedPromotions, 1)
	s.Require().NotEmpty(summary.SkippedPromotions, "free shipping should be in skipped list")
}

// TestNonStackableFreeShippingWinsOverSmallerItemDiscount validates that free
// shipping wins when its total shipping benefit exceeds the competing item discount.
func (s *FreeShippingPromotionTestSuite) TestNonStackableFreeShippingWinsOverSmallerItemDiscount() {
	// Item discount: only 2000 off (small discount)
	s.createGenericFreeShippingPromotion(
		"Fixed 20 Off",
		promoTypeFixedAmount,
		model.FixedAmountConfig{AmountCents: 2000},
		promotionCreateOptions{},
	)

	// Free shipping: 15000 shipping discount (larger benefit)
	s.createFreeShippingPromotion("Big Free Shipping", freeShippingOpts{})

	summary := s.applyPromotions(
		buildCartRequestWithShipping(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			15000, // shipping = 15000 > item discount 2000
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// Free shipping 15000 > item discount 2000 => free shipping wins
	assertPromotionSummary(s.T(), summary, 100000, 0, 100000)
	s.Require().Equal(int64(15000), summary.ShippingDiscount, "free shipping should win")
	s.Require().Len(summary.AppliedPromotions, 1)
	s.Require().NotEmpty(summary.SkippedPromotions, "item discount should be in skipped list")
}

// TestHigherPriorityFreeShippingBlocksLower validates that a non-stackable free
// shipping promotion at a higher priority (lower number) blocks lower-priority
// promotions from being evaluated.
func (s *FreeShippingPromotionTestSuite) TestHigherPriorityFreeShippingBlocksLower() {
	s.createFreeShippingPromotion("High Priority Free Shipping", freeShippingOpts{
		promotionCreateOptions: promotionCreateOptions{priority: intPtr(10)},
	})

	s.createGenericFreeShippingPromotion(
		"Low Priority 50% Off",
		promoTypePercentage,
		model.PercentageDiscountConfig{Percentage: 50},
		promotionCreateOptions{
			canStack: boolPtr(true),
			priority: intPtr(100),
		},
	)

	summary := s.applyPromotions(
		buildCartRequestWithShipping(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			10000,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	// non-stackable free shipping at priority 10 blocks lower priority percentage promo
	assertPromotionSummary(s.T(), summary, 100000, 0, 100000)
	s.Require().Equal(int64(10000), summary.ShippingDiscount)
	s.Require().Len(summary.AppliedPromotions, 1, "only free shipping should apply")
}

// TestSamePriorityNonStackableFreeShippingLoserSkipped validates that when two
// non-stackable promotions compete at the same priority, the losing one appears
// explicitly in SkippedPromotions.
func (s *FreeShippingPromotionTestSuite) TestSamePriorityNonStackableFreeShippingLoserSkipped() {
	// Item discount: 50000 (winner)
	s.createGenericFreeShippingPromotion(
		"Fixed 500 Off",
		promoTypeFixedAmount,
		model.FixedAmountConfig{AmountCents: 50000},
		promotionCreateOptions{},
	)
	// Free shipping: 5000 (loser)
	s.createFreeShippingPromotion("Free Shipping", freeShippingOpts{})

	summary := s.applyPromotions(
		buildCartRequestWithShipping(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			5000,
			cartItem("1", 1, 1, 100000, 1),
		),
	)
	assertPromotionSummary(s.T(), summary, 100000, 50000, 50000)
	s.Require().Len(summary.AppliedPromotions, 1)
	s.Require().NotEmpty(summary.SkippedPromotions, "losing promo must appear in skipped list")
}

// ---------------------------------------------------------------------------
// Negative Paths
// ---------------------------------------------------------------------------

// TestFreeShippingBothPromosFail validates that when both a free shipping and
// an item discount promo fail their respective conditions, nothing applies.
func (s *FreeShippingPromotionTestSuite) TestFreeShippingBothPromosFail() {
	// Free shipping requires min 500, cart is only 300
	s.createFreeShippingPromotion("Free Shipping Min 500", freeShippingOpts{
		promotionCreateOptions: promotionCreateOptions{canStack: boolPtr(true)},
		minOrderCents:          helpers.Int64Ptr(50000),
	})

	// Item discount requires min 400, cart is only 300
	s.createGenericFreeShippingPromotion(
		"Fixed 100 Min 400",
		promoTypeFixedAmount,
		model.FixedAmountConfig{AmountCents: 10000, MinOrderCents: helpers.Int64Ptr(40000)},
		promotionCreateOptions{canStack: boolPtr(true)},
	)

	summary := s.applyPromotions(
		buildCartRequestWithShipping(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			10000,
			cartItem("1", 1, 1, 30000, 1), // 30000 < all thresholds
		),
	)
	assertPromotionSummary(s.T(), summary, 30000, 0, 30000)
	s.Require().Equal(int64(0), summary.ShippingDiscount)
	s.Require().Empty(summary.AppliedPromotions)
}

// TestStackableItemDiscountMakesSubtotalBelowFreeShippingThreshold validates
// that when two stackable promos exist and the item discount reduces the cart
// below the free-shipping minimum, only the item discount applies.
func (s *FreeShippingPromotionTestSuite) TestStackableItemDiscountMakesSubtotalBelowFreeShippingThreshold() {
	// Item discount reduces subtotal from 70000 to 10000
	s.createGenericFreeShippingPromotion(
		"Fixed 600 Stackable",
		promoTypeFixedAmount,
		model.FixedAmountConfig{AmountCents: 60000},
		promotionCreateOptions{canStack: boolPtr(true)},
	)

	// Free shipping needs subtotal >= 50000; after the item discount it is 10000
	s.createFreeShippingPromotion("Free Shipping Min 500 Stackable", freeShippingOpts{
		promotionCreateOptions: promotionCreateOptions{canStack: boolPtr(true)},
		minOrderCents:          helpers.Int64Ptr(50000),
	})

	summary := s.applyPromotions(
		buildCartRequestWithShipping(
			helpers.Seller2UserID,
			helpers.CustomerUserID,
			10000,
			cartItem("1", 1, 1, 70000, 1),
		),
	)
	assertPromotionSummary(s.T(), summary, 70000, 60000, 10000)
	s.Require().Equal(int64(0), summary.ShippingDiscount, "threshold not met after item discount")
	s.Require().Len(summary.AppliedPromotions, 1, "only item discount should apply")
}

// ---------------------------------------------------------------------------
// Validation and Security
// ---------------------------------------------------------------------------

// TestCreateValidFreeShippingPromotion validates that a correctly formed free
// shipping payload (with no required fields, all optional) returns HTTP 201.
func (s *FreeShippingPromotionTestSuite) TestCreateValidFreeShippingPromotion() {
	payload := buildFreeShippingPayload("Valid Free Shipping", nil, nil)
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code)
}

// TestCreateFreeShippingWithMinOrderAndCap validates that a free shipping promotion
// with both optional fields set is accepted.
func (s *FreeShippingPromotionTestSuite) TestCreateFreeShippingWithMinOrderAndCap() {
	payload := buildFreeShippingPayload("Capped Free Shipping", helpers.Int64Ptr(50000), helpers.Int64Ptr(10000))
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code)
}

// TestCreateFreeShippingMissingName validates that a promotion payload without the
// required name field is rejected with HTTP 400.
func (s *FreeShippingPromotionTestSuite) TestCreateFreeShippingMissingName() {
	payload := map[string]interface{}{
		// "name" intentionally omitted
		"promotionType":  promoTypeFreeShipping,
		"discountConfig": model.FreeShippingConfig{},
		"appliesTo":      appliesAllProducts,
		"eligibleFor":    eligibleEveryone,
		"startsAt":       "2023-01-01T00:00:00Z",
		"endsAt":         "2029-12-31T23:59:59Z",
		"status":         promoStatusActive,
	}
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

// TestCrossTenantFreeShippingDoesNotLeak validates that a free shipping promotion
// created by Seller2 does not apply to a cart belonging to Seller4.
func (s *FreeShippingPromotionTestSuite) TestCrossTenantFreeShippingDoesNotLeak() {
	s.createFreeShippingPromotion("Seller2 Free Shipping", freeShippingOpts{})

	summary := s.applyPromotions(
		buildCartRequestWithShipping(
			helpers.Seller4UserID,
			helpers.Customer3UserID,
			10000,
			cartItem("1", 8, 16, 100000, 1),
		),
	)
	s.Require().Equal(int64(0), summary.ShippingDiscount)
	assertPromotionSummary(s.T(), summary, 100000, 0, 100000)
}

// TestUnauthorizedFreeShippingCreation validates that a customer token cannot
// create promotions (RBAC).
func (s *FreeShippingPromotionTestSuite) TestUnauthorizedFreeShippingCreation() {
	payload := buildFreeShippingPayload("Unauthorized", nil, nil)
	res := s.customerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusForbidden, res.Code)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (s *FreeShippingPromotionTestSuite) createFreeShippingPromotion(
	name string,
	opts freeShippingOpts,
) uint {
	config := model.FreeShippingConfig{
		MinOrderCents:            opts.minOrderCents,
		MaxShippingDiscountCents: opts.maxShippingDiscountCents,
	}
	return s.createGenericFreeShippingPromotion(
		name,
		promoTypeFreeShipping,
		config,
		opts.promotionCreateOptions,
	)
}

func (s *FreeShippingPromotionTestSuite) createGenericFreeShippingPromotion(
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

	return s.createPromotionFromFSPayload(payload)
}

func (s *FreeShippingPromotionTestSuite) createPromotionFromFSPayload(
	payload map[string]interface{},
) uint {
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code, "promotion creation should succeed")

	respData := helpers.ParseResponse(s.T(), res.Body)
	promo := respData["data"].(map[string]interface{})["promotion"].(map[string]interface{})
	return uint(promo["id"].(float64))
}

func (s *FreeShippingPromotionTestSuite) applyPromotions(
	cart *model.CartValidationRequest,
) *model.AppliedPromotionSummary {
	promotionService := singleton.GetInstance().GetPromotionService()
	summary, err := promotionService.ApplyPromotionsToCart(context.Background(), cart)
	s.Require().NoError(err)
	return summary
}

func (s *FreeShippingPromotionTestSuite) linkFreeShippingProducts(
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

// buildCartRequestWithShipping constructs a CartValidationRequest with a non-zero
// shipping cost — required for any free_shipping test to observe a discount.
func buildCartRequestWithShipping(
	sellerID uint,
	customerID uint,
	shippingCents int64,
	items ...model.CartItem,
) *model.CartValidationRequest {
	req := buildCartRequest(sellerID, customerID, items...)
	req.ShippingCents = shippingCents
	return req
}

// buildCartRequestFirstOrderWithShipping builds a cart with shipping and an
// explicit IsFirstOrder flag for new-customer eligibility tests.
func buildCartRequestFirstOrderWithShipping(
	sellerID uint,
	customerID uint,
	shippingCents int64,
	isFirstOrder bool,
	items ...model.CartItem,
) *model.CartValidationRequest {
	req := buildCartRequestWithShipping(sellerID, customerID, shippingCents, items...)
	req.IsFirstOrder = isFirstOrder
	return req
}

func buildFreeShippingPayload(
	name string,
	minOrderCents *int64,
	maxShippingDiscountCents *int64,
) map[string]interface{} {
	config := model.FreeShippingConfig{
		MinOrderCents:            minOrderCents,
		MaxShippingDiscountCents: maxShippingDiscountCents,
	}
	return map[string]interface{}{
		"name":           name,
		"promotionType":  promoTypeFreeShipping,
		"discountConfig": config,
		"appliesTo":      appliesAllProducts,
		"eligibleFor":    eligibleEveryone,
		"startsAt":       "2023-01-01T00:00:00Z",
		"endsAt":         "2029-12-31T23:59:59Z",
		"status":         promoStatusActive,
	}
}

func (s *FreeShippingPromotionTestSuite) cleanupPromotions() {
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
