package user

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CurrencyTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler
	client    *helpers.APIClient
}

func (s *CurrencyTestSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())
	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)
	s.client = helpers.NewAPIClient(s.server)
}

func (s *CurrencyTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func TestCurrencySuite(t *testing.T) {
	suite.Run(t, new(CurrencyTestSuite))
}

func (s *CurrencyTestSuite) TestListActiveCurrencies_Public() {
	s.client.SetToken("")

	w := s.client.Get(s.T(), "/api/user/currency?page=1&limit=20")
	assert.Equal(s.T(), http.StatusOK, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.True(s.T(), response["success"].(bool))

	data, ok := response["data"].(map[string]any)
	assert.True(s.T(), ok)

	currencies, ok := data["currencies"].([]any)
	assert.True(s.T(), ok)
	assert.NotEmpty(s.T(), currencies)

	first, ok := currencies[0].(map[string]any)
	assert.True(s.T(), ok)
	assert.NotEmpty(s.T(), first["code"])
}

func (s *CurrencyTestSuite) TestGetCurrencyByID_Public() {
	s.client.SetToken("")

	w := s.client.Get(s.T(), "/api/user/currency/4")
	assert.Equal(s.T(), http.StatusOK, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.True(s.T(), response["success"].(bool))

	data, ok := response["data"].(map[string]any)
	assert.True(s.T(), ok)

	currency, ok := data["currency"].(map[string]any)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), float64(4), currency["id"])
	assert.Equal(s.T(), "INR", currency["code"])
}

func (s *CurrencyTestSuite) TestListAllCurrencies_Admin() {
	s.client.SetToken(helpers.Login(s.T(), s.client, helpers.AdminEmail, helpers.AdminPassword))

	w := s.client.Get(s.T(), "/api/user/admin/currency?page=1&limit=20")
	assert.Equal(s.T(), http.StatusOK, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.True(s.T(), response["success"].(bool))

	data, ok := response["data"].(map[string]any)
	assert.True(s.T(), ok)

	currencies, ok := data["currencies"].([]any)
	assert.True(s.T(), ok)
	assert.NotEmpty(s.T(), currencies)
}
