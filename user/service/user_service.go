package service

import (
	"errors"
	"os"
	"time"

	"datun.com/be/common"
	"datun.com/be/user/entity"
	"datun.com/be/user/model"
	"datun.com/be/user/repositories"
	"golang.org/x/crypto/bcrypt"
)

// UserService defines the interface for user-related business logic
type UserService interface {
	Register(req model.UserRegisterRequest) (*entity.User, string, error)
	Login(req model.UserLoginRequest) (*entity.User, string, error)
	GetProfile(userID uint) (*entity.User, error)
	UpdateProfile(userID uint, req model.UserUpdateRequest) (*entity.User, error)
	ChangePassword(userID uint, req model.UserPasswordChangeRequest) error
	RefreshToken(userID uint, email string) (string, error)
}

// UserServiceImpl implements the UserService interface
type UserServiceImpl struct {
	userRepo repositories.UserRepository
}

// NewUserService creates a new instance of UserService
func NewUserService(userRepo repositories.UserRepository) UserService {
	return &UserServiceImpl{
		userRepo: userRepo,
	}
}

// Register creates a new user account
func (s *UserServiceImpl) Register(req model.UserRegisterRequest) (*entity.User, string, error) {
	// Check if user already exists
	existingUser, _ := s.userRepo.FindByEmail(req.Email)
	if existingUser != nil {
		return nil, "", errors.New("user with this email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
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
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save user to database
	if err := s.userRepo.Create(user); err != nil {
		return nil, "", err
	}

	// Generate JWT token
	token, err := common.GenerateToken(user.ID, user.Email, os.Getenv("JWT_SECRET"))
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// Login authenticates a user and returns a token
func (s *UserServiceImpl) Login(req model.UserLoginRequest) (*entity.User, string, error) {
	// Find user by email
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, "", errors.New("invalid email or password")
	}

	// Check if account is active
	if !user.IsActive {
		return nil, "", errors.New("account is deactivated")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, "", errors.New("invalid email or password")
	}

	// Generate JWT token
	token, err := common.GenerateToken(user.ID, user.Email, os.Getenv("JWT_SECRET"))
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// GetProfile retrieves user profile information
func (s *UserServiceImpl) GetProfile(userID uint) (*entity.User, error) {
	return s.userRepo.FindByID(userID)
}

// UpdateProfile updates user profile information
func (s *UserServiceImpl) UpdateProfile(userID uint, req model.UserUpdateRequest) (*entity.User, error) {
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

	return user, nil
}

// ChangePassword updates a user's password
func (s *UserServiceImpl) ChangePassword(userID uint, req model.UserPasswordChangeRequest) error {
	// Check if new password matches confirmation
	if req.NewPassword != req.ConfirmPassword {
		return errors.New("new password and confirmation do not match")
	}

	// Find user by ID
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		return errors.New("current password is incorrect")
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
func (s *UserServiceImpl) RefreshToken(userID uint, email string) (string, error) {
	return common.GenerateToken(userID, email, os.Getenv("JWT_SECRET"))
}
