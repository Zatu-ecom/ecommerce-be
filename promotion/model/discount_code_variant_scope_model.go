package model

import (
	"ecommerce-be/common/helper"
)

// DiscountCodeVariantResponse is the base response model for a discount-code variant
type DiscountCodeVariantResponse struct {
	BaseDiscountCodeScopeResponse
	VariantID uint   `json:"variantId"`
	ProductID uint   `json:"productId"`
	SKU       string `json:"sku,omitempty"`
	Price     string `json:"price,omitempty"`
}

// AddDiscountCodeVariantRequest is the request to add variants to a discount code
type AddDiscountCodeVariantRequest struct {
	BaseDiscountCodeScopeRequest
	VariantIDs []uint `json:"variantIds" binding:"required,min=1"`
}

// RemoveDiscountCodeVariantRequest is the request to remove variants from a discount code
type RemoveDiscountCodeVariantRequest struct {
	BaseDiscountCodeScopeRequest
	VariantIDs []uint `json:"variantIds" binding:"required,min=1"`
}

// GetDiscountCodeVariantsRequest is the request to get variants for a discount code
type GetDiscountCodeVariantsRequest struct {
	GetDiscountCodeScopeRequest
	VariantIDs []uint `json:"variantIds" form:"variantIds"`
}

// GetDiscountCodeVariantsQueryParams is the query params for getting variants for a discount code
type GetDiscountCodeVariantsQueryParams struct {
	GetDiscountCodeScopeRequest
	VariantIDs *string `form:"variantIds" binding:"omitempty"`
}

func (p *GetDiscountCodeVariantsQueryParams) ToRequest() GetDiscountCodeVariantsRequest {
	req := GetDiscountCodeVariantsRequest{
		GetDiscountCodeScopeRequest: p.GetDiscountCodeScopeRequest,
	}

	if p.VariantIDs != nil {
		req.VariantIDs = helper.ParseCommaSeparatedPtr[uint](p.VariantIDs)
	}

	return req
}

// GetDiscountCodeVariantsResponse is the response for listing variants on a discount code
type GetDiscountCodeVariantsResponse struct {
	BaseDiscountCodeScopeResponse
	Variants   []DiscountCodeVariantResponse `json:"variants"`
	Pagination PaginationResponse            `json:"pagination"`
}
