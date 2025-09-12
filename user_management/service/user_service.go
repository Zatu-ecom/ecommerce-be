package service

import (
	"errors"
	"os"
	"time"

	"ecommerce-be/common"
	commonEntity "ecommerce-be/common/entity"
	"ecommerce-be/user_management/entity"
	"ecommerce-be/user_management/model"
	"ecommerce-be/user_management/repositories"
	"ecommerce-be/user_management/utils"

	"golang.org/x/crypto/bcrypt"
)

// UserService defines the interface for user-related business logic
type UserService interface {
	Register(req model.UserRegisterRequest) (*model.AuthResponse, error)
	Login(req model.UserLoginRequest) (*model.AuthResponse, error)
	GetProfile(userID uint) (*model.ProfileResponse, error)
	UpdateProfile(userID uint, req model.UserUpdateRequest) (*model.UserResponse, error)
	ChangePassword(userID uint, req model.UserPasswordChangeRequest) error
	RefreshToken(userID uint, email string) (*model.TokenResponse, error)
}

// UserServiceImpl implements the UserService interface
type UserServiceImpl struct {
	userRepo       repositories.UserRepository
	addressService AddressService
}

// NewUserService creates a new instance of UserService
func NewUserService(
	userRepo repositories.UserRepository,
	addressService AddressService,
) UserService {
	return &UserServiceImpl{
		userRepo:       userRepo,
		addressService: addressService,
	}
}

// Register creates a new user account
func (s *UserServiceImpl) Register(req model.UserRegisterRequest) (*model.AuthResponse, error) {
	// Validate password confirmation
	if req.Password != req.ConfirmPassword {
		return nil, errors.New(utils.PasswordMismatchMsg)
	}

	// Check if user already exists
	existingUser, _ := s.userRepo.FindByEmail(req.Email)
	if existingUser != nil {
		return nil, errors.New(utils.UserExistsMsg)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user entity
	user := &entity.User{
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		Password:    string(hashedPassword),
		Phone:       req.Phone,
		DateOfBirth: req.DateOfBirth,
		Gender:      req.Gender,
		IsActive:    true,
		BaseEntity: commonEntity.BaseEntity{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// Save user to database
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	// Generate JWT token
	token, err := common.GenerateToken(user.ID, user.Email, os.Getenv("JWT_SECRET"))
	if err != nil {
		return nil, err
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

	authResponse := &model.AuthResponse{
		User:      userResponse,
		Token:     token,
		ExpiresIn: utils.TokenExpirationDisplay,
	}

	return authResponse, nil
}

// Login authenticates a user and returns a token
func (s *UserServiceImpl) Login(req model.UserLoginRequest) (*model.AuthResponse, error) {
	// Find user by email
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, errors.New(utils.InvalidCredentialsMsg)
	}

	// Check if account is active
	if !user.IsActive {
		return nil, errors.New(utils.AccountDeactivatedMsg)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New(utils.InvalidCredentialsMsg)
	}

	// Generate JWT token
	token, err := common.GenerateToken(user.ID, user.Email, os.Getenv("JWT_SECRET"))
	if err != nil {
		return nil, err
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

	authResponse := &model.AuthResponse{
		User:      userResponse,
		Token:     token,
		ExpiresIn: utils.TokenExpirationDisplay,
	}

	return authResponse, nil
}

// GetProfile retrieves user profile information including addresses
func (s *UserServiceImpl) GetProfile(userID uint) (*model.ProfileResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New(utils.UserNotFoundMsg)
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
	}

	addresses, err := s.addressService.GetAddresses(userID)
	if err != nil {
		return nil, err
	}

	// For now, return empty addresses array
	addressesResList := []model.AddressResponse{}
	for _, address := range addresses {
		addressesResList = append(addressesResList,
			model.AddressResponse{
				ID:      address.ID,
				Street:  address.Street,
				City:    address.City,
				State:   address.State,
				ZipCode: address.ZipCode,
				Country: address.Country,
			})
	}

	profileResponse := &model.ProfileResponse{
		UserResponse: userResponse,
		Addresses:    addressesResList,
	}

	return profileResponse, nil
}

// UpdateProfile updates user profile information
func (s *UserServiceImpl) UpdateProfile(
	userID uint,
	req model.UserUpdateRequest,
) (*model.UserResponse, error) {
	// Find user by ID
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	// Update user fields
	user.FirstName = req.FirstName
	user.LastName = req.LastName
	user.Phone = req.Phone
	user.DateOfBirth = req.DateOfBirth
	user.Gender = req.Gender
	user.UpdatedAt = time.Now()

	// Save changes to database
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	// Create response
	userResponse := &model.UserResponse{
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

	return userResponse, nil
}

// ChangePassword updates a user's password
func (s *UserServiceImpl) ChangePassword(userID uint, req model.UserPasswordChangeRequest) error {
	// Check if new password matches confirmation
	if req.NewPassword != req.ConfirmPassword {
		return errors.New(utils.PasswordMismatchMsg)
	}

	// Find user by ID
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		return errors.New(utils.InvalidCurrentPasswordMsg)
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
	return s.userRepo.Update(user)
}

// RefreshToken generates a new JWT token
func (s *UserServiceImpl) RefreshToken(userID uint, email string) (*model.TokenResponse, error) {
	token, err := common.GenerateToken(userID, email, os.Getenv("JWT_SECRET"))
	if err != nil {
		return nil, err
	}

	tokenResponse := &model.TokenResponse{
		Token:     token,
		ExpiresIn: utils.TokenExpirationDisplay,
	}

	return tokenResponse, nil
}
