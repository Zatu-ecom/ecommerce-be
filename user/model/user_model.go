package model

// UserRegisterRequest represents the request body for user registration
type UserRegisterRequest struct {
	FirstName       string `json:"firstName"       binding:"required"`
	LastName        string `json:"lastName"        binding:"required"`
	Email           string `json:"email"           binding:"required,email"`
	Password        string `json:"password"        binding:"required,min=6"`
	ConfirmPassword string `json:"confirmPassword" binding:"required"`
	SellerID        uint   `json:"sellerId"`
	Phone           string `json:"phone"`
	DateOfBirth     string `json:"dateOfBirth"`
	Gender          string `json:"gender"`
}

// UserLoginRequest represents the request body for user login
type UserLoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// UserUpdateRequest represents the request body for updating user profile
type UserUpdateRequest struct {
	FirstName   string `json:"firstName"   binding:"required"`
	LastName    string `json:"lastName"    binding:"required"`
	Phone       string `json:"phone"`
	DateOfBirth string `json:"dateOfBirth"`
	Gender      string `json:"gender"`
}

// UserPasswordChangeRequest represents the request body for changing user password
type UserPasswordChangeRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword"     binding:"required,min=6"`
	ConfirmPassword string `json:"confirmPassword" binding:"required"`
}

// UserResponse represents the user data returned in API responses
type UserResponse struct {
	ID          uint   `json:"id"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	DateOfBirth string `json:"dateOfBirth"`
	Gender      string `json:"gender"`
	IsActive    bool   `json:"isActive"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// AuthResponse represents the authentication response with user data and token
type AuthResponse struct {
	User      UserResponse `json:"user"`
	Token     string       `json:"token"`
	ExpiresIn string       `json:"expiresIn"`
}

// TokenResponse represents the token refresh response
type TokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn string `json:"expiresIn"`
}

// Create profile response that includes user data and addresses
type ProfileResponse struct {
	UserResponse
	Addresses []AddressResponse `json:"addresses"`
}
