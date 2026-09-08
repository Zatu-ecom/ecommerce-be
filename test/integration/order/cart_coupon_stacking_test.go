package order_test

import (
	"fmt"
	"net/http"

	"ecommerce-be/test/integration/helpers"
)

func (s *CartCouponTestSuite) TestStackingRejectsCouponWhenPromoDisallowsCoupons() {
	s.createSellerPromotion(
		helpers.NewPercentagePromotion("NoCouponStack", 5).CanStackWithCoupons(false).Build(),
	)

	s.createSellerDiscountCode(helpers.PercentageDiscountCode("STACKPROMO", 10))
	s.addCartItem(1, 1)

	// Ensure promotion is on the cart before coupon apply
	cartResp := s.getCartOK()
	promos := cartResp["data"].(map[string]any)["appliedPromotions"]
	s.Require().NotNil(promos)
	s.Require().NotEmpty(promos.([]any))

	res := s.customerClient.Post(s.T(), CartCouponAPIEndpoint, helpers.ApplyCouponPayload("STACKPROMO"))
	s.Require().Equal(http.StatusBadRequest, res.Code, res.Body.String())
	response := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("COUPON_CANNOT_COMBINE", response["code"])
}

func (s *CartCouponTestSuite) TestStackingAllowsCouponWhenPromoAllowsCoupons() {
	s.createSellerPromotion(
		helpers.NewPercentagePromotion("CouponStackOK", 5).CanStackWithCoupons(true).Build(),
	)

	s.createSellerDiscountCode(helpers.PercentageDiscountCode("STACKOK", 10))
	s.addCartItem(1, 1)

	response := s.applyCouponOK("STACKOK")
	cart := response["data"].(map[string]any)
	s.Require().Len(cart["appliedCoupons"].([]any), 1)
	s.Require().NotEmpty(cart["appliedPromotions"].([]any))
	summary := cart["summary"].(map[string]any)
	s.Greater(helpers.MoneyCents(summary["promotionDiscount"]), int64(0))
	s.Greater(helpers.MoneyCents(summary["couponDiscount"]), int64(0))
}

func (s *CartCouponTestSuite) TestStackingTwoCombinableCoupons() {
	s.createSellerDiscountCode(
		helpers.NewDiscountCodePayload("COMBINEA").Percentage(5).CanCombineWithOtherDiscounts(true).Build(),
	)
	s.createSellerDiscountCode(
		helpers.NewDiscountCodePayload("COMBINEB").Percentage(5).CanCombineWithOtherDiscounts(true).Build(),
	)

	s.addCartItem(1, 1)
	s.applyCouponOK("COMBINEA")
	response := s.applyCouponOK("COMBINEB")
	cart := response["data"].(map[string]any)
	s.Len(cart["appliedCoupons"].([]any), 2)
}

func (s *CartCouponTestSuite) TestStackingSelfHealDropsCouponWhenPromoBecomesNonStackable() {
	promoID := s.createSellerPromotion(
		helpers.NewPercentagePromotion("LaterBlockCoupons", 5).CanStackWithCoupons(true).Build(),
	)

	s.createSellerDiscountCode(helpers.PercentageDiscountCode("HEALSTACK", 10))
	s.addCartItem(1, 1)
	s.applyCouponOK("HEALSTACK")
	s.Equal(int64(1), s.countCartAppliedCoupons())

	res := s.sellerClient.Put(s.T(), fmt.Sprintf("/api/promotion/%d", promoID), map[string]any{
		"canStackWithCoupons": false,
	})
	s.Require().Equal(http.StatusOK, res.Code, res.Body.String())

	response := s.getCartOK()
	cart := response["data"].(map[string]any)
	s.Empty(cart["appliedCoupons"])
	s.Equal(int64(0), s.countCartAppliedCoupons())
}
