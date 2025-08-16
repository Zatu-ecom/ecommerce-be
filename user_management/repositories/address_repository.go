package repositories

import (
	"errors"

	"datun.com/be/user_management/entity"
	"datun.com/be/user_management/utils"
	"gorm.io/gorm"
)

// AddressRepository defines the interface for address data operations
type AddressRepository interface {
	// Address CRUD operations
	Create(address *entity.Address) error
	FindByID(id uint, userID uint) (*entity.Address, error)
	FindByUserID(userID uint) ([]entity.Address, error)
	Update(address *entity.Address) error
	Delete(id uint, userID uint) error
	SetDefault(id uint, userID uint) error
}

// AddressRepositoryImpl implements the AddressRepository interface
type AddressRepositoryImpl struct {
	db *gorm.DB
}

// NewAddressRepository creates a new instance of AddressRepository
func NewAddressRepository(db *gorm.DB) AddressRepository {
	return &AddressRepositoryImpl{db: db}
}

// Create creates a new address in the database
func (r *AddressRepositoryImpl) Create(address *entity.Address) error {
	// If this is the first address or marked as default, ensure it's set as default
	var count int64
	r.db.Model(&entity.Address{}).Where("user_id = ?", address.UserID).Count(&count)

	if count == 0 || address.IsDefault {
		// If this is the first address or marked as default
		tx := r.db.Begin()
		// Reset all existing addresses to non-default if this one is default
		if address.IsDefault {
			if err := tx.Model(&entity.Address{}).Where("user_id = ?", address.UserID).Update("is_default", false).Error; err != nil {
				tx.Rollback()
				return err
			}
		} else if count == 0 {
			// If this is the first address, make it default
			address.IsDefault = true
		}

		if err := tx.Create(address).Error; err != nil {
			tx.Rollback()
			return err
		}

		return tx.Commit().Error
	}

	return r.db.Create(address).Error
}

// FindByID finds an address by ID and user ID
func (r *AddressRepositoryImpl) FindByID(id uint, userID uint) (*entity.Address, error) {
	var address entity.Address
	result := r.db.Where("id = ? AND user_id = ?", id, userID).First(&address)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New(utils.AddressNotFoundMsg)
		}
		return nil, result.Error
	}
	return &address, nil
}

// FindByUserID finds all addresses for a user
func (r *AddressRepositoryImpl) FindByUserID(userID uint) ([]entity.Address, error) {
	var addresses []entity.Address
	result := r.db.Where("user_id = ?", userID).Find(&addresses)
	if result.Error != nil {
		return nil, result.Error
	}
	return addresses, nil
}

// Update updates an existing address
func (r *AddressRepositoryImpl) Update(address *entity.Address) error {
	tx := r.db.Begin()

	// If setting as default, reset other addresses
	if address.IsDefault {
		if err := tx.Model(&entity.Address{}).
			Where("user_id = ? AND id != ?", address.UserID, address.ID).
			Update("is_default", false).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Save(address).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// Delete deletes an address by ID and user ID
func (r *AddressRepositoryImpl) Delete(id uint, userID uint) error {
	// Check if it's the default address
	var address entity.Address
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&address).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(utils.AddressNotFoundMsg)
		}
		return err
	}

	// Don't allow deletion if it's the only address and is default
	var count int64
	r.db.Model(&entity.Address{}).Where("user_id = ?", userID).Count(&count)

	if address.IsDefault && count == 1 {
		return errors.New(utils.CannotDeleteOnlyDefaultAddressMsg)
	}

	tx := r.db.Begin()

	if err := tx.Where("id = ? AND user_id = ?", id, userID).Delete(&entity.Address{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// If we deleted the default address and there are other addresses, set the first one as default
	if address.IsDefault && count > 1 {
		var newDefaultAddress entity.Address
		if err := tx.Where("user_id = ?", userID).First(&newDefaultAddress).Error; err == nil {
			newDefaultAddress.IsDefault = true
			if err := tx.Save(&newDefaultAddress).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit().Error
}

// SetDefault sets an address as the default address
func (r *AddressRepositoryImpl) SetDefault(id uint, userID uint) error {
	tx := r.db.Begin()

	// Reset all addresses to non-default
	if err := tx.Model(&entity.Address{}).Where("user_id = ?", userID).Update("is_default", false).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Set the specified address as default
	address := entity.Address{}
	if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&address).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(utils.AddressNotFoundMsg)
		}
		return err
	}

	address.IsDefault = true
	if err := tx.Save(&address).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
