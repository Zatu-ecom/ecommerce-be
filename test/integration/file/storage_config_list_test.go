package file_test

import (
	"fmt"
	"net/http"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// GET /storage-config — Listing + Filter + Error Schema Tests
// ============================================================================
// All setup exclusively via the real API (createConfigAndGetID helper).
// Assertions against standardized response envelopes.
// ============================================================================

// ── Scope tests ────────────────────────────────────────────────────────────

// Scenario: Seller calls listing endpoint.
// Validates: 200 response; all returned configs belong to SELLER scope.
func (s *ConfigTestSuite) TestListConfigs_SellerScope() {
	// Ensure at least one seller config exists.
	s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Seller List Config", "seller-list-bucket", "", "AKSL", "SKSL", false),
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Get(s.T(), StorageConfigEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data := resp["data"].(map[string]interface{})
	items := data["configs"].([]interface{})
	assert.NotEmpty(s.T(), items)

	for _, item := range items {
		entry := item.(map[string]interface{})
		assert.Equal(s.T(), "SELLER", entry["ownerType"],
			"expected only SELLER configs in seller scope, got %v", entry["ownerType"])
	}
	assert.NotNil(s.T(), data["pagination"])
}

// Scenario: Admin (platform-scope) calls listing endpoint.
// Validates: 200 response; all returned configs belong to PLATFORM scope.
func (s *ConfigTestSuite) TestListConfigs_PlatformScope() {
	// Ensure at least one platform config exists.
	s.createConfigAndGetID(
		s.adminToken,
		s.buildCreateConfigRequest(s.providerID, "Platform List Config", "plat-list-bucket", "", "AKPL", "SKPL", false),
	)

	s.client.SetToken(s.adminToken)
	w := s.client.Get(s.T(), StorageConfigEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data := resp["data"].(map[string]interface{})
	items := data["configs"].([]interface{})
	assert.NotEmpty(s.T(), items)

	for _, item := range items {
		entry := item.(map[string]interface{})
		assert.Equal(s.T(), "PLATFORM", entry["ownerType"],
			"expected only PLATFORM configs in platform scope, got %v", entry["ownerType"])
	}
}

// Scenario: Seller with no configs calls listing endpoint.
// Validates: 200 response with an empty configs array (not 404).
func (s *ConfigTestSuite) TestListConfigs_EmptyListForNewSeller() {
	// seller2 hasn't created any configs in this test's scope; endpoint must still return 200.
	s.client.SetToken(s.seller2Token)
	w := s.client.Get(s.T(), StorageConfigEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data := resp["data"].(map[string]interface{})
	assert.NotNil(s.T(), data["configs"], "configs key must be present even when empty")
}

// Scenario: Unauthenticated request to listing endpoint.
// Validates: 401 returned by auth middleware.
func (s *ConfigTestSuite) TestListConfigs_Unauthenticated() {
	s.client.SetToken("")
	w := s.client.Get(s.T(), StorageConfigEndpoint)
	helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
}

// ── Filter tests ────────────────────────────────────────────────────────────

// Scenario: Filter by isActive=true.
// Validates: All returned configs have isActive=true.
func (s *ConfigTestSuite) TestListConfigs_FilterByIsActive() {
	s.client.SetToken(s.sellerToken)
	w := s.client.Get(s.T(), StorageConfigEndpoint+"?isActive=true")
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data := resp["data"].(map[string]interface{})
	items := data["configs"].([]interface{})
	for _, item := range items {
		assert.Equal(s.T(), true, item.(map[string]interface{})["isActive"])
	}
}

// Scenario: Filter by isDefault=false.
// Validates: All returned configs have isDefault=false.
func (s *ConfigTestSuite) TestListConfigs_FilterByIsDefault() {
	s.client.SetToken(s.sellerToken)
	w := s.client.Get(s.T(), StorageConfigEndpoint+"?isDefault=false")
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data := resp["data"].(map[string]interface{})
	items := data["configs"].([]interface{})
	for _, item := range items {
		assert.Equal(s.T(), false, item.(map[string]interface{})["isDefault"])
	}
}

// Scenario: Filter by validationStatuses=PENDING.
// Validates: All returned configs have validationStatus=PENDING.
func (s *ConfigTestSuite) TestListConfigs_FilterByValidationStatuses() {
	// New configs get PENDING status by default; ensure at least one exists.
	s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Pending Status Config", "pending-bucket", "", "AKPND", "SKPND", false),
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Get(s.T(), StorageConfigEndpoint+"?validationStatuses=PENDING")
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data := resp["data"].(map[string]interface{})
	items := data["configs"].([]interface{})
	assert.NotEmpty(s.T(), items)
	for _, item := range items {
		assert.Equal(s.T(), "PENDING", item.(map[string]interface{})["validationStatus"])
	}
}

// Scenario: Filter by specific config IDs.
// Validates: Only the config matching the given ID is returned.
func (s *ConfigTestSuite) TestListConfigs_FilterByIDs() {
	configID := s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Filter By IDs Config", "filter-ids-bucket", "", "AKFI", "SKFI", false),
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Get(s.T(), fmt.Sprintf("%s?ids=%d", StorageConfigEndpoint, configID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data := resp["data"].(map[string]interface{})
	items := data["configs"].([]interface{})
	assert.Len(s.T(), items, 1)
	assert.Equal(s.T(), float64(configID), items[0].(map[string]interface{})["id"])
}

// Scenario: Filter by providerIds.
// Validates: All returned configs use the specified provider.
func (s *ConfigTestSuite) TestListConfigs_FilterByProviderIDs() {
	// Ensure at least one config with this provider exists.
	s.createConfigAndGetID(
		s.sellerToken,
		s.buildCreateConfigRequest(s.providerID, "Provider Filter Config", "provider-filter-bucket", "", "AKPF", "SKPF", false),
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Get(s.T(), fmt.Sprintf("%s?providerIds=%d", StorageConfigEndpoint, s.providerID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data := resp["data"].(map[string]interface{})
	items := data["configs"].([]interface{})
	assert.NotEmpty(s.T(), items)
	for _, item := range items {
		assert.Equal(s.T(), float64(s.providerID), item.(map[string]interface{})["providerId"])
	}
}

// Scenario: Combine isActive and validationStatuses filters.
// Validates: Compound filter returns 200 within the caller's scope.
func (s *ConfigTestSuite) TestListConfigs_CombinedFilters() {
	s.client.SetToken(s.sellerToken)
	w := s.client.Get(s.T(), StorageConfigEndpoint+"?isActive=true&validationStatuses=PENDING")
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
}

// ── Forbidden filter + pagination edge-case tests ───────────────────────────

// Scenario: Caller supplies 'sellerId' as a query param.
// Validates: 400 is returned with a field-level errors array — sellerId is a forbidden filter.
func (s *ConfigTestSuite) TestListConfigs_ForbiddenSellerIDFilter() {
	s.client.SetToken(s.sellerToken)
	w := s.client.Get(s.T(), StorageConfigEndpoint+"?sellerId=123")
	resp := helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)

	// Must include an errors array with the offending field indicated.
	errs, hasErrors := resp["errors"]
	assert.True(s.T(), hasErrors, "forbidden filter response must include errors array")
	assert.NotEmpty(s.T(), errs)
}

// Scenario: Empty search param (ignored).
// Validates: 200 — optional param is a no-op when blank.
func (s *ConfigTestSuite) TestListConfigs_EmptySearchParam() {
	s.client.SetToken(s.sellerToken)
	w := s.client.Get(s.T(), StorageConfigEndpoint+"?search=")
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
}

// Scenario: pageSize=0 (invalid, below minimum).
// Validates: 200 — service normalises to default; request does not error.
func (s *ConfigTestSuite) TestListConfigs_InvalidPageSizeNormalized() {
	s.client.SetToken(s.sellerToken)
	w := s.client.Get(s.T(), StorageConfigEndpoint+"?pageSize=0")
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
}

// Scenario: pageSize=200 (above maximum of 100).
// Validates: 200 with itemsPerPage clamped to ≤ 100 in the pagination block.
func (s *ConfigTestSuite) TestListConfigs_PageSizeClamped() {
	s.client.SetToken(s.sellerToken)
	w := s.client.Get(s.T(), StorageConfigEndpoint+"?pageSize=200")
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data := resp["data"].(map[string]interface{})
	pagination := data["pagination"].(map[string]interface{})
	assert.LessOrEqual(s.T(), int(pagination["itemsPerPage"].(float64)), 100)
}

// ── Error-schema contract tests ─────────────────────────────────────────────

// Scenario: Activation with non-numeric ID — standardised 400 error schema.
// Validates: success=false, non-empty message, non-empty code.
func (s *ConfigTestSuite) TestErrorSchema_ActivateInvalidID() {
	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), FileAPIBase+"/storage-config/not-a-number/activate", map[string]interface{}{})
	resp := helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
	assert.NotEmpty(s.T(), resp["code"])
}

// Scenario: Activation of a non-existent config — standardised 404 error schema.
func (s *ConfigTestSuite) TestErrorSchema_ActivateNotFound() {
	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), s.activateURL(999999997), map[string]interface{}{})
	resp := helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
	assert.NotEmpty(s.T(), resp["code"])
}

// Scenario: List with forbidden sellerId — standardised 400 with errors array.
func (s *ConfigTestSuite) TestErrorSchema_ListForbiddenSellerID() {
	s.client.SetToken(s.sellerToken)
	w := s.client.Get(s.T(), StorageConfigEndpoint+"?sellerId=99")
	resp := helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
	assert.NotEmpty(s.T(), resp["code"])

	errs, hasErrors := resp["errors"]
	assert.True(s.T(), hasErrors, "forbidden-filter error response must include errors array")
	assert.NotEmpty(s.T(), errs)
}

// Scenario: Unauthenticated activate — standardised 401.
func (s *ConfigTestSuite) TestErrorSchema_ActivateUnauthenticated() {
	s.client.SetToken("")
	w := s.client.Post(s.T(), s.activateURL(1), map[string]interface{}{})
	resp := helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
	assert.NotEmpty(s.T(), resp["message"])
}

// Scenario: Unauthenticated list — standardised 401.
func (s *ConfigTestSuite) TestErrorSchema_ListUnauthenticated() {
	s.client.SetToken("")
	w := s.client.Get(s.T(), StorageConfigEndpoint)
	resp := helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
	assert.NotEmpty(s.T(), resp["message"])
}

// Scenario: Cross-tenant activation — 403 or 404 with proper error schema.
func (s *ConfigTestSuite) TestErrorSchema_ActivateCrossTenantForbidden() {
	configID := s.createConfigAndGetID(
		s.seller2Token,
		s.buildCreateConfigRequest(s.providerID, "Schema Cross Tenant", "schema-ct-bucket", "", "AKS2", "SKS2", false),
	)

	s.client.SetToken(s.sellerToken)
	w := s.client.Post(s.T(), s.activateURL(configID), map[string]interface{}{})
	helpers.AssertStatusCodeOneOf(s.T(), w, http.StatusForbidden, http.StatusNotFound)

	resp := helpers.ParseResponse(s.T(), w.Body)
	assert.Equal(s.T(), false, resp["success"])
	assert.NotEmpty(s.T(), resp["message"])
	assert.NotEmpty(s.T(), resp["code"])
}
