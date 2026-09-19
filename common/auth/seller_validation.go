package auth

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/config"
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

// Seller validation cache TTLs. The success TTL is capped by the
// subscription end date so a cached "active" verdict can never outlive the
// subscription (pre-spec §4.5). Failure TTL bounds negative caching of
// database errors (fail-open retries refill quickly).
const (
	sellerValidationTTL         = 5 * time.Minute
	sellerValidationFailureTTL  = 2 * time.Minute
	sellerValidationNegativeTTL = 60 * time.Second
)

// sellerValidationFlight collapses concurrent validation misses per process.
var sellerValidationFlight cachekit.Flight

// sellerValidationRecorder records seller-validation cache events.
var sellerValidationRecorder = cachekit.LogRecorder()

// sellerValidationEnabled reports whether seller-validation caching applies.
// Nil cache or disabled flags fall back to the legacy direct-DB path.
func sellerValidationEnabled() bool {
	if cachekit.DefaultCache() == nil {
		return false
	}
	cfg := config.Get()
	return cfg != nil && cfg.Cache.Enabled && cfg.Cache.SellerValidation
}

// ValidateSellerCompleteCached validates a seller with cache-aside. The ctx
// carries timeouts and tracing into cache and fill paths.
func ValidateSellerCompleteCached(ctx context.Context, db *gorm.DB, sellerID uint) (*SellerValidationResult, error) {
	if !sellerValidationEnabled() {
		return validateSellerComplete(db, sellerID)
	}
	cacheKey, err := cachekit.BuildSellerKey(sellerID, "seller", "complete")
	if err != nil {
		return validateSellerComplete(db, sellerID)
	}
	c := cachekit.DefaultCache()
	start := time.Now()

	body, err := c.Get(ctx, cacheKey)
	if err == nil {
		if cachekit.IsTombstone(body) {
			sellerValidationRecorder.Record(ctx, "user", "seller-validation", cachekit.StoreCache, cachekit.ResultMiss, time.Since(start))
			return nil, errors.New(constants.INVALID_SELLER_MSG)
		}
		var hit SellerValidationResult
		if jsonErr := json.Unmarshal(body, &hit); jsonErr == nil {
			sellerValidationRecorder.Record(ctx, "user", "seller-validation", cachekit.StoreCache, cachekit.ResultHit, time.Since(start))
			return &hit, nil
		}
	}

	// Miss or backend failure: singleflight fill from the database.
	res, _, fillErr := sellerValidationFlight.Do(cacheKey, func() (any, error) {
		return fillSellerValidation(ctx, c, cacheKey, db, sellerID)
	})
	if fillErr != nil {
		// Fill ran and failed (database error): identical to the legacy path.
		return nil, fillErr
	}
	result, ok := res.(*SellerValidationResult)
	if !ok || result == nil {
		return nil, errors.New(constants.INVALID_SELLER_MSG)
	}
	return result, nil
}

// fillSellerValidation loads validation from the database and stores it with
// a subscription-capped TTL. Database failures store a short tombstone and
// return the legacy INVALID_SELLER error.
func fillSellerValidation(
	ctx context.Context,
	c cachekit.Cache,
	cacheKey string,
	db *gorm.DB,
	sellerID uint,
) (*SellerValidationResult, error) {
	start := time.Now()
	result, err := validateSellerComplete(db, sellerID)
	if err != nil {
		_ = c.Set(ctx, cacheKey, cachekit.TombstoneBytes,
			cachekit.JitteredTTL(sellerValidationFailureTTL))
		sellerValidationRecorder.Record(ctx, "user", "seller-validation",
			cachekit.StoreCache, cachekit.ResultMiss, time.Since(start))
		return nil, err
	}
	ttl := validationTTLFor(result)
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if len(raw) > cachekit.MaxValueBytes {
		return result, nil // serve without storing; never truncate
	}
	if setErr := c.Set(ctx, cacheKey, raw, cachekit.JitteredTTL(ttl)); setErr != nil {
		sellerValidationRecorder.Record(ctx, "user", "seller-validation",
			cachekit.StoreCache, cachekit.ResultDropped, time.Since(start))
	}
	return result, nil
}

// validationTTLFor caps the verdict lifetime at the subscription end so a
// cached "active" cannot outlive the subscription. Past/near ends fall back
// to the negative window (fail-open refills quickly).
func validationTTLFor(result *SellerValidationResult) time.Duration {
	if result.SubscriptionEndDate == nil {
		return sellerValidationTTL
	}
	remaining := time.Until(*result.SubscriptionEndDate)
	switch {
	case remaining <= 0:
		return sellerValidationNegativeTTL
	case remaining < sellerValidationTTL:
		return remaining
	default:
		return sellerValidationTTL
	}
}

// validateSellerComplete runs the single optimized validation query without
// caching (legacy path and cache fill).
func validateSellerComplete(db *gorm.DB, sellerID uint) (*SellerValidationResult, error) {
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
		return nil, errors.New(constants.INVALID_SELLER_MSG)
	}

	if result.SellerID == 0 {
		return nil, errors.New(constants.INVALID_SELLER_MSG)
	}

	return &result, nil
}

// Optimized wrapper functions using the single query approach

// ValidateSellerSubscriptionOptimized - OPTIMIZED: Uses single query with caching
func ValidateSellerSubscriptionOptimized(ctx context.Context, db *gorm.DB, sellerID uint) error {
	result, err := ValidateSellerCompleteCached(ctx, db, sellerID)
	if err != nil {
		return err
	}

	if !result.IsSubscriptionActive() {
		return errors.New(constants.SELLER_SUBSCRIPTION_INACTIVE_MSG)
	}

	return nil
}

func ValidateSellerDetailsOptimized(ctx context.Context, db *gorm.DB, sellerID uint) error {
	result, err := ValidateSellerCompleteCached(ctx, db, sellerID)
	if err != nil {
		return err
	}

	return result.ValidateForAccess()
}

func GetSellerValidationData(ctx context.Context, db *gorm.DB, sellerID uint) (*SellerValidationResult, error) {
	return ValidateSellerCompleteCached(ctx, db, sellerID)
}
