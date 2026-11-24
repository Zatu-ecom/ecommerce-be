package promotion_test

import (
	"net/http"

	"ecommerce-be/common/constants"
	"ecommerce-be/test/integration/helpers"
)

func (s *DiscountCodeTestSuite) TestUnauthenticatedCreateDiscountCode() {
	res := s.anonymousClient.Post(
		s.T(),
		DiscountCodeAPIEndpoint,
		s.defaultDiscountCodePayload("NOAUTH"),
	)
	s.Require().Equal(http.StatusUnauthorized, res.Code)
}

func (s *DiscountCodeTestSuite) TestUnauthenticatedListDiscountCodes() {
	res := s.anonymousClient.Get(s.T(), DiscountCodeAPIEndpoint)
	s.Require().Equal(http.StatusUnauthorized, res.Code)
}

func (s *DiscountCodeTestSuite) TestCustomerCannotCreateDiscountCode() {
	res := s.customerClient.Post(
		s.T(),
		DiscountCodeAPIEndpoint,
		s.defaultDiscountCodePayload("CUST"),
	)
	s.Require().Equal(http.StatusForbidden, res.Code)
}

func (s *DiscountCodeTestSuite) TestMissingCorrelationID() {
	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerClient.Token)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "")

	res := client.Post(
		s.T(),
		DiscountCodeAPIEndpoint,
		s.defaultDiscountCodePayload("NOCORR"),
	)
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

func (s *DiscountCodeTestSuite) TestCreateDuplicateDiscountCode() {
	s.createDiscountCode(s.sellerClient, "DUPCODE")

	res := s.sellerClient.Post(
		s.T(),
		DiscountCodeAPIEndpoint,
		s.defaultDiscountCodePayload("DUPCODE"),
	)
	s.Require().Equal(http.StatusConflict, res.Code)

	response := helpers.AssertErrorResponse(s.T(), res, http.StatusConflict)
	s.Equal("DISCOUNT_CODE_EXISTS", response["code"])
}

func (s *DiscountCodeTestSuite) TestCreateInvalidPercentageValue() {
	payload := s.defaultDiscountCodePayload("BADPCT")
	payload["value"] = 101

	res := s.sellerClient.Post(s.T(), DiscountCodeAPIEndpoint, payload)
	s.Require().Equal(http.StatusBadRequest, res.Code)

	response := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("INVALID_DISCOUNT_CODE_VALUE", response["code"])
}

func (s *DiscountCodeTestSuite) TestCreateInvalidFixedAmountValue() {
	payload := s.defaultDiscountCodePayload("BADFIX")
	payload["discountType"] = "fixed_amount"
	payload["value"] = 0

	res := s.sellerClient.Post(s.T(), DiscountCodeAPIEndpoint, payload)
	s.Require().Equal(http.StatusBadRequest, res.Code)
	response := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("INVALID_DISCOUNT_CODE_VALUE", response["code"])
}

func (s *DiscountCodeTestSuite) TestCreateFreeShippingNonZeroValue() {
	payload := s.defaultDiscountCodePayload("BADSHIP")
	payload["discountType"] = "free_shipping"
	payload["value"] = 100

	res := s.sellerClient.Post(s.T(), DiscountCodeAPIEndpoint, payload)
	s.Require().Equal(http.StatusBadRequest, res.Code)
	response := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("INVALID_DISCOUNT_CODE_VALUE", response["code"])
}

func (s *DiscountCodeTestSuite) TestCreateInvalidDateRange() {
	payload := s.defaultDiscountCodePayload("BADRANGE")
	payload["startsAt"] = "2026-12-31T23:59:59Z"
	payload["endsAt"] = "2026-01-01T00:00:00Z"

	res := s.sellerClient.Post(s.T(), DiscountCodeAPIEndpoint, payload)
	s.Require().Equal(http.StatusBadRequest, res.Code)
	response := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("INVALID_DISCOUNT_CODE_DATE_RANGE", response["code"])
}

func (s *DiscountCodeTestSuite) TestGetDiscountCodeNotFound() {
	res := s.sellerClient.Get(s.T(), discountCodeURL(999999))
	s.Require().Equal(http.StatusNotFound, res.Code)
}

func (s *DiscountCodeTestSuite) TestGetDiscountCodeInvalidID() {
	res := s.sellerClient.Get(s.T(), DiscountCodeAPIEndpoint+"/abc")
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

func (s *DiscountCodeTestSuite) TestCrossSellerCannotGetDiscountCode() {
	id := s.createDiscountCode(s.sellerClient, "XSELL")

	res := s.otherSellerClient.Get(s.T(), discountCodeURL(id))
	s.Require().Equal(http.StatusNotFound, res.Code)
}

func (s *DiscountCodeTestSuite) TestCrossSellerCannotUpdateDiscountCode() {
	id := s.createDiscountCode(s.sellerClient, "XSELLUPD")

	res := s.otherSellerClient.Put(s.T(), discountCodeURL(id), map[string]any{
		"title": "Hijacked",
	})
	s.Require().Equal(http.StatusNotFound, res.Code)
}

func (s *DiscountCodeTestSuite) TestCrossSellerCannotDeleteDiscountCode() {
	id := s.createDiscountCode(s.sellerClient, "XSELLDEL")

	res := s.otherSellerClient.Delete(s.T(), discountCodeURL(id))
	s.Require().Equal(http.StatusNotFound, res.Code)
}

func (s *DiscountCodeTestSuite) TestListDiscountCodesIsolatedPerSeller() {
	seller2ID := s.createDiscountCode(s.sellerClient, "ISOA")
	seller3ID := s.createDiscountCode(s.otherSellerClient, "ISOB")

	seller2IDs := s.listDiscountCodeIDs(s.sellerClient)
	seller3IDs := s.listDiscountCodeIDs(s.otherSellerClient)

	s.Contains(seller2IDs, seller2ID)
	s.NotContains(seller2IDs, seller3ID)
	s.Contains(seller3IDs, seller3ID)
	s.NotContains(seller3IDs, seller2ID)
}
