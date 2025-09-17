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

// ConvertProductToDetailResponse converts Product entity to ProductDetailResponse model
func ConvertProductResponse(
	product *entity.Product,
	categoryInfo model.CategoryHierarchyInfo,
	attribute []model.ProductAttributeResponse,
	packageOption []model.PackageOptionResponse,
) *model.ProductResponse {
	return &model.ProductResponse{
		ID:               product.ID,
		Name:             product.Name,
		CategoryID:       product.CategoryID,
		Category:         categoryInfo,
		Brand:            product.Brand,
		SKU:              product.SKU,
		Price:            product.Price,
		Currency:         product.Currency,
		ShortDescription: product.ShortDescription,
		LongDescription:  product.LongDescription,
		Images:           product.Images,
		InStock:          product.InStock,
		IsPopular:        product.IsPopular,
		Discount:         product.Discount,
		Tags:             product.Tags,
		Attributes:       attribute,
		PackageOptions:   packageOption,
		CreatedAt:        product.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
	}
}

// ConvertProductToSearchResult converts Product entity to SearchResult model
func ConvertProductToSearchResult(product *entity.Product) *model.SearchResult {
	return &model.SearchResult{
		ID:               product.ID,
		Name:             product.Name,
		Price:            product.Price,
		ShortDescription: product.ShortDescription,
		Images:           product.Images,
		RelevanceScore:   0.8,                             // Placeholder - implement actual relevance scoring
		MatchedFields:    []string{"name", "description"}, // Placeholder
	}
}

// ConvertProductToRelatedProduct converts Product entity to RelatedProductResponse model
func ConvertProductToRelatedProduct(product *entity.Product) *model.RelatedProductResponse {
	return &model.RelatedProductResponse{
		ID:               product.ID,
		Name:             product.Name,
		Price:            product.Price,
		ShortDescription: product.ShortDescription,
		Images:           product.Images,
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

func ConvertProductCreateRequestToEntity(req model.ProductCreateRequest) *entity.Product {
	// TODO: This is not the correct way to set currency.
	// In the future, we will create separate tables for Currency and Country.
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}
	return &entity.Product{
		Name:             req.Name,
		CategoryID:       req.CategoryID,
		Brand:            req.Brand,
		SKU:              req.SKU,
		Price:            req.Price,
		Currency:         currency,
		ShortDescription: req.ShortDescription,
		LongDescription:  req.LongDescription,
		Images:           req.Images,
		InStock:          true,
		IsPopular:        req.IsPopular,
		Discount:         req.Discount,
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
