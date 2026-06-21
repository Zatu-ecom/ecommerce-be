package factory

import (
	"time"

	"ecommerce-be/common/filegateway"
	"ecommerce-be/product/entity"
	"ecommerce-be/product/model"
	"ecommerce-be/product/utils/helper"
)

func defaultIsActive(isActive *bool) bool {
	if isActive == nil {
		return true
	}
	return *isActive
}

// BuildCollectionEntityFromCreateRequest creates a Collection entity from a create request
func BuildCollectionEntityFromCreateRequest(
	req model.CollectionCreateRequest,
	sellerID uint,
) *entity.Collection {
	isActive := true
	return &entity.Collection{
		SellerID:    sellerID,
		Name:        req.Name,
		Description: req.Description,
		ImageFileID: req.ImageFileID,
		IsActive:    &isActive,
		BaseEntity:  helper.NewBaseEntity(),
	}
}

// ApplyCollectionUpdateRequest updates an existing Collection entity from an update request
func ApplyCollectionUpdateRequest(
	collection *entity.Collection,
	req model.CollectionUpdateRequest,
) *entity.Collection {
	collection.Name = req.Name
	collection.Description = req.Description
	if req.ImageFileID != nil {
		collection.ImageFileID = req.ImageFileID
	}
	if req.IsActive != nil {
		collection.IsActive = req.IsActive
	}
	collection.UpdatedAt = time.Now().UTC()
	return collection
}

// BuildCollectionResponse builds CollectionResponse from entity
func BuildCollectionResponse(
	collection *entity.Collection,
	productCount int64,
	image *filegateway.FileAssetResponse,
) *model.CollectionResponse {
	return &model.CollectionResponse{
		ID:           collection.ID,
		SellerID:     collection.SellerID,
		Name:         collection.Name,
		Slug:         collection.Slug,
		Description:  collection.Description,
		Image:        image,
		IsActive:     defaultIsActive(collection.IsActive),
		ProductCount: productCount,
		CreatedAt:    helper.FormatTimestamp(collection.CreatedAt.UTC()),
		UpdatedAt:    helper.FormatTimestamp(collection.UpdatedAt.UTC()),
	}
}

// BuildCollectionProductResponse builds CollectionProductResponse from entity
func BuildCollectionProductResponse(cp *entity.CollectionProduct) model.CollectionProductResponse {
	name := ""
	if cp.Product != nil {
		name = cp.Product.Name
	}
	return model.CollectionProductResponse{
		ID:          cp.ID,
		ProductID:   cp.ProductID,
		ProductName: name,
		Position:    cp.Position,
		CreatedAt:   helper.FormatTimestamp(cp.CreatedAt.UTC()),
	}
}
