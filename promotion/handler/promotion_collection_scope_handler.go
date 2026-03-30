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

type PromotionCollectionScopeHandler struct {
	*commonHandler.BaseHandler
	service service.PromotionCollectionScopeService
}

func NewPromotionCollectionScopeHandler(
	service service.PromotionCollectionScopeService,
) *PromotionCollectionScopeHandler {
	return &PromotionCollectionScopeHandler{
		BaseHandler: commonHandler.NewBaseHandler(),
		service:     service,
	}
}

// AddCollections adds collections to a promotion
func (h *PromotionCollectionScopeHandler) AddCollections(c *gin.Context) {
	var req model.AddPromotionCollectionRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, _, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.AddCollections(c, req); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_ADD_PROMOTION_COLLECTIONS_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.PROMOTION_COLLECTIONS_ADDED_MSG, nil)
}

// RemoveCollections removes collections from a promotion
func (h *PromotionCollectionScopeHandler) RemoveCollections(c *gin.Context) {
	var req model.RemovePromotionCollectionRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, _, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.RemoveCollections(c, req); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_REMOVE_PROMOTION_COLLECTIONS_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.PROMOTION_COLLECTIONS_REMOVED_MSG, nil)
}

// RemoveAllCollections removes all collections from a promotion
func (h *PromotionCollectionScopeHandler) RemoveAllCollections(c *gin.Context) {
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

	if err := h.service.RemoveAllCollections(c, uint(promotionID)); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_REMOVE_ALL_PROMOTION_COLLECTIONS_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.PROMOTION_ALL_COLLECTIONS_REMOVED_MSG, nil)
}

// GetCollections retrieves collections for a promotion
func (h *PromotionCollectionScopeHandler) GetCollections(c *gin.Context) {
	var params model.GetPromotionCollectionsQueryParams
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

	response, err := h.service.GetCollections(c, req)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_GET_PROMOTION_COLLECTIONS_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		promotionConstants.PROMOTION_COLLECTIONS_RETRIEVED_MSG,
		promotionConstants.PROMOTION_COLLECTIONS_FIELD,
		response,
	)
}
