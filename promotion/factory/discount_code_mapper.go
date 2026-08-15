package factory

import (
	"strings"
	"time"

	"ecommerce-be/common/db"
	commonModel "ecommerce-be/common/model"
	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/model"
)

// NormalizeDiscountCode trims whitespace and uppercases the coupon code.
func NormalizeDiscountCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// convertValueToCents interprets a major-unit value in the seller's currency.
// For percentage discounts the value stays a plain number (FR-009); for
// fixed_amount it is converted to integer minor units.
func convertValueToCents(
	discountType entity.DiscountType,
	value float64,
	ccy commonModel.CurrencyInfo,
) (int64, error) {
	if discountType == entity.DiscountPercentage {
		// Percentage stays as-is, stored in the shared value column as a plain number.
		return int64(value), nil
	}
	return ccy.ToCents(value)
}

// DiscountCodeRequestToEntity maps a create request to a DiscountCode entity.
func DiscountCodeRequestToEntity(
	req model.CreateDiscountCodeRequest,
	sellerID uint,
	ccy commonModel.CurrencyInfo,
) (*entity.DiscountCode, error) {
	startsAt, endsAt, err := parseDiscountCodeDateRange(req.StartsAt, req.EndsAt)
	if err != nil {
		return nil, err
	}

	valueCents, err := convertValueToCents(req.DiscountType, req.Value, ccy)
	if err != nil {
		return nil, err
	}
	if err := validateDiscountCodeValue(req.DiscountType, valueCents, req.Metadata); err != nil {
		return nil, err
	}

	eligibility := req.CustomerEligibility
	if eligibility == "" {
		eligibility = entity.EligibleEveryone
	}
	if err := validateCustomerEligibility(eligibility, req.CustomerSegmentID); err != nil {
		return nil, err
	}

	resetType := req.UsageResetTimeType
	if resetType == "" {
		resetType = entity.ResetTimeTypeNone
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	autoStart := true
	if req.AutoStart != nil {
		autoStart = *req.AutoStart
	}
	autoEnd := true
	if req.AutoEnd != nil {
		autoEnd = *req.AutoEnd
	}

	// Mirror promotion "scheduled": future starts_at + auto_start → inactive until cron activates.
	now := time.Now().UTC()
	if autoStart && startsAt.After(now) {
		isActive = false
	}

	canCombine := false
	if req.CanCombineWithOtherDiscounts != nil {
		canCombine = *req.CanCombineWithOtherDiscounts
	}

	metadata := db.JSONMap{}
	if req.Metadata != nil {
		metadata = db.JSONMap(req.Metadata)
	}

	code := &entity.DiscountCode{
		SellerID:                     sellerID,
		Code:                         NormalizeDiscountCode(req.Code),
		Title:                        req.Title,
		Description:                  req.Description,
		DiscountType:                 req.DiscountType,
		Value:                        valueCents,
		MaxDiscountAmountCents:       majorToCentsPtr(req.MaxDiscountAmount, ccy),
		AppliesTo:                    req.AppliesTo,
		MinPurchaseAmountCents:       majorToCentsPtr(req.MinPurchaseAmount, ccy),
		MinQuantity:                  req.MinQuantity,
		CustomerEligibility:          eligibility,
		CustomerSegmentID:            req.CustomerSegmentID,
		UsageLimitTotal:              req.UsageLimitTotal,
		UsageLimitPerCustomer:        req.UsageLimitPerCustomer,
		UsageResetTimeType:           resetType,
		UsageResetAmount:             req.UsageResetAmount,
		CanCombineWithOtherDiscounts: &canCombine,
		StartsAt:                     &startsAt,
		EndsAt:                       endsAt,
		IsActive:                     &isActive,
		AutoStart:                    &autoStart,
		AutoEnd:                      &autoEnd,
		Metadata:                     metadata,
	}

	return code, nil
}

// majorToCentsPtr converts an optional major-unit amount to cents via the
// currency. Invalid precision is returned as an error through the caller.
func majorToCentsPtr(major *float64, ccy commonModel.CurrencyInfo) *int64 {
	if major == nil {
		return nil
	}
	cents, err := ccy.ToCents(*major)
	if err != nil {
		return nil
	}
	return &cents
}

// ApplyUpdateDiscountCodeRequest applies partial updates onto an existing entity.
// Code is intentionally immutable and is never changed here.
func ApplyUpdateDiscountCodeRequest(
	existing *entity.DiscountCode,
	req model.UpdateDiscountCodeRequest,
	ccy commonModel.CurrencyInfo,
) (*entity.DiscountCode, error) {
	discountType := existing.DiscountType
	if req.DiscountType != nil {
		discountType = *req.DiscountType
	}

	value := existing.Value
	if req.Value != nil {
		converted, err := convertValueToCents(discountType, *req.Value, ccy)
		if err != nil {
			return nil, err
		}
		value = converted
	}

	metadata := map[string]any(existing.Metadata)
	if req.Metadata != nil {
		metadata = *req.Metadata
	}

	if err := validateDiscountCodeValue(discountType, value, metadata); err != nil {
		return nil, err
	}

	startsAtStr := ""
	if existing.StartsAt != nil {
		startsAtStr = existing.StartsAt.UTC().Format(time.RFC3339)
	}
	if req.StartsAt != nil {
		startsAtStr = *req.StartsAt
	}

	var endsAtStr *string
	if existing.EndsAt != nil {
		formatted := existing.EndsAt.UTC().Format(time.RFC3339)
		endsAtStr = &formatted
	}
	if req.EndsAt != nil {
		endsAtStr = req.EndsAt
	}

	startsAt, endsAt, err := parseDiscountCodeDateRange(startsAtStr, endsAtStr)
	if err != nil {
		return nil, err
	}

	eligibility := existing.CustomerEligibility
	if req.CustomerEligibility != nil {
		eligibility = *req.CustomerEligibility
	}
	segmentID := existing.CustomerSegmentID
	if req.CustomerSegmentID != nil {
		segmentID = req.CustomerSegmentID
	}
	if err := validateCustomerEligibility(eligibility, segmentID); err != nil {
		return nil, err
	}

	if req.Title != nil {
		existing.Title = req.Title
	}
	if req.Description != nil {
		existing.Description = req.Description
	}
	existing.DiscountType = discountType
	existing.Value = value
	if req.MaxDiscountAmount != nil {
		if cents := majorToCentsPtr(req.MaxDiscountAmount, ccy); cents != nil {
			existing.MaxDiscountAmountCents = cents
		}
	}
	if req.AppliesTo != nil {
		existing.AppliesTo = *req.AppliesTo
	}
	if req.MinPurchaseAmount != nil {
		if cents := majorToCentsPtr(req.MinPurchaseAmount, ccy); cents != nil {
			existing.MinPurchaseAmountCents = cents
		}
	}
	if req.MinQuantity != nil {
		existing.MinQuantity = req.MinQuantity
	}
	existing.CustomerEligibility = eligibility
	existing.CustomerSegmentID = segmentID
	if req.UsageLimitTotal != nil {
		existing.UsageLimitTotal = req.UsageLimitTotal
	}
	if req.UsageLimitPerCustomer != nil {
		existing.UsageLimitPerCustomer = req.UsageLimitPerCustomer
	}
	if req.UsageResetTimeType != nil {
		existing.UsageResetTimeType = *req.UsageResetTimeType
	}
	if req.UsageResetAmount != nil {
		existing.UsageResetAmount = req.UsageResetAmount
	}
	if req.CanCombineWithOtherDiscounts != nil {
		existing.CanCombineWithOtherDiscounts = req.CanCombineWithOtherDiscounts
	}
	existing.StartsAt = &startsAt
	existing.EndsAt = endsAt
	if req.IsActive != nil {
		existing.IsActive = req.IsActive
	}
	if req.AutoStart != nil {
		existing.AutoStart = req.AutoStart
	}
	if req.AutoEnd != nil {
		existing.AutoEnd = req.AutoEnd
	}
	// If dates move into the future and auto-start is on, schedule (inactive) again.
	autoStart := existing.AutoStart == nil || *existing.AutoStart
	if autoStart && startsAt.After(time.Now().UTC()) {
		inactive := false
		existing.IsActive = &inactive
	}
	if req.Metadata != nil {
		existing.Metadata = db.JSONMap(*req.Metadata)
	}
	existing.UpdatedAt = time.Now().UTC()

	return existing, nil
}

// DiscountCodeEntityToResponse maps an entity to the API response model.
func DiscountCodeEntityToResponse(
	code *entity.DiscountCode,
	ccy commonModel.CurrencyInfo,
) *model.DiscountCodeResponse {
	isActive := true
	if code.IsActive != nil {
		isActive = *code.IsActive
	}

	var startsAt string
	if code.StartsAt != nil {
		startsAt = code.StartsAt.UTC().Format(time.RFC3339)
	}

	var endsAt *string
	if code.EndsAt != nil {
		formatted := code.EndsAt.UTC().Format(time.RFC3339)
		endsAt = &formatted
	}

	metadata := map[string]any(code.Metadata)
	if metadata == nil {
		metadata = map[string]any{}
	}

	// Value: Money for fixed_amount, plain number for percentage (FR-009).
	var value any
	if code.DiscountType == entity.DiscountPercentage {
		value = float64(code.Value)
	} else {
		value = commonModel.NewMoney(code.Value, ccy)
	}

	return &model.DiscountCodeResponse{
		ID:                           code.ID,
		SellerID:                     code.SellerID,
		Code:                         code.Code,
		Title:                        code.Title,
		Description:                  code.Description,
		DiscountType:                 code.DiscountType,
		Value:                        value,
		MaxDiscountAmount:            centsToMoneyPtr(code.MaxDiscountAmountCents, ccy),
		AppliesTo:                    code.AppliesTo,
		MinPurchaseAmount:            centsToMoneyPtr(code.MinPurchaseAmountCents, ccy),
		MinQuantity:                  code.MinQuantity,
		CustomerEligibility:          code.CustomerEligibility,
		CustomerSegmentID:            code.CustomerSegmentID,
		UsageLimitTotal:              code.UsageLimitTotal,
		UsageLimitPerCustomer:        code.UsageLimitPerCustomer,
		CurrentUsageCount:            code.CurrentUsageCount,
		UsageResetTimeType:           code.UsageResetTimeType,
		UsageResetAmount:             code.UsageResetAmount,
		CanCombineWithOtherDiscounts: code.CanCombineWithOtherDiscounts,
		StartsAt:                     startsAt,
		EndsAt:                       endsAt,
		IsActive:                     isActive,
		AutoStart:                    code.AutoStart,
		AutoEnd:                      code.AutoEnd,
		Metadata:                     metadata,
		CreatedAt:                    code.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:                    code.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// centsToMoneyPtr wraps an optional cents value as a Money pointer.
func centsToMoneyPtr(cents *int64, ccy commonModel.CurrencyInfo) *commonModel.Money {
	if cents == nil {
		return nil
	}
	m := commonModel.NewMoney(*cents, ccy)
	return &m
}

func parseDiscountCodeDateRange(startsAt string, endsAt *string) (time.Time, *time.Time, error) {
	start, err := time.Parse(time.RFC3339, startsAt)
	if err != nil {
		return time.Time{}, nil, promoErrors.ErrInvalidDiscountCodeDateRange
	}
	start = start.UTC()

	if endsAt == nil || *endsAt == "" {
		return start, nil, nil
	}

	end, err := time.Parse(time.RFC3339, *endsAt)
	if err != nil {
		return time.Time{}, nil, promoErrors.ErrInvalidDiscountCodeDateRange
	}
	end = end.UTC()
	if !end.After(start) {
		return time.Time{}, nil, promoErrors.ErrInvalidDiscountCodeDateRange.WithMessage(
			"endsAt must be after startsAt",
		)
	}
	return start, &end, nil
}

func validateDiscountCodeValue(
	discountType entity.DiscountType,
	value int64,
	metadata map[string]any,
) error {
	switch discountType {
	case entity.DiscountPercentage:
		if value < 1 || value > 100 {
			return promoErrors.ErrInvalidDiscountCodeValue.WithMessage(
				"percentage value must be between 1 and 100",
			)
		}
	case entity.DiscountFixedAmount:
		if value <= 0 {
			return promoErrors.ErrInvalidDiscountCodeValue.WithMessage(
				"fixed_amount value must be greater than 0",
			)
		}
	case entity.DiscountFreeShipping:
		if value != 0 {
			return promoErrors.ErrInvalidDiscountCodeValue.WithMessage(
				"free_shipping value must be 0",
			)
		}
	case entity.DiscountBuyXGetY:
		if err := validateBuyXGetYMetadata(metadata); err != nil {
			return err
		}
	default:
		return promoErrors.ErrInvalidDiscountCodeValue
	}
	return nil
}

func validateBuyXGetYMetadata(metadata map[string]any) error {
	if metadata == nil {
		return promoErrors.ErrInvalidDiscountCodeValue.WithMessage(
			"buy_x_get_y requires metadata with buyQuantity, getQuantity, getDiscountPercent",
		)
	}

	buyQty, ok := asPositiveInt(metadata["buyQuantity"])
	if !ok {
		return promoErrors.ErrInvalidDiscountCodeValue.WithMessage(
			"buyQuantity must be an integer >= 1",
		)
	}
	getQty, ok := asPositiveInt(metadata["getQuantity"])
	if !ok {
		return promoErrors.ErrInvalidDiscountCodeValue.WithMessage(
			"getQuantity must be an integer >= 1",
		)
	}
	_ = buyQty
	_ = getQty

	pct, ok := asNumberInRange(metadata["getDiscountPercent"], 0, 100)
	if !ok {
		return promoErrors.ErrInvalidDiscountCodeValue.WithMessage(
			"getDiscountPercent must be between 0 and 100",
		)
	}
	_ = pct

	return nil
}

func validateCustomerEligibility(
	eligibility entity.EligibilityType,
	segmentID *uint,
) error {
	if eligibility == entity.EligibleSpecificSegment {
		if segmentID == nil || *segmentID == 0 {
			return promoErrors.ErrInvalidDiscountCodeValue.WithMessage(
				"customerSegmentId is required when customerEligibility is specific_segment",
			)
		}
	}
	return nil
}

func asPositiveInt(v any) (int, bool) {
	n, ok := asNumberInRange(v, 1, 1<<31-1)
	if !ok {
		return 0, false
	}
	return int(n), true
}

func asNumberInRange(v any, min, max float64) (float64, bool) {
	var n float64
	switch t := v.(type) {
	case float64:
		n = t
	case float32:
		n = float64(t)
	case int:
		n = float64(t)
	case int64:
		n = float64(t)
	case int32:
		n = float64(t)
	default:
		return 0, false
	}
	if n < min || n > max {
		return 0, false
	}
	return n, true
}
