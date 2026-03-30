package handler

import (
	"net/http"

	"ecommerce-be/common/handler"
	"ecommerce-be/user/model"
	"ecommerce-be/user/service"
	"ecommerce-be/user/utils/constant"

	"github.com/gin-gonic/gin"
)

// CountryHandler handles HTTP requests related to countries
type CountryHandler struct {
	*handler.BaseHandler
	countryService service.CountryService
}

// NewCountryHandler creates a new instance of CountryHandler
func NewCountryHandler(countryService service.CountryService) *CountryHandler {
	return &CountryHandler{
		BaseHandler:    handler.NewBaseHandler(),
		countryService: countryService,
	}
}

// ========================================
// PUBLIC ROUTES
// ========================================

// ListActiveCountries handles listing all active countries (public API)
// GET /api/user/country
func (h *CountryHandler) ListActiveCountries(c *gin.Context) {
	// Parse query parameters
	var params model.CountryQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Get countries (only active for public API)
	response, err := h.countryService.GetAllCountries(c, params, false)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_LIST_COUNTRIES_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		constant.COUNTRIES_LISTED_MSG,
		constant.COUNTRIES_FIELD_NAME,
		response.Countries,
	)
}

// GetCountryByID handles getting a country by ID (public API)
// GET /api/user/country/:id
func (h *CountryHandler) GetCountryByID(c *gin.Context) {
	countryID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, constant.INVALID_COUNTRY_ID_MSG)
		return
	}

	response, err := h.countryService.GetCountryByID(c, countryID)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_GET_COUNTRY_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		constant.COUNTRY_RETRIEVED_MSG,
		constant.COUNTRY_FIELD_NAME,
		response,
	)
}

// ========================================
// ADMIN ROUTES
// ========================================

// ListAllCountries handles listing all countries including inactive (admin API)
// GET /api/user/admin/country
func (h *CountryHandler) ListAllCountries(c *gin.Context) {
	// Parse query parameters
	var params model.CountryQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Get all countries (including inactive for admin)
	response, err := h.countryService.GetAllCountries(c, params, true)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_LIST_COUNTRIES_MSG)
		return
	}

	// Return with pagination for admin
	h.Success(c, http.StatusOK, constant.COUNTRIES_LISTED_MSG, response)
}

// GetCountryByIDAdmin handles getting a country by ID (admin API)
// GET /api/user/admin/country/:id
func (h *CountryHandler) GetCountryByIDAdmin(c *gin.Context) {
	countryID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, constant.INVALID_COUNTRY_ID_MSG)
		return
	}

	response, err := h.countryService.GetCountryByID(c, countryID)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_GET_COUNTRY_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		constant.COUNTRY_RETRIEVED_MSG,
		constant.COUNTRY_FIELD_NAME,
		response,
	)
}

// TODO: Add Create, Update, Delete handlers when needed
// CreateCountry handles country creation (admin only)
// UpdateCountry handles country updates (admin only)
// DeleteCountry handles country deletion (admin only)

// ========================================
// ADMIN MUTATION ROUTES
// ========================================

// CreateCountry handles creating a new country (admin only)
// POST /api/user/admin/country
func (h *CountryHandler) CreateCountry(c *gin.Context) {
	var req model.CountryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	response, err := h.countryService.CreateCountry(c, req)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_CREATE_COUNTRY_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusCreated,
		constant.COUNTRY_CREATED_MSG,
		constant.COUNTRY_FIELD_NAME,
		response,
	)
}

// UpdateCountry handles updating an existing country (admin only)
// PUT /api/user/admin/country/:id
func (h *CountryHandler) UpdateCountry(c *gin.Context) {
	countryID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, constant.INVALID_COUNTRY_ID_MSG)
		return
	}

	var req model.CountryUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	response, err := h.countryService.UpdateCountry(c, countryID, req)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_UPDATE_COUNTRY_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		constant.COUNTRY_UPDATED_MSG,
		constant.COUNTRY_FIELD_NAME,
		response,
	)
}

// DeleteCountry handles deleting a country (admin only)
// DELETE /api/user/admin/country/:id
func (h *CountryHandler) DeleteCountry(c *gin.Context) {
	countryID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, constant.INVALID_COUNTRY_ID_MSG)
		return
	}

	if err := h.countryService.DeleteCountry(c, countryID); err != nil {
		h.HandleError(c, err, constant.FAILED_TO_DELETE_COUNTRY_MSG)
		return
	}

	h.Success(c, http.StatusOK, constant.COUNTRY_DELETED_MSG, nil)
}
