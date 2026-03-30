package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/promotion/entity"
)

// PromotionCollectionScopeRepository defines the interface for promotion-collection scope operations
type PromotionCollectionScopeRepository interface {
	AddPromotionCollections(ctx context.Context, collections []entity.PromotionCollection) error
	DeletePromotionCollections(ctx context.Context, promotionID uint, collectionIDs []uint) error
	DeletePromotionCollectionByPromotionID(ctx context.Context, promotionID uint) error
	GetPromotionCollections(
		ctx context.Context,
		promotionID uint,
		collectionIDs []uint,
		offset, limit int,
	) ([]entity.PromotionCollection, int64, error)
}

// PromotionCollectionScopeRepositoryImpl implements the PromotionCollectionRepository interface
type PromotionCollectionScopeRepositoryImpl struct{}

// NewPromotionCollectionScopeRepository creates a new instance of PromotionCollectionRepository
func NewPromotionCollectionScopeRepository() PromotionCollectionScopeRepository {
	return &PromotionCollectionScopeRepositoryImpl{}
}

// AddPromotionCollections bulk inserts promotion-collection mappings
func (r *PromotionCollectionScopeRepositoryImpl) AddPromotionCollections(
	ctx context.Context,
	collections []entity.PromotionCollection,
) error {
	if len(collections) == 0 {
		return nil
	}
	return db.DB(ctx).Create(&collections).Error
}

// DeletePromotionCollections deletes specific collection mappings from a promotion
func (r *PromotionCollectionScopeRepositoryImpl) DeletePromotionCollections(
	ctx context.Context,
	promotionID uint,
	collectionIDs []uint,
) error {
	if len(collectionIDs) == 0 {
		return nil
	}
	return db.DB(ctx).
		Where("promotion_id = ? AND collection_id IN ?", promotionID, collectionIDs).
		Delete(&entity.PromotionCollection{}).Error
}

// DeletePromotionCollectionByPromotionID removes all collection mappings for a promotion
func (r *PromotionCollectionScopeRepositoryImpl) DeletePromotionCollectionByPromotionID(
	ctx context.Context,
	promotionID uint,
) error {
	return db.DB(ctx).
		Where("promotion_id = ?", promotionID).
		Delete(&entity.PromotionCollection{}).Error
}

// GetPromotionCollections retrieves all collection mappings for a promotion with pagination
func (r *PromotionCollectionScopeRepositoryImpl) GetPromotionCollections(
	ctx context.Context,
	promotionID uint,
	collectionIDs []uint,
	offset, limit int,
) ([]entity.PromotionCollection, int64, error) {
	var collections []entity.PromotionCollection
	var total int64

	query := db.DB(ctx).Model(&entity.PromotionCollection{}).Where("promotion_id = ?", promotionID)

	if len(collectionIDs) > 0 {
		query = query.Where("collection_id IN ?", collectionIDs)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Offset(offset).Limit(limit).Find(&collections).Error
	if err != nil {
		return nil, 0, err
	}

	return collections, total, nil
}
