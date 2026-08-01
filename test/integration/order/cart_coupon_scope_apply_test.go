package order_test

import (
	"net/http"

	"ecommerce-be/test/integration/helpers"
)

func (s *CartCouponTestSuite) TestScopedCouponNotApplicableForIneligibleCart() {
	codeID := s.createSellerDiscountCode(
		helpers.NewDiscountCodePayload("SCOPEMISS").
			Percentage(10).
			AppliesTo("specific_products").
			Build(),
	)
	s.addDiscountCodeProducts(codeID, 1) // iPhone product

	s.addCartItem(5, 1) // Samsung variant → product 2

	res := s.customerClient.Post(s.T(), CartCouponAPIEndpoint, helpers.ApplyCouponPayload("SCOPEMISS"))
	s.Require().Equal(http.StatusBadRequest, res.Code, res.Body.String())
	response := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("COUPON_NOT_APPLICABLE", response["code"])
}

func (s *CartCouponTestSuite) TestScopedCouponAppliesToEligibleItems() {
	codeID := s.createSellerDiscountCode(
		helpers.NewDiscountCodePayload("SCOPEHIT").
			Percentage(10).
			AppliesTo("specific_products").
			Build(),
	)
	s.addDiscountCodeProducts(codeID, 1)

	s.addCartItem(1, 1) // iPhone product 1

	response := s.applyCouponOK("SCOPEHIT")
	cart := response["data"].(map[string]any)
	s.Require().Len(cart["appliedCoupons"].([]any), 1)
	s.Equal(float64(9990), cart["summary"].(map[string]any)["couponDiscount"])
}

func (s *CartCouponTestSuite) TestScopedCouponOnlyDiscountsEligibleLines() {
	codeID := s.createSellerDiscountCode(
		helpers.NewDiscountCodePayload("SCOPEMIX").
			Percentage(10).
			AppliesTo("specific_products").
			Build(),
	)
	s.addDiscountCodeProducts(codeID, 1)

	s.addCartItem(1, 1) // product 1 @ 99900
	s.addCartItem(5, 1) // product 2 @ 79900 — out of scope

	response := s.applyCouponOK("SCOPEMIX")
	cart := response["data"].(map[string]any)
	// 10% of eligible line only (99900)
	s.Equal(float64(9990), cart["summary"].(map[string]any)["couponDiscount"])
}
