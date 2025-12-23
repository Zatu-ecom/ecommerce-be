package service

import (
	"context"

	"ecommerce-be/inventory/model"
)

// InventoryManageService defines the interface for inventory-related business logic
type InventoryManageService interface {
	// ManageInventory creates or updates inventory for a variant at a location
	ManageInventory(
		ctx context.Context,
		req model.ManageInventoryRequest,
		sellerID uint,
		userID uint,
	) (*model.ManageInventoryResponse, error)

	// BulkManageInventory manages multiple inventory records in a single transaction
	BulkManageInventory(
		ctx context.Context,
		req model.BulkManageInventoryRequest,
		sellerID uint,
		userID uint,
	) (*model.BulkManageInventoryResponse, error)
}

type InventoryQueryService interface {
	GetInventoryByVariant(
		ctx context.Context,
		variantID uint,
		sellerID uint,
	) ([]model.InventoryDetailResponse, error)

	GetInventoryByLocation(
		ctx context.Context,
		locationID uint,
		sellerID uint,
	) ([]model.InventoryResponse, error)

	GetInventories(
		ctx context.Context,
		sellerID *uint,
		filter model.GetInventoriesFilter,
	) (*model.InventoryResponseWithPagination, error)

	GetInventoryByVariantAndLocationPriority(
		ctx context.Context,
		items []model.ReservationItem,
		sellerID uint,
	) ([]model.InventoryResponse, error)
}
