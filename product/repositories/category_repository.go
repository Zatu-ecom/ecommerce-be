package repositories

import (
	"errors"

	"ecommerce-be/product/entity"
	"ecommerce-be/product/query"
	"ecommerce-be/product/utils"

	"gorm.io/gorm"
)

// CategoryRepository defines the interface for category-related database operations
type CategoryRepository interface {
	Create(category *entity.Category) error
	Update(category *entity.Category) error
	FindByID(id uint) (*entity.Category, error)
	FindByNameAndParent(name string, parentID *uint) (*entity.Category, error)
	FindAllHierarchical(sellerID *uint) ([]entity.Category, error)
	FindByParentID(parentID *uint, sellerID *uint) ([]entity.Category, error)
	Delete(id uint) error
	CheckHasProducts(id uint) (bool, error)
	CheckHasChildren(id uint) (bool, error)
	Exists(id uint) error
	FindAttributesByCategoryIDWithInheritance(catagoryID uint) ([]entity.AttributeDefinition, error)
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
func (r *CategoryRepositoryImpl) Create(category *entity.Category) error {
	return r.db.Create(category).Error
}

// Update updates an existing category
func (r *CategoryRepositoryImpl) Update(category *entity.Category) error {
	return r.db.Save(category).Error
}

// FindByID finds a category by ID with eager loading
func (r *CategoryRepositoryImpl) FindByID(id uint) (*entity.Category, error) {
	var category entity.Category
	result := r.db.Preload("Parent").Preload("Children").Where("id = ?", id).First(&category)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New(utils.CATEGORY_NOT_FOUND_MSG)
		}
		return nil, result.Error
	}
	return &category, nil
}

// FindByNameAndParent finds a category by name and parent ID
func (r *CategoryRepositoryImpl) FindByNameAndParent(
	name string,
	parentID *uint,
) (*entity.Category, error) {
	var category entity.Category
	var query *gorm.DB

	if parentID != nil {
		query = r.db.Where("name = ? AND parent_id = ?", name, *parentID)
	} else {
		query = r.db.Where("name = ? AND parent_id IS NULL", name)
	}

	result := query.First(&category)
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
func (r *CategoryRepositoryImpl) FindAllHierarchical(sellerID *uint) ([]entity.Category, error) {
	var categories []entity.Category
	query := r.db.Model(&entity.Category{})

	// Multi-tenant filter: Return global categories + seller-specific categories
	if sellerID != nil {
		query = query.Where("is_global = ? OR seller_id = ?", true, *sellerID)
	}
	// If sellerID is nil (admin), no filter applied - get all categories

	result := query.Order("name ASC").Find(&categories)
	if result.Error != nil {
		return nil, result.Error
	}
	return categories, nil
}

// FindByParentID finds categories by parent ID
func (r *CategoryRepositoryImpl) FindByParentID(
	parentID *uint,
	sellerID *uint,
) ([]entity.Category, error) {
	var categories []entity.Category
	var query *gorm.DB

	if parentID != nil {
		query = r.db.Preload("Parent").Preload("Children").Where("parent_id = ?", *parentID)
	} else {
		query = r.db.Preload("Parent").Preload("Children").Where("parent_id IS NULL")
	}

	// Apply seller filter: categories are accessible if global OR owned by seller
	if sellerID != nil {
		query = query.Where("is_global = ? OR seller_id = ?", true, *sellerID)
	}

	result := query.Order("name ASC").Find(&categories)
	if result.Error != nil {
		return nil, result.Error
	}
	return categories, nil
}

// SoftDelete soft deletes a category
func (r *CategoryRepositoryImpl) Delete(id uint) error {
	return r.db.Model(&entity.Category{}).Delete("id = ?", id).Error
}

// CheckHasProducts checks if a category has active products
func (r *CategoryRepositoryImpl) CheckHasProducts(id uint) (bool, error) {
	var count int64
	result := r.db.Model(&entity.Product{}).Where("category_id = ?", id).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// CheckHasChildren checks if a category has active child categories
func (r *CategoryRepositoryImpl) CheckHasChildren(id uint) (bool, error) {
	var count int64
	result := r.db.Model(&entity.Category{}).Where("parent_id = ?", id).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// Exists checks if a category exists
func (r *CategoryRepositoryImpl) Exists(id uint) error {
	var count int64
	result := r.db.Model(&entity.Category{}).Where("id = ?", id).Count(&count)
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
	catagoryID uint,
) ([]entity.AttributeDefinition, error) {
	var attributes []entity.AttributeDefinition
	result := r.db.Raw(query.FIND_ATTRIBUTES_BY_CATEGORY_ID_WITH_INHERITANCE_QUERY, catagoryID).
		Scan(&attributes)
	if result.Error != nil {
		return nil, result.Error
	}

	return attributes, nil
}
