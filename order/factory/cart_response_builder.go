package factory

import (
	"fmt"
	"strconv"

	"ecommerce-be/order/entity"
	"ecommerce-be/order/model"
	productModel "ecommerce-be/product/model"
	promotionModel "ecommerce-be/promotion/model"
	userModel "ecommerce-be/user/model"
)

const defaultFallbackUnitPriceCents int64 = 100000

// BuildCartResponse converts cart entities and promotion/coupon summaries into CartResponse.
func BuildCartResponse(
	cart *entity.Cart,
	items []entity.CartItem,
	promo *promotionModel.AppliedPromotionSummary,
	coupon *promotionModel.AppliedCouponSummary,
	appliedRows []entity.CartAppliedCoupon,
	available *promotionModel.AvailableCouponsResponse,
	currencyMap *userModel.CurrencyResponse,
	variantMap map[uint]productModel.VariantDetailResponse,
) *model.CartResponse {
	response := &model.CartResponse{
		CartBase:            buildCartBase(cart, currencyMap),
		Summary:             buildCartSummary(len(items), promo, coupon, currencyMap),
		Items:               make([]model.CartItemWithPricingResponse, len(items)),
		AppliedPromotions:   buildAppliedPromotions(promo, currencyMap),
		AppliedCoupons:      buildAppliedCoupons(coupon, appliedRows, currencyMap),
		AvailablePromotions: buildAvailablePromotions(promo, currencyMap),
		AvailableCoupons:    buildAvailableCoupons(available, currencyMap),
	}

	itemPromoMap := buildItemPromotionMap(promo)
	for i, item := range items {
		response.Summary.ItemCount += item.Quantity
		itemResp, err := buildCartItemResponse(item, itemPromoMap, currencyMap, variantMap)
		if err != nil {
			return nil
		}
		response.Items[i] = itemResp
	}

	attachSavingsIfAny(&response.Summary)
	return response
}

func buildAppliedPromotions(
	promo *promotionModel.AppliedPromotionSummary,
	currencyMap *userModel.CurrencyResponse,
) []model.AppliedPromotionInfo {
	if promo == nil || len(promo.AppliedPromotions) == 0 {
		return nil
	}

	applied := make([]model.AppliedPromotionInfo, 0, len(promo.AppliedPromotions))
	for _, p := range promo.AppliedPromotions {
		if p.Promotion == nil {
			continue
		}

		info := model.AppliedPromotionInfo{
			PromotionID:      p.Promotion.ID,
			Name:             p.Promotion.Name,
			Type:             string(p.Promotion.PromotionType),
			Discount:         p.DiscountCents,
			ShippingDiscount: p.ShippingDiscount,
		}
		if p.DiscountCents > 0 {
			info.DiscountFormatted = formatCurrencyWithSymbol(
				p.DiscountCents,
				currencyMap.Symbol,
				currencyMap.DecimalDigits,
			)
		}
		if p.ShippingDiscount > 0 {
			info.ShippingDiscountFormatted = formatCurrencyWithSymbol(
				p.ShippingDiscount,
				currencyMap.Symbol,
				currencyMap.DecimalDigits,
			)
		}
		applied = append(applied, info)
	}
	return applied
}

func buildAvailablePromotions(
	promo *promotionModel.AppliedPromotionSummary,
	currencyMap *userModel.CurrencyResponse,
) []model.AvailablePromotionInfo {
	if promo == nil || len(promo.SkippedPromotions) == 0 {
		return nil
	}

	available := make([]model.AvailablePromotionInfo, 0)
	for _, skipped := range promo.SkippedPromotions {
		if skipped.Promotion == nil {
			continue
		}

		info := model.AvailablePromotionInfo{
			ID:               skipped.Promotion.ID,
			Name:             skipped.Promotion.Name,
			Type:             string(skipped.Promotion.PromotionType),
			Reason:           skipped.Reason,
			Requirement:      skipped.Requirement,
			PotentialSavings: skipped.PotentialSavings,
		}
		if skipped.PotentialSavings > 0 {
			info.PotentialSavingsFormatted = formatCurrencyWithSymbol(
				skipped.PotentialSavings,
				currencyMap.Symbol,
				currencyMap.DecimalDigits,
			)
		}
		available = append(available, info)
	}
	return available
}

func buildCartBase(
	cart *entity.Cart,
	currencyMap *userModel.CurrencyResponse,
) model.CartBase {
	return model.CartBase{
		ID:     cart.ID,
		UserID: cart.UserID,
		Currency: model.CurrencyInfo{
			Code:          currencyMap.Code,
			Symbol:        currencyMap.Symbol,
			DecimalDigits: currencyMap.DecimalDigits,
		},
		Metadata: cart.Metadata,
	}
}

func buildCartSummary(
	uniqueItems int,
	promo *promotionModel.AppliedPromotionSummary,
	coupon *promotionModel.AppliedCouponSummary,
	currencyMap *userModel.CurrencyResponse,
) model.CartSummary {
	couponDiscount := int64(0)
	couponCount := 0
	shippingDiscount := int64(0)
	if coupon != nil {
		couponDiscount = coupon.TotalDiscountCents
		couponCount = len(coupon.AppliedCoupons)
		shippingDiscount = coupon.ShippingDiscount
	}
	if promo != nil {
		shippingDiscount += promo.ShippingDiscount
	}

	totalDiscount := promo.TotalDiscountCents + couponDiscount
	afterDiscount := promo.FinalSubtotal - couponDiscount
	if afterDiscount < 0 {
		afterDiscount = 0
	}
	// Shipping is not yet in total; free-shipping coupons reduce shippingDiscount for display
	_ = shippingDiscount
	total := afterDiscount

	return model.CartSummary{
		ItemCount:   0,
		UniqueItems: uniqueItems,
		Subtotal:    promo.OriginalSubtotal,
		SubtotalFormatted: formatCurrencyWithSymbol(
			promo.OriginalSubtotal,
			currencyMap.Symbol,
			currencyMap.DecimalDigits,
		),
		PromotionCount:    len(promo.AppliedPromotions),
		PromotionDiscount: promo.TotalDiscountCents,
		PromotionDiscountFormatted: formatCurrencyWithSymbol(
			promo.TotalDiscountCents,
			currencyMap.Symbol,
			currencyMap.DecimalDigits,
		),
		CouponCount:    couponCount,
		CouponDiscount: couponDiscount,
		CouponDiscountFormatted: formatCurrencyWithSymbol(
			couponDiscount,
			currencyMap.Symbol,
			currencyMap.DecimalDigits,
		),
		TotalDiscount: totalDiscount,
		TotalDiscountFormatted: formatCurrencyWithSymbol(
			totalDiscount,
			currencyMap.Symbol,
			currencyMap.DecimalDigits,
		),
		AfterDiscount: afterDiscount,
		AfterDiscountFormatted: formatCurrencyWithSymbol(
			afterDiscount,
			currencyMap.Symbol,
			currencyMap.DecimalDigits,
		),
		Total: total,
		TotalFormatted: formatCurrencyWithSymbol(
			total,
			currencyMap.Symbol,
			currencyMap.DecimalDigits,
		),
	}
}

func buildAppliedCoupons(
	coupon *promotionModel.AppliedCouponSummary,
	appliedRows []entity.CartAppliedCoupon,
	currencyMap *userModel.CurrencyResponse,
) []model.AppliedCouponInfo {
	if coupon == nil || len(coupon.AppliedCoupons) == 0 {
		return []model.AppliedCouponInfo{}
	}

	rowByCodeID := map[uint]uint{}
	for _, row := range appliedRows {
		rowByCodeID[row.DiscountCodeID] = row.ID
	}

	out := make([]model.AppliedCouponInfo, 0, len(coupon.AppliedCoupons))
	for _, c := range coupon.AppliedCoupons {
		if c.DiscountCode == nil {
			continue
		}
		title := ""
		if c.DiscountCode.Title != nil {
			title = *c.DiscountCode.Title
		}
		info := model.AppliedCouponInfo{
			ID:               rowByCodeID[c.DiscountCode.ID],
			DiscountCodeID:   c.DiscountCode.ID,
			Code:             c.DiscountCode.Code,
			Title:            title,
			DiscountType:     string(c.DiscountCode.DiscountType),
			Discount:         c.DiscountCents,
			ShippingDiscount: c.ShippingDiscount,
		}
		info.DiscountFormatted = formatCurrencyWithSymbol(
			c.DiscountCents, currencyMap.Symbol, currencyMap.DecimalDigits,
		)
		info.ShippingDiscountFormatted = formatCurrencyWithSymbol(
			c.ShippingDiscount, currencyMap.Symbol, currencyMap.DecimalDigits,
		)
		out = append(out, info)
	}
	return out
}

func buildAvailableCoupons(
	available *promotionModel.AvailableCouponsResponse,
	currencyMap *userModel.CurrencyResponse,
) *model.CartAvailableCouponsResponse {
	if available == nil {
		return nil
	}
	out := &model.CartAvailableCouponsResponse{
		Applicable:    make([]model.CartAvailableCouponInfo, 0, len(available.Applicable)),
		NotApplicable: make([]model.CartUnavailableCouponInfo, 0, len(available.NotApplicable)),
	}
	for _, a := range available.Applicable {
		info := model.CartAvailableCouponInfo{
			ID:                           a.ID,
			Code:                         a.Code,
			Title:                        a.Title,
			DiscountType:                 a.DiscountType,
			Value:                        a.Value,
			MaxDiscountAmountCents:       a.MaxDiscountAmountCents,
			PotentialDiscount:            a.PotentialDiscount,
			MinPurchaseAmountCents:       a.MinPurchaseAmountCents,
			CanCombineWithOtherDiscounts: a.CanCombineWithOtherDiscounts,
			StartsAt:                     a.StartsAt,
			EndsAt:                       a.EndsAt,
		}
		info.PotentialDiscountFormatted = formatCurrencyWithSymbol(
			a.PotentialDiscount, currencyMap.Symbol, currencyMap.DecimalDigits,
		)
		out.Applicable = append(out.Applicable, info)
	}
	for _, n := range available.NotApplicable {
		out.NotApplicable = append(out.NotApplicable, model.CartUnavailableCouponInfo{
			ID: n.ID, Code: n.Code, Title: n.Title, Reason: n.Reason,
		})
	}
	return out
}

func buildItemPromotionMap(
	promo *promotionModel.AppliedPromotionSummary,
) map[string]promotionModel.CartItemSummary {
	itemPromoMap := make(map[string]promotionModel.CartItemSummary, len(promo.Items))
	for _, summaryItem := range promo.Items {
		itemPromoMap[summaryItem.ItemID] = summaryItem
	}
	return itemPromoMap
}

func buildCartItemResponse(
	item entity.CartItem,
	itemPromoMap map[string]promotionModel.CartItemSummary,
	currencyMap *userModel.CurrencyResponse,
	variantMap map[uint]productModel.VariantDetailResponse,
) (model.CartItemWithPricingResponse, error) {
	itemIDStr := strconv.Itoa(int(item.ID))
	summaryItem, exists := itemPromoMap[itemIDStr]

	unitPrice := defaultFallbackUnitPriceCents
	lineTotal := unitPrice * int64(item.Quantity)
	discountedLineTotal := lineTotal
	totalItemDiscount := int64(0)
	appliedPromos := make([]model.ItemAppliedPromotionInfo, 0)

	if exists {
		unitPrice = summaryItem.OriginalUnitPriceCents
		lineTotal = unitPrice * int64(item.Quantity)
		discountedLineTotal = summaryItem.FinalPriceCents
		totalItemDiscount = summaryItem.TotalDiscountCents
		appliedPromos = buildAppliedPromotionInfos(summaryItem, currencyMap)
	}

	variant, ok := variantMap[item.VariantID]
	if !ok {
		return model.CartItemWithPricingResponse{}, fmt.Errorf(
			"variant %d not found in variant map",
			item.VariantID,
		)
	}

	var options []model.VariantOptionInfo
	for _, opt := range variant.SelectedOptions {
		options = append(options, model.VariantOptionInfo{
			Name:  opt.OptionDisplayName,
			Value: opt.ValueDisplayName,
		})
	}

	variantInfo := model.VariantInfo{
		ID:            item.VariantID,
		SKU:           variant.SKU,
		Images:        variantMediaURLs(variant.Media),
		ImageFileID:   primaryVariantMediaFileID(variant.Media),
		AllowPurchase: variant.AllowPurchase,
		Product: model.ProductBasicInfo{
			ID:   variant.Product.ID,
			Name: variant.Product.Name,
		},
		Options: options,
	}

	return model.CartItemWithPricingResponse{
		CartItemBase: model.CartItemBase{
			ID:        item.ID,
			CartID:    item.CartID,
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
			Variant:   variantInfo,
		},
		UnitPrice:              unitPrice,
		LineTotal:              lineTotal,
		TotalPromotionDiscount: totalItemDiscount,
		DiscountedLineTotal:    discountedLineTotal,
		AppliedPromotions:      appliedPromos,
	}, nil
}

func buildAppliedPromotionInfos(
	summaryItem promotionModel.CartItemSummary,
	currencyMap *userModel.CurrencyResponse,
) []model.ItemAppliedPromotionInfo {
	appliedPromos := make([]model.ItemAppliedPromotionInfo, 0, len(summaryItem.AppliedPromotions))
	for _, p := range summaryItem.AppliedPromotions {
		appliedPromos = append(appliedPromos, model.ItemAppliedPromotionInfo{
			PromotionID: p.PromotionID,
			Name:        p.PromotionName,
			Type:        "applied_promotion", // Generic type for now
			Discount:    p.DiscountCents,
			DiscountFormatted: formatCurrencyWithSymbol(
				p.DiscountCents,
				currencyMap.Symbol,
				currencyMap.DecimalDigits,
			),
		})
	}
	return appliedPromos
}

func attachSavingsIfAny(summary *model.CartSummary) {
	if summary.TotalDiscount <= 0 || summary.Subtotal <= 0 {
		return
	}

	percentage := float64(summary.TotalDiscount) / float64(summary.Subtotal) * 100
	summary.Savings = &model.SavingsInfo{
		Amount:     summary.TotalDiscount,
		Percentage: percentage,
		Message: fmt.Sprintf(
			"You're saving %s (%.0f%% off)!",
			summary.TotalDiscountFormatted,
			percentage,
		),
	}
}

func formatCurrencyWithSymbol(cents int64, symbol string, decimalDigits int) string {
	formatStr := fmt.Sprintf("%%s%%.%df", decimalDigits)
	return fmt.Sprintf(formatStr, symbol, float64(cents)/100.0)
}

func variantMediaURLs(media []productModel.VariantMediaResponse) []string {
	urls := make([]string, 0, len(media))
	for _, m := range media {
		if m.URL != "" {
			urls = append(urls, m.URL)
		}
	}
	return urls
}

func primaryVariantMediaFileID(media []productModel.VariantMediaResponse) *string {
	for _, m := range media {
		if m.IsPrimary && m.URL != "" {
			return &m.FileID
		}
	}
	for _, m := range media {
		if m.URL != "" {
			return &m.FileID
		}
	}
	return nil
}
