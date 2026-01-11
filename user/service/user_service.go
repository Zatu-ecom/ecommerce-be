package service

import (
	"context"
	"errors"
	"log"
	"time"

	"ecommerce-be/common/cache"
	"ecommerce-be/common/constants"
	commonEntity "ecommerce-be/common/db"
	"ecommerce-be/user/entity"
	userErrors "ecommerce-be/user/error"
	"ecommerce-be/user/factory"
	"ecommerce-be/user/model"
	"ecommerce-be/user/repository"
	"ecommerce-be/user/utils/constant"

	"golang.org/x/crypto/bcrypt"
)

// UserService defines the interface for user-related business logic
type UserService interface {
	Register(ctx context.Context, req model.UserRegisterRequest) (*model.AuthResponse, error)
	Login(ctx context.Context, req model.UserLoginRequest) (*model.AuthResponse, error)
	GetProfile(ctx context.Context, userID uint) (*model.ProfileResponse, error)
	UpdateProfile(
		ctx context.Context,
		userID uint,
		req model.UserUpdateRequest,
	) (*model.UserResponse, error)
	ChangePassword(ctx context.Context, userID uint, req model.UserPasswordChangeRequest) error
	RefreshToken(ctx context.Context, userID uint, email string) (*model.TokenResponse, error)
	// CreateUserWithRole creates a user with a specific role (for internal service use)
	// Used by SellerRegistrationService to create seller users
	CreateUserWithRole(
		ctx context.Context,
		req model.CreateUserRequest,
		roleName string,
	) (*entity.User, *entity.Role, error) // GetUserByID retrieves a user by ID (for internal service use)
}

// UserServiceImpl implements the UserService interface
type UserServiceImpl struct {
	userRepo       repository.UserRepository
	addressService AddressService
}

// NewUserService creates a new instance of UserService
func NewUserService(
	userRepo repository.UserRepository,
	addressService AddressService,
) UserService {
	return &UserServiceImpl{
		userRepo:       userRepo,
		addressService: addressService,
	}
}

// Register creates a new user account (customer registration)
func (s *UserServiceImpl) Register(
	ctx context.Context,
	req model.UserRegisterRequest,
) (*model.AuthResponse, error) {
	// Validate password confirmation
	if req.Password != req.ConfirmPassword {
		return nil, userErrors.ErrPasswordMismatch
	}

	// Use CreateUserWithRole to create customer (reuses validation, hashing logic)
	createUserReq := model.CreateUserRequest{
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		Password:    req.Password,
		Phone:       req.Phone,
		DateOfBirth: req.DateOfBirth,
		Gender:      req.Gender,
		SellerID:    req.SellerID,
	}

	user, customerRole, err := s.CreateUserWithRole(
		ctx,
		createUserReq,
		constants.CUSTOMER_ROLE_NAME,
	)
	if err != nil {
		return nil, err
	}

	// Build auth response using factory (eliminates duplication)
	return factory.BuildAuthResponse(user, customerRole, &user.SellerID)
}

// Login authenticates a user and returns a token
func (s *UserServiceImpl) Login(
	ctx context.Context,
	req model.UserLoginRequest,
) (*model.AuthResponse, error) {
	// Find user by email with role information
	user, role, err := s.userRepo.FindByEmailWithRole(ctx, req.Email)
	if err != nil {
		return nil, errors.New(constant.INVALID_CREDENTIALS_MSG)
	}

	// Check if account is active
	if !user.IsActive {
		return nil, errors.New(constant.ACCOUNT_DEACTIVATED_MSG)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New(constant.INVALID_CREDENTIALS_MSG)
	}

	// Resolve seller ID using factory helper (eliminates duplication)
	sellerID := factory.ResolveSellerID(user, role)

	// Build auth response using factory (eliminates duplication)
	return factory.BuildAuthResponse(user, role, sellerID)
}

// GetProfile retrieves user profile information including addresses
func (s *UserServiceImpl) GetProfile(
	ctx context.Context,
	userID uint,
) (*model.ProfileResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, errors.New(constant.USER_NOT_FOUND_MSG)
	}

	// Create user response
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
		// Preferences (Note: User's country is derived from default address)
		CurrencyID: user.CurrencyID,
		Locale:     user.Locale,
	}

	addresses, err := s.addressService.GetAddresses(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Build addresses response - already in response format from service
	addressesResList := addresses

	profileResponse := &model.ProfileResponse{
		UserResponse: userResponse,
		Addresses:    addressesResList,
	}

	return profileResponse, nil
}

// UpdateProfile updates user profile information
func (s *UserServiceImpl) UpdateProfile(
	ctx context.Context,
	userID uint,
	req model.UserUpdateRequest,
) (*model.UserResponse, error) {
	// Find user by ID
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update user fields only if provided (pointer is not nil)
	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.Phone != nil {
		user.Phone = *req.Phone
	}
	if req.DateOfBirth != nil {
		user.DateOfBirth = *req.DateOfBirth
	}
	if req.Gender != nil {
		user.Gender = *req.Gender
	}

	// Update preferences if provided (Note: Country is derived from default address)
	if req.CurrencyID != nil {
		user.CurrencyID = req.CurrencyID
	}
	if req.Locale != nil {
		user.Locale = *req.Locale
	}

	user.UpdatedAt = time.Now()

	// Save changes to database
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	// Invalidate seller cache if user is associated with a seller
	if user.SellerID != 0 {
		if err := cache.InvalidateSellerDetailsCache(user.SellerID); err != nil {
			// Log the error but don't fail the request
			log.Printf(
				"Failed to invalidate seller details cache for seller %d: %v",
				user.SellerID,
				err,
			)
		}
	}

	// Build user response using factory (eliminates duplication)
	userResponse := factory.BuildUserResponse(user)

	return &userResponse, nil
}

// ChangePassword updates a user's password
func (s *UserServiceImpl) ChangePassword(
	ctx context.Context,
	userID uint,
	req model.UserPasswordChangeRequest,
) error {
	// Check if new password matches confirmation
	if req.NewPassword != req.ConfirmPassword {
		return errors.New(constant.PASSWORD_MISMATCH_MSG)
	}

	// Find user by ID
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		return errors.New(constant.INVALID_CURRENT_PASSWORD_MSG)
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update password
	user.Password = string(hashedPassword)
	user.UpdatedAt = time.Now()

	// Save changes to database
	return s.userRepo.Update(ctx, user)
}

// RefreshToken generates a new JWT token
func (s *UserServiceImpl) RefreshToken(
	ctx context.Context,
	userID uint,
	email string,
) (*model.TokenResponse, error) {
	// Get user with role information
	user, role, err := s.userRepo.FindByIDWithRole(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Resolve seller ID using factory helper (eliminates duplication)
	sellerID := factory.ResolveSellerID(user, role)

	// Build token response using factory (eliminates duplication)
	return factory.BuildTokenResponse(user, role, sellerID)
}

// CreateUserWithRole creates a user with a specific role
// This is used internally by other services (e.g., SellerRegistrationService)
// It handles: email validation, password hashing, role assignment
// The caller is responsible for transaction management
func (s *UserServiceImpl) CreateUserWithRole(
	ctx context.Context,
	req model.CreateUserRequest,
	roleName string,
) (*entity.User, *entity.Role, error) {
	// 1. Check if user already exists
	existingUser, _ := s.userRepo.FindByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, nil, errors.New(constant.USER_EXISTS_MSG)
	}

	// 2. Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, errors.New("failed to hash password")
	}

	// 3. Get role from database
	role, err := s.userRepo.FindRoleByName(ctx, roleName)
	if err != nil {
		return nil, nil, errors.New("failed to find role: " + roleName)
	}

	// 4. Create user entity
	now := time.Now()
	user := &entity.User{
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		Password:    string(hashedPassword),
		Phone:       req.Phone,
		DateOfBirth: req.DateOfBirth,
		Gender:      req.Gender,
		IsActive:    true,
		RoleID:      role.ID,
		SellerID:    req.SellerID,
		BaseEntity: commonEntity.BaseEntity{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	// 5. Save user to database
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	return user, role, nil
}
