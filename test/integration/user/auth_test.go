package user

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AuthTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler
	client    *helpers.APIClient
}

func (s *AuthTestSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())

	// Run migrations and seeds
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())

	// Setup test server
	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)

	// Create API client
	s.client = helpers.NewAPIClient(s.server)
}

func (s *AuthTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func TestAuthSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}

// TestScenario1_SuccessfulLogin
func (s *AuthTestSuite) TestSuccessfulLogin() {
	requestBody := map[string]any{
		"email":    "jane.merchant@example.com",
		"password": "seller123",
	}

	w := s.client.Post(s.T(), "/api/user/auth/login", requestBody)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)

	assert.True(s.T(), response["success"].(bool))
	assert.Equal(s.T(), "Login successful", response["message"])

	data, ok := response["data"].(map[string]any)
	assert.True(s.T(), ok)

	token, tokenOk := data["token"].(string)
	assert.True(s.T(), tokenOk)
	assert.NotEmpty(s.T(), token)

	sellerProfile, sellerProfileOk := data["sellerProfile"].(map[string]any)
	assert.True(s.T(), sellerProfileOk)

	profile, profileOk := sellerProfile["profile"].(map[string]any)
	assert.True(s.T(), profileOk)
	assert.Equal(s.T(), "Fashion Forward", profile["businessName"])

	addresses, addressesOk := sellerProfile["addresses"].([]any)
	assert.True(s.T(), addressesOk)
	assert.NotEmpty(s.T(), addresses)

	settings, settingsOk := sellerProfile["settings"].(map[string]any)
	assert.True(s.T(), settingsOk)
	assert.NotNil(s.T(), settings["sellerId"])
}

func (s *AuthTestSuite) TestCustomerLoginDoesNotReturnSellerProfile() {
	requestBody := map[string]any{
		"email":    "alice.j@example.com",
		"password": "customer123",
	}

	w := s.client.Post(s.T(), "/api/user/auth/login", requestBody)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	data, ok := response["data"].(map[string]any)
	assert.True(s.T(), ok)

	_, exists := data["sellerProfile"]
	assert.False(s.T(), exists)
}

// TestScenario2_InvalidCredentials
func (s *AuthTestSuite) TestInvalidCredentials() {
	requestBody := map[string]any{
		"email":    "jane.merchant@example.com",
		"password": "wrongpassword",
	}

	w := s.client.Post(s.T(), "/api/user/auth/login", requestBody)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.False(s.T(), response["success"].(bool))
	assert.Equal(s.T(), "Invalid email or password", response["message"])
}

// TestScenario3_MissingEmail
func (s *AuthTestSuite) TestMissingEmail() {
	requestBody := map[string]any{
		"password": "seller123",
	}

	w := s.client.Post(s.T(), "/api/user/auth/login", requestBody)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.False(s.T(), response["success"].(bool))
}

// TestScenario4_MissingPassword
func (s *AuthTestSuite) TestMissingPassword() {
	requestBody := map[string]any{
		"email": "jane.merchant@example.com",
	}

	w := s.client.Post(s.T(), "/api/user/auth/login", requestBody)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.False(s.T(), response["success"].(bool))
}

// TestScenario5_InvalidEmailFormat
func (s *AuthTestSuite) TestInvalidEmailFormat() {
	requestBody := map[string]any{
		"email":    "not-an-email",
		"password": "seller123",
	}

	w := s.client.Post(s.T(), "/api/user/auth/login", requestBody)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.False(s.T(), response["success"].(bool))
}

// TestScenario6_NonExistentUser
func (s *AuthTestSuite) TestNonExistentUser() {
	requestBody := map[string]any{
		"email":    "nonexistent@example.com",
		"password": "somepassword",
	}

	w := s.client.Post(s.T(), "/api/user/auth/login", requestBody)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.False(s.T(), response["success"].(bool))
	assert.Equal(s.T(), "Invalid email or password", response["message"])
}
