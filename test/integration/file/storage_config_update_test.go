package file_test

import (
	"fmt"
	"net/http"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// PUT /storage-config/:id — Update + default switching
// ============================================================================

// Scenario: Seller updates their own config (display name and flags).
func (s *ConfigTestSuite) TestUpdate_SellerHappyPath() {
	configID := s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Before update", "upd-bucket-a", "", "AK1", "SK1", false),
	)

	s.client.SetToken(s.sellerToken)
	body := s.buildUpdateConfigBody(
		s.providerID,
		"After update",
		"upd-bucket-a",
		"AK1",
		"SK1",
		false,
		true,
	)
	w := s.client.Put(s.T(), s.storageConfigURL(configID), body)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	assert.Equal(s.T(), "After update", data["displayName"])
	assert.Equal(s.T(), false, data["isActive"])
	assert.Equal(s.T(), true, data["isDefault"])
}

// Scenario: Update path param is not numeric.
func (s *ConfigTestSuite) TestUpdate_InvalidIDFormat() {
	s.client.SetToken(s.sellerToken)
	body := s.buildUpdateConfigBody(s.providerID, "x", "b", "ak", "sk", true, false)
	w := s.client.Put(s.T(), FileAPIBase+"/storage-config/not-a-number", body)
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// Scenario: Seller updates another seller's config.
func (s *ConfigTestSuite) TestUpdate_CrossTenantForbidden() {
	configID := s.createConfigAndGetID(
		s.seller2Token,
		s.buildCreateConfigRequest(s.providerID, "S2 for update", "s2-upd-bucket", "", "AK2", "SK2", false),
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Put(s.T(), s.storageConfigURL(configID), s.buildUpdateConfigBody(
		s.providerID, "Hijack", "s2-upd-bucket", "AK", "SK", true, false,
	))
	helpers.AssertStatusCodeOneOf(s.T(), w, http.StatusForbidden, http.StatusNotFound)
}

// Scenario: Unauthenticated PUT returns 401.
func (s *ConfigTestSuite) TestUpdate_Unauthenticated() {
	s.client.SetToken("")
	body := s.buildUpdateConfigBody(s.providerID, "x", "b", "ak", "sk", true, false)
	w := s.client.Put(s.T(), fmt.Sprintf("%s/%d", StorageConfigEndpoint, 1), body)
	helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
}

// Scenario: Non-existent config ID returns 404.
func (s *ConfigTestSuite) TestUpdate_NotFound() {
	s.client.SetToken(s.sellerToken)
	w := s.client.Put(s.T(), s.storageConfigURL(999999999), s.buildUpdateConfigBody(
		s.providerID, "Nope", "b", "ak", "sk", true, false,
	))
	helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
}

// Scenario: Seller cannot update a platform-owned config.
func (s *ConfigTestSuite) TestUpdate_SellerCannotUpdatePlatformConfig() {
	platformConfigID := s.createConfigAndGetID(
		s.adminToken,
		s.buildCreateConfigRequest(s.providerID, "Platform for PUT auth", "plat-upd-bucket", "", "AKP", "SKP", false),
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Put(s.T(), s.storageConfigURL(platformConfigID), s.buildUpdateConfigBody(
		s.providerID, "Seller tries", "plat-upd-bucket", "AKS", "SKS", true, false,
	))
	helpers.AssertStatusCodeOneOf(s.T(), w, http.StatusForbidden, http.StatusNotFound)
}

// Scenario: Repeating the same PUT succeeds (idempotent payload).
func (s *ConfigTestSuite) TestUpdate_IdempotentPayload() {
	configID := s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Idempotent PUT", "idem-bucket", "", "AKI", "SKI", false),
	)
	body := s.buildUpdateConfigBody(
		s.providerID, "Idempotent PUT", "idem-bucket", "AKI", "SKI", true, false,
	)

	s.client.SetToken(s.sellerToken)
	w1 := s.client.Put(s.T(), s.storageConfigURL(configID), body)
	helpers.AssertSuccessResponse(s.T(), w1, http.StatusOK)
	w2 := s.client.Put(s.T(), s.storageConfigURL(configID), body)
	helpers.AssertSuccessResponse(s.T(), w2, http.StatusOK)
}

// Scenario: Only one seller config may be default; setting B default clears A.
func (s *ConfigTestSuite) TestUpdate_SingleDefaultConvergence() {
	configA := s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Default A", "def-a-bucket", "", "AKA", "SKA", true),
	)
	configB := s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Default B", "def-b-bucket", "", "AKB", "SKB", false),
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Put(s.T(), s.storageConfigURL(configB), s.buildUpdateConfigBody(
		s.providerID,
		"Default B",
		"def-b-bucket",
		"AKB",
		"SKB",
		true,
		true,
	))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	listW := s.client.Get(s.T(), StorageConfigEndpoint+"?isDefault=true")
	listResp := helpers.AssertSuccessResponse(s.T(), listW, http.StatusOK)
	items := listResp["data"].(map[string]any)["configs"].([]any)
	defaultIDs := make(map[uint]struct{})
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["isDefault"].(bool) {
			defaultIDs[uint(entry["id"].(float64))] = struct{}{}
		}
	}
	_, hasA := defaultIDs[configA]
	_, hasB := defaultIDs[configB]
	assert.False(s.T(), hasA, "previous default A should be cleared")
	assert.True(s.T(), hasB, "B should be the default")
}

// Scenario: Second created seller config clears the first default when both use create defaults.
func (s *ConfigTestSuite) TestSave_SecondCreateClearsPriorDefault() {
	firstID := s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "First default", "first-def-bucket", "", "AK1", "SK1", true),
	)
	secondID := s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Second default", "second-def-bucket", "", "AK2", "SK2", true),
	)

	s.client.SetToken(s.sellerToken)
	listW := s.client.Get(s.T(), StorageConfigEndpoint+"?isDefault=true")
	listResp := helpers.AssertSuccessResponse(s.T(), listW, http.StatusOK)
	items := listResp["data"].(map[string]any)["configs"].([]any)
	var defaultID uint
	count := 0
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["isDefault"].(bool) {
			count++
			defaultID = uint(entry["id"].(float64))
		}
	}
	assert.Equal(s.T(), 1, count)
	assert.Equal(s.T(), secondID, defaultID)
	assert.NotEqual(s.T(), firstID, defaultID)
}
