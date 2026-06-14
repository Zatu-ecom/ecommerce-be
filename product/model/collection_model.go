package model

import (
	"ecommerce-be/common"
	"ecommerce-be/common/helper"
)

// CollectionCreateRequest represents the request body for creating a collection
type CollectionCreateRequest struct {
	Name        string  `json:"name"        binding:"required,min=3,max=255"`
	Description *string `json:"description" binding:"omitempty,max=2000"`
	Image       *string `json:"image"`
}

// CollectionUpdateRequest represents the request body for updating a collection
type CollectionUpdateRequest struct {
	Name        string  `json:"name"        binding:"required,min=3,max=255"`
	Description *string `json:"description" binding:"omitempty,max=2000"`
	Image       *string `json:"image"`
	IsActive    *bool   `json:"isActive"`
}

// CollectionResponse represents collection data returned in API responses
type CollectionResponse struct {
	ID           uint   `json:"id"`
	SellerID     uint   `json:"sellerId"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Description  *string `json:"description,omitempty"`
	Image        *string `json:"image,omitempty"`
	IsActive     bool   `json:"isActive"`
	ProductCount int64  `json:"productCount,omitempty"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// CollectionsResponse represents the response for listing collections
type CollectionsResponse struct {
	Collections []CollectionResponse `json:"collections"`
}

// AddCollectionProductsRequest is the request to add products to a collection
type AddCollectionProductsRequest struct {
	ProductIDs []uint `json:"productIds" binding:"required,min=1"`
}

// RemoveCollectionProductsRequest is the request to remove products from a collection
type RemoveCollectionProductsRequest struct {
	ProductIDs []uint `json:"productIds" binding:"required,min=1"`
}

// CollectionProductPositionItem represents a product position in a collection
type CollectionProductPositionItem struct {
	ProductID uint `json:"productId" binding:"required"`
	Position  int  `json:"position"   binding:"min=0"`
}

// ReorderCollectionProductsRequest is the request to reorder products in a collection
type ReorderCollectionProductsRequest struct {
	Items []CollectionProductPositionItem `json:"items" binding:"required,min=1,dive"`
}

// CollectionProductResponse represents a product membership in a collection
type CollectionProductResponse struct {
	ID          uint   `json:"id"`
	ProductID   uint   `json:"productId"`
	ProductName string `json:"productName,omitempty"`
	Position    int    `json:"position"`
	CreatedAt   string `json:"createdAt"`
}

// GetCollectionProductsRequest is the request to list products in a collection
type GetCollectionProductsRequest struct {
	common.BaseListParams
	ProductIDs []uint `json:"productIds" form:"productIds"`
}

// GetCollectionProductsQueryParams is the query params for listing collection products
type GetCollectionProductsQueryParams struct {
	common.BaseListParams
	ProductIDs *string `form:"productIds" binding:"omitempty"`
}

func (p *GetCollectionProductsQueryParams) ToRequest() GetCollectionProductsRequest {
	req := GetCollectionProductsRequest{
		BaseListParams: p.BaseListParams,
	}
	if p.ProductIDs != nil {
		req.ProductIDs = helper.ParseCommaSeparatedPtr[uint](p.ProductIDs)
	}
	return req
}

// GetCollectionProductsResponse is the response for listing products in a collection
type GetCollectionProductsResponse struct {
	Products   []CollectionProductResponse `json:"products"`
	Pagination PaginationResponse          `json:"pagination"`
}
