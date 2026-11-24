package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"
)

// PhysicalSpecRepository reads the shippable spec catalog. Rows are
// seed-managed (core seed 005); no write methods by design.
type PhysicalSpecRepository interface {
	// FindAllActive returns every active unit row ordered for display.
	FindAllActive(ctx context.Context) ([]entity.PhysicalSpecUnit, error)
	// FindByDefinitionKey returns the catalog row for an attribute key,
	// or nil when the key is not a shippable spec.
	FindByDefinitionKey(ctx context.Context, key string) (*entity.PhysicalSpecUnit, error)
}

type PhysicalSpecRepositoryImpl struct{}

func NewPhysicalSpecRepository() PhysicalSpecRepository {
	return &PhysicalSpecRepositoryImpl{}
}

func (r *PhysicalSpecRepositoryImpl) FindAllActive(ctx context.Context) ([]entity.PhysicalSpecUnit, error) {
	var units []entity.PhysicalSpecUnit
	err := db.DB(ctx).
		Where("is_active = ?", true).
		Order("parameter ASC, display_order ASC, key ASC").
		Find(&units).Error
	if err != nil {
		return nil, err
	}
	return units, nil
}

func (r *PhysicalSpecRepositoryImpl) FindByDefinitionKey(
	ctx context.Context,
	key string,
) (*entity.PhysicalSpecUnit, error) {
	var unit entity.PhysicalSpecUnit
	err := db.DB(ctx).
		Where("key = ? AND is_active = ?", key, true).
		First(&unit).Error
	if err != nil {
		return nil, err
	}
	return &unit, nil
}
