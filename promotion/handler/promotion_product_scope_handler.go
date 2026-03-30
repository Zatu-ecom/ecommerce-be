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

type PromotionProductScopeHandler struct {
	*commonHandler.BaseHandler
	service service.PromotionProductScopeService
}

func NewPromotionProductScopeHandler(
	service service.PromotionProductScopeService,
) *PromotionProductScopeHandler {
	return &PromotionProductScopeHandler{
		BaseHandler: commonHandler.NewBaseHandler(),
		service:     service,
	}
}

// AddProducts adds products to a promotion
func (h *PromotionProductScopeHandler) AddProducts(c *gin.Context) {
	var req model.AddPromotionProductRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Validate seller access if needed (assuming promotion management is restricted)
	_, _, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.AddProducts(c, req); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_ADD_PROMOTION_PRODUCTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.PROMOTION_PRODUCTS_ADDED_MSG, nil)
}

// RemoveProducts removes products from a promotion
func (h *PromotionProductScopeHandler) RemoveProducts(c *gin.Context) {
	var req model.RemovePromotionProductRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, _, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.RemoveProducts(c, req); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_REMOVE_PROMOTION_PRODUCTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.PROMOTION_PRODUCTS_REMOVED_MSG, nil)
}

// RemoveAllProducts removes all products from a promotion
func (h *PromotionProductScopeHandler) RemoveAllProducts(c *gin.Context) {
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

	if err := h.service.RemoveAllProducts(c, uint(promotionID)); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_REMOVE_ALL_PROMOTION_PRODUCTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.PROMOTION_ALL_PRODUCTS_REMOVED_MSG, nil)
}

// GetProducts retrieves products for a promotion
func (h *PromotionProductScopeHandler) GetProducts(c *gin.Context) {
	var params model.GetPromotionProductsQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	req := params.ToRequest()

	// Bind promotionID from path if not in query (usually it's path param for REST)
	// But typical Get request might use query params for filters.
	// Let's assume promotionID is passed via query or path.
	// If path param exists, override it.
	promotionIDStr := c.Param("promotionId")
	if promotionIDStr != "" {
		id, err := strconv.ParseUint(promotionIDStr, 10, 64)
		if err == nil {
			req.PromotionID = uint(id)
		}
	}

	response, err := h.service.GetProducts(c, req)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_GET_PROMOTION_PRODUCTS_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		promotionConstants.PROMOTION_PRODUCTS_RETRIEVED_MSG,
		promotionConstants.PROMOTION_PRODUCTS_FIELD,
		response,
	)
}
