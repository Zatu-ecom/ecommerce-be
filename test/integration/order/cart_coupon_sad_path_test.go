package order_test

import (
	"net/http"

	promotionEntity "ecommerce-be/promotion/entity"
	"ecommerce-be/test/integration/helpers"
)

func (s *CartCouponTestSuite) TestApplyInvalidCoupon() {
	s.addCartItem(1, 1)
	res := s.customerClient.Post(s.T(), CartCouponAPIEndpoint, helpers.ApplyCouponPayload("NOPE"))
	s.Require().Equal(http.StatusBadRequest, res.Code)
	response := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("INVALID_COUPON", response["code"])
}

func (s *CartCouponTestSuite) TestApplyExpiredCoupon() {
	s.createSellerDiscountCode(
		helpers.NewDiscountCodePayload("EXPIRED").
			Percentage(10).
			StartsAt(helpers.PastRFC3339(30)).
			EndsAt(helpers.PastRFC3339(1)).
			Build(),
	)
	s.addCartItem(1, 1)

	res := s.customerClient.Post(s.T(), CartCouponAPIEndpoint, helpers.ApplyCouponPayload("EXPIRED"))
	s.Require().Equal(http.StatusBadRequest, res.Code)
	response := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("COUPON_EXPIRED", response["code"])
}

func (s *CartCouponTestSuite) TestApplyNotStartedCoupon() {
	s.createSellerDiscountCode(
		helpers.NewDiscountCodePayload("FUTURE").
			Percentage(10).
			StartsAt(helpers.FutureRFC3339(2)).
			EndsAt(helpers.FutureRFC3339(30)).
			Build(),
	)
	s.addCartItem(1, 1)

	res := s.customerClient.Post(s.T(), CartCouponAPIEndpoint, helpers.ApplyCouponPayload("FUTURE"))
	s.Require().Equal(http.StatusBadRequest, res.Code)
	response := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("COUPON_NOT_STARTED", response["code"])
}

func (s *CartCouponTestSuite) TestApplyAlreadyAppliedCoupon() {
	s.createSellerDiscountCode(helpers.PercentageDiscountCode("DUPAPP", 10))
	s.addCartItem(1, 1)
	s.applyCouponOK("DUPAPP")

	res := s.customerClient.Post(s.T(), CartCouponAPIEndpoint, helpers.ApplyCouponPayload("DUPAPP"))
	s.Require().Equal(http.StatusBadRequest, res.Code)
	response := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("COUPON_ALREADY_APPLIED", response["code"])
}

func (s *CartCouponTestSuite) TestApplyMinPurchaseNotMet() {
	s.createSellerDiscountCode(
		helpers.NewDiscountCodePayload("MINPUR").
			Percentage(10).
			MinPurchaseCents(500000).
			Build(),
	)
	s.addCartItem(1, 1) // 99900 < 500000

	res := s.customerClient.Post(s.T(), CartCouponAPIEndpoint, helpers.ApplyCouponPayload("MINPUR"))
	s.Require().Equal(http.StatusBadRequest, res.Code)
	response := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("COUPON_MIN_PURCHASE_NOT_MET", response["code"])
}

func (s *CartCouponTestSuite) TestApplyMinQuantityNotMet() {
	s.createSellerDiscountCode(
		helpers.NewDiscountCodePayload("MINQTY").
			Percentage(10).
			MinQuantity(5).
			Build(),
	)
	s.addCartItem(1, 1)

	res := s.customerClient.Post(s.T(), CartCouponAPIEndpoint, helpers.ApplyCouponPayload("MINQTY"))
	s.Require().Equal(http.StatusBadRequest, res.Code)
	response := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("COUPON_MIN_QUANTITY_NOT_MET", response["code"])
}

func (s *CartCouponTestSuite) TestApplySpecificSegmentNotEligible() {
	segment := promotionEntity.CustomerSegment{
		SellerID: helpers.Seller2UserID,
		Name:     "VIP Stub Segment",
		Rules:    promotionEntity.SegmentRules{"operator": "AND", "conditions": []any{}},
	}
	s.Require().NoError(s.container.DB.Create(&segment).Error)

	s.createSellerDiscountCode(
		helpers.NewDiscountCodePayload("SEGONLY").
			Percentage(10).
			CustomerEligibility("specific_segment").
			CustomerSegmentID(segment.ID).
			Build(),
	)
	s.addCartItem(1, 1)

	res := s.customerClient.Post(s.T(), CartCouponAPIEndpoint, helpers.ApplyCouponPayload("SEGONLY"))
	s.Require().Equal(http.StatusBadRequest, res.Code)
	response := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("COUPON_NOT_ELIGIBLE", response["code"])
}

func (s *CartCouponTestSuite) TestApplyCannotCombine() {
	s.createSellerDiscountCode(
		helpers.NewDiscountCodePayload("STACKA").
			Percentage(5).
			CanCombineWithOtherDiscounts(false).
			Build(),
	)
	s.createSellerDiscountCode(
		helpers.NewDiscountCodePayload("STACKB").
			Percentage(5).
			CanCombineWithOtherDiscounts(false).
			Build(),
	)

	s.addCartItem(1, 1)
	s.applyCouponOK("STACKA")

	res := s.customerClient.Post(s.T(), CartCouponAPIEndpoint, helpers.ApplyCouponPayload("STACKB"))
	s.Require().Equal(http.StatusBadRequest, res.Code)
	response := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("COUPON_CANNOT_COMBINE", response["code"])
}
