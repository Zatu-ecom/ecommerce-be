package mapper

import "time"

// WishlistWithItemCount represents a wishlist with its item count from a single query
type WishlistWithItemCount struct {
	ID        uint      `gorm:"column:id"`
	UserID    uint      `gorm:"column:user_id"`
	Name      string    `gorm:"column:name"`
	IsDefault bool      `gorm:"column:is_default"`
	ItemCount int       `gorm:"column:item_count"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}
