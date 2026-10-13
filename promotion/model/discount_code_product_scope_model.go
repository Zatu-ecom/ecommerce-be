package model

import (
	"ecommerce-be/common/helper"
)

// DiscountCodeProductResponse is the base response model for a discount-code product
type DiscountCodeProductResponse struct {
	BaseDiscountCodeScopeResponse
	ProductID   uint   `json:"productId"`
	ProductName string `json:"productName,omitempty"`
	ProductSlug string `json:"productSlug,omitempty"`
	ImageURL    string `json:"imageUrl,omitempty"`
}

// AddDiscountCodeProductRequest is the request to add products to a discount code
type AddDiscountCodeProductRequest struct {
	BaseDiscountCodeScopeRequest
	ProductIDs []uint `json:"productIds" binding:"required,min=1"`
}

// RemoveDiscountCodeProductRequest is the request to remove products from a discount code
type RemoveDiscountCodeProductRequest struct {
	BaseDiscountCodeScopeRequest
	ProductIDs []uint `json:"productIds" binding:"required,min=1"`
}

// GetDiscountCodeProductsRequest is the request to get products for a discount code
type GetDiscountCodeProductsRequest struct {
	GetDiscountCodeScopeRequest
	ProductIDs []uint `json:"productIds" form:"productIds"`
}

// GetDiscountCodeProductsQueryParams is the query params for getting products for a discount code
type GetDiscountCodeProductsQueryParams struct {
	GetDiscountCodeScopeRequest
	ProductIDs *string `form:"productIds" binding:"omitempty"`
}

func (p *GetDiscountCodeProductsQueryParams) ToRequest() GetDiscountCodeProductsRequest {
	req := GetDiscountCodeProductsRequest{
		GetDiscountCodeScopeRequest: p.GetDiscountCodeScopeRequest,
	}

	if p.ProductIDs != nil {
		req.ProductIDs = helper.ParseCommaSeparatedPtr[uint](p.ProductIDs)
	}

	return req
}

// GetDiscountCodeProductsResponse is the response for listing products on a discount code
type GetDiscountCodeProductsResponse struct {
	BaseDiscountCodeScopeResponse
	Products   []DiscountCodeProductResponse `json:"products"`
	Pagination PaginationResponse            `json:"pagination"`
}
