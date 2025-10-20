package utils

import (
	"time"

	commonEntity "ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
)

// ConvertCategoryToResponse converts Category entity to CategoryResponse model
func ConvertCategoryToResponse(category *entity.Category) *model.CategoryResponse {
	var responseParentID *uint
	if category.ParentID != nil && *category.ParentID != 0 {
		responseParentID = category.ParentID
	}

	return &model.CategoryResponse{
		ID:          category.ID,
		Name:        category.Name,
		ParentID:    responseParentID,
		Description: category.Description,
		CreatedAt:   category.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   category.UpdatedAt.Format(time.RFC3339),
	}
}

// ConvertCategoryToHierarchyResponse converts Category entity to CategoryHierarchyResponse model
func ConvertCategoryToHierarchyResponse(
	category *entity.Category,
) *model.CategoryHierarchyResponse {
	var responseParentID *uint
	if category.ParentID != nil && *category.ParentID != 0 {
		responseParentID = category.ParentID
	}

	// Convert children recursively
	var children []model.CategoryHierarchyResponse
	for _, child := range category.Children {
		childResponse := ConvertCategoryToHierarchyResponse(&child)
		children = append(children, *childResponse)
	}

	return &model.CategoryHierarchyResponse{
		ID:          category.ID,
		Name:        category.Name,
		ParentID:    responseParentID,
		Description: category.Description,
		Children:    children,
	}
}

// ConvertAttributeDefinitionToResponse converts AttributeDefinition entity to AttributeDefinitionResponse model
func ConvertAttributeDefinitionToResponse(
	attribute *entity.AttributeDefinition,
) *model.AttributeDefinitionResponse {
	return &model.AttributeDefinitionResponse{
		ID:            attribute.ID,
		Key:           attribute.Key,
		Name:          attribute.Name,
		Unit:          attribute.Unit,
		AllowedValues: attribute.AllowedValues,
		CreatedAt:     attribute.CreatedAt.Format(time.RFC3339),
	}
}

func ConvertProductAttributeDefinitionToResponse(
	productAttribute *entity.ProductAttribute,
) *model.ProductAttributeResponse {
	return &model.ProductAttributeResponse{
		ID:        productAttribute.ID,
		Key:       productAttribute.AttributeDefinition.Key,
		Value:     productAttribute.Value,
		Name:      productAttribute.AttributeDefinition.Name,
		Unit:      productAttribute.AttributeDefinition.Unit,
		SortOrder: productAttribute.SortOrder,
	}
}

// ConvertProductToDetailResponse - DEPRECATED
// Products now require variant data. Use service layer methods instead.
// This converter is kept for backward compatibility but should not be used.
func ConvertProductResponse(
	product *entity.Product,
	categoryInfo model.CategoryHierarchyInfo,
	attribute []model.ProductAttributeResponse,
	packageOption []model.PackageOptionResponse,
) *model.ProductResponse {
	// Return minimal response - actual implementation in service layer
	return &model.ProductResponse{
		ID:               product.ID,
		Name:             product.Name,
		CategoryID:       product.CategoryID,
		Category:         categoryInfo,
		Brand:            product.Brand,
		SKU:              product.BaseSKU,
		ShortDescription: product.ShortDescription,
		LongDescription:  product.LongDescription,
		Tags:             product.Tags,
		Attributes:       attribute,
		PackageOptions:   packageOption,
		CreatedAt:        product.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
		// Variant fields must be populated by service layer
	}
}

// ConvertProductToSearchResult - DEPRECATED
// Use service layer method that includes variant data
func ConvertProductToSearchResult(product *entity.Product) *model.SearchResult {
	return &model.SearchResult{
		ID:               product.ID,
		Name:             product.Name,
		Price:            0, // Must be fetched from variants
		ShortDescription: product.ShortDescription,
		Images:           []string{}, // Must be fetched from variants
		RelevanceScore:   0.8,
		MatchedFields:    []string{"name", "description"},
	}
}

// ConvertProductToRelatedProduct - DEPRECATED
// Use service layer method that includes variant data
func ConvertProductToRelatedProduct(product *entity.Product) *model.RelatedProductResponse {
	return &model.RelatedProductResponse{
		ID:               product.ID,
		Name:             product.Name,
		Price:            0, // Must be fetched from variants
		ShortDescription: product.ShortDescription,
		Images:           []string{}, // Must be fetched from variants
		RelationReason:   "Same category",
	}
}

// ConvertCategoryToHierarchyInfo converts Category entity to CategoryHierarchyInfo model
func ConvertCategoryToHierarchyInfo(
	category *entity.Category,
	parentCategory *entity.Category,
) *model.CategoryHierarchyInfo {
	var parentInfo *model.CategoryInfo
	if parentCategory != nil {
		parentInfo = &model.CategoryInfo{
			ID:   parentCategory.ID,
			Name: parentCategory.Name,
		}
	}

	return &model.CategoryHierarchyInfo{
		ID:     category.ID,
		Name:   category.Name,
		Parent: parentInfo,
	}
}

func ConvertPackageOptionToResponse(
	packageOption *entity.PackageOption,
) *model.PackageOptionResponse {
	return &model.PackageOptionResponse{
		ID:          packageOption.ID,
		Name:        packageOption.Name,
		Description: packageOption.Description,
		Price:       packageOption.Price,
		Quantity:    packageOption.Quantity,
		CreatedAt:   packageOption.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   packageOption.UpdatedAt.Format(time.RFC3339),
	}
}

// ConvertProductCreateRequestToEntity - DEPRECATED
// Product creation now handled in service layer with variant creation
// This function is kept for backward compatibility but should not be used
func ConvertProductCreateRequestToEntity(req model.ProductCreateRequest) *entity.Product {
	return &entity.Product{
		Name:             req.Name,
		CategoryID:       req.CategoryID,
		Brand:            req.Brand,
		BaseSKU:          req.BaseSKU,
		ShortDescription: req.ShortDescription,
		LongDescription:  req.LongDescription,
		Tags:             req.Tags,
		BaseEntity: commonEntity.BaseEntity{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
}

func ConvertProductAttributesEntityToResponse(
	productAttributes []entity.ProductAttribute,
) []model.ProductAttributeResponse {
	var attribute []model.ProductAttributeResponse
	for _, productAttribute := range productAttributes {
		attribute = append(
			attribute,
			*ConvertProductAttributeDefinitionToResponse(&productAttribute),
		)
	}
	return attribute
}

func ConvertPackageOptionsEntityToResponse(
	packageOptions []entity.PackageOption,
) []model.PackageOptionResponse {
	var options []model.PackageOptionResponse
	for _, option := range packageOptions {
		options = append(options, *ConvertPackageOptionToResponse(&option))
	}
	return options
}

func ConvertCategoriesToFilters(category mapper.CategoryWithProductCount) model.CategoryFilter {
	return model.CategoryFilter{
		ID:           category.CategoryID,
		Name:         category.CategoryName,
		ProductCount: category.ProductCount,
	}
}

func ConvertBrandsToFilters(brands []mapper.BrandWithProductCount) []model.BrandFilter {
	var brandFilters []model.BrandFilter
	for _, brand := range brands {
		brandFilters = append(brandFilters, ConvertBrandsToFilter(brand))
	}
	return brandFilters
}

func ConvertAttributesToFilters(
	attributes []mapper.AttributeWithProductCount,
) []model.AttributeFilter {
	var attributeFilters []model.AttributeFilter
	for _, attribute := range attributes {
		attributeFilters = append(attributeFilters, ConvertAttributesToFilter(attribute))
	}
	return attributeFilters
}

func ConvertBrandsToFilter(brand mapper.BrandWithProductCount) model.BrandFilter {
	return model.BrandFilter{
		Brand:        brand.Brand,
		ProductCount: brand.ProductCount,
	}
}

func ConvertAttributesToFilter(attribute mapper.AttributeWithProductCount) model.AttributeFilter {
	return model.AttributeFilter{
		Key:           attribute.Key,
		Name:          attribute.Name,
		AllowedValues: attribute.AllowedValues,
		ProductCount:  attribute.ProductCount,
	}
}

// ConvertProductOptionToResponse converts ProductOption entity to ProductOptionResponse model
func ConvertProductOptionToResponse(
	option *entity.ProductOption,
	productID uint,
) *model.ProductOptionResponse {
	response := &model.ProductOptionResponse{
		ID:          option.ID,
		ProductID:   productID,
		Name:        option.Name,
		DisplayName: option.DisplayName,
		Position:    option.Position,
		CreatedAt:   option.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   option.UpdatedAt.Format(time.RFC3339),
	}

	// Convert values
	if len(option.Values) > 0 {
		values := make([]model.ProductOptionValueResponse, 0, len(option.Values))
		for _, val := range option.Values {
			valueResp := ConvertProductOptionValueToResponse(&val)
			values = append(values, *valueResp)
		}
		response.Values = values
	}

	return response
}

// ConvertProductOptionValueToResponse converts ProductOptionValue entity to ProductOptionValueResponse model
func ConvertProductOptionValueToResponse(
	value *entity.ProductOptionValue,
) *model.ProductOptionValueResponse {
	return &model.ProductOptionValueResponse{
		ID:          value.ID,
		OptionID:    value.OptionID,
		Value:       value.Value,
		DisplayName: value.DisplayName,
		ColorCode:   value.ColorCode,
		Position:    value.Position,
		CreatedAt:   value.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   value.UpdatedAt.Format(time.RFC3339),
	}
}

// ConvertProductOptionValueRequestToEntity converts ProductOptionValueRequest to ProductOptionValue entity
func ConvertProductOptionValueRequestToEntity(
	req model.ProductOptionValueRequest,
	optionID uint,
) entity.ProductOptionValue {
	return entity.ProductOptionValue{
		OptionID:    optionID,
		Value:       req.Value,
		DisplayName: req.DisplayName,
		ColorCode:   req.ColorCode,
		Position:    req.Position,
	}
}

/***********************************************
 *    Variant Converters                        *
 ***********************************************/

// ConvertVariantToDetailResponse converts a variant entity to a detailed response
func ConvertVariantToDetailResponse(
	variant *entity.ProductVariant,
	product *entity.Product,
	selectedOptions []model.VariantOptionResponse,
) *model.VariantDetailResponse {
	images := []string{}
	if variant.Images != nil {
		images = variant.Images
	}

	response := &model.VariantDetailResponse{
		ID:              variant.ID,
		ProductID:       variant.ProductID,
		SKU:             variant.SKU,
		Price:           variant.Price,
		Stock:           variant.Stock,
		InStock:         variant.InStock,
		Images:          images,
		IsDefault:       variant.IsDefault,
		IsPopular:       variant.IsPopular,
		SelectedOptions: selectedOptions,
		CreatedAt:       variant.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       variant.UpdatedAt.Format(time.RFC3339),
	}

	// Add product basic info
	if product != nil {
		response.Product = model.ProductBasicInfo{
			ID:    product.ID,
			Name:  product.Name,
			Brand: product.Brand,
		}
	}

	return response
}

// ConvertVariantToResponse converts a variant entity to a basic response
func ConvertVariantToResponse(
	variant *entity.ProductVariant,
	selectedOptions []model.VariantOptionResponse,
) *model.VariantResponse {
	images := []string{}
	if variant.Images != nil {
		images = variant.Images
	}

	return &model.VariantResponse{
		ID:              variant.ID,
		SKU:             variant.SKU,
		Price:           variant.Price,
		Stock:           variant.Stock,
		InStock:         variant.InStock,
		Images:          images,
		IsDefault:       variant.IsDefault,
		IsPopular:       variant.IsPopular,
		SelectedOptions: selectedOptions,
	}
}

// ConvertVariantOptionValues converts variant option values to response objects
func ConvertVariantOptionValues(
	variantOptionValues []entity.VariantOptionValue,
	productOptions []entity.ProductOption,
	optionValues []entity.ProductOptionValue,
) []model.VariantOptionResponse {
	optionResponses := []model.VariantOptionResponse{}

	// Create maps for quick lookup
	optionMap := make(map[uint]entity.ProductOption)
	for _, opt := range productOptions {
		optionMap[opt.ID] = opt
	}

	valueMap := make(map[uint]entity.ProductOptionValue)
	for _, val := range optionValues {
		valueMap[val.ID] = val
	}

	for _, vov := range variantOptionValues {
		option, optionExists := optionMap[vov.OptionID]
		value, valueExists := valueMap[vov.OptionValueID]

		if optionExists && valueExists {
			optionResponse := model.VariantOptionResponse{
				OptionID:          option.ID,
				OptionName:        option.Name,
				OptionDisplayName: GetDisplayNameOrDefault(option.DisplayName, option.Name),
				ValueID:           value.ID,
				Value:             value.Value,
				ValueDisplayName:  GetDisplayNameOrDefault(value.DisplayName, value.Value),
			}

			// Add color code if it exists
			if value.ColorCode != "" {
				optionResponse.ColorCode = value.ColorCode
			}

			optionResponses = append(optionResponses, optionResponse)
		}
	}

	return optionResponses
}
