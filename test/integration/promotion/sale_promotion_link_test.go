package promotion_test

import (
	"net/http"

	"ecommerce-be/test/integration/helpers"
)

// ---------------------------------------------------------------------------
// Promotion ↔ sale linking
// ---------------------------------------------------------------------------

func (s *SaleTestSuite) TestCreatePromotionWithoutSaleIDRegression() {
	payload := s.minimalPromotionPayload("No Sale Link Promotion", nil)
	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	promotion := response["data"].(map[string]any)["promotion"].(map[string]any)
	_, hasSaleID := promotion["saleId"]
	s.False(hasSaleID, "saleId should be omitted when not linked")
}

func (s *SaleTestSuite) TestCreatePromotionWithValidSaleID() {
	saleID := s.createSale(s.sellerClient, "Promotion Link Sale")
	payload := s.minimalPromotionPayload("Sale-linked Promotion", &saleID)

	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	promotion := response["data"].(map[string]any)["promotion"].(map[string]any)
	s.Equal(float64(saleID), promotion["saleId"])
}

func (s *SaleTestSuite) TestCreatePromotionWithNonExistentSaleID() {
	missingID := uint(999999)
	payload := s.minimalPromotionPayload("Missing Sale Promotion", &missingID)

	res := s.sellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

func (s *SaleTestSuite) TestCreatePromotionWithCrossSellerSaleID() {
	saleID := s.createSale(s.sellerClient, "Seller Two Sale For Promo")
	payload := s.minimalPromotionPayload("Cross Seller Promotion", &saleID)

	res := s.otherSellerClient.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

func (s *SaleTestSuite) TestUpdatePromotionAttachValidSaleID() {
	saleID := s.createSale(s.sellerClient, "Update Link Sale")
	promotionID := s.createPromotionFromPayload(
		s.sellerClient,
		s.minimalPromotionPayload("Promotion To Link Later", nil),
	)

	res := s.sellerClient.Put(
		s.T(),
		promotionURL(promotionID),
		map[string]any{"saleId": saleID},
	)
	s.Require().Equal(http.StatusOK, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	promotion := response["data"].(map[string]any)["promotion"].(map[string]any)
	s.Equal(float64(saleID), promotion["saleId"])
}

func (s *SaleTestSuite) TestUpdatePromotionWithCrossSellerSaleID() {
	saleID := s.createSale(s.sellerClient, "Other Seller Sale")
	promotionID := s.createPromotionFromPayload(
		s.otherSellerClient,
		s.minimalPromotionPayload("Seller Three Promotion", nil),
	)

	res := s.otherSellerClient.Put(
		s.T(),
		promotionURL(promotionID),
		map[string]any{"saleId": saleID},
	)
	s.Require().Equal(http.StatusBadRequest, res.Code)
}

func (s *SaleTestSuite) TestDeleteLinkedSaleNullifiesPromotionSaleID() {
	saleID := s.createSale(s.sellerClient, "Cascade Null Sale")
	promotionID := s.createPromotionFromPayload(
		s.sellerClient,
		s.minimalPromotionPayload("Linked Before Delete", &saleID),
	)

	res := s.sellerClient.Delete(s.T(), saleURL(saleID))
	s.Require().Equal(http.StatusOK, res.Code)

	res = s.sellerClient.Get(s.T(), promotionURL(promotionID))
	s.Require().Equal(http.StatusOK, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	promotion := response["data"].(map[string]any)["promotion"].(map[string]any)
	_, hasSaleID := promotion["saleId"]
	s.False(hasSaleID, "saleId should be cleared after sale delete")
}
