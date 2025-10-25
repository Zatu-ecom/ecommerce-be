package service

import (
	"ecommerce-be/common/constants"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/factory"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/validator"
)

// CategoryService defines the interface for category-related business logic
type CategoryService interface {
	CreateCategory(
		req model.CategoryCreateRequest,
		roleLevel uint,
		sellerId uint,
	) (*model.CategoryResponse, error)
	UpdateCategory(
		id uint,
		req model.CategoryUpdateRequest,
		roleLevel uint,
		sellerId uint,
	) (*model.CategoryResponse, error)
	DeleteCategory(
		id uint,
		roleLevel uint,
		sellerId uint,
	) error
	GetAllCategories(sellerID *uint) (*model.CategoriesResponse, error)
	GetCategoryByID(id uint, sellerID *uint) (*model.CategoryResponse, error)
	GetCategoriesByParent(parentID *uint, sellerID *uint) (*model.CategoriesResponse, error)
	GetAttributesByCategoryIDWithInheritance(
		catagoryID uint,
		sellerID *uint,
	) (model.AttributeDefinitionsResponse, error)
}

// CategoryServiceImpl implements the CategoryService interface
type CategoryServiceImpl struct {
	categoryRepo     repositories.CategoryRepository
	productRepo      repositories.ProductRepository
	validator        *validator.CategoryValidator
	factory          *factory.CategoryFactory
	attributeFactory *factory.AttributeFactory
}

// NewCategoryService creates a new instance of CategoryService
func NewCategoryService(
	categoryRepo repositories.CategoryRepository,
	productRepo repositories.ProductRepository,
) CategoryService {
	return &CategoryServiceImpl{
		categoryRepo:     categoryRepo,
		productRepo:      productRepo,
		validator:        validator.NewCategoryValidator(categoryRepo),
		factory:          factory.NewCategoryFactory(),
		attributeFactory: factory.NewAttributeFactory(),
	}
}

// CreateCategory creates a new category
func (s *CategoryServiceImpl) CreateCategory(
	req model.CategoryCreateRequest,
	roleLevel uint,
	sellerId uint,
) (*model.CategoryResponse, error) {
	// Validate unique name within the same parent
	if err := s.validator.ValidateUniqueName(req.Name, req.ParentID, nil); err != nil {
		return nil, err
	}

	// Validate parent category if provided
	if err := s.validator.ValidateParentCategory(req.ParentID); err != nil {
		return nil, err
	}

	global := roleLevel < constants.SELLER_ROLE_LEVEL

	// Create category entity using factory
	category := s.factory.CreateFromRequest(req, global, &sellerId)

	// Save category to database
	if err := s.categoryRepo.Create(category); err != nil {
		return nil, err
	}

	// Create response using converter utility
	categoryResponse := s.factory.BuildCategoryResponse(category)

	return categoryResponse, nil
}

// UpdateCategory updates an existing category
func (s *CategoryServiceImpl) UpdateCategory(
	id uint,
	req model.CategoryUpdateRequest,
	roleLevel uint,
	sellerId uint,
) (*model.CategoryResponse, error) {
	// Find existing category
	category, err := s.categoryRepo.FindByID(id)
	if err != nil {
		// Check if it's a "not found" error
		if err.Error() == "Category not found" {
			return nil, prodErrors.ErrCategoryNotFound
		}
		return nil, err
	}

	if err := s.validator.ValidateCategoryOwnershipOrAdminAccess(
		roleLevel,
		sellerId,
		category,
	); err != nil {
		return nil, err
	}

	// Validate name change (check uniqueness within same parent)
	if err := s.validator.ValidateNameChange(category.Name, req.Name, req.ParentID, id); err != nil {
		return nil, err
	}

	// Validate circular reference
	if err := s.validator.ValidateCircularReference(id, req.ParentID); err != nil {
		return nil, err
	}

	// Validate parent category if provided
	if err := s.validator.ValidateParentCategory(req.ParentID); err != nil {
		return nil, err
	}

	// Update category using factory
	category = s.factory.UpdateEntity(category, req)

	// Save updated category
	if err := s.categoryRepo.Update(category); err != nil {
		return nil, err
	}

	// Create response using converter utility
	categoryResponse := s.factory.BuildCategoryResponse(category)

	return categoryResponse, nil
}

// DeleteCategory soft deletes a category
func (s *CategoryServiceImpl) DeleteCategory(
	id uint,
	roleLevel uint,
	sellerId uint,
) error {
	// Validate that category can be deleted
	if err := s.validator.ValidateCanDelete(id); err != nil {
		return err
	}

	category, err := s.categoryRepo.FindByID(id)
	if err != nil {
		// Check if it's a "not found" error
		if err.Error() == "Category not found" {
			return prodErrors.ErrCategoryNotFound
		}
		return err
	}

	if err := s.validator.ValidateCategoryOwnershipOrAdminAccess(
		roleLevel,
		sellerId,
		category,
	); err != nil {
		return err
	}

	// Soft delete category
	return s.categoryRepo.Delete(id)
}

// GetAllCategories gets all categories in hierarchical structure
// Multi-tenant: Returns global categories + seller-specific categories
// If sellerID is nil (admin), returns all categories
func (s *CategoryServiceImpl) GetAllCategories(sellerID *uint) (*model.CategoriesResponse, error) {
	categories, err := s.categoryRepo.FindAllHierarchical(sellerID)
	if err != nil {
		return nil, err
	}

	// Build hierarchical structure
	categoryMap := make(map[uint]*model.CategoryHierarchyResponse)
	var rootCategories []*model.CategoryHierarchyResponse

	for _, category := range categories {
		categoryResponse := s.factory.BuildCategoryHierarchyResponse(&category)
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
// Multi-tenant: If sellerID is provided, verify category is accessible
// (global or belongs to seller). If nil (admin), allow access to any.
func (s *CategoryServiceImpl) GetCategoryByID(
	id uint,
	sellerID *uint,
) (*model.CategoryResponse, error) {
	category, err := s.categoryRepo.FindByID(id)
	if err != nil {
		// Check if it's a "not found" error
		if err.Error() == "Category not found" {
			return nil, prodErrors.ErrCategoryNotFound
		}
		return nil, err
	}

	// Multi-tenant check: If seller ID is provided, verify category is accessible
	// Category is accessible if:
	// 1. IsGlobal = true (available to all sellers)
	// 2. SellerID matches the requesting seller
	if sellerID != nil {
		isAccessible := category.IsGlobal ||
			(category.SellerID != nil && *category.SellerID == *sellerID)

		if !isAccessible {
			return nil, prodErrors.ErrCategoryNotFound // Return proper 404 error
		}
	}

	// Create response using converter utility
	categoryResponse := s.factory.BuildCategoryResponse(category)

	return categoryResponse, nil
}

// GetCategoriesByParent gets categories by parent ID
func (s *CategoryServiceImpl) GetCategoriesByParent(
	parentID *uint,
	sellerID *uint,
) (*model.CategoriesResponse, error) {
	categories, err := s.categoryRepo.FindByParentID(parentID, sellerID)
	if err != nil {
		return nil, err
	}

	var categoriesResponse []model.CategoryHierarchyResponse
	for _, category := range categories {
		categoryResponse := s.factory.BuildCategoryHierarchyResponse(&category)
		categoriesResponse = append(categoriesResponse, *categoryResponse)
	}

	return &model.CategoriesResponse{
		Categories: categoriesResponse,
	}, nil
}

func (s *CategoryServiceImpl) GetAttributesByCategoryIDWithInheritance(
	catagoryID uint,
	sellerID *uint,
) (model.AttributeDefinitionsResponse, error) {
	// Validate category exists and seller has access
	category, err := s.categoryRepo.FindByID(catagoryID)
	if err != nil {
		return model.AttributeDefinitionsResponse{}, err
	}

	// Validate seller access: categories are accessible if global OR owned by seller
	if sellerID != nil && !category.IsGlobal &&
		(category.SellerID == nil || *category.SellerID != *sellerID) {
		return model.AttributeDefinitionsResponse{}, prodErrors.ErrCategoryNotFound
	}

	attributes, err := s.categoryRepo.FindAttributesByCategoryIDWithInheritance(catagoryID)
	if err != nil {
		return model.AttributeDefinitionsResponse{}, err
	}

	var attributesResponse []model.AttributeDefinitionResponse
	for _, attribute := range attributes {
		ar := s.attributeFactory.BuildAttributeResponse(&attribute)
		attributesResponse = append(attributesResponse, *ar)
	}

	return model.AttributeDefinitionsResponse{
		Attributes: attributesResponse,
	}, nil
}
