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

type DiscountCodeTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler

	sellerClient      *helpers.APIClient
	otherSellerClient *helpers.APIClient
	customerClient    *helpers.APIClient
	anonymousClient   *helpers.APIClient
}

func (s *DiscountCodeTestSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())

	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)

	s.sellerClient = helpers.NewAPIClient(s.server)
	s.sellerClient.SetToken(helpers.Login(
		s.T(),
		s.sellerClient,
		helpers.Seller2Email,
		helpers.Seller2Password,
	))

	s.otherSellerClient = helpers.NewAPIClient(s.server)
	s.otherSellerClient.SetToken(helpers.Login(
		s.T(),
		s.otherSellerClient,
		helpers.SellerEmail,
		helpers.SellerPassword,
	))

	s.customerClient = helpers.NewAPIClient(s.server)
	s.customerClient.SetToken(helpers.Login(
		s.T(),
		s.customerClient,
		helpers.CustomerEmail,
		helpers.CustomerPassword,
	))

	s.anonymousClient = helpers.NewAPIClient(s.server)
}

func (s *DiscountCodeTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func (s *DiscountCodeTestSuite) SetupTest() {
	s.cleanupDiscountCodes()
}

func TestDiscountCodeAPI(t *testing.T) {
	suite.Run(t, new(DiscountCodeTestSuite))
}

func discountCodeURL(id uint) string {
	return fmt.Sprintf("%s/%d", DiscountCodeAPIEndpoint, id)
}

func discountCodeStatusURL(id uint) string {
	return fmt.Sprintf("%s/%d/status", DiscountCodeAPIEndpoint, id)
}

func (s *DiscountCodeTestSuite) defaultDiscountCodePayload(code string) map[string]any {
	return helpers.NewDiscountCodePayload(code).
		Title("Test Discount").
		Percentage(15).
		Build()
}

func (s *DiscountCodeTestSuite) createDiscountCode(client *helpers.APIClient, code string) uint {
	res := client.Post(s.T(), DiscountCodeAPIEndpoint, s.defaultDiscountCodePayload(code))
	s.Require().Equal(http.StatusCreated, res.Code, "discount code creation should succeed: %s", res.Body.String())

	response := helpers.ParseResponse(s.T(), res.Body)
	dc := response["data"].(map[string]any)["discountCode"].(map[string]any)
	return uint(dc["id"].(float64))
}

func (s *DiscountCodeTestSuite) listDiscountCodeIDs(client *helpers.APIClient) []uint {
	res := client.Get(s.T(), DiscountCodeAPIEndpoint)
	s.Require().Equal(http.StatusOK, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	codes := response["data"].(map[string]any)["discountCodes"].([]any)

	ids := make([]uint, 0, len(codes))
	for _, item := range codes {
		dc := item.(map[string]any)
		ids = append(ids, uint(dc["id"].(float64)))
	}
	return ids
}

func (s *DiscountCodeTestSuite) cleanupDiscountCodes() {
	sellerIDs := []uint{helpers.SellerUserID, helpers.Seller2UserID, helpers.Seller4UserID}

	var codeIDs []uint
	err := s.container.DB.
		Model(&promotionEntity.DiscountCode{}).
		Where("seller_id IN ?", sellerIDs).
		Pluck("id", &codeIDs).Error
	s.Require().NoError(err)

	if len(codeIDs) > 0 {
		s.Require().NoError(
			s.container.DB.Where("discount_code_id IN ?", codeIDs).
				Delete(&promotionEntity.DiscountCodeUsage{}).Error,
		)
		s.Require().NoError(
			s.container.DB.Where("discount_code_id IN ?", codeIDs).
				Delete(&promotionEntity.DiscountCodeProduct{}).Error,
		)
		s.Require().NoError(
			s.container.DB.Where("discount_code_id IN ?", codeIDs).
				Delete(&promotionEntity.DiscountCodeCategory{}).Error,
		)
		s.Require().NoError(
			s.container.DB.Where("discount_code_id IN ?", codeIDs).
				Delete(&promotionEntity.DiscountCodeCollection{}).Error,
		)
		s.Require().NoError(
			s.container.DB.Unscoped().
				Where("id IN ?", codeIDs).
				Delete(&promotionEntity.DiscountCode{}).Error,
		)
	}
}

func (s *DiscountCodeTestSuite) scopedDiscountCodePayload(code, appliesTo string) map[string]any {
	payload := s.defaultDiscountCodePayload(code)
	payload["appliesTo"] = appliesTo
	return payload
}

func (s *DiscountCodeTestSuite) createScopedDiscountCode(
	client *helpers.APIClient,
	code, appliesTo string,
) uint {
	res := client.Post(s.T(), DiscountCodeAPIEndpoint, s.scopedDiscountCodePayload(code, appliesTo))
	s.Require().Equal(http.StatusCreated, res.Code, "scoped discount code creation should succeed: %s", res.Body.String())
	response := helpers.ParseResponse(s.T(), res.Body)
	dc := response["data"].(map[string]any)["discountCode"].(map[string]any)
	return uint(dc["id"].(float64))
}

func scopeProductURL(discountCodeID uint) string {
	return fmt.Sprintf("%s/%d/product", DiscountCodeScopeAPIEndpoint, discountCodeID)
}

func scopeVariantURL(discountCodeID uint) string {
	return fmt.Sprintf("%s/%d/variant", DiscountCodeScopeAPIEndpoint, discountCodeID)
}

func scopeCategoryURL(discountCodeID uint) string {
	return fmt.Sprintf("%s/%d/category", DiscountCodeScopeAPIEndpoint, discountCodeID)
}

func scopeCollectionURL(discountCodeID uint) string {
	return fmt.Sprintf("%s/%d/collection", DiscountCodeScopeAPIEndpoint, discountCodeID)
}

func (s *DiscountCodeTestSuite) seedProductID(sellerID uint) uint {
	var id uint
	err := s.container.DB.Table("product").
		Where("seller_id = ?", sellerID).
		Order("id ASC").
		Limit(1).
		Pluck("id", &id).Error
	s.Require().NoError(err)
	s.Require().NotZero(id)
	return id
}

func (s *DiscountCodeTestSuite) seedVariantID(sellerID uint) uint {
	var id uint
	err := s.container.DB.Raw(`
		SELECT pv.id FROM product_variant pv
		INNER JOIN product p ON p.id = pv.product_id
		WHERE p.seller_id = ?
		ORDER BY pv.id ASC
		LIMIT 1
	`, sellerID).Scan(&id).Error
	s.Require().NoError(err)
	s.Require().NotZero(id)
	return id
}

func (s *DiscountCodeTestSuite) seedCategoryID() uint {
	var id uint
	err := s.container.DB.Table("category").Order("id ASC").Limit(1).Pluck("id", &id).Error
	s.Require().NoError(err)
	s.Require().NotZero(id)
	return id
}

func (s *DiscountCodeTestSuite) createCollection(name string) uint {
	res := s.sellerClient.Post(s.T(), "/api/product/collection", map[string]any{
		"name":        name,
		"description": name + " description",
	})
	s.Require().Equal(http.StatusCreated, res.Code, "collection create should succeed: %s", res.Body.String())
	response := helpers.ParseResponse(s.T(), res.Body)
	col := response["data"].(map[string]any)["collection"].(map[string]any)
	return uint(col["id"].(float64))
}

func extractScopeList(data map[string]any, field string) []any {
	raw, ok := data[field]
	if !ok {
		return nil
	}
	switch payload := raw.(type) {
	case map[string]any:
		list, _ := payload[field].([]any)
		return list
	case []any:
		return payload
	default:
		return nil
	}
}
