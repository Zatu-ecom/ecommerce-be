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

// PromotionHandler handles HTTP requests for promotions
type PromotionHandler struct {
	*commonHandler.BaseHandler
	service service.PromotionService
}

// NewPromotionHandler creates a new instance of PromotionHandler
func NewPromotionHandler(service service.PromotionService) *PromotionHandler {
	return &PromotionHandler{
		BaseHandler: commonHandler.NewBaseHandler(),
		service:     service,
	}
}

// CreatePromotion creates a new promotion
func (h *PromotionHandler) CreatePromotion(c *gin.Context) {
	var req model.CreatePromotionRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	response, err := h.service.CreatePromotion(c, req, sellerID)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_CREATE_PROMOTION_MSG)
		return
	}

	h.SuccessWithData(c, http.StatusCreated, promotionConstants.PROMOTION_CREATED_MSG, promotionConstants.PROMOTION_FIELD, response)
}

// GetPromotion retrieves a promotion by ID
func (h *PromotionHandler) GetPromotion(c *gin.Context) {
	id, err := h.ParseUintParam(c, "promotionId")
	if err != nil {
		h.HandleError(c, err, promotionConstants.INVALID_PROMOTION_ID_MSG)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	response, err := h.service.GetPromotionByID(c, id, sellerID)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_GET_PROMOTION_MSG)
		return
	}

	h.SuccessWithData(c, http.StatusOK, promotionConstants.PROMOTION_RETRIEVED_MSG, promotionConstants.PROMOTION_FIELD, response)
}

// ListPromotions lists promotions with optional filters
func (h *PromotionHandler) ListPromotions(c *gin.Context) {
	var req model.ListPromotionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	// Override seller ID from auth context
	req.SellerID = sellerID

	response, err := h.service.ListPromotions(c, req)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_LIST_PROMOTIONS_MSG)
		return
	}

	h.SuccessWithData(c, http.StatusOK, promotionConstants.PROMOTIONS_LISTED_MSG, promotionConstants.PROMOTIONS_FIELD, response)
}

// UpdatePromotion updates a promotion
func (h *PromotionHandler) UpdatePromotion(c *gin.Context) {
	id, err := h.ParseUintParam(c, "promotionId")
	if err != nil {
		h.HandleError(c, err, promotionConstants.INVALID_PROMOTION_ID_MSG)
		return
	}

	var req model.UpdatePromotionRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	response, err := h.service.UpdatePromotion(c, id, req, sellerID)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_UPDATE_PROMOTION_MSG)
		return
	}

	h.SuccessWithData(c, http.StatusOK, promotionConstants.PROMOTION_UPDATED_MSG, promotionConstants.PROMOTION_FIELD, response)
}

// UpdateStatus updates the status of a promotion
func (h *PromotionHandler) UpdateStatus(c *gin.Context) {
	id, err := h.ParseUintParam(c, "promotionId")
	if err != nil {
		h.HandleError(c, err, promotionConstants.INVALID_PROMOTION_ID_MSG)
		return
	}

	var req model.UpdateStatusRequest
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
		h.HandleError(c, err, promotionConstants.FAILED_TO_UPDATE_PROMOTION_STATUS_MSG)
		return
	}

	h.SuccessWithData(c, http.StatusOK, promotionConstants.PROMOTION_STATUS_UPDATED_MSG, promotionConstants.PROMOTION_FIELD, response)
}

// DeletePromotion soft deletes a promotion
func (h *PromotionHandler) DeletePromotion(c *gin.Context) {
	id, err := h.ParseUintParam(c, "promotionId")
	if err != nil {
		h.HandleError(c, err, promotionConstants.INVALID_PROMOTION_ID_MSG)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.DeletePromotion(c, id, sellerID); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_DELETE_PROMOTION_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.PROMOTION_DELETED_MSG, nil)
}
