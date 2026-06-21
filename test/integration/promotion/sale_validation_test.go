package promotion_test

import (
	"net/http"

	"ecommerce-be/test/integration/helpers"
)

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

func (s *SaleTestSuite) TestCreateSaleNameTooShort() {
	res := s.sellerClient.Post(
		s.T(),
		SaleAPIEndpoint,
		s.defaultSalePayload("ab"),
	)
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

func (s *SaleTestSuite) TestCreateSaleMissingStartAt() {
	res := s.sellerClient.Post(s.T(), SaleAPIEndpoint, map[string]any{
		"name":  "Valid Name",
		"endAt": "2026-12-31T23:59:59Z",
	})
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

func (s *SaleTestSuite) TestCreateSaleMissingEndAt() {
	res := s.sellerClient.Post(s.T(), SaleAPIEndpoint, map[string]any{
		"name":    "Valid Name",
		"startAt": "2026-01-01T00:00:00Z",
	})
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

func (s *SaleTestSuite) TestCreateSaleInvalidDateFormat() {
	payload := s.defaultSalePayload("Invalid Date Sale")
	payload["startAt"] = "not-a-date"
	res := s.sellerClient.Post(s.T(), SaleAPIEndpoint, payload)
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

func (s *SaleTestSuite) TestCreateSaleEndAtBeforeStartAt() {
	payload := s.defaultSalePayload("Bad Range Sale")
	payload["startAt"] = "2026-12-31T23:59:59Z"
	payload["endAt"] = "2026-01-01T00:00:00Z"
	res := s.sellerClient.Post(s.T(), SaleAPIEndpoint, payload)
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

func (s *SaleTestSuite) TestCreateSaleDuplicateSlug() {
	slug := "duplicate-slug-test"
	first := s.defaultSalePayload("First Duplicate Sale")
	first["slug"] = slug
	res := s.sellerClient.Post(s.T(), SaleAPIEndpoint, first)
	s.Require().Equal(http.StatusCreated, res.Code)

	second := s.defaultSalePayload("Second Duplicate Sale")
	second["slug"] = slug
	res = s.sellerClient.Post(s.T(), SaleAPIEndpoint, second)
	s.Require().Equal(http.StatusConflict, res.Code)
}

func (s *SaleTestSuite) TestCreateSaleInvalidBannerFile() {
	payload := s.defaultSalePayload("Invalid Banner Sale")
	payload["bannerFileIds"] = []string{"non-existent-file-id"}
	res := s.sellerClient.Post(s.T(), SaleAPIEndpoint, payload)
	s.Require().Equal(http.StatusUnprocessableEntity, res.Code)
}

func (s *SaleTestSuite) TestGetSaleInvalidIDPath() {
	res := s.sellerClient.Get(s.T(), SaleAPIEndpoint+"/abc")
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

func (s *SaleTestSuite) TestGetSaleNotFound() {
	res := s.sellerClient.Get(s.T(), saleURL(999999))
	s.Require().Equal(http.StatusNotFound, res.Code)
}

// ---------------------------------------------------------------------------
// Status transitions
// ---------------------------------------------------------------------------

func (s *SaleTestSuite) TestSaleStatusDraftToActive() {
	saleID := s.createSale(s.sellerClient, "Draft To Active")

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

func (s *SaleTestSuite) TestSaleStatusDraftToEndedInvalid() {
	saleID := s.createSale(s.sellerClient, "Draft To Ended Invalid")

	res := s.sellerClient.Patch(
		s.T(),
		saleStatusURL(saleID),
		map[string]any{"status": "ended"},
	)
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

func (s *SaleTestSuite) TestSaleStatusEndedToActiveInvalid() {
	saleID := s.createSale(s.sellerClient, "Ended Terminal Sale")

	res := s.sellerClient.Patch(
		s.T(),
		saleStatusURL(saleID),
		map[string]any{"status": "active"},
	)
	s.Require().Equal(http.StatusOK, res.Code)

	res = s.sellerClient.Patch(
		s.T(),
		saleStatusURL(saleID),
		map[string]any{"status": "ended"},
	)
	s.Require().Equal(http.StatusOK, res.Code)

	res = s.sellerClient.Patch(
		s.T(),
		saleStatusURL(saleID),
		map[string]any{"status": "active"},
	)
	s.Require().Equal(http.StatusBadRequest, res.Code)
}
