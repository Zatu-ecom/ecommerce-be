package service

import (
	"context"

	"ecommerce-be/common/config"
	"ecommerce-be/common/log"
	commonModel "ecommerce-be/common/model"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repository"
)

// RecentlyViewedService defines the contract for recently viewed operations.
type RecentlyViewedService interface {
	// RecordRecentlyViewed records a product view for a user.
	// If the user has > 10 recently viewed entries, the oldest are trimmed.
	// Errors are logged but not returned (fire-and-forget pattern).
	RecordRecentlyViewed(ctx context.Context, userID, sellerID, productID uint)

	// GetRecentlyViewedProducts returns full product details for the user's
	// recently viewed products, ordered newest first (by view history).
	// Internally calls ProductQueryService.GetAllProducts with the IDs filter
	// and re-orders results to match the view history order.
	GetRecentlyViewedProducts(
		ctx context.Context,
		userID uint,
		limit int,
	) (*model.ProductsResponse, error)
}

// RecentlyViewedServiceImpl implements RecentlyViewedService.
type RecentlyViewedServiceImpl struct {
	repo                repository.RecentlyViewedRepository
	productQueryService ProductQueryService
}

// NewRecentlyViewedService creates a new RecentlyViewedServiceImpl.
func NewRecentlyViewedService(
	repo repository.RecentlyViewedRepository,
	productQueryService ProductQueryService,
) *RecentlyViewedServiceImpl {
	return &RecentlyViewedServiceImpl{
		repo:                repo,
		productQueryService: productQueryService,
	}
}

// RecordRecentlyViewed upserts the view entry and trims to maxRecentlyViewed.
// Errors are logged but not propagated — the product query must not fail
// because of a side-effect recording failure.
func (s *RecentlyViewedServiceImpl) RecordRecentlyViewed(
	ctx context.Context, userID, sellerID, productID uint,
) {
	// 1. Upsert (insert or update viewed_at and seller_id)
	if err := s.repo.UpsertByUserAndProduct(ctx, userID, sellerID, productID); err != nil {
		log.ErrorWithContext(ctx, "Failed to record recently viewed product", err)
		return
	}

	// 2. Check if we need to trim
	count, err := s.repo.CountByUserID(ctx, userID)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to count recently viewed products", err)
		return
	}

	// 3. Trim oldest entries if over the limit
	maxSize := config.Get().App.RecentlyViewedQueueSize
	if count > int64(maxSize) {
		if err := s.repo.DeleteOldestByUserID(ctx, userID, maxSize); err != nil {
			log.ErrorWithContext(ctx, "Failed to trim recently viewed products", err)
		}
	}
}

// GetRecentlyViewedProducts returns full product details for the user's
// recently viewed products, ordered newest first by view history.
//
// Business logic flow:
//  1. Query recently viewed product IDs from repository (newest first)
//  2. If empty, return empty ProductsResponse
//  3. Fetch full product details via ProductQueryService.GetAllProducts with IDs filter
//  4. Re-order products to match view history order (GetAllProducts sorts by created_at desc)
func (s *RecentlyViewedServiceImpl) GetRecentlyViewedProducts(
	ctx context.Context,
	userID uint,
	limit int,
) (*model.ProductsResponse, error) {
	// Step 1: Get product IDs from recently viewed history
	entries, err := s.repo.FindByUserID(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return &model.ProductsResponse{
			Products:   []model.ProductResponse{},
			Pagination: commonModel.NewPaginationResponse(1, limit, 0),
		}, nil
	}

	productIDs := make([]uint, len(entries))
	for i, entry := range entries {
		productIDs[i] = entry.ProductID
	}

	// Step 2: Fetch full product details via ProductQueryService.
	// This reuses the same optimized batch processing (variants, media, options, etc.)
	// as the main product listing API, preventing N+1 queries.
	filter := model.GetProductsFilter{
		IDs: productIDs,
	}
	productsResponse, err := s.productQueryService.GetAllProducts(
		ctx,
		1,     // page 1 — no pagination needed (filtered by IDs)
		limit, // same limit as the view history
		filter,
		&userID, // pass user ID for wishlist status
	)
	if err != nil {
		return nil, err
	}

	// Step 3: Re-order products to match view history order (newest first).
	// GetAllProducts sorts by DB column (created_at desc), but we need
	// view history order which is returned from FindByUserID.
	productsByID := make(map[uint]model.ProductResponse, len(productsResponse.Products))
	for _, p := range productsResponse.Products {
		productsByID[p.ID] = p
	}
	orderedProducts := make([]model.ProductResponse, 0, len(productIDs))
	for _, pid := range productIDs {
		if p, ok := productsByID[pid]; ok {
			orderedProducts = append(orderedProducts, p)
		}
	}
	productsResponse.Products = orderedProducts

	return productsResponse, nil
}
