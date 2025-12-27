package repositories

import (
	"context"
	"errors"

	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/query"
	"ecommerce-be/product/utils"

	"gorm.io/gorm"
)

// CategoryRepository defines the interface for category-related database operations
type CategoryRepository interface {
	Create(ctx context.Context, category *entity.Category) error
	Update(ctx context.Context, category *entity.Category) error
	FindByID(ctx context.Context, id uint) (*entity.Category, error)
	FindByNameAndParent(ctx context.Context, name string, parentID *uint) (*entity.Category, error)
	FindAllHierarchical(ctx context.Context, sellerID *uint) ([]entity.Category, error)
	FindByParentID(ctx context.Context, parentID *uint, sellerID *uint) ([]entity.Category, error)
	Delete(ctx context.Context, id uint) error
	CheckHasProducts(ctx context.Context, id uint) (bool, error)
	CheckHasChildren(ctx context.Context, id uint) (bool, error)
	Exists(ctx context.Context, id uint) error
	FindAttributesByCategoryIDWithInheritance(ctx context.Context, catagoryID uint) ([]entity.AttributeDefinition, error)
	LinkAttribute(ctx context.Context, categoryAttribute *entity.CategoryAttribute) error
	UnlinkAttribute(ctx context.Context, categoryID uint, attributeID uint) error
	CheckAttributeLinked(ctx context.Context, categoryID uint, attributeID uint) (*entity.CategoryAttribute, error)
}

// CategoryRepositoryImpl implements the CategoryRepository interface
type CategoryRepositoryImpl struct {
	db *gorm.DB
}

// NewCategoryRepository creates a new instance of CategoryRepository
func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &CategoryRepositoryImpl{db: db}
}

// Create creates a new category
func (r *CategoryRepositoryImpl) Create(ctx context.Context, category *entity.Category) error {
	return db.DB(ctx).Create(category).Error
}

// Update updates an existing category
func (r *CategoryRepositoryImpl) Update(ctx context.Context, category *entity.Category) error {
	// Use Updates with Select to handle pointer fields (ParentID) and force timestamp updates
	return db.DB(ctx).Model(category).
		Select("Name", "Description", "ParentID", "UpdatedAt").
		Updates(map[string]interface{}{
			"name":        category.Name,
			"description": category.Description,
			"parent_id":   category.ParentID,
			"updated_at":  category.UpdatedAt,
		}).Error
}

// FindByID finds a category by ID with eager loading
func (r *CategoryRepositoryImpl) FindByID(ctx context.Context, id uint) (*entity.Category, error) {
	var category entity.Category
	result := db.DB(ctx).Preload("Parent").Preload("Children").Where("id = ?", id).First(&category)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, prodErrors.ErrCategoryNotFound
		}
		return nil, result.Error
	}
	return &category, nil
}

// FindByNameAndParent finds a category by name and parent ID
func (r *CategoryRepositoryImpl) FindByNameAndParent(
	ctx context.Context,
	name string,
	parentID *uint,
) (*entity.Category, error) {
	var category entity.Category
	var q *gorm.DB

	if parentID != nil {
		q = db.DB(ctx).Where("name = ? AND parent_id = ?", name, *parentID)
	} else {
		q = db.DB(ctx).Where("name = ? AND parent_id IS NULL", name)
	}

	result := q.First(&category)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil // Not found, but not an error
		}
		return nil, result.Error
	}
	return &category, nil
}

// FindAllHierarchical finds all categories with hierarchical structure
// Multi-tenant: Returns global categories + seller-specific categories
// If sellerID is nil (admin), returns all categories
func (r *CategoryRepositoryImpl) FindAllHierarchical(ctx context.Context, sellerID *uint) ([]entity.Category, error) {
	var categories []entity.Category
	q := db.DB(ctx).Model(&entity.Category{})

	// Multi-tenant filter: Return global categories + seller-specific categories
	if sellerID != nil {
		q = q.Where("is_global = ? OR seller_id = ?", true, *sellerID)
	}
	// If sellerID is nil (admin), no filter applied - get all categories

	result := q.Order("name ASC").Find(&categories)
	if result.Error != nil {
		return nil, result.Error
	}
	return categories, nil
}

// FindByParentID finds categories by parent ID
func (r *CategoryRepositoryImpl) FindByParentID(
	ctx context.Context,
	parentID *uint,
	sellerID *uint,
) ([]entity.Category, error) {
	var categories []entity.Category
	var q *gorm.DB

	if parentID != nil {
		q = db.DB(ctx).Preload("Parent").Preload("Children").Where("parent_id = ?", *parentID)
	} else {
		q = db.DB(ctx).Preload("Parent").Preload("Children").Where("parent_id IS NULL")
	}

	// Apply seller filter: categories are accessible if global OR owned by seller
	if sellerID != nil {
		q = q.Where("is_global = ? OR seller_id = ?", true, *sellerID)
	}

	result := q.Order("name ASC").Find(&categories)
	if result.Error != nil {
		return nil, result.Error
	}
	return categories, nil
}

// SoftDelete soft deletes a category
func (r *CategoryRepositoryImpl) Delete(ctx context.Context, id uint) error {
	return db.DB(ctx).Model(&entity.Category{}).Delete("id = ?", id).Error
}

// CheckHasProducts checks if a category has active products
func (r *CategoryRepositoryImpl) CheckHasProducts(ctx context.Context, id uint) (bool, error) {
	var count int64
	result := db.DB(ctx).Model(&entity.Product{}).Where("category_id = ?", id).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// CheckHasChildren checks if a category has active child categories
func (r *CategoryRepositoryImpl) CheckHasChildren(ctx context.Context, id uint) (bool, error) {
	var count int64
	result := db.DB(ctx).Model(&entity.Category{}).Where("parent_id = ?", id).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// Exists checks if a category exists
func (r *CategoryRepositoryImpl) Exists(ctx context.Context, id uint) error {
	var count int64
	result := db.DB(ctx).Model(&entity.Category{}).Where("id = ?", id).Count(&count)
	if result.Error != nil {
		return result.Error
	}
	if count == 0 {
		return errors.New(utils.CATEGORY_NOT_FOUND_MSG)
	}
	return nil
}

// This method will return all attributes associated with a category,
// including inherited attributes from parent categories
func (r *CategoryRepositoryImpl) FindAttributesByCategoryIDWithInheritance(
	ctx context.Context,
	catagoryID uint,
) ([]entity.AttributeDefinition, error) {
	var attributes []entity.AttributeDefinition
	result := db.DB(ctx).Raw(query.FIND_ATTRIBUTES_BY_CATEGORY_ID_WITH_INHERITANCE_QUERY, catagoryID).
		Scan(&attributes)
	if result.Error != nil {
		return nil, result.Error
	}

	return attributes, nil
}

// LinkAttribute creates a link between a category and an attribute
func (r *CategoryRepositoryImpl) LinkAttribute(ctx context.Context, categoryAttribute *entity.CategoryAttribute) error {
	result := db.DB(ctx).Create(categoryAttribute)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// UnlinkAttribute removes the link between a category and an attribute
func (r *CategoryRepositoryImpl) UnlinkAttribute(ctx context.Context, categoryID uint, attributeID uint) error {
	result := db.DB(ctx).Where("category_id = ? AND attribute_definition_id = ?", categoryID, attributeID).
		Delete(&entity.CategoryAttribute{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return prodErrors.ErrAttributeNotLinked
	}
	return nil
}

// CheckAttributeLinked checks if an attribute is already linked to a category
func (r *CategoryRepositoryImpl) CheckAttributeLinked(
	ctx context.Context,
	categoryID uint,
	attributeID uint,
) (*entity.CategoryAttribute, error) {
	var categoryAttribute entity.CategoryAttribute
	result := db.DB(ctx).Where("category_id = ? AND attribute_definition_id = ?", categoryID, attributeID).
		First(&categoryAttribute)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &categoryAttribute, nil
}
