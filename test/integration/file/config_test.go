package file_test

import (
	"fmt"
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
	FileAPIBase                      = "/api/file"
	StorageProvidersEndpoint         = FileAPIBase + "/storage/providers"
	StorageConfigEndpoint            = FileAPIBase + "/storage-config"
	StorageConfigTestEndpoint        = FileAPIBase + "/storage-config/test"
	StorageConfigActivateEndpointTpl = FileAPIBase + "/storage-config/%d/activate"
)

// ConfigTestSuite holds shared state for all file storage config integration tests.
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

	s.sellerToken = helpers.Login(s.T(), s.client, helpers.SellerEmail, helpers.SellerPassword)
	s.seller2Token = helpers.Login(s.T(), s.client, helpers.Seller4Email, helpers.Seller4Password)
	s.adminToken = helpers.Login(s.T(), s.client, helpers.AdminEmail, helpers.AdminPassword)
}

func (s *ConfigTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

// ============================================================================
// Helpers — shared across all config test files in this package
// ============================================================================

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
		providerID, displayName, bucket, "", accessKey, secretKey, false,
	)
	reqBody["id"] = configID
	return reqBody
}

// createConfigAndGetID creates a storage config via the real API and returns its ID.
// Uses the standard POST /storage-config endpoint — no direct DB insertion.
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

// activateURL formats the activate endpoint URL.
func (s *ConfigTestSuite) activateURL(configID uint) string {
	return fmt.Sprintf(StorageConfigActivateEndpointTpl, configID)
}

// ============================================================================
// PROVIDER TESTS
// ============================================================================

// Scenario: Authenticated seller requests active storage providers.
// Validates: 200 response and non-empty provider list payload.
func (s *ConfigTestSuite) TestGetProvidersSuccess() {
	s.client.SetToken(s.sellerToken)
	w := s.client.Get(s.T(), StorageProvidersEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	assert.NotEmpty(s.T(), resp["data"])
}

// ============================================================================
// SAVE / CREATE CONFIG TESTS
// ============================================================================

// Scenario: Seller creates a storage config while passing isDefault=true.
// Validates: Seller ownership is enforced and default flag is forced to false.
func (s *ConfigTestSuite) TestSaveConfig_SellerSuccessForcesNotDefault() {
	reqBody := s.buildCreateConfigRequest(
		s.providerID, "Seller Storage", "seller-bucket", "ap-south-1", "SELLER_KEY", "SELLER_SECRET", true,
	)
	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), StorageConfigEndpoint, reqBody)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data := resp["data"].(map[string]interface{})
	assert.Equal(s.T(), "SELLER", data["ownerType"])
	assert.Equal(s.T(), false, data["isDefault"])
}

// Scenario: Admin creates a platform default configuration.
// Validates: Platform ownership and default flag are persisted for admin requests.
func (s *ConfigTestSuite) TestSaveConfig_AdminSuccessPlatformDefault() {
	reqBody := s.buildCreateConfigRequest(
		s.providerID, "Platform Storage", "platform-bucket", "us-east-1", "ADMIN_KEY", "ADMIN_SECRET", true,
	)
	s.client.SetToken(s.adminToken)
	w := s.client.Post(s.T(), StorageConfigEndpoint, reqBody)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data := resp["data"].(map[string]interface{})
	assert.Equal(s.T(), "PLATFORM", data["ownerType"])
	assert.Equal(s.T(), true, data["isDefault"])
}

// Scenario: Request is sent without a bearer token.
// Validates: Middleware rejects unauthenticated access with 401.
func (s *ConfigTestSuite) TestSaveConfig_FailsWithoutAuth() {
	s.client.SetToken("")
	w := s.client.Post(s.T(), StorageConfigEndpoint, map[string]interface{}{})
	helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
}

// Scenario: Request is sent with an invalid bearer token.
// Validates: Middleware rejects invalid JWT with 401.
func (s *ConfigTestSuite) TestSaveConfig_FailsWithInvalidToken() {
	s.client.SetToken("invalid-token")
	w := s.client.Post(s.T(), StorageConfigEndpoint, map[string]interface{}{})
	helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
}

// Scenario: Request body omits required fields.
// Validates: Binding/validation errors return 400.
func (s *ConfigTestSuite) TestSaveConfig_ValidationFailure() {
	reqBody := map[string]interface{}{
		"displayName": "Missing provider and bucket",
		"credentials": map[string]string{"foo": "bar"},
	}
	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), StorageConfigEndpoint, reqBody)
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// Scenario: Request body is malformed JSON.
// Validates: Invalid payload format returns 400.
func (s *ConfigTestSuite) TestSaveConfig_MalformedPayload() {
	s.client.SetToken(s.sellerToken)
	w := s.client.PostRaw(s.T(), StorageConfigEndpoint, []byte(`{"providerId":`))
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// Scenario: Request references a non-existent provider.
// Validates: Service rejects unknown provider IDs with 400.
func (s *ConfigTestSuite) TestSaveConfig_UnknownProvider() {
	reqBody := s.buildCreateConfigRequest(999999, "Unknown provider", "bucket", "", "AK", "SK", false)
	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), StorageConfigEndpoint, reqBody)
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// Scenario: Request references an inactive provider.
// Validates: Service rejects inactive providers with 400.
func (s *ConfigTestSuite) TestSaveConfig_InactiveProvider() {
	reqBody := s.buildCreateConfigRequest(s.inactiveProvider, "Inactive provider", "bucket", "", "AK", "SK", false)
	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), StorageConfigEndpoint, reqBody)
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// Scenario: Seller tries to update another seller's config ID.
// Validates: Cross-tenant config mutation is blocked with 403.
func (s *ConfigTestSuite) TestSaveConfig_CrossTenantUpdateForbidden() {
	configID := s.createConfigAndGetID(
		s.seller2Token,
		s.buildCreateConfigRequest(s.providerID, "Seller 2 Config", "seller2-bucket", "", "AK2", "SK2", false),
	)
	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), StorageConfigEndpoint, s.buildUpdateConfigRequest(
		configID, s.providerID, "Attempt Cross Tenant Update", "seller1-bucket", "AK1", "SK1",
	))
	helpers.AssertErrorResponse(s.T(), w, http.StatusForbidden)
}

// Scenario: Seller tries to update a platform-owned config.
// Validates: Owner-type authorization is enforced with 403.
func (s *ConfigTestSuite) TestSaveConfig_SellerCannotManagePlatformConfig() {
	platformConfigID := s.createConfigAndGetID(
		s.adminToken,
		s.buildCreateConfigRequest(s.providerID, "Platform Config For Auth Test", "platform-auth-bucket", "", "AKA", "SKA", false),
	)
	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), StorageConfigEndpoint, s.buildUpdateConfigRequest(
		platformConfigID, s.providerID, "Seller Attempt Platform Update", "seller-bucket", "AKS", "SKS",
	))
	helpers.AssertErrorResponse(s.T(), w, http.StatusForbidden)
}

// Scenario: Request updates a config ID that does not exist.
// Validates: Unknown config updates return 404.
func (s *ConfigTestSuite) TestSaveConfig_UnknownConfigID() {
	reqBody := s.buildUpdateConfigRequest(999999, s.providerID, "Unknown Config", "bucket", "AK", "SK")
	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), StorageConfigEndpoint, reqBody)
	helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
}

// Scenario: Authenticated request omits X-Correlation-ID header.
// Validates: Correlation middleware blocks request with 400.
func (s *ConfigTestSuite) TestCorrelationIDRequired() {
	s.client.SetHeader("X-Correlation-ID", "")
	s.client.SetToken(s.sellerToken)
	w := s.client.Get(s.T(), StorageProvidersEndpoint)
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
	// Restore default behavior for subsequent tests.
	s.client.SetHeader("X-Correlation-ID", "test-correlation-id-file-config")
}

// Scenario: Existing request shape for save-config is used (legacy-compatible payload).
// Validates: Legacy clients can still create configs without contract changes.
func (s *ConfigTestSuite) TestSaveConfig_BackwardCompatible() {
	reqBody := s.buildCreateConfigRequest(
		s.providerID, "Legacy Payload Config", "legacy-bucket", "eu-west-1", "LEGACY_AK", "LEGACY_SK", false,
	)
	reqBody["configJson"] = map[string]interface{}{
		"customEndpoint": "https://s3.example.com",
	}
	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), StorageConfigEndpoint, reqBody)
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
}

// Scenario: TestConfig stub endpoint returns 501.
func (s *ConfigTestSuite) TestStubTestConfigEndpointReturnsNotImplemented() {
	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), StorageConfigTestEndpoint, map[string]interface{}{})
	assert.Equal(s.T(), http.StatusNotImplemented, w.Code)
}
