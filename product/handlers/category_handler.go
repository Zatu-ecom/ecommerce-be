package handlers

import (
	"net/http"
	"strconv"

	"ecommerce-be/product/model"
	"ecommerce-be/product/service"
	"ecommerce-be/product/utils"

	"github.com/gin-gonic/gin"
)

// CategoryHandler handles HTTP requests related to categories
type CategoryHandler struct {
	*BaseHandler
	categoryService service.CategoryService
}

// NewCategoryHandler creates a new instance of CategoryHandler
func NewCategoryHandler(categoryService service.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		BaseHandler:     NewBaseHandler(),
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

	categoryResponse, err := h.categoryService.CreateCategory(req)
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

	categoryResponse, err := h.categoryService.UpdateCategory(categoryID, req)
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
	categoriesResponse, err := h.categoryService.GetAllCategories()
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

	categoryResponse, err := h.categoryService.GetCategoryByID(categoryID)
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

	categoriesResponse, err := h.categoryService.GetCategoriesByParent(parentID)
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

	attributesResponse, err := h.categoryService.GetAttributesByCategoryIDWithInheritance(
		categoryID,
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
