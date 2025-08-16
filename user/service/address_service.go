package service

import (
	"datun.com/be/user/entity"
	"datun.com/be/user/model"
	"datun.com/be/user/repositories"
)

// AddressService defines the interface for address-related business logic
type AddressService interface {
	GetAddresses(userID uint) ([]entity.Address, error)
	AddAddress(userID uint, req model.AddressRequest) (*entity.Address, error)
	UpdateAddress(addressID uint, userID uint, req model.AddressRequest) (*entity.Address, error)
	DeleteAddress(addressID uint, userID uint) error
	SetDefaultAddress(addressID uint, userID uint) (*entity.Address, error)
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
func (s *AddressServiceImpl) GetAddresses(userID uint) ([]entity.Address, error) {
	return s.addressRepo.FindByUserID(userID)
}

// AddAddress adds a new address for a user
func (s *AddressServiceImpl) AddAddress(userID uint, req model.AddressRequest) (*entity.Address, error) {
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

	return address, nil
}

// UpdateAddress updates an existing address
func (s *AddressServiceImpl) UpdateAddress(addressID uint, userID uint, req model.AddressRequest) (*entity.Address, error) {
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

	return address, nil
}

// DeleteAddress deletes an address
func (s *AddressServiceImpl) DeleteAddress(addressID uint, userID uint) error {
	return s.addressRepo.Delete(addressID, userID)
}

// SetDefaultAddress sets an address as the default address
func (s *AddressServiceImpl) SetDefaultAddress(addressID uint, userID uint) (*entity.Address, error) {
	if err := s.addressRepo.SetDefault(addressID, userID); err != nil {
		return nil, err
	}

	return s.addressRepo.FindByID(addressID, userID)
}
