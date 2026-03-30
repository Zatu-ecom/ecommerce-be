package handler

import (
	"fmt"
	"net/http"
	"strings"

	"ecommerce-be/common"
	"ecommerce-be/common/cache"
	"ecommerce-be/user/model"
	"ecommerce-be/user/service"
	"ecommerce-be/user/utils/constant"

	"github.com/gin-gonic/gin"
)

// UserHandler handles HTTP requests related to users
type UserHandler struct {
	userService service.UserService
}

// NewUserHandler creates a new instance of UserHandler
func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// Register handles user registration
func (h *UserHandler) Register(c *gin.Context) {
	var req model.UserRegisterRequest
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

	authResponse, err := h.userService.Register(c, req)
	if err != nil {
		if err.Error() == constant.USER_EXISTS_MSG {
			common.ErrorWithCode(c, http.StatusConflict, err.Error(), constant.USER_EXISTS_CODE)
			return
		}
		if err.Error() == constant.PASSWORD_MISMATCH_MSG {
			common.ErrorWithCode(
				c,
				http.StatusBadRequest,
				err.Error(),
				constant.PASSWORD_MISMATCH_CODE,
			)
			return
		}
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			constant.FAILED_TO_REGISTER_USER_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusCreated, constant.REGISTER_SUCCESS_MSG, authResponse)
}

// Login handles user authentication
func (h *UserHandler) Login(c *gin.Context) {
	var req model.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			constant.INVALID_REQUEST_FORMAT_MSG,
			constant.VALIDATION_ERROR_CODE,
		)
		return
	}

	authResponse, err := h.userService.Login(c, req)
	if err != nil {
		if err.Error() == constant.ACCOUNT_DEACTIVATED_MSG {
			common.ErrorWithCode(
				c,
				http.StatusForbidden,
				err.Error(),
				constant.ACCOUNT_DEACTIVATED_CODE,
			)
			return
		}
		common.ErrorWithCode(
			c,
			http.StatusUnauthorized,
			constant.INVALID_CREDENTIALS_MSG,
			constant.INVALID_CREDENTIALS_CODE,
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, constant.LOGIN_SUCCESS_MSG, authResponse)
}

// RefreshToken handles token refresh
func (h *UserHandler) RefreshToken(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(constant.USER_ID_KEY)
	if !exists {
		common.ErrorWithCode(
			c,
			http.StatusUnauthorized,
			constant.TOKEN_INVALID_MSG,
			constant.TOKEN_INVALID_CODE,
		)
		return
	}

	email, exists := c.Get(constant.EMAIL_KEY)
	if !exists {
		common.ErrorWithCode(
			c,
			http.StatusUnauthorized,
			constant.TOKEN_INVALID_MSG,
			constant.TOKEN_INVALID_CODE,
		)
		return
	}

	// Generate new token
	tokenResponse, err := h.userService.RefreshToken(
		c,
		userID.(uint),
		email.(string),
	)
	if err != nil {
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			constant.FAILED_TO_REFRESH_TOKEN_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, constant.TOKEN_REFRESHED_MSG, tokenResponse)
}

// GetProfile handles retrieving user profile
func (h *UserHandler) GetProfile(c *gin.Context) {
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

	// Get user profile
	profileResponse, err := h.userService.GetProfile(c, userID.(uint))
	if err != nil {
		if err.Error() == constant.USER_NOT_FOUND_MSG {
			common.ErrorWithCode(c, http.StatusNotFound, err.Error(), constant.USER_NOT_FOUND_CODE)
			return
		}
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			constant.FAILED_TO_GET_PROFILE_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, constant.PROFILE_RETRIEVED_MSG,
		map[string]interface{}{
			constant.USER_FIELD_NAME: profileResponse,
		})
}

// UpdateProfile handles updating user profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
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

	var req model.UserUpdateRequest
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

	// Update profile
	userResponse, err := h.userService.UpdateProfile(c, userID.(uint), req)
	if err != nil {
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			constant.FAILED_TO_UPDATE_PROFILE_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, constant.PROFILE_UPDATED_MSG,
		map[string]interface{}{
			constant.USER_FIELD_NAME: userResponse,
		})
}

// ChangePassword handles changing user password
func (h *UserHandler) ChangePassword(c *gin.Context) {
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

	var req model.UserPasswordChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			constant.INVALID_REQUEST_FORMAT_MSG,
			constant.VALIDATION_ERROR_CODE,
		)
		return
	}

	// Check if new password and confirm password match
	if req.NewPassword != req.ConfirmPassword {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			constant.PASSWORD_MISMATCH_MSG,
			constant.PASSWORD_MISMATCH_CODE,
		)
		return
	}

	// Change password
	if err := h.userService.ChangePassword(c, userID.(uint), req); err != nil {
		if err.Error() == constant.INVALID_CURRENT_PASSWORD_MSG {
			common.ErrorWithCode(
				c,
				http.StatusBadRequest,
				err.Error(),
				constant.INVALID_CURRENT_PASSWORD_CODE,
			)
			return
		}
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			"Failed to change password: "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, constant.PASSWORD_CHANGED_MSG, nil)
}

// Logout handles user logout
func (h *UserHandler) Logout(c *gin.Context) {
	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			constant.NO_TOKEN_PROVIDED_MSG,
			constant.TOKEN_REQUIRED_CODE,
		)
		return
	}

	// Check if the header has the Bearer prefix
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			constant.INVALID_AUTH_FORMAT_MSG,
			constant.INVALID_AUTH_FORMAT_CODE,
		)
		return
	}

	// Get the token
	tokenString := parts[1]

	// Add token to blacklist in Redis
	// The token will be blacklisted for the same duration as the token's validity
	err := cache.BlacklistToken(tokenString, constant.TOKEN_EXPIRE_DURATION)
	if err != nil {
		fmt.Printf("Warning: Failed to blacklist token: %v\n", err)
		// Continue anyway, as this is not critical
	}

	common.SuccessResponse(c, http.StatusOK, constant.LOGOUT_SUCCESS_MSG, nil)
}
