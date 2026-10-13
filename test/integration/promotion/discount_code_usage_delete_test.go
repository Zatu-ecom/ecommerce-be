package promotion_test

import (
	"net/http"

	"ecommerce-be/test/integration/helpers"
)

func (s *DiscountCodeTestSuite) TestCannotDeleteDiscountCodeAfterRedemption() {
	codeID := s.createDiscountCode(s.sellerClient, "USEDDEL")

	add := s.customerClient.Post(s.T(), "/api/order/cart/item", map[string]any{
		"items": []map[string]any{{"variantId": uint(1), "quantity": 1}},
	})
	s.Require().Equal(http.StatusCreated, add.Code, add.Body.String())

	apply := s.customerClient.Post(s.T(), "/api/order/cart/coupon", map[string]any{"code": "USEDDEL"})
	s.Require().Equal(http.StatusOK, apply.Code, apply.Body.String())

	create := s.customerClient.Post(s.T(), "/api/order", map[string]any{
		"shippingAddressId": 1,
		"billingAddressId":  1,
		"fulfillmentType":   "directship",
	})
	s.Require().Equal(http.StatusCreated, create.Code, create.Body.String())

	res := s.sellerClient.Delete(s.T(), discountCodeURL(codeID))
	s.Require().Equal(http.StatusConflict, res.Code, res.Body.String())
	response := helpers.AssertErrorResponse(s.T(), res, http.StatusConflict)
	s.Equal("DISCOUNT_CODE_HAS_USAGE", response["code"])
}
