package repositories

import (
	"errors"

	"ecommerce-be/product/entity"
	producterrors "ecommerce-be/product/errors"
	"ecommerce-be/product/mapper"
	productQuery "ecommerce-be/product/query"

	"gorm.io/gorm"
)

// VariantRepository defines the interface for variant-related database operations
type VariantRepository interface {
	FindVariantByID(variantID uint) (*entity.ProductVariant, error)
	FindVariantByProductIDAndVariantID(productID, variantID uint) (*entity.ProductVariant, error)
	FindVariantByOptions(
		productID uint,
		optionValues map[string]string,
	) (*entity.ProductVariant, error)
	GetVariantOptionValues(variantID uint) ([]entity.VariantOptionValue, error)
	CreateVariant(variant *entity.ProductVariant) error
	BulkCreateVariants(variants []*entity.ProductVariant) error
	CreateVariantOptionValues(variantOptionValues []entity.VariantOptionValue) error
	UpdateVariant(variant *entity.ProductVariant) error
	DeleteVariant(variantID uint) error
	CountVariantsByProductID(productID uint) (int64, error)
	DeleteVariantOptionValues(variantID uint) error
	FindVariantsByIDs(variantIDs []uint) ([]entity.ProductVariant, error)
	BulkUpdateVariants(variants []*entity.ProductVariant) error
	UnsetAllDefaultVariantsForProduct(productID uint) error
	GetProductVariantAggregation(productID uint) (*mapper.VariantAggregation, error)
	GetProductsVariantAggregations(productIDs []uint) (map[uint]*mapper.VariantAggregation, error)
	GetProductVariantsWithOptions(productID uint) ([]mapper.VariantWithOptions, error)
	FindVariantsByProductID(productID uint) ([]entity.ProductVariant, error)
	DeleteVariantsByProductID(productID uint) error
	DeleteVariantOptionValuesByVariantIDs(variantIDs []uint) error
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
			return nil, producterrors.ErrVariantNotFound
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
			return nil, producterrors.ErrVariantNotFound
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
		return nil, producterrors.ErrProductHasNoOptions
	}

	// Build a map of option name to option ID
	optionNameToID := make(map[string]uint)
	for _, opt := range productOptions {
		optionNameToID[opt.Name] = opt.ID
	}

	// Validate that all provided options exist
	for optionName := range optionValues {
		if _, exists := optionNameToID[optionName]; !exists {
			return nil, producterrors.ErrInvalidOptionName.WithMessagef(
				"Invalid option name: %s",
				optionName,
			)
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

	return nil, producterrors.ErrVariantNotFoundWithOptions
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

// CreateVariant creates a new variant for a product
func (r *VariantRepositoryImpl) CreateVariant(variant *entity.ProductVariant) error {
	return r.db.Create(variant).Error
}

// BulkCreateVariants creates multiple variants in a single INSERT query
// Uses RETURNING clause to get generated IDs efficiently
func (r *VariantRepositoryImpl) BulkCreateVariants(variants []*entity.ProductVariant) error {
	if len(variants) == 0 {
		return nil
	}
	// GORM's Create with slice automatically uses bulk insert and populates IDs
	return r.db.Create(&variants).Error
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

// UnsetAllDefaultVariantsForProduct sets is_default=false for all variants of a product
// This is used to enforce "only one default variant per product" constraint
func (r *VariantRepositoryImpl) UnsetAllDefaultVariantsForProduct(productID uint) error {
	return r.db.Model(&entity.ProductVariant{}).
		Where("product_id = ? AND is_default = ?", productID, true).
		Update("is_default", false).Error
}

// GetProductVariantAggregation retrieves aggregated variant data for a single product
func (r *VariantRepositoryImpl) GetProductVariantAggregation(
	productID uint,
) (*mapper.VariantAggregation, error) {
	var aggregation mapper.VariantAggregation
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

	// Get price range and availability (using allow_purchase instead of stock)
	var priceAgg struct {
		MinPrice      float64
		MaxPrice      float64
		AllowPurchase bool
		MainImage     string
	}

	err := r.db.Model(&entity.ProductVariant{}).
		Select(productQuery.VARIANT_PRICE_AGGREGATION_QUERY, productID).
		Where("product_id = ?", productID).
		Scan(&priceAgg).Error
	if err != nil {
		return nil, err
	}

	aggregation.MinPrice = priceAgg.MinPrice
	aggregation.MaxPrice = priceAgg.MaxPrice
	aggregation.AllowPurchase = priceAgg.AllowPurchase
	if priceAgg.MainImage != "" {
		aggregation.MainImage = priceAgg.MainImage
	}

	// Get option names and values
	var optionData []struct {
		OptionName  string
		OptionValue string
	}

	err = r.db.Table("variant_option_value vov").
		Select("po.name as option_name, pov.value as option_value").
		Joins("JOIN product_option_value pov ON vov.option_value_id = pov.id").
		Joins("JOIN product_option po ON pov.option_id = po.id").
		Joins("JOIN product_variant pv ON vov.variant_id = pv.id").
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
) (map[uint]*mapper.VariantAggregation, error) {
	result := make(map[uint]*mapper.VariantAggregation)

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
		result[vc.ProductID] = &mapper.VariantAggregation{
			HasVariants:   true,
			TotalVariants: int(vc.Count),
			OptionValues:  make(map[string][]string),
		}
	}

	// Initialize products without variants
	for _, productID := range productIDs {
		if !productsWithVariants[productID] {
			result[productID] = &mapper.VariantAggregation{
				HasVariants:  false,
				OptionValues: make(map[string][]string),
			}
		}
	}

	// Get price range and total stock for products with variants
	if len(productsWithVariants) > 0 {
		var priceAggData []struct {
			ProductID     uint
			MinPrice      float64
			MaxPrice      float64
			AllowPurchase bool
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
				BOOL_OR(allow_purchase) as allow_purchase
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
				result[agg.ProductID].AllowPurchase = agg.AllowPurchase
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

		err = r.db.Table("variant_option_value vov").
			Select("pv.product_id as product_id, po.name as option_name, pov.value as option_value").
			Joins("JOIN product_option_value pov ON vov.option_value_id = pov.id").
			Joins("JOIN product_option po ON pov.option_id = po.id").
			Joins("JOIN product_variant pv ON vov.variant_id = pv.id").
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
) ([]mapper.VariantWithOptions, error) {
	// First, get all variants for the product
	var variants []entity.ProductVariant
	if err := r.db.Where("product_id = ?", productID).Find(&variants).Error; err != nil {
		return nil, err
	}

	if len(variants) == 0 {
		return []mapper.VariantWithOptions{}, nil
	}

	// Extract variant IDs for batch query
	variantIDs := make([]uint, len(variants))
	variantMap := make(map[uint]*entity.ProductVariant)
	for i, v := range variants {
		variantIDs[i] = v.ID
		variantCopy := v
		variantMap[v.ID] = &variantCopy
	}

	var optionData []mapper.OptionValueData
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
	variantOptionsMap := make(map[uint][]mapper.SelectedOptionValue)
	for _, od := range optionData {
		variantOptionsMap[od.VariantID] = append(
			variantOptionsMap[od.VariantID],
			mapper.SelectedOptionValue{
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
	result := make([]mapper.VariantWithOptions, 0, len(variants))
	for _, variant := range variants {
		result = append(result, mapper.VariantWithOptions{
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
func (r *VariantRepositoryImpl) FindVariantsByProductID(
	productID uint,
) ([]entity.ProductVariant, error) {
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
