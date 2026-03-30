package handler

import (
	"net/http"
	"strconv"

	"ecommerce-be/common"
	"ecommerce-be/user/model"
	"ecommerce-be/user/service"
	"ecommerce-be/user/utils/constant"

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
	userID, exists := c.Get(constant.USER_ID_KEY)
	if !exists {
		common.ErrorWithCode(
			c,
			http.StatusUnauthorized,
			constant.AUTHENTICATION_REQUIRED_MSG,
			constant.AUTH_REQUIRED_CODE,
		)
		return
	}

	// Get addresses
	addresses, err := h.addressService.GetAddresses(c, userID.(uint))
	if err != nil {
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			constant.FAILED_TO_GET_ADDRESSES_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(
		c,
		http.StatusOK,
		constant.ADDRESSES_RETRIEVED_MSG,
		map[string]interface{}{
			constant.ADDRESSES_FIELD_NAME: addresses,
		},
	)
}

// GetAddressByID handles retrieving a specific address by ID
func (h *AddressHandler) GetAddressByID(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(constant.USER_ID_KEY)
	if !exists {
		common.ErrorWithCode(
			c,
			http.StatusUnauthorized,
			constant.AUTHENTICATION_REQUIRED_MSG,
			constant.AUTH_REQUIRED_CODE,
		)
		return
	}

	// Get address ID from path parameter
	addressID, err := getAddressIDParam(c)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			constant.INVALID_ADDRESS_ID_MSG,
			constant.INVALID_ID_CODE,
		)
		return
	}

	// Get address
	address, err := h.addressService.GetAddressByID(c, addressID, userID.(uint))
	if err != nil {
		if err.Error() == constant.ADDRESS_NOT_FOUND_MSG {
			common.ErrorWithCode(
				c,
				http.StatusNotFound,
				err.Error(),
				constant.ADDRESS_NOT_FOUND_CODE,
			)
			return
		}
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			constant.FAILED_TO_GET_ADDRESSES_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, constant.ADDRESS_RETRIEVED_MSG, map[string]interface{}{
		constant.ADDRESS_FIELD_NAME: address,
	})
}

// AddAddress handles adding a new address
func (h *AddressHandler) AddAddress(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(constant.USER_ID_KEY)
	if !exists {
		common.ErrorWithCode(
			c,
			http.StatusUnauthorized,
			constant.AUTHENTICATION_REQUIRED_MSG,
			constant.AUTH_REQUIRED_CODE,
		)
		return
	}

	var req model.AddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErrors []common.ValidationError
		validationErrors = append(validationErrors, common.ValidationError{
			Field:   constant.REQUEST_FIELD_NAME,
			Message: err.Error(),
		})
		common.ErrorWithValidation(
			c,
			http.StatusBadRequest,
			constant.VALIDATION_FAILED_MSG,
			validationErrors,
			constant.VALIDATION_ERROR_CODE,
		)
		return
	}

	// Add address
	address, err := h.addressService.AddAddress(c, userID.(uint), req)
	if err != nil {
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			constant.FAILED_TO_ADD_ADDRESS_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(
		c,
		http.StatusCreated,
		constant.ADDRESS_CREATED_MSG,
		map[string]interface{}{
			constant.ADDRESS_FIELD_NAME: address,
		},
	)
}

// UpdateAddress handles updating an existing address
func (h *AddressHandler) UpdateAddress(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(constant.USER_ID_KEY)
	if !exists {
		common.ErrorWithCode(
			c,
			http.StatusUnauthorized,
			constant.AUTHENTICATION_REQUIRED_MSG,
			constant.AUTH_REQUIRED_CODE,
		)
		return
	}

	// Get address ID from path parameter
	addressID, err := getAddressIDParam(c)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			constant.INVALID_ADDRESS_ID_MSG,
			constant.INVALID_ID_CODE,
		)
		return
	}

	var req model.AddressUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErrors []common.ValidationError
		validationErrors = append(validationErrors, common.ValidationError{
			Field:   constant.REQUEST_FIELD_NAME,
			Message: err.Error(),
		})
		common.ErrorWithValidation(
			c,
			http.StatusBadRequest,
			constant.VALIDATION_FAILED_MSG,
			validationErrors,
			constant.VALIDATION_ERROR_CODE,
		)
		return
	}

	// Update address
	address, err := h.addressService.UpdateAddress(c, addressID, userID.(uint), req)
	if err != nil {
		if err.Error() == constant.ADDRESS_NOT_FOUND_MSG {
			common.ErrorWithCode(
				c,
				http.StatusNotFound,
				err.Error(),
				constant.ADDRESS_NOT_FOUND_CODE,
			)
			return
		}
		common.ErrorWithCode(
			c,
			http.StatusForbidden,
			constant.PERMISSION_DENIED_MSG,
			constant.PERMISSION_DENIED_CODE,
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, constant.ADDRESS_UPDATED_MSG, map[string]interface{}{
		constant.ADDRESS_FIELD_NAME: address,
	})
}

// DeleteAddress handles deleting an address
func (h *AddressHandler) DeleteAddress(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(constant.USER_ID_KEY)
	if !exists {
		common.ErrorWithCode(
			c,
			http.StatusUnauthorized,
			constant.AUTHENTICATION_REQUIRED_MSG,
			constant.AUTH_REQUIRED_CODE,
		)
		return
	}

	// Get address ID from path parameter
	addressID, err := getAddressIDParam(c)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			constant.INVALID_ADDRESS_ID_MSG,
			constant.INVALID_ID_CODE,
		)
		return
	}

	// Delete address
	err = h.addressService.DeleteAddress(c, addressID, userID.(uint))
	if err != nil {
		if err.Error() == constant.ADDRESS_NOT_FOUND_MSG {
			common.ErrorWithCode(
				c,
				http.StatusNotFound,
				err.Error(),
				constant.ADDRESS_NOT_FOUND_CODE,
			)
			return
		}
		if err.Error() == constant.CANNOT_DELETE_ONLY_DEFAULT_ADDRESS_MSG {
			common.ErrorWithCode(
				c,
				http.StatusBadRequest,
				constant.CANNOT_DELETE_DEFAULT_MSG,
				constant.CANNOT_DELETE_DEFAULT_CODE,
			)
			return
		}
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			constant.FAILED_TO_DELETE_ADDRESS_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, constant.ADDRESS_DELETED_MSG, nil)
}

// SetDefaultAddress handles setting an address as the default address
func (h *AddressHandler) SetDefaultAddress(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(constant.USER_ID_KEY)
	if !exists {
		common.ErrorWithCode(
			c,
			http.StatusUnauthorized,
			constant.AUTHENTICATION_REQUIRED_MSG,
			constant.AUTH_REQUIRED_CODE,
		)
		return
	}

	// Get address ID from path parameter
	addressID, err := getAddressIDParam(c)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			constant.INVALID_ADDRESS_ID_MSG,
			constant.INVALID_ID_CODE,
		)
		return
	}

	// Set default address
	address, err := h.addressService.SetDefaultAddress(c, addressID, userID.(uint))
	if err != nil {
		if err.Error() == constant.ADDRESS_NOT_FOUND_MSG {
			common.ErrorWithCode(
				c,
				http.StatusNotFound,
				err.Error(),
				constant.ADDRESS_NOT_FOUND_CODE,
			)
			return
		}
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			constant.FAILED_TO_SET_DEFAULT_ADDRESS_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(
		c,
		http.StatusOK,
		constant.DEFAULT_ADDRESS_UPDATED_MSG,
		map[string]interface{}{
			constant.ADDRESS_FIELD_NAME: address,
		},
	)
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
