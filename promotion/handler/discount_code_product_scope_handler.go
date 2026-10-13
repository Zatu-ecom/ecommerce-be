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

type DiscountCodeProductScopeHandler struct {
	*commonHandler.BaseHandler
	service service.DiscountCodeProductScopeService
}

func NewDiscountCodeProductScopeHandler(
	svc service.DiscountCodeProductScopeService,
) *DiscountCodeProductScopeHandler {
	return &DiscountCodeProductScopeHandler{
		BaseHandler: commonHandler.NewBaseHandler(),
		service:     svc,
	}
}

func (h *DiscountCodeProductScopeHandler) AddProducts(c *gin.Context) {
	var req model.AddDiscountCodeProductRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.AddProducts(c, req, sellerID); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_ADD_DISCOUNT_CODE_PRODUCTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.DISCOUNT_CODE_PRODUCTS_ADDED_MSG, nil)
}

func (h *DiscountCodeProductScopeHandler) RemoveProducts(c *gin.Context) {
	var req model.RemoveDiscountCodeProductRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.RemoveProducts(c, req, sellerID); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_REMOVE_DISCOUNT_CODE_PRODUCTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.DISCOUNT_CODE_PRODUCTS_REMOVED_MSG, nil)
}

func (h *DiscountCodeProductScopeHandler) RemoveAllProducts(c *gin.Context) {
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

	if err := h.service.RemoveAllProducts(c, uint(id), sellerID); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_REMOVE_ALL_DISCOUNT_CODE_PRODUCTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.DISCOUNT_CODE_ALL_PRODUCTS_REMOVED_MSG, nil)
}

func (h *DiscountCodeProductScopeHandler) GetProducts(c *gin.Context) {
	var params model.GetDiscountCodeProductsQueryParams
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

	response, err := h.service.GetProducts(c, params.ToRequest(), sellerID)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_GET_DISCOUNT_CODE_PRODUCTS_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		promotionConstants.DISCOUNT_CODE_PRODUCTS_RETRIEVED_MSG,
		promotionConstants.DISCOUNT_CODE_PRODUCTS_FIELD,
		response,
	)
}
