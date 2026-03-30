package handler

import (
	"net/http"
	"strconv"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	commonHandler "ecommerce-be/common/handler"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/service"
	promotionConstants "ecommerce-be/promotion/utils/constant"

	"github.com/gin-gonic/gin"
)

type PromotionCategoryScopeHandler struct {
	*commonHandler.BaseHandler
	service service.PromotionCategoryScopeService
}

func NewPromotionCategoryScopeHandler(
	service service.PromotionCategoryScopeService,
) *PromotionCategoryScopeHandler {
	return &PromotionCategoryScopeHandler{
		BaseHandler: commonHandler.NewBaseHandler(),
		service:     service,
	}
}

// AddCategories adds categories to a promotion
func (h *PromotionCategoryScopeHandler) AddCategories(c *gin.Context) {
	var req model.AddPromotionCategoryRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, _, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.AddCategories(c, req); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_ADD_PROMOTION_CATEGORIES_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.PROMOTION_CATEGORIES_ADDED_MSG, nil)
}

// RemoveCategories removes categories from a promotion
func (h *PromotionCategoryScopeHandler) RemoveCategories(c *gin.Context) {
	var req model.RemovePromotionCategoryRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, _, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.RemoveCategories(c, req); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_REMOVE_PROMOTION_CATEGORIES_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.PROMOTION_CATEGORIES_REMOVED_MSG, nil)
}

// RemoveAllCategories removes all categories from a promotion
func (h *PromotionCategoryScopeHandler) RemoveAllCategories(c *gin.Context) {
	promotionIDStr := c.Param("promotionId")
	promotionID, err := strconv.ParseUint(promotionIDStr, 10, 64)
	if err != nil {
		h.HandleError(c, err, promotionConstants.INVALID_PROMOTION_ID_MSG)
		return
	}

	_, _, err = auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.RemoveAllCategories(c, uint(promotionID)); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_REMOVE_ALL_PROMOTION_CATEGORIES_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.PROMOTION_ALL_CATEGORIES_REMOVED_MSG, nil)
}

// GetCategories retrieves categories for a promotion
func (h *PromotionCategoryScopeHandler) GetCategories(c *gin.Context) {
	var params model.GetPromotionCategoriesQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	req := params.ToRequest()

	promotionIDStr := c.Param("promotionId")
	if promotionIDStr != "" {
		id, err := strconv.ParseUint(promotionIDStr, 10, 64)
		if err == nil {
			req.PromotionID = uint(id)
		}
	}

	response, err := h.service.GetCategories(c, req)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_GET_PROMOTION_CATEGORIES_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		promotionConstants.PROMOTION_CATEGORIES_RETRIEVED_MSG,
		promotionConstants.PROMOTION_CATEGORIES_FIELD,
		response,
	)
}
