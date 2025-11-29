package repository

import (
	"errors"

	"ecommerce-be/inventory/entity"
	invErrors "ecommerce-be/inventory/error"

	"gorm.io/gorm"
)

// LocationRepository defines the interface for location-related database operations
type LocationRepository interface {
	Create(location *entity.Location) error
	Update(location *entity.Location) error
	FindByID(id uint, sellerID uint) (*entity.Location, error)
	FindByName(name string, sellerID uint) (*entity.Location, error)
	FindAll(sellerID uint, isActive *bool) ([]entity.Location, error)
	Delete(id uint) error
	Exists(id uint, sellerID uint) error
}

// LocationRepositoryImpl implements the LocationRepository interface
type LocationRepositoryImpl struct {
	db *gorm.DB
}

// NewLocationRepository creates a new instance of LocationRepository
func NewLocationRepository(db *gorm.DB) LocationRepository {
	return &LocationRepositoryImpl{db: db}
}

// Create creates a new location
func (r *LocationRepositoryImpl) Create(location *entity.Location) error {
	return r.db.Create(location).Error
}

// Update updates an existing location
func (r *LocationRepositoryImpl) Update(location *entity.Location) error {
	return r.db.Model(location).
		Select("Name", "Priority", "IsActive", "Type", "UpdatedAt").
		Updates(location).Error
}

// FindByID finds a location by ID with eager loading of address
// Enforces seller isolation - only returns location if it belongs to the seller
func (r *LocationRepositoryImpl) FindByID(id uint, sellerID uint) (*entity.Location, error) {
	var location entity.Location
	result := r.db.
		Where("id = ? AND seller_id = ?", id, sellerID).
		First(&location)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, invErrors.ErrLocationNotFound
		}
		return nil, result.Error
	}
	return &location, nil
}

// FindByName finds a location by name for a specific seller
// Used for duplicate name validation
func (r *LocationRepositoryImpl) FindByName(name string, sellerID uint) (*entity.Location, error) {
	var location entity.Location
	result := r.db.Where("name = ? AND seller_id = ?", name, sellerID).First(&location)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, invErrors.ErrLocationNotFound
		}
		return nil, result.Error
	}
	return &location, nil
}

// FindAll finds all locations for a seller with optional active filter
func (r *LocationRepositoryImpl) FindAll(sellerID uint, isActive *bool) ([]entity.Location, error) {
	var locations []entity.Location
	query := r.db.Where("seller_id = ?", sellerID)

	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	result := query.Order("priority DESC, name ASC").Find(&locations)
	if result.Error != nil {
		return nil, result.Error
	}
	return locations, nil
}

// Delete soft deletes a location
func (r *LocationRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&entity.Location{}, id).Error
}

// Exists checks if a location exists for a seller
func (r *LocationRepositoryImpl) Exists(id uint, sellerID uint) error {
	var count int64
	result := r.db.Model(&entity.Location{}).
		Where("id = ? AND seller_id = ?", id, sellerID).
		Count(&count)

	if result.Error != nil {
		return result.Error
	}
	if count == 0 {
		return invErrors.ErrLocationNotFound
	}
	return nil
}
