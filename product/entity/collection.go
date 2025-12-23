package entity

import (
	"ecommerce-be/common/db"

	"gorm.io/gorm"
)

// Collection represents a custom product collection (like "Summer Sale", "Best Sellers")
// Collections allow sellers to group products for marketing purposes
type Collection struct {
	db.BaseEntity

	// Owner
	SellerID uint `json:"sellerId" gorm:"column:seller_id;not null;index"`

	// Collection Info
	Name        string  `json:"name"        gorm:"column:name;size:255;not null"`
	Slug        string  `json:"slug"        gorm:"column:slug;size:255"`
	Description *string `json:"description" gorm:"column:description;type:text"`

	// Display
	Image *string `json:"image" gorm:"column:image;type:text"`

	// Status
	IsActive *bool `json:"isActive" gorm:"column:is_active;default:true;index"`

	// Relationships
	Products []CollectionProduct `json:"products,omitempty" gorm:"foreignKey:CollectionID"`
}

// TableName specifies the table name
func (Collection) TableName() string {
	return "collection"
}

// BeforeCreate hook
func (c *Collection) BeforeCreate(tx *gorm.DB) error {
	// Generate slug if empty
	if c.Slug == "" {
		c.Slug = generateSlug(c.Name)
	}
	return nil
}

// Helper function to generate slug (basic implementation)
func generateSlug(name string) string {
	// This is a simple implementation
	// In production, use a proper slug generation library
	// For now, just return the name (will be improved later)
	return name
}
