package handler

import (
	"net/http"
	"strconv"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	"ecommerce-be/common/handler"
	"ecommerce-be/common/validator"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/service"
	invConstants "ecommerce-be/inventory/utils/constant"

	"github.com/gin-gonic/gin"
)

// LocationHandler handles HTTP requests related to locations
type LocationHandler struct {
	*handler.BaseHandler
	locationService service.LocationService
}

// NewLocationHandler creates a new instance of LocationHandler
func NewLocationHandler(locationService service.LocationService) *LocationHandler {
	return &LocationHandler{
		BaseHandler:     handler.NewBaseHandler(),
		locationService: locationService,
	}
}

// CreateLocation handles location creation
func (h *LocationHandler) CreateLocation(c *gin.Context) {
	var req model.LocationCreateRequest

	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Get seller ID from authenticated user
	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	locationResponse, err := h.locationService.CreateLocation(c, req, sellerID)
	if err != nil {
		h.HandleError(c, err, invConstants.FAILED_TO_CREATE_LOCATION_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusCreated,
		invConstants.LOCATION_CREATED_MSG,
		invConstants.LOCATION_FIELD_NAME,
		locationResponse,
	)
}

// UpdateLocation handles location updates
func (h *LocationHandler) UpdateLocation(c *gin.Context) {
	locationID, err := h.ParseUintParam(c, "locationId")
	if err != nil {
		h.HandleError(c, err, "Invalid location ID")
		return
	}

	var req model.LocationUpdateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	err = validator.RequireAtLeastOneNonNilPointer(req)
	if err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Get seller ID from authenticated user
	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	locationResponse, err := h.locationService.UpdateLocation(c, locationID, req, sellerID)
	if err != nil {
		h.HandleError(c, err, invConstants.FAILED_TO_UPDATE_LOCATION_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		invConstants.LOCATION_UPDATED_MSG,
		invConstants.LOCATION_FIELD_NAME,
		locationResponse,
	)
}

// GetLocationByID handles getting a location by ID
func (h *LocationHandler) GetLocationByID(c *gin.Context) {
	locationID, err := h.ParseUintParam(c, "locationId")
	if err != nil {
		h.HandleError(c, err, "Invalid location ID")
		return
	}

	// Get seller ID from authenticated user
	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	locationResponse, err := h.locationService.GetLocationByID(c, locationID, sellerID)
	if err != nil {
		h.HandleError(c, err, invConstants.FAILED_TO_GET_LOCATION_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		invConstants.LOCATION_RETRIEVED_MSG,
		invConstants.LOCATION_FIELD_NAME,
		locationResponse,
	)
}

// GetAllLocations handles getting all locations
func (h *LocationHandler) GetAllLocations(c *gin.Context) {
	// Get seller ID from authenticated user
	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	// Parse optional isActive query parameter
	var isActive *bool
	if isActiveStr := c.Query("isActive"); isActiveStr != "" {
		if activeVal, err := strconv.ParseBool(isActiveStr); err == nil {
			isActive = &activeVal
		}
	}

	locationsResponse, err := h.locationService.GetAllLocations(c, sellerID, isActive)
	if err != nil {
		h.HandleError(c, err, invConstants.FAILED_TO_GET_LOCATIONS_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		invConstants.LOCATIONS_RETRIEVED_MSG,
		invConstants.LOCATIONS_FIELD_NAME,
		locationsResponse,
	)
}

// DeleteLocation handles location deletion
func (h *LocationHandler) DeleteLocation(c *gin.Context) {
	locationID, err := h.ParseUintParam(c, "locationId")
	if err != nil {
		h.HandleError(c, err, "Invalid location ID")
		return
	}

	// Get seller ID from authenticated user
	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	err = h.locationService.DeleteLocation(c, locationID, sellerID)
	if err != nil {
		h.HandleError(c, err, invConstants.FAILED_TO_DELETE_LOCATION_MSG)
		return
	}

	h.Success(c, http.StatusOK, invConstants.LOCATION_DELETED_MSG, nil)
}
