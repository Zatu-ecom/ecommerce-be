package model

import "ecommerce-be/common"

// BaseDiscountCodeScopeRequest contains common fields for all discount-code scope requests
type BaseDiscountCodeScopeRequest struct {
	DiscountCodeID uint `json:"discountCodeId" binding:"required"`
}

// GetDiscountCodeScopeRequest contains pagination parameters for get requests
type GetDiscountCodeScopeRequest struct {
	BaseDiscountCodeScopeRequest
	common.BaseListParams
}

// BaseDiscountCodeScopeResponse contains common fields for all discount-code scope responses
type BaseDiscountCodeScopeResponse struct {
	DiscountCodeID uint `json:"discountCodeId"`
}
