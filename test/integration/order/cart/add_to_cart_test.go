package CartTest

import (
	"net/http"
	"testing"

	orderEntity "ecommerce-be/order/entity"
	promotionEntity "ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/model"
	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const msgItemAddedToCart = "Item added to cart"

// Variant 1: iPhone NAT-128, price 999.00 -> 99900 cents (seed seller 2)
// Variant 2: iPhone NAT-256, 1099.00 -> 109900 cents
// Variant 5: Samsung BLK-128, price 799.00 -> 79900 cents
const (
	unitPriceCentsVariant1 = int64(99900)
	unitPriceCentsVariant2 = int64(109900)
	unitPriceCentsVariant5 = int64(79900)
)

// AddToCartTestSuite exercises POST /api/order/item (CustomerAuth + promotion pipeline).
// Pattern: bundle_test.go (suite, seller/customer clients, createBundlePromotion, cleanupPromotions).
type AddToCartTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler

	sellerClient   *helpers.APIClient
	customerClient *helpers.APIClient
}

func (s *AddToCartTestSuite) SetupSuite() {
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

func (s *AddToCartTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func (s *AddToCartTestSuite) SetupTest() {
	s.cleanupPromotionsCart()
}

func TestAddToCartIntegration(t *testing.T) {
	suite.Run(t, new(AddToCartTestSuite))
}

func (s *AddToCartTestSuite) cleanupPromotionsCart() {
	s.cleanupPromotions()
	s.cleanupCartsForTestUsers()
}

func (s *AddToCartTestSuite) cleanupPromotions() {
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

func (s *AddToCartTestSuite) cleanupCartsForTestUsers() {
	uids := []uint{helpers.CustomerUserID, helpers.Customer2UserID, helpers.Seller2UserID}
	var cartIDs []uint
	_ = s.container.DB.Model(&orderEntity.Cart{}).
		Where("user_id IN ?", uids).
		Pluck("id", &cartIDs)
	if len(cartIDs) == 0 {
		return
	}
	s.Require().NoError(
		s.container.DB.Where("cart_id IN ?", cartIDs).Delete(&orderEntity.CartItem{}).Error,
	)
	s.Require().NoError(
		s.container.DB.Where("id IN ?", cartIDs).Delete(&orderEntity.Cart{}).Error,
	)
}

// --- Happy path ---

func (s *AddToCartTestSuite) TestHP001AddFirstItem() {
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": uint(1),
		"quantity":  1,
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	assert.Equal(s.T(), msgItemAddedToCart, resp["message"].(string))
	data := resp["data"].(map[string]any)
	s.assertCartEnvelope(data, helpers.CustomerUserID)
	items := data["items"].([]any)
	assert.Len(s.T(), items, 1)
	item := items[0].(map[string]any)
	s.assertCartItemPricing(item, 1, 1, unitPriceCentsVariant1, unitPriceCentsVariant1)
	s.assertSummaryBasics(data, 1, 1)
}

func (s *AddToCartTestSuite) TestHP002SameVariantAccumulates() {
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": uint(1),
		"quantity":  1,
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	w = s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": uint(1),
		"quantity":  1,
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	assert.Len(s.T(), items, 1)
	item := items[0].(map[string]any)
	s.assertCartItemPricing(item, 1, 2, unitPriceCentsVariant1, unitPriceCentsVariant1*2)
	s.assertSummaryBasics(data, 2, 1)
}

func (s *AddToCartTestSuite) TestHP003TwoDifferentVariants() {
	s.cleanupCartsForTestUsers()
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": uint(1),
		"quantity":  1,
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	w = s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": uint(5),
		"quantity":  1,
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	assert.Len(s.T(), items, 2)
	s.assertSummaryBasics(data, 2, 2)
	summary := data["summary"].(map[string]any)
	assert.Equal(s.T(), float64(2), summary["uniqueItems"])
	assert.Equal(s.T(), float64(2), summary["itemCount"])
}

func (s *AddToCartTestSuite) TestHP004QuantityMinBoundary() {
	s.cleanupCartsForTestUsers()
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": uint(2),
		"quantity":  1,
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	require.Len(s.T(), items, 1)
	item := items[0].(map[string]any)
	s.assertCartItemPricing(item, 2, 1, unitPriceCentsVariant2, unitPriceCentsVariant2)
	_ = resp
}

func (s *AddToCartTestSuite) TestHP005QuantityMaxBoundary() {
	s.cleanupCartsForTestUsers()
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": uint(2),
		"quantity":  50,
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
}

// --- Auth ---

func (s *AddToCartTestSuite) TestNEG001NoAuth() {
	cl := helpers.NewAPIClient(s.server)
	cl.SetToken("")
	w := cl.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": 1,
		"quantity":  1,
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
}

func (s *AddToCartTestSuite) TestNEG002InvalidToken() {
	cl := helpers.NewAPIClient(s.server)
	cl.SetToken("not-a-valid-jwt")
	w := cl.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": 1,
		"quantity":  1,
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
}

// --- Authorization: seller can use cart ---

func (s *AddToCartTestSuite) TestNEGAuthz001SellerCanAddToCart() {
	s.cleanupCartsForTestUsers()
	sellerCl := helpers.NewAPIClient(s.server)
	tok := helpers.Login(s.T(), sellerCl, helpers.Seller2Email, helpers.Seller2Password)
	sellerCl.SetToken(tok)
	w := sellerCl.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": uint(1),
		"quantity":  1,
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	// Seller2 user id = 2
	assert.Equal(s.T(), float64(helpers.Seller2UserID), data["userId"])
}

// --- Validation ---

func (s *AddToCartTestSuite) TestNEGValMissingVariantId() {
	s.customerClient.SetToken(
		helpers.Login(s.T(), s.customerClient, helpers.CustomerEmail, helpers.CustomerPassword),
	)
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"quantity": 1,
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *AddToCartTestSuite) TestNEGValMissingQuantity() {
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": 1,
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *AddToCartTestSuite) TestNEGValQuantityZero() {
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": 1,
		"quantity":  0,
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *AddToCartTestSuite) TestNEGValQuantityOver99() {
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": 1,
		"quantity":  100,
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *AddToCartTestSuite) TestNEGValQuantityNegative() {
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": 1,
		"quantity":  -1,
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *AddToCartTestSuite) TestNEGValEmptyBody() {
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *AddToCartTestSuite) TestNEGValMalformedJSON() {
	w := s.customerClient.PostRaw(s.T(), CartItemAPIEndpoint, []byte(`{`))
	assert.True(s.T(), w.Code == http.StatusBadRequest || w.Code == http.StatusInternalServerError)
}

func (s *AddToCartTestSuite) TestNEGValVariantIdWrongType() {
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": "not-a-number",
		"quantity":  1,
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// --- Business ---

func (s *AddToCartTestSuite) TestNEGBizUnknownVariant() {
	s.cleanupCartsForTestUsers()
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": uint(999999),
		"quantity":  1,
	})
	resp := helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
	code, _ := resp["code"].(string)
	assert.Equal(s.T(), "VARIANT_NOT_FOUND", code)
}

func (s *AddToCartTestSuite) TestNEGBizInsufficientStock() {
	s.cleanupCartsForTestUsers()
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": uint(2),
		"quantity":  80,
	})
	resp := helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
	code, _ := resp["code"].(string)
	assert.Equal(s.T(), "INSUFFICIENT_STOCK", code)
}

func (s *AddToCartTestSuite) TestNEGValQuantityAboveAllowedRange() {
	s.cleanupCartsForTestUsers()
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": uint(1),
		"quantity":  111,
	})
	resp := helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
	code, _ := resp["code"].(string)
	assert.Equal(s.T(), "VALIDATION_ERROR", code)
}

func (s *AddToCartTestSuite) TestNEGBizCrossSellerVariant() {
	s.cleanupCartsForTestUsers()
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": uint(9),
		"quantity":  1,
	})
	resp := helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
	code, _ := resp["code"].(string)
	assert.Equal(s.T(), "VARIANT_NOT_FOUND", code)
}

// --- Edge ---

func (s *AddToCartTestSuite) TestEDGEVariantIdZero() {
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": 0,
		"quantity":  1,
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *AddToCartTestSuite) TestEDGEVariantIdVeryLarge() {
	s.cleanupCartsForTestUsers()
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": uint(999999999),
		"quantity":  1,
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// --- Promotion ---

func (s *AddToCartTestSuite) TestPROMO001NoPromotionConfigured() {
	s.cleanupCartsForTestUsers()
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": uint(1),
		"quantity":  1,
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	summary := data["summary"].(map[string]any)
	assert.Equal(s.T(), float64(0), summary["promotionCount"])
	items := data["items"].([]any)
	item := items[0].(map[string]any)
	apps, _ := item["appliedPromotions"].([]any)
	assert.Len(s.T(), apps, 0)
}

func (s *AddToCartTestSuite) TestPROMO002BundleFixedPrice() {
	createBundlePromotion(
		s.T(),
		s.sellerClient,
		"Phone + Samsung Fixed Price",
		bundleDiscountTypeFixedPrice,
		nil,
		helpers.Int64Ptr(150000),
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(2, 5, 1),
		},
	)

	s.cleanupCartsForTestUsers()
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": uint(1),
		"quantity":  1,
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	w = s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": uint(5),
		"quantity":  1,
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	summary := data["summary"].(map[string]any)
	// Aligns with bundle_test TestScenario2: subtotal 179800, discount 29800, final 150000
	assert.Equal(s.T(), float64(179800), summary["subtotal"])
	assert.Equal(s.T(), float64(29800), summary["promotionDiscount"])
	assert.Equal(s.T(), float64(150000), summary["total"])
	assert.Equal(s.T(), float64(150000), summary["afterDiscount"])
}

func (s *AddToCartTestSuite) TestPROMO004MultipleLinesAggregate() {
	s.cleanupCartsForTestUsers()
	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": uint(1),
		"quantity":  2,
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	w = s.customerClient.Post(s.T(), CartItemAPIEndpoint, map[string]any{
		"variantId": uint(5),
		"quantity":  1,
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	summary := data["summary"].(map[string]any)
	assert.Equal(s.T(), float64(3), summary["itemCount"])
	assert.Equal(s.T(), float64(2), summary["uniqueItems"])
	_ = resp
}

// --- Assertions ---

func (s *AddToCartTestSuite) assertCartEnvelope(data map[string]any, expectedUserID uint) {
	assert.NotNil(s.T(), data["id"])
	assert.Equal(s.T(), float64(expectedUserID), data["userId"])
	curr, ok := data["currency"].(map[string]any)
	require.True(s.T(), ok)
	assert.NotEmpty(s.T(), curr["code"])
	assert.NotEmpty(s.T(), curr["symbol"])
	_, hasDecimal := curr["decimalDigits"]
	assert.True(s.T(), hasDecimal)
	meta, ok := data["metadata"].(map[string]any)
	require.True(s.T(), ok)
	assert.NotNil(s.T(), meta)
}

func (s *AddToCartTestSuite) assertCartItemPricing(
	item map[string]any,
	expectedVariantID uint,
	expectedQty int,
	expectedUnit int64,
	expectedLine int64,
) {
	assert.Equal(s.T(), float64(expectedVariantID), item["variantId"])
	assert.Equal(s.T(), float64(expectedQty), item["quantity"])
	assert.Equal(s.T(), float64(expectedUnit), item["unitPrice"])
	assert.Equal(s.T(), float64(expectedLine), item["lineTotal"])
	assert.Equal(s.T(), float64(expectedLine), item["discountedLineTotal"])
	v, ok := item["variant"].(map[string]any)
	require.True(s.T(), ok)
	assert.Equal(s.T(), float64(expectedVariantID), v["id"])
	assert.NotEmpty(s.T(), v["sku"])
	p, ok := v["product"].(map[string]any)
	require.True(s.T(), ok)
	assert.NotEmpty(s.T(), p["name"])
}

func (s *AddToCartTestSuite) assertSummaryBasics(
	data map[string]any,
	expectedItemCount int,
	expectedUnique int,
) {
	summary := data["summary"].(map[string]any)
	assert.Equal(s.T(), float64(expectedItemCount), summary["itemCount"])
	assert.Equal(s.T(), float64(expectedUnique), summary["uniqueItems"])
	coupons, _ := data["appliedCoupons"].([]any)
	assert.NotNil(s.T(), coupons)
}
