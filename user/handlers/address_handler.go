package handlers

import (
	"net/http"
	"strconv"

	"ecommerce-be/common"
	"ecommerce-be/user/model"
	"ecommerce-be/user/service"
	"ecommerce-be/user/utils"

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
	userID, exists := c.Get(utils.UserIDKey)
	if !exists {
		common.ErrorWithCode(
			c,
			http.StatusUnauthorized,
			utils.AuthenticationRequiredMsg,
			utils.AuthRequiredCode,
		)
		return
	}

	// Get addresses
	addresses, err := h.addressService.GetAddresses(userID.(uint))
	if err != nil {
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FailedToGetAddressesMsg+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.AddressesRetrievedMsg, map[string]interface{}{
		utils.AddressesFieldName: addresses,
	})
}

// GetAddressByID handles retrieving a specific address by ID
func (h *AddressHandler) GetAddressByID(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(utils.UserIDKey)
	if !exists {
		common.ErrorWithCode(
			c,
			http.StatusUnauthorized,
			utils.AuthenticationRequiredMsg,
			utils.AuthRequiredCode,
		)
		return
	}

	// Get address ID from path parameter
	addressID, err := getAddressIDParam(c)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.InvalidAddressIDMsg,
			utils.InvalidIDCode,
		)
		return
	}

	// Get address
	address, err := h.addressService.GetAddressByID(addressID, userID.(uint))
	if err != nil {
		if err.Error() == utils.AddressNotFoundMsg {
			common.ErrorWithCode(c, http.StatusNotFound, err.Error(), utils.AddressNotFoundCode)
			return
		}
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FailedToGetAddressesMsg+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.AddressRetrievedMsg, map[string]interface{}{
		utils.AddressFieldName: address,
	})
}

// AddAddress handles adding a new address
func (h *AddressHandler) AddAddress(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(utils.UserIDKey)
	if !exists {
		common.ErrorWithCode(
			c,
			http.StatusUnauthorized,
			utils.AuthenticationRequiredMsg,
			utils.AuthRequiredCode,
		)
		return
	}

	var req model.AddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErrors []common.ValidationError
		validationErrors = append(validationErrors, common.ValidationError{
			Field:   utils.RequestFieldName,
			Message: err.Error(),
		})
		common.ErrorWithValidation(
			c,
			http.StatusBadRequest,
			utils.ValidationFailedMsg,
			validationErrors,
			utils.ValidationErrorCode,
		)
		return
	}

	// Add address
	address, err := h.addressService.AddAddress(userID.(uint), req)
	if err != nil {
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FailedToAddAddressMsg+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusCreated, utils.AddressCreatedMsg, map[string]interface{}{
		utils.AddressFieldName: address,
	})
}

// UpdateAddress handles updating an existing address
func (h *AddressHandler) UpdateAddress(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(utils.UserIDKey)
	if !exists {
		common.ErrorWithCode(
			c,
			http.StatusUnauthorized,
			utils.AuthenticationRequiredMsg,
			utils.AuthRequiredCode,
		)
		return
	}

	// Get address ID from path parameter
	addressID, err := getAddressIDParam(c)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.InvalidAddressIDMsg,
			utils.InvalidIDCode,
		)
		return
	}

	var req model.AddressUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErrors []common.ValidationError
		validationErrors = append(validationErrors, common.ValidationError{
			Field:   utils.RequestFieldName,
			Message: err.Error(),
		})
		common.ErrorWithValidation(
			c,
			http.StatusBadRequest,
			utils.ValidationFailedMsg,
			validationErrors,
			utils.ValidationErrorCode,
		)
		return
	}

	// Update address
	address, err := h.addressService.UpdateAddress(addressID, userID.(uint), req)
	if err != nil {
		if err.Error() == utils.AddressNotFoundMsg {
			common.ErrorWithCode(c, http.StatusNotFound, err.Error(), utils.AddressNotFoundCode)
			return
		}
		common.ErrorWithCode(
			c,
			http.StatusForbidden,
			utils.PermissionDeniedMsg,
			utils.PermissionDeniedCode,
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.AddressUpdatedMsg, map[string]interface{}{
		utils.AddressFieldName: address,
	})
}

// DeleteAddress handles deleting an address
func (h *AddressHandler) DeleteAddress(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(utils.UserIDKey)
	if !exists {
		common.ErrorWithCode(
			c,
			http.StatusUnauthorized,
			utils.AuthenticationRequiredMsg,
			utils.AuthRequiredCode,
		)
		return
	}

	// Get address ID from path parameter
	addressID, err := getAddressIDParam(c)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.InvalidAddressIDMsg,
			utils.InvalidIDCode,
		)
		return
	}

	// Delete address
	err = h.addressService.DeleteAddress(addressID, userID.(uint))
	if err != nil {
		if err.Error() == utils.AddressNotFoundMsg {
			common.ErrorWithCode(c, http.StatusNotFound, err.Error(), utils.AddressNotFoundCode)
			return
		}
		if err.Error() == utils.CannotDeleteOnlyDefaultAddressMsg {
			common.ErrorWithCode(
				c,
				http.StatusBadRequest,
				utils.CannotDeleteDefaultMsg,
				utils.CannotDeleteDefaultCode,
			)
			return
		}
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FailedToDeleteAddressMsg+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.AddressDeletedMsg, nil)
}

// SetDefaultAddress handles setting an address as the default address
func (h *AddressHandler) SetDefaultAddress(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(utils.UserIDKey)
	if !exists {
		common.ErrorWithCode(
			c,
			http.StatusUnauthorized,
			utils.AuthenticationRequiredMsg,
			utils.AuthRequiredCode,
		)
		return
	}

	// Get address ID from path parameter
	addressID, err := getAddressIDParam(c)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.InvalidAddressIDMsg,
			utils.InvalidIDCode,
		)
		return
	}

	// Set default address
	address, err := h.addressService.SetDefaultAddress(addressID, userID.(uint))
	if err != nil {
		if err.Error() == utils.AddressNotFoundMsg {
			common.ErrorWithCode(c, http.StatusNotFound, err.Error(), utils.AddressNotFoundCode)
			return
		}
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FailedToSetDefaultAddressMsg+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.DefaultAddressUpdatedMsg, map[string]interface{}{
		utils.AddressFieldName: address,
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
