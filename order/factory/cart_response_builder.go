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

// BuildCartResponse converts cart entities and promotion summary into CartResponse.
func BuildCartResponse(
	cart *entity.Cart,
	items []entity.CartItem,
	promo *promotionModel.AppliedPromotionSummary,
	currencyMap *userModel.CurrencyResponse,
	variantMap map[uint]productModel.VariantDetailResponse,
) *model.CartResponse {
	response := &model.CartResponse{
		CartBase:       buildCartBase(cart, currencyMap),
		Summary:        buildCartSummary(len(items), promo, currencyMap),
		Items:          make([]model.CartItemWithPricingResponse, len(items)),
		AppliedCoupons: make([]model.AppliedCouponInfo, 0), // Not implemented yet
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
	currencyMap *userModel.CurrencyResponse,
) model.CartSummary {
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
		TotalDiscount: promo.TotalDiscountCents, // No coupons yet
		TotalDiscountFormatted: formatCurrencyWithSymbol(
			promo.TotalDiscountCents,
			currencyMap.Symbol,
			currencyMap.DecimalDigits,
		),
		AfterDiscount: promo.FinalSubtotal,
		AfterDiscountFormatted: formatCurrencyWithSymbol(
			promo.FinalSubtotal,
			currencyMap.Symbol,
			currencyMap.DecimalDigits,
		),
		Total: promo.FinalSubtotal,
		TotalFormatted: formatCurrencyWithSymbol(
			promo.FinalSubtotal,
			currencyMap.Symbol,
			currencyMap.DecimalDigits,
		),
	}
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
		discountedLineTotal = summaryItem.FinalPriceCents * int64(item.Quantity)
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
		Images:        variant.Images,
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
