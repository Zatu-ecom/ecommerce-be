package repository

import (
	"context"
	"errors"

	"ecommerce-be/common/db"
	"ecommerce-be/user/entity"

	"gorm.io/gorm"
)

// CountryCurrencyRepository defines the interface for country-currency mapping operations
type CountryCurrencyRepository interface {
	// Query methods
	FindByCountryID(ctx context.Context, countryID uint) ([]entity.CountryCurrency, error)
	FindByCountryAndCurrency(ctx context.Context, countryID, currencyID uint) (*entity.CountryCurrency, error)

	// Mutation methods
	Create(ctx context.Context, mapping *entity.CountryCurrency) error
	Update(ctx context.Context, mapping *entity.CountryCurrency) error
	Delete(ctx context.Context, countryID, currencyID uint) error
	ClearPrimaryForCountry(ctx context.Context, countryID uint) error
}

// CountryCurrencyRepositoryImpl implements the CountryCurrencyRepository interface
type CountryCurrencyRepositoryImpl struct{}

// NewCountryCurrencyRepository creates a new instance of CountryCurrencyRepository
func NewCountryCurrencyRepository() CountryCurrencyRepository {
	return &CountryCurrencyRepositoryImpl{}
}

// FindByCountryID finds all currency mappings for a country
func (r *CountryCurrencyRepositoryImpl) FindByCountryID(
	ctx context.Context,
	countryID uint,
) ([]entity.CountryCurrency, error) {
	var mappings []entity.CountryCurrency
	err := db.DB(ctx).
		Where("country_id = ?", countryID).
		Preload("Currency").
		Order("is_primary DESC, id ASC").
		Find(&mappings).Error

	if err != nil {
		return nil, err
	}

	return mappings, nil
}

// FindByCountryAndCurrency finds a specific country-currency mapping
func (r *CountryCurrencyRepositoryImpl) FindByCountryAndCurrency(
	ctx context.Context,
	countryID, currencyID uint,
) (*entity.CountryCurrency, error) {
	var mapping entity.CountryCurrency
	result := db.DB(ctx).
		Where("country_id = ? AND currency_id = ?", countryID, currencyID).
		First(&mapping)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil // Not found is not an error for existence check
		}
		return nil, result.Error
	}

	return &mapping, nil
}

// Create creates a new country-currency mapping
func (r *CountryCurrencyRepositoryImpl) Create(
	ctx context.Context,
	mapping *entity.CountryCurrency,
) error {
	return db.DB(ctx).Create(mapping).Error
}

// Update updates an existing country-currency mapping
func (r *CountryCurrencyRepositoryImpl) Update(
	ctx context.Context,
	mapping *entity.CountryCurrency,
) error {
	return db.DB(ctx).Save(mapping).Error
}

// Delete removes a country-currency mapping
func (r *CountryCurrencyRepositoryImpl) Delete(
	ctx context.Context,
	countryID, currencyID uint,
) error {
	return db.DB(ctx).
		Where("country_id = ? AND currency_id = ?", countryID, currencyID).
		Delete(&entity.CountryCurrency{}).Error
}

// ClearPrimaryForCountry clears the primary flag for all currencies of a country
func (r *CountryCurrencyRepositoryImpl) ClearPrimaryForCountry(
	ctx context.Context,
	countryID uint,
) error {
	return db.DB(ctx).
		Model(&entity.CountryCurrency{}).
		Where("country_id = ?", countryID).
		Update("is_primary", false).Error
}
