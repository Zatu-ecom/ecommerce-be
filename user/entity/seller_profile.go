package entity

import (
	"ecommerce-be/common/db"
)

// SellerProfile holds all business-specific data for a User whose role is SELLER.
type SellerProfile struct {
	db.BaseEntityWithoutID
	UserID uint `json:"userId" gorm:"primaryKey"`

	BusinessName       string  `json:"businessName" binding:"required"`
	BusinessLogoFileID *string `json:"businessLogoFileId" gorm:"column:business_logo_file_id;size:80"`
	TaxID              string  `json:"taxId" gorm:"unique"`
	IsVerified         bool    `json:"isVerified" gorm:"default:false"`
}
