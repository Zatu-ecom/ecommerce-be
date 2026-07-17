package entity

import (
	"time"

	"ecommerce-be/common/db"
)

// PasswordResetToken represents a password reset token in the database
type PasswordResetToken struct {
	db.BaseEntity
	UserID    uint       `json:"userId"    gorm:"not null;index"`
	TokenHash string     `json:"tokenHash" gorm:"size:64;not null;index"`
	ExpiresAt time.Time  `json:"expiresAt" gorm:"not null;index"`
	IsUsed    bool       `json:"isUsed"    gorm:"not null;default:false"`
	UsedAt    *time.Time `json:"usedAt"`
}
