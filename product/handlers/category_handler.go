package handlers

import (
	"net/http"
	"strconv"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/handler"
	"ecommerce-be/product/model"
	"ecommerce-be/product/service"
	"ecommerce-be/product/utils"

	"github.com/gin-gonic/gin"
)

// CategoryHandler handles HTTP requests related to categories
type CategoryHandler struct {
	*handler.BaseHandler
	categoryService service.CategoryService
}

// NewCategoryHandler creates a new instance of CategoryHandler
func NewCategoryHandler(categoryService service.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		BaseHandler:     handler.NewBaseHandler(),
		categoryService: categoryService,
	}
}

// CreateCategory handles category creation
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req model.CategoryCreateRequest

	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}
	roleLevel, exists := auth.GetUserRoleLevelFromContext(c)
	if !exists {
		h.HandleError(c, commonError.ErrRoleDataMissing, constants.ROLE_DATA_MISSING_MSG)
		return
	}
	sellerId, sellerExists := auth.GetSellerIDFromContext(c)
	if !sellerExists && roleLevel >= constants.SELLER_ROLE_LEVEL {
		h.HandleError(c, commonError.UnauthorizedError, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}
	categoryResponse, err := h.categoryService.CreateCategory(req, roleLevel, sellerId)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_CREATE_CATEGORY_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusCreated,
		utils.CATEGORY_CREATED_MSG,
		utils.CATEGORY_FIELD_NAME,
		categoryResponse,
	)
}

// UpdateCategory handles category updates
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	categoryID, err := h.ParseUintParam(c, "categoryId")
	if err != nil {
		h.HandleError(c, err, "Invalid category ID")
		return
	}

	var req model.CategoryUpdateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}
	roleLevel, exists := auth.GetUserRoleLevelFromContext(c)
	if !exists {
		h.HandleError(c, commonError.ErrRoleDataMissing, constants.ROLE_DATA_MISSING_MSG)
		return
	}
	sellerId, sellerExists := auth.GetSellerIDFromContext(c)
	if !sellerExists && roleLevel >= constants.SELLER_ROLE_LEVEL {
		h.HandleError(c, commonError.UnauthorizedError, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}
	categoryResponse, err := h.categoryService.UpdateCategory(categoryID, req, roleLevel, sellerId)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_UPDATE_CATEGORY_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.CATEGORY_UPDATED_MSG,
		utils.CATEGORY_FIELD_NAME,
		categoryResponse,
	)
}

// DeleteCategory handles category deletion
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	categoryID, err := h.ParseUintParam(c, "categoryId")
	if err != nil {
		h.HandleError(c, err, "Invalid category ID")
		return
	}

	err = h.categoryService.DeleteCategory(categoryID)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_DELETE_CATEGORY_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.CATEGORY_DELETED_MSG, nil)
}

// GetAllCategories handles getting all categories
func (h *CategoryHandler) GetAllCategories(c *gin.Context) {
	// Get seller ID from context if available (for multi-tenant isolation)
	// Returns global categories + seller-specific categories
	// If not present (admin), returns all categories
	var sellerIDPtr *uint
	if sellerID, exists := auth.GetSellerIDFromContext(c); exists {
		sellerIDPtr = &sellerID
	}

	categoriesResponse, err := h.categoryService.GetAllCategories(sellerIDPtr)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_GET_CATEGORIES_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.CATEGORIES_RETRIEVED_MSG, categoriesResponse)
}

// GetCategoryByID handles getting a category by ID
func (h *CategoryHandler) GetCategoryByID(c *gin.Context) {
	categoryID, err := h.ParseUintParam(c, "categoryId")
	if err != nil {
		h.HandleError(c, err, "Invalid category ID")
		return
	}

	// Get seller ID from context if available (for multi-tenant isolation)
	// Verify category is accessible (global or belongs to seller)
	// If not present (admin), allow access to any category
	var sellerIDPtr *uint
	if sellerID, exists := auth.GetSellerIDFromContext(c); exists {
		sellerIDPtr = &sellerID
	}

	categoryResponse, err := h.categoryService.GetCategoryByID(categoryID, sellerIDPtr)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_GET_CATEGORIES_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.CATEGORIES_RETRIEVED_MSG,
		utils.CATEGORY_FIELD_NAME,
		categoryResponse,
	)
}

// GetCategoriesByParent handles getting categories by parent ID
func (h *CategoryHandler) GetCategoriesByParent(c *gin.Context) {
	parentIDStr := c.Query("parentId")
	var parentID *uint

	if parentIDStr != "" {
		parsedID, err := h.ParseUintParam(c, "parentId")
		if err != nil {
			// Try parsing from query string
			val, parseErr := strconv.ParseUint(parentIDStr, 10, 32)
			if parseErr != nil {
				h.HandleError(c, err, "Invalid parent ID")
				return
			}
			parsedID = uint(val)
		}
		parentID = &parsedID
	}

	// Extract seller ID from context (set by PublicAPIAuth middleware)
	var sellerID *uint
	if id, exists := auth.GetSellerIDFromContext(c); exists {
		sellerID = &id
	}

	categoriesResponse, err := h.categoryService.GetCategoriesByParent(parentID, sellerID)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_GET_CATEGORIES_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.CATEGORIES_RETRIEVED_MSG, categoriesResponse)
}

func (h *CategoryHandler) GetAttributesByCategoryIDWithInheritance(c *gin.Context) {
	categoryID, err := h.ParseUintParam(c, "categoryId")
	if err != nil {
		h.HandleError(c, err, "Invalid category ID")
		return
	}

	// Extract seller ID from context (set by PublicAPIAuth middleware)
	var sellerID *uint
	if id, exists := auth.GetSellerIDFromContext(c); exists {
		sellerID = &id
	}

	attributesResponse, err := h.categoryService.GetAttributesByCategoryIDWithInheritance(
		categoryID,
		sellerID,
	)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_GET_CATEGORY_ATTRIBUTES_MSG)
		return
	}

	h.Success(
		c,
		http.StatusOK,
		utils.CATEGORY_ATTRIBUTES_RETRIEVED_MSG,
		attributesResponse,
	)
}
