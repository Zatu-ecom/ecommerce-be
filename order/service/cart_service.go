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
}

type CartServiceImpl struct {
	// Embed shared cart operations (ISP — extracted to avoid duplication with GuestCartServiceImpl)
	cartOperations
	orderRepo repository.OrderRepository
	userSvc   userService.UserService
	promotionSvc    promotionService.PromotionService
}

func NewCartService(
	cartRepo repository.CartRepository,
	orderRepo repository.OrderRepository,
	promotionSvc promotionService.PromotionService,
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
		orderRepo:       orderRepo,
		userSvc:         userSvc,
		promotionSvc:    promotionSvc,
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

	return factory.BuildCartResponse(
		cart,
		items,
		promoSummary,
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
