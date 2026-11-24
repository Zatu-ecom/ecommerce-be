package order_test

import (
	"net/http"

	orderEntity "ecommerce-be/order/entity"
	promotionEntity "ecommerce-be/promotion/entity"
	"ecommerce-be/test/integration/helpers"
)

func (s *OrderSuite) TestCheckoutSnapshotsAppliedCouponAndRecordsUsage() {
	codeID := s.createSellerDiscountCode(helpers.PercentageDiscountCode("CHKOUT10", 10))
	s.addItemToCart(1, 1)
	s.applyCoupon("CHKOUT10")

	orderData := s.createOrderOK()
	orderID := uint(orderData["id"].(float64))

	// Snapshot on create response / GET
	w := s.customerClient.Get(s.T(), s.getOrderByIDURL(orderID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	coupons, ok := data["appliedCoupons"].([]any)
	s.Require().True(ok)
	s.Require().Len(coupons, 1)
	applied := coupons[0].(map[string]any)
	s.Equal("CHKOUT10", applied["couponCode"])
	s.Equal("percentage", applied["discountType"])
	s.Greater(helpers.MoneyCents(applied["discount"]), int64(0))

	var snap orderEntity.OrderAppliedCoupon
	s.Require().NoError(
		s.container.DB.Where("order_id = ?", orderID).First(&snap).Error,
	)
	s.Equal("CHKOUT10", snap.CouponCode)
	s.Require().NotNil(snap.DiscountCodeID)
	s.Equal(codeID, *snap.DiscountCodeID)
	s.Greater(snap.DiscountCents, int64(0))

	var usage promotionEntity.DiscountCodeUsage
	s.Require().NoError(
		s.container.DB.Where("discount_code_id = ? AND order_id = ?", codeID, orderID).
			First(&usage).Error,
	)
	s.Equal(uint(helpers.CustomerUserID), usage.UserID)
	s.Greater(usage.DiscountAmountCents, int64(0))

	var dc promotionEntity.DiscountCode
	s.Require().NoError(s.container.DB.First(&dc, codeID).Error)
	s.Equal(1, dc.CurrentUsageCount)

	// New active cart must not inherit converted cart coupons
	w = s.customerClient.Get(s.T(), "/api/order/cart")
	cartResp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	cart := cartResp["data"].(map[string]any)
	s.Empty(cart["appliedCoupons"])
}

func (s *OrderSuite) TestCheckoutOrderDiscountIncludesCoupon() {
	s.createSellerDiscountCode(helpers.PercentageDiscountCode("CHKDISC", 10))
	s.addItemToCart(1, 1)
	s.applyCoupon("CHKDISC")

	orderData := s.createOrderOK()
	s.Greater(helpers.MoneyCents(orderData["discount"]), int64(0))
	s.Greater(helpers.MoneyCents(orderData["subtotal"]), helpers.MoneyCents(orderData["total"]))
}

func (s *OrderSuite) TestConvertedCartCouponsNotOnNewActiveCart() {
	s.createSellerDiscountCode(helpers.PercentageDiscountCode("CHKNEW", 5))
	s.addItemToCart(1, 1)
	s.applyCoupon("CHKNEW")
	orderData := s.createOrderOK()
	orderID := uint(orderData["id"].(float64))

	var converted orderEntity.Cart
	s.Require().NoError(
		s.container.DB.Where("order_id = ? AND status = ?", orderID, orderEntity.CART_STATUS_CONVERTED).
			First(&converted).Error,
	)
	var convertedCouponCount int64
	s.Require().NoError(
		s.container.DB.Model(&orderEntity.CartAppliedCoupon{}).
			Where("cart_id = ?", converted.ID).Count(&convertedCouponCount).Error,
	)
	s.Equal(int64(1), convertedCouponCount)

	// New storefront cart view must not surface converted cart coupons.
	w := s.customerClient.Get(s.T(), "/api/order/cart")
	cartResp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	s.Empty(cartResp["data"].(map[string]any)["appliedCoupons"])

	// Adding a new item creates a fresh active cart without inherited coupons.
	s.addItemToCart(1, 1)
	w = s.customerClient.Get(s.T(), "/api/order/cart")
	cartResp = helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	s.Empty(cartResp["data"].(map[string]any)["appliedCoupons"])

	var active orderEntity.Cart
	s.Require().NoError(
		s.container.DB.Where("user_id = ? AND status = ?", helpers.CustomerUserID, orderEntity.CART_STATUS_ACTIVE).
			First(&active).Error,
	)
	s.NotEqual(converted.ID, active.ID)
	var activeCouponCount int64
	s.Require().NoError(
		s.container.DB.Model(&orderEntity.CartAppliedCoupon{}).
			Where("cart_id = ?", active.ID).Count(&activeCouponCount).Error,
	)
	s.Equal(int64(0), activeCouponCount)
}
