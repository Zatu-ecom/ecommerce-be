package factory

import (
	"time"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/config"
	"ecommerce-be/common/constants"
	"ecommerce-be/user/entity"
	"ecommerce-be/user/model"
	"ecommerce-be/user/utils"
)

/***********************************************
 *          User Response Builders             *
 ***********************************************/

// BuildUserResponse converts a user entity to a user response model
// Eliminates code duplication in Register, Login, GetProfile, UpdateProfile
func BuildUserResponse(user *entity.User) model.UserResponse {
	return model.UserResponse{
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
}

/***********************************************
 *          Auth Response Builders             *
 ***********************************************/

// BuildAuthResponse creates an auth response with user info and JWT token
// Eliminates code duplication in Register and Login
func BuildAuthResponse(
	user *entity.User,
	role *entity.Role,
	sellerID *uint,
) (*model.AuthResponse, error) {
	// Generate JWT token with role information
	tokenInfo := auth.TokenUserInfo{
		UserID:    user.ID,
		Email:     user.Email,
		RoleID:    user.RoleID,
		RoleName:  role.Name.ToString(),
		RoleLevel: role.Level.ToUint(),
		SellerID:  sellerID,
	}

	token, err := auth.GenerateToken(tokenInfo, config.Get().Auth.JWTSecret)
	if err != nil {
		return nil, err
	}

	// Build user response using factory
	userResponse := BuildUserResponse(user)

	authResponse := &model.AuthResponse{
		User:      userResponse,
		Token:     token,
		ExpiresIn: utils.TokenExpirationDisplay,
	}

	return authResponse, nil
}

// BuildTokenResponse creates a token response for token refresh
// Used by RefreshToken endpoint
func BuildTokenResponse(
	user *entity.User,
	role *entity.Role,
	sellerID *uint,
) (*model.TokenResponse, error) {
	// Generate JWT token with role information
	tokenInfo := auth.TokenUserInfo{
		UserID:    user.ID,
		Email:     user.Email,
		RoleID:    user.RoleID,
		RoleName:  role.Name.ToString(),
		RoleLevel: role.Level.ToUint(),
		SellerID:  sellerID,
	}

	token, err := auth.GenerateToken(tokenInfo, config.Get().Auth.JWTSecret)
	if err != nil {
		return nil, err
	}

	tokenResponse := &model.TokenResponse{
		Token:     token,
		ExpiresIn: utils.TokenExpirationDisplay,
	}

	return tokenResponse, nil
}

/***********************************************
 *          Helper Functions                   *
 ***********************************************/

// ResolveSellerID determines the seller ID based on user and role
// Eliminates code duplication in Login and RefreshToken
// Returns:
// - user.SellerID if user is associated with a seller (non-zero)
// - user.ID if user has SELLER role (seller IS the user)
// - nil otherwise
func ResolveSellerID(user *entity.User, role *entity.Role) *uint {
	// If user is associated with a seller, use that seller ID
	if user.SellerID != 0 {
		return &user.SellerID
	}

	// If user IS a seller, their ID is the seller ID
	if role.Name.ToString() == constants.SELLER_ROLE_NAME {
		return &user.ID
	}

	return nil
}
