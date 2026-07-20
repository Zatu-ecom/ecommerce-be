package service

import (
	"context"

	"ecommerce-be/common/db"
	errs "ecommerce-be/common/error"
	"ecommerce-be/order/entity"
	"ecommerce-be/order/model"
	"ecommerce-be/order/repository"

	inventoryService "ecommerce-be/inventory/service"
	productVariantService "ecommerce-be/product/service"
	promotionService "ecommerce-be/promotion/service"
	userModel "ecommerce-be/user/model"
	userService "ecommerce-be/user/service"
)

// GuestCartService defines operations for guest (device-identified) carts.
// Following ISP (Interface Segregation) — focused interface for guest cart operations only.
type GuestCartService interface {
	AddToGuestCart(
		ctx context.Context,
		deviceID string,
		sellerID uint,
		req model.AddCartItemRequest,
	) (*model.CartResponse, error)
	GetGuestCart(ctx context.Context, deviceID string, sellerID uint) (*model.CartResponse, error)
	DeleteGuestCart(
		ctx context.Context,
		deviceID string,
		sellerID uint,
		cartID uint,
	) (*model.CartResponse, error)
}

// GuestCartServiceImpl implements GuestCartService.
// Reuses cartOperations for shared logic (inventory, variant fetch, promotion).
// Differs from CartServiceImpl in cart lookup (by device ID vs user ID) and currency (seller default vs user preferred).
type GuestCartServiceImpl struct {
	cartOperations
	userSvc userService.UserService
}

// NewGuestCartService creates a new GuestCartService.
func NewGuestCartService(
	cartRepo repository.CartRepository,
	promotionSvc promotionService.PromotionService,
	inventorySvc inventoryService.InventoryQueryService,
	variantQuerySvc productVariantService.VariantQueryService,
	userSvc userService.UserService,
) GuestCartService {
	return &GuestCartServiceImpl{
		cartOperations: cartOperations{
			cartRepo:        cartRepo,
			promotionSvc:    promotionSvc,
			inventorySvc:    inventorySvc,
			variantQuerySvc: variantQuerySvc,
		},
		userSvc: userSvc,
	}
}

// AddToGuestCart adds items to a guest cart identified by device ID.
// Follows same flow as CartServiceImpl.AddToCart but uses device ID for cart lookup.
func (s *GuestCartServiceImpl) AddToGuestCart(
	ctx context.Context,
	deviceID string,
	sellerID uint,
	req model.AddCartItemRequest,
) (*model.CartResponse, error) {
	return db.WithTransactionResult(ctx, func(txCtx context.Context) (*model.CartResponse, error) {
		currencyMap, err := s.userSvc.GetSellerDefaultCurrency(txCtx, sellerID)
		if err != nil {
			return nil, err
		}

		hasPositiveQty := hasPositiveQuantity(req.Items)

		cart, err := s.getExistingOrCreateGuestCart(txCtx, deviceID, hasPositiveQty)
		if err != nil {
			return nil, err
		}

		if cart == nil && !hasPositiveQty {
			return s.cartOperations.buildEmptyCartResponse(nil, currencyMap), nil
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
			return s.cartOperations.buildEmptyCartResponse(nil, currencyMap), nil
		}

		return s.buildGuestCartResponse(txCtx, sellerID, cart, items, currencyMap)
	})
}

// GetGuestCart retrieves the guest cart identified by device ID.
func (s *GuestCartServiceImpl) GetGuestCart(
	ctx context.Context,
	deviceID string,
	sellerID uint,
) (*model.CartResponse, error) {
	return db.WithTransactionResult(ctx, func(txCtx context.Context) (*model.CartResponse, error) {
		currencyMap, err := s.userSvc.GetSellerDefaultCurrency(txCtx, sellerID)
		if err != nil {
			return nil, err
		}

		cart, err := s.cartRepo.FindByDeviceID(txCtx, deviceID)
		if err != nil {
			if appErr, ok := err.(*errs.AppError); ok && appErr.Code == errs.INVALID_ID_CODE {
				return s.cartOperations.buildEmptyCartResponse(nil, currencyMap), nil
			}
			return nil, err
		}

		items, err := s.cartRepo.FindItemsByCartID(txCtx, cart.ID)
		if err != nil {
			return nil, err
		}

		if len(items) == 0 {
			return s.cartOperations.buildEmptyCartResponse(nil, currencyMap), nil
		}

		return s.buildGuestCartResponse(txCtx, sellerID, cart, items, currencyMap)
	})
}

// DeleteGuestCart deletes a guest cart by cart ID, verifying device ownership.
func (s *GuestCartServiceImpl) DeleteGuestCart(
	ctx context.Context,
	deviceID string,
	sellerID uint,
	cartID uint,
) (*model.CartResponse, error) {
	return db.WithTransactionResult(ctx, func(txCtx context.Context) (*model.CartResponse, error) {
		currencyMap, err := s.userSvc.GetSellerDefaultCurrency(txCtx, sellerID)
		if err != nil {
			return nil, err
		}

		cart, err := s.cartRepo.FindByID(txCtx, cartID)
		if err != nil {
			return nil, err
		}
		if cart.DeviceID == nil || *cart.DeviceID != deviceID {
			return nil, errs.NewAppError(errs.INVALID_ID_CODE, "Cart not found", 404)
		}

		if err := s.cartRepo.DeleteItemsByCartID(txCtx, cart.ID); err != nil {
			return nil, err
		}
		if err := s.cartRepo.DeleteCart(txCtx, cart.ID); err != nil {
			return nil, err
		}
		return s.cartOperations.buildEmptyCartResponse(nil, currencyMap), nil
	})
}

// ============================================================================
// Private helpers
// ============================================================================

// getExistingOrCreateGuestCart finds or creates an active cart for a device.
func (s *GuestCartServiceImpl) getExistingOrCreateGuestCart(
	ctx context.Context,
	deviceID string,
	createIfMissing bool,
) (*entity.Cart, error) {
	cart, err := s.cartRepo.FindByDeviceID(ctx, deviceID)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok && appErr.Code == errs.INVALID_ID_CODE {
			if !createIfMissing {
				return nil, nil
			}
			// Cart doesn't exist, create it
			deviceIDCopy := deviceID
			cart = &entity.Cart{
				DeviceID: &deviceIDCopy,
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

// buildGuestCartResponse constructs the full CartResponse for a guest cart.
// Guest carts use seller default currency and have no customer-specific promotions.
func (s *GuestCartServiceImpl) buildGuestCartResponse(
	ctx context.Context,
	sellerID uint,
	cart *entity.Cart,
	items []entity.CartItem,
	currencyMap *userModel.CurrencyResponse,
) (*model.CartResponse, error) {
	return s.cartOperations.buildCartResponseWithItems(
		ctx,
		sellerID,
		cart,
		items,
		currencyMap,
		0, // guest: userID = 0 (no authenticated user)
	)
}
