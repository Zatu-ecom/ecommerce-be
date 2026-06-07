package repository

import (
	"context"
	"errors"
	"strings"

	"ecommerce-be/common/db"
	"ecommerce-be/common/helper"
	"ecommerce-be/product/entity"
	producterrors "ecommerce-be/product/error"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
	productQuery "ecommerce-be/product/query"
	productUtils "ecommerce-be/product/utils"

	"gorm.io/gorm"
)

// VariantRepository defines the interface for variant-related database operations
type VariantRepository interface {
	FindVariantByID(ctx context.Context, variantID uint) (*entity.ProductVariant, error)
	FindVariantByProductIDAndVariantID(
		ctx context.Context,
		productID, variantID uint,
	) (*entity.ProductVariant, error)
	FindVariantByOptions(
		ctx context.Context,
		productID uint,
		optionValues map[string]string,
	) (*entity.ProductVariant, error)
	GetVariantOptionValues(ctx context.Context, variantID uint) ([]entity.VariantOptionValue, error)
	CreateVariant(ctx context.Context, variant *entity.ProductVariant) error
	BulkCreateVariants(ctx context.Context, variants []*entity.ProductVariant) error
	CreateVariantOptionValues(
		ctx context.Context,
		variantOptionValues []entity.VariantOptionValue,
	) error
	UpdateVariant(ctx context.Context, variant *entity.ProductVariant) error
	DeleteVariant(ctx context.Context, variantID uint) error
	CountVariantsByProductID(ctx context.Context, productID uint) (int64, error)
	DeleteVariantOptionValues(ctx context.Context, variantID uint) error
	FindVariantsByIDs(ctx context.Context, variantIDs []uint) ([]entity.ProductVariant, error)
	BulkUpdateVariants(ctx context.Context, variants []*entity.ProductVariant) error
	UnsetAllDefaultVariantsForProduct(ctx context.Context, productID uint) error
	GetProductVariantAggregation(
		ctx context.Context,
		productID uint,
		userID *uint, // Optional: if provided, checks if any variant is wishlisted by this user
	) (*mapper.VariantAggregation, error)
	GetProductsVariantAggregations(
		ctx context.Context,
		productIDs []uint,
		userID *uint, // Optional: if provided, checks if any variant is wishlisted by this user
	) (map[uint]*mapper.VariantAggregation, error)
	GetProductVariantsWithOptions(
		ctx context.Context,
		productID uint,
	) ([]mapper.VariantWithOptions, error)
	FindVariantsByProductID(ctx context.Context, productID uint) ([]entity.ProductVariant, error)
	DeleteVariantsByProductID(ctx context.Context, productID uint) error
	DeleteVariantOptionValuesByVariantIDs(ctx context.Context, variantIDs []uint) error
	FindPlaceholderVariants(ctx context.Context, productID uint) ([]entity.ProductVariant, error)
	FindFirstOptionDerivedVariant(ctx context.Context, productID uint) (*entity.ProductVariant, error)
	UpdateAllVariantsFlags(
		ctx context.Context,
		productID uint,
		allowPurchase *bool,
		isPopular *bool,
	) error
	ListVariantsWithFilters(
		ctx context.Context,
		filters *model.ListVariantsRequest,
		sellerID *uint,
		optionFilters map[string]string,
	) ([]mapper.VariantWithOptions, int64, error)
	GetProductCountByVariantIDs(
		ctx context.Context,
		variantIDs []uint,
		sellerID *uint,
	) (uint, error)
	GetProductBasicInfoByVariantIDs(
		ctx context.Context,
		variantIDs []uint,
		sellerID *uint,
	) ([]mapper.VariantBasicInfoRow, error)
}

// VariantRepositoryImpl implements the VariantRepository interface
type VariantRepositoryImpl struct{}

// NewVariantRepository creates a new instance of VariantRepository
func NewVariantRepository() VariantRepository {
	return &VariantRepositoryImpl{}
}

// FindVariantByID retrieves a variant by its ID
func (r *VariantRepositoryImpl) FindVariantByID(
	ctx context.Context,
	variantID uint,
) (*entity.ProductVariant, error) {
	var variant entity.ProductVariant
	result := db.DB(ctx).Where("id = ?", variantID).First(&variant)

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
	ctx context.Context,
	productID, variantID uint,
) (*entity.ProductVariant, error) {
	var variant entity.ProductVariant
	result := db.DB(ctx).Where("id = ? AND product_id = ?", variantID, productID).First(&variant)

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
	ctx context.Context,
	productID uint,
	optionValues map[string]string,
) (*entity.ProductVariant, error) {
	// First, get all options for the product
	var productOptions []entity.ProductOption
	if err := db.DB(ctx).Where("product_id = ?", productID).Find(&productOptions).Error; err != nil {
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
	if err := db.DB(ctx).Where("product_id = ?", productID).Find(&variants).Error; err != nil {
		return nil, err
	}

	// For each variant, check if it matches all the selected options
	for _, variant := range variants {
		// Get all option values for this variant
		var variantOptionValues []entity.VariantOptionValue
		err := db.DB(ctx).Where("variant_id = ?", variant.ID).Find(&variantOptionValues).Error
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
					err := db.DB(ctx).Where("id = ?", vov.OptionValueID).First(&optionValue).Error
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
	ctx context.Context,
	variantID uint,
) ([]entity.VariantOptionValue, error) {
	var variantOptionValues []entity.VariantOptionValue
	result := db.DB(ctx).Where("variant_id = ?", variantID).Find(&variantOptionValues)

	if result.Error != nil {
		return nil, result.Error
	}

	return variantOptionValues, nil
}

// CreateVariant creates a new variant for a product
func (r *VariantRepositoryImpl) CreateVariant(
	ctx context.Context,
	variant *entity.ProductVariant,
) error {
	return db.DB(ctx).Create(variant).Error
}

// BulkCreateVariants creates multiple variants in a single INSERT query
// Uses RETURNING clause to get generated IDs efficiently
func (r *VariantRepositoryImpl) BulkCreateVariants(
	ctx context.Context,
	variants []*entity.ProductVariant,
) error {
	if len(variants) == 0 {
		return nil
	}
	// GORM's Create with slice automatically uses bulk insert and populates IDs
	return db.DB(ctx).Create(&variants).Error
}

// CreateVariantOptionValues creates variant option value associations
func (r *VariantRepositoryImpl) CreateVariantOptionValues(
	ctx context.Context,
	variantOptionValues []entity.VariantOptionValue,
) error {
	if len(variantOptionValues) == 0 {
		return nil
	}
	return db.DB(ctx).Create(&variantOptionValues).Error
}

// UpdateVariant updates an existing variant
func (r *VariantRepositoryImpl) UpdateVariant(
	ctx context.Context,
	variant *entity.ProductVariant,
) error {
	return db.DB(ctx).Save(variant).Error
}

// DeleteVariant deletes a variant by ID
func (r *VariantRepositoryImpl) DeleteVariant(ctx context.Context, variantID uint) error {
	return db.DB(ctx).Delete(&entity.ProductVariant{}, variantID).Error
}

// CountVariantsByProductID counts the number of variants for a product
func (r *VariantRepositoryImpl) CountVariantsByProductID(
	ctx context.Context,
	productID uint,
) (int64, error) {
	var count int64
	err := db.DB(ctx).Model(&entity.ProductVariant{}).
		Where("product_id = ?", productID).
		Count(&count).Error
	return count, err
}

// DeleteVariantOptionValues deletes all option values for a variant
func (r *VariantRepositoryImpl) DeleteVariantOptionValues(
	ctx context.Context,
	variantID uint,
) error {
	return db.DB(ctx).Where("variant_id = ?", variantID).
		Delete(&entity.VariantOptionValue{}).Error
}

// FindVariantsByIDs retrieves multiple variants by their IDs
func (r *VariantRepositoryImpl) FindVariantsByIDs(
	ctx context.Context,
	variantIDs []uint,
) ([]entity.ProductVariant, error) {
	var variants []entity.ProductVariant
	result := db.DB(ctx).Where("id IN ?", variantIDs).Find(&variants)
	if result.Error != nil {
		return nil, result.Error
	}
	return variants, nil
}

// BulkUpdateVariants updates multiple variants in a transaction
func (r *VariantRepositoryImpl) BulkUpdateVariants(
	ctx context.Context,
	variants []*entity.ProductVariant,
) error {
	return db.DB(ctx).Transaction(func(tx *gorm.DB) error {
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
func (r *VariantRepositoryImpl) UnsetAllDefaultVariantsForProduct(
	ctx context.Context,
	productID uint,
) error {
	return db.DB(ctx).Model(&entity.ProductVariant{}).
		Where("product_id = ? AND is_default = ?", productID, true).
		Update("is_default", false).Error
}

// FindPlaceholderVariants returns variants with no linked option values (internal simple-product rows).
func (r *VariantRepositoryImpl) FindPlaceholderVariants(
	ctx context.Context,
	productID uint,
) ([]entity.ProductVariant, error) {
	var variants []entity.ProductVariant
	err := db.DB(ctx).
		Where("product_id = ?", productID).
		Where(`NOT EXISTS (
			SELECT 1 FROM variant_option_value vov WHERE vov.variant_id = product_variant.id
		)`).
		Order("id ASC").
		Find(&variants).Error
	if err != nil {
		return nil, err
	}
	return variants, nil
}

// FindFirstOptionDerivedVariant returns the earliest variant linked to at least one option value.
func (r *VariantRepositoryImpl) FindFirstOptionDerivedVariant(
	ctx context.Context,
	productID uint,
) (*entity.ProductVariant, error) {
	var variant entity.ProductVariant
	err := db.DB(ctx).
		Where("product_id = ?", productID).
		Where(`EXISTS (
			SELECT 1 FROM variant_option_value vov WHERE vov.variant_id = product_variant.id
		)`).
		Order("id ASC").
		First(&variant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &variant, nil
}

// UpdateAllVariantsFlags bulk-updates allow_purchase and/or is_popular for every variant of a product.
func (r *VariantRepositoryImpl) UpdateAllVariantsFlags(
	ctx context.Context,
	productID uint,
	allowPurchase *bool,
	isPopular *bool,
) error {
	updates := map[string]any{}
	if allowPurchase != nil {
		updates["allow_purchase"] = *allowPurchase
	}
	if isPopular != nil {
		updates["is_popular"] = *isPopular
	}
	if len(updates) == 0 {
		return nil
	}

	return db.DB(ctx).Model(&entity.ProductVariant{}).
		Where("product_id = ?", productID).
		Updates(updates).Error
}

// GetProductVariantAggregation retrieves aggregated variant data for a single product
// If userID is provided, also checks if any variant is wishlisted by that user
func (r *VariantRepositoryImpl) GetProductVariantAggregation(
	ctx context.Context,
	productID uint,
	userID *uint,
) (*mapper.VariantAggregation, error) {
	aggregation := &mapper.VariantAggregation{
		OptionValues: make(map[string][]string),
	}

	var variantCount int64
	if err := db.DB(ctx).Model(&entity.ProductVariant{}).
		Where("product_id = ?", productID).
		Count(&variantCount).Error; err != nil {
		return nil, err
	}

	if variantCount == 0 {
		return aggregation, nil
	}

	if err := r.loadProductOptionsCount(ctx, productID, aggregation); err != nil {
		return nil, err
	}
	if err := r.loadOptionDerivedCount(ctx, productID, aggregation); err != nil {
		return nil, err
	}
	if err := r.loadAllVariantFlags(ctx, productID, aggregation); err != nil {
		return nil, err
	}
	if err := r.loadOptionDerivedPriceRange(ctx, productID, aggregation); err != nil {
		return nil, err
	}
	if err := r.loadOptionPreviewForProduct(ctx, productID, aggregation); err != nil {
		return nil, err
	}

	if userID != nil {
		var isWishlisted bool
		if err := db.DB(ctx).
			Raw(productQuery.WISHLIST_CHECK_SINGLE_PRODUCT, productID, *userID).
			Scan(&isWishlisted).Error; err != nil {
			return nil, err
		}
		aggregation.IsWishlisted = isWishlisted
	}

	productUtils.ApplyAggregationSemantics(aggregation)
	return aggregation, nil
}

// GetProductsVariantAggregations retrieves aggregated variant data for multiple products
// If userID is provided, also checks if any variant of each product is wishlisted by that user
func (r *VariantRepositoryImpl) GetProductsVariantAggregations(
	ctx context.Context,
	productIDs []uint,
	userID *uint,
) (map[uint]*mapper.VariantAggregation, error) {
	result := make(map[uint]*mapper.VariantAggregation, len(productIDs))
	for _, productID := range productIDs {
		result[productID] = &mapper.VariantAggregation{
			OptionValues: make(map[string][]string),
		}
	}

	if len(productIDs) == 0 {
		return result, nil
	}

	var variantCounts []struct {
		ProductID uint
		Count     int64
	}
	if err := db.DB(ctx).Model(&entity.ProductVariant{}).
		Select("product_id, COUNT(*) as count").
		Where("product_id IN ?", productIDs).
		Group("product_id").
		Scan(&variantCounts).Error; err != nil {
		return nil, err
	}

	productsWithVariants := make([]uint, 0, len(variantCounts))
	for _, vc := range variantCounts {
		if vc.Count > 0 {
			productsWithVariants = append(productsWithVariants, vc.ProductID)
		}
	}

	if len(productsWithVariants) == 0 {
		return result, nil
	}

	if err := r.loadBatchProductOptionsCounts(ctx, productsWithVariants, result); err != nil {
		return nil, err
	}
	if err := r.loadBatchOptionDerivedCounts(ctx, productsWithVariants, result); err != nil {
		return nil, err
	}
	if err := r.loadBatchAllVariantFlags(ctx, productsWithVariants, result); err != nil {
		return nil, err
	}
	if err := r.loadBatchOptionDerivedPriceRanges(ctx, productsWithVariants, result); err != nil {
		return nil, err
	}
	if err := r.loadBatchOptionPreview(ctx, productsWithVariants, result); err != nil {
		return nil, err
	}

	if userID != nil {
		var wishlistedProducts []struct {
			ProductID uint
		}
		if err := db.DB(ctx).
			Raw(productQuery.WISHLIST_CHECK_MULTIPLE_PRODUCTS, productsWithVariants, *userID).
			Scan(&wishlistedProducts).Error; err != nil {
			return nil, err
		}
		for _, wp := range wishlistedProducts {
			if result[wp.ProductID] != nil {
				result[wp.ProductID].IsWishlisted = true
			}
		}
	}

	for _, productID := range productsWithVariants {
		productUtils.ApplyAggregationSemantics(result[productID])
	}

	return result, nil
}

func (r *VariantRepositoryImpl) loadProductOptionsCount(
	ctx context.Context,
	productID uint,
	aggregation *mapper.VariantAggregation,
) error {
	var count int64
	if err := db.DB(ctx).Model(&entity.ProductOption{}).
		Where("product_id = ?", productID).
		Count(&count).Error; err != nil {
		return err
	}
	aggregation.ProductOptionsCount = int(count)
	return nil
}

func (r *VariantRepositoryImpl) loadOptionDerivedCount(
	ctx context.Context,
	productID uint,
	aggregation *mapper.VariantAggregation,
) error {
	var count int64
	err := db.DB(ctx).Table("variant_option_value vov").
		Joins("JOIN product_variant pv ON pv.id = vov.variant_id").
		Where("pv.product_id = ?", productID).
		Distinct("vov.variant_id").
		Count(&count).Error
	if err != nil {
		return err
	}
	aggregation.OptionDerivedCount = int(count)
	return nil
}

func (r *VariantRepositoryImpl) loadAllVariantFlags(
	ctx context.Context,
	productID uint,
	aggregation *mapper.VariantAggregation,
) error {
	var flags struct {
		DefaultPrice  float64
		AllowPurchase bool
		IsPopular     bool
	}
	err := db.DB(ctx).Model(&entity.ProductVariant{}).
		Select(productQuery.VARIANT_ALL_FLAGS_AGGREGATION_QUERY).
		Where("product_id = ?", productID).
		Scan(&flags).Error
	if err != nil {
		return err
	}
	aggregation.DefaultPrice = flags.DefaultPrice
	aggregation.AllowPurchase = flags.AllowPurchase
	aggregation.IsPopular = flags.IsPopular
	return nil
}

func (r *VariantRepositoryImpl) loadOptionDerivedPriceRange(
	ctx context.Context,
	productID uint,
	aggregation *mapper.VariantAggregation,
) error {
	var priceAgg struct {
		MinPrice float64
		MaxPrice float64
	}
	err := db.DB(ctx).Table("product_variant pv").
		Select(productQuery.VARIANT_OPTION_DERIVED_PRICE_AGGREGATION_QUERY).
		Joins(`INNER JOIN variant_option_value vov ON vov.variant_id = pv.id`).
		Where("pv.product_id = ?", productID).
		Scan(&priceAgg).Error
	if err != nil {
		return err
	}
	if aggregation.OptionDerivedCount > 0 {
		aggregation.MinPrice = priceAgg.MinPrice
		aggregation.MaxPrice = priceAgg.MaxPrice
	}
	return nil
}

func (r *VariantRepositoryImpl) loadOptionPreviewForProduct(
	ctx context.Context,
	productID uint,
	aggregation *mapper.VariantAggregation,
) error {
	var optionData []struct {
		OptionName  string
		OptionValue string
	}
	err := db.DB(ctx).Table("variant_option_value vov").
		Select("po.name as option_name, pov.value as option_value").
		Joins("JOIN product_option_value pov ON vov.option_value_id = pov.id").
		Joins("JOIN product_option po ON pov.option_id = po.id").
		Joins("JOIN product_variant pv ON vov.variant_id = pv.id").
		Where("pv.product_id = ?", productID).
		Group("po.name, pov.value").
		Order("po.name, pov.value").
		Scan(&optionData).Error
	if err != nil {
		return err
	}

	optionNamesSet := make(map[string]bool)
	for _, od := range optionData {
		optionNamesSet[od.OptionName] = true
		if _, exists := aggregation.OptionValues[od.OptionName]; !exists {
			aggregation.OptionValues[od.OptionName] = []string{}
		}
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
	for name := range optionNamesSet {
		aggregation.OptionNames = append(aggregation.OptionNames, name)
	}
	return nil
}

func (r *VariantRepositoryImpl) loadBatchProductOptionsCounts(
	ctx context.Context,
	productIDs []uint,
	result map[uint]*mapper.VariantAggregation,
) error {
	var rows []struct {
		ProductID uint
		Count     int64
	}
	err := db.DB(ctx).Model(&entity.ProductOption{}).
		Select("product_id, COUNT(*) as count").
		Where("product_id IN ?", productIDs).
		Group("product_id").
		Scan(&rows).Error
	if err != nil {
		return err
	}
	for _, row := range rows {
		if result[row.ProductID] != nil {
			result[row.ProductID].ProductOptionsCount = int(row.Count)
		}
	}
	return nil
}

func (r *VariantRepositoryImpl) loadBatchOptionDerivedCounts(
	ctx context.Context,
	productIDs []uint,
	result map[uint]*mapper.VariantAggregation,
) error {
	var rows []struct {
		ProductID uint
		Count     int64
	}
	err := db.DB(ctx).Table("variant_option_value vov").
		Select("pv.product_id, COUNT(DISTINCT vov.variant_id) as count").
		Joins("JOIN product_variant pv ON pv.id = vov.variant_id").
		Where("pv.product_id IN ?", productIDs).
		Group("pv.product_id").
		Scan(&rows).Error
	if err != nil {
		return err
	}
	for _, row := range rows {
		if result[row.ProductID] != nil {
			result[row.ProductID].OptionDerivedCount = int(row.Count)
		}
	}
	return nil
}

func (r *VariantRepositoryImpl) loadBatchAllVariantFlags(
	ctx context.Context,
	productIDs []uint,
	result map[uint]*mapper.VariantAggregation,
) error {
	var rows []struct {
		ProductID     uint
		DefaultPrice  float64
		AllowPurchase bool
		IsPopular     bool
	}
	err := db.DB(ctx).Model(&entity.ProductVariant{}).
		Select(productQuery.VARIANT_BATCH_ALL_FLAGS_AGGREGATION_QUERY).
		Where("product_id IN ?", productIDs).
		Group("product_id").
		Scan(&rows).Error
	if err != nil {
		return err
	}
	for _, row := range rows {
		if result[row.ProductID] != nil {
			result[row.ProductID].DefaultPrice = row.DefaultPrice
			result[row.ProductID].AllowPurchase = row.AllowPurchase
			result[row.ProductID].IsPopular = row.IsPopular
		}
	}
	return nil
}

func (r *VariantRepositoryImpl) loadBatchOptionDerivedPriceRanges(
	ctx context.Context,
	productIDs []uint,
	result map[uint]*mapper.VariantAggregation,
) error {
	var rows []struct {
		ProductID uint
		MinPrice  float64
		MaxPrice  float64
	}
	err := db.DB(ctx).Table("product_variant pv").
		Select(productQuery.VARIANT_BATCH_OPTION_DERIVED_PRICE_AGGREGATION_QUERY).
		Joins(`INNER JOIN variant_option_value vov ON vov.variant_id = pv.id`).
		Where("pv.product_id IN ?", productIDs).
		Group("pv.product_id").
		Scan(&rows).Error
	if err != nil {
		return err
	}
	for _, row := range rows {
		if result[row.ProductID] != nil && result[row.ProductID].OptionDerivedCount > 0 {
			result[row.ProductID].MinPrice = row.MinPrice
			result[row.ProductID].MaxPrice = row.MaxPrice
		}
	}
	return nil
}

func (r *VariantRepositoryImpl) loadBatchOptionPreview(
	ctx context.Context,
	productIDs []uint,
	result map[uint]*mapper.VariantAggregation,
) error {
	var optionData []struct {
		ProductID   uint
		OptionName  string
		OptionValue string
	}
	err := db.DB(ctx).Table("variant_option_value vov").
		Select("pv.product_id as product_id, po.name as option_name, pov.value as option_value").
		Joins("JOIN product_option_value pov ON vov.option_value_id = pov.id").
		Joins("JOIN product_option po ON pov.option_id = po.id").
		Joins("JOIN product_variant pv ON vov.variant_id = pv.id").
		Where("pv.product_id IN ?", productIDs).
		Group("pv.product_id, po.name, pov.value").
		Order("pv.product_id, po.name, pov.value").
		Scan(&optionData).Error
	if err != nil {
		return err
	}

	optionNamesMap := make(map[uint]map[string]bool)
	for _, od := range optionData {
		if result[od.ProductID] == nil {
			continue
		}
		if _, exists := optionNamesMap[od.ProductID]; !exists {
			optionNamesMap[od.ProductID] = make(map[string]bool)
		}
		optionNamesMap[od.ProductID][od.OptionName] = true
		if _, exists := result[od.ProductID].OptionValues[od.OptionName]; !exists {
			result[od.ProductID].OptionValues[od.OptionName] = []string{}
		}
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
	for productID, optionNames := range optionNamesMap {
		if result[productID] != nil {
			for name := range optionNames {
				result[productID].OptionNames = append(result[productID].OptionNames, name)
			}
		}
	}
	return nil
}

func (r *VariantRepositoryImpl) GetProductVariantsWithOptions(
	ctx context.Context,
	productID uint,
) ([]mapper.VariantWithOptions, error) {
	// First, get all variants for the product
	var variants []entity.ProductVariant
	if err := db.DB(ctx).Where("product_id = ?", productID).Find(&variants).Error; err != nil {
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
	err := db.DB(ctx).Table("variant_option_value AS vov").
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

// FindVariantsByProductID retrieves all variants for a product
func (r *VariantRepositoryImpl) FindVariantsByProductID(
	ctx context.Context,
	productID uint,
) ([]entity.ProductVariant, error) {
	var variants []entity.ProductVariant
	err := db.DB(ctx).Where("product_id = ?", productID).Find(&variants).Error
	return variants, err
}

// DeleteVariantsByProductID deletes all variants for a product
func (r *VariantRepositoryImpl) DeleteVariantsByProductID(
	ctx context.Context,
	productID uint,
) error {
	return db.DB(ctx).Where("product_id = ?", productID).Delete(&entity.ProductVariant{}).Error
}

// DeleteVariantOptionValuesByVariantIDs deletes all variant option values for given variant IDs
func (r *VariantRepositoryImpl) DeleteVariantOptionValuesByVariantIDs(
	ctx context.Context,
	variantIDs []uint,
) error {
	if len(variantIDs) == 0 {
		return nil
	}
	return db.DB(ctx).
		Where("variant_id IN ?", variantIDs).
		Delete(&entity.VariantOptionValue{}).
		Error
}

// ListVariantsWithFilters retrieves variants with comprehensive filtering and pagination
func (r *VariantRepositoryImpl) ListVariantsWithFilters(
	ctx context.Context,
	filters *model.ListVariantsRequest,
	sellerID *uint,
	optionFilters map[string]string,
) ([]mapper.VariantWithOptions, int64, error) {
	// Build query with all filters
	query := r.buildVariantFilterQuery(ctx, filters, sellerID, optionFilters)

	// Get total count before pagination
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting and pagination
	query = r.applySortingAndPagination(query, filters)

	// Fetch variants
	variants, err := r.fetchVariants(query)
	if err != nil {
		return nil, 0, err
	}

	if len(variants) == 0 {
		return []mapper.VariantWithOptions{}, total, nil
	}

	// Fetch options for variants
	result, err := r.enrichVariantsWithOptions(ctx, variants)
	if err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

// buildVariantFilterQuery constructs the base query with all filters applied
func (r *VariantRepositoryImpl) buildVariantFilterQuery(
	ctx context.Context,
	filters *model.ListVariantsRequest,
	sellerID *uint,
	optionFilters map[string]string,
) *gorm.DB {
	query := db.DB(ctx).Model(&entity.ProductVariant{})

	// Handle JOIN with product table if we need it for filtering by category or seller
	needsProductJoin := sellerID != nil || strings.TrimSpace(filters.CategoryIDs) != ""

	if needsProductJoin {
		query = query.Joins("JOIN product ON product_variant.product_id = product.id")
	}

	// Apply seller filter (multi-tenancy)
	if sellerID != nil {
		query = query.Where("product.seller_id = ?", *sellerID)
	}

	// Apply Category filter
	if strings.TrimSpace(filters.CategoryIDs) != "" {
		categoryIDs := helper.ParseCommaSeparated[uint](filters.CategoryIDs)
		if len(categoryIDs) > 0 {
			query = query.Where("product.category_id IN ?", categoryIDs)
		}
	}

	// Apply ID and product filters
	query = r.applyIDFilters(query, filters)

	// Apply price and status filters
	query = r.applyPriceAndStatusFilters(query, filters)

	// Apply SKU search
	if filters.SKU != "" {
		query = query.Where("product_variant.sku ILIKE ?", "%"+filters.SKU+"%")
	}

	// Apply option filters
	query = r.applyOptionFilters(query, optionFilters)

	return query
}

// applyIDFilters applies variant ID and product ID filters
func (r *VariantRepositoryImpl) applyIDFilters(
	query *gorm.DB,
	filters *model.ListVariantsRequest,
) *gorm.DB {
	// Apply variant ID filter
	if strings.TrimSpace(filters.IDs) != "" {
		ids := helper.ParseCommaSeparated[uint](filters.IDs)
		if len(ids) > 0 {
			query = query.Where("product_variant.id IN ?", ids)
		}
	}

	// Apply product ID filter
	if strings.TrimSpace(filters.ProductIDs) != "" {
		pids := helper.ParseCommaSeparated[uint](filters.ProductIDs)
		if len(pids) > 0 {
			query = query.Where("product_variant.product_id IN ?", pids)
		}
	}

	return query
}

// applyPriceAndStatusFilters applies price range and status filters
func (r *VariantRepositoryImpl) applyPriceAndStatusFilters(
	query *gorm.DB,
	filters *model.ListVariantsRequest,
) *gorm.DB {
	// Apply price range filters
	if filters.MinPrice != nil {
		query = query.Where("product_variant.price >= ?", *filters.MinPrice)
	}
	if filters.MaxPrice != nil {
		query = query.Where("product_variant.price <= ?", *filters.MaxPrice)
	}

	// Apply status filters
	if filters.AllowPurchase != nil {
		query = query.Where("product_variant.allow_purchase = ?", *filters.AllowPurchase)
	}
	if filters.IsPopular != nil {
		query = query.Where("product_variant.is_popular = ?", *filters.IsPopular)
	}
	if filters.IsDefault != nil {
		query = query.Where("product_variant.is_default = ?", *filters.IsDefault)
	}

	return query
}

// applyOptionFilters applies option-based filters (e.g., color=red, size=M)
func (r *VariantRepositoryImpl) applyOptionFilters(
	query *gorm.DB,
	optionFilters map[string]string,
) *gorm.DB {
	if len(optionFilters) == 0 {
		return query
	}

	// For each option filter, ensure the variant has that option value
	for optionName, optionValue := range optionFilters {
		query = query.Where(`
			EXISTS (
				SELECT 1 FROM variant_option_value vov
				JOIN product_option_value pov ON vov.option_value_id = pov.id
				JOIN product_option po ON pov.option_id = po.id
				WHERE vov.variant_id = product_variant.id
				AND po.name = ?
				AND pov.value = ?
			)
		`, optionName, optionValue)
	}

	return query
}

// applySortingAndPagination applies sorting and pagination to the query
func (r *VariantRepositoryImpl) applySortingAndPagination(
	query *gorm.DB,
	filters *model.ListVariantsRequest,
) *gorm.DB {
	// Apply sorting
	sortColumn := "product_variant." + filters.SortBy
	sortDirection := filters.SortOrder
	query = query.Order(sortColumn + " " + sortDirection)

	// Apply pagination
	offset := (filters.Page - 1) * filters.PageSize
	query = query.Limit(filters.PageSize).Offset(offset)

	return query
}

// fetchVariants executes the query and returns variants
func (r *VariantRepositoryImpl) fetchVariants(query *gorm.DB) ([]entity.ProductVariant, error) {
	var variants []entity.ProductVariant
	// Preload Product so CategoryID can be returned in the response mapping
	if err := query.Preload("Product").Find(&variants).Error; err != nil {
		return nil, err
	}
	return variants, nil
}

// enrichVariantsWithOptions fetches option values for variants and builds result
func (r *VariantRepositoryImpl) enrichVariantsWithOptions(
	ctx context.Context,
	variants []entity.ProductVariant,
) ([]mapper.VariantWithOptions, error) {
	// Extract variant IDs
	variantIDs := make([]uint, len(variants))
	for i := range variants {
		variantIDs[i] = variants[i].ID
	}

	// Batch fetch option values
	optionData, err := r.fetchVariantOptions(ctx, variantIDs)
	if err != nil {
		return nil, err
	}

	// Group options by variant ID
	variantOptionsMap := r.groupOptionsByVariantID(optionData)

	// Build final result
	return r.buildVariantWithOptionsResult(variants, variantOptionsMap), nil
}

// fetchVariantOptions retrieves option values for given variant IDs
func (r *VariantRepositoryImpl) fetchVariantOptions(
	ctx context.Context,
	variantIDs []uint,
) ([]mapper.OptionValueData, error) {
	var optionData []mapper.OptionValueData
	err := db.DB(ctx).Table("variant_option_value AS vov").
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

	return optionData, err
}

// groupOptionsByVariantID groups option values by variant ID
func (r *VariantRepositoryImpl) groupOptionsByVariantID(
	optionData []mapper.OptionValueData,
) map[uint][]mapper.SelectedOptionValue {
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

	return variantOptionsMap
}

// buildVariantWithOptionsResult builds final result with variants and their options
func (r *VariantRepositoryImpl) buildVariantWithOptionsResult(
	variants []entity.ProductVariant,
	variantOptionsMap map[uint][]mapper.SelectedOptionValue,
) []mapper.VariantWithOptions {
	result := make([]mapper.VariantWithOptions, 0, len(variants))

	for _, variant := range variants {
		result = append(result, mapper.VariantWithOptions{
			Variant:         variant,
			SelectedOptions: variantOptionsMap[variant.ID],
		})
	}

	return result
}

// GetProductCountByVariantIDs counts unique products from variant IDs
// Microservice-ready: Enables inventory service to count products without DB joins
// Filters by seller_id if provided for multi-tenant isolation
func (r *VariantRepositoryImpl) GetProductCountByVariantIDs(
	ctx context.Context,
	variantIDs []uint,
	sellerID *uint,
) (uint, error) {
	if len(variantIDs) == 0 {
		return 0, nil
	}

	// Query to count distinct product_ids from variant_ids
	var count int64
	query := db.DB(ctx).Model(&entity.ProductVariant{}).
		Where("product_variant.id IN ?", variantIDs)

	// Apply seller filter if provided (for multi-tenant isolation)
	if sellerID != nil {
		query = query.Joins("INNER JOIN product ON product.id = product_variant.product_id").
			Where("product.seller_id = ?", *sellerID)
	}

	err := query.Distinct("product_id").Count(&count).Error
	if err != nil {
		return 0, err
	}

	return uint(count), nil
}

// GetProductBasicInfoByVariantIDs retrieves basic product info for a list of variant IDs
// Microservice-ready: Enables inventory service to get product details without direct joins
// Returns flat rows with variant ID for in-memory grouping by product
func (r *VariantRepositoryImpl) GetProductBasicInfoByVariantIDs(
	ctx context.Context,
	variantIDs []uint,
	sellerID *uint,
) ([]mapper.VariantBasicInfoRow, error) {
	if len(variantIDs) == 0 {
		return []mapper.VariantBasicInfoRow{}, nil
	}

	var results []mapper.VariantBasicInfoRow

	query := db.DB(ctx).Model(&entity.ProductVariant{}).
		Select(
			"product_variant.id as variant_id",
			"product.id as product_id",
			"product.name as product_name",
			"product.category_id as category_id",
			"product.base_sku as base_sku",
			"product.seller_id as seller_id",
			"product_variant.price as price",
		).
		Joins("INNER JOIN product ON product.id = product_variant.product_id").
		Where("product_variant.id IN ?", variantIDs)

	// Apply seller filter if provided (for multi-tenant isolation)
	if sellerID != nil {
		query = query.Where("product.seller_id = ?", *sellerID)
	}

	err := query.Scan(&results).Error
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil for consistency
	if results == nil {
		results = []mapper.VariantBasicInfoRow{}
	}

	return results, nil
}
