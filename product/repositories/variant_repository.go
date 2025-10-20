package repositories

import (
	"errors"
	"fmt"

	"ecommerce-be/product/entity"
	"ecommerce-be/product/utils"

	"gorm.io/gorm"
)

// VariantRepository defines the interface for variant-related database operations
type VariantRepository interface {
	// FindVariantByID retrieves a variant by its ID with all related data
	FindVariantByID(variantID uint) (*entity.ProductVariant, error)

	// FindVariantByProductIDAndVariantID retrieves a variant by product ID and variant ID
	FindVariantByProductIDAndVariantID(productID, variantID uint) (*entity.ProductVariant, error)

	// FindVariantByOptions finds a variant based on selected option values
	FindVariantByOptions(
		productID uint,
		optionValues map[string]string,
	) (*entity.ProductVariant, error)

	// GetVariantOptionValues retrieves all option values for a specific variant
	GetVariantOptionValues(variantID uint) ([]entity.VariantOptionValue, error)

	// GetProductOptionByName retrieves a product option by name for a specific product
	GetProductOptionByName(productID uint, optionName string) (*entity.ProductOption, error)

	// GetProductOptionValueByValue retrieves an option value by its value string
	GetProductOptionValueByValue(optionID uint, value string) (*entity.ProductOptionValue, error)

	// GetAvailableOptionsForProduct retrieves all available options and their values for a product
	GetAvailableOptionsForProduct(productID uint) (map[string][]string, error)

	// GetProductOptionsWithVariantCounts retrieves detailed options with variant counts for each value
	GetProductOptionsWithVariantCounts(productID uint) ([]entity.ProductOption, map[uint]int, error)

	// GetProductOptionByID retrieves a product option by its ID
	GetProductOptionByID(optionID uint) (*entity.ProductOption, error)

	// GetOptionValueByID retrieves an option value by its ID
	GetOptionValueByID(
		optionValueID uint,
	) (*entity.ProductOptionValue, error)

	// CreateVariant creates a new variant for a product
	CreateVariant(variant *entity.ProductVariant) error

	// CreateVariantOptionValues creates variant option value associations
	CreateVariantOptionValues(variantOptionValues []entity.VariantOptionValue) error

	// UpdateVariant updates an existing variant
	UpdateVariant(variant *entity.ProductVariant) error

	// DeleteVariant deletes a variant by ID
	DeleteVariant(variantID uint) error

	// CountVariantsByProductID counts the number of variants for a product
	CountVariantsByProductID(productID uint) (int64, error)

	// DeleteVariantOptionValues deletes all option values for a variant
	DeleteVariantOptionValues(variantID uint) error

	// FindVariantsByIDs retrieves multiple variants by their IDs
	FindVariantsByIDs(variantIDs []uint) ([]entity.ProductVariant, error)

	// BulkUpdateVariants updates multiple variants in a transaction
	BulkUpdateVariants(variants []*entity.ProductVariant) error

	// GetProductVariantAggregation retrieves aggregated variant data for a product
	GetProductVariantAggregation(productID uint) (*VariantAggregation, error)

	// GetProductsVariantAggregations retrieves aggregated variant data for multiple products
	GetProductsVariantAggregations(productIDs []uint) (map[uint]*VariantAggregation, error)

	// GetProductVariantsWithOptions retrieves all variants for a product with their selected options
	GetProductVariantsWithOptions(productID uint) ([]VariantWithOptions, error)

	// Bulk deletion methods for product cleanup
	FindVariantsByProductID(productID uint) ([]entity.ProductVariant, error)
	DeleteVariantsByProductID(productID uint) error
	DeleteVariantOptionValuesByVariantIDs(variantIDs []uint) error
}

// VariantAggregation represents aggregated variant data for a product
type VariantAggregation struct {
	HasVariants   bool
	TotalVariants int
	MinPrice      float64
	MaxPrice      float64
	TotalStock    int
	InStock       bool
	Currency      string
	MainImage     string
	OptionNames   []string
	OptionValues  map[string][]string // optionName -> []values
}

// VariantWithOptions represents a variant with its selected option values
type VariantWithOptions struct {
	Variant         entity.ProductVariant
	SelectedOptions []SelectedOptionValue
}

// SelectedOptionValue represents a selected option value for a variant
type SelectedOptionValue struct {
	OptionID          uint
	OptionName        string
	OptionDisplayName string
	ValueID           uint
	Value             string
	ValueDisplayName  string
	ColorCode         string
}

// VariantRepositoryImpl implements the VariantRepository interface
type VariantRepositoryImpl struct {
	db *gorm.DB
}

// NewVariantRepository creates a new instance of VariantRepository
func NewVariantRepository(db *gorm.DB) VariantRepository {
	return &VariantRepositoryImpl{db: db}
}

// FindVariantByID retrieves a variant by its ID
func (r *VariantRepositoryImpl) FindVariantByID(variantID uint) (*entity.ProductVariant, error) {
	var variant entity.ProductVariant
	result := r.db.Where("id = ?", variantID).First(&variant)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New(utils.VARIANT_NOT_FOUND_MSG)
		}
		return nil, result.Error
	}

	return &variant, nil
}

// FindVariantByProductIDAndVariantID retrieves a variant by product ID and variant ID
func (r *VariantRepositoryImpl) FindVariantByProductIDAndVariantID(
	productID, variantID uint,
) (*entity.ProductVariant, error) {
	var variant entity.ProductVariant
	result := r.db.Where("id = ? AND product_id = ?", variantID, productID).First(&variant)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New(utils.VARIANT_NOT_FOUND_MSG)
		}
		return nil, result.Error
	}

	return &variant, nil
}

// FindVariantByOptions finds a variant based on selected option values
func (r *VariantRepositoryImpl) FindVariantByOptions(
	productID uint,
	optionValues map[string]string,
) (*entity.ProductVariant, error) {
	// First, get all options for the product
	var productOptions []entity.ProductOption
	if err := r.db.Where("product_id = ?", productID).Find(&productOptions).Error; err != nil {
		return nil, err
	}

	if len(productOptions) == 0 {
		return nil, errors.New(utils.PRODUCT_HAS_NO_OPTIONS_MSG)
	}

	// Build a map of option name to option ID
	optionNameToID := make(map[string]uint)
	for _, opt := range productOptions {
		optionNameToID[opt.Name] = opt.ID
	}

	// Validate that all provided options exist
	for optionName := range optionValues {
		if _, exists := optionNameToID[optionName]; !exists {
			return nil, fmt.Errorf("%s: %s", utils.INVALID_OPTION_NAME_MSG, optionName)
		}
	}

	// Get all variants for the product
	var variants []entity.ProductVariant
	if err := r.db.Where("product_id = ?", productID).Find(&variants).Error; err != nil {
		return nil, err
	}

	// For each variant, check if it matches all the selected options
	for _, variant := range variants {
		// Get all option values for this variant
		var variantOptionValues []entity.VariantOptionValue
		err := r.db.Where("variant_id = ?", variant.ID).Find(&variantOptionValues).Error
		if err != nil {
			continue
		}

		// Check if this variant matches all selected options
		matchCount := 0
		for optionName, selectedValue := range optionValues {
			optionID := optionNameToID[optionName]

			// Find the variant option value for this option
			for _, vov := range variantOptionValues {
				if vov.OptionID == optionID {
					// Get the actual value
					var optionValue entity.ProductOptionValue
					err := r.db.Where("id = ?", vov.OptionValueID).First(&optionValue).Error
					if err != nil {
						break
					}

					if optionValue.Value == selectedValue {
						matchCount++
					}
					break
				}
			}
		}

		// If all selected options match, return this variant
		if matchCount == len(optionValues) {
			return &variant, nil
		}
	}

	return nil, errors.New(utils.VARIANT_NOT_FOUND_WITH_OPTIONS_MSG)
}

// GetVariantOptionValues retrieves all option values for a specific variant
func (r *VariantRepositoryImpl) GetVariantOptionValues(
	variantID uint,
) ([]entity.VariantOptionValue, error) {
	var variantOptionValues []entity.VariantOptionValue
	result := r.db.Where("variant_id = ?", variantID).Find(&variantOptionValues)

	if result.Error != nil {
		return nil, result.Error
	}

	return variantOptionValues, nil
}

// GetProductOptionByName retrieves a product option by name for a specific product
func (r *VariantRepositoryImpl) GetProductOptionByName(
	productID uint,
	optionName string,
) (*entity.ProductOption, error) {
	var option entity.ProductOption
	result := r.db.Where("product_id = ? AND name = ?", productID, optionName).First(&option)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%s: %s", utils.OPTION_NOT_FOUND_MSG, optionName)
		}
		return nil, result.Error
	}

	return &option, nil
}

// GetProductOptionValueByValue retrieves an option value by its value string
func (r *VariantRepositoryImpl) GetProductOptionValueByValue(
	optionID uint,
	value string,
) (*entity.ProductOptionValue, error) {
	var optionValue entity.ProductOptionValue
	result := r.db.Where("option_id = ? AND value = ?", optionID, value).First(&optionValue)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%s: %s", utils.OPTION_VALUE_NOT_FOUND_MSG, value)
		}
		return nil, result.Error
	}

	return &optionValue, nil
}

// GetAvailableOptionsForProduct retrieves all available options and their values for a product
func (r *VariantRepositoryImpl) GetAvailableOptionsForProduct(
	productID uint,
) (map[string][]string, error) {
	// Get all options for the product
	var options []entity.ProductOption
	if err := r.db.Where("product_id = ?", productID).Find(&options).Error; err != nil {
		return nil, err
	}

	availableOptions := make(map[string][]string)

	for _, option := range options {
		// Get all values for this option
		var optionValues []entity.ProductOptionValue
		if err := r.db.Where("option_id = ?", option.ID).Find(&optionValues).Error; err != nil {
			continue
		}

		values := make([]string, 0, len(optionValues))
		for _, ov := range optionValues {
			values = append(values, ov.Value)
		}

		availableOptions[option.Name] = values
	}

	return availableOptions, nil
}

// GetProductOptionsWithVariantCounts retrieves detailed options with variant counts for each value
func (r *VariantRepositoryImpl) GetProductOptionsWithVariantCounts(
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

// GetProductOptionByID retrieves a product option by its ID
func (r *VariantRepositoryImpl) GetProductOptionByID(optionID uint) (*entity.ProductOption, error) {
	var option entity.ProductOption
	result := r.db.Where("id = ?", optionID).First(&option)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%s: %d", utils.OPTION_NOT_FOUND_MSG, optionID)
		}
		return nil, result.Error
	}

	return &option, nil
}

// GetOptionValueByID retrieves an option value by its ID
func (r *VariantRepositoryImpl) GetOptionValueByID(
	optionValueID uint,
) (*entity.ProductOptionValue, error) {
	var optionValue entity.ProductOptionValue
	result := r.db.Where("id = ?", optionValueID).First(&optionValue)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%s: %d", utils.OPTION_VALUE_NOT_FOUND_MSG, optionValueID)
		}
		return nil, result.Error
	}

	return &optionValue, nil
}

// CreateVariant creates a new variant for a product
func (r *VariantRepositoryImpl) CreateVariant(variant *entity.ProductVariant) error {
	return r.db.Create(variant).Error
}

// CreateVariantOptionValues creates variant option value associations
func (r *VariantRepositoryImpl) CreateVariantOptionValues(
	variantOptionValues []entity.VariantOptionValue,
) error {
	if len(variantOptionValues) == 0 {
		return nil
	}
	return r.db.Create(&variantOptionValues).Error
}

// UpdateVariant updates an existing variant
func (r *VariantRepositoryImpl) UpdateVariant(variant *entity.ProductVariant) error {
	return r.db.Save(variant).Error
}

// DeleteVariant deletes a variant by ID
func (r *VariantRepositoryImpl) DeleteVariant(variantID uint) error {
	return r.db.Delete(&entity.ProductVariant{}, variantID).Error
}

// CountVariantsByProductID counts the number of variants for a product
func (r *VariantRepositoryImpl) CountVariantsByProductID(productID uint) (int64, error) {
	var count int64
	err := r.db.Model(&entity.ProductVariant{}).
		Where("product_id = ?", productID).
		Count(&count).Error
	return count, err
}

// DeleteVariantOptionValues deletes all option values for a variant
func (r *VariantRepositoryImpl) DeleteVariantOptionValues(variantID uint) error {
	return r.db.Where("variant_id = ?", variantID).
		Delete(&entity.VariantOptionValue{}).Error
}

// FindVariantsByIDs retrieves multiple variants by their IDs
func (r *VariantRepositoryImpl) FindVariantsByIDs(
	variantIDs []uint,
) ([]entity.ProductVariant, error) {
	var variants []entity.ProductVariant
	result := r.db.Where("id IN ?", variantIDs).Find(&variants)
	if result.Error != nil {
		return nil, result.Error
	}
	return variants, nil
}

// BulkUpdateVariants updates multiple variants in a transaction
func (r *VariantRepositoryImpl) BulkUpdateVariants(variants []*entity.ProductVariant) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, variant := range variants {
			if err := tx.Save(variant).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetProductVariantAggregation retrieves aggregated variant data for a single product
func (r *VariantRepositoryImpl) GetProductVariantAggregation(
	productID uint,
) (*VariantAggregation, error) {
	var aggregation VariantAggregation
	aggregation.OptionValues = make(map[string][]string)

	// Check if product has variants
	var variantCount int64
	if err := r.db.Model(&entity.ProductVariant{}).
		Where("product_id = ?", productID).
		Count(&variantCount).Error; err != nil {
		return nil, err
	}

	if variantCount == 0 {
		aggregation.HasVariants = false
		return &aggregation, nil
	}

	aggregation.HasVariants = true
	aggregation.TotalVariants = int(variantCount)

	// Get price range and total stock
	var priceAgg struct {
		MinPrice   float64
		MaxPrice   float64
		TotalStock int
		InStock    bool
		Currency   string
		MainImage  string
	}

	err := r.db.Model(&entity.ProductVariant{}).
		Select(`
			MIN(price) as min_price,
			MAX(price) as max_price,
			SUM(stock) as total_stock,
			BOOL_OR(in_stock) as in_stock,
			MAX(currency) as currency,
			(SELECT images FROM product_variants WHERE product_id = ? AND is_default = true AND images IS NOT NULL AND images != '{}' LIMIT 1) as main_image
		`, productID).
		Where("product_id = ?", productID).
		Scan(&priceAgg).Error
	if err != nil {
		return nil, err
	}

	aggregation.MinPrice = priceAgg.MinPrice
	aggregation.MaxPrice = priceAgg.MaxPrice
	aggregation.TotalStock = priceAgg.TotalStock
	aggregation.InStock = priceAgg.InStock
	aggregation.Currency = priceAgg.Currency
	if priceAgg.MainImage != "" {
		aggregation.MainImage = priceAgg.MainImage
	}

	// Get option names and values
	var optionData []struct {
		OptionName  string
		OptionValue string
	}

	err = r.db.Table("variant_option_values vov").
		Select("po.name as option_name, pov.value as option_value").
		Joins("JOIN product_option_values pov ON vov.option_value_id = pov.id").
		Joins("JOIN product_options po ON pov.option_id = po.id").
		Joins("JOIN product_variants pv ON vov.variant_id = pv.id").
		Where("pv.product_id = ?", productID).
		Group("po.name, pov.value").
		Order("po.name, pov.value").
		Scan(&optionData).Error
	if err != nil {
		return nil, err
	}

	// Build option names and option values map
	optionNamesSet := make(map[string]bool)
	for _, od := range optionData {
		optionNamesSet[od.OptionName] = true

		if _, exists := aggregation.OptionValues[od.OptionName]; !exists {
			aggregation.OptionValues[od.OptionName] = []string{}
		}

		// Check if value already exists to avoid duplicates
		valueExists := false
		for _, v := range aggregation.OptionValues[od.OptionName] {
			if v == od.OptionValue {
				valueExists = true
				break
			}
		}
		if !valueExists {
			aggregation.OptionValues[od.OptionName] = append(
				aggregation.OptionValues[od.OptionName],
				od.OptionValue,
			)
		}
	}

	// Convert option names set to slice
	for name := range optionNamesSet {
		aggregation.OptionNames = append(aggregation.OptionNames, name)
	}

	return &aggregation, nil
}

// GetProductsVariantAggregations retrieves aggregated variant data for multiple products
func (r *VariantRepositoryImpl) GetProductsVariantAggregations(
	productIDs []uint,
) (map[uint]*VariantAggregation, error) {
	result := make(map[uint]*VariantAggregation)

	if len(productIDs) == 0 {
		return result, nil
	}

	// Get variant counts per product
	var variantCounts []struct {
		ProductID uint
		Count     int64
	}

	if err := r.db.Model(&entity.ProductVariant{}).
		Select("product_id, COUNT(*) as count").
		Where("product_id IN ?", productIDs).
		Group("product_id").
		Scan(&variantCounts).Error; err != nil {
		return nil, err
	}

	// Initialize result map with products that have variants
	productsWithVariants := make(map[uint]bool)
	for _, vc := range variantCounts {
		productsWithVariants[vc.ProductID] = true
		result[vc.ProductID] = &VariantAggregation{
			HasVariants:   true,
			TotalVariants: int(vc.Count),
			OptionValues:  make(map[string][]string),
		}
	}

	// Initialize products without variants
	for _, productID := range productIDs {
		if !productsWithVariants[productID] {
			result[productID] = &VariantAggregation{
				HasVariants:  false,
				OptionValues: make(map[string][]string),
			}
		}
	}

	// Get price range and total stock for products with variants
	if len(productsWithVariants) > 0 {
		var priceAggData []struct {
			ProductID  uint
			MinPrice   float64
			MaxPrice   float64
			TotalStock int
			InStock    bool
			Currency   string
		}

		variantProductIDs := make([]uint, 0, len(productsWithVariants))
		for pid := range productsWithVariants {
			variantProductIDs = append(variantProductIDs, pid)
		}

		err := r.db.Model(&entity.ProductVariant{}).
			Select(`
				product_id,
				MIN(price) as min_price,
				MAX(price) as max_price,
				SUM(stock) as total_stock,
				BOOL_OR(in_stock) as in_stock,
				MAX(currency) as currency
			`).
			Where("product_id IN ?", variantProductIDs).
			Group("product_id").
			Scan(&priceAggData).Error
		if err != nil {
			return nil, err
		}

		for _, agg := range priceAggData {
			if result[agg.ProductID] != nil {
				result[agg.ProductID].MinPrice = agg.MinPrice
				result[agg.ProductID].MaxPrice = agg.MaxPrice
				result[agg.ProductID].TotalStock = agg.TotalStock
				result[agg.ProductID].InStock = agg.InStock
				result[agg.ProductID].Currency = agg.Currency
			}
		}

		// Get main images from default variants for products
		var imageData []struct {
			ProductID uint
			Images    string
		}

		err = r.db.Model(&entity.ProductVariant{}).
			Select("DISTINCT ON (product_id) product_id, images").
			Where("product_id IN ? AND is_default = true AND images IS NOT NULL AND images != '{}'", variantProductIDs).
			Scan(&imageData).Error
		if err != nil {
			return nil, err
		}

		for _, img := range imageData {
			if result[img.ProductID] != nil {
				result[img.ProductID].MainImage = img.Images
			}
		}

		// Get option names and values for all products
		var optionData []struct {
			ProductID   uint
			OptionName  string
			OptionValue string
		}

		err = r.db.Table("variant_option_values vov").
			Select("pv.product_id as product_id, po.name as option_name, pov.value as option_value").
			Joins("JOIN product_option_values pov ON vov.option_value_id = pov.id").
			Joins("JOIN product_options po ON pov.option_id = po.id").
			Joins("JOIN product_variants pv ON vov.variant_id = pv.id").
			Where("pv.product_id IN ?", variantProductIDs).
			Group("pv.product_id, po.name, pov.value").
			Order("pv.product_id, po.name, pov.value").
			Scan(&optionData).Error
		if err != nil {
			return nil, err
		}

		// Build option names and option values map for each product
		optionNamesMap := make(map[uint]map[string]bool)
		for _, od := range optionData {
			if result[od.ProductID] == nil {
				continue
			}

			// Track option names
			if _, exists := optionNamesMap[od.ProductID]; !exists {
				optionNamesMap[od.ProductID] = make(map[string]bool)
			}
			optionNamesMap[od.ProductID][od.OptionName] = true

			// Add option values
			if _, exists := result[od.ProductID].OptionValues[od.OptionName]; !exists {
				result[od.ProductID].OptionValues[od.OptionName] = []string{}
			}

			// Check if value already exists to avoid duplicates
			valueExists := false
			for _, v := range result[od.ProductID].OptionValues[od.OptionName] {
				if v == od.OptionValue {
					valueExists = true
					break
				}
			}
			if !valueExists {
				result[od.ProductID].OptionValues[od.OptionName] = append(
					result[od.ProductID].OptionValues[od.OptionName],
					od.OptionValue,
				)
			}
		}

		// Convert option names sets to slices
		for productID, optionNames := range optionNamesMap {
			if result[productID] != nil {
				for name := range optionNames {
					result[productID].OptionNames = append(result[productID].OptionNames, name)
				}
			}
		}
	}

	return result, nil
}

/*******************************************************************************
*  GetProductVariantsWithOptions retrieves all variants for a product          *
*  with their selected option values in a single optimized query               *
*******************************************************************************/
func (r *VariantRepositoryImpl) GetProductVariantsWithOptions(
	productID uint,
) ([]VariantWithOptions, error) {
	// First, get all variants for the product
	var variants []entity.ProductVariant
	if err := r.db.Where("product_id = ?", productID).Find(&variants).Error; err != nil {
		return nil, err
	}

	if len(variants) == 0 {
		return []VariantWithOptions{}, nil
	}

	// Extract variant IDs for batch query
	variantIDs := make([]uint, len(variants))
	variantMap := make(map[uint]*entity.ProductVariant)
	for i, v := range variants {
		variantIDs[i] = v.ID
		variantCopy := v
		variantMap[v.ID] = &variantCopy
	}

	// Batch query to get all variant option values with option and option value details
	// This is a single JOIN query for performance
	type OptionValueData struct {
		VariantID         uint
		OptionID          uint
		OptionName        string
		OptionDisplayName string
		ValueID           uint
		Value             string
		ValueDisplayName  string
		ColorCode         string
	}

	var optionData []OptionValueData
	err := r.db.Table("variant_option_value AS vov").
		Select(`
			vov.variant_id,
			po.id AS option_id,
			po.name AS option_name,
			po.display_name AS option_display_name,
			pov.id AS value_id,
			pov.value,
			pov.display_name AS value_display_name,
			pov.color_code
		`).
		Joins("JOIN product_option po ON vov.option_id = po.id").
		Joins("JOIN product_option_value pov ON vov.option_value_id = pov.id").
		Where("vov.variant_id IN ?", variantIDs).
		Order("po.position ASC, pov.position ASC").
		Find(&optionData).Error
	if err != nil {
		return nil, err
	}

	// Group option values by variant ID
	variantOptionsMap := make(map[uint][]SelectedOptionValue)
	for _, od := range optionData {
		variantOptionsMap[od.VariantID] = append(
			variantOptionsMap[od.VariantID],
			SelectedOptionValue{
				OptionID:          od.OptionID,
				OptionName:        od.OptionName,
				OptionDisplayName: od.OptionDisplayName,
				ValueID:           od.ValueID,
				Value:             od.Value,
				ValueDisplayName:  od.ValueDisplayName,
				ColorCode:         od.ColorCode,
			},
		)
	}

	// Build result
	result := make([]VariantWithOptions, 0, len(variants))
	for _, variant := range variants {
		result = append(result, VariantWithOptions{
			Variant:         variant,
			SelectedOptions: variantOptionsMap[variant.ID],
		})
	}

	return result, nil
}

/***********************************************
 *    Bulk Deletion Methods for Product Cleanup
 ***********************************************/

// FindVariantsByProductID retrieves all variants for a product
func (r *VariantRepositoryImpl) FindVariantsByProductID(productID uint) ([]entity.ProductVariant, error) {
	var variants []entity.ProductVariant
	err := r.db.Where("product_id = ?", productID).Find(&variants).Error
	return variants, err
}

// DeleteVariantsByProductID deletes all variants for a product
func (r *VariantRepositoryImpl) DeleteVariantsByProductID(productID uint) error {
	return r.db.Where("product_id = ?", productID).Delete(&entity.ProductVariant{}).Error
}

// DeleteVariantOptionValuesByVariantIDs deletes all variant option values for given variant IDs
func (r *VariantRepositoryImpl) DeleteVariantOptionValuesByVariantIDs(variantIDs []uint) error {
	if len(variantIDs) == 0 {
		return nil
	}
	return r.db.Where("variant_id IN ?", variantIDs).Delete(&entity.VariantOptionValue{}).Error
}
