package model

import (
	"ecommerce-be/common/helper"
)

// PromotionProductResponse is the base response model for a promotion product
type PromotionProductResponse struct {
	BasePromotionScopeResponse
	ProductID   uint   `json:"productId"`
	ProductName string `json:"productName,omitempty"`
	ProductSlug string `json:"productSlug,omitempty"`
	ImageURL    string `json:"imageUrl,omitempty"`
}

// AddPromotionProductRequest is the request to add products to a promotion
type AddPromotionProductRequest struct {
	BasePromotionScopeRequest
	ProductIDs []uint `json:"productIds" binding:"required,min=1"`
}

// RemovePromotionProductRequest is the request to remove products from a promotion
type RemovePromotionProductRequest struct {
	BasePromotionScopeRequest
	ProductIDs []uint `json:"productIds" binding:"required,min=1"`
}

// GetPromotionProductsRequest is the request to get products for a promotion
type GetPromotionProductsRequest struct {
	GetPromotionScopeRequest
	ProductIDs []uint `json:"productIds" form:"productIds"`
}

// GetPromotionProductsQueryParams is the query params for getting products for a promotion
type GetPromotionProductsQueryParams struct {
	GetPromotionScopeRequest
	ProductIDs *string `form:"productIds" binding:"omitempty"`
}

func (p *GetPromotionProductsQueryParams) ToRequest() GetPromotionProductsRequest {
	req := GetPromotionProductsRequest{
		GetPromotionScopeRequest: p.GetPromotionScopeRequest,
	}
	
	if p.ProductIDs != nil {
		req.ProductIDs = helper.ParseCommaSeparatedPtr[uint](p.ProductIDs)
	}

	return req
}

// GetPromotionProductsResponse is the response for listing products in a promotion
type GetPromotionProductsResponse struct {
	BasePromotionScopeResponse
	Products   []PromotionProductResponse `json:"products"`
	Pagination PaginationResponse         `json:"pagination"`
}
