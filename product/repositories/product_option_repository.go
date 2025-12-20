package repositories

import (
	"context"
	"errors"

	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"

	"gorm.io/gorm"
)

// ProductOptionRepository defines the interface for product option-related database operations
type ProductOptionRepository interface {
	// Option operations
	CreateOption(ctx context.Context, option *entity.ProductOption) error
	BulkCreateOptions(ctx context.Context, options []*entity.ProductOption) error
	UpdateOption(ctx context.Context, option *entity.ProductOption) error
	FindOptionByID(ctx context.Context, id uint) (*entity.ProductOption, error)
	FindOptionByName(ctx context.Context, productID uint, optionName string) (*entity.ProductOption, error)
	FindOptionsByProductID(ctx context.Context, productID uint) ([]entity.ProductOption, error)
	FindOptionsByProductIDs(ctx context.Context, productIDs []uint) (map[uint][]entity.ProductOption, error)
	DeleteOption(ctx context.Context, id uint) error
	CheckOptionInUse(ctx context.Context, optionID uint) (bool, []uint, error)
	BulkUpdateOptions(ctx context.Context, options []*entity.ProductOption) error

	// Option value operations
	CreateOptionValue(ctx context.Context, value *entity.ProductOptionValue) error
	CreateOptionValues(ctx context.Context, values []entity.ProductOptionValue) error
	BulkCreateOptionValues(ctx context.Context, values []*entity.ProductOptionValue) error
	UpdateOptionValue(ctx context.Context, value *entity.ProductOptionValue) error
	FindOptionValueByID(ctx context.Context, id uint) (*entity.ProductOptionValue, error)
	FindOptionValueByValue(ctx context.Context, optionID uint, value string) (*entity.ProductOptionValue, error)
	FindOptionValuesByOptionID(ctx context.Context, optionID uint) ([]entity.ProductOptionValue, error)
	DeleteOptionValue(ctx context.Context, id uint) error
	CheckOptionValueInUse(ctx context.Context, valueID uint) (bool, []uint, error)
	BulkUpdateOptionValues(ctx context.Context, values []*entity.ProductOptionValue) error

	// Get options with variant counts
	GetProductOptionsWithVariantCounts(ctx context.Context, productID uint) ([]entity.ProductOption, map[uint]int, error)

	// Bulk deletion methods for product cleanup
	DeleteOptionValuesByOptionID(ctx context.Context, optionID uint) error
}

// ProductOptionRepositoryImpl implements the ProductOptionRepository interface
type ProductOptionRepositoryImpl struct {
	db *gorm.DB
}

// NewProductOptionRepository creates a new instance of ProductOptionRepository
func NewProductOptionRepository(db *gorm.DB) ProductOptionRepository {
	return &ProductOptionRepositoryImpl{db: db}
}

/***********************************************
 *    Option Operations                         *
 ***********************************************/

// CreateOption creates a new product option
func (r *ProductOptionRepositoryImpl) CreateOption(ctx context.Context, option *entity.ProductOption) error {
	return db.DB(ctx).Create(option).Error
}

// BulkCreateOptions creates multiple product options in a single INSERT query
// Uses RETURNING clause to get generated IDs efficiently
func (r *ProductOptionRepositoryImpl) BulkCreateOptions(ctx context.Context, options []*entity.ProductOption) error {
	if len(options) == 0 {
		return nil
	}
	// GORM's Create with slice automatically uses bulk insert and populates IDs
	return db.DB(ctx).Create(&options).Error
}

// UpdateOption updates an existing product option
func (r *ProductOptionRepositoryImpl) UpdateOption(ctx context.Context, option *entity.ProductOption) error {
	return db.DB(ctx).Save(option).Error
}

// FindOptionByID finds a product option by ID
func (r *ProductOptionRepositoryImpl) FindOptionByID(ctx context.Context, id uint) (*entity.ProductOption, error) {
	var option entity.ProductOption
	err := db.DB(ctx).Preload("Values").First(&option, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, prodErrors.ErrProductOptionNotFound
		}
		return nil, err
	}
	return &option, nil
}

// FindOptionByName retrieves a product option by name for a specific product
func (r *ProductOptionRepositoryImpl) FindOptionByName(
	ctx context.Context,
	productID uint,
	optionName string,
) (*entity.ProductOption, error) {
	var option entity.ProductOption
	result := db.DB(ctx).Where("product_id = ? AND name = ?", productID, optionName).First(&option)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, prodErrors.ErrProductOptionNotFound.WithMessagef(
				"Product option not found: %s",
				optionName,
			)
		}
		return nil, result.Error
	}

	return &option, nil
}

// FindOptionsByProductID finds all options for a product
func (r *ProductOptionRepositoryImpl) FindOptionsByProductID(
	ctx context.Context,
	productID uint,
) ([]entity.ProductOption, error) {
	var options []entity.ProductOption
	err := db.DB(ctx).Where("product_id = ?", productID).
		Preload("Values").
		Order("position ASC").
		Find(&options).Error
	if err != nil {
		return nil, err
	}
	return options, nil
}

// FindOptionsByProductIDs finds all options for multiple products in batch
// Returns a map of productID -> []ProductOption to prevent N+1 queries
func (r *ProductOptionRepositoryImpl) FindOptionsByProductIDs(
	ctx context.Context,
	productIDs []uint,
) (map[uint][]entity.ProductOption, error) {
	if len(productIDs) == 0 {
		return make(map[uint][]entity.ProductOption), nil
	}

	var options []entity.ProductOption
	err := db.DB(ctx).Where("product_id IN ?", productIDs).
		Preload("Values", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Order("product_id ASC, position ASC").
		Find(&options).Error
	if err != nil {
		return nil, err
	}

	// Group options by product ID
	productOptionsMap := make(map[uint][]entity.ProductOption)
	for _, option := range options {
		productOptionsMap[option.ProductID] = append(
			productOptionsMap[option.ProductID],
			option,
		)
	}

	return productOptionsMap, nil
}

// DeleteOption deletes a product option
func (r *ProductOptionRepositoryImpl) DeleteOption(ctx context.Context, id uint) error {
	return db.DB(ctx).Delete(&entity.ProductOption{}, id).Error
}

// CheckOptionInUse checks if an option is being used by any variants
func (r *ProductOptionRepositoryImpl) CheckOptionInUse(ctx context.Context, optionID uint) (bool, []uint, error) {
	var variantIDs []uint
	err := db.DB(ctx).Model(&entity.VariantOptionValue{}).
		Where("option_id = ?", optionID).
		Distinct("variant_id").
		Pluck("variant_id", &variantIDs).Error
	if err != nil {
		return false, nil, err
	}

	return len(variantIDs) > 0, variantIDs, nil
}

/***********************************************
 *    Option Value Operations                   *
 ***********************************************/

// CreateOptionValue creates a new product option value
func (r *ProductOptionRepositoryImpl) CreateOptionValue(ctx context.Context, value *entity.ProductOptionValue) error {
	return db.DB(ctx).Create(value).Error
}

// CreateOptionValues creates multiple product option values
func (r *ProductOptionRepositoryImpl) CreateOptionValues(ctx context.Context, values []entity.ProductOptionValue) error {
	if len(values) == 0 {
		return nil
	}
	return db.DB(ctx).Create(&values).Error
}

// BulkCreateOptionValues creates multiple product option values in a single INSERT query
// Uses RETURNING clause to get generated IDs efficiently
func (r *ProductOptionRepositoryImpl) BulkCreateOptionValues(
	ctx context.Context,
	values []*entity.ProductOptionValue,
) error {
	if len(values) == 0 {
		return nil
	}
	// GORM's Create with slice automatically uses bulk insert and populates IDs
	return db.DB(ctx).Create(&values).Error
}

// UpdateOptionValue updates an existing product option value
func (r *ProductOptionRepositoryImpl) UpdateOptionValue(ctx context.Context, value *entity.ProductOptionValue) error {
	return db.DB(ctx).Save(value).Error
}

// FindOptionValueByID finds a product option value by ID
func (r *ProductOptionRepositoryImpl) FindOptionValueByID(
	ctx context.Context,
	id uint,
) (*entity.ProductOptionValue, error) {
	var value entity.ProductOptionValue
	err := db.DB(ctx).First(&value, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, prodErrors.ErrProductOptionValueNotFound
		}
		return nil, err
	}
	return &value, nil
}

// FindOptionValueByValue retrieves an option value by its value string
func (r *ProductOptionRepositoryImpl) FindOptionValueByValue(
	ctx context.Context,
	optionID uint,
	value string,
) (*entity.ProductOptionValue, error) {
	var optionValue entity.ProductOptionValue
	result := db.DB(ctx).Where("option_id = ? AND value = ?", optionID, value).First(&optionValue)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, prodErrors.ErrProductOptionValueNotFound.WithMessagef(
				"Product option value not found: %s",
				value,
			)
		}
		return nil, result.Error
	}

	return &optionValue, nil
}

// FindOptionValuesByOptionID finds all values for an option
func (r *ProductOptionRepositoryImpl) FindOptionValuesByOptionID(
	ctx context.Context,
	optionID uint,
) ([]entity.ProductOptionValue, error) {
	var values []entity.ProductOptionValue
	err := db.DB(ctx).Where("option_id = ?", optionID).
		Order("position ASC").
		Find(&values).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, prodErrors.ErrProductOptionValueNotFound
		}
		return nil, err
	}
	return values, nil
}

// DeleteOptionValue deletes a product option value
func (r *ProductOptionRepositoryImpl) DeleteOptionValue(ctx context.Context, id uint) error {
	return db.DB(ctx).Delete(&entity.ProductOptionValue{}, id).Error
}

// CheckOptionValueInUse checks if an option value is being used by any variants
func (r *ProductOptionRepositoryImpl) CheckOptionValueInUse(ctx context.Context, valueID uint) (bool, []uint, error) {
	var variantIDs []uint
	err := db.DB(ctx).Model(&entity.VariantOptionValue{}).
		Where("option_value_id = ?", valueID).
		Distinct("variant_id").
		Pluck("variant_id", &variantIDs).Error
	if err != nil {
		return false, nil, err
	}

	return len(variantIDs) > 0, variantIDs, nil
}

/***********************************************
 *    Get Options with Variant Counts          *
 ***********************************************/

// GetProductOptionsWithVariantCounts retrieves detailed options with variant counts for each value
func (r *ProductOptionRepositoryImpl) GetProductOptionsWithVariantCounts(
	ctx context.Context,
	productID uint,
) ([]entity.ProductOption, map[uint]int, error) {
	// Get all options for the product with their values preloaded
	var options []entity.ProductOption
	if err := db.DB(ctx).
		Where("product_id = ?", productID).
		Order("position ASC").
		Preload("Values", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Find(&options).Error; err != nil {
		return nil, nil, err
	}

	// Count variants for each option value
	variantCounts := make(map[uint]int)

	for _, option := range options {
		for _, value := range option.Values {
			// Count how many variants use this option value
			var count int64
			db.DB(ctx).Model(&entity.VariantOptionValue{}).
				Where("option_value_id = ?", value.ID).
				Count(&count)

			variantCounts[value.ID] = int(count)
		}
	}

	return options, variantCounts, nil
}

/***********************************************
 *    Bulk Deletion Methods for Product Cleanup
 ***********************************************/

// DeleteOptionValuesByOptionID deletes all option values for a given option
func (r *ProductOptionRepositoryImpl) DeleteOptionValuesByOptionID(ctx context.Context, optionID uint) error {
	return db.DB(ctx).Where("option_id = ?", optionID).Delete(&entity.ProductOptionValue{}).Error
}

/***********************************************
 *    Bulk Update Methods                      *
 ***********************************************/

// BulkUpdateOptions updates multiple options in a transaction
func (r *ProductOptionRepositoryImpl) BulkUpdateOptions(ctx context.Context, options []*entity.ProductOption) error {
	if len(options) == 0 {
		return nil
	}

	return db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		for _, option := range options {
			if err := tx.Model(&entity.ProductOption{}).
				Where("id = ?", option.ID).
				Updates(map[string]interface{}{
					"display_name": option.DisplayName,
					"position":     option.Position,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// BulkUpdateOptionValues updates multiple option values in a transaction
func (r *ProductOptionRepositoryImpl) BulkUpdateOptionValues(
	ctx context.Context,
	values []*entity.ProductOptionValue,
) error {
	if len(values) == 0 {
		return nil
	}

	return db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		for _, value := range values {
			updates := map[string]interface{}{}

			// Only update non-empty fields
			if value.DisplayName != "" {
				updates["display_name"] = value.DisplayName
			}
			if value.ColorCode != "" {
				updates["color_code"] = value.ColorCode
			}
			// Always update position
			updates["position"] = value.Position

			if len(updates) > 0 {
				if err := tx.Model(&entity.ProductOptionValue{}).
					Where("id = ?", value.ID).
					Updates(updates).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
