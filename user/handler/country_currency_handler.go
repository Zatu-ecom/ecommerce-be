package handler

import (
	"net/http"

	"ecommerce-be/common/handler"
	"ecommerce-be/user/model"
	"ecommerce-be/user/service"
	"ecommerce-be/user/utils/constant"

	"github.com/gin-gonic/gin"
)

// CountryCurrencyHandler handles HTTP requests related to country-currency mappings
type CountryCurrencyHandler struct {
	*handler.BaseHandler
	countryCurrencyService service.CountryCurrencyService
}

// NewCountryCurrencyHandler creates a new instance of CountryCurrencyHandler
func NewCountryCurrencyHandler(
	countryCurrencyService service.CountryCurrencyService,
) *CountryCurrencyHandler {
	return &CountryCurrencyHandler{
		BaseHandler:            handler.NewBaseHandler(),
		countryCurrencyService: countryCurrencyService,
	}
}

// ListCountryCurrencies handles listing all currencies for a country
// GET /api/user/admin/country/:id/currency
func (h *CountryCurrencyHandler) ListCountryCurrencies(c *gin.Context) {
	countryID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, constant.INVALID_COUNTRY_ID_MSG)
		return
	}

	response, err := h.countryCurrencyService.GetCurrenciesByCountryID(c, countryID)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_LIST_COUNTRY_CURRENCY_MSG)
		return
	}

	h.Success(c, http.StatusOK, constant.COUNTRY_CURRENCIES_LISTED_MSG, response)
}

// AddCurrencyToCountry handles adding a currency to a country
// POST /api/user/admin/country/:id/currency
func (h *CountryCurrencyHandler) AddCurrencyToCountry(c *gin.Context) {
	countryID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, constant.INVALID_COUNTRY_ID_MSG)
		return
	}

	var req model.CountryCurrencyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	response, err := h.countryCurrencyService.AddCurrencyToCountry(c, countryID, req)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_ADD_COUNTRY_CURRENCY_MSG)
		return
	}

	h.Success(c, http.StatusCreated, constant.COUNTRY_CURRENCY_ADDED_MSG, response)
}

// BulkAddCurrenciesToCountry handles adding multiple currencies to a country
// POST /api/user/admin/country/:id/currency/bulk
func (h *CountryCurrencyHandler) BulkAddCurrenciesToCountry(c *gin.Context) {
	countryID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, constant.INVALID_COUNTRY_ID_MSG)
		return
	}

	var req model.CountryCurrencyBulkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	responses, err := h.countryCurrencyService.BulkAddCurrenciesToCountry(c, countryID, req)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_BULK_ADD_COUNTRY_CURRENCY_MSG)
		return
	}

	h.Success(c, http.StatusCreated, constant.COUNTRY_CURRENCIES_BULK_ADDED_MSG, responses)
}

// UpdateCountryCurrency handles updating a country-currency mapping (e.g., set as primary)
// PUT /api/user/admin/country/:id/currency/:currencyId
func (h *CountryCurrencyHandler) UpdateCountryCurrency(c *gin.Context) {
	countryID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, constant.INVALID_COUNTRY_ID_MSG)
		return
	}

	currencyID, err := h.ParseUintParam(c, "currencyId")
	if err != nil {
		h.HandleError(c, err, constant.INVALID_CURRENCY_ID_MSG)
		return
	}

	var req model.CountryCurrencyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	response, err := h.countryCurrencyService.UpdateCountryCurrency(c, countryID, currencyID, req)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_UPDATE_COUNTRY_CURRENCY_MSG)
		return
	}

	h.Success(c, http.StatusOK, constant.COUNTRY_CURRENCY_UPDATED_MSG, response)
}

// RemoveCurrencyFromCountry handles removing a currency from a country
// DELETE /api/user/admin/country/:id/currency/:currencyId
func (h *CountryCurrencyHandler) RemoveCurrencyFromCountry(c *gin.Context) {
	countryID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, constant.INVALID_COUNTRY_ID_MSG)
		return
	}

	currencyID, err := h.ParseUintParam(c, "currencyId")
	if err != nil {
		h.HandleError(c, err, constant.INVALID_CURRENCY_ID_MSG)
		return
	}

	if err := h.countryCurrencyService.RemoveCurrencyFromCountry(c, countryID, currencyID); err != nil {
		h.HandleError(c, err, constant.FAILED_TO_REMOVE_COUNTRY_CURRENCY_MSG)
		return
	}

	h.Success(c, http.StatusOK, constant.COUNTRY_CURRENCY_REMOVED_MSG, nil)
}
