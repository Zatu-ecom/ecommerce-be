package entity

import (
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/common/helper"

	"gorm.io/gorm"
)

type CampaignStatus string

const (
	StatusDraft     CampaignStatus = "draft"
	StatusScheduled CampaignStatus = "scheduled"
	StatusActive    CampaignStatus = "active"
	StatusPaused    CampaignStatus = "paused"
	StatusEnded     CampaignStatus = "ended"
	StatusCancelled CampaignStatus = "cancelled"
)

// Sale represents a seller-created sales campaign that groups promotions
type Sale struct {
	db.BaseEntity

	SellerID     uint           `json:"sellerId"     gorm:"column:seller_id;not null;index"`
	Name         string         `json:"name"         gorm:"column:name;size:255;not null"`
	Description  *string        `json:"description" gorm:"column:description;type:text"`
	Slug         string         `json:"slug"         gorm:"column:slug;size:255;not null"`
	BannerFileIDs db.StringArray `json:"bannerFileIds" gorm:"column:banner_file_ids;type:text[]"`
	Status       CampaignStatus `json:"status"       gorm:"column:status;size:20;not null;default:draft"`
	StartAt      time.Time      `json:"startAt"      gorm:"column:start_at;not null"`
	EndAt        time.Time      `json:"endAt"        gorm:"column:end_at;not null"`
}

func (Sale) TableName() string {
	return "sale"
}

func (s *Sale) BeforeCreate(tx *gorm.DB) error {
	if s.Slug == "" {
		s.Slug = helper.GenerateSlug(s.Name)
	}
	return nil
}
