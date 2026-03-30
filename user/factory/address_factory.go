package factory

import (
	"ecommerce-be/user/entity"
	"ecommerce-be/user/model"
)

/***********************************************
 *          Address Entity Builders            *
 ***********************************************/

// BuildAddressEntity creates a new address entity from create request
func BuildAddressEntity(userID uint, req model.AddressRequest) *entity.Address {
	// Default to HOME if type is not provided
	addrType := req.Type
	if addrType == "" {
		addrType = entity.ADDR_HOME
	}

	return &entity.Address{
		UserID:    userID,
		Type:      addrType,
		Address:   req.Address,
		Landmark:  req.Landmark,
		City:      req.City,
		State:     req.State,
		ZipCode:   req.ZipCode,
		CountryID: req.CountryID,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		IsDefault: req.IsDefault,
	}
}

// UpdateAddressEntity updates an existing address entity from update request
// Only updates fields that are provided (not nil)
func UpdateAddressEntity(address *entity.Address, req model.AddressUpdateRequest) {
	if req.Type != nil {
		address.Type = *req.Type
	}
	if req.Address != nil {
		address.Address = *req.Address
	}
	if req.Landmark != nil {
		address.Landmark = *req.Landmark
	}
	if req.City != nil {
		address.City = *req.City
	}
	if req.State != nil {
		address.State = *req.State
	}
	if req.ZipCode != nil {
		address.ZipCode = *req.ZipCode
	}
	if req.CountryID != nil {
		address.CountryID = *req.CountryID
	}
	if req.Latitude != nil {
		address.Latitude = req.Latitude
	}
	if req.Longitude != nil {
		address.Longitude = req.Longitude
	}
	if req.IsDefault != nil {
		address.IsDefault = *req.IsDefault
	}
}

// BuildAddressResponse converts an address entity to response model
func BuildAddressResponse(address *entity.Address) model.AddressResponse {
	response := model.AddressResponse{
		ID:        address.ID,
		Type:      address.Type,
		Address:   address.Address,
		Landmark:  address.Landmark,
		City:      address.City,
		State:     address.State,
		ZipCode:   address.ZipCode,
		CountryID: address.CountryID,
		Latitude:  address.Latitude,
		Longitude: address.Longitude,
		IsDefault: address.IsDefault,
	}

	// Include expanded country info if relationship is loaded
	if address.Country.ID != 0 {
		response.Country = BuildCountryResponsePtr(&address.Country)
	}

	return response
}
