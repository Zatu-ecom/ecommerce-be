package repositories

import (
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"

	"gorm.io/gorm"
)

// ProductAttributeRepository defines the interface for product attribute data operations
type ProductAttributeRepository interface {
	Create(productAttribute *entity.ProductAttribute) error
	Update(productAttribute *entity.ProductAttribute) error
	Delete(id uint) error
	FindByID(id uint) (*entity.ProductAttribute, error)
	FindByProductIDAndAttributeID(productID, attributeDefID uint) (*entity.ProductAttribute, error)
	FindAllByProductID(productID uint) ([]entity.ProductAttribute, error)
	ExistsByProductIDAndAttributeID(productID, attributeDefID uint) (bool, error)
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
func (r *ProductAttributeRepositoryImpl) Create(productAttribute *entity.ProductAttribute) error {
	return r.db.Create(productAttribute).Error
}

// Update updates an existing product attribute
func (r *ProductAttributeRepositoryImpl) Update(productAttribute *entity.ProductAttribute) error {
	return r.db.Save(productAttribute).Error
}

// Delete deletes a product attribute by ID
func (r *ProductAttributeRepositoryImpl) Delete(id uint) error {
	result := r.db.Delete(&entity.ProductAttribute{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return prodErrors.ErrProductAttributeNotFound
	}
	return nil
}

// FindByID finds a product attribute by ID
func (r *ProductAttributeRepositoryImpl) FindByID(id uint) (*entity.ProductAttribute, error) {
	var productAttribute entity.ProductAttribute
	err := r.db.Preload("AttributeDefinition").First(&productAttribute, id).Error
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
	productID, attributeDefID uint,
) (*entity.ProductAttribute, error) {
	var productAttribute entity.ProductAttribute
	err := r.db.Preload("AttributeDefinition").
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
func (r *ProductAttributeRepositoryImpl) FindAllByProductID(productID uint) ([]entity.ProductAttribute, error) {
	var productAttributes []entity.ProductAttribute
	err := r.db.Preload("AttributeDefinition").
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
	productID, attributeDefID uint,
) (bool, error) {
	var count int64
	err := r.db.Model(&entity.ProductAttribute{}).
		Where("product_id = ? AND attribute_definition_id = ?", productID, attributeDefID).
		Count(&count).Error
	
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
