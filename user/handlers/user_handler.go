package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"ecommerce-be/common"
	"ecommerce-be/user/model"
	"ecommerce-be/user/service"
	"ecommerce-be/user/utils"

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

	authResponse, err := h.userService.Register(req)
	if err != nil {
		if err.Error() == utils.UserExistsMsg {
			common.ErrorWithCode(c, http.StatusConflict, err.Error(), utils.UserExistsCode)
			return
		}
		if err.Error() == utils.PasswordMismatchMsg {
			common.ErrorWithCode(c, http.StatusBadRequest, err.Error(), utils.PasswordMismatchCode)
			return
		}
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FailedToRegisterUserMsg+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusCreated, utils.RegisterSuccessMsg, authResponse)
}

// Login handles user authentication
func (h *UserHandler) Login(c *gin.Context) {
	var req model.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.InvalidRequestFormatMsg,
			utils.ValidationErrorCode,
		)
		return
	}

	authResponse, err := h.userService.Login(req)
	if err != nil {
		if err.Error() == utils.AccountDeactivatedMsg {
			common.ErrorWithCode(c, http.StatusForbidden, err.Error(), utils.AccountDeactivatedCode)
			return
		}
		common.ErrorWithCode(
			c,
			http.StatusUnauthorized,
			utils.InvalidCredentialsMsg,
			utils.InvalidCredentialsCode,
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.LoginSuccessMsg, authResponse)
}

// RefreshToken handles token refresh
func (h *UserHandler) RefreshToken(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(utils.UserIDKey)
	if !exists {
		common.ErrorWithCode(
			c,
			http.StatusUnauthorized,
			utils.TokenInvalidMsg,
			utils.TokenInvalidCode,
		)
		return
	}

	email, exists := c.Get(utils.EmailKey)
	if !exists {
		common.ErrorWithCode(
			c,
			http.StatusUnauthorized,
			utils.TokenInvalidMsg,
			utils.TokenInvalidCode,
		)
		return
	}

	// Generate new token
	tokenResponse, err := h.userService.RefreshToken(userID.(uint), email.(string))
	if err != nil {
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FailedToRefreshTokenMsg+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.TokenRefreshedMsg, tokenResponse)
}

// GetProfile handles retrieving user profile
func (h *UserHandler) GetProfile(c *gin.Context) {
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

	// Get user profile
	profileResponse, err := h.userService.GetProfile(userID.(uint))
	if err != nil {
		if err.Error() == utils.UserNotFoundMsg {
			common.ErrorWithCode(c, http.StatusNotFound, err.Error(), utils.UserNotFoundCode)
			return
		}
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FailedToGetProfileMsg+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.ProfileRetrievedMsg, map[string]interface{}{
		utils.UserFieldName: profileResponse,
	})
}

// UpdateProfile handles updating user profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
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

	var req model.UserUpdateRequest
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

	// Update profile
	userResponse, err := h.userService.UpdateProfile(userID.(uint), req)
	if err != nil {
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FailedToUpdateProfileMsg+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.ProfileUpdatedMsg, map[string]interface{}{
		utils.UserFieldName: userResponse,
	})
}

// ChangePassword handles changing user password
func (h *UserHandler) ChangePassword(c *gin.Context) {
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

	var req model.UserPasswordChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.InvalidRequestFormatMsg,
			utils.ValidationErrorCode,
		)
		return
	}

	// Check if new password and confirm password match
	if req.NewPassword != req.ConfirmPassword {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.PasswordMismatchMsg,
			utils.PasswordMismatchCode,
		)
		return
	}

	// Change password
	if err := h.userService.ChangePassword(userID.(uint), req); err != nil {
		if err.Error() == utils.InvalidCurrentPasswordMsg {
			common.ErrorWithCode(
				c,
				http.StatusBadRequest,
				err.Error(),
				utils.InvalidCurrentPasswordCode,
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

	common.SuccessResponse(c, http.StatusOK, utils.PasswordChangedMsg, nil)
}

// Logout handles user logout
func (h *UserHandler) Logout(c *gin.Context) {
	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.NoTokenProvidedMsg,
			utils.TokenRequiredCode,
		)
		return
	}

	// Check if the header has the Bearer prefix
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.InvalidAuthFormatMsg,
			utils.InvalidAuthFormatCode,
		)
		return
	}

	// Get the token
	tokenString := parts[1]

	// Add token to blacklist in Redis
	// The token will be blacklisted for the same duration as the token's validity
	err := common.BlacklistToken(tokenString, utils.TokenExpireDuration)
	if err != nil {
		fmt.Printf("Warning: Failed to blacklist token: %v\n", err)
		// Continue anyway, as this is not critical
	}

	common.SuccessResponse(c, http.StatusOK, utils.LogoutSuccessMsg, nil)
}

// getUserIDParam gets a user ID from a path parameter
func getUserIDParam(c *gin.Context) (uint, error) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}
