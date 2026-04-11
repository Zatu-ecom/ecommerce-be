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

const (
	FileAPIBase                 = "/api/file"
	StorageProvidersEndpoint    = FileAPIBase + "/storage/providers"
	StorageConfigEndpoint       = FileAPIBase + "/storage-config"
	StorageConfigTestEndpoint   = FileAPIBase + "/storage-config/test"
	StorageConfigActiveEndpoint = FileAPIBase + "/storage-config/active"
)

type ConfigTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler
	client    *helpers.APIClient

	providerID       uint
	inactiveProvider uint
	sellerToken      string
	seller2Token     string
	adminToken       string
}

func (s *ConfigTestSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())

	cfg := config.Get()
	if cfg != nil && cfg.App.EncryptionKey == "" {
		cfg.App.EncryptionKey = "0123456789abcdef0123456789abcdef"
	}

	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())

	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)
	s.client = helpers.NewAPIClient(s.server)

	var provider entity.StorageProvider
	err := s.container.DB.Where("code = ?", "aws_s3").First(&provider).Error
	assert.NoError(s.T(), err)
	s.providerID = provider.ID

	inactive := entity.StorageProvider{
		Code:        "aws_s3_inactive",
		Name:        "AWS S3 Inactive",
		AdapterType: "s3_compatible",
	}
	err = s.container.DB.Create(&inactive).Error
	assert.NoError(s.T(), err)
	err = s.container.DB.Model(&inactive).Update("is_active", false).Error
	assert.NoError(s.T(), err)
	s.inactiveProvider = inactive.ID

	s.sellerToken = helpers.Login(
		s.T(),
		s.client,
		helpers.SellerEmail,
		helpers.SellerPassword,
	)
	s.seller2Token = helpers.Login(
		s.T(),
		s.client,
		helpers.Seller4Email,
		helpers.Seller4Password,
	)
	s.adminToken = helpers.Login(
		s.T(),
		s.client,
		helpers.AdminEmail,
		helpers.AdminPassword,
	)
}

func (s *ConfigTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (s *ConfigTestSuite) buildCredentials(accessKey, secretKey string) map[string]string {
	return map[string]string{
		"access_key_id":     accessKey,
		"secret_access_key": secretKey,
	}
}

func (s *ConfigTestSuite) buildCreateConfigRequest(
	providerID uint,
	displayName string,
	bucket string,
	region string,
	accessKey string,
	secretKey string,
	isDefault bool,
) map[string]interface{} {
	reqBody := map[string]interface{}{
		"providerId":        providerID,
		"displayName":       displayName,
		"bucketOrContainer": bucket,
		"credentials":       s.buildCredentials(accessKey, secretKey),
		"isDefault":         isDefault,
	}
	if region != "" {
		reqBody["region"] = region
	}
	return reqBody
}

func (s *ConfigTestSuite) buildUpdateConfigRequest(
	configID uint,
	providerID uint,
	displayName string,
	bucket string,
	accessKey string,
	secretKey string,
) map[string]interface{} {
	reqBody := s.buildCreateConfigRequest(
		providerID,
		displayName,
		bucket,
		"",
		accessKey,
		secretKey,
		false,
	)
	reqBody["id"] = configID
	return reqBody
}

func (s *ConfigTestSuite) createConfigAndGetID(
	token string,
	reqBody map[string]interface{},
) uint {
	s.client.SetToken(token)
	resp := s.client.Post(s.T(), StorageConfigEndpoint, reqBody)
	assert.Equal(s.T(), http.StatusOK, resp.Code)
	data := helpers.ParseResponse(s.T(), resp.Body)["data"].(map[string]interface{})
	return uint(data["id"].(float64))
}

// Scenario: Authenticated seller requests active storage providers.
// Validates: 200 response and non-empty provider list payload.
func (s *ConfigTestSuite) TestGetProvidersSuccess() {
	s.client.SetToken(s.sellerToken)
	w := s.client.Get(s.T(), StorageProvidersEndpoint)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	resp := helpers.ParseResponse(s.T(), w.Body)
	assert.Equal(s.T(), true, resp["success"])
	assert.NotEmpty(s.T(), resp["data"])
}

// Scenario: Seller creates a storage config while passing isDefault=true.
// Validates: Seller ownership is enforced and default flag is forced to false.
func (s *ConfigTestSuite) TestSaveConfigSellerSuccessForcesNotDefault() {
	reqBody := s.buildCreateConfigRequest(
		s.providerID,
		"Seller Storage",
		"seller-bucket",
		"ap-south-1",
		"SELLER_KEY",
		"SELLER_SECRET",
		true,
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), StorageConfigEndpoint, reqBody)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	resp := helpers.ParseResponse(s.T(), w.Body)
	assert.Equal(s.T(), true, resp["success"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(s.T(), "SELLER", data["ownerType"])
	assert.Equal(s.T(), false, data["isDefault"])
}

// Scenario: Admin creates a platform default configuration.
// Validates: Platform ownership and default flag are persisted for admin requests.
func (s *ConfigTestSuite) TestSaveConfigAdminSuccessPlatformDefault() {
	reqBody := s.buildCreateConfigRequest(
		s.providerID,
		"Platform Storage",
		"platform-bucket",
		"us-east-1",
		"ADMIN_KEY",
		"ADMIN_SECRET",
		true,
	)

	s.client.SetToken(s.adminToken)
	w := s.client.Post(s.T(), StorageConfigEndpoint, reqBody)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	resp := helpers.ParseResponse(s.T(), w.Body)
	assert.Equal(s.T(), true, resp["success"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(s.T(), "PLATFORM", data["ownerType"])
	assert.Equal(s.T(), true, data["isDefault"])
}

// Scenario: Request is sent without a bearer token.
// Validates: Middleware rejects unauthenticated access with 401.
func (s *ConfigTestSuite) TestSaveConfigFailsWithoutAuth() {
	s.client.SetToken("")
	w := s.client.Post(s.T(), StorageConfigEndpoint, map[string]interface{}{})

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}

// Scenario: Request is sent with an invalid bearer token.
// Validates: Middleware rejects invalid JWT with 401.
func (s *ConfigTestSuite) TestSaveConfigFailsWithInvalidToken() {
	s.client.SetToken("invalid-token")
	w := s.client.Post(s.T(), StorageConfigEndpoint, map[string]interface{}{})

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}

// Scenario: Request body omits required fields.
// Validates: Binding/validation errors return 400.
func (s *ConfigTestSuite) TestSaveConfigValidationFailure() {
	reqBody := map[string]interface{}{
		"displayName": "Missing provider and bucket",
		"credentials": map[string]string{"foo": "bar"},
	}

	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), StorageConfigEndpoint, reqBody)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}

// Scenario: Request body is malformed JSON.
// Validates: Invalid payload format returns 400.
func (s *ConfigTestSuite) TestSaveConfigMalformedPayload() {
	s.client.SetToken(s.sellerToken)
	w := s.client.PostRaw(s.T(), StorageConfigEndpoint, []byte(`{"providerId":`))

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}

// Scenario: Request references a non-existent provider.
// Validates: Service rejects unknown provider IDs with 400.
func (s *ConfigTestSuite) TestSaveConfigUnknownProvider() {
	reqBody := s.buildCreateConfigRequest(
		999999,
		"Unknown provider",
		"bucket",
		"",
		"AK",
		"SK",
		false,
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), StorageConfigEndpoint, reqBody)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}

// Scenario: Request references an inactive provider.
// Validates: Service rejects inactive providers with 400.
func (s *ConfigTestSuite) TestSaveConfigInactiveProvider() {
	reqBody := s.buildCreateConfigRequest(
		s.inactiveProvider,
		"Inactive provider",
		"bucket",
		"",
		"AK",
		"SK",
		false,
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), StorageConfigEndpoint, reqBody)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}

// Scenario: Seller tries to update another seller's config ID.
// Validates: Cross-tenant config mutation is blocked with 403.
func (s *ConfigTestSuite) TestSellerIsolationCrossTenantUpdateForbidden() {
	createBody := s.buildCreateConfigRequest(
		s.providerID,
		"Seller 2 Config",
		"seller2-bucket",
		"",
		"AK2",
		"SK2",
		false,
	)
	configID := s.createConfigAndGetID(s.seller2Token, createBody)

	s.client.SetToken(s.sellerToken)
	updateBody := s.buildUpdateConfigRequest(
		configID,
		s.providerID,
		"Attempt Cross Tenant Update",
		"seller1-bucket",
		"AK1",
		"SK1",
	)
	w := s.client.Post(s.T(), StorageConfigEndpoint, updateBody)

	assert.Equal(s.T(), http.StatusForbidden, w.Code)
}

// Scenario: Seller tries to update a platform-owned config.
// Validates: Owner-type authorization is enforced with 403.
func (s *ConfigTestSuite) TestSellerCannotManagePlatformConfig() {
	adminCreate := s.buildCreateConfigRequest(
		s.providerID,
		"Platform Config For Auth Test",
		"platform-auth-bucket",
		"",
		"AKA",
		"SKA",
		false,
	)
	platformConfigID := s.createConfigAndGetID(s.adminToken, adminCreate)

	s.client.SetToken(s.sellerToken)
	sellerUpdate := s.buildUpdateConfigRequest(
		platformConfigID,
		s.providerID,
		"Seller Attempt Platform Update",
		"seller-bucket",
		"AKS",
		"SKS",
	)
	w := s.client.Post(s.T(), StorageConfigEndpoint, sellerUpdate)

	assert.Equal(s.T(), http.StatusForbidden, w.Code)
}

// Scenario: Request updates a config ID that does not exist.
// Validates: Unknown config updates return 404.
func (s *ConfigTestSuite) TestSaveConfigUnknownConfigID() {
	reqBody := s.buildUpdateConfigRequest(
		999999,
		s.providerID,
		"Unknown Config",
		"bucket",
		"AK",
		"SK",
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), StorageConfigEndpoint, reqBody)

	assert.Equal(s.T(), http.StatusNotFound, w.Code)
}

// Scenario: Authenticated request omits X-Correlation-ID header.
// Validates: Correlation middleware blocks request with 400.
func (s *ConfigTestSuite) TestCorrelationIDRequired() {
	s.client.SetHeader("X-Correlation-ID", "")
	s.client.SetToken(s.sellerToken)
	w := s.client.Get(s.T(), StorageProvidersEndpoint)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)

	// Restore default behavior for subsequent tests.
	s.client.SetHeader("X-Correlation-ID", "test-correlation-id-file-config")
}

// Scenario: Existing request shape for save-config is used (legacy-compatible payload).
// Validates: Legacy clients can still create configs without contract changes.
func (s *ConfigTestSuite) TestBackwardCompatibilityExistingRequestShape() {
	reqBody := s.buildCreateConfigRequest(
		s.providerID,
		"Legacy Payload Config",
		"legacy-bucket",
		"eu-west-1",
		"LEGACY_AK",
		"LEGACY_SK",
		false,
	)
	reqBody["configJson"] = map[string]interface{}{
		"customEndpoint": "https://s3.example.com",
	}

	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), StorageConfigEndpoint, reqBody)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	resp := helpers.ParseResponse(s.T(), w.Body)
	assert.Equal(s.T(), true, resp["success"])
}

// Scenario: Stub endpoints are called for yet-to-be-implemented functionality.
// Validates: Endpoints respond consistently with explicit not-implemented status.
func (s *ConfigTestSuite) TestStubEndpointsReturnNotImplemented() {
	s.client.SetToken(s.sellerToken)

	testResp := s.client.Post(s.T(), StorageConfigTestEndpoint, map[string]interface{}{})
	assert.Equal(s.T(), http.StatusNotImplemented, testResp.Code)

	activeResp := s.client.Get(s.T(), StorageConfigActiveEndpoint)
	assert.Equal(s.T(), http.StatusNotImplemented, activeResp.Code)
}
