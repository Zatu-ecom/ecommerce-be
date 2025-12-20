package repositories

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"

	"gorm.io/gorm"
)

// ProductAttributeRepository defines the interface for product attribute data operations
type ProductAttributeRepository interface {
	Create(ctx context.Context, productAttribute *entity.ProductAttribute) error
	BulkCreate(ctx context.Context, productAttributes []*entity.ProductAttribute) error
	Update(ctx context.Context, productAttribute *entity.ProductAttribute) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*entity.ProductAttribute, error)
	FindByProductIDAndAttributeID(
		ctx context.Context,
		productID, attributeDefID uint,
	) (*entity.ProductAttribute, error)
	FindAllByProductID(ctx context.Context, productID uint) ([]entity.ProductAttribute, error)
	ExistsByProductIDAndAttributeID(
		ctx context.Context,
		productID, attributeDefID uint,
	) (bool, error)
}

// ProductAttributeRepositoryImpl implements the ProductAttributeRepository interface
type ProductAttributeRepositoryImpl struct {
	db *gorm.DB
}

// NewProductAttributeRepository creates a new instance of ProductAttributeRepository
func NewProductAttributeRepository(db *gorm.DB) ProductAttributeRepository {
	return &ProductAttributeRepositoryImpl{db: db}
}

// Create creates a new product attribute
func (r *ProductAttributeRepositoryImpl) Create(
	ctx context.Context,
	productAttribute *entity.ProductAttribute,
) error {
	return db.DB(ctx).Create(productAttribute).Error
}

// BulkCreate creates multiple product attributes in a single INSERT query
// Uses RETURNING clause to get generated IDs efficiently
func (r *ProductAttributeRepositoryImpl) BulkCreate(
	ctx context.Context,
	productAttributes []*entity.ProductAttribute,
) error {
	if len(productAttributes) == 0 {
		return nil
	}
	// GORM's Create with slice automatically uses bulk insert and populates IDs
	return db.DB(ctx).Create(&productAttributes).Error
}

// Update updates an existing product attribute
func (r *ProductAttributeRepositoryImpl) Update(
	ctx context.Context,
	productAttribute *entity.ProductAttribute,
) error {
	return db.DB(ctx).Save(productAttribute).Error
}

// Delete deletes a product attribute by ID
func (r *ProductAttributeRepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := db.DB(ctx).Delete(&entity.ProductAttribute{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return prodErrors.ErrProductAttributeNotFound
	}
	return nil
}

// FindByID finds a product attribute by ID
func (r *ProductAttributeRepositoryImpl) FindByID(
	ctx context.Context,
	id uint,
) (*entity.ProductAttribute, error) {
	var productAttribute entity.ProductAttribute
	err := db.DB(ctx).Preload("AttributeDefinition").First(&productAttribute, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, prodErrors.ErrProductAttributeNotFound
		}
		return nil, err
	}
	return &productAttribute, nil
}

// FindByProductIDAndAttributeID finds a product attribute by product ID and attribute definition ID
func (r *ProductAttributeRepositoryImpl) FindByProductIDAndAttributeID(
	ctx context.Context,
	productID, attributeDefID uint,
) (*entity.ProductAttribute, error) {
	var productAttribute entity.ProductAttribute
	err := db.DB(ctx).Preload("AttributeDefinition").
		Where("product_id = ? AND attribute_definition_id = ?", productID, attributeDefID).
		First(&productAttribute).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, prodErrors.ErrProductAttributeNotFound
		}
		return nil, err
	}
	return &productAttribute, nil
}

// FindAllByProductID finds all product attributes for a given product
func (r *ProductAttributeRepositoryImpl) FindAllByProductID(
	ctx context.Context,
	productID uint,
) ([]entity.ProductAttribute, error) {
	var productAttributes []entity.ProductAttribute
	err := db.DB(ctx).Preload("AttributeDefinition").
		Where("product_id = ?", productID).
		Order("sort_order ASC, id ASC").
		Find(&productAttributes).Error
	if err != nil {
		return nil, err
	}
	return productAttributes, nil
}

// ExistsByProductIDAndAttributeID checks if a product attribute exists
func (r *ProductAttributeRepositoryImpl) ExistsByProductIDAndAttributeID(
	ctx context.Context,
	productID, attributeDefID uint,
) (bool, error) {
	var count int64
	err := db.DB(ctx).Model(&entity.ProductAttribute{}).
		Where("product_id = ? AND attribute_definition_id = ?", productID, attributeDefID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
