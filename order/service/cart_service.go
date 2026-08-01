package service

import (
	"context"
	"strconv"

	"ecommerce-be/common/db"
	errs "ecommerce-be/common/error"
	"ecommerce-be/common/log"
	"ecommerce-be/order/entity"
	orderError "ecommerce-be/order/error"
	"ecommerce-be/order/factory"
	"ecommerce-be/order/model"
	"ecommerce-be/order/repository"

	promotionModel "ecommerce-be/promotion/model"
	promotionRepo "ecommerce-be/promotion/repository"
	promotionService "ecommerce-be/promotion/service"

	inventoryService "ecommerce-be/inventory/service"

	productModel "ecommerce-be/product/model"
	productVariantService "ecommerce-be/product/service"
	userModel "ecommerce-be/user/model"
	userService "ecommerce-be/user/service"
)

// TODO: EDD and shipping cost calculation will be added in a future PR after fulfillment service is implemented.
// For now, we are using a fixed shipping cost in the promotion engine to allow testing of shipping promotions.
type CartService interface {
	AddToCart(
		ctx context.Context,
		userID, sellerID uint,
		req model.AddCartItemRequest,
	) (*model.CartResponse, error)
	GetUserCart(
		ctx context.Context,
		userID, sellerID uint,
	) (*model.CartResponse, error)
	DeleteCart(
		ctx context.Context,
		userID, sellerID, cartID uint,
	) (*model.CartResponse, error)
	LockActiveCartForCheckout(ctx context.Context, userID uint) (*entity.Cart, error)
	UnlockCheckoutCart(ctx context.Context, cartID uint) error
	MarkCartConverted(ctx context.Context, cartID, orderID, userID uint) error
	ReactivateCartByOrderID(ctx context.Context, orderID uint) error
	ApplyCoupon(ctx context.Context, userID, sellerID uint, code string) (*model.CartResponse, error)
	RemoveCoupon(ctx context.Context, userID, sellerID uint, code string) (*model.CartResponse, error)
	RemoveAllCoupons(ctx context.Context, userID, sellerID uint) (*model.CartResponse, error)
	GetAvailableCoupons(
		ctx context.Context,
		userID, sellerID uint,
	) (*promotionModel.AvailableCouponsResponse, error)
	// RevalidateCouponsForCheckout rebuilds the coupon cart request from the locked cart
	// and fails checkout if any applied coupon is no longer valid.
	RevalidateCouponsForCheckout(
		ctx context.Context,
		userID, sellerID, cartID uint,
		appliedDiscountCodeIDs []uint,
	) error
}

type CartServiceImpl struct {
	// Embed shared cart operations (ISP — extracted to avoid duplication with GuestCartServiceImpl)
	cartOperations
	orderRepo        repository.OrderRepository
	userSvc          userService.UserService
	promotionSvc     promotionService.PromotionService
	couponApplySvc   promotionService.CouponApplyService
	discountCodeRepo promotionRepo.DiscountCodeRepository
}

func NewCartService(
	cartRepo repository.CartRepository,
	orderRepo repository.OrderRepository,
	promotionSvc promotionService.PromotionService,
	couponApplySvc promotionService.CouponApplyService,
	discountCodeRepo promotionRepo.DiscountCodeRepository,
	inventorySvc inventoryService.InventoryQueryService,
	variantQuerySvc productVariantService.VariantQueryService,
	userSvc userService.UserService,
) CartService {
	return &CartServiceImpl{
		cartOperations: cartOperations{
			cartRepo:        cartRepo,
			promotionSvc:    promotionSvc,
			inventorySvc:    inventorySvc,
			variantQuerySvc: variantQuerySvc,
		},
		orderRepo:        orderRepo,
		userSvc:          userSvc,
		promotionSvc:     promotionSvc,
		couponApplySvc:   couponApplySvc,
		discountCodeRepo: discountCodeRepo,
	}
}

func (s *CartServiceImpl) AddToCart(
	ctx context.Context,
	userID, sellerID uint,
	req model.AddCartItemRequest,
) (*model.CartResponse, error) {
	return db.WithTransactionResult(ctx, func(txCtx context.Context) (*model.CartResponse, error) {
		currencyMap, err := s.userSvc.GetPreferredCurrency(txCtx, userID, sellerID)
		if err != nil {
			return nil, err
		}

		hasPositiveQty := hasPositiveQuantity(req.Items)

		cart, err := s.getExistingOrCreateCart(txCtx, userID, hasPositiveQty)
		if err != nil {
			return nil, err
		}

		if cart == nil && !hasPositiveQty {
			return s.cartOperations.buildEmptyCartResponse(&userID, currencyMap), nil
		}

		existingItemByVariant, finalQuantityByVariant, err := s.cartOperations.loadCartMutationState(
			txCtx,
			cart,
		)
		if err != nil {
			return nil, err
		}

		variantsNeedingValidation := applyRequestedQuantities(req.Items, finalQuantityByVariant)

		if err := s.cartOperations.validateInventoryForFinalQuantities(
			txCtx,
			sellerID,
			variantsNeedingValidation,
			finalQuantityByVariant,
		); err != nil {
			return nil, err
		}

		if err := s.cartOperations.applyFinalQuantitiesToCart(
			txCtx,
			cart.ID,
			existingItemByVariant,
			finalQuantityByVariant,
		); err != nil {
			return nil, err
		}

		items, err := s.cartRepo.FindItemsByCartID(txCtx, cart.ID)
		if err != nil {
			return nil, err
		}

		if len(items) == 0 {
			if err := s.cartRepo.DeleteCart(txCtx, cart.ID); err != nil {
				return nil, err
			}
			return s.cartOperations.buildEmptyCartResponse(&userID, currencyMap), nil
		}

		return s.buildCartResponseWithItems(
			txCtx,
			sellerID,
			userID,
			cart,
			items,
			currencyMap,
		)
	})
}

func (s *CartServiceImpl) GetUserCart(
	ctx context.Context,
	userID, sellerID uint,
) (*model.CartResponse, error) {
	return db.WithTransactionResult(ctx, func(txCtx context.Context) (*model.CartResponse, error) {
		currencyMap, err := s.userSvc.GetPreferredCurrency(txCtx, userID, sellerID)
		if err != nil {
			return nil, err
		}

		cart, err := s.getExistingOrCreateCart(txCtx, userID, false)
		if err != nil {
			return nil, err
		}
		if cart == nil {
			return s.cartOperations.buildEmptyCartResponse(&userID, currencyMap), nil
		}

		items, err := s.cartRepo.FindItemsByCartID(txCtx, cart.ID)
		if err != nil {
			return nil, err
		}

		if len(items) == 0 {
			return s.cartOperations.buildEmptyCartResponse(&userID, currencyMap), nil
		}

		return s.buildCartResponseWithItems(txCtx, sellerID, userID, cart, items, currencyMap)
	})
}

func (s *CartServiceImpl) DeleteCart(
	ctx context.Context,
	userID, sellerID, cartID uint,
) (*model.CartResponse, error) {
	return db.WithTransactionResult(ctx, func(txCtx context.Context) (*model.CartResponse, error) {
		currencyMap, err := s.userSvc.GetPreferredCurrency(txCtx, userID, sellerID)
		if err != nil {
			return nil, err
		}

		cart, err := s.cartRepo.FindByID(txCtx, cartID)
		if err != nil {
			return nil, err
		}
		if cart.UserID == nil || *cart.UserID != userID {
			return nil, errs.NewAppError(errs.INVALID_ID_CODE, "Cart not found", 404)
		}

		if err := s.cartRepo.DeleteItemsByCartID(txCtx, cart.ID); err != nil {
			return nil, err
		}
		if err := s.cartRepo.DeleteCart(txCtx, cart.ID); err != nil {
			return nil, err
		}
		return s.cartOperations.buildEmptyCartResponse(&userID, currencyMap), nil
	})
}

// LockActiveCartForCheckout transitions one active cart into checkout state.
// Step 1: find current active cart for user.
// Step 2: atomically transition status active -> checkout.
// Step 3: if transition fails, surface checkout lock/state conflict.
func (s *CartServiceImpl) LockActiveCartForCheckout(
	ctx context.Context,
	userID uint,
) (*entity.Cart, error) {
	cart, err := s.cartRepo.FindActiveCartByUserID(ctx, userID)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok && appErr.Code == errs.INVALID_ID_CODE {
			checkoutCart, checkoutErr := s.cartRepo.FindCheckoutCartByUserID(ctx, userID)
			if checkoutErr != nil {
				return nil, checkoutErr
			}
			if checkoutCart != nil {
				return nil, orderError.ErrCartAlreadyInCheckout
			}
		}
		return nil, err
	}

	locked, err := s.cartRepo.UpdateCartStatusIfCurrent(
		ctx,
		cart.ID,
		entity.CART_STATUS_ACTIVE,
		entity.CART_STATUS_CHECKOUT,
	)
	if err != nil {
		return nil, err
	}
	if locked {
		cart.Status = entity.CART_STATUS_CHECKOUT
		return cart, nil
	}

	latest, err := s.cartRepo.FindByID(ctx, cart.ID)
	if err != nil {
		return nil, err
	}
	if latest.Status == entity.CART_STATUS_CHECKOUT {
		return nil, orderError.ErrCartAlreadyInCheckout
	}
	return nil, orderError.ErrCartNotActive
}

// UnlockCheckoutCart reverts a locked cart back to active for retry.
// This is used as compensation when order creation fails after lock acquisition.
func (s *CartServiceImpl) UnlockCheckoutCart(ctx context.Context, cartID uint) error {
	_, err := s.cartRepo.UpdateCartStatusIfCurrent(
		ctx,
		cartID,
		entity.CART_STATUS_CHECKOUT,
		entity.CART_STATUS_ACTIVE,
	)
	return err
}

// MarkCartConverted finalizes checkout state after order commit data is ready.
// Step 1: link cart -> order via order_id.
// Step 2: transition checkout -> converted.
func (s *CartServiceImpl) MarkCartConverted(
	ctx context.Context,
	cartID, orderID, userID uint,
) error {
	if err := s.cartRepo.SetCartOrderID(ctx, cartID, orderID); err != nil {
		return err
	}
	converted, err := s.cartRepo.UpdateCartStatusIfCurrent(
		ctx,
		cartID,
		entity.CART_STATUS_CHECKOUT,
		entity.CART_STATUS_CONVERTED,
	)
	if err != nil {
		return err
	}
	if !converted {
		return orderError.ErrCartNotActive
	}
	return err
}

// ReactivateCartByOrderID reopens the cart associated with a failed/cancelled order.
// The operation is idempotent: if no cart is linked, it succeeds with no-op.
func (s *CartServiceImpl) ReactivateCartByOrderID(ctx context.Context, orderID uint) error {
	converted, err := s.cartRepo.FindByOrderID(ctx, orderID)
	if err != nil {
		return err
	}
	if converted == nil {
		return nil
	}
	if err := s.cartRepo.UpdateCartStatus(ctx, converted.ID, entity.CART_STATUS_ACTIVE); err != nil {
		return err
	}
	return s.cartRepo.ClearCartOrderID(ctx, converted.ID)
}

// ============================================================================
// Private helpers unique to CartServiceImpl (not shared with GuestCartServiceImpl)
// ============================================================================

func (s *CartServiceImpl) getExistingOrCreateCart(
	ctx context.Context,
	userID uint,
	createIfMissing bool,
) (*entity.Cart, error) {
	cart, err := s.cartRepo.FindByUserID(ctx, userID)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok && appErr.Code == errs.INVALID_ID_CODE {
			if !createIfMissing {
				return nil, nil
			}
			// Cart doesn't exist, create it
			cart = &entity.Cart{
				UserID:   &userID,
				Metadata: db.JSONMap{},
			}
			if err := s.cartRepo.CreateCart(ctx, cart); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return cart, nil
}

// buildCartResponseWithItems constructs the full CartResponse with promotions for authenticated users.
func (s *CartServiceImpl) buildCartResponseWithItems(
	ctx context.Context,
	sellerID, userID uint,
	cart *entity.Cart,
	items []entity.CartItem,
	currencyMap *userModel.CurrencyResponse,
) (*model.CartResponse, error) {
	variantMap, err := s.cartOperations.fetchVariantMap(ctx, items, sellerID)
	if err != nil {
		return nil, err
	}

	promoReq, err := s.buildPromotionRequest(ctx, sellerID, userID, items, variantMap)
	if err != nil {
		return nil, err
	}

	log.InfoWithContext(ctx, "Calling Promotion Service for Cart validation")
	promoSummary, err := s.promotionSvc.ApplyPromotionsToCart(ctx, promoReq)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to apply promotions", err)
		return nil, orderError.ErrPromotionServiceUnavailable(err)
	}

	if err := s.syncCartItemPromotions(ctx, items, promoSummary); err != nil {
		return nil, err
	}

	appliedRows, err := s.cartRepo.FindAppliedCouponsByCartID(ctx, cart.ID)
	if err != nil {
		return nil, err
	}
	appliedIDs := make([]uint, 0, len(appliedRows))
	for _, row := range appliedRows {
		appliedIDs = append(appliedIDs, row.DiscountCodeID)
	}

	couponReq := buildCouponCartRequestFromPromo(userID, sellerID, promoSummary, promoReq, appliedIDs)
	couponSummary, err := s.couponApplySvc.ApplyCouponsToCart(ctx, couponReq)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to apply coupons", err)
		return nil, err
	}

	appliedRows, err = s.selfHealInvalidAppliedCoupons(ctx, cart.ID, appliedRows, couponSummary)
	if err != nil {
		return nil, err
	}
	healedIDs := make([]uint, 0, len(appliedRows))
	for _, row := range appliedRows {
		healedIDs = append(healedIDs, row.DiscountCodeID)
	}
	couponReq.AppliedDiscountCodeIDs = healedIDs

	available, err := s.couponApplySvc.ListAvailableCouponsForCart(ctx, couponReq)
	if err != nil {
		log.WarnWithContext(ctx, "Failed to list available coupons: "+err.Error())
		available = &promotionModel.AvailableCouponsResponse{}
	}

	return factory.BuildCartResponse(
		cart,
		items,
		promoSummary,
		couponSummary,
		appliedRows,
		available,
		currencyMap,
		variantMap,
	), nil
}

// buildPromotionRequest constructs a CartValidationRequest for authenticated users.
// Checks real order history to determine IsFirstOrder flag.
func (s *CartServiceImpl) buildPromotionRequest(
	ctx context.Context,
	sellerID, userID uint,
	items []entity.CartItem,
	variantMap map[uint]productModel.VariantDetailResponse,
) (*promotionModel.CartValidationRequest, error) {
	hasPastOrders, err := s.orderRepo.HasPastOrders(ctx, userID)
	if err != nil {
		log.WarnWithContext(ctx, "Failed to check user order history: "+err.Error())
		// Safest is to assume true so we don't accidentally give a first-order discount
		hasPastOrders = true
	}

	promoReq := &promotionModel.CartValidationRequest{
		SellerID:      sellerID,
		CustomerID:    &userID, // Optional but good for segment targeting
		IsFirstOrder:  !hasPastOrders,
		Items:         make([]promotionModel.CartItem, len(items)),
		SubtotalCents: 0,
		// TODO [FULFILLMENT]: Replace metadata fallback with Fulfillment Service quote
		// (shipping, handling, delivery method constraints) when service is ready.
		ShippingCents: 5000,
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

		variantPriceCents := int64(variant.Price * 100) // Convert floating price format to cents
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

func buildCouponCartRequestFromPromo(
	userID, sellerID uint,
	promo *promotionModel.AppliedPromotionSummary,
	promoReq *promotionModel.CartValidationRequest,
	appliedIDs []uint,
) *promotionModel.CouponCartRequest {
	categoryByItemID := map[string]uint{}
	for _, item := range promoReq.Items {
		categoryByItemID[item.ItemID] = item.CategoryID
	}

	items := make([]promotionModel.CartItem, 0, len(promo.Items))
	for _, summaryItem := range promo.Items {
		qty := summaryItem.Quantity
		if qty <= 0 {
			qty = 1
		}
		unit := summaryItem.FinalPriceCents / int64(qty)
		if unit <= 0 {
			unit = summaryItem.OriginalUnitPriceCents
		}
		items = append(items, promotionModel.CartItem{
			ItemID:     summaryItem.ItemID,
			ProductID:  summaryItem.ProductID,
			VariantID:  summaryItem.VariantID,
			CategoryID: categoryByItemID[summaryItem.ItemID],
			Quantity:   qty,
			PriceCents: unit,
			TotalCents: summaryItem.FinalPriceCents,
		})
	}

	promotionsAllow := true
	for _, p := range promo.AppliedPromotions {
		if p.Promotion != nil && p.Promotion.CanStackWithCoupons != nil && !*p.Promotion.CanStackWithCoupons {
			promotionsAllow = false
			break
		}
	}

	return &promotionModel.CouponCartRequest{
		SellerID:               sellerID,
		CustomerID:             &userID,
		IsFirstOrder:           promoReq.IsFirstOrder,
		Items:                  items,
		SubtotalCents:          promo.FinalSubtotal,
		ShippingCents:          promoReq.ShippingCents,
		AppliedDiscountCodeIDs: appliedIDs,
		PromotionsAllowCoupons: promotionsAllow,
	}
}

// syncCartItemPromotions replaces stored promotion IDs per cart line with the set just applied.
func (s *CartServiceImpl) syncCartItemPromotions(
	ctx context.Context,
	items []entity.CartItem,
	promo *promotionModel.AppliedPromotionSummary,
) error {
	promoIDsByItemID := map[string][]uint{}
	if promo != nil {
		for _, summaryItem := range promo.Items {
			ids := make([]uint, 0, len(summaryItem.AppliedPromotions))
			for _, ap := range summaryItem.AppliedPromotions {
				ids = append(ids, ap.PromotionID)
			}
			promoIDsByItemID[summaryItem.ItemID] = ids
		}
	}

	for _, item := range items {
		itemIDStr := strconv.Itoa(int(item.ID))
		desired := promoIDsByItemID[itemIDStr]
		if desired == nil {
			desired = []uint{}
		}
		if err := s.cartRepo.SyncCartItemPromotions(ctx, item.ID, desired); err != nil {
			return err
		}
	}
	return nil
}

// selfHealInvalidAppliedCoupons deletes cart_applied_coupon rows that did not survive ApplyCouponsToCart.
func (s *CartServiceImpl) selfHealInvalidAppliedCoupons(
	ctx context.Context,
	cartID uint,
	appliedRows []entity.CartAppliedCoupon,
	couponSummary *promotionModel.AppliedCouponSummary,
) ([]entity.CartAppliedCoupon, error) {
	valid := map[uint]struct{}{}
	if couponSummary != nil {
		for _, c := range couponSummary.AppliedCoupons {
			if c.DiscountCode != nil {
				valid[c.DiscountCode.ID] = struct{}{}
			}
		}
	}

	invalidIDs := make([]uint, 0)
	kept := make([]entity.CartAppliedCoupon, 0, len(appliedRows))
	for _, row := range appliedRows {
		if _, ok := valid[row.DiscountCodeID]; ok {
			kept = append(kept, row)
			continue
		}
		invalidIDs = append(invalidIDs, row.DiscountCodeID)
	}
	if len(invalidIDs) > 0 {
		if err := s.cartRepo.RemoveAppliedCouponsByDiscountCodeIDs(ctx, cartID, invalidIDs); err != nil {
			return nil, err
		}
		log.InfoWithContext(ctx, "Self-healed invalid applied coupons")
	}
	return kept, nil
}
