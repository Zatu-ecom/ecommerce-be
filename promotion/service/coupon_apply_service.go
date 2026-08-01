package service

import (
	"context"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/common/log"
	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/factory"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/repository"
	discountCodeStrategy "ecommerce-be/promotion/service/discountCodeStrategy"
	promoConstant "ecommerce-be/promotion/utils/constant"
)

// CouponApplyService orchestrates cart coupon validation and calculation.
type CouponApplyService interface {
	ValidateCouponForCart(
		ctx context.Context,
		code string,
		req *model.CouponCartRequest,
	) (*entity.DiscountCode, error)
	ApplyCouponsToCart(
		ctx context.Context,
		req *model.CouponCartRequest,
	) (*model.AppliedCouponSummary, error)
	ListAvailableCouponsForCart(
		ctx context.Context,
		req *model.CouponCartRequest,
	) (*model.AvailableCouponsResponse, error)
	// EnsureCouponsValidAtCheckout re-validates every applied coupon inside the checkout TX
	// so stale cart snapshots cannot redeem deactivated/expired/ineligible codes.
	EnsureCouponsValidAtCheckout(ctx context.Context, req *model.CouponCartRequest) error
	RecordCouponUsages(
		ctx context.Context,
		orderID, userID uint,
		records []model.CouponUsageRecord,
	) error
}

type CouponApplyServiceImpl struct {
	discountCodeRepo    repository.DiscountCodeRepository
	usageRepo           repository.DiscountCodeUsageRepository
	productScopeRepo    repository.DiscountCodeProductScopeRepository
	variantScopeRepo    repository.DiscountCodeVariantScopeRepository
	categoryScopeRepo   repository.DiscountCodeCategoryScopeRepository
	collectionScopeRepo repository.DiscountCodeCollectionScopeRepository
}

func NewCouponApplyService(
	discountCodeRepo repository.DiscountCodeRepository,
	usageRepo repository.DiscountCodeUsageRepository,
	productScopeRepo repository.DiscountCodeProductScopeRepository,
	variantScopeRepo repository.DiscountCodeVariantScopeRepository,
	categoryScopeRepo repository.DiscountCodeCategoryScopeRepository,
	collectionScopeRepo repository.DiscountCodeCollectionScopeRepository,
) CouponApplyService {
	return &CouponApplyServiceImpl{
		discountCodeRepo:    discountCodeRepo,
		usageRepo:           usageRepo,
		productScopeRepo:    productScopeRepo,
		variantScopeRepo:    variantScopeRepo,
		categoryScopeRepo:   categoryScopeRepo,
		collectionScopeRepo: collectionScopeRepo,
	}
}

func (s *CouponApplyServiceImpl) ValidateCouponForCart(
	ctx context.Context,
	code string,
	req *model.CouponCartRequest,
) (*entity.DiscountCode, error) {
	normalized := factory.NormalizeDiscountCode(code)
	dc, err := s.discountCodeRepo.FindByCode(ctx, req.SellerID, normalized)
	if err != nil {
		return nil, err
	}
	if dc == nil {
		return nil, promoErrors.ErrInvalidCoupon
	}

	for _, id := range req.AppliedDiscountCodeIDs {
		if id == dc.ID {
			return nil, promoErrors.ErrCouponAlreadyApplied
		}
	}

	if err := s.validateCouponRules(ctx, dc, req); err != nil {
		return nil, err
	}

	eligible := s.eligibleItemIDs(ctx, dc, req)
	if len(eligible) == 0 && dc.DiscountType != entity.DiscountFreeShipping {
		return nil, promoErrors.ErrCouponNotApplicable
	}

	strategy := discountCodeStrategy.GetDiscountStrategy(dc.DiscountType)
	if strategy == nil {
		return nil, promoErrors.ErrInvalidCoupon
	}
	merch, ship, err := strategy.Calculate(ctx, dc, req, eligible)
	if err != nil {
		return nil, err
	}
	if merch+ship <= 0 {
		return nil, promoErrors.ErrCouponNotApplicable
	}

	return dc, nil
}

func (s *CouponApplyServiceImpl) ApplyCouponsToCart(
	ctx context.Context,
	req *model.CouponCartRequest,
) (*model.AppliedCouponSummary, error) {
	summary := &model.AppliedCouponSummary{
		AppliedCoupons: make([]model.CouponValidationResult, 0),
		SkippedCoupons: make([]model.SkippedCouponResult, 0),
	}

	if len(req.AppliedDiscountCodeIDs) == 0 {
		return summary, nil
	}

	codes := make([]*entity.DiscountCode, 0, len(req.AppliedDiscountCodeIDs))
	for _, id := range req.AppliedDiscountCodeIDs {
		dc, err := s.discountCodeRepo.FindByID(ctx, id)
		if err != nil {
			summary.SkippedCoupons = append(summary.SkippedCoupons, model.SkippedCouponResult{
				Reason: promoErrors.INVALID_COUPON_MSG,
			})
			continue
		}
		codes = append(codes, dc)
	}

	appliedSoFar := make([]uint, 0, len(codes))
	for _, dc := range codes {
		partialReq := *req
		partialReq.AppliedDiscountCodeIDs = append([]uint(nil), appliedSoFar...)

		if err := s.validateCouponRules(ctx, dc, &partialReq); err != nil {
			summary.SkippedCoupons = append(summary.SkippedCoupons, model.SkippedCouponResult{
				DiscountCode: factory.DiscountCodeEntityToResponse(dc),
				Reason:       err.Error(),
			})
			continue
		}

		eligible := s.eligibleItemIDs(ctx, dc, req)
		strategy := discountCodeStrategy.GetDiscountStrategy(dc.DiscountType)
		if strategy == nil {
			continue
		}
		merch, ship, err := strategy.Calculate(ctx, dc, req, eligible)
		if err != nil || merch+ship <= 0 {
			summary.SkippedCoupons = append(summary.SkippedCoupons, model.SkippedCouponResult{
				DiscountCode: factory.DiscountCodeEntityToResponse(dc),
				Reason:       promoErrors.COUPON_NOT_APPLICABLE_MSG,
			})
			continue
		}

		summary.AppliedCoupons = append(summary.AppliedCoupons, model.CouponValidationResult{
			DiscountCode:     factory.DiscountCodeEntityToResponse(dc),
			IsValid:          true,
			DiscountCents:    merch,
			ShippingDiscount: ship,
		})
		summary.TotalDiscountCents += merch
		summary.ShippingDiscount += ship
		appliedSoFar = append(appliedSoFar, dc.ID)
	}

	return summary, nil
}

func (s *CouponApplyServiceImpl) ListAvailableCouponsForCart(
	ctx context.Context,
	req *model.CouponCartRequest,
) (*model.AvailableCouponsResponse, error) {
	active := true
	codes, _, err := s.discountCodeRepo.List(ctx, repository.ListDiscountCodeFilter{
		SellerID: req.SellerID,
		IsActive: &active,
		Page:     1,
		Limit:    promoConstant.AVAILABLE_COUPONS_LIST_LIMIT,
	})
	if err != nil {
		return nil, err
	}

	applied := map[uint]struct{}{}
	for _, id := range req.AppliedDiscountCodeIDs {
		applied[id] = struct{}{}
	}

	resp := &model.AvailableCouponsResponse{
		Applicable:    make([]model.AvailableCouponInfo, 0),
		NotApplicable: make([]model.UnavailableCouponInfo, 0),
	}

	for _, dc := range codes {
		if _, ok := applied[dc.ID]; ok {
			continue
		}
		title := ""
		if dc.Title != nil {
			title = *dc.Title
		}

		if err := s.validateCouponRules(ctx, dc, req); err != nil {
			resp.NotApplicable = append(resp.NotApplicable, model.UnavailableCouponInfo{
				ID:     dc.ID,
				Code:   dc.Code,
				Title:  title,
				Reason: err.Error(),
			})
			continue
		}

		eligible := s.eligibleItemIDs(ctx, dc, req)
		strategy := discountCodeStrategy.GetDiscountStrategy(dc.DiscountType)
		var potential int64
		if strategy != nil {
			merch, ship, calcErr := strategy.Calculate(ctx, dc, req, eligible)
			if calcErr == nil {
				potential = merch + ship
			}
		}
		if potential <= 0 {
			resp.NotApplicable = append(resp.NotApplicable, model.UnavailableCouponInfo{
				ID:     dc.ID,
				Code:   dc.Code,
				Title:  title,
				Reason: promoErrors.COUPON_NOT_APPLICABLE_MSG,
			})
			continue
		}

		canCombine := false
		if dc.CanCombineWithOtherDiscounts != nil {
			canCombine = *dc.CanCombineWithOtherDiscounts
		}
		info := model.AvailableCouponInfo{
			ID:                           dc.ID,
			Code:                         dc.Code,
			Title:                        title,
			DiscountType:                 string(dc.DiscountType),
			Value:                        dc.Value,
			MaxDiscountAmountCents:       dc.MaxDiscountAmountCents,
			PotentialDiscount:            potential,
			MinPurchaseAmountCents:       dc.MinPurchaseAmountCents,
			CanCombineWithOtherDiscounts: canCombine,
		}
		if dc.StartsAt != nil {
			info.StartsAt = dc.StartsAt.UTC().Format(time.RFC3339)
		}
		if dc.EndsAt != nil {
			formatted := dc.EndsAt.UTC().Format(time.RFC3339)
			info.EndsAt = &formatted
		}
		resp.Applicable = append(resp.Applicable, info)
	}

	return resp, nil
}

// EnsureCouponsValidAtCheckout re-runs rule + calculator checks for every applied code.
// Must be called inside the create-order transaction before usage rows are written.
func (s *CouponApplyServiceImpl) EnsureCouponsValidAtCheckout(
	ctx context.Context,
	req *model.CouponCartRequest,
) error {
	if req == nil || len(req.AppliedDiscountCodeIDs) == 0 {
		return nil
	}

	appliedSoFar := make([]uint, 0, len(req.AppliedDiscountCodeIDs))
	for _, id := range req.AppliedDiscountCodeIDs {
		dc, err := s.discountCodeRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}

		partialReq := *req
		partialReq.AppliedDiscountCodeIDs = append([]uint(nil), appliedSoFar...)
		if err := s.validateCouponRules(ctx, dc, &partialReq); err != nil {
			return err
		}

		eligible := s.eligibleItemIDs(ctx, dc, req)
		if len(eligible) == 0 && dc.DiscountType != entity.DiscountFreeShipping {
			return promoErrors.ErrCouponNotApplicable
		}

		strategy := discountCodeStrategy.GetDiscountStrategy(dc.DiscountType)
		if strategy == nil {
			return promoErrors.ErrInvalidCoupon
		}
		merch, ship, err := strategy.Calculate(ctx, dc, req, eligible)
		if err != nil {
			return err
		}
		if merch+ship <= 0 {
			return promoErrors.ErrCouponNotApplicable
		}
		appliedSoFar = append(appliedSoFar, dc.ID)
	}
	return nil
}

func (s *CouponApplyServiceImpl) validateCouponRules(
	ctx context.Context,
	dc *entity.DiscountCode,
	req *model.CouponCartRequest,
) error {
	if dc.IsActive == nil || !*dc.IsActive {
		return promoErrors.ErrInvalidCoupon
	}
	if dc.SellerID != req.SellerID {
		return promoErrors.ErrInvalidCoupon
	}

	now := time.Now().UTC()
	if dc.StartsAt != nil && now.Before(dc.StartsAt.UTC()) {
		return promoErrors.ErrCouponNotStarted
	}
	if dc.EndsAt != nil && now.After(dc.EndsAt.UTC()) {
		return promoErrors.ErrCouponExpired
	}

	if dc.UsageLimitTotal != nil && dc.CurrentUsageCount >= *dc.UsageLimitTotal {
		return promoErrors.ErrCouponUsageLimitReached
	}

	if req.CustomerID != nil && dc.UsageLimitPerCustomer != nil {
		windowStart := usageWindowStart(dc, now)
		count, err := s.usageRepo.CountByUserInWindow(ctx, dc.ID, *req.CustomerID, windowStart)
		if err != nil {
			return err
		}
		if int(count) >= *dc.UsageLimitPerCustomer {
			return promoErrors.ErrCouponAlreadyUsed
		}
	}

	switch dc.CustomerEligibility {
	case entity.EligibleNewCustomers:
		if !req.IsFirstOrder {
			return promoErrors.ErrCouponNotEligible
		}
	case entity.EligibleSpecificSegment:
		return promoErrors.ErrCouponNotEligible // stub until segment engine
	}

	eligible := s.eligibleItemIDs(ctx, dc, req)
	eligibleQty := 0
	var eligibleSubtotal int64
	set := map[string]struct{}{}
	for _, id := range eligible {
		set[id] = struct{}{}
	}
	for _, item := range req.Items {
		if _, ok := set[item.ItemID]; ok {
			eligibleQty += item.Quantity
			eligibleSubtotal += item.TotalCents
		}
	}

	if dc.MinPurchaseAmountCents != nil && eligibleSubtotal < *dc.MinPurchaseAmountCents {
		return promoErrors.ErrCouponMinPurchaseNotMet
	}
	if dc.MinQuantity != nil && eligibleQty < *dc.MinQuantity {
		return promoErrors.ErrCouponMinQuantityNotMet
	}

	if !req.PromotionsAllowCoupons {
		return promoErrors.ErrCouponCannotCombine
	}

	if len(req.AppliedDiscountCodeIDs) > 0 {
		canCombine := dc.CanCombineWithOtherDiscounts != nil && *dc.CanCombineWithOtherDiscounts
		if !canCombine {
			return promoErrors.ErrCouponCannotCombine
		}
		for _, id := range req.AppliedDiscountCodeIDs {
			existing, err := s.discountCodeRepo.FindByID(ctx, id)
			if err != nil {
				continue
			}
			existingCombine := existing.CanCombineWithOtherDiscounts != nil &&
				*existing.CanCombineWithOtherDiscounts
			if !existingCombine {
				return promoErrors.ErrCouponCannotCombine
			}
		}
	}

	return nil
}

func (s *CouponApplyServiceImpl) eligibleItemIDs(
	ctx context.Context,
	dc *entity.DiscountCode,
	req *model.CouponCartRequest,
) []string {
	ids := make([]string, 0, len(req.Items))
	switch dc.AppliesTo {
	case entity.ScopeAllProducts:
		for _, item := range req.Items {
			ids = append(ids, item.ItemID)
		}
		return ids
	case entity.ScopeSpecificProducts:
		productIDs := uniqueProductIDs(req.Items)
		if len(productIDs) == 0 {
			return nil
		}
		// Filter by cart product IDs so large scopes never silently truncate eligibility.
		products, _, err := s.productScopeRepo.GetProducts(ctx, dc.ID, productIDs, 0, len(productIDs))
		if err != nil {
			log.WarnWithContext(ctx, "failed to load discount code products: "+err.Error())
			return nil
		}
		allowed := map[uint]struct{}{}
		for _, p := range products {
			allowed[p.ProductID] = struct{}{}
		}
		for _, item := range req.Items {
			if _, ok := allowed[item.ProductID]; ok {
				ids = append(ids, item.ItemID)
			}
		}
	case entity.ScopeSpecficVariant:
		variantIDs := uniqueVariantIDs(req.Items)
		if len(variantIDs) == 0 {
			return nil
		}
		variants, _, err := s.variantScopeRepo.GetVariants(ctx, dc.ID, variantIDs, 0, len(variantIDs))
		if err != nil {
			return nil
		}
		allowed := map[uint]struct{}{}
		for _, v := range variants {
			if v.VariantID != nil {
				allowed[*v.VariantID] = struct{}{}
			}
		}
		for _, item := range req.Items {
			if item.VariantID != nil {
				if _, ok := allowed[*item.VariantID]; ok {
					ids = append(ids, item.ItemID)
				}
			}
		}
	case entity.ScopeSpecificCategories:
		categoryIDs := uniqueCategoryIDs(req.Items)
		if len(categoryIDs) == 0 {
			return nil
		}
		categories, _, err := s.categoryScopeRepo.GetCategories(ctx, dc.ID, categoryIDs, 0, len(categoryIDs))
		if err != nil {
			return nil
		}
		allowed := map[uint]struct{}{}
		for _, c := range categories {
			allowed[c.CategoryID] = struct{}{}
		}
		for _, item := range req.Items {
			if _, ok := allowed[item.CategoryID]; ok {
				ids = append(ids, item.ItemID)
			}
		}
	case entity.ScopeSpecificCollections:
		// Collection membership resolved in later phases; treat empty as no eligible items
		_ = s.collectionScopeRepo
		return nil
	}
	return ids
}

func uniqueProductIDs(items []model.CartItem) []uint {
	seen := map[uint]struct{}{}
	out := make([]uint, 0, len(items))
	for _, item := range items {
		if item.ProductID == 0 {
			continue
		}
		if _, ok := seen[item.ProductID]; ok {
			continue
		}
		seen[item.ProductID] = struct{}{}
		out = append(out, item.ProductID)
	}
	return out
}

func uniqueVariantIDs(items []model.CartItem) []uint {
	seen := map[uint]struct{}{}
	out := make([]uint, 0, len(items))
	for _, item := range items {
		if item.VariantID == nil || *item.VariantID == 0 {
			continue
		}
		id := *item.VariantID
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func uniqueCategoryIDs(items []model.CartItem) []uint {
	seen := map[uint]struct{}{}
	out := make([]uint, 0, len(items))
	for _, item := range items {
		if item.CategoryID == 0 {
			continue
		}
		if _, ok := seen[item.CategoryID]; ok {
			continue
		}
		seen[item.CategoryID] = struct{}{}
		out = append(out, item.CategoryID)
	}
	return out
}

func usageWindowStart(dc *entity.DiscountCode, now time.Time) *time.Time {
	if dc.UsageResetTimeType == entity.ResetTimeTypeNone || dc.UsageResetAmount == nil {
		return nil
	}
	amount := *dc.UsageResetAmount
	var start time.Time
	switch dc.UsageResetTimeType {
	case entity.ResetTimeTypeDay:
		start = now.AddDate(0, 0, -amount)
	case entity.ResetTimeTypeWeek:
		start = now.AddDate(0, 0, -7*amount)
	case entity.ResetTimeTypeMonth:
		start = now.AddDate(0, -amount, 0)
	case entity.ResetTimeTypeYear:
		start = now.AddDate(-amount, 0, 0)
	default:
		return nil
	}
	return &start
}

// RecordCouponUsages inserts discount_code_usage rows and increments current_usage_count.
// Must run inside the order create transaction. Failed atomic increment aborts checkout.
func (s *CouponApplyServiceImpl) RecordCouponUsages(
	ctx context.Context,
	orderID, userID uint,
	records []model.CouponUsageRecord,
) error {
	if len(records) == 0 {
		return nil
	}

	now := time.Now().UTC()
	for _, rec := range records {
		dc, err := s.discountCodeRepo.FindByID(ctx, rec.DiscountCodeID)
		if err != nil {
			return err
		}

		usage := &entity.DiscountCodeUsage{
			DiscountCodeID:      rec.DiscountCodeID,
			UserID:              userID,
			OrderID:             orderID,
			DiscountAmountCents: rec.DiscountAmountCents,
			OriginalAmountCents: rec.OriginalAmountCents,
			UsedAt:              now,
			Metadata:            db.JSONMap{},
		}
		if err := s.usageRepo.Create(ctx, usage); err != nil {
			log.ErrorWithContext(ctx, "Failed to record discount code usage", err)
			return err
		}

		if dc.UsageLimitTotal != nil {
			ok, err := s.discountCodeRepo.IncrementUsageAtomically(ctx, dc.ID, *dc.UsageLimitTotal)
			if err != nil {
				return err
			}
			if !ok {
				return promoErrors.ErrCouponUsageLimitReached
			}
		} else {
			if err := s.discountCodeRepo.IncrementUsage(ctx, dc.ID); err != nil {
				return err
			}
		}
	}
	return nil
}
