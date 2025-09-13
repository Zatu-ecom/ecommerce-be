package service

import (
	"ecommerce-be/user_management/entity"
	"ecommerce-be/user_management/model"
	"ecommerce-be/user_management/repositories"
)

// AddressService defines the interface for address-related business logic
type AddressService interface {
	GetAddresses(userID uint) ([]model.AddressResponse, error)
	AddAddress(userID uint, req model.AddressRequest) (*model.AddressResponse, error)
	UpdateAddress(
		addressID uint,
		userID uint,
		req model.AddressRequest,
	) (*model.AddressResponse, error)
	DeleteAddress(addressID uint, userID uint) error
	SetDefaultAddress(addressID uint, userID uint) (*model.AddressResponse, error)
}

// AddressServiceImpl implements the AddressService interface
type AddressServiceImpl struct {
	addressRepo repositories.AddressRepository
}

// NewAddressService creates a new instance of AddressService
func NewAddressService(addressRepo repositories.AddressRepository) AddressService {
	return &AddressServiceImpl{
		addressRepo: addressRepo,
	}
}

// GetAddresses retrieves all addresses for a user
func (s *AddressServiceImpl) GetAddresses(userID uint) ([]model.AddressResponse, error) {
	addresses, err := s.addressRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	// Transform addresses
	var addressResponses []model.AddressResponse
	for _, address := range addresses {
		addressResponses = append(addressResponses, model.AddressResponse{
			ID:        address.ID,
			Street:    address.Street,
			City:      address.City,
			State:     address.State,
			ZipCode:   address.ZipCode,
			Country:   address.Country,
			IsDefault: address.IsDefault,
		})
	}

	return addressResponses, nil
}

// AddAddress adds a new address for a user
func (s *AddressServiceImpl) AddAddress(
	userID uint,
	req model.AddressRequest,
) (*model.AddressResponse, error) {
	address := &entity.Address{
		UserID:    userID,
		Street:    req.Street,
		City:      req.City,
		State:     req.State,
		ZipCode:   req.ZipCode,
		Country:   req.Country,
		IsDefault: req.IsDefault,
	}

	if err := s.addressRepo.Create(address); err != nil {
		return nil, err
	}

	// Create response
	addressResponse := model.AddressResponse{
		ID:        address.ID,
		Street:    address.Street,
		City:      address.City,
		State:     address.State,
		ZipCode:   address.ZipCode,
		Country:   address.Country,
		IsDefault: address.IsDefault,
	}

	return &addressResponse, nil
}

// UpdateAddress updates an existing address
func (s *AddressServiceImpl) UpdateAddress(
	addressID uint,
	userID uint,
	req model.AddressRequest,
) (*model.AddressResponse, error) {
	// Find the address by ID and user ID
	address, err := s.addressRepo.FindByID(addressID, userID)
	if err != nil {
		return nil, err
	}

	// Update address fields
	address.Street = req.Street
	address.City = req.City
	address.State = req.State
	address.ZipCode = req.ZipCode
	address.Country = req.Country
	address.IsDefault = req.IsDefault

	// Save changes to database
	if err := s.addressRepo.Update(address); err != nil {
		return nil, err
	}

	// Create response
	addressResponse := model.AddressResponse{
		ID:        address.ID,
		Street:    address.Street,
		City:      address.City,
		State:     address.State,
		ZipCode:   address.ZipCode,
		Country:   address.Country,
		IsDefault: address.IsDefault,
	}

	return &addressResponse, nil
}

// DeleteAddress deletes an address
func (s *AddressServiceImpl) DeleteAddress(addressID uint, userID uint) error {
	return s.addressRepo.Delete(addressID, userID)
}

// SetDefaultAddress sets an address as the default address
func (s *AddressServiceImpl) SetDefaultAddress(
	addressID uint,
	userID uint,
) (*model.AddressResponse, error) {
	if err := s.addressRepo.SetDefault(addressID, userID); err != nil {
		return nil, err
	}

	address, err := s.addressRepo.FindByID(addressID, userID)
	if err != nil {
		return nil, err
	}

	// Create response
	addressResponse := model.AddressResponse{
		ID:        address.ID,
		Street:    address.Street,
		City:      address.City,
		State:     address.State,
		ZipCode:   address.ZipCode,
		Country:   address.Country,
		IsDefault: address.IsDefault,
	}

	return &addressResponse, nil
}
