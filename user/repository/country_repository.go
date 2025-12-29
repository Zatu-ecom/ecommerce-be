package repository

import (
	"context"
	"errors"

	"ecommerce-be/common/db"
	"ecommerce-be/user/entity"
	userErrors "ecommerce-be/user/error"
	"ecommerce-be/user/model"

	"gorm.io/gorm"
)

// CountryRepository defines the interface for country-related database operations
type CountryRepository interface {
	// Query methods
	FindAll(
		ctx context.Context,
		filter model.CountryQueryParams,
		includeInactive bool,
	) ([]entity.Country, error)
	CountAll(
		ctx context.Context,
		filter model.CountryQueryParams,
		includeInactive bool,
	) (int64, error)
	FindByID(ctx context.Context, id uint) (*entity.Country, error)
	FindByIDWithCurrencies(ctx context.Context, id uint) (*entity.Country, error)
	FindByCode(ctx context.Context, code string) (*entity.Country, error)

	// Mutation methods
	Create(ctx context.Context, country *entity.Country) error
	Update(ctx context.Context, country *entity.Country) error
	Delete(ctx context.Context, id uint) error
}

// CountryRepositoryImpl implements the CountryRepository interface
type CountryRepositoryImpl struct{}

// NewCountryRepository creates a new instance of CountryRepository
func NewCountryRepository() CountryRepository {
	return &CountryRepositoryImpl{}
}

// FindAll finds all countries with optional filters
func (r *CountryRepositoryImpl) FindAll(
	ctx context.Context,
	filter model.CountryQueryParams,
	includeInactive bool,
) ([]entity.Country, error) {
	var countries []entity.Country
	query := db.DB(ctx).Model(&entity.Country{})

	// Apply active filter (public API only shows active)
	if !includeInactive {
		query = query.Where("is_active = ?", true)
	} else if filter.IsActive != nil {
		// Admin can filter by active status
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	// Apply search filter (by name or code)
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where(
			"LOWER(name) LIKE LOWER(?) OR LOWER(code) LIKE LOWER(?) OR LOWER(code_alpha3) LIKE LOWER(?)",
			searchPattern,
			searchPattern,
			searchPattern,
		)
	}

	// Apply region filter
	if filter.Region != "" {
		query = query.Where("region = ?", filter.Region)
	}

	// Apply sorting (by name ascending)
	query = query.Order("name ASC")

	// Apply pagination
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Page > 0 && filter.Limit > 0 {
		offset := (filter.Page - 1) * filter.Limit
		query = query.Offset(offset)
	}

	if err := query.Find(&countries).Error; err != nil {
		return nil, err
	}

	return countries, nil
}

// CountAll counts all countries with optional filters
func (r *CountryRepositoryImpl) CountAll(
	ctx context.Context,
	filter model.CountryQueryParams,
	includeInactive bool,
) (int64, error) {
	var count int64
	query := db.DB(ctx).Model(&entity.Country{})

	// Apply active filter
	if !includeInactive {
		query = query.Where("is_active = ?", true)
	} else if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	// Apply search filter (by name or code)
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where(
			"LOWER(name) LIKE LOWER(?) OR LOWER(code) LIKE LOWER(?) OR LOWER(code_alpha3) LIKE LOWER(?)",
			searchPattern,
			searchPattern,
			searchPattern,
		)
	}

	// Apply region filter
	if filter.Region != "" {
		query = query.Where("region = ?", filter.Region)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

// FindByID finds a country by ID
func (r *CountryRepositoryImpl) FindByID(ctx context.Context, id uint) (*entity.Country, error) {
	var country entity.Country
	result := db.DB(ctx).First(&country, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, userErrors.ErrCountryNotFound
		}
		return nil, result.Error
	}

	return &country, nil
}

// FindByIDWithCurrencies finds a country by ID with preloaded currencies
func (r *CountryRepositoryImpl) FindByIDWithCurrencies(
	ctx context.Context,
	id uint,
) (*entity.Country, error) {
	var country entity.Country
	result := db.DB(ctx).
		Preload("Currencies", "is_active = ?", true).
		First(&country, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, userErrors.ErrCountryNotFound
		}
		return nil, result.Error
	}

	return &country, nil
}

// FindByCode finds a country by its ISO code (alpha-2)
func (r *CountryRepositoryImpl) FindByCode(
	ctx context.Context,
	code string,
) (*entity.Country, error) {
	var country entity.Country
	result := db.DB(ctx).Where("code = ?", code).First(&country)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil // Not found is not an error for duplicate check
		}
		return nil, result.Error
	}

	return &country, nil
}

// Create creates a new country
func (r *CountryRepositoryImpl) Create(ctx context.Context, country *entity.Country) error {
	return db.DB(ctx).Create(country).Error
}

// Update updates an existing country
func (r *CountryRepositoryImpl) Update(ctx context.Context, country *entity.Country) error {
	return db.DB(ctx).Save(country).Error
}

// Delete soft deletes a country by ID
func (r *CountryRepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := db.DB(ctx).Delete(&entity.Country{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return userErrors.ErrCountryNotFound
	}
	return nil
}
