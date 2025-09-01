package repositories

import (
	"errors"

	"ecommerce-be/product_management/entity"
	"ecommerce-be/product_management/utils"

	"gorm.io/gorm"
)

// CategoryRepository defines the interface for category-related database operations
type CategoryRepository interface {
	Create(category *entity.Category) error
	Update(category *entity.Category) error
	FindByID(id uint) (*entity.Category, error)
	FindByNameAndParent(name string, parentID *uint) (*entity.Category, error)
	FindAllHierarchical() ([]entity.Category, error)
	FindByParentID(parentID *uint) ([]entity.Category, error)
	SoftDelete(id uint) error
	CheckHasProducts(id uint) (bool, error)
	CheckHasChildren(id uint) (bool, error)
	Exists(id uint) error
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
	result := r.db.Preload("Parent").Preload("Children").Where("id = ? AND is_active = true", id).First(&category)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New(utils.CATEGORY_NOT_FOUND_MSG)
		}
		return nil, result.Error
	}
	return &category, nil
}

// FindByNameAndParent finds a category by name and parent ID
func (r *CategoryRepositoryImpl) FindByNameAndParent(name string, parentID *uint) (*entity.Category, error) {
	var category entity.Category
	var query *gorm.DB

	if parentID != nil {
		query = r.db.Where("name = ? AND parent_id = ? AND is_active = true", name, *parentID)
	} else {
		query = r.db.Where("name = ? AND parent_id IS NULL AND is_active = true", name)
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
func (r *CategoryRepositoryImpl) FindAllHierarchical() ([]entity.Category, error) {
	var categories []entity.Category
	result := r.db.Preload("Parent").Preload("Children").Where("is_active = true").Order("name ASC").Find(&categories)
	if result.Error != nil {
		return nil, result.Error
	}
	return categories, nil
}

// FindByParentID finds categories by parent ID
func (r *CategoryRepositoryImpl) FindByParentID(parentID *uint) ([]entity.Category, error) {
	var categories []entity.Category
	var query *gorm.DB

	if parentID != nil {
		query = r.db.Preload("Parent").Preload("Children").Where("parent_id = ? AND is_active = true", *parentID)
	} else {
		query = r.db.Preload("Parent").Preload("Children").Where("parent_id IS NULL AND is_active = true")
	}

	result := query.Order("name ASC").Find(&categories)
	if result.Error != nil {
		return nil, result.Error
	}
	return categories, nil
}

// SoftDelete soft deletes a category
func (r *CategoryRepositoryImpl) SoftDelete(id uint) error {
	return r.db.Model(&entity.Category{}).Where("id = ?", id).Update("is_active", false).Error
}

// CheckHasProducts checks if a category has active products
func (r *CategoryRepositoryImpl) CheckHasProducts(id uint) (bool, error) {
	var count int64
	result := r.db.Model(&entity.Product{}).Where("category_id = ? AND is_active = true", id).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// CheckHasChildren checks if a category has active child categories
func (r *CategoryRepositoryImpl) CheckHasChildren(id uint) (bool, error) {
	var count int64
	result := r.db.Model(&entity.Category{}).Where("parent_id = ? AND is_active = true", id).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// Exists checks if a category exists
func (r *CategoryRepositoryImpl) Exists(id uint) error {
	var count int64
	result := r.db.Model(&entity.Category{}).Where("id = ? AND is_active = true", id).Count(&count)
	if result.Error != nil {
		return result.Error
	}
	if count == 0 {
		return errors.New(utils.CATEGORY_NOT_FOUND_MSG)
	}
	return nil
}
