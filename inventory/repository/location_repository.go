package repository

import (
	"errors"

	"ecommerce-be/inventory/entity"
	invErrors "ecommerce-be/inventory/error"
	"ecommerce-be/inventory/model"

	"gorm.io/gorm"
)

// LocationRepository defines the interface for location-related database operations
type LocationRepository interface {
	Create(location *entity.Location) error
	Update(location *entity.Location) error
	FindByID(id uint, sellerID uint) (*entity.Location, error)
	FindByIDs(ids []uint, sellerID uint) ([]entity.Location, error)
	FindByName(name string, sellerID uint) (*entity.Location, error)
	FindAll(sellerID uint, filter model.LocationsFilter) ([]entity.Location, error)
	CountAll(sellerID uint, filter model.LocationsFilter) (int64, error)
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

// FindAll finds all locations for a seller with optional filters
func (r *LocationRepositoryImpl) FindAll(
	sellerID uint,
	filter model.LocationsFilter,
) ([]entity.Location, error) {
	var locations []entity.Location
	query := r.db.Where("seller_id = ?", sellerID)

	// Apply isActive filter
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	// Apply location type filter
	if len(filter.LocationTypes) > 0 {
		query = query.Where("type IN ?", filter.LocationTypes)
	}

	// Apply sorting (default to priority DESC if not specified)
	sortField := filter.SortBy
	sortOrder := filter.SortOrder

	if sortField == "" {
		sortField = "priority"
	}
	if sortOrder == "" {
		sortOrder = "DESC"
	}

	orderClause := sortField + " " + sortOrder
	if sortField != "priority" {
		orderClause += ", priority DESC" // Secondary sort by priority
	}

	result := query.Order(orderClause).Find(&locations)
	if result.Error != nil {
		return nil, result.Error
	}
	return locations, nil
}

// CountAll counts total locations for a seller with filters applied
func (r *LocationRepositoryImpl) CountAll(sellerID uint, filter model.LocationsFilter) (int64, error) {
	query := r.db.Model(&entity.Location{}).Where("seller_id = ?", sellerID)

	// Apply isActive filter if specified
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	// Apply location type filter if specified
	if len(filter.LocationTypes) > 0 {
		query = query.Where("type IN ?", filter.LocationTypes)
	}

	var count int64
	err := query.Count(&count).Error
	return count, err
}

// Delete soft deletes a location
func (r *LocationRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&entity.Location{}, id).Error
}

// FindByIDs finds multiple locations by IDs for a seller (bulk query - avoids N+1)
func (r *LocationRepositoryImpl) FindByIDs(ids []uint, sellerID uint) ([]entity.Location, error) {
	var locations []entity.Location
	if len(ids) == 0 {
		return locations, nil
	}

	result := r.db.Where("id IN ? AND seller_id = ?", ids, sellerID).Find(&locations)
	if result.Error != nil {
		return nil, result.Error
	}
	return locations, nil
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
