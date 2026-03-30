package model

import "ecommerce-be/common"

// BasePromotionScopeRequest contains common fields for all scope requests
type BasePromotionScopeRequest struct {
	PromotionID uint `json:"promotionId" binding:"required"`
}

// GetPromotionScopeRequest contains pagination parameters for get requests
type GetPromotionScopeRequest struct {
	BasePromotionScopeRequest
	common.BaseListParams
}

// BasePromotionScopeResponse contains common fields for all scope responses
type BasePromotionScopeResponse struct {
	PromotionID uint `json:"promotionId"`
}

// PaginationResponse alias for common pagination response
type PaginationResponse = common.PaginationResponse
