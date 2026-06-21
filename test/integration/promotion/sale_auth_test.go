package promotion_test

import (
	"net/http"
)

// ---------------------------------------------------------------------------
// Auth and ownership
// ---------------------------------------------------------------------------

func (s *SaleTestSuite) TestUnauthenticatedCreateSale() {
	res := s.anonymousClient.Post(
		s.T(),
		SaleAPIEndpoint,
		s.defaultSalePayload("No Auth Sale"),
	)
	s.Require().Equal(http.StatusUnauthorized, res.Code)
}

func (s *SaleTestSuite) TestUnauthenticatedListSales() {
	res := s.anonymousClient.Get(s.T(), SaleAPIEndpoint)
	s.Require().Equal(http.StatusUnauthorized, res.Code)
}

func (s *SaleTestSuite) TestUnauthenticatedGetSaleByID() {
	saleID := s.createSale(s.sellerClient, "Auth Target Sale")

	res := s.anonymousClient.Get(s.T(), saleURL(saleID))
	s.Require().Equal(http.StatusUnauthorized, res.Code)
}

func (s *SaleTestSuite) TestUnauthenticatedUpdateSale() {
	saleID := s.createSale(s.sellerClient, "Auth Target Sale")

	res := s.anonymousClient.Put(
		s.T(),
		saleURL(saleID),
		s.defaultSalePayload("Hijack"),
	)
	s.Require().Equal(http.StatusUnauthorized, res.Code)
}

func (s *SaleTestSuite) TestUnauthenticatedDeleteSale() {
	saleID := s.createSale(s.sellerClient, "Auth Target Sale")

	res := s.anonymousClient.Delete(s.T(), saleURL(saleID))
	s.Require().Equal(http.StatusUnauthorized, res.Code)
}

func (s *SaleTestSuite) TestUnauthenticatedPatchSaleStatus() {
	saleID := s.createSale(s.sellerClient, "Auth Target Sale")

	res := s.anonymousClient.Patch(
		s.T(),
		saleStatusURL(saleID),
		map[string]any{"status": "active"},
	)
	s.Require().Equal(http.StatusUnauthorized, res.Code)
}

func (s *SaleTestSuite) TestCustomerCannotCreateSale() {
	res := s.customerClient.Post(
		s.T(),
		SaleAPIEndpoint,
		s.defaultSalePayload("Customer Sale"),
	)
	s.Require().Equal(http.StatusForbidden, res.Code)
}

func (s *SaleTestSuite) TestCrossSellerCannotGetSale() {
	saleID := s.createSale(s.sellerClient, "Auth Target Sale")

	res := s.otherSellerClient.Get(s.T(), saleURL(saleID))
	s.Require().Equal(http.StatusForbidden, res.Code)
}

func (s *SaleTestSuite) TestCrossSellerCannotUpdateSale() {
	saleID := s.createSale(s.sellerClient, "Auth Target Sale")

	res := s.otherSellerClient.Put(
		s.T(),
		saleURL(saleID),
		s.defaultSalePayload("Hijacked"),
	)
	s.Require().Equal(http.StatusForbidden, res.Code)
}

func (s *SaleTestSuite) TestCrossSellerCannotPatchSaleStatus() {
	saleID := s.createSale(s.sellerClient, "Auth Target Sale")

	res := s.otherSellerClient.Patch(
		s.T(),
		saleStatusURL(saleID),
		map[string]any{"status": "active"},
	)
	s.Require().Equal(http.StatusForbidden, res.Code)
}

func (s *SaleTestSuite) TestCrossSellerCannotDeleteSale() {
	saleID := s.createSale(s.sellerClient, "Auth Target Sale")

	res := s.otherSellerClient.Delete(s.T(), saleURL(saleID))
	s.Require().Equal(http.StatusForbidden, res.Code)
}

func (s *SaleTestSuite) TestListSalesIsolatedPerSeller() {
	seller2SaleID := s.createSale(s.sellerClient, "Seller Two Isolated Sale")
	seller3SaleID := s.createSale(s.otherSellerClient, "Seller Three Isolated Sale")

	seller2IDs := s.listSaleIDs(s.sellerClient)
	seller3IDs := s.listSaleIDs(s.otherSellerClient)

	s.Contains(seller2IDs, seller2SaleID)
	s.NotContains(seller2IDs, seller3SaleID)
	s.Contains(seller3IDs, seller3SaleID)
	s.NotContains(seller3IDs, seller2SaleID)
}
