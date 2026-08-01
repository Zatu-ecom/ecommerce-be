package service

import (
	"context"
	"strconv"
	"strings"

	"ecommerce-be/common/log"
	"ecommerce-be/order/entity"
	orderError "ecommerce-be/order/error"
	"ecommerce-be/order/factory"
	"ecommerce-be/order/model"
	"ecommerce-be/order/repository"

	inventoryModel "ecommerce-be/inventory/model"
	inventoryService "ecommerce-be/inventory/service"

	productModel "ecommerce-be/product/model"
	productVariantService "ecommerce-be/product/service"

	promotionModel "ecommerce-be/promotion/model"
	promotionService "ecommerce-be/promotion/service"

	userModel "ecommerce-be/user/model"
)

// cartOperations holds shared methods used by both CartServiceImpl and GuestCartServiceImpl.
// Following ISP (Interface Segregation) — shared logic extracted to avoid duplication.
type cartOperations struct {
	cartRepo        repository.CartRepository
	promotionSvc    promotionService.PromotionService
	inventorySvc    inventoryService.InventoryQueryService
	variantQuerySvc productVariantService.VariantQueryService
}

// loadCartMutationState loads existing cart items and builds state maps for mutation.
func (o *cartOperations) loadCartMutationState(
	ctx context.Context,
	cart *entity.Cart,
) (
	map[uint]*entity.CartItem,
	map[uint]int,
	error,
) {
	existingItems := []entity.CartItem{}
	if cart != nil {
		items, err := o.cartRepo.FindItemsByCartID(ctx, cart.ID)
		if err != nil {
			return nil, nil, err
		}
		existingItems = items
	}

	existingItemByVariant := make(map[uint]*entity.CartItem, len(existingItems))
	finalQuantityByVariant := make(map[uint]int, len(existingItems))
	for i := range existingItems {
		item := &existingItems[i]
		existingItemByVariant[item.VariantID] = item
		finalQuantityByVariant[item.VariantID] = item.Quantity
	}

	return existingItemByVariant, finalQuantityByVariant, nil
}

// validateInventoryForFinalQuantities checks stock availability for variants needing validation.
func (o *cartOperations) validateInventoryForFinalQuantities(
	ctx context.Context,
	sellerID uint,
	variantsNeedingValidation map[uint]struct{},
	finalQuantityByVariant map[uint]int,
) error {
	if len(variantsNeedingValidation) == 0 {
		return nil
	}

	variantIDs := make([]uint, 0, len(variantsNeedingValidation))
	for variantID := range variantsNeedingValidation {
		variantIDs = append(variantIDs, variantID)
	}

	invReq := inventoryModel.TotalAvailableQuantityRequest{
		VariantIDs: variantIDs,
	}
	invRes, err := o.inventorySvc.GetTotalAvailableQuantities(ctx, invReq, sellerID)
	if err != nil {
		return err
	}

	availableByVariant := make(map[uint]int, len(invRes.Items))
	for _, item := range invRes.Items {
		availableByVariant[item.VariantID] = item.TotalAvailable
	}

	for variantID := range variantsNeedingValidation {
		available, exists := availableByVariant[variantID]
		if !exists {
			return orderError.ErrVariantNotFound
		}
		if finalQuantityByVariant[variantID] > available {
			return orderError.ErrInsufficientStock(available)
		}
	}
	return nil
}

// fetchVariantMap retrieves variant details for all items in a cart.
func (o *cartOperations) fetchVariantMap(
	ctx context.Context,
	items []entity.CartItem,
	sellerID uint,
) (map[uint]productModel.VariantDetailResponse, error) {
	variantMap := make(map[uint]productModel.VariantDetailResponse)
	if len(items) == 0 {
		return variantMap, nil
	}

	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = strconv.Itoa(int(item.VariantID))
	}

	listReq := &productModel.ListVariantsRequest{
		IDs:      strings.Join(ids, ","),
		PageSize: len(items),
	}

	listResp, err := o.variantQuerySvc.ListVariants(ctx, listReq, &sellerID, nil, nil)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to fetch variant information using ListVariants", err)
		return nil, err
	}

	for _, v := range listResp.Variants {
		variantMap[v.ID] = v
	}
	return variantMap, nil
}

// buildCartResponseWithItems constructs the full CartResponse with promotions and variant info.
func (o *cartOperations) buildCartResponseWithItems(
	ctx context.Context,
	sellerID uint,
	cart *entity.Cart,
	items []entity.CartItem,
	currencyMap *userModel.CurrencyResponse,
	userID uint,
) (*model.CartResponse, error) {
	variantMap, err := o.fetchVariantMap(ctx, items, sellerID)
	if err != nil {
		return nil, err
	}

	promoReq, err := o.buildPromotionRequest(ctx, sellerID, userID, items, variantMap)
	if err != nil {
		return nil, err
	}

	log.InfoWithContext(ctx, "Calling Promotion Service for Cart validation")
	promoSummary, err := o.promotionSvc.ApplyPromotionsToCart(ctx, promoReq)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to apply promotions", err)
		return nil, orderError.ErrPromotionServiceUnavailable(err)
	}

	return factory.BuildCartResponse(
		cart,
		items,
		promoSummary,
		nil,
		nil,
		nil,
		currencyMap,
		variantMap,
	), nil
}

// buildEmptyCartResponse creates a zeroed-out CartResponse for empty/cleared carts.
// userID is a pointer because guest carts have no user ID (nil).
func (o *cartOperations) buildEmptyCartResponse(
	userID *uint,
	currencyMap *userModel.CurrencyResponse,
) *model.CartResponse {
	return &model.CartResponse{
		CartBase: model.CartBase{
			ID:     0,
			UserID: userID,
			Currency: model.CurrencyInfo{
				Code:          currencyMap.Code,
				Symbol:        currencyMap.Symbol,
				DecimalDigits: currencyMap.DecimalDigits,
			},
			Metadata: map[string]any{},
		},
		Items:               []model.CartItemWithPricingResponse{},
		AppliedPromotions:   []model.AppliedPromotionInfo{},
		AppliedCoupons:      []model.AppliedCouponInfo{},
		Summary:             model.CartSummary{},
		AvailablePromotions: []model.AvailablePromotionInfo{},
	}
}

// applyFinalQuantitiesToCart persists the final quantities (insert, update, delete items).
func (o *cartOperations) applyFinalQuantitiesToCart(
	ctx context.Context,
	cartID uint,
	existingItemByVariant map[uint]*entity.CartItem,
	finalQuantityByVariant map[uint]int,
) error {
	for variantID, finalQty := range finalQuantityByVariant {
		existingItem := existingItemByVariant[variantID]
		switch {
		case finalQty <= 0:
			if existingItem != nil {
				if err := o.cartRepo.DeleteItem(ctx, existingItem.ID); err != nil {
					return err
				}
			}
		case existingItem != nil:
			existingItem.Quantity = finalQty
			if err := o.cartRepo.UpdateItem(ctx, existingItem); err != nil {
				return err
			}
		default:
			if err := o.cartRepo.AddItem(ctx, &entity.CartItem{
				CartID:    cartID,
				VariantID: variantID,
				Quantity:  finalQty,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

// buildPromotionRequest constructs a CartValidationRequest from cart items and variant data.
func (o *cartOperations) buildPromotionRequest(
	ctx context.Context,
	sellerID, userID uint,
	items []entity.CartItem,
	variantMap map[uint]productModel.VariantDetailResponse,
) (*promotionModel.CartValidationRequest, error) {
	promoReq := &promotionModel.CartValidationRequest{
		SellerID:      sellerID,
		CustomerID:    &userID,
		IsFirstOrder:  true, // default; overridden per service if past orders exist
		Items:         make([]promotionModel.CartItem, len(items)),
		SubtotalCents: 0,
		ShippingCents: 5000, // TODO [FULFILLMENT]: Replace with Fulfillment Service quote
	}

	for i, item := range items {
		variant, exists := variantMap[item.VariantID]
		if !exists {
			log.WarnWithContext(
				ctx,
				"Variant information not found for variant ID: "+strconv.Itoa(int(item.VariantID)),
			)
			return nil, orderError.ErrVariantNotFound
		}

		variantPriceCents := int64(variant.Price * 100)
		lineTotal := variantPriceCents * int64(item.Quantity)
		promoReq.SubtotalCents += lineTotal

		promoReq.Items[i] = promotionModel.CartItem{
			ItemID:     strconv.Itoa(int(item.ID)),
			VariantID:  &item.VariantID,
			ProductID:  variant.ProductID,
			CategoryID: variant.Product.CategoryID,
			Quantity:   item.Quantity,
			PriceCents: variantPriceCents,
			TotalCents: lineTotal,
		}
	}
	return promoReq, nil
}

// ============================================================================
// Package-level helper functions (used by both services)
// ============================================================================

// hasPositiveQuantity returns true if any item in the list has a positive quantity.
func hasPositiveQuantity(items []model.AddCartItemDetail) bool {
	for _, item := range items {
		if item.Quantity != nil && *item.Quantity > 0 {
			return true
		}
	}
	return false
}

// applyRequestedQuantities applies request items to the final quantity map.
// Returns set of variant IDs that need inventory validation.
func applyRequestedQuantities(
	reqItems []model.AddCartItemDetail,
	finalQuantityByVariant map[uint]int,
) map[uint]struct{} {
	variantsNeedingValidation := make(map[uint]struct{})
	for _, item := range reqItems {
		quantity := 0
		if item.Quantity != nil {
			quantity = *item.Quantity
		}

		if quantity == 0 {
			finalQuantityByVariant[item.VariantID] = 0
			continue
		}

		variantsNeedingValidation[item.VariantID] = struct{}{}
		finalQuantityByVariant[item.VariantID] += quantity
	}

	return variantsNeedingValidation
}
