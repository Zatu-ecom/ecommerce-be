package order_test

import (
	"net/http"

	"ecommerce-be/test/integration/helpers"
)

func (s *CartCouponTestSuite) TestApplyPercentageCouponHappyPath() {
	s.createSellerDiscountCode(helpers.PercentageDiscountCode("SAVE10", 10))
	s.addCartItem(1, 1) // 99900 cents

	response := s.applyCouponOK("SAVE10")
	cart := response["data"].(map[string]any)
	summary := cart["summary"].(map[string]any)
	coupons := cart["appliedCoupons"].([]any)

	s.Require().Len(coupons, 1)
	applied := coupons[0].(map[string]any)
	s.Equal("SAVE10", applied["code"])
	s.Equal("percentage", applied["discountType"])
	s.Equal(float64(9990), applied["discount"]) // 10% of 99900
	s.Equal(float64(0), applied["shippingDiscount"])
	s.Equal(float64(1), summary["couponCount"])
	s.Equal(float64(9990), summary["couponDiscount"])
	s.Greater(summary["totalDiscount"].(float64), float64(0))
}

func (s *CartCouponTestSuite) TestApplyFreeShippingCoupon() {
	s.createSellerDiscountCode(helpers.NewDiscountCodePayload("FREESHIP").FreeShipping().Build())
	s.addCartItem(1, 1)

	response := s.applyCouponOK("FREESHIP")
	cart := response["data"].(map[string]any)
	applied := cart["appliedCoupons"].([]any)[0].(map[string]any)
	s.Equal("free_shipping", applied["discountType"])
	s.Equal(float64(5000), applied["shippingDiscount"]) // hardcoded shipping in cart build
	s.NotEmpty(applied["shippingDiscountFormatted"])
}

func (s *CartCouponTestSuite) TestApplyBuyXGetYCoupon() {
	s.createSellerDiscountCode(
		helpers.NewDiscountCodePayload("BXGY1").BuyXGetY(1, 1, 100).Build(),
	)
	s.addCartItem(1, 2) // two units @ 99900

	response := s.applyCouponOK("BXGY1")
	cart := response["data"].(map[string]any)
	applied := cart["appliedCoupons"].([]any)[0].(map[string]any)
	s.Equal("buy_x_get_y", applied["discountType"])
	s.Equal(float64(99900), applied["discount"]) // one free unit
}

func (s *CartCouponTestSuite) TestRemoveCouponRevertsTotals() {
	s.createSellerDiscountCode(helpers.PercentageDiscountCode("REM10", 10))
	s.addCartItem(1, 1)

	applied := s.applyCouponOK("REM10")
	beforeTotal := applied["data"].(map[string]any)["summary"].(map[string]any)["total"].(float64)

	res := s.customerClient.Delete(s.T(), couponURL("REM10"), nil)
	s.Require().Equal(http.StatusOK, res.Code, res.Body.String())
	response := helpers.ParseResponse(s.T(), res.Body)
	cart := response["data"].(map[string]any)
	s.Empty(cart["appliedCoupons"])
	afterTotal := cart["summary"].(map[string]any)["total"].(float64)
	s.Greater(afterTotal, beforeTotal)
}

func (s *CartCouponTestSuite) TestRemoveAllCoupons() {
	s.createSellerDiscountCode(helpers.PercentageDiscountCode("ALL1", 5))
	s.addCartItem(1, 1)
	s.applyCouponOK("ALL1")

	res := s.customerClient.Delete(s.T(), CartCouponAPIEndpoint, nil)
	s.Require().Equal(http.StatusOK, res.Code, res.Body.String())
	response := helpers.ParseResponse(s.T(), res.Body)
	s.Empty(response["data"].(map[string]any)["appliedCoupons"])
}

func (s *CartCouponTestSuite) TestListAvailableCoupons() {
	s.createSellerDiscountCode(helpers.PercentageDiscountCode("AVAIL1", 10))
	s.addCartItem(1, 1)

	res := s.customerClient.Get(s.T(), CartAvailableCouponAPIEndpoint)
	s.Require().Equal(http.StatusOK, res.Code, res.Body.String())
	response := helpers.ParseResponse(s.T(), res.Body)
	data := response["data"].(map[string]any)
	applicable := data["applicable"].([]any)
	s.NotEmpty(applicable)
}
