package file_test

import (
	"fmt"
	"net/http"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// POST /storage-config/:id/activate — Activation Tests
// ============================================================================
// Tests in this file follow TDD-first integration test convention:
// - All setup via the real API (no direct DB inserts for configs)
// - All assertions via API responses (side effects verified via list API)
// ============================================================================

// Scenario: Seller activates their own existing config.
// Validates: 200 response, isActive=true, correct ownerType returned.
func (s *ConfigTestSuite) TestActivate_SellerHappyPath() {
	configID := s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Seller Activate Happy", "activate-bucket", "", "AK", "SK", false),
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), s.activateURL(configID), map[string]interface{}{})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data := resp["data"].(map[string]interface{})
	assert.Equal(s.T(), true, data["isActive"])
	assert.Equal(s.T(), "SELLER", data["ownerType"])
	assert.Equal(s.T(), float64(configID), data["id"])
}

// Scenario: Activation is called with a non-numeric path ID.
// Validates: 400 is returned for invalid path param.
func (s *ConfigTestSuite) TestActivate_InvalidIDFormat() {
	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), FileAPIBase+"/storage-config/not-a-number/activate", map[string]interface{}{})
	resp := helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
	assert.NotEmpty(s.T(), resp["code"])
}

// Scenario: Seller tries to activate a config owned by a different seller.
// Validates: 403 or 404 — out-of-scope access is rejected.
func (s *ConfigTestSuite) TestActivate_CrossTenantForbidden() {
	configID := s.createConfigAndGetID(
		s.seller2Token,
		s.buildCreateConfigRequest(s.providerID, "Seller2 Config For Activate", "s2-act-bucket", "", "AK2", "SK2", false),
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), s.activateURL(configID), map[string]interface{}{})
	helpers.AssertStatusCodeOneOf(s.T(), w, http.StatusForbidden, http.StatusNotFound)

	resp := helpers.ParseResponse(s.T(), w.Body)
	assert.Equal(s.T(), false, resp["success"])
	assert.NotEmpty(s.T(), resp["message"])
	assert.NotEmpty(s.T(), resp["code"])
}

// Scenario: Activate is called without a token.
// Validates: 401 is returned by auth middleware.
func (s *ConfigTestSuite) TestActivate_Unauthenticated() {
	s.client.SetToken("")
	w := s.client.Post(s.T(), fmt.Sprintf(StorageConfigActivateEndpointTpl, 1), map[string]interface{}{})
	helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
}

// Scenario: Seller activates a config ID that does not exist.
// Validates: 404 is returned.
func (s *ConfigTestSuite) TestActivate_NotFound() {
	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), s.activateURL(999999999), map[string]interface{}{})
	resp := helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
	assert.NotEmpty(s.T(), resp["code"])
}

// Scenario: Seller tries to activate a platform config.
// Validates: 403 or 404 — scope mismatch is rejected.
func (s *ConfigTestSuite) TestActivate_SellerCannotActivatePlatformConfig() {
	platformConfigID := s.createConfigAndGetID(
		s.adminToken,
		s.buildCreateConfigRequest(s.providerID, "Platform Config Activate Test", "plat-act-bucket", "", "AKP", "SKP", false),
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), s.activateURL(platformConfigID), map[string]interface{}{})
	helpers.AssertStatusCodeOneOf(s.T(), w, http.StatusForbidden, http.StatusNotFound)

	resp := helpers.ParseResponse(s.T(), w.Body)
	assert.Equal(s.T(), false, resp["success"])
	assert.NotEmpty(s.T(), resp["code"])
}

// ============================================================================
// Idempotency tests
// ============================================================================

// Scenario: Seller activates an already-active config.
// Validates: Idempotent — returns 200 with isActive=true on repeated call.
func (s *ConfigTestSuite) TestActivate_Idempotent() {
	configID := s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Idempotent Activate", "idempotent-bucket", "", "AKI", "SKI", false),
	)

	s.client.SetToken(s.sellerToken)
	// First activation
	w1 := s.client.Post(s.T(), s.activateURL(configID), map[string]interface{}{})
	helpers.AssertSuccessResponse(s.T(), w1, http.StatusOK)

	// Second activation — must still succeed
	w2 := s.client.Post(s.T(), s.activateURL(configID), map[string]interface{}{})
	resp := helpers.AssertSuccessResponse(s.T(), w2, http.StatusOK)
	data := resp["data"].(map[string]interface{})
	assert.Equal(s.T(), true, data["isActive"])
}

// Scenario: Two different seller configs are activated in sequence.
// Validates: Single-active convergence — only the last activated config is active.
func (s *ConfigTestSuite) TestActivate_SingleActiveConvergence() {
	configA := s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Config A Convergence", "conv-bucket-a", "", "AKA", "SKA", false),
	)
	configB := s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Config B Convergence", "conv-bucket-b", "", "AKB", "SKB", false),
	)

	s.client.SetToken(s.sellerToken)

	// Activate A first
	wA := s.client.Post(s.T(), s.activateURL(configA), map[string]interface{}{})
	helpers.AssertSuccessResponse(s.T(), wA, http.StatusOK)

	// Activate B — B should now be the only active config
	wB := s.client.Post(s.T(), s.activateURL(configB), map[string]interface{}{})
	helpers.AssertSuccessResponse(s.T(), wB, http.StatusOK)

	// Verify via list API: only one active config in seller scope
	listW := s.client.Get(s.T(), StorageConfigEndpoint+"?isActive=true")
	listResp := helpers.AssertSuccessResponse(s.T(), listW, http.StatusOK)

	listData := listResp["data"].(map[string]interface{})
	items := listData["configs"].([]interface{})
	activeCount := 0
	for _, item := range items {
		entry := item.(map[string]interface{})
		if entry["isActive"].(bool) {
			activeCount++
		}
	}
	assert.Equal(s.T(), 1, activeCount, "only one config must be active after B activation")
}
