package user

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type SellerSettingsTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler
	client    *helpers.APIClient
}

func (s *SellerSettingsTestSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())
	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)
	s.client = helpers.NewAPIClient(s.server)
}

func (s *SellerSettingsTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func TestSellerSettingsSuite(t *testing.T) {
	suite.Run(t, new(SellerSettingsTestSuite))
}

func (s *SellerSettingsTestSuite) seller2Token() string {
	return helpers.Login(s.T(), s.client, helpers.Seller2Email, helpers.Seller2Password)
}

func (s *SellerSettingsTestSuite) TestGetSettings_Success() {
	s.client.SetToken(s.seller2Token())

	w := s.client.Get(s.T(), "/api/user/seller/settings")
	assert.Equal(s.T(), http.StatusOK, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.True(s.T(), response["success"].(bool))

	data, ok := response["data"].(map[string]any)
	assert.True(s.T(), ok)

	settings, ok := data["settings"].(map[string]any)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), float64(2), settings["sellerId"])
	assert.Equal(s.T(), float64(21), settings["businessCountryId"])
	assert.Equal(s.T(), float64(4), settings["baseCurrencyId"])
}

func (s *SellerSettingsTestSuite) TestCreateSettings_Conflict() {
	s.client.SetToken(s.seller2Token())

	w := s.client.Post(s.T(), "/api/user/seller/settings", map[string]any{
		"businessCountryId": 21,
		"baseCurrencyId":    4,
	})
	assert.Equal(s.T(), http.StatusConflict, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.False(s.T(), response["success"].(bool))
}

func (s *SellerSettingsTestSuite) TestUpdateSettings_Success() {
	s.client.SetToken(s.seller2Token())

	displayInBuyerCurrency := true
	w := s.client.Put(s.T(), "/api/user/seller/settings", map[string]any{
		"displayPricesInBuyerCurrency": displayInBuyerCurrency,
	})
	assert.Equal(s.T(), http.StatusOK, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.True(s.T(), response["success"].(bool))

	data := response["data"].(map[string]any)
	settings := data["settings"].(map[string]any)
	assert.Equal(s.T(), displayInBuyerCurrency, settings["displayPricesInBuyerCurrency"])
}

func (s *SellerSettingsTestSuite) TestGetSettings_Unauthorized() {
	s.client.SetToken("")

	w := s.client.Get(s.T(), "/api/user/seller/settings")
	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}
