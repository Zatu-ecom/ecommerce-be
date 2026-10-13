package order_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	commonError "ecommerce-be/common/error"
	orderSingleton "ecommerce-be/order/factory/singleton"
	promotionEntity "ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	promotionSingleton "ecommerce-be/promotion/factory/singleton"
	"ecommerce-be/test/integration/helpers"
)

func (s *OrderSuite) TestEnsureCouponsValidAtCheckoutRejectsInactive() {
	codeID := s.createSellerDiscountCode(helpers.PercentageDiscountCode("REVALOFF", 10))
	s.deactivateDiscountCode(codeID)

	svc := promotionSingleton.GetInstance().GetCouponApplyService()
	err := svc.EnsureCouponsValidAtCheckout(
		context.Background(),
		helpers.DefaultCheckoutCouponCartRequest(
			uint(helpers.Seller2UserID),
			uint(helpers.CustomerUserID),
			codeID,
		),
	)
	s.Require().Error(err)
	appErr, ok := err.(*commonError.AppError)
	s.Require().True(ok)
	s.Equal(promoErrors.INVALID_COUPON_CODE, appErr.Code)
}

func (s *OrderSuite) TestEnsureCouponsValidAtCheckoutRejectsExpired() {
	codeID := s.createSellerDiscountCode(helpers.PercentageDiscountCode("REVALEXP", 10))
	past := time.Now().UTC().Add(-2 * time.Hour)
	s.Require().NoError(
		s.container.DB.Model(&promotionEntity.DiscountCode{}).
			Where("id = ?", codeID).
			Update("ends_at", past).Error,
	)

	svc := promotionSingleton.GetInstance().GetCouponApplyService()
	err := svc.EnsureCouponsValidAtCheckout(
		context.Background(),
		helpers.DefaultCheckoutCouponCartRequest(
			uint(helpers.Seller2UserID),
			uint(helpers.CustomerUserID),
			codeID,
		),
	)
	s.Require().Error(err)
	appErr, ok := err.(*commonError.AppError)
	s.Require().True(ok)
	s.Equal(promoErrors.COUPON_EXPIRED_CODE, appErr.Code)
}

func (s *OrderSuite) TestEnsureCouponsValidAtCheckoutAllowsActive() {
	codeID := s.createSellerDiscountCode(helpers.PercentageDiscountCode("REVALOK", 10))

	svc := promotionSingleton.GetInstance().GetCouponApplyService()
	err := svc.EnsureCouponsValidAtCheckout(
		context.Background(),
		helpers.DefaultCheckoutCouponCartRequest(
			uint(helpers.Seller2UserID),
			uint(helpers.CustomerUserID),
			codeID,
		),
	)
	s.Require().NoError(err)
}

func (s *OrderSuite) TestRevalidateCouponsForCheckoutRejectsWhenCodeDeactivatedWithoutCartGet() {
	codeID := s.createSellerDiscountCode(helpers.PercentageDiscountCode("REVALCART", 10))
	s.addItemToCart(1, 1)
	s.applyCoupon("REVALCART")
	s.deactivateDiscountCode(codeID)

	// DB still has cart_applied_coupon until a cart rebuild self-heals; revalidate must fail.
	cartSvc := orderSingleton.GetInstance().GetCartService()
	err := cartSvc.RevalidateCouponsForCheckout(
		context.Background(),
		uint(helpers.CustomerUserID),
		uint(helpers.Seller2UserID),
		s.currentCustomerCartID(),
		[]uint{codeID},
	)
	s.Require().Error(err)
	appErr, ok := err.(*commonError.AppError)
	s.Require().True(ok)
	s.Equal(promoErrors.INVALID_COUPON_CODE, appErr.Code)
}

func (s *OrderSuite) deactivateDiscountCode(id uint) {
	res := s.sellerClient.Patch(s.T(), fmt.Sprintf("/api/promotion/discount-code/%d/status", id), map[string]any{
		"isActive": false,
	})
	s.Require().Equal(http.StatusOK, res.Code, res.Body.String())
}

func (s *OrderSuite) currentCustomerCartID() uint {
	var cartID uint
	s.Require().NoError(
		s.container.DB.Raw(
			`SELECT id FROM cart WHERE user_id = ? AND status = 'active' ORDER BY id DESC LIMIT 1`,
			helpers.CustomerUserID,
		).Scan(&cartID).Error,
	)
	s.Require().NotZero(cartID)
	return cartID
}
