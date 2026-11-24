package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/promotion/entity"
)

type DiscountCodeCollectionScopeRepository interface {
	AddCollections(ctx context.Context, collections []entity.DiscountCodeCollection) error
	DeleteCollections(ctx context.Context, discountCodeID uint, collectionIDs []uint) error
	DeleteAllCollections(ctx context.Context, discountCodeID uint) error
	GetCollections(
		ctx context.Context,
		discountCodeID uint,
		collectionIDs []uint,
		offset, limit int,
	) ([]entity.DiscountCodeCollection, int64, error)
}

type DiscountCodeCollectionScopeRepositoryImpl struct{}

func NewDiscountCodeCollectionScopeRepository() DiscountCodeCollectionScopeRepository {
	return &DiscountCodeCollectionScopeRepositoryImpl{}
}

func (r *DiscountCodeCollectionScopeRepositoryImpl) AddCollections(
	ctx context.Context,
	collections []entity.DiscountCodeCollection,
) error {
	if len(collections) == 0 {
		return nil
	}
	return db.DB(ctx).Create(&collections).Error
}

func (r *DiscountCodeCollectionScopeRepositoryImpl) DeleteCollections(
	ctx context.Context,
	discountCodeID uint,
	collectionIDs []uint,
) error {
	if len(collectionIDs) == 0 {
		return nil
	}
	return db.DB(ctx).
		Where("discount_code_id = ? AND collection_id IN ?", discountCodeID, collectionIDs).
		Delete(&entity.DiscountCodeCollection{}).Error
}

func (r *DiscountCodeCollectionScopeRepositoryImpl) DeleteAllCollections(
	ctx context.Context,
	discountCodeID uint,
) error {
	return db.DB(ctx).
		Where("discount_code_id = ?", discountCodeID).
		Delete(&entity.DiscountCodeCollection{}).Error
}

func (r *DiscountCodeCollectionScopeRepositoryImpl) GetCollections(
	ctx context.Context,
	discountCodeID uint,
	collectionIDs []uint,
	offset, limit int,
) ([]entity.DiscountCodeCollection, int64, error) {
	var collections []entity.DiscountCodeCollection
	var total int64

	query := db.DB(ctx).Model(&entity.DiscountCodeCollection{}).
		Where("discount_code_id = ?", discountCodeID)

	if len(collectionIDs) > 0 {
		query = query.Where("collection_id IN ?", collectionIDs)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Offset(offset).Limit(limit).Find(&collections).Error; err != nil {
		return nil, 0, err
	}
	return collections, total, nil
}
