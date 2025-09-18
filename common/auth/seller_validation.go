package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ecommerce-be/common/cache"
	"ecommerce-be/common/constants"

	"gorm.io/gorm"
)

/********************************************************************
*		Cached seller validation functions (USED IN MIDDLEWARE)		*
*		These functions provide Redis caching for performance		*
*********************************************************************/

// SellerValidationResult represents the simplified, correct seller validation data
type SellerValidationResult struct {
	SellerID            uint       `json:"sellerId"`
	IsActive            bool       `json:"isActive"`
	SubscriptionStatus  string     `json:"subscriptionStatus"`
	SubscriptionEndDate *time.Time `json:"subscriptionEndDate"`
	PlanID              uint       `json:"planId"`
	PlanName            string     `json:"planName"`
	ValidationTimestamp time.Time  `json:"validationTimestamp"`
}

// IsSubscriptionActive checks if the subscription is currently active
func (svr *SellerValidationResult) IsSubscriptionActive() bool {
	activeStatuses := map[string]bool{
		"active":   true,
		"trialing": true,
		"past_due": true, // Grace period
	}

	if !activeStatuses[svr.SubscriptionStatus] {
		return false
	}

	if svr.SubscriptionEndDate != nil && svr.SubscriptionEndDate.Before(time.Now()) {
		return false
	}

	return true
}

// ValidateForAccess performs simplified validation based on current models
func (svr *SellerValidationResult) ValidateForAccess() error {
	if !svr.IsActive {
		return errors.New(constants.INVALID_SELLER_MSG)
	}

	if !svr.IsSubscriptionActive() {
		return errors.New(constants.SELLER_SUBSCRIPTION_INACTIVE_MSG)
	}

	return nil
}

func ValidateSellerCompleteCached(db *gorm.DB, sellerID uint) (*SellerValidationResult, error) {
	cacheKey := fmt.Sprintf("%s%d", constants.SELLER_COMPLETE_CACHE_KEY, sellerID)

	// Try to get from cache first
	cachedData, err := cache.GetKey(cacheKey)
	if err == nil {
		var result SellerValidationResult
		if jsonErr := json.Unmarshal([]byte(cachedData), &result); jsonErr == nil {
			return &result, nil
		}
	}

	// Cache miss - single optimized query with correct JOINs
	var result SellerValidationResult
	query := `
		SELECT 
			u.id as seller_id,
			u.is_active as is_active,
			COALESCE(s.status, 'unpaid') as subscription_status,
			s.end_date as subscription_end_date,
			COALESCE(p.id, 0) as plan_id,
			COALESCE(p.name, '') as plan_name,
			NOW() as validation_timestamp
		FROM "user" u
		LEFT JOIN subscription s ON u.id = s.seller_id 
			AND LOWER(s.status) IN ('active', 'trialing', 'past_due')
			AND (s.end_date IS NULL OR s.end_date > NOW())
		LEFT JOIN plan p ON s.plan_id = p.id
		WHERE u.id = ? AND u.role_id = (SELECT id FROM role WHERE UPPER(name) = 'SELLER' LIMIT 1)
	`

	dbErr := db.Raw(query, sellerID).Scan(&result).Error
	if dbErr != nil {
		failureResult := SellerValidationResult{
			SellerID:           sellerID,
			IsActive:           false,
			SubscriptionStatus: "inactive",
		}
		if jsonData, _ := json.Marshal(failureResult); jsonData != nil {
			cache.SetKey(cacheKey, string(jsonData), constants.SELLER_CACHE_SHORT_EXPIRATION)
		}
		return nil, errors.New(constants.INVALID_SELLER_MSG)
	}

	if result.SellerID == 0 {
		return nil, errors.New(constants.INVALID_SELLER_MSG)
	}

	// Cache the complete result as JSON
	if jsonData, jsonErr := json.Marshal(result); jsonErr == nil {
		cache.SetKey(cacheKey, string(jsonData), constants.SELLER_CACHE_EXPIRATION)
	}

	return &result, nil
}

// Optimized wrapper functions using the single query approach

// ValidateSellerSubscriptionOptimized - OPTIMIZED: Uses single query with caching
func ValidateSellerSubscriptionOptimized(db *gorm.DB, sellerID uint) error {
	result, err := ValidateSellerCompleteCached(db, sellerID)
	if err != nil {
		return err
	}

	if !result.IsSubscriptionActive() {
		return errors.New(constants.SELLER_SUBSCRIPTION_INACTIVE_MSG)
	}

	return nil
}

func ValidateSellerDetailsOptimized(db *gorm.DB, sellerID uint) error {
	result, err := ValidateSellerCompleteCached(db, sellerID)
	if err != nil {
		return err
	}

	return result.ValidateForAccess()
}

func GetSellerValidationData(db *gorm.DB, sellerID uint) (*SellerValidationResult, error) {
	return ValidateSellerCompleteCached(db, sellerID)
}
