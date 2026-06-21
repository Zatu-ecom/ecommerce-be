package handler

import (
	"net/http"

	"ecommerce-be/common/auth"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/handler"
	"ecommerce-be/user/model"
	"ecommerce-be/user/service"
	"ecommerce-be/user/utils/constant"

	"github.com/gin-gonic/gin"
)

// SellerSettingsHandler handles HTTP requests for seller settings management.
type SellerSettingsHandler struct {
	*handler.BaseHandler
	sellerSettingsService service.SellerSettingsService
}

// NewSellerSettingsHandler creates a new SellerSettingsHandler.
func NewSellerSettingsHandler(
	sellerSettingsService service.SellerSettingsService,
) *SellerSettingsHandler {
	return &SellerSettingsHandler{
		BaseHandler:           handler.NewBaseHandler(),
		sellerSettingsService: sellerSettingsService,
	}
}

// GetSellerSettings handles GET /api/user/seller/settings
func (h *SellerSettingsHandler) GetSellerSettings(c *gin.Context) {
	sellerID, ok := h.sellerIDFromContext(c)
	if !ok {
		return
	}

	response, err := h.sellerSettingsService.GetBySellerID(c, sellerID)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_GET_SELLER_SETTINGS_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		constant.SELLER_SETTINGS_RETRIEVED_MSG,
		constant.SELLER_SETTINGS_FIELD_NAME,
		response,
	)
}

// CreateSellerSettings handles POST /api/user/seller/settings
func (h *SellerSettingsHandler) CreateSellerSettings(c *gin.Context) {
	sellerID, ok := h.sellerIDFromContext(c)
	if !ok {
		return
	}

	var req model.SellerSettingsCreateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	response, err := h.sellerSettingsService.Create(c, sellerID, &req)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_CREATE_SELLER_SETTINGS_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusCreated,
		constant.SELLER_SETTINGS_CREATED_MSG,
		constant.SELLER_SETTINGS_FIELD_NAME,
		response,
	)
}

// UpdateSellerSettings handles PUT /api/user/seller/settings
func (h *SellerSettingsHandler) UpdateSellerSettings(c *gin.Context) {
	sellerID, ok := h.sellerIDFromContext(c)
	if !ok {
		return
	}

	var req model.SellerSettingsUpdateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	response, err := h.sellerSettingsService.Update(c, sellerID, req)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_UPDATE_SELLER_SETTINGS_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		constant.SELLER_SETTINGS_UPDATED_MSG,
		constant.SELLER_SETTINGS_FIELD_NAME,
		response,
	)
}

func (h *SellerSettingsHandler) sellerIDFromContext(c *gin.Context) (uint, bool) {
	sellerID, exists := auth.GetSellerIDFromContext(c)
	if !exists || sellerID == 0 {
		h.HandleError(c, commonError.UnauthorizedError, constant.FAILED_TO_GET_SELLER_SETTINGS_MSG)
		return 0, false
	}
	return sellerID, true
}
