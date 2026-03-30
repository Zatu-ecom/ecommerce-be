package CartTest

import (
	"fmt"
	"net/http"

	"ecommerce-be/promotion/model"
	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *CartTestSuite) TestGET001ReturnsEmptyCartWhenNoActiveCart() {
	s.cleanupCartsForTestUsers()

	w := s.customerClient.Get(s.T(), CartAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	appliedPromotions := data["appliedPromotions"].([]any)
	assert.Len(s.T(), items, 0)
	assert.Len(s.T(), appliedPromotions, 0)
	assert.Equal(s.T(), float64(helpers.CustomerUserID), data["userId"])
}

func (s *CartTestSuite) TestGET002ReturnsFullCartResponseForActiveCart() {
	s.cleanupCartsForTestUsers()

	w := s.customerClient.Post(
		s.T(),
		CartItemAPIEndpoint,
		cartItemsPayload(cartItem(uint(1), 1), cartItem(uint(5), 2)),
	)
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	w = s.customerClient.Get(s.T(), CartAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	require.Len(s.T(), items, 2)
	s.assertSummaryBasics(data, 3, 2)
}

func (s *CartTestSuite) TestGET003NoAuth() {
	s.cleanupCartsForTestUsers()
	cl := helpers.NewAPIClient(s.server)
	cl.SetToken("")

	w := cl.Get(s.T(), CartAPIEndpoint)
	helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
}

func (s *CartTestSuite) TestGET004InvalidToken() {
	s.cleanupCartsForTestUsers()
	cl := helpers.NewAPIClient(s.server)
	cl.SetToken("invalid-token")

	w := cl.Get(s.T(), CartAPIEndpoint)
	helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
}

func (s *CartTestSuite) TestGET005AfterCartDeletedReturnsEmptyCart() {
	s.cleanupCartsForTestUsers()

	w := s.customerClient.Post(
		s.T(),
		CartItemAPIEndpoint,
		cartItemsPayload(cartItem(uint(1), 1), cartItem(uint(5), 1)),
	)
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	cartID := s.getActiveCartID(helpers.CustomerUserID)

	w = s.customerClient.Delete(s.T(), fmt.Sprintf("%s/%d", CartAPIEndpoint, cartID))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	w = s.customerClient.Get(s.T(), CartAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	assert.Len(s.T(), items, 0)
	assert.Equal(s.T(), float64(helpers.CustomerUserID), data["userId"])
}

func (s *CartTestSuite) TestGET006SellerCanFetchOwnCart() {
	s.cleanupCartsForTestUsers()

	sellerCl := helpers.NewAPIClient(s.server)
	tok := helpers.Login(s.T(), sellerCl, helpers.Seller2Email, helpers.Seller2Password)
	sellerCl.SetToken(tok)

	w := sellerCl.Post(s.T(), CartItemAPIEndpoint, cartItemsPayload(cartItem(uint(1), 2)))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	w = sellerCl.Get(s.T(), CartAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	assert.Equal(s.T(), float64(helpers.Seller2UserID), data["userId"])
	items := data["items"].([]any)
	require.Len(s.T(), items, 1)
	item := items[0].(map[string]any)
	s.assertCartItemPricing(item, 1, 2, unitPriceCentsVariant1, unitPriceCentsVariant1*2)
}

func (s *CartTestSuite) TestGET007InvalidTokenRejected() {
	s.cleanupCartsForTestUsers()
	cl := helpers.NewAPIClient(s.server)
	cl.SetToken("bad-token")

	w := cl.Get(s.T(), CartAPIEndpoint)
	helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
}

func (s *CartTestSuite) TestGET008StaleCartWithoutItemsReturnsEmptyResponse() {
	s.cleanupCartsForTestUsers()

	_ = s.createCartOnly(helpers.CustomerUserID)

	w := s.customerClient.Get(s.T(), CartAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	assert.Len(s.T(), items, 0)
	assert.Equal(s.T(), float64(helpers.CustomerUserID), data["userId"])
}

func (s *CartTestSuite) TestGET009PromotionSummaryIsAppliedOnGetCart() {
	s.cleanupCartsForTestUsers()
	createBundlePromotion(
		s.T(),
		s.sellerClient,
		"GET Cart Promo Bundle",
		bundleDiscountTypeFixedPrice,
		nil,
		helpers.Int64Ptr(150000),
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(2, 5, 1),
		},
	)

	w := s.customerClient.Post(
		s.T(),
		CartItemAPIEndpoint,
		cartItemsPayload(cartItem(uint(1), 1), cartItem(uint(5), 1)),
	)
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	w = s.customerClient.Get(s.T(), CartAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	summary := data["summary"].(map[string]any)
	appliedPromotions := data["appliedPromotions"].([]any)

	assert.Equal(s.T(), float64(179800), summary["subtotal"])
	assert.Equal(s.T(), float64(29800), summary["promotionDiscount"])
	assert.Equal(s.T(), float64(150000), summary["afterDiscount"])
	assert.Equal(s.T(), float64(150000), summary["total"])
	require.Len(s.T(), appliedPromotions, 1)
	ap := appliedPromotions[0].(map[string]any)
	assert.Equal(s.T(), float64(29800), ap["discount"])
	assert.Equal(s.T(), float64(0), ap["shippingDiscount"])
	assert.NotEmpty(s.T(), ap["name"])
	assert.NotEmpty(s.T(), ap["type"])
}

func (s *CartTestSuite) TestGET010PromotionLineItemsContainAppliedPromotions() {
	s.cleanupCartsForTestUsers()
	createBundlePromotion(
		s.T(),
		s.sellerClient,
		"GET Cart Item Promotions",
		bundleDiscountTypeFixedPrice,
		nil,
		helpers.Int64Ptr(150000),
		[]model.BundleItemConfig{
			bundleItem(1, 1, 1),
			bundleItem(2, 5, 1),
		},
	)

	w := s.customerClient.Post(
		s.T(),
		CartItemAPIEndpoint,
		cartItemsPayload(cartItem(uint(1), 1), cartItem(uint(5), 1)),
	)
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	w = s.customerClient.Get(s.T(), CartAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	require.Len(s.T(), items, 2)

	totalApplied := 0
	for _, raw := range items {
		item := raw.(map[string]any)
		applied, ok := item["appliedPromotions"].([]any)
		require.True(s.T(), ok)
		totalApplied += len(applied)
	}
	assert.Greater(s.T(), totalApplied, 0)
}

func (s *CartTestSuite) TestGET011FreeShippingPromotionAppearsAtCartLevel() {
	s.cleanupCartsForTestUsers()

	_ = postPromotion(s.T(), s.sellerClient, map[string]any{
		"name":          "GET Cart Free Shipping",
		"promotionType": "free_shipping",
		"discountConfig": map[string]any{
			"min_order_cents":             1000,
			"max_shipping_discount_cents": 5000,
		},
		"appliesTo":   appliesAllProducts,
		"eligibleFor": eligibleEveryone,
		"startsAt":    "2023-01-01T00:00:00Z",
		"endsAt":      "2029-12-31T23:59:59Z",
		"status":      promoStatusActive,
	})

	w := s.customerClient.Post(
		s.T(),
		CartItemAPIEndpoint,
		cartItemsPayload(cartItem(uint(1), 1)),
	)
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	w = s.customerClient.Get(s.T(), CartAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	appliedPromotions := data["appliedPromotions"].([]any)
	require.NotEmpty(s.T(), appliedPromotions)

	foundFreeShipping := false
	for _, raw := range appliedPromotions {
		ap := raw.(map[string]any)
		promoType, _ := ap["type"].(string)
		if promoType != "free_shipping" {
			continue
		}
		foundFreeShipping = true
		assert.Equal(s.T(), float64(0), ap["discount"])
		assert.Equal(s.T(), float64(5000), ap["shippingDiscount"])
		assert.NotEmpty(s.T(), ap["shippingDiscountFormatted"])
	}
	assert.True(s.T(), foundFreeShipping)
}
