package factory

import (
	"fmt"
	"strconv"

	commonModel "ecommerce-be/common/model"
	"ecommerce-be/order/entity"
	"ecommerce-be/order/model"
	productModel "ecommerce-be/product/model"
	promotionModel "ecommerce-be/promotion/model"
	userFactory "ecommerce-be/user/factory"
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
	ccy := userFactory.ToCurrencyInfo(currencyMap)
	response := &model.CartResponse{
		CartBase:            buildCartBase(cart, ccy),
		Summary:             buildCartSummary(len(items), promo, coupon, ccy),
		Items:               make([]model.CartItemWithPricingResponse, len(items)),
		AppliedPromotions:   buildAppliedPromotions(promo, ccy),
		AppliedCoupons:      buildAppliedCoupons(coupon, appliedRows, ccy),
		AvailablePromotions: buildAvailablePromotions(promo, ccy),
		AvailableCoupons:    buildAvailableCoupons(available, ccy),
	}

	itemPromoMap := buildItemPromotionMap(promo)
	for i, item := range items {
		response.Summary.ItemCount += item.Quantity
		itemResp, err := buildCartItemResponse(item, itemPromoMap, ccy, variantMap)
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
	ccy commonModel.CurrencyInfo,
) []model.AppliedPromotionInfo {
	if promo == nil || len(promo.AppliedPromotions) == 0 {
		return nil
	}

	applied := make([]model.AppliedPromotionInfo, 0, len(promo.AppliedPromotions))
	for _, p := range promo.AppliedPromotions {
		if p.Promotion == nil {
			continue
		}

		applied = append(applied, model.AppliedPromotionInfo{
			PromotionID:      p.Promotion.ID,
			Name:             p.Promotion.Name,
			Type:             string(p.Promotion.PromotionType),
			Discount:         commonModel.NewMoney(p.DiscountCents, ccy),
			ShippingDiscount: commonModel.NewMoney(p.ShippingDiscount, ccy),
		})
	}
	return applied
}

func buildAvailablePromotions(
	promo *promotionModel.AppliedPromotionSummary,
	ccy commonModel.CurrencyInfo,
) []model.AvailablePromotionInfo {
	if promo == nil || len(promo.SkippedPromotions) == 0 {
		return nil
	}

	available := make([]model.AvailablePromotionInfo, 0)
	for _, skipped := range promo.SkippedPromotions {
		if skipped.Promotion == nil {
			continue
		}

		available = append(available, model.AvailablePromotionInfo{
			ID:               skipped.Promotion.ID,
			Name:             skipped.Promotion.Name,
			Type:             string(skipped.Promotion.PromotionType),
			Reason:           skipped.Reason,
			Requirement:      skipped.Requirement,
			PotentialSavings: commonModel.NewMoney(skipped.PotentialSavings, ccy),
		})
	}
	return available
}

func buildCartBase(
	cart *entity.Cart,
	ccy commonModel.CurrencyInfo,
) model.CartBase {
	return model.CartBase{
		ID:       cart.ID,
		UserID:   cart.UserID,
		Currency: ccy,
		Metadata: cart.Metadata,
	}
}

func buildCartSummary(
	uniqueItems int,
	promo *promotionModel.AppliedPromotionSummary,
	coupon *promotionModel.AppliedCouponSummary,
	ccy commonModel.CurrencyInfo,
) model.CartSummary {
	couponDiscount := int64(0)
	couponCount := 0
	if coupon != nil {
		couponDiscount = coupon.TotalDiscountCents
		couponCount = len(coupon.AppliedCoupons)
	}

	totalDiscount := promo.TotalDiscountCents + couponDiscount
	afterDiscount := promo.FinalSubtotal - couponDiscount
	if afterDiscount < 0 {
		afterDiscount = 0
	}
	total := afterDiscount

	return model.CartSummary{
		ItemCount:         0,
		UniqueItems:       uniqueItems,
		Subtotal:          commonModel.NewMoney(promo.OriginalSubtotal, ccy),
		PromotionCount:    len(promo.AppliedPromotions),
		PromotionDiscount: commonModel.NewMoney(promo.TotalDiscountCents, ccy),
		CouponCount:       couponCount,
		CouponDiscount:    commonModel.NewMoney(couponDiscount, ccy),
		TotalDiscount:     commonModel.NewMoney(totalDiscount, ccy),
		AfterDiscount:     commonModel.NewMoney(afterDiscount, ccy),
		Total:             commonModel.NewMoney(total, ccy),
	}
}

func buildAppliedCoupons(
	coupon *promotionModel.AppliedCouponSummary,
	appliedRows []entity.CartAppliedCoupon,
	ccy commonModel.CurrencyInfo,
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
		out = append(out, model.AppliedCouponInfo{
			ID:               rowByCodeID[c.DiscountCode.ID],
			DiscountCodeID:   c.DiscountCode.ID,
			Code:             c.DiscountCode.Code,
			Title:            title,
			DiscountType:     string(c.DiscountCode.DiscountType),
			Discount:         commonModel.NewMoney(c.DiscountCents, ccy),
			ShippingDiscount: commonModel.NewMoney(c.ShippingDiscount, ccy),
		})
	}
	return out
}

func buildAvailableCoupons(
	available *promotionModel.AvailableCouponsResponse,
	ccy commonModel.CurrencyInfo,
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
			Value:                        commonModel.NewMoney(a.Value, ccy),
			PotentialDiscount:            commonModel.NewMoney(a.PotentialDiscount, ccy),
			CanCombineWithOtherDiscounts: a.CanCombineWithOtherDiscounts,
			StartsAt:                     a.StartsAt,
			EndsAt:                       a.EndsAt,
		}
		if a.MaxDiscountAmountCents != nil {
			m := commonModel.NewMoney(*a.MaxDiscountAmountCents, ccy)
			info.MaxDiscountAmount = &m
		}
		if a.MinPurchaseAmountCents != nil {
			m := commonModel.NewMoney(*a.MinPurchaseAmountCents, ccy)
			info.MinPurchaseAmount = &m
		}
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
	ccy commonModel.CurrencyInfo,
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
		appliedPromos = buildAppliedPromotionInfos(summaryItem, ccy)
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
		UnitPrice:              commonModel.NewMoney(unitPrice, ccy),
		LineTotal:              commonModel.NewMoney(lineTotal, ccy),
		TotalPromotionDiscount: commonModel.NewMoney(totalItemDiscount, ccy),
		DiscountedLineTotal:    commonModel.NewMoney(discountedLineTotal, ccy),
		AppliedPromotions:      appliedPromos,
	}, nil
}

func buildAppliedPromotionInfos(
	summaryItem promotionModel.CartItemSummary,
	ccy commonModel.CurrencyInfo,
) []model.ItemAppliedPromotionInfo {
	appliedPromos := make([]model.ItemAppliedPromotionInfo, 0, len(summaryItem.AppliedPromotions))
	for _, p := range summaryItem.AppliedPromotions {
		appliedPromos = append(appliedPromos, model.ItemAppliedPromotionInfo{
			PromotionID: p.PromotionID,
			Name:        p.PromotionName,
			Type:        "applied_promotion", // Generic type for now
			Discount:    commonModel.NewMoney(p.DiscountCents, ccy),
		})
	}
	return appliedPromos
}

func attachSavingsIfAny(summary *model.CartSummary) {
	if summary.TotalDiscount.AmountCents <= 0 || summary.Subtotal.AmountCents <= 0 {
		return
	}

	percentage := float64(summary.TotalDiscount.AmountCents) / float64(summary.Subtotal.AmountCents) * 100
	summary.Savings = &model.SavingsInfo{
		Amount:     summary.TotalDiscount,
		Percentage: percentage,
		Message: fmt.Sprintf(
			"You're saving %s (%.0f%% off)!",
			summary.TotalDiscount.Formatted,
			percentage,
		),
	}
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
