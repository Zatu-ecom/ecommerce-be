package model

import (
	"ecommerce-be/common/helper"
)

// DiscountCodeCategoryResponse is the base response model for a discount-code category
type DiscountCodeCategoryResponse struct {
	BaseDiscountCodeScopeResponse
	CategoryID           uint   `json:"categoryId"`
	CategoryName         string `json:"categoryName,omitempty"`
	IncludeSubcategories bool   `json:"includeSubcategories"`
}

// AddDiscountCodeCategoryItem represents a single category item in the add request
type AddDiscountCodeCategoryItem struct {
	CategoryID           uint `json:"categoryId"           binding:"required"`
	IncludeSubcategories bool `json:"includeSubcategories"`
}

// AddDiscountCodeCategoryRequest is the request to add categories to a discount code
type AddDiscountCodeCategoryRequest struct {
	BaseDiscountCodeScopeRequest
	Categories []AddDiscountCodeCategoryItem `json:"categories" binding:"required,min=1,dive"`
}

// RemoveDiscountCodeCategoryRequest is the request to remove categories from a discount code
type RemoveDiscountCodeCategoryRequest struct {
	BaseDiscountCodeScopeRequest
	CategoryIDs []uint `json:"categoryIds" binding:"required,min=1"`
}

// GetDiscountCodeCategoriesRequest is the request to get categories for a discount code
type GetDiscountCodeCategoriesRequest struct {
	GetDiscountCodeScopeRequest
	CategoryIDs []uint `json:"categoryIds" form:"categoryIds"`
}

// GetDiscountCodeCategoriesQueryParams is the query params for getting categories for a discount code
type GetDiscountCodeCategoriesQueryParams struct {
	GetDiscountCodeScopeRequest
	CategoryIDs *string `form:"categoryIds" binding:"omitempty"`
}

func (p *GetDiscountCodeCategoriesQueryParams) ToRequest() GetDiscountCodeCategoriesRequest {
	req := GetDiscountCodeCategoriesRequest{
		GetDiscountCodeScopeRequest: p.GetDiscountCodeScopeRequest,
	}

	if p.CategoryIDs != nil {
		req.CategoryIDs = helper.ParseCommaSeparatedPtr[uint](p.CategoryIDs)
	}

	return req
}

// GetDiscountCodeCategoriesResponse is the response for listing categories on a discount code
type GetDiscountCodeCategoriesResponse struct {
	BaseDiscountCodeScopeResponse
	Categories []DiscountCodeCategoryResponse `json:"categories"`
	Pagination PaginationResponse             `json:"pagination"`
}
