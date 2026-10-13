package model

import (
	"ecommerce-be/common/helper"
)

// DiscountCodeCollectionResponse is the base response model for a discount-code collection
type DiscountCodeCollectionResponse struct {
	BaseDiscountCodeScopeResponse
	CollectionID   uint   `json:"collectionId"`
	CollectionName string `json:"collectionName,omitempty"`
	CollectionSlug string `json:"collectionSlug,omitempty"`
}

// AddDiscountCodeCollectionRequest is the request to add collections to a discount code
type AddDiscountCodeCollectionRequest struct {
	BaseDiscountCodeScopeRequest
	CollectionIDs []uint `json:"collectionIds" binding:"required,min=1"`
}

// RemoveDiscountCodeCollectionRequest is the request to remove collections from a discount code
type RemoveDiscountCodeCollectionRequest struct {
	BaseDiscountCodeScopeRequest
	CollectionIDs []uint `json:"collectionIds" binding:"required,min=1"`
}

// GetDiscountCodeCollectionsRequest is the request to get collections for a discount code
type GetDiscountCodeCollectionsRequest struct {
	GetDiscountCodeScopeRequest
	CollectionIDs []uint `json:"collectionIds" form:"collectionIds"`
}

// GetDiscountCodeCollectionsQueryParams is the query params for getting collections for a discount code
type GetDiscountCodeCollectionsQueryParams struct {
	GetDiscountCodeScopeRequest
	CollectionIDs *string `form:"collectionIds" binding:"omitempty"`
}

func (p *GetDiscountCodeCollectionsQueryParams) ToRequest() GetDiscountCodeCollectionsRequest {
	req := GetDiscountCodeCollectionsRequest{
		GetDiscountCodeScopeRequest: p.GetDiscountCodeScopeRequest,
	}

	if p.CollectionIDs != nil {
		req.CollectionIDs = helper.ParseCommaSeparatedPtr[uint](p.CollectionIDs)
	}

	return req
}

// GetDiscountCodeCollectionsResponse is the response for listing collections on a discount code
type GetDiscountCodeCollectionsResponse struct {
	BaseDiscountCodeScopeResponse
	Collections []DiscountCodeCollectionResponse `json:"collections"`
	Pagination  PaginationResponse               `json:"pagination"`
}
