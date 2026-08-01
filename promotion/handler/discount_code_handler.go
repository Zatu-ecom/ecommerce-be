package handler

import (
	"net/http"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	commonHandler "ecommerce-be/common/handler"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/service"
	promotionConstants "ecommerce-be/promotion/utils/constant"

	"github.com/gin-gonic/gin"
)

// DiscountCodeHandler handles HTTP requests for seller discount codes.
type DiscountCodeHandler struct {
	*commonHandler.BaseHandler
	service service.DiscountCodeService
}

// NewDiscountCodeHandler creates a new DiscountCodeHandler.
func NewDiscountCodeHandler(service service.DiscountCodeService) *DiscountCodeHandler {
	return &DiscountCodeHandler{
		BaseHandler: commonHandler.NewBaseHandler(),
		service:     service,
	}
}

func (h *DiscountCodeHandler) CreateDiscountCode(c *gin.Context) {
	var req model.CreateDiscountCodeRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	response, err := h.service.CreateDiscountCode(c, req, sellerID)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_CREATE_DISCOUNT_CODE_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusCreated,
		promotionConstants.DISCOUNT_CODE_CREATED_MSG,
		promotionConstants.DISCOUNT_CODE_FIELD,
		response,
	)
}

func (h *DiscountCodeHandler) UpdateDiscountCode(c *gin.Context) {
	id, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, promotionConstants.INVALID_DISCOUNT_CODE_ID_MSG)
		return
	}

	var req model.UpdateDiscountCodeRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	response, err := h.service.UpdateDiscountCode(c, id, req, sellerID)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_UPDATE_DISCOUNT_CODE_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		promotionConstants.DISCOUNT_CODE_UPDATED_MSG,
		promotionConstants.DISCOUNT_CODE_FIELD,
		response,
	)
}

func (h *DiscountCodeHandler) DeleteDiscountCode(c *gin.Context) {
	id, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, promotionConstants.INVALID_DISCOUNT_CODE_ID_MSG)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.DeleteDiscountCode(c, id, sellerID); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_DELETE_DISCOUNT_CODE_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.DISCOUNT_CODE_DELETED_MSG, nil)
}

func (h *DiscountCodeHandler) GetDiscountCode(c *gin.Context) {
	id, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, promotionConstants.INVALID_DISCOUNT_CODE_ID_MSG)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	response, err := h.service.GetDiscountCodeByID(c, id, sellerID)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_GET_DISCOUNT_CODE_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		promotionConstants.DISCOUNT_CODE_RETRIEVED_MSG,
		promotionConstants.DISCOUNT_CODE_FIELD,
		response,
	)
}

func (h *DiscountCodeHandler) ListDiscountCodes(c *gin.Context) {
	var req model.ListDiscountCodesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}
	req.SellerID = sellerID

	response, err := h.service.ListDiscountCodes(c, req)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_LIST_DISCOUNT_CODES_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.DISCOUNT_CODES_LISTED_MSG, response)
}

func (h *DiscountCodeHandler) UpdateStatus(c *gin.Context) {
	id, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, promotionConstants.INVALID_DISCOUNT_CODE_ID_MSG)
		return
	}

	var req model.UpdateDiscountCodeStatusRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	response, err := h.service.UpdateStatus(c, id, req, sellerID)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_UPDATE_DISCOUNT_CODE_STATUS_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		promotionConstants.DISCOUNT_CODE_STATUS_UPDATED_MSG,
		promotionConstants.DISCOUNT_CODE_FIELD,
		response,
	)
}
