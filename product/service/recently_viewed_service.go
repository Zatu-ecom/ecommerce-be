package service

import (
	"context"

	"ecommerce-be/common/config"
	"ecommerce-be/common/log"
	"ecommerce-be/product/repository"
)

// RecentlyViewedService defines the contract for recently viewed operations.
type RecentlyViewedService interface {
	// RecordRecentlyViewed records a product view for a user.
	// If the user has > 10 recently viewed entries, the oldest are trimmed.
	// Errors are logged but not returned (fire-and-forget pattern).
	RecordRecentlyViewed(ctx context.Context, userID, sellerID, productID uint)

	// GetRecentlyViewed returns the most recent N product IDs viewed by the user.
	GetRecentlyViewed(ctx context.Context, userID uint, limit int) ([]uint, error)
}

// RecentlyViewedServiceImpl implements RecentlyViewedService using a repository.
type RecentlyViewedServiceImpl struct {
	repo repository.RecentlyViewedRepository
}

// NewRecentlyViewedService creates a new RecentlyViewedServiceImpl.
func NewRecentlyViewedService(repo repository.RecentlyViewedRepository) *RecentlyViewedServiceImpl {
	return &RecentlyViewedServiceImpl{repo: repo}
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

// GetRecentlyViewed returns product IDs of recently viewed products, newest first.
func (s *RecentlyViewedServiceImpl) GetRecentlyViewed(
	ctx context.Context, userID uint, limit int,
) ([]uint, error) {
	entries, err := s.repo.FindByUserID(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	productIDs := make([]uint, len(entries))
	for i, entry := range entries {
		productIDs[i] = entry.ProductID
	}
	return productIDs, nil
}
