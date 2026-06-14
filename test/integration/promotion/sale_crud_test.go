package promotion_test

import (
	"net/http"

	"ecommerce-be/test/integration/helpers"
)

// ---------------------------------------------------------------------------
// CRUD
// ---------------------------------------------------------------------------

func (s *SaleTestSuite) TestCreateSale() {
	res := s.sellerClient.Post(
		s.T(),
		SaleAPIEndpoint,
		s.defaultSalePayload("Summer Sale 2026"),
	)
	s.Require().Equal(http.StatusCreated, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	sale := response["data"].(map[string]any)["sale"].(map[string]any)

	s.Equal("Summer Sale 2026", sale["name"])
	s.Equal("summer-sale-2026", sale["slug"])
	s.Equal(float64(helpers.Seller2UserID), sale["sellerId"])
	s.Equal("draft", sale["status"])
}

func (s *SaleTestSuite) TestListSales() {
	s.createSale(s.sellerClient, "List Test Sale")

	res := s.sellerClient.Get(s.T(), SaleAPIEndpoint)
	s.Require().Equal(http.StatusOK, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	sales := response["data"].(map[string]any)["sales"].([]any)
	s.NotEmpty(sales)
}

func (s *SaleTestSuite) TestGetSaleByID() {
	saleID := s.createSale(s.sellerClient, "Get By ID Sale")

	res := s.sellerClient.Get(s.T(), saleURL(saleID))
	s.Require().Equal(http.StatusOK, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	sale := response["data"].(map[string]any)["sale"].(map[string]any)

	s.Equal(float64(saleID), sale["id"])
	s.Equal("Get By ID Sale", sale["name"])
}

func (s *SaleTestSuite) TestUpdateSale() {
	saleID := s.createSale(s.sellerClient, "Update Me")

	body := map[string]any{
		"name":         "Updated Sale Name",
		"description":  "Updated description",
		"bannerImages": []string{"https://example.com/banner.jpg"},
		"startAt":      "2026-02-01T00:00:00Z",
		"endAt":        "2026-11-30T23:59:59Z",
		"status":       "scheduled",
	}
	res := s.sellerClient.Put(s.T(), saleURL(saleID), body)
	s.Require().Equal(http.StatusOK, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	sale := response["data"].(map[string]any)["sale"].(map[string]any)

	s.Equal("Updated Sale Name", sale["name"])
	s.Equal("Updated description", sale["description"])
	s.Equal("scheduled", sale["status"])
	s.NotEmpty(sale["updatedAt"])
}

func (s *SaleTestSuite) TestUpdateSaleStatus() {
	saleID := s.createSale(s.sellerClient, "Status Update Sale")

	res := s.sellerClient.Patch(
		s.T(),
		saleStatusURL(saleID),
		map[string]any{"status": "active"},
	)
	s.Require().Equal(http.StatusOK, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	sale := response["data"].(map[string]any)["sale"].(map[string]any)
	s.Equal("active", sale["status"])
}

func (s *SaleTestSuite) TestDeleteSale() {
	saleID := s.createSale(s.sellerClient, "Delete Me")

	res := s.sellerClient.Delete(s.T(), saleURL(saleID))
	s.Require().Equal(http.StatusOK, res.Code)

	res = s.sellerClient.Get(s.T(), saleURL(saleID))
	s.Require().Equal(http.StatusNotFound, res.Code)
}
