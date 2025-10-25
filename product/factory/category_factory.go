package factory

import (
	"time"

	commonEntity "ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	"ecommerce-be/product/model"
)

// CategoryFactory handles the creation of category entities from requests
type CategoryFactory struct{}

// NewCategoryFactory creates a new instance of CategoryFactory
func NewCategoryFactory() *CategoryFactory {
	return &CategoryFactory{}
}

// CreateFromRequest creates a Category entity from a create request
func (f *CategoryFactory) CreateFromRequest(
	req model.CategoryCreateRequest,
	isGlobal bool,
	sellerID *uint,
) *entity.Category {
	now := time.Now()
	if isGlobal {
		sellerID = nil
	}
	return &entity.Category{
		Name:        req.Name,
		ParentID:    req.ParentID,
		Description: req.Description,
		IsGlobal:    isGlobal,
		SellerID:    sellerID,
		BaseEntity: commonEntity.BaseEntity{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// UpdateEntity updates an existing Category entity from an update request
func (f *CategoryFactory) UpdateEntity(
	category *entity.Category,
	req model.CategoryUpdateRequest,
) *entity.Category {
	category.Name = req.Name
	category.ParentID = req.ParentID
	category.Description = req.Description
	category.UpdatedAt = time.Now()

	return category
}

// BuildCategoryResponse builds CategoryResponse from entity
func (f *CategoryFactory) BuildCategoryResponse(
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
		CreatedAt:   category.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   category.UpdatedAt.Format(time.RFC3339),
	}
}

// BuildCategoryHierarchyResponse builds CategoryHierarchyResponse from entity (recursive)
func (f *CategoryFactory) BuildCategoryHierarchyResponse(
	category *entity.Category,
) *model.CategoryHierarchyResponse {
	var responseParentID *uint
	if category.ParentID != nil && *category.ParentID != 0 {
		responseParentID = category.ParentID
	}

	// Convert children recursively
	children := make([]model.CategoryHierarchyResponse, 0, len(category.Children))
	for _, child := range category.Children {
		childResponse := f.BuildCategoryHierarchyResponse(&child)
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
func (f *CategoryFactory) BuildCategoryHierarchyInfo(
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
