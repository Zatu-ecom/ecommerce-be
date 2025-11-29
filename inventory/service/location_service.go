package service

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/inventory/entity"
	invErrors "ecommerce-be/inventory/error"
	factory "ecommerce-be/inventory/factory"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/repository"
	"ecommerce-be/inventory/validator"
	userModel "ecommerce-be/user/model"
	userService "ecommerce-be/user/service"

	"gorm.io/gorm"
)

// LocationService defines the interface for location-related business logic
type LocationService interface {
	CreateLocation(
		c context.Context,
		req model.LocationCreateRequest,
		sellerID uint,
	) (*model.LocationResponse, error)
	UpdateLocation(
		c context.Context,
		id uint,
		req model.LocationUpdateRequest,
		sellerID uint,
	) (*model.LocationResponse, error)
	GetLocationByID(
		c context.Context,
		id uint,
		sellerID uint) (*model.LocationResponse, error)
	GetAllLocations(
		c context.Context,
		sellerID uint,
		isActive *bool,
	) ([]model.LocationResponse, error)
	DeleteLocation(c context.Context, id uint, sellerID uint) error
}

// LocationServiceImpl implements the LocationService interface
type LocationServiceImpl struct {
	locationRepo   repository.LocationRepository
	addressService userService.AddressService
}

// NewLocationService creates a new instance of LocationService
func NewLocationService(
	locationRepo repository.LocationRepository,
	addressService userService.AddressService,
) *LocationServiceImpl {
	return &LocationServiceImpl{
		locationRepo:   locationRepo,
		addressService: addressService,
	}
}

// CreateLocation creates a new location with address
func (s *LocationServiceImpl) CreateLocation(
	c context.Context,
	req model.LocationCreateRequest,
	sellerID uint,
) (*model.LocationResponse, error) {
	// Validate location type and unique name
	if err := s.validateLocationCreate(req, sellerID); err != nil {
		return nil, err
	}

	var response *model.LocationResponse
	err := db.Atomic(func(tx *gorm.DB) error {
		// Create address
		address, err := s.createAddress(sellerID, req.Address)
		if err != nil {
			return err
		}

		// Create location entity
		locationEntity := factory.BuildLocationEntity(req, sellerID)
		locationEntity.AddressID = address.ID

		if err := s.locationRepo.Create(locationEntity); err != nil {
			return err
		}

		// Build response
		response = s.buildLocationResponseWithAddress(locationEntity, address)
		return nil
	})

	return response, err
}

// UpdateLocation updates an existing location
func (s *LocationServiceImpl) UpdateLocation(
	c context.Context,
	id uint,
	req model.LocationUpdateRequest,
	sellerID uint,
) (*model.LocationResponse, error) {
	// Find and validate existing location
	location, err := s.locationRepo.FindByID(id, sellerID)
	if err != nil {
		return nil, err
	}

	// Validate name uniqueness if being updated
	if err := s.validateLocationUpdate(req, location, sellerID, id); err != nil {
		return nil, err
	}

	var response *model.LocationResponse
	err = db.Atomic(func(tx *gorm.DB) error {
		// Update location entity
		factory.BuildUpdateLocationEntity(location, req)
		if err := s.locationRepo.Update(location); err != nil {
			return err
		}

		// Handle address update or fetch
		address, err := s.handleAddressForUpdate(req.Address, location.AddressID, sellerID)
		if err != nil {
			return err
		}

		// Build response
		response = s.buildLocationResponseWithAddress(location, address)
		return nil
	})

	return response, err
}

// GetLocationByID retrieves a location by ID
func (s *LocationServiceImpl) GetLocationByID(
	c context.Context,
	id uint,
	sellerID uint,
) (*model.LocationResponse, error) {
	location, err := s.locationRepo.FindByID(id, sellerID)
	if err != nil {
		return nil, err
	}

	// Build response using factory
	response := factory.BuildLocationResponse(location)

	// Fetch address if address_id exists
	if location.AddressID != 0 {
		// TODO [MICROSERVICE]: Replace with HTTP call to User Service Address API
		address, err := s.addressService.GetAddresses(sellerID)
		if err == nil {
			for _, addr := range address {
				if addr.ID == location.AddressID {
					response.Address = factory.BuildUserAddressResponseToInventoryAddressResponse(
						&addr,
					)
					break
				}
			}
		}
	}

	return response, nil
}

// GetAllLocations retrieves all locations for a seller
func (s *LocationServiceImpl) GetAllLocations(
	c context.Context,
	sellerID uint,
	isActive *bool,
) ([]model.LocationResponse, error) {
	locations, err := s.locationRepo.FindAll(sellerID, isActive)
	if err != nil {
		return nil, err
	}

	// Fetch all addresses for seller once
	addresses, _ := s.addressService.GetAddresses(sellerID)
	addressMap := make(map[uint]model.AddressResponse)
	for _, addr := range addresses {
		addressMap[addr.ID] = factory.BuildUserAddressResponseToInventoryAddressResponse(&addr)
	}

	// Build location responses
	var locationResponses []model.LocationResponse
	for i := range locations {
		response := factory.BuildLocationResponse(&locations[i])

		// Add address if exists
		if addr, found := addressMap[locations[i].AddressID]; found {
			response.Address = addr
		}

		locationResponses = append(locationResponses, *response)
	}

	return locationResponses, nil
}

// DeleteLocation soft deletes a location
func (s *LocationServiceImpl) DeleteLocation(
	c context.Context,
	id uint,
	sellerID uint,
) error {
	// Check if location exists and belongs to seller
	if err := s.locationRepo.Exists(id, sellerID); err != nil {
		return err
	}

	// TODO: Add validation to check if location has inventory
	// This should be implemented when inventory management is added

	return s.locationRepo.Delete(id)
}

// ============================================================================
// Private Helper Methods
// ============================================================================

// validateLocationCreate validates location creation request
func (s *LocationServiceImpl) validateLocationCreate(
	req model.LocationCreateRequest,
	sellerID uint,
) error {
	// Validate location type
	if err := validator.ValidateLocationType(req.Type); err != nil {
		return err
	}

	// Check for duplicate location name
	return s.validateNameUniqueness(req.Name, sellerID, nil)
}

// validateLocationUpdate validates location update request
func (s *LocationServiceImpl) validateLocationUpdate(
	req model.LocationUpdateRequest,
	location *entity.Location,
	sellerID uint,
	locationID uint,
) error {
	// Validate location type if being updated
	if req.Type != nil {
		if err := validator.ValidateLocationType(*req.Type); err != nil {
			return err
		}
	}

	// If name is being updated, check for duplicates
	if req.Name != nil && *req.Name != location.Name {
		return s.validateNameUniqueness(*req.Name, sellerID, &locationID)
	}

	return nil
}

// validateNameUniqueness checks if a location name is unique for a seller
func (s *LocationServiceImpl) validateNameUniqueness(
	name string,
	sellerID uint,
	excludeID *uint,
) error {
	existingLocation, err := s.locationRepo.FindByName(name, sellerID)
	if err != nil && err != invErrors.ErrLocationNotFound {
		return err
	}

	return validator.ValidateUniqueName(name, sellerID, existingLocation, excludeID)
}

// createAddress creates a new address via user service
// TODO [MICROSERVICE]: Replace with HTTP call to User Service Address API
func (s *LocationServiceImpl) createAddress(
	sellerID uint,
	addressReq model.AddressRequest,
) (*userModel.AddressResponse, error) {
	return s.addressService.AddAddress(
		sellerID,
		factory.BuildUserAddressReqToInventoryAddressReq(addressReq),
	)
}

// handleAddressForUpdate updates or fetches address based on request
// TODO [MICROSERVICE]: Replace with HTTP call to User Service Address API
func (s *LocationServiceImpl) handleAddressForUpdate(
	addressReq *model.AddressUpdateRequest,
	addressID uint,
	sellerID uint,
) (*userModel.AddressResponse, error) {
	if addressReq != nil {
		// Update address
		return s.addressService.UpdateAddress(
			addressID,
			sellerID,
			factory.BuildInventoryUserUpdateReqToUserAddressUpdateReq(*addressReq),
		)
	}

	// Fetch existing address
	return s.addressService.GetAddressByID(addressID, sellerID)
}

// buildLocationResponseWithAddress builds location response with address
func (s *LocationServiceImpl) buildLocationResponseWithAddress(
	location *entity.Location,
	address *userModel.AddressResponse,
) *model.LocationResponse {
	response := factory.BuildLocationResponse(location)
	response.Address = factory.BuildUserAddressResponseToInventoryAddressResponse(address)
	return response
}
