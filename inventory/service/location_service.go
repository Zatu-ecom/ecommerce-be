package service

import (
	"context"

	"ecommerce-be/common"
	"ecommerce-be/common/db"
	"ecommerce-be/inventory/entity"
	invErrors "ecommerce-be/inventory/error"
	factory "ecommerce-be/inventory/factory"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/repository"
	"ecommerce-be/inventory/validator"
	userModel "ecommerce-be/user/model"
	userService "ecommerce-be/user/service"
)

// LocationService defines the interface for location-related business logic
type LocationService interface {
	CreateLocation(
		ctx context.Context,
		req model.LocationCreateRequest,
		sellerID uint,
	) (*model.LocationResponse, error)
	UpdateLocation(
		ctx context.Context,
		id uint,
		req model.LocationUpdateRequest,
		sellerID uint,
	) (*model.LocationResponse, error)
	GetLocationByID(
		ctx context.Context,
		id uint,
		sellerID uint) (*model.LocationResponse, error)
	GetAllLocations(
		ctx context.Context,
		sellerID uint,
		filter model.LocationsFilter,
	) (*model.LocationsResponse, error)
	DeleteLocation(ctx context.Context, id uint, sellerID uint) error
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
	ctx context.Context,
	req model.LocationCreateRequest,
	sellerID uint,
) (*model.LocationResponse, error) {
	// Validate location type and unique name
	if err := s.validateLocationCreate(ctx, req, sellerID); err != nil {
		return nil, err
	}

	var response *model.LocationResponse
	err := db.WithTransaction(ctx, func(txCtx context.Context) error {
		// Create address (sync address type with location type)
		address, err := s.createAddress(txCtx, sellerID, req.Address, req.Type)
		if err != nil {
			return err
		}

		// Create location entity
		locationEntity := factory.BuildLocationEntity(req, sellerID)
		locationEntity.AddressID = address.ID

		if err := s.locationRepo.Create(txCtx, locationEntity); err != nil {
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
	ctx context.Context,
	id uint,
	req model.LocationUpdateRequest,
	sellerID uint,
) (*model.LocationResponse, error) {
	// Find and validate existing location
	location, err := s.locationRepo.FindByID(ctx, id, sellerID)
	if err != nil {
		return nil, err
	}

	// Validate name uniqueness if being updated
	if err := s.validateLocationUpdate(ctx, req, location, sellerID, id); err != nil {
		return nil, err
	}

	var response *model.LocationResponse
	err = db.WithTransaction(ctx, func(txCtx context.Context) error {
		// Update location entity
		factory.BuildUpdateLocationEntity(location, req)
		if err := s.locationRepo.Update(txCtx, location); err != nil {
			return err
		}

		// Handle address update or fetch (sync address type if location type changed)
		address, err := s.handleAddressForUpdate(
			ctx,
			req.Address,
			location.AddressID,
			sellerID,
			req.Type,
		)
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
	ctx context.Context,
	id uint,
	sellerID uint,
) (*model.LocationResponse, error) {
	location, err := s.locationRepo.FindByID(ctx, id, sellerID)
	if err != nil {
		return nil, err
	}

	// Build response using factory
	response := factory.BuildLocationResponse(location)

	// Fetch address if address_id exists
	if location.AddressID != 0 {
		// TODO [MICROSERVICE]: Replace with HTTP call to User Service Address API
		address, err := s.addressService.GetAddresses(ctx, sellerID)
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

// GetAllLocations retrieves all locations for a seller with pagination
func (s *LocationServiceImpl) GetAllLocations(
	ctx context.Context,
	sellerID uint,
	filter model.LocationsFilter,
) (*model.LocationsResponse, error) {
	// Set pagination defaults
	filter.SetDefaults()

	// Get total count for pagination
	totalCount, err := s.locationRepo.CountAll(ctx, sellerID, filter)
	if err != nil {
		return nil, err
	}

	if totalCount == 0 {
		return &model.LocationsResponse{
			Locations:  []model.LocationResponse{},
			Pagination: common.NewPaginationResponse(filter.Page, filter.PageSize, 0),
		}, nil
	}

	// Fetch locations
	locations, err := s.locationRepo.FindAll(ctx, sellerID, filter)
	if err != nil {
		return nil, err
	}

	// Fetch all addresses for seller once
	addresses, _ := s.addressService.GetAddresses(ctx, sellerID)
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

	// Build paginated response
	pagination := common.NewPaginationResponse(filter.Page, filter.PageSize, totalCount)

	return &model.LocationsResponse{
		Locations:  locationResponses,
		Pagination: pagination,
	}, nil
}

// DeleteLocation soft deletes a location
func (s *LocationServiceImpl) DeleteLocation(
	ctx context.Context,
	id uint,
	sellerID uint,
) error {
	// Check if location exists and belongs to seller
	if err := s.locationRepo.Exists(ctx, id, sellerID); err != nil {
		return err
	}

	// TODO: Add validation to check if location has inventory
	// This should be implemented when inventory management is added

	return s.locationRepo.Delete(ctx, id)
}

// ============================================================================
// Private Helper Methods
// ============================================================================

// validateLocationCreate validates location creation request
func (s *LocationServiceImpl) validateLocationCreate(
	ctx context.Context,
	req model.LocationCreateRequest,
	sellerID uint,
) error {
	// Validate location type
	if err := validator.ValidateLocationType(req.Type); err != nil {
		return err
	}

	// Check for duplicate location name
	return s.validateNameUniqueness(ctx, req.Name, sellerID, nil)
}

// validateLocationUpdate validates location update request
func (s *LocationServiceImpl) validateLocationUpdate(
	ctx context.Context,
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
		return s.validateNameUniqueness(ctx, *req.Name, sellerID, &locationID)
	}

	return nil
}

// validateNameUniqueness checks if a location name is unique for a seller
func (s *LocationServiceImpl) validateNameUniqueness(
	ctx context.Context,
	name string,
	sellerID uint,
	excludeID *uint,
) error {
	existingLocation, err := s.locationRepo.FindByName(ctx, name, sellerID)
	if err != nil && err != invErrors.ErrLocationNotFound {
		return err
	}

	return validator.ValidateUniqueName(name, sellerID, existingLocation, excludeID)
}

// createAddress creates a new address via user service
// TODO [MICROSERVICE]: Replace with HTTP call to User Service Address API
func (s *LocationServiceImpl) createAddress(
	ctx context.Context,
	sellerID uint,
	addressReq model.AddressRequest,
	locType entity.LocationType,
) (*userModel.AddressResponse, error) {
	return s.addressService.AddAddress(
		ctx,
		sellerID,
		factory.BuildUserAddressReqToInventoryAddressReq(addressReq, locType),
	)
}

// handleAddressForUpdate updates or fetches address based on request
// TODO [MICROSERVICE]: Replace with HTTP call to User Service Address API
func (s *LocationServiceImpl) handleAddressForUpdate(
	ctx context.Context,
	addressReq *model.AddressUpdateRequest,
	addressID uint,
	sellerID uint,
	locType *entity.LocationType,
) (*userModel.AddressResponse, error) {
	if addressReq != nil || locType != nil {
		// Update address (include address type sync if location type changed)
		var addrReq model.AddressUpdateRequest
		if addressReq != nil {
			addrReq = *addressReq
		}
		return s.addressService.UpdateAddress(
			ctx,
			addressID,
			sellerID,
			factory.BuildInventoryUserUpdateReqToUserAddressUpdateReq(addrReq, locType),
		)
	}

	// Fetch existing address
	return s.addressService.GetAddressByID(ctx, addressID, sellerID)
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
