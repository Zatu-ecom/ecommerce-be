package factory

import (
	"time"

	"ecommerce-be/inventory/entity"
	"ecommerce-be/inventory/mapper"
	"ecommerce-be/inventory/model"
	userEntity "ecommerce-be/user/entity"
	userModel "ecommerce-be/user/model"
)

// LocationTypeToAddressType maps inventory location type to user address type
func LocationTypeToAddressType(locType entity.LocationType) userEntity.AddressType {
	switch locType {
	case entity.LOC_WAREHOUSE:
		return userEntity.ADDR_WAREHOUSE
	case entity.LOC_STORE:
		return userEntity.ADDR_STORE
	case entity.LOC_RETURN_CENTER:
		return userEntity.ADDR_RETURN_CENTER
	default:
		return userEntity.ADDR_OTHER
	}
}

// BuildLocationResponse converts a Location entity to LocationResponse DTO
func BuildLocationResponse(location *entity.Location) *model.LocationResponse {
	if location == nil {
		return nil
	}
	response := &model.LocationResponse{
		ID:       location.ID,
		Name:     location.Name,
		Type:     location.Type,
		IsActive: location.IsActive,
		Priority: location.Priority,
	}

	return response
}

// BuildAddressResponse converts an Address entity to AddressResponse DTO
func BuildAddressResponse(address *userEntity.Address) model.AddressResponse {
	return model.AddressResponse{
		ID:        address.ID,
		Address:   address.Address,
		Landmark:  address.Landmark,
		City:      address.City,
		State:     address.State,
		ZipCode:   address.ZipCode,
		CountryID: address.CountryID,
		Latitude:  address.Latitude,
		Longitude: address.Longitude,
	}
}

// BuildLocationEntity creates a Location entity from CreateRequest
func BuildLocationEntity(req model.LocationCreateRequest, sellerID uint) *entity.Location {
	return &entity.Location{
		Name:     req.Name,
		Type:     req.Type,
		IsActive: true, // Default to active on creation
		Priority: req.Priority,
		SellerID: sellerID,
	}
}

// BuildUpdateLocationEntity updates a Location entity from UpdateRequest
func BuildUpdateLocationEntity(location *entity.Location, req model.LocationUpdateRequest) {
	if req.Name != nil {
		location.Name = *req.Name
	}
	if req.Priority != nil {
		location.Priority = *req.Priority
	}
	if req.Type != nil {
		location.Type = *req.Type
	}
	if req.IsActive != nil {
		location.IsActive = *req.IsActive
	}
	location.UpdatedAt = time.Now().UTC()
}

func BuildUserAddressReqToInventoryAddressReq(
	address model.AddressRequest,
	locType entity.LocationType,
) userModel.AddressRequest {
	return userModel.AddressRequest{
		Type:      LocationTypeToAddressType(locType), // Sync address type with location type
		Address:   address.Address,
		Landmark:  address.Landmark,
		City:      address.City,
		State:     address.State,
		ZipCode:   address.ZipCode,
		CountryID: address.CountryID,
		Latitude:  address.Latitude,
		Longitude: address.Longitude,
		IsDefault: false,
	}
}

// BuildUserAddressResponseToInventoryAddressResponse converts user address response to inventory address response
func BuildUserAddressResponseToInventoryAddressResponse(
	userAddress *userModel.AddressResponse,
) model.AddressResponse {
	return model.AddressResponse{
		ID:        userAddress.ID,
		Address:   userAddress.Address,
		Landmark:  userAddress.Landmark,
		City:      userAddress.City,
		State:     userAddress.State,
		ZipCode:   userAddress.ZipCode,
		CountryID: userAddress.CountryID,
		Latitude:  userAddress.Latitude,
		Longitude: userAddress.Longitude,
	}
}

func BuildInventoryUserUpdateReqToUserAddressUpdateReq(
	userAddress model.AddressUpdateRequest,
	locType *entity.LocationType,
) userModel.AddressUpdateRequest {
	req := userModel.AddressUpdateRequest{
		Address:   userAddress.Address,
		Landmark:  userAddress.Landmark,
		City:      userAddress.City,
		State:     userAddress.State,
		ZipCode:   userAddress.ZipCode,
		CountryID: userAddress.CountryID,
		Latitude:  userAddress.Latitude,
		Longitude: userAddress.Longitude,
	}

	// Sync address type if location type is being updated
	if locType != nil {
		addrType := LocationTypeToAddressType(*locType)
		req.Type = &addrType
	}

	return req
}

// BuildLocationSummaryResponse creates a LocationSummaryResponse with inventory data
func BuildLocationSummaryResponse(
	locationResp model.LocationResponse,
	productCount uint,
	invSummary mapper.LocationInventorySummaryAggregate,
	averageStockValue float64,
	stockStatus model.StockStatus,
) model.LocationSummaryResponse {
	return model.LocationSummaryResponse{
		LocationResponse: locationResp,
		InventorySummary: model.InventorySummary{
			ProductCount:      productCount,
			VariantCount:      invSummary.VariantCount,
			TotalStock:        invSummary.TotalStock,
			TotalReserved:     invSummary.TotalReserved,
			TotalAvailable:    invSummary.TotalStock - invSummary.TotalReserved,
			LowStockCount:     invSummary.LowStockCount,
			OutOfStockCount:   invSummary.OutOfStockCount,
			AverageStockValue: averageStockValue,
			StockStatus:       stockStatus,
		},
	}
}
