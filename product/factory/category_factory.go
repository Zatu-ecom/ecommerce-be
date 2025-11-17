package factory

import (
	"time"

	"ecommerce-be/product/entity"
	"ecommerce-be/product/model"
	"ecommerce-be/product/utils/helper"
)

// BuildCategoryEntityFromCreateRequest creates a Category entity from a create request
func BuildCategoryEntityFromCreateRequest(
	req model.CategoryCreateRequest,
	isGlobal bool,
	sellerID *uint,
) *entity.Category {
	if isGlobal {
		sellerID = nil
	}
	return &entity.Category{
		Name:        req.Name,
		ParentID:    req.ParentID,
		Description: req.Description,
		IsGlobal:    isGlobal,
		SellerID:    sellerID,
		BaseEntity:  helper.NewBaseEntity(),
	}
}

// BuildCategoryEntityFromUpdateReq updates an existing Category entity from an update request
func BuildCategoryEntityFromUpdateReq(
	category *entity.Category,
	req model.CategoryUpdateRequest,
) *entity.Category {
	category.Name = req.Name
	category.ParentID = req.ParentID
	category.Description = req.Description
	category.UpdatedAt = time.Now().UTC() // Force UTC

	return category
}

// BuildCategoryResponse builds CategoryResponse from entity
func BuildCategoryResponse(
	category *entity.Category,
) *model.CategoryResponse {
	var responseParentID *uint
	if category.ParentID != nil && *category.ParentID != 0 {
		responseParentID = category.ParentID
	}

	return &model.CategoryResponse{
		ID:          category.ID,
		Name:        category.Name,
		ParentID:    responseParentID,
		Description: category.Description,
		IsGlobal:    category.IsGlobal,
		SellerID:    category.SellerID,
		CreatedAt:   helper.FormatTimestamp(category.CreatedAt.UTC()),
		UpdatedAt:   helper.FormatTimestamp(category.UpdatedAt.UTC()),
	}
}

// BuildCategoryHierarchyResponse builds CategoryHierarchyResponse from entity (recursive)
func BuildCategoryHierarchyResponse(
	category *entity.Category,
) *model.CategoryHierarchyResponse {
	var responseParentID *uint
	if category.ParentID != nil && *category.ParentID != 0 {
		responseParentID = category.ParentID
	}

	// Convert children recursively
	children := make([]model.CategoryHierarchyResponse, 0, len(category.Children))
	for _, child := range category.Children {
		childResponse := BuildCategoryHierarchyResponse(&child)
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

// BuildCategoryHierarchyInfo builds CategoryHierarchyInfo from entity and parent
func BuildCategoryHierarchyInfo(
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
