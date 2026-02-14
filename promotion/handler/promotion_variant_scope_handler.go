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

type PromotionVariantScopeHandler struct {
	*commonHandler.BaseHandler
	service service.PromotionVariantScopeService
}

func NewPromotionVariantScopeHandler(
	service service.PromotionVariantScopeService,
) *PromotionVariantScopeHandler {
	return &PromotionVariantScopeHandler{
		BaseHandler: commonHandler.NewBaseHandler(),
		service:     service,
	}
}

// AddVariants adds variants to a promotion
func (h *PromotionVariantScopeHandler) AddVariants(c *gin.Context) {
	var req model.AddPromotionVariantRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, _, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.AddVariants(c, req); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_ADD_PROMOTION_VARIANTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.PROMOTION_VARIANTS_ADDED_MSG, nil)
}

// RemoveVariants removes variants from a promotion
func (h *PromotionVariantScopeHandler) RemoveVariants(c *gin.Context) {
	var req model.RemovePromotionVariantRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, _, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.RemoveVariants(c, req); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_REMOVE_PROMOTION_VARIANTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.PROMOTION_VARIANTS_REMOVED_MSG, nil)
}

// RemoveAllVariants removes all variants from a promotion
func (h *PromotionVariantScopeHandler) RemoveAllVariants(c *gin.Context) {
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

	if err := h.service.RemoveAllVariants(c, uint(promotionID)); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_REMOVE_ALL_PROMOTION_VARIANTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.PROMOTION_ALL_VARIANTS_REMOVED_MSG, nil)
}

// GetVariants retrieves variants for a promotion
func (h *PromotionVariantScopeHandler) GetVariants(c *gin.Context) {
	var params model.GetPromotionVariantsQueryParams
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

	response, err := h.service.GetVariants(c, req)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_GET_PROMOTION_VARIANTS_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		promotionConstants.PROMOTION_VARIANTS_RETRIEVED_MSG,
		promotionConstants.PROMOTION_VARIANTS_FIELD,
		response,
	)
}
