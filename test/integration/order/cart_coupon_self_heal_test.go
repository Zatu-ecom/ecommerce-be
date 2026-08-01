package order_test

import (
	"net/http"

	orderEntity "ecommerce-be/order/entity"
	"ecommerce-be/test/integration/helpers"
)

func (s *CartCouponTestSuite) TestSelfHealRemovesDeactivatedCoupon() {
	codeID := s.createSellerDiscountCode(helpers.PercentageDiscountCode("HEALOFF", 10))
	s.addCartItem(1, 1)
	s.applyCouponOK("HEALOFF")
	s.Equal(int64(1), s.countCartAppliedCoupons())

	s.deactivateDiscountCode(codeID)

	response := s.getCartOK()
	cart := response["data"].(map[string]any)
	s.Empty(cart["appliedCoupons"])
	s.Equal(float64(0), cart["summary"].(map[string]any)["couponCount"])
	s.Equal(float64(0), cart["summary"].(map[string]any)["couponDiscount"])
	s.Equal(int64(0), s.countCartAppliedCoupons())
}

func (s *CartCouponTestSuite) TestSelfHealRemovesExpiredCoupon() {
	codeID := s.createSellerDiscountCode(helpers.PercentageDiscountCode("HEALEXP", 10))
	s.addCartItem(1, 1)
	s.applyCouponOK("HEALEXP")

	s.updateDiscountCode(codeID, map[string]any{
		"startsAt": helpers.PastRFC3339(30),
		"endsAt":   helpers.PastRFC3339(1),
	})

	response := s.getCartOK()
	cart := response["data"].(map[string]any)
	s.Empty(cart["appliedCoupons"])
	s.Equal(int64(0), s.countCartAppliedCoupons())
}

func (s *CartCouponTestSuite) TestSelfHealRemovesWhenMinPurchaseNoLongerMet() {
	codeID := s.createSellerDiscountCode(helpers.PercentageDiscountCode("HEALMIN", 10))
	s.addCartItem(1, 1) // 99900 cents
	s.applyCouponOK("HEALMIN")

	minPurchase := int64(200000)
	s.updateDiscountCode(codeID, map[string]any{
		"minPurchaseAmountCents": minPurchase,
	})

	response := s.getCartOK()
	cart := response["data"].(map[string]any)
	s.Empty(cart["appliedCoupons"])
	s.Equal(int64(0), s.countCartAppliedCoupons())

	// Ensure no orphaned junction rows for this cart
	var cartID uint
	s.Require().NoError(s.container.DB.Model(&orderEntity.Cart{}).
		Where("user_id = ? AND status = ?", helpers.CustomerUserID, "active").
		Select("id").Scan(&cartID).Error)
	var count int64
	s.Require().NoError(s.container.DB.Model(&orderEntity.CartAppliedCoupon{}).
		Where("cart_id = ?", cartID).Count(&count).Error)
	s.Equal(int64(0), count)
}

func (s *CartCouponTestSuite) TestGetCartAfterSelfHealReturnsOK() {
	codeID := s.createSellerDiscountCode(helpers.PercentageDiscountCode("HEALOK", 15))
	s.addCartItem(1, 1)
	s.applyCouponOK("HEALOK")
	s.deactivateDiscountCode(codeID)

	res := s.customerClient.Get(s.T(), "/api/order/cart")
	s.Require().Equal(http.StatusOK, res.Code, res.Body.String())
}
