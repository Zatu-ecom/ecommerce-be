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
	return &entity.Address{
		UserID:    userID,
		Street:    req.Street,
		City:      req.City,
		State:     req.State,
		ZipCode:   req.ZipCode,
		CountryID: req.CountryID,
		IsDefault: req.IsDefault,
	}
}

// UpdateAddressEntity updates an existing address entity from update request
// Only updates fields that are provided (not nil)
func UpdateAddressEntity(address *entity.Address, req model.AddressUpdateRequest) {
	if req.Street != nil {
		address.Street = *req.Street
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
	if req.IsDefault != nil {
		address.IsDefault = *req.IsDefault
	}
}

// BuildAddressResponse converts an address entity to response model
func BuildAddressResponse(address *entity.Address) model.AddressResponse {
	response := model.AddressResponse{
		ID:        address.ID,
		Street:    address.Street,
		City:      address.City,
		State:     address.State,
		ZipCode:   address.ZipCode,
		CountryID: address.CountryID,
		IsDefault: address.IsDefault,
	}

	// Include expanded country info if relationship is loaded
	if address.Country.ID != 0 {
		response.Country = BuildCountryResponsePtr(&address.Country)
	}

	return response
}
