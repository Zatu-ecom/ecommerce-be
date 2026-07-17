package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"ecommerce-be/common/config"
	"ecommerce-be/user/entity"
	userErrors "ecommerce-be/user/error"
	"ecommerce-be/user/model"
	"ecommerce-be/user/repository"

	"golang.org/x/crypto/bcrypt"
)

// PasswordResetService defines the interface for password reset business logic
type PasswordResetService interface {
	// ForgotPassword initiates the password reset flow.
	// Returns the raw reset token that should be sent to the user.
	// If the email does not exist, returns a nil token with no error (no user enumeration).
	ForgotPassword(ctx context.Context, req model.ForgotPasswordRequest) (*string, error)

	// ResetPassword validates the reset token and updates the user's password.
	ResetPassword(ctx context.Context, req model.ResetPasswordRequest) error
}

// PasswordResetServiceImpl implements the PasswordResetService interface
type PasswordResetServiceImpl struct {
	passwordResetRepo repository.PasswordResetRepository
	userRepo          repository.UserRepository
}

// NewPasswordResetService creates a new instance of PasswordResetService
func NewPasswordResetService(
	passwordResetRepo repository.PasswordResetRepository,
	userRepo repository.UserRepository,
) PasswordResetService {
	return &PasswordResetServiceImpl{
		passwordResetRepo: passwordResetRepo,
		userRepo:          userRepo,
	}
}

// ForgotPassword initiates the password reset flow.
// It generates a cryptographic reset token, stores its SHA-256 hash in the database,
// and returns the raw token. The caller (handler) is responsible for sending the token
// to the user via email or other channels.
//
// IMPORTANT: If the email does not exist, this function returns (nil, nil) to prevent
// user enumeration attacks. The handler must return the same success message either way.
func (s *PasswordResetServiceImpl) ForgotPassword(
	ctx context.Context,
	req model.ForgotPasswordRequest,
) (*string, error) {
	// Look up user by email (case-insensitive)
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		// User not found — return nil, nil to prevent user enumeration
		return nil, nil
	}

	// Generate a cryptographically secure random token
	rawToken, err := s.generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate reset token: %w", err)
	}

	// Hash the token for storage (SHA-256)
	tokenHash := hashToken(rawToken)

	// Get reset token expiry duration from config
	expiry := config.Get().PasswordReset.TokenExpiry()

	// Create the password reset token entity
	passwordResetToken := &entity.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().UTC().Add(expiry),
		IsUsed:    false,
	}

	// Store the token hash in the database
	if err := s.passwordResetRepo.Create(ctx, passwordResetToken); err != nil {
		return nil, fmt.Errorf("failed to store reset token: %w", err)
	}

	// TODO: Call notification service to send email with reset link when notification module is implemented
	// Reference: https://github.com/Societyfys/notification

	return &rawToken, nil
}

// ResetPassword validates the reset token and updates the user's password.
func (s *PasswordResetServiceImpl) ResetPassword(
	ctx context.Context,
	req model.ResetPasswordRequest,
) error {
	// Validate password confirmation
	if req.NewPassword != req.ConfirmPassword {
		return userErrors.ErrPasswordMismatch
	}

	// Hash the provided token to look up in the database
	tokenHash := hashToken(req.Token)

	// Find the token by its hash
	token, err := s.passwordResetRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to find reset token: %w", err)
	}
	if token == nil {
		return userErrors.ErrInvalidResetToken
	}

	// Check if the token has expired
	if time.Now().UTC().After(token.ExpiresAt) {
		return userErrors.ErrExpiredResetToken
	}

	// Check if the token has already been used
	if token.IsUsed {
		return userErrors.ErrUsedResetToken
	}

	// Look up the user
	user, err := s.userRepo.FindByID(ctx, token.UserID)
	if err != nil {
		return fmt.Errorf("failed to find user for reset: %w", err)
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update the user's password
	user.Password = string(hashedPassword)
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Mark the token as used
	if err := s.passwordResetRepo.MarkAsUsed(ctx, token); err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}

	// Invalidate all other active tokens for this user
	if err := s.passwordResetRepo.InvalidateAllForUser(ctx, user.ID); err != nil {
		return fmt.Errorf("failed to invalidate other tokens: %w", err)
	}

	return nil
}

// generateToken generates a cryptographically secure random token
func (s *PasswordResetServiceImpl) generateToken() (string, error) {
	byteLength := config.Get().PasswordReset.TokenByteLength
	if byteLength <= 0 {
		byteLength = 32
	}
	tokenBytes := make([]byte, byteLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(tokenBytes), nil
}

// hashToken returns the SHA-256 hash of a token as a hex string
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Ensure PasswordResetServiceImpl implements PasswordResetService
var _ PasswordResetService = (*PasswordResetServiceImpl)(nil)
