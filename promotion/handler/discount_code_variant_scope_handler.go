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

type DiscountCodeVariantScopeHandler struct {
	*commonHandler.BaseHandler
	service service.DiscountCodeVariantScopeService
}

func NewDiscountCodeVariantScopeHandler(
	svc service.DiscountCodeVariantScopeService,
) *DiscountCodeVariantScopeHandler {
	return &DiscountCodeVariantScopeHandler{
		BaseHandler: commonHandler.NewBaseHandler(),
		service:     svc,
	}
}

func (h *DiscountCodeVariantScopeHandler) AddVariants(c *gin.Context) {
	var req model.AddDiscountCodeVariantRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.AddVariants(c, req, sellerID); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_ADD_DISCOUNT_CODE_VARIANTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.DISCOUNT_CODE_VARIANTS_ADDED_MSG, nil)
}

func (h *DiscountCodeVariantScopeHandler) RemoveVariants(c *gin.Context) {
	var req model.RemoveDiscountCodeVariantRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.RemoveVariants(c, req, sellerID); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_REMOVE_DISCOUNT_CODE_VARIANTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.DISCOUNT_CODE_VARIANTS_REMOVED_MSG, nil)
}

func (h *DiscountCodeVariantScopeHandler) RemoveAllVariants(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("discountCodeId"), 10, 64)
	if err != nil {
		h.HandleError(c, err, promotionConstants.INVALID_DISCOUNT_CODE_ID_MSG)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.RemoveAllVariants(c, uint(id), sellerID); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_REMOVE_ALL_DISCOUNT_CODE_VARIANTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.DISCOUNT_CODE_ALL_VARIANTS_REMOVED_MSG, nil)
}

func (h *DiscountCodeVariantScopeHandler) GetVariants(c *gin.Context) {
	var params model.GetDiscountCodeVariantsQueryParams
	if idStr := c.Param("discountCodeId"); idStr != "" {
		if id, err := strconv.ParseUint(idStr, 10, 64); err == nil {
			params.DiscountCodeID = uint(id)
		}
	}
	if err := c.ShouldBindQuery(&params); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	response, err := h.service.GetVariants(c, params.ToRequest(), sellerID)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_GET_DISCOUNT_CODE_VARIANTS_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		promotionConstants.DISCOUNT_CODE_VARIANTS_RETRIEVED_MSG,
		promotionConstants.DISCOUNT_CODE_VARIANTS_FIELD,
		response,
	)
}
