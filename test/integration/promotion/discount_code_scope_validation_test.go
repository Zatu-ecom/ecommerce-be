package promotion_test

import (
	"net/http"

	"ecommerce-be/test/integration/helpers"
)

func (s *DiscountCodeTestSuite) TestAllProductsRejectsScopeAdd() {
	codeID := s.createDiscountCode(s.sellerClient, "ALLSC1") // appliesTo=all_products
	productID := s.seedProductID(helpers.Seller2UserID)

	res := s.sellerClient.Post(s.T(), DiscountCodeScopeAPIEndpoint+"/product", map[string]any{
		"discountCodeId": codeID,
		"productIds":     []uint{productID},
	})
	s.Require().Equal(http.StatusBadRequest, res.Code)
	response := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("INVALID_DISCOUNT_CODE_SCOPE", response["code"])
}

func (s *DiscountCodeTestSuite) TestAppliesToMismatchRejectsWrongResource() {
	codeID := s.createScopedDiscountCode(s.sellerClient, "MISMATCH1", "specific_products")
	categoryID := s.seedCategoryID()

	res := s.sellerClient.Post(s.T(), DiscountCodeScopeAPIEndpoint+"/category", map[string]any{
		"discountCodeId": codeID,
		"categories":     []map[string]any{{"categoryId": categoryID}},
	})
	s.Require().Equal(http.StatusBadRequest, res.Code)
	response := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("INVALID_DISCOUNT_CODE_SCOPE", response["code"])
}

func (s *DiscountCodeTestSuite) TestCrossSellerCannotAddDiscountCodeScope() {
	codeID := s.createScopedDiscountCode(s.sellerClient, "XSELLSC", "specific_products")
	productID := s.seedProductID(helpers.Seller2UserID)

	res := s.otherSellerClient.Post(s.T(), DiscountCodeScopeAPIEndpoint+"/product", map[string]any{
		"discountCodeId": codeID,
		"productIds":     []uint{productID},
	})
	s.Require().Equal(http.StatusNotFound, res.Code)
}

func (s *DiscountCodeTestSuite) TestCrossSellerCannotListDiscountCodeScope() {
	codeID := s.createScopedDiscountCode(s.sellerClient, "XSELLSC2", "specific_products")
	productID := s.seedProductID(helpers.Seller2UserID)

	res := s.sellerClient.Post(s.T(), DiscountCodeScopeAPIEndpoint+"/product", map[string]any{
		"discountCodeId": codeID,
		"productIds":     []uint{productID},
	})
	s.Require().Equal(http.StatusOK, res.Code)

	res = s.otherSellerClient.Get(s.T(), scopeProductURL(codeID))
	s.Require().Equal(http.StatusNotFound, res.Code)
}

func (s *DiscountCodeTestSuite) TestUnauthenticatedCannotAddDiscountCodeScope() {
	codeID := s.createScopedDiscountCode(s.sellerClient, "NOAUTHSC", "specific_products")
	productID := s.seedProductID(helpers.Seller2UserID)

	res := s.anonymousClient.Post(s.T(), DiscountCodeScopeAPIEndpoint+"/product", map[string]any{
		"discountCodeId": codeID,
		"productIds":     []uint{productID},
	})
	s.Require().Equal(http.StatusUnauthorized, res.Code)
}
