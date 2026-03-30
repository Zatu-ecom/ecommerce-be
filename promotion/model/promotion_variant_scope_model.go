package model

import (
	"ecommerce-be/common/helper"
)

// PromotionVariantResponse is the base response model for a promotion variant
type PromotionVariantResponse struct {
	BasePromotionScopeResponse
	VariantID uint   `json:"variantId"`
	ProductID uint   `json:"productId"`
	SKU       string `json:"sku,omitempty"`
	Price     string `json:"price,omitempty"` // String to support formatted price
}

// AddPromotionVariantRequest is the request to add variants to a promotion
type AddPromotionVariantRequest struct {
	BasePromotionScopeRequest
	VariantIDs []uint `json:"variantIds" binding:"required,min=1"`
}

// RemovePromotionVariantRequest is the request to remove variants from a promotion
type RemovePromotionVariantRequest struct {
	BasePromotionScopeRequest
	VariantIDs []uint `json:"variantIds" binding:"required,min=1"`
}

// GetPromotionVariantsRequest is the request to get variants for a promotion
type GetPromotionVariantsRequest struct {
	GetPromotionScopeRequest
	VariantIDs []uint `json:"variantIds" form:"variantIds"`
}

// GetPromotionVariantsQueryParams is the query params for getting variants for a promotion
type GetPromotionVariantsQueryParams struct {
	GetPromotionScopeRequest
	VariantIDs *string `form:"variantIds" binding:"omitempty"`
}

func (p *GetPromotionVariantsQueryParams) ToRequest() GetPromotionVariantsRequest {
	req := GetPromotionVariantsRequest{
		GetPromotionScopeRequest: p.GetPromotionScopeRequest,
	}

	if p.VariantIDs != nil {
		req.VariantIDs = helper.ParseCommaSeparatedPtr[uint](p.VariantIDs)
	}

	return req
}

// GetPromotionVariantsResponse is the response for listing variants in a promotion
type GetPromotionVariantsResponse struct {
	BasePromotionScopeResponse
	Variants   []PromotionVariantResponse `json:"variants"`
	Pagination PaginationResponse         `json:"pagination"`
}
