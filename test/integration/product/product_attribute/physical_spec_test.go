package product_attribute

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/require"
)

// setupPhysicalSpecServer boots containers with migrations + core seeds
// (005 physical specs) + users + products.
func setupPhysicalSpecServer(t *testing.T) (*setup.TestContainer, *helpers.APIClient) {
	t.Helper()
	containers := setup.SetupTestContainers(t)
	t.Cleanup(func() { containers.Cleanup(t) })

	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
	return containers, helpers.NewAPIClient(server)
}

func loginSeller(t *testing.T, client *helpers.APIClient) {
	t.Helper()
	token := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
	client.SetToken(token)
}

// ─── Spec catalog ────────────────────────────────────────────────────────────
// Scenario: the fulfillment scope returns 4 parameter groups with unit
// options; unknown scope is rejected.
func TestPhysicalSpecCatalog(t *testing.T) {
	_, client := setupPhysicalSpecServer(t)
	loginSeller(t, client)

	w := client.Get(t, "/api/product/attribute/definitions?scope=fulfillment")
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	groups, ok := resp["data"].(map[string]any)["parameters"].([]any)
	require.True(t, ok, "data.parameters must be a list")
	require.Len(t, groups, 4)

	byParam := map[string]map[string]any{}
	for _, g := range groups {
		group := g.(map[string]any)
		byParam[group["parameter"].(string)] = group
	}
	weight, ok := byParam["weight"]
	require.True(t, ok, "weight parameter present")
	options := weight["options"].([]any)
	require.Len(t, options, 2)
	units := map[string]bool{}
	for _, o := range options {
		opt := o.(map[string]any)
		unit := opt["unit"].(map[string]any)
		require.Contains(t, []string{"g", "kg"}, unit["short"])
		require.NotEmpty(t, unit["full"])
		units[unit["short"].(string)] = true
	}
	require.True(t, units["g"] && units["kg"])

	w = client.Get(t, "/api/product/attribute/definitions?scope=whatever")
	helpers.AssertErrorResponse(t, w, http.StatusBadRequest)

	w = client.Get(t, "/api/product/attribute/definitions")
	helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
}

// ─── Shippable write validation ─────────────────────────────────────────────
// Scenario: weight_g with a non-numeric value is rejected; a second key
// from the same family (weight_kg after weight_g) is rejected.
func TestPhysicalSpecWriteValidation(t *testing.T) {
	_, client := setupPhysicalSpecServer(t)
	loginSeller(t, client)

	// Jane (seller 3) owns product 5. weight_g definition id is 100.
	w := client.Post(t, "/api/product/5/attribute", map[string]any{
		"attributeDefinitionId": 100,
		"value":                 "heavy-ish",
	})
	helpers.AssertErrorResponse(t, w, http.StatusBadRequest)

	w = client.Post(t, "/api/product/5/attribute", map[string]any{
		"attributeDefinitionId": 100,
		"value":                 "650",
	})
	helpers.AssertSuccessResponse(t, w, http.StatusCreated)

	w = client.Post(t, "/api/product/5/attribute", map[string]any{
		"attributeDefinitionId": 101,
		"value":                 "0.65",
	})
	errResp := helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	require.Contains(t, []string{
		"PHYSICAL_SPEC_FAMILY_CONFLICT", "PRODUCT_ATTRIBUTE_EXISTS",
	}, errResp["code"])
}

// ─── Inactive units hidden ───────────────────────────────────────────────────
// Scenario: deactivating weight_kg removes it from the catalog options.
func TestPhysicalSpecInactiveHidden(t *testing.T) {
	containers, client := setupPhysicalSpecServer(t)
	loginSeller(t, client)

	require.NoError(t, containers.DB.Exec(
		"UPDATE physical_spec_unit SET is_active = false WHERE key = 'weight_kg'").Error)

	w := client.Get(t, "/api/product/attribute/definitions?scope=fulfillment")
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	for _, g := range resp["data"].(map[string]any)["parameters"].([]any) {
		group := g.(map[string]any)
		if group["parameter"] == "weight" {
			require.Len(t, group["options"], 1)
			opt := group["options"].([]any)[0].(map[string]any)
			require.Equal(t, "weight_g", opt["key"])
		}
	}
}

// ─── Shipping-specs badge ────────────────────────────────────────────────────
// Scenario: a product without specs reports all families missing; after
// adding weight_g, weight leaves the missing list.
func TestPhysicalSpecBadge(t *testing.T) {
	_, client := setupPhysicalSpecServer(t)
	loginSeller(t, client)

	w := client.Get(t, "/api/product/5/attribute/shipping-specs")
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	data := resp["data"].(map[string]any)
	require.Empty(t, data["present"])
	require.Len(t, data["missing"], 4)

	w = client.Post(t, "/api/product/5/attribute", map[string]any{
		"attributeDefinitionId": 100,
		"value":                 "650",
	})
	helpers.AssertSuccessResponse(t, w, http.StatusCreated)

	w = client.Get(t, fmt.Sprintf("/api/product/%d/attribute/shipping-specs", 5))
	resp = helpers.AssertSuccessResponse(t, w, http.StatusOK)
	data = resp["data"].(map[string]any)
	require.Len(t, data["present"], 1)
	require.Len(t, data["missing"], 3)
}
