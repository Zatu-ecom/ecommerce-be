package entity

import (
	"time"

	"ecommerce-be/common/db"
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

type Sale struct {
	db.BaseEntity
	sellerId     uint           `gorm:"column:seller_id;type:integer;not null;index"`
	Name         string         `gorm:"column:name;type:varchar(255);not null"`
	Description  string         `gorm:"column:description;type:text"`
	Slug         string         `gorm:"column:slug;type:varchar(255);not null;unique"`
	BannerImages db.StringArray `gorm:"column:banner_images;type:text[]"`
	Status       CampaignStatus `gorm:"column:status;type:varchar(20);not null"`
	StartAt      time.Time      `gorm:"column:start_at;type:timestamp;not null"`
	EndAt        time.Time      `gorm:"column:end_at;type:timestamp;not null"`
}
