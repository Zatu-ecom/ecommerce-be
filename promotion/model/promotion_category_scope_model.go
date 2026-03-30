package model

import (
	"ecommerce-be/common/helper"
)

// PromotionCategoryResponse is the base response model for a promotion category
type PromotionCategoryResponse struct {
	BasePromotionScopeResponse
	CategoryID           uint   `json:"categoryId"`
	CategoryName         string `json:"categoryName,omitempty"`
	IncludeSubcategories bool   `json:"includeSubcategories"`
}

// AddPromotionCategoryItem represents a single category item in the add request
type AddPromotionCategoryItem struct {
	CategoryID           uint `json:"categoryId"           binding:"required"`
	IncludeSubcategories bool `json:"includeSubcategories"`
}

// AddPromotionCategoryRequest is the request to add categories to a promotion
type AddPromotionCategoryRequest struct {
	BasePromotionScopeRequest
	Categories []AddPromotionCategoryItem `json:"categories" binding:"required,min=1,dive"`
}

// RemovePromotionCategoryRequest is the request to remove categories from a promotion
type RemovePromotionCategoryRequest struct {
	BasePromotionScopeRequest
	CategoryIDs []uint `json:"categoryIds" binding:"required,min=1"`
}

// GetPromotionCategoriesRequest is the request to get categories for a promotion
type GetPromotionCategoriesRequest struct {
	GetPromotionScopeRequest
	CategoryIDs []uint `json:"categoryIds" form:"categoryIds"`
}

// GetPromotionCategoriesQueryParams is the query params for getting categories for a promotion
type GetPromotionCategoriesQueryParams struct {
	GetPromotionScopeRequest
	CategoryIDs *string `form:"categoryIds" binding:"omitempty"`
}

func (p *GetPromotionCategoriesQueryParams) ToRequest() GetPromotionCategoriesRequest {
	req := GetPromotionCategoriesRequest{
		GetPromotionScopeRequest: p.GetPromotionScopeRequest,
	}

	if p.CategoryIDs != nil {
		req.CategoryIDs = helper.ParseCommaSeparatedPtr[uint](p.CategoryIDs)
	}

	return req
}

// GetPromotionCategoriesResponse is the response for listing categories in a promotion
type GetPromotionCategoriesResponse struct {
	BasePromotionScopeResponse
	Categories []PromotionCategoryResponse `json:"categories"`
	Pagination PaginationResponse          `json:"pagination"`
}
