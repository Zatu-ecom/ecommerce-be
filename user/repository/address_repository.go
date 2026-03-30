package repository

import (
	"context"
	"errors"

	"ecommerce-be/common/db"
	"ecommerce-be/user/entity"
	"ecommerce-be/user/utils/constant"

	"gorm.io/gorm"
)

// AddressRepository defines the interface for address data operations
type AddressRepository interface {
	// Address CRUD operations
	Create(ctx context.Context, address *entity.Address) error
	FindByID(ctx context.Context, id uint, userID uint) (*entity.Address, error)
	FindByUserID(ctx context.Context, userID uint) ([]entity.Address, error)
	Update(ctx context.Context, address *entity.Address) error
	Delete(ctx context.Context, id uint, userID uint) error
	SetDefault(ctx context.Context, id uint, userID uint) error
}

// AddressRepositoryImpl implements the AddressRepository interface
type AddressRepositoryImpl struct{}

// NewAddressRepository creates a new instance of AddressRepository
func NewAddressRepository(db *gorm.DB) AddressRepository {
	return &AddressRepositoryImpl{}
}

// Create creates a new address in the database
func (r *AddressRepositoryImpl) Create(ctx context.Context, address *entity.Address) error {
	// If this is the first address or marked as default, ensure it's set as default
	var count int64
	db.DB(ctx).Model(&entity.Address{}).Where("user_id = ?", address.UserID).Count(&count)

	if count == 0 || address.IsDefault {
		// If this is the first address or marked as default
		tx := db.DB(ctx).Begin()
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

	return db.DB(ctx).Create(address).Error
}

// FindByID finds an address by ID and user ID
func (r *AddressRepositoryImpl) FindByID(
	ctx context.Context,
	id uint,
	userID uint,
) (*entity.Address, error) {
	var address entity.Address
	result := db.DB(ctx).
		Preload("Country").
		Where("id = ? AND user_id = ?", id, userID).
		First(&address)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New(constant.ADDRESS_NOT_FOUND_MSG)
		}
		return nil, result.Error
	}
	return &address, nil
}

// FindByUserID finds all addresses for a user
func (r *AddressRepositoryImpl) FindByUserID(
	ctx context.Context,
	userID uint,
) ([]entity.Address, error) {
	var addresses []entity.Address
	result := db.DB(ctx).Preload("Country").Where("user_id = ?", userID).Find(&addresses)
	if result.Error != nil {
		return nil, result.Error
	}
	return addresses, nil
}

// Update updates an existing address
func (r *AddressRepositoryImpl) Update(ctx context.Context, address *entity.Address) error {
	tx := db.DB(ctx).Begin()

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
func (r *AddressRepositoryImpl) Delete(ctx context.Context, id uint, userID uint) error {
	// Check if it's the default address
	var address entity.Address
	if err := db.DB(ctx).Where("id = ? AND user_id = ?", id, userID).First(&address).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(constant.ADDRESS_NOT_FOUND_MSG)
		}
		return err
	}

	// Don't allow deletion if it's the only address and is default
	var count int64
	db.DB(ctx).Model(&entity.Address{}).Where("user_id = ?", userID).Count(&count)

	if address.IsDefault && count == 1 {
		return errors.New(constant.CANNOT_DELETE_ONLY_DEFAULT_ADDRESS_MSG)
	}

	tx := db.DB(ctx).Begin()

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
func (r *AddressRepositoryImpl) SetDefault(ctx context.Context, id uint, userID uint) error {
	tx := db.DB(ctx).Begin()

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
			return errors.New(constant.ADDRESS_NOT_FOUND_MSG)
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
