package factory

import (
	"time"

	"ecommerce-be/inventory/entity"
	"ecommerce-be/inventory/mapper"
	"ecommerce-be/inventory/model"
	userEntity "ecommerce-be/user/entity"
	userModel "ecommerce-be/user/model"
)

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
		ID:      address.ID,
		Street:  address.Street,
		City:    address.City,
		State:   address.State,
		ZipCode: address.ZipCode,
		Country: address.Country,
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
) userModel.AddressRequest {
	return userModel.AddressRequest{
		Street:    address.Street,
		City:      address.City,
		State:     address.State,
		ZipCode:   address.ZipCode,
		Country:   address.Country,
		IsDefault: false,
	}
}

// BuildUserAddressResponseToInventoryAddressResponse converts user address response to inventory address response
func BuildUserAddressResponseToInventoryAddressResponse(
	userAddress *userModel.AddressResponse,
) model.AddressResponse {
	return model.AddressResponse{
		ID:      userAddress.ID,
		Street:  userAddress.Street,
		City:    userAddress.City,
		State:   userAddress.State,
		ZipCode: userAddress.ZipCode,
		Country: userAddress.Country,
	}
}

func BuildInventoryUserUpdateReqToUserAddressUpdateReq(
	userAddress model.AddressUpdateRequest,
) userModel.AddressUpdateRequest {
	return userModel.AddressUpdateRequest{
		Street:  userAddress.Street,
		City:    userAddress.City,
		State:   userAddress.State,
		ZipCode: userAddress.ZipCode,
		Country: userAddress.Country,
	}
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
