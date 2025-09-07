package utils

import (
	"time"

	"ecommerce-be/product_management/entity"
	"ecommerce-be/product_management/model"
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
		IsActive:    category.IsActive,
		CreatedAt:   category.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   category.UpdatedAt.Format(time.RFC3339),
	}
}

// ConvertCategoryToHierarchyResponse converts Category entity to CategoryHierarchyResponse model
func ConvertCategoryToHierarchyResponse(category *entity.Category) *model.CategoryHierarchyResponse {
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
		IsActive:    category.IsActive,
		Children:    children,
	}
}

// ConvertAttributeDefinitionToResponse converts AttributeDefinition entity to AttributeDefinitionResponse model
func ConvertAttributeDefinitionToResponse(attribute *entity.AttributeDefinition) *model.AttributeDefinitionResponse {
	return &model.AttributeDefinitionResponse{
		ID:            attribute.ID,
		Key:           attribute.Key,
		Name:          attribute.Name,
		DataType:      attribute.DataType,
		Unit:          attribute.Unit,
		Description:   attribute.Description,
		AllowedValues: attribute.AllowedValues,
		IsActive:      attribute.IsActive,
		CreatedAt:     attribute.CreatedAt.Format(time.RFC3339),
	}
}

// ConvertProductToResponse converts Product entity to ProductResponse model
func ConvertProductToResponse(product *entity.Product) *model.ProductResponse {
	var categoryInfo model.CategoryInfo
	if product.Category != nil && product.Category.ID != 0 {
		categoryInfo = model.CategoryInfo{
			ID:   product.Category.ID,
			Name: product.Category.Name,
		}
	}

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
		IsActive:         product.IsActive,
		Discount:         product.Discount,
		Tags:             product.Tags,
		Attributes:       make(map[string]string),
		PackageOptions:   []model.PackageOptionResponse{},
		CreatedAt:        product.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
	}
}

// ConvertProductToDetailResponse converts Product entity to ProductDetailResponse model
func ConvertProductToDetailResponse(product *entity.Product, categoryInfo model.CategoryHierarchyInfo) *model.ProductDetailResponse {
	return &model.ProductDetailResponse{
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
		IsActive:         product.IsActive,
		Discount:         product.Discount,
		Tags:             product.Tags,
		Attributes:       []model.ProductAttributeResponse{},
		PackageOptions:   []model.PackageOptionResponse{},
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

// ConvertCategoryToFilter converts Category entity to CategoryFilter model
func ConvertCategoryToFilter(category *entity.Category, productCount int) *model.CategoryFilter {
	return &model.CategoryFilter{
		ID:           category.ID,
		Name:         category.Name,
		ProductCount: productCount,
	}
}

// ConvertCategoryToInfo converts Category entity to CategoryInfo model
func ConvertCategoryToInfo(category *entity.Category) *model.CategoryInfo {
	return &model.CategoryInfo{
		ID:   category.ID,
		Name: category.Name,
	}
}

// ConvertCategoryToHierarchyInfo converts Category entity to CategoryHierarchyInfo model
func ConvertCategoryToHierarchyInfo(category *entity.Category, parentCategory *entity.Category) *model.CategoryHierarchyInfo {
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
