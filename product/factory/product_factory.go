package factory

import (
	"time"

	"ecommerce-be/product/entity"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
	"ecommerce-be/product/utils/helper"
)

// ProductFactory handles creation and updates of product entities
// Stateless factory - all methods are pure functions

// CreateProductFromRequest creates a new Product entity from a creation request
func CreateProductFromRequest(
	req model.ProductCreateRequest,
	sellerID uint,
) *entity.Product {
	return &entity.Product{
		Name:             req.Name,
		CategoryID:       req.CategoryID,
		Brand:            req.Brand,
		BaseSKU:          req.BaseSKU,
		ShortDescription: req.ShortDescription,
		LongDescription:  req.LongDescription,
		Tags:             req.Tags,
		SellerID:         sellerID,
		BaseEntity:       helper.NewBaseEntity(),
	}
}

// CreateProductEntityFromUpdateRequest updates an existing Product entity with new data
// Uses pointer fields to distinguish between null (don't update) and empty (clear field)
func CreateProductEntityFromUpdateRequest(
	product *entity.Product,
	req model.ProductUpdateRequest,
) *entity.Product {
	// Update basic fields if provided (not nil)
	// If pointer is not nil, update even if value is empty string/zero
	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Brand != nil {
		product.Brand = *req.Brand
	}
	if req.CategoryID != nil {
		product.CategoryID = *req.CategoryID
	}
	if req.ShortDescription != nil {
		product.ShortDescription = *req.ShortDescription
	}
	if req.LongDescription != nil {
		product.LongDescription = *req.LongDescription
	}
	if req.Tags != nil {
		product.Tags = *req.Tags
	}

	product.UpdatedAt = time.Now()
	return product
}

// CreateProductOptionsFromRequests creates ProductOption entities from requests
func CreateProductOptionsFromRequests(
	productID uint,
	optionReqs []model.ProductOptionCreateRequest,
) []*entity.ProductOption {
	options := make([]*entity.ProductOption, 0, len(optionReqs))

	for i, optionReq := range optionReqs {
		position := helper.GetPositionOrDefault(optionReq.Position, i+1)

		option := &entity.ProductOption{
			ProductID:   productID,
			Name:        optionReq.Name,
			DisplayName: optionReq.DisplayName,
			Position:    position,
		}
		options = append(options, option)
	}

	return options
}

// CreateProductAttributesFromRequests creates ProductAttribute entities from requests
func CreateProductAttributesFromRequests(
	productID uint,
	attributes []model.ProductAttributeRequest,
	attributeMap map[string]*entity.AttributeDefinition,
) []*entity.ProductAttribute {
	productAttributes := make([]*entity.ProductAttribute, 0, len(attributes))

	for _, attr := range attributes {
		attributeDefinition := attributeMap[attr.Key]

		productAttribute := &entity.ProductAttribute{
			ProductID:             productID,
			AttributeDefinitionID: attributeDefinition.ID,
			Value:                 attr.Value,
			SortOrder:             attr.SortOrder,
			AttributeDefinition:   attributeDefinition,
		}
		productAttributes = append(productAttributes, productAttribute)
	}

	return productAttributes
}

// CreateNewAttributeDefinition creates a new AttributeDefinition entity
func CreateNewAttributeDefinition(
	attr model.ProductAttributeRequest,
) *entity.AttributeDefinition {
	return &entity.AttributeDefinition{
		Key:           attr.Key,
		Name:          attr.Name,
		Unit:          attr.Unit,
		AllowedValues: []string{attr.Value},
	}
}

// UpdateAttributeDefinitionValues adds a new value to an existing AttributeDefinition
// Returns true if the value was added, false if it already existed
func UpdateAttributeDefinitionValues(
	attribute *entity.AttributeDefinition,
	newValue string,
) bool {
	// Check if value already exists using map for O(1) lookup
	valueMap := make(map[string]bool)
	for _, val := range attribute.AllowedValues {
		valueMap[val] = true
	}

	// Only add if value doesn't exist
	if !valueMap[newValue] {
		attribute.AllowedValues = append(attribute.AllowedValues, newValue)
		return true
	}
	return false
}

// CreatePackageOptionsFromRequests creates PackageOption entities from requests
func CreatePackageOptionsFromRequests(
	productID uint,
	options []model.PackageOptionRequest,
) []entity.PackageOption {
	packageOptions := make([]entity.PackageOption, 0, len(options))

	for _, option := range options {
		packageOption := entity.PackageOption{
			Name:        option.Name,
			Description: option.Description,
			Price:       option.Price,
			Quantity:    option.Quantity,
			ProductID:   productID,
			BaseEntity:  helper.NewBaseEntity(),
		}
		packageOptions = append(packageOptions, packageOption)
	}

	return packageOptions
}

// FlattenProductAttributes converts []*entity.ProductAttribute to []entity.ProductAttribute
func FlattenProductAttributes(
	attrs []*entity.ProductAttribute,
) []entity.ProductAttribute {
	result := make([]entity.ProductAttribute, 0, len(attrs))
	for _, attr := range attrs {
		result = append(result, *attr)
	}
	return result
}

/***********************************************
 *    Response Builders                         *
 ***********************************************/

// BuildPackageOptionResponse builds PackageOptionResponse from entity
func BuildPackageOptionResponse(
	packageOption *entity.PackageOption,
) *model.PackageOptionResponse {
	return &model.PackageOptionResponse{
		ID:          packageOption.ID,
		Name:        packageOption.Name,
		Description: packageOption.Description,
		Price:       packageOption.Price,
		Quantity:    packageOption.Quantity,
		CreatedAt:   helper.FormatTimestamp(packageOption.CreatedAt),
		UpdatedAt:   helper.FormatTimestamp(packageOption.UpdatedAt),
	}
}

// BuildPackageOptionResponses builds multiple PackageOptionResponse from entities
func BuildPackageOptionResponses(
	packageOptions []entity.PackageOption,
) []model.PackageOptionResponse {
	responses := make([]model.PackageOptionResponse, 0, len(packageOptions))
	for _, option := range packageOptions {
		responses = append(responses, *BuildPackageOptionResponse(&option))
	}
	return responses
}

// BuildCategoryFilter builds CategoryFilter from mapper data
func BuildCategoryFilter(
	category mapper.CategoryWithProductCount,
) model.CategoryFilter {
	return model.CategoryFilter{
		ID:           category.CategoryID,
		Name:         category.CategoryName,
		ProductCount: category.ProductCount,
	}
}

// BuildBrandFilter builds BrandFilter from mapper data
func BuildBrandFilter(brand mapper.BrandWithProductCount) model.BrandFilter {
	return model.BrandFilter{
		Brand:        brand.Brand,
		ProductCount: brand.ProductCount,
	}
}

// BuildBrandFilters builds multiple BrandFilter from mapper data
func BuildBrandFilters(
	brands []mapper.BrandWithProductCount,
) []model.BrandFilter {
	filters := make([]model.BrandFilter, 0, len(brands))
	for _, brand := range brands {
		filters = append(filters, BuildBrandFilter(brand))
	}
	return filters
}

// BuildAttributeFilter builds AttributeFilter from mapper data
func BuildAttributeFilter(
	attribute mapper.AttributeWithProductCount,
) model.AttributeFilter {
	return model.AttributeFilter{
		Key:           attribute.Key,
		Name:          attribute.Name,
		AllowedValues: attribute.AllowedValues,
		ProductCount:  attribute.ProductCount,
	}
}

// BuildAttributeFilters builds multiple AttributeFilter from mapper data
func BuildAttributeFilters(
	attributes []mapper.AttributeWithProductCount,
) []model.AttributeFilter {
	filters := make([]model.AttributeFilter, 0, len(attributes))
	for _, attribute := range attributes {
		filters = append(filters, BuildAttributeFilter(attribute))
	}
	return filters
}

// BuildPriceRangeFilter builds PriceRangeFilter from mapper data
func BuildPriceRangeFilter(
	data *mapper.PriceRangeData,
) *model.PriceRangeFilter {
	if data == nil || data.ProductCount == 0 {
		return nil
	}
	return &model.PriceRangeFilter{
		Min:          data.MinPrice,
		Max:          data.MaxPrice,
		ProductCount: data.ProductCount,
	}
}

// BuildVariantTypeFilters builds VariantTypeFilter list from variant options data
func BuildVariantTypeFilters(
	variantOptions []mapper.VariantOptionData,
) []model.VariantTypeFilter {
	// Group by option ID to create VariantTypeFilter
	optionMap := make(map[uint]*model.VariantTypeFilter)
	optionOrder := []uint{} // Preserve order

	for _, vo := range variantOptions {
		if _, exists := optionMap[vo.OptionID]; !exists {
			// First time seeing this option
			optionMap[vo.OptionID] = &model.VariantTypeFilter{
				Name:         vo.OptionName,
				DisplayName:  vo.OptionDisplayName,
				Values:       []model.VariantOptionFilter{},
				ProductCount: 0,
			}
			optionOrder = append(optionOrder, vo.OptionID)
		}

		// Add value to this option
		optionMap[vo.OptionID].Values = append(
			optionMap[vo.OptionID].Values,
			model.VariantOptionFilter{
				Value:        vo.OptionValue,
				DisplayName:  vo.ValueDisplayName,
				ColorCode:    vo.ColorCode,
				ProductCount: vo.ProductCount,
			},
		)

		// Update product count for this option type
		optionMap[vo.OptionID].ProductCount += vo.ProductCount
	}

	// Convert map to slice maintaining order
	result := make([]model.VariantTypeFilter, 0, len(optionMap))
	for _, optionID := range optionOrder {
		result = append(result, *optionMap[optionID])
	}

	return result
}

// BuildStockStatusFilter builds StockStatusFilter from mapper data
func BuildStockStatusFilter(
	data *mapper.StockStatusData,
) *model.StockStatusFilter {
	if data == nil || data.TotalProducts == 0 {
		return nil
	}
	return &model.StockStatusFilter{
		InStock:       data.InStock,
		OutOfStock:    data.OutOfStock,
		TotalProducts: data.TotalProducts,
	}
}

// BuildProductResponse builds a ProductResponse from product entity and variant aggregation
// Used for list views (GetAllProducts, GetRelatedProducts, etc.)
func BuildProductResponse(
	product *entity.Product,
	variantAgg *mapper.VariantAggregation,
) model.ProductResponse {
	// Build category hierarchy using existing helper method
	categoryInfo := BuildCategoryHierarchyInfo(product.Category, product.Category.Parent)

	// Build base product response
	productResp := model.ProductResponse{
		ID:               product.ID,
		Name:             product.Name,
		CategoryID:       product.CategoryID,
		Category:         *categoryInfo,
		Brand:            product.Brand,
		SKU:              product.BaseSKU,
		ShortDescription: product.ShortDescription,
		LongDescription:  product.LongDescription,
		Tags:             product.Tags,
		SellerID:         product.SellerID,
		HasVariants:      variantAgg.HasVariants,
		AllowPurchase:    variantAgg.AllowPurchase,
		CreatedAt:        helper.FormatTimestamp(product.CreatedAt),
		UpdatedAt:        helper.FormatTimestamp(product.UpdatedAt),
	}

	// Set price range if product has variants
	if variantAgg.HasVariants {
		productResp.PriceRange = &model.PriceRange{
			Min: variantAgg.MinPrice,
			Max: variantAgg.MaxPrice,
		}
	}

	// Add main product image if available
	if variantAgg.MainImage != "" {
		productResp.Images = []string{variantAgg.MainImage}
	}

	// Build variant preview for product listings
	if variantAgg.TotalVariants > 0 {
		variantPreview := &model.VariantPreview{
			TotalVariants: variantAgg.TotalVariants,
			Options:       []model.OptionPreview{},
		}

		// Add available option values for each option
		for _, optionName := range variantAgg.OptionNames {
			optionValues := variantAgg.OptionValues[optionName]
			variantPreview.Options = append(variantPreview.Options, model.OptionPreview{
				Name:            optionName,
				DisplayName:     optionName,
				AvailableValues: optionValues,
			})
		}

		productResp.VariantPreview = variantPreview
	}

	return productResp
}

// BuildRelatedProductItemScored builds a RelatedProductItemScored from RelatedProductScored mapper
// Used for related products API with scoring information
func BuildRelatedProductItemScored(
	scoredResult *mapper.RelatedProductScored,
	optionsPreview []model.OptionPreview,
) model.RelatedProductItemScored {
	// Build category hierarchy info
	var parentInfo *model.CategoryInfo
	if scoredResult.ParentCategoryID != nil && scoredResult.ParentCategoryName != nil {
		parentInfo = &model.CategoryInfo{
			ID:   *scoredResult.ParentCategoryID,
			Name: *scoredResult.ParentCategoryName,
		}
	}

	categoryInfo := model.CategoryHierarchyInfo{
		ID:     scoredResult.CategoryID,
		Name:   scoredResult.CategoryName,
		Parent: parentInfo,
	}

	// Build base product response
	productResponse := model.ProductResponse{
		ID:               scoredResult.ProductID,
		Name:             scoredResult.ProductName,
		CategoryID:       scoredResult.CategoryID,
		Category:         categoryInfo,
		Brand:            scoredResult.Brand,
		SKU:              scoredResult.SKU,
		ShortDescription: scoredResult.ShortDescription,
		LongDescription:  scoredResult.LongDescription,
		Tags:             scoredResult.Tags,
		SellerID:         scoredResult.SellerID,
		HasVariants:      scoredResult.HasVariants,
		PriceRange: &model.PriceRange{
			Min: scoredResult.MinPrice,
			Max: scoredResult.MaxPrice,
		},
		AllowPurchase: scoredResult.AllowPurchase,
		Images:        []string{},
	}

	// Add variant preview if available
	if scoredResult.TotalVariants > 0 && len(optionsPreview) > 0 {
		productResponse.VariantPreview = &model.VariantPreview{
			TotalVariants: int(scoredResult.TotalVariants),
			Options:       optionsPreview,
		}
	}

	// Build scored item with relation metadata
	return model.RelatedProductItemScored{
		ProductResponse: productResponse,
		RelationReason:  scoredResult.RelationReason,
		Score:           scoredResult.FinalScore,
		StrategyUsed:    scoredResult.StrategyUsed,
	}
}

// BuildOptionsPreview builds option preview list from product options
// Reusable helper for variant preview construction
func BuildOptionsPreview(options []entity.ProductOption) []model.OptionPreview {
	optionsPreview := make([]model.OptionPreview, 0, len(options))

	for _, option := range options {
		// Extract available values from option values
		availableValues := make([]string, 0, len(option.Values))
		for _, val := range option.Values {
			availableValues = append(availableValues, val.Value)
		}

		optionsPreview = append(optionsPreview, model.OptionPreview{
			Name:            option.Name,
			DisplayName:     option.DisplayName,
			AvailableValues: availableValues,
		})
	}

	return optionsPreview
}

func BuildRelatedProductsScoredResponse(
	relatedItems []model.RelatedProductItemScored,
	strategiesUsed []string,
	avgScore float64,
	pagination model.PaginationResponse,
	totalStrategies int,
) *model.RelatedProductsScoredResponse {
	return &model.RelatedProductsScoredResponse{
		RelatedProducts: relatedItems,
		Pagination:      pagination,
		Meta: model.RelatedProductsMeta{
			StrategiesUsed:  strategiesUsed,
			AvgScore:        avgScore,
			TotalStrategies: totalStrategies,
		},
	}
}