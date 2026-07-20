package repository

import (
	"context"
	"errors"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/user/entity"

	"gorm.io/gorm"
)

// PasswordResetRepositoryImpl implements the PasswordResetRepository interface
type PasswordResetRepositoryImpl struct{}

// NewPasswordResetRepository creates a new instance of PasswordResetRepository
func NewPasswordResetRepository() PasswordResetRepository {
	return &PasswordResetRepositoryImpl{}
}

// Create stores a new password reset token in the database
func (r *PasswordResetRepositoryImpl) Create(
	ctx context.Context,
	token *entity.PasswordResetToken,
) error {
	return db.DB(ctx).Create(token).Error
}

// FindByTokenHash retrieves a password reset token by its token hash
func (r *PasswordResetRepositoryImpl) FindByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*entity.PasswordResetToken, error) {
	var token entity.PasswordResetToken
	result := db.DB(ctx).Where("token_hash = ?", tokenHash).First(&token)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &token, nil
}

// MarkAsUsed marks a password reset token as used at the current time
func (r *PasswordResetRepositoryImpl) MarkAsUsed(
	ctx context.Context,
	token *entity.PasswordResetToken,
) error {
	now := time.Now().UTC()
	return db.DB(ctx).Model(token).Updates(map[string]interface{}{
		"is_used": true,
		"used_at": now,
	}).Error
}

// InvalidateAllForUser marks all active (unused and not expired) tokens for a user as used
func (r *PasswordResetRepositoryImpl) InvalidateAllForUser(ctx context.Context, userID uint) error {
	now := time.Now().UTC()
	return db.DB(ctx).Model(&entity.PasswordResetToken{}).
		Where("user_id = ? AND is_used = ? AND expires_at > ?", userID, false, now).
		Updates(map[string]interface{}{
			"is_used": true,
			"used_at": now,
		}).Error
}
