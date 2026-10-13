package order_test

import (
	"net/http"

	"ecommerce-be/common/constants"
	"ecommerce-be/test/integration/helpers"
)

func (s *CartCouponTestSuite) TestGuestCannotApplyCoupon() {
	s.createSellerDiscountCode(helpers.PercentageDiscountCode("GUESTDENY", 10))

	res := s.anonymousClient.Post(s.T(), CartCouponAPIEndpoint, helpers.ApplyCouponPayload("GUESTDENY"))
	s.Require().Equal(http.StatusUnauthorized, res.Code)
}

func (s *CartCouponTestSuite) TestUnauthenticatedAvailableCoupons() {
	res := s.anonymousClient.Get(s.T(), CartAvailableCouponAPIEndpoint)
	s.Require().Equal(http.StatusUnauthorized, res.Code)
}

func (s *CartCouponTestSuite) TestMissingCorrelationIDOnApply() {
	s.createSellerDiscountCode(helpers.PercentageDiscountCode("NOCORR", 10))
	s.addCartItem(1, 1)

	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.customerClient.Token)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "")

	res := client.Post(s.T(), CartCouponAPIEndpoint, helpers.ApplyCouponPayload("NOCORR"))
	s.Require().Equal(http.StatusBadRequest, res.Code)
}
