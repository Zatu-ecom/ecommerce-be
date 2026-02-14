package model

import (
	"ecommerce-be/common/helper"
)

// PromotionCollectionResponse is the base response model for a promotion collection
type PromotionCollectionResponse struct {
	BasePromotionScopeResponse
	CollectionID   uint   `json:"collectionId"`
	CollectionName string `json:"collectionName,omitempty"`
	CollectionSlug string `json:"collectionSlug,omitempty"`
}

// AddPromotionCollectionRequest is the request to add collections to a promotion
type AddPromotionCollectionRequest struct {
	BasePromotionScopeRequest
	CollectionIDs []uint `json:"collectionIds" binding:"required,min=1"`
}

// RemovePromotionCollectionRequest is the request to remove collections from a promotion
type RemovePromotionCollectionRequest struct {
	BasePromotionScopeRequest
	CollectionIDs []uint `json:"collectionIds" binding:"required,min=1"`
}

// GetPromotionCollectionsRequest is the request to get collections for a promotion
type GetPromotionCollectionsRequest struct {
	GetPromotionScopeRequest
	CollectionIDs []uint `json:"collectionIds" form:"collectionIds"`
}

// GetPromotionCollectionsQueryParams is the query params for getting collections for a promotion
type GetPromotionCollectionsQueryParams struct {
	GetPromotionScopeRequest
	CollectionIDs *string `form:"collectionIds" binding:"omitempty"`
}

func (p *GetPromotionCollectionsQueryParams) ToRequest() GetPromotionCollectionsRequest {
	req := GetPromotionCollectionsRequest{
		GetPromotionScopeRequest: p.GetPromotionScopeRequest,
	}
	
	if p.CollectionIDs != nil {
		req.CollectionIDs = helper.ParseCommaSeparatedPtr[uint](p.CollectionIDs)
	}

	return req
}

// GetPromotionCollectionsResponse is the response for listing collections in a promotion
type GetPromotionCollectionsResponse struct {
	BasePromotionScopeResponse
	Collections []PromotionCollectionResponse `json:"collections"`
	Pagination  PaginationResponse            `json:"pagination"`
}
