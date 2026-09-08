package promotion_test

import (
	"net/http"

	"ecommerce-be/test/integration/helpers"
)

func (s *DiscountCodeTestSuite) TestAddAndListDiscountCodeProducts() {
	codeID := s.createScopedDiscountCode(s.sellerClient, "PRODSC1", "specific_products")
	productID := s.seedProductID(helpers.Seller2UserID)

	res := s.sellerClient.Post(s.T(), DiscountCodeScopeAPIEndpoint+"/product", map[string]any{
		"discountCodeId": codeID,
		"productIds":     []uint{productID},
	})
	s.Require().Equal(http.StatusOK, res.Code, res.Body.String())

	res = s.sellerClient.Get(s.T(), scopeProductURL(codeID))
	s.Require().Equal(http.StatusOK, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	data := response["data"].(map[string]any)
	products := extractScopeList(data, "products")
	s.Require().Len(products, 1)
	s.Equal(float64(productID), products[0].(map[string]any)["productId"])
}

func (s *DiscountCodeTestSuite) TestRemoveDiscountCodeProducts() {
	codeID := s.createScopedDiscountCode(s.sellerClient, "PRODSC2", "specific_products")
	productID := s.seedProductID(helpers.Seller2UserID)

	res := s.sellerClient.Post(s.T(), DiscountCodeScopeAPIEndpoint+"/product", map[string]any{
		"discountCodeId": codeID,
		"productIds":     []uint{productID},
	})
	s.Require().Equal(http.StatusOK, res.Code)

	res = s.sellerClient.Delete(s.T(), DiscountCodeScopeAPIEndpoint+"/product", map[string]any{
		"discountCodeId": codeID,
		"productIds":     []uint{productID},
	})
	s.Require().Equal(http.StatusOK, res.Code)

	res = s.sellerClient.Get(s.T(), scopeProductURL(codeID))
	s.Require().Equal(http.StatusOK, res.Code)
	response := helpers.ParseResponse(s.T(), res.Body)
	products := extractScopeList(response["data"].(map[string]any), "products")
	s.Empty(products)
}

func (s *DiscountCodeTestSuite) TestRemoveAllDiscountCodeProducts() {
	codeID := s.createScopedDiscountCode(s.sellerClient, "PRODSC3", "specific_products")
	productID := s.seedProductID(helpers.Seller2UserID)

	res := s.sellerClient.Post(s.T(), DiscountCodeScopeAPIEndpoint+"/product", map[string]any{
		"discountCodeId": codeID,
		"productIds":     []uint{productID},
	})
	s.Require().Equal(http.StatusOK, res.Code)

	res = s.sellerClient.Delete(s.T(), scopeProductURL(codeID), nil)
	s.Require().Equal(http.StatusOK, res.Code)

	res = s.sellerClient.Get(s.T(), scopeProductURL(codeID))
	s.Require().Equal(http.StatusOK, res.Code)
	response := helpers.ParseResponse(s.T(), res.Body)
	products := extractScopeList(response["data"].(map[string]any), "products")
	s.Empty(products)
}
