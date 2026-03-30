package handler

import (
	"net/http"

	"ecommerce-be/common/handler"
	"ecommerce-be/user/model"
	"ecommerce-be/user/service"
	"ecommerce-be/user/utils/constant"

	"github.com/gin-gonic/gin"
)

// CurrencyHandler handles HTTP requests related to currencies
type CurrencyHandler struct {
	*handler.BaseHandler
	currencyService service.CurrencyService
}

// NewCurrencyHandler creates a new instance of CurrencyHandler
func NewCurrencyHandler(currencyService service.CurrencyService) *CurrencyHandler {
	return &CurrencyHandler{
		BaseHandler:     handler.NewBaseHandler(),
		currencyService: currencyService,
	}
}

// ========================================
// ADMIN ROUTES
// ========================================

// ListAllCurrencies handles listing all currencies including inactive (admin API)
// GET /api/user/admin/currency
func (h *CurrencyHandler) ListAllCurrencies(c *gin.Context) {
	// Parse query parameters
	var params model.CurrencyQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Get all currencies (including inactive for admin)
	response, err := h.currencyService.GetAllCurrencies(c, params, true)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_LIST_CURRENCIES_MSG)
		return
	}

	// Return with pagination for admin
	h.Success(c, http.StatusOK, constant.CURRENCIES_LISTED_MSG, response)
}

// GetCurrencyByIDAdmin handles getting a currency by ID (admin API)
// GET /api/user/admin/currency/:id
func (h *CurrencyHandler) GetCurrencyByIDAdmin(c *gin.Context) {
	currencyID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, "Invalid currency ID")
		return
	}

	response, err := h.currencyService.GetCurrencyByID(c, currencyID)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_GET_CURRENCY_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		constant.CURRENCY_RETRIEVED_MSG,
		constant.CURRENCY_FIELD_NAME,
		response,
	)
}

// ========================================
// ADMIN MUTATION ROUTES
// ========================================

// CreateCurrency handles creating a new currency (admin only)
// POST /api/user/admin/currency
func (h *CurrencyHandler) CreateCurrency(c *gin.Context) {
	var req model.CurrencyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	response, err := h.currencyService.CreateCurrency(c, req)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_CREATE_CURRENCY_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusCreated,
		constant.CURRENCY_CREATED_MSG,
		constant.CURRENCY_FIELD_NAME,
		response,
	)
}

// UpdateCurrency handles updating an existing currency (admin only)
// PUT /api/user/admin/currency/:id
func (h *CurrencyHandler) UpdateCurrency(c *gin.Context) {
	currencyID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, "Invalid currency ID")
		return
	}

	var req model.CurrencyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	response, err := h.currencyService.UpdateCurrency(c, currencyID, req)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_UPDATE_CURRENCY_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		constant.CURRENCY_UPDATED_MSG,
		constant.CURRENCY_FIELD_NAME,
		response,
	)
}

// DeleteCurrency handles deleting a currency (admin only)
// DELETE /api/user/admin/currency/:id
func (h *CurrencyHandler) DeleteCurrency(c *gin.Context) {
	currencyID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, "Invalid currency ID")
		return
	}

	if err := h.currencyService.DeleteCurrency(c, currencyID); err != nil {
		h.HandleError(c, err, constant.FAILED_TO_DELETE_CURRENCY_MSG)
		return
	}

	h.Success(c, http.StatusOK, constant.CURRENCY_DELETED_MSG, nil)
}
