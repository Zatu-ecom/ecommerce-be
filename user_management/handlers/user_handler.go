package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"datun.com/be/common"
	"datun.com/be/user_management/model"
	"datun.com/be/user_management/service"
	"datun.com/be/user_management/utils"
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
		common.ErrorWithValidation(c, http.StatusBadRequest, utils.ValidationFailedMsg, validationErrors, utils.ValidationErrorCode)
		return
	}

	user, token, err := h.userService.Register(req)
	if err != nil {
		if err.Error() == utils.UserExistsMsg {
			common.ErrorWithCode(c, http.StatusConflict, err.Error(), utils.UserExistsCode)
			return
		}
		if err.Error() == utils.PasswordMismatchMsg {
			common.ErrorWithCode(c, http.StatusBadRequest, err.Error(), utils.PasswordMismatchCode)
			return
		}
		common.ErrorResp(c, http.StatusInternalServerError, utils.FailedToRegisterUserMsg+": "+err.Error())
		return
	}

	// Create response
	userResponse := model.UserResponse{
		ID:          user.ID,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Email:       user.Email,
		Phone:       user.Phone,
		DateOfBirth: user.DateOfBirth,
		Gender:      user.Gender,
		IsActive:    user.IsActive,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   user.UpdatedAt.Format(time.RFC3339),
	}

	authResponse := model.AuthResponse{
		User:  userResponse,
		Token: token,
	}

	common.SuccessResponse(c, http.StatusCreated, utils.RegisterSuccessMsg, map[string]interface{}{
		utils.UserFieldName:  authResponse.User,
		utils.TokenFieldName: authResponse.Token,
	})
}

// Login handles user authentication
func (h *UserHandler) Login(c *gin.Context) {
	var req model.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorWithCode(c, http.StatusBadRequest, utils.InvalidRequestFormatMsg, utils.ValidationErrorCode)
		return
	}

	user, token, err := h.userService.Login(req)
	if err != nil {
		if err.Error() == utils.AccountDeactivatedMsg {
			common.ErrorWithCode(c, http.StatusForbidden, err.Error(), utils.AccountDeactivatedCode)
			return
		}
		common.ErrorWithCode(c, http.StatusUnauthorized, utils.InvalidCredentialsMsg, utils.InvalidCredentialsCode)
		return
	}

	// Create response
	userResponse := model.UserResponse{
		ID:          user.ID,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Email:       user.Email,
		Phone:       user.Phone,
		DateOfBirth: user.DateOfBirth,
		Gender:      user.Gender,
		IsActive:    user.IsActive,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   user.UpdatedAt.Format(time.RFC3339),
	}

	common.SuccessResponse(c, http.StatusOK, utils.LoginSuccessMsg, map[string]interface{}{
		utils.UserFieldName:      userResponse,
		utils.TokenFieldName:     token,
		utils.ExpiresInFieldName: utils.TokenExpirationDisplay,
	})
}

// RefreshToken handles token refresh
func (h *UserHandler) RefreshToken(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(utils.UserIDKey)
	if !exists {
		common.ErrorWithCode(c, http.StatusUnauthorized, utils.TokenInvalidMsg, utils.TokenInvalidCode)
		return
	}

	email, exists := c.Get(utils.EmailKey)
	if !exists {
		common.ErrorWithCode(c, http.StatusUnauthorized, utils.TokenInvalidMsg, utils.TokenInvalidCode)
		return
	}

	// Generate new token
	token, err := h.userService.RefreshToken(userID.(uint), email.(string))
	if err != nil {
		common.ErrorResp(c, http.StatusInternalServerError, utils.FailedToRefreshTokenMsg+": "+err.Error())
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.TokenRefreshedMsg, map[string]interface{}{
		utils.TokenFieldName:     token,
		utils.ExpiresInFieldName: utils.TokenExpirationDisplay,
	})
}

// GetProfile handles retrieving user profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(utils.UserIDKey)
	if !exists {
		common.ErrorWithCode(c, http.StatusUnauthorized, utils.AuthenticationRequiredMsg, utils.AuthRequiredCode)
		return
	}

	// Get user profile
	user, err := h.userService.GetProfile(userID.(uint))
	if err != nil {
		if err.Error() == utils.UserNotFoundMsg {
			common.ErrorWithCode(c, http.StatusNotFound, err.Error(), utils.UserNotFoundCode)
			return
		}
		common.ErrorResp(c, http.StatusInternalServerError, utils.FailedToGetProfileMsg+": "+err.Error())
		return
	}

	// Create response
	userResponse := model.UserResponse{
		ID:          user.ID,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Email:       user.Email,
		Phone:       user.Phone,
		DateOfBirth: user.DateOfBirth,
		Gender:      user.Gender,
		IsActive:    user.IsActive,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   user.UpdatedAt.Format(time.RFC3339),
	}

	// Transform addresses
	var addressResponses []model.AddressResponse
	for _, address := range user.Addresses {
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

	common.SuccessResponse(c, http.StatusOK, utils.ProfileRetrievedMsg, map[string]interface{}{
		utils.UserFieldName: map[string]interface{}{
			"id":                     userResponse.ID,
			"firstName":              userResponse.FirstName,
			"lastName":               userResponse.LastName,
			"email":                  userResponse.Email,
			"phone":                  userResponse.Phone,
			"dateOfBirth":            userResponse.DateOfBirth,
			"gender":                 userResponse.Gender,
			"isActive":               userResponse.IsActive,
			"createdAt":              userResponse.CreatedAt,
			"updatedAt":              userResponse.UpdatedAt,
			utils.AddressesFieldName: addressResponses,
		},
	})
}

// UpdateProfile handles updating user profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(utils.UserIDKey)
	if !exists {
		common.ErrorWithCode(c, http.StatusUnauthorized, utils.AuthenticationRequiredMsg, utils.AuthRequiredCode)
		return
	}

	var req model.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErrors []common.ValidationError
		validationErrors = append(validationErrors, common.ValidationError{
			Field:   utils.RequestFieldName,
			Message: err.Error(),
		})
		common.ErrorWithValidation(c, http.StatusBadRequest, utils.ValidationFailedMsg, validationErrors, utils.ValidationErrorCode)
		return
	}

	// Update profile
	user, err := h.userService.UpdateProfile(userID.(uint), req)
	if err != nil {
		common.ErrorResp(c, http.StatusInternalServerError, utils.FailedToUpdateProfileMsg+": "+err.Error())
		return
	}

	// Create response
	userResponse := model.UserResponse{
		ID:          user.ID,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Email:       user.Email,
		Phone:       user.Phone,
		DateOfBirth: user.DateOfBirth,
		Gender:      user.Gender,
		IsActive:    user.IsActive,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   user.UpdatedAt.Format(time.RFC3339),
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
		common.ErrorWithCode(c, http.StatusUnauthorized, utils.AuthenticationRequiredMsg, utils.AuthRequiredCode)
		return
	}

	var req model.UserPasswordChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorWithCode(c, http.StatusBadRequest, utils.InvalidRequestFormatMsg, utils.ValidationErrorCode)
		return
	}

	// Check if new password and confirm password match
	if req.NewPassword != req.ConfirmPassword {
		common.ErrorWithCode(c, http.StatusBadRequest, utils.PasswordMismatchMsg, utils.PasswordMismatchCode)
		return
	}

	// Change password
	if err := h.userService.ChangePassword(userID.(uint), req); err != nil {
		if err.Error() == utils.InvalidCurrentPasswordMsg {
			common.ErrorWithCode(c, http.StatusBadRequest, err.Error(), utils.InvalidCurrentPasswordCode)
			return
		}
		common.ErrorResp(c, http.StatusInternalServerError, "Failed to change password: "+err.Error())
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.PasswordChangedMsg, nil)
}

// Logout handles user logout
func (h *UserHandler) Logout(c *gin.Context) {
	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		common.ErrorWithCode(c, http.StatusBadRequest, utils.NoTokenProvidedMsg, utils.TokenRequiredCode)
		return
	}

	// Check if the header has the Bearer prefix
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		common.ErrorWithCode(c, http.StatusBadRequest, utils.InvalidAuthFormatMsg, utils.InvalidAuthFormatCode)
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
