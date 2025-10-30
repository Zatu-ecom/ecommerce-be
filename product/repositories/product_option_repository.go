package repositories

import (
	"ecommerce-be/product/entity"

	"gorm.io/gorm"
)

// ProductOptionRepository defines the interface for product option-related database operations
type ProductOptionRepository interface {
	// Option operations
	CreateOption(option *entity.ProductOption) error
	UpdateOption(option *entity.ProductOption) error
	FindOptionByID(id uint) (*entity.ProductOption, error)
	FindOptionsByProductID(productID uint) ([]entity.ProductOption, error)
	DeleteOption(id uint) error
	CheckOptionInUse(optionID uint) (bool, []uint, error)
	BulkUpdateOptions(options []*entity.ProductOption) error

	// Option value operations
	CreateOptionValue(value *entity.ProductOptionValue) error
	CreateOptionValues(values []entity.ProductOptionValue) error
	UpdateOptionValue(value *entity.ProductOptionValue) error
	FindOptionValueByID(id uint) (*entity.ProductOptionValue, error)
	FindOptionValuesByOptionID(optionID uint) ([]entity.ProductOptionValue, error)
	DeleteOptionValue(id uint) error
	CheckOptionValueInUse(valueID uint) (bool, []uint, error)
	BulkUpdateOptionValues(values []*entity.ProductOptionValue) error

	// Get options with variant counts
	GetProductOptionsWithVariantCounts(productID uint) ([]entity.ProductOption, map[uint]int, error)

	// Bulk deletion methods for product cleanup
	DeleteOptionValuesByOptionID(optionID uint) error
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
func (r *ProductOptionRepositoryImpl) CreateOption(option *entity.ProductOption) error {
	return r.db.Create(option).Error
}

// UpdateOption updates an existing product option
func (r *ProductOptionRepositoryImpl) UpdateOption(option *entity.ProductOption) error {
	return r.db.Save(option).Error
}

// FindOptionByID finds a product option by ID
func (r *ProductOptionRepositoryImpl) FindOptionByID(id uint) (*entity.ProductOption, error) {
	var option entity.ProductOption
	err := r.db.Preload("Values").First(&option, id).Error
	if err != nil {
		return nil, err
	}
	return &option, nil
}

// FindOptionsByProductID finds all options for a product
func (r *ProductOptionRepositoryImpl) FindOptionsByProductID(
	productID uint,
) ([]entity.ProductOption, error) {
	var options []entity.ProductOption
	err := r.db.Where("product_id = ?", productID).
		Preload("Values").
		Order("position ASC").
		Find(&options).Error
	if err != nil {
		return nil, err
	}
	return options, nil
}

// DeleteOption deletes a product option
func (r *ProductOptionRepositoryImpl) DeleteOption(id uint) error {
	return r.db.Delete(&entity.ProductOption{}, id).Error
}

// CheckOptionInUse checks if an option is being used by any variants
func (r *ProductOptionRepositoryImpl) CheckOptionInUse(optionID uint) (bool, []uint, error) {
	var variantIDs []uint
	err := r.db.Model(&entity.VariantOptionValue{}).
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
func (r *ProductOptionRepositoryImpl) CreateOptionValue(value *entity.ProductOptionValue) error {
	return r.db.Create(value).Error
}

// CreateOptionValues creates multiple product option values
func (r *ProductOptionRepositoryImpl) CreateOptionValues(values []entity.ProductOptionValue) error {
	if len(values) == 0 {
		return nil
	}
	return r.db.Create(&values).Error
}

// UpdateOptionValue updates an existing product option value
func (r *ProductOptionRepositoryImpl) UpdateOptionValue(value *entity.ProductOptionValue) error {
	return r.db.Save(value).Error
}

// FindOptionValueByID finds a product option value by ID
func (r *ProductOptionRepositoryImpl) FindOptionValueByID(
	id uint,
) (*entity.ProductOptionValue, error) {
	var value entity.ProductOptionValue
	err := r.db.First(&value, id).Error
	if err != nil {
		return nil, err
	}
	return &value, nil
}

// FindOptionValuesByOptionID finds all values for an option
func (r *ProductOptionRepositoryImpl) FindOptionValuesByOptionID(
	optionID uint,
) ([]entity.ProductOptionValue, error) {
	var values []entity.ProductOptionValue
	err := r.db.Where("option_id = ?", optionID).
		Order("position ASC").
		Find(&values).Error
	if err != nil {
		return nil, err
	}
	return values, nil
}

// DeleteOptionValue deletes a product option value
func (r *ProductOptionRepositoryImpl) DeleteOptionValue(id uint) error {
	return r.db.Delete(&entity.ProductOptionValue{}, id).Error
}

// CheckOptionValueInUse checks if an option value is being used by any variants
func (r *ProductOptionRepositoryImpl) CheckOptionValueInUse(valueID uint) (bool, []uint, error) {
	var variantIDs []uint
	err := r.db.Model(&entity.VariantOptionValue{}).
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
	productID uint,
) ([]entity.ProductOption, map[uint]int, error) {
	// Get all options for the product with their values preloaded
	var options []entity.ProductOption
	if err := r.db.
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
			r.db.Model(&entity.VariantOptionValue{}).
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
func (r *ProductOptionRepositoryImpl) DeleteOptionValuesByOptionID(optionID uint) error {
	return r.db.Where("option_id = ?", optionID).Delete(&entity.ProductOptionValue{}).Error
}

/***********************************************
 *    Bulk Update Methods                      *
 ***********************************************/

// BulkUpdateOptions updates multiple options in a transaction
func (r *ProductOptionRepositoryImpl) BulkUpdateOptions(options []*entity.ProductOption) error {
	if len(options) == 0 {
		return nil
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
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
	values []*entity.ProductOptionValue,
) error {
	if len(values) == 0 {
		return nil
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
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
