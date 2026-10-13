package promotion_test

import (
	"net/http"

	"ecommerce-be/test/integration/helpers"
)

func (s *DiscountCodeTestSuite) TestCreateDiscountCode() {
	res := s.sellerClient.Post(
		s.T(),
		DiscountCodeAPIEndpoint,
		s.defaultDiscountCodePayload("SAVE15"),
	)
	s.Require().Equal(http.StatusCreated, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	dc := response["data"].(map[string]any)["discountCode"].(map[string]any)

	s.Equal("SAVE15", dc["code"])
	s.Equal("Test Discount", dc["title"])
	s.Equal("percentage", dc["discountType"])
	s.Equal(float64(15), dc["value"])
	s.Equal("all_products", dc["appliesTo"])
	s.Equal(float64(helpers.Seller2UserID), dc["sellerId"])
	s.Equal(true, dc["isActive"])
	s.Equal(float64(0), dc["currentUsageCount"])
	s.NotEmpty(dc["createdAt"])
	s.NotEmpty(dc["updatedAt"])
}

func (s *DiscountCodeTestSuite) TestCreateFixedAmountDiscountCode() {
	payload := s.defaultDiscountCodePayload("FLAT100")
	payload["discountType"] = "fixed_amount"
	payload["value"] = 100.00 // major units → 10000 cents (INR)

	res := s.sellerClient.Post(s.T(), DiscountCodeAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	dc := response["data"].(map[string]any)["discountCode"].(map[string]any)
	s.Equal("fixed_amount", dc["discountType"])
	helpers.AssertMoney(s.T(), dc["value"])
	helpers.AssertMoneyCents(s.T(), dc["value"], 10000)
	helpers.AssertMoneyAmount(s.T(), dc["value"], 100.00)
}

func (s *DiscountCodeTestSuite) TestCreateFreeShippingDiscountCode() {
	payload := s.defaultDiscountCodePayload("FREESHIP")
	payload["discountType"] = "free_shipping"
	payload["value"] = 0

	res := s.sellerClient.Post(s.T(), DiscountCodeAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	dc := response["data"].(map[string]any)["discountCode"].(map[string]any)
	s.Equal("free_shipping", dc["discountType"])
	helpers.AssertMoney(s.T(), dc["value"])
	helpers.AssertMoneyCents(s.T(), dc["value"], 0)
}

func (s *DiscountCodeTestSuite) TestCreateBuyXGetYDiscountCode() {
	payload := s.defaultDiscountCodePayload("BXGY")
	payload["discountType"] = "buy_x_get_y"
	payload["value"] = 0
	payload["metadata"] = map[string]any{
		"buyQuantity":        2,
		"getQuantity":        1,
		"getDiscountPercent": 100,
	}

	res := s.sellerClient.Post(s.T(), DiscountCodeAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	dc := response["data"].(map[string]any)["discountCode"].(map[string]any)
	s.Equal("buy_x_get_y", dc["discountType"])
	meta := dc["metadata"].(map[string]any)
	s.Equal(float64(2), meta["buyQuantity"])
	s.Equal(float64(1), meta["getQuantity"])
	s.Equal(float64(100), meta["getDiscountPercent"])
}

func (s *DiscountCodeTestSuite) TestListDiscountCodes() {
	s.createDiscountCode(s.sellerClient, "LIST1")
	s.createDiscountCode(s.sellerClient, "LIST2")

	res := s.sellerClient.Get(s.T(), DiscountCodeAPIEndpoint)
	s.Require().Equal(http.StatusOK, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	data := response["data"].(map[string]any)
	codes := data["discountCodes"].([]any)
	s.Len(codes, 2)
	s.NotNil(data["pagination"])
}

func (s *DiscountCodeTestSuite) TestListDiscountCodesFilterIsActive() {
	id := s.createDiscountCode(s.sellerClient, "ACTIVE1")
	res := s.sellerClient.Patch(
		s.T(),
		discountCodeStatusURL(id),
		map[string]any{"isActive": false},
	)
	s.Require().Equal(http.StatusOK, res.Code)

	s.createDiscountCode(s.sellerClient, "ACTIVE2")

	res = s.sellerClient.Get(s.T(), DiscountCodeAPIEndpoint+"?isActive=true")
	s.Require().Equal(http.StatusOK, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	codes := response["data"].(map[string]any)["discountCodes"].([]any)
	s.Len(codes, 1)
	s.Equal("ACTIVE2", codes[0].(map[string]any)["code"])
}

func (s *DiscountCodeTestSuite) TestGetDiscountCodeByID() {
	id := s.createDiscountCode(s.sellerClient, "GETME")

	res := s.sellerClient.Get(s.T(), discountCodeURL(id))
	s.Require().Equal(http.StatusOK, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	dc := response["data"].(map[string]any)["discountCode"].(map[string]any)
	s.Equal(float64(id), dc["id"])
	s.Equal("GETME", dc["code"])
}

func (s *DiscountCodeTestSuite) TestUpdateDiscountCode() {
	id := s.createDiscountCode(s.sellerClient, "UPDME")

	res := s.sellerClient.Put(s.T(), discountCodeURL(id), map[string]any{
		"title": "Updated Title",
		"value": 25,
	})
	s.Require().Equal(http.StatusOK, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	dc := response["data"].(map[string]any)["discountCode"].(map[string]any)
	s.Equal("Updated Title", dc["title"])
	s.Equal(float64(25), dc["value"])
	s.Equal("UPDME", dc["code"])
	s.NotEmpty(dc["updatedAt"])
}

func (s *DiscountCodeTestSuite) TestUpdateDiscountCodeStatus() {
	id := s.createDiscountCode(s.sellerClient, "STATUS1")

	res := s.sellerClient.Patch(
		s.T(),
		discountCodeStatusURL(id),
		map[string]any{"isActive": false},
	)
	s.Require().Equal(http.StatusOK, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	dc := response["data"].(map[string]any)["discountCode"].(map[string]any)
	s.Equal(false, dc["isActive"])

	res = s.sellerClient.Patch(
		s.T(),
		discountCodeStatusURL(id),
		map[string]any{"isActive": true},
	)
	s.Require().Equal(http.StatusOK, res.Code)
	response = helpers.ParseResponse(s.T(), res.Body)
	dc = response["data"].(map[string]any)["discountCode"].(map[string]any)
	s.Equal(true, dc["isActive"])
}

func (s *DiscountCodeTestSuite) TestDeleteUnusedDiscountCode() {
	id := s.createDiscountCode(s.sellerClient, "DELME")

	res := s.sellerClient.Delete(s.T(), discountCodeURL(id))
	s.Require().Equal(http.StatusOK, res.Code)

	res = s.sellerClient.Get(s.T(), discountCodeURL(id))
	s.Require().Equal(http.StatusNotFound, res.Code)
}
