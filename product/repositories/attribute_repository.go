package repositories

import (
	"errors"

	"ecommerce-be/product/entity"
	"ecommerce-be/product/utils"

	"gorm.io/gorm"
)

// AttributeDefinitionRepository defines the interface for attribute definition data operations
type AttributeDefinitionRepository interface {
	Create(attribute *entity.AttributeDefinition) error
	CreateBulk(attributes []*entity.AttributeDefinition) error
	FindByID(id uint) (*entity.AttributeDefinition, error)
	FindByKey(key string) (*entity.AttributeDefinition, error)
	FindByKeys(keys []string) (map[string]*entity.AttributeDefinition, error)
	FindAll() ([]entity.AttributeDefinition, error)
	Update(attribute *entity.AttributeDefinition) error
	UpdateBulk(attributes []*entity.AttributeDefinition) error
	Delete(id uint) error
	CreateCategoryAttributeDefinition(attribute *entity.AttributeDefinition, categoryID uint) error
	CreateProductAttribute(attribute *entity.ProductAttribute) error
	CreateProductAttributesBulk(attributes []*entity.ProductAttribute) error
	FindProductAttributeByProductID(productID uint) ([]entity.ProductAttribute, error)
	
	// Bulk deletion methods for product cleanup
	DeleteProductAttributesByProductID(productID uint) error
}

// AttributeDefinitionRepositoryImpl implements the AttributeDefinitionRepository interface
type AttributeDefinitionRepositoryImpl struct {
	db *gorm.DB
}

// NewAttributeDefinitionRepository creates a new instance of AttributeDefinitionRepository
func NewAttributeDefinitionRepository(db *gorm.DB) AttributeDefinitionRepository {
	return &AttributeDefinitionRepositoryImpl{
		db: db,
	}
}

// Create creates a new attribute definition in the database
func (r *AttributeDefinitionRepositoryImpl) Create(attribute *entity.AttributeDefinition) error {
	return r.db.Create(attribute).Error
}

// CreateBulk creates multiple attribute definitions in a single transaction
func (r *AttributeDefinitionRepositoryImpl) CreateBulk(
	attributes []*entity.AttributeDefinition,
) error {
	if len(attributes) == 0 {
		return nil
	}
	return r.db.CreateInBatches(attributes, 100).Error
}

// FindByID finds an attribute definition by ID
func (r *AttributeDefinitionRepositoryImpl) FindByID(id uint) (*entity.AttributeDefinition, error) {
	var attribute entity.AttributeDefinition
	result := r.db.First(&attribute, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New(utils.ATTRIBUTE_DEFINITION_NOT_FOUND_MSG)
		}
		return nil, result.Error
	}
	return &attribute, nil
}

// FindByKey finds an attribute definition by key
func (r *AttributeDefinitionRepositoryImpl) FindByKey(
	key string,
) (*entity.AttributeDefinition, error) {
	var attribute entity.AttributeDefinition
	result := r.db.Where("key = ?", key).First(&attribute)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &attribute, nil
}

// FindByKeys finds multiple attribute definitions by keys in a single query
func (r *AttributeDefinitionRepositoryImpl) FindByKeys(
	keys []string,
) (map[string]*entity.AttributeDefinition, error) {
	if len(keys) == 0 {
		return make(map[string]*entity.AttributeDefinition), nil
	}

	var attributes []entity.AttributeDefinition
	result := r.db.Where("key IN ?", keys).Find(&attributes)
	if result.Error != nil {
		return nil, result.Error
	}

	// Convert slice to map for O(1) lookup
	attributeMap := make(map[string]*entity.AttributeDefinition)
	for i := range attributes {
		attributeMap[attributes[i].Key] = &attributes[i]
	}

	return attributeMap, nil
}

// FindAll finds all active attribute definitions
func (r *AttributeDefinitionRepositoryImpl) FindAll() ([]entity.AttributeDefinition, error) {
	var attributes []entity.AttributeDefinition
	result := r.db.Order("name ASC").Find(&attributes)
	if result.Error != nil {
		return nil, result.Error
	}
	return attributes, nil
}

// Update updates an existing attribute definition
func (r *AttributeDefinitionRepositoryImpl) Update(attribute *entity.AttributeDefinition) error {
	return r.db.Save(attribute).Error
}

// UpdateBulk updates multiple attribute definitions in a single transaction
func (r *AttributeDefinitionRepositoryImpl) UpdateBulk(
	attributes []*entity.AttributeDefinition,
) error {
	if len(attributes) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, attribute := range attributes {
			if err := tx.Save(attribute).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Delete soft deletes an attribute definition by setting isActive to false
func (r *AttributeDefinitionRepositoryImpl) Delete(id uint) error {
	return r.db.Model(&entity.AttributeDefinition{}).Delete("id = ?", id).Error
}

func (s *AttributeDefinitionRepositoryImpl) CreateCategoryAttributeDefinition(
	attribute *entity.AttributeDefinition,
	categoryID uint,
) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Step 1: Create the new attribute definition.
		// GORM will automatically populate the 'ID' field of the 'attribute' object upon successful creation.
		if err := tx.Create(attribute).Error; err != nil {
			// If creation fails, rollback the transaction.
			return err
		}

		// Step 2: Create the association in the join table (category_attributes).
		categoryAttribute := entity.CategoryAttribute{
			CategoryID:            categoryID,
			AttributeDefinitionID: attribute.ID,
		}

		if err := tx.Create(&categoryAttribute).Error; err != nil {
			// If association fails, rollback the transaction.
			return err
		}

		// If both operations succeed, the transaction will be committed.
		return nil
	})
}

func (r *AttributeDefinitionRepositoryImpl) CreateProductAttribute(
	attribute *entity.ProductAttribute,
) error {
	return r.db.Create(attribute).Error
}

// CreateProductAttributesBulk creates multiple product attributes in a single transaction
func (r *AttributeDefinitionRepositoryImpl) CreateProductAttributesBulk(
	attributes []*entity.ProductAttribute,
) error {
	if len(attributes) == 0 {
		return nil
	}
	return r.db.CreateInBatches(attributes, 100).Error
}

func (r *AttributeDefinitionRepositoryImpl) FindProductAttributeByProductID(
	productID uint,
) ([]entity.ProductAttribute, error) {
	var attributes []entity.ProductAttribute
	result := r.db.Preload("AttributeDefinition").
		Where("product_id = ?", productID).
		Order("sort_order ASC").
		Find(&attributes)
	if result.Error != nil {
		return nil, result.Error
	}
	return attributes, nil
}

/***********************************************
 *    Bulk Deletion Methods for Product Cleanup
 ***********************************************/

// DeleteProductAttributesByProductID deletes all product attributes for a given product
func (r *AttributeDefinitionRepositoryImpl) DeleteProductAttributesByProductID(productID uint) error {
	return r.db.Where("product_id = ?", productID).Delete(&entity.ProductAttribute{}).Error
}
