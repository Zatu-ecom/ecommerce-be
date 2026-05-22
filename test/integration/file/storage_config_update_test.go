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

// Scenario: Admin updates a platform config they created.
func (s *ConfigTestSuite) TestUpdate_AdminHappyPath() {
	configID := s.createConfigAndGetID(
		s.adminToken,
		s.buildCreateConfigRequest(s.providerID, "Platform Before update", "plat-upd-bucket-a", "", "PAK1", "PSK1", true),
	)

	s.client.SetToken(s.adminToken)
	body := s.buildUpdateConfigBody(
		s.providerID,
		"Platform After update",
		"plat-upd-bucket-a",
		"PAK1",
		"PSK1",
		true,
		false,
	)
	w := s.client.Put(s.T(), s.storageConfigURL(configID), body)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	assert.Equal(s.T(), "PLATFORM", data["ownerType"])
	assert.Equal(s.T(), "Platform After update", data["displayName"])
	assert.Equal(s.T(), true, data["isActive"])
	assert.Equal(s.T(), false, data["isDefault"])
}

// Scenario: Update path param is not numeric.
func (s *ConfigTestSuite) TestUpdate_InvalidIDFormat() {
	s.client.SetToken(s.sellerToken)
	body := s.buildUpdateConfigBody(s.providerID, "x", "b", "ak", "sk", true, false)
	w := s.client.Put(s.T(), FileAPIBase+"/storage-config/not-a-number", body)
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// Scenario: PUT with missing required body fields returns 400.
func (s *ConfigTestSuite) TestUpdate_BodyValidationFailure() {
	configID := s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Body validation base", "upd-body-val", "", "AK", "SK", false),
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Put(s.T(), s.storageConfigURL(configID), map[string]any{
		// missing providerId, displayName, bucketOrContainer, config, isActive, isDefault
	})
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

// Scenario: Admin cannot update another seller's config.
func (s *ConfigTestSuite) TestUpdate_AdminCannotUpdateSellerConfig() {
	configID := s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Seller config for admin auth", "adm-upd-seller-bucket", "", "AKS", "SKS", false),
	)

	s.client.SetToken(s.adminToken)
	w := s.client.Put(s.T(), s.storageConfigURL(configID), s.buildUpdateConfigBody(
		s.providerID, "Admin tries seller", "adm-upd-seller-bucket", "AK", "SK", true, false,
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

// Scenario: Only one platform config may be default; updating B to default clears A.
func (s *ConfigTestSuite) TestUpdate_PlatformSingleDefaultConvergence() {
	configA := s.createConfigAndGetID(
		s.adminToken,
		s.buildCreateConfigRequest(s.providerID, "Platform Default A", "plat-def-a", "", "AKA", "SKA", true),
	)
	configB := s.createConfigAndGetID(
		s.adminToken,
		s.buildCreateConfigRequest(s.providerID, "Platform Default B", "plat-def-b", "", "AKB", "SKB", false),
	)

	s.client.SetToken(s.adminToken)
	w := s.client.Put(s.T(), s.storageConfigURL(configB), s.buildUpdateConfigBody(
		s.providerID,
		"Platform Default B",
		"plat-def-b",
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
	assert.False(s.T(), hasA, "previous platform default A should be cleared")
	assert.True(s.T(), hasB, "platform B should be the default")
}

// Scenario: Setting isDefault=false is allowed and may result in zero defaults.
func (s *ConfigTestSuite) TestUpdate_UnsetDefault_AllowsNoDefault() {
	configID := s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Unset default", "unset-def-bucket", "", "AK", "SK", true),
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Put(s.T(), s.storageConfigURL(configID), s.buildUpdateConfigBody(
		s.providerID,
		"Unset default",
		"unset-def-bucket",
		"AK",
		"SK",
		true,
		false,
	))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	listW := s.client.Get(s.T(), StorageConfigEndpoint+"?isDefault=true")
	listResp := helpers.AssertSuccessResponse(s.T(), listW, http.StatusOK)
	items := listResp["data"].(map[string]any)["configs"].([]any)
	assert.Len(s.T(), items, 0, "no defaults should remain after unsetting the only default")
}

// Scenario: isActive is independent of isDefault; multiple configs may be active while only one is default.
func (s *ConfigTestSuite) TestUpdate_IsActiveIndependentOfDefault() {
	configA := s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Active A", "active-a", "", "AKA", "SKA", true),
	)
	configB := s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Active B", "active-b", "", "AKB", "SKB", false),
	)

	s.client.SetToken(s.sellerToken)
	// Ensure both are active; make B default as well (should clear A default only).
	w := s.client.Put(s.T(), s.storageConfigURL(configB), s.buildUpdateConfigBody(
		s.providerID, "Active B", "active-b", "AKB", "SKB", true, true,
	))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Both active:
	activeW := s.client.Get(s.T(), StorageConfigEndpoint+"?isActive=true")
	activeResp := helpers.AssertSuccessResponse(s.T(), activeW, http.StatusOK)
	activeItems := activeResp["data"].(map[string]any)["configs"].([]any)
	assert.GreaterOrEqual(s.T(), len(activeItems), 2)

	// Only one default:
	defaultW := s.client.Get(s.T(), StorageConfigEndpoint+"?isDefault=true")
	defaultResp := helpers.AssertSuccessResponse(s.T(), defaultW, http.StatusOK)
	defaultItems := defaultResp["data"].(map[string]any)["configs"].([]any)
	assert.Len(s.T(), defaultItems, 1)
	gotDefaultID := uint(defaultItems[0].(map[string]any)["id"].(float64))
	assert.NotEqual(s.T(), configA, gotDefaultID)
	assert.Equal(s.T(), configB, gotDefaultID)
}

// Scenario: Update with an unknown provider returns 400.
func (s *ConfigTestSuite) TestUpdate_UnknownProvider() {
	configID := s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Base for unknown provider", "upd-unk-prov", "", "AK", "SK", false),
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Put(s.T(), s.storageConfigURL(configID), s.buildUpdateConfigBody(
		999999,
		"Unknown provider update",
		"upd-unk-prov",
		"AK",
		"SK",
		true,
		false,
	))
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// Scenario: Update with an inactive provider returns 400.
func (s *ConfigTestSuite) TestUpdate_InactiveProvider() {
	configID := s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Base for inactive provider", "upd-inact-prov", "", "AK", "SK", false),
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Put(s.T(), s.storageConfigURL(configID), s.buildUpdateConfigBody(
		s.inactiveProvider,
		"Inactive provider update",
		"upd-inact-prov",
		"AK",
		"SK",
		true,
		false,
	))
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
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

// Scenario: Second created platform config clears the first default when both are created default=true.
func (s *ConfigTestSuite) TestSave_PlatformSecondCreateClearsPriorDefault() {
	firstID := s.createConfigAndGetID(
		s.adminToken,
		s.buildCreateConfigRequest(s.providerID, "Platform First default", "plat-first-def", "", "AK1", "SK1", true),
	)
	secondID := s.createConfigAndGetID(
		s.adminToken,
		s.buildCreateConfigRequest(s.providerID, "Platform Second default", "plat-second-def", "", "AK2", "SK2", true),
	)

	s.client.SetToken(s.adminToken)
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
