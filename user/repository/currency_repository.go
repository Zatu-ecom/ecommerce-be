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

// CurrencyRepository defines the interface for currency-related database operations
type CurrencyRepository interface {
	// Query methods
	FindAll(
		ctx context.Context,
		filter model.CurrencyQueryParams,
		includeInactive bool,
	) ([]entity.Currency, error)
	CountAll(
		ctx context.Context,
		filter model.CurrencyQueryParams,
		includeInactive bool,
	) (int64, error)
	FindByID(ctx context.Context, id uint) (*entity.Currency, error)
	FindByIDWithCountries(ctx context.Context, id uint) (*entity.Currency, error)
	FindByCode(ctx context.Context, code string) (*entity.Currency, error)

	// Mutation methods
	Create(ctx context.Context, currency *entity.Currency) error
	Update(ctx context.Context, currency *entity.Currency) error
	Delete(ctx context.Context, id uint) error
}

// CurrencyRepositoryImpl implements the CurrencyRepository interface
type CurrencyRepositoryImpl struct{}

// NewCurrencyRepository creates a new instance of CurrencyRepository
func NewCurrencyRepository() CurrencyRepository {
	return &CurrencyRepositoryImpl{}
}

// FindAll finds all currencies with optional filters
func (r *CurrencyRepositoryImpl) FindAll(
	ctx context.Context,
	filter model.CurrencyQueryParams,
	includeInactive bool,
) ([]entity.Currency, error) {
	var currencies []entity.Currency
	query := db.DB(ctx).Model(&entity.Currency{})

	// Apply active filter (public API only shows active)
	if !includeInactive {
		query = query.Where("is_active = ?", true)
	} else if filter.IsActive != nil {
		// Admin can filter by active status
		query = query.Where("is_active = ?", *filter.IsActive)
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

	if err := query.Find(&currencies).Error; err != nil {
		return nil, err
	}

	return currencies, nil
}

// CountAll counts all currencies with optional filters
func (r *CurrencyRepositoryImpl) CountAll(
	ctx context.Context,
	filter model.CurrencyQueryParams,
	includeInactive bool,
) (int64, error) {
	var count int64
	query := db.DB(ctx).Model(&entity.Currency{})

	// Apply active filter
	if !includeInactive {
		query = query.Where("is_active = ?", true)
	} else if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

// FindByID finds a currency by ID
func (r *CurrencyRepositoryImpl) FindByID(ctx context.Context, id uint) (*entity.Currency, error) {
	var currency entity.Currency
	result := db.DB(ctx).First(&currency, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, userErrors.ErrCurrencyNotFound
		}
		return nil, result.Error
	}

	return &currency, nil
}

// FindByIDWithCountries finds a currency by ID with preloaded countries
func (r *CurrencyRepositoryImpl) FindByIDWithCountries(
	ctx context.Context,
	id uint,
) (*entity.Currency, error) {
	var currency entity.Currency
	result := db.DB(ctx).
		Preload("Countries", "is_active = ?", true).
		First(&currency, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, userErrors.ErrCurrencyNotFound
		}
		return nil, result.Error
	}

	return &currency, nil
}

// FindByCode finds a currency by its ISO code
func (r *CurrencyRepositoryImpl) FindByCode(
	ctx context.Context,
	code string,
) (*entity.Currency, error) {
	var currency entity.Currency
	result := db.DB(ctx).Where("code = ?", code).First(&currency)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil // Not found is not an error for duplicate check
		}
		return nil, result.Error
	}

	return &currency, nil
}

// Create creates a new currency
func (r *CurrencyRepositoryImpl) Create(ctx context.Context, currency *entity.Currency) error {
	return db.DB(ctx).Create(currency).Error
}

// Update updates an existing currency
func (r *CurrencyRepositoryImpl) Update(ctx context.Context, currency *entity.Currency) error {
	return db.DB(ctx).Save(currency).Error
}

// Delete soft deletes a currency by ID
func (r *CurrencyRepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := db.DB(ctx).Delete(&entity.Currency{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return userErrors.ErrCurrencyNotFound
	}
	return nil
}
