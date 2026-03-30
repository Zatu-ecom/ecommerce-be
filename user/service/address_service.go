package service

import (
	"context"

	"ecommerce-be/user/factory"
	"ecommerce-be/user/model"
	"ecommerce-be/user/repository"
)

// AddressService defines the interface for address-related business logic
type AddressService interface {
	GetAddresses(ctx context.Context, userID uint) ([]model.AddressResponse, error)
	GetAddressByID(ctx context.Context, addressID uint, userID uint) (*model.AddressResponse, error)
	AddAddress(
		ctx context.Context,
		userID uint,
		req model.AddressRequest,
	) (*model.AddressResponse, error)
	UpdateAddress(
		ctx context.Context,
		addressID uint,
		userID uint,
		req model.AddressUpdateRequest,
	) (*model.AddressResponse, error)
	DeleteAddress(ctx context.Context, addressID uint, userID uint) error
	SetDefaultAddress(
		ctx context.Context,
		addressID uint,
		userID uint,
	) (*model.AddressResponse, error)
}

// AddressServiceImpl implements the AddressService interface
type AddressServiceImpl struct {
	addressRepo repository.AddressRepository
}

// NewAddressService creates a new instance of AddressService
func NewAddressService(addressRepo repository.AddressRepository) AddressService {
	return &AddressServiceImpl{
		addressRepo: addressRepo,
	}
}

// GetAddresses retrieves all addresses for a user
func (s *AddressServiceImpl) GetAddresses(
	ctx context.Context,
	userID uint,
) ([]model.AddressResponse, error) {
	addresses, err := s.addressRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Transform addresses using factory
	var addressResponses []model.AddressResponse
	for _, address := range addresses {
		addressResponses = append(addressResponses, factory.BuildAddressResponse(&address))
	}

	return addressResponses, nil
}

// GetAddressByID retrieves a specific address by ID for a user
func (s *AddressServiceImpl) GetAddressByID(
	ctx context.Context,
	addressID uint,
	userID uint,
) (*model.AddressResponse, error) {
	address, err := s.addressRepo.FindByID(ctx, addressID, userID)
	if err != nil {
		return nil, err
	}

	// Build response using factory
	addressResponse := factory.BuildAddressResponse(address)

	return &addressResponse, nil
}

// AddAddress adds a new address for a user
func (s *AddressServiceImpl) AddAddress(
	ctx context.Context,
	userID uint,
	req model.AddressRequest,
) (*model.AddressResponse, error) {
	// Build address entity using factory
	address := factory.BuildAddressEntity(userID, req)

	if err := s.addressRepo.Create(ctx, address); err != nil {
		return nil, err
	}

	// Build response using factory
	addressResponse := factory.BuildAddressResponse(address)

	return &addressResponse, nil
}

// UpdateAddress updates an existing address
func (s *AddressServiceImpl) UpdateAddress(
	ctx context.Context,
	addressID uint,
	userID uint,
	req model.AddressUpdateRequest,
) (*model.AddressResponse, error) {
	// Find the address by ID and user ID
	address, err := s.addressRepo.FindByID(ctx, addressID, userID)
	if err != nil {
		return nil, err
	}

	// Update address fields using factory (only non-nil fields)
	factory.UpdateAddressEntity(address, req)

	// Save changes to database
	if err := s.addressRepo.Update(ctx, address); err != nil {
		return nil, err
	}

	// Build response using factory
	addressResponse := factory.BuildAddressResponse(address)

	return &addressResponse, nil
}

// DeleteAddress deletes an address
func (s *AddressServiceImpl) DeleteAddress(ctx context.Context, addressID uint, userID uint) error {
	return s.addressRepo.Delete(ctx, addressID, userID)
}

// SetDefaultAddress sets an address as the default address
func (s *AddressServiceImpl) SetDefaultAddress(
	ctx context.Context,
	addressID uint,
	userID uint,
) (*model.AddressResponse, error) {
	if err := s.addressRepo.SetDefault(ctx, addressID, userID); err != nil {
		return nil, err
	}

	address, err := s.addressRepo.FindByID(ctx, addressID, userID)
	if err != nil {
		return nil, err
	}

	// Build response using factory
	addressResponse := factory.BuildAddressResponse(address)

	return &addressResponse, nil
}
