package promotion_test

import (
	"net/http"

	"ecommerce-be/test/integration/helpers"
)

func (s *DiscountCodeTestSuite) TestCreateNormalizesCodeToUppercase() {
	res := s.sellerClient.Post(
		s.T(),
		DiscountCodeAPIEndpoint,
		s.defaultDiscountCodePayload("  save20  "),
	)
	s.Require().Equal(http.StatusCreated, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	dc := response["data"].(map[string]any)["discountCode"].(map[string]any)
	s.Equal("SAVE20", dc["code"])
}

func (s *DiscountCodeTestSuite) TestDuplicateIgnoresCaseAndWhitespace() {
	s.createDiscountCode(s.sellerClient, "CASEDUP")

	res := s.sellerClient.Post(
		s.T(),
		DiscountCodeAPIEndpoint,
		s.defaultDiscountCodePayload("  casedup  "),
	)
	s.Require().Equal(http.StatusConflict, res.Code)
}

func (s *DiscountCodeTestSuite) TestCodeIsImmutableOnUpdate() {
	id := s.createDiscountCode(s.sellerClient, "IMMUTABLE")

	// code is not part of UpdateDiscountCodeRequest — body field must be ignored
	res := s.sellerClient.Put(s.T(), discountCodeURL(id), map[string]any{
		"code":  "CHANGED",
		"title": "Still Immutable",
	})
	s.Require().Equal(http.StatusOK, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	dc := response["data"].(map[string]any)["discountCode"].(map[string]any)
	s.Equal("IMMUTABLE", dc["code"])
	s.Equal("Still Immutable", dc["title"])
}

func (s *DiscountCodeTestSuite) TestCreateBuyXGetYRequiresMetadata() {
	payload := s.defaultDiscountCodePayload("BXGYBAD")
	payload["discountType"] = "buy_x_get_y"
	payload["value"] = 0
	// missing metadata

	res := s.sellerClient.Post(s.T(), DiscountCodeAPIEndpoint, payload)
	s.Require().Equal(http.StatusBadRequest, res.Code)
	response := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("INVALID_DISCOUNT_CODE_VALUE", response["code"])
}

func (s *DiscountCodeTestSuite) TestCreateBuyXGetYInvalidGetDiscountPercent() {
	payload := s.defaultDiscountCodePayload("BXGYBAD2")
	payload["discountType"] = "buy_x_get_y"
	payload["value"] = 0
	payload["metadata"] = map[string]any{
		"buyQuantity":        2,
		"getQuantity":        1,
		"getDiscountPercent": 150,
	}

	res := s.sellerClient.Post(s.T(), DiscountCodeAPIEndpoint, payload)
	s.Require().Equal(http.StatusBadRequest, res.Code)
	response := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("INVALID_DISCOUNT_CODE_VALUE", response["code"])
}

func (s *DiscountCodeTestSuite) TestSpecificSegmentRequiresCustomerSegmentID() {
	payload := s.defaultDiscountCodePayload("SEG1")
	payload["customerEligibility"] = "specific_segment"

	res := s.sellerClient.Post(s.T(), DiscountCodeAPIEndpoint, payload)
	s.Require().Equal(http.StatusBadRequest, res.Code)
}
