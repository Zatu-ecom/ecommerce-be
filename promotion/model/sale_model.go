package model

import "ecommerce-be/promotion/entity"

// CreateSaleRequest represents the request body for creating a sale
type CreateSaleRequest struct {
	Name         string                `json:"name"         binding:"required,min=3,max=255"`
	Description  *string               `json:"description"  binding:"omitempty,max=2000"`
	Slug         *string               `json:"slug"         binding:"omitempty,max=255"`
	BannerImages []string              `json:"bannerImages" binding:"omitempty"`
	Status       entity.CampaignStatus `json:"status"       binding:"omitempty,oneof=draft scheduled active paused ended cancelled"`
	StartAt      string                `json:"startAt"      binding:"required"`
	EndAt        string                `json:"endAt"        binding:"required"`
}

// UpdateSaleRequest represents the request body for updating a sale
type UpdateSaleRequest struct {
	Name         string                `json:"name"         binding:"required,min=3,max=255"`
	Description  *string               `json:"description"  binding:"omitempty,max=2000"`
	Slug         *string               `json:"slug"         binding:"omitempty,max=255"`
	BannerImages []string              `json:"bannerImages" binding:"omitempty"`
	Status       entity.CampaignStatus `json:"status"       binding:"omitempty,oneof=draft scheduled active paused ended cancelled"`
	StartAt      string                `json:"startAt"      binding:"required"`
	EndAt        string                `json:"endAt"        binding:"required"`
}

// UpdateSaleStatusRequest represents the request body for updating sale status
type UpdateSaleStatusRequest struct {
	Status entity.CampaignStatus `json:"status" binding:"required,oneof=draft scheduled active paused ended cancelled"`
}

// SaleResponse represents sale data returned in API responses
type SaleResponse struct {
	ID           uint                  `json:"id"`
	SellerID     uint                  `json:"sellerId"`
	Name         string                `json:"name"`
	Description  *string               `json:"description,omitempty"`
	Slug         string                `json:"slug"`
	BannerImages []string              `json:"bannerImages,omitempty"`
	Status       entity.CampaignStatus `json:"status"`
	StartAt      string                `json:"startAt"`
	EndAt        string                `json:"endAt"`
	CreatedAt    string                `json:"createdAt"`
	UpdatedAt    string                `json:"updatedAt"`
}

// SalesResponse represents the response for listing sales
type SalesResponse struct {
	Sales []SaleResponse `json:"sales"`
}
