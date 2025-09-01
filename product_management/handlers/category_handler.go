package handlers

import (
	"net/http"
	"strconv"

	"ecommerce-be/common"
	"ecommerce-be/product_management/model"
	"ecommerce-be/product_management/service"
	"ecommerce-be/product_management/utils"

	"github.com/gin-gonic/gin"
)

// CategoryHandler handles HTTP requests related to categories
type CategoryHandler struct {
	categoryService service.CategoryService
}

// NewCategoryHandler creates a new instance of CategoryHandler
func NewCategoryHandler(categoryService service.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
	}
}

// CreateCategory handles category creation
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req model.CategoryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErrors []common.ValidationError
		validationErrors = append(validationErrors, common.ValidationError{
			Field:   utils.REQUEST_FIELD_NAME,
			Message: err.Error(),
		})
		common.ErrorWithValidation(c, http.StatusBadRequest, utils.VALIDATION_FAILED_MSG, validationErrors, utils.VALIDATION_ERROR_CODE)
		return
	}

	categoryResponse, err := h.categoryService.CreateCategory(req)
	if err != nil {
		if err.Error() == utils.CATEGORY_EXISTS_MSG {
			common.ErrorWithCode(c, http.StatusConflict, err.Error(), utils.CATEGORY_EXISTS_CODE)
			return
		}
		if err.Error() == utils.INVALID_PARENT_CATEGORY_MSG {
			common.ErrorWithCode(c, http.StatusBadRequest, err.Error(), utils.INVALID_PARENT_CATEGORY_CODE)
			return
		}
		common.ErrorResp(c, http.StatusInternalServerError, utils.FAILED_TO_CREATE_CATEGORY_MSG+": "+err.Error())
		return
	}

	common.SuccessResponse(c, http.StatusCreated, utils.CATEGORY_CREATED_MSG, map[string]interface{}{
		utils.CATEGORY_FIELD_NAME: categoryResponse,
	})
}

// UpdateCategory handles category updates
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	categoryID, err := strconv.ParseUint(c.Param("categoryId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(c, http.StatusBadRequest, "Invalid category ID", utils.VALIDATION_ERROR_CODE)
		return
	}

	var req model.CategoryUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErrors []common.ValidationError
		validationErrors = append(validationErrors, common.ValidationError{
			Field:   utils.REQUEST_FIELD_NAME,
			Message: err.Error(),
		})
		common.ErrorWithValidation(c, http.StatusBadRequest, utils.VALIDATION_FAILED_MSG, validationErrors, utils.VALIDATION_ERROR_CODE)
		return
	}

	categoryResponse, err := h.categoryService.UpdateCategory(uint(categoryID), req)
	if err != nil {
		if err.Error() == utils.CATEGORY_NOT_FOUND_MSG {
			common.ErrorWithCode(c, http.StatusNotFound, err.Error(), utils.CATEGORY_NOT_FOUND_CODE)
			return
		}
		if err.Error() == utils.CATEGORY_EXISTS_MSG {
			common.ErrorWithCode(c, http.StatusConflict, err.Error(), utils.CATEGORY_EXISTS_CODE)
			return
		}
		if err.Error() == utils.INVALID_PARENT_CATEGORY_MSG {
			common.ErrorWithCode(c, http.StatusBadRequest, err.Error(), utils.INVALID_PARENT_CATEGORY_CODE)
			return
		}
		common.ErrorResp(c, http.StatusInternalServerError, utils.FAILED_TO_UPDATE_CATEGORY_MSG+": "+err.Error())
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.CATEGORY_UPDATED_MSG, map[string]interface{}{
		utils.CATEGORY_FIELD_NAME: categoryResponse,
	})
}

// DeleteCategory handles category deletion
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	categoryID, err := strconv.ParseUint(c.Param("categoryId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(c, http.StatusBadRequest, "Invalid category ID", utils.VALIDATION_ERROR_CODE)
		return
	}

	err = h.categoryService.DeleteCategory(uint(categoryID))
	if err != nil {
		if err.Error() == utils.CATEGORY_NOT_FOUND_MSG {
			common.ErrorWithCode(c, http.StatusNotFound, err.Error(), utils.CATEGORY_NOT_FOUND_CODE)
			return
		}
		if err.Error() == utils.CATEGORY_HAS_PRODUCTS_MSG {
			common.ErrorWithCode(c, http.StatusConflict, err.Error(), utils.CATEGORY_HAS_PRODUCTS_CODE)
			return
		}
		if err.Error() == utils.CATEGORY_HAS_CHILDREN_MSG {
			common.ErrorWithCode(c, http.StatusConflict, err.Error(), utils.CATEGORY_HAS_CHILDREN_CODE)
			return
		}
		common.ErrorResp(c, http.StatusInternalServerError, utils.FAILED_TO_DELETE_CATEGORY_MSG+": "+err.Error())
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.CATEGORY_DELETED_MSG, nil)
}

// GetAllCategories handles getting all categories
func (h *CategoryHandler) GetAllCategories(c *gin.Context) {
	categoriesResponse, err := h.categoryService.GetAllCategories()
	if err != nil {
		common.ErrorResp(c, http.StatusInternalServerError, utils.FAILED_TO_GET_CATEGORIES_MSG+": "+err.Error())
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.CATEGORIES_RETRIEVED_MSG, categoriesResponse)
}

// GetCategoryByID handles getting a category by ID
func (h *CategoryHandler) GetCategoryByID(c *gin.Context) {
	categoryID, err := strconv.ParseUint(c.Param("categoryId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(c, http.StatusBadRequest, "Invalid category ID", utils.VALIDATION_ERROR_CODE)
		return
	}

	categoryResponse, err := h.categoryService.GetCategoryByID(uint(categoryID))
	if err != nil {
		if err.Error() == utils.CATEGORY_NOT_FOUND_MSG {
			common.ErrorWithCode(c, http.StatusNotFound, err.Error(), utils.CATEGORY_NOT_FOUND_CODE)
			return
		}
		common.ErrorResp(c, http.StatusInternalServerError, utils.FAILED_TO_GET_CATEGORIES_MSG+": "+err.Error())
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.CATEGORIES_RETRIEVED_MSG, map[string]interface{}{
		utils.CATEGORY_FIELD_NAME: categoryResponse,
	})
}

// GetCategoriesByParent handles getting categories by parent ID
func (h *CategoryHandler) GetCategoriesByParent(c *gin.Context) {
	parentIDStr := c.Query("parentId")
	var parentID *uint

	if parentIDStr != "" {
		parsedID, err := strconv.ParseUint(parentIDStr, 10, 32)
		if err != nil {
			common.ErrorWithCode(c, http.StatusBadRequest, "Invalid parent ID", utils.VALIDATION_ERROR_CODE)
			return
		}
		parentID = new(uint)
		*parentID = uint(parsedID)
	}

	categoriesResponse, err := h.categoryService.GetCategoriesByParent(parentID)
	if err != nil {
		common.ErrorResp(c, http.StatusInternalServerError, utils.FAILED_TO_GET_CATEGORIES_MSG+": "+err.Error())
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.CATEGORIES_RETRIEVED_MSG, categoriesResponse)
}
