package user

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/suite"
)

const AddressAPIEndpoint = "/api/user/address"

// AddressTestSuite proves and guards address create/update/default flows.
// Notably: creating/updating with isDefault=true must not fail address-type
// validation when clearing sibling defaults (GORM empty-model hooks bug).
type AddressTestSuite struct {
	suite.Suite
	container      *setup.TestContainer
	server         http.Handler
	customerClient *helpers.APIClient
}

func (s *AddressTestSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())

	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)

	s.customerClient = helpers.NewAPIClient(s.server)
	token := helpers.Login(
		s.T(),
		s.customerClient,
		helpers.CustomerEmail,
		helpers.CustomerPassword,
	)
	s.customerClient.SetToken(token)
}

func (s *AddressTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func TestAddressSuite(t *testing.T) {
	suite.Run(t, new(AddressTestSuite))
}

func (s *AddressTestSuite) validCreatePayload(isDefault bool) map[string]any {
	return map[string]any{
		"type":      "HOME",
		"address":   "Amee Residency A Block",
		"landmark":  "Near ring road",
		"city":      "Rajkot",
		"state":     "Gujarat",
		"zipCode":   "360005",
		"countryId": 1,
		"isDefault": isDefault,
	}
}

// TestCreateDefaultAddressSucceedsWhenUserAlreadyHasAddresses
// Customer seed user (id=5) already has addresses. Creating another as default
// must return 201 — previously failed with invalid address type via GORM hooks.
func (s *AddressTestSuite) TestCreateDefaultAddressSucceedsWhenUserAlreadyHasAddresses() {
	payload := s.validCreatePayload(true)

	w := s.customerClient.Post(s.T(), AddressAPIEndpoint, payload)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	data, ok := resp["data"].(map[string]any)
	s.Require().True(ok)
	addr, ok := data["address"].(map[string]any)
	s.Require().True(ok)

	s.Equal("HOME", addr["type"])
	s.Equal(true, addr["isDefault"])
	s.Equal("Rajkot", addr["city"])
}

// TestCreateAddressWithLatLngPersistsCoordinates
func (s *AddressTestSuite) TestCreateAddressWithLatLngPersistsCoordinates() {
	payload := s.validCreatePayload(true)
	payload["type"] = "WORK"
	payload["latitude"] = 22.3039
	payload["longitude"] = 70.8022

	w := s.customerClient.Post(s.T(), AddressAPIEndpoint, payload)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	data := resp["data"].(map[string]any)
	addr := data["address"].(map[string]any)

	s.Equal("WORK", addr["type"])
	s.InDelta(22.3039, addr["latitude"].(float64), 0.0001)
	s.InDelta(70.8022, addr["longitude"].(float64), 0.0001)
	s.Equal(true, addr["isDefault"])
}

// TestSetDefaultAddressSucceeds
func (s *AddressTestSuite) TestSetDefaultAddressSucceeds() {
	payload := s.validCreatePayload(false)
	payload["type"] = "WORK"
	payload["address"] = "Non-default office address for set-default test"

	w := s.customerClient.Post(s.T(), AddressAPIEndpoint, payload)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	addr := resp["data"].(map[string]any)["address"].(map[string]any)
	addressID := uint(addr["id"].(float64))
	// isDefault=false is omitted from JSON (omitempty) — ensure it is not true
	s.NotEqual(true, addr["isDefault"])

	w = s.customerClient.Patch(
		s.T(),
		fmt.Sprintf("%s/%d/default", AddressAPIEndpoint, addressID),
		map[string]any{},
	)
	resp = helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	updated := resp["data"].(map[string]any)["address"].(map[string]any)
	s.Equal(true, updated["isDefault"])
	s.Equal(float64(addressID), updated["id"])
}

// TestUpdateAddressAsDefaultSucceeds
func (s *AddressTestSuite) TestUpdateAddressAsDefaultSucceeds() {
	payload := s.validCreatePayload(false)
	payload["type"] = "HOME"
	payload["address"] = "Address to update as default later"

	w := s.customerClient.Post(s.T(), AddressAPIEndpoint, payload)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	addr := resp["data"].(map[string]any)["address"].(map[string]any)
	addressID := uint(addr["id"].(float64))

	isDefault := true
	updatePayload := map[string]any{
		"isDefault": isDefault,
		"city":      "Ahmedabad",
	}

	w = s.customerClient.Put(
		s.T(),
		fmt.Sprintf("%s/%d", AddressAPIEndpoint, addressID),
		updatePayload,
	)
	resp = helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	updated := resp["data"].(map[string]any)["address"].(map[string]any)
	s.Equal(true, updated["isDefault"])
	s.Equal("Ahmedabad", updated["city"])
	s.Equal("HOME", updated["type"])
}
