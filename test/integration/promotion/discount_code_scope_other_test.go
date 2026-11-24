package promotion_test

import (
	"net/http"

	"ecommerce-be/test/integration/helpers"
)

func (s *DiscountCodeTestSuite) TestAddAndListDiscountCodeVariants() {
	codeID := s.createScopedDiscountCode(s.sellerClient, "VARSC1", "specific_variant")
	variantID := s.seedVariantID(helpers.Seller2UserID)

	res := s.sellerClient.Post(s.T(), DiscountCodeScopeAPIEndpoint+"/variant", map[string]any{
		"discountCodeId": codeID,
		"variantIds":     []uint{variantID},
	})
	s.Require().Equal(http.StatusOK, res.Code, res.Body.String())

	res = s.sellerClient.Get(s.T(), scopeVariantURL(codeID))
	s.Require().Equal(http.StatusOK, res.Code)
	response := helpers.ParseResponse(s.T(), res.Body)
	variants := extractScopeList(response["data"].(map[string]any), "variants")
	s.Require().Len(variants, 1)
	s.Equal(float64(variantID), variants[0].(map[string]any)["variantId"])
}

func (s *DiscountCodeTestSuite) TestAddAndListDiscountCodeCategories() {
	codeID := s.createScopedDiscountCode(s.sellerClient, "CATSC1", "specific_categories")
	categoryID := s.seedCategoryID()

	res := s.sellerClient.Post(s.T(), DiscountCodeScopeAPIEndpoint+"/category", map[string]any{
		"discountCodeId": codeID,
		"categories": []map[string]any{
			{"categoryId": categoryID, "includeSubcategories": true},
		},
	})
	s.Require().Equal(http.StatusOK, res.Code, res.Body.String())

	res = s.sellerClient.Get(s.T(), scopeCategoryURL(codeID))
	s.Require().Equal(http.StatusOK, res.Code)
	response := helpers.ParseResponse(s.T(), res.Body)
	categories := extractScopeList(response["data"].(map[string]any), "categories")
	s.Require().Len(categories, 1)
	s.Equal(float64(categoryID), categories[0].(map[string]any)["categoryId"])
}

func (s *DiscountCodeTestSuite) TestAddAndListDiscountCodeCollections() {
	codeID := s.createScopedDiscountCode(s.sellerClient, "COLSC1", "specific_collections")
	collectionID := s.createCollection("DC Scope Collection")

	res := s.sellerClient.Post(s.T(), DiscountCodeScopeAPIEndpoint+"/collection", map[string]any{
		"discountCodeId": codeID,
		"collectionIds":  []uint{collectionID},
	})
	s.Require().Equal(http.StatusOK, res.Code, res.Body.String())

	res = s.sellerClient.Get(s.T(), scopeCollectionURL(codeID))
	s.Require().Equal(http.StatusOK, res.Code)
	response := helpers.ParseResponse(s.T(), res.Body)
	collections := extractScopeList(response["data"].(map[string]any), "collections")
	s.Require().Len(collections, 1)
	s.Equal(float64(collectionID), collections[0].(map[string]any)["collectionId"])
}

func (s *DiscountCodeTestSuite) TestRemoveDiscountCodeVariantsCategoriesCollections() {
	// Variant remove
	codeID := s.createScopedDiscountCode(s.sellerClient, "VARSC2", "specific_variant")
	variantID := s.seedVariantID(helpers.Seller2UserID)
	res := s.sellerClient.Post(s.T(), DiscountCodeScopeAPIEndpoint+"/variant", map[string]any{
		"discountCodeId": codeID,
		"variantIds":     []uint{variantID},
	})
	s.Require().Equal(http.StatusOK, res.Code)
	res = s.sellerClient.Delete(s.T(), DiscountCodeScopeAPIEndpoint+"/variant", map[string]any{
		"discountCodeId": codeID,
		"variantIds":     []uint{variantID},
	})
	s.Require().Equal(http.StatusOK, res.Code)

	// Category remove-all
	catCodeID := s.createScopedDiscountCode(s.sellerClient, "CATSC2", "specific_categories")
	categoryID := s.seedCategoryID()
	res = s.sellerClient.Post(s.T(), DiscountCodeScopeAPIEndpoint+"/category", map[string]any{
		"discountCodeId": catCodeID,
		"categories":     []map[string]any{{"categoryId": categoryID}},
	})
	s.Require().Equal(http.StatusOK, res.Code)
	res = s.sellerClient.Delete(s.T(), scopeCategoryURL(catCodeID), nil)
	s.Require().Equal(http.StatusOK, res.Code)

	// Collection remove
	colCodeID := s.createScopedDiscountCode(s.sellerClient, "COLSC2", "specific_collections")
	collectionID := s.createCollection("DC Scope Collection Remove")
	res = s.sellerClient.Post(s.T(), DiscountCodeScopeAPIEndpoint+"/collection", map[string]any{
		"discountCodeId": colCodeID,
		"collectionIds":  []uint{collectionID},
	})
	s.Require().Equal(http.StatusOK, res.Code)
	res = s.sellerClient.Delete(s.T(), DiscountCodeScopeAPIEndpoint+"/collection", map[string]any{
		"discountCodeId": colCodeID,
		"collectionIds":  []uint{collectionID},
	})
	s.Require().Equal(http.StatusOK, res.Code)
}
