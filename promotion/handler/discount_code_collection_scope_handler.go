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

type DiscountCodeCollectionScopeHandler struct {
	*commonHandler.BaseHandler
	service service.DiscountCodeCollectionScopeService
}

func NewDiscountCodeCollectionScopeHandler(
	svc service.DiscountCodeCollectionScopeService,
) *DiscountCodeCollectionScopeHandler {
	return &DiscountCodeCollectionScopeHandler{
		BaseHandler: commonHandler.NewBaseHandler(),
		service:     svc,
	}
}

func (h *DiscountCodeCollectionScopeHandler) AddCollections(c *gin.Context) {
	var req model.AddDiscountCodeCollectionRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.AddCollections(c, req, sellerID); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_ADD_DISCOUNT_CODE_COLLECTIONS_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.DISCOUNT_CODE_COLLECTIONS_ADDED_MSG, nil)
}

func (h *DiscountCodeCollectionScopeHandler) RemoveCollections(c *gin.Context) {
	var req model.RemoveDiscountCodeCollectionRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.RemoveCollections(c, req, sellerID); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_REMOVE_DISCOUNT_CODE_COLLECTIONS_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.DISCOUNT_CODE_COLLECTIONS_REMOVED_MSG, nil)
}

func (h *DiscountCodeCollectionScopeHandler) RemoveAllCollections(c *gin.Context) {
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

	if err := h.service.RemoveAllCollections(c, uint(id), sellerID); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_REMOVE_ALL_DISCOUNT_CODE_COLLECTIONS_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.DISCOUNT_CODE_ALL_COLLECTIONS_REMOVED_MSG, nil)
}

func (h *DiscountCodeCollectionScopeHandler) GetCollections(c *gin.Context) {
	var params model.GetDiscountCodeCollectionsQueryParams
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

	response, err := h.service.GetCollections(c, params.ToRequest(), sellerID)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_GET_DISCOUNT_CODE_COLLECTIONS_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		promotionConstants.DISCOUNT_CODE_COLLECTIONS_RETRIEVED_MSG,
		promotionConstants.DISCOUNT_CODE_COLLECTIONS_FIELD,
		response,
	)
}
