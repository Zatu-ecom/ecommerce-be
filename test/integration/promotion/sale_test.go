package promotion_test

import (
	"fmt"
	"net/http"
	"testing"

	promotionEntity "ecommerce-be/promotion/entity"
	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/suite"
)

type SaleTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler

	sellerClient      *helpers.APIClient
	otherSellerClient *helpers.APIClient
	customerClient    *helpers.APIClient
	anonymousClient   *helpers.APIClient
}

func (s *SaleTestSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())

	helpers.AttachMinIOStorage(s.T(), s.container, helpers.DefaultFileStorageEnvConfig())

	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)

	s.sellerClient = helpers.NewAPIClient(s.server)
	sellerToken := helpers.Login(
		s.T(),
		s.sellerClient,
		helpers.Seller2Email,
		helpers.Seller2Password,
	)
	s.sellerClient.SetToken(sellerToken)

	s.otherSellerClient = helpers.NewAPIClient(s.server)
	otherSellerToken := helpers.Login(
		s.T(),
		s.otherSellerClient,
		helpers.SellerEmail,
		helpers.SellerPassword,
	)
	s.otherSellerClient.SetToken(otherSellerToken)

	s.customerClient = helpers.NewAPIClient(s.server)
	customerToken := helpers.Login(
		s.T(),
		s.customerClient,
		helpers.CustomerEmail,
		helpers.CustomerPassword,
	)
	s.customerClient.SetToken(customerToken)

	s.anonymousClient = helpers.NewAPIClient(s.server)
}

func (s *SaleTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func (s *SaleTestSuite) SetupTest() {
	s.cleanupSalesAndPromotions()
}

func TestSaleAPI(t *testing.T) {
	suite.Run(t, new(SaleTestSuite))
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func saleURL(id uint) string {
	return fmt.Sprintf("%s/%d", SaleAPIEndpoint, id)
}

func saleStatusURL(id uint) string {
	return fmt.Sprintf("%s/%d/status", SaleAPIEndpoint, id)
}

func promotionURL(id uint) string {
	return fmt.Sprintf("%s/%d", PromotionAPIEndpoint, id)
}

func (s *SaleTestSuite) defaultSalePayload(name string) map[string]any {
	return map[string]any{
		"name":    name,
		"startAt": "2026-01-01T00:00:00Z",
		"endAt":   "2026-12-31T23:59:59Z",
		"status":  "draft",
	}
}

func (s *SaleTestSuite) createSale(client *helpers.APIClient, name string) uint {
	res := client.Post(s.T(), SaleAPIEndpoint, s.defaultSalePayload(name))
	s.Require().Equal(http.StatusCreated, res.Code, "sale creation should succeed")

	response := helpers.ParseResponse(s.T(), res.Body)
	sale := response["data"].(map[string]any)["sale"].(map[string]any)
	return uint(sale["id"].(float64))
}

func (s *SaleTestSuite) minimalPromotionPayload(name string, saleID *uint) map[string]any {
	payload := map[string]any{
		"name":          name,
		"promotionType": "percentage_discount",
		"discountConfig": map[string]any{
			"percentage": 10,
		},
		"appliesTo":   "all_products",
		"eligibleFor": "everyone",
		"startsAt":    "2026-01-01T00:00:00Z",
		"endsAt":      "2026-12-31T23:59:59Z",
		"status":      "draft",
	}
	if saleID != nil {
		payload["saleId"] = *saleID
	}
	return payload
}

func (s *SaleTestSuite) createPromotionFromPayload(
	client *helpers.APIClient,
	payload map[string]any,
) uint {
	res := client.Post(s.T(), PromotionAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code, "promotion creation should succeed")

	response := helpers.ParseResponse(s.T(), res.Body)
	promotion := response["data"].(map[string]any)["promotion"].(map[string]any)
	return uint(promotion["id"].(float64))
}

func (s *SaleTestSuite) listSaleIDs(client *helpers.APIClient) []uint {
	res := client.Get(s.T(), SaleAPIEndpoint)
	s.Require().Equal(http.StatusOK, res.Code, "list sales should succeed")

	response := helpers.ParseResponse(s.T(), res.Body)
	sales := response["data"].(map[string]any)["sales"].([]any)

	ids := make([]uint, 0, len(sales))
	for _, item := range sales {
		sale := item.(map[string]any)
		ids = append(ids, uint(sale["id"].(float64)))
	}
	return ids
}

func (s *SaleTestSuite) cleanupSalesAndPromotions() {
	sellerIDs := []uint{helpers.SellerUserID, helpers.Seller2UserID, helpers.Seller4UserID}

	var promoIDs []uint
	err := s.container.DB.
		Model(&promotionEntity.Promotion{}).
		Where("seller_id IN ?", sellerIDs).
		Pluck("id", &promoIDs).Error
	s.Require().NoError(err)

	if len(promoIDs) > 0 {
		s.Require().NoError(
			s.container.DB.Where("promotion_id IN ?", promoIDs).
				Delete(&promotionEntity.PromotionUsage{}).Error,
		)
		s.Require().NoError(
			s.container.DB.Where("promotion_id IN ?", promoIDs).
				Delete(&promotionEntity.PromotionProductVariant{}).Error,
		)
		s.Require().NoError(
			s.container.DB.Where("promotion_id IN ?", promoIDs).
				Delete(&promotionEntity.PromotionProduct{}).Error,
		)
		s.Require().NoError(
			s.container.DB.Where("promotion_id IN ?", promoIDs).
				Delete(&promotionEntity.PromotionCategory{}).Error,
		)
		s.Require().NoError(
			s.container.DB.Where("promotion_id IN ?", promoIDs).
				Delete(&promotionEntity.PromotionCollection{}).Error,
		)
		s.Require().NoError(
			s.container.DB.Unscoped().
				Where("id IN ?", promoIDs).
				Delete(&promotionEntity.Promotion{}).Error,
		)
	}

	s.Require().NoError(
		s.container.DB.Unscoped().
			Where("seller_id IN ?", sellerIDs).
			Delete(&promotionEntity.Sale{}).Error,
	)
}
