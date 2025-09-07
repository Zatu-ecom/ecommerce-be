package service

import (
	"errors"
	"time"

	commonEntity "ecommerce-be/common/entity"
	"ecommerce-be/product_management/entity"
	"ecommerce-be/product_management/model"
	"ecommerce-be/product_management/repositories"
	"ecommerce-be/product_management/utils"
)

// CategoryService defines the interface for category-related business logic
type CategoryService interface {
	CreateCategory(req model.CategoryCreateRequest) (*model.CategoryResponse, error)
	UpdateCategory(id uint, req model.CategoryUpdateRequest) (*model.CategoryResponse, error)
	DeleteCategory(id uint) error
	GetAllCategories() (*model.CategoriesResponse, error)
	GetCategoryByID(id uint) (*model.CategoryResponse, error)
	GetCategoriesByParent(parentID *uint) (*model.CategoriesResponse, error)
}

// CategoryServiceImpl implements the CategoryService interface
type CategoryServiceImpl struct {
	categoryRepo repositories.CategoryRepository
}

// NewCategoryService creates a new instance of CategoryService
func NewCategoryService(categoryRepo repositories.CategoryRepository) CategoryService {
	return &CategoryServiceImpl{
		categoryRepo: categoryRepo,
	}
}

// CreateCategory creates a new category
func (s *CategoryServiceImpl) CreateCategory(req model.CategoryCreateRequest) (*model.CategoryResponse, error) {
	// Check if category with same name exists in the same parent

	existingCategory, err := s.categoryRepo.FindByNameAndParent(req.Name, req.ParentID)
	if err != nil {
		return nil, err
	}
	if existingCategory != nil {
		return nil, errors.New(utils.CATEGORY_EXISTS_MSG)
	}

	// Validate parent category if provided
	if req.ParentID != nil {
		parentCategory, err := s.categoryRepo.FindByID(*req.ParentID)
		if err != nil {
			return nil, err
		}
		if parentCategory == nil {
			return nil, errors.New(utils.INVALID_PARENT_CATEGORY_MSG)
		}
	}

	// Create category entity
	category := &entity.Category{
		Name:        req.Name,
		ParentID:    req.ParentID,
		Description: req.Description,
		IsActive:    true,
		BaseEntity: commonEntity.BaseEntity{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// Save category to database
	if err := s.categoryRepo.Create(category); err != nil {
		return nil, err
	}

	// Create response using converter utility
	categoryResponse := utils.ConvertCategoryToResponse(category)

	return categoryResponse, nil
}

// UpdateCategory updates an existing category
func (s *CategoryServiceImpl) UpdateCategory(id uint, req model.CategoryUpdateRequest) (*model.CategoryResponse, error) {
	// Find existing category
	category, err := s.categoryRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Check if category with same name exists in the same parent (excluding current category)
	if req.Name != category.Name {
		existingCategory, err := s.categoryRepo.FindByNameAndParent(req.Name, req.ParentID)
		if err != nil {
			return nil, err
		}
		if existingCategory != nil && existingCategory.ID != id {
			return nil, errors.New(utils.CATEGORY_EXISTS_MSG)
		}
	}

	// Validate parent category if provided
	if req.ParentID != nil && *req.ParentID != 0 {
		// Prevent circular reference
		if *req.ParentID == id {
			return nil, errors.New("Category cannot be its own parent")
		}

		parentCategory, err := s.categoryRepo.FindByID(*req.ParentID)
		if err != nil {
			return nil, err
		}
		if parentCategory == nil {
			return nil, errors.New(utils.INVALID_PARENT_CATEGORY_MSG)
		}
	}

	// Update category fields
	category.Name = req.Name
	category.ParentID = req.ParentID
	category.Description = req.Description
	category.IsActive = req.IsActive
	category.UpdatedAt = time.Now()

	// Save updated category
	if err := s.categoryRepo.Update(category); err != nil {
		return nil, err
	}

	// Create response using converter utility
	categoryResponse := utils.ConvertCategoryToResponse(category)

	return categoryResponse, nil
}

// DeleteCategory soft deletes a category
func (s *CategoryServiceImpl) DeleteCategory(id uint) error {
	// Check if category has active products
	hasProducts, err := s.categoryRepo.CheckHasProducts(id)
	if err != nil {
		return err
	}
	if hasProducts {
		return errors.New(utils.CATEGORY_HAS_PRODUCTS_MSG)
	}

	// Check if category has active child categories
	hasChildren, err := s.categoryRepo.CheckHasChildren(id)
	if err != nil {
		return err
	}
	if hasChildren {
		return errors.New(utils.CATEGORY_HAS_CHILDREN_MSG)
	}

	// Soft delete category
	return s.categoryRepo.SoftDelete(id)
}

// GetAllCategories gets all categories in hierarchical structure
func (s *CategoryServiceImpl) GetAllCategories() (*model.CategoriesResponse, error) {
	categories, err := s.categoryRepo.FindAllHierarchical()
	if err != nil {
		return nil, err
	}

	// Build hierarchical structure
	categoryMap := make(map[uint]*model.CategoryHierarchyResponse)
	var rootCategories []*model.CategoryHierarchyResponse

	for _, category := range categories {
		categoryResponse := utils.ConvertCategoryToHierarchyResponse(&category)
		categoryMap[category.ID] = categoryResponse

		if category.ParentID == nil || *category.ParentID == 0 {
			rootCategories = append(rootCategories, categoryResponse)
		}
	}

	// Build parent-child relationships
	for _, category := range categories {
		if category.ParentID != nil && *category.ParentID != 0 {
			if parent, exists := categoryMap[*category.ParentID]; exists {
				child := categoryMap[category.ID]
				parent.Children = append(parent.Children, *child)
			}
		}
	}

	// Convert to slice
	var categoriesResponse []model.CategoryHierarchyResponse
	for _, root := range rootCategories {
		categoriesResponse = append(categoriesResponse, *root)
	}

	return &model.CategoriesResponse{
		Categories: categoriesResponse,
	}, nil
}

// GetCategoryByID gets a category by ID
func (s *CategoryServiceImpl) GetCategoryByID(id uint) (*model.CategoryResponse, error) {
	category, err := s.categoryRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Create response using converter utility
	categoryResponse := utils.ConvertCategoryToResponse(category)

	return categoryResponse, nil
}

// GetCategoriesByParent gets categories by parent ID
func (s *CategoryServiceImpl) GetCategoriesByParent(parentID *uint) (*model.CategoriesResponse, error) {
	categories, err := s.categoryRepo.FindByParentID(parentID)
	if err != nil {
		return nil, err
	}

	var categoriesResponse []model.CategoryHierarchyResponse
	for _, category := range categories {
		categoryResponse := utils.ConvertCategoryToHierarchyResponse(&category)
		categoriesResponse = append(categoriesResponse, *categoryResponse)
	}

	return &model.CategoriesResponse{
		Categories: categoriesResponse,
	}, nil
}
