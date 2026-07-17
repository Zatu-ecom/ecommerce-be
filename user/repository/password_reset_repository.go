package repository

import (
	"context"

	"ecommerce-be/user/entity"
)

// PasswordResetRepository defines the interface for password reset token data operations
type PasswordResetRepository interface {
	// Create stores a new password reset token
	Create(ctx context.Context, token *entity.PasswordResetToken) error

	// FindByTokenHash retrieves a password reset token by its token hash
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.PasswordResetToken, error)

	// MarkAsUsed marks a password reset token as used at the current time
	MarkAsUsed(ctx context.Context, token *entity.PasswordResetToken) error

	// InvalidateAllForUser marks all active tokens for a user as used (in case of prior usage)
	InvalidateAllForUser(ctx context.Context, userID uint) error
}
