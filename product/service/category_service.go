package service

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/product/entity"
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

	// GetCategoryWithParent retrieves a category and its parent (if exists) in optimized way
	// Used by product queries to build category hierarchy efficiently
	GetCategoryWithParent(categoryID uint) (*entity.Category, *entity.Category, error)

	GetAttributesByCategoryIDWithInheritance(
		catagoryID uint,
		sellerID *uint,
	) (model.AttributeDefinitionsResponse, error)
	LinkAttributeToCategory(
		categoryID uint,
		req model.LinkAttributeRequest,
		roleLevel uint,
		sellerID uint,
	) (*model.LinkAttributeResponse, error)
	UnlinkAttributeFromCategory(
		categoryID uint,
		attributeID uint,
		roleLevel uint,
		sellerID uint,
	) error
}

// CategoryServiceImpl implements the CategoryService interface
type CategoryServiceImpl struct {
	categoryRepo     repositories.CategoryRepository
	productRepo      repositories.ProductRepository
	attributeRepo    repositories.AttributeDefinitionRepository
}

// NewCategoryService creates a new instance of CategoryService
func NewCategoryService(
	categoryRepo repositories.CategoryRepository,
	productRepo repositories.ProductRepository,
	attributeRepo repositories.AttributeDefinitionRepository,
) CategoryService {
	return &CategoryServiceImpl{
		categoryRepo:     categoryRepo,
		productRepo:      productRepo,
		attributeRepo:    attributeRepo,
	}
}

// CreateCategory creates a new category
func (s *CategoryServiceImpl) CreateCategory(
	req model.CategoryCreateRequest,
	roleLevel uint,
	sellerId uint,
) (*model.CategoryResponse, error) {
	// Fetch existing category with same name and parent (if any)
	existingCategory, err := s.categoryRepo.FindByNameAndParent(req.Name, req.ParentID)
	if err != nil {
		return nil, err
	}

	// Validate unique name within the same parent
	if err := validator.ValidateUniqueName(req.Name, req.ParentID, nil, existingCategory); err != nil {
		return nil, err
	}

	// Fetch parent category if parent ID is provided
	var parentCategory *entity.Category
	if req.ParentID != nil && *req.ParentID != 0 {
		parentCategory, err = s.categoryRepo.FindByID(*req.ParentID)
		if err != nil {
			// Convert to validation error if parent not found
			parentCategory = nil
		}
	}

	// Validate parent category if provided
	if err := validator.ValidateParentCategory(req.ParentID, parentCategory); err != nil {
		return nil, err
	}

	global := roleLevel < constants.SELLER_ROLE_LEVEL

	// Create category entity using factory
	category := factory.BuildCategoryEntityFromCreateRequest(req, global, &sellerId)

	// Save category to database
	if err := s.categoryRepo.Create(category); err != nil {
		return nil, err
	}

	// Create response using converter utility
	categoryResponse := factory.BuildCategoryResponse(category)

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

	if err := validator.ValidateCategoryOwnershipOrAdminAccess(
		roleLevel,
		sellerId,
		category,
	); err != nil {
		return nil, err
	}

	// Fetch category with new name and parent (if any) for uniqueness check
	existingCategoryWithNewName, err := s.categoryRepo.FindByNameAndParent(req.Name, req.ParentID)
	if err != nil {
		return nil, err
	}

	// Validate name change (check uniqueness within same parent)
	if err := validator.ValidateNameChange(category.Name, req.Name, req.ParentID, id, existingCategoryWithNewName); err != nil {
		return nil, err
	}

	// Build parent chain for circular reference validation
	var parentChain []*entity.Category
	if req.ParentID != nil && *req.ParentID != 0 {
		currentParentID := req.ParentID
		visited := make(map[uint]bool)

		for currentParentID != nil && *currentParentID != 0 {
			// Prevent infinite loops
			if visited[*currentParentID] {
				break
			}
			visited[*currentParentID] = true

			parent, err := s.categoryRepo.FindByID(*currentParentID)
			if err != nil {
				break
			}
			parentChain = append(parentChain, parent)
			currentParentID = parent.ParentID
		}
	}

	// Validate circular reference
	if err := validator.ValidateCircularReference(id, req.ParentID, parentChain); err != nil {
		return nil, err
	}

	// Fetch parent category if parent ID is provided
	var parentCategory *entity.Category
	if req.ParentID != nil && *req.ParentID != 0 {
		parentCategory, err = s.categoryRepo.FindByID(*req.ParentID)
		if err != nil {
			parentCategory = nil
		}
	}

	// Validate parent category if provided
	if err := validator.ValidateParentCategory(req.ParentID, parentCategory); err != nil {
		return nil, err
	}

	// Update category using factory
	category = factory.BuildCategoryEntityFromUpdateReq(category, req)

	// Save updated category
	if err := s.categoryRepo.Update(category); err != nil {
		return nil, err
	}

	// Create response using converter utility
	categoryResponse := factory.BuildCategoryResponse(category)

	return categoryResponse, nil
}

// DeleteCategory soft deletes a category
func (s *CategoryServiceImpl) DeleteCategory(
	id uint,
	roleLevel uint,
	sellerId uint,
) error {
	// Check if category has active products
	hasProducts, err := s.categoryRepo.CheckHasProducts(id)
	if err != nil {
		return err
	}

	// Check if category has active child categories
	hasChildren, err := s.categoryRepo.CheckHasChildren(id)
	if err != nil {
		return err
	}

	// Validate that category can be deleted
	if err := validator.ValidateCanDelete(hasProducts, hasChildren); err != nil {
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

	if err := validator.ValidateCategoryOwnershipOrAdminAccess(
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
		categoryResponse := factory.BuildCategoryHierarchyResponse(&category)
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
	categoryResponse := factory.BuildCategoryResponse(category)

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
		categoryResponse := factory.BuildCategoryHierarchyResponse(&category)
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
		ar := factory.BuildAttributeResponse(&attribute)
		attributesResponse = append(attributesResponse, *ar)
	}

	return model.AttributeDefinitionsResponse{
		Attributes: attributesResponse,
	}, nil
}

// LinkAttributeToCategory links an existing attribute to a category
func (s *CategoryServiceImpl) LinkAttributeToCategory(
	categoryID uint,
	req model.LinkAttributeRequest,
	roleLevel uint,
	sellerID uint,
) (*model.LinkAttributeResponse, error) {
	// Validate category exists
	category, err := s.categoryRepo.FindByID(categoryID)
	if err != nil {
		return nil, err
	}

	// Validate seller access (admin can link to any category, seller only to owned categories)
	if roleLevel == constants.SELLER_ROLE_LEVEL {
		if category.IsGlobal || category.SellerID == nil || *category.SellerID != sellerID {
			return nil, prodErrors.ErrUnauthorizedCategoryUpdate
		}
	}

	// Validate attribute exists
	_, err = s.attributeRepo.FindByID(req.AttributeDefinitionID)
	if err != nil {
		return nil, prodErrors.ErrAttributeNotFound
	}

	// Check if already linked
	existingLink, err := s.categoryRepo.CheckAttributeLinked(categoryID, req.AttributeDefinitionID)
	if err != nil {
		return nil, err
	}
	if existingLink != nil {
		return nil, prodErrors.ErrAttributeAlreadyLinked
	}

	// Create the link
	categoryAttribute := &entity.CategoryAttribute{
		CategoryID:            categoryID,
		AttributeDefinitionID: req.AttributeDefinitionID,
	}

	err = s.categoryRepo.LinkAttribute(categoryAttribute)
	if err != nil {
		return nil, err
	}

	return &model.LinkAttributeResponse{
		CategoryID:            categoryID,
		AttributeDefinitionID: req.AttributeDefinitionID,
		CreatedAt:             categoryAttribute.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// UnlinkAttributeFromCategory removes the link between an attribute and a category
func (s *CategoryServiceImpl) UnlinkAttributeFromCategory(
	categoryID uint,
	attributeID uint,
	roleLevel uint,
	sellerID uint,
) error {
	// Validate category exists
	category, err := s.categoryRepo.FindByID(categoryID)
	if err != nil {
		return err
	}

	// Validate seller access (admin can unlink from any category, seller only from owned categories)
	if roleLevel == constants.SELLER_ROLE_LEVEL {
		if category.IsGlobal || category.SellerID == nil || *category.SellerID != sellerID {
			return prodErrors.ErrUnauthorizedCategoryUpdate
		}
	}

	// Unlink the attribute
	err = s.categoryRepo.UnlinkAttribute(categoryID, attributeID)
	if err != nil {
		return err
	}

	return nil
}

/***********************************************
 *    Query Helper Methods                     *
 ***********************************************/

// GetCategoryWithParent retrieves a category and its parent (if exists)
// Optimized method to fetch both category and parent efficiently
// Used by product queries to build category hierarchy without multiple queries
func (s *CategoryServiceImpl) GetCategoryWithParent(
	categoryID uint,
) (*entity.Category, *entity.Category, error) {
	// Get category
	category, err := s.categoryRepo.FindByID(categoryID)
	if err != nil {
		return nil, nil, err
	}

	// Get parent category if exists
	var parentCategory *entity.Category
	if category.ParentID != nil && *category.ParentID != 0 {
		if pc, err := s.categoryRepo.FindByID(*category.ParentID); err == nil {
			parentCategory = pc
		}
	}

	return category, parentCategory, nil
}
