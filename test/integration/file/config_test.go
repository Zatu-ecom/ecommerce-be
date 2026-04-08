package file_test

import (
	"net/http"
	"testing"

	"ecommerce-be/common/config"
	"ecommerce-be/file/entity"
	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler
	client    *helpers.APIClient

	providerID  uint
	sellerToken string
	adminToken  string
}

func (s *ConfigTestSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())

	// Set required config for encryption
	cfg := config.Get()
	if cfg != nil && cfg.App.EncryptionKey == "" {
		cfg.App.EncryptionKey = "0123456789abcdef0123456789abcdef" // 32 bytes
	}

	// Run migrations and seeds
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())

	// Setup test server
	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)

	// Create API client
	s.client = helpers.NewAPIClient(s.server)

	// Create dummy provider
	provider := entity.StorageProvider{
		Code:        "aws_s3",
		Name:        "AWS S3",
		AdapterType: "s3_compatible",
		IsActive:    true,
	}
	err := s.container.DB.Create(&provider).Error
	assert.NoError(s.T(), err)
	s.providerID = provider.ID

	// Login as seller
	loginReq := map[string]interface{}{
		"email":    "jane.merchant@example.com",
		"password": "seller123",
	}
	w := s.client.Post(s.T(), "/api/user/auth/login", loginReq)
	if w.Code == http.StatusOK {
		resp := helpers.ParseResponse(s.T(), w.Body)
		data := resp["data"].(map[string]interface{})
		s.sellerToken = data["token"].(string)
	}

	// Login as admin
	adminLoginReq := map[string]interface{}{
		"email":    "admin@example.com",
		"password": "adminpassword",
	}
	aw := s.client.Post(s.T(), "/api/user/auth/login", adminLoginReq)
	if aw.Code == http.StatusOK {
		resp := helpers.ParseResponse(s.T(), aw.Body)
		data := resp["data"].(map[string]interface{})
		s.adminToken = data["token"].(string)
	}
}

func (s *ConfigTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (s *ConfigTestSuite) Test1_GetProviders() {
	s.client.SetToken(s.sellerToken)
	w := s.client.Get(s.T(), "/api/v1/files/storage/providers")

	assert.True(s.T(), w.Code == http.StatusOK || w.Code == http.StatusUnauthorized)

	if w.Code == http.StatusOK {
		resp := helpers.ParseResponse(s.T(), w.Body)
		assert.True(s.T(), resp["success"].(bool))
		data := resp["data"].([]interface{})
		assert.NotEmpty(s.T(), data)
	}
}

func (s *ConfigTestSuite) Test2_SuccessfulCreation_Seller() {
	if s.sellerToken == "" {
		s.T().Skip("Requires valid seller auth token")
	}

	reqBody := map[string]interface{}{
		"providerId":        s.providerID,
		"displayName":       "My Seller S3",
		"bucketOrContainer": "my-seller-bucket",
		"region":            "us-east-1",
		"credentials": map[string]string{
			"access_key_id":     "AKIAIOSFODNN7EXAMPLE",
			"secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		},
		"isDefault": true, // Should be ignored since it's a seller
	}

	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), "/api/v1/files/storage-config", reqBody)
	assert.Equal(s.T(), http.StatusOK, w.Code)

	resp := helpers.ParseResponse(s.T(), w.Body)
	assert.True(s.T(), resp["success"].(bool))

	data := resp["data"].(map[string]interface{})
	assert.Equal(s.T(), "My Seller S3", data["displayName"])
	assert.Equal(s.T(), "SELLER", data["ownerType"])
	assert.False(s.T(), data["isDefault"].(bool), "Seller should not be able to set isDefault")
}

func (s *ConfigTestSuite) Test3_SuccessfulCreation_Platform() {
	if s.adminToken == "" {
		s.T().Skip("Requires valid admin auth token")
	}

	reqBody := map[string]interface{}{
		"providerId":        s.providerID,
		"displayName":       "Platform S3",
		"bucketOrContainer": "platform-bucket",
		"region":            "eu-west-1",
		"credentials": map[string]string{
			"access_key_id":     "ADMIN_KEY",
			"secret_access_key": "ADMIN_SECRET",
		},
		"isDefault": true,
	}

	s.client.SetToken(s.adminToken)
	w := s.client.Post(s.T(), "/api/v1/files/storage-config", reqBody)
	assert.Equal(s.T(), http.StatusOK, w.Code)

	resp := helpers.ParseResponse(s.T(), w.Body)
	assert.True(s.T(), resp["success"].(bool))

	data := resp["data"].(map[string]interface{})
	assert.Equal(s.T(), "Platform S3", data["displayName"])
	assert.Equal(s.T(), "PLATFORM", data["ownerType"])
	assert.True(s.T(), data["isDefault"].(bool))
}

func (s *ConfigTestSuite) Test4_ValidationFailure() {
	if s.sellerToken == "" {
		s.T().Skip("Requires valid seller auth token")
	}

	reqBody := map[string]interface{}{
		// Missing required field "providerId"
		"displayName": "Missing Provider ID",
		"credentials": map[string]string{"foo": "bar"},
	}

	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), "/api/v1/files/storage-config", reqBody)
	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}
