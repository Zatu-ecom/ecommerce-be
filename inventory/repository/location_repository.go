package repository

import (
	"context"
	"errors"

	"ecommerce-be/common/db"
	"ecommerce-be/inventory/entity"
	invErrors "ecommerce-be/inventory/error"
	"ecommerce-be/inventory/model"

	"gorm.io/gorm"
)

// LocationRepository defines the interface for location-related database operations
type LocationRepository interface {
	Create(ctx context.Context, location *entity.Location) error
	Update(ctx context.Context, location *entity.Location) error
	FindByID(ctx context.Context, id uint, sellerID uint) (*entity.Location, error)
	FindByIDs(ctx context.Context, ids []uint, sellerID uint) ([]entity.Location, error)
	FindByName(ctx context.Context, name string, sellerID uint) (*entity.Location, error)
	FindAll(ctx context.Context, sellerID uint, filter model.LocationsFilter) ([]entity.Location, error)
	CountAll(ctx context.Context, sellerID uint, filter model.LocationsFilter) (int64, error)
	Delete(ctx context.Context, id uint) error
	Exists(ctx context.Context, id uint, sellerID uint) error
	FindActiveByPriority(ctx context.Context, sellerID uint) ([]entity.Location, error)
}

// LocationRepositoryImpl implements the LocationRepository interface
type LocationRepositoryImpl struct{}

// NewLocationRepository creates a new instance of LocationRepository
func NewLocationRepository() LocationRepository {
	return &LocationRepositoryImpl{}
}

// Create creates a new location
func (r *LocationRepositoryImpl) Create(ctx context.Context, location *entity.Location) error {
	return db.DB(ctx).Create(location).Error
}

// Update updates an existing location
func (r *LocationRepositoryImpl) Update(ctx context.Context, location *entity.Location) error {
	return db.DB(ctx).Model(location).
		Select("Name", "Priority", "IsActive", "Type", "UpdatedAt").
		Updates(location).Error
}

// FindByID finds a location by ID with eager loading of address
// Enforces seller isolation - only returns location if it belongs to the seller
func (r *LocationRepositoryImpl) FindByID(ctx context.Context, id uint, sellerID uint) (*entity.Location, error) {
	var location entity.Location
	result := db.DB(ctx).
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
func (r *LocationRepositoryImpl) FindByName(ctx context.Context, name string, sellerID uint) (*entity.Location, error) {
	var location entity.Location
	result := db.DB(ctx).Where("name = ? AND seller_id = ?", name, sellerID).First(&location)

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
	ctx context.Context,
	sellerID uint,
	filter model.LocationsFilter,
) ([]entity.Location, error) {
	var locations []entity.Location
	query := db.DB(ctx).Where("seller_id = ?", sellerID)

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
func (r *LocationRepositoryImpl) CountAll(ctx context.Context, sellerID uint, filter model.LocationsFilter) (int64, error) {
	query := db.DB(ctx).Model(&entity.Location{}).Where("seller_id = ?", sellerID)

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
func (r *LocationRepositoryImpl) Delete(ctx context.Context, id uint) error {
	return db.DB(ctx).Delete(&entity.Location{}, id).Error
}

// FindByIDs finds multiple locations by IDs for a seller (bulk query - avoids N+1)
func (r *LocationRepositoryImpl) FindByIDs(ctx context.Context, ids []uint, sellerID uint) ([]entity.Location, error) {
	var locations []entity.Location
	if len(ids) == 0 {
		return locations, nil
	}

	result := db.DB(ctx).Where("id IN ? AND seller_id = ?", ids, sellerID).Find(&locations)
	if result.Error != nil {
		return nil, result.Error
	}
	return locations, nil
}

// Exists checks if a location exists for a seller
func (r *LocationRepositoryImpl) Exists(ctx context.Context, id uint, sellerID uint) error {
	var count int64
	result := db.DB(ctx).Model(&entity.Location{}).
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

// FindActiveByPriority finds all active locations for a seller sorted by priority (DESC)
func (r *LocationRepositoryImpl) FindActiveByPriority(ctx context.Context, sellerID uint) ([]entity.Location, error) {
	var locations []entity.Location
	err := db.DB(ctx).Where("seller_id = ? AND is_active = ?", sellerID, true).
		Order("priority DESC").
		Find(&locations).Error
	return locations, err
}
