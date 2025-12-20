package repositories

import (
	"context"
	"errors"

	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	"ecommerce-be/product/utils"

	"gorm.io/gorm"
)

// AttributeDefinitionRepository defines the interface for attribute definition data operations
type AttributeDefinitionRepository interface {
	Create(ctx context.Context, attribute *entity.AttributeDefinition) error
	CreateBulk(ctx context.Context, attributes []*entity.AttributeDefinition) error
	FindByID(ctx context.Context, id uint) (*entity.AttributeDefinition, error)
	FindByKey(ctx context.Context, key string) (*entity.AttributeDefinition, error)
	FindByKeys(ctx context.Context, keys []string) (map[string]*entity.AttributeDefinition, error)
	FindAll(ctx context.Context) ([]entity.AttributeDefinition, error)
	Update(ctx context.Context, attribute *entity.AttributeDefinition) error
	UpdateBulk(ctx context.Context, attributes []*entity.AttributeDefinition) error
	Delete(ctx context.Context, id uint) error
	CreateCategoryAttributeDefinition(ctx context.Context, attribute *entity.AttributeDefinition, categoryID uint) error
	CreateProductAttribute(ctx context.Context, attribute *entity.ProductAttribute) error
	CreateProductAttributesBulk(ctx context.Context, attributes []*entity.ProductAttribute) error
	FindProductAttributeByProductID(ctx context.Context, productID uint) ([]entity.ProductAttribute, error)

	// Bulk deletion methods for product cleanup
	DeleteProductAttributesByProductID(ctx context.Context, productID uint) error
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
func (r *AttributeDefinitionRepositoryImpl) Create(ctx context.Context, attribute *entity.AttributeDefinition) error {
	return db.DB(ctx).Create(attribute).Error
}

// CreateBulk creates multiple attribute definitions in a single transaction
func (r *AttributeDefinitionRepositoryImpl) CreateBulk(
	ctx context.Context,
	attributes []*entity.AttributeDefinition,
) error {
	if len(attributes) == 0 {
		return nil
	}
	return db.DB(ctx).CreateInBatches(attributes, 100).Error
}

// FindByID finds an attribute definition by ID
func (r *AttributeDefinitionRepositoryImpl) FindByID(ctx context.Context, id uint) (*entity.AttributeDefinition, error) {
	var attribute entity.AttributeDefinition
	result := db.DB(ctx).First(&attribute, id)
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
	ctx context.Context,
	key string,
) (*entity.AttributeDefinition, error) {
	var attribute entity.AttributeDefinition
	result := db.DB(ctx).Where("key = ?", key).First(&attribute)
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
	ctx context.Context,
	keys []string,
) (map[string]*entity.AttributeDefinition, error) {
	if len(keys) == 0 {
		return make(map[string]*entity.AttributeDefinition), nil
	}

	var attributes []entity.AttributeDefinition
	result := db.DB(ctx).Where("key IN ?", keys).Find(&attributes)
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
func (r *AttributeDefinitionRepositoryImpl) FindAll(ctx context.Context) ([]entity.AttributeDefinition, error) {
	var attributes []entity.AttributeDefinition
	result := db.DB(ctx).Order("name ASC").Find(&attributes)
	if result.Error != nil {
		return nil, result.Error
	}
	return attributes, nil
}

// Update updates an existing attribute definition
func (r *AttributeDefinitionRepositoryImpl) Update(ctx context.Context, attribute *entity.AttributeDefinition) error {
	return db.DB(ctx).Save(attribute).Error
}

// UpdateBulk updates multiple attribute definitions in a single transaction
func (r *AttributeDefinitionRepositoryImpl) UpdateBulk(
	ctx context.Context,
	attributes []*entity.AttributeDefinition,
) error {
	if len(attributes) == 0 {
		return nil
	}
	// Use context-based transaction if available, otherwise use db.DB(ctx)
	gormDB := db.DB(ctx)
	for _, attribute := range attributes {
		if err := gormDB.Save(attribute).Error; err != nil {
			return err
		}
	}
	return nil
}

// Delete soft deletes an attribute definition by setting isActive to false
func (r *AttributeDefinitionRepositoryImpl) Delete(ctx context.Context, id uint) error {
	return db.DB(ctx).Model(&entity.AttributeDefinition{}).Delete("id = ?", id).Error
}

func (s *AttributeDefinitionRepositoryImpl) CreateCategoryAttributeDefinition(
	ctx context.Context,
	attribute *entity.AttributeDefinition,
	categoryID uint,
) error {
	gormDB := db.DB(ctx)
	// Step 1: Create the new attribute definition.
	// GORM will automatically populate the 'ID' field of the 'attribute' object upon successful creation.
	if err := gormDB.Create(attribute).Error; err != nil {
		return err
	}

	// Step 2: Create the association in the join table (category_attributes).
	categoryAttribute := entity.CategoryAttribute{
		CategoryID:            categoryID,
		AttributeDefinitionID: attribute.ID,
	}

	if err := gormDB.Create(&categoryAttribute).Error; err != nil {
		return err
	}

	return nil
}

func (r *AttributeDefinitionRepositoryImpl) CreateProductAttribute(
	ctx context.Context,
	attribute *entity.ProductAttribute,
) error {
	return db.DB(ctx).Create(attribute).Error
}

// CreateProductAttributesBulk creates multiple product attributes in a single transaction
func (r *AttributeDefinitionRepositoryImpl) CreateProductAttributesBulk(
	ctx context.Context,
	attributes []*entity.ProductAttribute,
) error {
	if len(attributes) == 0 {
		return nil
	}
	return db.DB(ctx).CreateInBatches(attributes, 100).Error
}

func (r *AttributeDefinitionRepositoryImpl) FindProductAttributeByProductID(
	ctx context.Context,
	productID uint,
) ([]entity.ProductAttribute, error) {
	var attributes []entity.ProductAttribute
	result := db.DB(ctx).Preload("AttributeDefinition").
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
func (r *AttributeDefinitionRepositoryImpl) DeleteProductAttributesByProductID(
	ctx context.Context,
	productID uint,
) error {
	return db.DB(ctx).Where("product_id = ?", productID).Delete(&entity.ProductAttribute{}).Error
}
