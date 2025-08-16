package handlers

import (
	"net/http"
	"strconv"

	"datun.com/be/common"
	"datun.com/be/user/model"
	"datun.com/be/user/service"
	"github.com/gin-gonic/gin"
)

// AddressHandler handles HTTP requests related to addresses
type AddressHandler struct {
	addressService service.AddressService
}

// NewAddressHandler creates a new instance of AddressHandler
func NewAddressHandler(addressService service.AddressService) *AddressHandler {
	return &AddressHandler{
		addressService: addressService,
	}
}

// GetAddresses handles retrieving all addresses for a user
func (h *AddressHandler) GetAddresses(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorWithCode(c, http.StatusUnauthorized, "Authentication required", "AUTH_REQUIRED")
		return
	}

	// Get addresses
	addresses, err := h.addressService.GetAddresses(userID.(uint))
	if err != nil {
		common.ErrorResp(c, http.StatusInternalServerError, "Failed to get addresses: "+err.Error())
		return
	}

	// Transform addresses
	var addressResponses []model.AddressResponse
	for _, address := range addresses {
		addressResponses = append(addressResponses, model.AddressResponse{
			ID:        address.ID,
			Street:    address.Street,
			City:      address.City,
			State:     address.State,
			ZipCode:   address.ZipCode,
			Country:   address.Country,
			IsDefault: address.IsDefault,
		})
	}

	common.SuccessResponse(c, http.StatusOK, "Addresses retrieved successfully", map[string]interface{}{
		"addresses": addressResponses,
	})
}

// AddAddress handles adding a new address
func (h *AddressHandler) AddAddress(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorWithCode(c, http.StatusUnauthorized, "Authentication required", "AUTH_REQUIRED")
		return
	}

	var req model.AddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErrors []common.ValidationError
		validationErrors = append(validationErrors, common.ValidationError{
			Field:   "request",
			Message: err.Error(),
		})
		common.ErrorWithValidation(c, http.StatusBadRequest, "Validation failed", validationErrors, "VALIDATION_ERROR")
		return
	}

	// Add address
	address, err := h.addressService.AddAddress(userID.(uint), req)
	if err != nil {
		common.ErrorResp(c, http.StatusInternalServerError, "Failed to add address: "+err.Error())
		return
	}

	// Create response
	addressResponse := model.AddressResponse{
		ID:        address.ID,
		Street:    address.Street,
		City:      address.City,
		State:     address.State,
		ZipCode:   address.ZipCode,
		Country:   address.Country,
		IsDefault: address.IsDefault,
	}

	common.SuccessResponse(c, http.StatusCreated, "Address added successfully", map[string]interface{}{
		"address": addressResponse,
	})
}

// UpdateAddress handles updating an existing address
func (h *AddressHandler) UpdateAddress(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorWithCode(c, http.StatusUnauthorized, "Authentication required", "AUTH_REQUIRED")
		return
	}

	// Get address ID from path parameter
	addressID, err := getAddressIDParam(c)
	if err != nil {
		common.ErrorWithCode(c, http.StatusBadRequest, "Invalid address ID", "INVALID_ID")
		return
	}

	var req model.AddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErrors []common.ValidationError
		validationErrors = append(validationErrors, common.ValidationError{
			Field:   "request",
			Message: err.Error(),
		})
		common.ErrorWithValidation(c, http.StatusBadRequest, "Validation failed", validationErrors, "VALIDATION_ERROR")
		return
	}

	// Update address
	address, err := h.addressService.UpdateAddress(addressID, userID.(uint), req)
	if err != nil {
		if err.Error() == "address not found" {
			common.ErrorWithCode(c, http.StatusNotFound, err.Error(), "ADDRESS_NOT_FOUND")
			return
		}
		common.ErrorWithCode(c, http.StatusForbidden, "You don't have permission to update this address", "PERMISSION_DENIED")
		return
	}

	// Create response
	addressResponse := model.AddressResponse{
		ID:        address.ID,
		Street:    address.Street,
		City:      address.City,
		State:     address.State,
		ZipCode:   address.ZipCode,
		Country:   address.Country,
		IsDefault: address.IsDefault,
	}

	common.SuccessResponse(c, http.StatusOK, "Address updated successfully", map[string]interface{}{
		"address": addressResponse,
	})
}

// DeleteAddress handles deleting an address
func (h *AddressHandler) DeleteAddress(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorWithCode(c, http.StatusUnauthorized, "Authentication required", "AUTH_REQUIRED")
		return
	}

	// Get address ID from path parameter
	addressID, err := getAddressIDParam(c)
	if err != nil {
		common.ErrorWithCode(c, http.StatusBadRequest, "Invalid address ID", "INVALID_ID")
		return
	}

	// Delete address
	err = h.addressService.DeleteAddress(addressID, userID.(uint))
	if err != nil {
		if err.Error() == "address not found" {
			common.ErrorWithCode(c, http.StatusNotFound, err.Error(), "ADDRESS_NOT_FOUND")
			return
		}
		if err.Error() == "cannot delete the only default address" {
			common.ErrorWithCode(c, http.StatusBadRequest, "Cannot delete default address. Please set another address as default first.", "CANNOT_DELETE_DEFAULT")
			return
		}
		common.ErrorResp(c, http.StatusInternalServerError, "Failed to delete address: "+err.Error())
		return
	}

	common.SuccessResponse(c, http.StatusOK, "Address deleted successfully", nil)
}

// SetDefaultAddress handles setting an address as the default address
func (h *AddressHandler) SetDefaultAddress(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorWithCode(c, http.StatusUnauthorized, "Authentication required", "AUTH_REQUIRED")
		return
	}

	// Get address ID from path parameter
	addressID, err := getAddressIDParam(c)
	if err != nil {
		common.ErrorWithCode(c, http.StatusBadRequest, "Invalid address ID", "INVALID_ID")
		return
	}

	// Set default address
	address, err := h.addressService.SetDefaultAddress(addressID, userID.(uint))
	if err != nil {
		if err.Error() == "address not found" {
			common.ErrorWithCode(c, http.StatusNotFound, err.Error(), "ADDRESS_NOT_FOUND")
			return
		}
		common.ErrorResp(c, http.StatusInternalServerError, "Failed to set default address: "+err.Error())
		return
	}

	// Create response
	addressResponse := model.AddressResponse{
		ID:        address.ID,
		Street:    address.Street,
		City:      address.City,
		State:     address.State,
		ZipCode:   address.ZipCode,
		Country:   address.Country,
		IsDefault: address.IsDefault,
	}

	common.SuccessResponse(c, http.StatusOK, "Default address updated successfully", map[string]interface{}{
		"address": addressResponse,
	})
}

// getAddressIDParam gets an address ID from a path parameter
func getAddressIDParam(c *gin.Context) (uint, error) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}
